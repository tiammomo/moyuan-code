package release

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	gitadapter "moyuan-code/internal/git"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/process"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type CandidateApplyOptions struct {
	CandidateID string `json:"candidate_id"`
	Mode        string `json:"mode,omitempty"`
	Approved    bool   `json:"approved,omitempty"`
	RequestedBy string `json:"requested_by,omitempty"`
}

type CandidateApply struct {
	ID             string   `json:"id"`
	CandidateID    string   `json:"candidate_id"`
	ReleaseBatchID string   `json:"release_batch_id,omitempty"`
	Mode           string   `json:"mode"`
	Status         string   `json:"status"`
	Decision       string   `json:"decision"`
	Reasons        []string `json:"reasons"`
	Approved       bool     `json:"approved"`
	RequestedBy    string   `json:"requested_by,omitempty"`
	SourceBranch   string   `json:"source_branch,omitempty"`
	ReleaseBranch  string   `json:"release_branch,omitempty"`
	WriteEnabled   bool     `json:"write_enabled"`
	ActionCount    int      `json:"action_count"`
	Actions        []string `json:"actions"`
	StartedAt      string   `json:"started_at"`
	FinishedAt     string   `json:"finished_at,omitempty"`
}

func ApplyCandidate(ctx context.Context, rootDir string, options CandidateApplyOptions) (CandidateApply, error) {
	options = normalizeCandidateApplyOptions(options)
	if options.CandidateID == "" {
		return CandidateApply{}, errors.New("candidate_id_required")
	}
	now := time.Now().UTC()
	apply := CandidateApply{
		ID:          "release-candidate-apply-" + textutil.Slugify(options.CandidateID) + "-" + now.Format("20060102150405"),
		CandidateID: options.CandidateID,
		Mode:        options.Mode,
		Status:      "blocked",
		Decision:    "RELEASE_BRANCH_APPLY_BLOCKED",
		Reasons:     []string{},
		Approved:    options.Approved,
		RequestedBy: options.RequestedBy,
		Actions:     []string{},
		StartedAt:   now.Format(time.RFC3339Nano),
	}
	candidate, found, err := LoadCandidate(rootDir, options.CandidateID)
	if err != nil {
		return CandidateApply{}, err
	}
	if !found {
		apply.Reasons = append(apply.Reasons, "release_candidate_missing")
		return finishCandidateApply(rootDir, apply)
	}
	apply.ReleaseBatchID = candidate.ReleaseBatchID
	apply.SourceBranch = candidate.SourceBranch
	apply.ReleaseBranch = candidate.ReleaseBranch
	if candidate.Status != "ready" || candidate.Decision != "RELEASE_CANDIDATE_READY" {
		apply.Reasons = append(apply.Reasons, "release_candidate_not_ready:"+candidate.Decision)
		return finishCandidateApply(rootDir, apply)
	}
	if apply.SourceBranch == "" {
		apply.Reasons = append(apply.Reasons, "source_branch_missing")
		return finishCandidateApply(rootDir, apply)
	}
	if apply.ReleaseBranch == "" {
		apply.Reasons = append(apply.Reasons, "release_branch_missing")
		return finishCandidateApply(rootDir, apply)
	}
	apply.Actions = append(apply.Actions, "validate_release_candidate_ready")
	if apply.Mode == "dry_run" {
		apply.Status = "planned"
		apply.Decision = "RELEASE_BRANCH_APPLY_DRY_RUN"
		apply.Reasons = append(apply.Reasons, "no_git_ref_updated")
		apply.ActionCount = len(apply.Actions)
		return finishCandidateApply(rootDir, apply)
	}
	if apply.Mode != "apply" {
		apply.Reasons = append(apply.Reasons, "unsupported_apply_mode:"+apply.Mode)
		return finishCandidateApply(rootDir, apply)
	}
	if !apply.Approved {
		apply.Reasons = append(apply.Reasons, "release_branch_apply_approval_required")
		return finishCandidateApply(rootDir, apply)
	}
	if !releaseBranchApplyEnabled() {
		apply.Reasons = append(apply.Reasons, "release_branch_apply_not_enabled")
		return finishCandidateApply(rootDir, apply)
	}
	if !gitadapter.IsRepo(ctx, rootDir) {
		apply.Reasons = append(apply.Reasons, "not_git_repository")
		return finishCandidateApply(rootDir, apply)
	}
	update := process.RunCommand(ctx, rootDir, "git", "branch", "-f", apply.ReleaseBranch, apply.SourceBranch)
	apply.Actions = append(apply.Actions, "update_local_release_branch")
	apply.ActionCount = len(apply.Actions)
	if update.Code != 0 {
		apply.Status = "failed"
		apply.Decision = "RELEASE_BRANCH_APPLY_FAILED"
		apply.Reasons = append(apply.Reasons, "git_branch_update_failed:"+shortCandidateApplyReason(update.Stderr, update.Stdout))
		return finishCandidateApply(rootDir, apply)
	}
	apply.Status = "applied"
	apply.Decision = "RELEASE_BRANCH_APPLY_COMPLETED"
	apply.WriteEnabled = true
	apply.Reasons = append(apply.Reasons, "local_release_branch_updated")
	return finishCandidateApply(rootDir, apply)
}

func LoadCandidateApply(rootDir string, id string) (CandidateApply, bool, error) {
	var apply CandidateApply
	found, err := fsutil.ReadJSON(candidateApplyPath(rootDir, id), &apply)
	return apply, found, err
}

func ListCandidateApplies(rootDir string, candidateID string, limit int) ([]CandidateApply, error) {
	if limit <= 0 {
		limit = 20
	}
	lines, err := fsutil.TailLines(candidateAppliesJSONLPath(rootDir), limit*4)
	if err != nil {
		return nil, err
	}
	applies := []CandidateApply{}
	for _, line := range lines {
		var apply CandidateApply
		if err := json.Unmarshal([]byte(line), &apply); err != nil {
			return nil, err
		}
		if apply.ID == "" {
			continue
		}
		if candidateID != "" && apply.CandidateID != candidateID {
			continue
		}
		applies = append(applies, apply)
	}
	sort.SliceStable(applies, func(i, j int) bool {
		return applies[i].StartedAt > applies[j].StartedAt
	})
	if len(applies) > limit {
		return applies[:limit], nil
	}
	return applies, nil
}

func normalizeCandidateApplyOptions(options CandidateApplyOptions) CandidateApplyOptions {
	options.CandidateID = strings.TrimSpace(options.CandidateID)
	options.Mode = strings.TrimSpace(strings.ToLower(options.Mode))
	if options.Mode == "" {
		options.Mode = "dry_run"
	}
	if options.RequestedBy == "" {
		options.RequestedBy = "system"
	}
	return options
}

func finishCandidateApply(rootDir string, apply CandidateApply) (CandidateApply, error) {
	apply.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := fsutil.WriteJSON(candidateApplyPath(rootDir, apply.ID), apply); err != nil {
		return CandidateApply{}, err
	}
	if err := fsutil.AppendJSONL(candidateAppliesJSONLPath(rootDir), apply); err != nil {
		return CandidateApply{}, err
	}
	_ = logging.Log(rootDir, "release", "release.candidate_apply.created", map[string]any{
		"release_candidate_apply_id": apply.ID,
		"release_candidate_id":       apply.CandidateID,
		"decision":                   apply.Decision,
		"status":                     apply.Status,
		"release_branch":             apply.ReleaseBranch,
		"write_enabled":              apply.WriteEnabled,
	})
	return apply, nil
}

func releaseBranchApplyEnabled() bool {
	return strings.TrimSpace(strings.ToLower(os.Getenv("MOYUAN_ALLOW_RELEASE_BRANCH_APPLY"))) == "1"
}

func shortCandidateApplyReason(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) > 18 {
			fields = fields[:18]
		}
		return strings.Join(fields, " ")
	}
	return "unknown"
}

func candidateApplyPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "candidate-applies", id+".json")
}

func candidateAppliesJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ReleasesDir, "release-candidate-applies.jsonl")
}
