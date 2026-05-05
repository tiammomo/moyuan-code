package controlloop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"moyuan-code/internal/comprehension"
	"moyuan-code/internal/deployment"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/operations"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/serverresources"
	"moyuan-code/internal/workspace"
)

const (
	StepResourceLifecycleScan       = "resource_lifecycle_scan"
	StepResourceHealthScan          = "resource_health_scan"
	StepProviderOpsRefresh          = "provider_ops_refresh"
	StepProjectComprehensionRefresh = "project_comprehension_refresh"
	StepOperationsAuditExport       = "operations_audit_export"
	StepDecisionLedgerRefresh       = "decision_ledger_refresh"
	StepPostDeploymentVerification  = "post_deployment_verification"
)

type RunOptions struct {
	Trigger               string   `json:"trigger,omitempty"`
	RequestedBy           string   `json:"requested_by,omitempty"`
	IdempotencyKey        string   `json:"idempotency_key,omitempty"`
	RetryBudget           int      `json:"retry_budget,omitempty"`
	RetryAttempt          int      `json:"retry_attempt,omitempty"`
	Steps                 []string `json:"steps,omitempty"`
	MaxSteps              int      `json:"max_steps,omitempty"`
	StepTimeoutMS         int      `json:"step_timeout_ms,omitempty"`
	Environment           string   `json:"environment,omitempty"`
	ResourceIDs           []string `json:"resource_ids,omitempty"`
	DeploymentExecutionID string   `json:"deployment_execution_id,omitempty"`
	MonitorLimit          int      `json:"monitor_limit,omitempty"`
	AuditFormat           string   `json:"audit_format,omitempty"`
	ProviderID            string   `json:"provider_id,omitempty"`
	IncludeDisabled       bool     `json:"include_disabled,omitempty"`
	Probe                 bool     `json:"probe,omitempty"`
	ProbeApproved         bool     `json:"probe_approved,omitempty"`
	ProbeTimeoutMS        int      `json:"probe_timeout_ms,omitempty"`
	ComprehensionSince    string   `json:"comprehension_since,omitempty"`
}

type RunRecord struct {
	ID               string       `json:"id"`
	Status           string       `json:"status"`
	Decision         string       `json:"decision"`
	Trigger          string       `json:"trigger"`
	RequestedBy      string       `json:"requested_by,omitempty"`
	IdempotencyKey   string       `json:"idempotency_key,omitempty"`
	IdempotentReplay bool         `json:"idempotent_replay,omitempty"`
	RetryBudget      int          `json:"retry_budget"`
	RetryAttempt     int          `json:"retry_attempt"`
	MaxSteps         int          `json:"max_steps"`
	StepTimeoutMS    int          `json:"step_timeout_ms"`
	Steps            []StepRecord `json:"steps"`
	Reasons          []string     `json:"reasons,omitempty"`
	StartedAt        string       `json:"started_at"`
	FinishedAt       string       `json:"finished_at"`
}

type StepRecord struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Decision    string                 `json:"decision"`
	Summary     string                 `json:"summary,omitempty"`
	Reasons     []string               `json:"reasons,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Artifacts   []evidence.ArtifactRef `json:"artifacts,omitempty"`
	EvidenceIDs []string               `json:"evidence_ids,omitempty"`
	StartedAt   string                 `json:"started_at"`
	FinishedAt  string                 `json:"finished_at"`
	DurationMS  int64                  `json:"duration_ms"`
}

type idempotencyRecord struct {
	Key       string `json:"key"`
	RunID     string `json:"run_id"`
	CreatedAt string `json:"created_at"`
}

func Run(ctx context.Context, rootDir string, options RunOptions) (RunRecord, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return RunRecord{}, err
	}
	options = normalizeOptions(options)
	if len(options.Steps) > options.MaxSteps {
		return RunRecord{}, errors.New("control_loop_steps_exceed_max")
	}
	if options.IdempotencyKey != "" {
		existing, found, err := loadIdempotentRun(rootDir, options.IdempotencyKey)
		if err != nil {
			return RunRecord{}, err
		}
		if found {
			existing.IdempotentReplay = true
			return existing, nil
		}
	}
	start := time.Now().UTC()
	run := RunRecord{
		ID:             "control-loop-" + timeID(start),
		Status:         "running",
		Decision:       "CONTROL_LOOP_RUNNING",
		Trigger:        options.Trigger,
		RequestedBy:    options.RequestedBy,
		IdempotencyKey: options.IdempotencyKey,
		RetryBudget:    options.RetryBudget,
		RetryAttempt:   options.RetryAttempt,
		MaxSteps:       options.MaxSteps,
		StepTimeoutMS:  options.StepTimeoutMS,
		Steps:          []StepRecord{},
		Reasons:        []string{},
		StartedAt:      start.Format(time.RFC3339Nano),
	}
	if err := writeRunFile(rootDir, run); err != nil {
		return RunRecord{}, err
	}
	if options.IdempotencyKey != "" {
		if err := saveIdempotency(rootDir, options.IdempotencyKey, run.ID, run.StartedAt); err != nil {
			return RunRecord{}, err
		}
	}
	for idx, stepType := range options.Steps {
		step := runStep(ctx, rootDir, run.ID, idx+1, stepType, options)
		run.Steps = append(run.Steps, step)
		if step.Status == "failed" {
			run.Status = "failed"
			run.Decision = "CONTROL_LOOP_FAILED"
			run.Reasons = append(run.Reasons, step.Error)
		}
		if step.Status == "attention_required" || step.Status == "blocked" || step.Status == "skipped" {
			run.Reasons = append(run.Reasons, step.Reasons...)
		}
	}
	finished := time.Now().UTC()
	run.FinishedAt = finished.Format(time.RFC3339Nano)
	if run.Status == "failed" && retryBudgetExhausted(run) {
		run.Status = "manual_required"
		run.Decision = "CONTROL_RUNNER_RETRY_BUDGET_EXHAUSTED"
		run.Reasons = append(run.Reasons, "retry_budget_exhausted")
	} else if run.Status != "failed" {
		run.Status = "completed"
		run.Decision = "CONTROL_LOOP_COMPLETED"
		if hasAttention(run.Steps) {
			run.Decision = "CONTROL_LOOP_COMPLETED_WITH_ATTENTION"
		}
	}
	if err := save(rootDir, run); err != nil {
		return RunRecord{}, err
	}
	_ = logging.Log(rootDir, "audit", "control_loop.completed", map[string]any{
		"run_id":   run.ID,
		"decision": run.Decision,
		"status":   run.Status,
		"steps":    len(run.Steps),
		"trigger":  run.Trigger,
	})
	return run, nil
}

func Load(rootDir string, id string) (RunRecord, bool, error) {
	id, ok := cleanID(id)
	if !ok {
		return RunRecord{}, false, nil
	}
	var run RunRecord
	found, err := fsutil.ReadJSON(filepath.Join(runsDir(rootDir), id+".json"), &run)
	return run, found, err
}

func List(rootDir string, limit int) ([]RunRecord, error) {
	lines, err := fsutil.TailLines(runsJSONLPath(rootDir), limit)
	if err != nil {
		return nil, err
	}
	runs := []RunRecord{}
	for _, line := range lines {
		var run RunRecord
		if err := json.Unmarshal([]byte(line), &run); err != nil {
			return nil, err
		}
		if run.ID != "" {
			runs = append(runs, run)
		}
	}
	sort.SliceStable(runs, func(i, j int) bool {
		return runs[i].StartedAt > runs[j].StartedAt
	})
	if limit > 0 && len(runs) > limit {
		return runs[:limit], nil
	}
	return runs, nil
}

func runStep(ctx context.Context, rootDir string, runID string, ordinal int, stepType string, options RunOptions) StepRecord {
	start := time.Now().UTC()
	step := StepRecord{
		ID:        runID + "-step-" + fmt.Sprintf("%02d", ordinal) + "-" + timeID(start),
		Type:      stepType,
		Status:    "running",
		Decision:  "CONTROL_LOOP_STEP_RUNNING",
		Reasons:   []string{},
		Artifacts: []evidence.ArtifactRef{},
		StartedAt: start.Format(time.RFC3339Nano),
	}
	switch stepType {
	case StepResourceLifecycleScan:
		step = runResourceLifecycle(rootDir, step)
	case StepResourceHealthScan:
		step = runResourceHealthScan(ctx, rootDir, step, options)
	case StepProviderOpsRefresh:
		step = runProviderOps(rootDir, step, options)
	case StepProjectComprehensionRefresh:
		step = runComprehension(ctx, rootDir, step, options)
	case StepOperationsAuditExport:
		step = runOperationsAuditExport(rootDir, step, options)
	case StepDecisionLedgerRefresh:
		step = runDecisionLedger(rootDir, step, options)
	case StepPostDeploymentVerification:
		step = runPostDeploymentVerification(rootDir, step, options)
	default:
		step.Status = "failed"
		step.Decision = "CONTROL_LOOP_STEP_FAILED"
		step.Summary = "unsupported control runner step"
		step.Reasons = append(step.Reasons, "unsupported_step:"+stepType)
		step.Error = "unsupported_step:" + stepType
	}
	finishStep(rootDir, runID, &step)
	return step
}

func runResourceLifecycle(rootDir string, step StepRecord) StepRecord {
	report, err := serverresources.LifecycleScan(rootDir)
	if err != nil {
		return failStep(step, err)
	}
	step.Status = "completed"
	if report.Status == "attention_required" {
		step.Status = "attention_required"
	}
	step.Decision = report.Decision
	step.Summary = plural(len(report.Alerts), "resource lifecycle alert")
	step.Reasons = append(step.Reasons, report.Reasons...)
	step.Artifacts = append(step.Artifacts, evidence.ArtifactRef{
		Kind: "resource_lifecycle_scan",
		ID:   report.ID,
		Path: ".moyuan/resources/lifecycle-scans/" + report.ID + ".json",
	})
	return step
}

func runResourceHealthScan(ctx context.Context, rootDir string, step StepRecord, options RunOptions) StepRecord {
	report, err := serverresources.HealthScan(ctx, rootDir, serverresources.HealthScanOptions{
		Environment: options.Environment,
		ResourceIDs: append([]string{}, options.ResourceIDs...),
	})
	if err != nil {
		return failStep(step, err)
	}
	step.Status = "completed"
	if report.Status == "attention_required" || report.Status == "blocked" {
		step.Status = report.Status
	}
	step.Decision = report.Decision
	step.Summary = plural(len(report.Results), "resource health result")
	step.Reasons = append(step.Reasons, report.Reasons...)
	step.Artifacts = append(step.Artifacts, evidence.ArtifactRef{
		Kind: "resource_health_scan",
		ID:   report.ID,
		Path: ".moyuan/resources/checks/" + report.ID + ".json",
	})
	return step
}

func runProviderOps(rootDir string, step StepRecord, options RunOptions) StepRecord {
	result, err := providers.RefreshOps(rootDir, providers.OpsRefreshOptions{
		ProviderID:      options.ProviderID,
		IncludeDisabled: options.IncludeDisabled,
		Probe:           options.Probe,
		ProbeTimeoutMS:  options.ProbeTimeoutMS,
		Approved:        options.ProbeApproved,
	})
	if err != nil {
		return failStep(step, err)
	}
	step.Status = "completed"
	step.Decision = "PROVIDER_OPS_REFRESH_COMPLETED"
	if result.Updated == 0 && result.Skipped > 0 {
		step.Status = "skipped"
		step.Decision = "PROVIDER_OPS_REFRESH_SKIPPED"
	}
	if result.ApprovalID != "" {
		step.Status = "blocked"
		step.Decision = "PROVIDER_OPS_REFRESH_APPROVAL_REQUIRED"
		step.Artifacts = append(step.Artifacts, evidence.ArtifactRef{Kind: "approval", ID: result.ApprovalID})
	}
	step.Summary = plural(result.Updated, "provider update") + ", " + plural(result.Skipped, "provider skip")
	for _, decision := range result.Decisions {
		if decision.Reason != "" {
			step.Reasons = append(step.Reasons, decision.ProviderID+":"+decision.Reason)
		}
	}
	step.Artifacts = append(step.Artifacts, evidence.ArtifactRef{
		Kind: "provider_telemetry",
		Path: ".moyuan/models/provider-telemetry.jsonl",
	})
	return step
}

func runOperationsAuditExport(rootDir string, step StepRecord, options RunOptions) StepRecord {
	report, err := operations.ExportAudit(rootDir, operations.AuditExportOptions{
		Environment: options.Environment,
		Limit:       options.MaxSteps * 10,
		Format:      options.AuditFormat,
	})
	if err != nil {
		return failStep(step, err)
	}
	step.Status = "completed"
	if report.Summary.AttentionItemCount > 0 {
		step.Status = "attention_required"
	}
	step.Decision = "OPERATIONS_AUDIT_EXPORT_READY"
	step.Summary = plural(report.Summary.TimelineItemCount, "audit timeline item")
	step.Reasons = append(step.Reasons, "evidence_refs:"+strconv.Itoa(report.Summary.EvidenceRefCount))
	return step
}

func runDecisionLedger(rootDir string, step StepRecord, options RunOptions) StepRecord {
	ledger, err := operations.BuildDecisionLedger(rootDir, operations.DecisionLedgerOptions{
		Environment: options.Environment,
		Limit:       options.MaxSteps * 10,
	})
	if err != nil {
		return failStep(step, err)
	}
	step.Status = "completed"
	if ledger.Summary.AttentionCount > 0 {
		step.Status = "attention_required"
	}
	step.Decision = "DECISION_LEDGER_REFRESHED"
	step.Summary = plural(ledger.Summary.EntryCount, "decision ledger entry")
	step.Reasons = append(step.Reasons, "evidence_refs:"+strconv.Itoa(ledger.Summary.EvidenceRefCount))
	return step
}

func runPostDeploymentVerification(rootDir string, step StepRecord, options RunOptions) StepRecord {
	if strings.TrimSpace(options.DeploymentExecutionID) == "" {
		step.Status = "skipped"
		step.Decision = "POST_DEPLOYMENT_VERIFICATION_SKIPPED"
		step.Summary = "deployment execution id required"
		step.Reasons = append(step.Reasons, "deployment_execution_id_required")
		return step
	}
	verification, err := deployment.BuildPostDeploymentVerification(rootDir, deployment.PostDeploymentVerificationOptions{
		ExecutionID:  options.DeploymentExecutionID,
		Environment:  options.Environment,
		MonitorLimit: options.MonitorLimit,
	})
	if err != nil {
		return failStep(step, err)
	}
	step.Status = "completed"
	if verification.Status == "attention_required" || verification.Status == "blocked" || verification.Status == "failed" {
		step.Status = verification.Status
	}
	step.Decision = verification.Decision
	step.Summary = "post-deployment verification " + verification.Status
	step.Reasons = append(step.Reasons, verification.Reasons...)
	step.Artifacts = append(step.Artifacts, evidence.ArtifactRef{
		Kind: "post_deployment_verification",
		ID:   verification.ID,
		Path: ".moyuan/lifecycle/deployments/post-deployment-verifications/" + verification.ID + ".json",
	})
	return step
}

func runComprehension(ctx context.Context, rootDir string, step StepRecord, options RunOptions) StepRecord {
	var profile comprehension.Profile
	var err error
	if strings.TrimSpace(options.ComprehensionSince) != "" {
		profile, err = comprehension.Incremental(ctx, rootDir, options.ComprehensionSince)
	} else {
		profile, err = comprehension.Full(ctx, rootDir, nil)
	}
	if err != nil {
		return failStep(step, err)
	}
	step.Status = "completed"
	step.Decision = "PROJECT_COMPREHENSION_REFRESHED"
	step.Summary = profile.Mode + " comprehension refreshed for " + profile.ProjectID
	step.Artifacts = append(step.Artifacts,
		evidence.ArtifactRef{Kind: "project_profile", Path: ".moyuan/comprehension/project-profile.md"},
		evidence.ArtifactRef{Kind: "module_map", Path: ".moyuan/comprehension/module-map.md"},
		evidence.ArtifactRef{Kind: "commands", Path: ".moyuan/comprehension/commands.md"},
	)
	return step
}

func finishStep(rootDir string, runID string, step *StepRecord) {
	finished := time.Now().UTC()
	step.FinishedAt = finished.Format(time.RFC3339Nano)
	started, err := time.Parse(time.RFC3339Nano, step.StartedAt)
	if err == nil {
		step.DurationMS = finished.Sub(started).Milliseconds()
	}
	record, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType: "control_loop",
		ParentID:   runID,
		Operation:  step.Type,
		Status:     step.Status,
		Decision:   step.Decision,
		Reasons:    stepReasons(*step),
		Artifacts:  step.Artifacts,
		Source:     "moyuan_control_loop",
	})
	if err == nil && record.ID != "" {
		step.EvidenceIDs = append(step.EvidenceIDs, record.ID)
	}
	_ = logging.Log(rootDir, "audit", "control_loop.step."+step.Status, map[string]any{
		"run_id":   runID,
		"step_id":  step.ID,
		"type":     step.Type,
		"decision": step.Decision,
	})
}

func failStep(step StepRecord, err error) StepRecord {
	step.Status = "failed"
	step.Decision = "CONTROL_LOOP_STEP_FAILED"
	step.Error = sanitizeReason(err.Error())
	step.Reasons = append(step.Reasons, step.Error)
	return step
}

func normalizeOptions(options RunOptions) RunOptions {
	options.Trigger = normalizeToken(options.Trigger)
	if options.Trigger == "" {
		options.Trigger = "manual"
	}
	options.RequestedBy = strings.TrimSpace(options.RequestedBy)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Environment = normalizeToken(options.Environment)
	options.DeploymentExecutionID = strings.TrimSpace(options.DeploymentExecutionID)
	options.AuditFormat = normalizeToken(options.AuditFormat)
	options.ProviderID = strings.TrimSpace(options.ProviderID)
	options.ComprehensionSince = strings.TrimSpace(options.ComprehensionSince)
	if options.RetryBudget < 0 {
		options.RetryBudget = 0
	}
	if options.RetryAttempt < 0 {
		options.RetryAttempt = 0
	}
	if options.MonitorLimit <= 0 {
		options.MonitorLimit = 20
	}
	if options.MaxSteps <= 0 {
		options.MaxSteps = 10
	}
	if options.MaxSteps > 20 {
		options.MaxSteps = 20
	}
	if options.StepTimeoutMS <= 0 {
		options.StepTimeoutMS = 30000
	}
	options.Steps = normalizeSteps(options.Steps)
	if len(options.Steps) == 0 {
		options.Steps = []string{StepResourceLifecycleScan, StepProviderOpsRefresh, StepProjectComprehensionRefresh}
	}
	options.ResourceIDs = compactOptionStrings(options.ResourceIDs)
	return options
}

func normalizeSteps(steps []string) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, step := range steps {
		step = normalizeToken(step)
		if step == "" || seen[step] {
			continue
		}
		seen[step] = true
		result = append(result, step)
	}
	return result
}

func save(rootDir string, run RunRecord) error {
	if err := fsutil.EnsureDir(runsDir(rootDir)); err != nil {
		return err
	}
	if err := writeRunFile(rootDir, run); err != nil {
		return err
	}
	return fsutil.AppendJSONL(runsJSONLPath(rootDir), run)
}

func writeRunFile(rootDir string, run RunRecord) error {
	if err := fsutil.EnsureDir(runsDir(rootDir)); err != nil {
		return err
	}
	return fsutil.WriteJSON(filepath.Join(runsDir(rootDir), run.ID+".json"), run)
}

func retryBudgetExhausted(run RunRecord) bool {
	if run.RetryBudget <= 0 {
		return true
	}
	return run.RetryAttempt >= run.RetryBudget
}

func saveIdempotency(rootDir string, key string, runID string, createdAt string) error {
	if key == "" {
		return nil
	}
	if err := fsutil.EnsureDir(idempotencyDir(rootDir)); err != nil {
		return err
	}
	record := idempotencyRecord{Key: key, RunID: runID, CreatedAt: createdAt}
	return fsutil.WriteJSON(idempotencyPath(rootDir, key), record)
}

func loadIdempotentRun(rootDir string, key string) (RunRecord, bool, error) {
	var record idempotencyRecord
	found, err := fsutil.ReadJSON(idempotencyPath(rootDir, key), &record)
	if err != nil || !found || record.RunID == "" {
		return RunRecord{}, false, err
	}
	run, runFound, err := Load(rootDir, record.RunID)
	return run, runFound, err
}

func hasAttention(steps []StepRecord) bool {
	for _, step := range steps {
		if step.Status == "attention_required" || step.Status == "blocked" || step.Status == "skipped" {
			return true
		}
	}
	return false
}

func stepReasons(step StepRecord) []string {
	reasons := append([]string{}, step.Reasons...)
	if step.Error != "" {
		reasons = append(reasons, step.Error)
	}
	if len(reasons) == 0 && step.Summary != "" {
		reasons = append(reasons, step.Summary)
	}
	return reasons
}

func plural(count int, label string) string {
	if count == 1 {
		return "1 " + label
	}
	return strconv.Itoa(count) + " " + label + "s"
}

func runsDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ControlLoopDir, "runs")
}

func runsJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ControlLoopDir, "runs.jsonl")
}

func idempotencyDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ControlLoopDir, "idempotency")
}

func idempotencyPath(rootDir string, key string) string {
	return filepath.Join(idempotencyDir(rootDir), safeKey(key)+".json")
}

func safeKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	key := strings.Trim(b.String(), "-")
	if key == "" {
		return "run"
	}
	if len(key) > 120 {
		return strings.Trim(key[:120], "-")
	}
	return key
}

func cleanID(id string) (string, bool) {
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "\\") || strings.Contains(id, "..") {
		return "", false
	}
	return id, true
}

func normalizeToken(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
}

func sanitizeReason(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	if len(value) > 240 {
		return value[:240]
	}
	return value
}

func compactOptionStrings(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func timeID(t time.Time) string {
	return strings.ReplaceAll(t.UTC().Format("20060102150405.000000000"), ".", "")
}
