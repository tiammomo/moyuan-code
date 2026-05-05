package deployment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/release"
	"moyuan-code/internal/serverresources"
	"moyuan-code/internal/workspace"
)

func TestCreatePlanUsesDefaultDeploymentCheckTemplates(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	releasePlan := release.Plan{ID: "release-template", Status: "ready", Decision: "RELEASE_SUGGESTED", Version: "v1.2.3"}
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(root).ReleasesDir, releasePlan.ID+".json"), releasePlan); err != nil {
		t.Fatal(err)
	}
	if _, err := serverresources.Add(root, serverresources.Resource{
		ID:          "template-host",
		Environment: "test_dev",
		Host:        "127.0.0.1",
		Provider:    "local_vm",
		Owner:       "devops",
		AuthRef:     "env:DEV_SERVER_SSH_KEY",
		Healthcheck: serverresources.Healthcheck{Type: "http", Target: "http://127.0.0.1/healthz"},
	}); err != nil {
		t.Fatal(err)
	}

	plan, err := CreatePlan(root, PlanOptions{ReleaseID: releasePlan.ID, Environment: "test_dev"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Decision != "DEPLOY_PLAN_READY" {
		t.Fatalf("expected deployment plan ready, got %+v", plan)
	}
	if plan.SmokePlan.TemplateID != "deploy-smoke-test_dev-v1" || plan.SmokePlan.Severity != "high" || !containsString(plan.SmokePlan.FailureClasses, "smoke_failed") {
		t.Fatalf("expected smoke template policy, got %+v", plan.SmokePlan)
	}
	if plan.MonitorPlan.TemplateID != "deploy-monitor-test_dev-v1" || plan.MonitorPlan.Severity != "medium" || plan.MonitorPlan.Window != "30m" || !containsString(plan.MonitorPlan.FailureClasses, "monitor_failed") {
		t.Fatalf("expected monitor template policy, got %+v", plan.MonitorPlan)
	}
}

func TestCreatePlanFromCandidateUsesReleaseCandidateAndResources(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	candidate := release.Candidate{
		ID:             "release-candidate-deploy",
		ReleaseBatchID: "release-batch-deploy",
		Status:         "ready",
		Decision:       "RELEASE_CANDIDATE_READY",
		Version:        "v0.2.0",
		ReleaseBranch:  "release/v0.2.0",
		SourceBranch:   "moyuan/integration/release",
		CreatedAt:      "2026-05-05T00:00:00Z",
	}
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(root).ReleasesDir, "candidates", candidate.ID+".json"), candidate); err != nil {
		t.Fatal(err)
	}
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer okServer.Close()
	if _, err := serverresources.Add(root, serverresources.Resource{
		ID:          "candidate-host",
		Environment: "test_dev",
		Host:        "127.0.0.1",
		Provider:    "local_vm",
		Owner:       "devops",
		AuthRef:     "env:DEV_SERVER_SSH_KEY",
		Status:      "active",
		Healthcheck: serverresources.Healthcheck{Type: "http", Target: okServer.URL},
	}); err != nil {
		t.Fatal(err)
	}

	plan, err := CreatePlanFromCandidate(root, CandidatePlanOptions{CandidateID: candidate.ID, Environment: "test_dev"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Decision != "DEPLOY_PLAN_READY" || plan.ReleaseID != candidate.ID || len(plan.Resources) != 1 {
		t.Fatalf("expected deployment handoff from candidate, got %+v", plan)
	}
	if !containsString(plan.Reasons, "release_candidate_and_resources_ready") || plan.SmokePlan.TemplateID == "" || plan.MonitorPlan.TemplateID == "" {
		t.Fatalf("expected candidate deployment check plans, got %+v", plan)
	}
	execution, err := ExecuteFromCandidate(context.Background(), root, CandidateExecuteOptions{CandidateID: candidate.ID, Environment: "test_dev"})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Decision != "DEPLOY_EXECUTION_DRY_RUN" || execution.DeploymentID != plan.ID || execution.ReleaseID != candidate.ID {
		t.Fatalf("expected dry-run execution from candidate deployment plan, got %+v", execution)
	}
	completed, err := ExecuteFromCandidate(context.Background(), root, CandidateExecuteOptions{
		CandidateID:  candidate.ID,
		Environment:  "test_dev",
		Mode:         "local_shell",
		Approved:     true,
		ApprovalID:   approveDeploymentExecution(t, root, plan.ID, "local_shell"),
		Commands:     []string{"printf candidate-deploy-ok"},
		DeploymentID: plan.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	feedback, found, err := FeedbackForCandidate(root, candidate.ID, 5)
	if err != nil || !found {
		t.Fatalf("expected candidate deployment feedback, found=%v err=%v", found, err)
	}
	if feedback.Decision != "CANDIDATE_DEPLOYMENT_HEALTHY" || feedback.LatestExecutionID != completed.ID || feedback.RollbackRequired {
		t.Fatalf("expected healthy candidate deployment feedback, got %+v", feedback)
	}
	if feedback.HistoryCount < 2 || len(feedback.EvidenceIDs) == 0 {
		t.Fatalf("expected candidate feedback to aggregate histories and evidence, got %+v", feedback)
	}
}

func TestCreatePlanFromCandidateBlocksWhenCandidateNotReady(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	candidate := release.Candidate{
		ID:        "release-candidate-blocked-deploy",
		Status:    "blocked",
		Decision:  "RELEASE_CANDIDATE_BLOCKED",
		Version:   "v0.2.0",
		CreatedAt: "2026-05-05T00:00:00Z",
	}
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(root).ReleasesDir, "candidates", candidate.ID+".json"), candidate); err != nil {
		t.Fatal(err)
	}
	plan, err := CreatePlanFromCandidate(root, CandidatePlanOptions{CandidateID: candidate.ID, Environment: "test_dev"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Decision != "DEPLOY_BLOCKED" || !containsString(plan.Reasons, "release_candidate_not_ready:RELEASE_CANDIDATE_BLOCKED") {
		t.Fatalf("expected deployment handoff blocked by candidate readiness, got %+v", plan)
	}
	execution, err := ExecuteFromCandidate(context.Background(), root, CandidateExecuteOptions{CandidateID: candidate.ID})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Decision != "DEPLOY_EXECUTION_BLOCKED" || !containsString(execution.Reasons, "release_candidate_not_ready:RELEASE_CANDIDATE_BLOCKED") {
		t.Fatalf("expected deployment execution blocked by candidate readiness, got %+v", execution)
	}
}

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
		ApprovalID:   approveDeploymentExecution(t, root, okPlan.ID, "local_shell"),
		Commands:     []string{"printf deploy-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if okExecution.Decision != "DEPLOY_EXECUTION_COMPLETED" || okExecution.SmokeReport.Status != "passed" || okExecution.MonitorReport.Status != "passed" || okExecution.RollbackSuggestion.Required {
		t.Fatalf("expected successful smoke and monitor, got %+v", okExecution)
	}
	if okExecution.SmokeReport.TemplateID != "deploy-smoke-test_dev-v1" || okExecution.SmokeReport.Severity != "high" || okExecution.SmokeReport.FailureClass != "none" {
		t.Fatalf("expected smoke report template policy, got %+v", okExecution.SmokeReport)
	}
	if okExecution.MonitorReport.TemplateID != "deploy-monitor-test_dev-v1" || okExecution.MonitorReport.Severity != "medium" || okExecution.MonitorReport.FailureClass != "none" {
		t.Fatalf("expected monitor report template policy, got %+v", okExecution.MonitorReport)
	}
	evidenceRecords, err := evidence.List(root, evidence.ListOptions{ParentType: "deployment_execution", ParentID: okExecution.ID, Limit: 10})
	if err != nil || len(evidenceRecords) != 4 {
		t.Fatalf("expected deployment execution evidence, records=%+v err=%v", evidenceRecords, err)
	}
	if !hasEvidenceOperation(evidenceRecords, "deployment.execute.local_shell", okExecution.Decision) ||
		!hasEvidenceOperation(evidenceRecords, "deployment.smoke.check", "SMOKE_PASSED") ||
		!hasEvidenceOperation(evidenceRecords, "deployment.monitor.check", "MONITOR_PASSED") ||
		!hasEvidenceOperation(evidenceRecords, "deployment.rollback.not_required", "ROLLBACK_NOT_REQUIRED") {
		t.Fatalf("expected execution, smoke, monitor, and rollback evidence, got %+v", evidenceRecords)
	}
	okHistory, found, err := LoadPostDeploymentHistory(root, okExecution.ID)
	if err != nil || !found {
		t.Fatalf("expected post deployment history, found=%v err=%v", found, err)
	}
	if okHistory.Status != "passed" || okHistory.FailureClass != "none" || okHistory.Rollback.Status != "not_required" || len(okHistory.Checks) != 2 || len(okHistory.EvidenceIDs) != 4 {
		t.Fatalf("expected passed post deployment history, got %+v", okHistory)
	}
	if okHistory.Checks[0].TemplateID != "deploy-smoke-test_dev-v1" || okHistory.Checks[0].Severity != "high" || okHistory.Checks[0].FailureClass != "none" {
		t.Fatalf("expected smoke history template policy, got %+v", okHistory.Checks[0])
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
		ApprovalID:   approveDeploymentExecution(t, root, failPlan.ID, "local_shell"),
		Commands:     []string{"printf deploy-fail"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if failedExecution.Decision != "DEPLOY_SMOKE_FAILED" || failedExecution.SmokeReport.Status != "failed" || !failedExecution.RollbackSuggestion.Required {
		t.Fatalf("expected smoke failure rollback suggestion, got %+v", failedExecution)
	}
	if failedExecution.SmokeReport.TemplateID != "deploy-smoke-test_dev-v1" || failedExecution.SmokeReport.Severity != "high" || failedExecution.SmokeReport.FailureClass != "smoke_failed" {
		t.Fatalf("expected failed smoke report template policy, got %+v", failedExecution.SmokeReport)
	}
	if failedExecution.RollbackSuggestion.Runbook == nil || failedExecution.RollbackSuggestion.Runbook.Decision != "ROLLBACK_RUNBOOK_READY" || len(failedExecution.RollbackSuggestion.Runbook.Steps) < 3 {
		t.Fatalf("expected structured rollback runbook, got %+v", failedExecution.RollbackSuggestion.Runbook)
	}
	if !hasStep(failedExecution.Steps, "rollback", "suggested") {
		t.Fatalf("expected rollback suggestion step: %+v", failedExecution.Steps)
	}
	failedEvidenceRecords, err := evidence.List(root, evidence.ListOptions{ParentType: "deployment_execution", ParentID: failedExecution.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !hasEvidenceOperation(failedEvidenceRecords, "deployment.smoke.check", "SMOKE_FAILED") ||
		!hasEvidenceOperation(failedEvidenceRecords, "deployment.rollback.suggested", "ROLLBACK_RECOMMENDED") {
		t.Fatalf("expected smoke failure and rollback evidence, got %+v", failedEvidenceRecords)
	}
	if !hasEvidenceArtifact(failedEvidenceRecords, "deployment.rollback.suggested", "rollback_runbook") {
		t.Fatalf("expected rollback runbook artifact evidence, got %+v", failedEvidenceRecords)
	}
	failedHistory, found, err := LoadPostDeploymentHistory(root, failedExecution.ID)
	if err != nil || !found {
		t.Fatalf("expected failed post deployment history, found=%v err=%v", found, err)
	}
	if failedHistory.Status != "failed" || failedHistory.FailureClass != "smoke_failed" || failedHistory.Rollback.Status != "suggested" || failedHistory.Rollback.RunbookPath == "" || failedHistory.Rollback.StepCount < 3 {
		t.Fatalf("expected failed smoke history with rollback runbook, got %+v", failedHistory)
	}
	if failedHistory.Severity != "high" || failedHistory.Checks[0].TemplateID != "deploy-smoke-test_dev-v1" || failedHistory.Checks[0].FailureClass != "smoke_failed" {
		t.Fatalf("expected failed history severity and template policy, got %+v", failedHistory)
	}
	histories, err := ListPostDeploymentHistories(root, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(histories) != 2 {
		t.Fatalf("expected two post deployment histories, got %+v", histories)
	}
	assertDeploymentFileExists(t, filepath.Join(workspace.ForRoot(root).DeploymentsDir, "rollback-runbooks", failedExecution.ID+".json"))
	assertDeploymentFileExists(t, filepath.Join(workspace.ForRoot(root).DeploymentsDir, "post-deployment-history", failedExecution.ID+".json"))
	releaseLog, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(root).LogsDir, "release.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found ||
		!strings.Contains(releaseLog, "deployment.smoke.completed") ||
		!strings.Contains(releaseLog, "deployment.monitor.completed") ||
		!strings.Contains(releaseLog, "deployment.rollback.suggested") ||
		!strings.Contains(releaseLog, "deployment.post_deployment_history.recorded") {
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
		ApprovalID:   approveDeploymentExecution(t, root, plan.ID, "ssh_execute"),
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

func TestExecuteSSHRunnerExecutesAllowedCommands(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MOYUAN_ALLOW_SSH_EXECUTE", "1")
	t.Setenv("DEV_SERVER_SSH_KEY", "ssh-key-path-secret")
	prependFakeSSH(t, 0, "remote deploy ok ssh-key-path-secret", "")
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	plan := createDeploymentPlanWithHealthTarget(t, root, "ssh-runner-safe", server.URL)
	execution, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "ssh_execute",
		Approved:     true,
		ApprovalID:   approveDeploymentExecution(t, root, plan.ID, "ssh_execute"),
		Commands:     []string{"printf deploy-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != "completed" || execution.Decision != "DEPLOY_EXECUTION_COMPLETED" || !execution.RemoteExecEnabled {
		t.Fatalf("expected completed SSH execution, got %+v", execution)
	}
	if execution.RemotePlan == nil || execution.RemotePlan.Decision != "SSH_EXECUTION_READY" {
		t.Fatalf("expected SSH execution plan, got %+v", execution.RemotePlan)
	}
	if !hasStep(execution.Steps, "ssh_execute", "completed") {
		t.Fatalf("expected completed SSH execute step, got %+v", execution.Steps)
	}
	if !containsReason(execution.Reasons, "allowed_ssh_commands_completed") {
		t.Fatalf("expected completed SSH command reason, got %+v", execution.Reasons)
	}
	releaseLog, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(root).LogsDir, "release.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found || !strings.Contains(releaseLog, "deployment.ssh.commands.completed") {
		t.Fatalf("expected completed ssh execution log, found=%v log=%s", found, releaseLog)
	}
	assertDeploymentFileDoesNotContain(t, executionPath(root, execution.ID), "ssh-key-path-secret")
	assertDeploymentFileDoesNotContain(t, filepath.Join(workspace.ForRoot(root).DeploymentsDir, "executions.jsonl"), "ssh-key-path-secret")
	assertDeploymentFileDoesNotContain(t, filepath.Join(workspace.ForRoot(root).LogsDir, "audit.jsonl"), "ssh-key-path-secret")
}

func TestExecuteSSHRunnerFailsAndSuggestsRollback(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MOYUAN_ALLOW_SSH_EXECUTE", "1")
	t.Setenv("DEV_SERVER_SSH_KEY", "ssh-key-path-secret")
	prependFakeSSH(t, 7, "", "remote failed ssh-key-path-secret")
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	plan := createDeploymentPlanWithHealthTarget(t, root, "ssh-runner-fail", "http://127.0.0.1/healthz")
	execution, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "ssh_execute",
		Approved:     true,
		ApprovalID:   approveDeploymentExecution(t, root, plan.ID, "ssh_execute"),
		Commands:     []string{"printf deploy-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != "failed" || execution.Decision != "DEPLOY_SSH_EXECUTION_FAILED" {
		t.Fatalf("expected failed SSH execution, got %+v", execution)
	}
	if !execution.RollbackSuggestion.Required || execution.RollbackSuggestion.Reason != "ssh_command_failed" {
		t.Fatalf("expected rollback suggestion for SSH command failure, got %+v", execution.RollbackSuggestion)
	}
	if execution.RollbackSuggestion.Runbook == nil || execution.RollbackSuggestion.Runbook.Decision != "ROLLBACK_RUNBOOK_READY" {
		t.Fatalf("expected rollback runbook for SSH command failure, got %+v", execution.RollbackSuggestion.Runbook)
	}
	if !hasStep(execution.Steps, "ssh_execute", "failed") {
		t.Fatalf("expected failed SSH execute step, got %+v", execution.Steps)
	}
	assertDeploymentFileDoesNotContain(t, executionPath(root, execution.ID), "ssh-key-path-secret")
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
		ApprovalID:   approveDeploymentExecution(t, root, plan.ID, "ssh_execute"),
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

func TestExecuteRequiresAndConsumesApprovalProof(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	plan := createDeploymentPlanWithHealthTarget(t, root, "approval-proof", server.URL)

	blocked, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "local_shell",
		Commands:     []string{"printf gated"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Status != "blocked" || blocked.ApprovalID == "" || !containsReason(blocked.Reasons, "execution_approval_required") {
		t.Fatalf("expected approval request before real execution, got %+v", blocked)
	}
	approval, found, err := approvals.Load(root, blocked.ApprovalID)
	if err != nil || !found {
		t.Fatalf("expected approval record, found=%v err=%v", found, err)
	}
	if approval.TargetType != "deployment_execution" || approval.TargetID != plan.ID || approval.Action != "deploy.execute.local.shell" {
		t.Fatalf("expected stable deployment execution approval scope, got %+v", approval)
	}

	missingProof, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "local_shell",
		Approved:     true,
		Commands:     []string{"printf gated"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if missingProof.Status != "blocked" || !containsReason(missingProof.Reasons, "approval_id_required_before_deployment_execution") {
		t.Fatalf("expected approval id proof requirement, got %+v", missingProof)
	}
	if _, _, err := approvals.Decide(root, blocked.ApprovalID, approvals.DecisionOptions{Decision: "approved", DecidedBy: "reviewer", Reason: "deployment approved"}); err != nil {
		t.Fatal(err)
	}
	completed, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "local_shell",
		Approved:     true,
		ApprovalID:   blocked.ApprovalID,
		Commands:     []string{"printf gated"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Decision != "DEPLOY_EXECUTION_COMPLETED" || !completed.ApprovalConsumed || !containsReason(completed.Reasons, "approval_consumed_before_deployment_execution") {
		t.Fatalf("expected approval consumption before deployment execution, got %+v", completed)
	}
	consumed, found, err := approvals.Load(root, blocked.ApprovalID)
	if err != nil || !found {
		t.Fatalf("expected consumed approval record, found=%v err=%v", found, err)
	}
	if consumed.Status != "consumed" || consumed.ConsumedBy != "deployment-executor" {
		t.Fatalf("expected consumed deployment approval, got %+v", consumed)
	}
	replay, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "local_shell",
		Approved:     true,
		ApprovalID:   blocked.ApprovalID,
		Commands:     []string{"printf gated"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if replay.Status != "blocked" || !containsReason(replay.Reasons, "approval_not_approved") {
		t.Fatalf("expected consumed approval replay to be blocked, got %+v", replay)
	}
}

func TestExecuteDoesNotConsumeApprovalForUnsafeLocalShellCommand(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	plan := createDeploymentPlanWithHealthTarget(t, root, "approval-unsafe", "http://127.0.0.1/healthz")
	approvalID := approveDeploymentExecution(t, root, plan.ID, "local_shell")
	blocked, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "local_shell",
		Approved:     true,
		ApprovalID:   approvalID,
		Commands:     []string{"rm -rf /tmp/nope"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Status != "blocked" || blocked.ApprovalConsumed || !containsReason(blocked.Reasons, "command_not_allowed") {
		t.Fatalf("expected unsafe local shell command blocked before approval consumption, got %+v", blocked)
	}
	approval, found, err := approvals.Load(root, approvalID)
	if err != nil || !found {
		t.Fatalf("expected approval record, found=%v err=%v", found, err)
	}
	if approval.Status != "approved" {
		t.Fatalf("expected unsafe command not to consume approval, got %+v", approval)
	}
}

func TestExecuteRollbackPreviewsRequestsApprovalAndConsumesBeforeLocalShell(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	failedExecution := createRollbackRequiredExecution(t, root, "rollback-controller")

	preview, err := ExecuteRollback(context.Background(), root, RollbackExecuteOptions{ExecutionID: failedExecution.ID})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Status != "completed" || preview.Decision != "ROLLBACK_PREVIEW_READY" || preview.Runbook == nil || len(preview.Steps) == 0 {
		t.Fatalf("expected rollback preview from runbook, got %+v", preview)
	}
	assertDeploymentFileExists(t, rollbackExecutionPath(root, preview.ID))
	approvalRequired, err := ExecuteRollback(context.Background(), root, RollbackExecuteOptions{
		ExecutionID: failedExecution.ID,
		Mode:        "local_shell",
		Commands:    []string{"printf rollback-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if approvalRequired.Decision != "ROLLBACK_EXECUTION_APPROVAL_REQUIRED" || approvalRequired.ApprovalID == "" || !containsReason(approvalRequired.Reasons, "approval_required_before_rollback_execution") {
		t.Fatalf("expected rollback approval request, got %+v", approvalRequired)
	}
	if _, _, err := approvals.Decide(root, approvalRequired.ApprovalID, approvals.DecisionOptions{Decision: "approved", DecidedBy: "ops", Reason: "rollback approved"}); err != nil {
		t.Fatal(err)
	}
	previewOnly, err := ExecuteRollback(context.Background(), root, RollbackExecuteOptions{
		ExecutionID: failedExecution.ID,
		Mode:        "local_shell",
		Approved:    true,
		ApprovalID:  approvalRequired.ApprovalID,
		Commands:    []string{"printf rollback-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if previewOnly.Decision != "ROLLBACK_EXECUTION_PREVIEW_ONLY" || previewOnly.ApprovalConsumed || !containsReason(previewOnly.Reasons, "rollback_execution_not_enabled") {
		t.Fatalf("expected rollback execution disabled without consuming approval, got %+v", previewOnly)
	}
	stillApproved, found, err := approvals.Load(root, approvalRequired.ApprovalID)
	if err != nil || !found {
		t.Fatalf("expected rollback approval record, found=%v err=%v", found, err)
	}
	if stillApproved.Status != "approved" {
		t.Fatalf("expected disabled rollback execution not to consume approval, got %+v", stillApproved)
	}

	t.Setenv("MOYUAN_ALLOW_ROLLBACK_EXECUTE", "1")
	completed, err := ExecuteRollback(context.Background(), root, RollbackExecuteOptions{
		ExecutionID: failedExecution.ID,
		Mode:        "local_shell",
		Approved:    true,
		ApprovalID:  approvalRequired.ApprovalID,
		Commands:    []string{"printf rollback-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Decision != "ROLLBACK_EXECUTION_COMPLETED" || !completed.ApprovalConsumed || !containsReason(completed.Reasons, "approval_consumed_before_rollback_execution") {
		t.Fatalf("expected rollback approval consumption and execution, got %+v", completed)
	}
	consumed, found, err := approvals.Load(root, approvalRequired.ApprovalID)
	if err != nil || !found {
		t.Fatalf("expected consumed rollback approval, found=%v err=%v", found, err)
	}
	if consumed.Status != "consumed" || consumed.ConsumedBy != "rollback-executor" {
		t.Fatalf("expected consumed rollback approval, got %+v", consumed)
	}
	replay, err := ExecuteRollback(context.Background(), root, RollbackExecuteOptions{
		ExecutionID: failedExecution.ID,
		Mode:        "local_shell",
		Approved:    true,
		ApprovalID:  approvalRequired.ApprovalID,
		Commands:    []string{"printf rollback-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if replay.Decision != "ROLLBACK_EXECUTION_APPROVAL_REQUIRED" || !containsReason(replay.Reasons, "approval_not_approved") {
		t.Fatalf("expected consumed rollback approval replay blocked, got %+v", replay)
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

func createRollbackRequiredExecution(t *testing.T, root string, id string) Execution {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	plan := createDeploymentPlanWithHealthTarget(t, root, id, server.URL)
	execution, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: plan.ID,
		Mode:         "local_shell",
		Approved:     true,
		ApprovalID:   approveDeploymentExecution(t, root, plan.ID, "local_shell"),
		Commands:     []string{"printf deploy-fail"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !execution.RollbackSuggestion.Required || execution.RollbackSuggestion.Runbook == nil {
		t.Fatalf("expected rollback required execution fixture, got %+v", execution)
	}
	return execution
}

func approveDeploymentExecution(t *testing.T, root string, deploymentID string, mode string) string {
	t.Helper()
	approval, err := approvals.Request(root, approvals.RequestOptions{
		TargetType:  "deployment_execution",
		TargetID:    deploymentID,
		Action:      "deploy.execute." + mode,
		RiskLevel:   "high",
		RequestedBy: "test",
		Reason:      "test deployment execution approval",
	})
	if err != nil {
		t.Fatal(err)
	}
	decided, found, err := approvals.Decide(root, approval.ID, approvals.DecisionOptions{Decision: "approved", DecidedBy: "reviewer", Reason: "test approved"})
	if err != nil || !found {
		t.Fatalf("expected approval decision, found=%v err=%v", found, err)
	}
	return decided.ID
}

func hasStep(steps []ExecutionStep, name string, status string) bool {
	for _, step := range steps {
		if step.Name == name && step.Status == status {
			return true
		}
	}
	return false
}

func hasEvidenceOperation(records []evidence.Record, operation string, decision string) bool {
	for _, record := range records {
		if record.Operation == operation && record.Decision == decision {
			return true
		}
	}
	return false
}

func hasEvidenceArtifact(records []evidence.Record, operation string, kind string) bool {
	for _, record := range records {
		if record.Operation != operation {
			continue
		}
		for _, artifact := range record.Artifacts {
			if artifact.Kind == kind {
				return true
			}
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

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func prependFakeSSH(t *testing.T, exitCode int, stdout string, stderr string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "ssh")
	script := "#!/bin/sh\n"
	if stdout != "" {
		script += "printf '%s\\n' '" + stdout + "'\n"
	}
	if stderr != "" {
		script += "printf '%s\\n' '" + stderr + "' >&2\n"
	}
	script += "exit " + strconv.Itoa(exitCode) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func assertDeploymentFileExists(t *testing.T, path string) {
	t.Helper()
	if !fsutil.Exists(path) {
		t.Fatalf("expected file to exist: %s", path)
	}
}

func assertDeploymentFileDoesNotContain(t *testing.T, path string, value string) {
	t.Helper()
	text, _, err := fsutil.ReadText(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, value) {
		t.Fatalf("expected %s not to contain secret value", path)
	}
}
