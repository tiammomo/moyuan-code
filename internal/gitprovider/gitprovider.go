package gitprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/fsutil"
	gitadapter "moyuan-code/internal/git"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/process"
	"moyuan-code/internal/review"
	"moyuan-code/internal/secrets"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"

	"gopkg.in/yaml.v3"
)

type Plan struct {
	ID             string               `json:"id"`
	IssueID        string               `json:"issue_id"`
	CandidateID    string               `json:"candidate_id,omitempty"`
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
	Type            string `json:"type"`
	Title           string `json:"title,omitempty"`
	Body            string `json:"body,omitempty"`
	CreateMode      string `json:"create_mode"`
	RemoteID        string `json:"remote_id,omitempty"`
	RemoteLink      string `json:"remote_link,omitempty"`
	RemoteStatus    string `json:"remote_status,omitempty"`
	PreviewDecision string `json:"preview_decision,omitempty"`
	PreviewReason   string `json:"preview_reason,omitempty"`
	PreviewedAt     string `json:"previewed_at,omitempty"`
	CreateDecision  string `json:"create_decision,omitempty"`
	CreateReason    string `json:"create_reason,omitempty"`
	ApprovalID      string `json:"approval_id,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	SyncDecision    string `json:"sync_decision,omitempty"`
	SyncReason      string `json:"sync_reason,omitempty"`
	SyncedAt        string `json:"synced_at,omitempty"`
	BlockReason     string `json:"block_reason,omitempty"`
}

type CreateOptions struct {
	Approved   bool   `json:"approved,omitempty"`
	ApprovalID string `json:"approval_id,omitempty"`
}

type ReleaseCandidatePlanOptions struct {
	CandidateID    string   `json:"candidate_id"`
	CandidateReady bool     `json:"candidate_ready"`
	Version        string   `json:"version,omitempty"`
	Provider       string   `json:"provider,omitempty"`
	RemoteName     string   `json:"remote_name,omitempty"`
	RemoteURL      string   `json:"remote_url,omitempty"`
	ReleaseBranch  string   `json:"release_branch,omitempty"`
	Reasons        []string `json:"reasons,omitempty"`
}

type providerAPIConfig struct {
	Provider   string
	Owner      string
	Repo       string
	APIBaseURL string
	WebBaseURL string
	TokenRef   string
	Draft      bool
	Labels     []string
	Reviewers  []string
	Assignees  []string
}

type repositoryProviderConfigFile struct {
	Repository struct {
		ProviderConfig struct {
			Owner      string `yaml:"owner"`
			Repo       string `yaml:"repo"`
			Host       string `yaml:"host"`
			APIBaseURL string `yaml:"api_base_url"`
			WebBaseURL string `yaml:"web_base_url"`
			Auth       struct {
				TokenRef string `yaml:"token_ref"`
			} `yaml:"auth"`
		} `yaml:"provider_config"`
	} `yaml:"repository"`
	Git struct {
		PullRequest struct {
			Draft     bool     `yaml:"draft"`
			Labels    []string `yaml:"labels"`
			Reviewers []string `yaml:"reviewers"`
			Assignees []string `yaml:"assignees"`
		} `yaml:"pull_request"`
	} `yaml:"git"`
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
	plan.PRMR = prmrPlan(rootDir, plan.Provider, issueID, plan.BaseBranch, plan.TargetBranch, remoteURL)
	if !plan.Capabilities.PullRequest && !plan.Capabilities.MergeRequest {
		plan.Status = "push_plan_ready"
		plan.Decision = "PUSH_ALLOWED_PR_MR_UNSUPPORTED"
		plan.ManualRequired = true
		plan.Reasons = append(plan.Reasons, "provider_pr_mr_unsupported")
		return finish(rootDir, plan)
	}
	if !apiAuthAvailable(rootDir, plan.Provider, remoteURL) {
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

func PlanReleaseCandidate(rootDir string, options ReleaseCandidatePlanOptions) (Plan, error) {
	options.CandidateID = strings.TrimSpace(options.CandidateID)
	options.Provider = normalize(options.Provider)
	options.RemoteName = strings.TrimSpace(options.RemoteName)
	options.RemoteURL = strings.TrimSpace(options.RemoteURL)
	options.ReleaseBranch = strings.TrimSpace(options.ReleaseBranch)
	if options.CandidateID == "" {
		return Plan{}, errors.New("candidate_id_required")
	}
	_ = workspace.EnsureDirs(workspace.ForRoot(rootDir))
	now := time.Now().UTC()
	plan := Plan{
		ID:           "git-provider-plan-release-candidate-" + textutil.Slugify(options.CandidateID),
		IssueID:      options.CandidateID,
		CandidateID:  options.CandidateID,
		Status:       "blocked",
		Decision:     "GIT_PROVIDER_BLOCKED",
		Reasons:      append([]string{}, options.Reasons...),
		Provider:     options.Provider,
		RemoteName:   defaultString(options.RemoteName, "origin"),
		RemoteURL:    options.RemoteURL,
		TargetBranch: options.ReleaseBranch,
		CreatedAt:    now.Format(time.RFC3339Nano),
	}
	if !options.CandidateReady {
		if len(plan.Reasons) == 0 {
			plan.Reasons = append(plan.Reasons, "release_candidate_not_ready")
		}
		return finish(rootDir, plan)
	}
	if plan.TargetBranch == "" {
		plan.Reasons = append(plan.Reasons, "release_branch_missing")
		return finish(rootDir, plan)
	}
	if plan.RemoteURL == "" {
		plan.Reasons = append(plan.Reasons, "remote_url_missing")
		return finish(rootDir, plan)
	}
	if plan.Provider == "" {
		plan.Provider = detectProvider("", plan.RemoteURL)
	}
	ws, err := workspace.Load(rootDir)
	if err != nil {
		return Plan{}, err
	}
	plan.BaseBranch = baseBranch(ws, "")
	plan.Capabilities = capabilitiesFor(plan.Provider)
	if !plan.Capabilities.Push {
		plan.Reasons = append(plan.Reasons, "provider_push_unsupported:"+plan.Provider)
		return finish(rootDir, plan)
	}
	plan.PushCommand = "git push " + plan.RemoteName + " " + plan.TargetBranch
	plan.PRMR = prmrPlan(rootDir, plan.Provider, options.CandidateID, plan.BaseBranch, plan.TargetBranch, plan.RemoteURL)
	plan.PRMR.Title = "Release " + defaultString(options.Version, options.CandidateID)
	plan.PRMR.Body = "Release candidate " + options.CandidateID + " from " + plan.TargetBranch + " to " + plan.BaseBranch + "."
	if !plan.Capabilities.PullRequest && !plan.Capabilities.MergeRequest {
		plan.Status = "push_plan_ready"
		plan.Decision = "PUSH_ALLOWED_PR_MR_UNSUPPORTED"
		plan.ManualRequired = true
		plan.Reasons = append(plan.Reasons, "provider_pr_mr_unsupported")
		return finish(rootDir, plan)
	}
	if !apiAuthAvailable(rootDir, plan.Provider, plan.RemoteURL) {
		plan.Status = "push_plan_ready"
		plan.Decision = "PUSH_ALLOWED_PR_MR_MANUAL"
		plan.ManualRequired = true
		plan.Reasons = append(plan.Reasons, "api_auth_missing_for_pr_mr")
		return finish(rootDir, plan)
	}
	plan.Status = "pr_mr_plan_ready"
	plan.Decision = "PR_MR_ALLOWED"
	plan.Reasons = append(plan.Reasons, "release_candidate_ready_remote_ready")
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
	} else if !apiAuthAvailable(rootDir, plan.Provider, plan.RemoteURL) {
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

func Preview(rootDir string, id string) (Plan, bool, error) {
	plan, found, err := Load(rootDir, id)
	if err != nil || !found {
		return plan, found, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if plan.PRMR.Type == "" || plan.PRMR.Type == "manual" {
		plan.PRMR.RemoteStatus = "manual_required"
		plan.PRMR.PreviewDecision = "PR_MR_PREVIEW_MANUAL_REQUIRED"
		plan.PRMR.PreviewReason = "manual_create_mode_or_provider_without_pr_mr_api"
	} else {
		if plan.PRMR.RemoteLink == "" {
			plan.PRMR.RemoteLink = remotePRMRLink(plan.Provider, plan.RemoteURL, plan.BaseBranch, plan.TargetBranch)
		}
		plan.PRMR.RemoteStatus = "preview_ready"
		plan.PRMR.PreviewDecision = "PR_MR_PREVIEW_READY"
		plan.PRMR.PreviewReason = "remote_create_payload_ready"
	}
	plan.PRMR.PreviewedAt = now
	if err := fsutil.WriteJSON(planPath(rootDir, plan.ID), plan); err != nil {
		return Plan{}, true, err
	}
	_ = logging.Log(rootDir, "git", "git_provider.pr_mr.previewed", map[string]any{"plan_id": plan.ID, "issue_id": plan.IssueID, "decision": plan.PRMR.PreviewDecision, "remote_status": plan.PRMR.RemoteStatus})
	return plan, true, nil
}

func Create(ctx context.Context, rootDir string, id string, options CreateOptions) (Plan, bool, error) {
	plan, found, err := Load(rootDir, id)
	if err != nil || !found {
		return plan, found, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if !options.Approved {
		metadata := map[string]any{
			"plan_id":       plan.ID,
			"issue_id":      plan.IssueID,
			"provider":      plan.Provider,
			"base_branch":   plan.BaseBranch,
			"target_branch": plan.TargetBranch,
		}
		if plan.CandidateID != "" {
			metadata["candidate_id"] = plan.CandidateID
		}
		approval, err := approvals.Request(rootDir, approvals.RequestOptions{
			TargetType:  "git_provider_pr_mr",
			TargetID:    plan.ID,
			Action:      "git.pr_mr.create",
			RiskLevel:   "high",
			RequestedBy: "system",
			Reason:      "PR/MR creation writes to the remote Git provider",
			Metadata:    metadata,
		})
		if err != nil {
			return Plan{}, true, err
		}
		plan.PRMR.ApprovalID = approval.ID
		plan.PRMR.RemoteStatus = "approval_required"
		plan.PRMR.CreateDecision = "PR_MR_CREATE_APPROVAL_REQUIRED"
		plan.PRMR.CreateReason = "approval_required_before_remote_write"
		plan.PRMR.CreatedAt = now
		return saveCreateResult(rootDir, plan)
	}
	if strings.TrimSpace(options.ApprovalID) == "" {
		plan.PRMR.RemoteStatus = "approval_required"
		plan.PRMR.CreateDecision = "PR_MR_CREATE_APPROVAL_REQUIRED"
		plan.PRMR.CreateReason = "approval_id_required_before_remote_write"
		plan.PRMR.CreatedAt = now
		return saveCreateResult(rootDir, plan)
	}
	approval, found, err := approvals.VerifyApproved(rootDir, options.ApprovalID, approvals.RequestOptions{
		TargetType: "git_provider_pr_mr",
		TargetID:   plan.ID,
		Action:     "git.pr_mr.create",
	})
	if err != nil {
		plan.PRMR.ApprovalID = strings.TrimSpace(options.ApprovalID)
		plan.PRMR.RemoteStatus = "approval_required"
		plan.PRMR.CreateDecision = "PR_MR_CREATE_APPROVAL_REQUIRED"
		plan.PRMR.CreateReason = err.Error()
		plan.PRMR.CreatedAt = now
		return saveCreateResult(rootDir, plan)
	}
	if !found {
		plan.PRMR.ApprovalID = strings.TrimSpace(options.ApprovalID)
		plan.PRMR.RemoteStatus = "approval_required"
		plan.PRMR.CreateDecision = "PR_MR_CREATE_APPROVAL_REQUIRED"
		plan.PRMR.CreateReason = "approval_not_found"
		plan.PRMR.CreatedAt = now
		return saveCreateResult(rootDir, plan)
	}
	plan.PRMR.ApprovalID = approval.ID
	if plan.PRMR.Type == "" || plan.PRMR.Type == "manual" || !ensureAPICreateMode(rootDir, &plan) {
		plan.PRMR.RemoteStatus = "manual_required"
		plan.PRMR.CreateDecision = "PR_MR_CREATE_MANUAL_REQUIRED"
		plan.PRMR.CreateReason = "manual_create_mode_or_provider_without_pr_mr_api"
		plan.PRMR.CreatedAt = now
		return saveCreateResult(rootDir, plan)
	}
	if os.Getenv("MOYUAN_ALLOW_GIT_PROVIDER_WRITE") != "1" {
		plan.PRMR.RemoteStatus = "preview_ready"
		plan.PRMR.CreateDecision = "PR_MR_CREATE_PREVIEW_ONLY"
		plan.PRMR.CreateReason = "git_provider_write_not_enabled"
		plan.PRMR.CreatedAt = now
		return saveCreateResult(rootDir, plan)
	}
	if _, _, err := approvals.ConsumeApproved(rootDir, approval.ID, approvals.ConsumeOptions{
		TargetType: "git_provider_pr_mr",
		TargetID:   plan.ID,
		Action:     "git.pr_mr.create",
		ConsumedBy: "git_provider_adapter",
		Reason:     "remote PR/MR create",
	}); err != nil {
		plan.PRMR.RemoteStatus = "approval_required"
		plan.PRMR.CreateDecision = "PR_MR_CREATE_APPROVAL_REQUIRED"
		plan.PRMR.CreateReason = err.Error()
		plan.PRMR.CreatedAt = now
		return saveCreateResult(rootDir, plan)
	}
	created, err := createRemotePRMR(ctx, rootDir, plan)
	if err != nil {
		plan.PRMR.RemoteStatus = "create_failed"
		plan.PRMR.CreateDecision = "PR_MR_CREATE_FAILED"
		plan.PRMR.CreateReason = err.Error()
		plan.PRMR.CreatedAt = now
		return saveCreateResult(rootDir, plan)
	}
	plan.PRMR.RemoteID = created.RemoteID
	plan.PRMR.RemoteLink = created.RemoteLink
	plan.PRMR.RemoteStatus = created.RemoteStatus
	plan.PRMR.CreateDecision = "PR_MR_CREATED"
	plan.PRMR.CreateReason = "remote_provider_created"
	plan.PRMR.CreatedAt = now
	return saveCreateResult(rootDir, plan)
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

func saveCreateResult(rootDir string, plan Plan) (Plan, bool, error) {
	if err := fsutil.WriteJSON(planPath(rootDir, plan.ID), plan); err != nil {
		return Plan{}, true, err
	}
	_ = logging.Log(rootDir, "git", "git_provider.pr_mr.create", map[string]any{"plan_id": plan.ID, "issue_id": plan.IssueID, "decision": plan.PRMR.CreateDecision, "remote_status": plan.PRMR.RemoteStatus})
	return plan, true, nil
}

func ensureAPICreateMode(rootDir string, plan *Plan) bool {
	if plan.PRMR.CreateMode == "api" && !plan.ManualRequired {
		return true
	}
	if plan.PRMR.Type == "" || plan.PRMR.Type == "manual" {
		return false
	}
	if !apiAuthAvailable(rootDir, plan.Provider, plan.RemoteURL) {
		return false
	}
	plan.PRMR.CreateMode = "api"
	plan.ManualRequired = false
	return true
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

func prmrPlan(rootDir string, provider string, issueID string, base string, branch string, remoteURL string) PRMRPlan {
	title := "[Moyuan] " + issueID
	body := "Generated by Moyuan after review merge gate accepted. Base: " + base + ". Branch: " + branch + "."
	remoteLink := remotePRMRLink(provider, remoteURL, base, branch)
	switch provider {
	case "github", "gitee":
		return PRMRPlan{Type: "pull_request", Title: title, Body: body, CreateMode: prCreateMode(rootDir, provider, remoteURL), RemoteLink: remoteLink, RemoteStatus: "not_created"}
	case "gitlab":
		return PRMRPlan{Type: "merge_request", Title: title, Body: body, CreateMode: prCreateMode(rootDir, provider, remoteURL), RemoteLink: remoteLink, RemoteStatus: "not_created"}
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

func prCreateMode(rootDir string, provider string, remoteURL string) string {
	if apiAuthAvailable(rootDir, provider, remoteURL) {
		return "api"
	}
	return "manual"
}

func apiAuthAvailable(rootDir string, provider string, remoteURL string) bool {
	cfg := loadProviderAPIConfig(rootDir, Plan{Provider: provider, RemoteURL: remoteURL})
	if cfg.TokenRef == "" {
		return false
	}
	status, err := secrets.Status(rootDir, cfg.TokenRef, secrets.ResolveOptions{Purpose: "pull_request.create", AdapterID: provider})
	return err == nil && status.Status == "ok"
}

type remoteCreateResult struct {
	RemoteID     string
	RemoteLink   string
	RemoteStatus string
}

func createRemotePRMR(ctx context.Context, rootDir string, plan Plan) (remoteCreateResult, error) {
	cfg := loadProviderAPIConfig(rootDir, plan)
	if cfg.TokenRef == "" {
		return remoteCreateResult{}, errors.New("git_provider_token_ref_missing")
	}
	token, err := secrets.Resolve(rootDir, cfg.TokenRef, secrets.ResolveOptions{Purpose: "pull_request.create", AdapterID: plan.Provider, Required: true})
	if err != nil {
		return remoteCreateResult{}, err
	}
	if token.Status != "ok" {
		return remoteCreateResult{}, fmt.Errorf("git_provider_token_%s:%s", token.Status, token.Reason)
	}
	if cfg.Owner == "" || cfg.Repo == "" {
		return remoteCreateResult{}, errors.New("git_provider_repo_coordinates_missing")
	}
	switch plan.Provider {
	case "github":
		return createGitHubPullRequest(ctx, cfg, plan, token.Value())
	case "gitee":
		return createGiteePullRequest(ctx, cfg, plan, token.Value())
	default:
		return remoteCreateResult{}, errors.New("git_provider_create_not_supported:" + plan.Provider)
	}
}

func createGitHubPullRequest(ctx context.Context, cfg providerAPIConfig, plan Plan, token string) (remoteCreateResult, error) {
	body := map[string]any{
		"title": plan.PRMR.Title,
		"body":  plan.PRMR.Body,
		"head":  plan.TargetBranch,
		"base":  plan.BaseBranch,
		"draft": cfg.Draft,
	}
	return postPRMR(ctx, cfg.APIBaseURL+"/repos/"+url.PathEscape(cfg.Owner)+"/"+url.PathEscape(cfg.Repo)+"/pulls", "Bearer "+token, body)
}

func createGiteePullRequest(ctx context.Context, cfg providerAPIConfig, plan Plan, token string) (remoteCreateResult, error) {
	body := map[string]any{
		"title":        plan.PRMR.Title,
		"body":         plan.PRMR.Body,
		"head":         plan.TargetBranch,
		"base":         plan.BaseBranch,
		"access_token": token,
	}
	return postPRMR(ctx, cfg.APIBaseURL+"/repos/"+url.PathEscape(cfg.Owner)+"/"+url.PathEscape(cfg.Repo)+"/pulls", "", body)
}

func postPRMR(ctx context.Context, endpoint string, authorization string, body map[string]any) (remoteCreateResult, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return remoteCreateResult{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return remoteCreateResult{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "moyuan-code-git-provider/1")
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return remoteCreateResult{}, errors.New("git_provider_pr_mr_request_failed")
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return remoteCreateResult{}, fmt.Errorf("git_provider_pr_mr_http_%d", response.StatusCode)
	}
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return remoteCreateResult{}, errors.New("git_provider_pr_mr_response_invalid")
	}
	return remoteCreateResult{
		RemoteID:     jsonString(raw, "number", "id"),
		RemoteLink:   jsonString(raw, "html_url", "url"),
		RemoteStatus: defaultString(jsonString(raw, "state", "status"), "open"),
	}, nil
}

func loadProviderAPIConfig(rootDir string, plan Plan) providerAPIConfig {
	provider := normalize(plan.Provider)
	cfg := providerAPIConfig{Provider: provider, TokenRef: "secret:git_provider_token"}
	switch provider {
	case "github":
		cfg.APIBaseURL = "https://api.github.com"
		cfg.WebBaseURL = "https://github.com"
	case "gitee":
		cfg.APIBaseURL = "https://gitee.com/api/v5"
		cfg.WebBaseURL = "https://gitee.com"
	case "gitlab":
		cfg.APIBaseURL = "https://gitlab.com/api/v4"
		cfg.WebBaseURL = "https://gitlab.com"
	}
	if owner, repo := repoCoordinates(plan.RemoteURL); owner != "" && repo != "" {
		cfg.Owner = owner
		cfg.Repo = repo
	}
	raw, found, err := readRepositoryProviderConfig(rootDir)
	if err == nil && found {
		providerConfig := raw.Repository.ProviderConfig
		if strings.TrimSpace(providerConfig.Owner) != "" {
			cfg.Owner = strings.TrimSpace(providerConfig.Owner)
		}
		if strings.TrimSpace(providerConfig.Repo) != "" {
			cfg.Repo = strings.TrimSpace(providerConfig.Repo)
		}
		if strings.TrimSpace(providerConfig.APIBaseURL) != "" {
			cfg.APIBaseURL = strings.TrimRight(strings.TrimSpace(providerConfig.APIBaseURL), "/")
		}
		if strings.TrimSpace(providerConfig.WebBaseURL) != "" {
			cfg.WebBaseURL = strings.TrimRight(strings.TrimSpace(providerConfig.WebBaseURL), "/")
		}
		if strings.TrimSpace(providerConfig.Auth.TokenRef) != "" {
			cfg.TokenRef = strings.TrimSpace(providerConfig.Auth.TokenRef)
		}
		cfg.Draft = raw.Git.PullRequest.Draft
		cfg.Labels = normalizeList(raw.Git.PullRequest.Labels)
		cfg.Reviewers = normalizeList(raw.Git.PullRequest.Reviewers)
		cfg.Assignees = normalizeList(raw.Git.PullRequest.Assignees)
	}
	return cfg
}

func readRepositoryProviderConfig(rootDir string) (repositoryProviderConfigFile, bool, error) {
	text, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(rootDir).MoyuanDir, "repository.yaml"))
	if err != nil || !found {
		return repositoryProviderConfigFile{}, found, err
	}
	var raw repositoryProviderConfigFile
	if err := yaml.Unmarshal([]byte(text), &raw); err != nil {
		return repositoryProviderConfigFile{}, true, err
	}
	return raw, true, nil
}

func repoCoordinates(remoteURL string) (string, string) {
	web := repoWebURL("github", remoteURL)
	if web == "" {
		web = repoWebURL("gitee", remoteURL)
	}
	if web == "" {
		return "", ""
	}
	parsed, err := url.Parse(strings.TrimSuffix(web, ".git"))
	if err != nil {
		return "", ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git")
}

func jsonString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			switch typed := value.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					return strings.TrimSpace(typed)
				}
			case float64:
				return fmt.Sprintf("%.0f", typed)
			}
		}
	}
	return ""
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func normalizeList(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func normalize(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.ReplaceAll(value, "-", "_")
}

func timeID(value time.Time) string {
	return value.Format("20060102150405") + "-" + value.Format("000000000")
}
