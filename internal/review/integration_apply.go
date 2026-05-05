package review

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
	"moyuan-code/internal/logging"
	"moyuan-code/internal/process"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type IntegrationApplyOptions struct {
	PreviewID    string `json:"preview_id"`
	Mode         string `json:"mode,omitempty"`
	Approved     bool   `json:"approved,omitempty"`
	RequestedBy  string `json:"requested_by,omitempty"`
	TargetBranch string `json:"target_branch,omitempty"`
}

type IntegrationApply struct {
	ID           string   `json:"id"`
	PreviewID    string   `json:"preview_id"`
	MergeQueueID string   `json:"merge_queue_id,omitempty"`
	BatchID      string   `json:"batch_id,omitempty"`
	EpicID       string   `json:"epic_id,omitempty"`
	Mode         string   `json:"mode"`
	Status       string   `json:"status"`
	Decision     string   `json:"decision"`
	Reasons      []string `json:"reasons"`
	Approved     bool     `json:"approved"`
	RequestedBy  string   `json:"requested_by,omitempty"`
	SourceBranch string   `json:"source_branch,omitempty"`
	TargetBranch string   `json:"target_branch,omitempty"`
	WriteEnabled bool     `json:"write_enabled"`
	ActionCount  int      `json:"action_count"`
	Actions      []string `json:"actions"`
	StartedAt    string   `json:"started_at"`
	FinishedAt   string   `json:"finished_at,omitempty"`
}

func ApplyIntegrationPreview(ctx context.Context, rootDir string, options IntegrationApplyOptions) (IntegrationApply, error) {
	options = normalizeIntegrationApplyOptions(options)
	if options.PreviewID == "" {
		return IntegrationApply{}, errors.New("preview_id_required")
	}
	now := time.Now().UTC()
	apply := IntegrationApply{
		ID:           "integration-apply-" + textutil.Slugify(options.PreviewID) + "-" + now.Format("20060102150405"),
		PreviewID:    options.PreviewID,
		Mode:         options.Mode,
		Status:       "blocked",
		Decision:     "INTEGRATION_APPLY_BLOCKED",
		Reasons:      []string{},
		Approved:     options.Approved,
		RequestedBy:  options.RequestedBy,
		TargetBranch: options.TargetBranch,
		Actions:      []string{},
		StartedAt:    now.Format(time.RFC3339Nano),
	}
	preview, found, err := LoadIntegrationPreview(rootDir, options.PreviewID)
	if err != nil {
		return IntegrationApply{}, err
	}
	if !found {
		apply.Reasons = append(apply.Reasons, "integration_preview_missing")
		return finishIntegrationApply(rootDir, apply)
	}
	apply.MergeQueueID = preview.MergeQueueID
	apply.BatchID = preview.BatchID
	apply.EpicID = preview.EpicID
	apply.SourceBranch = preview.IntegrationBranch
	if apply.TargetBranch == "" {
		apply.TargetBranch = "moyuan/integration/applied/" + textutil.Slugify(preview.MergeQueueID)
	}
	if preview.Status != "ready" || preview.Decision != "INTEGRATION_PREVIEW_READY" {
		apply.Reasons = append(apply.Reasons, "integration_preview_not_ready:"+preview.Decision)
		return finishIntegrationApply(rootDir, apply)
	}
	if apply.SourceBranch == "" {
		apply.Reasons = append(apply.Reasons, "source_branch_missing")
		return finishIntegrationApply(rootDir, apply)
	}
	apply.Actions = append(apply.Actions, "validate_preview_ready")
	if apply.Mode == "dry_run" {
		apply.Status = "planned"
		apply.Decision = "INTEGRATION_APPLY_DRY_RUN"
		apply.Reasons = append(apply.Reasons, "no_git_ref_updated")
		apply.ActionCount = len(apply.Actions)
		return finishIntegrationApply(rootDir, apply)
	}
	if apply.Mode != "apply" {
		apply.Reasons = append(apply.Reasons, "unsupported_apply_mode:"+apply.Mode)
		return finishIntegrationApply(rootDir, apply)
	}
	if !apply.Approved {
		apply.Reasons = append(apply.Reasons, "integration_apply_approval_required")
		return finishIntegrationApply(rootDir, apply)
	}
	if !integrationApplyEnabled() {
		apply.Reasons = append(apply.Reasons, "integration_apply_not_enabled")
		return finishIntegrationApply(rootDir, apply)
	}
	if !gitRepo(ctx, rootDir) {
		apply.Reasons = append(apply.Reasons, "not_git_repository")
		return finishIntegrationApply(rootDir, apply)
	}
	update := process.RunCommand(ctx, rootDir, "git", "branch", "-f", apply.TargetBranch, apply.SourceBranch)
	apply.Actions = append(apply.Actions, "update_local_integration_branch")
	apply.ActionCount = len(apply.Actions)
	if update.Code != 0 {
		apply.Status = "failed"
		apply.Decision = "INTEGRATION_APPLY_FAILED"
		apply.Reasons = append(apply.Reasons, "git_branch_update_failed:"+shortPreviewReason(update.Stderr, update.Stdout))
		return finishIntegrationApply(rootDir, apply)
	}
	apply.Status = "applied"
	apply.Decision = "INTEGRATION_APPLY_COMPLETED"
	apply.WriteEnabled = true
	apply.Reasons = append(apply.Reasons, "local_integration_branch_updated")
	return finishIntegrationApply(rootDir, apply)
}

func LoadIntegrationApply(rootDir string, id string) (IntegrationApply, bool, error) {
	var apply IntegrationApply
	found, err := fsutil.ReadJSON(integrationApplyPath(rootDir, id), &apply)
	return apply, found, err
}

func ListIntegrationApplies(rootDir string, previewID string, limit int) ([]IntegrationApply, error) {
	if limit <= 0 {
		limit = 20
	}
	lines, err := fsutil.TailLines(integrationAppliesJSONLPath(rootDir), limit*4)
	if err != nil {
		return nil, err
	}
	applies := []IntegrationApply{}
	for _, line := range lines {
		var apply IntegrationApply
		if err := json.Unmarshal([]byte(line), &apply); err != nil {
			return nil, err
		}
		if apply.ID == "" {
			continue
		}
		if previewID != "" && apply.PreviewID != previewID {
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

func normalizeIntegrationApplyOptions(options IntegrationApplyOptions) IntegrationApplyOptions {
	options.PreviewID = strings.TrimSpace(options.PreviewID)
	options.Mode = strings.TrimSpace(strings.ToLower(options.Mode))
	options.TargetBranch = strings.TrimSpace(options.TargetBranch)
	if options.Mode == "" {
		options.Mode = "dry_run"
	}
	if options.RequestedBy == "" {
		options.RequestedBy = "system"
	}
	return options
}

func finishIntegrationApply(rootDir string, apply IntegrationApply) (IntegrationApply, error) {
	apply.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := fsutil.WriteJSON(integrationApplyPath(rootDir, apply.ID), apply); err != nil {
		return IntegrationApply{}, err
	}
	if err := fsutil.AppendJSONL(integrationAppliesJSONLPath(rootDir), apply); err != nil {
		return IntegrationApply{}, err
	}
	_ = logging.Log(rootDir, "git", "review.integration_apply.created", map[string]any{
		"integration_apply_id": apply.ID,
		"preview_id":           apply.PreviewID,
		"decision":             apply.Decision,
		"status":               apply.Status,
		"target_branch":        apply.TargetBranch,
		"write_enabled":        apply.WriteEnabled,
	})
	return apply, nil
}

func integrationApplyEnabled() bool {
	return strings.TrimSpace(strings.ToLower(os.Getenv("MOYUAN_ALLOW_INTEGRATION_APPLY"))) == "1"
}

func integrationApplyPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).MergeReportsDir, "integration-applies", id+".json")
}

func integrationAppliesJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).MergeReportsDir, "integration-applies.jsonl")
}
