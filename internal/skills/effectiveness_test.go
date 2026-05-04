package skills

import (
	"testing"

	"moyuan-code/internal/workspace"
)

func TestRecordAndListSkillEffectiveness(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Upsert(root, Definition{ID: "tdd", Source: "github:mattpocock/skills", Enabled: true, RiskLevel: "low"}); err != nil {
		t.Fatal(err)
	}

	record, err := RecordEffectiveness(root, Effectiveness{
		SkillID:         "tdd",
		RunID:           "run-1",
		IssueID:         "issue-1",
		Outcome:         "helped",
		QualityImpact:   "improved",
		ReworkReduced:   true,
		DurationSeconds: 42,
		Findings:        []string{"reduced rework", "token=secret should be dropped"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.ID == "" || len(record.Findings) != 1 {
		t.Fatalf("unexpected record: %+v", record)
	}

	records, err := ListEffectiveness(root, "tdd", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].ID != record.ID {
		t.Fatalf("expected record in list, got %+v", records)
	}

	if _, err := RecordEffectiveness(root, Effectiveness{SkillID: "missing", IssueID: "issue-1"}); err == nil {
		t.Fatal("expected missing skill to be rejected")
	}
	if _, err := RecordEffectiveness(root, Effectiveness{SkillID: "tdd"}); err == nil {
		t.Fatal("expected missing reference to be rejected")
	}
	if _, err := RecordEffectiveness(root, Effectiveness{SkillID: "tdd", IssueID: "issue-1", Outcome: "amazing"}); err == nil {
		t.Fatal("expected invalid outcome to be rejected")
	}
}
