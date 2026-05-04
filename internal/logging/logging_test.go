package logging

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestListAggregatesFiltersAndRedactsEvents(t *testing.T) {
	root := t.TempDir()
	if err := Log(root, "audit", "provider.upserted", map[string]any{
		"issue_id": "issue-1",
		"token":    "sk-plainsecret123456",
		"reason":   "password=plain should not leak",
	}); err != nil {
		t.Fatal(err)
	}
	if err := Log(root, "run", "orchestrator.run.transitioned", map[string]any{
		"issue_id": "issue-1",
		"run_id":   "run-1",
		"status":   "completed",
	}); err != nil {
		t.Fatal(err)
	}

	records, err := List(root, Query{Stream: "all", IssueID: "issue-1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d: %+v", len(records), records)
	}
	encoded, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	if strings.Contains(body, "plainsecret") || strings.Contains(body, "password=plain") {
		t.Fatalf("audit records leaked sensitive value: %s", body)
	}
	if !strings.Contains(body, `"channel":"audit"`) || !strings.Contains(body, `"channel":"run"`) {
		t.Fatalf("expected audit and run channels: %s", body)
	}

	auditRecords, err := List(root, Query{Stream: "audit", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(auditRecords) != 1 || auditRecords[0].Event != "provider.upserted" {
		t.Fatalf("expected audit provider event, got %+v", auditRecords)
	}
}

func TestListRejectsUnsafeStreamName(t *testing.T) {
	_, err := List(t.TempDir(), Query{Stream: "../audit"})
	if !IsInvalidStreamError(err) {
		t.Fatalf("expected invalid stream error, got %v", err)
	}
	if err := Log(t.TempDir(), "../audit", "bad.event", map[string]any{}); !IsInvalidStreamError(err) {
		t.Fatalf("expected unsafe log stream to be rejected, got %v", err)
	}
}
