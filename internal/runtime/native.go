package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/process"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type nativeSpec struct {
	RuntimeID string
	Command   string
	Provider  string
	Args      []string
	Stdin     string
}

func runNativeCLI(ctx context.Context, rootDir string, invocation Invocation, command string) (CommandResult, string, string, error) {
	promptPath, err := writePromptFile(rootDir, invocation)
	if err != nil {
		return CommandResult{}, "", "", err
	}
	spec := nativeInvocationSpec(invocation, command, promptPath)
	startedAt := time.Now().UTC().Format(time.RFC3339Nano)
	res := process.RunCommandInput(ctx, invocation.WorktreePath, spec.Stdin, nativeEnv(), spec.Command, spec.Args...)
	status := "passed"
	if res.Code != 0 {
		status = "failed"
	}
	commandLine := strings.TrimSpace(spec.Command + " " + strings.Join(spec.Args, " "))
	result := CommandResult{
		Command:  commandLine,
		Status:   status,
		ExitCode: res.Code,
	}
	metadata := map[string]any{
		"runtime_id":  invocation.RuntimeID,
		"provider":    spec.Provider,
		"command":     spec.Command,
		"args":        spec.Args,
		"cwd":         invocation.WorktreePath,
		"prompt_path": promptPath,
		"started_at":  startedAt,
		"finished_at": time.Now().UTC().Format(time.RFC3339Nano),
		"exit_code":   res.Code,
		"status":      status,
		"stdout":      res.Stdout,
		"stderr":      res.Stderr,
	}
	metadataPath := nativeMetadataPath(rootDir, invocation, spec)
	if err := fsutil.WriteJSON(metadataPath, metadata); err != nil {
		return CommandResult{}, "", "", err
	}
	summary := strings.TrimSpace(res.Stdout)
	if summary == "" {
		summary = strings.TrimSpace(res.Stderr)
	}
	return result, summary, promptPath, nil
}

func writePromptFile(rootDir string, invocation Invocation) (string, error) {
	dir := filepath.Join(workspace.ForRoot(rootDir).RuntimeDir, "prompts")
	name := textutil.Slugify(invocation.RunID + "-" + invocation.RuntimeID)
	if name == "" {
		name = "runtime-prompt"
	}
	path := filepath.Join(dir, name+".md")
	lines := []string{
		"# Moyuan Runtime Prompt",
		"",
		"- Run ID: `" + invocation.RunID + "`",
		"- Issue ID: `" + invocation.IssueID + "`",
		"- Role: `" + invocation.Role + "`",
		"- Runtime ID: `" + invocation.RuntimeID + "`",
		"- Mode: `" + invocation.Mode + "`",
		"- Worktree: `" + invocation.WorktreePath + "`",
		"",
		"## Task",
		"",
		strings.TrimSpace(invocation.Prompt),
		"",
		"## Constraints",
		"",
		"- Do not push, tag, deploy, or merge branches.",
		"- Keep all code changes inside the assigned worktree and allowed scope.",
		"- Do not write protected paths.",
		"- Leave changes for Moyuan quality gates and review.",
	}
	if len(invocation.AllowedPaths) > 0 {
		lines = append(lines, "", "## Allowed Paths")
		for _, item := range invocation.AllowedPaths {
			lines = append(lines, "- `"+item+"`")
		}
	}
	if len(invocation.ProtectedPaths) > 0 {
		lines = append(lines, "", "## Protected Paths")
		for _, item := range invocation.ProtectedPaths {
			lines = append(lines, "- `"+item+"`")
		}
	}
	return path, fsutil.WriteText(path, strings.Join(lines, "\n")+"\n")
}

func nativeInvocationSpec(invocation Invocation, command string, promptPath string) nativeSpec {
	promptText := strings.TrimSpace(invocation.Prompt)
	switch normalizedRuntimeID(invocation.RuntimeID) {
	case "claude_cli":
		return nativeSpec{
			RuntimeID: invocation.RuntimeID,
			Command:   command,
			Provider:  "claude_code",
			Args:      []string{"-p", promptText},
		}
	case "codex_cli":
		return nativeSpec{
			RuntimeID: invocation.RuntimeID,
			Command:   command,
			Provider:  "codex",
			Args:      []string{"exec", "--skip-git-repo-check", "--cd", invocation.WorktreePath, "-"},
			Stdin:     promptText,
		}
	default:
		return nativeSpec{
			RuntimeID: invocation.RuntimeID,
			Command:   command,
			Provider:  "unknown",
			Args:      []string{promptPath},
		}
	}
}

func nativeMetadataPath(rootDir string, invocation Invocation, spec nativeSpec) string {
	name := textutil.Slugify(fmt.Sprintf("%s-%s-native", invocation.RunID, spec.RuntimeID))
	if name == "" {
		name = "runtime-native"
	}
	return filepath.Join(workspace.ForRoot(rootDir).RuntimeDir, name+".json")
}

func nativeEnv() []string {
	env := os.Environ()
	allow := []string{}
	for _, item := range env {
		key := strings.SplitN(item, "=", 2)[0]
		if key == "PATH" || key == "HOME" || key == "SHELL" || key == "TMPDIR" || key == "TEMP" || key == "TMP" {
			allow = append(allow, item)
		}
	}
	return allow
}

func normalizedRuntimeID(runtimeID string) string {
	switch runtimeID {
	case "claude":
		return "claude_cli"
	case "codex":
		return "codex_cli"
	default:
		return runtimeID
	}
}
