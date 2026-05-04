package skills

import (
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

func TestSkillBindingRequiresEnabledSkillAndCanDisable(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Upsert(root, Definition{ID: "tdd", Source: "github:mattpocock/skills", Enabled: true, RiskLevel: "low", CompatibleRoles: []string{"backend"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := Upsert(root, Definition{ID: "dangerous", Source: "local", Enabled: true, RiskLevel: "high"}); err != nil {
		t.Fatal(err)
	}

	binding, err := UpsertBinding(root, Binding{SkillID: "tdd", TargetType: "role", TargetID: "backend"})
	if err != nil {
		t.Fatal(err)
	}
	if binding.ID == "" || binding.Status != "enabled" || binding.Priority != 200 {
		t.Fatalf("unexpected binding: %+v", binding)
	}
	list, err := ListBindings(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one binding, got %+v", list)
	}

	disabled, ok, err := DisableBinding(root, binding.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || disabled.Status != "disabled" {
		t.Fatalf("expected disabled binding, ok=%v binding=%+v", ok, disabled)
	}
	content, found, err := fsutil.ReadText(filepath.Join(root, ".moyuan/skills/bindings.events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found || !strings.Contains(content, "skill.binding.disabled") {
		t.Fatalf("expected binding event, found=%v content=%s", found, content)
	}

	if _, err := UpsertBinding(root, Binding{SkillID: "missing", TargetType: "role", TargetID: "backend"}); err == nil {
		t.Fatal("expected missing skill to be rejected")
	}
	if _, err := UpsertBinding(root, Binding{SkillID: "dangerous", TargetType: "project"}); err == nil {
		t.Fatal("expected high risk project binding to be rejected")
	}
	if _, err := UpsertBinding(root, Binding{SkillID: "tdd", TargetType: "role", TargetID: "backend", Config: map[string]string{"token": "sk-plain-secret"}}); err == nil {
		t.Fatal("expected secret-bearing binding config to be rejected")
	}
}
