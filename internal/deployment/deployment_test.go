package deployment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/serverresources"
	"moyuan-code/internal/workspace"
)

func TestExecuteRunsSmokeMonitorAndSuggestsRollback(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer okServer.Close()
	okPlan := createDeploymentPlanWithHealthTarget(t, root, "smoke-ok", okServer.URL)
	okExecution, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: okPlan.ID,
		Mode:         "local_shell",
		Approved:     true,
		Commands:     []string{"printf deploy-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if okExecution.Decision != "DEPLOY_EXECUTION_COMPLETED" || okExecution.SmokeReport.Status != "passed" || okExecution.MonitorReport.Status != "passed" || okExecution.RollbackSuggestion.Required {
		t.Fatalf("expected successful smoke and monitor, got %+v", okExecution)
	}
	evidenceRecords, err := evidence.List(root, evidence.ListOptions{ParentType: "deployment_execution", ParentID: okExecution.ID, Limit: 10})
	if err != nil || len(evidenceRecords) != 1 || evidenceRecords[0].Decision != okExecution.Decision {
		t.Fatalf("expected deployment execution evidence, records=%+v err=%v", evidenceRecords, err)
	}

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer failServer.Close()
	failPlan := createDeploymentPlanWithHealthTarget(t, root, "smoke-fail", failServer.URL)
	failedExecution, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: failPlan.ID,
		Mode:         "local_shell",
		Approved:     true,
		Commands:     []string{"printf deploy-fail"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if failedExecution.Decision != "DEPLOY_SMOKE_FAILED" || failedExecution.SmokeReport.Status != "failed" || !failedExecution.RollbackSuggestion.Required {
		t.Fatalf("expected smoke failure rollback suggestion, got %+v", failedExecution)
	}
	if !hasStep(failedExecution.Steps, "rollback", "suggested") {
		t.Fatalf("expected rollback suggestion step: %+v", failedExecution.Steps)
	}
	releaseLog, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(root).LogsDir, "release.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found ||
		!strings.Contains(releaseLog, "deployment.smoke.completed") ||
		!strings.Contains(releaseLog, "deployment.monitor.completed") ||
		!strings.Contains(releaseLog, "deployment.rollback.suggested") {
		t.Fatalf("expected smoke, monitor, and rollback logs, found=%v log=%s", found, releaseLog)
	}
}

func TestExecuteBuildsSSHPreviewWithoutRemoteExecution(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	plan := createDeploymentPlanWithHealthTarget(t, root, "ssh-preview", "http://127.0.0.1/healthz")
	execution, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "ssh_preview",
		Commands:     []string{"deploy api"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != "completed" || execution.Decision != "DEPLOY_SSH_PREVIEW_READY" {
		t.Fatalf("expected ssh preview ready, got %+v", execution)
	}
	if execution.RemotePlan == nil || execution.RemotePlan.Decision != "SSH_PREVIEW_READY" || len(execution.RemotePlan.Targets) != 1 {
		t.Fatalf("expected remote plan target, got %+v", execution.RemotePlan)
	}
	target := execution.RemotePlan.Targets[0]
	if target.Status != "planned" || target.AuthRef != "env:DEV_SERVER_SSH_KEY" || target.Commands[0] != "deploy api" {
		t.Fatalf("expected planned target with auth ref and preview command, got %+v", target)
	}
	if !hasStep(execution.Steps, "ssh_preview", "planned") {
		t.Fatalf("expected ssh preview step, got %+v", execution.Steps)
	}
	releaseLog, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(root).LogsDir, "release.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found || !strings.Contains(releaseLog, "deployment.ssh.previewed") {
		t.Fatalf("expected ssh preview log, found=%v log=%s", found, releaseLog)
	}
}

func TestExecuteBlocksRealSSHExecution(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	plan := createDeploymentPlanWithHealthTarget(t, root, "ssh-execute", "http://127.0.0.1/healthz")
	execution, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "ssh_execute",
		Approved:     true,
		Commands:     []string{"deploy api"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != "blocked" || execution.Decision != "DEPLOY_EXECUTION_BLOCKED" {
		t.Fatalf("expected blocked ssh execution, got %+v", execution)
	}
	if !containsReason(execution.Reasons, "ssh_real_execution_not_enabled") {
		t.Fatalf("expected ssh execution disabled reason, got %+v", execution.Reasons)
	}
	if execution.RemotePlan == nil || execution.RemotePlan.Decision != "SSH_EXECUTION_NOT_ENABLED" {
		t.Fatalf("expected blocked remote execution plan, got %+v", execution.RemotePlan)
	}
	if !hasStep(execution.Steps, "ssh_execute", "blocked") {
		t.Fatalf("expected blocked ssh execute step, got %+v", execution.Steps)
	}
}

func TestExecuteSSHGuardedRunnerValidatesAllowlist(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MOYUAN_ALLOW_SSH_EXECUTE", "1")
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	plan := createDeploymentPlanWithHealthTarget(t, root, "ssh-guarded-safe", "http://127.0.0.1/healthz")
	execution, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "ssh_execute",
		Approved:     true,
		Commands:     []string{"printf deploy-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != "blocked" || execution.Decision != "DEPLOY_SSH_EXECUTION_GUARDED_READY" || !execution.RemoteExecEnabled {
		t.Fatalf("expected guarded SSH execution boundary, got %+v", execution)
	}
	if execution.RemotePlan == nil || execution.RemotePlan.Decision != "SSH_EXECUTION_GUARDED_READY" {
		t.Fatalf("expected guarded remote plan, got %+v", execution.RemotePlan)
	}
	if !hasStep(execution.Steps, "ssh_execute", "planned") {
		t.Fatalf("expected planned SSH execute step, got %+v", execution.Steps)
	}
	if !containsReason(execution.Reasons, "remote_ssh_command_runner_not_enabled") {
		t.Fatalf("expected runner boundary reason, got %+v", execution.Reasons)
	}
	releaseLog, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(root).LogsDir, "release.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found || !strings.Contains(releaseLog, "deployment.ssh.execution.guarded") {
		t.Fatalf("expected guarded ssh execution log, found=%v log=%s", found, releaseLog)
	}
}

func TestExecuteSSHGuardedRunnerBlocksUnsafeCommand(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MOYUAN_ALLOW_SSH_EXECUTE", "1")
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	plan := createDeploymentPlanWithHealthTarget(t, root, "ssh-guarded-unsafe", "http://127.0.0.1/healthz")
	execution, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "ssh_execute",
		Approved:     true,
		Commands:     []string{"rm -rf /"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != "blocked" || execution.Decision != "DEPLOY_SSH_EXECUTION_BLOCKED" {
		t.Fatalf("expected unsafe SSH command to be blocked, got %+v", execution)
	}
	if execution.RemotePlan == nil || execution.RemotePlan.Decision != "SSH_EXECUTION_BLOCKED" {
		t.Fatalf("expected blocked remote plan, got %+v", execution.RemotePlan)
	}
	if !containsReason(execution.Reasons, "command_not_allowed") {
		t.Fatalf("expected command allowlist reason, got %+v", execution.Reasons)
	}
	if !hasStep(execution.Steps, "ssh_execute", "blocked") {
		t.Fatalf("expected blocked SSH execute step, got %+v", execution.Steps)
	}
}

func createDeploymentPlanWithHealthTarget(t *testing.T, root string, id string, target string) Plan {
	t.Helper()
	resource, err := serverresources.Add(root, serverresources.Resource{
		ID:          id,
		Environment: "test_dev",
		Host:        "127.0.0.1",
		Provider:    "local_vm",
		Owner:       "devops",
		AuthRef:     "env:DEV_SERVER_SSH_KEY",
		Healthcheck: serverresources.Healthcheck{Type: "http", Target: target},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := finish(root, Plan{
		ID:          "deployment-" + id,
		ReleaseID:   "release-" + id,
		Environment: "test_dev",
		Status:      "planned",
		Decision:    "DEPLOY_PLAN_READY",
		Reasons:     []string{"test_fixture"},
		Resources: []ResourceSummary{{
			ID:          resource.ID,
			Environment: resource.Environment,
			Host:        resource.Host,
			Status:      resource.Status,
		}},
		SmokePlan:    StepPlan{Status: "planned", Required: true, Actions: []string{"http smoke"}},
		MonitorPlan:  StepPlan{Status: "planned", Required: true, Window: "1m", Actions: []string{"http monitor"}},
		RollbackPlan: StepPlan{Status: "planned", Required: true, Actions: []string{"rollback release-" + id}},
		CreatedAt:    "2026-05-05T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func hasStep(steps []ExecutionStep, name string, status string) bool {
	for _, step := range steps {
		if step.Name == name && step.Status == status {
			return true
		}
	}
	return false
}

func containsReason(reasons []string, expected string) bool {
	for _, reason := range reasons {
		if reason == expected {
			return true
		}
	}
	return false
}
