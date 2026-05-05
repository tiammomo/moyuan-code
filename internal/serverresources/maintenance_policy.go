package serverresources

import (
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

const defaultMaintenancePolicyID = "maintenance-policy-default-v1"

type MaintenancePolicyPack struct {
	ID                 string                                  `json:"id" yaml:"id"`
	Version            string                                  `json:"version" yaml:"version"`
	Source             string                                  `json:"source,omitempty" yaml:"-"`
	DefaultEnvironment string                                  `json:"default_environment,omitempty" yaml:"default_environment"`
	Environments       map[string]MaintenanceEnvironmentPolicy `json:"environments,omitempty" yaml:"environments"`
}

type MaintenanceEnvironmentPolicy struct {
	MaintenanceWindows    []string `json:"maintenance_windows,omitempty" yaml:"maintenance_windows"`
	FreezeWindows         []string `json:"freeze_windows,omitempty" yaml:"freeze_windows"`
	AllowedActions        []string `json:"allowed_actions,omitempty" yaml:"allowed_actions"`
	ManualRequiredActions []string `json:"manual_required_actions,omitempty" yaml:"manual_required_actions"`
	BlockedActions        []string `json:"blocked_actions,omitempty" yaml:"blocked_actions"`
	OutsideWindowEffect   string   `json:"outside_window_effect,omitempty" yaml:"outside_window_effect"`
}

type MaintenancePolicyContext struct {
	Environment string `json:"environment,omitempty"`
	Action      string `json:"action,omitempty"`
	ResourceID  string `json:"resource_id,omitempty"`
	RequestedAt string `json:"requested_at,omitempty"`
}

type MaintenancePolicyDecision struct {
	PolicyID                string                       `json:"policy_id"`
	PolicyVersion           string                       `json:"policy_version,omitempty"`
	PolicySource            string                       `json:"policy_source,omitempty"`
	Environment             string                       `json:"environment,omitempty"`
	Action                  string                       `json:"action,omitempty"`
	ResourceID              string                       `json:"resource_id,omitempty"`
	RequestedAt             string                       `json:"requested_at,omitempty"`
	Status                  string                       `json:"status"`
	Decision                string                       `json:"decision"`
	Reasons                 []string                     `json:"reasons"`
	MatchedRules            []MaintenancePolicyRuleMatch `json:"matched_rules,omitempty"`
	WithinMaintenanceWindow bool                         `json:"within_maintenance_window"`
	InFreezeWindow          bool                         `json:"in_freeze_window"`
	Blocked                 bool                         `json:"blocked"`
	ManualRequired          bool                         `json:"manual_required"`
	Allowed                 bool                         `json:"allowed"`
}

type MaintenancePolicyRuleMatch struct {
	PolicyID string `json:"policy_id"`
	RuleID   string `json:"rule_id"`
	Effect   string `json:"effect"`
	Reason   string `json:"reason"`
}

type serverResourcesPolicyConfigFile struct {
	MaintenancePolicyPack MaintenancePolicyPack `json:"maintenance_policy_pack" yaml:"maintenance_policy_pack"`
}

func DefaultMaintenancePolicyPack() MaintenancePolicyPack {
	allowedActions := []string{"health_scan", "lifecycle_scan", "maintenance_scan", "renew", "retire", "deploy", "rollback", "release_publish"}
	writeActions := []string{"renew", "retire", "deploy", "rollback", "release_publish"}
	return MaintenancePolicyPack{
		ID:                 defaultMaintenancePolicyID,
		Version:            "2026-05-05",
		Source:             "builtin",
		DefaultEnvironment: "default",
		Environments: map[string]MaintenanceEnvironmentPolicy{
			"default": {
				MaintenanceWindows:    []string{"always"},
				AllowedActions:        allowedActions,
				ManualRequiredActions: writeActions,
				OutsideWindowEffect:   "manual",
			},
			"test_dev": {
				MaintenanceWindows:    []string{"always"},
				AllowedActions:        allowedActions,
				ManualRequiredActions: []string{"retire"},
				OutsideWindowEffect:   "manual",
			},
			"production": {
				AllowedActions:        allowedActions,
				ManualRequiredActions: writeActions,
				OutsideWindowEffect:   "manual",
			},
		},
	}
}

func LoadMaintenancePolicyPack(rootDir string, environment string) (MaintenancePolicyPack, error) {
	pack := DefaultMaintenancePolicyPack()
	text, found, err := fsutil.ReadText(workspace.ForRoot(rootDir).ServerResourcesYAML)
	if err != nil || !found {
		return normalizeMaintenancePolicyPack(pack, environment), err
	}
	var raw serverResourcesPolicyConfigFile
	if err := yaml.Unmarshal([]byte(text), &raw); err != nil {
		return MaintenancePolicyPack{}, err
	}
	custom := raw.MaintenancePolicyPack
	if emptyMaintenancePolicyPack(custom) {
		return normalizeMaintenancePolicyPack(pack, environment), nil
	}
	if strings.TrimSpace(custom.ID) != "" {
		pack.ID = strings.TrimSpace(custom.ID)
	}
	if strings.TrimSpace(custom.Version) != "" {
		pack.Version = strings.TrimSpace(custom.Version)
	}
	if strings.TrimSpace(custom.DefaultEnvironment) != "" {
		pack.DefaultEnvironment = normalizeToken(custom.DefaultEnvironment)
	}
	if pack.Environments == nil {
		pack.Environments = map[string]MaintenanceEnvironmentPolicy{}
	}
	for key, policy := range custom.Environments {
		normalizedKey := normalizeToken(key)
		base, ok := pack.Environments[normalizedKey]
		if !ok {
			base = pack.Environments[pack.DefaultEnvironment]
		}
		pack.Environments[normalizedKey] = mergeMaintenanceEnvironmentPolicy(base, policy)
	}
	pack.Source = "configured"
	return normalizeMaintenancePolicyPack(pack, environment), nil
}

func EvaluateMaintenancePolicy(pack MaintenancePolicyPack, context MaintenancePolicyContext) MaintenancePolicyDecision {
	context.Environment = normalizeToken(context.Environment)
	context.Action = normalizeToken(context.Action)
	context.ResourceID = normalizeID(context.ResourceID)
	requestedAt := parsePolicyTime(context.RequestedAt)
	if strings.TrimSpace(context.RequestedAt) == "" {
		context.RequestedAt = requestedAt.Format(time.RFC3339)
	}
	pack = normalizeMaintenancePolicyPack(pack, context.Environment)
	envPolicy := maintenanceEnvironmentPolicy(pack, context.Environment)
	withinWindow, hasMaintenanceWindow := matchesAnyPolicyWindow(envPolicy.MaintenanceWindows, requestedAt)
	inFreezeWindow, _ := matchesAnyPolicyWindow(envPolicy.FreezeWindows, requestedAt)
	decision := MaintenancePolicyDecision{
		PolicyID:                pack.ID,
		PolicyVersion:           pack.Version,
		PolicySource:            pack.Source,
		Environment:             context.Environment,
		Action:                  context.Action,
		ResourceID:              context.ResourceID,
		RequestedAt:             context.RequestedAt,
		Status:                  "allowed",
		Decision:                "MAINTENANCE_POLICY_ALLOWED",
		Reasons:                 []string{},
		MatchedRules:            []MaintenancePolicyRuleMatch{},
		WithinMaintenanceWindow: withinWindow,
		InFreezeWindow:          inFreezeWindow,
		Allowed:                 true,
	}
	if context.Action == "" {
		return applyMaintenanceDecision(pack, decision, "action_required", "block")
	}
	if contains(envPolicy.BlockedActions, context.Action) {
		return applyMaintenanceDecision(pack, decision, "action_blocked:"+context.Action, "block")
	}
	if len(envPolicy.AllowedActions) > 0 && !contains(envPolicy.AllowedActions, context.Action) {
		return applyMaintenanceDecision(pack, decision, "action_not_allowed:"+context.Action, "block")
	}
	if inFreezeWindow {
		return applyMaintenanceDecision(pack, decision, "freeze_window_active", "block")
	}
	if !withinWindow {
		reason := "outside_maintenance_window"
		if !hasMaintenanceWindow {
			reason = "maintenance_window_missing"
		}
		decision = applyMaintenanceDecision(pack, decision, reason, envPolicy.OutsideWindowEffect)
		if decision.Blocked {
			return decision
		}
	}
	if contains(envPolicy.ManualRequiredActions, context.Action) {
		decision = applyMaintenanceDecision(pack, decision, "manual_required_action:"+context.Action, "manual")
	}
	if !decision.Blocked && !decision.ManualRequired {
		decision.Reasons = appendUniqueStrings(decision.Reasons, "maintenance_policy_allowed")
	}
	return decision
}

func normalizeMaintenancePolicyPack(pack MaintenancePolicyPack, environment string) MaintenancePolicyPack {
	pack.ID = strings.TrimSpace(pack.ID)
	if pack.ID == "" {
		pack.ID = defaultMaintenancePolicyID
	}
	pack.Version = strings.TrimSpace(pack.Version)
	if pack.Version == "" {
		pack.Version = "2026-05-05"
	}
	if pack.Source == "" {
		pack.Source = "builtin"
	}
	pack.DefaultEnvironment = normalizeToken(pack.DefaultEnvironment)
	if pack.DefaultEnvironment == "" {
		pack.DefaultEnvironment = "default"
	}
	normalized := map[string]MaintenanceEnvironmentPolicy{}
	for key, policy := range pack.Environments {
		normalized[normalizeToken(key)] = normalizeMaintenanceEnvironmentPolicy(policy)
	}
	if _, ok := normalized[pack.DefaultEnvironment]; !ok {
		normalized[pack.DefaultEnvironment] = normalizeMaintenanceEnvironmentPolicy(MaintenanceEnvironmentPolicy{MaintenanceWindows: []string{"always"}, OutsideWindowEffect: "manual"})
	}
	if environment != "" {
		environment = normalizeToken(environment)
		if _, ok := normalized[environment]; !ok {
			normalized[environment] = normalized[pack.DefaultEnvironment]
		}
	}
	pack.Environments = normalized
	return pack
}

func normalizeMaintenanceEnvironmentPolicy(policy MaintenanceEnvironmentPolicy) MaintenanceEnvironmentPolicy {
	policy.MaintenanceWindows = normalizePolicyWindows(policy.MaintenanceWindows)
	policy.FreezeWindows = normalizePolicyWindows(policy.FreezeWindows)
	policy.AllowedActions = normalizePolicyTokens(policy.AllowedActions)
	policy.ManualRequiredActions = normalizePolicyTokens(policy.ManualRequiredActions)
	policy.BlockedActions = normalizePolicyTokens(policy.BlockedActions)
	policy.OutsideWindowEffect = normalizeMaintenanceEffect(policy.OutsideWindowEffect)
	if policy.OutsideWindowEffect == "" {
		policy.OutsideWindowEffect = "manual"
	}
	return policy
}

func mergeMaintenanceEnvironmentPolicy(base MaintenanceEnvironmentPolicy, custom MaintenanceEnvironmentPolicy) MaintenanceEnvironmentPolicy {
	base = normalizeMaintenanceEnvironmentPolicy(base)
	custom = normalizeMaintenanceEnvironmentPolicy(custom)
	return normalizeMaintenanceEnvironmentPolicy(MaintenanceEnvironmentPolicy{
		MaintenanceWindows:    appendUniqueStrings(base.MaintenanceWindows, custom.MaintenanceWindows...),
		FreezeWindows:         appendUniqueStrings(base.FreezeWindows, custom.FreezeWindows...),
		AllowedActions:        appendUniqueStrings(base.AllowedActions, custom.AllowedActions...),
		ManualRequiredActions: appendUniqueStrings(base.ManualRequiredActions, custom.ManualRequiredActions...),
		BlockedActions:        appendUniqueStrings(base.BlockedActions, custom.BlockedActions...),
		OutsideWindowEffect:   stricterMaintenanceEffect(base.OutsideWindowEffect, custom.OutsideWindowEffect),
	})
}

func maintenanceEnvironmentPolicy(pack MaintenancePolicyPack, environment string) MaintenanceEnvironmentPolicy {
	environment = normalizeToken(environment)
	if policy, ok := pack.Environments[environment]; ok {
		return policy
	}
	return pack.Environments[pack.DefaultEnvironment]
}

func emptyMaintenancePolicyPack(pack MaintenancePolicyPack) bool {
	return strings.TrimSpace(pack.ID) == "" && strings.TrimSpace(pack.Version) == "" && strings.TrimSpace(pack.DefaultEnvironment) == "" && len(pack.Environments) == 0
}

func normalizeMaintenanceEffect(value string) string {
	value = normalizeToken(value)
	switch value {
	case "block", "manual", "allow":
		return value
	default:
		return ""
	}
}

func stricterMaintenanceEffect(left string, right string) string {
	priority := map[string]int{"": 0, "allow": 1, "manual": 2, "block": 3}
	left = normalizeMaintenanceEffect(left)
	right = normalizeMaintenanceEffect(right)
	if priority[right] > priority[left] {
		return right
	}
	return left
}

func applyMaintenanceDecision(pack MaintenancePolicyPack, decision MaintenancePolicyDecision, reason string, effect string) MaintenancePolicyDecision {
	effect = normalizeMaintenanceEffect(effect)
	if effect == "" {
		effect = "manual"
	}
	reason = strings.TrimSpace(reason)
	if reason != "" {
		decision.Reasons = appendUniqueStrings(decision.Reasons, reason)
		decision.MatchedRules = append(decision.MatchedRules, MaintenancePolicyRuleMatch{
			PolicyID: pack.ID,
			RuleID:   reason,
			Effect:   effect,
			Reason:   reason,
		})
	}
	switch effect {
	case "block":
		decision.Status = "blocked"
		decision.Decision = "MAINTENANCE_POLICY_BLOCKED"
		decision.Blocked = true
		decision.ManualRequired = false
		decision.Allowed = false
	case "manual":
		if !decision.Blocked {
			decision.Status = "manual_required"
			decision.Decision = "MAINTENANCE_POLICY_MANUAL_REVIEW_REQUIRED"
			decision.ManualRequired = true
			decision.Allowed = false
		}
	case "allow":
		if !decision.Blocked && !decision.ManualRequired {
			decision.Status = "allowed"
			decision.Decision = "MAINTENANCE_POLICY_ALLOWED"
			decision.Allowed = true
		}
	}
	return decision
}

func matchesAnyPolicyWindow(windows []string, at time.Time) (bool, bool) {
	if len(windows) == 0 {
		return false, false
	}
	for _, window := range windows {
		if matchesPolicyWindow(window, at) {
			return true, true
		}
	}
	return false, true
}

func matchesPolicyWindow(window string, at time.Time) bool {
	window = strings.TrimSpace(strings.ToLower(window))
	if window == "" {
		return false
	}
	if window == "always" || window == "*" {
		return true
	}
	currentDate := at.Format("2006-01-02")
	if !strings.Contains(window, "..") {
		return window == currentDate
	}
	parts := strings.SplitN(window, "..", 2)
	if len(parts) != 2 {
		return false
	}
	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])
	if start != "" && currentDate < start {
		return false
	}
	if end != "" && currentDate > end {
		return false
	}
	return true
}

func parsePolicyTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().UTC()
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.UTC()
	}
	return time.Now().UTC()
}

func normalizePolicyTokens(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = normalizeToken(value)
		if value != "" {
			out = appendUniqueStrings(out, value)
		}
	}
	return out
}

func normalizePolicyWindows(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value != "" {
			out = appendUniqueStrings(out, value)
		}
	}
	return out
}

func appendUniqueStrings(values []string, additions ...string) []string {
	out := append([]string{}, values...)
	for _, value := range additions {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		exists := false
		for _, existing := range out {
			if existing == value {
				exists = true
				break
			}
		}
		if !exists {
			out = append(out, value)
		}
	}
	return out
}
