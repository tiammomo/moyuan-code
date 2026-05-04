package controlplane

import (
	"path/filepath"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

type Project struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Root         string         `json:"root"`
	Source       map[string]any `json:"source"`
	OwnerID      string         `json:"owner_id"`
	Status       string         `json:"status"`
	RegisteredAt string         `json:"registered_at"`
}

type Registry struct {
	SchemaVersion int       `json:"schema_version"`
	Projects      []Project `json:"projects"`
}

func registryPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).MoyuanDir, "projects.json")
}

func Load(rootDir string) (Registry, error) {
	reg := Registry{SchemaVersion: 1, Projects: []Project{}}
	_, err := fsutil.ReadJSON(registryPath(rootDir), &reg)
	if reg.Projects == nil {
		reg.Projects = []Project{}
	}
	return reg, err
}

func Save(rootDir string, reg Registry) error {
	if reg.SchemaVersion == 0 {
		reg.SchemaVersion = 1
	}
	return fsutil.WriteJSON(registryPath(rootDir), reg)
}

func Register(rootDir string, project Project) (Project, error) {
	reg, err := Load(rootDir)
	if err != nil {
		return Project{}, err
	}
	project.RegisteredAt = time.Now().UTC().Format(time.RFC3339Nano)
	next := []Project{project}
	for _, existing := range reg.Projects {
		if existing.Root != project.Root {
			next = append(next, existing)
		}
	}
	reg.Projects = next
	if err := Save(rootDir, reg); err != nil {
		return Project{}, err
	}
	return project, nil
}

func List(rootDir string) ([]Project, error) {
	reg, err := Load(rootDir)
	return reg.Projects, err
}
