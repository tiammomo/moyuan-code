package scheduler

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/subagent"
	"moyuan-code/internal/workspace"
)

type Plan struct {
	EpicID          string                `json:"epic_id"`
	ReadyQueue      []string              `json:"ready_queue"`
	BlockedQueue    []string              `json:"blocked_queue"`
	RunningQueue    []string              `json:"running_queue"`
	ReviewQueue     []string              `json:"review_queue"`
	DispatchQueue   []DispatchDecision    `json:"dispatch_queue"`
	WaitingQueue    []DispatchDecision    `json:"waiting_queue"`
	SubagentBacklog []SubagentBacklogItem `json:"subagent_backlog"`
	BlockedReason   map[string]string     `json:"blocked_reason"`
	Parallelism     int                   `json:"parallelism"`
	MaxParallel     int                   `json:"max_parallel"`
	RuntimeSlots    int                   `json:"runtime_slots"`
	CreatedAt       string                `json:"created_at"`
}

type DispatchDecision struct {
	IssueID        string   `json:"issue_id"`
	Decision       string   `json:"decision"`
	Reason         string   `json:"reason,omitempty"`
	Role           string   `json:"role"`
	RuntimeID      string   `json:"runtime_id"`
	WriteScopes    []string `json:"write_scopes"`
	ConflictsWith  []string `json:"conflicts_with,omitempty"`
	DependencyIDs  []string `json:"dependency_ids,omitempty"`
	SubagentID     string   `json:"subagent_id,omitempty"`
	SubagentStatus string   `json:"subagent_status,omitempty"`
	RecoveryID     string   `json:"recovery_id,omitempty"`
	RetryCount     int      `json:"retry_count,omitempty"`
	MaxRetries     int      `json:"max_retries,omitempty"`
}

type SubagentBacklogItem struct {
	IssueID         string `json:"issue_id"`
	SubagentID      string `json:"subagent_id"`
	Status          string `json:"status"`
	Reason          string `json:"reason,omitempty"`
	RecoveryID      string `json:"recovery_id,omitempty"`
	FailureCategory string `json:"failure_category,omitempty"`
	RetryCount      int    `json:"retry_count"`
	MaxRetries      int    `json:"max_retries"`
}

func Build(rootDir string, epicID string, maxParallel int) (Plan, error) {
	if maxParallel <= 0 {
		maxParallel = 1
	}
	graph, ok, err := issues.LoadGraph(rootDir, epicID)
	if err != nil {
		return Plan{}, err
	}
	if !ok {
		return emptyPlan(epicID, maxParallel), nil
	}
	plan := Plan{
		EpicID:          epicID,
		ReadyQueue:      []string{},
		BlockedQueue:    []string{},
		RunningQueue:    []string{},
		ReviewQueue:     []string{},
		SubagentBacklog: []SubagentBacklogItem{},
		BlockedReason:   map[string]string{},
		MaxParallel:     maxParallel,
		RuntimeSlots:    maxParallel,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339Nano),
	}
	signals, backlog, err := subagentSignals(rootDir)
	if err != nil {
		return Plan{}, err
	}
	plan.SubagentBacklog = backlog
	for _, node := range graph.Nodes {
		switch node.Status {
		case "ready":
			plan.ReadyQueue = append(plan.ReadyQueue, node.ID)
		case "blocked":
			plan.BlockedQueue = append(plan.BlockedQueue, node.ID)
			plan.BlockedReason[node.ID] = blockedReasonFor(node)
		case "running", "quality_checking", "verifying":
			plan.RunningQueue = append(plan.RunningQueue, node.ID)
		case "reviewing":
			plan.ReviewQueue = append(plan.ReviewQueue, node.ID)
		}
	}
	plan.DispatchQueue, plan.WaitingQueue = dispatchDecisions(graph, maxParallel, signals)
	plan.Parallelism = len(plan.DispatchQueue)
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(rootDir).SchedulerDir, epicID+"-plan.json"), plan); err != nil {
		return Plan{}, err
	}
	_ = logging.Log(rootDir, "run", "scheduler.plan.created", map[string]any{"epic_id": epicID, "ready": len(plan.ReadyQueue), "parallelism": plan.Parallelism})
	return plan, nil
}

func emptyPlan(epicID string, maxParallel int) Plan {
	return Plan{
		EpicID:          epicID,
		ReadyQueue:      []string{},
		BlockedQueue:    []string{},
		RunningQueue:    []string{},
		ReviewQueue:     []string{},
		DispatchQueue:   []DispatchDecision{},
		WaitingQueue:    []DispatchDecision{},
		SubagentBacklog: []SubagentBacklogItem{},
		BlockedReason:   map[string]string{},
		Parallelism:     0,
		MaxParallel:     maxParallel,
		RuntimeSlots:    maxParallel,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func dispatchDecisions(graph issues.Graph, maxParallel int, signals map[string]subagent.Instance) ([]DispatchDecision, []DispatchDecision) {
	dispatch := []DispatchDecision{}
	waiting := []DispatchDecision{}
	claimedScopes := map[string]string{}
	for _, node := range graph.Nodes {
		if node.Status != "ready" {
			continue
		}
		decision := decisionFor(node)
		if signal, ok := signals[node.ID]; ok {
			decision.SubagentID = signal.ID
			decision.SubagentStatus = signal.Status
			decision.RecoveryID = signal.RecoveryID
			decision.RetryCount = signal.RetryCount
			decision.MaxRetries = signal.MaxRetries
			if signal.Status == "waiting_runtime" {
				decision.Decision = "waiting"
				decision.Reason = "subagent_waiting_runtime"
				waiting = append(waiting, decision)
				continue
			}
			if signal.Status == "needs_rework" {
				decision.Decision = "waiting"
				decision.Reason = "subagent_needs_rework"
				waiting = append(waiting, decision)
				continue
			}
			if signal.Status == "archived" && signal.MaxRetries > 0 && signal.RetryCount >= signal.MaxRetries {
				decision.Decision = "waiting"
				decision.Reason = "subagent_retry_exhausted"
				waiting = append(waiting, decision)
				continue
			}
			if signal.Status == "retrying" {
				decision.Reason = "subagent_retrying"
			}
		}
		if len(dispatch) >= maxParallel {
			decision.Decision = "waiting"
			decision.Reason = "runtime_slot"
			waiting = append(waiting, decision)
			continue
		}
		conflicts := conflictingIssues(decision.WriteScopes, claimedScopes)
		if len(conflicts) > 0 {
			decision.Decision = "waiting"
			decision.Reason = "write_scope_conflict"
			decision.ConflictsWith = conflicts
			waiting = append(waiting, decision)
			continue
		}
		decision.Decision = "dispatch"
		dispatch = append(dispatch, decision)
		for _, scope := range decision.WriteScopes {
			claimedScopes[scope] = decision.IssueID
		}
	}
	return dispatch, waiting
}

func subagentSignals(rootDir string) (map[string]subagent.Instance, []SubagentBacklogItem, error) {
	instances, err := subagent.List(rootDir, 200)
	if err != nil {
		return nil, nil, err
	}
	signals := map[string]subagent.Instance{}
	backlog := []SubagentBacklogItem{}
	for _, instance := range instances {
		if instance.IssueID == "" {
			continue
		}
		switch instance.Status {
		case "archived", "waiting_runtime", "retrying", "needs_rework":
			if _, exists := signals[instance.IssueID]; !exists {
				signals[instance.IssueID] = instance
			}
			backlog = append(backlog, SubagentBacklogItem{
				IssueID:         instance.IssueID,
				SubagentID:      instance.ID,
				Status:          instance.Status,
				Reason:          backlogReason(instance),
				RecoveryID:      instance.RecoveryID,
				FailureCategory: instance.FailureCategory,
				RetryCount:      instance.RetryCount,
				MaxRetries:      instance.MaxRetries,
			})
		}
	}
	return signals, backlog, nil
}

func backlogReason(instance subagent.Instance) string {
	if instance.ArchiveReason != "" {
		return instance.ArchiveReason
	}
	if instance.BlockedReason != "" {
		return instance.BlockedReason
	}
	if instance.FailureCategory != "" {
		return instance.FailureCategory
	}
	return instance.Status
}

func decisionFor(node issues.Node) DispatchDecision {
	role := roleFor(node)
	return DispatchDecision{
		IssueID:       node.ID,
		Decision:      "ready",
		Role:          role,
		RuntimeID:     runtimeFor(role),
		WriteScopes:   writeScopesFor(node),
		DependencyIDs: node.DependsOn,
	}
}

func roleFor(node issues.Node) string {
	if strings.TrimSpace(node.Role) != "" {
		return strings.TrimSpace(node.Role)
	}
	text := strings.ToLower(node.Title)
	switch {
	case strings.Contains(text, "frontend") || strings.Contains(text, "ui"):
		return "frontend"
	case strings.Contains(text, "tuning") || strings.Contains(text, "performance"):
		return "backend_tuning"
	case strings.Contains(text, "quality") || strings.Contains(text, "review"):
		return "quality_owner"
	case strings.Contains(text, "contract") || strings.Contains(text, "design"):
		return "architect"
	default:
		return "backend"
	}
}

func runtimeFor(role string) string {
	return providers.DefaultRuntimeForRole(role)
}

func writeScopesFor(node issues.Node) []string {
	text := strings.ToLower(node.Title + " " + node.ID)
	scopes := []string{"internal"}
	if strings.Contains(text, "frontend") || strings.Contains(text, "ui") {
		scopes = []string{"web", "src"}
	}
	if strings.Contains(text, "docs") || strings.Contains(text, "contract") || strings.Contains(text, "design") {
		scopes = append(scopes, "docs")
	}
	if strings.Contains(text, "quality") || strings.Contains(text, "review") {
		scopes = []string{".moyuan/lifecycle/quality"}
	}
	return unique(scopes)
}

func conflictingIssues(scopes []string, claimed map[string]string) []string {
	conflicts := []string{}
	seen := map[string]bool{}
	for _, scope := range scopes {
		if owner, ok := claimed[scope]; ok && !seen[owner] {
			conflicts = append(conflicts, owner)
			seen[owner] = true
		}
	}
	sort.Strings(conflicts)
	return conflicts
}

func blockedReasonFor(node issues.Node) string {
	if len(node.DependsOn) == 0 {
		return "blocked"
	}
	return "waiting_dependencies"
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
	sort.Strings(result)
	return result
}
