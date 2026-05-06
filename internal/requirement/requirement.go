package requirement

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type Plan struct {
	ID                    string          `json:"id"`
	EpicID                string          `json:"epic_id"`
	RawText               string          `json:"raw_text"`
	ClarifiedRequirement  string          `json:"clarified_requirement"`
	ClarificationDecision Decision        `json:"clarification_decision"`
	AcceptanceCriteria    []string        `json:"acceptance_criteria"`
	TestPlan              []string        `json:"test_plan"`
	Issues                []IssueSpec     `json:"issues"`
	IssueGraph            issues.Graph    `json:"issue_graph"`
	Schedule              issues.Schedule `json:"schedule"`
	CreatedAt             string          `json:"created_at"`
}

type Decision struct {
	Status    string   `json:"status"`
	Required  bool     `json:"required"`
	Questions []string `json:"questions"`
	Reasons   []string `json:"reasons"`
}

type IssueSpec struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Status             string   `json:"status"`
	Role               string   `json:"role"`
	DependsOn          []string `json:"depends_on"`
	WriteScopes        []string `json:"write_scopes"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	TestPlan           []string `json:"test_plan"`
}

func PlanFromText(rootDir string, text string) (Plan, error) {
	now := time.Now().UTC()
	raw := strings.TrimSpace(text)
	id, err := newPlanID(now)
	if err != nil {
		return Plan{}, err
	}
	epicID := id + "-" + shortSlug(raw)
	decision := decide(raw)
	criteria := acceptanceCriteria(raw)
	testPlan := testPlanFor(raw)
	issueSpecs := issueSpecsFor(epicID, raw, criteria, testPlan, decision.Required)
	graph := graphFor(epicID, raw, issueSpecs, decision.Required)
	schedule := issues.Summarize(graph)
	plan := Plan{
		ID:                    id,
		EpicID:                epicID,
		RawText:               raw,
		ClarifiedRequirement:  clarifiedRequirement(raw, decision.Required),
		ClarificationDecision: decision,
		AcceptanceCriteria:    criteria,
		TestPlan:              testPlan,
		Issues:                issueSpecs,
		IssueGraph:            graph,
		Schedule:              schedule,
		CreatedAt:             now.Format(time.RFC3339Nano),
	}
	if err := fsutil.WriteJSON(planPath(rootDir, plan.ID), plan); err != nil {
		return Plan{}, err
	}
	if err := issues.SaveGraph(rootDir, graph); err != nil {
		return Plan{}, err
	}
	if err := issues.SaveSchedule(rootDir, schedule); err != nil {
		return Plan{}, err
	}
	_ = logging.Log(rootDir, "run", "requirement.planned", map[string]any{"requirement_id": plan.ID, "epic_id": plan.EpicID, "clarification_required": decision.Required, "issues": len(issueSpecs)})
	return plan, nil
}

func newPlanID(now time.Time) (string, error) {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", err
	}
	return "req-" + now.Format("20060102150405") + "-" + hex.EncodeToString(suffix[:]), nil
}

func Load(rootDir string, id string) (Plan, bool, error) {
	var plan Plan
	found, err := fsutil.ReadJSON(planPath(rootDir, id), &plan)
	if found {
		plan = refreshIssueGraph(rootDir, plan)
	}
	return plan, found, err
}

func List(rootDir string, limit int) ([]Plan, error) {
	dir := filepath.Join(workspace.ForRoot(rootDir).LifecycleDir, "requirements")
	if err := fsutil.EnsureDir(dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	plans := []Plan{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var plan Plan
		found, err := fsutil.ReadJSON(filepath.Join(dir, entry.Name()), &plan)
		if err != nil {
			return nil, err
		}
		if found && plan.ID != "" {
			plans = append(plans, refreshIssueGraph(rootDir, plan))
		}
	}
	sort.SliceStable(plans, func(i, j int) bool {
		return plans[i].CreatedAt > plans[j].CreatedAt
	})
	if limit > 0 && len(plans) > limit {
		return plans[:limit], nil
	}
	return plans, nil
}

func planPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).LifecycleDir, "requirements", id+".json")
}

func refreshIssueGraph(rootDir string, plan Plan) Plan {
	graph, ok, err := issues.LoadGraph(rootDir, plan.EpicID)
	if err == nil && ok {
		plan.IssueGraph = graph
		plan.Schedule = issues.Summarize(graph)
		plan.Issues = syncIssueSpecStatuses(plan.Issues, graph)
	}
	return plan
}

func syncIssueSpecStatuses(specs []IssueSpec, graph issues.Graph) []IssueSpec {
	statusByID := map[string]string{}
	for _, node := range graph.Nodes {
		statusByID[node.ID] = node.Status
	}
	next := append([]IssueSpec{}, specs...)
	for index := range next {
		if status := statusByID[next[index].ID]; status != "" {
			next[index].Status = status
		}
	}
	return next
}

func decide(text string) Decision {
	reasons := []string{}
	questions := []string{}
	trimmed := strings.TrimSpace(text)
	if len([]rune(trimmed)) < 12 {
		reasons = append(reasons, "requirement_too_short")
		questions = append(questions, "Please provide the goal, affected scope, and expected outcome.")
	}
	if lacksVerifiableGoal(trimmed) {
		reasons = append(reasons, "missing_verifiable_goal")
		questions = append(questions, "Please explain how completion will be verified, such as tests, UI behavior, API responses, or command output.")
	}
	if containsAmbiguity(trimmed) {
		reasons = append(reasons, "ambiguous_requirement")
		questions = append(questions, "Please confirm the chosen approach, priority, or implementation boundary.")
	}
	if len(reasons) > 0 {
		return Decision{Status: "needs_user_input", Required: true, Questions: unique(questions), Reasons: unique(reasons)}
	}
	return Decision{Status: "proceed", Required: false, Questions: []string{}, Reasons: []string{"sufficient_for_initial_issue_graph"}}
}

func clarifiedRequirement(text string, blocked bool) string {
	if strings.TrimSpace(text) == "" {
		return "Requirement is empty and must include a goal and verification method before development."
	}
	if blocked {
		return "This requirement needs clarification before development: " + strings.TrimSpace(text)
	}
	return "Based on current project context, the requirement goal is: " + strings.TrimSpace(text)
}

func acceptanceCriteria(text string) []string {
	base := []string{
		"Implementation must satisfy the original user goal.",
		"Existing tests must remain passing.",
		"New or changed behavior must have automated tests or clear manual verification steps.",
	}
	lower := strings.ToLower(text)
	if containsAny(lower, []string{"api", "接口", "后端", "backend"}) {
		base = append(base, "API responses, error handling, and status codes must be verifiable.")
	}
	if containsAny(lower, []string{"ui", "页面", "前端", "frontend"}) {
		base = append(base, "Critical UI flows must be verifiable through interaction checks or screenshots.")
	}
	return base
}

func testPlanFor(text string) []string {
	plan := []string{
		"Run the test commands identified for the project.",
		"Run regression tests for the changed scope.",
	}
	lower := strings.ToLower(text)
	if containsAny(lower, []string{"api", "接口", "后端", "backend"}) {
		plan = append(plan, "Add or run API unit tests and API smoke checks.")
	}
	if containsAny(lower, []string{"ui", "页面", "前端", "frontend"}) {
		plan = append(plan, "Add or run UI interaction checks and screenshots.")
	}
	return plan
}

func issueSpecsFor(epicID string, text string, criteria []string, tests []string, blocked bool) []IssueSpec {
	prefix := issuePrefix(epicID)
	issueStatus := "ready"
	if blocked {
		issueStatus = "blocked"
	}
	specs := []IssueSpec{
		{
			ID:                 prefix + "-001",
			Title:              "requirement-contract",
			Status:             issueStatus,
			Role:               "architect",
			DependsOn:          []string{},
			WriteScopes:        []string{"docs", "internal"},
			AcceptanceCriteria: criteria,
			TestPlan:           tests,
		},
		{
			ID:                 prefix + "-002",
			Title:              implementationTitle(text),
			Status:             "blocked",
			Role:               implementationRole(text),
			DependsOn:          []string{prefix + "-001"},
			WriteScopes:        writeScopesFor(text),
			AcceptanceCriteria: criteria,
			TestPlan:           tests,
		},
		{
			ID:                 prefix + "-003",
			Title:              "quality-review",
			Status:             "blocked",
			Role:               "quality_owner",
			DependsOn:          []string{prefix + "-002"},
			WriteScopes:        []string{".moyuan/lifecycle/quality"},
			AcceptanceCriteria: []string{"All quality gates must pass.", "review_status must not be rejected."},
			TestPlan:           []string{"Run quality check.", "Inspect issue graph and schedule status."},
		},
	}
	return specs
}

func graphFor(epicID string, text string, specs []IssueSpec, blocked bool) issues.Graph {
	epicStatus := "planned"
	if blocked {
		epicStatus = "needs_clarification"
	}
	nodes := []issues.Node{}
	for _, spec := range specs {
		nodes = append(nodes, issues.Node{ID: spec.ID, Title: spec.Title, Status: spec.Status, DependsOn: spec.DependsOn})
	}
	return issues.Graph{
		Epic:  issues.Epic{ID: epicID, Title: shortTitle(text), Status: epicStatus},
		Nodes: nodes,
	}
}

func issuePrefix(epicID string) string {
	return textutil.TrimSlug(textutil.Slugify(epicID), 42)
}

func shortSlug(text string) string {
	return textutil.TrimSlug(textutil.Slugify(text), 32)
}

func shortTitle(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "clarify-requirement"
	}
	runes := []rune(text)
	if len(runes) > 60 {
		return string(runes[:60])
	}
	return text
}

func implementationTitle(text string) string {
	lower := strings.ToLower(text)
	switch {
	case containsAny(lower, []string{"api", "接口", "后端", "backend"}):
		return "backend-implementation"
	case containsAny(lower, []string{"ui", "页面", "前端", "frontend"}):
		return "frontend-implementation"
	default:
		return "implementation"
	}
}

func implementationRole(text string) string {
	lower := strings.ToLower(text)
	switch {
	case containsAny(lower, []string{"ui", "页面", "前端", "frontend"}):
		return "frontend"
	case containsAny(lower, []string{"性能", "调优", "优化", "performance"}):
		return "backend_tuning"
	default:
		return "backend"
	}
}

func writeScopesFor(text string) []string {
	lower := strings.ToLower(text)
	scopes := []string{"internal"}
	if containsAny(lower, []string{"api", "接口", "后端", "backend"}) {
		scopes = append(scopes, "cmd")
	}
	if containsAny(lower, []string{"ui", "页面", "前端", "frontend"}) {
		scopes = append(scopes, "web", "src")
	}
	return unique(scopes)
}

func lacksVerifiableGoal(text string) bool {
	return !containsAny(strings.ToLower(text), []string{"实现", "新增", "修复", "优化", "支持", "生成", "测试", "验证", "api", "页面", "接口", "build", "test", "add", "fix", "support", "implement", "verify"})
}

func containsAmbiguity(text string) bool {
	return containsAny(strings.ToLower(text), []string{"随便", "都行", "看情况", "可能", "大概", "不确定", "maybe", "whatever"})
}

func containsAny(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func unique(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
