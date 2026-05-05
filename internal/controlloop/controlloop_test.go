package controlloop

import (
	"context"
	"testing"
	"time"

	"moyuan-code/internal/evidence"
	"moyuan-code/internal/operations"
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

func TestQueueRunsDueItemsAndWaitsForMaintenanceWindow(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	due, err := Enqueue(root, QueueOptions{
		RequestedBy:       "ops",
		Steps:             []string{StepResourceLifecycleScan},
		MaintenanceWindow: "always",
		RetryBudget:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	future, err := Enqueue(root, QueueOptions{
		RequestedBy:       "ops",
		Steps:             []string{StepResourceLifecycleScan},
		MaintenanceWindow: "after:" + time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err := RunQueue(context.Background(), root, QueueRunOptions{MaxItems: 5})
	if err != nil {
		t.Fatal(err)
	}
	if report.Processed != 2 || report.Executed != 1 || report.Waiting != 1 {
		t.Fatalf("unexpected queue run report: %+v", report)
	}
	completed := queueItemByID(report.QueueItems, due.ID)
	if completed.ID == "" || completed.Status != "completed" || completed.RunID == "" {
		t.Fatalf("expected due item to execute, got %+v", completed)
	}
	waiting := queueItemByID(report.QueueItems, future.ID)
	if waiting.ID == "" || waiting.Status != "waiting" || waiting.Decision != "CONTROL_QUEUE_WAITING_MAINTENANCE_WINDOW" {
		t.Fatalf("expected future item to wait, got %+v", waiting)
	}
	list, err := ListQueue(root, QueueListOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected queue items to persist, got %+v", list)
	}
}

func TestQueueInvalidMaintenanceWindowRequiresManualHandoff(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	item, err := Enqueue(root, QueueOptions{Steps: []string{StepResourceLifecycleScan}, MaintenanceWindow: "between:bad"})
	if err != nil {
		t.Fatal(err)
	}
	report, err := RunQueue(context.Background(), root, QueueRunOptions{MaxItems: 5})
	if err != nil {
		t.Fatal(err)
	}
	updated := queueItemByID(report.QueueItems, item.ID)
	if updated.Status != "manual_required" || updated.Decision != "CONTROL_QUEUE_MANUAL_HANDOFF" {
		t.Fatalf("expected manual handoff for invalid window, got %+v", updated)
	}
}

func TestQueueRequiresBoundReviewPacketBeforeExecution(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	item, err := Enqueue(root, QueueOptions{
		Steps:             []string{StepResourceLifecycleScan},
		MaintenanceWindow: "always",
		ReviewPacketID:    "missing-review-packet",
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err := RunQueue(context.Background(), root, QueueRunOptions{MaxItems: 5})
	if err != nil {
		t.Fatal(err)
	}
	updated := queueItemByID(report.QueueItems, item.ID)
	if updated.Status != "manual_required" || updated.Decision != "CONTROL_QUEUE_REVIEW_GATE_MANUAL_REQUIRED" {
		t.Fatalf("expected review gate manual handoff, got %+v", updated)
	}
	if len(updated.Reasons) == 0 || updated.Reasons[len(updated.Reasons)-1] != "write_review_packet_missing:missing-review-packet" {
		t.Fatalf("expected missing review packet reason, got %+v", updated.Reasons)
	}
}

func TestQueueAdapterRecoveryRequiresReviewBeforeExecution(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	adapterReport, err := operations.CreateWriteAdapterExecutions(root, operations.WriteAdapterExecutionOptions{ExecutionPlanID: "missing-execution-plan", Mode: "preview", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	recoveryReport, err := operations.ListWriteAdapterRecoveries(root, operations.WriteAdapterRecoveryOptions{ExecutionID: adapterReport.Executions[0].ID, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(recoveryReport.Recoveries) != 1 {
		t.Fatalf("expected adapter recovery, got %+v", recoveryReport)
	}
	item, err := Enqueue(root, QueueOptions{
		Steps:             []string{StepDecisionLedgerRefresh},
		MaintenanceWindow: "always",
		AdapterRecoveryID: recoveryReport.Recoveries[0].ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err := RunQueue(context.Background(), root, QueueRunOptions{MaxItems: 5})
	if err != nil {
		t.Fatal(err)
	}
	updated := queueItemByID(report.QueueItems, item.ID)
	if updated.Status != "manual_required" || updated.Decision != "CONTROL_QUEUE_RECOVERY_REVIEW_REQUIRED" || updated.RunID != "" {
		t.Fatalf("expected adapter recovery queue item to require review before execution, got %+v", updated)
	}
	if !queueReasonsContain(updated.Reasons, "write_adapter_recovery_open:"+recoveryReport.Recoveries[0].ID) {
		t.Fatalf("expected recovery binding reason, got %+v", updated.Reasons)
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

func queueItemByID(items []QueueItem, id string) QueueItem {
	for _, item := range items {
		if item.ID == id {
			return item
		}
	}
	return QueueItem{}
}

func queueReasonsContain(reasons []string, expected string) bool {
	for _, reason := range reasons {
		if reason == expected {
			return true
		}
	}
	return false
}
