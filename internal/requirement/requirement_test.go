package requirement

import (
	"strings"
	"testing"
	"unicode/utf8"

	"moyuan-code/internal/issues"
	"moyuan-code/internal/workspace"
)

func TestPlanFromTextCreatesIssueGraphAndSchedule(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanFromText(root, "新增后端 API 查询项目状态，并补充 go test 验证")
	if err != nil {
		t.Fatal(err)
	}
	if plan.ClarificationDecision.Required {
		t.Fatalf("did not expect clarification: %+v", plan.ClarificationDecision)
	}
	if len(plan.Issues) != 3 {
		t.Fatalf("issues length = %d", len(plan.Issues))
	}
	if plan.Issues[1].Role != "backend" || plan.Issues[1].Title != "backend-implementation" {
		t.Fatalf("unexpected implementation issue: %+v", plan.Issues[1])
	}

	graph, ok, err := issues.LoadGraph(root, plan.EpicID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected issue graph to be saved")
	}
	if len(graph.Nodes) != 3 || graph.Nodes[0].Status != "ready" {
		t.Fatalf("unexpected graph: %+v", graph)
	}

	schedule, ok, err := issues.LoadSchedule(root, plan.EpicID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(schedule.ReadyQueue) != 1 || len(schedule.BlockedQueue) != 2 {
		t.Fatalf("unexpected schedule ok=%v schedule=%+v", ok, schedule)
	}
}

func TestPlanFromTextCreatesUniqueIDsForRapidRequests(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	first, err := PlanFromText(root, "实现后端 API 支持恢复队列审批消费，并补充 go test 验证")
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanFromText(root, "实现后端 API 支持 release provider 发布，并补充 go test 验证")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID {
		t.Fatalf("expected unique plan IDs, got %s", first.ID)
	}

	plans, err := List(root, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected both plans to be listed, got %d", len(plans))
	}
}

func TestPlanFromTextDoesNotCreateGarbledIssueIDs(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanFromText(root, "Phase 29：实现 GitHub/Gitee release provider 真实发布 adapter，并补充 API 测试验证")
	if err != nil {
		t.Fatal(err)
	}
	if !utf8.ValidString(plan.EpicID) {
		t.Fatalf("epic id is not valid utf8: %q", plan.EpicID)
	}
	if strings.ContainsRune(plan.EpicID, '\uFFFD') {
		t.Fatalf("epic id contains replacement character: %q", plan.EpicID)
	}
	for _, issue := range plan.Issues {
		if !utf8.ValidString(issue.ID) {
			t.Fatalf("issue id is not valid utf8: %q", issue.ID)
		}
		if strings.ContainsRune(issue.ID, '\uFFFD') {
			t.Fatalf("issue id contains replacement character: %q", issue.ID)
		}
	}
}

func TestPlanFromTextRequiresClarificationForWeakRequirement(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanFromText(root, "优化一下")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.ClarificationDecision.Required {
		t.Fatalf("expected clarification for weak requirement")
	}
	if plan.IssueGraph.Epic.Status != "needs_clarification" {
		t.Fatalf("unexpected epic status: %s", plan.IssueGraph.Epic.Status)
	}
	if plan.IssueGraph.Nodes[0].Status != "blocked" {
		t.Fatalf("clarification graph should not have ready work: %+v", plan.IssueGraph.Nodes)
	}
}
