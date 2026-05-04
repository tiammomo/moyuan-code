package workspace

import (
	"path/filepath"
	"testing"

	"moyuan-code/internal/fsutil"
)

func TestValidateReportsWorkspaceSchemaStatus(t *testing.T) {
	root := t.TempDir()
	if _, err := Ensure(root); err != nil {
		t.Fatal(err)
	}

	report, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || len(report.Issues) != 0 {
		t.Fatalf("expected valid workspace, got %+v", report)
	}

	if err := fsutil.WriteJSON(filepath.Join(root, DirName, "workspace.json"), map[string]any{
		"schema_ver": 1,
		"project": map[string]any{
			"schema_version": 1,
			"project": map[string]any{
				"name": "missing-id",
				"root": ".",
				"type": "single-repo",
			},
		},
		"repository": DefaultRepositoryConfig(root),
		"access":     DefaultAccessConfig(),
	}); err != nil {
		t.Fatal(err)
	}

	report, err = Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" || !hasValidationIssue(report, "project_id_required") {
		t.Fatalf("expected project_id_required validation failure, got %+v", report)
	}
}

func hasValidationIssue(report ValidationReport, code string) bool {
	for _, issue := range report.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
