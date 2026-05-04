package store

import (
	"path/filepath"
	"testing"

	"moyuan-code/internal/controlplane"
	"moyuan-code/internal/workspace"
)

func TestGORMStoreMigratesAndUpsertsProjects(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	project := controlplane.Project{
		ID:           "sample",
		Name:         "sample",
		Root:         filepath.Join(root, "sample"),
		Source:       map[string]any{"type": "remote_git", "provider": "github"},
		OwnerID:      "owner",
		Status:       "active",
		RegisteredAt: "2026-05-04T00:00:00Z",
	}
	if err := db.UpsertProject(project); err != nil {
		t.Fatal(err)
	}
	project.Status = "archived"
	if err := db.UpsertProject(project); err != nil {
		t.Fatal(err)
	}

	projects, err := db.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 {
		t.Fatalf("projects length = %d", len(projects))
	}
	if projects[0].Status != "archived" || projects[0].Provider != "github" {
		t.Fatalf("unexpected project: %+v", projects[0])
	}
}
