package approvals

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type RequestOptions struct {
	TargetType  string         `json:"target_type"`
	TargetID    string         `json:"target_id"`
	Action      string         `json:"action"`
	RiskLevel   string         `json:"risk_level,omitempty"`
	RequestedBy string         `json:"requested_by,omitempty"`
	Reason      string         `json:"reason,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type DecisionOptions struct {
	Decision  string `json:"decision"`
	DecidedBy string `json:"decided_by,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type ListOptions struct {
	Status string
	Limit  int
}

type Record struct {
	ID             string         `json:"id"`
	TargetType     string         `json:"target_type"`
	TargetID       string         `json:"target_id"`
	Action         string         `json:"action"`
	RiskLevel      string         `json:"risk_level"`
	Status         string         `json:"status"`
	Decision       string         `json:"decision"`
	RequestedBy    string         `json:"requested_by"`
	RequestReason  string         `json:"request_reason,omitempty"`
	DecidedBy      string         `json:"decided_by,omitempty"`
	DecisionReason string         `json:"decision_reason,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	RequestedAt    string         `json:"requested_at"`
	DecidedAt      string         `json:"decided_at,omitempty"`
}

var (
	credentialPattern     = regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret|credential|private[_-]?key)\s*[:=]\s*[^,\s]+`)
	openAIKeyPattern      = regexp.MustCompile(`sk-[A-Za-z0-9_-]{8,}`)
	privateKeyPattern     = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`)
	sensitiveFieldPattern = regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret|credential|private[_-]?key)`)
	validIDPattern        = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	validDecision         = map[string]bool{"approved": true, "rejected": true}
	errInvalidApprovalID  = errors.New("approval_id_invalid")
)

func Request(rootDir string, options RequestOptions) (Record, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Record{}, err
	}
	options.TargetType = normalizeToken(options.TargetType)
	options.TargetID = strings.TrimSpace(options.TargetID)
	options.Action = normalizeAction(options.Action)
	options.RiskLevel = normalizeRisk(options.RiskLevel)
	options.RequestedBy = normalizeActor(options.RequestedBy)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.TargetType == "" {
		return Record{}, errors.New("approval_target_type_required")
	}
	if options.TargetID == "" {
		return Record{}, errors.New("approval_target_id_required")
	}
	if options.Action == "" {
		return Record{}, errors.New("approval_action_required")
	}
	if containsSensitive(options.Reason) || containsSensitiveValue(options.Metadata) {
		return Record{}, errors.New("approval_payload_must_not_contain_secret")
	}
	now := time.Now().UTC()
	record := Record{
		ID:            "approval-" + textutil.Slugify(options.Action+"-"+options.TargetID) + "-" + now.Format("20060102150405"),
		TargetType:    options.TargetType,
		TargetID:      options.TargetID,
		Action:        options.Action,
		RiskLevel:     options.RiskLevel,
		Status:        "pending",
		Decision:      "APPROVAL_PENDING",
		RequestedBy:   options.RequestedBy,
		RequestReason: options.Reason,
		Metadata:      safeMetadata(options.Metadata),
		RequestedAt:   now.Format(time.RFC3339Nano),
	}
	if err := save(rootDir, record); err != nil {
		return Record{}, err
	}
	_ = logging.Log(rootDir, "audit", "approval.requested", map[string]any{
		"approval_id":  record.ID,
		"target_type":  record.TargetType,
		"target_id":    record.TargetID,
		"action":       record.Action,
		"risk_level":   record.RiskLevel,
		"requested_by": record.RequestedBy,
	})
	return record, nil
}

func Decide(rootDir string, id string, options DecisionOptions) (Record, bool, error) {
	record, found, err := Load(rootDir, id)
	if err != nil || !found {
		return Record{}, found, err
	}
	if record.Status != "pending" {
		return Record{}, true, errors.New("approval_already_decided")
	}
	decision := normalizeToken(options.Decision)
	if !validDecision[decision] {
		return Record{}, true, errors.New("approval_decision_must_be_approved_or_rejected")
	}
	reason := strings.TrimSpace(options.Reason)
	if containsSensitive(reason) {
		return Record{}, true, errors.New("approval_payload_must_not_contain_secret")
	}
	record.Status = decision
	record.Decision = "APPROVAL_" + strings.ToUpper(decision)
	record.DecidedBy = normalizeActor(options.DecidedBy)
	record.DecisionReason = reason
	record.DecidedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := save(rootDir, record); err != nil {
		return Record{}, true, err
	}
	_ = logging.Log(rootDir, "audit", "approval.decided", map[string]any{
		"approval_id": record.ID,
		"target_type": record.TargetType,
		"target_id":   record.TargetID,
		"action":      record.Action,
		"decision":    record.Decision,
		"decided_by":  record.DecidedBy,
	})
	return record, true, nil
}

func Load(rootDir string, id string) (Record, bool, error) {
	id, err := validateID(id)
	if err != nil {
		return Record{}, false, err
	}
	var record Record
	found, err := fsutil.ReadJSON(recordPath(rootDir, id), &record)
	return record, found, err
}

func IsInvalidIDError(err error) bool {
	return errors.Is(err, errInvalidApprovalID)
}

func List(rootDir string, options ListOptions) ([]Record, error) {
	if err := fsutil.EnsureDir(dir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir(rootDir))
	if err != nil {
		return nil, err
	}
	status := normalizeToken(options.Status)
	records := []Record{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var record Record
		found, err := fsutil.ReadJSON(filepath.Join(dir(rootDir), entry.Name()), &record)
		if err != nil {
			return nil, err
		}
		if !found || record.ID == "" {
			continue
		}
		if status != "" && record.Status != status {
			continue
		}
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].RequestedAt > records[j].RequestedAt
	})
	limit := options.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if len(records) > limit {
		return records[:limit], nil
	}
	return records, nil
}

func save(rootDir string, record Record) error {
	if err := fsutil.WriteJSON(recordPath(rootDir, record.ID), record); err != nil {
		return err
	}
	return fsutil.AppendJSONL(filepath.Join(dir(rootDir), "approvals.jsonl"), record)
}

func dir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).LifecycleDir, "approvals")
}

func recordPath(rootDir string, id string) string {
	return filepath.Join(dir(rootDir), id+".json")
}

func normalizeAction(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", ".")
	value = strings.ReplaceAll(value, "_", ".")
	return strings.Trim(value, ".")
}

func normalizeRisk(value string) string {
	value = normalizeToken(value)
	switch value {
	case "low", "medium", "high", "critical":
		return value
	default:
		return "high"
	}
}

func normalizeActor(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "system"
	}
	return value
}

func validateID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || !validIDPattern.MatchString(value) {
		return "", errInvalidApprovalID
	}
	return value, nil
}

func normalizeToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func safeMetadata(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}
	out := map[string]any{}
	for key, item := range value {
		out[key] = item
	}
	return out
}

func containsSensitiveValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return containsSensitive(typed)
	case map[string]any:
		for key, item := range typed {
			if sensitiveFieldPattern.MatchString(key) {
				text, ok := item.(string)
				if !ok || !isSecretReference(text) {
					return true
				}
			}
			if containsSensitiveValue(item) {
				return true
			}
		}
		return false
	case []any:
		for _, item := range typed {
			if containsSensitiveValue(item) {
				return true
			}
		}
		return false
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return true
		}
		return containsSensitive(string(data))
	}
}

func containsSensitive(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return credentialPattern.MatchString(value) || openAIKeyPattern.MatchString(value) || privateKeyPattern.MatchString(value)
}

func isSecretReference(value string) bool {
	return strings.HasPrefix(value, "env:") || strings.HasPrefix(value, "secret:")
}
