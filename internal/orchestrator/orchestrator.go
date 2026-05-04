package orchestrator

import (
	"context"
	"path/filepath"
	"time"

	"moyuan-code/internal/auth"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
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
	CreatedAt     string         `json:"created_at"`
}

func Plan(rootDir string, epicID string) (scheduler.Plan, error) {
	return scheduler.Build(rootDir, epicID, 1)
}

func RunIssue(ctx context.Context, rootDir string, issueID string, runtimeID string, prompt string) (Result, error) {
	if issueID == "" {
		issueID = "task-unknown"
	}
	if runtimeID == "" {
		runtimeID = "local_shell"
	}
	authCtx, err := auth.NewContext(rootDir, "issue.run", "normal")
	if err != nil {
		return Result{}, err
	}
	graph, _, _ := issues.LoadGraph(rootDir, "phase1-epic")
	_ = graph
	run, err := runrecord.Create(rootDir, issueID, map[string]any{"issue_id": issueID, "auth_context": authCtx, "mode": "orchestrated"})
	if err != nil {
		return Result{}, err
	}
	rt, err := runtime.Invoke(ctx, rootDir, runtime.Invocation{
		RunID:          run.ID,
		ProjectID:      workspace.ForRoot(rootDir).RootDir,
		IssueID:        issueID,
		Role:           "backend",
		RuntimeID:      runtimeID,
		Mode:           "code",
		WorkspaceRoot:  rootDir,
		WorktreePath:   rootDir,
		Prompt:         prompt,
		ProtectedPaths: protectedPaths(rootDir),
	})
	if err != nil {
		return Result{}, err
	}
	report, err := quality.Run(ctx, rootDir, issueID)
	if err != nil {
		return Result{}, err
	}
	status := "accepted"
	if rt.Status != "completed" || report.Status != "passed" {
		status = "needs_rework"
	}
	result := Result{
		IssueID:       issueID,
		RunID:         run.ID,
		RuntimeResult: rt,
		QualityReport: report,
		Status:        status,
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
