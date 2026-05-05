package review

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/batch"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/quality"
	"moyuan-code/internal/workspace"
)

func TestDecideMergeAllowsAcceptedIssueWithAcceptedQuality(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	writeIssueState(t, root, orchestrator.IssueState{IssueID: "issue-1", Status: "accepted", QualityReportID: "quality-1"})
	writeQualityReport(t, root, quality.Report{ID: "quality-1", TaskID: "issue-1", Status: "passed", ReviewStatus: "accepted"})

	decision, err := DecideMerge(root, "issue-1")
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != "ready_to_merge" || decision.Decision != "MERGE_ALLOWED" {
		t.Fatalf("unexpected merge decision: %+v", decision)
	}

	loaded, ok, err := Load(root, decision.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || loaded.ID != decision.ID {
		t.Fatalf("expected merge decision to be saved, ok=%v loaded=%+v", ok, loaded)
	}
}

func TestDecideMergeBlocksMissingOrRejectedFacts(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	missing, err := DecideMerge(root, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if missing.Status != "blocked" || missing.Reasons[0] != "issue_state_missing" {
		t.Fatalf("unexpected missing decision: %+v", missing)
	}

	writeIssueState(t, root, orchestrator.IssueState{IssueID: "issue-2", Status: "accepted", QualityReportID: "quality-2"})
	writeQualityReport(t, root, quality.Report{ID: "quality-2", TaskID: "issue-2", Status: "failed", ReviewStatus: "rejected"})
	rejected, err := DecideMerge(root, "issue-2")
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Status != "needs_rework" || rejected.Decision != "NEEDS_REWORK" {
		t.Fatalf("unexpected rejected decision: %+v", rejected)
	}
}

func TestBuildMergeQueueAllowsAcceptedBatchItem(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	initReviewGitRepo(t, root)
	t.Setenv("MOYUAN_ALLOW_BATCH_RUN", "1")
	graph := issues.Graph{
		Epic: issues.Epic{ID: "phase11-merge", Title: "merge queue", Status: "planned"},
		Nodes: []issues.Node{
			{ID: "backend-api", Title: "backend api", Status: "ready"},
		},
	}
	if err := issues.SaveGraph(root, graph); err != nil {
		t.Fatal(err)
	}
	plan, err := batch.CreatePlan(root, batch.PlanOptions{EpicID: graph.Epic.ID, MaxParallel: 1, RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	run, err := batch.Run(context.Background(), root, batch.RunOptions{BatchID: plan.ID, Mode: "local_shell", Approved: true, Prompt: "printf ok"})
	if err != nil {
		t.Fatal(err)
	}
	if run.Decision != "BATCH_RUN_COMPLETED" {
		t.Fatalf("expected completed batch run, got %+v", run)
	}

	queue, err := BuildMergeQueue(root, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if queue.Decision != "MERGE_QUEUE_READY" || queue.ReadyCount != 1 || len(queue.Items) != 1 {
		t.Fatalf("expected ready merge queue, got %+v", queue)
	}
	if queue.Items[0].Decision != "MERGE_QUEUE_ITEM_READY" || queue.Items[0].MergeDecision.Decision != "MERGE_ALLOWED" {
		t.Fatalf("expected ready queue item, got %+v", queue.Items[0])
	}
	loaded, found, err := LoadMergeQueue(root, queue.ID)
	if err != nil || !found || loaded.ID != queue.ID {
		t.Fatalf("expected persisted merge queue, found=%v loaded=%+v err=%v", found, loaded, err)
	}
	queues, err := ListMergeQueues(root, plan.ID, 10)
	if err != nil || len(queues) != 1 || queues[0].ID != queue.ID {
		t.Fatalf("expected listed merge queue, queues=%+v err=%v", queues, err)
	}
}

func TestBuildMergeQueueBlocksWithoutBatchRun(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	graph := issues.Graph{
		Epic: issues.Epic{ID: "phase11-no-run", Title: "merge queue blocked", Status: "planned"},
		Nodes: []issues.Node{
			{ID: "backend-api", Title: "backend api", Status: "ready"},
		},
	}
	if err := issues.SaveGraph(root, graph); err != nil {
		t.Fatal(err)
	}
	plan, err := batch.CreatePlan(root, batch.PlanOptions{EpicID: graph.Epic.ID, MaxParallel: 1, RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	queue, err := BuildMergeQueue(root, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if queue.Decision != "MERGE_QUEUE_BLOCKED" || !containsReason(queue.Reasons, "batch_run_missing") {
		t.Fatalf("expected missing run block, got %+v", queue)
	}
}

func writeIssueState(t *testing.T, root string, state orchestrator.IssueState) {
	t.Helper()
	path := workspace.ForRoot(root).OrchestratorDir + "/issue-states/" + state.IssueID + ".json"
	if err := fsutil.WriteJSON(path, state); err != nil {
		t.Fatal(err)
	}
}

func writeQualityReport(t *testing.T, root string, report quality.Report) {
	t.Helper()
	path := workspace.ForRoot(root).QualityDir + "/reports/" + report.ID + ".json"
	if err := fsutil.WriteJSON(path, report); err != nil {
		t.Fatal(err)
	}
}

func initReviewGitRepo(t *testing.T, root string) {
	t.Helper()
	runReviewGit(t, root, "init")
	runReviewGit(t, root, "config", "user.email", "test@example.com")
	runReviewGit(t, root, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# review test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runReviewGit(t, root, "add", ".")
	runReviewGit(t, root, "commit", "-m", "initial")
}

func runReviewGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

func containsReason(reasons []string, target string) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}
