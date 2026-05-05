package operations

import (
	"context"
	"testing"

	"moyuan-code/internal/deployment"
	"moyuan-code/internal/release"
	"moyuan-code/internal/workspace"
)

func TestLoadAggregatesReleaseProviderExecutionEvidence(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	plan, err := release.Suggest(context.Background(), root, release.SuggestOptions{Version: "v0.1.0", MinIssues: 1})
	if err != nil {
		t.Fatal(err)
	}
	execution, found, err := release.ProviderPreview(root, plan.ID)
	if err != nil || !found {
		t.Fatalf("expected release provider execution, found=%v err=%v", found, err)
	}

	detail, found, err := Load(root, "release_provider", execution.ID)
	if err != nil || !found {
		t.Fatalf("expected operation detail, found=%v err=%v", found, err)
	}
	if detail.OperationType != "release_provider" || detail.Operation != "release.provider.preview" || detail.PrimaryRef != plan.ID {
		t.Fatalf("unexpected release provider detail: %+v", detail)
	}
	if detail.Summary.EvidenceCount != 1 || detail.Summary.ArtifactCount != 1 || len(detail.Evidence) != 1 || len(detail.Artifacts) != 1 {
		t.Fatalf("expected evidence and artifact aggregation, got %+v", detail)
	}
}

func TestLoadAggregatesDeploymentExecutionEvidence(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	execution, err := deployment.Execute(context.Background(), root, deployment.ExecuteOptions{DeploymentID: "missing-deployment", Mode: "dry_run"})
	if err != nil {
		t.Fatal(err)
	}

	detail, found, err := Load(root, "deployment", execution.ID)
	if err != nil || !found {
		t.Fatalf("expected deployment detail, found=%v err=%v", found, err)
	}
	if detail.OperationType != "deployment" || detail.Operation != "deployment.execute.dry_run" || detail.PrimaryRef != "missing-deployment" {
		t.Fatalf("unexpected deployment detail: %+v", detail)
	}
	if detail.Summary.StepCount != len(execution.Steps) || detail.Summary.EvidenceCount != 1 || detail.Summary.ArtifactCount != 1 {
		t.Fatalf("expected deployment summary counts, got detail=%+v execution=%+v", detail, execution)
	}
}

func TestLoadEvidenceDetailAndUnsupportedType(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	execution, err := deployment.Execute(context.Background(), root, deployment.ExecuteOptions{DeploymentID: "missing-deployment", Mode: "dry_run"})
	if err != nil {
		t.Fatal(err)
	}
	detail, found, err := Load(root, "deployment", execution.ID)
	if err != nil || !found || len(detail.Evidence) == 0 {
		t.Fatalf("expected deployment evidence, found=%v err=%v detail=%+v", found, err, detail)
	}

	evidenceDetail, found, err := Load(root, "evidence", detail.Evidence[0].ID)
	if err != nil || !found {
		t.Fatalf("expected evidence detail, found=%v err=%v", found, err)
	}
	if evidenceDetail.OperationType != "evidence" || evidenceDetail.Summary.EvidenceCount != 1 {
		t.Fatalf("unexpected evidence detail: %+v", evidenceDetail)
	}
	if _, found, err := Load(root, "visual_render", "missing"); err != nil || found {
		t.Fatalf("unsupported operation type should not be found, found=%v err=%v", found, err)
	}
}
