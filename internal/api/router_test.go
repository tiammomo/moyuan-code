package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/auth"
	"moyuan-code/internal/controlplane"
	"moyuan-code/internal/fsutil"
	gitadapter "moyuan-code/internal/git"
	"moyuan-code/internal/memory"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/repair"
	"moyuan-code/internal/store"
	"moyuan-code/internal/workspace"
)

func TestGinRouterServesHealthAndProjectsFromGORMStore(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	projectRoot := filepath.Join(root, "managed")
	if err := db.UpsertProject(controlplane.Project{
		ID:           "managed",
		Name:         "managed",
		Root:         projectRoot,
		Source:       map[string]any{"type": "local_path", "provider": "local"},
		OwnerID:      "actor-local-owner",
		Status:       "active",
		RegisteredAt: "2026-05-04T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}

	router := NewRouter(Options{RootDir: root, Store: &db})
	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d body=%s", health.Code, health.Body.String())
	}
	if !jsonContains(health.Body.Bytes(), "phase1-gin-gorm") {
		t.Fatalf("health missing version: %s", health.Body.String())
	}

	projects := httptest.NewRecorder()
	router.ServeHTTP(projects, httptest.NewRequest(http.MethodGet, "/v1/projects", nil))
	if projects.Code != http.StatusOK {
		t.Fatalf("projects status = %d body=%s", projects.Code, projects.Body.String())
	}
	if !jsonContains(projects.Body.Bytes(), "managed") {
		t.Fatalf("projects missing managed project: %s", projects.Body.String())
	}
}

func TestGinRouterServesProjectStateEndpoints(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.InitOwner(root, "managed"); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, root)

	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertProject(controlplane.Project{
		ID:           "managed",
		Name:         "managed",
		Root:         root,
		Source:       map[string]any{"type": "local_path", "provider": "local"},
		OwnerID:      "owner",
		Status:       "active",
		RegisteredAt: "2026-05-04T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := orchestrator.RunIssue(context.Background(), root, "phase1-001", "local_shell", "printf api-state")
	if err != nil {
		t.Fatal(err)
	}
	decision, err := memory.Submit(root, "decision", "Beta API should expose project quality memory for future issue runs", []string{"api", "quality"}, "api-test")
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != "recorded" {
		t.Fatalf("expected memory to record, got %s", decision.Status)
	}
	signal, err := repair.CaptureSignal(root, "test_failure", "sample API repair status", result.RunID)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := repair.Classify(root, signal)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := repair.PlanRepair(root, candidate)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := repair.RunAttempt(context.Background(), root, plan.ID, "local_shell", "printf repaired > api-repair.txt")
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Status != "repaired" {
		t.Fatalf("expected repair attempt to pass, got %s", attempt.Status)
	}

	router := NewRouter(Options{RootDir: root, Store: &db})
	assertGETContains(t, router, "/v1/projects/managed", http.StatusOK, `"project"`, `"managed"`)
	assertGETContains(t, router, "/v1/projects/managed/issues/phase1-001", http.StatusOK, `"issue"`, `"accepted"`)
	assertGETContains(t, router, "/v1/projects/managed/runs/"+result.RunID, http.StatusOK, `"run"`, `"completed"`)
	assertGETContains(t, router, "/v1/projects/managed/quality/"+result.QualityReport.ID, http.StatusOK, `"quality_report"`, `"accepted"`)
	assertGETContains(t, router, "/v1/projects/managed/memory/search?q=Beta&limit=1", http.StatusOK, `"records"`, `Beta API should expose`)
	assertGETContains(t, router, "/v1/projects/managed/memory/candidates?limit=5", http.StatusOK, `"candidates"`, `"recorded"`)
	assertGETContains(t, router, "/v1/projects/managed/repair/attempts/"+attempt.ID, http.StatusOK, `"repair_attempt"`, `"repaired"`)
	assertGETContains(t, router, "/v1/projects/missing", http.StatusNotFound, `"project not found"`)
	assertGETContains(t, router, "/v1/projects/managed/issues/missing", http.StatusNotFound, `"issue state not found"`)
}

func TestGinRouterResolvesProjectsFromControlplaneRegistryWithoutStore(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "managed")
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Ensure(projectRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := controlplane.Register(root, controlplane.Project{
		ID:      "managed",
		Name:    "managed",
		Root:    projectRoot,
		Source:  map[string]any{"type": "local_path", "provider": "local"},
		OwnerID: "owner",
		Status:  "active",
	}); err != nil {
		t.Fatal(err)
	}

	router := NewRouter(Options{RootDir: root})
	assertGETContains(t, router, "/v1/projects/managed", http.StatusOK, `"project"`, `"managed"`)
}

func jsonContains(data []byte, value string) bool {
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return false
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	return strings.Contains(string(encoded), value)
}

func assertGETContains(t *testing.T, router http.Handler, path string, status int, values ...string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != status {
		t.Fatalf("GET %s status = %d body=%s", path, recorder.Code, recorder.Body.String())
	}
	for _, value := range values {
		if !strings.Contains(recorder.Body.String(), value) {
			t.Fatalf("GET %s missing %q in body=%s", path, value, recorder.Body.String())
		}
	}
}

func initGitRepo(t *testing.T, root string) {
	t.Helper()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	if err := fsutil.WriteText(filepath.Join(root, "go.mod"), "module apitest\n\ngo 1.22\n"); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteText(filepath.Join(root, "api_test.go"), "package apitest\n\nimport \"testing\"\n\nfunc TestAPI(t *testing.T) {}\n"); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	if err := gitadapter.BindLocal(root); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}
