package batch

import (
	"testing"

	"moyuan-code/internal/issues"
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

func findItem(items []IssueItem, issueID string) IssueItem {
	for _, item := range items {
		if item.IssueID == issueID {
			return item
		}
	}
	return IssueItem{}
}
