package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"moyuan-code/internal/approvals"
)

func TestPhase1E2ESmokeCoversLocalAndRemoteProjectLifecycle(t *testing.T) {
	t.Run("local project", func(t *testing.T) {
		root := createTempRepo(t)

		result := runCLI(t, root, "project", "add", "--local", root)
		assertContains(t, result.stdout, "project added:")
		assertFileContains(t, root, ".moyuan/projects.json", root)
		assertFileExists(t, root, ".moyuan/state.db")

		exercisePhase1Lifecycle(t, root)
	})

	t.Run("remote project", func(t *testing.T) {
		source := createTempRepo(t)
		remote := createBareRemote(t, source)
		controlRoot := t.TempDir()
		dest := filepath.Join(t.TempDir(), "managed-remote")

		result := runCLI(t, controlRoot, "project", "add", "--remote", remote, "--dest", dest)
		assertContains(t, result.stdout, "project added:")
		assertFileContains(t, controlRoot, ".moyuan/projects.json", dest)
		assertFileExists(t, controlRoot, ".moyuan/state.db")

		list := runCLI(t, controlRoot, "project", "list")
		assertContains(t, list.stdout, dest)

		exercisePhase1Lifecycle(t, dest)

		sync := runCLI(t, dest, "git", "sync", "--comprehend")
		assertContains(t, sync.stdout, "remote")
		assertFileContains(t, dest, ".moyuan/comprehension/events.jsonl", `"mode":"incremental"`)
	})
}

type cliResult struct {
	stdout string
	stderr string
	code   int
}

func exercisePhase1Lifecycle(t *testing.T, root string) {
	t.Helper()

	assertCoreWorkspaceArtifacts(t, root)

	doctor := runCLI(t, root, "workspace", "doctor")
	assertContains(t, doctor.stdout, "project")
	assertContains(t, doctor.stdout, "repository")
	assertContains(t, doctor.stdout, `"state_db"`)
	assertContains(t, doctor.stdout, `"validation"`)

	validate := runCLI(t, root, "workspace", "validate")
	assertContains(t, validate.stdout, `"status": "passed"`)

	whoami := runCLI(t, root, "auth", "whoami")
	assertContains(t, whoami.stdout, "local_single_user")

	full := runCLI(t, root, "comprehend", "--full")
	assertContains(t, full.stdout, `"mode": "full"`)

	status := runCLI(t, root, "git", "status")
	assertContains(t, status.stdout, `"isRepo": true`)

	branches := runCLI(t, root, "git", "branch", "list")
	if strings.TrimSpace(branches.stdout) == "" {
		t.Fatalf("expected git branch list to return at least one branch")
	}

	graph := runCLI(t, root, "issue", "graph", "phase1-epic")
	assertContains(t, graph.stdout, "phase1-013")

	schedule := runCLI(t, root, "issue", "schedule", "phase1-epic")
	assertContains(t, schedule.stdout, "ready_queue")

	plan := runCLI(t, root, "orchestrator", "plan", "phase1-epic")
	assertContains(t, plan.stdout, "blocked_reason")

	health := runCLI(t, root, "runtime", "health", "local_shell")
	assertContains(t, health.stdout, `"ok": true`)

	runtimeResult := runCLI(t, root, "runtime", "invoke", "local_shell", "--prompt", "printf runtime-ok")
	assertContains(t, runtimeResult.stdout, "runtime-ok")
	assertContains(t, runtimeResult.stdout, "diff_summary_path")
	assertContains(t, runtimeResult.stdout, `"git_before"`)

	qualityResult := runCLI(t, root, "quality", "check", "phase1-001")
	report := decodeQualityReport(t, qualityResult.stdout)
	if report.Status != "passed" {
		t.Fatalf("quality report status = %s\n%s", report.Status, qualityResult.stdout)
	}
	if report.ID == "" {
		t.Fatalf("quality report missing id: %s", qualityResult.stdout)
	}
	if !report.HasCheck("test", "go test ./...") {
		t.Fatalf("quality report missing go test check: %+v", report.Checks)
	}

	reportResult := runCLI(t, root, "quality", "report", report.ID)
	assertContains(t, reportResult.stdout, report.ID)
	reportsResult := runCLI(t, root, "quality", "reports", "--limit", "1")
	assertContains(t, reportsResult.stdout, report.ID)
	explainResult := runCLI(t, root, "quality", "explain", report.ID)
	assertContains(t, explainResult.stdout, `"decision": "QUALITY_ACCEPTED"`)
	assertContains(t, explainResult.stdout, `"quality_and_review_accepted"`)
	policyResult := runCLI(t, root, "quality", "policy")
	assertContains(t, policyResult.stdout, `"required_checks"`)
	assertContains(t, policyResult.stdout, `"blocking_finding_categories"`)

	orchestrated := runCLI(t, root, "orchestrator", "run", "phase1-001", "--runtime", "local_shell", "--prompt", "printf orchestrator-ok")
	assertContains(t, orchestrated.stdout, "accepted")
	assertContains(t, orchestrated.stdout, `"issue_state"`)
	assertContains(t, orchestrated.stdout, `"run_state"`)

	addMemory := runCLI(t, root, "memory", "add", "--kind", "fact", "--summary", "phase1 memory fact should be used by future project quality tasks")
	assertContains(t, addMemory.stdout, `"status": "recorded"`)
	assertContains(t, addMemory.stdout, `"confidence"`)
	search := runCLI(t, root, "memory", "search", "phase1")
	assertContains(t, search.stdout, "phase1 memory fact")
	compact := runCLI(t, root, "memory", "compact")
	assertContains(t, compact.stdout, "records_seen")

	repair := runCLI(t, root, "repair", "signal", "--type", "test_failure", "--summary", "sample test failure")
	assertContains(t, repair.stdout, "CONFIRMED_BUG")

	logs := runCLI(t, root, "logs", "tail", "--stream", "run", "--limit", "50")
	assertContains(t, logs.stdout, "runtime.completed")

	assertLifecycleArtifacts(t, root)
}

func TestMemoryRecordGateStagesDedupesRejectsAndCompacts(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	valuableSummary := "phase1 memory record gate must remember quality test policy for future issue runs"
	recorded := runCLI(t, root, "memory", "add", "--kind", "decision", "--summary", valuableSummary)
	assertContains(t, recorded.stdout, `"status": "recorded"`)
	assertContains(t, recorded.stdout, `"scope": "quality"`)
	assertContains(t, recorded.stdout, `"created_by": "cli"`)
	assertContains(t, recorded.stdout, `"trace_id"`)

	duplicate := runCLI(t, root, "memory", "add", "--kind", "decision", "--summary", valuableSummary)
	assertContains(t, duplicate.stdout, `"status": "deduped"`)
	assertContains(t, duplicate.stdout, `"duplicate_of"`)

	staged := runCLI(t, root, "memory", "add", "--kind", "fact", "--summary", "tiny")
	assertContains(t, staged.stdout, `"status": "staged"`)
	assertContains(t, staged.stdout, "below_record_threshold")

	rejected := runCLI(t, root, "memory", "add", "--kind", "fact", "--summary", "-----BEGIN PRIVATE KEY----- should not be stored")
	assertContains(t, rejected.stdout, `"status": "rejected"`)
	assertContains(t, rejected.stdout, "sensitive_content")
	assertFileContains(t, root, ".moyuan/memory/candidates.jsonl", "[REDACTED_PRIVATE_KEY]")

	search := runCLI(t, root, "memory", "search", "PRIVATE KEY")
	if strings.TrimSpace(search.stdout) != "" {
		t.Fatalf("sensitive memory should not be searchable: %s", search.stdout)
	}

	candidates := runCLI(t, root, "memory", "candidates")
	assertContains(t, candidates.stdout, `"status": "recorded"`)
	assertContains(t, candidates.stdout, `"status": "deduped"`)
	assertContains(t, candidates.stdout, `"status": "staged"`)
	assertContains(t, candidates.stdout, `"status": "rejected"`)

	compact := runCLI(t, root, "memory", "compact")
	assertContains(t, compact.stdout, `"strategy": "phase1-record-gate-summary"`)
	assertContains(t, compact.stdout, `"records_seen": 1`)
	assertContains(t, compact.stdout, `"topics"`)
	assertFileExists(t, root, ".moyuan/memory/staging.jsonl")
	assertGlob(t, root, ".moyuan/memory/compactions/*.json")
	assertFileContains(t, root, ".moyuan/memory/records.jsonl", `"confidence":`)
	assertFileContains(t, root, ".moyuan/logs/memory.jsonl", "memory.candidate.evaluated")
}

func TestOperationsTimelineCLIListsDeploymentFacts(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	executed := runCLIAllowFailure(t, root, "deploy", "execute", "missing-deployment")
	assertContains(t, executed.stdout, `"deployment_not_found"`)
	executionID := decodeStringField(t, executed.stdout, "id")

	timeline := runCLI(t, root, "operations", "timeline", "--type", "deployment_execution", "--limit", "5")
	assertContains(t, timeline.stdout, `"operations_timeline"`)
	assertContains(t, timeline.stdout, `"type": "deployment_execution"`)
	assertContains(t, timeline.stdout, `"primary_ref": "missing-deployment"`)
	assertContains(t, timeline.stdout, `"evidence_refs"`)

	report := runCLI(t, root, "operations", "audit-export", "--type", "deployment_execution", "--limit", "5", "--format", "markdown")
	assertContains(t, report.stdout, `"operations_audit_export"`)
	assertContains(t, report.stdout, `"format": "markdown"`)
	assertContains(t, report.stdout, `"markdown":`)
	assertContains(t, report.stdout, `"deployment_execution"`)

	verify := runCLIAllowFailure(t, root, "deploy", "verify", "create", "--execution-id", executionID, "--environment", "test_dev")
	assertContains(t, verify.stdout, `"POST_DEPLOYMENT_VERIFICATION_ATTENTION_REQUIRED"`)
	ledger := runCLI(t, root, "operations", "decision-ledger", "--source-type", "post_deployment_verification", "--environment", "test_dev", "--limit", "5")
	assertContains(t, ledger.stdout, `"decision_ledger"`)
	assertContains(t, ledger.stdout, `"source_type": "post_deployment_verification"`)
	assertContains(t, ledger.stdout, `"POST_DEPLOYMENT_VERIFICATION_ATTENTION_REQUIRED"`)

	writeProofs := runCLI(t, root, "operations", "write-proofs", "--operation-type", "deployment_execution", "--limit", "5")
	assertContains(t, writeProofs.stdout, `"write_proofs"`)
	assertContains(t, writeProofs.stdout, `"operation_type": "deployment_execution"`)
	assertContains(t, writeProofs.stdout, `"WRITE_PROOF_WRITE_DISABLED"`)

	writeAdmissions := runCLI(t, root, "operations", "write-admissions", "--operation-type", "deployment_execution", "--limit", "5")
	assertContains(t, writeAdmissions.stdout, `"write_admissions"`)
	assertContains(t, writeAdmissions.stdout, `"operation_type": "deployment_execution"`)
	assertContains(t, writeAdmissions.stdout, `"WRITE_ADMISSION_WRITE_DISABLED"`)
}

func TestControlLoopCLIRunsDurableIdempotentSteps(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)
	runCLI(t, root, "resources", "add", "--id", "dev-loop", "--environment", "test_dev", "--host", "10.0.0.20", "--provider", "local_vm", "--owner", "ops", "--auth-ref", "env:DEV_SERVER_SSH_KEY", "--expires-at", "2099-01-01")

	first := runCLI(t, root, "control-loop", "run", "--idempotency-key", "cli-phase19", "--retry-budget", "1", "--environment", "test_dev", "--step", "resource_health_scan", "--step", "operations_audit_export", "--step", "decision_ledger_refresh")
	assertContains(t, first.stdout, `"control_loop_run"`)
	assertContains(t, first.stdout, `"idempotency_key": "cli-phase19"`)
	runID := decodeControlLoopRunID(t, first.stdout)
	replayed := runCLI(t, root, "control-loop", "run", "--idempotency-key", "cli-phase19", "--step", "resource_health_scan")
	assertContains(t, replayed.stdout, runID)
	assertContains(t, replayed.stdout, `"idempotent_replay": true`)
	list := runCLI(t, root, "control-loop", "list")
	assertContains(t, list.stdout, `"control_loop_runs"`)
	assertContains(t, list.stdout, runID)
	shown := runCLI(t, root, "control-loop", "show", runID)
	assertContains(t, shown.stdout, `"decision_ledger_refresh"`)
}

func TestMaintenancePolicyCLIExplainsProductionGate(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	policy := runCLI(t, root, "resources", "maintenance", "policy", "--environment", "production", "--action", "deploy", "--requested-at", "2026-05-05")
	assertContains(t, policy.stdout, `"maintenance_policy_pack"`)
	assertContains(t, policy.stdout, `"maintenance_policy_decision"`)
	assertContains(t, policy.stdout, `"decision": "MAINTENANCE_POLICY_MANUAL_REVIEW_REQUIRED"`)
	assertContains(t, policy.stdout, `"maintenance_window_missing"`)
}

func TestSkillsCLIRegistersListsAndDisablesSkillDefinitions(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	added := runCLI(t, root, "skills", "add", "--id", "/tdd", "--source", "github:mattpocock/skills", "--risk", "low", "--role", "backend", "--role", "tester", "--tag", "quality", "--tool", "go-test")
	assertContains(t, added.stdout, `"id": "tdd"`)
	assertContains(t, added.stdout, `"enabled": true`)

	list := runCLI(t, root, "skills", "list")
	assertContains(t, list.stdout, `"source": "github:mattpocock/skills"`)
	assertContains(t, list.stdout, `"compatible_roles"`)

	recommendation := runCLI(t, root, "skills", "recommend", "--role", "backend", "--task-type", "quality", "--risk", "medium")
	assertContains(t, recommendation.stdout, `"skill_id": "tdd"`)
	assertFileContains(t, root, ".moyuan/skills/recommendations.jsonl", `"role":"backend"`)

	bound := runCLI(t, root, "skills", "bind", "--skill", "tdd", "--target-type", "role", "--target", "backend")
	assertContains(t, bound.stdout, `"id": "binding-role-backend-tdd"`)
	assertContains(t, bound.stdout, `"status": "enabled"`)
	bindings := runCLI(t, root, "skills", "bindings")
	assertContains(t, bindings.stdout, `"skill_id": "tdd"`)
	unbound := runCLI(t, root, "skills", "binding", "disable", "binding-role-backend-tdd")
	assertContains(t, unbound.stdout, `"status": "disabled"`)
	assertFileContains(t, root, ".moyuan/skills/bindings.json", `"id": "binding-role-backend-tdd"`)

	effectiveness := runCLI(t, root, "skills", "effectiveness", "add", "--skill", "tdd", "--issue", "phase1-001", "--outcome", "helped", "--quality-impact", "improved", "--rework-reduced")
	assertContains(t, effectiveness.stdout, `"outcome": "helped"`)
	assertContains(t, effectiveness.stdout, `"quality_impact": "improved"`)
	effectivenessList := runCLI(t, root, "skills", "effectiveness", "list", "--skill", "tdd")
	assertContains(t, effectivenessList.stdout, `"rework_reduced": true`)
	assertFileContains(t, root, ".moyuan/skills/effectiveness/effectiveness.jsonl", `"skill_id":"tdd"`)
	recommendationAfterUse := runCLI(t, root, "skills", "recommend", "--role", "backend", "--task-type", "quality", "--risk", "medium")
	assertContains(t, recommendationAfterUse.stdout, `"effectiveness_helped"`)

	disabled := runCLI(t, root, "skills", "disable", "tdd")
	assertContains(t, disabled.stdout, `"enabled": false`)
	assertFileContains(t, root, ".moyuan/skills/registry.json", `"id": "tdd"`)
	assertFileContains(t, root, ".moyuan/skills/events.jsonl", "skill.disabled")

	rejected := runCLIAllowFailure(t, root, "skills", "add", "--id", "bad-secret", "--source", "local", "--auth-ref", "sk-plain-secret")
	if rejected.code == 0 {
		t.Fatalf("expected plain secret skill auth ref to be rejected: %s", rejected.stdout)
	}
	assertContains(t, rejected.stderr, "auth_ref_must_be_reference")
}

func TestVisualsDiagramPlanSanitizesAndIndexesAssets(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	plan := runCLI(t, root, "visuals", "diagram", "plan", "--type", "multi-agent", "--scope", "token=plain-secret 10.0.0.1")
	assertContains(t, plan.stdout, `"diagram_type": "multi_agent"`)
	assertContains(t, plan.stdout, `"prompt_path"`)
	assertContains(t, plan.stdout, `"spec_path"`)
	assetID := decodeVisualAssetID(t, plan.stdout)

	asset := runCLI(t, root, "visuals", "asset", "show", assetID)
	assertContains(t, asset.stdout, assetID)
	assertContains(t, asset.stdout, `"size": "3072x2048"`)
	assets := runCLI(t, root, "visuals", "assets", "--limit", "1")
	assertContains(t, assets.stdout, assetID)
	render := runCLI(t, root, "visuals", "asset", "render", assetID)
	assertContains(t, render.stdout, `"decision": "VISUAL_RENDER_DRY_RUN"`)
	assertContains(t, render.stdout, `"no_image_api_called"`)
	renderID := decodeStringField(t, render.stdout, "id")
	renders := runCLI(t, root, "visuals", "renders", "--limit", "1")
	assertContains(t, renders.stdout, renderID)
	shownRender := runCLI(t, root, "visuals", "render", "show", renderID)
	assertContains(t, shownRender.stdout, `"script_preview"`)
	blockedRender := runCLI(t, root, "visuals", "asset", "render", assetID, "--mode", "script")
	assertContains(t, blockedRender.stdout, `"visual_render_approval_required"`)

	assertGlob(t, root, ".moyuan/visuals/specs/*.json")
	assertGlob(t, root, ".moyuan/visuals/prompts/*.prompt.md")
	assertGlobFileContains(t, root, ".moyuan/visuals/specs/*.json", "[REDACTED_PRIVATE_IP]")
	assertGlobFileContains(t, root, ".moyuan/visuals/specs/*.json", "token=[REDACTED]")
	assertGlobFileNotContains(t, root, ".moyuan/visuals/specs/*.json", "10.0.0.1")
	assertGlobFileNotContains(t, root, ".moyuan/visuals/prompts/*.prompt.md", "plain-secret")
	assertFileContains(t, root, ".moyuan/visuals/assets/assets.jsonl", assetID)
	assertFileContains(t, root, ".moyuan/visuals/executions/events.jsonl", renderID)
	assertFileContains(t, root, ".moyuan/logs/model.jsonl", "visual.diagram.planned")
	assertFileContains(t, root, ".moyuan/logs/model.jsonl", "visual.render.execution.created")
}

func TestRepairControlledLoopRunsQualityAndStopsAfterMaxAttempts(t *testing.T) {
	root := createTempRepo(t)
	brokenSource := "package phase1smoke\n\nfunc Ready() bool { return false }\n"
	if err := os.WriteFile(filepath.Join(root, "smoke.go"), []byte(brokenSource), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root, "introduce failing readiness")
	runCLI(t, root, "project", "add", "--local", root)

	signal := runCLI(t, root, "repair", "signal", "--type", "test_failure", "--summary", "Ready returns false and go test fails")
	assertContains(t, signal.stdout, "CONFIRMED_BUG")
	assertContains(t, signal.stdout, `"quality_gate_required": true`)
	planID := decodeRepairPlanID(t, signal.stdout)

	fixedSource := "package phase1smoke\n\nfunc Ready() bool { return true }\n"
	first := runCLI(t, root, "repair", "run", planID, "--runtime", "local_shell", "--prompt", "printf '"+fixedSource+"' > smoke.go")
	assertContains(t, first.stdout, `"status": "repaired"`)
	assertContains(t, first.stdout, `"quality_status": "passed"`)
	assertContains(t, first.stdout, `"review_status": "accepted"`)
	assertContains(t, first.stdout, `"memory_status": "recorded"`)
	firstAttemptID := decodeRepairAttemptID(t, first.stdout)

	status := runCLI(t, root, "repair", "status", firstAttemptID)
	assertContains(t, status.stdout, `"status": "repaired"`)
	assertContains(t, status.stdout, `"runtime_status": "completed"`)

	assertFileExists(t, root, ".moyuan/repair/attempts.jsonl")
	assertFileExists(t, root, ".moyuan/repair/attempts/"+firstAttemptID+".json")
	assertFileContains(t, root, ".moyuan/logs/run.jsonl", "self_repair.repair.completed")
	assertFileContains(t, root, ".moyuan/memory/records.jsonl", "Repair succeeded")

	commitAll(t, root, "first repair attempt")
	second := runCLI(t, root, "repair", "run", planID, "--runtime", "local_shell", "--prompt", "printf retry > repair-second.txt")
	assertContains(t, second.stdout, `"status": "repaired"`)
	commitAll(t, root, "second repair attempt")

	third := runCLIAllowFailure(t, root, "repair", "run", planID, "--runtime", "local_shell", "--prompt", "printf retry > repair-third.txt")
	if third.code == 0 {
		t.Fatalf("expected third repair attempt to be blocked: %s", third.stdout)
	}
	assertContains(t, third.stdout, `"status": "blocked"`)
	assertContains(t, third.stdout, "max_attempts_exceeded")
}

func TestOrchestratorStateMachinePersistsAcceptedAndNeedsRework(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	accepted := runCLI(t, root, "orchestrator", "run", "phase1-001", "--runtime", "local_shell", "--prompt", "printf accepted")
	acceptedRunID := decodeRunID(t, accepted.stdout)
	acceptedSubagentID := decodeStringField(t, accepted.stdout, "subagent_id")
	assertContains(t, accepted.stdout, `"status": "accepted"`)
	assertContains(t, accepted.stdout, `"quality_report_id"`)
	assertContains(t, accepted.stdout, `"subagent_id"`)

	issueState := runCLI(t, root, "orchestrator", "status", "phase1-001")
	assertContains(t, issueState.stdout, `"status": "accepted"`)
	assertContains(t, issueState.stdout, acceptedRunID)

	issueStateAlias := runCLI(t, root, "orchestrator", "issue", "status", "phase1-001")
	assertContains(t, issueStateAlias.stdout, `"status": "accepted"`)

	runState := runCLI(t, root, "orchestrator", "run", "status", acceptedRunID)
	assertContains(t, runState.stdout, `"status": "completed"`)
	assertContains(t, runState.stdout, acceptedSubagentID)
	assertContains(t, runState.stdout, `"runtime_status": "completed"`)
	assertContains(t, runState.stdout, `"quality_status": "passed"`)

	subagentShown := runCLI(t, root, "orchestrator", "subagent", "show", acceptedSubagentID)
	assertContains(t, subagentShown.stdout, `"role": "backend"`)
	assertContains(t, subagentShown.stdout, `"output_contract"`)
	assertContains(t, subagentShown.stdout, `"output_converged": true`)
	subagentList := runCLI(t, root, "orchestrator", "subagent", "list", "--limit", "1")
	assertContains(t, subagentList.stdout, acceptedSubagentID)

	runList := runCLI(t, root, "orchestrator", "run", "list", "--limit", "1")
	assertContains(t, runList.stdout, acceptedRunID)
	assertContains(t, runList.stdout, `"issue_id": "phase1-001"`)

	graph := runCLI(t, root, "issue", "graph", "phase1-epic")
	assertContains(t, graph.stdout, `"id": "phase1-001"`)
	assertContains(t, graph.stdout, `"status": "accepted"`)

	plan := runCLI(t, root, "orchestrator", "plan", "phase1-epic")
	assertContains(t, plan.stdout, `"ready_queue": []`)

	mergeDecision := runCLI(t, root, "review", "merge-decision", "phase1-001")
	assertContains(t, mergeDecision.stdout, `"status": "ready_to_merge"`)
	assertContains(t, mergeDecision.stdout, `"decision": "MERGE_ALLOWED"`)
	assertFileContains(t, root, ".moyuan/lifecycle/reviews/merge-decisions.jsonl", `"decision":"MERGE_ALLOWED"`)

	needsRework := runCLIAllowFailure(t, root, "orchestrator", "run", "phase1-002", "--runtime", "local_shell", "--prompt", "printf blocked > .env")
	if needsRework.code == 0 {
		t.Fatalf("expected needs_rework to return non-zero: %s", needsRework.stdout)
	}
	needsReworkRunID := decodeRunID(t, needsRework.stdout)
	assertContains(t, needsRework.stdout, `"status": "needs_rework"`)
	assertContains(t, needsRework.stdout, "runtime_blocked")

	needsReworkIssue := runCLI(t, root, "orchestrator", "status", "phase1-002")
	assertContains(t, needsReworkIssue.stdout, `"status": "needs_rework"`)
	assertContains(t, needsReworkIssue.stdout, "runtime_blocked")

	needsReworkRun := runCLI(t, root, "orchestrator", "run", "status", needsReworkRunID)
	assertContains(t, needsReworkRun.stdout, `"status": "failed"`)
	assertContains(t, needsReworkRun.stdout, `"runtime_status": "blocked"`)
	assertContains(t, needsRework.stdout, `"review_status": "rejected"`)
	assertContains(t, needsRework.stdout, `"category": "protected_path"`)
	blockedMerge := runCLIAllowFailure(t, root, "review", "merge-decision", "phase1-002")
	assertContains(t, blockedMerge.stdout, `"status": "blocked"`)
	assertContains(t, blockedMerge.stdout, `"issue_not_accepted"`)

	assertFileExists(t, root, ".moyuan/orchestrator/issue-states/phase1-001.json")
	assertFileExists(t, root, ".moyuan/orchestrator/issue-states/phase1-002.json")
	assertFileExists(t, root, ".moyuan/orchestrator/run-states/"+acceptedRunID+".json")
	assertFileExists(t, root, ".moyuan/orchestrator/run-states/"+needsReworkRunID+".json")
	assertFileContains(t, root, ".moyuan/logs/run.jsonl", "orchestrator.issue.transitioned")
	assertFileContains(t, root, ".moyuan/logs/run.jsonl", "orchestrator.run.transitioned")
}

func TestRequirementPlanCreatesReadableIssueGraph(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	result := runCLI(t, root, "requirement", "plan", "--text", "add backend API to inspect issue graph with go test verification")
	assertContains(t, result.stdout, `"clarification_decision"`)
	assertContains(t, result.stdout, `"status": "proceed"`)
	assertContains(t, result.stdout, `"backend-implementation"`)
	epicID := decodeStringField(t, result.stdout, "epic_id")
	requirementID := decodeStringField(t, result.stdout, "id")

	graph := runCLI(t, root, "issue", "graph", epicID)
	assertContains(t, graph.stdout, `"backend-implementation"`)

	schedule := runCLI(t, root, "issue", "schedule", epicID)
	assertContains(t, schedule.stdout, `"ready_queue"`)
	assertContains(t, schedule.stdout, `"blocked_queue"`)

	assertFileExists(t, root, ".moyuan/lifecycle/requirements/"+requirementID+".json")

	weak := runCLIAllowFailure(t, root, "requirement", "plan", "--text", "tune")
	if weak.code == 0 {
		t.Fatalf("expected weak requirement to require clarification: %s", weak.stdout)
	}
	assertContains(t, weak.stdout, `"needs_user_input"`)
	assertContains(t, weak.stdout, `"missing_verifiable_goal"`)
}

func TestProviderRegistryCLIManagesProvidersAndRoutesRoles(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	defaults := runCLI(t, root, "model", "provider", "list")
	assertContains(t, defaults.stdout, `"id": "claude_cli"`)
	assertContains(t, defaults.stdout, `"id": "codex_cli"`)

	added := runCLI(t, root, "model", "provider", "add",
		"--id", "glm-main",
		"--vendor", "zhipu",
		"--api-type", "openai-compatible",
		"--auth-ref", "env:GLM_API_KEY",
		"--model", "glm-4",
		"--allow-project-memory",
		"--use-case", "memory_extraction",
	)
	assertContains(t, added.stdout, `"id": "glm-main"`)
	assertContains(t, added.stdout, `"auth_ref": "env:GLM_API_KEY"`)
	if strings.Contains(added.stdout, "sk-") {
		t.Fatalf("provider output leaked raw secret-like value: %s", added.stdout)
	}

	show := runCLI(t, root, "model", "provider", "show", "glm-main")
	assertContains(t, show.stdout, `"models"`)
	assertContains(t, show.stdout, `"glm-4"`)

	backend := runCLI(t, root, "model", "route", "--role", "backend", "--repo-edit")
	assertContains(t, backend.stdout, `"decision": "ROUTE_ALLOWED"`)
	assertContains(t, backend.stdout, `"runtime_id": "codex_cli"`)

	frontend := runCLI(t, root, "model", "route", "--role", "frontend", "--repo-edit")
	assertContains(t, frontend.stdout, `"runtime_id": "claude_cli"`)

	memoryRoute := runCLI(t, root, "model", "route", "--role", "memory_curator", "--task-type", "memory_extraction", "--includes-project-memory")
	assertContains(t, memoryRoute.stdout, `"provider_id": "glm-main"`)
	assertContains(t, memoryRoute.stdout, `"model_id": "glm-4"`)

	strategyRoute := runCLI(t, root, "model", "route", "--strategy", "low-cost-memory", "--includes-project-memory")
	assertContains(t, strategyRoute.stdout, `"strategy": "low_cost_memory"`)
	assertContains(t, strategyRoute.stdout, `"provider_id": "glm-main"`)

	ops := runCLI(t, root, "model", "provider", "ops", "glm-main", "--health", "ok", "--quota-status", "ok", "--limit-tokens", "1000", "--used-tokens", "250", "--requests", "3", "--currency", "usd", "--estimated-cost", "0.4", "--budget", "5", "--cost-status", "ok")
	assertContains(t, ops.stdout, `"remaining_tokens": 750`)
	assertContains(t, ops.stdout, `"currency": "USD"`)
	telemetry := runCLI(t, root, "model", "provider", "telemetry", "--provider", "glm-main")
	assertContains(t, telemetry.stdout, `"provider_id": "glm-main"`)
	assertContains(t, telemetry.stdout, `"PROVIDER_TELEMETRY_OK"`)

	t.Setenv("GLM_API_KEY", "")
	refresh := runCLI(t, root, "model", "provider", "refresh", "--provider", "glm-main")
	assertContains(t, refresh.stdout, `"updated": 1`)
	assertContains(t, refresh.stdout, `"auth_ref_env_missing:GLM_API_KEY"`)

	exhausted := runCLI(t, root, "model", "provider", "ops", "glm-main", "--quota-status", "exhausted")
	assertContains(t, exhausted.stdout, `"status": "exhausted"`)

	disabled := runCLI(t, root, "model", "provider", "disable", "glm-main")
	assertContains(t, disabled.stdout, `"enabled": false`)

	rawSecret := runCLIAllowFailure(t, root, "model", "provider", "add",
		"--id", "bad-provider",
		"--vendor", "openai",
		"--api-type", "openai",
		"--auth-ref", "plain-secret-should-not-be-stored",
	)
	if rawSecret.code == 0 {
		t.Fatalf("expected raw secret provider registration to fail: %s", rawSecret.stdout)
	}
	assertContains(t, rawSecret.stderr, "auth_ref_must_be_reference")
	assertFileContains(t, root, ".moyuan/models/providers.json", `"id": "glm-main"`)
	assertFileContains(t, root, ".moyuan/logs/audit.jsonl", "provider.route.decided")
}

func TestGitProviderPlanCLIRequiresReviewAndPlansRemotePush(t *testing.T) {
	root := createTempRepo(t)
	remote := createBareRemote(t, root)
	run(t, root, "git", "remote", "add", "origin", remote)
	runCLI(t, root, "project", "add", "--local", root)
	commitAll(t, root, "moyuan project artifacts")

	blocked := runCLIAllowFailure(t, root, "git", "provider", "plan", "phase1-001")
	if blocked.code == 0 {
		t.Fatalf("expected git provider plan to block before issue is accepted: %s", blocked.stdout)
	}
	assertContains(t, blocked.stdout, `"decision": "GIT_PROVIDER_BLOCKED"`)
	assertContains(t, blocked.stdout, "review_merge_not_allowed")
	commitAll(t, root, "blocked git provider plan")

	accepted := runCLI(t, root, "orchestrator", "run", "phase1-001", "--runtime", "local_shell", "--prompt", "printf pr-plan > pr-plan.txt")
	assertContains(t, accepted.stdout, `"status": "accepted"`)
	commitAll(t, root, "accepted issue output")

	plan := runCLI(t, root, "git", "provider", "plan", "phase1-001")
	assertContains(t, plan.stdout, `"status": "push_plan_ready"`)
	assertContains(t, plan.stdout, `"decision": "PUSH_ALLOWED_PR_MR_UNSUPPORTED"`)
	assertContains(t, plan.stdout, `"provider": "generic_git"`)
	assertContains(t, plan.stdout, `"push_command": "git push origin`)
	planID := decodeStringField(t, plan.stdout, "id")

	shown := runCLI(t, root, "git", "provider", "show", planID)
	assertContains(t, shown.stdout, planID)
	assertContains(t, shown.stdout, `"manual_required": true`)
	listed := runCLI(t, root, "git", "provider", "list")
	assertContains(t, listed.stdout, planID)
	assertContains(t, listed.stdout, `"git_provider_plans"`)
	synced := runCLI(t, root, "git", "provider", "sync", planID)
	assertContains(t, synced.stdout, `"sync_decision": "PR_MR_STATUS_MANUAL_REQUIRED"`)
	assertContains(t, synced.stdout, `"remote_status": "manual_required"`)
	assertFileExists(t, root, ".moyuan/lifecycle/pull-requests/"+planID+".json")
	assertFileContains(t, root, ".moyuan/lifecycle/pull-requests/plans.jsonl", planID)
	assertFileContains(t, root, ".moyuan/logs/git.jsonl", "git_provider.plan.created")
	assertFileContains(t, root, ".moyuan/logs/git.jsonl", "git_provider.status.synced")
	commitAll(t, root, "git provider plan")

	releasePlan := runCLI(t, root, "release", "suggest", "--version", "v0.1.0", "--min-issues", "1")
	assertContains(t, releasePlan.stdout, `"status": "suggested"`)
	assertContains(t, releasePlan.stdout, `"decision": "RELEASE_SUGGESTED"`)
	assertContains(t, releasePlan.stdout, `"release_branch": "release/v0.1.0"`)
	assertContains(t, releasePlan.stdout, `"phase1-001"`)
	releaseID := decodeStringField(t, releasePlan.stdout, "id")

	releaseShown := runCLI(t, root, "release", "show", releaseID)
	assertContains(t, releaseShown.stdout, releaseID)
	assertContains(t, releaseShown.stdout, `"notes_path"`)
	assertFileExists(t, root, ".moyuan/lifecycle/releases/"+releaseID+".json")
	assertFileExists(t, root, ".moyuan/lifecycle/releases/"+releaseID+".md")
	assertFileContains(t, root, ".moyuan/lifecycle/releases/plans.jsonl", releaseID)
	assertFileContains(t, root, ".moyuan/logs/release.jsonl", "release.plan.created")

	releaseProviderPreview := runCLI(t, root, "release", "provider", "preview", releaseID)
	assertContains(t, releaseProviderPreview.stdout, `"decision": "RELEASE_PROVIDER_PREVIEW_READY"`)
	assertContains(t, releaseProviderPreview.stdout, `"create_release"`)
	assertContains(t, releaseProviderPreview.stdout, `"trigger_workflow"`)
	releaseProviderExecutionID := decodeStringField(t, releaseProviderPreview.stdout, "id")
	releaseProviderShown := runCLI(t, root, "release", "provider", "execution", releaseProviderExecutionID)
	assertContains(t, releaseProviderShown.stdout, releaseProviderExecutionID)
	assertFileExists(t, root, ".moyuan/lifecycle/releases/provider-executions/"+releaseProviderExecutionID+".json")
	assertFileContains(t, root, ".moyuan/logs/release.jsonl", "release.provider.previewed")
	releaseEvidence := runCLI(t, root, "evidence", "list", "--parent-type", "release_provider_execution", "--parent-id", releaseProviderExecutionID)
	assertContains(t, releaseEvidence.stdout, `"evidence"`)
	assertContains(t, releaseEvidence.stdout, `"release.provider.preview"`)
	releaseEvidenceID := decodeFirstEvidenceID(t, releaseEvidence.stdout)
	releaseEvidenceShown := runCLI(t, root, "evidence", "show", releaseEvidenceID)
	assertContains(t, releaseEvidenceShown.stdout, releaseProviderExecutionID)

	releaseProviderPublish := runCLIAllowFailure(t, root, "release", "provider", "publish", releaseID)
	if releaseProviderPublish.code == 0 {
		t.Fatalf("expected release provider publish to require approval: %s", releaseProviderPublish.stdout)
	}
	assertContains(t, releaseProviderPublish.stdout, `"RELEASE_PROVIDER_PUBLISH_APPROVAL_REQUIRED"`)
	assertContains(t, releaseProviderPublish.stdout, `"approval_id"`)

	deployResource := runCLI(t, root, "resources", "add",
		"--id", "deploy-dev",
		"--environment", "test_dev",
		"--host", "10.0.2.10",
		"--provider", "local_vm",
		"--owner", "dev-owner",
		"--auth-ref", "env:DEV_SERVER_SSH_KEY",
		"--expires-at", "2099-01-01",
	)
	assertContains(t, deployResource.stdout, `"deploy-dev"`)

	deployPlan := runCLI(t, root, "deploy", "plan", releaseID, "--environment", "test_dev", "--resource", "deploy-dev")
	assertContains(t, deployPlan.stdout, `"status": "planned"`)
	assertContains(t, deployPlan.stdout, `"decision": "DEPLOY_PLAN_READY"`)
	assertContains(t, deployPlan.stdout, `"smoke_plan"`)
	assertContains(t, deployPlan.stdout, `"monitor_plan"`)
	deploymentID := decodeStringField(t, deployPlan.stdout, "id")

	deployShown := runCLI(t, root, "deploy", "show", deploymentID)
	assertContains(t, deployShown.stdout, deploymentID)
	assertContains(t, deployShown.stdout, `"rollback_plan"`)
	assertFileExists(t, root, ".moyuan/lifecycle/deployments/"+deploymentID+".json")
	assertFileContains(t, root, ".moyuan/lifecycle/deployments/plans.jsonl", deploymentID)
	assertFileContains(t, root, ".moyuan/logs/release.jsonl", "deployment.plan.created")

	deployDryRun := runCLI(t, root, "deploy", "execute", deploymentID)
	assertContains(t, deployDryRun.stdout, `"status": "completed"`)
	assertContains(t, deployDryRun.stdout, `"decision": "DEPLOY_EXECUTION_DRY_RUN"`)
	assertContains(t, deployDryRun.stdout, `"no_remote_or_local_commands_executed"`)
	dryRunExecutionID := decodeStringField(t, deployDryRun.stdout, "id")
	assertFileExists(t, root, ".moyuan/lifecycle/deployments/executions/"+dryRunExecutionID+".json")
	resourceAfterDeployment := runCLI(t, root, "resources", "show", "deploy-dev")
	assertContains(t, resourceAfterDeployment.stdout, `"last_deployment"`)
	assertContains(t, resourceAfterDeployment.stdout, dryRunExecutionID)
	deploymentRefs := runCLI(t, root, "resources", "deployment-refs")
	assertContains(t, deploymentRefs.stdout, `"resource_deployment_refs"`)
	assertContains(t, deploymentRefs.stdout, dryRunExecutionID)
	assertContains(t, deploymentRefs.stdout, `"deployment_execution"`)
	missingRollback := runCLIAllowFailure(t, root, "deploy", "rollback", "missing-execution")
	if missingRollback.code == 0 {
		t.Fatalf("expected missing rollback execution to fail: %s", missingRollback.stdout)
	}
	assertContains(t, missingRollback.stdout, `"deployment_execution_not_found"`)
	monitorSummary := runCLI(t, root, "deploy", "monitor", "summarize", "--environment", "test_dev")
	assertContains(t, monitorSummary.stdout, `"decision": "DEPLOYMENT_MONITOR_HEALTHY"`)
	monitorSummaryID := decodeStringField(t, monitorSummary.stdout, "id")
	monitorSummaryShown := runCLI(t, root, "deploy", "monitor-summary", monitorSummaryID)
	assertContains(t, monitorSummaryShown.stdout, monitorSummaryID)
	verification := runCLI(t, root, "deploy", "verify", "create", "--execution-id", dryRunExecutionID, "--environment", "test_dev")
	assertContains(t, verification.stdout, `"decision": "POST_DEPLOYMENT_VERIFICATION_PASSED"`)
	assertContains(t, verification.stdout, `"risk_handoff_recommended": false`)
	verificationID := decodeStringField(t, verification.stdout, "id")
	verificationShown := runCLI(t, root, "deploy", "verify", "show", verificationID)
	assertContains(t, verificationShown.stdout, verificationID)
	verifications := runCLI(t, root, "deploy", "verify", "list")
	assertContains(t, verifications.stdout, `"post_deployment_verifications"`)
	rehearsal := runCLI(t, root, "deploy", "rehearsal", "create", "--deployment-id", deploymentID, "--execution-id", dryRunExecutionID)
	assertContains(t, rehearsal.stdout, `"decision": "DEPLOYMENT_REHEARSAL_READY"`)
	assertContains(t, rehearsal.stdout, `"deployment_execution"`)
	assertContains(t, rehearsal.stdout, `"monitor_summary_id"`)
	rehearsalID := decodeStringField(t, rehearsal.stdout, "id")
	rehearsalShown := runCLI(t, root, "deploy", "rehearsal", rehearsalID)
	assertContains(t, rehearsalShown.stdout, rehearsalID)
	rehearsals := runCLI(t, root, "deploy", "rehearsals")
	assertContains(t, rehearsals.stdout, `"rehearsals"`)
	assertContains(t, rehearsals.stdout, rehearsalID)
	admission := runCLI(t, root, "release", "admission", "create", "--rehearsal-id", rehearsalID)
	assertContains(t, admission.stdout, `"decision": "RELEASE_ADMISSION_ALLOWED"`)
	assertContains(t, admission.stdout, `"deployment_rehearsal"`)
	assertContains(t, admission.stdout, `"policy_id": "release-admission-default-v1"`)
	admissionID := decodeStringField(t, admission.stdout, "id")
	admissionPolicy := runCLI(t, root, "release", "admission", "policy", "--environment", "production")
	assertContains(t, admissionPolicy.stdout, `"release_admission_policy_pack"`)
	assertContains(t, admissionPolicy.stdout, `"production"`)
	admissionShown := runCLI(t, root, "release", "admission", admissionID)
	assertContains(t, admissionShown.stdout, admissionID)
	admissions := runCLI(t, root, "release", "admissions")
	assertContains(t, admissions.stdout, `"release_admissions"`)
	assertContains(t, admissions.stdout, admissionID)
	schedulerRun := runCLI(t, root, "deploy", "rehearsal", "schedule", "--execution-id", dryRunExecutionID, "--max-targets", "1")
	assertContains(t, schedulerRun.stdout, `"decision": "REHEARSAL_SCHEDULER_NOOP"`)
	assertContains(t, schedulerRun.stdout, `"admission_already_exists"`)
	schedulerRunID := decodeStringField(t, schedulerRun.stdout, "id")
	schedulerShown := runCLI(t, root, "deploy", "rehearsal-scheduler", schedulerRunID)
	assertContains(t, schedulerShown.stdout, schedulerRunID)
	schedulerRuns := runCLI(t, root, "deploy", "rehearsal-schedulers")
	assertContains(t, schedulerRuns.stdout, `"rehearsal_scheduler_runs"`)
	riskHandoff := runCLI(t, root, "repair", "deployment-risk", "create", "--admission-id", admissionID)
	assertContains(t, riskHandoff.stdout, `"decision": "DEPLOYMENT_RISK_HANDOFF_NOT_REQUIRED"`)
	riskHandoffID := decodeStringField(t, riskHandoff.stdout, "id")
	riskQueue := runCLI(t, root, "repair", "deployment-risk", "queue")
	assertContains(t, riskQueue.stdout, `"deployment_risk_review_queue"`)
	riskHandoffShown := runCLI(t, root, "repair", "deployment-risk", riskHandoffID)
	assertContains(t, riskHandoffShown.stdout, riskHandoffID)
	riskHandoffs := runCLI(t, root, "repair", "deployment-risks")
	assertContains(t, riskHandoffs.stdout, `"deployment_risk_handoffs"`)

	deployApprovalRequired := runCLIAllowFailure(t, root, "deploy", "execute", deploymentID, "--mode", "local_shell", "--command", "printf deployment-ok")
	if deployApprovalRequired.code == 0 {
		t.Fatalf("expected deployment execution to require approval: %s", deployApprovalRequired.stdout)
	}
	assertContains(t, deployApprovalRequired.stdout, `"execution_approval_required"`)
	deployApprovalID := decodeStringField(t, deployApprovalRequired.stdout, "approval_id")
	if _, _, err := approvals.Decide(root, deployApprovalID, approvals.DecisionOptions{Decision: "approved", DecidedBy: "reviewer", Reason: "deployment ready"}); err != nil {
		t.Fatal(err)
	}

	deployExecution := runCLI(t, root, "deploy", "execute", deploymentID, "--mode", "local_shell", "--approved", "--approval-id", deployApprovalID, "--command", "printf deployment-ok")
	assertContains(t, deployExecution.stdout, `"status": "completed"`)
	assertContains(t, deployExecution.stdout, `"decision": "DEPLOY_EXECUTION_COMPLETED"`)
	assertContains(t, deployExecution.stdout, `"approval_consumed": true`)
	assertContains(t, deployExecution.stdout, `"deployment-ok"`)
	executionID := decodeStringField(t, deployExecution.stdout, "id")
	executionShown := runCLI(t, root, "deploy", "execution", executionID)
	assertContains(t, executionShown.stdout, executionID)
	assertContains(t, executionShown.stdout, `"local_shell"`)
	deployEvidence := runCLI(t, root, "evidence", "list", "--parent-type", "deployment_execution", "--parent-id", executionID)
	assertContains(t, deployEvidence.stdout, `"deployment.execute.local_shell"`)
	assertFileContains(t, root, ".moyuan/lifecycle/deployments/executions.jsonl", executionID)
	assertFileContains(t, root, ".moyuan/logs/release.jsonl", "deployment.execution.created")

	unsafeApprovalID := approveCLIDeploymentExecution(t, root, deploymentID, "local_shell")
	unsafeExecution := runCLIAllowFailure(t, root, "deploy", "execute", deploymentID, "--mode", "local_shell", "--approved", "--approval-id", unsafeApprovalID, "--command", "rm -rf /tmp/nope")
	if unsafeExecution.code == 0 {
		t.Fatalf("expected unsafe deployment command to fail: %s", unsafeExecution.stdout)
	}
	assertContains(t, unsafeExecution.stdout, `"command_not_allowed"`)

	prodResource := runCLI(t, root, "resources", "add",
		"--id", "deploy-prod",
		"--environment", "production",
		"--host", "prod.deploy.internal",
		"--provider", "aliyun",
		"--owner", "ops-owner",
		"--auth-ref", "secret:prod_ssh_key",
		"--expires-at", "2099-01-01",
		"--health-status", "healthy",
	)
	assertContains(t, prodResource.stdout, `"deploy-prod"`)
	prodBlocked := runCLIAllowFailure(t, root, "deploy", "plan", releaseID, "--environment", "production", "--resource", "deploy-prod")
	if prodBlocked.code == 0 {
		t.Fatalf("expected production deployment without approval to block: %s", prodBlocked.stdout)
	}
	assertContains(t, prodBlocked.stdout, "production_approval_required")
}

func TestServerResourcesCLIRegistersListsAndValidatesProductionHosts(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer healthServer.Close()

	dev := runCLI(t, root, "resources", "add",
		"--id", "dev-1",
		"--environment", "test_dev",
		"--host", "10.0.0.10",
		"--provider", "local_vm",
		"--owner", "dev-owner",
		"--auth-ref", "env:DEV_SERVER_SSH_KEY",
		"--expires-at", "2099-01-01",
		"--cpu", "2",
		"--memory-gb", "4",
		"--disk-gb", "80",
		"--health-type", "http",
		"--health-target", healthServer.URL,
	)
	assertContains(t, dev.stdout, `"id": "dev-1"`)
	assertContains(t, dev.stdout, `"environment": "test_dev"`)
	assertContains(t, dev.stdout, `"expiration_state": "ok"`)

	badProduction := runCLIAllowFailure(t, root, "resources", "add",
		"--id", "prod-bad",
		"--environment", "production",
		"--host", "10.0.1.10",
		"--provider", "aliyun",
		"--owner", "ops-owner",
		"--auth-ref", "secret:prod_ssh_key",
	)
	if badProduction.code == 0 {
		t.Fatalf("expected production resource without expiry to fail: %s", badProduction.stdout)
	}
	assertContains(t, badProduction.stderr, "production_expires_at_required")

	prod := runCLI(t, root, "resources", "add",
		"--id", "prod-1",
		"--environment", "production",
		"--host", "prod.example.internal",
		"--provider", "aliyun",
		"--owner", "ops-owner",
		"--auth-ref", "secret:prod_ssh_key",
		"--expires-at", "2099-01-01",
		"--region", "cn-shanghai",
		"--instance-id", "i-prod001",
	)
	assertContains(t, prod.stdout, `"id": "prod-1"`)

	list := runCLI(t, root, "resources", "list")
	assertContains(t, list.stdout, `"dev-1"`)
	assertContains(t, list.stdout, `"prod-1"`)

	shown := runCLI(t, root, "resources", "show", "prod-1")
	assertContains(t, shown.stdout, `"environment": "production"`)
	assertContains(t, shown.stdout, `"owner": "ops-owner"`)

	scan := runCLI(t, root, "resources", "expiration", "scan")
	assertContains(t, scan.stdout, "[]")

	healthScan := runCLI(t, root, "resources", "health", "scan", "--environment", "test_dev")
	assertContains(t, healthScan.stdout, `"decision": "HEALTH_SCAN_COMPLETED"`)
	assertContains(t, healthScan.stdout, `"status": "healthy"`)
	assertFileContains(t, root, ".moyuan/resources/checks.jsonl", "HEALTH_SCAN_COMPLETED")
	assertFileContains(t, root, ".moyuan/logs/audit.jsonl", "server_resource.health_scan")

	prodHealthBlocked := runCLIAllowFailure(t, root, "resources", "health", "scan", "--resource", "prod-1")
	if prodHealthBlocked.code == 0 {
		t.Fatalf("expected production health scan without approval to block: %s", prodHealthBlocked.stdout)
	}
	assertContains(t, prodHealthBlocked.stdout, "production_approval_required")

	oldDev := runCLI(t, root, "resources", "add",
		"--id", "old-dev",
		"--environment", "test_dev",
		"--host", "10.0.0.12",
		"--provider", "local_vm",
		"--owner", "dev-owner",
		"--auth-ref", "env:DEV_SERVER_SSH_KEY",
		"--expires-at", "2000-01-01",
	)
	assertContains(t, oldDev.stdout, `"expiration_state": "expired"`)
	maintenance := runCLI(t, root, "resources", "maintenance", "scan")
	assertContains(t, maintenance.stdout, `"maintenance_records"`)
	assertContains(t, maintenance.stdout, `"old-dev"`)
	assertContains(t, maintenance.stdout, `"MAINTENANCE_REQUIRED"`)
	maintenanceList := runCLI(t, root, "resources", "maintenance", "list")
	assertContains(t, maintenanceList.stdout, `"expiration_alert"`)
	renewed := runCLI(t, root, "resources", "renew", "old-dev", "--expires-at", "2099-03-01", "--actor", "ops-owner", "--reason", "renewed")
	assertContains(t, renewed.stdout, `"RESOURCE_RENEWAL_RECORDED"`)
	assertContains(t, renewed.stdout, `"expires_at": "2099-03-01"`)
	retired := runCLI(t, root, "resources", "retire", "old-dev", "--actor", "ops-owner", "--reason", "decommissioned")
	assertContains(t, retired.stdout, `"RESOURCE_RETIRED"`)
	assertContains(t, retired.stdout, `"status": "retired"`)

	disabled := runCLI(t, root, "resources", "disable", "dev-1")
	assertContains(t, disabled.stdout, `"status": "disabled"`)
	assertFileContains(t, root, ".moyuan/resources/inventory.json", `"prod-1"`)
	assertFileContains(t, root, ".moyuan/resources/events.jsonl", "resource.added")
	assertFileContains(t, root, ".moyuan/logs/audit.jsonl", "server_resource.added")
	assertFileContains(t, root, ".moyuan/resources/maintenance.jsonl", "RESOURCE_RETIRED")
	assertFileContains(t, root, ".moyuan/logs/audit.jsonl", "server_resource.renewed")
}

func approveCLIDeploymentExecution(t *testing.T, root string, deploymentID string, mode string) string {
	t.Helper()
	approval, err := approvals.Request(root, approvals.RequestOptions{
		TargetType:  "deployment_execution",
		TargetID:    deploymentID,
		Action:      "deploy.execute." + mode,
		RiskLevel:   "high",
		RequestedBy: "test",
		Reason:      "test deployment execution approval",
	})
	if err != nil {
		t.Fatal(err)
	}
	decided, found, err := approvals.Decide(root, approval.ID, approvals.DecisionOptions{Decision: "approved", DecidedBy: "reviewer", Reason: "test approved"})
	if err != nil || !found {
		t.Fatalf("expected deployment approval decision, found=%v err=%v", found, err)
	}
	return decided.ID
}

func TestQualityReviewHardeningFindingsDriveNeedsRework(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	normal := runCLI(t, root, "orchestrator", "run", "phase1-001", "--runtime", "local_shell", "--prompt", "printf ok > generated.txt")
	assertContains(t, normal.stdout, `"status": "accepted"`)
	assertContains(t, normal.stdout, `"review_status": "accepted"`)

	commitAll(t, root, "normal generated file")

	secret := runCLIAllowFailure(t, root, "orchestrator", "run", "phase1-002", "--runtime", "local_shell", "--prompt", "printf secret > api-token.txt")
	if secret.code == 0 {
		t.Fatalf("expected secret-like file change to fail quality review: %s", secret.stdout)
	}
	assertContains(t, secret.stdout, `"status": "needs_rework"`)
	assertContains(t, secret.stdout, `"review_status": "rejected"`)
	assertContains(t, secret.stdout, `"category": "secret_file"`)
	assertContains(t, secret.stdout, `"blocking": true`)

	reportID := decodeQualityReportID(t, secret.stdout)
	report := runCLI(t, root, "quality", "report", reportID)
	assertContains(t, report.stdout, `"category": "secret_file"`)
	assertContains(t, report.stdout, `"status": "failed"`)
	assertFileContains(t, root, ".moyuan/logs/quality.jsonl", `"review_status":"rejected"`)
}

func TestRuntimeDiffCaptureTracksGeneratedFilesAndBlocksDirtyWorktree(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	generated := "generated-by-runtime.txt"
	result := runCLI(t, root, "runtime", "invoke", "local_shell", "--prompt", "printf generated > "+generated)
	assertContains(t, result.stdout, generated)
	assertContains(t, result.stdout, `"new_dirty": true`)
	assertGlob(t, root, ".moyuan/runtime/*-local_shell-diff.md")
	assertFileContains(t, root, ".moyuan/logs/run.jsonl", "diff_summary_path")

	blocked := runCLIAllowFailure(t, root, "runtime", "invoke", "local_shell", "--prompt", "printf should-not-run")
	if blocked.code == 0 {
		t.Fatalf("expected dirty worktree runtime invoke to fail, stdout=%s", blocked.stdout)
	}
	assertContains(t, blocked.stdout, "pre_existing_dirty_worktree")
}

func TestRuntimeDiffCaptureBlocksProtectedPathChanges(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

	result := runCLIAllowFailure(t, root, "runtime", "invoke", "local_shell", "--prompt", "printf secret > .env")
	if result.code == 0 {
		t.Fatalf("expected protected path runtime invoke to fail, stdout=%s", result.stdout)
	}
	assertContains(t, result.stdout, `"status": "blocked"`)
	assertContains(t, result.stdout, "protected_paths_changed")
	assertContains(t, result.stdout, ".env")
	assertGlob(t, root, ".moyuan/runtime/*-local_shell-diff.md")
}

func TestNativeRuntimeAdaptersUseFakeClaudeAndCodexCLI(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)
	restorePath := prependFakeRuntimeCLIs(t)
	defer restorePath()

	claude := runCLI(t, root, "runtime", "invoke", "claude_cli", "--prompt", "write claude output")
	assertContains(t, claude.stdout, "fake claude completed")
	assertContains(t, claude.stdout, "claude-output.txt")
	assertContains(t, claude.stdout, "claude -p")

	commitAll(t, root, "claude output")

	codex := runCLI(t, root, "runtime", "invoke", "codex_cli", "--prompt", "write codex output")
	assertContains(t, codex.stdout, "fake codex completed")
	assertContains(t, codex.stdout, "codex-output.txt")
	assertContains(t, codex.stdout, "codex exec")

	assertGlob(t, root, ".moyuan/runtime/prompts/*.md")
	assertGlob(t, root, ".moyuan/runtime/*-claude_cli.json")
	assertGlob(t, root, ".moyuan/runtime/*-codex_cli.json")
	assertGlob(t, root, ".moyuan/runtime/*-native.json")
}

func TestClaudeRuntimeProviderEnvProfileInjectsMiniMax(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)
	t.Setenv("MINIMAX_TEST_AUTH", "runtime-auth-value")
	restorePath := prependClaudeEnvRuntimeCLI(t)
	defer restorePath()

	added := runCLI(t, root, "model", "provider", "add",
		"--id", "minimax-m27-claude",
		"--name", "MiniMax M2.7 via Claude CLI",
		"--vendor", "minimax",
		"--api-type", "anthropic-compatible",
		"--base-url", "https://api.minimaxi.com/anthropic",
		"--auth-ref", "env:MINIMAX_TEST_AUTH",
		"--runtime", "claude_cli",
		"--model", "MiniMax-M2.7",
		"--use-case", "frontend",
		"--allow-sensitive-code",
		"--allow-project-memory",
	)
	assertContains(t, added.stdout, `"auth_ref": "env:MINIMAX_TEST_AUTH"`)

	route := runCLI(t, root, "model", "route", "--role", "frontend", "--repo-edit")
	assertContains(t, route.stdout, `"provider_id": "minimax-m27-claude"`)
	assertContains(t, route.stdout, `"model_id": "MiniMax-M2.7"`)

	result := runCLI(t, root, "runtime", "invoke", "claude_cli", "--provider", "minimax-m27-claude", "--prompt", "render frontend shell")
	assertContains(t, result.stdout, `"provider_id": "minimax-m27-claude"`)
	assertContains(t, result.stdout, `"model_id": "MiniMax-M2.7"`)
	assertFileContains(t, root, "claude-env.txt", "base-ok")
	assertFileContains(t, root, "claude-env.txt", "auth-ok")
	assertFileContains(t, root, "claude-env.txt", "model-ok")
	assertFileContains(t, root, ".moyuan/models/providers.json", `"auth_ref": "env:MINIMAX_TEST_AUTH"`)
	assertGlob(t, root, ".moyuan/runtime/*-native.json")
	assertRuntimeMetadataDoesNotContain(t, root, "runtime-auth-value")
}

func TestNativeRuntimeAdaptersClassifyUnavailableAndFailedCLI(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)
	withoutClaude := withPath(t, t.TempDir())
	unavailable := runCLIAllowFailure(t, root, "runtime", "invoke", "claude_cli", "--prompt", "noop")
	withoutClaude()
	if unavailable.code == 0 {
		t.Fatalf("expected unavailable claude_cli to fail: %s", unavailable.stdout)
	}
	assertContains(t, unavailable.stdout, "runtime_unavailable: claude")

	restorePath := prependFailingCodex(t)
	defer restorePath()
	failed := runCLIAllowFailure(t, root, "runtime", "invoke", "codex_cli", "--prompt", "noop")
	if failed.code == 0 {
		t.Fatalf("expected failing codex_cli to fail: %s", failed.stdout)
	}
	assertContains(t, failed.stdout, "runtime_failed")
	assertContains(t, failed.stdout, "codex failed")
	assertContains(t, failed.stdout, `"recovery_id"`)
	assertContains(t, failed.stdout, `"native_session_id"`)

	recoveryID := decodeStringField(t, failed.stdout, "recovery_id")
	recovery := runCLI(t, root, "runtime", "recovery", "show", recoveryID)
	assertContains(t, recovery.stdout, `"failure_category": "runtime_failed"`)
	assertContains(t, recovery.stdout, `"fallback_candidate": "claude_cli"`)
	assertContains(t, recovery.stdout, `"stdout_path"`)
	assertContains(t, recovery.stdout, `"stderr_path"`)

	recoveries := runCLI(t, root, "runtime", "recovery", "list", "--limit", "1")
	assertContains(t, recoveries.stdout, recoveryID)
	assertGlob(t, root, ".moyuan/runtimes/sessions/*/stderr.txt")
	assertFileContains(t, root, ".moyuan/runtimes/recoveries/events.jsonl", recoveryID)
	assertFileContains(t, root, ".moyuan/logs/run.jsonl", "runtime.recovery.archived")
}

type qualityReport struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Checks []struct {
		Type    string  `json:"type"`
		Command *string `json:"command"`
		Status  string  `json:"status"`
	} `json:"checks"`
}

func (r qualityReport) HasCheck(typ string, command string) bool {
	for _, check := range r.Checks {
		if check.Type == typ && check.Command != nil && *check.Command == command && check.Status == "passed" {
			return true
		}
	}
	return false
}

func decodeQualityReport(t *testing.T, raw string) qualityReport {
	t.Helper()
	var report qualityReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("decode quality report: %v\n%s", err, raw)
	}
	return report
}

func decodeRunID(t *testing.T, raw string) string {
	t.Helper()
	var payload struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode run id: %v\n%s", err, raw)
	}
	if payload.RunID == "" {
		t.Fatalf("missing run_id in output: %s", raw)
	}
	return payload.RunID
}

func decodeQualityReportID(t *testing.T, raw string) string {
	t.Helper()
	var payload struct {
		QualityReport struct {
			ID string `json:"id"`
		} `json:"quality_report"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode quality report id: %v\n%s", err, raw)
	}
	if payload.QualityReport.ID == "" {
		t.Fatalf("missing quality_report.id in output: %s", raw)
	}
	return payload.QualityReport.ID
}

func decodeRepairPlanID(t *testing.T, raw string) string {
	t.Helper()
	var payload struct {
		RepairPlan struct {
			ID string `json:"id"`
		} `json:"repair_plan"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode repair plan id: %v\n%s", err, raw)
	}
	if payload.RepairPlan.ID == "" {
		t.Fatalf("missing repair_plan.id in output: %s", raw)
	}
	return payload.RepairPlan.ID
}

func decodeRepairAttemptID(t *testing.T, raw string) string {
	t.Helper()
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode repair attempt id: %v\n%s", err, raw)
	}
	if payload.ID == "" {
		t.Fatalf("missing repair attempt id in output: %s", raw)
	}
	return payload.ID
}

func decodeVisualAssetID(t *testing.T, raw string) string {
	t.Helper()
	var payload struct {
		Asset struct {
			ID string `json:"id"`
		} `json:"asset"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode visual asset id: %v\n%s", err, raw)
	}
	if payload.Asset.ID == "" {
		t.Fatalf("missing asset.id in output: %s", raw)
	}
	return payload.Asset.ID
}

func decodeControlLoopRunID(t *testing.T, raw string) string {
	t.Helper()
	var payload struct {
		Run struct {
			ID string `json:"id"`
		} `json:"control_loop_run"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode control loop run id: %v\n%s", err, raw)
	}
	if payload.Run.ID == "" {
		t.Fatalf("missing control_loop_run.id in output: %s", raw)
	}
	return payload.Run.ID
}

func decodeStringField(t *testing.T, raw string, field string) string {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode %s: %v\n%s", field, err, raw)
	}
	value, _ := payload[field].(string)
	if value == "" {
		t.Fatalf("missing %s in output: %s", field, raw)
	}
	return value
}

func decodeFirstEvidenceID(t *testing.T, raw string) string {
	t.Helper()
	var payload struct {
		Evidence []struct {
			ID string `json:"id"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode evidence: %v\n%s", err, raw)
	}
	if len(payload.Evidence) == 0 || payload.Evidence[0].ID == "" {
		t.Fatalf("missing evidence id in output: %s", raw)
	}
	return payload.Evidence[0].ID
}

func runCLI(t *testing.T, root string, args ...string) cliResult {
	t.Helper()
	result := runCLIAllowFailure(t, root, args...)
	if result.code != 0 {
		t.Fatalf("moyuan %s failed: code=%d stdout=%s stderr=%s", strings.Join(args, " "), result.code, result.stdout, result.stderr)
	}
	return result
}

func runCLIAllowFailure(t *testing.T, root string, args ...string) cliResult {
	t.Helper()
	argv := append([]string{}, args...)
	if root != "" {
		argv = append(argv, "--root", root)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), argv, &stdout, &stderr)
	return cliResult{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

func createTempRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run(t, root, "git", "init", "-q")
	goMod := "module example.com/phase1smoke\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	source := "package phase1smoke\n\nfunc Ready() bool { return true }\n"
	if err := os.WriteFile(filepath.Join(root, "smoke.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	testSource := "package phase1smoke\n\nimport \"testing\"\n\nfunc TestReady(t *testing.T) {\n\tif !Ready() {\n\t\tt.Fatal(\"not ready\")\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(root, "smoke_test.go"), []byte(testSource), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "add", "go.mod", "smoke.go", "smoke_test.go")
	run(t, root, "git", "-c", "user.email=test@example.com", "-c", "user.name=test", "commit", "-qm", "init")
	return root
}

func createBareRemote(t *testing.T, source string) string {
	t.Helper()
	remote := filepath.Join(t.TempDir(), "remote.git")
	run(t, "", "git", "clone", "--bare", source, remote)
	return remote
}

func run(t *testing.T, cwd string, command string, args ...string) {
	t.Helper()
	cmd := exec.Command(command, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", command, args, err, string(out))
	}
}

func prependFakeRuntimeCLIs(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "claude"), `#!/bin/sh
printf 'fake claude completed\n'
printf 'claude output\n' > claude-output.txt
`)
	writeExecutable(t, filepath.Join(dir, "codex"), `#!/bin/sh
printf 'fake codex completed\n'
printf 'codex output\n' > codex-output.txt
`)
	return withPath(t, dir)
}

func prependClaudeEnvRuntimeCLI(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "claude"), `#!/bin/sh
{
	if [ "$ANTHROPIC_BASE_URL" = "https://api.minimaxi.com/anthropic" ]; then printf 'base-ok\n'; else printf 'base-missing\n'; fi
	if [ "$ANTHROPIC_AUTH_TOKEN" = "runtime-auth-value" ]; then printf 'auth-ok\n'; else printf 'auth-missing\n'; fi
	if [ "$ANTHROPIC_MODEL" = "MiniMax-M2.7" ]; then printf 'model-ok\n'; else printf 'model-missing\n'; fi
} > claude-env.txt
printf 'fake claude completed\n'
`)
	return withPath(t, dir)
}

func prependFailingCodex(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "codex"), `#!/bin/sh
printf 'codex failed\n' >&2
exit 42
`)
	return withPath(t, dir)
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func withPath(t *testing.T, firstDir string) func() {
	t.Helper()
	previous := os.Getenv("PATH")
	if err := os.Setenv("PATH", firstDir+string(os.PathListSeparator)+previous); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := os.Setenv("PATH", previous); err != nil {
			t.Fatal(err)
		}
	}
}

func commitAll(t *testing.T, root string, message string) {
	t.Helper()
	run(t, root, "git", "add", ".")
	run(t, root, "git", "-c", "user.email=test@example.com", "-c", "user.name=test", "commit", "-qm", message)
}

func assertCoreWorkspaceArtifacts(t *testing.T, root string) {
	t.Helper()
	assertFileExists(t, root, ".moyuan/project.yaml")
	assertFileExists(t, root, ".moyuan/repository.yaml")
	assertFileExists(t, root, ".moyuan/policies/access.yaml")
	assertFileExists(t, root, ".moyuan/workspace.json")
	assertFileExists(t, root, ".moyuan/auth/owner.json")
	assertFileExists(t, root, ".moyuan/comprehension/project-profile.md")
	assertFileExists(t, root, ".moyuan/comprehension/module-map.md")
	assertFileExists(t, root, ".moyuan/comprehension/commands.md")
	assertFileExists(t, root, ".moyuan/comprehension/events.jsonl")
	assertFileExists(t, root, ".moyuan/lifecycle/issue-graphs/phase1-epic.json")
	assertFileExists(t, root, ".moyuan/lifecycle/schedules/phase1-epic.json")
	assertFileExists(t, root, ".moyuan/memory/candidates.jsonl")
	assertFileExists(t, root, ".moyuan/logs/run.jsonl")
	assertFileExists(t, root, ".moyuan/logs/audit.jsonl")
}

func assertLifecycleArtifacts(t *testing.T, root string) {
	t.Helper()
	assertGlob(t, root, ".moyuan/runtime/*.json")
	assertGlob(t, root, ".moyuan/lifecycle/runs/*.json")
	assertGlob(t, root, ".moyuan/lifecycle/quality/reports/*.json")
	assertGlob(t, root, ".moyuan/lifecycle/quality/reports/*.md")
	assertGlob(t, root, ".moyuan/orchestrator/*-result.json")
	assertFileExists(t, root, ".moyuan/memory/records.jsonl")
	assertFileExists(t, root, ".moyuan/memory/staging.jsonl")
	assertFileExists(t, root, ".moyuan/memory/compact-latest.json")
	assertGlob(t, root, ".moyuan/memory/compactions/*.json")
	assertFileExists(t, root, ".moyuan/repair/signals.jsonl")
	assertFileExists(t, root, ".moyuan/repair/bug-candidates.jsonl")
	assertGlob(t, root, ".moyuan/repair/repair-plan-*.json")
	assertFileExists(t, root, ".moyuan/logs/quality.jsonl")
	assertFileExists(t, root, ".moyuan/logs/memory.jsonl")
}

func assertFileExists(t *testing.T, root string, rel string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file to exist: %s: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("expected file, got directory: %s", path)
	}
}

func assertFileContains(t *testing.T, root string, rel string, needle string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	assertContains(t, string(data), needle)
}

func assertGlob(t *testing.T, root string, pattern string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
	if err != nil {
		t.Fatalf("bad glob %s: %v", pattern, err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected at least one match for %s", filepath.Join(root, filepath.FromSlash(pattern)))
	}
}

func assertGlobFileContains(t *testing.T, root string, pattern string, needle string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
	if err != nil {
		t.Fatalf("bad glob %s: %v", pattern, err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected at least one match for %s", filepath.Join(root, filepath.FromSlash(pattern)))
	}
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			t.Fatalf("read %s: %v", match, err)
		}
		if strings.Contains(string(data), needle) {
			return
		}
	}
	t.Fatalf("expected one file matching %s to contain %q", pattern, needle)
}

func assertGlobFileNotContains(t *testing.T, root string, pattern string, needle string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
	if err != nil {
		t.Fatalf("bad glob %s: %v", pattern, err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected at least one match for %s", filepath.Join(root, filepath.FromSlash(pattern)))
	}
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			t.Fatalf("read %s: %v", match, err)
		}
		if strings.Contains(string(data), needle) {
			t.Fatalf("%s leaked %q", match, needle)
		}
	}
}

func assertRuntimeMetadataDoesNotContain(t *testing.T, root string, needle string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, ".moyuan", "runtime", "*-native.json"))
	if err != nil {
		t.Fatalf("bad runtime metadata glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected native runtime metadata")
	}
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			t.Fatalf("read %s: %v", match, err)
		}
		if strings.Contains(string(data), needle) {
			t.Fatalf("runtime metadata leaked secret value in %s", match)
		}
	}
}

func assertContains(t *testing.T, value string, needle string) {
	t.Helper()
	if !strings.Contains(value, needle) {
		t.Fatalf("expected output to contain %q\n%s", needle, value)
	}
}
