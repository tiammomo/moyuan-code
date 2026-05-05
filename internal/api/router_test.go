package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/auth"
	"moyuan-code/internal/batch"
	"moyuan-code/internal/controlloop"
	"moyuan-code/internal/controlplane"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	gitadapter "moyuan-code/internal/git"
	"moyuan-code/internal/issues"
	"moyuan-code/internal/memory"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/release"
	"moyuan-code/internal/repair"
	"moyuan-code/internal/requirement"
	"moyuan-code/internal/review"
	runtimemgr "moyuan-code/internal/runtime"
	"moyuan-code/internal/serverresources"
	"moyuan-code/internal/store"
	"moyuan-code/internal/visuals"
	"moyuan-code/internal/workspace"
	issueworktree "moyuan-code/internal/worktree"
)

func TestGinRouterServesHealthAndProjectsFromGORMStore(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	projectRoot := filepath.Join(root, "managed")
	if err := db.UpsertProject(controlplane.Project{
		ID:           "managed",
		Name:         "managed",
		Root:         projectRoot,
		Source:       map[string]any{"type": "local_path", "provider": "local"},
		OwnerID:      "actor-local-owner",
		Status:       "active",
		RegisteredAt: "2026-05-04T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}

	router := NewRouter(Options{RootDir: root, Store: &db})
	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d body=%s", health.Code, health.Body.String())
	}
	if !jsonContains(health.Body.Bytes(), "phase1-gin-gorm") {
		t.Fatalf("health missing version: %s", health.Body.String())
	}

	projects := httptest.NewRecorder()
	router.ServeHTTP(projects, httptest.NewRequest(http.MethodGet, "/v1/projects", nil))
	if projects.Code != http.StatusOK {
		t.Fatalf("projects status = %d body=%s", projects.Code, projects.Body.String())
	}
	if !jsonContains(projects.Body.Bytes(), "managed") {
		t.Fatalf("projects missing managed project: %s", projects.Body.String())
	}
}

func TestGinRouterAuthzMiddlewareProtectsHighRiskWrites(t *testing.T) {
	root := t.TempDir()
	ws, err := workspace.Ensure(root)
	if err != nil {
		t.Fatal(err)
	}
	orgID := "org-test"
	ws.Access.Access.Mode = "team_server"
	ws.Access.Access.OrganizationID = &orgID
	ws.Access.Access.LocalOwnerID = nil
	if err := workspace.SaveAccess(root, ws.Access); err != nil {
		t.Fatal(err)
	}
	if _, err := serverresources.Add(root, serverresources.Resource{
		ID:          "dev-sec",
		Environment: "test_dev",
		Host:        "10.0.0.20",
		Provider:    "local_vm",
		Owner:       "ops",
		AuthRef:     "env:DEV_SERVER_SSH_KEY",
		ExpiresAt:   "2099-01-01",
	}); err != nil {
		t.Fatal(err)
	}
	readOnly, err := auth.CreateAPIToken(root, auth.CreateTokenOptions{Name: "read-only", ActorID: "svc-ci", Scopes: []string{"project:read"}})
	if err != nil {
		t.Fatal(err)
	}
	resourceWriter, err := auth.CreateAPIToken(root, auth.CreateTokenOptions{Name: "resource-writer", ActorID: "svc-ops", Scopes: []string{"resource:write"}})
	if err != nil {
		t.Fatal(err)
	}
	authWriter, err := auth.CreateAPIToken(root, auth.CreateTokenOptions{Name: "auth-writer", ActorID: "svc-security", Scopes: []string{"auth:write"}})
	if err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertProject(controlplane.Project{ID: "managed", Name: "managed", Root: root, Status: "active"}); err != nil {
		t.Fatal(err)
	}

	router := NewRouter(Options{RootDir: root, Store: &db})
	body := `{"expires_at":"2099-04-01","actor_id":"ops","reason":"authz test"}`
	assertPostContains(t, router, "/v1/projects/managed/resources/dev-sec/renew", body, http.StatusForbidden, "AUTH_MISSING_CREDENTIAL")
	assertPostWithHeadersContains(t, router, "/v1/projects/managed/resources/dev-sec/renew", body, map[string]string{"Authorization": "Bearer " + readOnly.TokenValue}, http.StatusForbidden, "AUTH_TOKEN_SCOPE_MISMATCH")
	assertPostWithHeadersContains(t, router, "/v1/projects/managed/resources/dev-sec/renew", body, map[string]string{"Authorization": "Bearer " + resourceWriter.TokenValue}, http.StatusOK, `"RESOURCE_RENEWAL_RECORDED"`)
	tokenBody := `{"name":"console","actor_id":"svc-console","scopes":["project:read"]}`
	assertPostContains(t, router, "/v1/projects/managed/auth/api-tokens", tokenBody, http.StatusForbidden, "AUTH_MISSING_CREDENTIAL")
	assertPostWithHeadersContains(t, router, "/v1/projects/managed/auth/api-tokens", tokenBody, map[string]string{"Authorization": "Bearer " + resourceWriter.TokenValue}, http.StatusForbidden, "AUTH_TOKEN_SCOPE_MISMATCH")
	assertPostWithHeadersContains(t, router, "/v1/projects/managed/auth/api-tokens", tokenBody, map[string]string{"Authorization": "Bearer " + authWriter.TokenValue}, http.StatusCreated, `"api_token"`)
	assertGETContains(t, router, "/v1/projects/managed/audit-events?event=auth.decision.deny&limit=5", http.StatusOK, `"auth.decision.deny"`)
}

func TestGinRouterServesProjectStateEndpoints(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.InitOwner(root, "managed"); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, root)

	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertProject(controlplane.Project{
		ID:           "managed",
		Name:         "managed",
		Root:         root,
		Source:       map[string]any{"type": "local_path", "provider": "local"},
		OwnerID:      "owner",
		Status:       "active",
		RegisteredAt: "2026-05-04T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := issues.GeneratePhase1(root); err != nil {
		t.Fatal(err)
	}
	restoreRuntimePath := prependAPIFailingCodex(t)
	recoveryResult, err := runtimemgr.Invoke(context.Background(), root, runtimemgr.Invocation{
		RunID:        "api-runtime-fail",
		RuntimeID:    "codex_cli",
		IssueID:      "api-recovery",
		Prompt:       "noop",
		WorktreePath: root,
	})
	restoreRuntimePath()
	if err != nil {
		t.Fatal(err)
	}
	if recoveryResult.RecoveryID == "" {
		t.Fatalf("expected API fixture runtime failure to create recovery: %+v", recoveryResult)
	}

	result, err := orchestrator.RunIssue(context.Background(), root, "phase1-001", "local_shell", "printf api-state")
	if err != nil {
		t.Fatal(err)
	}
	decision, err := memory.Submit(root, "decision", "Beta API should expose project quality memory for future issue runs", []string{"api", "quality"}, "api-test")
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != "recorded" {
		t.Fatalf("expected memory to record, got %s", decision.Status)
	}
	reqPlan, err := requirement.PlanFromText(root, "add backend API to inspect issue graph with go test verification")
	if err != nil {
		t.Fatal(err)
	}
	worktreeRecord, err := issueworktree.Prepare(context.Background(), root, issueworktree.PrepareOptions{EpicID: reqPlan.EpicID, IssueID: "api-worktree", RequestedBy: "api-test"})
	if err != nil {
		t.Fatal(err)
	}
	signal, err := repair.CaptureSignal(root, "test_failure", "sample API repair status", result.RunID)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := repair.Classify(root, signal)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := repair.PlanRepair(root, candidate)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := repair.RunAttempt(context.Background(), root, plan.ID, "local_shell", "printf repaired > api-repair.txt")
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Status != "repaired" {
		t.Fatalf("expected repair attempt to pass, got %s", attempt.Status)
	}

	router := NewRouter(Options{RootDir: root, Store: &db})
	assertGETContains(t, router, "/v1/projects/managed", http.StatusOK, `"project"`, `"managed"`)
	assertGETContains(t, router, "/v1/projects/managed/epics/phase1-epic/issue-graph", http.StatusOK, `"issue_graph"`, `"phase1-013"`)
	assertGETContains(t, router, "/v1/projects/managed/epics/phase1-epic/schedule", http.StatusOK, `"schedule"`, `"ready_queue"`, `"blocked_reason"`, `"dispatch_queue"`)
	assertPostContains(t, router, "/v1/projects/managed/epics/"+reqPlan.EpicID+"/batches/plan", `{"max_parallel":2,"requested_by":"api-test"}`, http.StatusCreated, `"batch_plan"`, `"BATCH_PLAN_READY"`, `"route_decision"`)
	assertGETContains(t, router, "/v1/projects/managed/epics/"+reqPlan.EpicID+"/batches?limit=3", http.StatusOK, `"batch_plans"`, `"BATCH_PLAN_READY"`)
	assertGETContains(t, router, "/v1/projects/managed/batches?limit=3", http.StatusOK, `"batch_plans"`, `"BATCH_PLAN_READY"`)
	batchPlans, err := batch.List(root, reqPlan.EpicID, 1)
	if err != nil || len(batchPlans) != 1 {
		t.Fatalf("expected API batch plan, plans=%+v err=%v", batchPlans, err)
	}
	assertPostContains(t, router, "/v1/projects/managed/batches/"+batchPlans[0].ID+"/run", `{"mode":"dry_run","requested_by":"api-test"}`, http.StatusOK, `"batch_run"`, `"BATCH_RUN_DRY_RUN"`)
	batchRuns, err := batch.ListRuns(root, batchPlans[0].ID, 1)
	if err != nil || len(batchRuns) != 1 {
		t.Fatalf("expected API batch run, runs=%+v err=%v", batchRuns, err)
	}
	assertGETContains(t, router, "/v1/projects/managed/batch-runs?limit=3", http.StatusOK, `"batch_runs"`, `"BATCH_RUN_DRY_RUN"`)
	assertGETContains(t, router, "/v1/projects/managed/batch-runs/"+batchRuns[0].ID, http.StatusOK, `"batch_run"`, batchPlans[0].ID)
	assertPostContains(t, router, "/v1/projects/managed/batches/"+batchPlans[0].ID+"/merge-queue", `{}`, http.StatusAccepted, `"merge_queue"`, `"MERGE_QUEUE_BLOCKED"`, `"batch_item_dry_run"`)
	mergeQueues, err := review.ListMergeQueues(root, batchPlans[0].ID, 1)
	if err != nil || len(mergeQueues) != 1 {
		t.Fatalf("expected API merge queue, queues=%+v err=%v", mergeQueues, err)
	}
	assertGETContains(t, router, "/v1/projects/managed/merge-queues?batch_id="+batchPlans[0].ID, http.StatusOK, `"merge_queues"`, mergeQueues[0].ID)
	assertGETContains(t, router, "/v1/projects/managed/merge-queues/"+mergeQueues[0].ID, http.StatusOK, `"merge_queue"`, `"MERGE_QUEUE_BLOCKED"`)
	assertPostContains(t, router, "/v1/projects/managed/merge-queues/"+mergeQueues[0].ID+"/integration-preview", `{}`, http.StatusAccepted, `"integration_preview"`, `"INTEGRATION_PREVIEW_BLOCKED"`, "merge_queue_not_ready:")
	integrationPreviews, err := review.ListIntegrationPreviews(root, mergeQueues[0].ID, 1)
	if err != nil || len(integrationPreviews) != 1 {
		t.Fatalf("expected API integration preview, previews=%+v err=%v", integrationPreviews, err)
	}
	assertGETContains(t, router, "/v1/projects/managed/integration-previews?queue_id="+mergeQueues[0].ID, http.StatusOK, `"integration_previews"`, integrationPreviews[0].ID)
	assertGETContains(t, router, "/v1/projects/managed/integration-previews/"+integrationPreviews[0].ID, http.StatusOK, `"integration_preview"`, `"INTEGRATION_PREVIEW_BLOCKED"`)
	assertPostContains(t, router, "/v1/projects/managed/integration-previews/"+integrationPreviews[0].ID+"/apply", `{"mode":"dry_run","requested_by":"api-test"}`, http.StatusAccepted, `"integration_apply"`, `"INTEGRATION_APPLY_BLOCKED"`, "integration_preview_not_ready:")
	integrationApplies, err := review.ListIntegrationApplies(root, integrationPreviews[0].ID, 1)
	if err != nil || len(integrationApplies) != 1 {
		t.Fatalf("expected API integration apply, applies=%+v err=%v", integrationApplies, err)
	}
	assertGETContains(t, router, "/v1/projects/managed/integration-applies?preview_id="+integrationPreviews[0].ID, http.StatusOK, `"integration_applies"`, integrationApplies[0].ID)
	assertGETContains(t, router, "/v1/projects/managed/integration-applies/"+integrationApplies[0].ID, http.StatusOK, `"integration_apply"`, `"INTEGRATION_APPLY_BLOCKED"`)
	assertPostContains(t, router, "/v1/projects/managed/integration-applies/"+integrationApplies[0].ID+"/release-batch", `{"version":"v0.2.0","min_items":2,"requested_by":"api-test"}`, http.StatusAccepted, `"release_batch"`, `"RELEASE_BATCH_BLOCKED"`, "integration_apply_not_completed:")
	releaseBatches, err := release.ListBatchPlans(root, integrationApplies[0].ID, 1)
	if err != nil || len(releaseBatches) != 1 {
		t.Fatalf("expected API release batch, batches=%+v err=%v", releaseBatches, err)
	}
	assertGETContains(t, router, "/v1/projects/managed/release-batches?integration_apply_id="+integrationApplies[0].ID, http.StatusOK, `"release_batches"`, releaseBatches[0].ID)
	assertGETContains(t, router, "/v1/projects/managed/release-batches/"+releaseBatches[0].ID, http.StatusOK, `"release_batch"`, `"RELEASE_BATCH_BLOCKED"`)
	assertPostContains(t, router, "/v1/projects/managed/release-batches/"+releaseBatches[0].ID+"/candidate", `{"deployment_targets":["test_dev"],"requested_by":"api-test"}`, http.StatusAccepted, `"release_candidate"`, `"RELEASE_CANDIDATE_BLOCKED"`, "release_batch_not_suggested:")
	releaseCandidates, err := release.ListCandidates(root, releaseBatches[0].ID, 1)
	if err != nil || len(releaseCandidates) != 1 {
		t.Fatalf("expected API release candidate, candidates=%+v err=%v", releaseCandidates, err)
	}
	assertGETContains(t, router, "/v1/projects/managed/release-candidates?release_batch_id="+releaseBatches[0].ID, http.StatusOK, `"release_candidates"`, releaseCandidates[0].ID)
	assertGETContains(t, router, "/v1/projects/managed/release-candidates/"+releaseCandidates[0].ID, http.StatusOK, `"release_candidate"`, `"RELEASE_CANDIDATE_BLOCKED"`)
	assertGETContains(t, router, "/v1/projects/managed/worktrees?issue_id=api-worktree", http.StatusOK, `"worktrees"`, worktreeRecord.ID, `"WORKTREE_READY"`)
	assertGETContains(t, router, "/v1/projects/managed/worktrees/"+worktreeRecord.ID, http.StatusOK, `"worktree"`, `"api-worktree"`)
	assertGETContains(t, router, "/v1/projects/managed/issues/phase1-001", http.StatusOK, `"issue"`, `"accepted"`)
	assertGETContains(t, router, "/v1/projects/managed/runs?limit=1", http.StatusOK, `"runs"`, result.RunID)
	assertGETContains(t, router, "/v1/projects/managed/runs/"+result.RunID, http.StatusOK, `"run"`, `"completed"`)
	assertGETContains(t, router, "/v1/projects/managed/audit-events?channel=audit&limit=20", http.StatusOK, `"audit_events"`, `"channel":"audit"`, `"auth.owner.initialized"`)
	assertGETContains(t, router, "/v1/projects/managed/audit-events?channel=../audit", http.StatusBadRequest, `"invalid_log_stream"`)
	approval := assertPostApproval(t, router, "/v1/projects/managed/approvals", `{"target_type":"deployment","target_id":"deployment-api","action":"deploy.production","risk_level":"critical","requested_by":"owner","reason":"production gate"}`, http.StatusCreated)
	assertGETContains(t, router, "/v1/projects/managed/approvals?status=pending", http.StatusOK, `"approvals"`, approval.ID, `"APPROVAL_PENDING"`)
	assertGETContains(t, router, "/v1/projects/managed/approvals/"+approval.ID, http.StatusOK, `"approval"`, `"deploy.production"`)
	assertPostContains(t, router, "/v1/projects/managed/approvals/"+approval.ID+"/decide", `{"decision":"approved","decided_by":"reviewer","reason":"release gates passed"}`, http.StatusOK, `"approval"`, `"APPROVAL_APPROVED"`)
	assertPostContains(t, router, "/v1/projects/managed/approvals", `{"target_type":"provider","target_id":"glm-api","action":"provider.probe","reason":"token=plain"}`, http.StatusBadRequest, `"approval_payload_must_not_contain_secret"`)
	session := assertPostSession(t, router, "/v1/projects/managed/auth/sessions", `{"user_id":"alice","display_name":"Alice","roles":["developer","reviewer"]}`, http.StatusCreated)
	assertGETContains(t, router, "/v1/projects/managed/auth/sessions", http.StatusOK, `"sessions"`, session.ID, `"alice"`)
	assertPostContains(t, router, "/v1/projects/managed/auth/sessions/"+session.ID+"/revoke", `{"actor_id":"owner","reason":"test"}`, http.StatusOK, `"session"`, `"revoked"`)
	apiToken := assertPostAPIToken(t, router, "/v1/projects/managed/auth/api-tokens", `{"name":"ci","actor_id":"svc-ci","scopes":["project:read"]}`, http.StatusCreated)
	assertGETContains(t, router, "/v1/projects/managed/auth/api-tokens", http.StatusOK, `"api_tokens"`, apiToken.ID, `"token_prefix"`)
	assertPostContains(t, router, "/v1/projects/managed/auth/api-tokens/"+apiToken.ID+"/revoke", `{"actor_id":"owner"}`, http.StatusOK, `"api_token"`, `"revoked"`)
	assertPostContains(t, router, "/v1/projects/managed/auth/service-accounts", `{"name":"Release Bot","roles":["release_bot"]}`, http.StatusCreated, `"service_account"`, `"svc-release-bot"`)
	assertGETContains(t, router, "/v1/projects/managed/auth/service-accounts", http.StatusOK, `"service_accounts"`, `"release_bot"`)
	assertGETContains(t, router, "/v1/projects/managed/runtime-recoveries?limit=1", http.StatusOK, `"runtime_recoveries"`, recoveryResult.RecoveryID, `"runtime_failed"`)
	assertGETContains(t, router, "/v1/projects/managed/runtime-recoveries/"+recoveryResult.RecoveryID, http.StatusOK, `"runtime_recovery"`, `"fallback_candidate"`)
	assertGETContains(t, router, "/v1/projects/managed/runtime-recoveries/"+recoveryResult.RecoveryID+"/artifacts", http.StatusOK, `"runtime_recovery_artifacts"`, `"stderr"`, "api codex failed")
	assertGETContains(t, router, "/v1/projects/managed/subagents?limit=1", http.StatusOK, `"subagents"`, result.SubagentID)
	assertGETContains(t, router, "/v1/projects/managed/subagents/"+result.SubagentID, http.StatusOK, `"subagent"`, `"output_contract"`)
	assertGETContains(t, router, "/v1/projects/managed/quality/"+result.QualityReport.ID, http.StatusOK, `"quality_report"`, `"accepted"`)
	assertGETContains(t, router, "/v1/projects/managed/quality-policy", http.StatusOK, `"quality_policy"`, `"required_checks"`)
	assertPostContains(t, router, "/v1/projects/managed/visuals/diagrams/plan", `{"diagram_type":"multi-agent","scope":"password=plain 192.168.1.2"}`, http.StatusCreated, `"visual_plan"`, `"multi_agent"`, `[REDACTED_PRIVATE_IP]`)
	visualAssets, err := visuals.ListAssets(root, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(visualAssets) != 1 {
		t.Fatalf("expected visual asset from API plan")
	}
	assertGETContains(t, router, "/v1/projects/managed/visuals/assets?limit=1", http.StatusOK, `"visual_assets"`, visualAssets[0].ID)
	assertGETContains(t, router, "/v1/projects/managed/visuals/assets/"+visualAssets[0].ID, http.StatusOK, `"visual_asset"`, `"prompt_path"`)
	assertPostContains(t, router, "/v1/projects/managed/visuals/assets/"+visualAssets[0].ID+"/render", `{"mode":"dry_run"}`, http.StatusOK, `"visual_render_execution"`, `"VISUAL_RENDER_DRY_RUN"`, `"no_image_api_called"`)
	visualRenderExecutions, err := visuals.ListRenderExecutions(root, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(visualRenderExecutions) != 1 {
		t.Fatalf("expected visual render execution from API render")
	}
	assertGETContains(t, router, "/v1/projects/managed/visuals/render-executions?limit=1", http.StatusOK, `"visual_render_executions"`, visualRenderExecutions[0].ID)
	assertGETContains(t, router, "/v1/projects/managed/visuals/render-executions/"+visualRenderExecutions[0].ID, http.StatusOK, `"visual_render_execution"`, `"script_preview"`)
	assertGETContains(t, router, "/v1/projects/managed/quality-reports?limit=1", http.StatusOK, `"quality_reports"`, `"review_status"`)
	assertGETContains(t, router, "/v1/projects/managed/quality/"+result.QualityReport.ID+"/explain", http.StatusOK, `"quality_explanation"`, `"QUALITY_ACCEPTED"`)
	assertPostContains(t, router, "/v1/projects/managed/issues/phase1-001/merge-decision", `{}`, http.StatusOK, `"merge_decision"`, `"MERGE_ALLOWED"`)
	assertPostContains(t, router, "/v1/projects/managed/issues/missing/merge-decision", `{}`, http.StatusAccepted, `"issue_state_missing"`)
	assertPostContains(t, router, "/v1/projects/managed/issues/phase1-001/git-provider-plan", `{}`, http.StatusAccepted, `"git_provider_plan"`, `"GIT_PROVIDER_BLOCKED"`, `"dirty_worktree"`)
	assertGETContains(t, router, "/v1/projects/managed/git-provider-plans?limit=5", http.StatusOK, `"git_provider_plans"`, `"GIT_PROVIDER_BLOCKED"`)
	assertPostContains(t, router, "/v1/projects/managed/releases/suggest", `{"version":"v0.1.0","min_issues":1}`, http.StatusAccepted, `"release"`, `"RELEASE_BLOCKED"`, `"dirty_worktree"`)
	releasePlan, err := release.Suggest(context.Background(), root, release.SuggestOptions{Version: "v0.1.1", MinIssues: 1})
	if err != nil {
		t.Fatal(err)
	}
	releaseProviderExecution, found, err := release.ProviderPreview(root, releasePlan.ID)
	if err != nil || !found {
		t.Fatalf("expected release provider preview execution, found=%v err=%v", found, err)
	}
	assertGETContains(t, router, "/v1/projects/managed/release-provider-executions?limit=5", http.StatusOK, `"release_provider_executions"`, releaseProviderExecution.ID)
	assertGETContains(t, router, "/v1/projects/managed/release-provider-executions/"+releaseProviderExecution.ID, http.StatusOK, `"release_provider_execution"`, releaseProviderExecution.ID)
	assertGETContains(t, router, "/v1/projects/managed/operations/release_provider/"+releaseProviderExecution.ID, http.StatusOK, `"operation_detail"`, `"release.provider.preview"`, `"artifact_count":1`)
	assertPostContains(t, router, "/v1/projects/managed/resources", `{"id":"dev-api","environment":"test_dev","host":"10.0.0.11","provider":"local_vm","owner":"dev-owner","auth_ref":"env:DEV_SERVER_SSH_KEY","expires_at":"2099-01-01"}`, http.StatusCreated, `"resource"`, `"dev-api"`)
	assertPostContains(t, router, "/v1/projects/managed/resources", `{"id":"dev-expired","environment":"test_dev","host":"10.0.0.12","provider":"local_vm","owner":"dev-owner","auth_ref":"env:DEV_SERVER_SSH_KEY","expires_at":"2000-01-01","maintenance_window":"due:2000-01-01"}`, http.StatusCreated, `"resource"`, `"dev-expired"`)
	assertGETContains(t, router, "/v1/projects/managed/resources", http.StatusOK, `"resources"`, `"dev-api"`)
	assertGETContains(t, router, "/v1/projects/managed/resources/dev-api", http.StatusOK, `"resource"`, `"test_dev"`)
	assertGETContains(t, router, "/v1/projects/managed/resources/expiration-scan", http.StatusOK, `"resources"`)
	assertPostContains(t, router, "/v1/projects/managed/resources/lifecycle/scan", `{}`, http.StatusAccepted, `"lifecycle_scan"`, `"RESOURCE_LIFECYCLE_ATTENTION_REQUIRED"`, `"RESOURCE_EXPIRED"`, `"RESOURCE_MAINTENANCE_DUE"`)
	assertGETContains(t, router, "/v1/projects/managed/resources/lifecycle-alerts?limit=5", http.StatusOK, `"lifecycle_alerts"`, `"dev-expired"`)
	assertPostContains(t, router, "/v1/projects/managed/resources/maintenance/scan", `{}`, http.StatusOK, `"maintenance_records"`)
	assertGETContains(t, router, "/v1/projects/managed/resources/maintenance?limit=5", http.StatusOK, `"maintenance_records"`)
	assertPostContains(t, router, "/v1/projects/managed/resources/dev-api/renew", `{"expires_at":"2099-02-01","actor_id":"ops","reason":"renewal test"}`, http.StatusOK, `"maintenance_record"`, `"RESOURCE_RENEWAL_RECORDED"`)
	assertPostContains(t, router, "/v1/projects/managed/resources/health-scan", `{"environment":"test_dev"}`, http.StatusOK, `"health_scan"`, `"HEALTH_SCAN_COMPLETED"`)
	assertPostContains(t, router, "/v1/projects/managed/resources/dev-api/retire", `{"actor_id":"ops","reason":"retire test"}`, http.StatusOK, `"maintenance_record"`, `"RESOURCE_RETIRED"`)
	assertPostContains(t, router, "/v1/projects/managed/resources/dev-api/disable", `{}`, http.StatusOK, `"resource"`, `"disabled"`)
	assertPostContains(t, router, "/v1/projects/managed/resources", `{"id":"prod-api","environment":"production","host":"prod.internal","provider":"aliyun","owner":"ops","auth_ref":"secret:prod_ssh_key"}`, http.StatusBadRequest, `"production_expires_at_required"`)
	assertPostContains(t, router, "/v1/projects/managed/deployments/plan", `{"release_id":"missing-release","environment":"test_dev","resource_ids":["dev-api"]}`, http.StatusAccepted, `"deployment"`, `"release_not_found"`)
	assertGETContains(t, router, "/v1/projects/managed/deployments", http.StatusOK, `"deployments"`, `"release_not_found"`)
	assertPostContains(t, router, "/v1/projects/managed/deployments/missing-deployment/execute", `{}`, http.StatusAccepted, `"execution"`, `"deployment_not_found"`)
	assertGETContains(t, router, "/v1/projects/managed/deployment-executions", http.StatusOK, `"executions"`, `"deployment_not_found"`)
	assertGETContains(t, router, "/v1/projects/managed/evidence?parent_type=deployment_execution&limit=5", http.StatusOK, `"evidence"`, `"deployment.execute.dry_run"`, `"deployment_not_found"`)
	evidenceRecords, err := evidence.List(root, evidence.ListOptions{ParentType: "deployment_execution", Limit: 1})
	if err != nil || len(evidenceRecords) != 1 {
		t.Fatalf("expected deployment evidence record, records=%+v err=%v", evidenceRecords, err)
	}
	assertGETContains(t, router, "/v1/projects/managed/deployment-monitor-history?limit=5", http.StatusOK, `"post_deployment_histories"`, `"POST_DEPLOYMENT_NOT_STARTED"`, `"execution_blocked"`)
	assertGETContains(t, router, "/v1/projects/managed/deployment-executions/"+evidenceRecords[0].ParentID+"/post-deployment-history", http.StatusOK, `"post_deployment_history"`, `"execution_blocked"`)
	assertGETContains(t, router, "/v1/projects/managed/evidence/"+evidenceRecords[0].ID, http.StatusOK, `"evidence"`, evidenceRecords[0].ID)
	assertGETContains(t, router, "/v1/projects/managed/operations/deployment/"+evidenceRecords[0].ParentID, http.StatusOK, `"operation_detail"`, `"deployment.execute.dry_run"`, `"artifact_count":1`)
	assertPostContains(t, router, "/v1/projects/managed/operations/deployment/"+evidenceRecords[0].ParentID+"/repair-candidate", `{}`, http.StatusCreated, `"operation_repair_candidate"`, `"REPAIR_CANDIDATE_CREATED"`, `"candidate_review_required"`)
	assertGETContains(t, router, "/v1/projects/managed/repair/operation-candidates?limit=5", http.StatusOK, `"operation_repair_candidates"`, `"operation_blocked"`)
	operationCandidates, err := repair.ListOperationRepairCandidates(root, 1)
	if err != nil || len(operationCandidates) != 1 {
		t.Fatalf("expected operation repair candidate for review, candidates=%+v err=%v", operationCandidates, err)
	}
	assertPostContains(t, router, "/v1/projects/managed/repair/operation-candidates/"+operationCandidates[0].ID+"/review", `{"decision":"approved","reviewer_id":"qa","reason":"open controlled repair task","next_step":"repair_attempt"}`, http.StatusOK, `"operation_repair_review"`, `"REPAIR_CANDIDATE_APPROVED"`, `"review_ready"`)
	assertGETContains(t, router, "/v1/projects/managed/repair/operation-candidates?limit=5", http.StatusOK, `"operation_repair_candidates"`, `"approved"`)
	assertGETContains(t, router, "/v1/projects/managed/operations/evidence/"+evidenceRecords[0].ID, http.StatusOK, `"operation_detail"`, `"evidence_count":1`)
	assertGETContains(t, router, "/v1/projects/managed/operations/visual_render/missing", http.StatusNotFound, `"operation not found"`)
	assertGETContains(t, router, "/v1/projects/managed/deployment-executions/missing-execution", http.StatusNotFound, `"deployment execution not found"`)
	assertGETContains(t, router, "/v1/projects/managed/requirements/"+reqPlan.ID, http.StatusOK, `"requirement"`, `"clarification_decision"`)
	assertPostContains(t, router, "/v1/projects/managed/requirements/plan", `{"text":"add backend API to inspect requirements with go test verification"}`, http.StatusCreated, `"requirement"`, `"backend-implementation"`)
	assertPostContains(t, router, "/v1/projects/managed/requirements/plan", `{"text":"tune"}`, http.StatusAccepted, `"needs_user_input"`)
	assertGETContains(t, router, "/v1/projects/managed/providers", http.StatusOK, `"providers"`, `"claude_cli"`, `"codex_cli"`)
	assertPostContains(t, router, "/v1/projects/managed/providers", `{"id":"glm-api","vendor":"zhipu","api_type":"openai-compatible","auth_ref":"env:GLM_API_KEY","enabled":true,"data_policy":{"allow_project_memory":true},"models":[{"id":"glm-4"}]}`, http.StatusCreated, `"provider"`, `"glm-api"`)
	assertGETContains(t, router, "/v1/projects/managed/providers/glm-api", http.StatusOK, `"provider"`, `"glm-4"`)
	assertPostContains(t, router, "/v1/projects/managed/providers/glm-api/ops", `{"health":{"status":"ok"},"quota":{"status":"ok","limit_tokens":1000,"used_tokens":250},"usage":{"window":"daily","requests":3},"cost":{"currency":"usd","estimated_amount":0.4,"budget_amount":5,"status":"ok"}}`, http.StatusOK, `"provider"`, `"remaining_tokens":750`, `"currency":"USD"`)
	assertPostContains(t, router, "/v1/projects/managed/provider-route", `{"role":"backend","requires_repo_edit":true}`, http.StatusOK, `"route"`, `"codex_cli"`, `"explanation"`, `"candidates"`)
	assertPostContains(t, router, "/v1/projects/managed/provider-route", `{"role":"memory_curator","task_type":"memory_extraction","includes_project_memory":true}`, http.StatusOK, `"route"`, `"glm-api"`)
	assertPostContains(t, router, "/v1/projects/managed/provider-route", `{"model_strategy":"low-cost-memory","includes_project_memory":true}`, http.StatusOK, `"route"`, `"low_cost_memory"`, `"glm-api"`)
	t.Setenv("GLM_API_KEY", "")
	assertPostContains(t, router, "/v1/projects/managed/providers/ops/refresh", `{"provider_id":"glm-api"}`, http.StatusOK, `"provider_ops_refresh"`, `"updated":1`, `"auth_ref_env_missing:GLM_API_KEY"`)
	controlLoopRun := assertPostControlLoop(t, router, "/v1/projects/managed/control-loop/run", `{}`, http.StatusAccepted)
	if len(controlLoopRun.Steps) != 3 {
		t.Fatalf("expected 3 control loop steps, got %+v", controlLoopRun.Steps)
	}
	assertGETContains(t, router, "/v1/projects/managed/control-loop/runs?limit=5", http.StatusOK, `"control_loop_runs"`, controlLoopRun.ID, `"resource_lifecycle_scan"`)
	assertGETContains(t, router, "/v1/projects/managed/control-loop/runs/"+controlLoopRun.ID, http.StatusOK, `"control_loop_run"`, `"project_comprehension_refresh"`)
	assertPostContains(t, router, "/v1/projects/managed/providers/glm-api/ops", `{"quota":{"status":"exhausted"}}`, http.StatusOK, `"quota"`, `"exhausted"`)
	assertPostContains(t, router, "/v1/projects/managed/provider-route", `{"role":"memory_curator","task_type":"memory_extraction","includes_project_memory":true}`, http.StatusOK, `"route"`, `"codex_cli"`)
	assertPostContains(t, router, "/v1/projects/managed/provider-route", `{"role":"backend","requires_repo_edit":true,"includes_secrets":true}`, http.StatusAccepted, `"ROUTE_BLOCKED"`, `"contains_secret_context"`)
	assertPostContains(t, router, "/v1/projects/managed/providers/glm-api/disable", `{}`, http.StatusOK, `"provider"`, `"enabled":false`)
	assertPostContains(t, router, "/v1/projects/managed/providers", `{"id":"bad-api","vendor":"openai","api_type":"openai","auth_ref":"plain-secret-should-not-be-stored","enabled":true}`, http.StatusBadRequest, `"auth_ref_must_be_reference"`)
	assertPostContains(t, router, "/v1/projects/managed/skills", `{"id":"tdd","source":"github:mattpocock/skills","enabled":true,"risk_level":"low","compatible_roles":["backend","tester"],"tags":["quality"]}`, http.StatusCreated, `"skill"`, `"tdd"`)
	assertGETContains(t, router, "/v1/projects/managed/skills", http.StatusOK, `"skills"`, `"github:mattpocock/skills"`)
	assertPostContains(t, router, "/v1/projects/managed/skills/recommend", `{"role":"backend","task_type":"quality","risk_level":"medium"}`, http.StatusCreated, `"skill_recommendation"`, `"tdd"`)
	assertPostContains(t, router, "/v1/projects/managed/skills/bindings", `{"skill_id":"tdd","target_type":"role","target_id":"backend"}`, http.StatusCreated, `"skill_binding"`, `"binding-role-backend-tdd"`)
	assertGETContains(t, router, "/v1/projects/managed/skills/bindings", http.StatusOK, `"skill_bindings"`, `"binding-role-backend-tdd"`)
	assertPostContains(t, router, "/v1/projects/managed/skills/bindings/binding-role-backend-tdd/disable", `{}`, http.StatusOK, `"skill_binding"`, `"disabled"`)
	assertPostContains(t, router, "/v1/projects/managed/skills/effectiveness", `{"skill_id":"tdd","issue_id":"phase1-001","outcome":"helped","quality_impact":"improved","rework_reduced":true}`, http.StatusCreated, `"skill_effectiveness"`, `"helped"`)
	assertGETContains(t, router, "/v1/projects/managed/skills/effectiveness?skill_id=tdd", http.StatusOK, `"skill_effectiveness"`, `"improved"`)
	assertPostContains(t, router, "/v1/projects/managed/skills/recommend", `{"role":"backend","task_type":"quality","risk_level":"medium"}`, http.StatusCreated, `"skill_recommendation"`, `"effectiveness_helped"`)
	assertPostContains(t, router, "/v1/projects/managed/skills/tdd/disable", `{}`, http.StatusOK, `"skill"`, `"enabled":false`)
	assertPostContains(t, router, "/v1/projects/managed/skills", `{"id":"bad-secret","source":"local","auth_ref":"sk-plain-secret"}`, http.StatusBadRequest, `"auth_ref_must_be_reference"`)
	assertGETContains(t, router, "/v1/projects/managed/memory/search?q=Beta&limit=1", http.StatusOK, `"records"`, `Beta API should expose`)
	assertGETContains(t, router, "/v1/projects/managed/memory/candidates?limit=5", http.StatusOK, `"candidates"`, `"recorded"`)
	assertGETContains(t, router, "/v1/projects/managed/repair/attempts/"+attempt.ID, http.StatusOK, `"repair_attempt"`, `"repaired"`)
	assertGETContains(t, router, "/v1/projects/missing", http.StatusNotFound, `"project not found"`)
	assertGETContains(t, router, "/v1/projects/managed/epics/missing/issue-graph", http.StatusNotFound, `"issue graph not found"`)
	assertGETContains(t, router, "/v1/projects/managed/epics/missing/schedule", http.StatusNotFound, `"schedule not found"`)
	assertGETContains(t, router, "/v1/projects/managed/issues/missing", http.StatusNotFound, `"issue state not found"`)
	assertGETContains(t, router, "/v1/projects/managed/requirements/missing", http.StatusNotFound, `"requirement plan not found"`)
}

func TestGinRouterResolvesProjectsFromControlplaneRegistryWithoutStore(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "managed")
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Ensure(projectRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := controlplane.Register(root, controlplane.Project{
		ID:      "managed",
		Name:    "managed",
		Root:    projectRoot,
		Source:  map[string]any{"type": "local_path", "provider": "local"},
		OwnerID: "owner",
		Status:  "active",
	}); err != nil {
		t.Fatal(err)
	}

	router := NewRouter(Options{RootDir: root})
	assertGETContains(t, router, "/v1/projects/managed", http.StatusOK, `"project"`, `"managed"`)
}

func jsonContains(data []byte, value string) bool {
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return false
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	return strings.Contains(string(encoded), value)
}

func assertGETContains(t *testing.T, router http.Handler, path string, status int, values ...string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != status {
		t.Fatalf("GET %s status = %d body=%s", path, recorder.Code, recorder.Body.String())
	}
	for _, value := range values {
		if !strings.Contains(recorder.Body.String(), value) {
			t.Fatalf("GET %s missing %q in body=%s", path, value, recorder.Body.String())
		}
	}
}

func assertPostContains(t *testing.T, router http.Handler, path string, body string, status int, values ...string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != status {
		t.Fatalf("POST %s status = %d body=%s", path, recorder.Code, recorder.Body.String())
	}
	for _, value := range values {
		if !strings.Contains(recorder.Body.String(), value) {
			t.Fatalf("POST %s missing %q in body=%s", path, value, recorder.Body.String())
		}
	}
}

func assertPostWithHeadersContains(t *testing.T, router http.Handler, path string, body string, headers map[string]string, status int, values ...string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	router.ServeHTTP(recorder, request)
	if recorder.Code != status {
		t.Fatalf("POST %s status = %d body=%s", path, recorder.Code, recorder.Body.String())
	}
	for _, value := range values {
		if !strings.Contains(recorder.Body.String(), value) {
			t.Fatalf("POST %s missing %q in body=%s", path, value, recorder.Body.String())
		}
	}
}

func assertPostApproval(t *testing.T, router http.Handler, path string, body string, status int) approvals.Record {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != status {
		t.Fatalf("POST %s status = %d body=%s", path, recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Approval approvals.Record `json:"approval"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode approval response failed: %v body=%s", err, recorder.Body.String())
	}
	if payload.Approval.ID == "" {
		t.Fatalf("approval response missing id: %s", recorder.Body.String())
	}
	return payload.Approval
}

func assertPostSession(t *testing.T, router http.Handler, path string, body string, status int) auth.Session {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != status {
		t.Fatalf("POST %s status = %d body=%s", path, recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Session auth.Session `json:"session"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode session response failed: %v body=%s", err, recorder.Body.String())
	}
	if payload.Session.ID == "" {
		t.Fatalf("session response missing id: %s", recorder.Body.String())
	}
	return payload.Session
}

func assertPostAPIToken(t *testing.T, router http.Handler, path string, body string, status int) auth.APIToken {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != status {
		t.Fatalf("POST %s status = %d body=%s", path, recorder.Code, recorder.Body.String())
	}
	var payload struct {
		APIToken   auth.APIToken `json:"api_token"`
		TokenValue string        `json:"token_value"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode api token response failed: %v body=%s", err, recorder.Body.String())
	}
	if payload.APIToken.ID == "" || payload.TokenValue == "" {
		t.Fatalf("api token response missing id or one-time token: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "token_hash") {
		t.Fatalf("api token response leaked token hash: %s", recorder.Body.String())
	}
	return payload.APIToken
}

func assertPostControlLoop(t *testing.T, router http.Handler, path string, body string, status int) controlloop.RunRecord {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != status {
		t.Fatalf("POST %s status = %d body=%s", path, recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Run controlloop.RunRecord `json:"control_loop_run"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode control loop response failed: %v body=%s", err, recorder.Body.String())
	}
	if payload.Run.ID == "" {
		t.Fatalf("control loop response missing id: %s", recorder.Body.String())
	}
	return payload.Run
}

func initGitRepo(t *testing.T, root string) {
	t.Helper()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	if err := fsutil.WriteText(filepath.Join(root, "go.mod"), "module apitest\n\ngo 1.22\n"); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteText(filepath.Join(root, "api_test.go"), "package apitest\n\nimport \"testing\"\n\nfunc TestAPI(t *testing.T) {}\n"); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	if err := gitadapter.BindLocal(root); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

func prependAPIFailingCodex(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "codex")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf 'api codex failed\\n' >&2\nexit 42\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	previous := os.Getenv("PATH")
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+previous); err != nil {
		t.Fatal(err)
	}
	return func() {
		_ = os.Setenv("PATH", previous)
	}
}
