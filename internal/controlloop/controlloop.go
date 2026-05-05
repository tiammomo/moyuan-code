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
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/serverresources"
	"moyuan-code/internal/workspace"
)

const (
	StepResourceLifecycleScan       = "resource_lifecycle_scan"
	StepProviderOpsRefresh          = "provider_ops_refresh"
	StepProjectComprehensionRefresh = "project_comprehension_refresh"
)

type RunOptions struct {
	Trigger            string   `json:"trigger,omitempty"`
	RequestedBy        string   `json:"requested_by,omitempty"`
	Steps              []string `json:"steps,omitempty"`
	MaxSteps           int      `json:"max_steps,omitempty"`
	StepTimeoutMS      int      `json:"step_timeout_ms,omitempty"`
	ProviderID         string   `json:"provider_id,omitempty"`
	IncludeDisabled    bool     `json:"include_disabled,omitempty"`
	Probe              bool     `json:"probe,omitempty"`
	ProbeApproved      bool     `json:"probe_approved,omitempty"`
	ProbeTimeoutMS     int      `json:"probe_timeout_ms,omitempty"`
	ComprehensionSince string   `json:"comprehension_since,omitempty"`
}

type RunRecord struct {
	ID            string       `json:"id"`
	Status        string       `json:"status"`
	Decision      string       `json:"decision"`
	Trigger       string       `json:"trigger"`
	RequestedBy   string       `json:"requested_by,omitempty"`
	MaxSteps      int          `json:"max_steps"`
	StepTimeoutMS int          `json:"step_timeout_ms"`
	Steps         []StepRecord `json:"steps"`
	Reasons       []string     `json:"reasons,omitempty"`
	StartedAt     string       `json:"started_at"`
	FinishedAt    string       `json:"finished_at"`
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

func Run(ctx context.Context, rootDir string, options RunOptions) (RunRecord, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return RunRecord{}, err
	}
	options = normalizeOptions(options)
	if len(options.Steps) > options.MaxSteps {
		return RunRecord{}, errors.New("control_loop_steps_exceed_max")
	}
	start := time.Now().UTC()
	run := RunRecord{
		ID:            "control-loop-" + timeID(start),
		Status:        "running",
		Decision:      "CONTROL_LOOP_RUNNING",
		Trigger:       options.Trigger,
		RequestedBy:   options.RequestedBy,
		MaxSteps:      options.MaxSteps,
		StepTimeoutMS: options.StepTimeoutMS,
		Steps:         []StepRecord{},
		Reasons:       []string{},
		StartedAt:     start.Format(time.RFC3339Nano),
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
	if run.Status != "failed" {
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
	case StepProviderOpsRefresh:
		step = runProviderOps(rootDir, step, options)
	case StepProjectComprehensionRefresh:
		step = runComprehension(ctx, rootDir, step, options)
	default:
		step.Status = "skipped"
		step.Decision = "CONTROL_LOOP_STEP_SKIPPED"
		step.Summary = "unsupported control loop step"
		step.Reasons = append(step.Reasons, "unsupported_step:"+stepType)
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
	options.ProviderID = strings.TrimSpace(options.ProviderID)
	options.ComprehensionSince = strings.TrimSpace(options.ComprehensionSince)
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
	if err := fsutil.WriteJSON(filepath.Join(runsDir(rootDir), run.ID+".json"), run); err != nil {
		return err
	}
	return fsutil.AppendJSONL(runsJSONLPath(rootDir), run)
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

func timeID(t time.Time) string {
	return strings.ReplaceAll(t.UTC().Format("20060102150405.000000000"), ".", "")
}
