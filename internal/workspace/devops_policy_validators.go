package workspace

import (
	"fmt"
	"strings"

	"moyuan-code/internal/fsutil"

	"gopkg.in/yaml.v3"
)

func validateServerResourcesConfigFile(report *ValidationReport, path string) {
	raw, found := readOptionalYAMLMap(report, path, "server_resources_config_unreadable")
	if !found {
		return
	}
	if intField(raw, "schema_version") != 1 {
		report.add("error", "server_resources_schema_version_invalid", "server-resources.yaml schema_version must be 1", "policies/server-resources.yaml:schema_version")
	}
	cfg := mapField(raw, "server_resources")
	if cfg == nil {
		report.add("error", "server_resources_required", "server_resources is required", "server_resources")
		return
	}
	if _, ok := cfg["enabled"]; !ok {
		report.add("error", "server_resources_enabled_required", "server_resources.enabled is required", "server_resources.enabled")
	}
	if mapBool(cfg, "enabled") {
		for _, field := range []string{"registry", "categories", "access_policy", "inventory_checks"} {
			if !valuePresent(raw, field) {
				report.add("error", "server_resources_section_required", field+" is required when server_resources.enabled=true", field)
			}
		}
	}
	for index, item := range arrayField(raw, "hosts") {
		host, ok := item.(map[string]any)
		if !ok {
			report.add("error", "server_resource_host_invalid", "host must be an object", fmt.Sprintf("hosts[%d]", index))
			continue
		}
		validateServerResourceHost(report, host, fmt.Sprintf("hosts[%d]", index))
	}
}

func validateServerResourceHost(report *ValidationReport, host map[string]any, path string) {
	category := strings.ToLower(strings.TrimSpace(firstNonEmpty(mapString(host, "category"), mapString(host, "environment"))))
	if category != "production" {
		return
	}
	if strings.TrimSpace(mapString(host, "owner")) == "" {
		report.add("error", "production_host_owner_required", "production host owner is required", path+".owner")
	}
	authRef := strings.TrimSpace(mapString(host, "auth_ref"))
	if authRef == "" {
		report.add("error", "production_host_auth_ref_required", "production host auth_ref is required", path+".auth_ref")
	} else if !isConfigReference(authRef) {
		report.add("error", "production_host_auth_ref_must_be_reference", "production host auth_ref must use env: or secret: reference", path+".auth_ref")
	}
	lifecycle := mapField(host, "lifecycle")
	if lifecycle == nil || strings.TrimSpace(mapString(lifecycle, "expires_at")) == "" {
		report.add("error", "production_host_expires_at_required", "production host lifecycle.expires_at is required", path+".lifecycle.expires_at")
	}
}

func validateEnvironmentsConfigFile(report *ValidationReport, path string) {
	raw, found := readOptionalYAMLMap(report, path, "environments_config_unreadable")
	if !found {
		return
	}
	if intField(raw, "schema_version") != 1 {
		report.add("error", "environments_schema_version_invalid", "environments.yaml schema_version must be 1", "policies/environments.yaml:schema_version")
	}
	environments := mapField(raw, "environments")
	for name, value := range environments {
		env, ok := value.(map[string]any)
		if !ok {
			report.add("error", "environment_entry_invalid", "environment must be an object", "environments."+name)
			continue
		}
		validateEnvironmentEntry(report, name, env)
	}
}

func validateEnvironmentEntry(report *ValidationReport, name string, env map[string]any) {
	path := "environments." + name
	if _, ok := env["approval_required"]; !ok {
		report.add("error", "environment_approval_required_field_required", "environment.approval_required is required", path+".approval_required")
	}
	if strings.TrimSpace(mapString(env, "resource_group")) == "" && hasDeploymentSections(env) {
		report.add("error", "environment_resource_group_required", "environment.resource_group is required for deployment environments", path+".resource_group")
	}
	if hasDeploymentSections(env) {
		for _, field := range []string{"artifact", "deploy", "healthcheck"} {
			if mapField(env, field) == nil {
				report.add("error", "environment_deploy_section_required", "environment."+field+" is required for automatic deployment", path+"."+field)
			}
		}
	}
	if name == "production" {
		if !mapBool(env, "approval_required") {
			report.add("error", "production_approval_required", "production.approval_required must be true", path+".approval_required")
		}
		if len(arrayField(env, "smoke_tests")) == 0 {
			report.add("error", "production_smoke_tests_required", "production.smoke_tests must not be empty", path+".smoke_tests")
		}
		if mapField(env, "rollback") == nil {
			report.add("error", "production_rollback_required", "production.rollback must not be empty", path+".rollback")
		}
	}
}

func validateReleaseConfigFile(report *ValidationReport, path string) {
	raw, found := readOptionalYAMLMap(report, path, "release_config_unreadable")
	if !found {
		return
	}
	if intField(raw, "schema_version") != 1 {
		report.add("error", "release_schema_version_invalid", "release.yaml schema_version must be 1", "policies/release.yaml:schema_version")
	}
	release := mapField(raw, "release")
	if release == nil {
		report.add("error", "release_required", "release is required", "release")
		return
	}
	if _, ok := release["auto_suggest"]; !ok {
		report.add("error", "release_auto_suggest_required", "release.auto_suggest is required", "release.auto_suggest")
	}
	mode := strings.TrimSpace(mapString(release, "mode"))
	if mode == "" {
		report.add("error", "release_mode_required", "release.mode is required", "release.mode")
	}
	if len(arrayField(release, "remote_providers")) == 0 {
		report.add("error", "release_remote_providers_required", "release.remote_providers must contain at least one provider", "release.remote_providers")
	}
	if mapField(release, "default_batch") == nil {
		report.add("error", "release_default_batch_required", "release.default_batch is required", "release.default_batch")
	}
	gates := mapField(release, "gates")
	if gates == nil {
		report.add("error", "release_gates_required", "release.gates is required", "release.gates")
	} else {
		for _, field := range []string{"require_release_note", "require_coverage_passed", "require_rollback_plan"} {
			if !mapBool(gates, field) {
				report.add("error", "release_gate_required", "release.gates."+field+" must be true", "release.gates."+field)
			}
		}
	}
	if mapField(release, "git") == nil {
		report.add("error", "release_git_required", "release.git is required", "release.git")
	}
	deployment := mapField(release, "deployment")
	if mode == "deploy_to_environment" && deployment == nil {
		report.add("error", "release_deployment_required", "release.deployment is required when mode=deploy_to_environment", "release.deployment")
	}
	if mode == "branch_only" && deployment != nil && mapBool(deployment, "enabled") {
		report.add("error", "release_deployment_must_be_disabled", "release.deployment.enabled must be false when mode=branch_only", "release.deployment.enabled")
	}
}

func validateBudgetConfigFile(report *ValidationReport, path string) {
	raw, found := readOptionalYAMLMap(report, path, "budget_config_unreadable")
	if !found {
		return
	}
	if intField(raw, "schema_version") != 1 {
		report.add("error", "budget_schema_version_invalid", "budget.yaml schema_version must be 1", "policies/budget.yaml:schema_version")
	}
	budget := mapField(raw, "budget")
	if budget == nil {
		report.add("error", "budget_required", "budget is required", "budget")
		return
	}
	for _, field := range []string{"max_parallel_issues", "max_parallel_model_calls", "max_task_runtime_minutes"} {
		if intField(budget, field) <= 0 {
			report.add("error", "budget_positive_integer_required", "budget."+field+" must be greater than 0", "budget."+field)
		}
	}
	if _, ok := budget["fallback_to_low_cost_model"]; !ok {
		report.add("error", "budget_fallback_required", "budget.fallback_to_low_cost_model is required", "budget.fallback_to_low_cost_model")
	}
	if value, ok := budget["max_daily_model_cost_usd"]; ok && numericNotPositive(value) {
		report.add("error", "budget_daily_cost_must_be_null_or_positive", "budget.max_daily_model_cost_usd must be null or greater than 0", "budget.max_daily_model_cost_usd")
	}
}

func readOptionalYAMLMap(report *ValidationReport, path string, code string) (map[string]any, bool) {
	text, found, err := fsutil.ReadText(path)
	if err != nil {
		report.add("error", code, err.Error(), path)
		return nil, false
	}
	if !found {
		return nil, false
	}
	if providersConfigContainsPlaintextSecret(text) {
		report.add("error", code, "config must not contain plaintext API keys or tokens", path)
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(text), &raw); err != nil {
		report.add("error", code, err.Error(), path)
		return nil, false
	}
	return raw, true
}

func hasDeploymentSections(env map[string]any) bool {
	return mapField(env, "artifact") != nil || mapField(env, "deploy") != nil || mapField(env, "healthcheck") != nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func valuePresent(values map[string]any, key string) bool {
	value, ok := values[key]
	if !ok || value == nil {
		return false
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	return true
}

func numericNotPositive(value any) bool {
	switch typed := value.(type) {
	case int:
		return typed <= 0
	case int64:
		return typed <= 0
	case float64:
		return typed <= 0
	default:
		return false
	}
}
