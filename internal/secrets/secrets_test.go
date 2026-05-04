package secrets

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

func TestResolveEnvReferenceDoesNotSerializeValue(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MOYUAN_SECRET_TEST_KEY", "sk-secret-value")

	resolved, err := Resolve(root, "env:MOYUAN_SECRET_TEST_KEY", ResolveOptions{Purpose: "runtime.invoke", AdapterID: "codex_cli"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Status != "ok" || resolved.Value() != "sk-secret-value" || resolved.EnvKey != "MOYUAN_SECRET_TEST_KEY" {
		t.Fatalf("unexpected env resolution: %+v", resolved)
	}
	encoded, err := json.Marshal(resolved)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "sk-secret-value") || strings.Contains(fmt.Sprintf("%+v", resolved), "sk-secret-value") {
		t.Fatalf("resolved value leaked through serialization: json=%s fmt=%+v", encoded, resolved)
	}
	auditText, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(root).LogsDir, "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found || !strings.Contains(auditText, "secret.access.granted") {
		t.Fatalf("expected secret audit event, found=%v text=%s", found, auditText)
	}
	if strings.Contains(auditText, "sk-secret-value") {
		t.Fatalf("audit log leaked secret value: %s", auditText)
	}
}

func TestResolveSecretReferenceRequiresRegisteredUsage(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROVIDER_TOKEN_TEST", "provider-secret-value")
	writeSecretPolicy(t, root, `
schema_version: 1
secrets:
  provider_token:
    type: token
    ref: env:PROVIDER_TOKEN_TEST
    usage:
      - runtime.invoke
      - model.provider.*
`)

	resolved, err := Resolve(root, "secret:provider_token", ResolveOptions{Purpose: "model.provider.probe", AdapterID: "glm-probe"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Status != "ok" || resolved.Value() != "provider-secret-value" || resolved.SecretID != "provider_token" {
		t.Fatalf("unexpected secret resolution: %+v", resolved)
	}
	denied, err := Resolve(root, "secret:provider_token", ResolveOptions{Purpose: "release.tag_push", AdapterID: "release"})
	if err != nil {
		t.Fatal(err)
	}
	if denied.Status != "denied" || !strings.Contains(denied.Reason, "secret_usage_not_allowed") || denied.Value() != "" {
		t.Fatalf("expected usage denial without value, got %+v", denied)
	}
}

func TestSecretReferenceReportsMissingPolicy(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	status, err := Status(root, "secret:missing", ResolveOptions{Purpose: "runtime.invoke"})
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "missing" || status.Reason != "secret_policy_missing" || status.Value() != "" {
		t.Fatalf("expected missing policy status without value, got %+v", status)
	}
}

func writeSecretPolicy(t *testing.T, root string, text string) {
	t.Helper()
	path := filepath.Join(workspace.ForRoot(root).MoyuanDir, "policies", "secrets.yaml")
	if err := fsutil.WriteText(path, strings.TrimSpace(text)+"\n"); err != nil {
		t.Fatal(err)
	}
}
