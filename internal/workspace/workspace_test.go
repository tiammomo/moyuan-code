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

func hasValidationIssue(report ValidationReport, code string) bool {
	for _, issue := range report.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
