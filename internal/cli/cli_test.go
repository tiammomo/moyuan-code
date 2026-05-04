package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	assertContains(t, accepted.stdout, `"status": "accepted"`)
	assertContains(t, accepted.stdout, `"quality_report_id"`)

	issueState := runCLI(t, root, "orchestrator", "status", "phase1-001")
	assertContains(t, issueState.stdout, `"status": "accepted"`)
	assertContains(t, issueState.stdout, acceptedRunID)

	issueStateAlias := runCLI(t, root, "orchestrator", "issue", "status", "phase1-001")
	assertContains(t, issueStateAlias.stdout, `"status": "accepted"`)

	runState := runCLI(t, root, "orchestrator", "run", "status", acceptedRunID)
	assertContains(t, runState.stdout, `"status": "completed"`)
	assertContains(t, runState.stdout, `"runtime_status": "completed"`)
	assertContains(t, runState.stdout, `"quality_status": "passed"`)

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
	assertFileExists(t, root, ".moyuan/lifecycle/pull-requests/"+planID+".json")
	assertFileContains(t, root, ".moyuan/lifecycle/pull-requests/plans.jsonl", planID)
	assertFileContains(t, root, ".moyuan/logs/git.jsonl", "git_provider.plan.created")
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
}

func TestServerResourcesCLIRegistersListsAndValidatesProductionHosts(t *testing.T) {
	root := createTempRepo(t)
	runCLI(t, root, "project", "add", "--local", root)

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
		"--health-type", "tcp",
		"--health-target", "10.0.0.10:22",
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

	disabled := runCLI(t, root, "resources", "disable", "dev-1")
	assertContains(t, disabled.stdout, `"status": "disabled"`)
	assertFileContains(t, root, ".moyuan/resources/inventory.json", `"prod-1"`)
	assertFileContains(t, root, ".moyuan/resources/events.jsonl", "resource.added")
	assertFileContains(t, root, ".moyuan/logs/audit.jsonl", "server_resource.added")
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

func assertContains(t *testing.T, value string, needle string) {
	t.Helper()
	if !strings.Contains(value, needle) {
		t.Fatalf("expected output to contain %q\n%s", needle, value)
	}
}
