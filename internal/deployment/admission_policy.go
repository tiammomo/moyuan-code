package deployment

import (
	"strings"

	"gopkg.in/yaml.v3"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

const defaultReleaseAdmissionPolicyID = "release-admission-default-v1"

type ReleaseAdmissionPolicyPack struct {
	ID                 string                                `json:"id" yaml:"id"`
	Version            string                                `json:"version" yaml:"version"`
	Source             string                                `json:"source,omitempty" yaml:"-"`
	DefaultEnvironment string                                `json:"default_environment,omitempty" yaml:"default_environment"`
	Environments       map[string]AdmissionEnvironmentPolicy `json:"environments,omitempty" yaml:"environments"`
	Rules              []AdmissionPolicyRule                 `json:"rules" yaml:"rules"`
}

type AdmissionEnvironmentPolicy struct {
	RequireSignalTypes          []string `json:"require_signal_types,omitempty" yaml:"require_signal_types"`
	MissingRequiredSignalEffect string   `json:"missing_required_signal_effect,omitempty" yaml:"missing_required_signal_effect"`
	MonitorUnknownEffect        string   `json:"monitor_unknown_effect,omitempty" yaml:"monitor_unknown_effect"`
}

type AdmissionPolicyRule struct {
	ID               string   `json:"id" yaml:"id"`
	SignalType       string   `json:"signal_type,omitempty" yaml:"signal_type"`
	StatusIn         []string `json:"status_in,omitempty" yaml:"status_in"`
	DecisionIn       []string `json:"decision_in,omitempty" yaml:"decision_in"`
	DecisionContains []string `json:"decision_contains,omitempty" yaml:"decision_contains"`
	ReasonContains   []string `json:"reason_contains,omitempty" yaml:"reason_contains"`
	SeverityIn       []string `json:"severity_in,omitempty" yaml:"severity_in"`
	Effect           string   `json:"effect" yaml:"effect"`
	Reason           string   `json:"reason" yaml:"reason"`
	AppendSignalID   bool     `json:"append_signal_id,omitempty" yaml:"append_signal_id"`
}

type AdmissionRuleMatch struct {
	PolicyID   string `json:"policy_id"`
	RuleID     string `json:"rule_id"`
	SignalType string `json:"signal_type,omitempty"`
	SignalID   string `json:"signal_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Decision   string `json:"decision,omitempty"`
	Effect     string `json:"effect"`
	Reason     string `json:"reason"`
}

type ReleaseAdmissionPolicyDecision struct {
	PolicyID         string   `json:"policy_id"`
	PolicyVersion    string   `json:"policy_version,omitempty"`
	PolicySource     string   `json:"policy_source,omitempty"`
	Environment      string   `json:"environment,omitempty"`
	Status           string   `json:"status"`
	Decision         string   `json:"decision"`
	Reasons          []string `json:"reasons"`
	MatchedRuleCount int      `json:"matched_rule_count"`
	Blocked          bool     `json:"blocked"`
	ManualRequired   bool     `json:"manual_required"`
}

type releaseAdmissionPolicyConfigFile struct {
	ReleaseAdmissionPolicyPack ReleaseAdmissionPolicyPack `json:"release_admission_policy_pack" yaml:"release_admission_policy_pack"`
}

func DefaultReleaseAdmissionPolicyPack() ReleaseAdmissionPolicyPack {
	return ReleaseAdmissionPolicyPack{
		ID:                 defaultReleaseAdmissionPolicyID,
		Version:            "2026-05-05",
		Source:             "builtin",
		DefaultEnvironment: "default",
		Environments: map[string]AdmissionEnvironmentPolicy{
			"default": {
				MissingRequiredSignalEffect: "manual",
				MonitorUnknownEffect:        "manual",
			},
			"production": {
				RequireSignalTypes:          []string{"deployment_rehearsal", "monitor_summary"},
				MissingRequiredSignalEffect: "manual",
				MonitorUnknownEffect:        "manual",
			},
		},
		Rules: []AdmissionPolicyRule{
			{ID: "admission_rehearsal_missing", ReasonContains: []string{"deployment_rehearsal_missing"}, Effect: "block", Reason: "deployment_rehearsal_missing"},
			{ID: "rehearsal_blocked", SignalType: "deployment_rehearsal", StatusIn: []string{"blocked"}, Effect: "block", Reason: "rehearsal_blocked"},
			{ID: "deployment_execution_failed", SignalType: "deployment_rehearsal", ReasonContains: []string{"deployment_execution:failed"}, Effect: "block", Reason: "deployment_execution_failed"},
			{ID: "rehearsal_attention_required", SignalType: "deployment_rehearsal", StatusIn: []string{"attention_required"}, Effect: "manual", Reason: "rehearsal_attention_required"},
			{ID: "monitor_critical", SignalType: "monitor_summary", StatusIn: []string{"critical"}, Effect: "block", Reason: "monitor_critical"},
			{ID: "monitor_attention_required", SignalType: "monitor_summary", StatusIn: []string{"attention_required", "unknown"}, Effect: "manual", Reason: "monitor_attention_required"},
			{ID: "rollback_required", SignalType: "rollback_preview", Effect: "manual", Reason: "rollback_required"},
			{ID: "candidate_deployment_risk", SignalType: "candidate_deployment_feedback", StatusIn: []string{"failed", "blocked"}, Effect: "block", Reason: "candidate_deployment_risk"},
			{ID: "candidate_deployment_manual_review", SignalType: "candidate_deployment_feedback", StatusIn: []string{"manual_required", "pending"}, Effect: "manual", Reason: "candidate_deployment_manual_review"},
			{ID: "resource_not_available", SignalType: "resource_status", StatusIn: []string{"disabled", "retired", "expired"}, Effect: "block", Reason: "resource_not_available", AppendSignalID: true},
		},
	}
}

func LoadReleaseAdmissionPolicyPack(rootDir string, environment string) (ReleaseAdmissionPolicyPack, error) {
	pack := DefaultReleaseAdmissionPolicyPack()
	text, found, err := fsutil.ReadText(workspace.ForRoot(rootDir).ReleaseYAML)
	if err != nil || !found {
		return normalizeAdmissionPolicyPack(pack, environment), err
	}
	var raw releaseAdmissionPolicyConfigFile
	if err := yaml.Unmarshal([]byte(text), &raw); err != nil {
		return ReleaseAdmissionPolicyPack{}, err
	}
	custom := raw.ReleaseAdmissionPolicyPack
	if emptyAdmissionPolicyPack(custom) {
		return normalizeAdmissionPolicyPack(pack, environment), nil
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
	if custom.Environments != nil {
		if pack.Environments == nil {
			pack.Environments = map[string]AdmissionEnvironmentPolicy{}
		}
		for key, policy := range custom.Environments {
			pack.Environments[normalizeToken(key)] = normalizeEnvironmentPolicy(policy)
		}
	}
	if len(custom.Rules) > 0 {
		pack.Rules = append(pack.Rules, custom.Rules...)
	}
	pack.Source = "configured"
	return normalizeAdmissionPolicyPack(pack, environment), nil
}

func EvaluateReleaseAdmissionPolicy(pack ReleaseAdmissionPolicyPack, admission ReleaseAdmission) ReleaseAdmission {
	pack = normalizeAdmissionPolicyPack(pack, admission.Environment)
	admission.PolicyID = pack.ID
	admission.PolicyVersion = pack.Version
	admission.PolicySource = pack.Source
	admission.MatchedRules = []AdmissionRuleMatch{}
	envPolicy := environmentPolicy(pack, admission.Environment)
	blocked := false
	manual := false
	reasons := append([]string{}, admission.Reasons...)

	for _, required := range envPolicy.RequireSignalTypes {
		if hasAdmissionSignalType(admission.Signals, required) {
			continue
		}
		effect := normalizePolicyEffect(envPolicy.MissingRequiredSignalEffect)
		if effect == "" {
			effect = "manual"
		}
		reason := "required_signal_missing:" + required
		match := AdmissionRuleMatch{
			PolicyID:   pack.ID,
			RuleID:     "environment_required_signal",
			SignalType: required,
			Effect:     effect,
			Reason:     reason,
		}
		admission.MatchedRules = append(admission.MatchedRules, match)
		reasons = appendUnique(reasons, reason)
		blocked, manual = applyAdmissionEffect(effect, blocked, manual)
	}

	for _, rule := range pack.Rules {
		for _, signal := range admission.Signals {
			if !admissionRuleMatchesSignal(rule, signal) {
				continue
			}
			effect := normalizePolicyEffect(rule.Effect)
			reason := admissionRuleReason(rule, signal)
			match := AdmissionRuleMatch{
				PolicyID:   pack.ID,
				RuleID:     strings.TrimSpace(rule.ID),
				SignalType: signal.Type,
				SignalID:   signal.ID,
				Status:     signal.Status,
				Decision:   signal.Decision,
				Effect:     effect,
				Reason:     reason,
			}
			admission.MatchedRules = append(admission.MatchedRules, match)
			reasons = appendUnique(reasons, reason)
			blocked, manual = applyAdmissionEffect(effect, blocked, manual)
		}
		for _, reason := range admission.Reasons {
			signal := AdmissionSignal{Type: "admission_reason", Status: admission.Status, Decision: admission.Decision, Reason: reason}
			if !admissionRuleMatchesSignal(rule, signal) {
				continue
			}
			effect := normalizePolicyEffect(rule.Effect)
			matchReason := admissionRuleReason(rule, signal)
			match := AdmissionRuleMatch{
				PolicyID:   pack.ID,
				RuleID:     strings.TrimSpace(rule.ID),
				SignalType: signal.Type,
				Status:     signal.Status,
				Decision:   signal.Decision,
				Effect:     effect,
				Reason:     matchReason,
			}
			admission.MatchedRules = append(admission.MatchedRules, match)
			reasons = appendUnique(reasons, matchReason)
			blocked, manual = applyAdmissionEffect(effect, blocked, manual)
		}
	}

	if envPolicy.MonitorUnknownEffect != "" {
		for _, signal := range admission.Signals {
			if signal.Type != "monitor_summary" || signal.Status != "unknown" {
				continue
			}
			effect := normalizePolicyEffect(envPolicy.MonitorUnknownEffect)
			reason := "monitor_unknown_policy:" + effect
			admission.MatchedRules = append(admission.MatchedRules, AdmissionRuleMatch{
				PolicyID:   pack.ID,
				RuleID:     "environment_monitor_unknown",
				SignalType: signal.Type,
				SignalID:   signal.ID,
				Status:     signal.Status,
				Decision:   signal.Decision,
				Effect:     effect,
				Reason:     reason,
			})
			reasons = appendUnique(reasons, reason)
			blocked, manual = applyAdmissionEffect(effect, blocked, manual)
		}
	}

	admission.Reasons = reasons
	if blocked {
		admission.Status = "blocked"
		admission.Decision = "RELEASE_ADMISSION_BLOCKED"
	} else if manual {
		admission.Status = "manual_required"
		admission.Decision = "RELEASE_ADMISSION_MANUAL_REVIEW_REQUIRED"
	} else {
		admission.Status = "allowed"
		admission.Decision = "RELEASE_ADMISSION_ALLOWED"
		admission.Reasons = appendUnique(admission.Reasons, "release_admission_allowed")
	}
	admission.PolicyDecision = ReleaseAdmissionPolicyDecision{
		PolicyID:         pack.ID,
		PolicyVersion:    pack.Version,
		PolicySource:     pack.Source,
		Environment:      admission.Environment,
		Status:           admission.Status,
		Decision:         admission.Decision,
		Reasons:          admission.Reasons,
		MatchedRuleCount: len(admission.MatchedRules),
		Blocked:          admission.Status == "blocked",
		ManualRequired:   admission.Status == "manual_required",
	}
	return admission
}

func emptyAdmissionPolicyPack(pack ReleaseAdmissionPolicyPack) bool {
	return strings.TrimSpace(pack.ID) == "" &&
		strings.TrimSpace(pack.Version) == "" &&
		strings.TrimSpace(pack.DefaultEnvironment) == "" &&
		len(pack.Environments) == 0 &&
		len(pack.Rules) == 0
}

func normalizeAdmissionPolicyPack(pack ReleaseAdmissionPolicyPack, environment string) ReleaseAdmissionPolicyPack {
	if strings.TrimSpace(pack.ID) == "" {
		pack.ID = defaultReleaseAdmissionPolicyID
	}
	if strings.TrimSpace(pack.Version) == "" {
		pack.Version = "2026-05-05"
	}
	if strings.TrimSpace(pack.Source) == "" {
		pack.Source = "builtin"
	}
	pack.DefaultEnvironment = normalizeToken(pack.DefaultEnvironment)
	if pack.DefaultEnvironment == "" {
		pack.DefaultEnvironment = "default"
	}
	if pack.Environments == nil {
		pack.Environments = map[string]AdmissionEnvironmentPolicy{}
	}
	for key, policy := range pack.Environments {
		delete(pack.Environments, key)
		pack.Environments[normalizeToken(key)] = normalizeEnvironmentPolicy(policy)
	}
	for index := range pack.Rules {
		pack.Rules[index].ID = strings.TrimSpace(pack.Rules[index].ID)
		pack.Rules[index].SignalType = normalizeToken(pack.Rules[index].SignalType)
		pack.Rules[index].StatusIn = normalizeLowerList(pack.Rules[index].StatusIn)
		pack.Rules[index].DecisionIn = normalizeUpperList(pack.Rules[index].DecisionIn)
		pack.Rules[index].DecisionContains = normalizeLowerList(pack.Rules[index].DecisionContains)
		pack.Rules[index].ReasonContains = normalizeLowerList(pack.Rules[index].ReasonContains)
		pack.Rules[index].SeverityIn = normalizeLowerList(pack.Rules[index].SeverityIn)
		pack.Rules[index].Effect = normalizePolicyEffect(pack.Rules[index].Effect)
		if pack.Rules[index].Effect == "" {
			pack.Rules[index].Effect = "manual"
		}
		pack.Rules[index].Reason = normalizeToken(pack.Rules[index].Reason)
		if pack.Rules[index].Reason == "" {
			pack.Rules[index].Reason = pack.Rules[index].ID
		}
	}
	if environment != "" {
		normalized := normalizeToken(environment)
		if _, ok := pack.Environments[normalized]; !ok {
			pack.Environments[normalized] = environmentPolicy(pack, normalized)
		}
	}
	return pack
}

func normalizeEnvironmentPolicy(policy AdmissionEnvironmentPolicy) AdmissionEnvironmentPolicy {
	policy.RequireSignalTypes = normalizeLowerList(policy.RequireSignalTypes)
	policy.MissingRequiredSignalEffect = normalizePolicyEffect(policy.MissingRequiredSignalEffect)
	policy.MonitorUnknownEffect = normalizePolicyEffect(policy.MonitorUnknownEffect)
	return policy
}

func environmentPolicy(pack ReleaseAdmissionPolicyPack, environment string) AdmissionEnvironmentPolicy {
	environment = normalizeToken(environment)
	if environment == "" {
		environment = pack.DefaultEnvironment
	}
	if policy, ok := pack.Environments[environment]; ok {
		return normalizeEnvironmentPolicy(policy)
	}
	if policy, ok := pack.Environments[pack.DefaultEnvironment]; ok {
		return normalizeEnvironmentPolicy(policy)
	}
	return AdmissionEnvironmentPolicy{MissingRequiredSignalEffect: "manual", MonitorUnknownEffect: "manual"}
}

func admissionRuleMatchesSignal(rule AdmissionPolicyRule, signal AdmissionSignal) bool {
	if rule.SignalType != "" && rule.SignalType != normalizeToken(signal.Type) {
		return false
	}
	if len(rule.StatusIn) > 0 && !containsNormalized(rule.StatusIn, normalizeToken(signal.Status)) {
		return false
	}
	if len(rule.DecisionIn) > 0 && !containsNormalized(rule.DecisionIn, strings.ToUpper(strings.TrimSpace(signal.Decision))) {
		return false
	}
	if len(rule.DecisionContains) > 0 && !containsAny(strings.ToLower(signal.Decision), rule.DecisionContains) {
		return false
	}
	if len(rule.ReasonContains) > 0 && !containsAny(strings.ToLower(signal.Reason), rule.ReasonContains) {
		return false
	}
	if len(rule.SeverityIn) > 0 && !containsNormalized(rule.SeverityIn, normalizeToken(signal.Severity)) {
		return false
	}
	return true
}

func admissionRuleReason(rule AdmissionPolicyRule, signal AdmissionSignal) string {
	reason := normalizeToken(rule.Reason)
	if reason == "" {
		reason = normalizeToken(rule.ID)
	}
	if rule.AppendSignalID && strings.TrimSpace(signal.ID) != "" {
		return reason + ":" + strings.TrimSpace(signal.ID)
	}
	return reason
}

func applyAdmissionEffect(effect string, blocked bool, manual bool) (bool, bool) {
	switch normalizePolicyEffect(effect) {
	case "block":
		return true, manual
	case "manual":
		return blocked, true
	default:
		return blocked, manual
	}
}

func normalizePolicyEffect(effect string) string {
	switch normalizeToken(effect) {
	case "blocked", "block":
		return "block"
	case "manual_required", "manual", "review":
		return "manual"
	case "allowed", "allow", "record":
		return "allow"
	default:
		return ""
	}
}

func hasAdmissionSignalType(signals []AdmissionSignal, signalType string) bool {
	signalType = normalizeToken(signalType)
	for _, signal := range signals {
		if normalizeToken(signal.Type) == signalType {
			return true
		}
	}
	return false
}

func normalizeLowerList(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = normalizeToken(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizeUpperList(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func containsNormalized(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if needle != "" && strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
