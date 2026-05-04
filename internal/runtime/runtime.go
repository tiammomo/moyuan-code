package runtime

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/process"
	"moyuan-code/internal/workspace"
)

type Invocation struct {
	RunID          string   `json:"run_id"`
	SubagentID     string   `json:"subagent_id,omitempty"`
	ProjectID      string   `json:"project_id"`
	IssueID        string   `json:"issue_id"`
	Role           string   `json:"role"`
	RuntimeID      string   `json:"runtime_id"`
	Mode           string   `json:"mode"`
	WorkspaceRoot  string   `json:"workspace_root"`
	WorktreePath   string   `json:"worktree_path"`
	Branch         string   `json:"branch"`
	Prompt         string   `json:"prompt"`
	ContextFiles   []string `json:"context_files"`
	AllowedPaths   []string `json:"allowed_paths"`
	ProtectedPaths []string `json:"protected_paths"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type Health struct {
	RuntimeID     string  `json:"runtime_id"`
	Command       string  `json:"command"`
	OK            bool    `json:"ok"`
	Version       *string `json:"version,omitempty"`
	LastCheckedAt string  `json:"last_checked_at"`
	Error         *string `json:"error,omitempty"`
}

type CommandResult struct {
	Command  string `json:"command"`
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
}

type Result struct {
	RunID            string          `json:"run_id"`
	SubagentID       string          `json:"subagent_id,omitempty"`
	RuntimeID        string          `json:"runtime_id"`
	Status           string          `json:"status"`
	Summary          string          `json:"summary"`
	ChangedFiles     []string        `json:"changed_files"`
	Commands         []CommandResult `json:"commands"`
	Tests            []CommandResult `json:"tests"`
	Risks            []string        `json:"risks"`
	RuntimeSignals   []string        `json:"runtime_signals"`
	MemoryCandidates []string        `json:"memory_candidates"`
	NativeSessionID  string          `json:"native_session_id,omitempty"`
	CreatedAt        string          `json:"created_at"`
}

func HealthCheck(rootDir string, runtimeID string) Health {
	command := commandFor(runtimeID)
	health := Health{
		RuntimeID:     runtimeID,
		Command:       command,
		OK:            false,
		LastCheckedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if command == "" {
		message := "unknown runtime"
		health.Error = &message
		return health
	}
	path, err := exec.LookPath(command)
	if err != nil {
		message := err.Error()
		health.Error = &message
		return health
	}
	health.OK = true
	health.Version = &path
	_ = fsutil.WriteJSON(filepath.Join(workspace.ForRoot(rootDir).RuntimeDir, runtimeID+"-health.json"), health)
	_ = logging.Log(rootDir, "run", "runtime.health.checked", map[string]any{"runtime_id": runtimeID, "ok": health.OK})
	return health
}

func Invoke(ctx context.Context, rootDir string, invocation Invocation) (Result, error) {
	if invocation.WorktreePath == "" {
		invocation.WorktreePath = rootDir
	}
	if invocation.WorkspaceRoot == "" {
		invocation.WorkspaceRoot = rootDir
	}
	if invocation.TimeoutSeconds == 0 {
		invocation.TimeoutSeconds = 300
	}
	if invocation.RuntimeID == "" {
		invocation.RuntimeID = "local_shell"
	}
	_ = logging.Log(rootDir, "run", "runtime.started", map[string]any{"run_id": invocation.RunID, "runtime_id": invocation.RuntimeID, "issue_id": invocation.IssueID})
	status := "completed"
	summary := "runtime invocation recorded"
	commands := []CommandResult{}
	risks := []string{}
	command := commandFor(invocation.RuntimeID)
	if command == "" {
		status = "failed"
		risks = append(risks, "unknown runtime")
	} else if _, err := exec.LookPath(command); err != nil {
		status = "failed"
		risks = append(risks, "runtime unavailable: "+command)
	} else if invocation.RuntimeID == "local_shell" && strings.TrimSpace(invocation.Prompt) != "" {
		res := process.RunShell(ctx, invocation.WorktreePath, invocation.Prompt)
		cmdStatus := "passed"
		if res.Code != 0 {
			cmdStatus = "failed"
			status = "failed"
		}
		commands = append(commands, CommandResult{Command: invocation.Prompt, Status: cmdStatus, ExitCode: res.Code})
		summary = strings.TrimSpace(res.Stdout)
		if summary == "" {
			summary = strings.TrimSpace(res.Stderr)
		}
	}
	result := Result{
		RunID:            invocation.RunID,
		SubagentID:       invocation.SubagentID,
		RuntimeID:        invocation.RuntimeID,
		Status:           status,
		Summary:          summary,
		ChangedFiles:     []string{},
		Commands:         commands,
		Tests:            []CommandResult{},
		Risks:            risks,
		RuntimeSignals:   []string{},
		MemoryCandidates: []string{},
		CreatedAt:        time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := fsutil.WriteJSON(filepath.Join(workspace.ForRoot(rootDir).RuntimeDir, invocation.RunID+"-"+invocation.RuntimeID+".json"), result); err != nil {
		return Result{}, err
	}
	_ = logging.Log(rootDir, "run", "runtime.completed", map[string]any{"run_id": invocation.RunID, "runtime_id": invocation.RuntimeID, "status": result.Status})
	return result, nil
}

func commandFor(runtimeID string) string {
	switch runtimeID {
	case "claude_cli", "claude":
		return "claude"
	case "codex_cli", "codex":
		return "codex"
	case "local_shell", "shell":
		return "sh"
	default:
		return ""
	}
}
