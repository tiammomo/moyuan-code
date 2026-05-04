package evidence

import (
	"testing"

	"moyuan-code/internal/workspace"
)

func TestEvidenceAddListLoad(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	record, err := Add(root, AddOptions{
		ParentType:  "deployment_execution",
		ParentID:    "deploy-exec-1",
		SubjectType: "deployment",
		SubjectID:   "deployment-api",
		Operation:   "deployment.execute.local_shell",
		Status:      "completed",
		Decision:    "DEPLOY_EXECUTION_COMPLETED",
		Reasons:     []string{"smoke_passed"},
		Source:      "deployment",
		Artifacts:   []ArtifactRef{{Kind: "deployment_execution", ID: "deploy-exec-1", Path: ".moyuan/lifecycle/deployments/executions/deploy-exec-1.json"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.ID == "" || record.ParentType != "deployment_execution" || record.Operation != "deployment.execute.local_shell" {
		t.Fatalf("unexpected evidence record: %+v", record)
	}
	list, err := List(root, ListOptions{ParentType: "deployment_execution", ParentID: "deploy-exec-1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != record.ID {
		t.Fatalf("expected evidence list to include record, got %+v", list)
	}
	loaded, found, err := Load(root, record.ID)
	if err != nil || !found || loaded.ID != record.ID {
		t.Fatalf("expected evidence load, found=%v record=%+v err=%v", found, loaded, err)
	}
	if _, found, err := Load(root, "../bad"); err != nil || found {
		t.Fatalf("expected invalid evidence id to be ignored, found=%v err=%v", found, err)
	}
}
