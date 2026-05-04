package release

import (
	"context"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

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
