package operations

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/deployment"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/release"
	"moyuan-code/internal/serverresources"
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

func TestTimelineAggregatesOperationsAndFilters(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	plan, err := release.Suggest(context.Background(), root, release.SuggestOptions{Version: "v0.2.0", MinIssues: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := release.ProviderPreview(root, plan.ID); err != nil || !found {
		t.Fatalf("expected provider preview, found=%v err=%v", found, err)
	}
	if _, err := serverresources.Add(root, serverresources.Resource{
		ID:          "dev-api",
		Environment: "test_dev",
		Host:        "127.0.0.1",
		Provider:    "local_vm",
		Owner:       "ops",
		AuthRef:     "env:DEV_SERVER_SSH_KEY",
		ExpiresAt:   "2000-01-01",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := serverresources.HealthScan(context.Background(), root, serverresources.HealthScanOptions{Environment: "test_dev"}); err != nil {
		t.Fatal(err)
	}
	if _, err := serverresources.LifecycleScan(root); err != nil {
		t.Fatal(err)
	}
	if _, err := serverresources.MaintenanceScan(root); err != nil {
		t.Fatal(err)
	}
	if err := serverresources.RecordDeploymentReference(root, serverresources.DeploymentRef{
		ResourceID:   "dev-api",
		Kind:         "deployment_plan",
		DeploymentID: "deployment-resource-ref",
		ReleaseID:    plan.ID,
		Environment:  "test_dev",
		Status:       "planned",
		Decision:     "DEPLOY_PLAN_READY",
	}); err != nil {
		t.Fatal(err)
	}
	execution, err := deployment.Execute(context.Background(), root, deployment.ExecuteOptions{DeploymentID: "missing-deployment", Mode: "dry_run"})
	if err != nil {
		t.Fatal(err)
	}
	summary, err := deployment.BuildMonitorSummary(root, deployment.MonitorSummaryOptions{Environment: "test_dev", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	verification, err := deployment.BuildPostDeploymentVerification(root, deployment.PostDeploymentVerificationOptions{ExecutionID: execution.ID, Environment: "test_dev", MonitorLimit: 5})
	if err != nil {
		t.Fatal(err)
	}
	rehearsal, err := deployment.BuildRehearsal(context.Background(), root, deployment.RehearsalOptions{ExecutionID: execution.ID, Environment: "test_dev"})
	if err != nil {
		t.Fatal(err)
	}
	schedulerRun, err := deployment.RunRehearsalScheduler(context.Background(), root, deployment.RehearsalSchedulerOptions{ExecutionID: execution.ID, Environment: "test_dev", MaxTargets: 1})
	if err != nil {
		t.Fatal(err)
	}
	admission, err := deployment.BuildReleaseAdmission(context.Background(), root, deployment.ReleaseAdmissionOptions{RehearsalID: rehearsal.ID})
	if err != nil {
		t.Fatal(err)
	}
	handoff := deploymentRiskHandoffTimeline{
		ID:             "deployment-risk-handoff-test",
		SourceType:     "release_admission",
		SourceID:       admission.ID,
		Status:         "review_required",
		Decision:       "DEPLOYMENT_RISK_HANDOFF_REVIEW_REQUIRED",
		FailureClass:   "release_admission_blocked",
		EvidenceRefs:   append([]string{}, admission.EvidenceIDs...),
		Reasons:        append([]string{}, admission.Reasons...),
		ReviewRequired: true,
		CreatedAt:      admission.CreatedAt,
	}
	if err := fsutil.WriteJSON(filepath.Join(deploymentRiskHandoffTimelineDir(root), handoff.ID+".json"), handoff); err != nil {
		t.Fatal(err)
	}
	review := deploymentRiskReviewTimeline{
		ID:           "deployment-risk-review-test",
		HandoffID:    handoff.ID,
		SourceType:   handoff.SourceType,
		SourceID:     handoff.SourceID,
		Decision:     "approved",
		Status:       "completed",
		ReviewerID:   "qa",
		FailureClass: handoff.FailureClass,
		EvidenceRefs: append([]string{}, handoff.EvidenceRefs...),
		CreatedAt:    handoff.CreatedAt,
	}
	if err := fsutil.WriteJSON(filepath.Join(deploymentRiskReviewTimelineDir(root), review.ID+".json"), review); err != nil {
		t.Fatal(err)
	}

	items, err := Timeline(root, TimelineOptions{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range []string{
		"release_provider_execution",
		"deployment_execution",
		"deployment_monitor_summary",
		"post_deployment_verification",
		"deployment_rehearsal",
		"release_admission",
		"rehearsal_scheduler_run",
		"deployment_risk_handoff",
		"deployment_risk_review",
		"resource_health_scan",
		"resource_maintenance",
		"resource_lifecycle_alert",
		"resource_deployment_ref",
		"server_resource",
	} {
		if !timelineContainsType(items, typ) {
			t.Fatalf("expected timeline type %s in %+v", typ, items)
		}
	}
	filtered, err := Timeline(root, TimelineOptions{Type: "release-admission", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) == 0 || filtered[0].ID != admission.ID || filtered[0].Type != "release_admission" || len(filtered[0].EvidenceRefs) == 0 {
		t.Fatalf("expected filtered admission with evidence refs, got %+v", filtered)
	}
	envFiltered, err := Timeline(root, TimelineOptions{Environment: "test_dev", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if !timelineContainsID(envFiltered, schedulerRun.ID) || !timelineContainsID(envFiltered, summary.ID) {
		t.Fatalf("expected environment filtered operations, got %+v", envFiltered)
	}
	if !timelineContainsID(envFiltered, verification.ID) {
		t.Fatalf("expected verification in environment filtered operations, got %+v", envFiltered)
	}
}

func TestExportAuditBuildsMarkdownAndRedactsSecrets(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := serverresources.Add(root, serverresources.Resource{
		ID:          "dev-audit",
		Environment: "test_dev",
		Host:        "127.0.0.1",
		Provider:    "local_vm",
		Owner:       "ops",
		AuthRef:     "env:DEV_SERVER_SSH_KEY",
		ExpiresAt:   "2099-01-01",
	}); err != nil {
		t.Fatal(err)
	}
	if err := serverresources.RecordDeploymentReference(root, serverresources.DeploymentRef{
		ResourceID:   "dev-audit",
		Kind:         "deployment_plan",
		DeploymentID: "deployment-audit-ref",
		Environment:  "test_dev",
		Status:       "planned",
		Decision:     "DEPLOY_PLAN_READY",
	}); err != nil {
		t.Fatal(err)
	}
	execution, err := deployment.Execute(context.Background(), root, deployment.ExecuteOptions{DeploymentID: "missing-deployment", Mode: "dry_run"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := deployment.BuildMonitorSummary(root, deployment.MonitorSummaryOptions{Environment: "test_dev", Limit: 5}); err != nil {
		t.Fatal(err)
	}
	verification, err := deployment.BuildPostDeploymentVerification(root, deployment.PostDeploymentVerificationOptions{ExecutionID: execution.ID, Environment: "test_dev", MonitorLimit: 5})
	if err != nil {
		t.Fatal(err)
	}
	handoff := deploymentRiskHandoffTimeline{
		ID:             "deployment-risk-handoff-secret",
		SourceType:     "post_deployment_verification",
		SourceID:       verification.ID,
		Status:         "review_required",
		Decision:       "DEPLOYMENT_RISK_HANDOFF_REVIEW_REQUIRED",
		FailureClass:   "post_deployment_attention",
		Reasons:        []string{"token=plain-secret should be redacted"},
		ReviewRequired: true,
		CreatedAt:      verification.CreatedAt,
	}
	if err := fsutil.WriteJSON(filepath.Join(deploymentRiskHandoffTimelineDir(root), handoff.ID+".json"), handoff); err != nil {
		t.Fatal(err)
	}

	report, err := ExportAudit(root, AuditExportOptions{Format: "markdown", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if report.Format != "markdown" || report.Markdown == "" || !strings.Contains(report.Markdown, "Operations Audit Export") {
		t.Fatalf("expected markdown audit export, got %+v", report)
	}
	if report.Summary.TimelineItemCount == 0 || report.Summary.PostDeploymentVerificationCount == 0 || report.Summary.ResourceDeploymentRefCount == 0 {
		t.Fatalf("expected timeline, verification and resource refs in report: %+v", report.Summary)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "plain-secret") {
		t.Fatalf("audit export leaked secret-like text: %s", string(encoded))
	}
	if !report.Summary.RedactionApplied {
		t.Fatalf("expected redaction flag in summary: %+v", report.Summary)
	}
	typeOnly, err := ExportAudit(root, AuditExportOptions{Type: "resource-deployment-ref", Environment: "test_dev", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(typeOnly.Timeline) == 0 || typeOnly.Timeline[0].Type != "resource_deployment_ref" || len(typeOnly.PostDeploymentVerifications) != 0 {
		t.Fatalf("expected type filtered resource deployment refs only, got %+v", typeOnly)
	}
}

func TestBuildDecisionLedgerAggregatesDecisionSources(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	if _, err := serverresources.Add(root, serverresources.Resource{
		ID:          "dev-ledger",
		Environment: "test_dev",
		Host:        "127.0.0.1",
		Provider:    "local_vm",
		Owner:       "ops",
		AuthRef:     "env:DEV_SERVER_SSH_KEY",
		ExpiresAt:   "2000-01-01",
	}); err != nil {
		t.Fatal(err)
	}
	execution, err := deployment.Execute(context.Background(), root, deployment.ExecuteOptions{DeploymentID: "missing-deployment", Mode: "dry_run"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := deployment.BuildMonitorSummary(root, deployment.MonitorSummaryOptions{Environment: "test_dev", Limit: 5}); err != nil {
		t.Fatal(err)
	}
	verification, err := deployment.BuildPostDeploymentVerification(root, deployment.PostDeploymentVerificationOptions{ExecutionID: execution.ID, Environment: "test_dev", MonitorLimit: 5})
	if err != nil {
		t.Fatal(err)
	}
	rehearsal, err := deployment.BuildRehearsal(context.Background(), root, deployment.RehearsalOptions{ExecutionID: execution.ID, Environment: "test_dev"})
	if err != nil {
		t.Fatal(err)
	}
	admission, err := deployment.BuildReleaseAdmission(context.Background(), root, deployment.ReleaseAdmissionOptions{RehearsalID: rehearsal.ID})
	if err != nil {
		t.Fatal(err)
	}
	handoff := deploymentRiskHandoffTimeline{
		ID:             "deployment-risk-handoff-ledger",
		SourceType:     "release_admission",
		SourceID:       admission.ID,
		Status:         "review_required",
		Decision:       "DEPLOYMENT_RISK_HANDOFF_REVIEW_REQUIRED",
		FailureClass:   "release_admission_blocked",
		EvidenceRefs:   append([]string{}, admission.EvidenceIDs...),
		Reasons:        append([]string{}, admission.Reasons...),
		ReviewRequired: true,
		CreatedAt:      admission.CreatedAt,
	}
	if err := fsutil.WriteJSON(filepath.Join(deploymentRiskHandoffTimelineDir(root), handoff.ID+".json"), handoff); err != nil {
		t.Fatal(err)
	}
	review := deploymentRiskReviewTimeline{
		ID:           "deployment-risk-review-ledger",
		HandoffID:    handoff.ID,
		SourceType:   handoff.SourceType,
		SourceID:     handoff.SourceID,
		Decision:     "approved",
		Status:       "completed",
		ReviewerID:   "qa",
		Reason:       "ledger test",
		NextStep:     "repair_plan",
		FailureClass: handoff.FailureClass,
		EvidenceRefs: append([]string{}, handoff.EvidenceRefs...),
		CreatedAt:    admission.CreatedAt,
	}
	if err := fsutil.WriteJSON(filepath.Join(deploymentRiskReviewTimelineDir(root), review.ID+".json"), review); err != nil {
		t.Fatal(err)
	}

	ledger, err := BuildDecisionLedger(root, DecisionLedgerOptions{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	for _, sourceType := range []string{"release_admission", "maintenance_policy", "resource_readiness", "post_deployment_verification", "deployment_risk_handoff", "deployment_risk_review"} {
		if !decisionLedgerContainsSource(ledger.Entries, sourceType) {
			t.Fatalf("expected ledger source %s in %+v", sourceType, ledger.Entries)
		}
	}
	if ledger.Summary.EntryCount != len(ledger.Entries) || ledger.Summary.EvidenceRefCount == 0 || ledger.Summary.AttentionCount == 0 {
		t.Fatalf("unexpected ledger summary: %+v", ledger.Summary)
	}
	filtered, err := BuildDecisionLedger(root, DecisionLedgerOptions{SourceType: "post-deployment-verification", Environment: "test_dev", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Entries) != 1 || filtered.Entries[0].SourceID != verification.ID {
		t.Fatalf("expected filtered verification entry, got %+v", filtered.Entries)
	}
	approved, err := BuildDecisionLedger(root, DecisionLedgerOptions{Decision: "approved", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(approved.Entries) != 1 || approved.Entries[0].SourceID != review.ID {
		t.Fatalf("expected approved risk review entry, got %+v", approved.Entries)
	}
}

func TestBuildWriteProofsAggregatesProviderDeploymentAndResourceControls(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	plan, err := release.Suggest(context.Background(), root, release.SuggestOptions{Version: "v0.4.0", MinIssues: 1})
	if err != nil {
		t.Fatal(err)
	}
	releaseExecution, found, err := release.ProviderPreview(root, plan.ID)
	if err != nil || !found {
		t.Fatalf("expected release provider preview, found=%v err=%v", found, err)
	}
	deploymentExecution, err := deployment.Execute(context.Background(), root, deployment.ExecuteOptions{DeploymentID: "missing-deployment", Mode: "dry_run"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := serverresources.Add(root, serverresources.Resource{
		ID:          "dev-proof",
		Environment: "test_dev",
		Host:        "127.0.0.1",
		Provider:    "local_vm",
		Owner:       "ops",
		AuthRef:     "env:DEV_SERVER_SSH_KEY",
		ExpiresAt:   "2099-01-01",
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, found, err := serverresources.Renew(root, serverresources.RenewalOptions{ResourceID: "dev-proof", ExpiresAt: "2099-02-01", ActorID: "ops", Reason: "proof test"}); err != nil || !found {
		t.Fatalf("expected resource renewal, found=%v err=%v", found, err)
	}

	report, err := BuildWriteProofs(root, WriteProofOptions{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	for _, operationType := range []string{"release_provider_execution", "deployment_execution", "resource_maintenance"} {
		if !writeProofContainsOperation(report.Proofs, operationType) {
			t.Fatalf("expected write proof operation %s in %+v", operationType, report.Proofs)
		}
	}
	if report.Summary.ProofCount != len(report.Proofs) || report.Summary.BlockedCount == 0 {
		t.Fatalf("unexpected write proof summary: %+v", report.Summary)
	}
	releaseProof := writeProofForOperation(report.Proofs, "release_provider_execution", releaseExecution.ID)
	if releaseProof.OperationID == "" || releaseProof.Decision != "WRITE_PROOF_WRITE_DISABLED" || len(releaseProof.ProviderEvidenceRefs) == 0 {
		t.Fatalf("expected release provider proof with write switch block and evidence, got %+v", releaseProof)
	}
	deploymentProof := writeProofForOperation(report.Proofs, "deployment_execution", deploymentExecution.ID)
	if deploymentProof.OperationID == "" || deploymentProof.Decision != "WRITE_PROOF_WRITE_DISABLED" || deploymentProof.SecretRefStatus != "not_required_for_dry_run" || len(deploymentProof.ProviderEvidenceRefs) == 0 {
		t.Fatalf("expected dry-run deployment proof with evidence, got %+v", deploymentProof)
	}
	filtered, err := BuildWriteProofs(root, WriteProofOptions{OperationType: "release-provider-execution", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Proofs) != 1 || filtered.Proofs[0].OperationID != releaseExecution.ID {
		t.Fatalf("expected filtered release provider proof, got %+v", filtered.Proofs)
	}
}

func TestBuildProviderProofRequirementsFiltersProviderOperations(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	report, err := BuildProviderProofRequirements(root, ProviderProofRequirementOptions{Provider: "github", OperationType: "release-provider-execution", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if report.PolicyID != defaultProviderProofPolicyID || report.PolicyVersion != defaultProviderProofPolicyVersion {
		t.Fatalf("unexpected provider proof policy metadata: %+v", report)
	}
	if len(report.Requirements) != 1 {
		t.Fatalf("expected one github release requirement, got %+v", report.Requirements)
	}
	requirement := report.Requirements[0]
	if requirement.Provider != "github" || requirement.OperationType != "release_provider_execution" || !requirement.RequireEvidence || !requirement.RequireApproval || !requirement.RequireWriteSwitch {
		t.Fatalf("unexpected github requirement: %+v", requirement)
	}
	if len(requirement.LeastPrivilegeScopes) == 0 || requirement.ReplayGuard == "" || len(requirement.RuleRefs) == 0 {
		t.Fatalf("expected least privilege, replay guard and rule refs: %+v", requirement)
	}
	empty, err := BuildProviderProofRequirements(root, ProviderProofRequirementOptions{Provider: "unknown", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Requirements) != 0 || empty.Summary.RequirementCount != 0 {
		t.Fatalf("expected unknown provider filter to be empty, got %+v", empty)
	}
}

func TestBuildWriteAdmissionsEvaluatesWriteProofGates(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Ensure(root); err != nil {
		t.Fatal(err)
	}
	plan, err := release.Suggest(context.Background(), root, release.SuggestOptions{Version: "v0.5.0", MinIssues: 1})
	if err != nil {
		t.Fatal(err)
	}
	releaseExecution, found, err := release.ProviderPreview(root, plan.ID)
	if err != nil || !found {
		t.Fatalf("expected release provider preview, found=%v err=%v", found, err)
	}
	deploymentExecution, err := deployment.Execute(context.Background(), root, deployment.ExecuteOptions{DeploymentID: "missing-deployment", Mode: "dry_run"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := serverresources.Add(root, serverresources.Resource{
		ID:          "dev-admission",
		Environment: "test_dev",
		Host:        "127.0.0.1",
		Provider:    "local_vm",
		Owner:       "ops",
		AuthRef:     "env:DEV_SERVER_SSH_KEY",
		ExpiresAt:   "2099-01-01",
	}); err != nil {
		t.Fatal(err)
	}
	_, renewal, found, err := serverresources.Renew(root, serverresources.RenewalOptions{ResourceID: "dev-admission", ExpiresAt: "2099-02-01", ActorID: "ops", Reason: "admission test"})
	if err != nil || !found {
		t.Fatalf("expected resource renewal, found=%v err=%v", found, err)
	}

	report, err := BuildWriteAdmissions(root, WriteAdmissionOptions{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if report.PolicyID != defaultWriteAdmissionPolicyID || report.Target != "real_write" {
		t.Fatalf("unexpected admission policy metadata: %+v", report)
	}
	if report.Summary.EntryCount != len(report.Entries) || report.Summary.RehearsalOnlyCount == 0 || report.Summary.ReadyCount == 0 {
		t.Fatalf("unexpected write admission summary: %+v", report.Summary)
	}
	releaseAdmission := writeAdmissionForOperation(report.Entries, "release_provider_execution", releaseExecution.ID)
	if releaseAdmission.OperationID == "" || releaseAdmission.Status != "rehearsal_only" || releaseAdmission.Decision != "WRITE_ADMISSION_WRITE_DISABLED" || len(releaseAdmission.ProviderEvidenceRefs) == 0 || releaseAdmission.ProviderRequirementID == "" {
		t.Fatalf("expected release provider admission to require write enablement, got %+v", releaseAdmission)
	}
	deploymentAdmission := writeAdmissionForOperation(report.Entries, "deployment_execution", deploymentExecution.ID)
	if deploymentAdmission.OperationID == "" || deploymentAdmission.Status != "rehearsal_only" || deploymentAdmission.Decision != "WRITE_ADMISSION_WRITE_DISABLED" || !deploymentAdmission.RehearsalAllowed || deploymentAdmission.ProviderRequirementID == "" {
		t.Fatalf("expected dry-run deployment admission to be rehearsal-only, got %+v", deploymentAdmission)
	}
	resourceAdmission := writeAdmissionForOperation(report.Entries, "resource_maintenance", renewal.ID)
	if resourceAdmission.OperationID == "" || resourceAdmission.Status != "ready" || resourceAdmission.Decision != "WRITE_ADMISSION_READY" || resourceAdmission.ProviderRequirementID == "" {
		t.Fatalf("expected test_dev resource maintenance admission to be ready, got %+v", resourceAdmission)
	}
	readyOnly, err := BuildWriteAdmissions(root, WriteAdmissionOptions{Status: "ready", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(readyOnly.Entries) != 1 || readyOnly.Entries[0].OperationID != renewal.ID {
		t.Fatalf("expected ready filter to return resource admission, got %+v", readyOnly.Entries)
	}
}

func timelineContainsType(items []TimelineItem, typ string) bool {
	for _, item := range items {
		if item.Type == typ {
			return true
		}
	}
	return false
}

func timelineContainsID(items []TimelineItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func decisionLedgerContainsSource(entries []DecisionEntry, sourceType string) bool {
	for _, entry := range entries {
		if entry.SourceType == sourceType {
			return true
		}
	}
	return false
}

func writeProofContainsOperation(proofs []WriteProof, operationType string) bool {
	for _, proof := range proofs {
		if proof.OperationType == operationType {
			return true
		}
	}
	return false
}

func writeProofForOperation(proofs []WriteProof, operationType string, operationID string) WriteProof {
	for _, proof := range proofs {
		if proof.OperationType == operationType && proof.OperationID == operationID {
			return proof
		}
	}
	return WriteProof{}
}

func writeAdmissionForOperation(entries []WriteAdmissionEntry, operationType string, operationID string) WriteAdmissionEntry {
	for _, entry := range entries {
		if entry.OperationType == operationType && entry.OperationID == operationID {
			return entry
		}
	}
	return WriteAdmissionEntry{}
}
