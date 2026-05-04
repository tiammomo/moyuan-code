package review

import (
	"path/filepath"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/quality"
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

func decisionPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReviewsDir, "merge-decisions", id+".json")
}
