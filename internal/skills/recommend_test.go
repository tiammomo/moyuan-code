package skills

import (
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

func TestRecommendScoresEnabledCompatibleSkillsAndSkipsDisabled(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Upsert(root, Definition{ID: "tdd", Source: "github:mattpocock/skills", Enabled: true, RiskLevel: "low", CompatibleRoles: []string{"backend"}, Tags: []string{"testing"}, RequiredTools: []string{"go-test"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := Upsert(root, Definition{ID: "frontend-polish", Source: "local", Enabled: true, RiskLevel: "medium", CompatibleRoles: []string{"frontend"}, Tags: []string{"ui"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := Upsert(root, Definition{ID: "disabled-diagnose", Source: "local", Enabled: true, CompatibleRoles: []string{"backend"}, Tags: []string{"testing"}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Disable(root, "disabled-diagnose"); err != nil {
		t.Fatal(err)
	}

	report, err := Recommend(root, RecommendOptions{Role: "backend", TaskType: "testing", RiskLevel: "medium", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates) != 1 {
		t.Fatalf("expected one backend enabled recommendation, got %+v", report.Candidates)
	}
	candidate := report.Candidates[0]
	if candidate.SkillID != "tdd" {
		t.Fatalf("expected tdd candidate, got %+v", candidate)
	}
	if candidate.Score <= 0.7 {
		t.Fatalf("expected strong score, got %+v", candidate)
	}
	if len(candidate.Reasons) == 0 || candidate.Reasons[0] != "enabled_skill" {
		t.Fatalf("missing reasons: %+v", candidate)
	}
	assertRecommendFileContains(t, filepath.Join(root, ".moyuan/skills/recommendations.jsonl"), report.ID)
}

func TestRecommendUsesEffectivenessSignals(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Upsert(root, Definition{
		ID:              "tdd",
		Source:          "github:mattpocock/skills",
		Enabled:         true,
		RiskLevel:       "low",
		CompatibleRoles: []string{"backend"},
		Tags:            []string{"quality"},
	}); err != nil {
		t.Fatal(err)
	}
	before, err := Recommend(root, RecommendOptions{Role: "backend", TaskType: "quality", RiskLevel: "medium"})
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Candidates) != 1 {
		t.Fatalf("expected one recommendation, got %+v", before)
	}

	if _, err := RecordEffectiveness(root, Effectiveness{
		SkillID:       "tdd",
		IssueID:       "phase1-001",
		Outcome:       "helped",
		QualityImpact: "improved",
		ReworkReduced: true,
	}); err != nil {
		t.Fatal(err)
	}
	after, err := Recommend(root, RecommendOptions{Role: "backend", TaskType: "quality", RiskLevel: "medium"})
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Candidates) != 1 {
		t.Fatalf("expected one recommendation after effectiveness, got %+v", after)
	}
	if after.Candidates[0].Score <= before.Candidates[0].Score {
		t.Fatalf("expected effectiveness to increase score: before=%+v after=%+v", before.Candidates[0], after.Candidates[0])
	}
	if !contains(after.Candidates[0].Reasons, "effectiveness_helped") || !contains(after.Candidates[0].Reasons, "quality_improved") || !contains(after.Candidates[0].Reasons, "rework_reduced") {
		t.Fatalf("expected effectiveness reasons, got %+v", after.Candidates[0])
	}
}

func assertRecommendFileContains(t *testing.T, path string, expected string) {
	t.Helper()
	content, found, err := fsutil.ReadText(path)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatalf("expected file %s", path)
	}
	if !strings.Contains(content, expected) {
		t.Fatalf("expected %s to contain %q, got %s", path, expected, content)
	}
}
