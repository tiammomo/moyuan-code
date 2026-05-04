package review

import (
	"testing"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/orchestrator"
	"moyuan-code/internal/quality"
	"moyuan-code/internal/workspace"
)

func TestDecideMergeAllowsAcceptedIssueWithAcceptedQuality(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	writeIssueState(t, root, orchestrator.IssueState{IssueID: "issue-1", Status: "accepted", QualityReportID: "quality-1"})
	writeQualityReport(t, root, quality.Report{ID: "quality-1", TaskID: "issue-1", Status: "passed", ReviewStatus: "accepted"})

	decision, err := DecideMerge(root, "issue-1")
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != "ready_to_merge" || decision.Decision != "MERGE_ALLOWED" {
		t.Fatalf("unexpected merge decision: %+v", decision)
	}

	loaded, ok, err := Load(root, decision.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || loaded.ID != decision.ID {
		t.Fatalf("expected merge decision to be saved, ok=%v loaded=%+v", ok, loaded)
	}
}

func TestDecideMergeBlocksMissingOrRejectedFacts(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	missing, err := DecideMerge(root, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if missing.Status != "blocked" || missing.Reasons[0] != "issue_state_missing" {
		t.Fatalf("unexpected missing decision: %+v", missing)
	}

	writeIssueState(t, root, orchestrator.IssueState{IssueID: "issue-2", Status: "accepted", QualityReportID: "quality-2"})
	writeQualityReport(t, root, quality.Report{ID: "quality-2", TaskID: "issue-2", Status: "failed", ReviewStatus: "rejected"})
	rejected, err := DecideMerge(root, "issue-2")
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Status != "needs_rework" || rejected.Decision != "NEEDS_REWORK" {
		t.Fatalf("unexpected rejected decision: %+v", rejected)
	}
}

func writeIssueState(t *testing.T, root string, state orchestrator.IssueState) {
	t.Helper()
	path := workspace.ForRoot(root).OrchestratorDir + "/issue-states/" + state.IssueID + ".json"
	if err := fsutil.WriteJSON(path, state); err != nil {
		t.Fatal(err)
	}
}

func writeQualityReport(t *testing.T, root string, report quality.Report) {
	t.Helper()
	path := workspace.ForRoot(root).QualityDir + "/reports/" + report.ID + ".json"
	if err := fsutil.WriteJSON(path, report); err != nil {
		t.Fatal(err)
	}
}
