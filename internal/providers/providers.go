package providers

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

const (
	registryVersion = 1

	DecisionAllowed = "ROUTE_ALLOWED"
	DecisionBlocked = "ROUTE_BLOCKED"
)

type Registry struct {
	SchemaVersion int        `json:"schema_version"`
	Providers     []Provider `json:"providers"`
	UpdatedAt     string     `json:"updated_at"`
}

type Provider struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Vendor               string     `json:"vendor"`
	APIType              string     `json:"api_type"`
	BaseURL              string     `json:"base_url,omitempty"`
	AuthRef              string     `json:"auth_ref,omitempty"`
	Enabled              bool       `json:"enabled"`
	NativeRuntime        bool       `json:"native_runtime"`
	RuntimeID            string     `json:"runtime_id,omitempty"`
	DataPolicy           DataPolicy `json:"data_policy"`
	Models               []Model    `json:"models,omitempty"`
	UpstreamVendor       string     `json:"upstream_vendor,omitempty"`
	RequireProviderLabel bool       `json:"require_provider_label,omitempty"`
	AllowedUseCases      []string   `json:"allowed_use_cases,omitempty"`
	CreatedAt            string     `json:"created_at"`
	UpdatedAt            string     `json:"updated_at"`
}

type DataPolicy struct {
	AllowSensitiveCode     bool `json:"allow_sensitive_code"`
	AllowProjectMemory     bool `json:"allow_project_memory"`
	AllowProductionContext bool `json:"allow_production_context"`
}

type Model struct {
	ID           string   `json:"id"`
	Alias        string   `json:"alias,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type RouteRequest struct {
	Role                  string `json:"role"`
	TaskType              string `json:"task_type,omitempty"`
	OutputType            string `json:"output_type,omitempty"`
	RequiresRepoEdit      bool   `json:"requires_repo_edit"`
	IncludesSecrets       bool   `json:"includes_secrets"`
	IncludesSensitiveCode bool   `json:"includes_sensitive_code"`
	IncludesProjectMemory bool   `json:"includes_project_memory"`
}

type RouteDecision struct {
	Decision   string `json:"decision"`
	Blocked    bool   `json:"blocked"`
	ProviderID string `json:"provider_id,omitempty"`
	RuntimeID  string `json:"runtime_id,omitempty"`
	ModelID    string `json:"model_id,omitempty"`
	Reason     string `json:"reason"`
}

func Load(rootDir string) (Registry, error) {
	if err := workspace.EnsureDirs(workspace.ForRoot(rootDir)); err != nil {
		return Registry{}, err
	}
	path := registryPath(rootDir)
	var registry Registry
	found, err := fsutil.ReadJSON(path, &registry)
	if err != nil {
		return Registry{}, err
	}
	if !found || registry.SchemaVersion == 0 {
		registry = DefaultRegistry()
		if err := Save(rootDir, registry); err != nil {
			return Registry{}, err
		}
	}
	registry.Providers = normalizeProviderOrder(registry.Providers)
	return registry, nil
}

func Save(rootDir string, registry Registry) error {
	registry.SchemaVersion = registryVersion
	registry.Providers = normalizeProviderOrder(registry.Providers)
	registry.UpdatedAt = now()
	return fsutil.WriteJSON(registryPath(rootDir), registry)
}

func List(rootDir string) ([]Provider, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return nil, err
	}
	return registry.Providers, nil
}

func Show(rootDir string, id string) (Provider, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Provider{}, false, err
	}
	id = normalizeID(id)
	for _, provider := range registry.Providers {
		if provider.ID == id {
			return provider, true, nil
		}
	}
	return Provider{}, false, nil
}

func Upsert(rootDir string, provider Provider) (Provider, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Provider{}, err
	}
	provider, err = normalizeForSave(provider)
	if err != nil {
		return Provider{}, err
	}
	existingIndex := -1
	for i, existing := range registry.Providers {
		if existing.ID == provider.ID {
			existingIndex = i
			if provider.CreatedAt == "" {
				provider.CreatedAt = existing.CreatedAt
			}
			break
		}
	}
	if provider.CreatedAt == "" {
		provider.CreatedAt = now()
	}
	provider.UpdatedAt = now()
	if existingIndex >= 0 {
		registry.Providers[existingIndex] = provider
	} else {
		registry.Providers = append(registry.Providers, provider)
	}
	if err := Save(rootDir, registry); err != nil {
		return Provider{}, err
	}
	_ = logging.Log(rootDir, "audit", "provider.upserted", map[string]any{"provider_id": provider.ID, "api_type": provider.APIType, "native_runtime": provider.NativeRuntime})
	return provider, nil
}

func Disable(rootDir string, id string) (Provider, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Provider{}, false, err
	}
	id = normalizeID(id)
	for i, provider := range registry.Providers {
		if provider.ID != id {
			continue
		}
		provider.Enabled = false
		provider.UpdatedAt = now()
		registry.Providers[i] = provider
		if err := Save(rootDir, registry); err != nil {
			return Provider{}, false, err
		}
		_ = logging.Log(rootDir, "audit", "provider.disabled", map[string]any{"provider_id": provider.ID})
		return provider, true, nil
	}
	return Provider{}, false, nil
}

func Route(rootDir string, request RouteRequest) (RouteDecision, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return RouteDecision{}, err
	}
	decision := Decide(registry, request)
	_ = logging.Log(rootDir, "audit", "provider.route.decided", map[string]any{
		"decision":    decision.Decision,
		"provider_id": decision.ProviderID,
		"runtime_id":  decision.RuntimeID,
		"role":        request.Role,
		"task_type":   request.TaskType,
		"output_type": request.OutputType,
		"reason":      decision.Reason,
	})
	return decision, nil
}

func Decide(registry Registry, request RouteRequest) RouteDecision {
	request.Role = normalizeToken(request.Role)
	request.TaskType = normalizeToken(request.TaskType)
	request.OutputType = normalizeToken(request.OutputType)
	if request.IncludesSecrets {
		return blocked("contains_secret_context")
	}
	if request.OutputType == "architecture_diagram" || request.OutputType == "image" || request.TaskType == "image_generation" {
		return providerDecision(registry, request, "gpt_image_2", "architecture_diagram")
	}
	if request.RequiresRepoEdit {
		runtimeID := defaultRuntimeForRole(request.Role)
		return providerDecision(registry, request, runtimeID, "role_runtime_default")
	}
	if request.TaskType == "memory_extraction" || request.TaskType == "memory_compaction" {
		if decision, ok := firstMatchingAPIProvider(registry, request, []string{"glm", "minimax", "dashscope", "deepseek", "zhipu"}, "memory_low_cost_provider"); ok {
			return decision
		}
	}
	if request.TaskType == "architecture_planning" || request.TaskType == "requirement_planning" {
		if decision, ok := firstMatchingAPIProvider(registry, request, []string{"anthropic", "openai", "third_party"}, "planning_provider"); ok {
			return decision
		}
		return providerDecision(registry, request, "claude_cli", "planning_fallback_runtime")
	}
	return providerDecision(registry, request, defaultRuntimeForRole(request.Role), "default_runtime")
}

func DefaultRegistry() Registry {
	ts := now()
	return Registry{
		SchemaVersion: registryVersion,
		UpdatedAt:     ts,
		Providers: []Provider{
			{
				ID:            "claude_cli",
				Name:          "Claude Code CLI",
				Vendor:        "anthropic",
				APIType:       "claude_code_cli",
				Enabled:       true,
				NativeRuntime: true,
				RuntimeID:     "claude_cli",
				DataPolicy: DataPolicy{
					AllowSensitiveCode:     true,
					AllowProjectMemory:     true,
					AllowProductionContext: false,
				},
				AllowedUseCases: []string{"frontend", "architecture_planning", "requirement_planning"},
				CreatedAt:       ts,
				UpdatedAt:       ts,
			},
			{
				ID:            "codex_cli",
				Name:          "Codex CLI",
				Vendor:        "openai",
				APIType:       "codex_cli",
				Enabled:       true,
				NativeRuntime: true,
				RuntimeID:     "codex_cli",
				DataPolicy: DataPolicy{
					AllowSensitiveCode:     true,
					AllowProjectMemory:     true,
					AllowProductionContext: false,
				},
				AllowedUseCases: []string{"backend", "backend_tuning", "testing", "review", "repair"},
				CreatedAt:       ts,
				UpdatedAt:       ts,
			},
			{
				ID:      "gpt_image_2",
				Name:    "GPT Image 2",
				Vendor:  "openai",
				APIType: "image",
				AuthRef: "env:OPENAI_API_KEY",
				Models: []Model{{
					ID:           "gpt-image-2",
					Alias:        "gpt_image_2",
					Capabilities: []string{"architecture_diagram", "visual_explanation"},
				}},
				Enabled: false,
				DataPolicy: DataPolicy{
					AllowSensitiveCode:     false,
					AllowProjectMemory:     false,
					AllowProductionContext: false,
				},
				AllowedUseCases: []string{"architecture_diagram"},
				CreatedAt:       ts,
				UpdatedAt:       ts,
			},
		},
	}
}

func registryPath(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).MoyuanDir, "models", "providers.json")
}

func normalizeForSave(provider Provider) (Provider, error) {
	provider.ID = normalizeID(provider.ID)
	if provider.ID == "" {
		return Provider{}, errors.New("provider_id_required")
	}
	if provider.Name == "" {
		provider.Name = provider.ID
	}
	provider.Vendor = normalizeToken(provider.Vendor)
	provider.APIType = normalizeToken(provider.APIType)
	provider.RuntimeID = normalizeToken(provider.RuntimeID)
	provider.AuthRef = strings.TrimSpace(provider.AuthRef)
	provider.BaseURL = strings.TrimSpace(provider.BaseURL)
	provider.UpstreamVendor = normalizeToken(provider.UpstreamVendor)
	provider.AllowedUseCases = normalizeList(provider.AllowedUseCases)
	provider.Models = normalizeModels(provider.Models)
	if provider.Vendor == "" {
		return Provider{}, errors.New("provider_vendor_required")
	}
	if provider.APIType == "" {
		return Provider{}, errors.New("provider_api_type_required")
	}
	if provider.APIType == "openai_compatible" && provider.Vendor == "third_party" {
		provider.APIType = "third_party_openai_compatible"
	}
	if provider.AuthRef != "" && !isSafeAuthRef(provider.AuthRef) {
		return Provider{}, errors.New("auth_ref_must_be_reference")
	}
	if provider.NativeRuntime && provider.RuntimeID == "" {
		provider.RuntimeID = provider.ID
	}
	if provider.NativeRuntime && provider.RuntimeID == "" {
		return Provider{}, errors.New("runtime_id_required_for_native_provider")
	}
	if isThirdParty(provider) && !provider.RequireProviderLabel {
		return Provider{}, errors.New("third_party_provider_label_required")
	}
	if isThirdParty(provider) && len(provider.AllowedUseCases) == 0 {
		provider.AllowedUseCases = []string{"planning", "summary", "memory_extraction"}
	}
	if isThirdParty(provider) && provider.DataPolicy.AllowSensitiveCode {
		return Provider{}, errors.New("third_party_sensitive_code_not_allowed")
	}
	return provider, nil
}

func normalizeModels(models []Model) []Model {
	normalized := []Model{}
	for _, model := range models {
		model.ID = strings.TrimSpace(model.ID)
		model.Alias = normalizeToken(model.Alias)
		model.Capabilities = normalizeList(model.Capabilities)
		if model.ID == "" {
			continue
		}
		normalized = append(normalized, model)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].ID < normalized[j].ID })
	return normalized
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

func normalizeProviderOrder(providers []Provider) []Provider {
	out := append([]Provider{}, providers...)
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

func normalizeToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func isSafeAuthRef(value string) bool {
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

func defaultRuntimeForRole(role string) string {
	switch role {
	case "frontend", "architect", "requirement_refiner", "clarification_gate", "issue_planner", "dependency_planner":
		return "claude_cli"
	default:
		return "codex_cli"
	}
}

func DefaultRuntimeForRole(role string) string {
	return defaultRuntimeForRole(normalizeToken(role))
}

func providerDecision(registry Registry, request RouteRequest, providerID string, reason string) RouteDecision {
	provider, ok := providerByID(registry, providerID)
	if !ok {
		return blocked("provider_not_found:" + providerID)
	}
	if !provider.Enabled {
		return blocked("provider_disabled:" + providerID)
	}
	if violation := dataPolicyViolation(provider, request); violation != "" {
		return blocked(violation)
	}
	return allowed(provider, reason)
}

func firstMatchingAPIProvider(registry Registry, request RouteRequest, vendors []string, reason string) (RouteDecision, bool) {
	for _, vendor := range vendors {
		for _, provider := range registry.Providers {
			if !provider.Enabled || provider.NativeRuntime || provider.Vendor != vendor || provider.APIType == "image" {
				continue
			}
			if request.TaskType != "" && len(provider.AllowedUseCases) > 0 && !contains(provider.AllowedUseCases, request.TaskType) {
				continue
			}
			if violation := dataPolicyViolation(provider, request); violation != "" {
				continue
			}
			return allowed(provider, reason), true
		}
	}
	return RouteDecision{}, false
}

func providerByID(registry Registry, id string) (Provider, bool) {
	id = normalizeToken(id)
	for _, provider := range registry.Providers {
		if normalizeToken(provider.ID) == id || normalizeToken(provider.RuntimeID) == id {
			return provider, true
		}
	}
	return Provider{}, false
}

func dataPolicyViolation(provider Provider, request RouteRequest) string {
	if request.IncludesSensitiveCode && !provider.DataPolicy.AllowSensitiveCode {
		return fmt.Sprintf("provider_disallows_sensitive_code:%s", provider.ID)
	}
	if request.IncludesProjectMemory && !provider.DataPolicy.AllowProjectMemory {
		return fmt.Sprintf("provider_disallows_project_memory:%s", provider.ID)
	}
	if isThirdParty(provider) && request.RequiresRepoEdit {
		return fmt.Sprintf("third_party_provider_disallows_repo_edit:%s", provider.ID)
	}
	return ""
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func isThirdParty(provider Provider) bool {
	return provider.Vendor == "third_party" || provider.APIType == "third_party" || provider.APIType == "third_party_openai_compatible"
}

func allowed(provider Provider, reason string) RouteDecision {
	decision := RouteDecision{
		Decision:   DecisionAllowed,
		ProviderID: provider.ID,
		RuntimeID:  provider.RuntimeID,
		Reason:     reason,
	}
	if len(provider.Models) > 0 {
		decision.ModelID = provider.Models[0].ID
	}
	return decision
}

func blocked(reason string) RouteDecision {
	return RouteDecision{
		Decision: DecisionBlocked,
		Blocked:  true,
		Reason:   reason,
	}
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
