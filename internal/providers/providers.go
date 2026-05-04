package providers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/approvals"
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
	Health               Health     `json:"health,omitempty"`
	Quota                Quota      `json:"quota,omitempty"`
	Usage                Usage      `json:"usage,omitempty"`
	Cost                 Cost       `json:"cost,omitempty"`
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

type Health struct {
	Status        string `json:"status,omitempty"`
	Reason        string `json:"reason,omitempty"`
	LastCheckedAt string `json:"last_checked_at,omitempty"`
}

type Quota struct {
	Status          string `json:"status,omitempty"`
	LimitTokens     int64  `json:"limit_tokens,omitempty"`
	UsedTokens      int64  `json:"used_tokens,omitempty"`
	RemainingTokens int64  `json:"remaining_tokens,omitempty"`
	ResetAt         string `json:"reset_at,omitempty"`
}

type Usage struct {
	Window       string `json:"window,omitempty"`
	Requests     int64  `json:"requests,omitempty"`
	InputTokens  int64  `json:"input_tokens,omitempty"`
	OutputTokens int64  `json:"output_tokens,omitempty"`
	TotalTokens  int64  `json:"total_tokens,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type Cost struct {
	Currency        string  `json:"currency,omitempty"`
	EstimatedAmount float64 `json:"estimated_amount,omitempty"`
	BudgetAmount    float64 `json:"budget_amount,omitempty"`
	Status          string  `json:"status,omitempty"`
}

type OpsSnapshot struct {
	Health Health `json:"health"`
	Quota  Quota  `json:"quota"`
	Usage  Usage  `json:"usage"`
	Cost   Cost   `json:"cost"`
}

type OpsRefreshOptions struct {
	ProviderID      string `json:"provider_id,omitempty"`
	IncludeDisabled bool   `json:"include_disabled,omitempty"`
	Probe           bool   `json:"probe,omitempty"`
	ProbeTimeoutMS  int    `json:"probe_timeout_ms,omitempty"`
	Approved        bool   `json:"approved,omitempty"`
}

type OpsRefreshResult struct {
	UpdatedAt  string               `json:"updated_at"`
	Updated    int                  `json:"updated"`
	Skipped    int                  `json:"skipped"`
	Decisions  []OpsRefreshDecision `json:"decisions"`
	Providers  []Provider           `json:"providers"`
	ProviderID string               `json:"provider_id,omitempty"`
	ApprovalID string               `json:"approval_id,omitempty"`
}

type OpsRefreshDecision struct {
	ProviderID   string `json:"provider_id"`
	Status       string `json:"status"`
	Reason       string `json:"reason"`
	HealthStatus string `json:"health_status,omitempty"`
	QuotaStatus  string `json:"quota_status,omitempty"`
	CostStatus   string `json:"cost_status,omitempty"`
	ProbeStatus  string `json:"probe_status,omitempty"`
	ProbeReason  string `json:"probe_reason,omitempty"`
}

type RouteRequest struct {
	Role                  string `json:"role"`
	ModelStrategy         string `json:"model_strategy,omitempty"`
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
	Strategy   string `json:"strategy,omitempty"`
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

func ResolveRuntimeProvider(rootDir string, runtimeID string, providerID string) (Provider, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Provider{}, false, err
	}
	runtimeID = normalizeToken(runtimeID)
	providerID = normalizeID(providerID)
	if providerID != "" {
		provider, ok := providerByID(registry, providerID)
		if !ok {
			return Provider{}, false, fmt.Errorf("provider_not_found:%s", providerID)
		}
		if !provider.Enabled {
			return Provider{}, false, fmt.Errorf("provider_disabled:%s", provider.ID)
		}
		if violation := availabilityViolation(provider); violation != "" {
			return Provider{}, false, errors.New(violation)
		}
		if !runtimeMatches(provider, runtimeID) {
			return Provider{}, false, fmt.Errorf("provider_runtime_mismatch:%s", provider.ID)
		}
		return provider, true, nil
	}
	for _, provider := range registry.Providers {
		if provider.Enabled && availabilityViolation(provider) == "" && provider.ID == runtimeID && runtimeMatches(provider, runtimeID) {
			return provider, true, nil
		}
	}
	for _, provider := range registry.Providers {
		if provider.Enabled && availabilityViolation(provider) == "" && runtimeMatches(provider, runtimeID) {
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

func UpdateOps(rootDir string, id string, snapshot OpsSnapshot) (Provider, bool, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return Provider{}, false, err
	}
	id = normalizeID(id)
	for i, provider := range registry.Providers {
		if provider.ID != id {
			continue
		}
		if err := normalizeOpsSnapshot(&snapshot); err != nil {
			return Provider{}, false, err
		}
		provider.Health = mergeHealth(provider.Health, snapshot.Health)
		provider.Quota = mergeQuota(provider.Quota, snapshot.Quota)
		provider.Usage = mergeUsage(provider.Usage, snapshot.Usage)
		provider.Cost = mergeCost(provider.Cost, snapshot.Cost)
		provider.UpdatedAt = now()
		registry.Providers[i] = provider
		if err := Save(rootDir, registry); err != nil {
			return Provider{}, false, err
		}
		_ = logging.Log(rootDir, "audit", "provider.ops.updated", map[string]any{
			"provider_id":  provider.ID,
			"health":       provider.Health.Status,
			"quota_status": provider.Quota.Status,
			"cost_status":  provider.Cost.Status,
		})
		return provider, true, nil
	}
	return Provider{}, false, nil
}

func RefreshOps(rootDir string, options OpsRefreshOptions) (OpsRefreshResult, error) {
	registry, err := Load(rootDir)
	if err != nil {
		return OpsRefreshResult{}, err
	}
	options.ProviderID = normalizeID(options.ProviderID)
	result := OpsRefreshResult{
		UpdatedAt:  now(),
		Decisions:  []OpsRefreshDecision{},
		Providers:  []Provider{},
		ProviderID: options.ProviderID,
	}
	if options.Probe && !options.Approved {
		targetID := options.ProviderID
		if targetID == "" {
			targetID = "all"
		}
		approval, err := approvals.Request(rootDir, approvals.RequestOptions{
			TargetType:  "provider_ops",
			TargetID:    targetID,
			Action:      "provider.probe",
			RiskLevel:   "high",
			RequestedBy: "system",
			Reason:      "provider probe can call external model provider endpoints",
			Metadata: map[string]any{
				"provider_id": options.ProviderID,
			},
		})
		if err != nil {
			return OpsRefreshResult{}, err
		}
		result.ApprovalID = approval.ID
		result.Skipped = countRefreshTargets(registry, options.ProviderID)
		result.Decisions = append(result.Decisions, OpsRefreshDecision{ProviderID: targetID, Status: "blocked", Reason: "provider_probe_approval_required"})
		_ = logging.Log(rootDir, "audit", "provider.ops.refresh.blocked", map[string]any{
			"provider_id": options.ProviderID,
			"approval_id": approval.ID,
			"probe":       options.Probe,
		})
		return result, nil
	}
	for _, provider := range registry.Providers {
		if options.ProviderID != "" && provider.ID != options.ProviderID {
			continue
		}
		if !provider.Enabled && !options.IncludeDisabled {
			result.Skipped++
			result.Decisions = append(result.Decisions, OpsRefreshDecision{ProviderID: provider.ID, Status: "skipped", Reason: "provider_disabled"})
			continue
		}
		snapshot := refreshSnapshotFor(provider, options)
		updated, ok, err := UpdateOps(rootDir, provider.ID, snapshot)
		if err != nil {
			return OpsRefreshResult{}, err
		}
		if !ok {
			result.Skipped++
			result.Decisions = append(result.Decisions, OpsRefreshDecision{ProviderID: provider.ID, Status: "skipped", Reason: "provider_not_found"})
			continue
		}
		result.Updated++
		result.Providers = append(result.Providers, updated)
		decision := OpsRefreshDecision{
			ProviderID:   updated.ID,
			Status:       "updated",
			Reason:       updated.Health.Reason,
			HealthStatus: updated.Health.Status,
			QuotaStatus:  updated.Quota.Status,
			CostStatus:   updated.Cost.Status,
		}
		if options.Probe {
			decision.ProbeStatus = updated.Health.Status
			decision.ProbeReason = updated.Health.Reason
		}
		result.Decisions = append(result.Decisions, decision)
	}
	_ = logging.Log(rootDir, "audit", "provider.ops.refreshed", map[string]any{
		"provider_id": options.ProviderID,
		"updated":     result.Updated,
		"skipped":     result.Skipped,
		"probe":       options.Probe,
	})
	return result, nil
}

func countRefreshTargets(registry Registry, providerID string) int {
	count := 0
	for _, provider := range registry.Providers {
		if providerID == "" || provider.ID == providerID {
			count++
		}
	}
	return count
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
		"strategy":    decision.Strategy,
		"task_type":   request.TaskType,
		"output_type": request.OutputType,
		"reason":      decision.Reason,
	})
	return decision, nil
}

func Decide(registry Registry, request RouteRequest) RouteDecision {
	request.Role = normalizeToken(request.Role)
	request.ModelStrategy = normalizeToken(request.ModelStrategy)
	request.TaskType = normalizeToken(request.TaskType)
	request.OutputType = normalizeToken(request.OutputType)
	request = applyModelStrategy(request)
	if request.IncludesSecrets {
		return withStrategy(blocked("contains_secret_context"), request.ModelStrategy)
	}
	if request.OutputType == "architecture_diagram" || request.OutputType == "image" || request.TaskType == "image_generation" {
		return withStrategy(providerDecision(registry, request, "gpt_image_2", "architecture_diagram"), request.ModelStrategy)
	}
	if request.RequiresRepoEdit {
		runtimeID := defaultRuntimeForRole(request.Role)
		if decision, ok := firstMatchingRuntimeProvider(registry, request, runtimeID, "role_provider_override"); ok {
			return withStrategy(decision, request.ModelStrategy)
		}
		return withStrategy(providerDecision(registry, request, runtimeID, "role_runtime_default"), request.ModelStrategy)
	}
	if request.TaskType == "memory_extraction" || request.TaskType == "memory_compaction" {
		if decision, ok := firstMatchingAPIProvider(registry, request, []string{"glm", "minimax", "dashscope", "deepseek", "zhipu"}, "memory_low_cost_provider"); ok {
			return withStrategy(decision, request.ModelStrategy)
		}
	}
	if request.TaskType == "architecture_planning" || request.TaskType == "requirement_planning" {
		if decision, ok := firstMatchingAPIProvider(registry, request, []string{"anthropic", "openai", "third_party"}, "planning_provider"); ok {
			return withStrategy(decision, request.ModelStrategy)
		}
		return withStrategy(providerDecision(registry, request, "claude_cli", "planning_fallback_runtime"), request.ModelStrategy)
	}
	return withStrategy(providerDecision(registry, request, defaultRuntimeForRole(request.Role), "default_runtime"), request.ModelStrategy)
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
	if violation := availabilityViolation(provider); violation != "" {
		return blocked(violation)
	}
	return allowed(provider, reason)
}

func firstMatchingAPIProvider(registry Registry, request RouteRequest, vendors []string, reason string) (RouteDecision, bool) {
	for _, vendor := range vendors {
		for _, provider := range registry.Providers {
			if !provider.Enabled || provider.NativeRuntime || provider.Vendor != vendor || provider.APIType == "image" || availabilityViolation(provider) != "" {
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

func firstMatchingRuntimeProvider(registry Registry, request RouteRequest, runtimeID string, reason string) (RouteDecision, bool) {
	runtimeID = normalizeToken(runtimeID)
	for _, provider := range registry.Providers {
		if !provider.Enabled || provider.ID == runtimeID || !runtimeMatches(provider, runtimeID) || availabilityViolation(provider) != "" {
			continue
		}
		if len(provider.AllowedUseCases) > 0 && !matchesUseCase(provider.AllowedUseCases, request) {
			continue
		}
		if violation := dataPolicyViolation(provider, request); violation != "" {
			continue
		}
		return allowed(provider, reason), true
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

func availabilityViolation(provider Provider) string {
	switch provider.Health.Status {
	case "unhealthy", "down":
		return fmt.Sprintf("provider_unhealthy:%s:%s", provider.ID, provider.Health.Status)
	}
	if provider.Quota.Status == "exhausted" {
		return fmt.Sprintf("provider_quota_exhausted:%s", provider.ID)
	}
	if provider.Cost.Status == "exceeded" {
		return fmt.Sprintf("provider_budget_exceeded:%s", provider.ID)
	}
	return ""
}

func refreshSnapshotFor(provider Provider, options OpsRefreshOptions) OpsSnapshot {
	health := refreshHealth(provider)
	if options.Probe && provider.Enabled && !provider.NativeRuntime && health.Status == "ok" {
		health = probeProviderHealth(provider, options.ProbeTimeoutMS)
	}
	snapshot := OpsSnapshot{
		Health: health,
		Quota:  refreshQuota(provider.Quota),
		Usage:  refreshUsage(provider.Usage),
		Cost:   refreshCost(provider.Cost),
	}
	return snapshot
}

func refreshHealth(provider Provider) Health {
	health := Health{Status: "ok", Reason: "configuration_present", LastCheckedAt: now()}
	if !provider.Enabled {
		return Health{Status: "unknown", Reason: "provider_disabled", LastCheckedAt: health.LastCheckedAt}
	}
	if provider.NativeRuntime {
		command := runtimeCommand(provider.RuntimeID)
		if command == "" {
			return Health{Status: "unknown", Reason: "native_runtime_command_unknown", LastCheckedAt: health.LastCheckedAt}
		}
		if _, err := exec.LookPath(command); err != nil {
			return Health{Status: "unhealthy", Reason: "native_runtime_missing:" + command, LastCheckedAt: health.LastCheckedAt}
		}
		return Health{Status: "ok", Reason: "native_runtime_found:" + command, LastCheckedAt: health.LastCheckedAt}
	}
	authStatus, authReason := authRefStatus(provider.AuthRef)
	if authStatus != "ok" {
		return Health{Status: "unhealthy", Reason: authReason, LastCheckedAt: health.LastCheckedAt}
	}
	if requiresBaseURL(provider) && strings.TrimSpace(provider.BaseURL) == "" {
		return Health{Status: "unhealthy", Reason: "base_url_missing", LastCheckedAt: health.LastCheckedAt}
	}
	return health
}

func probeProviderHealth(provider Provider, timeoutMS int) Health {
	checkedAt := now()
	probeURL, adapter, ok := providerProbeURL(provider)
	if !ok {
		return Health{Status: "degraded", Reason: "probe_adapter_unsupported", LastCheckedAt: checkedAt}
	}
	authToken, authStatus, authReason := probeAuthToken(provider.AuthRef)
	switch authStatus {
	case "ok":
	case "skipped":
		return Health{Status: "degraded", Reason: authReason, LastCheckedAt: checkedAt}
	default:
		return Health{Status: "unhealthy", Reason: authReason, LastCheckedAt: checkedAt}
	}
	timeout := 3 * time.Second
	if timeoutMS > 0 {
		timeout = time.Duration(timeoutMS) * time.Millisecond
	}
	req, err := http.NewRequest(http.MethodGet, probeURL, nil)
	if err != nil {
		return Health{Status: "unhealthy", Reason: "probe_request_invalid", LastCheckedAt: checkedAt}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "moyuan-code-provider-probe/1")
	if strings.Contains(provider.APIType, "anthropic") {
		req.Header.Set("x-api-key", authToken)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return Health{Status: "unhealthy", Reason: "probe_request_failed", LastCheckedAt: checkedAt}
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 400:
		return Health{Status: "ok", Reason: "probe_ok:" + adapter, LastCheckedAt: checkedAt}
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return Health{Status: "unhealthy", Reason: fmt.Sprintf("probe_auth_failed:http_%d", resp.StatusCode), LastCheckedAt: checkedAt}
	case resp.StatusCode == http.StatusTooManyRequests:
		return Health{Status: "degraded", Reason: "probe_rate_limited:http_429", LastCheckedAt: checkedAt}
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed:
		return Health{Status: "degraded", Reason: fmt.Sprintf("probe_endpoint_not_supported:http_%d", resp.StatusCode), LastCheckedAt: checkedAt}
	case resp.StatusCode >= 500:
		return Health{Status: "unhealthy", Reason: fmt.Sprintf("probe_upstream_error:http_%d", resp.StatusCode), LastCheckedAt: checkedAt}
	default:
		return Health{Status: "degraded", Reason: fmt.Sprintf("probe_unexpected_status:http_%d", resp.StatusCode), LastCheckedAt: checkedAt}
	}
}

func providerProbeURL(provider Provider) (string, string, bool) {
	base, err := url.Parse(strings.TrimSpace(provider.BaseURL))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", "", false
	}
	apiType := normalizeToken(provider.APIType)
	if strings.Contains(apiType, "openai") || provider.Vendor == "openai" || apiType == "image" {
		appendProbePath(base, "models")
		return base.String(), "openai_compatible_models", true
	}
	if strings.Contains(apiType, "anthropic") {
		return base.String(), "anthropic_compatible_base", true
	}
	if strings.Contains(apiType, "compatible") || isThirdParty(provider) {
		appendProbePath(base, "models")
		return base.String(), "generic_compatible_models", true
	}
	return "", "", false
}

func appendProbePath(base *url.URL, segment string) {
	path := strings.TrimRight(base.Path, "/")
	if strings.HasSuffix(path, "/"+segment) {
		base.Path = path
		return
	}
	base.Path = path + "/" + segment
}

func probeAuthToken(authRef string) (string, string, string) {
	authRef = strings.TrimSpace(authRef)
	if authRef == "" {
		return "", "missing", "auth_ref_missing"
	}
	if strings.HasPrefix(authRef, "env:") {
		key := strings.TrimSpace(strings.TrimPrefix(authRef, "env:"))
		if key == "" {
			return "", "missing", "auth_ref_env_required"
		}
		token := os.Getenv(key)
		if token == "" {
			return "", "missing", "auth_ref_env_missing:" + key
		}
		return token, "ok", "auth_ref_env_present:" + key
	}
	if strings.HasPrefix(authRef, "secret:") {
		return "", "skipped", "probe_secret_ref_not_resolved"
	}
	return "", "invalid", "auth_ref_unsupported"
}

func refreshQuota(existing Quota) Quota {
	quota := existing
	if quota.LimitTokens <= 0 {
		if quota.Status == "" {
			quota.Status = "unknown"
		}
		return quota
	}
	if quota.LimitTokens >= quota.UsedTokens {
		quota.RemainingTokens = quota.LimitTokens - quota.UsedTokens
	} else {
		quota.RemainingTokens = 0
	}
	switch {
	case quota.UsedTokens >= quota.LimitTokens || quota.RemainingTokens <= 0:
		quota.Status = "exhausted"
	case float64(quota.UsedTokens)/float64(quota.LimitTokens) >= 0.8:
		quota.Status = "warning"
	default:
		quota.Status = "ok"
	}
	return quota
}

func refreshUsage(existing Usage) Usage {
	usage := existing
	if usage.UpdatedAt == "" && hasUsage(usage) {
		usage.UpdatedAt = now()
	}
	return usage
}

func refreshCost(existing Cost) Cost {
	cost := existing
	if cost.BudgetAmount <= 0 {
		if cost.Status == "" {
			cost.Status = "unknown"
		}
		return cost
	}
	switch {
	case cost.EstimatedAmount >= cost.BudgetAmount:
		cost.Status = "exceeded"
	case cost.EstimatedAmount/cost.BudgetAmount >= 0.8:
		cost.Status = "warning"
	default:
		cost.Status = "ok"
	}
	return cost
}

func authRefStatus(authRef string) (string, string) {
	authRef = strings.TrimSpace(authRef)
	if authRef == "" {
		return "missing", "auth_ref_missing"
	}
	if strings.HasPrefix(authRef, "env:") {
		key := strings.TrimSpace(strings.TrimPrefix(authRef, "env:"))
		if key == "" {
			return "missing", "auth_ref_env_required"
		}
		if os.Getenv(key) == "" {
			return "missing", "auth_ref_env_missing:" + key
		}
		return "ok", "auth_ref_env_present:" + key
	}
	if strings.HasPrefix(authRef, "secret:") {
		return "ok", "auth_ref_secret_reference_present"
	}
	return "invalid", "auth_ref_unsupported"
}

func requiresBaseURL(provider Provider) bool {
	if provider.BaseURL != "" {
		return false
	}
	if provider.NativeRuntime || provider.Vendor == "openai" || provider.APIType == "image" {
		return false
	}
	return strings.Contains(provider.APIType, "compatible") || isThirdParty(provider)
}

func runtimeCommand(runtimeID string) string {
	switch normalizeToken(runtimeID) {
	case "claude_cli":
		return "claude"
	case "codex_cli":
		return "codex"
	case "local_shell":
		return "sh"
	default:
		return ""
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func matchesUseCase(allowed []string, request RouteRequest) bool {
	candidates := []string{request.TaskType, request.Role, request.OutputType}
	for _, candidate := range candidates {
		if candidate != "" && contains(allowed, candidate) {
			return true
		}
	}
	return false
}

func runtimeMatches(provider Provider, runtimeID string) bool {
	runtimeID = normalizeToken(runtimeID)
	if runtimeID == "" {
		return false
	}
	if normalizeToken(provider.RuntimeID) == runtimeID {
		return true
	}
	return normalizeToken(provider.ID) == runtimeID && provider.RuntimeID == ""
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

func withStrategy(decision RouteDecision, strategy string) RouteDecision {
	if strategy != "" {
		decision.Strategy = strategy
	}
	return decision
}

func applyModelStrategy(request RouteRequest) RouteRequest {
	switch request.ModelStrategy {
	case "frontend_first":
		if request.Role == "" {
			request.Role = "frontend"
		}
		request.RequiresRepoEdit = true
	case "backend_safe":
		if request.Role == "" {
			request.Role = "backend"
		}
		request.RequiresRepoEdit = true
	case "low_cost_memory":
		if request.TaskType == "" {
			request.TaskType = "memory_extraction"
		}
	case "image_diagram":
		request.OutputType = "architecture_diagram"
		request.TaskType = "image_generation"
	case "planning":
		if request.TaskType == "" {
			request.TaskType = "architecture_planning"
		}
		if request.Role == "" {
			request.Role = "architect"
		}
	}
	return request
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func normalizeOpsSnapshot(snapshot *OpsSnapshot) error {
	snapshot.Health.Status = normalizeToken(snapshot.Health.Status)
	snapshot.Health.Reason = strings.TrimSpace(snapshot.Health.Reason)
	snapshot.Health.LastCheckedAt = strings.TrimSpace(snapshot.Health.LastCheckedAt)
	snapshot.Quota.Status = normalizeToken(snapshot.Quota.Status)
	snapshot.Quota.ResetAt = strings.TrimSpace(snapshot.Quota.ResetAt)
	snapshot.Usage.Window = normalizeToken(snapshot.Usage.Window)
	snapshot.Usage.UpdatedAt = strings.TrimSpace(snapshot.Usage.UpdatedAt)
	snapshot.Cost.Currency = strings.ToUpper(strings.TrimSpace(snapshot.Cost.Currency))
	snapshot.Cost.Status = normalizeToken(snapshot.Cost.Status)
	if snapshot.Health.Status != "" && !allowedHealthStatus(snapshot.Health.Status) {
		return errors.New("provider_health_status_invalid")
	}
	if snapshot.Quota.Status != "" && !allowedQuotaStatus(snapshot.Quota.Status) {
		return errors.New("provider_quota_status_invalid")
	}
	if snapshot.Cost.Status != "" && !allowedCostStatus(snapshot.Cost.Status) {
		return errors.New("provider_cost_status_invalid")
	}
	if snapshot.Quota.LimitTokens < 0 || snapshot.Quota.UsedTokens < 0 || snapshot.Quota.RemainingTokens < 0 ||
		snapshot.Usage.Requests < 0 || snapshot.Usage.InputTokens < 0 || snapshot.Usage.OutputTokens < 0 || snapshot.Usage.TotalTokens < 0 ||
		snapshot.Cost.EstimatedAmount < 0 || snapshot.Cost.BudgetAmount < 0 {
		return errors.New("provider_ops_values_must_not_be_negative")
	}
	if snapshot.Quota.RemainingTokens == 0 && snapshot.Quota.LimitTokens > 0 && snapshot.Quota.UsedTokens > 0 && snapshot.Quota.LimitTokens >= snapshot.Quota.UsedTokens {
		snapshot.Quota.RemainingTokens = snapshot.Quota.LimitTokens - snapshot.Quota.UsedTokens
	}
	if snapshot.Usage.TotalTokens == 0 {
		snapshot.Usage.TotalTokens = snapshot.Usage.InputTokens + snapshot.Usage.OutputTokens
	}
	if snapshot.Health.LastCheckedAt == "" && snapshot.Health.Status != "" {
		snapshot.Health.LastCheckedAt = now()
	}
	if snapshot.Usage.UpdatedAt == "" && hasUsage(snapshot.Usage) {
		snapshot.Usage.UpdatedAt = now()
	}
	return nil
}

func mergeHealth(existing Health, incoming Health) Health {
	if incoming.Status != "" {
		existing.Status = incoming.Status
	}
	if incoming.Reason != "" {
		existing.Reason = incoming.Reason
	}
	if incoming.LastCheckedAt != "" {
		existing.LastCheckedAt = incoming.LastCheckedAt
	}
	return existing
}

func mergeQuota(existing Quota, incoming Quota) Quota {
	if incoming.Status != "" {
		existing.Status = incoming.Status
	}
	if incoming.LimitTokens != 0 {
		existing.LimitTokens = incoming.LimitTokens
	}
	if incoming.UsedTokens != 0 {
		existing.UsedTokens = incoming.UsedTokens
	}
	if incoming.RemainingTokens != 0 || incoming.LimitTokens > 0 {
		existing.RemainingTokens = incoming.RemainingTokens
	}
	if incoming.ResetAt != "" {
		existing.ResetAt = incoming.ResetAt
	}
	return existing
}

func mergeUsage(existing Usage, incoming Usage) Usage {
	if incoming.Window != "" {
		existing.Window = incoming.Window
	}
	if incoming.Requests != 0 {
		existing.Requests = incoming.Requests
	}
	if incoming.InputTokens != 0 {
		existing.InputTokens = incoming.InputTokens
	}
	if incoming.OutputTokens != 0 {
		existing.OutputTokens = incoming.OutputTokens
	}
	if incoming.TotalTokens != 0 {
		existing.TotalTokens = incoming.TotalTokens
	}
	if incoming.UpdatedAt != "" {
		existing.UpdatedAt = incoming.UpdatedAt
	}
	return existing
}

func mergeCost(existing Cost, incoming Cost) Cost {
	if incoming.Currency != "" {
		existing.Currency = incoming.Currency
	}
	if incoming.EstimatedAmount != 0 {
		existing.EstimatedAmount = incoming.EstimatedAmount
	}
	if incoming.BudgetAmount != 0 {
		existing.BudgetAmount = incoming.BudgetAmount
	}
	if incoming.Status != "" {
		existing.Status = incoming.Status
	}
	return existing
}

func hasUsage(usage Usage) bool {
	return usage.Window != "" || usage.Requests != 0 || usage.InputTokens != 0 || usage.OutputTokens != 0 || usage.TotalTokens != 0
}

func allowedHealthStatus(value string) bool {
	return value == "ok" || value == "healthy" || value == "degraded" || value == "unhealthy" || value == "down" || value == "unknown"
}

func allowedQuotaStatus(value string) bool {
	return value == "ok" || value == "warning" || value == "exhausted" || value == "unknown"
}

func allowedCostStatus(value string) bool {
	return value == "ok" || value == "warning" || value == "exceeded" || value == "unknown"
}
