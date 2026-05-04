package approvals

import (
	"testing"
)

func TestApprovalLifecycle(t *testing.T) {
	root := t.TempDir()
	record, err := Request(root, RequestOptions{
		TargetType:  "deployment",
		TargetID:    "deployment-prod",
		Action:      "deploy.production",
		RiskLevel:   "critical",
		RequestedBy: "owner",
		Reason:      "production deployment requires manual approval",
		Metadata:    map[string]any{"environment": "production"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != "pending" || record.Decision != "APPROVAL_PENDING" {
		t.Fatalf("expected pending approval, got %+v", record)
	}

	list, err := List(root, ListOptions{Status: "pending", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != record.ID {
		t.Fatalf("expected pending approval in list, got %+v", list)
	}

	decided, found, err := Decide(root, record.ID, DecisionOptions{Decision: "approved", DecidedBy: "release-manager", Reason: "reviewed release gate"})
	if err != nil {
		t.Fatal(err)
	}
	if !found || decided.Status != "approved" || decided.Decision != "APPROVAL_APPROVED" {
		t.Fatalf("expected approved record, got found=%v record=%+v", found, decided)
	}
	if _, _, err := Decide(root, record.ID, DecisionOptions{Decision: "approved"}); err == nil {
		t.Fatal("expected already decided approval to fail")
	}
	if _, _, err := Decide(root, "../bad", DecisionOptions{Decision: "approved"}); !IsInvalidIDError(err) {
		t.Fatalf("expected invalid approval id to fail, got %v", err)
	}
}

func TestApprovalRejectsSensitivePayload(t *testing.T) {
	root := t.TempDir()
	if _, err := Request(root, RequestOptions{TargetType: "provider", TargetID: "glm", Action: "provider.probe", Reason: "token=plain"}); err == nil {
		t.Fatal("expected secret-bearing reason to be rejected")
	}
	if _, err := Request(root, RequestOptions{
		TargetType: "provider",
		TargetID:   "glm",
		Action:     "provider.probe",
		Metadata:   map[string]any{"api_key": "sk-plainsecret"},
	}); err == nil {
		t.Fatal("expected secret-bearing metadata to be rejected")
	}
}
