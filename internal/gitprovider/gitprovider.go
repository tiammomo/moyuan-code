package gitprovider

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	gitadapter "moyuan-code/internal/git"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/process"
	"moyuan-code/internal/review"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type Plan struct {
	ID             string               `json:"id"`
	IssueID        string               `json:"issue_id"`
	Status         string               `json:"status"`
	Decision       string               `json:"decision"`
	Reasons        []string             `json:"reasons"`
	Provider       string               `json:"provider"`
	Capabilities   Capabilities         `json:"capabilities"`
	RemoteName     string               `json:"remote_name,omitempty"`
	RemoteURL      string               `json:"remote_url,omitempty"`
	CurrentBranch  string               `json:"current_branch,omitempty"`
	TargetBranch   string               `json:"target_branch,omitempty"`
	BaseBranch     string               `json:"base_branch,omitempty"`
	PushCommand    string               `json:"push_command,omitempty"`
	PRMR           PRMRPlan             `json:"pr_mr"`
	MergeDecision  review.MergeDecision `json:"merge_decision"`
	ManualRequired bool                 `json:"manual_required"`
	CreatedAt      string               `json:"created_at"`
}

type Capabilities struct {
	Clone                bool `json:"clone"`
	Fetch                bool `json:"fetch"`
	Push                 bool `json:"push"`
	PullRequest          bool `json:"pull_request"`
	MergeRequest         bool `json:"merge_request"`
	BranchProtectionRead bool `json:"branch_protection_read"`
}

type PRMRPlan struct {
	Type         string `json:"type"`
	Title        string `json:"title,omitempty"`
	Body         string `json:"body,omitempty"`
	CreateMode   string `json:"create_mode"`
	RemoteLink   string `json:"remote_link,omitempty"`
	RemoteStatus string `json:"remote_status,omitempty"`
	SyncDecision string `json:"sync_decision,omitempty"`
	SyncReason   string `json:"sync_reason,omitempty"`
	SyncedAt     string `json:"synced_at,omitempty"`
	BlockReason  string `json:"block_reason,omitempty"`
}

func CreatePlan(ctx context.Context, rootDir string, issueID string) (Plan, error) {
	if issueID == "" {
		return Plan{}, errors.New("issue_id_required")
	}
	_ = workspace.EnsureDirs(workspace.ForRoot(rootDir))
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	plan := Plan{
		ID:        "git-provider-plan-" + textutil.Slugify(issueID) + "-" + timeID(time.Now().UTC()),
		IssueID:   issueID,
		Status:    "blocked",
		Decision:  "GIT_PROVIDER_BLOCKED",
		Reasons:   []string{},
		CreatedAt: createdAt,
	}
	status := gitadapter.StatusOf(ctx, rootDir)
	if !status.IsRepo {
		plan.Reasons = append(plan.Reasons, "not_git_repository")
		return finish(rootDir, plan)
	}
	if status.Branch != nil {
		plan.CurrentBranch = *status.Branch
	}
	if status.Dirty {
		plan.Reasons = append(plan.Reasons, "dirty_worktree")
		return finish(rootDir, plan)
	}
	mergeDecision, err := review.DecideMerge(rootDir, issueID)
	if err != nil {
		return Plan{}, err
	}
	plan.MergeDecision = mergeDecision
	if mergeDecision.Status != "ready_to_merge" || mergeDecision.Decision != "MERGE_ALLOWED" {
		plan.Reasons = append(plan.Reasons, "review_merge_not_allowed")
		plan.Reasons = append(plan.Reasons, mergeDecision.Reasons...)
		return finish(rootDir, plan)
	}
	ws, err := workspace.Load(rootDir)
	if err != nil {
		return Plan{}, err
	}
	plan.RemoteName = ws.Repository.Repository.DefaultRemote
	if plan.RemoteName == "" {
		plan.RemoteName = "origin"
	}
	remoteURL, ok := remoteURL(ctx, rootDir, plan.RemoteName)
	if !ok {
		plan.Reasons = append(plan.Reasons, "remote_missing:"+plan.RemoteName)
		return finish(rootDir, plan)
	}
	plan.RemoteURL = remoteURL
	plan.Provider = detectProvider(ws.Repository.Repository.Source.Provider, remoteURL)
	plan.Capabilities = capabilitiesFor(plan.Provider)
	plan.BaseBranch = baseBranch(ws, plan.CurrentBranch)
	plan.TargetBranch = targetBranch(issueID, plan.CurrentBranch)
	if !plan.Capabilities.Push {
		plan.Reasons = append(plan.Reasons, "provider_push_unsupported:"+plan.Provider)
		return finish(rootDir, plan)
	}
	plan.PushCommand = "git push " + plan.RemoteName + " " + plan.TargetBranch
	plan.PRMR = prmrPlan(plan.Provider, issueID, plan.BaseBranch, plan.TargetBranch, remoteURL)
	if !plan.Capabilities.PullRequest && !plan.Capabilities.MergeRequest {
		plan.Status = "push_plan_ready"
		plan.Decision = "PUSH_ALLOWED_PR_MR_UNSUPPORTED"
		plan.ManualRequired = true
		plan.Reasons = append(plan.Reasons, "provider_pr_mr_unsupported")
		return finish(rootDir, plan)
	}
	if !apiAuthAvailable(remoteURL) {
		plan.Status = "push_plan_ready"
		plan.Decision = "PUSH_ALLOWED_PR_MR_MANUAL"
		plan.ManualRequired = true
		plan.Reasons = append(plan.Reasons, "api_auth_missing_for_pr_mr")
		return finish(rootDir, plan)
	}
	plan.Status = "pr_mr_plan_ready"
	plan.Decision = "PR_MR_ALLOWED"
	plan.Reasons = append(plan.Reasons, "review_allowed_remote_ready")
	return finish(rootDir, plan)
}

func Load(rootDir string, id string) (Plan, bool, error) {
	var plan Plan
	found, err := fsutil.ReadJSON(planPath(rootDir, id), &plan)
	return plan, found, err
}

func List(rootDir string, limit int) ([]Plan, error) {
	if err := fsutil.EnsureDir(workspace.ForRoot(rootDir).PullRequestsDir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(workspace.ForRoot(rootDir).PullRequestsDir)
	if err != nil {
		return nil, err
	}
	plans := []Plan{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var plan Plan
		found, err := fsutil.ReadJSON(filepath.Join(workspace.ForRoot(rootDir).PullRequestsDir, entry.Name()), &plan)
		if err != nil {
			return nil, err
		}
		if found && plan.ID != "" {
			plans = append(plans, plan)
		}
	}
	sort.SliceStable(plans, func(i, j int) bool {
		return plans[i].CreatedAt > plans[j].CreatedAt
	})
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if len(plans) > limit {
		return plans[:limit], nil
	}
	return plans, nil
}

func SyncStatus(ctx context.Context, rootDir string, id string) (Plan, bool, error) {
	_ = ctx
	plan, found, err := Load(rootDir, id)
	if err != nil || !found {
		return plan, found, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	plan.PRMR.SyncedAt = now
	if plan.PRMR.Type == "" || plan.PRMR.Type == "manual" || plan.PRMR.CreateMode == "manual" || plan.ManualRequired {
		plan.PRMR.RemoteStatus = "manual_required"
		plan.PRMR.SyncDecision = "PR_MR_STATUS_MANUAL_REQUIRED"
		plan.PRMR.SyncReason = "manual_create_mode_or_provider_without_pr_mr_api"
	} else if !apiAuthAvailable(plan.RemoteURL) {
		plan.PRMR.RemoteStatus = "api_auth_missing"
		plan.PRMR.SyncDecision = "PR_MR_STATUS_AUTH_MISSING"
		plan.PRMR.SyncReason = "git_provider_api_auth_missing"
	} else {
		plan.PRMR.RemoteStatus = "unknown"
		plan.PRMR.SyncDecision = "PR_MR_STATUS_REFRESH_NOT_IMPLEMENTED"
		plan.PRMR.SyncReason = "provider_status_adapter_not_enabled"
	}
	if err := fsutil.WriteJSON(planPath(rootDir, plan.ID), plan); err != nil {
		return Plan{}, true, err
	}
	_ = logging.Log(rootDir, "git", "git_provider.status.synced", map[string]any{"plan_id": plan.ID, "issue_id": plan.IssueID, "decision": plan.PRMR.SyncDecision, "remote_status": plan.PRMR.RemoteStatus})
	return plan, true, nil
}

func finish(rootDir string, plan Plan) (Plan, error) {
	if err := fsutil.WriteJSON(planPath(rootDir, plan.ID), plan); err != nil {
		return Plan{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).PullRequestsDir, "plans.jsonl"), plan); err != nil {
		return Plan{}, err
	}
	_ = logging.Log(rootDir, "git", "git_provider.plan.created", map[string]any{"issue_id": plan.IssueID, "plan_id": plan.ID, "decision": plan.Decision, "status": plan.Status, "provider": plan.Provider})
	return plan, nil
}

func planPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).PullRequestsDir, id+".json")
}

func remoteURL(ctx context.Context, rootDir string, remoteName string) (string, bool) {
	res := process.RunCommand(ctx, rootDir, "git", "remote", "get-url", remoteName)
	if res.Code != 0 {
		return "", false
	}
	value := strings.TrimSpace(res.Stdout)
	return value, value != ""
}

func detectProvider(configured string, remoteURL string) string {
	configured = normalize(configured)
	remote := strings.ToLower(remoteURL)
	switch {
	case strings.Contains(remote, "github.com"):
		return "github"
	case strings.Contains(remote, "gitee.com"):
		return "gitee"
	case strings.Contains(remote, "gitlab"):
		return "gitlab"
	case configured != "" && configured != "local":
		return configured
	default:
		return "generic_git"
	}
}

func capabilitiesFor(provider string) Capabilities {
	base := Capabilities{Clone: true, Fetch: true, Push: true}
	switch provider {
	case "github":
		base.PullRequest = true
		base.BranchProtectionRead = true
	case "gitee":
		base.PullRequest = true
	case "gitlab":
		base.MergeRequest = true
		base.BranchProtectionRead = true
	case "generic_git":
	default:
		base.Push = false
	}
	return base
}

func baseBranch(ws workspace.Workspace, current string) string {
	if ws.Repository.Repository.DefaultBranch != nil && *ws.Repository.Repository.DefaultBranch != "" {
		return *ws.Repository.Repository.DefaultBranch
	}
	if current != "" {
		return current
	}
	return "main"
}

func targetBranch(issueID string, current string) string {
	if current != "" {
		return current
	}
	return "moyuan/" + textutil.Slugify(issueID)
}

func prmrPlan(provider string, issueID string, base string, branch string, remoteURL string) PRMRPlan {
	title := "[Moyuan] " + issueID
	body := "Generated by Moyuan after review merge gate accepted. Base: " + base + ". Branch: " + branch + "."
	remoteLink := remotePRMRLink(provider, remoteURL, base, branch)
	switch provider {
	case "github", "gitee":
		return PRMRPlan{Type: "pull_request", Title: title, Body: body, CreateMode: prCreateMode(remoteURL), RemoteLink: remoteLink, RemoteStatus: "not_created"}
	case "gitlab":
		return PRMRPlan{Type: "merge_request", Title: title, Body: body, CreateMode: prCreateMode(remoteURL), RemoteLink: remoteLink, RemoteStatus: "not_created"}
	default:
		return PRMRPlan{Type: "manual", CreateMode: "manual", RemoteStatus: "manual_required", BlockReason: "provider_pr_mr_unsupported"}
	}
}

func remotePRMRLink(provider string, remoteURL string, base string, branch string) string {
	webURL := repoWebURL(provider, remoteURL)
	if webURL == "" {
		return ""
	}
	switch provider {
	case "github", "gitee":
		return webURL + "/compare/" + url.PathEscape(base) + "..." + url.PathEscape(branch) + "?expand=1"
	case "gitlab":
		values := url.Values{}
		values.Set("merge_request[source_branch]", branch)
		values.Set("merge_request[target_branch]", base)
		return webURL + "/-/merge_requests/new?" + values.Encode()
	default:
		return ""
	}
}

func repoWebURL(provider string, remoteURL string) string {
	remoteURL = strings.TrimSuffix(strings.TrimSpace(remoteURL), ".git")
	if strings.HasPrefix(remoteURL, "https://") || strings.HasPrefix(remoteURL, "http://") {
		return remoteURL
	}
	hosts := map[string]string{
		"github": "github.com",
		"gitee":  "gitee.com",
		"gitlab": "gitlab.com",
	}
	host := hosts[provider]
	if host == "" {
		return ""
	}
	sshPrefix := "git@" + host + ":"
	if strings.HasPrefix(remoteURL, sshPrefix) {
		return "https://" + host + "/" + strings.TrimPrefix(remoteURL, sshPrefix)
	}
	return ""
}

func prCreateMode(remoteURL string) string {
	if apiAuthAvailable(remoteURL) {
		return "api"
	}
	return "manual"
}

func apiAuthAvailable(remoteURL string) bool {
	// Phase 4 only trusts explicit API auth when a future provider_config supplies it.
	// SSH or credential-helper Git auth may allow push, but it is not PR/MR API auth.
	return false
}

func normalize(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.ReplaceAll(value, "-", "_")
}

func timeID(value time.Time) string {
	return value.Format("20060102150405") + "-" + value.Format("000000000")
}
