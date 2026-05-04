package providers

import (
	"testing"

	"moyuan-code/internal/workspace"
)

func TestDefaultRoleRoutingUsesNativeAgentCLIs(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	frontend, err := Route(root, RouteRequest{Role: "frontend", RequiresRepoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	if frontend.Decision != DecisionAllowed || frontend.RuntimeID != "claude_cli" || frontend.ProviderID != "claude_cli" {
		t.Fatalf("unexpected frontend route: %+v", frontend)
	}

	backend, err := Route(root, RouteRequest{Role: "backend", RequiresRepoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	if backend.Decision != DecisionAllowed || backend.RuntimeID != "codex_cli" || backend.ProviderID != "codex_cli" {
		t.Fatalf("unexpected backend route: %+v", backend)
	}
}

func TestProviderRegistryRejectsPlainSecretsAndPersistsRefs(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	_, err := Upsert(root, Provider{
		ID:      "bad-secret",
		Vendor:  "openai",
		APIType: "openai",
		AuthRef: "plain-secret-should-not-be-stored",
		Enabled: true,
	})
	if err == nil || err.Error() != "auth_ref_must_be_reference" {
		t.Fatalf("expected auth_ref rejection, got %v", err)
	}

	saved, err := Upsert(root, Provider{
		ID:      "glm-main",
		Vendor:  "zhipu",
		APIType: "openai-compatible",
		AuthRef: "env:GLM_API_KEY",
		Enabled: true,
		Models:  []Model{{ID: "glm-4"}},
		DataPolicy: DataPolicy{
			AllowProjectMemory: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.AuthRef != "env:GLM_API_KEY" || saved.APIType != "openai_compatible" {
		t.Fatalf("unexpected saved provider: %+v", saved)
	}

	shown, ok, err := Show(root, "glm-main")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || shown.ID != "glm-main" || !shown.Enabled {
		t.Fatalf("provider not found/enabled: ok=%v provider=%+v", ok, shown)
	}

	disabled, ok, err := Disable(root, "glm-main")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || disabled.Enabled {
		t.Fatalf("provider not disabled: ok=%v provider=%+v", ok, disabled)
	}
}

func TestProviderRouteUsesEnabledImageProviderAndBlocksSensitiveData(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	blocked, err := Route(root, RouteRequest{OutputType: "architecture_diagram"})
	if err != nil {
		t.Fatal(err)
	}
	if !blocked.Blocked || blocked.Reason != "provider_disabled:gpt_image_2" {
		t.Fatalf("expected disabled image provider to block, got %+v", blocked)
	}

	imageProvider, ok, err := Show(root, "gpt_image_2")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("missing default gpt_image_2 provider")
	}
	imageProvider.Enabled = true
	if _, err := Upsert(root, imageProvider); err != nil {
		t.Fatal(err)
	}

	allowed, err := Route(root, RouteRequest{OutputType: "architecture_diagram"})
	if err != nil {
		t.Fatal(err)
	}
	if allowed.Decision != DecisionAllowed || allowed.ProviderID != "gpt_image_2" || allowed.ModelID != "gpt-image-2" {
		t.Fatalf("unexpected image route: %+v", allowed)
	}

	secret, err := Route(root, RouteRequest{Role: "backend", IncludesSecrets: true, RequiresRepoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	if !secret.Blocked || secret.Reason != "contains_secret_context" {
		t.Fatalf("secret route should be blocked: %+v", secret)
	}
}
