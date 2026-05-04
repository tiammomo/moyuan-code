package logging

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

type Event map[string]any

type Query struct {
	Stream  string
	IssueID string
	RunID   string
	Event   string
	Limit   int
}

type Record struct {
	ID         string         `json:"id"`
	Stream     string         `json:"stream"`
	Channel    string         `json:"channel"`
	Event      string         `json:"event"`
	Timestamp  string         `json:"ts"`
	IssueID    string         `json:"issue_id,omitempty"`
	RunID      string         `json:"run_id,omitempty"`
	SubagentID string         `json:"subagent_id,omitempty"`
	TraceID    string         `json:"trace_id,omitempty"`
	Status     string         `json:"status,omitempty"`
	Decision   string         `json:"decision,omitempty"`
	Reason     string         `json:"reason,omitempty"`
	Data       map[string]any `json:"data"`
}

var (
	validStreamPattern     = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	credentialPattern      = regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret|credential)\s*[:=]\s*[^,\s]+`)
	openAIKeyPattern       = regexp.MustCompile(`sk-[A-Za-z0-9_-]{8,}`)
	privateKeyPattern      = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`)
	sensitiveFieldPattern  = regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret|credential|private[_-]?key)`)
	errInvalidLogStream    = errors.New("invalid_log_stream")
	defaultStreamsFallback = []string{"audit", "run", "quality", "memory", "git", "model", "release", "error"}
)

func Log(rootDir string, stream string, event string, data map[string]any) error {
	stream, err := validateStream(stream)
	if err != nil {
		return err
	}
	record := Event{
		"ts":     time.Now().UTC().Format(time.RFC3339Nano),
		"stream": stream,
		"event":  event,
	}
	for k, v := range data {
		record[k] = v
	}
	path := filepath.Join(workspace.ForRoot(rootDir).LogsDir, stream+".jsonl")
	return fsutil.AppendJSONL(path, record)
}

func Tail(rootDir string, stream string, limit int) ([]string, error) {
	stream, err := validateStream(stream)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(workspace.ForRoot(rootDir).LogsDir, stream+".jsonl")
	return fsutil.TailLines(path, limit)
}

func List(rootDir string, query Query) ([]Record, error) {
	limit := normalizeLimit(query.Limit)
	streams, err := resolveStreams(rootDir, query.Stream)
	if err != nil {
		return nil, err
	}
	records := []Record{}
	for _, stream := range streams {
		lines, err := Tail(rootDir, stream, limit)
		if err != nil {
			return nil, err
		}
		for _, line := range lines {
			record, ok, err := parseRecord(stream, line)
			if err != nil {
				return nil, err
			}
			if !ok || !matches(record, query) {
				continue
			}
			records = append(records, record)
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		return timestampForSort(records[i]).After(timestampForSort(records[j]))
	})
	if len(records) > limit {
		records = records[:limit]
	}
	for i := range records {
		records[i].ID = fmt.Sprintf("%s-%03d", records[i].Stream, i+1)
	}
	return records, nil
}

func IsInvalidStreamError(err error) bool {
	return errors.Is(err, errInvalidLogStream)
}

func resolveStreams(rootDir string, stream string) ([]string, error) {
	stream = strings.TrimSpace(stream)
	if stream == "" || stream == "all" {
		return discoverStreams(rootDir)
	}
	stream, err := validateStream(stream)
	if err != nil {
		return nil, err
	}
	return []string{stream}, nil
}

func validateStream(stream string) (string, error) {
	stream = strings.TrimSpace(stream)
	if stream == "" || !validStreamPattern.MatchString(stream) {
		return "", errInvalidLogStream
	}
	return stream, nil
}

func discoverStreams(rootDir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(workspace.ForRoot(rootDir).LogsDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return defaultStreamsFallback, nil
	}
	streams := make([]string, 0, len(matches))
	for _, path := range matches {
		stream := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		if validStreamPattern.MatchString(stream) {
			streams = append(streams, stream)
		}
	}
	sort.Strings(streams)
	return streams, nil
}

func parseRecord(fallbackStream string, line string) (Record, bool, error) {
	raw := map[string]any{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return Record{}, false, fmt.Errorf("parse_log_record: %w", err)
	}
	event := stringField(raw, "event")
	if event == "" {
		return Record{}, false, nil
	}
	stream := stringField(raw, "stream")
	if stream == "" {
		stream = fallbackStream
	}
	data := map[string]any{}
	for key, value := range raw {
		if key == "ts" || key == "stream" || key == "event" {
			continue
		}
		data[key] = sanitizeValue(key, value)
	}
	return Record{
		Stream:     stream,
		Channel:    stream,
		Event:      event,
		Timestamp:  stringField(raw, "ts"),
		IssueID:    stringField(data, "issue_id"),
		RunID:      stringField(data, "run_id"),
		SubagentID: stringField(data, "subagent_id"),
		TraceID:    stringField(data, "trace_id"),
		Status:     stringField(data, "status"),
		Decision:   stringField(data, "decision"),
		Reason:     stringField(data, "reason"),
		Data:       data,
	}, true, nil
}

func matches(record Record, query Query) bool {
	if query.IssueID != "" && record.IssueID != query.IssueID {
		return false
	}
	if query.RunID != "" && record.RunID != query.RunID {
		return false
	}
	if query.Event != "" && record.Event != query.Event {
		return false
	}
	return true
}

func sanitizeValue(key string, value any) any {
	switch typed := value.(type) {
	case string:
		if sensitiveFieldPattern.MatchString(key) && !isSecretReference(typed) {
			return "[REDACTED]"
		}
		return redactText(typed)
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, sanitizeValue("", item))
		}
		return items
	case map[string]any:
		copy := map[string]any{}
		for nestedKey, nestedValue := range typed {
			copy[nestedKey] = sanitizeValue(nestedKey, nestedValue)
		}
		return copy
	default:
		return typed
	}
}

func redactText(value string) string {
	value = privateKeyPattern.ReplaceAllString(value, "[REDACTED_PRIVATE_KEY]")
	value = credentialPattern.ReplaceAllString(value, "$1=[REDACTED]")
	value = openAIKeyPattern.ReplaceAllString(value, "sk-[REDACTED]")
	return value
}

func isSecretReference(value string) bool {
	return strings.HasPrefix(value, "env:") || strings.HasPrefix(value, "secret:")
}

func stringField(record map[string]any, key string) string {
	value, ok := record[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func timestampForSort(record Record) time.Time {
	ts, err := time.Parse(time.RFC3339Nano, record.Timestamp)
	if err != nil {
		return time.Time{}
	}
	return ts
}
