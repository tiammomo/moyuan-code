package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase1E2ESmokeCoversLocalAndRemoteProjectLifecycle(t *testing.T) {
	t.Run("local project", func(t *testing.T) {
		root := createTempRepo(t)

		result := runCLI(t, root, "project", "add", "--local", root)
		assertContains(t, result.stdout, "project added:")
		assertFileContains(t, root, ".moyuan/projects.json", root)

		exercisePhase1Lifecycle(t, root)
	})

	t.Run("remote project", func(t *testing.T) {
		source := createTempRepo(t)
		remote := createBareRemote(t, source)
		controlRoot := t.TempDir()
		dest := filepath.Join(t.TempDir(), "managed-remote")

		result := runCLI(t, controlRoot, "project", "add", "--remote", remote, "--dest", dest)
		assertContains(t, result.stdout, "project added:")
		assertFileContains(t, controlRoot, ".moyuan/projects.json", dest)

		list := runCLI(t, controlRoot, "project", "list")
		assertContains(t, list.stdout, dest)

		exercisePhase1Lifecycle(t, dest)

		sync := runCLI(t, dest, "git", "sync", "--comprehend")
		assertContains(t, sync.stdout, "remote")
		assertFileContains(t, dest, ".moyuan/comprehension/events.jsonl", `"mode":"incremental"`)
	})
}

type cliResult struct {
	stdout string
	stderr string
	code   int
}

func exercisePhase1Lifecycle(t *testing.T, root string) {
	t.Helper()

	assertCoreWorkspaceArtifacts(t, root)

	doctor := runCLI(t, root, "workspace", "doctor")
	assertContains(t, doctor.stdout, "project")
	assertContains(t, doctor.stdout, "repository")

	whoami := runCLI(t, root, "auth", "whoami")
	assertContains(t, whoami.stdout, "local_single_user")

	full := runCLI(t, root, "comprehend", "--full")
	assertContains(t, full.stdout, `"mode": "full"`)

	status := runCLI(t, root, "git", "status")
	assertContains(t, status.stdout, `"isRepo": true`)

	branches := runCLI(t, root, "git", "branch", "list")
	if strings.TrimSpace(branches.stdout) == "" {
		t.Fatalf("expected git branch list to return at least one branch")
	}

	graph := runCLI(t, root, "issue", "graph", "phase1-epic")
	assertContains(t, graph.stdout, "phase1-013")

	schedule := runCLI(t, root, "issue", "schedule", "phase1-epic")
	assertContains(t, schedule.stdout, "ready_queue")

	plan := runCLI(t, root, "orchestrator", "plan", "phase1-epic")
	assertContains(t, plan.stdout, "blocked_reason")

	health := runCLI(t, root, "runtime", "health", "local_shell")
	assertContains(t, health.stdout, `"ok": true`)

	runtimeResult := runCLI(t, root, "runtime", "invoke", "local_shell", "--prompt", "printf runtime-ok")
	assertContains(t, runtimeResult.stdout, "runtime-ok")
	assertContains(t, runtimeResult.stdout, "diff_summary_path")
	assertContains(t, runtimeResult.stdout, `"git_before"`)

	qualityResult := runCLI(t, root, "quality", "check", "phase1-001")
	report := decodeQualityReport(t, qualityResult.stdout)
	if report.Status != "passed" {
		t.Fatalf("quality report status = %s\n%s", report.Status, qualityResult.stdout)
	}
	if report.ID == "" {
		t.Fatalf("quality report missing id: %s", qualityResult.stdout)
	}
	if !report.HasCheck("test", "go test ./...") {
		t.Fatalf("quality report missing go test check: %+v", report.Checks)
	}

	reportResult := runCLI(t, root, "quality", "report", report.ID)
	assertContains(t, reportResult.stdout, report.ID)

	orchestrated := runCLI(t, root, "orchestrator", "run", "phase1-001", "--runtime", "local_shell", "--prompt", "printf orchestrator-ok")
	assertContains(t, orchestrated.stdout, "accepted")

	runCLI(t, root, "memory", "add", "--kind", "fact", "--summary", "phase1 memory fact")
	search := runCLI(t, root, "memory", "search", "phase1")
	assertContains(t, search.stdout, "phase1 memory fact")
	compact := runCLI(t, root, "memory", "compact")
	assertContains(t, compact.stdout, "records_seen")

	repair := runCLI(t, root, "repair", "signal", "--type", "test_failure", "--summary", "sample test failure")
	assertContains(t, repair.stdout, "CONFIRMED_BUG")

	logs := runCLI(t, root, "logs", "tail", "--stream", "run", "--limit", "50")
	assertContains(t, logs.stdout, "runtime.completed")

	assertLifecycleArtifacts(t, root)
}

func TestRuntimeDiffCaptureTracksGeneratedFilesAndBlocksDirtyWorktree(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	generated := "generated-by-runtime.txt"
	result := runCLI(t, root, "runtime", "invoke", "local_shell", "--prompt", "printf generated > "+generated)
	assertContains(t, result.stdout, generated)
	assertContains(t, result.stdout, `"new_dirty": true`)
	assertGlob(t, root, ".moyuan/runtime/*-local_shell-diff.md")
	assertFileContains(t, root, ".moyuan/logs/run.jsonl", "diff_summary_path")

	blocked := runCLIAllowFailure(t, root, "runtime", "invoke", "local_shell", "--prompt", "printf should-not-run")
	if blocked.code == 0 {
		t.Fatalf("expected dirty worktree runtime invoke to fail, stdout=%s", blocked.stdout)
	}
	assertContains(t, blocked.stdout, "pre_existing_dirty_worktree")
}

func TestRuntimeDiffCaptureBlocksProtectedPathChanges(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	result := runCLIAllowFailure(t, root, "runtime", "invoke", "local_shell", "--prompt", "printf secret > .env")
	if result.code == 0 {
		t.Fatalf("expected protected path runtime invoke to fail, stdout=%s", result.stdout)
	}
	assertContains(t, result.stdout, `"status": "blocked"`)
	assertContains(t, result.stdout, "protected_paths_changed")
	assertContains(t, result.stdout, ".env")
	assertGlob(t, root, ".moyuan/runtime/*-local_shell-diff.md")
}

type qualityReport struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Checks []struct {
		Type    string  `json:"type"`
		Command *string `json:"command"`
		Status  string  `json:"status"`
	} `json:"checks"`
}

func (r qualityReport) HasCheck(typ string, command string) bool {
	for _, check := range r.Checks {
		if check.Type == typ && check.Command != nil && *check.Command == command && check.Status == "passed" {
			return true
		}
	}
	return false
}

func decodeQualityReport(t *testing.T, raw string) qualityReport {
	t.Helper()
	var report qualityReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("decode quality report: %v\n%s", err, raw)
	}
	return report
}

func runCLI(t *testing.T, root string, args ...string) cliResult {
	t.Helper()
	result := runCLIAllowFailure(t, root, args...)
	if result.code != 0 {
		t.Fatalf("moyuan %s failed: code=%d stdout=%s stderr=%s", strings.Join(args, " "), result.code, result.stdout, result.stderr)
	}
	return result
}

func runCLIAllowFailure(t *testing.T, root string, args ...string) cliResult {
	t.Helper()
	argv := append([]string{}, args...)
	if root != "" {
		argv = append(argv, "--root", root)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), argv, &stdout, &stderr)
	return cliResult{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

func createTempRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run(t, root, "git", "init", "-q")
	goMod := "module example.com/phase1smoke\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	source := "package phase1smoke\n\nfunc Ready() bool { return true }\n"
	if err := os.WriteFile(filepath.Join(root, "smoke.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	testSource := "package phase1smoke\n\nimport \"testing\"\n\nfunc TestReady(t *testing.T) {\n\tif !Ready() {\n\t\tt.Fatal(\"not ready\")\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(root, "smoke_test.go"), []byte(testSource), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "add", "go.mod", "smoke.go", "smoke_test.go")
	run(t, root, "git", "-c", "user.email=test@example.com", "-c", "user.name=test", "commit", "-qm", "init")
	return root
}

func createBareRemote(t *testing.T, source string) string {
	t.Helper()
	remote := filepath.Join(t.TempDir(), "remote.git")
	run(t, "", "git", "clone", "--bare", source, remote)
	return remote
}

func run(t *testing.T, cwd string, command string, args ...string) {
	t.Helper()
	cmd := exec.Command(command, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", command, args, err, string(out))
	}
}

func assertCoreWorkspaceArtifacts(t *testing.T, root string) {
	t.Helper()
	assertFileExists(t, root, ".moyuan/project.yaml")
	assertFileExists(t, root, ".moyuan/repository.yaml")
	assertFileExists(t, root, ".moyuan/policies/access.yaml")
	assertFileExists(t, root, ".moyuan/workspace.json")
	assertFileExists(t, root, ".moyuan/auth/owner.json")
	assertFileExists(t, root, ".moyuan/comprehension/project-profile.md")
	assertFileExists(t, root, ".moyuan/comprehension/module-map.md")
	assertFileExists(t, root, ".moyuan/comprehension/commands.md")
	assertFileExists(t, root, ".moyuan/comprehension/events.jsonl")
	assertFileExists(t, root, ".moyuan/lifecycle/issue-graphs/phase1-epic.json")
	assertFileExists(t, root, ".moyuan/lifecycle/schedules/phase1-epic.json")
	assertFileExists(t, root, ".moyuan/memory/candidates.jsonl")
	assertFileExists(t, root, ".moyuan/logs/run.jsonl")
	assertFileExists(t, root, ".moyuan/logs/audit.jsonl")
}

func assertLifecycleArtifacts(t *testing.T, root string) {
	t.Helper()
	assertGlob(t, root, ".moyuan/runtime/*.json")
	assertGlob(t, root, ".moyuan/lifecycle/runs/*.json")
	assertGlob(t, root, ".moyuan/lifecycle/quality/reports/*.json")
	assertGlob(t, root, ".moyuan/lifecycle/quality/reports/*.md")
	assertGlob(t, root, ".moyuan/orchestrator/*-result.json")
	assertFileExists(t, root, ".moyuan/memory/records.jsonl")
	assertFileExists(t, root, ".moyuan/memory/compact-latest.json")
	assertFileExists(t, root, ".moyuan/repair/signals.jsonl")
	assertFileExists(t, root, ".moyuan/repair/bug-candidates.jsonl")
	assertGlob(t, root, ".moyuan/repair/repair-plan-*.json")
	assertFileExists(t, root, ".moyuan/logs/quality.jsonl")
	assertFileExists(t, root, ".moyuan/logs/memory.jsonl")
}

func assertFileExists(t *testing.T, root string, rel string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file to exist: %s: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("expected file, got directory: %s", path)
	}
}

func assertFileContains(t *testing.T, root string, rel string, needle string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	assertContains(t, string(data), needle)
}

func assertGlob(t *testing.T, root string, pattern string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
	if err != nil {
		t.Fatalf("bad glob %s: %v", pattern, err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected at least one match for %s", filepath.Join(root, filepath.FromSlash(pattern)))
	}
}

func assertContains(t *testing.T, value string, needle string) {
	t.Helper()
	if !strings.Contains(value, needle) {
		t.Fatalf("expected output to contain %q\n%s", needle, value)
	}
}
