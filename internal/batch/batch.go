package batch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/scheduler"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
	"moyuan-code/internal/worktree"
)

type PlanOptions struct {
	EpicID      string `json:"epic_id"`
	Mode        string `json:"mode,omitempty"`
	MaxParallel int    `json:"max_parallel,omitempty"`
	RequestedBy string `json:"requested_by,omitempty"`
}

type RunOptions struct {
	BatchID           string `json:"batch_id"`
	Mode              string `json:"mode,omitempty"`
	Approved          bool   `json:"approved,omitempty"`
	MaxIssues         int    `json:"max_issues,omitempty"`
	RequestedBy       string `json:"requested_by,omitempty"`
	Prompt            string `json:"prompt,omitempty"`
	ContinueOnFailure bool   `json:"continue_on_failure,omitempty"`
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

type RunRecord struct {
	ID                string    `json:"id"`
	BatchID           string    `json:"batch_id"`
	EpicID            string    `json:"epic_id,omitempty"`
	Mode              string    `json:"mode"`
	Status            string    `json:"status"`
	Decision          string    `json:"decision"`
	RequestedBy       string    `json:"requested_by,omitempty"`
	Approved          bool      `json:"approved"`
	MaxIssues         int       `json:"max_issues"`
	ContinueOnFailure bool      `json:"continue_on_failure"`
	Items             []RunItem `json:"items"`
	Reasons           []string  `json:"reasons"`
	StartedAt         string    `json:"started_at"`
	FinishedAt        string    `json:"finished_at,omitempty"`
}

type RunItem struct {
	IssueID         string `json:"issue_id"`
	Status          string `json:"status"`
	Decision        string `json:"decision"`
	Reason          string `json:"reason,omitempty"`
	RuntimeID       string `json:"runtime_id,omitempty"`
	ProviderID      string `json:"provider_id,omitempty"`
	ModelID         string `json:"model_id,omitempty"`
	WorktreeID      string `json:"worktree_id,omitempty"`
	WorktreePath    string `json:"worktree_path,omitempty"`
	Branch          string `json:"branch,omitempty"`
	RunID           string `json:"run_id,omitempty"`
	SubagentID      string `json:"subagent_id,omitempty"`
	QualityReportID string `json:"quality_report_id,omitempty"`
	StartedAt       string `json:"started_at,omitempty"`
	FinishedAt      string `json:"finished_at,omitempty"`
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

func Run(ctx context.Context, rootDir string, options RunOptions) (RunRecord, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return RunRecord{}, err
	}
	options = normalizeRunOptions(options)
	if options.BatchID == "" {
		return RunRecord{}, errors.New("batch_id_required")
	}
	start := time.Now().UTC()
	run := RunRecord{
		ID:                "batch-run-" + textutil.Slugify(options.BatchID) + "-" + start.Format("20060102150405") + "-" + fmt.Sprintf("%09d", start.Nanosecond()),
		BatchID:           options.BatchID,
		Mode:              options.Mode,
		Status:            "blocked",
		Decision:          "BATCH_RUN_BLOCKED",
		RequestedBy:       options.RequestedBy,
		Approved:          options.Approved,
		MaxIssues:         options.MaxIssues,
		ContinueOnFailure: options.ContinueOnFailure,
		Items:             []RunItem{},
		Reasons:           []string{},
		StartedAt:         start.Format(time.RFC3339Nano),
	}
	plan, found, err := Load(rootDir, options.BatchID)
	if err != nil {
		return RunRecord{}, err
	}
	if !found {
		run.Reasons = append(run.Reasons, "batch_plan_not_found")
		return finishRun(rootDir, run)
	}
	run.EpicID = plan.EpicID
	options.MaxIssues = effectiveMaxIssues(options, plan)
	run.MaxIssues = options.MaxIssues
	if plan.Status != "planned" || plan.Decision != "BATCH_PLAN_READY" {
		run.Reasons = append(run.Reasons, "batch_plan_not_ready:"+plan.Decision)
		return finishRun(rootDir, run)
	}
	if options.Mode == "local_shell" && options.MaxIssues > 1 {
		run.Reasons = append(run.Reasons, "isolated_worktree_serial_execution")
	}
	items := dispatchItems(plan, options.MaxIssues)
	if len(items) == 0 {
		run.Status = "empty"
		run.Decision = "BATCH_RUN_EMPTY"
		run.Reasons = append(run.Reasons, "no_dispatch_items")
		return finishRun(rootDir, run)
	}
	switch options.Mode {
	case "dry_run":
		for _, item := range items {
			run.Items = append(run.Items, dryRunItem(item))
		}
		run.Status = "completed"
		run.Decision = "BATCH_RUN_DRY_RUN"
		run.Reasons = append(run.Reasons, "no_runtime_executed")
	case "local_shell":
		if !options.Approved {
			run.Reasons = append(run.Reasons, "batch_run_approval_required")
			return finishRun(rootDir, run)
		}
		if !batchRunEnabled() {
			run.Reasons = append(run.Reasons, "batch_run_not_enabled")
			return finishRun(rootDir, run)
		}
		if !safeBatchPrompt(options.Prompt) {
			run.Reasons = append(run.Reasons, "batch_prompt_not_allowed")
			return finishRun(rootDir, run)
		}
		run = runLocalShellBatch(ctx, rootDir, run, items, options)
	default:
		run.Reasons = append(run.Reasons, "unsupported_batch_run_mode:"+options.Mode)
	}
	return finishRun(rootDir, run)
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

func LoadRun(rootDir string, id string) (RunRecord, bool, error) {
	if !validID(id) {
		return RunRecord{}, false, nil
	}
	var run RunRecord
	found, err := fsutil.ReadJSON(filepath.Join(batchRunsDir(rootDir), id+".json"), &run)
	return run, found, err
}

func ListRuns(rootDir string, batchID string, limit int) ([]RunRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	lines, err := fsutil.TailLines(batchRunsJSONLPath(rootDir), limit*4)
	if err != nil {
		return nil, err
	}
	runs := []RunRecord{}
	for _, line := range lines {
		var run RunRecord
		if err := json.Unmarshal([]byte(line), &run); err != nil {
			return nil, err
		}
		if run.ID == "" {
			continue
		}
		if batchID != "" && run.BatchID != batchID {
			continue
		}
		runs = append(runs, run)
	}
	sort.SliceStable(runs, func(i, j int) bool {
		return runs[i].StartedAt > runs[j].StartedAt
	})
	if len(runs) > limit {
		return runs[:limit], nil
	}
	return runs, nil
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

func normalizeRunOptions(options RunOptions) RunOptions {
	options.BatchID = strings.TrimSpace(options.BatchID)
	options.Mode = strings.TrimSpace(strings.ToLower(options.Mode))
	if options.Mode == "" {
		options.Mode = "dry_run"
	}
	if options.MaxIssues <= 0 {
		options.MaxIssues = 0
	}
	if options.RequestedBy == "" {
		options.RequestedBy = "system"
	}
	return options
}

func effectiveMaxIssues(options RunOptions, plan Plan) int {
	if options.MaxIssues > 0 {
		return options.MaxIssues
	}
	if options.Mode == "dry_run" {
		if plan.DispatchCount > 0 {
			return plan.DispatchCount
		}
		return len(plan.Items)
	}
	return 1
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

func dispatchItems(plan Plan, maxIssues int) []IssueItem {
	items := []IssueItem{}
	for _, item := range plan.Items {
		if item.Decision != "dispatch" {
			continue
		}
		items = append(items, item)
		if maxIssues > 0 && len(items) >= maxIssues {
			break
		}
	}
	return items
}

func dryRunItem(item IssueItem) RunItem {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	return RunItem{
		IssueID:    item.IssueID,
		Status:     "dry_run",
		Decision:   "BATCH_ITEM_DRY_RUN",
		Reason:     "no_runtime_executed",
		RuntimeID:  item.RuntimeID,
		ProviderID: item.ProviderID,
		ModelID:    item.ModelID,
		StartedAt:  now,
		FinishedAt: now,
	}
}

func runLocalShellBatch(ctx context.Context, rootDir string, run RunRecord, items []IssueItem, options RunOptions) RunRecord {
	run.Status = "completed"
	run.Decision = "BATCH_RUN_COMPLETED"
	for _, item := range items {
		start := time.Now().UTC().Format(time.RFC3339Nano)
		runItem := RunItem{
			IssueID:    item.IssueID,
			Status:     "failed",
			Decision:   "BATCH_ITEM_FAILED",
			RuntimeID:  "local_shell",
			ProviderID: item.ProviderID,
			ModelID:    item.ModelID,
			StartedAt:  start,
			FinishedAt: time.Now().UTC().Format(time.RFC3339Nano),
		}
		wt, err := worktree.Prepare(ctx, rootDir, worktree.PrepareOptions{
			EpicID:      run.EpicID,
			BatchID:     run.BatchID,
			IssueID:     item.IssueID,
			RequestedBy: run.RequestedBy,
		})
		runItem.WorktreeID = wt.ID
		runItem.WorktreePath = wt.WorktreePath
		runItem.Branch = wt.Branch
		if err != nil {
			runItem.Reason = err.Error()
			run.Status = "failed"
			run.Decision = "BATCH_RUN_FAILED"
			run.Reasons = append(run.Reasons, "worktree_error:"+item.IssueID)
			run.Items = append(run.Items, runItem)
			if !options.ContinueOnFailure {
				break
			}
			continue
		}
		if wt.Decision != "WORKTREE_READY" {
			runItem.Decision = "BATCH_ITEM_WORKTREE_BLOCKED"
			runItem.Reason = strings.Join(wt.Reasons, ",")
			run.Status = "failed"
			run.Decision = "BATCH_RUN_FAILED"
			run.Reasons = append(run.Reasons, "worktree_blocked:"+item.IssueID)
			run.Items = append(run.Items, runItem)
			if !options.ContinueOnFailure {
				break
			}
			continue
		}
		result, err := orchestrator.RunIssueWithOptions(ctx, rootDir, item.IssueID, orchestrator.RunOptions{
			RuntimeID:    "local_shell",
			ProviderID:   item.ProviderID,
			ModelID:      item.ModelID,
			EpicID:       run.EpicID,
			Role:         item.Role,
			Prompt:       options.Prompt,
			WorktreePath: wt.WorktreePath,
			Branch:       wt.Branch,
		})
		runItem.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err != nil {
			runItem.Reason = err.Error()
			run.Status = "failed"
			run.Decision = "BATCH_RUN_FAILED"
			run.Reasons = append(run.Reasons, "issue_failed:"+item.IssueID)
			run.Items = append(run.Items, runItem)
			if !options.ContinueOnFailure {
				break
			}
			continue
		}
		runItem.RunID = result.RunID
		runItem.SubagentID = result.SubagentID
		runItem.QualityReportID = result.QualityReport.ID
		if result.Status == "accepted" {
			runItem.Status = "completed"
			runItem.Decision = "BATCH_ITEM_ACCEPTED"
		} else {
			runItem.Status = "failed"
			runItem.Decision = "BATCH_ITEM_NEEDS_REWORK"
			runItem.Reason = result.Status
			run.Status = "failed"
			run.Decision = "BATCH_RUN_FAILED"
			run.Reasons = append(run.Reasons, "issue_needs_rework:"+item.IssueID)
		}
		run.Items = append(run.Items, runItem)
		if runItem.Status == "failed" && !options.ContinueOnFailure {
			break
		}
	}
	if run.Status == "completed" {
		run.Reasons = append(run.Reasons, "batch_items_completed")
	}
	return run
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

func finishRun(rootDir string, run RunRecord) (RunRecord, error) {
	run.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := fsutil.EnsureDir(batchRunsDir(rootDir)); err != nil {
		return RunRecord{}, err
	}
	if err := fsutil.WriteJSON(filepath.Join(batchRunsDir(rootDir), run.ID+".json"), run); err != nil {
		return RunRecord{}, err
	}
	if err := fsutil.AppendJSONL(batchRunsJSONLPath(rootDir), run); err != nil {
		return RunRecord{}, err
	}
	_ = logging.Log(rootDir, "run", "batch.run.created", map[string]any{
		"batch_run_id": run.ID,
		"batch_id":     run.BatchID,
		"epic_id":      run.EpicID,
		"decision":     run.Decision,
		"status":       run.Status,
		"items":        len(run.Items),
	})
	return run, nil
}

func batchesDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "batches")
}

func batchRunsDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "batch-runs")
}

func batchesJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "batches.jsonl")
}

func batchRunsJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "batch-runs.jsonl")
}

func batchRunEnabled() bool {
	return os.Getenv("MOYUAN_ALLOW_BATCH_RUN") == "1"
}

func safeBatchPrompt(prompt string) bool {
	prompt = strings.TrimSpace(prompt)
	if strings.ContainsAny(prompt, "\n\r") {
		return false
	}
	for _, token := range []string{";", "&&", "||", "`", "$(", ">", "<", "|"} {
		if strings.Contains(prompt, token) {
			return false
		}
	}
	for _, prefix := range []string{"true", "echo ", "printf "} {
		if strings.HasSuffix(prefix, " ") {
			if strings.HasPrefix(prompt, prefix) {
				return true
			}
			continue
		}
		if prompt == prefix {
			return true
		}
	}
	return false
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
