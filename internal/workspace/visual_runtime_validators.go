package workspace

import (
	"fmt"
	"strings"

	"moyuan-code/internal/fsutil"

	"gopkg.in/yaml.v3"
)

func validateVisualsConfigFile(report *ValidationReport, path string) {
	text, found, err := fsutil.ReadText(path)
	if err != nil {
		report.add("error", "visuals_config_unreadable", err.Error(), path)
		return
	}
	if !found {
		return
	}
	if providersConfigContainsPlaintextSecret(text) || strings.Contains(strings.ToLower(text), ".env") {
		report.add("error", "visuals_plaintext_secret_forbidden", "visuals config must not contain plaintext secrets, tokens, or .env content", path)
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(text), &raw); err != nil {
		report.add("error", "visuals_config_unreadable", err.Error(), path)
		return
	}
	if intField(raw, "schema_version") != 1 {
		report.add("error", "visuals_schema_version_invalid", "visuals schema_version must be 1", "visuals/architecture-visuals.yaml:schema_version")
	}
	architecture := mapField(raw, "architecture_visuals")
	if architecture == nil {
		report.add("error", "visuals_architecture_required", "architecture_visuals is required", "architecture_visuals")
	} else if _, ok := architecture["enabled"]; !ok {
		report.add("error", "visuals_enabled_required", "architecture_visuals.enabled is required", "architecture_visuals.enabled")
	}
	providerPolicy := mapField(raw, "provider_policy")
	if providerPolicy == nil {
		report.add("error", "visuals_provider_policy_required", "provider_policy is required", "provider_policy")
	} else {
		if strings.TrimSpace(mapString(providerPolicy, "diagram_planning")) == "" && mapField(providerPolicy, "diagram_planning") == nil {
			report.add("error", "visuals_diagram_planning_required", "provider_policy.diagram_planning is required", "provider_policy.diagram_planning")
		}
		if strings.TrimSpace(mapString(providerPolicy, "image_generation")) == "" && mapField(providerPolicy, "image_generation") == nil {
			report.add("error", "visuals_image_generation_required", "provider_policy.image_generation is required", "provider_policy.image_generation")
		}
	}
	output := mapField(raw, "output")
	if output == nil || strings.TrimSpace(mapString(output, "base_dir")) == "" {
		report.add("error", "visuals_output_base_dir_required", "output.base_dir is required", "output.base_dir")
	}
	if len(arrayField(raw, "diagram_types")) == 0 {
		report.add("error", "visuals_diagram_types_required", "diagram_types must contain at least one diagram type", "diagram_types")
	}
	pipeline := mapField(raw, "pipeline")
	if pipeline == nil || len(arrayField(pipeline, "steps")) == 0 {
		report.add("error", "visuals_pipeline_steps_required", "pipeline.steps must contain at least one step", "pipeline.steps")
	}
	diagramSpec := mapField(raw, "diagram_spec")
	if diagramSpec == nil || len(arrayField(diagramSpec, "required_fields")) == 0 {
		report.add("error", "visuals_diagram_spec_required_fields_required", "diagram_spec.required_fields must contain at least one field", "diagram_spec.required_fields")
	}
	gptImage := mapField(raw, "gpt_image_2")
	if gptImage == nil || strings.TrimSpace(mapString(gptImage, "model")) == "" {
		report.add("error", "visuals_gpt_image_2_model_required", "gpt_image_2.model is required", "gpt_image_2.model")
	}
	if promptTemplate, ok := gptImage["prompt_template"]; ok && strings.TrimSpace(fmt.Sprint(promptTemplate)) == "" {
		report.add("error", "visuals_prompt_template_empty", "gpt_image_2.prompt_template must be omitted or non-empty", "gpt_image_2.prompt_template")
	}
	safety := mapField(raw, "safety")
	if safety == nil || !mapBool(safety, "strip_secrets") {
		report.add("error", "visuals_strip_secrets_required", "safety.strip_secrets must be true", "safety.strip_secrets")
	}
}

func validateAgentRuntimesConfigFile(report *ValidationReport, path string) {
	text, found, err := fsutil.ReadText(path)
	if err != nil {
		report.add("error", "agent_runtimes_config_unreadable", err.Error(), path)
		return
	}
	if !found {
		return
	}
	if providersConfigContainsPlaintextSecret(text) {
		report.add("error", "agent_runtimes_plaintext_secret_forbidden", "agent-runtimes.yaml must not contain plaintext API keys or tokens", path)
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(text), &raw); err != nil {
		report.add("error", "agent_runtimes_config_unreadable", err.Error(), path)
		return
	}
	if intField(raw, "schema_version") != 1 {
		report.add("error", "agent_runtimes_schema_version_invalid", "agent-runtimes.yaml schema_version must be 1", "runtimes/agent-runtimes.yaml:schema_version")
	}
	cfg := mapField(raw, "agent_runtimes")
	if cfg == nil {
		report.add("error", "agent_runtimes_required", "agent_runtimes is required", "agent_runtimes")
		return
	}
	if _, ok := cfg["enabled"]; !ok {
		report.add("error", "agent_runtimes_enabled_required", "agent_runtimes.enabled is required", "agent_runtimes.enabled")
	}
	if strings.TrimSpace(mapString(cfg, "default_runtime")) == "" {
		report.add("error", "agent_runtimes_default_runtime_required", "agent_runtimes.default_runtime is required", "agent_runtimes.default_runtime")
	}
	if strings.TrimSpace(mapString(cfg, "session_store")) == "" {
		report.add("error", "agent_runtimes_session_store_required", "agent_runtimes.session_store is required", "agent_runtimes.session_store")
	}
	if strings.TrimSpace(mapString(cfg, "output_store")) == "" {
		report.add("error", "agent_runtimes_output_store_required", "agent_runtimes.output_store is required", "agent_runtimes.output_store")
	}
	for index, item := range arrayField(cfg, "runtimes") {
		runtime, ok := item.(map[string]any)
		if !ok {
			report.add("error", "agent_runtime_invalid", "runtime entry must be an object", fmt.Sprintf("agent_runtimes.runtimes[%d]", index))
			continue
		}
		validateAgentRuntimeEntry(report, runtime, fmt.Sprintf("agent_runtimes.runtimes[%d]", index))
	}
	if len(arrayField(cfg, "runtimes")) == 0 {
		report.add("error", "agent_runtimes_entries_required", "agent_runtimes.runtimes must contain at least one runtime", "agent_runtimes.runtimes")
	}
	roleDefaults := mapField(cfg, "role_runtime_defaults")
	if roleDefaults == nil {
		report.add("error", "agent_runtimes_role_defaults_required", "agent_runtimes.role_runtime_defaults is required", "agent_runtimes.role_runtime_defaults")
	} else {
		for _, role := range []string{"frontend", "backend", "backend_tuning"} {
			if strings.TrimSpace(mapString(roleDefaults, role)) == "" {
				report.add("error", "agent_runtimes_role_default_required", "role runtime default is required", "agent_runtimes.role_runtime_defaults."+role)
			}
		}
	}
	isolation := mapField(cfg, "isolation")
	if isolation == nil || !mapBool(isolation, "require_issue_worktree") {
		report.add("error", "agent_runtimes_issue_worktree_required", "agent_runtimes.isolation.require_issue_worktree must be true", "agent_runtimes.isolation.require_issue_worktree")
	}
	if !mapBool(cfg, "require_quality_gate_after_run") {
		report.add("error", "agent_runtimes_quality_gate_required", "agent_runtimes.require_quality_gate_after_run must be true", "agent_runtimes.require_quality_gate_after_run")
	}
}

func validateAgentRuntimeEntry(report *ValidationReport, runtime map[string]any, path string) {
	if strings.TrimSpace(mapString(runtime, "type")) == "" {
		report.add("error", "agent_runtime_type_required", "runtime.type is required", path+".type")
	}
	provider := strings.TrimSpace(mapString(runtime, "provider"))
	if provider == "" {
		report.add("error", "agent_runtime_provider_required", "runtime.provider is required", path+".provider")
	}
	if _, ok := runtime["enabled"]; !ok {
		report.add("error", "agent_runtime_enabled_required", "runtime.enabled is required", path+".enabled")
	}
	if strings.TrimSpace(mapString(runtime, "command")) == "" {
		report.add("error", "agent_runtime_command_required", "runtime.command is required", path+".command")
	}
	auth := mapField(runtime, "auth")
	if auth == nil || strings.TrimSpace(mapString(auth, "mode")) == "" {
		report.add("error", "agent_runtime_auth_mode_required", "runtime.auth.mode is required", path+".auth.mode")
	}
	profile := mapField(runtime, "provider_env_profile")
	if profile != nil && !mapBool(profile, "enabled") && len(arrayField(profile, "allowed_env_keys")) > 0 {
		report.add("error", "agent_runtime_env_keys_must_be_empty", "provider_env_profile.allowed_env_keys must be empty when profile is disabled", path+".provider_env_profile.allowed_env_keys")
	}
	if profile != nil && mapBool(profile, "enabled") && len(arrayField(profile, "allowed_env_keys")) == 0 {
		report.add("error", "agent_runtime_env_keys_required", "provider_env_profile.allowed_env_keys is required when profile is enabled", path+".provider_env_profile.allowed_env_keys")
	}
	health := mapField(runtime, "health_check")
	if health == nil || strings.TrimSpace(mapString(health, "command")) == "" {
		report.add("error", "agent_runtime_health_command_required", "runtime.health_check.command is required", path+".health_check.command")
	}
	for _, field := range []string{"invocation", "context", "tools", "session", "audit"} {
		if mapField(runtime, field) == nil {
			report.add("error", "agent_runtime_section_required", "runtime."+field+" is required", path+"."+field)
		}
	}
	audit := mapField(runtime, "audit")
	if audit != nil && !mapBool(audit, "capture_diff_before_after") {
		report.add("error", "agent_runtime_capture_diff_required", "runtime.audit.capture_diff_before_after must be true", path+".audit.capture_diff_before_after")
	}
	invocation := mapField(runtime, "invocation")
	if provider == "claude_code" && invocation != nil && strings.TrimSpace(mapString(invocation, "ask")) != "" {
		report.add("error", "agent_runtime_ask_must_be_empty_for_claude", "invocation.ask must be empty for claude_code provider", path+".invocation.ask")
	}
	if provider == "codex" && invocation != nil && strings.TrimSpace(mapString(invocation, "one_shot")) != "" {
		report.add("error", "agent_runtime_one_shot_must_be_empty_for_codex", "invocation.one_shot must be empty for codex provider", path+".invocation.one_shot")
	}
}
