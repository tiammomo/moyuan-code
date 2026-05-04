package runtime

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type RecoveryRecord struct {
	ID                string   `json:"id"`
	RunID             string   `json:"run_id"`
	SubagentID        string   `json:"subagent_id,omitempty"`
	IssueID           string   `json:"issue_id,omitempty"`
	RuntimeID         string   `json:"runtime_id"`
	ProviderID        string   `json:"provider_id,omitempty"`
	ModelID           string   `json:"model_id,omitempty"`
	NativeSessionID   string   `json:"native_session_id,omitempty"`
	Status            string   `json:"status"`
	FailureCategory   string   `json:"failure_category"`
	FallbackCandidate string   `json:"fallback_candidate,omitempty"`
	FallbackReason    string   `json:"fallback_reason,omitempty"`
	ResumeHint        string   `json:"resume_hint,omitempty"`
	Command           string   `json:"command,omitempty"`
	PromptPath        string   `json:"prompt_path,omitempty"`
	MetadataPath      string   `json:"metadata_path,omitempty"`
	StdoutPath        string   `json:"stdout_path,omitempty"`
	StderrPath        string   `json:"stderr_path,omitempty"`
	DiffSummaryPath   string   `json:"diff_summary_path,omitempty"`
	ChangedFiles      []string `json:"changed_files"`
	Risks             []string `json:"risks"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

type nativeArtifacts struct {
	SessionID    string
	Command      string
	PromptPath   string
	MetadataPath string
	StdoutPath   string
	StderrPath   string
}

func LoadRecovery(rootDir string, id string) (RecoveryRecord, bool, error) {
	var record RecoveryRecord
	found, err := fsutil.ReadJSON(recoveryPath(rootDir, strings.TrimSpace(id)), &record)
	return record, found, err
}

func ListRecoveries(rootDir string, limit int) ([]RecoveryRecord, error) {
	dir := recoveriesDir(rootDir)
	if err := fsutil.EnsureDir(dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	records := []RecoveryRecord{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var record RecoveryRecord
		found, err := fsutil.ReadJSON(filepath.Join(dir, entry.Name()), &record)
		if err != nil {
			return nil, err
		}
		if found && record.ID != "" {
			records = append(records, record)
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].UpdatedAt > records[j].UpdatedAt
	})
	if limit > 0 && len(records) > limit {
		return records[:limit], nil
	}
	return records, nil
}

func recordNativeRecovery(rootDir string, invocation Invocation, result Result, artifacts nativeArtifacts, failureCategory string) (RecoveryRecord, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if failureCategory == "" {
		failureCategory = classifyRecoveryFailure(result.Risks)
	}
	sessionID := artifacts.SessionID
	if sessionID == "" {
		sessionID = nativeSessionID(invocation, invocation.RuntimeID)
	}
	record := RecoveryRecord{
		ID:                recoveryID(invocation.RunID, invocation.RuntimeID),
		RunID:             invocation.RunID,
		SubagentID:        invocation.SubagentID,
		IssueID:           invocation.IssueID,
		RuntimeID:         invocation.RuntimeID,
		ProviderID:        result.ProviderID,
		ModelID:           result.ModelID,
		NativeSessionID:   sessionID,
		Status:            recoveryStatus(failureCategory),
		FailureCategory:   failureCategory,
		FallbackCandidate: fallbackCandidateForRuntime(invocation.RuntimeID),
		FallbackReason:    failureCategory,
		ResumeHint:        resumeHint(failureCategory, sessionID),
		Command:           artifacts.Command,
		PromptPath:        artifacts.PromptPath,
		MetadataPath:      artifacts.MetadataPath,
		StdoutPath:        artifacts.StdoutPath,
		StderrPath:        artifacts.StderrPath,
		DiffSummaryPath:   result.DiffSummaryPath,
		ChangedFiles:      result.ChangedFiles,
		Risks:             result.Risks,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := fsutil.WriteJSON(recoveryPath(rootDir, record.ID), record); err != nil {
		return RecoveryRecord{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(recoveriesDir(rootDir), "events.jsonl"), record); err != nil {
		return RecoveryRecord{}, err
	}
	_ = logging.Log(rootDir, "run", "runtime.recovery.archived", map[string]any{
		"recovery_id":       record.ID,
		"run_id":            record.RunID,
		"runtime_id":        record.RuntimeID,
		"native_session_id": record.NativeSessionID,
		"failure_category":  record.FailureCategory,
		"status":            record.Status,
	})
	return record, nil
}

func recoveryID(runID string, runtimeID string) string {
	slug := textutil.Slugify(runID + "-" + normalizedRuntimeID(runtimeID))
	if slug == "" {
		slug = time.Now().UTC().Format("20060102150405")
	}
	return "recovery-" + slug
}

func nativeSessionID(invocation Invocation, runtimeID string) string {
	slug := textutil.Slugify(invocation.RunID + "-" + normalizedRuntimeID(runtimeID))
	if slug == "" {
		slug = time.Now().UTC().Format("20060102150405")
	}
	return "session-" + slug
}

func nativeSessionDir(rootDir string, sessionID string) string {
	return filepath.Join(workspace.ForRoot(rootDir).RuntimesDir, "sessions", sessionID)
}

func recoveriesDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).RuntimesDir, "recoveries")
}

func recoveryPath(rootDir string, id string) string {
	return filepath.Join(recoveriesDir(rootDir), textutil.Slugify(id)+".json")
}

func classifyRecoveryFailure(risks []string) string {
	for _, risk := range risks {
		switch {
		case strings.HasPrefix(risk, "runtime_unavailable"):
			return "runtime_unavailable"
		case risk == "runtime_failed":
			return "runtime_failed"
		case risk == "pre_existing_dirty_worktree":
			return "pre_existing_dirty_worktree"
		case risk == "protected_paths_changed":
			return "protected_paths_changed"
		case risk == "diff_unavailable":
			return "diff_unavailable"
		}
	}
	return "runtime_failed"
}

func recoveryStatus(failureCategory string) string {
	switch failureCategory {
	case "runtime_failed", "diff_unavailable":
		return "archived"
	default:
		return "blocked"
	}
}

func fallbackCandidateForRuntime(runtimeID string) string {
	switch normalizedRuntimeID(runtimeID) {
	case "claude_cli":
		return "codex_cli"
	case "codex_cli":
		return "claude_cli"
	default:
		return ""
	}
}

func resumeHint(failureCategory string, sessionID string) string {
	if sessionID == "" {
		return "retry_after_runtime_fix"
	}
	switch failureCategory {
	case "runtime_failed", "diff_unavailable":
		return "inspect_session_then_resume_or_retry"
	case "runtime_unavailable":
		return "install_or_auth_runtime_then_retry"
	case "pre_existing_dirty_worktree":
		return "clean_or_commit_user_changes_then_retry"
	case "protected_paths_changed":
		return "inspect_diff_and_revert_protected_paths"
	default:
		return "inspect_session_then_retry"
	}
}
