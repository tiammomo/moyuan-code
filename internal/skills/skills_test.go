package skills

import (
	"testing"

	"moyuan-code/internal/workspace"
)

func TestSkillRegistryUpsertDedupDisableAndRejectsPlainSecrets(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	saved, err := Upsert(root, Definition{
		ID:              "/tdd",
		Name:            "TDD",
		Source:          "github:mattpocock/skills",
		Version:         "latest",
		Enabled:         true,
		RiskLevel:       "low",
		CompatibleRoles: []string{"backend", "Backend", "tester"},
		Tags:            []string{"quality", "tdd"},
		RequiredTools:   []string{"go-test"},
		AuthRef:         "env:SKILLS_REGISTRY_TOKEN",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID != "tdd" {
		t.Fatalf("normalized id = %s", saved.ID)
	}
	if len(saved.CompatibleRoles) != 2 {
		t.Fatalf("roles were not normalized/deduped: %+v", saved.CompatibleRoles)
	}

	updated, err := Upsert(root, Definition{
		ID:        "tdd",
		Source:    "github:mattpocock/skills",
		Version:   "v2",
		Enabled:   true,
		RiskLevel: "medium",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CreatedAt != saved.CreatedAt {
		t.Fatalf("upsert should preserve created_at")
	}

	list, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one deduped skill, got %d", len(list))
	}

	disabled, ok, err := Disable(root, "tdd")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || disabled.Enabled {
		t.Fatalf("expected disabled skill, ok=%v enabled=%v", ok, disabled.Enabled)
	}

	if _, err := Upsert(root, Definition{ID: "private", Source: "github:private/skills", AuthRef: "sk-plain-secret"}); err == nil {
		t.Fatal("expected plain secret auth ref to be rejected")
	}
	if _, err := Upsert(root, Definition{ID: "secret-source", Source: "https://example.com?token=plain-secret"}); err == nil {
		t.Fatal("expected secret-bearing metadata to be rejected")
	}
	if _, err := Upsert(root, Definition{ID: "bad-risk", Source: "local", RiskLevel: "critical"}); err == nil {
		t.Fatal("expected invalid risk level to be rejected")
	}
}
