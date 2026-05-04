package release

import (
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/approvals"
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
	releaseLog, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(root).LogsDir, "release.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found || !strings.Contains(releaseLog, "release.provider.previewed") || !strings.Contains(releaseLog, "release.provider.execution.created") {
		t.Fatalf("expected provider release logs, found=%v log=%s", found, releaseLog)
	}
}

func TestProviderPublishConsumesApprovalWhenWriteSwitchEnabled(t *testing.T) {
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
	if !found || execution.Decision != "RELEASE_PROVIDER_PUBLISH_NOT_IMPLEMENTED" {
		t.Fatalf("expected write-enabled publish to reach guarded adapter boundary, found=%v execution=%+v", found, execution)
	}
	if !execution.WriteEnabled || !execution.ApprovalConsumed {
		t.Fatalf("expected approval consumption with write switch enabled, got %+v", execution)
	}
	if !containsReleaseReason(execution.Reasons, "approval_consumed_before_remote_release_write") {
		t.Fatalf("expected approval consumed reason, got %+v", execution.Reasons)
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

func containsReleaseReason(reasons []string, expected string) bool {
	for _, reason := range reasons {
		if reason == expected {
			return true
		}
	}
	return false
}
