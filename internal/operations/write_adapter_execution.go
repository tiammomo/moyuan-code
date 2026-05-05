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

type WriteAdapterExecutionOptions struct {
	ExecutionPlanID string `json:"execution_plan_id,omitempty"`
	Mode            string `json:"mode,omitempty"`
	AdapterID       string `json:"adapter_id,omitempty"`
	Status          string `json:"status,omitempty"`
	Decision        string `json:"decision,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type WriteAdapterExecutionReport struct {
	ID          string                       `json:"id"`
	GeneratedAt string                       `json:"generated_at"`
	Filters     WriteAdapterExecutionOptions `json:"filters"`
	Summary     WriteAdapterExecutionSummary `json:"summary"`
	Executions  []WriteAdapterExecution      `json:"executions"`
}

type WriteAdapterExecutionSummary struct {
	ExecutionCount       int            `json:"execution_count"`
	CompletedCount       int            `json:"completed_count"`
	BlockedCount         int            `json:"blocked_count"`
	ManualRequiredCount  int            `json:"manual_required_count"`
	ExternalAttemptCount int            `json:"external_attempt_count"`
	ExternalWriteCount   int            `json:"external_write_count"`
	ByAdapter            map[string]int `json:"by_adapter,omitempty"`
	ByMode               map[string]int `json:"by_mode,omitempty"`
	ByStatus             map[string]int `json:"by_status,omitempty"`
	ByDecision           map[string]int `json:"by_decision,omitempty"`
}

type WriteAdapterExecution struct {
	ID                     string                    `json:"id"`
	ExecutionPlanID        string                    `json:"execution_plan_id,omitempty"`
	ReviewPacketID         string                    `json:"review_packet_id,omitempty"`
	OperationType          string                    `json:"operation_type,omitempty"`
	OperationID            string                    `json:"operation_id,omitempty"`
	Provider               string                    `json:"provider,omitempty"`
	Environment            string                    `json:"environment,omitempty"`
	AdapterID              string                    `json:"adapter_id,omitempty"`
	Mode                   string                    `json:"mode"`
	Status                 string                    `json:"status"`
	Decision               string                    `json:"decision"`
	Reasons                []string                  `json:"reasons,omitempty"`
	RuleRefs               []string                  `json:"rule_refs,omitempty"`
	EvidenceRefs           []string                  `json:"evidence_refs,omitempty"`
	GuardResults           []WriteAdapterGuardResult `json:"guard_results,omitempty"`
	ApplyAllowed           bool                      `json:"apply_allowed"`
	ExternalWriteAttempted bool                      `json:"external_write_attempted"`
	ExternalWritePerformed bool                      `json:"external_write_performed"`
	CreatedAt              string                    `json:"created_at"`
	FinishedAt             string                    `json:"finished_at,omitempty"`
	Metadata               map[string]any            `json:"metadata,omitempty"`
}

type WriteAdapterGuardResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

func CreateWriteAdapterExecutions(rootDir string, options WriteAdapterExecutionOptions) (WriteAdapterExecutionReport, error) {
	options = normalizeWriteAdapterExecutionOptions(options)
	if options.Mode == "" {
		options.Mode = "preview"
	}
	execution, err := writeAdapterExecutionFromPlan(rootDir, options)
	if err != nil {
		return WriteAdapterExecutionReport{}, err
	}
	if writeAdapterExecutionMatches(execution, options) {
		execution, err = finishWriteAdapterExecution(rootDir, execution)
		if err != nil {
			return WriteAdapterExecutionReport{}, err
		}
	} else {
		execution = WriteAdapterExecution{}
	}
	executions := []WriteAdapterExecution{}
	if execution.ID != "" {
		executions = append(executions, execution)
	}
	now := time.Now().UTC()
	report := WriteAdapterExecutionReport{
		ID:          "write-adapter-execution-report-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Executions:  executions,
	}
	report.Summary = buildWriteAdapterExecutionSummary(executions)
	return report, nil
}

func ListWriteAdapterExecutions(rootDir string, options WriteAdapterExecutionOptions) (WriteAdapterExecutionReport, error) {
	options = normalizeWriteAdapterExecutionOptions(options)
	if err := fsutil.EnsureDir(writeAdapterExecutionDir(rootDir)); err != nil {
		return WriteAdapterExecutionReport{}, err
	}
	entries, err := os.ReadDir(writeAdapterExecutionDir(rootDir))
	if err != nil {
		return WriteAdapterExecutionReport{}, err
	}
	executions := []WriteAdapterExecution{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var execution WriteAdapterExecution
		found, err := fsutil.ReadJSON(filepath.Join(writeAdapterExecutionDir(rootDir), entry.Name()), &execution)
		if err != nil {
			return WriteAdapterExecutionReport{}, err
		}
		if found && execution.ID != "" && writeAdapterExecutionMatches(execution, options) {
			executions = append(executions, execution)
		}
	}
	sortWriteAdapterExecutions(executions)
	if len(executions) > options.Limit {
		executions = executions[:options.Limit]
	}
	now := time.Now().UTC()
	report := WriteAdapterExecutionReport{
		ID:          "write-adapter-execution-list-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Executions:  executions,
	}
	report.Summary = buildWriteAdapterExecutionSummary(executions)
	return report, nil
}

func LoadWriteAdapterExecution(rootDir string, id string) (WriteAdapterExecution, bool, error) {
	var execution WriteAdapterExecution
	found, err := fsutil.ReadJSON(writeAdapterExecutionPath(rootDir, strings.TrimSpace(id)), &execution)
	return execution, found, err
}

func writeAdapterExecutionFromPlan(rootDir string, options WriteAdapterExecutionOptions) (WriteAdapterExecution, error) {
	now := time.Now().UTC()
	execution := WriteAdapterExecution{
		ID:                     "write-adapter-execution-" + firstNonEmpty(options.ExecutionPlanID, "missing-plan") + "-" + options.Mode + "-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		ExecutionPlanID:        options.ExecutionPlanID,
		Mode:                   options.Mode,
		Status:                 "completed",
		Decision:               "WRITE_ADAPTER_PREVIEW_READY",
		Reasons:                []string{"write_adapter_execution_created"},
		RuleRefs:               []string{"write_adapter_dispatch_scaffold_no_external_write"},
		EvidenceRefs:           []string{},
		GuardResults:           []WriteAdapterGuardResult{},
		ApplyAllowed:           false,
		ExternalWriteAttempted: false,
		ExternalWritePerformed: false,
		CreatedAt:              now.Format(time.RFC3339Nano),
		Metadata: map[string]any{
			"external_write_attempted": false,
			"external_write_performed": false,
			"adapter_switch_env":       "MOYUAN_ENABLE_WRITE_ADAPTERS",
		},
	}
	block := func(decision string, reason string, rule string) {
		execution.Status = "blocked"
		execution.Decision = decision
		execution.Reasons = appendUnique(execution.Reasons, reason)
		execution.RuleRefs = appendUnique(execution.RuleRefs, rule)
	}
	manual := func(decision string, reason string, rule string) {
		if execution.Status != "blocked" {
			execution.Status = "manual_required"
			execution.Decision = decision
		}
		execution.Reasons = appendUnique(execution.Reasons, reason)
		execution.RuleRefs = appendUnique(execution.RuleRefs, rule)
	}

	if options.Mode != "preview" && options.Mode != "apply" {
		block("WRITE_ADAPTER_MODE_UNSUPPORTED", "write_adapter_mode_unsupported:"+options.Mode, "write_adapter_mode_must_be_preview_or_apply")
		execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return normalizeWriteAdapterExecution(execution), nil
	}
	if options.ExecutionPlanID == "" {
		block("WRITE_ADAPTER_EXECUTION_PLAN_REQUIRED", "write_execution_plan_id_required", "write_execution_plan_required")
		execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return normalizeWriteAdapterExecution(execution), nil
	}
	plan, found, err := LoadWriteExecutionPlan(rootDir, options.ExecutionPlanID)
	if err != nil {
		return WriteAdapterExecution{}, err
	}
	if !found {
		block("WRITE_ADAPTER_EXECUTION_PLAN_MISSING", "write_execution_plan_missing:"+options.ExecutionPlanID, "write_execution_plan_must_exist")
		execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return normalizeWriteAdapterExecution(execution), nil
	}

	execution.ReviewPacketID = plan.ReviewPacketID
	execution.OperationType = plan.OperationType
	execution.OperationID = plan.OperationID
	execution.Provider = plan.Provider
	execution.Environment = plan.Environment
	execution.AdapterID = firstNonEmpty(options.AdapterID, deriveWriteAdapterID(plan))
	execution.RuleRefs = appendUniqueStrings(execution.RuleRefs, plan.RuleRefs...)
	execution.EvidenceRefs = appendUniqueStrings(execution.EvidenceRefs, plan.EvidenceRefs...)
	execution.Metadata["execution_plan_status"] = plan.Status
	execution.Metadata["execution_plan_decision"] = plan.Decision

	addGuard := func(name string, status string, decision string, reason string) {
		execution.GuardResults = append(execution.GuardResults, WriteAdapterGuardResult{Name: name, Status: status, Decision: decision, Reason: reason})
	}
	if execution.AdapterID == "" {
		addGuard("adapter_resolution", "blocked", "WRITE_ADAPTER_UNSUPPORTED", "adapter_not_resolved")
		block("WRITE_ADAPTER_UNSUPPORTED", "write_adapter_not_resolved", "write_adapter_supported_operation_required")
		execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return normalizeWriteAdapterExecution(execution), nil
	}
	addGuard("adapter_resolution", "ready", "WRITE_ADAPTER_RESOLVED", execution.AdapterID)
	if plan.Status == "blocked" {
		addGuard("execution_plan_status", "blocked", "WRITE_ADAPTER_PLAN_BLOCKED", plan.Decision)
		block("WRITE_ADAPTER_PLAN_BLOCKED", "write_execution_plan_blocked:"+plan.Decision, "write_execution_plan_must_be_dispatchable")
	}
	if plan.Status == "manual_required" {
		addGuard("execution_plan_status", "manual_required", "WRITE_ADAPTER_PLAN_MANUAL_REQUIRED", plan.Decision)
		manual("WRITE_ADAPTER_PLAN_MANUAL_REQUIRED", "write_execution_plan_manual_required:"+plan.Decision, "write_execution_plan_must_be_dispatchable")
	}
	if options.Mode == "preview" && execution.Status == "completed" {
		addGuard("external_write", "ready", "WRITE_ADAPTER_PREVIEW_NO_EXTERNAL_WRITE", "preview_mode")
		execution.Reasons = appendUnique(execution.Reasons, "write_adapter_preview_ready")
		execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return normalizeWriteAdapterExecution(execution), nil
	}
	if options.Mode == "apply" {
		if plan.Status != "ready" || !plan.ApplyAllowed {
			addGuard("apply_plan", "manual_required", "WRITE_ADAPTER_APPLY_PLAN_NOT_READY", plan.Decision)
			manual("WRITE_ADAPTER_APPLY_PLAN_NOT_READY", "write_execution_plan_not_apply_ready:"+plan.Decision, "apply_requires_ready_execution_plan")
		}
		if os.Getenv("MOYUAN_ENABLE_WRITE_ADAPTERS") != "1" {
			addGuard("adapter_switch", "blocked", "WRITE_ADAPTER_SWITCH_DISABLED", "MOYUAN_ENABLE_WRITE_ADAPTERS")
			block("WRITE_ADAPTER_SWITCH_DISABLED", "write_adapter_switch_disabled", "write_adapter_requires_explicit_switch")
		}
		if execution.Status == "completed" {
			addGuard("adapter_implementation", "manual_required", "WRITE_ADAPTER_IMPLEMENTATION_REQUIRED", execution.AdapterID)
			manual("WRITE_ADAPTER_IMPLEMENTATION_REQUIRED", "real_write_adapter_not_implemented:"+execution.AdapterID, "real_adapter_implementation_required")
		}
	}
	execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return normalizeWriteAdapterExecution(execution), nil
}

func deriveWriteAdapterID(plan WriteExecutionPlan) string {
	switch normalizeType(plan.OperationType) {
	case "release_provider_execution":
		switch normalizeType(plan.Provider) {
		case "github":
			return "github_release_provider_adapter"
		case "gitee":
			return "gitee_release_provider_adapter"
		case "gitlab":
			return "gitlab_release_provider_adapter"
		default:
			return "generic_release_provider_adapter"
		}
	case "deployment_execution":
		switch normalizeType(plan.Provider) {
		case "ssh", "local_vm":
			return "ssh_deployment_adapter"
		case "cloud", "aliyun", "tencent_cloud":
			return normalizeType(plan.Provider) + "_deployment_adapter"
		default:
			return "generic_deployment_adapter"
		}
	case "resource_maintenance":
		return "server_resource_registry_adapter"
	default:
		return ""
	}
}

func normalizeWriteAdapterExecutionOptions(options WriteAdapterExecutionOptions) WriteAdapterExecutionOptions {
	options.ExecutionPlanID = strings.TrimSpace(options.ExecutionPlanID)
	options.Mode = normalizeType(options.Mode)
	options.AdapterID = normalizeType(options.AdapterID)
	options.Status = normalizeType(options.Status)
	options.Decision = normalizeType(options.Decision)
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	return options
}

func normalizeWriteAdapterExecution(execution WriteAdapterExecution) WriteAdapterExecution {
	execution.OperationType = normalizeType(execution.OperationType)
	execution.Provider = normalizeType(execution.Provider)
	execution.Environment = normalizeType(execution.Environment)
	execution.AdapterID = normalizeType(execution.AdapterID)
	execution.Mode = normalizeType(execution.Mode)
	execution.Status = normalizeType(execution.Status)
	execution.RuleRefs = compactStrings(execution.RuleRefs)
	execution.EvidenceRefs = compactStrings(execution.EvidenceRefs)
	return execution
}

func writeAdapterExecutionMatches(execution WriteAdapterExecution, options WriteAdapterExecutionOptions) bool {
	if options.ExecutionPlanID != "" && execution.ExecutionPlanID != options.ExecutionPlanID {
		return false
	}
	if options.Mode != "" && normalizeType(execution.Mode) != options.Mode {
		return false
	}
	if options.AdapterID != "" && normalizeType(execution.AdapterID) != options.AdapterID {
		return false
	}
	if options.Status != "" && normalizeType(execution.Status) != options.Status {
		return false
	}
	if options.Decision != "" && normalizeType(execution.Decision) != options.Decision {
		return false
	}
	return true
}

func buildWriteAdapterExecutionSummary(executions []WriteAdapterExecution) WriteAdapterExecutionSummary {
	summary := WriteAdapterExecutionSummary{
		ExecutionCount: len(executions),
		ByAdapter:      map[string]int{},
		ByMode:         map[string]int{},
		ByStatus:       map[string]int{},
		ByDecision:     map[string]int{},
	}
	for _, execution := range executions {
		if execution.AdapterID != "" {
			summary.ByAdapter[execution.AdapterID]++
		}
		summary.ByMode[execution.Mode]++
		summary.ByStatus[execution.Status]++
		summary.ByDecision[execution.Decision]++
		if execution.ExternalWriteAttempted {
			summary.ExternalAttemptCount++
		}
		if execution.ExternalWritePerformed {
			summary.ExternalWriteCount++
		}
		switch execution.Status {
		case "completed":
			summary.CompletedCount++
		case "blocked":
			summary.BlockedCount++
		case "manual_required":
			summary.ManualRequiredCount++
		}
	}
	return summary
}

func sortWriteAdapterExecutions(executions []WriteAdapterExecution) {
	sort.SliceStable(executions, func(i, j int) bool {
		left := parseTimelineTime(executions[i].CreatedAt)
		right := parseTimelineTime(executions[j].CreatedAt)
		if !left.Equal(right) {
			return left.After(right)
		}
		return executions[i].ID > executions[j].ID
	})
}

func finishWriteAdapterExecution(rootDir string, execution WriteAdapterExecution) (WriteAdapterExecution, error) {
	execution = normalizeWriteAdapterExecution(execution)
	if err := fsutil.EnsureDir(writeAdapterExecutionDir(rootDir)); err != nil {
		return WriteAdapterExecution{}, err
	}
	if err := fsutil.WriteJSON(writeAdapterExecutionPath(rootDir, execution.ID), execution); err != nil {
		return WriteAdapterExecution{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "write-adapter-executions.jsonl"), execution); err != nil {
		return WriteAdapterExecution{}, err
	}
	_ = logging.Log(rootDir, "release", "operations.write_adapter_execution.created", map[string]any{
		"adapter_execution_id":     execution.ID,
		"execution_plan_id":        execution.ExecutionPlanID,
		"adapter_id":               execution.AdapterID,
		"mode":                     execution.Mode,
		"status":                   execution.Status,
		"decision":                 execution.Decision,
		"external_write_attempted": execution.ExternalWriteAttempted,
		"external_write_performed": execution.ExternalWritePerformed,
	})
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "write_adapter_execution",
		ParentID:    execution.ID,
		SubjectType: firstNonEmpty(execution.OperationType, "write_adapter_execution"),
		SubjectID:   firstNonEmpty(execution.OperationID, execution.ID),
		Operation:   "operations.write_adapter_execution.create",
		Status:      execution.Status,
		Decision:    execution.Decision,
		Reasons:     execution.Reasons,
		Source:      "operations",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "write_adapter_execution",
			ID:   execution.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "write-adapter-executions", execution.ID+".json")),
		}},
	}); err != nil {
		return WriteAdapterExecution{}, err
	}
	return execution, nil
}

func writeAdapterExecutionDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "write-adapter-executions")
}

func writeAdapterExecutionPath(rootDir string, id string) string {
	return filepath.Join(writeAdapterExecutionDir(rootDir), id+".json")
}
