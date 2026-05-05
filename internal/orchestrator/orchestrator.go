package orchestrator

import (
	"context"
	"path/filepath"
	"time"

	"moyuan-code/internal/auth"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/quality"
	runrecord "moyuan-code/internal/run"
	"moyuan-code/internal/runtime"
	"moyuan-code/internal/scheduler"
	"moyuan-code/internal/subagent"
	"moyuan-code/internal/workspace"
)

type Result struct {
	IssueID       string         `json:"issue_id"`
	RunID         string         `json:"run_id"`
	SubagentID    string         `json:"subagent_id"`
	RuntimeResult runtime.Result `json:"runtime_result"`
	QualityReport quality.Report `json:"quality_report"`
	Status        string         `json:"status"`
	IssueState    IssueState     `json:"issue_state"`
	RunState      RunState       `json:"run_state"`
	CreatedAt     string         `json:"created_at"`
}

type RunOptions struct {
	RuntimeID    string `json:"runtime_id"`
	ProviderID   string `json:"provider_id,omitempty"`
	ModelID      string `json:"model_id,omitempty"`
	EpicID       string `json:"epic_id,omitempty"`
	Role         string `json:"role"`
	Prompt       string `json:"prompt"`
	WorktreePath string `json:"worktree_path,omitempty"`
	Branch       string `json:"branch,omitempty"`
}

func Plan(rootDir string, epicID string) (scheduler.Plan, error) {
	return scheduler.Build(rootDir, epicID, 1)
}

func RunIssue(ctx context.Context, rootDir string, issueID string, runtimeID string, prompt string) (Result, error) {
	return RunIssueWithOptions(ctx, rootDir, issueID, RunOptions{RuntimeID: runtimeID, Role: "backend", Prompt: prompt})
}

func RunIssueWithOptions(ctx context.Context, rootDir string, issueID string, options RunOptions) (Result, error) {
	if issueID == "" {
		issueID = "task-unknown"
	}
	if options.RuntimeID == "" {
		options.RuntimeID = "local_shell"
	}
	if options.Role == "" {
		options.Role = "backend"
	}
	if options.EpicID == "" {
		options.EpicID = "phase1-epic"
	}
	if options.WorktreePath == "" {
		options.WorktreePath = rootDir
	}
	if options.ProviderID == "" && options.RuntimeID != "local_shell" {
		decision, err := providers.Route(rootDir, providers.RouteRequest{
			Role:                  options.Role,
			RequiresRepoEdit:      true,
			IncludesSensitiveCode: true,
			IncludesProjectMemory: true,
		})
		if err != nil {
			return Result{}, err
		}
		if !decision.Blocked && decision.RuntimeID == options.RuntimeID {
			options.ProviderID = decision.ProviderID
			options.ModelID = decision.ModelID
		}
	}
	authCtx, err := auth.NewContext(rootDir, "issue.run", "normal")
	if err != nil {
		return Result{}, err
	}
	graph, _, _ := issues.LoadGraph(rootDir, options.EpicID)
	_ = graph
	run, err := runrecord.Create(rootDir, issueID, map[string]any{
		"issue_id":     issueID,
		"auth_context": authCtx,
		"mode":         "orchestrated",
		"role":         options.Role,
		"runtime_id":   options.RuntimeID,
		"provider_id":  options.ProviderID,
		"model_id":     options.ModelID,
	})
	if err != nil {
		return Result{}, err
	}
	instance, err := subagent.Create(rootDir, subagent.CreateOptions{
		ParentType:     "issue",
		ParentID:       issueID,
		IssueID:        issueID,
		RunID:          run.ID,
		Role:           options.Role,
		RuntimeID:      options.RuntimeID,
		ProviderID:     options.ProviderID,
		ModelID:        options.ModelID,
		Skills:         []string{"quality-gate", "diff-review"},
		MemoryScope:    []string{"project", "issue", "recent-runs"},
		ReadScope:      []string{"project:" + workspace.ForRoot(rootDir).RootDir},
		WriteScope:     []string{"issue:" + issueID},
		OutputContract: []string{"runtime_result", "quality_report", "review_status"},
	})
	if err != nil {
		return Result{}, err
	}
	if _, err := transitionIssue(rootDir, options.EpicID, issueID, "running", "", run.ID, nil); err != nil {
		return Result{}, err
	}
	if _, err := transitionRun(rootDir, issueID, run.ID, "running", "", func(state *RunState) {
		state.SubagentID = instance.ID
	}); err != nil {
		return Result{}, err
	}
	rt, err := runtime.Invoke(ctx, rootDir, runtime.Invocation{
		RunID:          run.ID,
		ProjectID:      workspace.ForRoot(rootDir).RootDir,
		IssueID:        issueID,
		Role:           options.Role,
		RuntimeID:      options.RuntimeID,
		ProviderID:     options.ProviderID,
		ModelID:        options.ModelID,
		Mode:           "code",
		WorkspaceRoot:  rootDir,
		WorktreePath:   options.WorktreePath,
		Branch:         options.Branch,
		Prompt:         options.Prompt,
		ProtectedPaths: protectedPaths(rootDir),
	})
	if err != nil {
		_, _ = transitionIssue(rootDir, options.EpicID, issueID, "failed", "runtime_error", run.ID, nil)
		_, _ = transitionRun(rootDir, issueID, run.ID, "failed", "runtime_error", nil)
		_, _, _ = subagent.Finish(rootDir, instance.ID, "failed")
		return Result{}, err
	}
	if _, err := transitionRun(rootDir, issueID, run.ID, "collecting_outputs", "", func(state *RunState) {
		state.SubagentID = instance.ID
		state.RuntimeID = options.RuntimeID
		state.RuntimeStatus = rt.Status
		state.RecoveryID = rt.RecoveryID
	}); err != nil {
		return Result{}, err
	}
	if _, err := transitionIssue(rootDir, options.EpicID, issueID, "quality_checking", "", run.ID, nil); err != nil {
		return Result{}, err
	}
	report, err := quality.RunWithReview(ctx, rootDir, issueID, quality.ReviewInput{
		ChangedFiles:    rt.ChangedFiles,
		DiffSummaryPath: rt.DiffSummaryPath,
		ProtectedFiles:  rt.Diff.ProtectedFiles,
		RuntimeRisks:    rt.Risks,
		WorktreePath:    options.WorktreePath,
	})
	if err != nil {
		_, _ = transitionIssue(rootDir, options.EpicID, issueID, "failed", "quality_error", run.ID, nil)
		_, _ = transitionRun(rootDir, issueID, run.ID, "failed", "quality_error", nil)
		_, _, _ = subagent.Finish(rootDir, instance.ID, "failed")
		return Result{}, err
	}
	if rt.ProviderID != "" {
		_, _, _ = providers.RecordQualityFeedback(rootDir, providers.FeedbackOptions{
			ProviderID:      rt.ProviderID,
			RuntimeID:       rt.RuntimeID,
			ModelID:         rt.ModelID,
			RunID:           rt.RunID,
			IssueID:         issueID,
			QualityReportID: report.ID,
			RuntimeStatus:   rt.Status,
			QualityStatus:   report.Status,
			Reason:          qualityFeedbackReason(report),
		})
	}
	status := "accepted"
	if rt.Status != "completed" || report.Status != "passed" {
		status = "needs_rework"
	}
	runStatus := "completed"
	if status != "accepted" {
		runStatus = "failed"
	}
	runState, err := transitionRun(rootDir, issueID, run.ID, runStatus, status, func(state *RunState) {
		state.SubagentID = instance.ID
		state.RuntimeID = options.RuntimeID
		state.RuntimeStatus = rt.Status
		state.RecoveryID = rt.RecoveryID
		state.QualityStatus = report.Status
		state.QualityReportID = report.ID
	})
	if err != nil {
		return Result{}, err
	}
	subagentStatus := "completed"
	finishOptions := subagent.FinishOptions{Status: subagentStatus, OutputConverged: true}
	if status != "accepted" {
		subagentStatus = "needs_rework"
		finishOptions = subagent.FinishOptions{Status: subagentStatus, BlockedReason: statusReason(status, rt.Status, report.Status)}
		if rt.RecoveryID != "" {
			subagentStatus = "archived"
			finishOptions = subagent.FinishOptions{
				Status:          subagentStatus,
				BlockedReason:   "runtime_" + rt.Status,
				ArchiveReason:   "native_runtime_recovery",
				RecoveryID:      rt.RecoveryID,
				FailureCategory: runtimeFailureCategory(rt),
				OutputConverged: false,
			}
		}
	}
	_, _, _ = subagent.FinishWithOptions(rootDir, instance.ID, finishOptions)
	issueState, err := transitionIssue(rootDir, options.EpicID, issueID, status, statusReason(status, rt.Status, report.Status), run.ID, func(state *IssueState) {
		state.QualityReportID = report.ID
	})
	if err != nil {
		return Result{}, err
	}
	result := Result{
		IssueID:       issueID,
		RunID:         run.ID,
		SubagentID:    instance.ID,
		RuntimeResult: rt,
		QualityReport: report,
		Status:        status,
		IssueState:    issueState,
		RunState:      runState,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, run.ID+"-result.json"), result); err != nil {
		return Result{}, err
	}
	_ = logging.Log(rootDir, "run", "orchestrator.issue.completed", map[string]any{"issue_id": issueID, "run_id": run.ID, "status": status})
	return result, nil
}

func qualityFeedbackReason(report quality.Report) string {
	if report.Status == "passed" {
		return "quality_status:passed"
	}
	for _, check := range report.Checks {
		if check.Status == "failed" {
			return "quality_check_failed:" + check.Type
		}
	}
	for _, finding := range report.Findings {
		if finding.Blocking {
			return "quality_blocking_finding:" + finding.Category
		}
	}
	return "quality_status:" + report.Status
}

func protectedPaths(rootDir string) []string {
	ws, err := workspace.Load(rootDir)
	if err != nil {
		return []string{}
	}
	return ws.Project.Workspace.ProtectedPaths
}

func statusReason(status string, runtimeStatus string, qualityStatus string) string {
	if status == "accepted" {
		return ""
	}
	if runtimeStatus != "completed" {
		return "runtime_" + runtimeStatus
	}
	if qualityStatus != "passed" {
		return "quality_" + qualityStatus
	}
	return status
}

func runtimeFailureCategory(result runtime.Result) string {
	for _, risk := range result.Risks {
		switch risk {
		case "runtime_failed", "pre_existing_dirty_worktree", "protected_paths_changed", "diff_unavailable":
			return risk
		}
		if len(risk) >= len("runtime_unavailable") && risk[:len("runtime_unavailable")] == "runtime_unavailable" {
			return "runtime_unavailable"
		}
	}
	if result.Status != "completed" {
		return "runtime_" + result.Status
	}
	return ""
}
