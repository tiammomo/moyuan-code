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

func TestLocalProjectAddCreatesWorkspaceOwnerComprehensionGraphAndQualityReport(t *testing.T) {
	root := createTempRepo(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"project", "add", "--local", root, "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project add failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "project added:") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"auth", "whoami", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("whoami failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "local_single_user") {
		t.Fatalf("whoami missing local_single_user: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"issue", "graph", "phase1-epic", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("issue graph failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "phase1-001") {
		t.Fatalf("issue graph missing phase1-001: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"quality", "check", "phase1-001", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("quality check failed: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	var report struct {
		Status string `json:"status"`
		Checks []struct {
			Type    string  `json:"type"`
			Command *string `json:"command"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode quality report: %v\n%s", err, stdout.String())
	}
	if report.Status != "passed" {
		t.Fatalf("quality report status = %s", report.Status)
	}
	foundTest := false
	for _, check := range report.Checks {
		if check.Type == "test" && check.Command != nil && *check.Command == "npm test" {
			foundTest = true
		}
	}
	if !foundTest {
		t.Fatalf("quality report missing npm test check: %+v", report.Checks)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"orchestrator", "plan", "phase1-epic", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("orchestrator plan failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "ready_queue") {
		t.Fatalf("orchestrator plan missing ready_queue: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"runtime", "invoke", "local_shell", "--prompt", "printf runtime-ok", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runtime invoke failed: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "runtime-ok") {
		t.Fatalf("runtime output missing: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"orchestrator", "run", "phase1-001", "--runtime", "local_shell", "--prompt", "printf orchestrator-ok", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("orchestrator run failed: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "accepted") {
		t.Fatalf("orchestrator run not accepted: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"memory", "add", "--kind", "fact", "--summary", "phase1 memory fact", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("memory add failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"memory", "search", "phase1", "--root", root}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "phase1 memory fact") {
		t.Fatalf("memory search failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"memory", "compact", "--root", root}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "records_seen") {
		t.Fatalf("memory compact failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"repair", "signal", "--type", "test_failure", "--summary", "sample test failure", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("repair signal failed: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "CONFIRMED_BUG") {
		t.Fatalf("repair classification missing: %s", stdout.String())
	}
}

func createTempRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run(t, root, "git", "init", "-q")
	packageJSON := `{"type":"module","scripts":{"test":"node --test"}}` + "\n"
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "add", "package.json")
	run(t, root, "git", "-c", "user.email=test@example.com", "-c", "user.name=test", "commit", "-qm", "init")
	return root
}

func run(t *testing.T, cwd string, command string, args ...string) {
	t.Helper()
	cmd := exec.Command(command, args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", command, args, err, string(out))
	}
}
