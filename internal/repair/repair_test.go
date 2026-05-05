package repair

import (
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
}
