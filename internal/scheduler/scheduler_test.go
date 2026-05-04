package scheduler

import (
	"testing"

	"moyuan-code/internal/issues"
	"moyuan-code/internal/workspace"
)

func TestBuildDispatchesOnlyNonConflictingReadyIssues(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	graph := issues.Graph{
		Epic: issues.Epic{ID: "beta-parallel", Title: "parallel", Status: "planned"},
		Nodes: []issues.Node{
			{ID: "backend-a", Title: "backend API", Status: "ready"},
			{ID: "backend-b", Title: "backend worker", Status: "ready"},
			{ID: "frontend-a", Title: "frontend UI", Status: "ready"},
			{ID: "blocked-a", Title: "quality review", Status: "blocked", DependsOn: []string{"backend-a"}},
		},
	}
	if err := issues.SaveGraph(root, graph); err != nil {
		t.Fatal(err)
	}

	plan, err := Build(root, "beta-parallel", 3)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Parallelism != 2 {
		t.Fatalf("parallelism = %d plan=%+v", plan.Parallelism, plan)
	}
	if len(plan.DispatchQueue) != 2 {
		t.Fatalf("dispatch queue length = %d plan=%+v", len(plan.DispatchQueue), plan)
	}
	if plan.DispatchQueue[0].IssueID != "backend-a" || plan.DispatchQueue[1].IssueID != "frontend-a" {
		t.Fatalf("unexpected dispatch queue: %+v", plan.DispatchQueue)
	}
	if len(plan.WaitingQueue) != 1 || plan.WaitingQueue[0].Reason != "write_scope_conflict" {
		t.Fatalf("unexpected waiting queue: %+v", plan.WaitingQueue)
	}
	if plan.BlockedReason["blocked-a"] != "waiting_dependencies" {
		t.Fatalf("unexpected blocked reason: %+v", plan.BlockedReason)
	}
}

func TestBuildHonorsRuntimeSlotBudget(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	graph := issues.Graph{
		Epic: issues.Epic{ID: "slot-budget", Title: "slot", Status: "planned"},
		Nodes: []issues.Node{
			{ID: "frontend-a", Title: "frontend UI", Status: "ready"},
			{ID: "backend-a", Title: "backend API", Status: "ready"},
		},
	}
	if err := issues.SaveGraph(root, graph); err != nil {
		t.Fatal(err)
	}

	plan, err := Build(root, "slot-budget", 1)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Parallelism != 1 || len(plan.DispatchQueue) != 1 {
		t.Fatalf("unexpected dispatch under slot budget: %+v", plan)
	}
	if len(plan.WaitingQueue) != 1 || plan.WaitingQueue[0].Reason != "runtime_slot" {
		t.Fatalf("expected runtime slot waiting reason: %+v", plan.WaitingQueue)
	}
}
