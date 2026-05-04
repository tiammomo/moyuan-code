package workspace

import (
	"path/filepath"
	"testing"

	"moyuan-code/internal/fsutil"
)

func TestValidateReportsWorkspaceSchemaStatus(t *testing.T) {
	root := t.TempDir()
	if _, err := Ensure(root); err != nil {
		t.Fatal(err)
	}

	report, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || len(report.Issues) != 0 {
		t.Fatalf("expected valid workspace, got %+v", report)
	}

	if err := fsutil.WriteJSON(filepath.Join(root, DirName, "workspace.json"), map[string]any{
		"schema_ver": 1,
		"project": map[string]any{
			"schema_version": 1,
			"project": map[string]any{
				"name": "missing-id",
				"root": ".",
				"type": "single-repo",
			},
		},
		"repository": DefaultRepositoryConfig(root),
		"access":     DefaultAccessConfig(),
	}); err != nil {
		t.Fatal(err)
	}

	report, err = Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" || !hasValidationIssue(report, "project_id_required") {
		t.Fatalf("expected project_id_required validation failure, got %+v", report)
	}
}

func TestLoadPrefersEditableYAMLConfig(t *testing.T) {
	root := t.TempDir()
	if _, err := Ensure(root); err != nil {
		t.Fatal(err)
	}

	if err := fsutil.WriteText(filepath.Join(root, DirName, "project.yaml"), `schema_version: 1
project:
  id: yaml-project
  name: YAML Project
  root: "."
  type: single-repo
  description: null
stack:
  languages:
    - go
  frameworks: []
  package_managers: []
  build_commands: []
  test_commands:
    - go test ./...
  lint_commands: []
workspace:
  protected_paths:
    - ".env"
    - ".env.*"
  writable_paths:
    - internal
`); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Project.Project.ID != "yaml-project" || ws.Project.Stack.Languages[0] != "go" {
		t.Fatalf("expected Load to prefer project.yaml, got %+v", ws.Project)
	}
}

func TestValidateParsesEditableYAMLConfigs(t *testing.T) {
	root := t.TempDir()
	if _, err := Ensure(root); err != nil {
		t.Fatal(err)
	}

	if err := fsutil.WriteText(filepath.Join(root, DirName, "repository.yaml"), `schema_version: 1
repository:
  source:
    type: remote_git
    provider: github
    local_path: "/tmp/should-be-empty"
    url: "git@github.com:tiammomo/moyuan-code.git"
    clone_path: null
  default_remote: origin
  default_branch: main
git:
  branch_policy:
    mode: task_branch
    naming: "moyuan/{issue_id}-{slug}"
  commit_policy:
    enabled: true
    format: conventional_commits
`); err != nil {
		t.Fatal(err)
	}

	report, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" || !hasValidationIssue(report, "repository_local_path_must_be_empty_for_remote_git") {
		t.Fatalf("expected repository yaml schema validation failure, got %+v", report)
	}
}

func TestValidateReportsMalformedYAML(t *testing.T) {
	root := t.TempDir()
	if _, err := Ensure(root); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteText(filepath.Join(root, DirName, "project.yaml"), "schema_version: ["); err != nil {
		t.Fatal(err)
	}

	report, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" || !hasValidationIssue(report, "project_config_unreadable") {
		t.Fatalf("expected malformed project yaml failure, got %+v", report)
	}
}

func TestValidateProviderYAMLConfig(t *testing.T) {
	root := t.TempDir()
	if _, err := Ensure(root); err != nil {
		t.Fatal(err)
	}
	providersPath := filepath.Join(root, DirName, "models", "providers.yaml")
	if err := fsutil.WriteText(providersPath, `schema_version: 1
model_provider_management:
  enabled: true
  registry_path: ".moyuan/models/providers.json"
  usage_path: ".moyuan/model-ops/usage.jsonl"
accounts:
  - vendor: minimax
    api_type: anthropic-compatible
    base_url: "https://api.minimaxi.com/anthropic"
    auth_ref: "env:ANTHROPIC_AUTH_TOKEN"
    enabled: true
    data_policy:
      allow_code_context: true
providers:
  - type: llm-api
    adapter: anthropic-compatible
    account: minimax
    enabled: true
    models:
      - MiniMax-M2.7
quotas:
  default: {}
health_checks:
  enabled: true
security:
  forbid_plaintext_api_key: true
`); err != nil {
		t.Fatal(err)
	}

	report, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" {
		t.Fatalf("expected valid providers config to pass, got %+v", report)
	}

	if err := fsutil.WriteText(providersPath, `schema_version: 1
model_provider_management:
  enabled: true
accounts:
  - vendor: openai
    api_type: openai-compatible
    base_url: "https://api.example.com/v1"
    auth_ref: "sk-plain-secret"
    enabled: true
    data_policy: {}
providers:
  - type: llm-api
    account: openai
    enabled: true
    models:
      - gpt-test
security:
  forbid_plaintext_api_key: true
`); err != nil {
		t.Fatal(err)
	}
	report, err = Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" || !hasValidationIssue(report, "providers_plaintext_secret_forbidden") || !hasValidationIssue(report, "provider_account_auth_ref_must_be_reference") {
		t.Fatalf("expected plaintext provider secret validation failure, got %+v", report)
	}
}

func TestValidateRoutingYAMLConfig(t *testing.T) {
	root := t.TempDir()
	if _, err := Ensure(root); err != nil {
		t.Fatal(err)
	}
	routingPath := filepath.Join(root, DirName, "models", "routing.yaml")
	if err := fsutil.WriteText(routingPath, `schema_version: 1
policies:
  coding:
    primary:
      provider: codex_cli
      model: default
    fallback:
      - provider: claude_cli
        model: default
`); err != nil {
		t.Fatal(err)
	}
	report, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" {
		t.Fatalf("expected valid routing config to pass, got %+v", report)
	}

	if err := fsutil.WriteText(routingPath, `schema_version: 1
policies:
  coding:
    primary:
      model: default
    fallback:
      - model: default
`); err != nil {
		t.Fatal(err)
	}
	report, err = Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" || !hasValidationIssue(report, "routing_primary_provider_required") || !hasValidationIssue(report, "routing_fallback_provider_required") {
		t.Fatalf("expected routing provider validation failure, got %+v", report)
	}
}

func TestValidateVisualsYAMLConfig(t *testing.T) {
	root := t.TempDir()
	if _, err := Ensure(root); err != nil {
		t.Fatal(err)
	}
	visualsPath := filepath.Join(root, DirName, "visuals", "architecture-visuals.yaml")
	if err := fsutil.WriteText(visualsPath, `schema_version: 1
architecture_visuals:
  enabled: true
provider_policy:
  diagram_planning: planning
  image_generation: gpt_image_2
output:
  base_dir: ".moyuan/visuals"
diagram_types:
  - architecture
pipeline:
  steps:
    - plan
    - render
diagram_spec:
  required_fields:
    - title
    - sections
gpt_image_2:
  model: gpt-image-2
safety:
  strip_secrets: true
`); err != nil {
		t.Fatal(err)
	}
	report, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" {
		t.Fatalf("expected valid visuals config to pass, got %+v", report)
	}

	if err := fsutil.WriteText(visualsPath, `schema_version: 1
architecture_visuals:
  enabled: true
provider_policy:
  diagram_planning: planning
output:
  base_dir: ".moyuan/visuals"
diagram_types: []
pipeline:
  steps: []
diagram_spec:
  required_fields: []
gpt_image_2:
  model: ""
safety:
  strip_secrets: false
`); err != nil {
		t.Fatal(err)
	}
	report, err = Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" || !hasValidationIssue(report, "visuals_image_generation_required") || !hasValidationIssue(report, "visuals_strip_secrets_required") {
		t.Fatalf("expected visuals validation failure, got %+v", report)
	}
}

func TestValidateAgentRuntimesYAMLConfig(t *testing.T) {
	root := t.TempDir()
	if _, err := Ensure(root); err != nil {
		t.Fatal(err)
	}
	runtimesPath := filepath.Join(root, DirName, "runtimes", "agent-runtimes.yaml")
	if err := fsutil.WriteText(runtimesPath, `schema_version: 1
agent_runtimes:
  enabled: true
  default_runtime: codex_cli
  session_store: ".moyuan/runtimes/sessions"
  output_store: ".moyuan/runtime"
  runtimes:
    - type: native_agent_cli
      provider: codex
      enabled: true
      command: codex
      auth:
        mode: local_cli_login
      provider_env_profile:
        enabled: false
        allowed_env_keys: []
      health_check:
        command: "codex --version"
      invocation: {}
      context: {}
      tools: {}
      session:
        enable_resume: true
      audit:
        capture_diff_before_after: true
  routing:
    task_modes:
      frontend:
        - claude_cli
      backend:
        - codex_cli
  role_runtime_defaults:
    frontend: claude_cli
    backend: codex_cli
    backend_tuning: codex_cli
  isolation:
    require_issue_worktree: true
  require_quality_gate_after_run: true
`); err != nil {
		t.Fatal(err)
	}
	report, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" {
		t.Fatalf("expected valid agent runtimes config to pass, got %+v", report)
	}

	if err := fsutil.WriteText(runtimesPath, `schema_version: 1
agent_runtimes:
  enabled: true
  default_runtime: codex_cli
  session_store: ".moyuan/runtimes/sessions"
  output_store: ".moyuan/runtime"
  runtimes:
    - type: native_agent_cli
      provider: codex
      enabled: true
      command: codex
      auth:
        mode: local_cli_login
      provider_env_profile:
        enabled: true
        allowed_env_keys: []
      health_check:
        command: "codex --version"
      invocation:
        one_shot: true
      context: {}
      tools: {}
      session:
        enable_resume: true
      audit:
        capture_diff_before_after: false
  role_runtime_defaults:
    frontend: claude_cli
    backend: codex_cli
    backend_tuning: codex_cli
  isolation:
    require_issue_worktree: false
  require_quality_gate_after_run: false
`); err != nil {
		t.Fatal(err)
	}
	report, err = Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" || !hasValidationIssue(report, "agent_runtime_env_keys_required") || !hasValidationIssue(report, "agent_runtime_one_shot_must_be_empty_for_codex") || !hasValidationIssue(report, "agent_runtimes_quality_gate_required") {
		t.Fatalf("expected agent runtimes validation failure, got %+v", report)
	}
}

func TestValidateDevOpsPolicyYAMLConfigs(t *testing.T) {
	root := t.TempDir()
	if _, err := Ensure(root); err != nil {
		t.Fatal(err)
	}
	paths := ForRoot(root)
	if err := fsutil.WriteText(paths.ServerResourcesYAML, `schema_version: 1
server_resources:
  enabled: true
registry: ".moyuan/resources/hosts.json"
categories:
  development: {}
  production: {}
access_policy:
  require_owner: true
inventory_checks:
  enabled: true
hosts:
  - id: prod-1
    category: production
    owner: ops
    auth_ref: "secret:prod-1-ssh"
    lifecycle:
      expires_at: "2027-05-05"
`); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteText(paths.EnvironmentsYAML, `schema_version: 1
environments:
  production:
    resource_group: production-main
    approval_required: true
    artifact:
      type: binary
    deploy:
      strategy: rolling
    healthcheck:
      path: /health
    smoke_tests:
      - production-smoke
    rollback:
      strategy: previous_release
`); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteText(paths.ReleaseYAML, `schema_version: 1
release:
  auto_suggest: true
  mode: branch_only
  remote_providers:
    - github
  default_batch:
    max_issues: 6
  gates:
    require_release_note: true
    require_coverage_passed: true
    require_rollback_plan: true
  git:
    release_branch: "release/{version}"
  deployment:
    enabled: false
`); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteText(paths.BudgetYAML, `schema_version: 1
budget:
  max_parallel_issues: 2
  max_parallel_model_calls: 3
  max_daily_model_cost_usd: null
  max_task_runtime_minutes: 60
  fallback_to_low_cost_model: true
`); err != nil {
		t.Fatal(err)
	}

	report, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" {
		t.Fatalf("expected valid devops configs to pass, got %+v", report)
	}
}

func TestValidateDevOpsPolicyYAMLConfigFailures(t *testing.T) {
	root := t.TempDir()
	if _, err := Ensure(root); err != nil {
		t.Fatal(err)
	}
	paths := ForRoot(root)
	if err := fsutil.WriteText(paths.ServerResourcesYAML, `schema_version: 1
server_resources:
  enabled: true
hosts:
  - id: prod-1
    category: production
`); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteText(paths.EnvironmentsYAML, `schema_version: 1
environments:
  production:
    approval_required: false
    artifact:
      type: binary
`); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteText(paths.ReleaseYAML, `schema_version: 1
release:
  mode: branch_only
  default_batch: {}
  gates:
    require_release_note: false
    require_coverage_passed: false
    require_rollback_plan: false
  git: {}
  deployment:
    enabled: true
`); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteText(paths.BudgetYAML, `schema_version: 1
budget:
  max_parallel_issues: 0
  max_parallel_model_calls: 0
  max_daily_model_cost_usd: -1
  max_task_runtime_minutes: 0
`); err != nil {
		t.Fatal(err)
	}

	report, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" ||
		!hasValidationIssue(report, "server_resources_section_required") ||
		!hasValidationIssue(report, "production_host_auth_ref_required") ||
		!hasValidationIssue(report, "production_approval_required") ||
		!hasValidationIssue(report, "production_smoke_tests_required") ||
		!hasValidationIssue(report, "release_remote_providers_required") ||
		!hasValidationIssue(report, "release_deployment_must_be_disabled") ||
		!hasValidationIssue(report, "budget_positive_integer_required") ||
		!hasValidationIssue(report, "budget_daily_cost_must_be_null_or_positive") {
		t.Fatalf("expected devops policy validation failures, got %+v", report)
	}
}

func hasValidationIssue(report ValidationReport, code string) bool {
	for _, issue := range report.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
