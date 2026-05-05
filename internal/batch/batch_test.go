package batch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/issues"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/workspace"
)

func TestCreatePlanExplainsDispatchWaitingAndBlockedItems(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	graph := issues.Graph{
		Epic: issues.Epic{ID: "phase11-test", Title: "batch preview", Status: "planned"},
		Nodes: []issues.Node{
			{ID: "backend-api", Title: "backend api", Status: "ready"},
			{ID: "backend-worker", Title: "backend worker", Status: "ready"},
			{ID: "frontend-ui", Title: "frontend ui", Status: "ready"},
			{ID: "release-check", Title: "release check", Status: "blocked", DependsOn: []string{"backend-api"}},
		},
	}
	if err := issues.SaveGraph(root, graph); err != nil {
		t.Fatal(err)
	}

	plan, err := CreatePlan(root, PlanOptions{EpicID: graph.Epic.ID, MaxParallel: 3, RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Decision != "BATCH_PLAN_READY" || plan.DispatchCount != 2 || plan.WaitingCount != 1 || plan.BlockedCount != 1 {
		t.Fatalf("expected dispatch/waiting/blocked counts, got %+v", plan)
	}
	if plan.WriteScopeConflictCount != 1 {
		t.Fatalf("expected one write scope conflict, got %+v", plan)
	}
	dispatch := findItem(plan.Items, "backend-api")
	if dispatch.Decision != "dispatch" || dispatch.RuntimeID != "codex_cli" || dispatch.ProviderID == "" || dispatch.RouteDecision == "" {
		t.Fatalf("expected backend dispatch with provider route, got %+v", dispatch)
	}
	waiting := findItem(plan.Items, "backend-worker")
	if waiting.Decision != "waiting" || waiting.Reason != "write_scope_conflict" || len(waiting.ConflictsWith) != 1 {
		t.Fatalf("expected backend worker to wait on write scope conflict, got %+v", waiting)
	}
	blocked := findItem(plan.Items, "release-check")
	if blocked.Decision != "blocked" || blocked.Reason != "waiting_dependencies" || len(blocked.DependencyIDs) != 1 {
		t.Fatalf("expected blocked dependency item, got %+v", blocked)
	}
	loaded, found, err := Load(root, plan.ID)
	if err != nil || !found || loaded.ID != plan.ID {
		t.Fatalf("expected persisted batch plan, found=%v loaded=%+v err=%v", found, loaded, err)
	}
	plans, err := List(root, graph.Epic.ID, 10)
	if err != nil || len(plans) != 1 || plans[0].ID != plan.ID {
		t.Fatalf("expected listed batch plan, plans=%+v err=%v", plans, err)
	}
}

func TestRunDryRunDoesNotExecuteRuntime(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	graph := issues.Graph{
		Epic: issues.Epic{ID: "phase11-dry-run", Title: "batch dry run", Status: "planned"},
		Nodes: []issues.Node{
			{ID: "backend-api", Title: "backend api", Status: "ready"},
			{ID: "frontend-ui", Title: "frontend ui", Status: "ready"},
		},
	}
	if err := issues.SaveGraph(root, graph); err != nil {
		t.Fatal(err)
	}
	plan, err := CreatePlan(root, PlanOptions{EpicID: graph.Epic.ID, MaxParallel: 2, RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}

	run, err := Run(context.Background(), root, RunOptions{BatchID: plan.ID, RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if run.Decision != "BATCH_RUN_DRY_RUN" || run.Status != "completed" || len(run.Items) != 2 {
		t.Fatalf("expected completed dry run with two items, got %+v", run)
	}
	if _, found, err := orchestrator.LoadIssueState(root, "backend-api"); err != nil || found {
		t.Fatalf("dry run should not create issue state, found=%v err=%v", found, err)
	}
	loaded, found, err := LoadRun(root, run.ID)
	if err != nil || !found || loaded.ID != run.ID {
		t.Fatalf("expected persisted batch run, found=%v loaded=%+v err=%v", found, loaded, err)
	}
	runs, err := ListRuns(root, plan.ID, 10)
	if err != nil || len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("expected listed batch run, runs=%+v err=%v", runs, err)
	}
}

func TestRunLocalShellRequiresApprovalAndEnablement(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	graph := singleReadyGraph("phase11-guarded-run")
	if err := issues.SaveGraph(root, graph); err != nil {
		t.Fatal(err)
	}
	plan, err := CreatePlan(root, PlanOptions{EpicID: graph.Epic.ID, MaxParallel: 1, RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}

	notApproved, err := Run(context.Background(), root, RunOptions{BatchID: plan.ID, Mode: "local_shell", Prompt: "printf ok"})
	if err != nil {
		t.Fatal(err)
	}
	if notApproved.Decision != "BATCH_RUN_BLOCKED" || !hasReason(notApproved.Reasons, "batch_run_approval_required") {
		t.Fatalf("expected approval guard, got %+v", notApproved)
	}
	notEnabled, err := Run(context.Background(), root, RunOptions{BatchID: plan.ID, Mode: "local_shell", Approved: true, Prompt: "printf ok"})
	if err != nil {
		t.Fatal(err)
	}
	if notEnabled.Decision != "BATCH_RUN_BLOCKED" || !hasReason(notEnabled.Reasons, "batch_run_not_enabled") {
		t.Fatalf("expected enablement guard, got %+v", notEnabled)
	}
}

func TestRunLocalShellExecutesOneIssueWhenEnabled(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	initBatchGitRepo(t, root)
	t.Setenv("MOYUAN_ALLOW_BATCH_RUN", "1")
	graph := singleReadyGraph("phase11-enabled-run")
	if err := issues.SaveGraph(root, graph); err != nil {
		t.Fatal(err)
	}
	plan, err := CreatePlan(root, PlanOptions{EpicID: graph.Epic.ID, MaxParallel: 1, RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}

	run, err := Run(context.Background(), root, RunOptions{
		BatchID:     plan.ID,
		Mode:        "local_shell",
		Approved:    true,
		Prompt:      "printf batch-ok",
		RequestedBy: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Decision != "BATCH_RUN_COMPLETED" || run.Status != "completed" || len(run.Items) != 1 {
		t.Fatalf("expected completed local shell run, got %+v", run)
	}
	item := run.Items[0]
	if item.Decision != "BATCH_ITEM_ACCEPTED" || item.RunID == "" || item.SubagentID == "" || item.QualityReportID == "" {
		t.Fatalf("expected accepted item with artifacts, got %+v", item)
	}
	state, found, err := orchestrator.LoadIssueState(root, item.IssueID)
	if err != nil || !found || state.EpicID != graph.Epic.ID || state.Status != "accepted" {
		t.Fatalf("expected issue state in custom epic, found=%v state=%+v err=%v", found, state, err)
	}
}

func findItem(items []IssueItem, issueID string) IssueItem {
	for _, item := range items {
		if item.IssueID == issueID {
			return item
		}
	}
	return IssueItem{}
}

func singleReadyGraph(epicID string) issues.Graph {
	return issues.Graph{
		Epic: issues.Epic{ID: epicID, Title: "batch run", Status: "planned"},
		Nodes: []issues.Node{
			{ID: "backend-api", Title: "backend api", Status: "ready"},
		},
	}
}

func hasReason(reasons []string, target string) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}

func initBatchGitRepo(t *testing.T, root string) {
	t.Helper()
	runBatchGit(t, root, "init")
	runBatchGit(t, root, "config", "user.email", "test@example.com")
	runBatchGit(t, root, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# batch test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runBatchGit(t, root, "add", ".")
	runBatchGit(t, root, "commit", "-m", "initial")
}

func runBatchGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}
