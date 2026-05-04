package store

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"moyuan-code/internal/controlplane"
	"moyuan-code/internal/workspace"
)

type Store struct {
	DB *gorm.DB
}

type Project struct {
	ID           string `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"not null" json:"name"`
	Root         string `gorm:"uniqueIndex;not null" json:"root"`
	SourceType   string `json:"source_type"`
	Provider     string `json:"provider"`
	OwnerID      string `json:"owner_id"`
	Status       string `gorm:"index" json:"status"`
	RegisteredAt string `json:"registered_at"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func Open(rootDir string) (Store, error) {
	db, err := gorm.Open(sqlite.Open(DefaultPath(rootDir)), &gorm.Config{})
	if err != nil {
		return Store{}, err
	}
	store := Store{DB: db}
	if err := store.Migrate(); err != nil {
		return Store{}, err
	}
	return store, nil
}

func DefaultPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).MoyuanDir, "state.db")
}

func (s Store) Migrate() error {
	return s.DB.AutoMigrate(&Project{})
}

func (s Store) UpsertProject(project controlplane.Project) error {
	model := Project{
		ID:           project.ID,
		Name:         project.Name,
		Root:         project.Root,
		SourceType:   sourceString(project.Source, "type"),
		Provider:     sourceString(project.Source, "provider"),
		OwnerID:      project.OwnerID,
		Status:       project.Status,
		RegisteredAt: project.RegisteredAt,
	}
	return s.DB.Save(&model).Error
}

func (s Store) ListProjects() ([]Project, error) {
	var projects []Project
	if err := s.DB.Order("registered_at desc, created_at desc").Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (s Store) FindProject(id string) (Project, bool, error) {
	var project Project
	err := s.DB.Where("id = ?", id).First(&project).Error
	if err == nil {
		return project, true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Project{}, false, nil
	}
	return Project{}, false, err
}

func (s Store) Close() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func sourceString(source map[string]any, key string) string {
	if source == nil {
		return ""
	}
	value, _ := source[key].(string)
	return value
}
