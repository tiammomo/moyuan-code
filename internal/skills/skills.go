package skills

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

const registryVersion = 1

type Registry struct {
	SchemaVersion int          `json:"schema_version"`
	Skills        []Definition `json:"skills"`
	UpdatedAt     string       `json:"updated_at"`
}

type Definition struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Source          string   `json:"source"`
	Version         string   `json:"version,omitempty"`
	Description     string   `json:"description,omitempty"`
	Enabled         bool     `json:"enabled"`
	RiskLevel       string   `json:"risk_level"`
	CompatibleRoles []string `json:"compatible_roles,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	RequiredTools   []string `json:"required_tools,omitempty"`
	AuthRef         string   `json:"auth_ref,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

func Load(rootDir string) (Registry, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Registry{}, err
	}
	var registry Registry
	found, err := fsutil.ReadJSON(registryPath(rootDir), &registry)
	if err != nil {
		return Registry{}, err
	}
	if !found || registry.SchemaVersion == 0 {
		registry = Registry{SchemaVersion: registryVersion, Skills: []Definition{}, UpdatedAt: now()}
		if err := save(rootDir, registry); err != nil {
			return Registry{}, err
		}
	}
	registry.Skills = normalizeOrder(registry.Skills)
	return registry, nil
}

func List(rootDir string) ([]Definition, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return nil, err
	}
	return registry.Skills, nil
}

func Show(rootDir string, id string) (Definition, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Definition{}, false, err
	}
	id = normalizeID(id)
	for _, skill := range registry.Skills {
		if skill.ID == id {
			return skill, true, nil
		}
	}
	return Definition{}, false, nil
}

func Upsert(rootDir string, skill Definition) (Definition, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Definition{}, err
	}
	skill, err = normalizeForSave(skill)
	if err != nil {
		return Definition{}, err
	}
	existingIndex := -1
	for idx, existing := range registry.Skills {
		if existing.ID != skill.ID {
			continue
		}
		existingIndex = idx
		if skill.CreatedAt == "" {
			skill.CreatedAt = existing.CreatedAt
		}
		break
	}
	if skill.CreatedAt == "" {
		skill.CreatedAt = now()
	}
	skill.UpdatedAt = now()
	if existingIndex >= 0 {
		registry.Skills[existingIndex] = skill
	} else {
		registry.Skills = append(registry.Skills, skill)
	}
	if err := save(rootDir, registry); err != nil {
		return Definition{}, err
	}
	if err := fsutil.AppendJSONL(eventsPath(rootDir), map[string]any{"event": "skill.upserted", "skill_id": skill.ID, "source": skill.Source, "ts": now()}); err != nil {
		return Definition{}, err
	}
	_ = logging.Log(rootDir, "audit", "skill.upserted", map[string]any{"skill_id": skill.ID, "source": skill.Source, "risk_level": skill.RiskLevel})
	return skill, nil
}

func Disable(rootDir string, id string) (Definition, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Definition{}, false, err
	}
	id = normalizeID(id)
	for idx, skill := range registry.Skills {
		if skill.ID != id {
			continue
		}
		skill.Enabled = false
		skill.UpdatedAt = now()
		registry.Skills[idx] = skill
		if err := save(rootDir, registry); err != nil {
			return Definition{}, false, err
		}
		if err := fsutil.AppendJSONL(eventsPath(rootDir), map[string]any{"event": "skill.disabled", "skill_id": skill.ID, "ts": now()}); err != nil {
			return Definition{}, false, err
		}
		_ = logging.Log(rootDir, "audit", "skill.disabled", map[string]any{"skill_id": skill.ID})
		return skill, true, nil
	}
	return Definition{}, false, nil
}

func save(rootDir string, registry Registry) error {
	registry.SchemaVersion = registryVersion
	registry.Skills = normalizeOrder(registry.Skills)
	registry.UpdatedAt = now()
	return fsutil.WriteJSON(registryPath(rootDir), registry)
}

func normalizeForSave(skill Definition) (Definition, error) {
	skill.ID = normalizeID(skill.ID)
	skill.Name = strings.TrimSpace(skill.Name)
	skill.Source = strings.TrimSpace(skill.Source)
	skill.Version = strings.TrimSpace(skill.Version)
	skill.Description = strings.TrimSpace(skill.Description)
	skill.RiskLevel = normalizeToken(skill.RiskLevel)
	skill.CompatibleRoles = normalizeList(skill.CompatibleRoles)
	skill.Tags = normalizeList(skill.Tags)
	skill.RequiredTools = normalizeList(skill.RequiredTools)
	skill.AuthRef = strings.TrimSpace(skill.AuthRef)
	if skill.ID == "" {
		return Definition{}, errors.New("skill_id_required")
	}
	if skill.Name == "" {
		skill.Name = skill.ID
	}
	if skill.Source == "" {
		return Definition{}, errors.New("skill_source_required")
	}
	if containsPlainSecret(skill.Name, skill.Source, skill.Version, skill.Description) {
		return Definition{}, errors.New("skill_metadata_must_not_contain_secret")
	}
	if skill.RiskLevel == "" {
		skill.RiskLevel = "medium"
	}
	if !allowedRisk(skill.RiskLevel) {
		return Definition{}, errors.New("skill_risk_level_invalid")
	}
	if skill.AuthRef != "" && !isSafeRef(skill.AuthRef) {
		return Definition{}, errors.New("auth_ref_must_be_reference")
	}
	return skill, nil
}

func registryPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).SkillsDir, "registry.json")
}

func eventsPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).SkillsDir, "events.jsonl")
}

func normalizeOrder(skills []Definition) []Definition {
	out := append([]Definition{}, skills...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func normalizeList(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = normalizeToken(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func normalizeID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastSep := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
			lastSep = false
			continue
		}
		if !lastSep {
			b.WriteByte('-')
			lastSep = true
		}
	}
	return strings.Trim(b.String(), "-_")
}

func normalizeToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func allowedRisk(value string) bool {
	return value == "low" || value == "medium" || value == "high"
}

func isSafeRef(value string) bool {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "env:") && len(value) > len("env:") {
		return !strings.Contains(value, "=") && !looksLikeSecret(value)
	}
	if strings.HasPrefix(value, "secret:") && len(value) > len("secret:") {
		return !strings.Contains(value, "=") && !looksLikeSecret(value)
	}
	return false
}

func looksLikeSecret(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "s"+"k-") ||
		strings.Contains(lower, "api"+"key=") ||
		strings.Contains(lower, "api_"+"key=") ||
		strings.Contains(lower, "token"+"=")
}

func containsPlainSecret(values ...string) bool {
	for _, value := range values {
		if looksLikeSecret(value) {
			return true
		}
	}
	return false
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
