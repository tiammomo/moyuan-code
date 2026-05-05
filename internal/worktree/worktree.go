package worktree

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type PrepareOptions struct {
	EpicID      string `json:"epic_id,omitempty"`
	BatchID     string `json:"batch_id,omitempty"`
	IssueID     string `json:"issue_id"`
	BaseRef     string `json:"base_ref,omitempty"`
	RequestedBy string `json:"requested_by,omitempty"`
}

type Record struct {
	ID           string   `json:"id"`
	EpicID       string   `json:"epic_id,omitempty"`
	BatchID      string   `json:"batch_id,omitempty"`
	IssueID      string   `json:"issue_id"`
	Status       string   `json:"status"`
	Decision     string   `json:"decision"`
	Reasons      []string `json:"reasons"`
	RootDir      string   `json:"root_dir"`
	WorktreePath string   `json:"worktree_path,omitempty"`
	Branch       string   `json:"branch,omitempty"`
	BaseRef      string   `json:"base_ref,omitempty"`
	RequestedBy  string   `json:"requested_by,omitempty"`
	CreatedAt    string   `json:"created_at"`
	RemovedAt    string   `json:"removed_at,omitempty"`
}

func Prepare(ctx context.Context, rootDir string, options PrepareOptions) (Record, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Record{}, err
	}
	options = normalizePrepareOptions(options)
	if options.IssueID == "" {
		return Record{}, errors.New("issue_id_required")
	}
	now := time.Now().UTC()
	id := "worktree-" + textutil.Slugify(options.IssueID) + "-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.Nanosecond())
	record := Record{
		ID:          id,
		EpicID:      options.EpicID,
		BatchID:     options.BatchID,
		IssueID:     options.IssueID,
		Status:      "blocked",
		Decision:    "WORKTREE_BLOCKED",
		Reasons:     []string{},
		RootDir:     workspace.ForRoot(rootDir).RootDir,
		RequestedBy: options.RequestedBy,
		CreatedAt:   now.Format(time.RFC3339Nano),
	}
	snapshot := gitadapter.CaptureSnapshot(ctx, rootDir)
	if !snapshot.IsRepo {
		record.Reasons = append(record.Reasons, "not_git_repository")
		return finish(rootDir, record)
	}
	if snapshot.UserDirty {
		record.Reasons = append(record.Reasons, "dirty_user_worktree")
		return finish(rootDir, record)
	}
	baseRef := options.BaseRef
	if baseRef == "" && snapshot.Branch != nil {
		baseRef = *snapshot.Branch
	}
	if baseRef == "" {
		baseRef = "HEAD"
	}
	record.BaseRef = baseRef
	record.Branch = branchName(options, id)
	record.WorktreePath = filepath.Join(worktreesDir(rootDir), id)
	if err := fsutil.EnsureDir(worktreesDir(rootDir)); err != nil {
		return Record{}, err
	}
	result := process.RunCommand(ctx, rootDir, "git", "worktree", "add", "-b", record.Branch, record.WorktreePath, baseRef)
	if result.Code != 0 {
		record.Reasons = append(record.Reasons, "git_worktree_add_failed:"+shortReason(result.Stderr, result.Stdout))
		return finish(rootDir, record)
	}
	record.Status = "ready"
	record.Decision = "WORKTREE_READY"
	record.Reasons = append(record.Reasons, "worktree_created")
	return finish(rootDir, record)
}

func Cleanup(ctx context.Context, rootDir string, id string) (Record, bool, error) {
	record, found, err := Load(rootDir, id)
	if err != nil || !found {
		return record, found, err
	}
	if record.WorktreePath == "" {
		record.Status = "removed"
		record.Decision = "WORKTREE_REMOVED"
		record.Reasons = append(record.Reasons, "worktree_path_empty")
		record.RemovedAt = time.Now().UTC().Format(time.RFC3339Nano)
		updated, err := finish(rootDir, record)
		return updated, true, err
	}
	if exists(record.WorktreePath) {
		result := process.RunCommand(ctx, rootDir, "git", "worktree", "remove", "--force", record.WorktreePath)
		if result.Code != 0 {
			record.Status = "cleanup_failed"
			record.Decision = "WORKTREE_CLEANUP_FAILED"
			record.Reasons = append(record.Reasons, "git_worktree_remove_failed:"+shortReason(result.Stderr, result.Stdout))
			updated, err := finish(rootDir, record)
			return updated, true, err
		}
	}
	record.Status = "removed"
	record.Decision = "WORKTREE_REMOVED"
	record.RemovedAt = time.Now().UTC().Format(time.RFC3339Nano)
	record.Reasons = append(record.Reasons, "worktree_removed")
	updated, err := finish(rootDir, record)
	return updated, true, err
}

func Load(rootDir string, id string) (Record, bool, error) {
	if !validID(id) {
		return Record{}, false, nil
	}
	var record Record
	found, err := fsutil.ReadJSON(filepath.Join(recordsDir(rootDir), id+".json"), &record)
	return record, found, err
}

func List(rootDir string, issueID string, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = 20
	}
	lines, err := fsutil.TailLines(recordsJSONLPath(rootDir), limit*4)
	if err != nil {
		return nil, err
	}
	records := []Record{}
	for _, line := range lines {
		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, err
		}
		if record.ID == "" {
			continue
		}
		if issueID != "" && record.IssueID != issueID {
			continue
		}
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].CreatedAt > records[j].CreatedAt
	})
	if len(records) > limit {
		return records[:limit], nil
	}
	return records, nil
}

func normalizePrepareOptions(options PrepareOptions) PrepareOptions {
	options.EpicID = strings.TrimSpace(options.EpicID)
	options.BatchID = strings.TrimSpace(options.BatchID)
	options.IssueID = strings.TrimSpace(options.IssueID)
	options.BaseRef = strings.TrimSpace(options.BaseRef)
	if options.RequestedBy == "" {
		options.RequestedBy = "system"
	}
	return options
}

func branchName(options PrepareOptions, id string) string {
	issue := textutil.Slugify(options.IssueID)
	epic := textutil.Slugify(options.EpicID)
	if epic == "project" {
		return "moyuan/issue/" + issue + "/" + id
	}
	return "moyuan/" + epic + "/" + issue + "/" + id
}

func finish(rootDir string, record Record) (Record, error) {
	if err := fsutil.EnsureDir(recordsDir(rootDir)); err != nil {
		return Record{}, err
	}
	if err := fsutil.WriteJSON(filepath.Join(recordsDir(rootDir), record.ID+".json"), record); err != nil {
		return Record{}, err
	}
	if err := fsutil.AppendJSONL(recordsJSONLPath(rootDir), record); err != nil {
		return Record{}, err
	}
	_ = logging.Log(rootDir, "git", "worktree.recorded", map[string]any{
		"worktree_id": record.ID,
		"issue_id":    record.IssueID,
		"epic_id":     record.EpicID,
		"decision":    record.Decision,
		"status":      record.Status,
		"branch":      record.Branch,
	})
	return record, nil
}

func recordsDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "worktrees")
}

func recordsJSONLPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).OrchestratorDir, "worktrees.jsonl")
}

func worktreesDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).MoyuanDir, "worktrees")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func shortReason(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		value = strings.ReplaceAll(value, "\n", " ")
		value = strings.ReplaceAll(value, "\r", " ")
		if len(value) > 160 {
			return value[:160]
		}
		return value
	}
	return "unknown"
}

func validID(id string) bool {
	id = strings.TrimSpace(id)
	return id != "" && !strings.Contains(id, "/") && !strings.Contains(id, "\\") && filepath.Base(id) == id
}
