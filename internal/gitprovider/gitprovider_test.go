package gitprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

func TestPRMRPreviewApprovalAndGitHubCreate(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_PROVIDER_TOKEN_TEST", "github-secret-token")
	writeGitProviderSecretPolicy(t, root)

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/repos/moyuan/example/pulls" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer github-secret-token" {
			t.Fatalf("unexpected authorization header")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["title"] != "[Moyuan] phase5-003" || body["head"] != "moyuan/phase5-003" || body["base"] != "main" {
			t.Fatalf("unexpected request body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"number": 42, "html_url": "https://github.com/moyuan/example/pull/42", "state": "open"}`))
	}))
	defer server.Close()
	writeGitProviderRepositoryConfig(t, root, server.URL)

	plan, err := finish(root, Plan{
		ID:           "git-provider-plan-phase5-003",
		IssueID:      "phase5-003",
		Status:       "pr_mr_plan_ready",
		Decision:     "PR_MR_ALLOWED",
		Provider:     "github",
		RemoteName:   "origin",
		RemoteURL:    "https://github.com/moyuan/example.git",
		BaseBranch:   "main",
		TargetBranch: "moyuan/phase5-003",
		PRMR: PRMRPlan{
			Type:         "pull_request",
			Title:        "[Moyuan] phase5-003",
			Body:         "test body",
			CreateMode:   "api",
			RemoteLink:   "https://github.com/moyuan/example/compare/main...moyuan/phase5-003?expand=1",
			RemoteStatus: "not_created",
		},
		CreatedAt: nowForTest(),
	})
	if err != nil {
		t.Fatal(err)
	}

	preview, ok, err := Preview(root, plan.ID)
	if err != nil || !ok {
		t.Fatalf("preview failed ok=%v err=%v", ok, err)
	}
	if preview.PRMR.PreviewDecision != "PR_MR_PREVIEW_READY" || preview.PRMR.RemoteStatus != "preview_ready" || requests != 0 {
		t.Fatalf("unexpected preview result requests=%d plan=%+v", requests, preview.PRMR)
	}

	approvalRequired, ok, err := Create(context.Background(), root, plan.ID, CreateOptions{})
	if err != nil || !ok {
		t.Fatalf("create approval failed ok=%v err=%v", ok, err)
	}
	if approvalRequired.PRMR.CreateDecision != "PR_MR_CREATE_APPROVAL_REQUIRED" || approvalRequired.PRMR.ApprovalID == "" || requests != 0 {
		t.Fatalf("expected approval required without remote request, requests=%d plan=%+v", requests, approvalRequired.PRMR)
	}

	missingProof, ok, err := Create(context.Background(), root, plan.ID, CreateOptions{Approved: true})
	if err != nil || !ok {
		t.Fatalf("create missing approval proof failed ok=%v err=%v", ok, err)
	}
	if missingProof.PRMR.CreateDecision != "PR_MR_CREATE_APPROVAL_REQUIRED" || missingProof.PRMR.CreateReason != "approval_id_required_before_remote_write" || requests != 0 {
		t.Fatalf("expected approval id requirement, requests=%d plan=%+v", requests, missingProof.PRMR)
	}

	if _, _, err := approvals.Decide(root, approvalRequired.PRMR.ApprovalID, approvals.DecisionOptions{Decision: "approved", DecidedBy: "reviewer", Reason: "test approved"}); err != nil {
		t.Fatal(err)
	}

	previewOnly, ok, err := Create(context.Background(), root, plan.ID, CreateOptions{Approved: true, ApprovalID: approvalRequired.PRMR.ApprovalID})
	if err != nil || !ok {
		t.Fatalf("create preview-only failed ok=%v err=%v", ok, err)
	}
	if previewOnly.PRMR.CreateDecision != "PR_MR_CREATE_PREVIEW_ONLY" || requests != 0 {
		t.Fatalf("expected preview-only with write gate disabled, requests=%d plan=%+v", requests, previewOnly.PRMR)
	}

	t.Setenv("MOYUAN_ALLOW_GIT_PROVIDER_WRITE", "1")
	created, ok, err := Create(context.Background(), root, plan.ID, CreateOptions{Approved: true, ApprovalID: approvalRequired.PRMR.ApprovalID})
	if err != nil || !ok {
		t.Fatalf("create remote failed ok=%v err=%v", ok, err)
	}
	if requests != 1 || created.PRMR.CreateDecision != "PR_MR_CREATED" || created.PRMR.RemoteID != "42" || created.PRMR.RemoteStatus != "open" {
		t.Fatalf("unexpected created result requests=%d plan=%+v", requests, created.PRMR)
	}
	consumed, found, err := approvals.Load(root, approvalRequired.PRMR.ApprovalID)
	if err != nil || !found {
		t.Fatalf("approval load failed found=%v err=%v", found, err)
	}
	if consumed.Status != "consumed" || consumed.Decision != "APPROVAL_CONSUMED" {
		t.Fatalf("expected consumed approval, got %+v", consumed)
	}
	replay, ok, err := Create(context.Background(), root, plan.ID, CreateOptions{Approved: true, ApprovalID: approvalRequired.PRMR.ApprovalID})
	if err != nil || !ok {
		t.Fatalf("create replay failed ok=%v err=%v", ok, err)
	}
	if requests != 1 || replay.PRMR.CreateDecision != "PR_MR_CREATE_APPROVAL_REQUIRED" || replay.PRMR.CreateReason != "approval_not_approved" {
		t.Fatalf("expected replay blocked, requests=%d plan=%+v", requests, replay.PRMR)
	}
	assertFileDoesNotContain(t, filepath.Join(workspace.ForRoot(root).LogsDir, "audit.jsonl"), "github-secret-token")
	assertFileDoesNotContain(t, planPath(root, plan.ID), "github-secret-token")
	assertFileDoesNotContain(t, filepath.Join(workspace.ForRoot(root).PullRequestsDir, "plans.jsonl"), "github-secret-token")
}

func writeGitProviderSecretPolicy(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(workspace.ForRoot(root).MoyuanDir, "policies", "secrets.yaml")
	err := fsutil.WriteText(path, strings.TrimSpace(`
schema_version: 1
secrets:
  git_provider_token:
    type: token
    ref: env:GIT_PROVIDER_TOKEN_TEST
    usage:
      - pull_request.create
`)+"\n")
	if err != nil {
		t.Fatal(err)
	}
}

func writeGitProviderRepositoryConfig(t *testing.T, root string, apiBaseURL string) {
	t.Helper()
	path := filepath.Join(workspace.ForRoot(root).MoyuanDir, "repository.yaml")
	err := fsutil.WriteText(path, strings.TrimSpace(`
schema_version: 1
repository:
  source:
    type: remote_git
    provider: github
    url: https://github.com/moyuan/example.git
  provider_config:
    owner: moyuan
    repo: example
    host: github.com
    api_base_url: `+apiBaseURL+`
    web_base_url: https://github.com
    auth:
      method: https_token
      token_ref: secret:git_provider_token
  default_remote: origin
  default_branch: main
git:
  pull_request:
    draft: true
`)+"\n")
	if err != nil {
		t.Fatal(err)
	}
}

func assertFileDoesNotContain(t *testing.T, path string, value string) {
	t.Helper()
	text, _, err := fsutil.ReadText(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, value) {
		t.Fatalf("%s leaked %q", path, value)
	}
}

func nowForTest() string {
	return "2026-05-05T00:00:00Z"
}
