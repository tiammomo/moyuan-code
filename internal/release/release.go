package release

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
	"moyuan-code/internal/secrets"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"

	"gopkg.in/yaml.v3"
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
	ReleaseID   string `json:"release_id"`
	CandidateID string `json:"candidate_id,omitempty"`
	Approved    bool   `json:"approved,omitempty"`
	ApprovalID  string `json:"approval_id,omitempty"`
}

type ProviderExecution struct {
	ID               string                 `json:"id"`
	ReleaseID        string                 `json:"release_id"`
	CandidateID      string                 `json:"candidate_id,omitempty"`
	Version          string                 `json:"version,omitempty"`
	Provider         string                 `json:"provider,omitempty"`
	Mode             string                 `json:"mode"`
	Status           string                 `json:"status"`
	Decision         string                 `json:"decision"`
	Reasons          []string               `json:"reasons"`
	RemotePlan       RemotePlan             `json:"remote_plan"`
	RemoteResults    []ProviderActionResult `json:"remote_results,omitempty"`
	ApprovalID       string                 `json:"approval_id,omitempty"`
	ApprovalConsumed bool                   `json:"approval_consumed"`
	WriteEnabled     bool                   `json:"write_enabled"`
	AdapterStatus    string                 `json:"adapter_status,omitempty"`
	StartedAt        string                 `json:"started_at"`
	FinishedAt       string                 `json:"finished_at,omitempty"`
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
	Type          string   `json:"type"`
	Status        string   `json:"status"`
	Command       string   `json:"command,omitempty"`
	Endpoint      string   `json:"endpoint,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	RiskLevel     string   `json:"risk_level,omitempty"`
	ExecutionMode string   `json:"execution_mode,omitempty"`
	Guardrails    []string `json:"guardrails,omitempty"`
}

type ProviderActionResult struct {
	Type       string   `json:"type"`
	Status     string   `json:"status"`
	Decision   string   `json:"decision"`
	Endpoint   string   `json:"endpoint,omitempty"`
	HTTPStatus int      `json:"http_status,omitempty"`
	RemoteID   string   `json:"remote_id,omitempty"`
	RemoteLink string   `json:"remote_link,omitempty"`
	Reason     string   `json:"reason,omitempty"`
	Guardrails []string `json:"guardrails,omitempty"`
}

type releaseProviderAPIConfig struct {
	Provider   string
	APIBaseURL string
	WebBaseURL string
	Owner      string
	Repo       string
	TokenRef   string
}

type releaseRepositoryProviderConfigFile struct {
	Repository struct {
		ProviderConfig struct {
			Owner      string `yaml:"owner"`
			Repo       string `yaml:"repo"`
			APIBaseURL string `yaml:"api_base_url"`
			WebBaseURL string `yaml:"web_base_url"`
			Auth       struct {
				TokenRef string `yaml:"token_ref"`
			} `yaml:"auth"`
		} `yaml:"provider_config"`
	} `yaml:"repository"`
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
	execution.RemotePlan = buildProviderRemotePlan(rootDir, plan)
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
	options.CandidateID = strings.TrimSpace(options.CandidateID)
	options.ApprovalID = strings.TrimSpace(options.ApprovalID)
	plan, found, err := Load(rootDir, options.ReleaseID)
	if err != nil || !found {
		return ProviderExecution{}, found, err
	}
	return providerPublishPlan(rootDir, plan, options)
}

func providerPublishPlan(rootDir string, plan Plan, options ProviderOptions) (ProviderExecution, bool, error) {
	execution := newProviderExecution(plan, "publish")
	execution.CandidateID = strings.TrimSpace(options.CandidateID)
	if !releaseReady(plan) {
		execution.Reasons = append(execution.Reasons, "release_not_suggested:"+plan.Decision)
		return finishProviderExecution(rootDir, execution)
	}
	execution.RemotePlan = buildProviderRemotePlan(rootDir, plan)
	if !options.Approved {
		metadata := map[string]any{
			"release_id": plan.ID,
			"version":    plan.Version,
			"provider":   plan.Provider,
		}
		if execution.CandidateID != "" {
			metadata["candidate_id"] = execution.CandidateID
		}
		approval, err := approvals.Request(rootDir, approvals.RequestOptions{
			TargetType:  "release_provider_publish",
			TargetID:    plan.ID,
			Action:      "release.provider.publish",
			RiskLevel:   "high",
			RequestedBy: "system",
			Reason:      "release provider publish writes branch, tag, release, or workflow state to remote Git provider",
			Metadata:    metadata,
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
	cfg := loadReleaseProviderAPIConfig(rootDir, plan)
	if _, ok := releaseEndpoint(cfg); !ok {
		execution.Status = "blocked"
		execution.Decision = "RELEASE_PROVIDER_PUBLISH_UNSUPPORTED"
		execution.Reasons = append(execution.Reasons, "release_provider_endpoint_unsupported:"+normalize(plan.Provider))
		return finishProviderExecution(rootDir, execution)
	}
	token, err := resolveReleaseProviderToken(rootDir, cfg)
	if err != nil {
		execution.Status = "blocked"
		execution.Decision = "RELEASE_PROVIDER_PUBLISH_AUTH_REQUIRED"
		execution.Reasons = append(execution.Reasons, err.Error())
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
	execution = executeReleaseProviderAdapter(context.Background(), execution, cfg, token)
	return finishProviderExecution(rootDir, execution)
}

func LoadProviderExecution(rootDir string, id string) (ProviderExecution, bool, error) {
	var execution ProviderExecution
	found, err := fsutil.ReadJSON(providerExecutionPath(rootDir, id), &execution)
	return execution, found, err
}

func ListProviderExecutions(rootDir string, limit int) ([]ProviderExecution, error) {
	dir := filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "provider-executions")
	if err := fsutil.EnsureDir(dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	executions := []ProviderExecution{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var execution ProviderExecution
		found, err := fsutil.ReadJSON(filepath.Join(dir, entry.Name()), &execution)
		if err != nil {
			return nil, err
		}
		if found && execution.ID != "" {
			executions = append(executions, execution)
		}
	}
	sort.SliceStable(executions, func(i, j int) bool {
		return executions[i].StartedAt > executions[j].StartedAt
	})
	if limit > 0 && len(executions) > limit {
		return executions[:limit], nil
	}
	return executions, nil
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
		"candidate_id": execution.CandidateID,
		"mode":         execution.Mode,
		"decision":     execution.Decision,
		"status":       execution.Status,
		"provider":     execution.Provider,
	})
	subjectType := "release"
	subjectID := execution.ReleaseID
	if execution.CandidateID != "" {
		subjectType = "release_candidate"
		subjectID = execution.CandidateID
	}
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "release_provider_execution",
		ParentID:    execution.ID,
		SubjectType: subjectType,
		SubjectID:   subjectID,
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

func buildProviderRemotePlan(rootDir string, plan Plan) RemotePlan {
	cfg := loadReleaseProviderAPIConfig(rootDir, plan)
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
		guardedAction("push_branch", "planned", "git_command_preview", "high", "git push "+defaultString(plan.RemoteName, "origin")+" "+plan.ReleaseBranch, ""),
		guardedAction("create_tag", "planned", "local_git_preview", "high", "git tag "+plan.Version, ""),
		guardedAction("push_tag", "planned", "git_command_preview", "high", "git push "+defaultString(plan.RemoteName, "origin")+" "+plan.Version, ""),
	)
	releaseEndpoint, ok := releaseEndpoint(cfg)
	if ok {
		action := guardedAction("create_release", "planned", "provider_api_preview", "high", "", "")
		action.Endpoint = releaseEndpoint
		remotePlan.Actions = append(remotePlan.Actions, action)
	} else {
		remotePlan.Actions = append(remotePlan.Actions, guardedAction("create_release", "manual_required", "manual", "medium", "", "provider_release_api_unsupported:"+normalize(plan.Provider)))
	}
	workflowEndpoint, ok := workflowEndpoint(cfg)
	if ok {
		action := guardedAction("trigger_workflow", "planned", "workflow_dispatch_preview", "high", "", "")
		action.Endpoint = workflowEndpoint
		remotePlan.Actions = append(remotePlan.Actions, action)
	} else {
		remotePlan.Actions = append(remotePlan.Actions, guardedAction("trigger_workflow", "manual_required", "manual", "medium", "", "provider_workflow_api_unsupported:"+normalize(plan.Provider)))
	}
	return remotePlan
}

func guardedAction(actionType string, status string, executionMode string, riskLevel string, command string, reason string) ProviderAction {
	return ProviderAction{
		Type:          actionType,
		Status:        status,
		Command:       command,
		Reason:        reason,
		RiskLevel:     riskLevel,
		ExecutionMode: executionMode,
		Guardrails:    releaseProviderActionGuardrails(actionType),
	}
}

func releaseProviderActionGuardrails(actionType string) []string {
	base := []string{"approval_required", "write_switch_required", "replay_guard_required", "secret_ref_required"}
	switch actionType {
	case "push_branch":
		return append(base, "clean_worktree_required", "release_branch_required")
	case "create_tag":
		return append(base, "tag_version_required", "tag_collision_check_required")
	case "push_tag":
		return append(base, "tag_created_or_existing_check_required")
	case "create_release":
		return append(base, "release_notes_required", "provider_endpoint_required")
	case "trigger_workflow":
		return append(base, "workflow_endpoint_required", "workflow_ref_required")
	default:
		return base
	}
}

func releaseEndpoint(cfg releaseProviderAPIConfig) (string, bool) {
	if cfg.Owner == "" || cfg.Repo == "" || cfg.APIBaseURL == "" {
		return "", false
	}
	switch normalize(cfg.Provider) {
	case "github":
		return strings.TrimRight(cfg.APIBaseURL, "/") + "/repos/" + url.PathEscape(cfg.Owner) + "/" + url.PathEscape(cfg.Repo) + "/releases", true
	case "gitee":
		return strings.TrimRight(cfg.APIBaseURL, "/") + "/repos/" + url.PathEscape(cfg.Owner) + "/" + url.PathEscape(cfg.Repo) + "/releases", true
	default:
		return "", false
	}
}

func workflowEndpoint(cfg releaseProviderAPIConfig) (string, bool) {
	if cfg.Owner == "" || cfg.Repo == "" || cfg.APIBaseURL == "" {
		return "", false
	}
	switch normalize(cfg.Provider) {
	case "github":
		return strings.TrimRight(cfg.APIBaseURL, "/") + "/repos/" + url.PathEscape(cfg.Owner) + "/" + url.PathEscape(cfg.Repo) + "/actions/workflows/release.yml/dispatches", true
	case "gitee":
		return strings.TrimRight(cfg.APIBaseURL, "/") + "/repos/" + url.PathEscape(cfg.Owner) + "/" + url.PathEscape(cfg.Repo) + "/actions/workflows/release.yml/dispatches", true
	default:
		return "", false
	}
}

func loadReleaseProviderAPIConfig(rootDir string, plan Plan) releaseProviderAPIConfig {
	provider := normalize(plan.Provider)
	cfg := releaseProviderAPIConfig{Provider: provider, TokenRef: "secret:git_provider_token"}
	switch provider {
	case "github":
		cfg.APIBaseURL = "https://api.github.com"
		cfg.WebBaseURL = "https://github.com"
	case "gitee":
		cfg.APIBaseURL = "https://gitee.com/api/v5"
		cfg.WebBaseURL = "https://gitee.com"
	}
	if owner, repo := repoCoordinates(plan.RemoteURL); owner != "" && repo != "" {
		cfg.Owner = owner
		cfg.Repo = repo
	}
	raw, found, err := readReleaseRepositoryProviderConfig(rootDir)
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
	}
	return cfg
}

func readReleaseRepositoryProviderConfig(rootDir string) (releaseRepositoryProviderConfigFile, bool, error) {
	text, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(rootDir).MoyuanDir, "repository.yaml"))
	if err != nil || !found {
		return releaseRepositoryProviderConfigFile{}, found, err
	}
	var raw releaseRepositoryProviderConfigFile
	if err := yaml.Unmarshal([]byte(text), &raw); err != nil {
		return releaseRepositoryProviderConfigFile{}, true, err
	}
	return raw, true, nil
}

func resolveReleaseProviderToken(rootDir string, cfg releaseProviderAPIConfig) (string, error) {
	if strings.TrimSpace(cfg.TokenRef) == "" {
		return "", errors.New("release_provider_token_ref_missing")
	}
	if strings.TrimSpace(cfg.Owner) == "" || strings.TrimSpace(cfg.Repo) == "" {
		return "", errors.New("release_provider_repo_coordinates_missing")
	}
	token, err := secrets.Resolve(rootDir, cfg.TokenRef, secrets.ResolveOptions{Purpose: "release.provider.publish", AdapterID: cfg.Provider, Required: true})
	if err != nil {
		return "", err
	}
	if token.Status != "ok" {
		return "", fmt.Errorf("release_provider_token_%s:%s", token.Status, token.Reason)
	}
	return token.Value(), nil
}

func executeReleaseProviderAdapter(ctx context.Context, execution ProviderExecution, cfg releaseProviderAPIConfig, token string) ProviderExecution {
	execution.AdapterStatus = "started"
	results := []ProviderActionResult{}
	for _, action := range execution.RemotePlan.Actions {
		switch action.Type {
		case "push_branch", "create_tag", "push_tag":
			results = append(results, ProviderActionResult{
				Type:       action.Type,
				Status:     "skipped",
				Decision:   "RELEASE_PROVIDER_ACTION_SKIPPED",
				Reason:     "git_command_execution_not_enabled_in_release_provider_adapter",
				Guardrails: append([]string{}, action.Guardrails...),
			})
		case "trigger_workflow":
			results = append(results, ProviderActionResult{
				Type:       action.Type,
				Status:     "skipped",
				Decision:   "RELEASE_PROVIDER_ACTION_SKIPPED",
				Endpoint:   action.Endpoint,
				Reason:     "workflow_dispatch_not_enabled_in_release_provider_adapter",
				Guardrails: append([]string{}, action.Guardrails...),
			})
		case "create_release":
			result := createRemoteRelease(ctx, cfg, execution, token)
			results = append(results, result)
			if result.Status == "failed" {
				execution.Status = "failed"
				execution.Decision = "RELEASE_PROVIDER_PUBLISH_FAILED"
				execution.Reasons = append(execution.Reasons, result.Reason)
				execution.AdapterStatus = "failed"
				execution.RemoteResults = results
				return execution
			}
		default:
			results = append(results, ProviderActionResult{Type: action.Type, Status: "skipped", Decision: "RELEASE_PROVIDER_ACTION_SKIPPED", Reason: "action_unsupported"})
		}
	}
	execution.Status = "completed"
	execution.Decision = "RELEASE_PROVIDER_PUBLISH_COMPLETED"
	execution.Reasons = append(execution.Reasons, "release_provider_adapter_completed")
	execution.AdapterStatus = "completed"
	execution.RemoteResults = results
	return execution
}

func createRemoteRelease(ctx context.Context, cfg releaseProviderAPIConfig, execution ProviderExecution, token string) ProviderActionResult {
	endpoint, ok := releaseEndpoint(cfg)
	if !ok {
		return ProviderActionResult{Type: "create_release", Status: "skipped", Decision: "RELEASE_PROVIDER_ACTION_SKIPPED", Reason: "release_endpoint_unsupported"}
	}
	body := map[string]any{
		"tag_name":   execution.Version,
		"name":       execution.Version,
		"body":       "Release " + execution.Version + " created by Moyuan Code.",
		"draft":      false,
		"prerelease": false,
	}
	if normalize(cfg.Provider) == "gitee" {
		body["access_token"] = token
	}
	result, err := postReleaseProvider(ctx, cfg, endpoint, token, body)
	if err != nil {
		result.Type = "create_release"
		return result
	}
	result.Type = "create_release"
	return result
}

func postReleaseProvider(ctx context.Context, cfg releaseProviderAPIConfig, endpoint string, token string, body map[string]any) (ProviderActionResult, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return ProviderActionResult{Status: "failed", Decision: "RELEASE_PROVIDER_ACTION_FAILED", Endpoint: endpoint, Reason: "release_provider_payload_invalid"}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ProviderActionResult{Status: "failed", Decision: "RELEASE_PROVIDER_ACTION_FAILED", Endpoint: endpoint, Reason: "release_provider_request_invalid"}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "moyuan-code-release-provider/1")
	if normalize(cfg.Provider) == "github" {
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return ProviderActionResult{Status: "failed", Decision: "RELEASE_PROVIDER_ACTION_FAILED", Endpoint: endpoint, Reason: "release_provider_request_failed"}, err
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ProviderActionResult{Status: "failed", Decision: "RELEASE_PROVIDER_ACTION_FAILED", Endpoint: endpoint, HTTPStatus: response.StatusCode, Reason: fmt.Sprintf("release_provider_http_%d", response.StatusCode)}, errors.New("release_provider_http_failed")
	}
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return ProviderActionResult{Status: "failed", Decision: "RELEASE_PROVIDER_ACTION_FAILED", Endpoint: endpoint, HTTPStatus: response.StatusCode, Reason: "release_provider_response_invalid"}, err
	}
	return ProviderActionResult{
		Status:     "completed",
		Decision:   "RELEASE_PROVIDER_ACTION_COMPLETED",
		Endpoint:   endpoint,
		HTTPStatus: response.StatusCode,
		RemoteID:   jsonString(raw, "id", "number"),
		RemoteLink: jsonString(raw, "html_url", "url"),
	}, nil
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
