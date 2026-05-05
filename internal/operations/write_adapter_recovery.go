package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type WriteAdapterRecoveryOptions struct {
	ExecutionID     string `json:"execution_id,omitempty"`
	ExecutionPlanID string `json:"execution_plan_id,omitempty"`
	AdapterID       string `json:"adapter_id,omitempty"`
	Status          string `json:"status,omitempty"`
	Decision        string `json:"decision,omitempty"`
	FailureClass    string `json:"failure_class,omitempty"`
	RecoveryAction  string `json:"recovery_action,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type WriteAdapterRecoveryReport struct {
	ID          string                       `json:"id"`
	GeneratedAt string                       `json:"generated_at"`
	Filters     WriteAdapterRecoveryOptions  `json:"filters"`
	Summary     WriteAdapterRecoverySummary  `json:"summary"`
	Recoveries  []WriteAdapterRecoveryRecord `json:"recoveries"`
}

type WriteAdapterRecoverySummary struct {
	RecoveryCount int            `json:"recovery_count"`
	OpenCount     int            `json:"open_count"`
	RepairCount   int            `json:"repair_count"`
	RetryCount    int            `json:"retry_count"`
	HandoffCount  int            `json:"handoff_count"`
	ByAdapter     map[string]int `json:"by_adapter,omitempty"`
	ByStatus      map[string]int `json:"by_status,omitempty"`
	ByDecision    map[string]int `json:"by_decision,omitempty"`
	ByFailure     map[string]int `json:"by_failure,omitempty"`
	ByAction      map[string]int `json:"by_action,omitempty"`
}

type WriteAdapterRecoveryRecord struct {
	ID                     string         `json:"id"`
	ExecutionID            string         `json:"execution_id"`
	ExecutionPlanID        string         `json:"execution_plan_id,omitempty"`
	ReviewPacketID         string         `json:"review_packet_id,omitempty"`
	OperationType          string         `json:"operation_type,omitempty"`
	OperationID            string         `json:"operation_id,omitempty"`
	Provider               string         `json:"provider,omitempty"`
	Environment            string         `json:"environment,omitempty"`
	AdapterID              string         `json:"adapter_id,omitempty"`
	Mode                   string         `json:"mode,omitempty"`
	SourceStatus           string         `json:"source_status"`
	SourceDecision         string         `json:"source_decision"`
	Status                 string         `json:"status"`
	Decision               string         `json:"decision"`
	FailureClass           string         `json:"failure_class"`
	RecoveryAction         string         `json:"recovery_action"`
	RepairAllowed          bool           `json:"repair_allowed"`
	RetryAllowed           bool           `json:"retry_allowed"`
	HandoffRequired        bool           `json:"handoff_required"`
	ReviewRequired         bool           `json:"review_required"`
	Reasons                []string       `json:"reasons,omitempty"`
	RuleRefs               []string       `json:"rule_refs,omitempty"`
	EvidenceRefs           []string       `json:"evidence_refs,omitempty"`
	ExternalWriteAttempted bool           `json:"external_write_attempted"`
	ExternalWritePerformed bool           `json:"external_write_performed"`
	CreatedAt              string         `json:"created_at"`
	Metadata               map[string]any `json:"metadata,omitempty"`
}

func ListWriteAdapterRecoveries(rootDir string, options WriteAdapterRecoveryOptions) (WriteAdapterRecoveryReport, error) {
	options = normalizeWriteAdapterRecoveryOptions(options)
	if err := fsutil.EnsureDir(writeAdapterRecoveryDir(rootDir)); err != nil {
		return WriteAdapterRecoveryReport{}, err
	}
	entries, err := os.ReadDir(writeAdapterRecoveryDir(rootDir))
	if err != nil {
		return WriteAdapterRecoveryReport{}, err
	}
	recoveries := []WriteAdapterRecoveryRecord{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var record WriteAdapterRecoveryRecord
		found, err := fsutil.ReadJSON(filepath.Join(writeAdapterRecoveryDir(rootDir), entry.Name()), &record)
		if err != nil {
			return WriteAdapterRecoveryReport{}, err
		}
		if found && record.ID != "" && writeAdapterRecoveryMatches(record, options) {
			recoveries = append(recoveries, record)
		}
	}
	sortWriteAdapterRecoveries(recoveries)
	if len(recoveries) > options.Limit {
		recoveries = recoveries[:options.Limit]
	}
	now := time.Now().UTC()
	report := WriteAdapterRecoveryReport{
		ID:          "write-adapter-recovery-list-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Recoveries:  recoveries,
	}
	report.Summary = buildWriteAdapterRecoverySummary(recoveries)
	return report, nil
}

func LoadWriteAdapterRecovery(rootDir string, id string) (WriteAdapterRecoveryRecord, bool, error) {
	var record WriteAdapterRecoveryRecord
	found, err := fsutil.ReadJSON(writeAdapterRecoveryPath(rootDir, strings.TrimSpace(id)), &record)
	return record, found, err
}

func recordWriteAdapterRecovery(rootDir string, execution WriteAdapterExecution) (WriteAdapterRecoveryRecord, error) {
	if !writeAdapterExecutionNeedsRecovery(execution) {
		return WriteAdapterRecoveryRecord{}, nil
	}
	record := writeAdapterRecoveryFromExecution(execution)
	if err := fsutil.EnsureDir(writeAdapterRecoveryDir(rootDir)); err != nil {
		return WriteAdapterRecoveryRecord{}, err
	}
	if err := fsutil.WriteJSON(writeAdapterRecoveryPath(rootDir, record.ID), record); err != nil {
		return WriteAdapterRecoveryRecord{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "write-adapter-recoveries.jsonl"), record); err != nil {
		return WriteAdapterRecoveryRecord{}, err
	}
	_ = logging.Log(rootDir, "release", "operations.write_adapter_recovery.recorded", map[string]any{
		"recovery_id":     record.ID,
		"execution_id":    record.ExecutionID,
		"adapter_id":      record.AdapterID,
		"failure_class":   record.FailureClass,
		"recovery_action": record.RecoveryAction,
		"decision":        record.Decision,
	})
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "write_adapter_recovery",
		ParentID:    record.ID,
		SubjectType: firstNonEmpty(record.OperationType, "write_adapter_execution"),
		SubjectID:   firstNonEmpty(record.OperationID, record.ExecutionID),
		Operation:   "operations.write_adapter_recovery.record",
		Status:      record.Status,
		Decision:    record.Decision,
		Reasons:     record.Reasons,
		Source:      "operations",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "write_adapter_recovery",
			ID:   record.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "write-adapter-recoveries", record.ID+".json")),
		}},
	}); err != nil {
		return WriteAdapterRecoveryRecord{}, err
	}
	return record, nil
}

func writeAdapterExecutionNeedsRecovery(execution WriteAdapterExecution) bool {
	switch execution.Status {
	case "blocked", "manual_required", "failed":
		return true
	default:
		return false
	}
}

func writeAdapterRecoveryFromExecution(execution WriteAdapterExecution) WriteAdapterRecoveryRecord {
	now := time.Now().UTC()
	failureClass, action, decision, repairAllowed, retryAllowed, handoffRequired := classifyWriteAdapterRecovery(execution)
	record := WriteAdapterRecoveryRecord{
		ID:                     "write-adapter-recovery-" + execution.ID,
		ExecutionID:            execution.ID,
		ExecutionPlanID:        execution.ExecutionPlanID,
		ReviewPacketID:         execution.ReviewPacketID,
		OperationType:          execution.OperationType,
		OperationID:            execution.OperationID,
		Provider:               execution.Provider,
		Environment:            execution.Environment,
		AdapterID:              execution.AdapterID,
		Mode:                   execution.Mode,
		SourceStatus:           execution.Status,
		SourceDecision:         execution.Decision,
		Status:                 "open",
		Decision:               decision,
		FailureClass:           failureClass,
		RecoveryAction:         action,
		RepairAllowed:          repairAllowed,
		RetryAllowed:           retryAllowed,
		HandoffRequired:        handoffRequired,
		ReviewRequired:         execution.Status == "manual_required" || handoffRequired,
		Reasons:                append([]string{}, execution.Reasons...),
		RuleRefs:               appendUnique(append([]string{}, execution.RuleRefs...), "write_adapter_failure_recovery_record"),
		EvidenceRefs:           append([]string{}, execution.EvidenceRefs...),
		ExternalWriteAttempted: execution.ExternalWriteAttempted,
		ExternalWritePerformed: execution.ExternalWritePerformed,
		CreatedAt:              now.Format(time.RFC3339Nano),
		Metadata: map[string]any{
			"source_status":             execution.Status,
			"source_decision":           execution.Decision,
			"guard_count":               len(execution.GuardResults),
			"sandbox_result_count":      len(execution.SandboxResults),
			"rollback_binding_decision": execution.RollbackBinding.Decision,
		},
	}
	return normalizeWriteAdapterRecovery(record)
}

func classifyWriteAdapterRecovery(execution WriteAdapterExecution) (string, string, string, bool, bool, bool) {
	combined := strings.ToLower(strings.Join(append(append([]string{execution.Decision}, execution.Reasons...), execution.RuleRefs...), " "))
	switch {
	case strings.Contains(combined, "ssh_sandbox") || strings.Contains(combined, "command_blocked") || strings.Contains(combined, "command_not_allowed"):
		return "ssh_sandbox_blocked", "repair_remote_command_or_target", "WRITE_ADAPTER_RECOVERY_REPAIR_RECOMMENDED", true, false, false
	case strings.Contains(combined, "rollback"):
		return "rollback_binding_blocked", "repair_rollback_plan_or_runbook", "WRITE_ADAPTER_RECOVERY_REPAIR_RECOMMENDED", true, false, false
	case strings.Contains(combined, "switch_disabled"):
		return "adapter_switch_disabled", "enable_adapter_switch_after_approval", "WRITE_ADAPTER_RECOVERY_RETRY_GATED", false, true, true
	case strings.Contains(combined, "execution_plan_missing") || strings.Contains(combined, "execution_plan_required"):
		return "execution_plan_missing", "repair_execution_plan_reference", "WRITE_ADAPTER_RECOVERY_REPAIR_RECOMMENDED", true, false, false
	case strings.Contains(combined, "plan_blocked") || strings.Contains(combined, "plan_manual_required") || strings.Contains(combined, "apply_plan_not_ready"):
		return "write_execution_plan_not_dispatchable", "repair_or_recreate_write_execution_plan", "WRITE_ADAPTER_RECOVERY_REPAIR_RECOMMENDED", true, false, false
	case strings.Contains(combined, "implementation_required") || strings.Contains(combined, "not_implemented"):
		return "adapter_implementation_missing", "handoff_to_adapter_owner", "WRITE_ADAPTER_RECOVERY_HANDOFF_REQUIRED", false, false, true
	default:
		return "adapter_execution_not_ready", "manual_handoff", "WRITE_ADAPTER_RECOVERY_HANDOFF_REQUIRED", false, false, true
	}
}

func normalizeWriteAdapterRecoveryOptions(options WriteAdapterRecoveryOptions) WriteAdapterRecoveryOptions {
	options.ExecutionID = strings.TrimSpace(options.ExecutionID)
	options.ExecutionPlanID = strings.TrimSpace(options.ExecutionPlanID)
	options.AdapterID = normalizeType(options.AdapterID)
	options.Status = normalizeType(options.Status)
	options.Decision = normalizeType(options.Decision)
	options.FailureClass = normalizeType(options.FailureClass)
	options.RecoveryAction = normalizeType(options.RecoveryAction)
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	return options
}

func normalizeWriteAdapterRecovery(record WriteAdapterRecoveryRecord) WriteAdapterRecoveryRecord {
	record.OperationType = normalizeType(record.OperationType)
	record.Provider = normalizeType(record.Provider)
	record.Environment = normalizeType(record.Environment)
	record.AdapterID = normalizeType(record.AdapterID)
	record.Mode = normalizeType(record.Mode)
	record.SourceStatus = normalizeType(record.SourceStatus)
	record.Status = normalizeType(record.Status)
	record.FailureClass = normalizeType(record.FailureClass)
	record.RecoveryAction = normalizeType(record.RecoveryAction)
	record.RuleRefs = compactStrings(record.RuleRefs)
	record.EvidenceRefs = compactStrings(record.EvidenceRefs)
	return record
}

func writeAdapterRecoveryMatches(record WriteAdapterRecoveryRecord, options WriteAdapterRecoveryOptions) bool {
	if options.ExecutionID != "" && record.ExecutionID != options.ExecutionID {
		return false
	}
	if options.ExecutionPlanID != "" && record.ExecutionPlanID != options.ExecutionPlanID {
		return false
	}
	if options.AdapterID != "" && normalizeType(record.AdapterID) != options.AdapterID {
		return false
	}
	if options.Status != "" && normalizeType(record.Status) != options.Status {
		return false
	}
	if options.Decision != "" && normalizeType(record.Decision) != options.Decision {
		return false
	}
	if options.FailureClass != "" && normalizeType(record.FailureClass) != options.FailureClass {
		return false
	}
	if options.RecoveryAction != "" && normalizeType(record.RecoveryAction) != options.RecoveryAction {
		return false
	}
	return true
}

func buildWriteAdapterRecoverySummary(records []WriteAdapterRecoveryRecord) WriteAdapterRecoverySummary {
	summary := WriteAdapterRecoverySummary{
		RecoveryCount: len(records),
		ByAdapter:     map[string]int{},
		ByStatus:      map[string]int{},
		ByDecision:    map[string]int{},
		ByFailure:     map[string]int{},
		ByAction:      map[string]int{},
	}
	for _, record := range records {
		if record.AdapterID != "" {
			summary.ByAdapter[record.AdapterID]++
		}
		summary.ByStatus[record.Status]++
		summary.ByDecision[record.Decision]++
		summary.ByFailure[record.FailureClass]++
		summary.ByAction[record.RecoveryAction]++
		if record.Status == "open" {
			summary.OpenCount++
		}
		if record.RepairAllowed {
			summary.RepairCount++
		}
		if record.RetryAllowed {
			summary.RetryCount++
		}
		if record.HandoffRequired {
			summary.HandoffCount++
		}
	}
	return summary
}

func sortWriteAdapterRecoveries(records []WriteAdapterRecoveryRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		left := parseTimelineTime(records[i].CreatedAt)
		right := parseTimelineTime(records[j].CreatedAt)
		if !left.Equal(right) {
			return left.After(right)
		}
		return records[i].ID > records[j].ID
	})
}

func writeAdapterRecoveryDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "write-adapter-recoveries")
}

func writeAdapterRecoveryPath(rootDir string, id string) string {
	return filepath.Join(writeAdapterRecoveryDir(rootDir), id+".json")
}
