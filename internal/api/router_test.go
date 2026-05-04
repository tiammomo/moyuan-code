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
	"moyuan-code/internal/issues"
	"moyuan-code/internal/memory"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/repair"
	"moyuan-code/internal/requirement"
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
	if _, _, err := issues.GeneratePhase1(root); err != nil {
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
	reqPlan, err := requirement.PlanFromText(root, "add backend API to inspect issue graph with go test verification")
	if err != nil {
		t.Fatal(err)
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
	assertGETContains(t, router, "/v1/projects/managed/epics/phase1-epic/issue-graph", http.StatusOK, `"issue_graph"`, `"phase1-013"`)
	assertGETContains(t, router, "/v1/projects/managed/epics/phase1-epic/schedule", http.StatusOK, `"schedule"`, `"ready_queue"`, `"blocked_reason"`, `"dispatch_queue"`)
	assertGETContains(t, router, "/v1/projects/managed/issues/phase1-001", http.StatusOK, `"issue"`, `"accepted"`)
	assertGETContains(t, router, "/v1/projects/managed/runs/"+result.RunID, http.StatusOK, `"run"`, `"completed"`)
	assertGETContains(t, router, "/v1/projects/managed/quality/"+result.QualityReport.ID, http.StatusOK, `"quality_report"`, `"accepted"`)
	assertPostContains(t, router, "/v1/projects/managed/issues/phase1-001/merge-decision", `{}`, http.StatusOK, `"merge_decision"`, `"MERGE_ALLOWED"`)
	assertPostContains(t, router, "/v1/projects/managed/issues/missing/merge-decision", `{}`, http.StatusAccepted, `"issue_state_missing"`)
	assertPostContains(t, router, "/v1/projects/managed/issues/phase1-001/git-provider-plan", `{}`, http.StatusAccepted, `"git_provider_plan"`, `"GIT_PROVIDER_BLOCKED"`, `"dirty_worktree"`)
	assertPostContains(t, router, "/v1/projects/managed/releases/suggest", `{"version":"v0.1.0","min_issues":1}`, http.StatusAccepted, `"release"`, `"RELEASE_BLOCKED"`, `"dirty_worktree"`)
	assertPostContains(t, router, "/v1/projects/managed/resources", `{"id":"dev-api","environment":"test_dev","host":"10.0.0.11","provider":"local_vm","owner":"dev-owner","auth_ref":"env:DEV_SERVER_SSH_KEY","expires_at":"2099-01-01"}`, http.StatusCreated, `"resource"`, `"dev-api"`)
	assertGETContains(t, router, "/v1/projects/managed/resources", http.StatusOK, `"resources"`, `"dev-api"`)
	assertGETContains(t, router, "/v1/projects/managed/resources/dev-api", http.StatusOK, `"resource"`, `"test_dev"`)
	assertGETContains(t, router, "/v1/projects/managed/resources/expiration-scan", http.StatusOK, `"resources"`)
	assertPostContains(t, router, "/v1/projects/managed/resources/dev-api/disable", `{}`, http.StatusOK, `"resource"`, `"disabled"`)
	assertPostContains(t, router, "/v1/projects/managed/resources", `{"id":"prod-api","environment":"production","host":"prod.internal","provider":"aliyun","owner":"ops","auth_ref":"secret:prod_ssh_key"}`, http.StatusBadRequest, `"production_expires_at_required"`)
	assertPostContains(t, router, "/v1/projects/managed/deployments/plan", `{"release_id":"missing-release","environment":"test_dev","resource_ids":["dev-api"]}`, http.StatusAccepted, `"deployment"`, `"release_not_found"`)
	assertPostContains(t, router, "/v1/projects/managed/deployments/missing-deployment/execute", `{}`, http.StatusAccepted, `"execution"`, `"deployment_not_found"`)
	assertGETContains(t, router, "/v1/projects/managed/deployment-executions/missing-execution", http.StatusNotFound, `"deployment execution not found"`)
	assertGETContains(t, router, "/v1/projects/managed/requirements/"+reqPlan.ID, http.StatusOK, `"requirement"`, `"clarification_decision"`)
	assertPostContains(t, router, "/v1/projects/managed/requirements/plan", `{"text":"add backend API to inspect requirements with go test verification"}`, http.StatusCreated, `"requirement"`, `"backend-implementation"`)
	assertPostContains(t, router, "/v1/projects/managed/requirements/plan", `{"text":"tune"}`, http.StatusAccepted, `"needs_user_input"`)
	assertGETContains(t, router, "/v1/projects/managed/providers", http.StatusOK, `"providers"`, `"claude_cli"`, `"codex_cli"`)
	assertPostContains(t, router, "/v1/projects/managed/providers", `{"id":"glm-api","vendor":"zhipu","api_type":"openai-compatible","auth_ref":"env:GLM_API_KEY","enabled":true,"data_policy":{"allow_project_memory":true},"models":[{"id":"glm-4"}]}`, http.StatusCreated, `"provider"`, `"glm-api"`)
	assertGETContains(t, router, "/v1/projects/managed/providers/glm-api", http.StatusOK, `"provider"`, `"glm-4"`)
	assertPostContains(t, router, "/v1/projects/managed/provider-route", `{"role":"backend","requires_repo_edit":true}`, http.StatusOK, `"route"`, `"codex_cli"`)
	assertPostContains(t, router, "/v1/projects/managed/provider-route", `{"role":"memory_curator","task_type":"memory_extraction","includes_project_memory":true}`, http.StatusOK, `"route"`, `"glm-api"`)
	assertPostContains(t, router, "/v1/projects/managed/provider-route", `{"role":"backend","requires_repo_edit":true,"includes_secrets":true}`, http.StatusAccepted, `"ROUTE_BLOCKED"`, `"contains_secret_context"`)
	assertPostContains(t, router, "/v1/projects/managed/providers/glm-api/disable", `{}`, http.StatusOK, `"provider"`, `"enabled":false`)
	assertPostContains(t, router, "/v1/projects/managed/providers", `{"id":"bad-api","vendor":"openai","api_type":"openai","auth_ref":"plain-secret-should-not-be-stored","enabled":true}`, http.StatusBadRequest, `"auth_ref_must_be_reference"`)
	assertGETContains(t, router, "/v1/projects/managed/memory/search?q=Beta&limit=1", http.StatusOK, `"records"`, `Beta API should expose`)
	assertGETContains(t, router, "/v1/projects/managed/memory/candidates?limit=5", http.StatusOK, `"candidates"`, `"recorded"`)
	assertGETContains(t, router, "/v1/projects/managed/repair/attempts/"+attempt.ID, http.StatusOK, `"repair_attempt"`, `"repaired"`)
	assertGETContains(t, router, "/v1/projects/missing", http.StatusNotFound, `"project not found"`)
	assertGETContains(t, router, "/v1/projects/managed/epics/missing/issue-graph", http.StatusNotFound, `"issue graph not found"`)
	assertGETContains(t, router, "/v1/projects/managed/epics/missing/schedule", http.StatusNotFound, `"schedule not found"`)
	assertGETContains(t, router, "/v1/projects/managed/issues/missing", http.StatusNotFound, `"issue state not found"`)
	assertGETContains(t, router, "/v1/projects/managed/requirements/missing", http.StatusNotFound, `"requirement plan not found"`)
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

func assertPostContains(t *testing.T, router http.Handler, path string, body string, status int, values ...string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != status {
		t.Fatalf("POST %s status = %d body=%s", path, recorder.Code, recorder.Body.String())
	}
	for _, value := range values {
		if !strings.Contains(recorder.Body.String(), value) {
			t.Fatalf("POST %s missing %q in body=%s", path, value, recorder.Body.String())
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
