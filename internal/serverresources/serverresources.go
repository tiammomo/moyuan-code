package serverresources

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type Registry struct {
	SchemaVersion int        `json:"schema_version"`
	Resources     []Resource `json:"resources"`
	UpdatedAt     string     `json:"updated_at"`
}

type Resource struct {
	ID                string      `json:"id"`
	Environment       string      `json:"environment"`
	Host              string      `json:"host"`
	Provider          string      `json:"provider"`
	Region            string      `json:"region,omitempty"`
	InstanceID        string      `json:"instance_id,omitempty"`
	Owner             string      `json:"owner"`
	Purpose           string      `json:"purpose,omitempty"`
	AuthRef           string      `json:"auth_ref"`
	ExpiresAt         string      `json:"expires_at,omitempty"`
	Spec              ServerSpec  `json:"spec"`
	Healthcheck       Healthcheck `json:"healthcheck"`
	Status            string      `json:"status"`
	ExpirationState   string      `json:"expiration_state"`
	MaintenanceWindow string      `json:"maintenance_window,omitempty"`
	CreatedAt         string      `json:"created_at"`
	UpdatedAt         string      `json:"updated_at"`
}

type ServerSpec struct {
	CPU      int    `json:"cpu,omitempty"`
	MemoryGB int    `json:"memory_gb,omitempty"`
	DiskGB   int    `json:"disk_gb,omitempty"`
	OS       string `json:"os,omitempty"`
}

type Healthcheck struct {
	Type       string `json:"type,omitempty"`
	Target     string `json:"target,omitempty"`
	LastStatus string `json:"last_status,omitempty"`
}

func List(rootDir string) ([]Resource, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return nil, err
	}
	return registry.Resources, nil
}

func Load(rootDir string) (Registry, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Registry{}, err
	}
	var registry Registry
	found, err := fsutil.ReadJSON(inventoryPath(rootDir), &registry)
	if err != nil {
		return Registry{}, err
	}
	if !found || registry.SchemaVersion == 0 {
		registry = Registry{SchemaVersion: 1, Resources: []Resource{}, UpdatedAt: now()}
		if err := save(rootDir, registry); err != nil {
			return Registry{}, err
		}
	}
	registry.Resources = normalizeOrder(registry.Resources)
	return registry, nil
}

func Add(rootDir string, resource Resource) (Resource, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Resource{}, err
	}
	resource, err = normalizeForSave(resource)
	if err != nil {
		return Resource{}, err
	}
	for _, existing := range registry.Resources {
		if existing.ID == resource.ID {
			return Resource{}, errors.New("resource_id_duplicate")
		}
	}
	resource.CreatedAt = now()
	resource.UpdatedAt = resource.CreatedAt
	resource.Status = "active"
	resource.ExpirationState = expirationState(resource.ExpiresAt, time.Now())
	registry.Resources = append(registry.Resources, resource)
	if err := save(rootDir, registry); err != nil {
		return Resource{}, err
	}
	if err := fsutil.AppendJSONL(eventsPath(rootDir), map[string]any{"event": "resource.added", "resource_id": resource.ID, "environment": resource.Environment, "ts": now()}); err != nil {
		return Resource{}, err
	}
	_ = logging.Log(rootDir, "audit", "server_resource.added", map[string]any{"resource_id": resource.ID, "environment": resource.Environment})
	return resource, nil
}

func Show(rootDir string, id string) (Resource, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Resource{}, false, err
	}
	id = normalizeID(id)
	for _, resource := range registry.Resources {
		if resource.ID == id {
			return resource, true, nil
		}
	}
	return Resource{}, false, nil
}

func Disable(rootDir string, id string) (Resource, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Resource{}, false, err
	}
	id = normalizeID(id)
	for idx, resource := range registry.Resources {
		if resource.ID != id {
			continue
		}
		resource.Status = "disabled"
		resource.UpdatedAt = now()
		registry.Resources[idx] = resource
		if err := save(rootDir, registry); err != nil {
			return Resource{}, false, err
		}
		if err := fsutil.AppendJSONL(eventsPath(rootDir), map[string]any{"event": "resource.disabled", "resource_id": resource.ID, "ts": now()}); err != nil {
			return Resource{}, false, err
		}
		_ = logging.Log(rootDir, "audit", "server_resource.disabled", map[string]any{"resource_id": resource.ID})
		return resource, true, nil
	}
	return Resource{}, false, nil
}

func ExpirationScan(rootDir string) ([]Resource, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return nil, err
	}
	nowTime := time.Now()
	result := []Resource{}
	changed := false
	for idx, resource := range registry.Resources {
		state := expirationState(resource.ExpiresAt, nowTime)
		if resource.ExpirationState != state {
			resource.ExpirationState = state
			resource.UpdatedAt = now()
			registry.Resources[idx] = resource
			changed = true
		}
		if state != "unknown" && state != "ok" {
			result = append(result, resource)
		}
	}
	if changed {
		if err := save(rootDir, registry); err != nil {
			return nil, err
		}
	}
	return normalizeOrder(result), nil
}

func save(rootDir string, registry Registry) error {
	registry.SchemaVersion = 1
	registry.Resources = normalizeOrder(registry.Resources)
	registry.UpdatedAt = now()
	return fsutil.WriteJSON(inventoryPath(rootDir), registry)
}

func normalizeForSave(resource Resource) (Resource, error) {
	resource.ID = normalizeID(resource.ID)
	resource.Environment = normalizeToken(resource.Environment)
	resource.Provider = normalizeToken(resource.Provider)
	resource.AuthRef = strings.TrimSpace(resource.AuthRef)
	resource.Host = strings.TrimSpace(resource.Host)
	resource.Owner = strings.TrimSpace(resource.Owner)
	resource.Region = strings.TrimSpace(resource.Region)
	resource.InstanceID = strings.TrimSpace(resource.InstanceID)
	resource.Purpose = strings.TrimSpace(resource.Purpose)
	resource.MaintenanceWindow = strings.TrimSpace(resource.MaintenanceWindow)
	resource.Healthcheck.Type = normalizeToken(resource.Healthcheck.Type)
	resource.Healthcheck.Target = strings.TrimSpace(resource.Healthcheck.Target)
	resource.Healthcheck.LastStatus = normalizeToken(resource.Healthcheck.LastStatus)
	if resource.ID == "" {
		return Resource{}, errors.New("resource_id_required")
	}
	if resource.Environment == "" {
		return Resource{}, errors.New("environment_required")
	}
	if !allowedEnvironment(resource.Environment) {
		return Resource{}, errors.New("environment_invalid")
	}
	if resource.Host == "" {
		return Resource{}, errors.New("host_required")
	}
	if resource.Provider == "" {
		return Resource{}, errors.New("provider_required")
	}
	if resource.Owner == "" {
		return Resource{}, errors.New("owner_required")
	}
	if resource.AuthRef == "" {
		return Resource{}, errors.New("auth_ref_required")
	}
	if !isSafeRef(resource.AuthRef) {
		return Resource{}, errors.New("auth_ref_must_be_reference")
	}
	if resource.Environment == "production" && resource.ExpiresAt == "" {
		return Resource{}, errors.New("production_expires_at_required")
	}
	if resource.ExpiresAt != "" {
		if _, err := time.Parse("2006-01-02", resource.ExpiresAt); err != nil {
			return Resource{}, errors.New("expires_at_must_be_date")
		}
	}
	if resource.Healthcheck.Type == "" {
		resource.Healthcheck.Type = "manual"
	}
	if resource.Healthcheck.LastStatus == "" {
		resource.Healthcheck.LastStatus = "unknown"
	}
	return resource, nil
}

func inventoryPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ResourcesDir, "inventory.json")
}

func eventsPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).ResourcesDir, "events.jsonl")
}

func normalizeOrder(resources []Resource) []Resource {
	out := append([]Resource{}, resources...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func normalizeID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	return strings.Trim(value, "-")
}

func normalizeToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.ReplaceAll(value, "-", "_")
}

func allowedEnvironment(value string) bool {
	return value == "test_dev" || value == "staging" || value == "production"
}

func isSafeRef(value string) bool {
	return (strings.HasPrefix(value, "env:") && len(value) > len("env:")) || (strings.HasPrefix(value, "secret:") && len(value) > len("secret:"))
}

func expirationState(value string, nowTime time.Time) string {
	if value == "" {
		return "unknown"
	}
	expiresAt, err := time.Parse("2006-01-02", value)
	if err != nil {
		return "unknown"
	}
	days := int(expiresAt.Sub(nowTime).Hours() / 24)
	switch {
	case days < 0:
		return "expired"
	case days <= 7:
		return "critical"
	case days <= 30:
		return "warning"
	default:
		return "ok"
	}
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
