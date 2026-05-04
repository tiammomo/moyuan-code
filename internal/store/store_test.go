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
	count, err := db.CountProjects()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("project count = %d", count)
	}
	if !db.DB.Migrator().HasIndex(&Project{}, "SourceType") || !db.DB.Migrator().HasIndex(&Project{}, "Provider") || !db.DB.Migrator().HasIndex(&Project{}, "RegisteredAt") {
		t.Fatal("expected state.db project lookup indexes")
	}
	if projects[0].Status != "archived" || projects[0].Provider != "github" {
		t.Fatalf("unexpected project: %+v", projects[0])
	}

	found, ok, err := db.FindProject("sample")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || found.Root != project.Root {
		t.Fatalf("expected to find sample project, ok=%v project=%+v", ok, found)
	}

	_, ok, err = db.FindProject("missing")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("expected missing project to return ok=false")
	}
}
