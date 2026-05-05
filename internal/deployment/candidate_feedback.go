package deployment

import (
	"strings"
	"time"

	"moyuan-code/internal/release"
	"moyuan-code/internal/workspace"
)

type CandidateDeploymentFeedback struct {
	ID                 string                  `json:"id"`
	CandidateID        string                  `json:"candidate_id"`
	Status             string                  `json:"status"`
	Decision           string                  `json:"decision"`
	FailureClass       string                  `json:"failure_class,omitempty"`
	Severity           string                  `json:"severity,omitempty"`
	LatestExecutionID  string                  `json:"latest_execution_id,omitempty"`
	LatestDeploymentID string                  `json:"latest_deployment_id,omitempty"`
	Environment        string                  `json:"environment,omitempty"`
	HistoryCount       int                     `json:"history_count"`
	RollbackRequired   bool                    `json:"rollback_required"`
	Histories          []PostDeploymentHistory `json:"histories"`
	EvidenceIDs        []string                `json:"evidence_ids,omitempty"`
	Reasons            []string                `json:"reasons"`
	CreatedAt          string                  `json:"created_at"`
}

func FeedbackForCandidate(rootDir string, candidateID string, limit int) (CandidateDeploymentFeedback, bool, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return CandidateDeploymentFeedback{}, false, err
	}
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return CandidateDeploymentFeedback{}, false, nil
	}
	if _, found, err := release.LoadCandidate(rootDir, candidateID); err != nil || !found {
		return CandidateDeploymentFeedback{}, found, err
	}
	if limit <= 0 {
		limit = 10
	}
	histories, err := ListPostDeploymentHistories(rootDir, limit*4)
	if err != nil {
		return CandidateDeploymentFeedback{}, true, err
	}
	filtered := []PostDeploymentHistory{}
	for _, history := range histories {
		if history.ReleaseID != candidateID {
			continue
		}
		filtered = append(filtered, history)
		if len(filtered) >= limit {
			break
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	feedback := CandidateDeploymentFeedback{
		ID:           "candidate-deployment-feedback-" + candidateID,
		CandidateID:  candidateID,
		Status:       "pending",
		Decision:     "CANDIDATE_DEPLOYMENT_FEEDBACK_PENDING",
		Histories:    filtered,
		HistoryCount: len(filtered),
		Reasons:      []string{},
		CreatedAt:    now,
	}
	if len(filtered) == 0 {
		feedback.Reasons = append(feedback.Reasons, "deployment_execution_missing")
		return feedback, true, nil
	}
	latest := filtered[0]
	feedback.Status = latest.Status
	feedback.Decision = candidateFeedbackDecision(latest)
	feedback.FailureClass = latest.FailureClass
	feedback.Severity = latest.Severity
	feedback.LatestExecutionID = latest.ExecutionID
	feedback.LatestDeploymentID = latest.DeploymentID
	feedback.Environment = latest.Environment
	feedback.RollbackRequired = latest.Rollback.Required
	feedback.EvidenceIDs = append([]string{}, latest.EvidenceIDs...)
	feedback.Reasons = append(feedback.Reasons, latest.Reasons...)
	return feedback, true, nil
}

func candidateFeedbackDecision(history PostDeploymentHistory) string {
	switch history.Status {
	case "passed":
		return "CANDIDATE_DEPLOYMENT_HEALTHY"
	case "failed":
		return "CANDIDATE_DEPLOYMENT_CHECKS_FAILED"
	case "blocked":
		return "CANDIDATE_DEPLOYMENT_BLOCKED"
	case "manual_required":
		return "CANDIDATE_DEPLOYMENT_MANUAL_REVIEW_REQUIRED"
	case "skipped":
		return "CANDIDATE_DEPLOYMENT_NOT_APPLICABLE"
	default:
		return "CANDIDATE_DEPLOYMENT_FEEDBACK_REVIEW"
	}
}
