package skills

import (
	"errors"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type BindingRegistry struct {
	SchemaVersion int       `json:"schema_version"`
	Bindings      []Binding `json:"bindings"`
	UpdatedAt     string    `json:"updated_at"`
}

type Binding struct {
	ID         string            `json:"id"`
	SkillID    string            `json:"skill_id"`
	TargetType string            `json:"target_type"`
	TargetID   string            `json:"target_id"`
	Priority   int               `json:"priority"`
	Status     string            `json:"status"`
	Config     map[string]string `json:"config,omitempty"`
	CreatedAt  string            `json:"created_at"`
	UpdatedAt  string            `json:"updated_at"`
}

func ListBindings(rootDir string) ([]Binding, error) {
	registry, err := loadBindingRegistry(rootDir)
	if err != nil {
		return nil, err
	}
	return registry.Bindings, nil
}

func UpsertBinding(rootDir string, binding Binding) (Binding, error) {
	registry, err := loadBindingRegistry(rootDir)
	if err != nil {
		return Binding{}, err
	}
	binding, err = normalizeBindingForSave(rootDir, binding)
	if err != nil {
		return Binding{}, err
	}
	existingIndex := -1
	for idx, existing := range registry.Bindings {
		if existing.ID != binding.ID {
			continue
		}
		existingIndex = idx
		if binding.CreatedAt == "" {
			binding.CreatedAt = existing.CreatedAt
		}
		break
	}
	if binding.CreatedAt == "" {
		binding.CreatedAt = now()
	}
	binding.UpdatedAt = now()
	if existingIndex >= 0 {
		registry.Bindings[existingIndex] = binding
	} else {
		registry.Bindings = append(registry.Bindings, binding)
	}
	if err := saveBindingRegistry(rootDir, registry); err != nil {
		return Binding{}, err
	}
	if err := fsutil.AppendJSONL(bindingEventsPath(rootDir), map[string]any{"event": "skill.binding.upserted", "binding_id": binding.ID, "skill_id": binding.SkillID, "target_type": binding.TargetType, "target_id": binding.TargetID, "ts": now()}); err != nil {
		return Binding{}, err
	}
	_ = logging.Log(rootDir, "audit", "skill.binding.upserted", map[string]any{"binding_id": binding.ID, "skill_id": binding.SkillID, "target_type": binding.TargetType, "target_id": binding.TargetID})
	return binding, nil
}

func DisableBinding(rootDir string, id string) (Binding, bool, error) {
	registry, err := loadBindingRegistry(rootDir)
	if err != nil {
		return Binding{}, false, err
	}
	id = normalizeID(id)
	for idx, binding := range registry.Bindings {
		if binding.ID != id {
			continue
		}
		binding.Status = "disabled"
		binding.UpdatedAt = now()
		registry.Bindings[idx] = binding
		if err := saveBindingRegistry(rootDir, registry); err != nil {
			return Binding{}, false, err
		}
		if err := fsutil.AppendJSONL(bindingEventsPath(rootDir), map[string]any{"event": "skill.binding.disabled", "binding_id": binding.ID, "ts": now()}); err != nil {
			return Binding{}, false, err
		}
		_ = logging.Log(rootDir, "audit", "skill.binding.disabled", map[string]any{"binding_id": binding.ID, "skill_id": binding.SkillID})
		return binding, true, nil
	}
	return Binding{}, false, nil
}

func loadBindingRegistry(rootDir string) (BindingRegistry, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return BindingRegistry{}, err
	}
	var registry BindingRegistry
	found, err := fsutil.ReadJSON(bindingsPath(rootDir), &registry)
	if err != nil {
		return BindingRegistry{}, err
	}
	if !found || registry.SchemaVersion == 0 {
		registry = BindingRegistry{SchemaVersion: registryVersion, Bindings: []Binding{}, UpdatedAt: now()}
		if err := saveBindingRegistry(rootDir, registry); err != nil {
			return BindingRegistry{}, err
		}
	}
	registry.Bindings = normalizeBindingOrder(registry.Bindings)
	return registry, nil
}

func saveBindingRegistry(rootDir string, registry BindingRegistry) error {
	registry.SchemaVersion = registryVersion
	registry.Bindings = normalizeBindingOrder(registry.Bindings)
	registry.UpdatedAt = now()
	return fsutil.WriteJSON(bindingsPath(rootDir), registry)
}

func normalizeBindingForSave(rootDir string, binding Binding) (Binding, error) {
	binding.SkillID = normalizeID(binding.SkillID)
	binding.TargetType = normalizeToken(binding.TargetType)
	binding.TargetID = strings.TrimSpace(binding.TargetID)
	binding.Status = normalizeToken(binding.Status)
	if binding.Config == nil {
		binding.Config = map[string]string{}
	}
	for key, value := range binding.Config {
		if containsPlainSecret(key, value) {
			return Binding{}, errors.New("binding_config_must_not_contain_secret")
		}
	}
	if binding.SkillID == "" {
		return Binding{}, errors.New("skill_id_required")
	}
	skill, found, err := Show(rootDir, binding.SkillID)
	if err != nil {
		return Binding{}, err
	}
	if !found {
		return Binding{}, errors.New("skill_not_found")
	}
	if !skill.Enabled {
		return Binding{}, errors.New("skill_disabled")
	}
	if binding.TargetType == "" {
		return Binding{}, errors.New("target_type_required")
	}
	if !allowedTargetType(binding.TargetType) {
		return Binding{}, errors.New("target_type_invalid")
	}
	if binding.TargetID == "" {
		if binding.TargetType == "project" {
			binding.TargetID = "project"
		} else {
			return Binding{}, errors.New("target_id_required")
		}
	}
	if binding.Status == "" {
		binding.Status = "enabled"
	}
	if !allowedBindingStatus(binding.Status) {
		return Binding{}, errors.New("binding_status_invalid")
	}
	if binding.Status == "enabled" && skill.RiskLevel == "high" && binding.TargetType == "project" {
		return Binding{}, errors.New("high_risk_skill_requires_narrow_target")
	}
	if binding.ID == "" {
		binding.ID = "binding-" + binding.TargetType + "-" + normalizeID(binding.TargetID) + "-" + binding.SkillID
	}
	binding.ID = normalizeID(binding.ID)
	if binding.Priority == 0 {
		binding.Priority = defaultBindingPriority(binding.TargetType)
	}
	return binding, nil
}

func allowedTargetType(value string) bool {
	return value == "project" || value == "role" || value == "issue" || value == "subagent"
}

func allowedBindingStatus(value string) bool {
	return value == "candidate" || value == "enabled" || value == "disabled"
}

func defaultBindingPriority(targetType string) int {
	switch targetType {
	case "subagent":
		return 400
	case "issue":
		return 300
	case "role":
		return 200
	default:
		return 100
	}
}

func normalizeBindingOrder(bindings []Binding) []Binding {
	out := append([]Binding{}, bindings...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority == out[j].Priority {
			return out[i].ID < out[j].ID
		}
		return out[i].Priority > out[j].Priority
	})
	return out
}

func bindingsPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).SkillsDir, "bindings.json")
}

func bindingEventsPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).SkillsDir, "bindings.events.jsonl")
}

func ConfigFromPairs(pairs []string) map[string]string {
	config := map[string]string{}
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			key = pair
			value = "true"
		}
		key = normalizeToken(key)
		if key == "" {
			continue
		}
		config[key] = strings.TrimSpace(value)
	}
	return config
}

func PriorityFromString(value string) int {
	priority, _ := strconv.Atoi(strings.TrimSpace(value))
	return priority
}
