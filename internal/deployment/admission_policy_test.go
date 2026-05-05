package deployment

import (
	"path/filepath"
	"testing"

	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/workspace"
)

func TestDefaultReleaseAdmissionPolicyMatchesPhase16Decisions(t *testing.T) {
	pack := DefaultReleaseAdmissionPolicyPack()
	cases := []struct {
		name          string
		admission     ReleaseAdmission
		wantStatus    string
		wantDecision  string
		wantReason    string
		wantRuleID    string
		wantRuleCount int
	}{
		{
			name: "monitor critical blocks",
			admission: ReleaseAdmission{Signals: []AdmissionSignal{
				{Type: "deployment_rehearsal", ID: "rehearsal-critical", Status: "attention_required", Decision: "DEPLOYMENT_REHEARSAL_ATTENTION_REQUIRED"},
				{Type: "monitor_summary", ID: "monitor-critical", Status: "critical", Decision: "DEPLOYMENT_MONITOR_CRITICAL"},
			}},
			wantStatus:    "blocked",
			wantDecision:  "RELEASE_ADMISSION_BLOCKED",
			wantReason:    "monitor_critical",
			wantRuleID:    "monitor_critical",
			wantRuleCount: 2,
		},
		{
			name: "failed execution blocks",
			admission: ReleaseAdmission{Signals: []AdmissionSignal{
				{Type: "deployment_rehearsal", ID: "rehearsal-failed", Status: "attention_required", Decision: "DEPLOYMENT_REHEARSAL_ATTENTION_REQUIRED", Reason: "deployment_execution:failed"},
				{Type: "monitor_summary", ID: "monitor-attention", Status: "attention_required", Decision: "DEPLOYMENT_MONITOR_ATTENTION_REQUIRED"},
			}},
			wantStatus:    "blocked",
			wantDecision:  "RELEASE_ADMISSION_BLOCKED",
			wantReason:    "deployment_execution_failed",
			wantRuleID:    "deployment_execution_failed",
			wantRuleCount: 3,
		},
		{
			name: "rollback preview requires manual review",
			admission: ReleaseAdmission{Signals: []AdmissionSignal{
				{Type: "deployment_rehearsal", ID: "rehearsal-rollback", Status: "attention_required", Decision: "DEPLOYMENT_REHEARSAL_ATTENTION_REQUIRED", Reason: "rollback_required"},
				{Type: "monitor_summary", ID: "monitor-healthy", Status: "healthy", Decision: "DEPLOYMENT_MONITOR_HEALTHY"},
				{Type: "rollback_preview", ID: "rollback-preview", Status: "completed", Decision: "ROLLBACK_PREVIEW_READY"},
			}},
			wantStatus:    "manual_required",
			wantDecision:  "RELEASE_ADMISSION_MANUAL_REVIEW_REQUIRED",
			wantReason:    "rollback_required",
			wantRuleID:    "rollback_required",
			wantRuleCount: 2,
		},
		{
			name: "healthy rehearsal allows release",
			admission: ReleaseAdmission{Signals: []AdmissionSignal{
				{Type: "deployment_rehearsal", ID: "rehearsal-ok", Status: "completed", Decision: "DEPLOYMENT_REHEARSAL_READY"},
				{Type: "monitor_summary", ID: "monitor-ok", Status: "healthy", Decision: "DEPLOYMENT_MONITOR_HEALTHY"},
			}},
			wantStatus:    "allowed",
			wantDecision:  "RELEASE_ADMISSION_ALLOWED",
			wantReason:    "release_admission_allowed",
			wantRuleCount: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateReleaseAdmissionPolicy(pack, tc.admission)
			if got.Status != tc.wantStatus || got.Decision != tc.wantDecision {
				t.Fatalf("unexpected admission decision: %+v", got)
			}
			if !containsReason(got.Reasons, tc.wantReason) {
				t.Fatalf("expected reason %q, got %+v", tc.wantReason, got.Reasons)
			}
			if got.PolicyID != defaultReleaseAdmissionPolicyID || got.PolicyDecision.PolicyID != defaultReleaseAdmissionPolicyID {
				t.Fatalf("expected default policy metadata, got %+v", got)
			}
			if got.PolicyDecision.MatchedRuleCount != tc.wantRuleCount || len(got.MatchedRules) != tc.wantRuleCount {
				t.Fatalf("expected %d matched rules, got %+v", tc.wantRuleCount, got.MatchedRules)
			}
			if tc.wantRuleID != "" && !hasAdmissionRuleMatch(got.MatchedRules, tc.wantRuleID) {
				t.Fatalf("expected matched rule %q, got %+v", tc.wantRuleID, got.MatchedRules)
			}
		})
	}
}

func TestReleaseAdmissionPolicyPackCanAddStrictEnvironmentRules(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WriteText(filepath.Join(workspace.ForRoot(root).MoyuanDir, "policies", "release.yaml"), `release_admission_policy_pack:
  id: strict-release-admission-v1
  version: "2026-05-05"
  environments:
    production:
      require_signal_types: ["deployment_rehearsal", "monitor_summary", "candidate_deployment_feedback"]
      missing_required_signal_effect: block
      monitor_unknown_effect: block
  rules:
    - id: candidate_warning_requires_release_review
      signal_type: candidate_deployment_feedback
      severity_in: ["warning"]
      effect: manual
      reason: candidate_warning_requires_release_review
`); err != nil {
		t.Fatal(err)
	}
	pack, err := LoadReleaseAdmissionPolicyPack(root, "production")
	if err != nil {
		t.Fatal(err)
	}
	if pack.ID != "strict-release-admission-v1" || pack.Source != "configured" {
		t.Fatalf("expected configured policy pack, got %+v", pack)
	}
	blocked := EvaluateReleaseAdmissionPolicy(pack, ReleaseAdmission{
		Environment: "production",
		Signals: []AdmissionSignal{
			{Type: "deployment_rehearsal", ID: "rehearsal-prod", Status: "completed", Decision: "DEPLOYMENT_REHEARSAL_READY"},
			{Type: "monitor_summary", ID: "monitor-prod", Status: "healthy", Decision: "DEPLOYMENT_MONITOR_HEALTHY"},
		},
	})
	if blocked.Status != "blocked" || !containsReason(blocked.Reasons, "required_signal_missing:candidate_deployment_feedback") {
		t.Fatalf("expected strict production policy to block missing candidate feedback, got %+v", blocked)
	}

	manual := EvaluateReleaseAdmissionPolicy(pack, ReleaseAdmission{
		Environment: "production",
		Signals: []AdmissionSignal{
			{Type: "deployment_rehearsal", ID: "rehearsal-prod", Status: "completed", Decision: "DEPLOYMENT_REHEARSAL_READY"},
			{Type: "monitor_summary", ID: "monitor-prod", Status: "healthy", Decision: "DEPLOYMENT_MONITOR_HEALTHY"},
			{Type: "candidate_deployment_feedback", ID: "feedback-prod", Status: "passed", Decision: "CANDIDATE_DEPLOYMENT_HEALTHY", Severity: "warning"},
		},
	})
	if manual.Status != "manual_required" || !hasAdmissionRuleMatch(manual.MatchedRules, "candidate_warning_requires_release_review") {
		t.Fatalf("expected configured warning rule to require manual review, got %+v", manual)
	}
}

func hasAdmissionRuleMatch(matches []AdmissionRuleMatch, ruleID string) bool {
	for _, match := range matches {
		if match.RuleID == ruleID {
			return true
		}
	}
	return false
}
