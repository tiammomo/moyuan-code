package batch

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/scheduler"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type PlanOptions struct {
	EpicID      string `json:"epic_id"`
	Mode        string `json:"mode,omitempty"`
	MaxParallel int    `json:"max_parallel,omitempty"`
	RequestedBy string `json:"requested_by,omitempty"`
}

type Plan struct {
	ID                      string      `json:"id"`
	EpicID                  string      `json:"epic_id"`
	Mode                    string      `json:"mode"`
	Status                  string      `json:"status"`
	Decision                string      `json:"decision"`
	RequestedBy             string      `json:"requested_by,omitempty"`
	MaxParallel             int         `json:"max_parallel"`
	DispatchCount           int         `json:"dispatch_count"`
	WaitingCount            int         `json:"waiting_count"`
	BlockedCount            int         `json:"blocked_count"`
	WriteScopeConflictCount int         `json:"write_scope_conflict_count"`
	RuntimeSlots            int         `json:"runtime_slots"`
	Reasons                 []string    `json:"reasons"`
	Items                   []IssueItem `json:"items"`
	CreatedAt               string      `json:"created_at"`
}

type IssueItem struct {
	IssueID        string                   `json:"issue_id"`
	Decision       string                   `json:"decision"`
	Reason         string                   `json:"reason,omitempty"`
	Role           string                   `json:"role,omitempty"`
	RuntimeID      string                   `json:"runtime_id,omitempty"`
	ProviderID     string                   `json:"provider_id,omitempty"`
	ModelID        string                   `json:"model_id,omitempty"`
	RouteDecision  string                   `json:"route_decision,omitempty"`
	RouteReason    string                   `json:"route_reason,omitempty"`
	WriteScopes    []string                 `json:"write_scopes,omitempty"`
	ConflictsWith  []string                 `json:"conflicts_with,omitempty"`
	DependencyIDs  []string                 `json:"dependency_ids,omitempty"`
	SubagentID     string                   `json:"subagent_id,omitempty"`
	SubagentStatus string                   `json:"subagent_status,omitempty"`
	RecoveryID     string                   `json:"recovery_id,omitempty"`
	RetryCount     int                      `json:"retry_count,omitempty"`
	MaxRetries     int                      `json:"max_retries,omitempty"`
	RouteSummary   *providers.RouteDecision `json:"route_summary,omitempty"`
}

func CreatePlan(rootDir string, options PlanOptions) (Plan, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Plan{}, err
	}
	options = normalizeOptions(options)
	if options.EpicID == "" {
		return Plan{}, errors.New("epic_id_required")
	}
	if options.Mode != "dry_run" {
		return Plan{}, errors.New("only_dry_run_batch_plan_supported")
	}
	now := time.Now().UTC()
	plan := Plan{
		ID:          "batch-" + textutil.Slugify(options.EpicID) + "-" + now.Format("20060102150405"),
		EpicID:      options.EpicID,
		Mode:        options.Mode,
		Status:      "blocked",
		Decision:    "BATCH_PLAN_BLOCKED",
		RequestedBy: options.RequestedBy,
		MaxParallel: options.MaxParallel,
		Reasons:     []string{},
		Items:       []IssueItem{},
		CreatedAt:   now.Format(time.RFC3339Nano),
	}
	graph, found, err := issues.LoadGraph(rootDir, options.EpicID)
	if err != nil {
		return Plan{}, err
	}
	if !found {
		plan.Reasons = append(plan.Reasons, "issue_graph_not_found")
		return finish(rootDir, plan)
	}
	schedule, err := scheduler.Build(rootDir, options.EpicID, options.MaxParallel)
	if err != nil {
		return Plan{}, err
	}
	plan.RuntimeSlots = schedule.RuntimeSlots
	registry, err := providers.Load(rootDir)
	if err != nil {
		return Plan{}, err
	}
	for _, decision := range schedule.DispatchQueue {
		plan.Items = append(plan.Items, itemFromDecision(registry, decision))
	}
	for _, decision := range schedule.WaitingQueue {
		item := itemFromDecision(registry, decision)
		if item.Reason == "write_scope_conflict" {
			plan.WriteScopeConflictCount++
		}
		plan.Items = append(plan.Items, item)
	}
	nodes := nodesByID(graph)
	for _, issueID := range schedule.BlockedQueue {
		node := nodes[issueID]
		plan.Items = append(plan.Items, IssueItem{
			IssueID:       issueID,
			Decision:      "blocked",
			Reason:        schedule.BlockedReason[issueID],
			DependencyIDs: append([]string{}, node.DependsOn...),
		})
	}
	plan.DispatchCount = len(schedule.DispatchQueue)
	plan.WaitingCount = len(schedule.WaitingQueue)
	plan.BlockedCount = len(schedule.BlockedQueue)
	switch {
	case plan.DispatchCount > 0:
		plan.Status = "planned"
		plan.Decision = "BATCH_PLAN_READY"
		plan.Reasons = append(plan.Reasons, "dispatch_ready")
	case plan.WaitingCount > 0 || plan.BlockedCount > 0:
		plan.Status = "waiting"
		plan.Decision = "BATCH_PLAN_WAITING"
		plan.Reasons = append(plan.Reasons, "no_dispatch_ready")
	default:
		plan.Status = "empty"
		plan.Decision = "BATCH_PLAN_EMPTY"
		plan.Reasons = append(plan.Reasons, "no_issue_items")
	}
	sort.SliceStable(plan.Items, func(i, j int) bool {
		if itemOrder(plan.Items[i].Decision) == itemOrder(plan.Items[j].Decision) {
			return plan.Items[i].IssueID < plan.Items[j].IssueID
		}
		return itemOrder(plan.Items[i].Decision) < itemOrder(plan.Items[j].Decision)
	})
	return finish(rootDir, plan)
}

func Load(rootDir string, id string) (Plan, bool, error) {
	if !validID(id) {
		return Plan{}, false, nil
	}
	var plan Plan
	found, err := fsutil.ReadJSON(filepath.Join(batchesDir(rootDir), id+".json"), &plan)
	return plan, found, err
}

func List(rootDir string, epicID string, limit int) ([]Plan, error) {
	if limit <= 0 {
		limit = 20
	}
	lines, err := fsutil.TailLines(batchesJSONLPath(rootDir), limit*4)
	if err != nil {
		return nil, err
	}
	plans := []Plan{}
	for _, line := range lines {
		var plan Plan
		if err := json.Unmarshal([]byte(line), &plan); err != nil {
			return nil, err
		}
		if plan.ID == "" {
			continue
		}
		if epicID != "" && plan.EpicID != epicID {
			continue
		}
		plans = append(plans, plan)
	}
	sort.SliceStable(plans, func(i, j int) bool {
		return plans[i].CreatedAt > plans[j].CreatedAt
	})
	if len(plans) > limit {
		return plans[:limit], nil
	}
	return plans, nil
}

func normalizeOptions(options PlanOptions) PlanOptions {
	options.EpicID = strings.TrimSpace(options.EpicID)
	options.Mode = strings.TrimSpace(strings.ToLower(options.Mode))
	if options.Mode == "" {
		options.Mode = "dry_run"
	}
	if options.MaxParallel <= 0 {
		options.MaxParallel = 2
	}
	if options.RequestedBy == "" {
		options.RequestedBy = "system"
	}
	return options
}

func itemFromDecision(registry providers.Registry, decision scheduler.DispatchDecision) IssueItem {
	route := providers.Decide(registry, providers.RouteRequest{
		Role:                  decision.Role,
		RequiresRepoEdit:      true,
		IncludesSensitiveCode: true,
		IncludesProjectMemory: true,
	})
	item := IssueItem{
		IssueID:        decision.IssueID,
		Decision:       decision.Decision,
		Reason:         decision.Reason,
		Role:           decision.Role,
		RuntimeID:      decision.RuntimeID,
		ProviderID:     route.ProviderID,
		ModelID:        route.ModelID,
		RouteDecision:  route.Decision,
		RouteReason:    route.Reason,
		WriteScopes:    append([]string{}, decision.WriteScopes...),
		ConflictsWith:  append([]string{}, decision.ConflictsWith...),
		DependencyIDs:  append([]string{}, decision.DependencyIDs...),
		SubagentID:     decision.SubagentID,
		SubagentStatus: decision.SubagentStatus,
		RecoveryID:     decision.RecoveryID,
		RetryCount:     decision.RetryCount,
		MaxRetries:     decision.MaxRetries,
		RouteSummary:   &route,
	}
	if item.ProviderID == "" {
		item.ProviderID = decision.RuntimeID
	}
	return item
}

func nodesByID(graph issues.Graph) map[string]issues.Node {
	nodes := map[string]issues.Node{}
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
	}
	return nodes
}

func finish(rootDir string, plan Plan) (Plan, error) {
	if err := fsutil.EnsureDir(batchesDir(rootDir)); err != nil {
		return Plan{}, err
	}
	if err := fsutil.WriteJSON(filepath.Join(batchesDir(rootDir), plan.ID+".json"), plan); err != nil {
		return Plan{}, err
	}
	if err := fsutil.AppendJSONL(batchesJSONLPath(rootDir), plan); err != nil {
		return Plan{}, err
	}
	_ = logging.Log(rootDir, "run", "batch.plan.created", map[string]any{
		"batch_id":       plan.ID,
		"epic_id":        plan.EpicID,
		"decision":       plan.Decision,
		"status":         plan.Status,
		"dispatch_count": plan.DispatchCount,
		"waiting_count":  plan.WaitingCount,
		"blocked_count":  plan.BlockedCount,
	})
	return plan, nil
}

func batchesDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "batches")
}

func batchesJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "batches.jsonl")
}

func validID(id string) bool {
	id = strings.TrimSpace(id)
	return id != "" && !strings.Contains(id, "/") && !strings.Contains(id, "\\") && filepath.Base(id) == id
}

func itemOrder(decision string) int {
	switch decision {
	case "dispatch":
		return 0
	case "waiting":
		return 1
	case "blocked":
		return 2
	default:
		return 3
	}
}
