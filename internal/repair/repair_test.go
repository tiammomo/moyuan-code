package repair

import (
	"context"
	"testing"

	"moyuan-code/internal/deployment"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/workspace"
)

func TestCandidateFromFailedOperationCreatesReviewOnlyPlan(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	record, err := evidence.Add(root, evidence.AddOptions{
		ParentType:  "deployment_execution",
		ParentID:    "deploy-exec-smoke-failed",
		SubjectType: "deployment",
		SubjectID:   "deployment-smoke-failed",
		Operation:   "deployment.smoke.check",
		Status:      "failed",
		Decision:    "SMOKE_FAILED",
		Reasons:     []string{"dev-api:failed:healthcheck_http_status:500"},
		Source:      "deployment",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "smoke_report",
			ID:   "deploy-exec-smoke-failed",
			Path: ".moyuan/lifecycle/deployments/executions/deploy-exec-smoke-failed.json",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	operationCandidate, found, err := CandidateFromOperation(root, "evidence", record.ID)
	if err != nil || !found {
		t.Fatalf("expected operation repair candidate, found=%v err=%v", found, err)
	}
	if operationCandidate.Decision != "REPAIR_CANDIDATE_CREATED" || operationCandidate.Status != "review_required" || operationCandidate.FailureClass != "smoke_failed" {
		t.Fatalf("unexpected operation repair candidate: %+v", operationCandidate)
	}
	if operationCandidate.Signal == nil || operationCandidate.Signal.SignalType != "smoke_failure" || len(operationCandidate.Signal.EvidenceRefs) != 1 {
		t.Fatalf("expected smoke failure signal with evidence refs, got %+v", operationCandidate.Signal)
	}
	if operationCandidate.Candidate == nil || operationCandidate.Candidate.Classification != "NEEDS_EVIDENCE" {
		t.Fatalf("expected review candidate classification, got %+v", operationCandidate.Candidate)
	}
	if operationCandidate.Plan == nil || !operationCandidate.Plan.RequiresApproval || operationCandidate.Plan.Status != "candidate_review_required" || operationCandidate.Plan.Strategy != "review_repair_candidate" {
		t.Fatalf("expected review-only repair plan, got %+v", operationCandidate.Plan)
	}
	loaded, ok, err := LoadOperationRepairCandidate(root, operationCandidate.ID)
	if err != nil || !ok || loaded.ID != operationCandidate.ID {
		t.Fatalf("expected operation repair candidate to load, ok=%v loaded=%+v err=%v", ok, loaded, err)
	}
	list, err := ListOperationRepairCandidates(root, 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected operation repair candidate list, list=%+v err=%v", list, err)
	}
	review, reviewedCandidate, attempt, found, err := ReviewOperationRepairCandidate(context.Background(), root, operationCandidate.ID, OperationRepairReviewOptions{
		Decision:   "approved",
		ReviewerID: "qa-owner",
		Reason:     "evidence chain is enough to open a controlled repair task",
		NextStep:   "repair_attempt",
	})
	if err != nil || !found {
		t.Fatalf("expected operation repair review, found=%v review=%+v err=%v", found, review, err)
	}
	if review.Decision != "approved" || review.IssueID == "" || review.RepairAttemptID == "" {
		t.Fatalf("unexpected review result: %+v", review)
	}
	if reviewedCandidate.Status != "approved" || reviewedCandidate.Decision != "REPAIR_CANDIDATE_APPROVED" || reviewedCandidate.IssueID == "" {
		t.Fatalf("unexpected reviewed candidate: %+v", reviewedCandidate)
	}
	if attempt == nil || attempt.Status != "review_ready" || attempt.RuntimeID != "review_only" {
		t.Fatalf("expected review-only repair attempt, got %+v", attempt)
	}
	plan, err := LoadPlan(root, operationCandidate.RepairPlanID)
	if err != nil || plan.IssueID != review.IssueID || plan.Status != "review_approved" {
		t.Fatalf("expected review to bind repair plan to issue, plan=%+v err=%v", plan, err)
	}
	list, err = ListOperationRepairCandidates(root, 10)
	if err != nil || len(list) != 1 || list[0].Status != "approved" {
		t.Fatalf("expected deduped approved candidate list, list=%+v err=%v", list, err)
	}
}

func TestDeploymentRiskHandoffCreatesReviewRepairPlan(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	admission, err := deployment.BuildReleaseAdmission(context.Background(), root, deployment.ReleaseAdmissionOptions{RehearsalID: "missing-rehearsal"})
	if err != nil {
		t.Fatal(err)
	}
	if admission.Status != "blocked" {
		t.Fatalf("expected blocked admission fixture, got %+v", admission)
	}
	handoff, err := CreateDeploymentRiskHandoff(root, DeploymentRiskHandoffOptions{AdmissionID: admission.ID})
	if err != nil {
		t.Fatal(err)
	}
	if handoff.Status != "review_required" || handoff.Decision != "DEPLOYMENT_RISK_HANDOFF_REVIEW_REQUIRED" || !handoff.ReviewRequired {
		t.Fatalf("expected review-required handoff, got %+v", handoff)
	}
	if handoff.SignalID == "" || handoff.BugCandidateID == "" || handoff.RepairPlanID == "" {
		t.Fatalf("expected repair artifacts, got %+v", handoff)
	}
	queue, err := ListDeploymentRiskReviewQueue(root, "pending", 10)
	if err != nil || len(queue) != 1 || queue[0].HandoffID != handoff.ID {
		t.Fatalf("expected pending risk review queue item, queue=%+v err=%v", queue, err)
	}
	deferredReview, deferredHandoff, found, err := ReviewDeploymentRiskHandoff(root, handoff.ID, DeploymentRiskReviewOptions{
		Decision:   "deferred",
		ReviewerID: "qa-owner",
		Reason:     "need more monitor evidence",
	})
	if err != nil || !found {
		t.Fatalf("expected deferred review, found=%v review=%+v err=%v", found, deferredReview, err)
	}
	if deferredHandoff.Status != "review_deferred" || deferredHandoff.Decision != "DEPLOYMENT_RISK_REVIEW_DEFERRED" || !deferredHandoff.ReviewRequired {
		t.Fatalf("expected deferred handoff to stay reviewable, got %+v", deferredHandoff)
	}
	queue, err = ListDeploymentRiskReviewQueue(root, "pending", 10)
	if err != nil || len(queue) != 1 || queue[0].ReviewDecision != "deferred" {
		t.Fatalf("expected deferred handoff to remain in pending queue, queue=%+v err=%v", queue, err)
	}
	approvedReview, approvedHandoff, found, err := ReviewDeploymentRiskHandoff(root, handoff.ID, DeploymentRiskReviewOptions{
		Decision:   "approved",
		ReviewerID: "release-owner",
		Reason:     "risk accepted for controlled repair planning",
		NextStep:   "repair_plan",
	})
	if err != nil || !found {
		t.Fatalf("expected approved review, found=%v review=%+v err=%v", found, approvedReview, err)
	}
	if approvedHandoff.Status != "review_approved" || approvedHandoff.Decision != "DEPLOYMENT_RISK_REVIEW_APPROVED" || approvedHandoff.ReviewRequired {
		t.Fatalf("expected approved handoff, got %+v", approvedHandoff)
	}
	if approvedReview.EvidenceRefs == nil || len(approvedReview.EvidenceRefs) == 0 || approvedHandoff.ReviewID != approvedReview.ID {
		t.Fatalf("expected review evidence and handoff link, review=%+v handoff=%+v", approvedReview, approvedHandoff)
	}
	if _, _, _, err := ReviewDeploymentRiskHandoff(root, handoff.ID, DeploymentRiskReviewOptions{Decision: "approved"}); err == nil {
		t.Fatalf("expected approved handoff to reject duplicate review")
	}
	queue, err = ListDeploymentRiskReviewQueue(root, "pending", 10)
	if err != nil || len(queue) != 0 {
		t.Fatalf("expected no pending queue items after approval, queue=%+v err=%v", queue, err)
	}
	reviewed, err := ListDeploymentRiskReviewQueue(root, "reviewed", 10)
	if err != nil || len(reviewed) != 1 || reviewed[0].ReviewID != approvedReview.ID {
		t.Fatalf("expected reviewed queue item, reviewed=%+v err=%v", reviewed, err)
	}
	reviews, err := ListDeploymentRiskReviews(root, 10)
	if err != nil || len(reviews) != 2 {
		t.Fatalf("expected two risk review records, reviews=%+v err=%v", reviews, err)
	}
	loadedReview, found, err := LoadDeploymentRiskReview(root, approvedReview.ID)
	if err != nil || !found || loadedReview.ID != approvedReview.ID {
		t.Fatalf("expected approved review to load, found=%v review=%+v err=%v", found, loadedReview, err)
	}
	loaded, found, err := LoadDeploymentRiskHandoff(root, handoff.ID)
	if err != nil || !found || loaded.ID != handoff.ID {
		t.Fatalf("expected handoff to load, found=%v loaded=%+v err=%v", found, loaded, err)
	}
	list, err := ListDeploymentRiskHandoffs(root, 10)
	if err != nil || len(list) != 1 || list[0].ID != handoff.ID {
		t.Fatalf("expected handoff list, list=%+v err=%v", list, err)
	}
	plan, err := LoadPlan(root, handoff.RepairPlanID)
	if err != nil || !plan.RequiresApproval || plan.Status != "requires_approval" {
		t.Fatalf("expected approval-gated repair plan, plan=%+v err=%v", plan, err)
	}
}
