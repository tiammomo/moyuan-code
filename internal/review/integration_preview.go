package review

import (
	"context"
	"encoding/json"
	"fmt"
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

type IntegrationPreview struct {
	ID                string                   `json:"id"`
	MergeQueueID      string                   `json:"merge_queue_id"`
	BatchID           string                   `json:"batch_id,omitempty"`
	EpicID            string                   `json:"epic_id,omitempty"`
	Status            string                   `json:"status"`
	Decision          string                   `json:"decision"`
	Reasons           []string                 `json:"reasons"`
	BaseRef           string                   `json:"base_ref,omitempty"`
	IntegrationBranch string                   `json:"integration_branch,omitempty"`
	WorktreePath      string                   `json:"worktree_path,omitempty"`
	ReadyCount        int                      `json:"ready_count"`
	ConflictCount     int                      `json:"conflict_count"`
	BlockedCount      int                      `json:"blocked_count"`
	Items             []IntegrationPreviewItem `json:"items"`
	CreatedAt         string                   `json:"created_at"`
}

type IntegrationPreviewItem struct {
	IssueID         string   `json:"issue_id"`
	Status          string   `json:"status"`
	Decision        string   `json:"decision"`
	Reason          string   `json:"reason,omitempty"`
	Branch          string   `json:"branch,omitempty"`
	WorktreeID      string   `json:"worktree_id,omitempty"`
	SourceWorktree  string   `json:"source_worktree,omitempty"`
	Commit          string   `json:"commit,omitempty"`
	ChangedFiles    []string `json:"changed_files,omitempty"`
	ConflictedFiles []string `json:"conflicted_files,omitempty"`
	ProtectedFiles  []string `json:"protected_files,omitempty"`
}

func BuildIntegrationPreview(ctx context.Context, rootDir string, queueID string) (IntegrationPreview, error) {
	now := time.Now().UTC()
	preview := IntegrationPreview{
		ID:           "integration-preview-" + textutil.Slugify(queueID) + "-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.Nanosecond()),
		MergeQueueID: queueID,
		Status:       "blocked",
		Decision:     "INTEGRATION_PREVIEW_BLOCKED",
		Reasons:      []string{},
		Items:        []IntegrationPreviewItem{},
		CreatedAt:    now.Format(time.RFC3339Nano),
	}
	queue, found, err := LoadMergeQueue(rootDir, queueID)
	if err != nil {
		return IntegrationPreview{}, err
	}
	if !found {
		preview.Reasons = append(preview.Reasons, "merge_queue_missing")
		return finishIntegrationPreview(rootDir, preview)
	}
	preview.BatchID = queue.BatchID
	preview.EpicID = queue.EpicID
	if queue.Status != "ready_to_merge" || queue.Decision != "MERGE_QUEUE_READY" {
		preview.Reasons = append(preview.Reasons, "merge_queue_not_ready:"+queue.Decision)
		return finishIntegrationPreview(rootDir, preview)
	}
	if !gitRepo(ctx, rootDir) {
		preview.Reasons = append(preview.Reasons, "not_git_repository")
		return finishIntegrationPreview(rootDir, preview)
	}
	preview.BaseRef = currentRef(ctx, rootDir)
	preview.IntegrationBranch = "moyuan/integration/" + textutil.Slugify(queueID) + "/" + now.Format("20060102150405")
	preview.WorktreePath = filepath.Join(workspace.ForRoot(rootDir).MoyuanDir, "integration-previews", "worktrees", preview.ID)
	if err := fsutil.EnsureDir(filepath.Dir(preview.WorktreePath)); err != nil {
		return IntegrationPreview{}, err
	}
	add := process.RunCommand(ctx, rootDir, "git", "worktree", "add", "-b", preview.IntegrationBranch, preview.WorktreePath, preview.BaseRef)
	if add.Code != 0 {
		preview.Reasons = append(preview.Reasons, "integration_worktree_failed:"+shortPreviewReason(add.Stderr, add.Stdout))
		return finishIntegrationPreview(rootDir, preview)
	}
	protected := protectedPaths(rootDir)
	for _, queueItem := range queue.Items {
		item := previewMergeQueueItem(ctx, preview.WorktreePath, protected, queueItem)
		preview.Items = append(preview.Items, item)
		switch item.Status {
		case "ready":
			preview.ReadyCount++
		case "conflict":
			preview.ConflictCount++
		default:
			preview.BlockedCount++
		}
	}
	switch {
	case len(preview.Items) == 0:
		preview.Status = "empty"
		preview.Decision = "INTEGRATION_PREVIEW_EMPTY"
		preview.Reasons = append(preview.Reasons, "merge_queue_empty")
	case preview.ConflictCount > 0:
		preview.Status = "conflict"
		preview.Decision = "INTEGRATION_PREVIEW_CONFLICT"
		preview.Reasons = append(preview.Reasons, "merge_conflicts_detected")
	case preview.BlockedCount > 0:
		preview.Status = "blocked"
		preview.Decision = "INTEGRATION_PREVIEW_BLOCKED"
		preview.Reasons = append(preview.Reasons, "items_blocked")
	default:
		preview.Status = "ready"
		preview.Decision = "INTEGRATION_PREVIEW_READY"
		preview.Reasons = append(preview.Reasons, "all_ready_items_merge_cleanly")
	}
	return finishIntegrationPreview(rootDir, preview)
}

func LoadIntegrationPreview(rootDir string, id string) (IntegrationPreview, bool, error) {
	var preview IntegrationPreview
	found, err := fsutil.ReadJSON(integrationPreviewPath(rootDir, id), &preview)
	return preview, found, err
}

func ListIntegrationPreviews(rootDir string, queueID string, limit int) ([]IntegrationPreview, error) {
	if limit <= 0 {
		limit = 20
	}
	lines, err := fsutil.TailLines(integrationPreviewsJSONLPath(rootDir), limit*4)
	if err != nil {
		return nil, err
	}
	previews := []IntegrationPreview{}
	for _, line := range lines {
		var preview IntegrationPreview
		if err := json.Unmarshal([]byte(line), &preview); err != nil {
			return nil, err
		}
		if preview.ID == "" {
			continue
		}
		if queueID != "" && preview.MergeQueueID != queueID {
			continue
		}
		previews = append(previews, preview)
	}
	sort.SliceStable(previews, func(i, j int) bool {
		return previews[i].CreatedAt > previews[j].CreatedAt
	})
	if len(previews) > limit {
		return previews[:limit], nil
	}
	return previews, nil
}

func previewMergeQueueItem(ctx context.Context, previewWorktree string, protected []string, queueItem MergeQueueItem) IntegrationPreviewItem {
	item := IntegrationPreviewItem{
		IssueID:        queueItem.IssueID,
		Status:         "blocked",
		Decision:       "INTEGRATION_ITEM_BLOCKED",
		Branch:         queueItem.Branch,
		WorktreeID:     queueItem.WorktreeID,
		SourceWorktree: queueItem.WorktreePath,
		ChangedFiles:   []string{},
	}
	if queueItem.Status != "ready_to_merge" || queueItem.Decision != "MERGE_QUEUE_ITEM_READY" {
		item.Reason = "merge_queue_item_not_ready:" + queueItem.Decision
		return item
	}
	if strings.TrimSpace(queueItem.Branch) == "" {
		item.Reason = "source_branch_missing"
		return item
	}
	merge := process.RunCommand(ctx, previewWorktree, "git", "merge", "--no-commit", "--no-ff", queueItem.Branch)
	if merge.Code != 0 {
		item.Status = "conflict"
		item.Decision = "INTEGRATION_ITEM_CONFLICT"
		item.Reason = shortPreviewReason(merge.Stderr, merge.Stdout)
		item.ConflictedFiles = conflictedFiles(ctx, previewWorktree)
		_ = process.RunCommand(ctx, previewWorktree, "git", "merge", "--abort")
		return item
	}
	item.ChangedFiles = changedFiles(ctx, previewWorktree)
	item.ProtectedFiles = protectedMatches(item.ChangedFiles, protected)
	if len(item.ProtectedFiles) > 0 {
		item.Status = "blocked"
		item.Decision = "INTEGRATION_ITEM_PROTECTED_PATH_BLOCKED"
		item.Reason = "protected_paths_changed"
		_ = process.RunCommand(ctx, previewWorktree, "git", "merge", "--abort")
		return item
	}
	if worktreeDirty(ctx, previewWorktree) {
		commit := process.RunCommand(ctx, previewWorktree, "git", "commit", "-m", "preview merge "+queueItem.IssueID)
		if commit.Code != 0 {
			item.Status = "blocked"
			item.Decision = "INTEGRATION_ITEM_COMMIT_BLOCKED"
			item.Reason = shortPreviewReason(commit.Stderr, commit.Stdout)
			_ = process.RunCommand(ctx, previewWorktree, "git", "merge", "--abort")
			return item
		}
		item.Commit = currentCommit(ctx, previewWorktree)
	} else {
		item.Reason = "branch_already_integrated"
	}
	item.Status = "ready"
	item.Decision = "INTEGRATION_ITEM_READY"
	if item.Reason == "" {
		item.Reason = "merge_clean"
	}
	return item
}

func finishIntegrationPreview(rootDir string, preview IntegrationPreview) (IntegrationPreview, error) {
	if err := fsutil.WriteJSON(integrationPreviewPath(rootDir, preview.ID), preview); err != nil {
		return IntegrationPreview{}, err
	}
	if err := fsutil.AppendJSONL(integrationPreviewsJSONLPath(rootDir), preview); err != nil {
		return IntegrationPreview{}, err
	}
	_ = logging.Log(rootDir, "git", "review.integration_preview.created", map[string]any{
		"integration_preview_id": preview.ID,
		"merge_queue_id":         preview.MergeQueueID,
		"decision":               preview.Decision,
		"status":                 preview.Status,
		"ready":                  preview.ReadyCount,
		"conflicts":              preview.ConflictCount,
		"blocked":                preview.BlockedCount,
	})
	return preview, nil
}

func gitRepo(ctx context.Context, rootDir string) bool {
	return strings.TrimSpace(process.RunCommand(ctx, rootDir, "git", "rev-parse", "--is-inside-work-tree").Stdout) == "true"
}

func currentRef(ctx context.Context, rootDir string) string {
	branch := strings.TrimSpace(process.RunCommand(ctx, rootDir, "git", "branch", "--show-current").Stdout)
	if branch != "" {
		return branch
	}
	return "HEAD"
}

func currentCommit(ctx context.Context, rootDir string) string {
	return strings.TrimSpace(process.RunCommand(ctx, rootDir, "git", "rev-parse", "HEAD").Stdout)
}

func conflictedFiles(ctx context.Context, rootDir string) []string {
	return splitLines(process.RunCommand(ctx, rootDir, "git", "diff", "--name-only", "--diff-filter=U").Stdout)
}

func changedFiles(ctx context.Context, rootDir string) []string {
	files := append(splitLines(process.RunCommand(ctx, rootDir, "git", "diff", "--name-only", "--cached").Stdout), splitLines(process.RunCommand(ctx, rootDir, "git", "diff", "--name-only").Stdout)...)
	return uniqueStrings(files)
}

func worktreeDirty(ctx context.Context, rootDir string) bool {
	return strings.TrimSpace(process.RunCommand(ctx, rootDir, "git", "status", "--porcelain").Stdout) != ""
}

func protectedPaths(rootDir string) []string {
	ws, err := workspace.Load(rootDir)
	if err != nil {
		return []string{}
	}
	return append([]string{}, ws.Project.Workspace.ProtectedPaths...)
}

func protectedMatches(files []string, protected []string) []string {
	matches := []string{}
	for _, file := range files {
		cleanFile := filepath.Clean(file)
		for _, pattern := range protected {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}
			if ok, _ := filepath.Match(pattern, cleanFile); ok || cleanFile == filepath.Clean(pattern) || strings.HasPrefix(cleanFile, filepath.Clean(pattern)+string(filepath.Separator)) {
				matches = append(matches, file)
				break
			}
		}
	}
	return uniqueStrings(matches)
}

func splitLines(value string) []string {
	items := []string{}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			items = append(items, line)
		}
	}
	return items
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func shortPreviewReason(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		value = strings.ReplaceAll(value, "\n", " ")
		value = strings.ReplaceAll(value, "\r", " ")
		if len(value) > 180 {
			return value[:180]
		}
		return value
	}
	return "unknown"
}

func integrationPreviewPath(rootDir string, id string) string {
	return filepath.Join(workspace.ForRoot(rootDir).MergeReportsDir, "integration-previews", id+".json")
}

func integrationPreviewsJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).MergeReportsDir, "integration-previews.jsonl")
}
