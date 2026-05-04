package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/process"
	"moyuan-code/internal/providers"
	"moyuan-code/internal/textutil"
	"moyuan-code/internal/workspace"
)

type nativeSpec struct {
	RuntimeID  string
	Command    string
	Provider   string
	ProviderID string
	ModelID    string
	Args       []string
	Stdin      string
	EnvKeys    []string
}

func runNativeCLI(ctx context.Context, rootDir string, invocation Invocation, command string) (CommandResult, string, string, string, string, error) {
	promptPath, err := writePromptFile(rootDir, invocation)
	if err != nil {
		return CommandResult{}, "", "", "", "", err
	}
	spec := nativeInvocationSpec(invocation, command, promptPath)
	env, err := nativeEnv(rootDir, invocation, &spec)
	if err != nil {
		return CommandResult{}, "", "", "", "", err
	}
	startedAt := time.Now().UTC().Format(time.RFC3339Nano)
	res := process.RunCommandInput(ctx, invocation.WorktreePath, spec.Stdin, env, spec.Command, spec.Args...)
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
		"provider_id": spec.ProviderID,
		"model_id":    spec.ModelID,
		"command":     spec.Command,
		"args":        spec.Args,
		"cwd":         invocation.WorktreePath,
		"env_keys":    spec.EnvKeys,
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
		return CommandResult{}, "", "", "", "", err
	}
	summary := strings.TrimSpace(res.Stdout)
	if summary == "" {
		summary = strings.TrimSpace(res.Stderr)
	}
	return result, summary, spec.ProviderID, spec.ModelID, promptPath, nil
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

func nativeEnv(rootDir string, invocation Invocation, spec *nativeSpec) ([]string, error) {
	env := os.Environ()
	allow := []string{}
	for _, item := range env {
		key := strings.SplitN(item, "=", 2)[0]
		if key == "PATH" || key == "HOME" || key == "SHELL" || key == "TMPDIR" || key == "TEMP" || key == "TMP" {
			allow = append(allow, item)
		}
	}
	provider, ok, err := providers.ResolveRuntimeProvider(rootDir, invocation.RuntimeID, invocation.ProviderID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return allow, nil
	}
	spec.ProviderID = provider.ID
	spec.ModelID = modelIDForInvocation(invocation, provider)
	keys := map[string]bool{}
	add := func(key string, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		allow = setEnvValue(allow, key, value)
		keys[key] = true
	}
	switch normalizedRuntimeID(invocation.RuntimeID) {
	case "claude_cli":
		add("ANTHROPIC_BASE_URL", provider.BaseURL)
		if provider.AuthRef != "" {
			auth, err := resolveAuthRef(provider.AuthRef)
			if err != nil {
				return nil, err
			}
			add("ANTHROPIC_AUTH_TOKEN", auth)
		}
		if spec.ModelID != "" {
			add("ANTHROPIC_MODEL", spec.ModelID)
			add("ANTHROPIC_DEFAULT_SONNET_MODEL", spec.ModelID)
			add("ANTHROPIC_DEFAULT_OPUS_MODEL", spec.ModelID)
			add("ANTHROPIC_DEFAULT_HAIKU_MODEL", spec.ModelID)
		}
		add("API_TIMEOUT_MS", "3000000")
		add("CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC", "1")
	case "codex_cli":
		add("OPENAI_BASE_URL", provider.BaseURL)
		if provider.AuthRef != "" {
			auth, err := resolveAuthRef(provider.AuthRef)
			if err != nil {
				return nil, err
			}
			add("OPENAI_API_KEY", auth)
		}
		if spec.ModelID != "" {
			add("OPENAI_MODEL", spec.ModelID)
		}
	}
	spec.EnvKeys = sortedKeys(keys)
	return allow, nil
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

func modelIDForInvocation(invocation Invocation, provider providers.Provider) string {
	if strings.TrimSpace(invocation.ModelID) != "" {
		return strings.TrimSpace(invocation.ModelID)
	}
	if len(provider.Models) == 0 {
		return ""
	}
	return provider.Models[0].ID
}

func resolveAuthRef(authRef string) (string, error) {
	authRef = strings.TrimSpace(authRef)
	if strings.HasPrefix(authRef, "env:") {
		key := strings.TrimSpace(strings.TrimPrefix(authRef, "env:"))
		if key == "" {
			return "", errors.New("provider_auth_ref_env_required")
		}
		value := os.Getenv(key)
		if value == "" {
			return "", fmt.Errorf("provider_auth_ref_env_missing:%s", key)
		}
		return value, nil
	}
	if strings.HasPrefix(authRef, "secret:") {
		return "", errors.New("provider_secret_ref_not_supported_for_native_env")
	}
	return "", errors.New("provider_auth_ref_unsupported")
}

func setEnvValue(env []string, key string, value string) []string {
	prefix := key + "="
	item := prefix + value
	for i, existing := range env {
		if strings.HasPrefix(existing, prefix) {
			env[i] = item
			return env
		}
	}
	return append(env, item)
}

func sortedKeys(keys map[string]bool) []string {
	out := []string{}
	for key := range keys {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
