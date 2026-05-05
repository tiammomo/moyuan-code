package repair

import (
	"context"
	"testing"

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
