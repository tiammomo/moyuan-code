package repair

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type DeploymentRiskReviewOptions struct {
	Decision   string `json:"decision"`
	ReviewerID string `json:"reviewer_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
	NextStep   string `json:"next_step,omitempty"`
}

type DeploymentRiskReview struct {
	ID             string   `json:"id"`
	HandoffID      string   `json:"handoff_id"`
	SourceType     string   `json:"source_type"`
	SourceID       string   `json:"source_id"`
	Decision       string   `json:"decision"`
	Status         string   `json:"status"`
	ReviewerID     string   `json:"reviewer_id,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	NextStep       string   `json:"next_step,omitempty"`
	FailureClass   string   `json:"failure_class,omitempty"`
	SignalID       string   `json:"signal_id,omitempty"`
	BugCandidateID string   `json:"bug_candidate_id,omitempty"`
	RepairPlanID   string   `json:"repair_plan_id,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
	CreatedAt      string   `json:"created_at"`
}

type DeploymentRiskReviewQueueItem struct {
	HandoffID      string   `json:"handoff_id"`
	SourceType     string   `json:"source_type"`
	SourceID       string   `json:"source_id"`
	Status         string   `json:"status"`
	Decision       string   `json:"decision"`
	FailureClass   string   `json:"failure_class"`
	ReviewRequired bool     `json:"review_required"`
	ReviewID       string   `json:"review_id,omitempty"`
	ReviewDecision string   `json:"review_decision,omitempty"`
	ReviewNextStep string   `json:"review_next_step,omitempty"`
	SignalID       string   `json:"signal_id,omitempty"`
	BugCandidateID string   `json:"bug_candidate_id,omitempty"`
	RepairPlanID   string   `json:"repair_plan_id,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
	Reasons        []string `json:"reasons,omitempty"`
	CreatedAt      string   `json:"created_at"`
	ReviewedAt     string   `json:"reviewed_at,omitempty"`
}

func ReviewDeploymentRiskHandoff(rootDir string, handoffID string, options DeploymentRiskReviewOptions) (DeploymentRiskReview, DeploymentRiskHandoff, bool, error) {
	handoffID = strings.TrimSpace(handoffID)
	options.Decision = normalizeDeploymentRiskReviewDecision(options.Decision)
	options.ReviewerID = strings.TrimSpace(options.ReviewerID)
	options.Reason = strings.TrimSpace(options.Reason)
	options.NextStep = normalizeDeploymentRiskReviewNextStep(options.NextStep, options.Decision)
	if options.Decision == "" {
		return DeploymentRiskReview{}, DeploymentRiskHandoff{}, false, errors.New("deployment_risk_review_decision_required")
	}
	handoff, found, err := LoadDeploymentRiskHandoff(rootDir, handoffID)
	if err != nil || !found {
		return DeploymentRiskReview{}, DeploymentRiskHandoff{}, found, err
	}
	if !deploymentRiskHandoffReviewable(handoff) {
		return DeploymentRiskReview{}, DeploymentRiskHandoff{}, true, errors.New("deployment_risk_handoff_not_reviewable")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	review := DeploymentRiskReview{
		ID:             "deployment-risk-review-" + textutil.Slugify(handoff.ID) + "-" + time.Now().UTC().Format("20060102150405.000000000"),
		HandoffID:      handoff.ID,
		SourceType:     handoff.SourceType,
		SourceID:       handoff.SourceID,
		Decision:       options.Decision,
		Status:         "completed",
		ReviewerID:     options.ReviewerID,
		Reason:         options.Reason,
		NextStep:       options.NextStep,
		FailureClass:   handoff.FailureClass,
		SignalID:       handoff.SignalID,
		BugCandidateID: handoff.BugCandidateID,
		RepairPlanID:   handoff.RepairPlanID,
		EvidenceRefs:   append([]string{}, handoff.EvidenceRefs...),
		CreatedAt:      now,
	}
	handoff.ReviewID = review.ID
	handoff.ReviewedAt = now
	handoff.ReviewedBy = options.ReviewerID
	handoff.ReviewDecision = options.Decision
	handoff.ReviewReason = options.Reason
	handoff.ReviewNextStep = options.NextStep
	switch options.Decision {
	case "approved":
		handoff.Status = "review_approved"
		handoff.Decision = "DEPLOYMENT_RISK_REVIEW_APPROVED"
		handoff.ReviewRequired = false
		handoff.Reasons = appendUniqueString(handoff.Reasons, "risk_review_approved")
	case "rejected":
		handoff.Status = "review_rejected"
		handoff.Decision = "DEPLOYMENT_RISK_REVIEW_REJECTED"
		handoff.ReviewRequired = false
		handoff.Reasons = appendUniqueString(handoff.Reasons, "risk_review_rejected")
	case "deferred":
		handoff.Status = "review_deferred"
		handoff.Decision = "DEPLOYMENT_RISK_REVIEW_DEFERRED"
		handoff.ReviewRequired = true
		handoff.Reasons = appendUniqueString(handoff.Reasons, "risk_review_deferred")
	}
	record, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "deployment_risk_review",
		ParentID:    review.ID,
		SubjectType: handoff.SourceType,
		SubjectID:   handoff.SourceID,
		Operation:   "deployment.risk.review",
		Status:      handoff.Status,
		Decision:    handoff.Decision,
		Reasons:     []string{options.Decision, options.Reason, options.NextStep},
		Source:      "repair",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "deployment_risk_review",
			ID:   review.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "repair", "deployment-risk-reviews", review.ID+".json")),
		}},
	})
	if err != nil {
		return DeploymentRiskReview{}, DeploymentRiskHandoff{}, true, err
	}
	review.EvidenceRefs = appendUniqueString(review.EvidenceRefs, record.ID)
	handoff.EvidenceRefs = appendUniqueString(handoff.EvidenceRefs, record.ID)
	if err := saveDeploymentRiskReview(rootDir, review); err != nil {
		return DeploymentRiskReview{}, DeploymentRiskHandoff{}, true, err
	}
	handoff, err = saveDeploymentRiskHandoff(rootDir, handoff)
	if err != nil {
		return DeploymentRiskReview{}, DeploymentRiskHandoff{}, true, err
	}
	_ = logging.Log(rootDir, "run", "self_repair.deployment_risk_handoff.reviewed", map[string]any{
		"handoff_id":  handoff.ID,
		"review_id":   review.ID,
		"decision":    options.Decision,
		"reviewer_id": options.ReviewerID,
		"next_step":   options.NextStep,
	})
	return review, handoff, true, nil
}

func ListDeploymentRiskReviewQueue(rootDir string, status string, limit int) ([]DeploymentRiskReviewQueueItem, error) {
	handoffs, err := ListDeploymentRiskHandoffs(rootDir, 1000)
	if err != nil {
		return nil, err
	}
	status = normalizeReviewQueueStatus(status)
	items := []DeploymentRiskReviewQueueItem{}
	for _, handoff := range handoffs {
		item := DeploymentRiskReviewQueueItem{
			HandoffID:      handoff.ID,
			SourceType:     handoff.SourceType,
			SourceID:       handoff.SourceID,
			Status:         handoff.Status,
			Decision:       handoff.Decision,
			FailureClass:   handoff.FailureClass,
			ReviewRequired: handoff.ReviewRequired,
			ReviewID:       handoff.ReviewID,
			ReviewDecision: handoff.ReviewDecision,
			ReviewNextStep: handoff.ReviewNextStep,
			SignalID:       handoff.SignalID,
			BugCandidateID: handoff.BugCandidateID,
			RepairPlanID:   handoff.RepairPlanID,
			EvidenceRefs:   append([]string{}, handoff.EvidenceRefs...),
			Reasons:        append([]string{}, handoff.Reasons...),
			CreatedAt:      handoff.CreatedAt,
			ReviewedAt:     handoff.ReviewedAt,
		}
		if !queueItemMatchesStatus(item, status) {
			continue
		}
		items = append(items, item)
	}
	if limit <= 0 {
		limit = 20
	}
	if len(items) > limit {
		return items[:limit], nil
	}
	return items, nil
}

func LoadDeploymentRiskReview(rootDir string, id string) (DeploymentRiskReview, bool, error) {
	var review DeploymentRiskReview
	found, err := fsutil.ReadJSON(deploymentRiskReviewPath(rootDir, id), &review)
	return review, found, err
}

func ListDeploymentRiskReviews(rootDir string, limit int) ([]DeploymentRiskReview, error) {
	if err := fsutil.EnsureDir(deploymentRiskReviewDir(rootDir)); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(deploymentRiskReviewDir(rootDir))
	if err != nil {
		return nil, err
	}
	reviews := []DeploymentRiskReview{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var review DeploymentRiskReview
		found, err := fsutil.ReadJSON(filepath.Join(deploymentRiskReviewDir(rootDir), entry.Name()), &review)
		if err != nil {
			return nil, err
		}
		if found && review.ID != "" {
			reviews = append(reviews, review)
		}
	}
	sort.SliceStable(reviews, func(i, j int) bool {
		return reviews[i].CreatedAt > reviews[j].CreatedAt
	})
	if limit <= 0 {
		limit = 20
	}
	if len(reviews) > limit {
		return reviews[:limit], nil
	}
	return reviews, nil
}

func saveDeploymentRiskReview(rootDir string, review DeploymentRiskReview) error {
	if err := fsutil.WriteJSON(deploymentRiskReviewPath(rootDir, review.ID), review); err != nil {
		return err
	}
	return fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).RepairDir, "deployment-risk-reviews.jsonl"), review)
}

func deploymentRiskHandoffReviewable(handoff DeploymentRiskHandoff) bool {
	if handoff.Status == "review_required" || handoff.Status == "review_deferred" {
		return true
	}
	return handoff.ReviewRequired && handoff.Status != "ignored" && handoff.Status != "review_approved" && handoff.Status != "review_rejected"
}

func normalizeDeploymentRiskReviewDecision(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "approve", "approved":
		return "approved"
	case "reject", "rejected":
		return "rejected"
	case "defer", "deferred":
		return "deferred"
	default:
		return ""
	}
}

func normalizeDeploymentRiskReviewNextStep(value string, decision string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch decision {
	case "approved":
		switch value {
		case "repair_plan", "create_issue", "monitor_only", "manual_followup":
			return value
		default:
			return "repair_plan"
		}
	case "rejected":
		if value == "archive" {
			return value
		}
		return "archive"
	case "deferred":
		switch value {
		case "wait_for_signal", "need_evidence", "manual_followup":
			return value
		default:
			return "need_evidence"
		}
	default:
		return value
	}
}

func normalizeReviewQueueStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "pending":
		return "pending"
	case "reviewed", "completed":
		return "reviewed"
	case "all":
		return "all"
	default:
		return "pending"
	}
}

func queueItemMatchesStatus(item DeploymentRiskReviewQueueItem, status string) bool {
	switch status {
	case "all":
		return true
	case "reviewed":
		return item.ReviewID != "" && !item.ReviewRequired
	default:
		return item.ReviewRequired || item.Status == "review_required" || item.Status == "review_deferred"
	}
}

func appendUniqueString(values []string, next ...string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range next {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		values = append(values, value)
		seen[value] = true
	}
	return values
}

func deploymentRiskReviewDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).RepairDir, "deployment-risk-reviews")
}

func deploymentRiskReviewPath(rootDir string, id string) string {
	return filepath.Join(deploymentRiskReviewDir(rootDir), id+".json")
}
