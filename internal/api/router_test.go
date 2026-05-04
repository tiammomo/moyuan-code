package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/controlplane"
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
