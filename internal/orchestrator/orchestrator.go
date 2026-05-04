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
	"moyuan-code/internal/workspace"
)

type Result struct {
	IssueID       string         `json:"issue_id"`
	RunID         string         `json:"run_id"`
	RuntimeResult runtime.Result `json:"runtime_result"`
	QualityReport quality.Report `json:"quality_report"`
	Status        string         `json:"status"`
	IssueState    IssueState     `json:"issue_state"`
	RunState      RunState       `json:"run_state"`
	CreatedAt     string         `json:"created_at"`
}

type RunOptions struct {
	RuntimeID  string `json:"runtime_id"`
	ProviderID string `json:"provider_id,omitempty"`
	ModelID    string `json:"model_id,omitempty"`
	Role       string `json:"role"`
	Prompt     string `json:"prompt"`
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
	graph, _, _ := issues.LoadGraph(rootDir, "phase1-epic")
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
	if _, err := transitionIssue(rootDir, "phase1-epic", issueID, "running", "", run.ID, nil); err != nil {
		return Result{}, err
	}
	if _, err := transitionRun(rootDir, issueID, run.ID, "running", "", nil); err != nil {
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
		WorktreePath:   rootDir,
		Prompt:         options.Prompt,
		ProtectedPaths: protectedPaths(rootDir),
	})
	if err != nil {
		_, _ = transitionIssue(rootDir, "phase1-epic", issueID, "failed", "runtime_error", run.ID, nil)
		_, _ = transitionRun(rootDir, issueID, run.ID, "failed", "runtime_error", nil)
		return Result{}, err
	}
	if _, err := transitionRun(rootDir, issueID, run.ID, "collecting_outputs", "", func(state *RunState) {
		state.RuntimeID = options.RuntimeID
		state.RuntimeStatus = rt.Status
	}); err != nil {
		return Result{}, err
	}
	if _, err := transitionIssue(rootDir, "phase1-epic", issueID, "quality_checking", "", run.ID, nil); err != nil {
		return Result{}, err
	}
	report, err := quality.RunWithReview(ctx, rootDir, issueID, quality.ReviewInput{
		ChangedFiles:    rt.ChangedFiles,
		DiffSummaryPath: rt.DiffSummaryPath,
		ProtectedFiles:  rt.Diff.ProtectedFiles,
		RuntimeRisks:    rt.Risks,
	})
	if err != nil {
		_, _ = transitionIssue(rootDir, "phase1-epic", issueID, "failed", "quality_error", run.ID, nil)
		_, _ = transitionRun(rootDir, issueID, run.ID, "failed", "quality_error", nil)
		return Result{}, err
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
		state.RuntimeID = options.RuntimeID
		state.RuntimeStatus = rt.Status
		state.QualityStatus = report.Status
		state.QualityReportID = report.ID
	})
	if err != nil {
		return Result{}, err
	}
	issueState, err := transitionIssue(rootDir, "phase1-epic", issueID, status, statusReason(status, rt.Status, report.Status), run.ID, func(state *IssueState) {
		state.QualityReportID = report.ID
	})
	if err != nil {
		return Result{}, err
	}
	result := Result{
		IssueID:       issueID,
		RunID:         run.ID,
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
