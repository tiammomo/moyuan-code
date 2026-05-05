package release

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/review"
	"moyuan-code/internal/workspace"
)

func TestPlanCandidateFromSuggestedReleaseBatch(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	initReleaseGitRepo(t, root)
	runReleaseGit(t, root, "remote", "add", "origin", "git@github.com:owner/repo.git")
	batch, err := finishBatchPlan(root, BatchPlan{
		ID:                   "release-batch-v0.2.0",
		IntegrationApplyID:   "integration-apply-release",
		IntegrationPreviewID: "integration-preview-release",
		MergeQueueID:         "merge-queue-release",
		BatchID:              "batch-release",
		EpicID:               "epic-release",
		Status:               "suggested",
		Decision:             "RELEASE_BATCH_SUGGESTED",
		Version:              "v0.2.0",
		ReleaseBranch:        "release/v0.2.0",
		SourceBranch:         "moyuan/integration/release",
		ReadyItemCount:       3,
		MinItems:             3,
		Reasons:              []string{"ready_item_threshold_met"},
		Commands:             []string{"git checkout -b release/v0.2.0 moyuan/integration/release"},
		CreatedAt:            "2026-05-05T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	candidate, err := PlanCandidate(context.Background(), root, CandidateOptions{ReleaseBatchID: batch.ID, DeploymentTargets: []string{"test_dev", "production"}, RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Decision != "RELEASE_CANDIDATE_READY" || candidate.Provider != "github" || candidate.RemoteName != "origin" {
		t.Fatalf("expected ready github candidate, got %+v", candidate)
	}
	if candidate.ReleaseBranch != "release/v0.2.0" || candidate.SourceBranch != "moyuan/integration/release" || candidate.ReadyItemCount != 3 {
		t.Fatalf("expected candidate to inherit release batch facts, got %+v", candidate)
	}
	if len(candidate.DeploymentTargets) != 2 || candidate.DeploymentTargets[1] != "production" {
		t.Fatalf("expected deployment targets, got %+v", candidate.DeploymentTargets)
	}
	loaded, found, err := LoadCandidate(root, candidate.ID)
	if err != nil || !found || loaded.ID != candidate.ID {
		t.Fatalf("expected persisted candidate, found=%v loaded=%+v err=%v", found, loaded, err)
	}
	candidates, err := ListCandidates(root, batch.ID, 10)
	if err != nil || len(candidates) != 1 || candidates[0].ID != candidate.ID {
		t.Fatalf("expected listed candidate, candidates=%+v err=%v", candidates, err)
	}
}

func TestPlanCandidateBlocksWhenReleaseBatchNotSuggested(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	batch, err := finishBatchPlan(root, BatchPlan{
		ID:            "release-batch-small",
		Status:        "not_ready",
		Decision:      "RELEASE_BATCH_NOT_READY",
		Version:       "v0.2.1",
		ReleaseBranch: "release/v0.2.1",
		MinItems:      3,
		Reasons:       []string{"ready_item_count_below_threshold:3"},
		CreatedAt:     "2026-05-05T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	candidate, err := PlanCandidate(context.Background(), root, CandidateOptions{ReleaseBatchID: batch.ID})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Decision != "RELEASE_CANDIDATE_BLOCKED" || !containsReleaseReason(candidate.Reasons, "release_batch_not_suggested:RELEASE_BATCH_NOT_READY") {
		t.Fatalf("expected candidate blocked by release batch, got %+v", candidate)
	}
}

func TestApplyCandidateDryRunAndGuardedReleaseBranchApply(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	initReleaseGitRepo(t, root)
	if err := fsutil.WriteText(filepath.Join(root, "README.md"), "# release candidate\n"); err != nil {
		t.Fatal(err)
	}
	runReleaseGit(t, root, "add", ".")
	runReleaseGit(t, root, "commit", "-m", "initial")
	runReleaseGit(t, root, "branch", "moyuan/integration/release")
	candidate, err := finishCandidate(root, Candidate{
		ID:             "release-candidate-ready",
		ReleaseBatchID: "release-batch-v0.2.0",
		Status:         "ready",
		Decision:       "RELEASE_CANDIDATE_READY",
		Version:        "v0.2.0",
		ReleaseBranch:  "release/v0.2.0",
		SourceBranch:   "moyuan/integration/release",
		Reasons:        []string{"test_fixture"},
		CreatedAt:      "2026-05-05T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	dryRun, err := ApplyCandidate(context.Background(), root, CandidateApplyOptions{CandidateID: candidate.ID, Mode: "dry_run", RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.Decision != "RELEASE_BRANCH_APPLY_DRY_RUN" || dryRun.WriteEnabled {
		t.Fatalf("expected dry-run apply, got %+v", dryRun)
	}
	blocked, err := ApplyCandidate(context.Background(), root, CandidateApplyOptions{CandidateID: candidate.ID, Mode: "apply", Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Decision != "RELEASE_BRANCH_APPLY_BLOCKED" || !containsReleaseReason(blocked.Reasons, "release_branch_apply_not_enabled") {
		t.Fatalf("expected release branch apply write switch block, got %+v", blocked)
	}
	t.Setenv("MOYUAN_ALLOW_RELEASE_BRANCH_APPLY", "1")
	applied, err := ApplyCandidate(context.Background(), root, CandidateApplyOptions{CandidateID: candidate.ID, Mode: "apply", Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	if applied.Decision != "RELEASE_BRANCH_APPLY_COMPLETED" || !applied.WriteEnabled || applied.ReleaseBranch != "release/v0.2.0" {
		t.Fatalf("expected local release branch apply, got %+v", applied)
	}
	runReleaseGit(t, root, "rev-parse", "release/v0.2.0")
	loaded, found, err := LoadCandidateApply(root, applied.ID)
	if err != nil || !found || loaded.ID != applied.ID {
		t.Fatalf("expected persisted candidate apply, found=%v loaded=%+v err=%v", found, loaded, err)
	}
	applies, err := ListCandidateApplies(root, candidate.ID, 10)
	if err != nil || len(applies) < 3 || applies[0].ID != applied.ID {
		t.Fatalf("expected listed candidate applies, applies=%+v err=%v", applies, err)
	}
}

func TestApplyCandidateBlocksWhenCandidateNotReady(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	candidate, err := finishCandidate(root, Candidate{
		ID:             "release-candidate-blocked",
		ReleaseBatchID: "release-batch-blocked",
		Status:         "blocked",
		Decision:       "RELEASE_CANDIDATE_BLOCKED",
		Version:        "v0.2.0",
		CreatedAt:      "2026-05-05T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	apply, err := ApplyCandidate(context.Background(), root, CandidateApplyOptions{CandidateID: candidate.ID})
	if err != nil {
		t.Fatal(err)
	}
	if apply.Decision != "RELEASE_BRANCH_APPLY_BLOCKED" || !containsReleaseReason(apply.Reasons, "release_candidate_not_ready:RELEASE_CANDIDATE_BLOCKED") {
		t.Fatalf("expected apply blocked by candidate readiness, got %+v", apply)
	}
}

func TestProviderPreviewForCandidateRequiresAppliedReleaseBranchAndBuildsPreview(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	writeReleaseProviderRepositoryConfig(t, root, "https://api.github.test")
	candidate, err := finishCandidate(root, Candidate{
		ID:             "release-candidate-provider",
		ReleaseBatchID: "release-batch-v0.2.0",
		Status:         "ready",
		Decision:       "RELEASE_CANDIDATE_READY",
		Version:        "v0.2.0",
		Provider:       "github",
		RemoteName:     "origin",
		RemoteURL:      "git@github.com:owner/repo.git",
		ReleaseBranch:  "release/v0.2.0",
		SourceBranch:   "moyuan/integration/release",
		CreatedAt:      "2026-05-05T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	blocked, found, err := ProviderPreviewForCandidate(root, candidate.ID)
	if err != nil || !found {
		t.Fatalf("expected blocked provider preview record, found=%v err=%v", found, err)
	}
	if blocked.Decision != "RELEASE_CANDIDATE_PROVIDER_PREVIEW_BLOCKED" || !containsReleaseReason(blocked.Reasons, "release_branch_apply_missing") {
		t.Fatalf("expected preview blocked by missing release branch apply, got %+v", blocked)
	}
	_, err = finishCandidateApply(root, CandidateApply{
		ID:             "release-candidate-apply-provider",
		CandidateID:    candidate.ID,
		ReleaseBatchID: candidate.ReleaseBatchID,
		Status:         "applied",
		Decision:       "RELEASE_BRANCH_APPLY_COMPLETED",
		ReleaseBranch:  candidate.ReleaseBranch,
		SourceBranch:   candidate.SourceBranch,
		StartedAt:      "2026-05-05T00:01:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	preview, found, err := ProviderPreviewForCandidate(root, candidate.ID)
	if err != nil || !found {
		t.Fatalf("expected provider preview, found=%v err=%v", found, err)
	}
	if preview.Decision != "RELEASE_CANDIDATE_PROVIDER_PREVIEW_READY" || preview.PRMR.Type != "pull_request" || preview.PRMR.HeadBranch != candidate.ReleaseBranch {
		t.Fatalf("expected candidate provider preview ready, got %+v", preview)
	}
	if preview.RemotePlan.Decision != "RELEASE_PROVIDER_REMOTE_PLAN_READY" || !hasProviderAction(preview.RemotePlan.Actions, "create_release", "planned") {
		t.Fatalf("expected release provider remote plan, got %+v", preview.RemotePlan)
	}
	loaded, found, err := LoadCandidateProviderPreview(root, preview.ID)
	if err != nil || !found || loaded.ID != preview.ID {
		t.Fatalf("expected persisted provider preview, found=%v loaded=%+v err=%v", found, loaded, err)
	}
	previews, err := ListCandidateProviderPreviews(root, candidate.ID, 10)
	if err != nil || len(previews) < 2 || previews[0].ID != preview.ID {
		t.Fatalf("expected listed provider previews, previews=%+v err=%v", previews, err)
	}
}

func TestPlanBatchSuggestsReleaseWhenIntegrationApplyThresholdMet(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	writeIntegrationPreviewForRelease(t, root, review.IntegrationPreview{
		ID:           "integration-preview-release",
		MergeQueueID: "merge-queue-release",
		BatchID:      "batch-release",
		EpicID:       "epic-release",
		Status:       "ready",
		Decision:     "INTEGRATION_PREVIEW_READY",
		Items: []review.IntegrationPreviewItem{
			{IssueID: "issue-1", Status: "ready", Decision: "INTEGRATION_ITEM_READY"},
			{IssueID: "issue-2", Status: "ready", Decision: "INTEGRATION_ITEM_READY"},
		},
	})
	writeIntegrationApplyForRelease(t, root, review.IntegrationApply{
		ID:           "integration-apply-release",
		PreviewID:    "integration-preview-release",
		MergeQueueID: "merge-queue-release",
		BatchID:      "batch-release",
		EpicID:       "epic-release",
		Status:       "applied",
		Decision:     "INTEGRATION_APPLY_COMPLETED",
		TargetBranch: "moyuan/integration/release",
	})

	plan, err := PlanBatch(root, BatchOptions{IntegrationApplyID: "integration-apply-release", Version: "v0.2.0", MinItems: 2, RequestedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Decision != "RELEASE_BATCH_SUGGESTED" || plan.ReadyItemCount != 2 || plan.SourceBranch != "moyuan/integration/release" {
		t.Fatalf("expected suggested release batch, got %+v", plan)
	}
	loaded, found, err := LoadBatchPlan(root, plan.ID)
	if err != nil || !found || loaded.ID != plan.ID {
		t.Fatalf("expected persisted release batch, found=%v loaded=%+v err=%v", found, loaded, err)
	}
	plans, err := ListBatchPlans(root, "integration-apply-release", 10)
	if err != nil || len(plans) != 1 || plans[0].ID != plan.ID {
		t.Fatalf("expected listed release batch, plans=%+v err=%v", plans, err)
	}
}

func TestPlanBatchWaitsWhenIntegrationApplyBelowThreshold(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	writeIntegrationPreviewForRelease(t, root, review.IntegrationPreview{
		ID:           "integration-preview-small",
		MergeQueueID: "merge-queue-small",
		Status:       "ready",
		Decision:     "INTEGRATION_PREVIEW_READY",
		Items: []review.IntegrationPreviewItem{
			{IssueID: "issue-1", Status: "ready", Decision: "INTEGRATION_ITEM_READY"},
		},
	})
	writeIntegrationApplyForRelease(t, root, review.IntegrationApply{
		ID:           "integration-apply-small",
		PreviewID:    "integration-preview-small",
		MergeQueueID: "merge-queue-small",
		Status:       "applied",
		Decision:     "INTEGRATION_APPLY_COMPLETED",
		TargetBranch: "moyuan/integration/small",
	})

	plan, err := PlanBatch(root, BatchOptions{IntegrationApplyID: "integration-apply-small", Version: "v0.2.1", MinItems: 2})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Decision != "RELEASE_BATCH_NOT_READY" || plan.ReadyItemCount != 1 {
		t.Fatalf("expected not ready release batch, got %+v", plan)
	}
}

func TestProviderPreviewAndPublishApprovalFlow(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	plan := createSuggestedReleasePlan(t, root)

	preview, found, err := ProviderPreview(root, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found || preview.Status != "completed" || preview.Decision != "RELEASE_PROVIDER_PREVIEW_READY" {
		t.Fatalf("expected preview ready, found=%v execution=%+v", found, preview)
	}
	if preview.RemotePlan.Decision != "RELEASE_PROVIDER_REMOTE_PLAN_READY" || !hasProviderAction(preview.RemotePlan.Actions, "create_release", "planned") || !hasProviderAction(preview.RemotePlan.Actions, "trigger_workflow", "planned") {
		t.Fatalf("expected provider release and workflow actions, got %+v", preview.RemotePlan)
	}
	if !hasProviderActionGuard(preview.RemotePlan.Actions, "push_branch", "release_branch_required") ||
		!hasProviderActionGuard(preview.RemotePlan.Actions, "create_tag", "tag_collision_check_required") ||
		!hasProviderActionGuard(preview.RemotePlan.Actions, "trigger_workflow", "workflow_ref_required") {
		t.Fatalf("expected branch/tag/workflow preview guardrails, got %+v", preview.RemotePlan.Actions)
	}

	blocked, found, err := ProviderPublish(root, ProviderOptions{ReleaseID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !found || blocked.Decision != "RELEASE_PROVIDER_PUBLISH_APPROVAL_REQUIRED" || blocked.ApprovalID == "" {
		t.Fatalf("expected publish approval requirement, found=%v execution=%+v", found, blocked)
	}
	_, found, err = approvals.Decide(root, blocked.ApprovalID, approvals.DecisionOptions{Decision: "approved", DecidedBy: "release-manager", Reason: "release gates passed"})
	if err != nil || !found {
		t.Fatalf("expected approval decision, found=%v err=%v", found, err)
	}
	previewOnly, found, err := ProviderPublish(root, ProviderOptions{ReleaseID: plan.ID, Approved: true, ApprovalID: blocked.ApprovalID})
	if err != nil {
		t.Fatal(err)
	}
	if !found || previewOnly.Status != "blocked" || previewOnly.Decision != "RELEASE_PROVIDER_PUBLISH_PREVIEW_ONLY" {
		t.Fatalf("expected publish preview-only block, found=%v execution=%+v", found, previewOnly)
	}
	if !containsReleaseReason(previewOnly.Reasons, "release_provider_write_not_enabled") {
		t.Fatalf("expected write gate reason, got %+v", previewOnly.Reasons)
	}
	if previewOnly.WriteEnabled || previewOnly.ApprovalConsumed {
		t.Fatalf("expected preview-only publish to leave approval unconsumed and write disabled, got %+v", previewOnly)
	}
	if _, found, err := approvals.VerifyApproved(root, blocked.ApprovalID, approvals.RequestOptions{TargetType: "release_provider_publish", TargetID: plan.ID, Action: "release.provider.publish"}); err != nil || !found {
		t.Fatalf("expected preview-only publish to keep approval reusable for a later real write, found=%v err=%v", found, err)
	}
	loaded, found, err := LoadProviderExecution(root, previewOnly.ID)
	if err != nil || !found || loaded.ID != previewOnly.ID {
		t.Fatalf("expected persisted provider execution, found=%v err=%v loaded=%+v", found, err, loaded)
	}
	executions, err := ListProviderExecutions(root, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(executions) < 3 || executions[0].ID != previewOnly.ID {
		t.Fatalf("expected newest provider execution first, got %+v", executions)
	}
	evidenceRecords, err := evidence.List(root, evidence.ListOptions{ParentType: "release_provider_execution", ParentID: previewOnly.ID, Limit: 10})
	if err != nil || len(evidenceRecords) != 1 || evidenceRecords[0].Decision != previewOnly.Decision {
		t.Fatalf("expected provider execution evidence, records=%+v err=%v", evidenceRecords, err)
	}
	releaseLog, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(root).LogsDir, "release.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found || !strings.Contains(releaseLog, "release.provider.previewed") || !strings.Contains(releaseLog, "release.provider.execution.created") {
		t.Fatalf("expected provider release logs, found=%v log=%s", found, releaseLog)
	}
}

func TestProviderPublishRequiresAuthBeforeConsumingApprovalWhenWriteSwitchEnabled(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE", "1")
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	plan := createSuggestedReleasePlan(t, root)

	blocked, found, err := ProviderPublish(root, ProviderOptions{ReleaseID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !found || blocked.ApprovalID == "" {
		t.Fatalf("expected approval requirement, found=%v execution=%+v", found, blocked)
	}
	_, found, err = approvals.Decide(root, blocked.ApprovalID, approvals.DecisionOptions{Decision: "approved", DecidedBy: "release-manager", Reason: "release gates passed"})
	if err != nil || !found {
		t.Fatalf("expected approval decision, found=%v err=%v", found, err)
	}

	execution, found, err := ProviderPublish(root, ProviderOptions{ReleaseID: plan.ID, Approved: true, ApprovalID: blocked.ApprovalID})
	if err != nil {
		t.Fatal(err)
	}
	if !found || execution.Decision != "RELEASE_PROVIDER_PUBLISH_AUTH_REQUIRED" {
		t.Fatalf("expected write-enabled publish to require release provider auth, found=%v execution=%+v", found, execution)
	}
	if !execution.WriteEnabled || execution.ApprovalConsumed {
		t.Fatalf("expected auth block to keep approval unconsumed, got %+v", execution)
	}
	if !containsReleaseReasonPrefix(execution.Reasons, "release_provider_token_missing:") {
		t.Fatalf("expected missing token reason, got %+v", execution.Reasons)
	}
	if _, found, err := approvals.VerifyApproved(root, blocked.ApprovalID, approvals.RequestOptions{TargetType: "release_provider_publish", TargetID: plan.ID, Action: "release.provider.publish"}); err != nil || !found {
		t.Fatalf("expected auth block to keep approval reusable, found=%v err=%v", found, err)
	}
}

func TestProviderPublishBlocksUnsupportedProviderBeforeConsumingApproval(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE", "1")
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	plan := createSuggestedReleasePlan(t, root)
	plan.Provider = "generic_git"
	plan.RemoteURL = "ssh://git.example.test/owner/repo.git"
	if err := fsutil.WriteJSON(planPath(root, plan.ID), plan); err != nil {
		t.Fatal(err)
	}

	blocked, found, err := ProviderPublish(root, ProviderOptions{ReleaseID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !found || blocked.ApprovalID == "" {
		t.Fatalf("expected approval requirement, found=%v execution=%+v", found, blocked)
	}
	_, found, err = approvals.Decide(root, blocked.ApprovalID, approvals.DecisionOptions{Decision: "approved", DecidedBy: "release-manager", Reason: "release gates passed"})
	if err != nil || !found {
		t.Fatalf("expected approval decision, found=%v err=%v", found, err)
	}

	execution, found, err := ProviderPublish(root, ProviderOptions{ReleaseID: plan.ID, Approved: true, ApprovalID: blocked.ApprovalID})
	if err != nil {
		t.Fatal(err)
	}
	if !found || execution.Decision != "RELEASE_PROVIDER_PUBLISH_UNSUPPORTED" {
		t.Fatalf("expected unsupported provider block, found=%v execution=%+v", found, execution)
	}
	if execution.ApprovalConsumed {
		t.Fatalf("expected unsupported provider block to keep approval unconsumed, got %+v", execution)
	}
	if _, found, err := approvals.VerifyApproved(root, blocked.ApprovalID, approvals.RequestOptions{TargetType: "release_provider_publish", TargetID: plan.ID, Action: "release.provider.publish"}); err != nil || !found {
		t.Fatalf("expected unsupported provider block to keep approval reusable, found=%v err=%v", found, err)
	}
}

func TestProviderPublishUsesReleaseProviderAdapterWhenWriteSwitchEnabled(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE", "1")
	t.Setenv("RELEASE_PROVIDER_TOKEN_TEST", "github-secret-token")
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	plan := createSuggestedReleasePlan(t, root)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.URL.Path != "/repos/owner/repo/releases" {
			t.Fatalf("unexpected release provider request method/path: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer github-secret-token" {
			t.Fatalf("expected bearer auth header, got %q", got)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "github-secret-token") {
			t.Fatalf("request body must not contain github token: %s", data)
		}
		body := map[string]any{}
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatal(err)
		}
		if body["tag_name"] != plan.Version || body["name"] != plan.Version {
			t.Fatalf("unexpected release body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":42,"html_url":"https://example.test/releases/42","state":"published"}`))
	}))
	defer server.Close()
	writeReleaseProviderRepositoryConfig(t, root, server.URL)
	writeReleaseProviderSecretPolicy(t, root)

	blocked, found, err := ProviderPublish(root, ProviderOptions{ReleaseID: plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !found || blocked.ApprovalID == "" {
		t.Fatalf("expected approval requirement, found=%v execution=%+v", found, blocked)
	}
	_, found, err = approvals.Decide(root, blocked.ApprovalID, approvals.DecisionOptions{Decision: "approved", DecidedBy: "release-manager", Reason: "release gates passed"})
	if err != nil || !found {
		t.Fatalf("expected approval decision, found=%v err=%v", found, err)
	}

	execution, found, err := ProviderPublish(root, ProviderOptions{ReleaseID: plan.ID, Approved: true, ApprovalID: blocked.ApprovalID})
	if err != nil {
		t.Fatal(err)
	}
	if !found || execution.Status != "completed" || execution.Decision != "RELEASE_PROVIDER_PUBLISH_COMPLETED" {
		t.Fatalf("expected release provider publish completion, found=%v execution=%+v", found, execution)
	}
	if requests != 1 {
		t.Fatalf("expected one remote release request, got %d", requests)
	}
	if !execution.WriteEnabled || !execution.ApprovalConsumed || execution.AdapterStatus != "completed" {
		t.Fatalf("expected consumed approval and completed adapter, got %+v", execution)
	}
	if !hasProviderResult(execution.RemoteResults, "push_branch", "skipped") ||
		!hasProviderResult(execution.RemoteResults, "create_tag", "skipped") ||
		!hasProviderResult(execution.RemoteResults, "push_tag", "skipped") ||
		!hasProviderResult(execution.RemoteResults, "trigger_workflow", "skipped") ||
		!hasProviderResult(execution.RemoteResults, "create_release", "completed") {
		t.Fatalf("expected controlled remote action results, got %+v", execution.RemoteResults)
	}
	if !hasProviderResultGuard(execution.RemoteResults, "push_branch", "release_branch_required") ||
		!hasProviderResultGuard(execution.RemoteResults, "trigger_workflow", "workflow_ref_required") {
		t.Fatalf("expected skipped action results to retain guardrails, got %+v", execution.RemoteResults)
	}
	if _, _, err := approvals.VerifyApproved(root, blocked.ApprovalID, approvals.RequestOptions{TargetType: "release_provider_publish", TargetID: plan.ID, Action: "release.provider.publish"}); err == nil {
		t.Fatal("expected consumed release provider approval to fail verification")
	}

	replayed, found, err := ProviderPublish(root, ProviderOptions{ReleaseID: plan.ID, Approved: true, ApprovalID: blocked.ApprovalID})
	if err != nil {
		t.Fatal(err)
	}
	if !found || replayed.Decision != "RELEASE_PROVIDER_PUBLISH_APPROVAL_REQUIRED" || !containsReleaseReason(replayed.Reasons, "approval_not_approved") {
		t.Fatalf("expected replayed approval to be blocked, found=%v execution=%+v", found, replayed)
	}
	assertReleaseFileDoesNotContain(t, filepath.Join(workspace.ForRoot(root).LogsDir, "audit.jsonl"), "github-secret-token")
	assertReleaseFileDoesNotContain(t, providerExecutionPath(root, execution.ID), "github-secret-token")
	assertReleaseFileDoesNotContain(t, filepath.Join(workspace.ForRoot(root).ReleasesDir, "provider-executions.jsonl"), "github-secret-token")
}

func createSuggestedReleasePlan(t *testing.T, root string) Plan {
	t.Helper()
	plan, err := finish(root, Plan{
		ID:            "release-v0.2.0",
		Status:        "suggested",
		Decision:      "RELEASE_SUGGESTED",
		Version:       "v0.2.0",
		ReleaseBranch: "release/v0.2.0",
		BaseBranch:    "main",
		RemoteName:    "origin",
		RemoteURL:     "git@github.com:owner/repo.git",
		Provider:      "github",
		Reasons:       []string{"test_fixture"},
		CreatedAt:     "2026-05-05T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func hasProviderAction(actions []ProviderAction, actionType string, status string) bool {
	for _, action := range actions {
		if action.Type == actionType && action.Status == status {
			return true
		}
	}
	return false
}

func hasProviderActionGuard(actions []ProviderAction, actionType string, guardrail string) bool {
	for _, action := range actions {
		if action.Type != actionType {
			continue
		}
		for _, item := range action.Guardrails {
			if item == guardrail {
				return true
			}
		}
	}
	return false
}

func hasProviderResult(results []ProviderActionResult, actionType string, status string) bool {
	for _, result := range results {
		if result.Type == actionType && result.Status == status {
			return true
		}
	}
	return false
}

func hasProviderResultGuard(results []ProviderActionResult, actionType string, guardrail string) bool {
	for _, result := range results {
		if result.Type != actionType {
			continue
		}
		for _, item := range result.Guardrails {
			if item == guardrail {
				return true
			}
		}
	}
	return false
}

func containsReleaseReason(reasons []string, expected string) bool {
	for _, reason := range reasons {
		if reason == expected {
			return true
		}
	}
	return false
}

func containsReleaseReasonPrefix(reasons []string, expectedPrefix string) bool {
	for _, reason := range reasons {
		if strings.HasPrefix(reason, expectedPrefix) {
			return true
		}
	}
	return false
}

func writeReleaseProviderSecretPolicy(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(workspace.ForRoot(root).MoyuanDir, "policies", "secrets.yaml")
	err := fsutil.WriteText(path, strings.TrimSpace(`
schema_version: 1
secrets:
  git_provider_token:
    type: token
    ref: env:RELEASE_PROVIDER_TOKEN_TEST
    usage:
      - release.provider.publish
`)+"\n")
	if err != nil {
		t.Fatal(err)
	}
}

func writeIntegrationPreviewForRelease(t *testing.T, root string, preview review.IntegrationPreview) {
	t.Helper()
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(root).MergeReportsDir, "integration-previews", preview.ID+".json"), preview); err != nil {
		t.Fatal(err)
	}
}

func writeIntegrationApplyForRelease(t *testing.T, root string, apply review.IntegrationApply) {
	t.Helper()
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(root).MergeReportsDir, "integration-applies", apply.ID+".json"), apply); err != nil {
		t.Fatal(err)
	}
}

func writeReleaseProviderRepositoryConfig(t *testing.T, root string, apiBaseURL string) {
	t.Helper()
	path := filepath.Join(workspace.ForRoot(root).MoyuanDir, "repository.yaml")
	err := fsutil.WriteText(path, strings.TrimSpace(`
schema_version: 1
repository:
  source:
    type: remote_git
    provider: github
    url: https://github.com/owner/repo.git
  provider_config:
    owner: owner
    repo: repo
    host: github.com
    api_base_url: `+apiBaseURL+`
    web_base_url: https://github.com
    auth:
      method: https_token
      token_ref: secret:git_provider_token
  default_remote: origin
  default_branch: main
`)+"\n")
	if err != nil {
		t.Fatal(err)
	}
}

func initReleaseGitRepo(t *testing.T, root string) {
	t.Helper()
	runReleaseGit(t, root, "init")
	runReleaseGit(t, root, "config", "user.email", "test@example.com")
	runReleaseGit(t, root, "config", "user.name", "Test User")
}

func runReleaseGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

func assertReleaseFileDoesNotContain(t *testing.T, path string, value string) {
	t.Helper()
	text, _, err := fsutil.ReadText(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, value) {
		t.Fatalf("expected %s not to contain secret value", path)
	}
}
