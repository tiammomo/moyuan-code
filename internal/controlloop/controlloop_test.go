package controlloop

import (
	"context"
	"testing"

	"moyuan-code/internal/evidence"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/serverresources"
	"moyuan-code/internal/workspace"
)

func TestRunExecutesBoundedSafeControlLoop(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := serverresources.Add(root, serverresources.Resource{
		ID:                "dev-expired",
		Environment:       "test_dev",
		Host:              "10.0.0.10",
		Provider:          "local_vm",
		Owner:             "ops",
		AuthRef:           "env:DEV_SERVER_SSH_KEY",
		ExpiresAt:         "2000-01-01",
		MaintenanceWindow: "due:2000-01-01",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := providers.Upsert(root, providers.Provider{
		ID:      "glm-api",
		Name:    "GLM API",
		Vendor:  "zhipu",
		APIType: "openai-compatible",
		AuthRef: "env:GLM_API_KEY",
		Enabled: true,
		DataPolicy: providers.DataPolicy{
			AllowProjectMemory: true,
		},
		Models: []providers.Model{{ID: "glm-4"}},
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLM_API_KEY", "")

	run, err := Run(context.Background(), root, RunOptions{RequestedBy: "ops"})
	if err != nil {
		t.Fatal(err)
	}
	if run.ID == "" || run.Status != "completed" || run.Decision != "CONTROL_LOOP_COMPLETED_WITH_ATTENTION" {
		t.Fatalf("unexpected run: %+v", run)
	}
	if len(run.Steps) != 3 {
		t.Fatalf("expected 3 default steps, got %d: %+v", len(run.Steps), run.Steps)
	}
	assertStep(t, run, StepResourceLifecycleScan)
	assertStep(t, run, StepProviderOpsRefresh)
	assertStep(t, run, StepProjectComprehensionRefresh)
	for _, step := range run.Steps {
		if step.FinishedAt == "" || len(step.EvidenceIDs) == 0 {
			t.Fatalf("step missing finish/evidence: %+v", step)
		}
	}
	loaded, found, err := Load(root, run.ID)
	if err != nil || !found {
		t.Fatalf("expected saved run, found=%v err=%v", found, err)
	}
	if loaded.ID != run.ID {
		t.Fatalf("loaded wrong run: %+v", loaded)
	}
	list, err := List(root, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != run.ID {
		t.Fatalf("unexpected run list: %+v", list)
	}
	records, err := evidence.List(root, evidence.ListOptions{ParentType: "control_loop", ParentID: run.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 evidence records, got %d", len(records))
	}
}

func TestRunUsesIdempotencyAndDurableSteps(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := serverresources.Add(root, serverresources.Resource{
		ID:          "dev-health",
		Environment: "test_dev",
		Host:        "10.0.0.11",
		Provider:    "local_vm",
		Owner:       "ops",
		AuthRef:     "env:DEV_SERVER_SSH_KEY",
		ExpiresAt:   "2099-01-01",
	}); err != nil {
		t.Fatal(err)
	}

	run, err := Run(context.Background(), root, RunOptions{
		IdempotencyKey: "phase19-health",
		RetryBudget:    1,
		Environment:    "test_dev",
		Steps:          []string{StepResourceHealthScan, StepOperationsAuditExport, StepDecisionLedgerRefresh},
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.ID == "" || run.IdempotencyKey != "phase19-health" || run.IdempotentReplay {
		t.Fatalf("unexpected durable run: %+v", run)
	}
	if len(run.Steps) != 3 {
		t.Fatalf("expected durable steps, got %+v", run.Steps)
	}
	replayed, err := Run(context.Background(), root, RunOptions{IdempotencyKey: "phase19-health", Steps: []string{StepResourceHealthScan}})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != run.ID || !replayed.IdempotentReplay {
		t.Fatalf("expected idempotent replay of %s, got %+v", run.ID, replayed)
	}
	list, err := List(root, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != run.ID {
		t.Fatalf("idempotent replay should not append a second run: %+v", list)
	}
}

func TestRunExhaustsRetryBudgetForFailedStep(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), root, RunOptions{
		RetryBudget:  0,
		RetryAttempt: 0,
		Steps:        []string{"unsupported_control_step"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "manual_required" || run.Decision != "CONTROL_RUNNER_RETRY_BUDGET_EXHAUSTED" {
		t.Fatalf("expected retry budget exhaustion, got %+v", run)
	}
	if len(run.Steps) != 1 || run.Steps[0].Status != "failed" {
		t.Fatalf("expected failed step evidence, got %+v", run.Steps)
	}
}

func TestRunRejectsUnboundedStepList(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	_, err := Run(context.Background(), root, RunOptions{
		MaxSteps: 1,
		Steps:    []string{StepResourceLifecycleScan, StepProviderOpsRefresh},
	})
	if err == nil || err.Error() != "control_loop_steps_exceed_max" {
		t.Fatalf("expected max step error, got %v", err)
	}
}

func assertStep(t *testing.T, run RunRecord, stepType string) {
	t.Helper()
	for _, step := range run.Steps {
		if step.Type == stepType {
			return
		}
	}
	t.Fatalf("missing step %s in %+v", stepType, run.Steps)
}
