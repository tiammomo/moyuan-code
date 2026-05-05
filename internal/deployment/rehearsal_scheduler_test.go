package deployment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/release"
	"moyuan-code/internal/workspace"
)

func TestRunRehearsalSchedulerNoTargets(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	run, err := RunRehearsalScheduler(context.Background(), root, RehearsalSchedulerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "blocked" || run.Decision != "REHEARSAL_SCHEDULER_NO_TARGETS" || !containsReason(run.Reasons, "scheduler_targets_missing") {
		t.Fatalf("expected no-target scheduler block, got %+v", run)
	}
	if run.ID == "" || len(run.EvidenceIDs) != 1 {
		t.Fatalf("expected persisted scheduler run with evidence, got %+v", run)
	}
}

func TestRunRehearsalSchedulerCreatesAndSkipsExistingAdmission(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	plan := createDeploymentPlanWithHealthTarget(t, root, "scheduler-ok", server.URL)
	execution, err := Execute(context.Background(), root, ExecuteOptions{DeploymentID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	run, err := RunRehearsalScheduler(context.Background(), root, RehearsalSchedulerOptions{ExecutionID: execution.ID, RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "completed" || run.Decision != "REHEARSAL_SCHEDULER_COMPLETED" || run.CreatedCount != 1 || len(run.Targets) != 1 {
		t.Fatalf("expected scheduler to create rehearsal/admission, got %+v", run)
	}
	target := run.Targets[0]
	if target.Status != "created" || target.Decision != "REHEARSAL_SCHEDULER_ADMISSION_ALLOWED" || target.RehearsalID == "" || target.AdmissionID == "" {
		t.Fatalf("expected allowed admission target, got %+v", target)
	}
	replayed, err := RunRehearsalScheduler(context.Background(), root, RehearsalSchedulerOptions{ExecutionID: execution.ID, RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Decision != "REHEARSAL_SCHEDULER_NOOP" || replayed.SkippedCount != 1 || replayed.Targets[0].Reason != "admission_already_exists" {
		t.Fatalf("expected replay to skip existing admission, got %+v", replayed)
	}
	loaded, found, err := LoadRehearsalSchedulerRun(root, run.ID)
	if err != nil || !found || loaded.ID != run.ID {
		t.Fatalf("expected scheduler run to load, found=%v run=%+v err=%v", found, loaded, err)
	}
	runs, err := ListRehearsalSchedulerRuns(root, 5)
	if err != nil || len(runs) != 2 {
		t.Fatalf("expected scheduler runs list, runs=%+v err=%v", runs, err)
	}
}

func TestRunRehearsalSchedulerSupportsDeploymentAndCandidateTargets(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	deploymentPlan := createDeploymentPlanWithHealthTarget(t, root, "scheduler-deployment", server.URL)
	if _, err := Execute(context.Background(), root, ExecuteOptions{DeploymentID: deploymentPlan.ID}); err != nil {
		t.Fatal(err)
	}
	deploymentRun, err := RunRehearsalScheduler(context.Background(), root, RehearsalSchedulerOptions{DeploymentID: deploymentPlan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if deploymentRun.Status != "completed" || deploymentRun.Targets[0].Type != "deployment" || deploymentRun.Targets[0].AdmissionID == "" {
		t.Fatalf("expected deployment target scheduler run, got %+v", deploymentRun)
	}

	candidate := release.Candidate{
		ID:             "release-candidate-scheduler",
		ReleaseBatchID: "release-batch-scheduler",
		Status:         "ready",
		Decision:       "RELEASE_CANDIDATE_READY",
		Version:        "v0.3.0",
		ReleaseBranch:  "release/v0.3.0",
		CreatedAt:      "2026-05-05T00:00:00Z",
	}
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(root).ReleasesDir, "candidates", candidate.ID+".json"), candidate); err != nil {
		t.Fatal(err)
	}
	candidatePlan, err := CreatePlanFromCandidate(root, CandidatePlanOptions{CandidateID: candidate.ID, Environment: "test_dev"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(context.Background(), root, ExecuteOptions{DeploymentID: candidatePlan.ID}); err != nil {
		t.Fatal(err)
	}
	candidateRun, err := RunRehearsalScheduler(context.Background(), root, RehearsalSchedulerOptions{CandidateID: candidate.ID, Environment: "test_dev"})
	if err != nil {
		t.Fatal(err)
	}
	if candidateRun.Status != "completed" || candidateRun.Targets[0].Type != "candidate" || candidateRun.Targets[0].AdmissionID == "" {
		t.Fatalf("expected candidate target scheduler run, got %+v", candidateRun)
	}
}

func TestRunRehearsalSchedulerReportsBlockedAdmission(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	failedExecution := createRollbackRequiredExecution(t, root, "scheduler-risk")
	run, err := RunRehearsalScheduler(context.Background(), root, RehearsalSchedulerOptions{ExecutionID: failedExecution.ID})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "attention_required" || run.Decision != "REHEARSAL_SCHEDULER_ATTENTION_REQUIRED" || run.BlockedCount != 1 {
		t.Fatalf("expected scheduler to surface blocked admission, got %+v", run)
	}
	if run.Targets[0].Decision != "REHEARSAL_SCHEDULER_ADMISSION_BLOCKED" || run.Targets[0].AdmissionID == "" {
		t.Fatalf("expected blocked admission target, got %+v", run.Targets[0])
	}
}
