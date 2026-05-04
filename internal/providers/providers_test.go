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

func TestProviderOpsSnapshotBlocksUnavailableRoutes(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
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

	updated, ok, err := UpdateOps(root, "gpt_image_2", OpsSnapshot{
		Health: Health{Status: "unhealthy", Reason: "upstream_timeout"},
		Quota:  Quota{Status: "ok", LimitTokens: 1000, UsedTokens: 200},
		Usage:  Usage{Window: "daily", Requests: 12, InputTokens: 100, OutputTokens: 50},
		Cost:   Cost{Currency: "usd", EstimatedAmount: 1.2, BudgetAmount: 10, Status: "ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || updated.Health.Status != "unhealthy" || updated.Quota.RemainingTokens != 800 || updated.Usage.TotalTokens != 150 || updated.Cost.Currency != "USD" {
		t.Fatalf("unexpected ops snapshot: ok=%v provider=%+v", ok, updated)
	}

	blocked, err := Route(root, RouteRequest{OutputType: "architecture_diagram"})
	if err != nil {
		t.Fatal(err)
	}
	if !blocked.Blocked || blocked.Reason != "provider_unhealthy:gpt_image_2:unhealthy" {
		t.Fatalf("expected unhealthy provider to block, got %+v", blocked)
	}

	updated, ok, err = UpdateOps(root, "gpt_image_2", OpsSnapshot{Health: Health{Status: "ok"}, Quota: Quota{Status: "exhausted"}})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || updated.Health.Status != "ok" || updated.Quota.Status != "exhausted" {
		t.Fatalf("unexpected quota update: ok=%v provider=%+v", ok, updated)
	}
	blocked, err = Route(root, RouteRequest{OutputType: "architecture_diagram"})
	if err != nil {
		t.Fatal(err)
	}
	if !blocked.Blocked || blocked.Reason != "provider_quota_exhausted:gpt_image_2" {
		t.Fatalf("expected exhausted quota to block, got %+v", blocked)
	}

	if _, _, err := UpdateOps(root, "gpt_image_2", OpsSnapshot{Health: Health{Status: "burning"}}); err == nil {
		t.Fatal("expected invalid health status to be rejected")
	}
	if _, _, err := UpdateOps(root, "gpt_image_2", OpsSnapshot{Quota: Quota{UsedTokens: -1}}); err == nil {
		t.Fatal("expected negative usage values to be rejected")
	}
}

func TestModelStrategySwitchesRouteWithoutBypassingPolicy(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Upsert(root, Provider{
		ID:              "glm-low-cost",
		Vendor:          "zhipu",
		APIType:         "openai-compatible",
		AuthRef:         "env:GLM_API_KEY",
		Enabled:         true,
		AllowedUseCases: []string{"memory_extraction"},
		Models:          []Model{{ID: "glm-4"}},
		DataPolicy:      DataPolicy{AllowProjectMemory: true},
	}); err != nil {
		t.Fatal(err)
	}
	imageProvider, ok, err := Show(root, "gpt_image_2")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("missing gpt image provider")
	}
	imageProvider.Enabled = true
	if _, err := Upsert(root, imageProvider); err != nil {
		t.Fatal(err)
	}

	memory, err := Route(root, RouteRequest{ModelStrategy: "low-cost-memory", IncludesProjectMemory: true})
	if err != nil {
		t.Fatal(err)
	}
	if memory.Decision != DecisionAllowed || memory.ProviderID != "glm-low-cost" || memory.Strategy != "low_cost_memory" {
		t.Fatalf("unexpected low-cost memory route: %+v", memory)
	}

	image, err := Route(root, RouteRequest{ModelStrategy: "image-diagram"})
	if err != nil {
		t.Fatal(err)
	}
	if image.Decision != DecisionAllowed || image.ProviderID != "gpt_image_2" || image.Strategy != "image_diagram" {
		t.Fatalf("unexpected image strategy route: %+v", image)
	}

	secret, err := Route(root, RouteRequest{ModelStrategy: "low-cost-memory", IncludesSecrets: true})
	if err != nil {
		t.Fatal(err)
	}
	if !secret.Blocked || secret.Reason != "contains_secret_context" || secret.Strategy != "low_cost_memory" {
		t.Fatalf("strategy should not bypass secret policy: %+v", secret)
	}
}
