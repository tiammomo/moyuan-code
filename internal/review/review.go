package review

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"time"

	"moyuan-code/internal/batch"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/quality"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type MergeDecision struct {
	ID              string   `json:"id"`
	IssueID         string   `json:"issue_id"`
	Status          string   `json:"status"`
	Decision        string   `json:"decision"`
	Reasons         []string `json:"reasons"`
	IssueStatus     string   `json:"issue_status,omitempty"`
	QualityReportID string   `json:"quality_report_id,omitempty"`
	QualityStatus   string   `json:"quality_status,omitempty"`
	ReviewStatus    string   `json:"review_status,omitempty"`
	CreatedAt       string   `json:"created_at"`
}

type MergeQueue struct {
	ID               string           `json:"id"`
	BatchID          string           `json:"batch_id"`
	EpicID           string           `json:"epic_id,omitempty"`
	BatchRunID       string           `json:"batch_run_id,omitempty"`
	Status           string           `json:"status"`
	Decision         string           `json:"decision"`
	Reasons          []string         `json:"reasons"`
	ReadyCount       int              `json:"ready_count"`
	NeedsReworkCount int              `json:"needs_rework_count"`
	BlockedCount     int              `json:"blocked_count"`
	Items            []MergeQueueItem `json:"items"`
	CreatedAt        string           `json:"created_at"`
}

type MergeQueueItem struct {
	IssueID         string        `json:"issue_id"`
	Status          string        `json:"status"`
	Decision        string        `json:"decision"`
	Reason          string        `json:"reason,omitempty"`
	RunID           string        `json:"run_id,omitempty"`
	SubagentID      string        `json:"subagent_id,omitempty"`
	QualityReportID string        `json:"quality_report_id,omitempty"`
	WorktreeID      string        `json:"worktree_id,omitempty"`
	WorktreePath    string        `json:"worktree_path,omitempty"`
	Branch          string        `json:"branch,omitempty"`
	MergeDecision   MergeDecision `json:"merge_decision,omitempty"`
}

func DecideMerge(rootDir string, issueID string) (MergeDecision, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	decision := MergeDecision{
		ID:        "merge-" + issueID + "-" + time.Now().UTC().Format("20060102150405"),
		IssueID:   issueID,
		Status:    "blocked",
		Decision:  "MERGE_BLOCKED",
		Reasons:   []string{},
		CreatedAt: now,
	}
	issueState, found, err := orchestrator.LoadIssueState(rootDir, issueID)
	if err != nil {
		return MergeDecision{}, err
	}
	if !found {
		decision.Reasons = append(decision.Reasons, "issue_state_missing")
		return finish(rootDir, decision)
	}
	decision.IssueStatus = issueState.Status
	decision.QualityReportID = issueState.QualityReportID
	if issueState.Status != "accepted" {
		decision.Reasons = append(decision.Reasons, "issue_not_accepted")
		return finish(rootDir, decision)
	}
	if issueState.QualityReportID == "" {
		decision.Reasons = append(decision.Reasons, "quality_report_missing")
		return finish(rootDir, decision)
	}
	report, ok, err := quality.Read(rootDir, issueState.QualityReportID)
	if err != nil {
		return MergeDecision{}, err
	}
	if !ok {
		decision.Reasons = append(decision.Reasons, "quality_report_missing")
		return finish(rootDir, decision)
	}
	decision.QualityStatus = report.Status
	decision.ReviewStatus = report.ReviewStatus
	if report.Status != "passed" {
		decision.Status = "needs_rework"
		decision.Decision = "NEEDS_REWORK"
		decision.Reasons = append(decision.Reasons, "quality_not_passed")
		return finish(rootDir, decision)
	}
	if report.ReviewStatus == "rejected" {
		decision.Status = "needs_rework"
		decision.Decision = "NEEDS_REWORK"
		decision.Reasons = append(decision.Reasons, "review_rejected")
		return finish(rootDir, decision)
	}
	decision.Status = "ready_to_merge"
	decision.Decision = "MERGE_ALLOWED"
	decision.Reasons = append(decision.Reasons, "quality_and_review_accepted")
	return finish(rootDir, decision)
}

func BuildMergeQueue(rootDir string, batchID string) (MergeQueue, error) {
	now := time.Now().UTC()
	queue := MergeQueue{
		ID:        "merge-queue-" + textutil.Slugify(batchID) + "-" + now.Format("20060102150405"),
		BatchID:   batchID,
		Status:    "blocked",
		Decision:  "MERGE_QUEUE_BLOCKED",
		Reasons:   []string{},
		Items:     []MergeQueueItem{},
		CreatedAt: now.Format(time.RFC3339Nano),
	}
	plan, found, err := batch.Load(rootDir, batchID)
	if err != nil {
		return MergeQueue{}, err
	}
	if !found {
		queue.Reasons = append(queue.Reasons, "batch_plan_missing")
		return finishMergeQueue(rootDir, queue)
	}
	queue.EpicID = plan.EpicID
	runs, err := batch.ListRuns(rootDir, batchID, 1)
	if err != nil {
		return MergeQueue{}, err
	}
	if len(runs) == 0 {
		queue.Reasons = append(queue.Reasons, "batch_run_missing")
		return finishMergeQueue(rootDir, queue)
	}
	run := runs[0]
	queue.BatchRunID = run.ID
	for _, item := range run.Items {
		queueItem, err := mergeQueueItem(rootDir, item)
		if err != nil {
			return MergeQueue{}, err
		}
		queue.Items = append(queue.Items, queueItem)
		switch queueItem.Status {
		case "ready_to_merge":
			queue.ReadyCount++
		case "needs_rework":
			queue.NeedsReworkCount++
		default:
			queue.BlockedCount++
		}
	}
	switch {
	case len(queue.Items) == 0:
		queue.Status = "empty"
		queue.Decision = "MERGE_QUEUE_EMPTY"
		queue.Reasons = append(queue.Reasons, "batch_run_empty")
	case queue.NeedsReworkCount > 0:
		queue.Status = "needs_rework"
		queue.Decision = "MERGE_QUEUE_NEEDS_REWORK"
		queue.Reasons = append(queue.Reasons, "items_need_rework")
	case queue.BlockedCount > 0:
		queue.Status = "blocked"
		queue.Decision = "MERGE_QUEUE_BLOCKED"
		queue.Reasons = append(queue.Reasons, "items_blocked")
	default:
		queue.Status = "ready_to_merge"
		queue.Decision = "MERGE_QUEUE_READY"
		queue.Reasons = append(queue.Reasons, "all_items_ready_to_merge")
	}
	return finishMergeQueue(rootDir, queue)
}

func LoadMergeQueue(rootDir string, id string) (MergeQueue, bool, error) {
	var queue MergeQueue
	found, err := fsutil.ReadJSON(mergeQueuePath(rootDir, id), &queue)
	return queue, found, err
}

func ListMergeQueues(rootDir string, batchID string, limit int) ([]MergeQueue, error) {
	if limit <= 0 {
		limit = 20
	}
	lines, err := fsutil.TailLines(mergeQueuesJSONLPath(rootDir), limit*4)
	if err != nil {
		return nil, err
	}
	queues := []MergeQueue{}
	for _, line := range lines {
		var queue MergeQueue
		if err := json.Unmarshal([]byte(line), &queue); err != nil {
			return nil, err
		}
		if queue.ID == "" {
			continue
		}
		if batchID != "" && queue.BatchID != batchID {
			continue
		}
		queues = append(queues, queue)
	}
	sort.SliceStable(queues, func(i, j int) bool {
		return queues[i].CreatedAt > queues[j].CreatedAt
	})
	if len(queues) > limit {
		return queues[:limit], nil
	}
	return queues, nil
}

func Load(rootDir string, id string) (MergeDecision, bool, error) {
	var decision MergeDecision
	found, err := fsutil.ReadJSON(decisionPath(rootDir, id), &decision)
	return decision, found, err
}

func finish(rootDir string, decision MergeDecision) (MergeDecision, error) {
	if err := fsutil.WriteJSON(decisionPath(rootDir, decision.ID), decision); err != nil {
		return MergeDecision{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).ReviewsDir, "merge-decisions.jsonl"), decision); err != nil {
		return MergeDecision{}, err
	}
	_ = logging.Log(rootDir, "quality", "review.merge_decision.created", map[string]any{"issue_id": decision.IssueID, "merge_decision_id": decision.ID, "decision": decision.Decision, "status": decision.Status})
	return decision, nil
}

func mergeQueueItem(rootDir string, item batch.RunItem) (MergeQueueItem, error) {
	queueItem := MergeQueueItem{
		IssueID:         item.IssueID,
		Status:          "blocked",
		Decision:        "MERGE_QUEUE_ITEM_BLOCKED",
		RunID:           item.RunID,
		SubagentID:      item.SubagentID,
		QualityReportID: item.QualityReportID,
		WorktreeID:      item.WorktreeID,
		WorktreePath:    item.WorktreePath,
		Branch:          item.Branch,
	}
	if item.Status != "completed" || item.Decision != "BATCH_ITEM_ACCEPTED" {
		queueItem.Status = "needs_rework"
		queueItem.Decision = "MERGE_QUEUE_ITEM_NEEDS_REWORK"
		queueItem.Reason = "batch_item_not_accepted:" + item.Decision
		if item.Status == "dry_run" {
			queueItem.Status = "blocked"
			queueItem.Decision = "MERGE_QUEUE_ITEM_BLOCKED"
			queueItem.Reason = "batch_item_dry_run"
		}
		return queueItem, nil
	}
	decision, err := DecideMerge(rootDir, item.IssueID)
	if err != nil {
		return MergeQueueItem{}, err
	}
	queueItem.MergeDecision = decision
	queueItem.QualityReportID = decision.QualityReportID
	switch decision.Decision {
	case "MERGE_ALLOWED":
		queueItem.Status = "ready_to_merge"
		queueItem.Decision = "MERGE_QUEUE_ITEM_READY"
		queueItem.Reason = "quality_and_review_accepted"
	case "NEEDS_REWORK":
		queueItem.Status = "needs_rework"
		queueItem.Decision = "MERGE_QUEUE_ITEM_NEEDS_REWORK"
		queueItem.Reason = "merge_decision_needs_rework"
	default:
		queueItem.Status = "blocked"
		queueItem.Decision = "MERGE_QUEUE_ITEM_BLOCKED"
		queueItem.Reason = "merge_decision_blocked"
	}
	return queueItem, nil
}

func finishMergeQueue(rootDir string, queue MergeQueue) (MergeQueue, error) {
	if err := fsutil.WriteJSON(mergeQueuePath(rootDir, queue.ID), queue); err != nil {
		return MergeQueue{}, err
	}
	if err := fsutil.AppendJSONL(mergeQueuesJSONLPath(rootDir), queue); err != nil {
		return MergeQueue{}, err
	}
	_ = logging.Log(rootDir, "quality", "review.merge_queue.created", map[string]any{
		"merge_queue_id": queue.ID,
		"batch_id":       queue.BatchID,
		"batch_run_id":   queue.BatchRunID,
		"decision":       queue.Decision,
		"status":         queue.Status,
		"ready":          queue.ReadyCount,
		"needs_rework":   queue.NeedsReworkCount,
		"blocked":        queue.BlockedCount,
	})
	return queue, nil
}

func decisionPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReviewsDir, "merge-decisions", id+".json")
}

func mergeQueuePath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).MergeReportsDir, "queues", id+".json")
}

func mergeQueuesJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).MergeReportsDir, "merge-queues.jsonl")
}
