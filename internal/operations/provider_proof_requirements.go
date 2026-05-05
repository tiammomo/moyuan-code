package operations

import (
	"fmt"
	"sort"
	"time"
)

const (
	defaultProviderProofPolicyID      = "provider-proof-requirements-v1"
	defaultProviderProofPolicyVersion = "2026-05-05"
)

type ProviderProofRequirementOptions struct {
	Provider      string `json:"provider,omitempty"`
	OperationType string `json:"operation_type,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

type ProviderProofRequirementReport struct {
	ID            string                          `json:"id"`
	GeneratedAt   string                          `json:"generated_at"`
	PolicyID      string                          `json:"policy_id"`
	PolicyVersion string                          `json:"policy_version"`
	Filters       ProviderProofRequirementOptions `json:"filters"`
	Summary       ProviderProofRequirementSummary `json:"summary"`
	Requirements  []ProviderProofRequirement      `json:"requirements"`
}

type ProviderProofRequirementSummary struct {
	RequirementCount int            `json:"requirement_count"`
	ByProvider       map[string]int `json:"by_provider,omitempty"`
	ByOperationType  map[string]int `json:"by_operation_type,omitempty"`
}

type ProviderProofRequirement struct {
	ID                       string   `json:"id"`
	Provider                 string   `json:"provider"`
	OperationType            string   `json:"operation_type"`
	Status                   string   `json:"status"`
	Decision                 string   `json:"decision"`
	RequiredSecretRefStatus  string   `json:"required_secret_ref_status,omitempty"`
	RequireEvidence          bool     `json:"require_evidence"`
	RequireApproval          bool     `json:"require_approval"`
	RequireWriteSwitch       bool     `json:"require_write_switch"`
	ProductionReviewRequired bool     `json:"production_review_required"`
	LeastPrivilegeScopes     []string `json:"least_privilege_scopes,omitempty"`
	ReplayGuard              string   `json:"replay_guard,omitempty"`
	RuleRefs                 []string `json:"rule_refs,omitempty"`
}

func BuildProviderProofRequirements(rootDir string, options ProviderProofRequirementOptions) (ProviderProofRequirementReport, error) {
	_ = rootDir
	options = normalizeProviderProofRequirementOptions(options)
	requirements := []ProviderProofRequirement{}
	for _, requirement := range defaultProviderProofRequirements() {
		requirement = normalizeProviderProofRequirement(requirement)
		if providerProofRequirementMatches(requirement, options) {
			requirements = append(requirements, requirement)
		}
	}
	sort.SliceStable(requirements, func(i, j int) bool {
		return requirements[i].Provider+"|"+requirements[i].OperationType < requirements[j].Provider+"|"+requirements[j].OperationType
	})
	if len(requirements) > options.Limit {
		requirements = requirements[:options.Limit]
	}
	now := time.Now().UTC()
	report := ProviderProofRequirementReport{
		ID:            "provider-proof-requirements-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt:   now.Format(time.RFC3339Nano),
		PolicyID:      defaultProviderProofPolicyID,
		PolicyVersion: defaultProviderProofPolicyVersion,
		Filters:       options,
		Requirements:  requirements,
	}
	report.Summary = buildProviderProofRequirementSummary(requirements)
	return report, nil
}

func providerRequirementFor(provider string, operationType string) (ProviderProofRequirement, bool) {
	provider = normalizeProviderAlias(normalizeType(provider), normalizeType(operationType))
	operationType = normalizeType(operationType)
	for _, requirement := range defaultProviderProofRequirements() {
		requirement = normalizeProviderProofRequirement(requirement)
		if requirement.Provider == provider && requirement.OperationType == operationType {
			return requirement, true
		}
	}
	switch operationType {
	case "release_provider_execution":
		return providerRequirementFor("generic_git", operationType)
	case "deployment_execution":
		return providerRequirementFor("local", operationType)
	case "resource_maintenance":
		return providerRequirementFor("local_registry", operationType)
	default:
		return ProviderProofRequirement{}, false
	}
}

func normalizeProviderProofRequirementOptions(options ProviderProofRequirementOptions) ProviderProofRequirementOptions {
	options.OperationType = normalizeType(options.OperationType)
	options.Provider = normalizeProviderAlias(normalizeType(options.Provider), options.OperationType)
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	return options
}

func normalizeProviderProofRequirement(requirement ProviderProofRequirement) ProviderProofRequirement {
	requirement.Provider = normalizeProviderAlias(normalizeType(requirement.Provider), normalizeType(requirement.OperationType))
	requirement.OperationType = normalizeType(requirement.OperationType)
	requirement.Status = firstNonEmpty(normalizeType(requirement.Status), "supported")
	requirement.Decision = firstNonEmpty(requirement.Decision, "PROVIDER_PROOF_REQUIREMENT_SUPPORTED")
	requirement.RequiredSecretRefStatus = normalizeType(requirement.RequiredSecretRefStatus)
	requirement.LeastPrivilegeScopes = compactStrings(requirement.LeastPrivilegeScopes)
	requirement.RuleRefs = compactStrings(requirement.RuleRefs)
	if requirement.ID == "" {
		requirement.ID = "provider-proof-" + requirement.Provider + "-" + requirement.OperationType
	}
	return requirement
}

func normalizeProviderAlias(provider string, operationType string) string {
	provider = normalizeType(provider)
	operationType = normalizeType(operationType)
	switch provider {
	case "":
		switch operationType {
		case "release_provider_execution":
			return "generic_git"
		case "deployment_execution":
			return "local"
		case "resource_maintenance":
			return "local_registry"
		default:
			return ""
		}
	case "github_enterprise":
		return "github"
	case "gitee_enterprise":
		return "gitee"
	case "local_vm":
		if operationType == "resource_maintenance" {
			return "local_registry"
		}
		return "ssh"
	case "local_registry":
		return "local_registry"
	default:
		return provider
	}
}

func providerProofRequirementMatches(requirement ProviderProofRequirement, options ProviderProofRequirementOptions) bool {
	if options.Provider != "" && requirement.Provider != options.Provider {
		return false
	}
	if options.OperationType != "" && requirement.OperationType != options.OperationType {
		return false
	}
	return true
}

func buildProviderProofRequirementSummary(requirements []ProviderProofRequirement) ProviderProofRequirementSummary {
	summary := ProviderProofRequirementSummary{
		RequirementCount: len(requirements),
		ByProvider:       map[string]int{},
		ByOperationType:  map[string]int{},
	}
	for _, requirement := range requirements {
		summary.ByProvider[requirement.Provider]++
		summary.ByOperationType[requirement.OperationType]++
	}
	return summary
}

func defaultProviderProofRequirements() []ProviderProofRequirement {
	return []ProviderProofRequirement{
		{
			Provider:                 "generic_git",
			OperationType:            "release_provider_execution",
			RequiredSecretRefStatus:  "referenced_by_provider_config",
			RequireEvidence:          true,
			RequireApproval:          true,
			RequireWriteSwitch:       true,
			ProductionReviewRequired: true,
			LeastPrivilegeScopes:     []string{"git:branch:write", "git:tag:write", "release:write", "workflow:dispatch"},
			ReplayGuard:              "release_provider_execution_id_and_approval_consumption",
			RuleRefs:                 []string{"provider_requirement:git_minimum_scope", "provider_requirement:release_evidence", "provider_requirement:release_replay_guard"},
		},
		{
			Provider:                 "github",
			OperationType:            "release_provider_execution",
			RequiredSecretRefStatus:  "referenced_by_provider_config",
			RequireEvidence:          true,
			RequireApproval:          true,
			RequireWriteSwitch:       true,
			ProductionReviewRequired: true,
			LeastPrivilegeScopes:     []string{"contents:write", "metadata:read", "pull_requests:read", "actions:write"},
			ReplayGuard:              "release_provider_execution_id_and_approval_consumption",
			RuleRefs:                 []string{"provider_requirement:github_token_scope", "provider_requirement:release_evidence", "provider_requirement:release_replay_guard"},
		},
		{
			Provider:                 "gitee",
			OperationType:            "release_provider_execution",
			RequiredSecretRefStatus:  "referenced_by_provider_config",
			RequireEvidence:          true,
			RequireApproval:          true,
			RequireWriteSwitch:       true,
			ProductionReviewRequired: true,
			LeastPrivilegeScopes:     []string{"repo:write", "pull_request:read", "release:write", "pipeline:trigger"},
			ReplayGuard:              "release_provider_execution_id_and_approval_consumption",
			RuleRefs:                 []string{"provider_requirement:gitee_token_scope", "provider_requirement:release_evidence", "provider_requirement:release_replay_guard"},
		},
		{
			Provider:                "local",
			OperationType:           "deployment_execution",
			RequiredSecretRefStatus: "not_applicable",
			RequireEvidence:         true,
			RequireApproval:         true,
			RequireWriteSwitch:      true,
			LeastPrivilegeScopes:    []string{"local_shell:safe_command_allowlist"},
			ReplayGuard:             "deployment_execution_id_and_approval_consumption",
			RuleRefs:                []string{"provider_requirement:local_command_allowlist", "provider_requirement:deployment_evidence", "provider_requirement:deployment_replay_guard"},
		},
		{
			Provider:                "ssh",
			OperationType:           "deployment_execution",
			RequiredSecretRefStatus: "referenced",
			RequireEvidence:         true,
			RequireApproval:         true,
			RequireWriteSwitch:      true,
			LeastPrivilegeScopes:    []string{"ssh:command:allowlist", "server:auth_ref:read"},
			ReplayGuard:             "deployment_execution_id_and_approval_consumption",
			RuleRefs:                []string{"provider_requirement:ssh_auth_ref", "provider_requirement:ssh_command_allowlist", "provider_requirement:deployment_evidence", "provider_requirement:deployment_replay_guard"},
		},
		{
			Provider:                "cloud",
			OperationType:           "deployment_execution",
			RequiredSecretRefStatus: "referenced",
			RequireEvidence:         true,
			RequireApproval:         true,
			RequireWriteSwitch:      true,
			LeastPrivilegeScopes:    []string{"cloud:deployment:write", "cloud:instance:read"},
			ReplayGuard:             "deployment_execution_id_and_approval_consumption",
			RuleRefs:                []string{"provider_requirement:cloud_auth_ref", "provider_requirement:cloud_least_privilege", "provider_requirement:deployment_evidence", "provider_requirement:deployment_replay_guard"},
		},
		{
			Provider:                "aliyun",
			OperationType:           "resource_maintenance",
			RequiredSecretRefStatus: "referenced",
			RequireEvidence:         false,
			RequireApproval:         true,
			RequireWriteSwitch:      true,
			LeastPrivilegeScopes:    []string{"ecs:instance:read", "ecs:metadata:write"},
			ReplayGuard:             "resource_id_and_maintenance_record_id",
			RuleRefs:                []string{"provider_requirement:cloud_resource_scope", "provider_requirement:resource_replay_guard"},
		},
		{
			Provider:                "tencent_cloud",
			OperationType:           "resource_maintenance",
			RequiredSecretRefStatus: "referenced",
			RequireEvidence:         false,
			RequireApproval:         true,
			RequireWriteSwitch:      true,
			LeastPrivilegeScopes:    []string{"cvm:instance:read", "cvm:metadata:write"},
			ReplayGuard:             "resource_id_and_maintenance_record_id",
			RuleRefs:                []string{"provider_requirement:cloud_resource_scope", "provider_requirement:resource_replay_guard"},
		},
		{
			Provider:                "local_registry",
			OperationType:           "resource_maintenance",
			RequiredSecretRefStatus: "not_applicable",
			RequireEvidence:         false,
			RequireApproval:         false,
			RequireWriteSwitch:      false,
			LeastPrivilegeScopes:    []string{"resource_registry:write"},
			ReplayGuard:             "resource_id_and_maintenance_record_id",
			RuleRefs:                []string{"provider_requirement:resource_registry_write_only", "provider_requirement:resource_replay_guard"},
		},
	}
}
