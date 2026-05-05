package serverresources

import (
	"path/filepath"
	"testing"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

func TestDefaultMaintenancePolicyExplainsWindowAndManualDecisions(t *testing.T) {
	pack := DefaultMaintenancePolicyPack()
	cases := []struct {
		name        string
		context     MaintenancePolicyContext
		wantStatus  string
		wantReason  string
		wantBlocked bool
		wantManual  bool
	}{
		{
			name:       "test dev health scan allowed",
			context:    MaintenancePolicyContext{Environment: "test_dev", Action: "health-scan", RequestedAt: "2026-05-05"},
			wantStatus: "allowed",
			wantReason: "maintenance_policy_allowed",
		},
		{
			name:       "production deploy without configured window requires manual",
			context:    MaintenancePolicyContext{Environment: "production", Action: "deploy", RequestedAt: "2026-05-05"},
			wantStatus: "manual_required",
			wantReason: "maintenance_window_missing",
			wantManual: true,
		},
		{
			name:        "unknown action blocks",
			context:     MaintenancePolicyContext{Environment: "test_dev", Action: "format_disk", RequestedAt: "2026-05-05"},
			wantStatus:  "blocked",
			wantReason:  "action_not_allowed:format_disk",
			wantBlocked: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := EvaluateMaintenancePolicy(pack, tc.context)
			if decision.Status != tc.wantStatus || decision.Blocked != tc.wantBlocked || decision.ManualRequired != tc.wantManual {
				t.Fatalf("unexpected decision: %+v", decision)
			}
			if !hasMaintenanceReason(decision.Reasons, tc.wantReason) {
				t.Fatalf("expected reason %q, got %+v", tc.wantReason, decision.Reasons)
			}
			if decision.PolicyID != defaultMaintenancePolicyID || decision.PolicySource != "builtin" {
				t.Fatalf("expected builtin policy metadata, got %+v", decision)
			}
		})
	}
}

func TestConfiguredMaintenancePolicyAddsWindowsAndFreeze(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteText(filepath.Join(workspace.ForRoot(root).MoyuanDir, "policies", "server-resources.yaml"), `maintenance_policy_pack:
  id: strict-maintenance-v1
  version: "2026-05-05"
  environments:
    production:
      maintenance_windows: ["2026-05-05..2026-05-06"]
      freeze_windows: ["2026-05-06"]
      allowed_actions: ["deploy", "rollback", "health_scan"]
      manual_required_actions: ["deploy"]
      outside_window_effect: allow
`); err != nil {
		t.Fatal(err)
	}
	pack, err := LoadMaintenancePolicyPack(root, "production")
	if err != nil {
		t.Fatal(err)
	}
	if pack.ID != "strict-maintenance-v1" || pack.Source != "configured" {
		t.Fatalf("expected configured maintenance policy, got %+v", pack)
	}
	manual := EvaluateMaintenancePolicy(pack, MaintenancePolicyContext{Environment: "production", Action: "deploy", RequestedAt: "2026-05-05"})
	if manual.Status != "manual_required" || !manual.WithinMaintenanceWindow || !hasMaintenanceReason(manual.Reasons, "manual_required_action:deploy") {
		t.Fatalf("expected deploy to require manual review inside configured window, got %+v", manual)
	}
	blocked := EvaluateMaintenancePolicy(pack, MaintenancePolicyContext{Environment: "production", Action: "health_scan", RequestedAt: "2026-05-06"})
	if blocked.Status != "blocked" || !blocked.InFreezeWindow || !hasMaintenanceReason(blocked.Reasons, "freeze_window_active") {
		t.Fatalf("expected freeze window to block health scan, got %+v", blocked)
	}
	outside := EvaluateMaintenancePolicy(pack, MaintenancePolicyContext{Environment: "production", Action: "rollback", RequestedAt: "2026-05-07"})
	if outside.Status != "manual_required" || !hasMaintenanceReason(outside.Reasons, "outside_maintenance_window") {
		t.Fatalf("custom outside allow must not lower default manual boundary, got %+v", outside)
	}
}

func hasMaintenanceReason(reasons []string, want string) bool {
	for _, reason := range reasons {
		if reason == want {
			return true
		}
	}
	return false
}
