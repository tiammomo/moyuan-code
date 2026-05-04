package release

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"moyuan-code/internal/approvals"
	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	gitadapter "moyuan-code/internal/git"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/process"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type SuggestOptions struct {
	Version   string `json:"version,omitempty"`
	MinIssues int    `json:"min_issues,omitempty"`
}

type Plan struct {
	ID             string         `json:"id"`
	Status         string         `json:"status"`
	Decision       string         `json:"decision"`
	Version        string         `json:"version"`
	ReleaseBranch  string         `json:"release_branch"`
	BaseBranch     string         `json:"base_branch,omitempty"`
	RemoteName     string         `json:"remote_name,omitempty"`
	RemoteURL      string         `json:"remote_url,omitempty"`
	Provider       string         `json:"provider,omitempty"`
	IncludedIssues []IssueSummary `json:"included_issues"`
	ExcludedIssues []IssueSummary `json:"excluded_issues"`
	Reasons        []string       `json:"reasons"`
	BatchScore     float64        `json:"batch_score"`
	MinIssues      int            `json:"min_issues"`
	Commands       []string       `json:"commands"`
	NotesPath      string         `json:"notes_path,omitempty"`
	CreatedAt      string         `json:"created_at"`
}

type IssueSummary struct {
	IssueID         string `json:"issue_id"`
	Status          string `json:"status"`
	QualityReportID string `json:"quality_report_id,omitempty"`
}

type ProviderOptions struct {
	ReleaseID  string `json:"release_id"`
	Approved   bool   `json:"approved,omitempty"`
	ApprovalID string `json:"approval_id,omitempty"`
}

type ProviderExecution struct {
	ID               string     `json:"id"`
	ReleaseID        string     `json:"release_id"`
	Version          string     `json:"version,omitempty"`
	Provider         string     `json:"provider,omitempty"`
	Mode             string     `json:"mode"`
	Status           string     `json:"status"`
	Decision         string     `json:"decision"`
	Reasons          []string   `json:"reasons"`
	RemotePlan       RemotePlan `json:"remote_plan"`
	ApprovalID       string     `json:"approval_id,omitempty"`
	ApprovalConsumed bool       `json:"approval_consumed"`
	WriteEnabled     bool       `json:"write_enabled"`
	StartedAt        string     `json:"started_at"`
	FinishedAt       string     `json:"finished_at,omitempty"`
}

type RemotePlan struct {
	Status        string           `json:"status"`
	Decision      string           `json:"decision"`
	RemoteName    string           `json:"remote_name,omitempty"`
	RemoteURL     string           `json:"remote_url,omitempty"`
	Provider      string           `json:"provider,omitempty"`
	ReleaseBranch string           `json:"release_branch,omitempty"`
	Version       string           `json:"version,omitempty"`
	Actions       []ProviderAction `json:"actions"`
	CreatedAt     string           `json:"created_at"`
}

type ProviderAction struct {
	Type     string `json:"type"`
	Status   string `json:"status"`
	Command  string `json:"command,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

func Suggest(ctx context.Context, rootDir string, options SuggestOptions) (Plan, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Plan{}, err
	}
	if options.MinIssues <= 0 {
		options.MinIssues = 3
	}
	now := time.Now().UTC()
	version := normalizeVersion(options.Version, now)
	plan := Plan{
		ID:             "release-" + textutil.Slugify(version) + "-" + now.Format("20060102150405"),
		Status:         "blocked",
		Decision:       "RELEASE_BLOCKED",
		Version:        version,
		ReleaseBranch:  "release/" + version,
		IncludedIssues: []IssueSummary{},
		ExcludedIssues: []IssueSummary{},
		Reasons:        []string{},
		MinIssues:      options.MinIssues,
		CreatedAt:      now.Format(time.RFC3339Nano),
	}
	status := gitadapter.StatusOf(ctx, rootDir)
	if !status.IsRepo {
		plan.Reasons = append(plan.Reasons, "not_git_repository")
		return finish(rootDir, plan)
	}
	if status.Dirty {
		plan.Reasons = append(plan.Reasons, "dirty_worktree")
		return finish(rootDir, plan)
	}
	if status.Branch != nil {
		plan.BaseBranch = *status.Branch
	}
	remoteName := "origin"
	ws, err := workspace.Load(rootDir)
	if err != nil {
		return Plan{}, err
	}
	if ws.Repository.Repository.DefaultRemote != "" {
		remoteName = ws.Repository.Repository.DefaultRemote
	}
	plan.RemoteName = remoteName
	if remoteURL, ok := remoteURL(ctx, rootDir, remoteName); ok {
		plan.RemoteURL = remoteURL
		plan.Provider = detectProvider(ws.Repository.Repository.Source.Provider, remoteURL)
	} else {
		plan.Reasons = append(plan.Reasons, "remote_missing:"+remoteName)
		return finish(rootDir, plan)
	}
	included, excluded, err := issueSummaries(rootDir)
	if err != nil {
		return Plan{}, err
	}
	plan.IncludedIssues = included
	plan.ExcludedIssues = excluded
	plan.BatchScore = float64(len(included))
	if len(included) == 0 {
		plan.Reasons = append(plan.Reasons, "no_accepted_issues")
		return finish(rootDir, plan)
	}
	for _, item := range excluded {
		if item.Status == "running" || item.Status == "quality_checking" || item.Status == "needs_rework" || item.Status == "failed" {
			plan.Reasons = append(plan.Reasons, "unresolved_issue:"+item.IssueID)
			return finish(rootDir, plan)
		}
	}
	plan.Commands = []string{
		"git checkout -b " + plan.ReleaseBranch,
		"git push " + plan.RemoteName + " " + plan.ReleaseBranch,
		"git tag " + plan.Version,
	}
	notesRelPath, err := writeNotes(rootDir, plan)
	if err != nil {
		return Plan{}, err
	}
	plan.NotesPath = notesRelPath
	if len(included) < options.MinIssues {
		plan.Status = "not_ready"
		plan.Decision = "RELEASE_NOT_READY"
		plan.Reasons = append(plan.Reasons, "accepted_issue_count_below_threshold:"+strconv.Itoa(options.MinIssues))
		return finish(rootDir, plan)
	}
	plan.Status = "suggested"
	plan.Decision = "RELEASE_SUGGESTED"
	plan.Reasons = append(plan.Reasons, "accepted_issue_threshold_met")
	return finish(rootDir, plan)
}

func Load(rootDir string, id string) (Plan, bool, error) {
	var plan Plan
	found, err := fsutil.ReadJSON(planPath(rootDir, id), &plan)
	return plan, found, err
}

func ProviderPreview(rootDir string, releaseID string) (ProviderExecution, bool, error) {
	plan, found, err := Load(rootDir, releaseID)
	if err != nil || !found {
		return ProviderExecution{}, found, err
	}
	execution := newProviderExecution(plan, "preview")
	if !releaseReady(plan) {
		execution.Reasons = append(execution.Reasons, "release_not_suggested:"+plan.Decision)
		return finishProviderExecution(rootDir, execution)
	}
	execution.RemotePlan = buildProviderRemotePlan(plan)
	execution.Status = "completed"
	execution.Decision = "RELEASE_PROVIDER_PREVIEW_READY"
	execution.Reasons = append(execution.Reasons, "no_remote_release_actions_executed")
	_ = logging.Log(rootDir, "release", "release.provider.previewed", map[string]any{
		"release_id": releaseID,
		"version":    plan.Version,
		"provider":   plan.Provider,
		"decision":   execution.Decision,
	})
	return finishProviderExecution(rootDir, execution)
}

func ProviderPublish(rootDir string, options ProviderOptions) (ProviderExecution, bool, error) {
	options.ReleaseID = strings.TrimSpace(options.ReleaseID)
	options.ApprovalID = strings.TrimSpace(options.ApprovalID)
	plan, found, err := Load(rootDir, options.ReleaseID)
	if err != nil || !found {
		return ProviderExecution{}, found, err
	}
	execution := newProviderExecution(plan, "publish")
	if !releaseReady(plan) {
		execution.Reasons = append(execution.Reasons, "release_not_suggested:"+plan.Decision)
		return finishProviderExecution(rootDir, execution)
	}
	execution.RemotePlan = buildProviderRemotePlan(plan)
	if !options.Approved {
		approval, err := approvals.Request(rootDir, approvals.RequestOptions{
			TargetType:  "release_provider_publish",
			TargetID:    plan.ID,
			Action:      "release.provider.publish",
			RiskLevel:   "high",
			RequestedBy: "system",
			Reason:      "release provider publish writes branch, tag, release, or workflow state to remote Git provider",
			Metadata: map[string]any{
				"release_id": plan.ID,
				"version":    plan.Version,
				"provider":   plan.Provider,
			},
		})
		if err != nil {
			return ProviderExecution{}, true, err
		}
		execution.ApprovalID = approval.ID
		execution.Status = "blocked"
		execution.Decision = "RELEASE_PROVIDER_PUBLISH_APPROVAL_REQUIRED"
		execution.Reasons = append(execution.Reasons, "approval_required_before_remote_release_write")
		return finishProviderExecution(rootDir, execution)
	}
	if options.ApprovalID == "" {
		execution.Status = "blocked"
		execution.Decision = "RELEASE_PROVIDER_PUBLISH_APPROVAL_REQUIRED"
		execution.Reasons = append(execution.Reasons, "approval_id_required_before_remote_release_write")
		return finishProviderExecution(rootDir, execution)
	}
	approval, approved, err := approvals.VerifyApproved(rootDir, options.ApprovalID, approvals.RequestOptions{
		TargetType: "release_provider_publish",
		TargetID:   plan.ID,
		Action:     "release.provider.publish",
	})
	execution.ApprovalID = options.ApprovalID
	if err != nil {
		execution.Status = "blocked"
		execution.Decision = "RELEASE_PROVIDER_PUBLISH_APPROVAL_REQUIRED"
		execution.Reasons = append(execution.Reasons, err.Error())
		return finishProviderExecution(rootDir, execution)
	}
	if !approved {
		execution.Status = "blocked"
		execution.Decision = "RELEASE_PROVIDER_PUBLISH_APPROVAL_REQUIRED"
		execution.Reasons = append(execution.Reasons, "approval_not_found")
		return finishProviderExecution(rootDir, execution)
	}
	execution.ApprovalID = approval.ID
	execution.WriteEnabled = releaseProviderWriteEnabled()
	if !execution.WriteEnabled {
		execution.Status = "blocked"
		execution.Decision = "RELEASE_PROVIDER_PUBLISH_PREVIEW_ONLY"
		execution.Reasons = append(execution.Reasons, "release_provider_write_not_enabled")
		return finishProviderExecution(rootDir, execution)
	}
	consumed, consumedFound, err := approvals.ConsumeApproved(rootDir, approval.ID, approvals.ConsumeOptions{
		TargetType: "release_provider_publish",
		TargetID:   plan.ID,
		Action:     "release.provider.publish",
		ConsumedBy: "release-provider-adapter",
		Reason:     "remote release provider publish",
	})
	if err != nil {
		execution.Status = "blocked"
		execution.Decision = "RELEASE_PROVIDER_PUBLISH_APPROVAL_REQUIRED"
		execution.Reasons = append(execution.Reasons, err.Error())
		return finishProviderExecution(rootDir, execution)
	}
	if !consumedFound {
		execution.Status = "blocked"
		execution.Decision = "RELEASE_PROVIDER_PUBLISH_APPROVAL_REQUIRED"
		execution.Reasons = append(execution.Reasons, "approval_not_found")
		return finishProviderExecution(rootDir, execution)
	}
	execution.ApprovalID = consumed.ID
	execution.ApprovalConsumed = true
	execution.Reasons = append(execution.Reasons, "approval_consumed_before_remote_release_write")
	execution.Status = "blocked"
	execution.Decision = "RELEASE_PROVIDER_PUBLISH_NOT_IMPLEMENTED"
	execution.Reasons = append(execution.Reasons, "remote_release_write_adapter_not_enabled")
	return finishProviderExecution(rootDir, execution)
}

func LoadProviderExecution(rootDir string, id string) (ProviderExecution, bool, error) {
	var execution ProviderExecution
	found, err := fsutil.ReadJSON(providerExecutionPath(rootDir, id), &execution)
	return execution, found, err
}

func finish(rootDir string, plan Plan) (Plan, error) {
	if err := fsutil.WriteJSON(planPath(rootDir, plan.ID), plan); err != nil {
		return Plan{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "plans.jsonl"), plan); err != nil {
		return Plan{}, err
	}
	_ = logging.Log(rootDir, "release", "release.plan.created", map[string]any{"release_id": plan.ID, "decision": plan.Decision, "status": plan.Status, "version": plan.Version})
	return plan, nil
}

func planPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, id+".json")
}

func providerExecutionPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "provider-executions", id+".json")
}

func finishProviderExecution(rootDir string, execution ProviderExecution) (ProviderExecution, bool, error) {
	execution.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := fsutil.EnsureDir(filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "provider-executions")); err != nil {
		return ProviderExecution{}, true, err
	}
	if err := fsutil.WriteJSON(providerExecutionPath(rootDir, execution.ID), execution); err != nil {
		return ProviderExecution{}, true, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "provider-executions.jsonl"), execution); err != nil {
		return ProviderExecution{}, true, err
	}
	_ = logging.Log(rootDir, "release", "release.provider.execution.created", map[string]any{
		"execution_id": execution.ID,
		"release_id":   execution.ReleaseID,
		"mode":         execution.Mode,
		"decision":     execution.Decision,
		"status":       execution.Status,
		"provider":     execution.Provider,
	})
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "release_provider_execution",
		ParentID:    execution.ID,
		SubjectType: "release",
		SubjectID:   execution.ReleaseID,
		Operation:   "release.provider." + execution.Mode,
		Status:      execution.Status,
		Decision:    execution.Decision,
		Reasons:     execution.Reasons,
		Source:      "release_provider",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "provider_execution",
			ID:   execution.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "releases", "provider-executions", execution.ID+".json")),
		}},
	}); err != nil {
		return ProviderExecution{}, true, err
	}
	return execution, true, nil
}

func newProviderExecution(plan Plan, mode string) ProviderExecution {
	now := time.Now().UTC()
	return ProviderExecution{
		ID:        "release-provider-exec-" + textutil.Slugify(plan.ID+"-"+mode) + "-" + strings.ReplaceAll(now.Format("20060102150405.000000000"), ".", ""),
		ReleaseID: plan.ID,
		Version:   plan.Version,
		Provider:  plan.Provider,
		Mode:      mode,
		Status:    "blocked",
		Decision:  "RELEASE_PROVIDER_BLOCKED",
		Reasons:   []string{},
		RemotePlan: RemotePlan{
			Status:    "blocked",
			Decision:  "RELEASE_PROVIDER_PLAN_BLOCKED",
			Actions:   []ProviderAction{},
			CreatedAt: now.Format(time.RFC3339Nano),
		},
		StartedAt: now.Format(time.RFC3339Nano),
	}
}

func releaseProviderWriteEnabled() bool {
	return os.Getenv("MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE") == "1"
}

func releaseReady(plan Plan) bool {
	return plan.Status == "suggested" && plan.Decision == "RELEASE_SUGGESTED"
}

func buildProviderRemotePlan(plan Plan) RemotePlan {
	remotePlan := RemotePlan{
		Status:        "planned",
		Decision:      "RELEASE_PROVIDER_REMOTE_PLAN_READY",
		RemoteName:    plan.RemoteName,
		RemoteURL:     plan.RemoteURL,
		Provider:      plan.Provider,
		ReleaseBranch: plan.ReleaseBranch,
		Version:       plan.Version,
		Actions:       []ProviderAction{},
		CreatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
	}
	remotePlan.Actions = append(remotePlan.Actions,
		ProviderAction{Type: "push_branch", Status: "planned", Command: "git push " + defaultString(plan.RemoteName, "origin") + " " + plan.ReleaseBranch},
		ProviderAction{Type: "create_tag", Status: "planned", Command: "git tag " + plan.Version},
		ProviderAction{Type: "push_tag", Status: "planned", Command: "git push " + defaultString(plan.RemoteName, "origin") + " " + plan.Version},
	)
	releaseEndpoint, ok := releaseEndpoint(plan)
	if ok {
		remotePlan.Actions = append(remotePlan.Actions, ProviderAction{Type: "create_release", Status: "planned", Endpoint: releaseEndpoint})
	} else {
		remotePlan.Actions = append(remotePlan.Actions, ProviderAction{Type: "create_release", Status: "manual_required", Reason: "provider_release_api_unsupported:" + normalize(plan.Provider)})
	}
	workflowEndpoint, ok := workflowEndpoint(plan)
	if ok {
		remotePlan.Actions = append(remotePlan.Actions, ProviderAction{Type: "trigger_workflow", Status: "planned", Endpoint: workflowEndpoint})
	} else {
		remotePlan.Actions = append(remotePlan.Actions, ProviderAction{Type: "trigger_workflow", Status: "manual_required", Reason: "provider_workflow_api_unsupported:" + normalize(plan.Provider)})
	}
	return remotePlan
}

func releaseEndpoint(plan Plan) (string, bool) {
	owner, repo := repoCoordinates(plan.RemoteURL)
	if owner == "" || repo == "" {
		return "", false
	}
	switch normalize(plan.Provider) {
	case "github":
		return "https://api.github.com/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/releases", true
	case "gitee":
		return "https://gitee.com/api/v5/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/releases", true
	default:
		return "", false
	}
}

func workflowEndpoint(plan Plan) (string, bool) {
	owner, repo := repoCoordinates(plan.RemoteURL)
	if owner == "" || repo == "" {
		return "", false
	}
	switch normalize(plan.Provider) {
	case "github":
		return "https://api.github.com/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/actions/workflows/release.yml/dispatches", true
	case "gitee":
		return "https://gitee.com/api/v5/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/actions/workflows/release.yml/dispatches", true
	default:
		return "", false
	}
}

func repoCoordinates(remoteURL string) (string, string) {
	remoteURL = strings.TrimSuffix(strings.TrimSpace(remoteURL), ".git")
	if remoteURL == "" {
		return "", ""
	}
	if strings.HasPrefix(remoteURL, "git@") {
		_, rest, ok := strings.Cut(remoteURL, ":")
		if !ok {
			return "", ""
		}
		parts := strings.Split(strings.Trim(rest, "/"), "/")
		if len(parts) < 2 {
			return "", ""
		}
		return parts[0], strings.TrimSuffix(parts[1], ".git")
	}
	parsed, err := url.Parse(remoteURL)
	if err != nil {
		return "", ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git")
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func writeNotes(rootDir string, plan Plan) (string, error) {
	name := plan.ID + ".md"
	rel := filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "releases", name))
	path := filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, name)
	lines := []string{
		"# Release " + plan.Version,
		"",
		"- Release ID: `" + plan.ID + "`",
		"- Release branch: `" + plan.ReleaseBranch + "`",
		"- Base branch: `" + plan.BaseBranch + "`",
		"- Provider: `" + plan.Provider + "`",
		"",
		"## Included Issues",
	}
	for _, issue := range plan.IncludedIssues {
		lines = append(lines, "- `"+issue.IssueID+"` status="+issue.Status+" quality="+issue.QualityReportID)
	}
	if len(plan.ExcludedIssues) > 0 {
		lines = append(lines, "", "## Excluded Issues")
		for _, issue := range plan.ExcludedIssues {
			lines = append(lines, "- `"+issue.IssueID+"` status="+issue.Status)
		}
	}
	lines = append(lines, "", "## Rollback Plan", "", "- Revert the release branch or tag after review.", "")
	return rel, fsutil.WriteText(path, strings.Join(lines, "\n"))
}

func issueSummaries(rootDir string) ([]IssueSummary, []IssueSummary, error) {
	pattern := filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "issue-states", "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, nil, err
	}
	included := []IssueSummary{}
	excluded := []IssueSummary{}
	for _, path := range matches {
		var state orchestrator.IssueState
		if _, err := fsutil.ReadJSON(path, &state); err != nil {
			return nil, nil, err
		}
		if state.IssueID == "" {
			continue
		}
		summary := IssueSummary{IssueID: state.IssueID, Status: state.Status, QualityReportID: state.QualityReportID}
		if state.Status == "accepted" {
			included = append(included, summary)
		} else {
			excluded = append(excluded, summary)
		}
	}
	sort.Slice(included, func(i, j int) bool { return included[i].IssueID < included[j].IssueID })
	sort.Slice(excluded, func(i, j int) bool { return excluded[i].IssueID < excluded[j].IssueID })
	return included, excluded, nil
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

func normalizeVersion(version string, now time.Time) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "v0.1." + now.Format("20060102150405")
	}
	return strings.Trim(strings.ReplaceAll(version, " ", "-"), "/")
}

func normalize(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.ReplaceAll(value, "-", "_")
}
