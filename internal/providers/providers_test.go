package providers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/fsutil"
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
	telemetry, err := ListTelemetry(root, "gpt_image_2", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(telemetry) != 1 || telemetry[0].Decision != "PROVIDER_TELEMETRY_BLOCKING" || telemetry[0].Source != "manual_ops" {
		t.Fatalf("expected blocking telemetry record, got %+v", telemetry)
	}

	blocked, err := Route(root, RouteRequest{OutputType: "architecture_diagram"})
	if err != nil {
		t.Fatal(err)
	}
	if !blocked.Blocked || blocked.Reason != "provider_unhealthy:gpt_image_2:unhealthy" {
		t.Fatalf("expected unhealthy provider to block, got %+v", blocked)
	}
	if !hasRouteSignal(blocked.Signals, "health", "unhealthy") {
		t.Fatalf("expected route decision health signal, got %+v", blocked.Signals)
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

func TestProviderExecutionAndQualityFeedbackUpdateTelemetry(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	runtimeRecord, ok, err := RecordExecutionFeedback(root, FeedbackOptions{
		ProviderID:    "codex_cli",
		RuntimeID:     "codex_cli",
		ModelID:       "gpt-5.5",
		RunID:         "run-1",
		IssueID:       "issue-1",
		RuntimeStatus: "failed",
		Reason:        "runtime_failed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || runtimeRecord.Source != "runtime_execution" || runtimeRecord.RuntimeStatus != "failed" || runtimeRecord.Decision != "PROVIDER_TELEMETRY_WARNING" {
		t.Fatalf("unexpected runtime feedback telemetry: ok=%v record=%+v", ok, runtimeRecord)
	}
	provider, ok, err := Show(root, "codex_cli")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || provider.Health.Status != "degraded" || provider.Usage.Requests != 1 {
		t.Fatalf("expected runtime feedback to degrade health and increment usage: ok=%v provider=%+v", ok, provider)
	}
	route, err := Route(root, RouteRequest{Role: "backend", RequiresRepoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	if route.Decision != DecisionAllowed || !hasRouteSignal(route.Signals, "health", "degraded") {
		t.Fatalf("expected degraded health signal without blocking route, got %+v", route)
	}

	qualityRecord, ok, err := RecordQualityFeedback(root, FeedbackOptions{
		ProviderID:      "codex_cli",
		RuntimeID:       "codex_cli",
		RunID:           "run-1",
		IssueID:         "issue-1",
		QualityReportID: "quality-1",
		RuntimeStatus:   "completed",
		QualityStatus:   "passed",
		Reason:          "quality_status:passed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || qualityRecord.Source != "quality_gate" || qualityRecord.QualityStatus != "passed" || qualityRecord.Decision != "PROVIDER_TELEMETRY_OK" {
		t.Fatalf("unexpected quality feedback telemetry: ok=%v record=%+v", ok, qualityRecord)
	}
	provider, ok, err = Show(root, "codex_cli")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || provider.Health.Status != "ok" || provider.Usage.Requests != 1 {
		t.Fatalf("expected quality feedback to restore health without usage increment: ok=%v provider=%+v", ok, provider)
	}
	records, err := ListTelemetry(root, "codex_cli", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !hasTelemetrySource(records, "runtime_execution") || !hasTelemetrySource(records, "quality_gate") || !hasTelemetrySource(records, "provider_route") {
		t.Fatalf("expected runtime, quality and route telemetry, got %+v", records)
	}
}

func TestProviderRouteExplainsSelectedAndSkippedCandidates(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Upsert(root, Provider{
		ID:      "glm-memory",
		Name:    "GLM Memory",
		Vendor:  "glm",
		APIType: "openai-compatible",
		AuthRef: "env:GLM_API_KEY",
		Enabled: true,
		DataPolicy: DataPolicy{
			AllowProjectMemory: true,
		},
		Models:          []Model{{ID: "glm-4"}},
		AllowedUseCases: []string{"memory_extraction"},
	}); err != nil {
		t.Fatal(err)
	}

	route, err := Route(root, RouteRequest{Role: "memory_curator", TaskType: "memory_extraction", IncludesProjectMemory: true})
	if err != nil {
		t.Fatal(err)
	}
	if route.Decision != DecisionAllowed || route.ProviderID != "glm-memory" || route.Explanation == nil {
		t.Fatalf("unexpected route explanation: %+v", route)
	}
	if route.Explanation.CandidateCount < 4 || route.Explanation.SelectedCount != 1 || route.Explanation.SkippedCount == 0 || route.Explanation.BlockedCount == 0 {
		t.Fatalf("expected selected/skipped/blocked candidate counts, got %+v candidates=%+v", route.Explanation, route.Candidates)
	}
	if !hasRouteCandidate(route.Candidates, "glm-memory", "selected", "memory_low_cost_provider") ||
		!hasRouteCandidate(route.Candidates, "codex_cli", "skipped", "memory_requires_api_provider") ||
		!hasRouteCandidate(route.Candidates, "gpt_image_2", "blocked", "provider_disabled") {
		t.Fatalf("expected selected and skipped candidate reasons, got %+v", route.Candidates)
	}
	if !hasCandidateSignal(route.Candidates, "glm-memory", "selection", "selected") {
		t.Fatalf("expected selected candidate signal, got %+v", route.Candidates)
	}
}

func TestProviderFeedbackAccruesUsageQuotaCostAndQualitySignals(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	_, ok, err := UpdateOps(root, "codex_cli", OpsSnapshot{
		Quota: Quota{Status: "ok", LimitTokens: 1000, UsedTokens: 100},
		Usage: Usage{Window: "daily"},
		Cost: Cost{
			Currency:             "usd",
			BudgetAmount:         1,
			InputTokenCostPer1K:  0.01,
			OutputTokenCostPer1K: 0.02,
			Status:               "ok",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected codex_cli provider")
	}

	record, ok, err := RecordExecutionFeedback(root, FeedbackOptions{
		ProviderID:    "codex_cli",
		RuntimeID:     "codex_cli",
		RunID:         "run-metered",
		IssueID:       "issue-metered",
		RuntimeStatus: "completed",
		InputTokens:   120,
		OutputTokens:  80,
		Reason:        "runtime_status:completed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || record.TotalTokens != 200 || !floatNear(record.IncrementalCost, 0.0028) {
		t.Fatalf("unexpected metered telemetry: ok=%v record=%+v", ok, record)
	}
	provider, ok, err := Show(root, "codex_cli")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("missing codex_cli provider")
	}
	if provider.Usage.InputTokens != 120 || provider.Usage.OutputTokens != 80 || provider.Usage.TotalTokens != 200 {
		t.Fatalf("expected feedback usage tokens to accrue, got %+v", provider.Usage)
	}
	if provider.Quota.UsedTokens != 300 || provider.Quota.RemainingTokens != 700 || provider.Quota.Status != "ok" {
		t.Fatalf("expected quota to be deducted from token feedback, got %+v", provider.Quota)
	}
	if !floatNear(provider.Cost.EstimatedAmount, 0.0028) || provider.Cost.Status != "ok" {
		t.Fatalf("expected estimated cost to accrue, got %+v", provider.Cost)
	}
	if provider.Feedback.RuntimeExecutions != 1 || provider.Feedback.RuntimeFailures != 0 {
		t.Fatalf("expected runtime feedback summary, got %+v", provider.Feedback)
	}

	_, ok, err = RecordQualityFeedback(root, FeedbackOptions{
		ProviderID:      "codex_cli",
		RuntimeID:       "codex_cli",
		RunID:           "run-metered",
		IssueID:         "issue-metered",
		QualityReportID: "quality-metered",
		RuntimeStatus:   "completed",
		QualityStatus:   "failed",
		Reason:          "quality_status:failed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected quality feedback to be recorded")
	}
	provider, ok, err = Show(root, "codex_cli")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || provider.Feedback.RuntimeExecutions != 1 || provider.Feedback.QualityStatus != "degraded" || provider.Feedback.QualityFailures != 1 {
		t.Fatalf("expected quality feedback summary without double-counted runtime execution: ok=%v provider=%+v", ok, provider)
	}
	route, err := Route(root, RouteRequest{Role: "backend", RequiresRepoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	if route.Decision != DecisionAllowed || !hasRouteSignal(route.Signals, "quality", "degraded") {
		t.Fatalf("expected quality route signal, got %+v", route)
	}
}

func TestProviderOpsRefreshUpdatesLocalSignals(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLM_REFRESH_KEY", "present")

	if _, err := Upsert(root, Provider{
		ID:      "glm-refresh",
		Vendor:  "zhipu",
		APIType: "openai-compatible",
		BaseURL: "https://example.invalid/v1",
		AuthRef: "env:GLM_REFRESH_KEY",
		Enabled: true,
		Models:  []Model{{ID: "glm-4"}},
		Quota:   Quota{LimitTokens: 100, UsedTokens: 85},
		Usage:   Usage{Window: "daily", Requests: 2},
		Cost:    Cost{Currency: "usd", EstimatedAmount: 9, BudgetAmount: 10},
		DataPolicy: DataPolicy{
			AllowProjectMemory: true,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := Upsert(root, Provider{
		ID:      "missing-auth",
		Vendor:  "zhipu",
		APIType: "openai-compatible",
		BaseURL: "https://example.invalid/v1",
		AuthRef: "env:GLM_REFRESH_MISSING",
		Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}

	result, err := RefreshOps(root, OpsRefreshOptions{ProviderID: "glm-refresh"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 1 || len(result.Providers) != 1 {
		t.Fatalf("unexpected refresh result: %+v", result)
	}
	refreshed := result.Providers[0]
	if refreshed.Health.Status != "ok" || refreshed.Quota.Status != "warning" || refreshed.Quota.RemainingTokens != 15 || refreshed.Cost.Status != "warning" {
		t.Fatalf("unexpected refreshed provider: %+v", refreshed)
	}
	if refreshed.Usage.UpdatedAt == "" {
		t.Fatalf("expected usage updated_at to be refreshed: %+v", refreshed.Usage)
	}

	result, err = RefreshOps(root, OpsRefreshOptions{ProviderID: "missing-auth"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 1 || result.Providers[0].Health.Status != "unhealthy" || !strings.Contains(result.Providers[0].Health.Reason, "auth_ref_env_missing") {
		t.Fatalf("expected missing env auth to mark provider unhealthy: %+v", result)
	}

	disabled, ok, err := Disable(root, "missing-auth")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || disabled.Enabled {
		t.Fatalf("expected provider disabled: ok=%v provider=%+v", ok, disabled)
	}
	result, err = RefreshOps(root, OpsRefreshOptions{ProviderID: "missing-auth"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 0 || result.Skipped != 1 || result.Decisions[0].Reason != "provider_disabled" {
		t.Fatalf("expected disabled provider to be skipped: %+v", result)
	}
}

func TestProviderOpsRefreshCanRunOptionalHTTPProbe(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	probeHits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probeHits++
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected probe path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer probe-token" {
			t.Fatalf("unexpected probe auth header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	t.Setenv("PROBE_API_KEY", "probe-token")
	writeProviderSecretPolicy(t, root, `
schema_version: 1
secrets:
  probe_api_token:
    type: token
    ref: env:PROBE_API_KEY
    usage:
      - model.provider.*
`)

	if _, err := Upsert(root, Provider{
		ID:      "glm-probe",
		Vendor:  "zhipu",
		APIType: "openai-compatible",
		BaseURL: server.URL + "/v1",
		AuthRef: "secret:probe_api_token",
		Enabled: true,
		Models:  []Model{{ID: "glm-4"}},
	}); err != nil {
		t.Fatal(err)
	}

	blocked, err := RefreshOps(root, OpsRefreshOptions{ProviderID: "glm-probe", Probe: true, ProbeTimeoutMS: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Updated != 0 || blocked.ApprovalID == "" || blocked.Decisions[0].Reason != "provider_probe_approval_required" || probeHits != 0 {
		t.Fatalf("expected unapproved probe to request approval without hitting upstream: hits=%d result=%+v", probeHits, blocked)
	}

	result, err := RefreshOps(root, OpsRefreshOptions{ProviderID: "glm-probe", Probe: true, ProbeTimeoutMS: 1000, Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 1 || probeHits != 1 {
		t.Fatalf("unexpected probe refresh result: hits=%d result=%+v", probeHits, result)
	}
	if result.Providers[0].Health.Status != "ok" || result.Providers[0].Health.Reason != "probe_ok:openai_compatible_models" {
		t.Fatalf("expected probe health to be ok, got %+v", result.Providers[0].Health)
	}
	if result.Decisions[0].ProbeStatus != "ok" || result.Decisions[0].ProbeReason != "probe_ok:openai_compatible_models" {
		t.Fatalf("expected probe decision metadata, got %+v", result.Decisions[0])
	}
	registry, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(fmt.Sprintf("%+v", registry), "probe-token") {
		t.Fatalf("provider registry leaked probe token")
	}
	auditText, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(root).LogsDir, "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found || !strings.Contains(auditText, "secret.access.granted") {
		t.Fatalf("expected secret access audit, found=%v text=%s", found, auditText)
	}
	if strings.Contains(auditText, "probe-token") {
		t.Fatalf("audit log leaked probe token: %s", auditText)
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

func writeProviderSecretPolicy(t *testing.T, root string, text string) {
	t.Helper()
	path := filepath.Join(workspace.ForRoot(root).MoyuanDir, "policies", "secrets.yaml")
	if err := fsutil.WriteText(path, strings.TrimSpace(text)+"\n"); err != nil {
		t.Fatal(err)
	}
}

func hasRouteSignal(signals []RouteSignal, signalType string, status string) bool {
	for _, signal := range signals {
		if signal.Type == signalType && signal.Status == status {
			return true
		}
	}
	return false
}

func hasRouteCandidate(candidates []RouteCandidate, providerID string, status string, reason string) bool {
	for _, candidate := range candidates {
		if candidate.ProviderID == providerID && candidate.Status == status && candidate.Reason == reason {
			return true
		}
	}
	return false
}

func hasCandidateSignal(candidates []RouteCandidate, providerID string, signalType string, status string) bool {
	for _, candidate := range candidates {
		if candidate.ProviderID == providerID && hasRouteSignal(candidate.Signals, signalType, status) {
			return true
		}
	}
	return false
}

func hasTelemetrySource(records []TelemetryRecord, source string) bool {
	for _, record := range records {
		if record.Source == source {
			return true
		}
	}
	return false
}

func floatNear(got float64, want float64) bool {
	delta := got - want
	if delta < 0 {
		delta = -delta
	}
	return delta < 0.000001
}
