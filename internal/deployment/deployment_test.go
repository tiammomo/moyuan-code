package deployment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/serverresources"
	"moyuan-code/internal/workspace"
)

func TestExecuteRunsSmokeMonitorAndSuggestsRollback(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}

	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer okServer.Close()
	okPlan := createDeploymentPlanWithHealthTarget(t, root, "smoke-ok", okServer.URL)
	okExecution, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: okPlan.ID,
		Mode:         "local_shell",
		Approved:     true,
		Commands:     []string{"printf deploy-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if okExecution.Decision != "DEPLOY_EXECUTION_COMPLETED" || okExecution.SmokeReport.Status != "passed" || okExecution.MonitorReport.Status != "passed" || okExecution.RollbackSuggestion.Required {
		t.Fatalf("expected successful smoke and monitor, got %+v", okExecution)
	}

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer failServer.Close()
	failPlan := createDeploymentPlanWithHealthTarget(t, root, "smoke-fail", failServer.URL)
	failedExecution, err := Execute(context.Background(), root, ExecuteOptions{
		DeploymentID: failPlan.ID,
		Mode:         "local_shell",
		Approved:     true,
		Commands:     []string{"printf deploy-fail"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if failedExecution.Decision != "DEPLOY_SMOKE_FAILED" || failedExecution.SmokeReport.Status != "failed" || !failedExecution.RollbackSuggestion.Required {
		t.Fatalf("expected smoke failure rollback suggestion, got %+v", failedExecution)
	}
	if !hasStep(failedExecution.Steps, "rollback", "suggested") {
		t.Fatalf("expected rollback suggestion step: %+v", failedExecution.Steps)
	}
	releaseLog, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(root).LogsDir, "release.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found ||
		!strings.Contains(releaseLog, "deployment.smoke.completed") ||
		!strings.Contains(releaseLog, "deployment.monitor.completed") ||
		!strings.Contains(releaseLog, "deployment.rollback.suggested") {
		t.Fatalf("expected smoke, monitor, and rollback logs, found=%v log=%s", found, releaseLog)
	}
}

func createDeploymentPlanWithHealthTarget(t *testing.T, root string, id string, target string) Plan {
	t.Helper()
	resource, err := serverresources.Add(root, serverresources.Resource{
		ID:          id,
		Environment: "test_dev",
		Host:        "127.0.0.1",
		Provider:    "local_vm",
		Owner:       "devops",
		AuthRef:     "env:DEV_SERVER_SSH_KEY",
		Healthcheck: serverresources.Healthcheck{Type: "http", Target: target},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := finish(root, Plan{
		ID:          "deployment-" + id,
		ReleaseID:   "release-" + id,
		Environment: "test_dev",
		Status:      "planned",
		Decision:    "DEPLOY_PLAN_READY",
		Reasons:     []string{"test_fixture"},
		Resources: []ResourceSummary{{
			ID:          resource.ID,
			Environment: resource.Environment,
			Host:        resource.Host,
			Status:      resource.Status,
		}},
		SmokePlan:    StepPlan{Status: "planned", Required: true, Actions: []string{"http smoke"}},
		MonitorPlan:  StepPlan{Status: "planned", Required: true, Window: "1m", Actions: []string{"http monitor"}},
		RollbackPlan: StepPlan{Status: "planned", Required: true, Actions: []string{"rollback release-" + id}},
		CreatedAt:    "2026-05-05T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func hasStep(steps []ExecutionStep, name string, status string) bool {
	for _, step := range steps {
		if step.Name == name && step.Status == status {
			return true
		}
	}
	return false
}
