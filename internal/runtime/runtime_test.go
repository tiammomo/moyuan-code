package runtime

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"moyuan-code/internal/providers"
	"moyuan-code/internal/workspace"
)

func TestInvokeRecordsNativeProviderExecutionFeedback(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	writeFakeCodex(t, bin)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	result, err := Invoke(context.Background(), root, Invocation{
		RunID:        "runtime-feedback",
		RuntimeID:    "codex_cli",
		IssueID:      "issue-runtime-feedback",
		Role:         "backend",
		Prompt:       "say ok",
		WorktreePath: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "completed" || result.ProviderID != "codex_cli" {
		t.Fatalf("unexpected runtime result: %+v", result)
	}

	records, err := providers.ListTelemetry(root, "codex_cli", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRuntimeExecutionFeedback(records, result.RunID, "completed") {
		t.Fatalf("expected runtime execution feedback telemetry, got %+v", records)
	}
	provider, ok, err := providers.Show(root, "codex_cli")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || provider.Usage.Requests != 1 || provider.Health.Status != "ok" {
		t.Fatalf("expected provider usage and health feedback, ok=%v provider=%+v", ok, provider)
	}
}

func writeFakeCodex(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "codex")
	script := "#!/bin/sh\ncat >/dev/null\nprintf 'fake codex ok'\n"
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake runtime is not portable to windows")
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func hasRuntimeExecutionFeedback(records []providers.TelemetryRecord, runID string, status string) bool {
	for _, record := range records {
		if record.Source == "runtime_execution" && record.RunID == runID && strings.EqualFold(record.RuntimeStatus, status) {
			return true
		}
	}
	return false
}
