package serverresources

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

func TestLifecycleScanCreatesAlertsMaintenanceAndAudit(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	expiringDate := now.AddDate(0, 0, 5).Format("2006-01-02")
	expiredDate := now.AddDate(0, 0, -1).Format("2006-01-02")
	dueDate := now.AddDate(0, 0, -1).Format("2006-01-02")
	if _, err := Add(root, Resource{ID: "dev-expiring", Environment: "test_dev", Host: "127.0.0.1", Provider: "local_vm", Owner: "ops", AuthRef: "env:DEV_SSH", ExpiresAt: expiringDate}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(root, Resource{ID: "prod-expired", Environment: "production", Host: "prod.internal", Provider: "aliyun", Owner: "ops", AuthRef: "secret:PROD_SSH", ExpiresAt: expiredDate}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(root, Resource{ID: "staging-due", Environment: "staging", Host: "staging.internal", Provider: "aliyun", Owner: "ops", AuthRef: "secret:STAGING_SSH", ExpiresAt: now.AddDate(0, 2, 0).Format("2006-01-02"), MaintenanceWindow: "due:" + dueDate}); err != nil {
		t.Fatal(err)
	}

	report, err := LifecycleScan(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "attention_required" || report.Decision != "RESOURCE_LIFECYCLE_ATTENTION_REQUIRED" || len(report.Alerts) != 3 {
		t.Fatalf("unexpected lifecycle scan: %+v", report)
	}
	if !hasLifecycleAlert(report.Alerts, "dev-expiring", "expiration", "RESOURCE_EXPIRING") ||
		!hasLifecycleAlert(report.Alerts, "prod-expired", "expiration", "RESOURCE_EXPIRED") ||
		!hasLifecycleAlert(report.Alerts, "staging-due", "maintenance_due", "RESOURCE_MAINTENANCE_DUE") {
		t.Fatalf("expected expiration and maintenance due alerts, got %+v", report.Alerts)
	}
	alerts, err := ListLifecycleAlerts(root, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 3 {
		t.Fatalf("expected lifecycle alerts to be queryable, got %+v", alerts)
	}
	records, err := ListMaintenance(root, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("expected lifecycle alerts to write maintenance records, got %+v", records)
	}
	audit, found, err := fsutil.ReadText(filepath.Join(workspace.ForRoot(root).LogsDir, "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !found || !strings.Contains(audit, "server_resource.lifecycle_scan") {
		t.Fatalf("expected lifecycle scan audit log, found=%v audit=%s", found, audit)
	}
}

func hasLifecycleAlert(alerts []LifecycleAlert, resourceID string, alertType string, decision string) bool {
	for _, alert := range alerts {
		if alert.ResourceID == resourceID && alert.Type == alertType && alert.Decision == decision {
			return true
		}
	}
	return false
}
