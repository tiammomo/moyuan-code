package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/workspace"
)

func TestPrepareCreatesIssueWorktreeAndRecord(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, root)

	record, err := Prepare(context.Background(), root, PrepareOptions{
		EpicID:      "phase11-worktree",
		BatchID:     "batch-test",
		IssueID:     "backend-api",
		RequestedBy: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Decision != "WORKTREE_READY" || record.WorktreePath == "" || record.Branch == "" {
		t.Fatalf("expected ready worktree, got %+v", record)
	}
	if _, err := os.Stat(record.WorktreePath); err != nil {
		t.Fatalf("expected worktree path to exist: %v", err)
	}
	if branch := gitOutput(t, record.WorktreePath, "branch", "--show-current"); strings.TrimSpace(branch) != record.Branch {
		t.Fatalf("expected worktree branch %q, got %q", record.Branch, branch)
	}
	loaded, found, err := Load(root, record.ID)
	if err != nil || !found || loaded.ID != record.ID {
		t.Fatalf("expected persisted record, found=%v loaded=%+v err=%v", found, loaded, err)
	}
	records, err := List(root, "backend-api", 10)
	if err != nil || len(records) != 1 || records[0].ID != record.ID {
		t.Fatalf("expected listed record, records=%+v err=%v", records, err)
	}

	removed, found, err := Cleanup(context.Background(), root, record.ID)
	if err != nil || !found {
		t.Fatalf("expected cleanup, found=%v err=%v", found, err)
	}
	if removed.Decision != "WORKTREE_REMOVED" {
		t.Fatalf("expected removed record, got %+v", removed)
	}
	if _, err := os.Stat(record.WorktreePath); !os.IsNotExist(err) {
		t.Fatalf("expected worktree path removed, err=%v", err)
	}
}

func TestPrepareBlocksDirtyUserWorktree(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, root)
	if err := os.WriteFile(filepath.Join(root, "user-change.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Prepare(context.Background(), root, PrepareOptions{IssueID: "backend-api"})
	if err != nil {
		t.Fatal(err)
	}
	if record.Decision != "WORKTREE_BLOCKED" || !hasReason(record.Reasons, "dirty_user_worktree") {
		t.Fatalf("expected dirty user worktree block, got %+v", record)
	}
}

func initGitRepo(t *testing.T, root string) {
	t.Helper()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# worktree test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
}

func gitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return string(output)
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	_ = gitOutput(t, root, args...)
}

func hasReason(reasons []string, target string) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}
