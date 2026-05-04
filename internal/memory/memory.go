package memory

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
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

const recordThreshold = 60

var ErrNotRecorded = errors.New("memory candidate was not recorded")

var credentialPattern = regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret)\s*[:=]\s*[^,\s]+`)
var openAIKeyPattern = regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`)
var privateKeyPattern = regexp.MustCompile(`(?i)-----BEGIN [A-Z ]*PRIVATE KEY-----`)

type Record struct {
	ID                string   `json:"id"`
	Kind              string   `json:"kind"`
	Summary           string   `json:"summary"`
	Tags              []string `json:"tags"`
	Source            string   `json:"source"`
	Scope             string   `json:"scope"`
	Scopes            []string `json:"scopes"`
	Confidence        float64  `json:"confidence"`
	CreatedBy         string   `json:"created_by"`
	TraceID           string   `json:"trace_id"`
	SourceCandidateID string   `json:"source_candidate_id,omitempty"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	Fingerprint       string   `json:"fingerprint"`
	Compact           bool     `json:"compact"`
}

type Candidate struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Summary     string   `json:"summary"`
	Tags        []string `json:"tags"`
	Source      string   `json:"source"`
	Scope       string   `json:"scope"`
	Scopes      []string `json:"scopes"`
	Score       int      `json:"score"`
	Confidence  float64  `json:"confidence"`
	Decision    string   `json:"decision"`
	Reasons     []string `json:"reasons"`
	CreatedBy   string   `json:"created_by"`
	TraceID     string   `json:"trace_id"`
	Fingerprint string   `json:"fingerprint"`
	CreatedAt   string   `json:"created_at"`
}

type GateDecision struct {
	ID          string    `json:"id"`
	CandidateID string    `json:"candidate_id"`
	Status      string    `json:"status"`
	Reasons     []string  `json:"reasons"`
	DuplicateOf string    `json:"duplicate_of,omitempty"`
	RecordID    string    `json:"record_id,omitempty"`
	Candidate   Candidate `json:"candidate"`
	Record      *Record   `json:"record,omitempty"`
	CreatedAt   string    `json:"created_at"`
}

type CompactSummary struct {
	ID              string         `json:"id"`
	CreatedAt       string         `json:"created_at"`
	RecordsSeen     int            `json:"records_seen"`
	CandidateEvents int            `json:"candidate_events"`
	Strategy        string         `json:"strategy"`
	OutputStatus    string         `json:"output_status"`
	SourceRecordIDs []string       `json:"source_record_ids"`
	Topics          []CompactTopic `json:"topics"`
}

type CompactTopic struct {
	Scope     string   `json:"scope"`
	Kind      string   `json:"kind"`
	Count     int      `json:"count"`
	RecordIDs []string `json:"record_ids"`
	Summaries []string `json:"summaries"`
}

type memoryPaths struct {
	records     string
	candidates  string
	staging     string
	latest      string
	compactions string
}

func Submit(rootDir string, kind string, summary string, tags []string, source string) (GateDecision, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	kind = normalizeKind(kind)
	tags = normalizeList(tags)
	source = normalizeSource(source)
	createdBy := source
	traceID := newID("trace")

	rawSummary := strings.TrimSpace(summary)
	sanitizedSummary := redactSensitive(rawSummary)
	scopes := inferScopes(kind, tags, sanitizedSummary)
	scope := primaryScope(scopes)
	fingerprint := fingerprint(sanitizedSummary)
	sensitive := containsSensitive(rawSummary)
	score := scoreCandidate(kind, sanitizedSummary, tags, scopes, source)
	reasons := scoreReasons(kind, sanitizedSummary, tags, scopes, source)
	if sensitive {
		score = 0
		reasons = append([]string{"sensitive_content"}, reasons...)
	}
	candidate := Candidate{
		ID:          newID("memcand"),
		Kind:        kind,
		Summary:     sanitizedSummary,
		Tags:        tags,
		Source:      source,
		Scope:       scope,
		Scopes:      scopes,
		Score:       score,
		Confidence:  confidence(score),
		Reasons:     reasons,
		CreatedBy:   createdBy,
		TraceID:     traceID,
		Fingerprint: fingerprint,
		CreatedAt:   now,
	}
	decision := GateDecision{
		ID:          newID("memgate"),
		CandidateID: candidate.ID,
		Reasons:     reasons,
		Candidate:   candidate,
		CreatedAt:   now,
	}
	paths := pathsFor(rootDir)

	if sensitive {
		candidate.Decision = "rejected"
		decision.Candidate = candidate
		decision.Status = "rejected"
		if err := appendDecision(rootDir, paths, candidate, decision); err != nil {
			return GateDecision{}, err
		}
		_ = logging.Log(rootDir, "memory", "memory.candidate.rejected", map[string]any{"candidate_id": candidate.ID, "reason": "sensitive_content", "trace_id": traceID})
		return decision, nil
	}

	records, err := readRecords(paths.records)
	if err != nil {
		return GateDecision{}, err
	}
	if duplicate := findDuplicate(records, fingerprint); duplicate != "" {
		candidate.Decision = "deduped"
		decision.Candidate = candidate
		decision.Status = "deduped"
		decision.DuplicateOf = duplicate
		decision.Reasons = append([]string{"duplicate_memory"}, decision.Reasons...)
		if err := appendDecision(rootDir, paths, candidate, decision); err != nil {
			return GateDecision{}, err
		}
		_ = logging.Log(rootDir, "memory", "memory.candidate.deduped", map[string]any{"candidate_id": candidate.ID, "duplicate_of": duplicate, "trace_id": traceID})
		return decision, nil
	}

	if score < recordThreshold {
		candidate.Decision = "staged"
		decision.Candidate = candidate
		decision.Status = "staged"
		decision.Reasons = append([]string{"below_record_threshold"}, decision.Reasons...)
		if err := appendDecision(rootDir, paths, candidate, decision); err != nil {
			return GateDecision{}, err
		}
		_ = logging.Log(rootDir, "memory", "memory.candidate.staged", map[string]any{"candidate_id": candidate.ID, "score": score, "trace_id": traceID})
		return decision, nil
	}

	record := Record{
		ID:                newID("mem"),
		Kind:              kind,
		Summary:           sanitizedSummary,
		Tags:              tags,
		Source:            source,
		Scope:             scope,
		Scopes:            scopes,
		Confidence:        confidence(score),
		CreatedBy:         createdBy,
		TraceID:           traceID,
		SourceCandidateID: candidate.ID,
		CreatedAt:         now,
		UpdatedAt:         now,
		Fingerprint:       fingerprint,
		Compact:           false,
	}
	candidate.Decision = "recorded"
	decision.Candidate = candidate
	decision.Status = "recorded"
	decision.RecordID = record.ID
	decision.Record = &record
	if err := appendDecision(rootDir, paths, candidate, decision); err != nil {
		return GateDecision{}, err
	}
	if err := fsutil.AppendJSONL(paths.records, record); err != nil {
		return GateDecision{}, err
	}
	_, _ = Compact(rootDir)
	_ = logging.Log(rootDir, "memory", "memory.record.added", map[string]any{"memory_id": record.ID, "candidate_id": candidate.ID, "kind": kind, "confidence": record.Confidence, "trace_id": traceID})
	return decision, nil
}

func Add(rootDir string, kind string, summary string, tags []string, source string) (Record, error) {
	decision, err := Submit(rootDir, kind, summary, tags, source)
	if err != nil {
		return Record{}, err
	}
	if decision.Record != nil {
		return *decision.Record, nil
	}
	return Record{}, fmt.Errorf("%w: %s", ErrNotRecorded, decision.Status)
}

func Search(rootDir string, query string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	lines, err := fsutil.TailLines(pathsFor(rootDir).records, 500)
	if err != nil {
		return nil, err
	}
	result := []string{}
	query = strings.ToLower(query)
	for _, line := range lines {
		if query == "" || strings.Contains(strings.ToLower(line), query) {
			result = append(result, line)
			if len(result) >= limit {
				break
			}
		}
	}
	_ = logging.Log(rootDir, "memory", "memory.retrieve.completed", map[string]any{"query": query, "count": len(result)})
	return result, nil
}

func ListCandidates(rootDir string, limit int) ([]GateDecision, error) {
	if limit <= 0 {
		limit = 20
	}
	lines, err := fsutil.TailLines(pathsFor(rootDir).staging, limit)
	if err != nil {
		return nil, err
	}
	decisions := []GateDecision{}
	for _, line := range lines {
		var decision GateDecision
		if err := json.Unmarshal([]byte(line), &decision); err == nil {
			decisions = append(decisions, decision)
		}
	}
	return decisions, nil
}

func Compact(rootDir string) (CompactSummary, error) {
	paths := pathsFor(rootDir)
	records, err := readRecords(paths.records)
	if err != nil {
		return CompactSummary{}, err
	}
	candidateEvents, err := fsutil.TailLines(paths.staging, 2000)
	if err != nil {
		return CompactSummary{}, err
	}
	grouped := map[string]*CompactTopic{}
	sourceIDs := []string{}
	for _, record := range records {
		sourceIDs = append(sourceIDs, record.ID)
		key := record.Scope + ":" + record.Kind
		topic, ok := grouped[key]
		if !ok {
			topic = &CompactTopic{Scope: record.Scope, Kind: record.Kind}
			grouped[key] = topic
		}
		topic.Count++
		topic.RecordIDs = append(topic.RecordIDs, record.ID)
		if len(topic.Summaries) < 5 {
			topic.Summaries = append(topic.Summaries, record.Summary)
		}
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	topics := []CompactTopic{}
	for _, key := range keys {
		topics = append(topics, *grouped[key])
	}
	summary := CompactSummary{
		ID:              newID("memcompact"),
		CreatedAt:       time.Now().UTC().Format(time.RFC3339Nano),
		RecordsSeen:     len(records),
		CandidateEvents: len(candidateEvents),
		Strategy:        "phase1-record-gate-summary",
		OutputStatus:    "ready",
		SourceRecordIDs: sourceIDs,
		Topics:          topics,
	}
	if err := fsutil.WriteJSON(paths.latest, summary); err != nil {
		return CompactSummary{}, err
	}
	if err := fsutil.WriteJSON(filepath.Join(paths.compactions, summary.ID+".json"), summary); err != nil {
		return CompactSummary{}, err
	}
	_ = logging.Log(rootDir, "memory", "memory.compact.completed", map[string]any{"compact_id": summary.ID, "records_seen": summary.RecordsSeen, "candidate_events": summary.CandidateEvents})
	return summary, nil
}

func appendDecision(rootDir string, paths memoryPaths, candidate Candidate, decision GateDecision) error {
	if err := fsutil.AppendJSONL(paths.candidates, candidate); err != nil {
		return err
	}
	if err := fsutil.AppendJSONL(paths.staging, decision); err != nil {
		return err
	}
	_ = logging.Log(rootDir, "memory", "memory.candidate.evaluated", map[string]any{"candidate_id": candidate.ID, "decision": candidate.Decision, "score": candidate.Score, "trace_id": candidate.TraceID})
	return nil
}

func readRecords(path string) ([]Record, error) {
	lines, err := fsutil.TailLines(path, 5000)
	if err != nil {
		return nil, err
	}
	records := []Record{}
	for _, line := range lines {
		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		if record.Fingerprint == "" {
			record.Fingerprint = fingerprint(record.Summary)
		}
		if record.Scope == "" {
			record.Scope = "project"
		}
		records = append(records, record)
	}
	return records, nil
}

func findDuplicate(records []Record, fp string) string {
	if fp == "" {
		return ""
	}
	for _, record := range records {
		if record.Fingerprint == fp {
			return record.ID
		}
	}
	return ""
}

func normalizeKind(kind string) string {
	kind = strings.TrimSpace(strings.ToLower(kind))
	if kind == "" {
		return "fact"
	}
	return kind
}

func normalizeSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "unknown"
	}
	return source
}

func normalizeList(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func inferScopes(kind string, tags []string, summary string) []string {
	text := strings.ToLower(kind + " " + strings.Join(tags, " ") + " " + summary)
	scopes := []string{"project"}
	if strings.Contains(text, "issue") || strings.Contains(text, "phase") || strings.Contains(text, "run") {
		scopes = append(scopes, "lifecycle")
	}
	if strings.Contains(text, "quality") || strings.Contains(text, "test") || strings.Contains(text, "lint") || strings.Contains(text, "build") || strings.Contains(text, "测试") || strings.Contains(text, "质量") {
		scopes = append(scopes, "quality")
	}
	if strings.Contains(text, "runtime") || strings.Contains(text, "agent") || strings.Contains(text, "cli") {
		scopes = append(scopes, "runtime")
	}
	if strings.Contains(text, "memory") || strings.Contains(text, "compact") || strings.Contains(text, "记忆") {
		scopes = append(scopes, "memory")
	}
	if strings.Contains(text, "security") || strings.Contains(text, "permission") || strings.Contains(text, "auth") || strings.Contains(text, "权限") {
		scopes = append(scopes, "security")
	}
	return normalizeList(scopes)
}

func primaryScope(scopes []string) string {
	if len(scopes) == 0 {
		return "project"
	}
	priority := []string{"security", "quality", "memory", "runtime", "lifecycle", "project"}
	available := map[string]bool{}
	for _, scope := range scopes {
		available[scope] = true
	}
	for _, scope := range priority {
		if available[scope] {
			return scope
		}
	}
	return scopes[0]
}

func scoreCandidate(kind string, summary string, tags []string, scopes []string, source string) int {
	score := 0
	if isKnownKind(kind) {
		score += 30
	} else {
		score += 10
	}
	if source != "" {
		score += 10
	}
	if len([]rune(summary)) >= 12 {
		score += 10
	}
	if len([]rune(summary)) >= 40 {
		score += 15
	}
	if len(tags) > 0 {
		score += 10
	}
	if len(scopes) > 1 {
		score += 5
	}
	if hasActionableSignal(summary) {
		score += 15
	}
	if hasProjectSignal(summary) {
		score += 10
	}
	if score > 100 {
		return 100
	}
	return score
}

func scoreReasons(kind string, summary string, tags []string, scopes []string, source string) []string {
	reasons := []string{}
	if isKnownKind(kind) {
		reasons = append(reasons, "known_kind")
	}
	if source != "" {
		reasons = append(reasons, "source_available")
	}
	if len([]rune(summary)) >= 40 {
		reasons = append(reasons, "specific_summary")
	}
	if len(tags) > 0 {
		reasons = append(reasons, "tagged")
	}
	if len(scopes) > 1 {
		reasons = append(reasons, "scoped")
	}
	if hasActionableSignal(summary) {
		reasons = append(reasons, "actionable")
	}
	if hasProjectSignal(summary) {
		reasons = append(reasons, "project_signal")
	}
	return reasons
}

func isKnownKind(kind string) bool {
	switch kind {
	case "fact", "decision", "preference", "lesson", "quality", "security", "release", "comprehension":
		return true
	default:
		return false
	}
}

func hasActionableSignal(summary string) bool {
	text := strings.ToLower(summary)
	terms := []string{"must", "should", "requires", "uses", "avoid", "fix", "decision", "需要", "必须", "使用", "避免", "修复", "测试", "决定"}
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func hasProjectSignal(summary string) bool {
	text := strings.ToLower(summary)
	terms := []string{"project", "phase", "issue", "module", "runtime", "quality", "agent", "memory", "git", "cli", "项目", "模块", "测试", "质量"}
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func containsSensitive(value string) bool {
	return credentialPattern.MatchString(value) || openAIKeyPattern.MatchString(value) || privateKeyPattern.MatchString(value)
}

func redactSensitive(value string) string {
	value = credentialPattern.ReplaceAllString(value, "$1=[REDACTED]")
	value = openAIKeyPattern.ReplaceAllString(value, "sk-[REDACTED]")
	value = privateKeyPattern.ReplaceAllString(value, "[REDACTED_PRIVATE_KEY]")
	return value
}

func fingerprint(value string) string {
	text := strings.ToLower(redactSensitive(value))
	var builder strings.Builder
	previousSpace := false
	for _, r := range text {
		isWord := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= '\u4e00' && r <= '\u9fff')
		if isWord {
			builder.WriteRune(r)
			previousSpace = false
			continue
		}
		if !previousSpace {
			builder.WriteByte(' ')
			previousSpace = true
		}
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

func confidence(score int) float64 {
	if score <= 0 {
		return 0
	}
	if score >= 100 {
		return 1
	}
	return float64(score) / 100
}

func newID(prefix string) string {
	return prefix + "-" + time.Now().UTC().Format("20060102150405.000000000")
}

func pathsFor(rootDir string) memoryPaths {
	memDir := workspace.ForRoot(rootDir).MemoryDir
	return memoryPaths{
		records:     filepath.Join(memDir, "records.jsonl"),
		candidates:  filepath.Join(memDir, "candidates.jsonl"),
		staging:     filepath.Join(memDir, "staging.jsonl"),
		latest:      filepath.Join(memDir, "compact-latest.json"),
		compactions: filepath.Join(memDir, "compactions"),
	}
}
