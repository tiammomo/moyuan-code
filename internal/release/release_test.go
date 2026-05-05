package release

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

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

func hasProviderResult(results []ProviderActionResult, actionType string, status string) bool {
	for _, result := range results {
		if result.Type == actionType && result.Status == status {
			return true
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
