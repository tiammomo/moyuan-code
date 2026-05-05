"use client";

import {
  Activity,
  AlertTriangle,
  Boxes,
  Brain,
  CheckCircle2,
  ChevronRight,
  CircleDotDashed,
  GitBranch,
  KeyRound,
  Layers3,
  Lock,
  MemoryStick,
  Network,
  Play,
  RefreshCw,
  Rocket,
  ScrollText,
  Search,
  Server,
  ShieldCheck,
  Sparkles,
  TerminalSquare,
  UserPlus,
  Wrench,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useMemo, useState, type FormEvent } from "react";
import type { ConsoleSnapshot, EvidenceSummary, IssueNode, StatusTone } from "@/lib/types";

const laneLabels: Record<IssueNode["lane"], string> = {
  plan: "Planning",
  backend: "Backend",
  frontend: "Frontend",
  quality: "Quality",
  release: "Release",
};

const nav = [
  { label: "Projects", icon: Boxes },
  { label: "Issue Graph", icon: Network },
  { label: "Batches", icon: Layers3 },
  { label: "Runs", icon: TerminalSquare },
  { label: "Quality", icon: ShieldCheck },
  { label: "Memory", icon: Brain },
  { label: "Providers", icon: Sparkles },
  { label: "Deployments", icon: Rocket },
  { label: "Operations", icon: Activity },
  { label: "Audit", icon: Lock },
] as const;

type ConsoleView = (typeof nav)[number]["label"];

export function ConsoleWorkbench({ snapshot }: { snapshot: ConsoleSnapshot }) {
  const router = useRouter();
  const [activeView, setActiveView] = useState<ConsoleView>("Projects");
  const [selectedIssueID, setSelectedIssueID] = useState(snapshot.issues[0]?.id ?? "");
  const [selectedOperationID, setSelectedOperationID] = useState(snapshot.operation_history[0]?.id ?? "");
  const [requirementText, setRequirementText] = useState("");
  const [requirementState, setRequirementState] = useState<RequirementSubmitState>({ status: "idle" });
  const [schemaErrors, setSchemaErrors] = useState<Record<string, string[]>>({});
  const [recoveryArtifactState, setRecoveryArtifactState] = useState<Record<string, RecoveryArtifactState>>({});
  const [visualActionState, setVisualActionState] = useState<Record<string, VisualActionState>>({});
  const [deploymentActionState, setDeploymentActionState] = useState<DeploymentActionState>({ status: "idle" });
  const [approvalForm, setApprovalForm] = useState({ decidedBy: "console-owner", reason: "reviewed in console" });
  const [approvalActionState, setApprovalActionState] = useState<Record<string, ActionState>>({});
  const [sessionForm, setSessionForm] = useState({ userID: "developer", displayName: "Developer", roles: "developer" });
  const [tokenForm, setTokenForm] = useState({ name: "console-token", actorID: "developer", scopes: "project:read" });
  const [serviceAccountForm, setServiceAccountForm] = useState({ id: "", name: "Release Bot", roles: "release_bot,deploy_executor" });
  const [accessActionState, setAccessActionState] = useState<Record<string, ActionState>>({});
  const [resourceForm, setResourceForm] = useState({ actorID: "ops-owner", expiresAt: "2099-01-01", reason: "console maintenance" });
  const [resourceActionState, setResourceActionState] = useState<Record<string, ActionState>>({});
  const [gitActionState, setGitActionState] = useState<Record<string, ActionState>>({});
  const [gitCreateApproved, setGitCreateApproved] = useState(false);
  const [gitCreateApprovalID, setGitCreateApprovalID] = useState("");
  const [providerRouteForm, setProviderRouteForm] = useState({
    role: "frontend",
    taskType: "requirement_planning",
    outputType: "code",
    modelStrategy: "default",
    requiresRepoEdit: true,
    includesSensitiveCode: false,
    includesProjectMemory: true,
  });
  const [providerRoute, setProviderRoute] = useState<ProviderRouteDecision | null>(null);
  const [providerRouteState, setProviderRouteState] = useState<ActionState>({ status: "idle" });
  const [controlLoopActionState, setControlLoopActionState] = useState<ActionState>({ status: "idle" });
  const [repairReviewForm, setRepairReviewForm] = useState({ reviewerID: "qa-owner", reason: "reviewed in console" });
  const [repairActionState, setRepairActionState] = useState<Record<string, ActionState>>({});
  const [batchActionState, setBatchActionState] = useState<Record<string, ActionState>>({});
  const [releaseProviderForm, setReleaseProviderForm] = useState({
    releaseID: snapshot.deployments[0]?.release_id ?? "",
    approved: false,
    approvalID: "",
  });
  const [releaseProviderActionState, setReleaseProviderActionState] = useState<ActionState>({ status: "idle" });
  const selectedIssue = snapshot.issues.find((issue) => issue.id === selectedIssueID) ?? snapshot.issues[0];
  const selectedOperation = snapshot.operation_history.find((operation) => operation.id === selectedOperationID) ?? snapshot.operation_history[0];
  const operationDetailByID = useMemo(() => new Map(snapshot.operation_details.map((detail) => [detail.id, detail])), [snapshot.operation_details]);
  const selectedOperationDetail = selectedOperation ? operationDetailByID.get(selectedOperation.id) : undefined;
  const evidenceByID = useMemo(() => new Map(snapshot.evidence.map((record) => [record.id, record])), [snapshot.evidence]);
  const selectedEvidenceRecords = useMemo(
    () =>
      selectedOperationDetail?.evidence.length
        ? selectedOperationDetail.evidence
        : selectedOperation?.evidence_ids.map((id) => evidenceByID.get(id)).filter((record): record is EvidenceSummary => Boolean(record)) ?? [],
    [evidenceByID, selectedOperation, selectedOperationDetail],
  );
  const groupedIssues = useMemo(() => groupIssues(snapshot.issues), [snapshot.issues]);
  const latestDeployment = snapshot.deployments[0];
  const latestVerification = snapshot.post_deployment_verifications[0];
  const latestResourceDeploymentRef = snapshot.resource_deployment_refs[0];
  const latestRollbackCandidate = snapshot.executions.find((execution) => execution.rollback_required);
  const latestMonitorSummary = snapshot.monitor_summaries[0];
  const latestRehearsal = snapshot.deployment_rehearsals[0];
  const latestAdmission = snapshot.release_admissions[0];
  const latestSchedulerRun = snapshot.rehearsal_scheduler_runs[0];
  const latestRiskHandoff = snapshot.deployment_risk_handoffs[0];
  const latestRiskReviewQueueItem = snapshot.deployment_risk_review_queue[0];
  const latestRiskReview = snapshot.deployment_risk_reviews[0];
  const hasDeploymentOpsHistory = Boolean(
    latestMonitorSummary ||
      latestRehearsal ||
      latestAdmission ||
      latestSchedulerRun ||
      latestRiskHandoff ||
      latestRiskReviewQueueItem ||
      latestRiskReview ||
      latestVerification ||
      snapshot.rollback_executions.length > 0,
  );
  const operationsAuditExport = snapshot.operations_audit_export;
  const decisionLedger = snapshot.decision_ledger;
  const writeProofReport = snapshot.write_proofs;
  const writeProofs = writeProofReport?.proofs ?? [];
  const decisionEntries = decisionLedger?.entries ?? [];
  const latestControlLoopRun = snapshot.control_loop_runs[0];
  const activeSessions = snapshot.auth_sessions.filter((session) => session.status === "active");
  const activeTokens = snapshot.api_tokens.filter((token) => token.status === "active");
  const activeServiceAccounts = snapshot.service_accounts.filter((account) => account.status === "active");

  function setSchemaResult(key: string, errors: string[]) {
    setSchemaErrors((current) => ({ ...current, [key]: errors }));
  }

  function requireFields(key: string, fields: Array<[string, string]>) {
    const errors = fields.filter(([, value]) => value.trim() === "").map(([label]) => `${label} is required`);
    setSchemaResult(key, errors);
    return errors.length === 0;
  }

  async function submitRequirement(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const text = requirementText.trim();
    if (!requireFields("requirement", [["Requirement", text]])) {
      setRequirementState({ status: "error", message: "Requirement text is required." });
      return;
    }
    setRequirementState({ status: "planning", message: "Planning issue graph..." });
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/requirements/plan`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text }),
      });
      const payload = (await response.json()) as RequirementPlanEnvelope;
      if (!payload.requirement) {
        throw new Error(payload.error ?? "Requirement planner returned no result.");
      }
      const decision = payload.requirement.clarification_decision;
      const needsInput = Boolean(decision?.required);
      setRequirementState({
        status: needsInput ? "needs_user_input" : "planned",
        id: payload.requirement.id,
        epic: payload.requirement.epic_id,
        message: needsInput
          ? decision?.questions?.[0] ?? "This requirement needs clarification."
          : `${payload.requirement.issues?.length ?? 0} issues generated.`,
      });
    } catch (error) {
      setRequirementState({ status: "error", message: error instanceof Error ? error.message : "Requirement planning failed." });
    }
  }

  async function loadRecoveryArtifacts(recoveryID: string) {
    setRecoveryArtifactState((current) => ({
      ...current,
      [recoveryID]: { status: "loading", message: "Loading archived artifacts..." },
    }));
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/runtime-recoveries/${encodeURIComponent(recoveryID)}/artifacts`);
      const payload = (await response.json()) as RecoveryArtifactsEnvelope;
      const artifacts = payload.runtime_recovery_artifacts?.artifacts;
      if (!response.ok || !artifacts) {
        throw new Error(payload.error ?? "Runtime recovery artifacts failed to load.");
      }
      setRecoveryArtifactState((current) => ({
        ...current,
        [recoveryID]: { status: "loaded", artifacts, message: `${artifacts.length} artifacts` },
      }));
    } catch (error) {
      setRecoveryArtifactState((current) => ({
        ...current,
        [recoveryID]: {
          status: "error",
          message: error instanceof Error ? error.message : "Runtime recovery artifacts failed to load.",
        },
      }));
    }
  }

  async function runVisualDryRun(assetID: string) {
    setVisualActionState((current) => ({
      ...current,
      [assetID]: { status: "running", message: "Creating dry-run render..." },
    }));
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/visuals/assets/${encodeURIComponent(assetID)}/render`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mode: "dry_run" }),
      });
      const payload = (await response.json()) as VisualRenderEnvelope;
      const execution = payload.visual_render_execution;
      if (!response.ok || !execution) {
        throw new Error(payload.error ?? "Visual render dry-run failed.");
      }
      setVisualActionState((current) => ({
        ...current,
        [assetID]: {
          status: execution.status === "completed" ? "completed" : "blocked",
          executionID: execution.id,
          message: `${execution.decision ?? execution.status ?? "dry-run recorded"}`,
        },
      }));
    } catch (error) {
      setVisualActionState((current) => ({
        ...current,
        [assetID]: {
          status: "error",
          message: error instanceof Error ? error.message : "Visual render dry-run failed.",
        },
      }));
    }
  }

  async function suggestRelease() {
    setDeploymentActionState({ status: "running", message: "Creating release suggestion..." });
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/releases/suggest`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ min_issues: 1 }),
      });
      const payload = (await response.json()) as ReleaseSuggestEnvelope;
      const release = payload.release;
      if (!response.ok || !release) {
        throw new Error(payload.error ?? "Release suggestion failed.");
      }
      setDeploymentActionState({
        status: release.status === "suggested" ? "completed" : "blocked",
        id: release.id,
        message: `${release.decision ?? release.status ?? "release decision recorded"}${release.reasons?.[0] ? ` / ${release.reasons[0]}` : ""}`,
      });
      if (release.id) {
        setReleaseProviderForm((current) => ({ ...current, releaseID: release.id ?? current.releaseID }));
      }
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Release suggestion failed." });
    }
  }

  async function runReleaseProviderAction(action: "preview" | "publish") {
    const releaseID = releaseProviderForm.releaseID.trim();
    const schemaKey = "releaseProvider";
    const fields: Array<[string, string]> = [["Release ID", releaseID]];
    if (action === "publish" && releaseProviderForm.approved) {
      fields.push(["Approval ID", releaseProviderForm.approvalID]);
    }
    if (!requireFields(schemaKey, fields)) {
      setReleaseProviderActionState({ status: "error", message: "Schema validation failed." });
      return;
    }
    setReleaseProviderActionState({ status: "running", message: `${action} release provider...` });
    try {
      const payload = await postJSON<ReleaseProviderExecutionEnvelope>(
        `/api/projects/${snapshot.project.id}/releases/${encodeURIComponent(releaseID)}/provider-${action}`,
        action === "publish"
          ? { approved: releaseProviderForm.approved, approval_id: releaseProviderForm.approvalID }
          : {},
      );
      const execution = payload.release_provider_execution;
      if (!execution) {
        throw new Error(payload.error ?? "Release provider action returned no execution.");
      }
      setReleaseProviderActionState({
        status: execution.status === "completed" ? "completed" : "blocked",
        id: execution.id,
        message: execution.decision ?? execution.status,
      });
      router.refresh();
    } catch (error) {
      setReleaseProviderActionState({
        status: "error",
        message: error instanceof Error ? error.message : "Release provider action failed.",
      });
    }
  }

  async function runDeploymentDryRun(deploymentID?: string) {
    if (!deploymentID) {
      setDeploymentActionState({ status: "error", message: "No deployment plan available for dry-run." });
      return;
    }
    setDeploymentActionState({ status: "running", message: "Creating deployment dry-run..." });
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/deployments/${encodeURIComponent(deploymentID)}/execute`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mode: "dry_run" }),
      });
      const payload = (await response.json()) as DeploymentExecutionEnvelope;
      const execution = payload.execution;
      if (!response.ok || !execution) {
        throw new Error(payload.error ?? "Deployment dry-run failed.");
      }
      setDeploymentActionState({
        status: execution.status === "completed" ? "completed" : "blocked",
        id: execution.id,
        message: `${execution.decision ?? execution.status ?? "deployment decision recorded"}${execution.reasons?.[0] ? ` / ${execution.reasons[0]}` : ""}`,
      });
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Deployment dry-run failed." });
    }
  }

  async function runResourceHealthScan() {
    setDeploymentActionState({ status: "running", message: "Running test_dev health scan..." });
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/resources/health-scan`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ environment: "test_dev" }),
      });
      const payload = (await response.json()) as ResourceHealthScanEnvelope;
      const scan = payload.health_scan;
      if (!response.ok || !scan) {
        throw new Error(payload.error ?? "Resource health scan failed.");
      }
      setDeploymentActionState({
        status: scan.status === "healthy" || scan.status === "completed" ? "completed" : "blocked",
        id: scan.id,
        message: `${scan.decision ?? scan.status ?? "health scan recorded"}${scan.results?.length ? ` / ${scan.results.length} resources` : ""}`,
      });
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Resource health scan failed." });
    }
  }

  async function previewRollbackExecution(executionID?: string) {
    if (!executionID) {
      setDeploymentActionState({ status: "error", message: "No rollback candidate available." });
      return;
    }
    setDeploymentActionState({ status: "running", message: "Creating rollback preview..." });
    try {
      const payload = await postJSON<RollbackExecutionEnvelope>(
        `/api/projects/${snapshot.project.id}/deployment-executions/${encodeURIComponent(executionID)}/rollback`,
        { mode: "preview" },
      );
      const rollback = payload.rollback_execution;
      if (!rollback) {
        throw new Error(payload.error ?? "Rollback preview returned no execution.");
      }
      setDeploymentActionState({
        status: rollback.status === "completed" ? "completed" : "blocked",
        id: rollback.id,
        message: rollback.decision ?? rollback.status ?? "rollback preview recorded",
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Rollback preview failed." });
    }
  }

  async function summarizeDeploymentMonitor() {
    setDeploymentActionState({ status: "running", message: "Creating monitor summary..." });
    try {
      const payload = await postJSON<MonitorSummaryEnvelope>(`/api/projects/${snapshot.project.id}/deployment-monitor-summary`, { limit: 10 });
      const summary = payload.monitor_summary;
      if (!summary) {
        throw new Error(payload.error ?? "Monitor summary returned no result.");
      }
      setDeploymentActionState({
        status: summary.status === "healthy" || summary.status === "completed" ? "completed" : "blocked",
        id: summary.id,
        message: summary.decision ?? summary.status ?? "monitor summary recorded",
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Monitor summary failed." });
    }
  }

  async function createPostDeploymentVerification() {
    const execution = snapshot.executions[0];
    if (!execution) {
      setDeploymentActionState({ status: "error", message: "No execution available for verification." });
      return;
    }
    setDeploymentActionState({ status: "running", message: "Creating post-deployment verification..." });
    try {
      const payload = await postJSON<PostDeploymentVerificationEnvelope>(`/api/projects/${snapshot.project.id}/post-deployment-verifications`, {
        execution_id: execution.id,
        environment: execution.environment,
        monitor_limit: 10,
      });
      const verification = payload.post_deployment_verification;
      if (!verification) {
        throw new Error(payload.error ?? "Post-deployment verification returned no record.");
      }
      setDeploymentActionState({
        status: verification.status === "completed" ? "completed" : "blocked",
        id: verification.id,
        message: `${verification.decision ?? verification.status}${verification.risk_handoff_recommended ? " / risk handoff recommended" : ""}`,
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Post-deployment verification failed." });
    }
  }

  async function createDeploymentRehearsal() {
    const execution = snapshot.executions[0];
    if (!latestDeployment && !execution) {
      setDeploymentActionState({ status: "error", message: "No deployment or execution available for rehearsal." });
      return;
    }
    setDeploymentActionState({ status: "running", message: "Creating deployment rehearsal..." });
    try {
      const payload = await postJSON<DeploymentRehearsalEnvelope>(`/api/projects/${snapshot.project.id}/deployment-rehearsals`, {
        deployment_id: latestDeployment?.id,
        execution_id: execution?.id,
        environment: latestDeployment?.environment || execution?.environment,
      });
      const rehearsal = payload.deployment_rehearsal;
      if (!rehearsal) {
        throw new Error(payload.error ?? "Deployment rehearsal returned no record.");
      }
      setDeploymentActionState({
        status: rehearsal.status === "blocked" ? "blocked" : "completed",
        id: rehearsal.id,
        message: rehearsal.decision ?? rehearsal.status ?? "deployment rehearsal recorded",
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Deployment rehearsal failed." });
    }
  }

  async function runRehearsalScheduler() {
    const execution = snapshot.executions[0];
    if (!latestDeployment && !execution && !snapshot.release_candidates[0]) {
      setDeploymentActionState({ status: "error", message: "No candidate, deployment, or execution available for scheduler." });
      return;
    }
    setDeploymentActionState({ status: "running", message: "Running bounded rehearsal scheduler..." });
    try {
      const payload = await postJSON<RehearsalSchedulerEnvelope>(`/api/projects/${snapshot.project.id}/deployment-rehearsal-scheduler-runs`, {
        candidate_id: snapshot.release_candidates[0]?.id,
        deployment_id: latestDeployment?.id,
        execution_id: execution?.id,
        environment: latestDeployment?.environment || execution?.environment,
        max_targets: 3,
      });
      const run = payload.rehearsal_scheduler_run;
      if (!run) {
        throw new Error(payload.error ?? "Rehearsal scheduler returned no run.");
      }
      setDeploymentActionState({
        status: run.status === "blocked" || run.status === "attention_required" ? "blocked" : "completed",
        id: run.id,
        message: `${run.decision ?? run.status ?? "scheduler run recorded"}${run.blocked_count ? ` / blocked ${run.blocked_count}` : ""}`,
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Rehearsal scheduler failed." });
    }
  }

  async function createReleaseAdmission() {
    setDeploymentActionState({ status: "running", message: "Creating release admission..." });
    try {
      const payload = await postJSON<ReleaseAdmissionEnvelope>(`/api/projects/${snapshot.project.id}/release-admissions`, {
        rehearsal_id: latestRehearsal?.id,
        deployment_id: latestDeployment?.id,
        execution_id: snapshot.executions[0]?.id,
        environment: latestDeployment?.environment || snapshot.executions[0]?.environment,
      });
      const admission = payload.release_admission;
      if (!admission) {
        throw new Error(payload.error ?? "Release admission returned no record.");
      }
      setDeploymentActionState({
        status: admission.status === "blocked" ? "blocked" : "completed",
        id: admission.id,
        message: admission.decision ?? admission.status ?? "release admission recorded",
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Release admission failed." });
    }
  }

  async function createDeploymentRiskHandoff() {
    if (!latestAdmission) {
      setDeploymentActionState({ status: "error", message: "No release admission available for risk handoff." });
      return;
    }
    setDeploymentActionState({ status: "running", message: "Creating deployment risk handoff..." });
    try {
      const payload = await postJSON<DeploymentRiskHandoffEnvelope>(`/api/projects/${snapshot.project.id}/repair/deployment-risk-handoffs`, {
        admission_id: latestAdmission.id,
      });
      const handoff = payload.deployment_risk_handoff;
      if (!handoff) {
        throw new Error(payload.error ?? "Deployment risk handoff returned no record.");
      }
      setDeploymentActionState({
        status: handoff.status === "blocked" ? "blocked" : "completed",
        id: handoff.id,
        message: handoff.decision ?? handoff.status ?? "deployment risk handoff recorded",
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Deployment risk handoff failed." });
    }
  }

  async function decideApproval(approvalID: string, decision: "approved" | "rejected") {
    if (!requireFields("approval", [["Decider", approvalForm.decidedBy], ["Reason", approvalForm.reason]])) {
      return;
    }
    setApprovalActionState((current) => ({
      ...current,
      [approvalID]: { status: "running", message: `${decision === "approved" ? "Approving" : "Rejecting"} approval...` },
    }));
    try {
      const payload = await postJSON<ApprovalDecisionEnvelope>(`/api/projects/${snapshot.project.id}/approvals/${encodeURIComponent(approvalID)}/decide`, {
        decision,
        decided_by: approvalForm.decidedBy,
        reason: approvalForm.reason,
      });
      const approval = payload.approval;
      if (!approval) {
        throw new Error(payload.error ?? "Approval decision returned no record.");
      }
      setApprovalActionState((current) => ({
        ...current,
        [approvalID]: { status: approval.status === "approved" ? "completed" : "blocked", id: approval.id, message: approval.decision ?? approval.status },
      }));
      router.refresh();
    } catch (error) {
      setApprovalActionState((current) => ({
        ...current,
        [approvalID]: { status: "error", message: error instanceof Error ? error.message : "Approval decision failed." },
      }));
    }
  }

  async function createSession(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("session", [["User", sessionForm.userID], ["Display", sessionForm.displayName], ["Roles", sessionForm.roles]])) {
      return;
    }
    setAccessActionState((current) => ({ ...current, session: { status: "running", message: "Creating session..." } }));
    try {
      const payload = await postJSON<AuthSessionEnvelope>(`/api/projects/${snapshot.project.id}/auth/sessions`, {
        user_id: sessionForm.userID,
        display_name: sessionForm.displayName,
        roles: splitCSV(sessionForm.roles),
      });
      if (!payload.session) {
        throw new Error(payload.error ?? "Session create returned no record.");
      }
      setAccessActionState((current) => ({
        ...current,
        session: { status: "completed", id: payload.session?.id, message: "SESSION_CREATED" },
      }));
      router.refresh();
    } catch (error) {
      setAccessActionState((current) => ({
        ...current,
        session: { status: "error", message: error instanceof Error ? error.message : "Session create failed." },
      }));
    }
  }

  async function createAPIToken(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("token", [["Name", tokenForm.name], ["Actor", tokenForm.actorID], ["Scopes", tokenForm.scopes]])) {
      return;
    }
    setAccessActionState((current) => ({ ...current, token: { status: "running", message: "Creating API token..." } }));
    try {
      const payload = await postJSON<APITokenCreateEnvelope>(`/api/projects/${snapshot.project.id}/auth/api-tokens`, {
        name: tokenForm.name,
        actor_id: tokenForm.actorID,
        scopes: splitCSV(tokenForm.scopes),
      });
      if (!payload.api_token) {
        throw new Error(payload.error ?? "API token create returned no record.");
      }
      setAccessActionState((current) => ({
        ...current,
        token: {
          status: "completed",
          id: payload.api_token?.id,
          message: "API_TOKEN_CREATED",
          secretPreview: payload.token_value ? `${payload.token_value.slice(0, 18)}...` : undefined,
        },
      }));
      router.refresh();
    } catch (error) {
      setAccessActionState((current) => ({
        ...current,
        token: { status: "error", message: error instanceof Error ? error.message : "API token create failed." },
      }));
    }
  }

  async function revokeAPIToken(tokenID: string) {
    setAccessActionState((current) => ({ ...current, [tokenID]: { status: "running", message: "Revoking token..." } }));
    try {
      const payload = await postJSON<APITokenRevokeEnvelope>(`/api/projects/${snapshot.project.id}/auth/api-tokens/${encodeURIComponent(tokenID)}/revoke`, {
        actor_id: resourceForm.actorID,
        reason: resourceForm.reason,
      });
      if (!payload.api_token) {
        throw new Error(payload.error ?? "API token revoke returned no record.");
      }
      setAccessActionState((current) => ({
        ...current,
        [tokenID]: { status: "completed", id: payload.api_token?.id, message: "API_TOKEN_REVOKED" },
      }));
      router.refresh();
    } catch (error) {
      setAccessActionState((current) => ({
        ...current,
        [tokenID]: { status: "error", message: error instanceof Error ? error.message : "API token revoke failed." },
      }));
    }
  }

  async function createServiceAccount(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("service", [["Name", serviceAccountForm.name], ["Roles", serviceAccountForm.roles]])) {
      return;
    }
    setAccessActionState((current) => ({ ...current, service: { status: "running", message: "Creating service account..." } }));
    try {
      const payload = await postJSON<ServiceAccountEnvelope>(`/api/projects/${snapshot.project.id}/auth/service-accounts`, {
        id: serviceAccountForm.id,
        name: serviceAccountForm.name,
        roles: splitCSV(serviceAccountForm.roles),
      });
      if (!payload.service_account) {
        throw new Error(payload.error ?? "Service account create returned no record.");
      }
      setAccessActionState((current) => ({
        ...current,
        service: { status: "completed", id: payload.service_account?.id, message: "SERVICE_ACCOUNT_UPSERTED" },
      }));
      router.refresh();
    } catch (error) {
      setAccessActionState((current) => ({
        ...current,
        service: { status: "error", message: error instanceof Error ? error.message : "Service account create failed." },
      }));
    }
  }

  async function runGitProviderAction(planID: string, action: "preview" | "sync" | "create") {
    if (action === "create" && gitCreateApproved && !requireFields("gitCreate", [["Approval ID", gitCreateApprovalID]])) {
      return;
    }
    setGitActionState((current) => ({ ...current, [planID]: { status: "running", message: `${action} PR/MR...` } }));
    try {
      const payload = await postJSON<GitProviderActionEnvelope>(`/api/projects/${snapshot.project.id}/git-provider-plans/${encodeURIComponent(planID)}/${action}`, {
        approved: action === "create" ? gitCreateApproved : undefined,
        approval_id: action === "create" ? gitCreateApprovalID : undefined,
      });
      const plan = payload.git_provider_plan;
      if (!plan) {
        throw new Error(payload.error ?? "PR/MR action returned no plan.");
      }
      setGitActionState((current) => ({
        ...current,
        [planID]: {
          status: isGitActionCompleted(plan) ? "completed" : "blocked",
          id: plan.id,
          message:
            plan.pr_mr?.create_decision ||
            plan.pr_mr?.preview_decision ||
            plan.pr_mr?.sync_decision ||
            plan.decision ||
            plan.pr_mr?.remote_status ||
            plan.status,
        },
      }));
      router.refresh();
    } catch (error) {
      setGitActionState((current) => ({
        ...current,
        [planID]: { status: "error", message: error instanceof Error ? error.message : "PR/MR action failed." },
      }));
    }
  }

  async function previewProviderRoute() {
    if (!requireFields("providerRoute", [["Role", providerRouteForm.role], ["Task Type", providerRouteForm.taskType]])) {
      setProviderRouteState({ status: "error", message: "Schema validation failed." });
      return;
    }
    setProviderRouteState({ status: "running", message: "Evaluating route candidates..." });
    try {
      const payload = await postJSON<ProviderRouteEnvelope>(`/api/projects/${snapshot.project.id}/provider-route`, {
        role: providerRouteForm.role,
        model_strategy: providerRouteForm.modelStrategy === "default" ? "" : providerRouteForm.modelStrategy,
        task_type: providerRouteForm.taskType,
        output_type: providerRouteForm.outputType,
        requires_repo_edit: providerRouteForm.requiresRepoEdit,
        includes_sensitive_code: providerRouteForm.includesSensitiveCode,
        includes_project_memory: providerRouteForm.includesProjectMemory,
        includes_secrets: false,
      });
      if (!payload.route) {
        throw new Error(payload.error ?? "Provider route returned no decision.");
      }
      setProviderRoute(payload.route);
      setProviderRouteState({
        status: payload.route.blocked ? "blocked" : "completed",
        id: payload.route.provider_id,
        message: `${payload.route.decision} / ${payload.route.candidates?.length ?? 0} candidates`,
      });
      router.refresh();
    } catch (error) {
      setProviderRouteState({ status: "error", message: error instanceof Error ? error.message : "Provider route preview failed." });
    }
  }

  async function runControlLoop() {
    setControlLoopActionState({ status: "running", message: "Running bounded control loop..." });
    try {
      const payload = await postJSON<ControlLoopRunEnvelope>(`/api/projects/${snapshot.project.id}/control-loop/run`, {
        trigger: "console_manual",
        requested_by: "console-owner",
      });
      const run = payload.control_loop_run;
      if (!run) {
        throw new Error(payload.error ?? "Control loop returned no run.");
      }
      setControlLoopActionState({
        status: run.status === "completed" && !(run.decision ?? "").includes("ATTENTION") ? "completed" : "blocked",
        id: run.id,
        message: `${run.decision ?? run.status} / ${run.steps?.length ?? 0} steps`,
      });
      router.refresh();
    } catch (error) {
      setControlLoopActionState({ status: "error", message: error instanceof Error ? error.message : "Control loop run failed." });
    }
  }

  async function reviewOperationRepairCandidate(candidateID: string, decision: "approved" | "rejected") {
    if (!requireFields("repairReview", [["Reviewer", repairReviewForm.reviewerID], ["Reason", repairReviewForm.reason]])) {
      return;
    }
    setRepairActionState((current) => ({
      ...current,
      [candidateID]: { status: "running", message: `${decision === "approved" ? "Approving" : "Rejecting"} repair candidate...` },
    }));
    try {
      const payload = await postJSON<OperationRepairReviewEnvelope>(
        `/api/projects/${snapshot.project.id}/repair/operation-candidates/${encodeURIComponent(candidateID)}/review`,
        {
          decision,
          reviewer_id: repairReviewForm.reviewerID,
          reason: repairReviewForm.reason,
          next_step: decision === "approved" ? "repair_attempt" : "",
        },
      );
      const review = payload.operation_repair_review;
      const candidate = payload.operation_repair_candidate;
      if (!review || !candidate) {
        throw new Error(payload.error ?? "Repair review returned no record.");
      }
      setRepairActionState((current) => ({
        ...current,
        [candidateID]: {
          status: candidate.status === "approved" ? "completed" : "blocked",
          id: payload.repair_attempt?.id ?? candidate.issue_id ?? candidate.id,
          message: `${review.decision ?? candidate.decision}${payload.repair_attempt?.status ? ` / ${payload.repair_attempt.status}` : ""}`,
        },
      }));
      router.refresh();
    } catch (error) {
      setRepairActionState((current) => ({
        ...current,
        [candidateID]: { status: "error", message: error instanceof Error ? error.message : "Repair candidate review failed." },
      }));
    }
  }

  async function runResourceAction(resourceID: string, action: "renew" | "retire") {
    const fields: Array<[string, string]> = [["Actor", resourceForm.actorID], ["Reason", resourceForm.reason]];
    if (action === "renew") {
      fields.push(["Expires", resourceForm.expiresAt]);
    }
    if (!requireFields("resource", fields)) {
      return;
    }
    setResourceActionState((current) => ({ ...current, [resourceID]: { status: "running", message: `${action} resource...` } }));
    try {
      const body =
        action === "renew"
          ? { actor_id: resourceForm.actorID, expires_at: resourceForm.expiresAt, reason: resourceForm.reason }
          : { actor_id: resourceForm.actorID, reason: resourceForm.reason };
      const payload = await postJSON<ResourceActionEnvelope>(`/api/projects/${snapshot.project.id}/resources/${encodeURIComponent(resourceID)}/${action}`, body);
      const record = payload.maintenance_record;
      if (!record) {
        throw new Error(payload.error ?? "Resource action returned no maintenance record.");
      }
      setResourceActionState((current) => ({
        ...current,
        [resourceID]: { status: record.status === "completed" ? "completed" : "blocked", id: record.id, message: record.decision ?? record.status },
      }));
      router.refresh();
    } catch (error) {
      setResourceActionState((current) => ({
        ...current,
        [resourceID]: { status: "error", message: error instanceof Error ? error.message : "Resource action failed." },
      }));
    }
  }

  async function runBatchDryRun(batchID: string) {
    setBatchActionState((current) => ({
      ...current,
      [batchID]: { status: "running", message: "Creating batch dry run..." },
    }));
    try {
      const payload = await postJSON<BatchRunEnvelope>(`/api/projects/${snapshot.project.id}/batches/${encodeURIComponent(batchID)}/run`, {
        mode: "dry_run",
        requested_by: "console",
      });
      const run = payload.batch_run;
      if (!run) {
        throw new Error(payload.error ?? "Batch dry run returned no run.");
      }
      setBatchActionState((current) => ({
        ...current,
        [batchID]: { status: run.status === "completed" ? "completed" : "blocked", id: run.id, message: run.decision ?? run.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [batchID]: { status: "error", message: error instanceof Error ? error.message : "Batch dry run failed." },
      }));
    }
  }

  async function buildMergeQueue(batchID: string) {
    const actionKey = `${batchID}:merge`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Building merge queue..." },
    }));
    try {
      const payload = await postJSON<MergeQueueEnvelope>(`/api/projects/${snapshot.project.id}/batches/${encodeURIComponent(batchID)}/merge-queue`, {});
      const queue = payload.merge_queue;
      if (!queue) {
        throw new Error(payload.error ?? "Merge queue returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: {
          status: queue.status === "ready_to_merge" ? "completed" : queue.status === "needs_rework" ? "blocked" : "blocked",
          id: queue.id,
          message: queue.decision ?? queue.status,
        },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Merge queue failed." },
      }));
    }
  }

  async function buildIntegrationPreview(queueID: string) {
    const actionKey = `${queueID}:integration-preview`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Creating integration preview..." },
    }));
    try {
      const payload = await postJSON<IntegrationPreviewEnvelope>(
        `/api/projects/${snapshot.project.id}/merge-queues/${encodeURIComponent(queueID)}/integration-preview`,
        {},
      );
      const preview = payload.integration_preview;
      if (!preview) {
        throw new Error(payload.error ?? "Integration preview returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: {
          status: preview.status === "ready" ? "completed" : "blocked",
          id: preview.id,
          message: preview.decision ?? preview.status,
        },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Integration preview failed." },
      }));
    }
  }

  async function dryRunIntegrationApply(previewID: string) {
    const actionKey = `${previewID}:apply-dry-run`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Planning integration apply..." },
    }));
    try {
      const payload = await postJSON<IntegrationApplyEnvelope>(`/api/projects/${snapshot.project.id}/integration-previews/${encodeURIComponent(previewID)}/apply`, {
        mode: "dry_run",
        requested_by: "console",
      });
      const apply = payload.integration_apply;
      if (!apply) {
        throw new Error(payload.error ?? "Integration apply returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: {
          status: apply.status === "planned" || apply.status === "applied" ? "completed" : "blocked",
          id: apply.id,
          message: apply.decision ?? apply.status,
        },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Integration apply failed." },
      }));
    }
  }

  async function planReleaseBatch(applyID: string) {
    const actionKey = `${applyID}:release-batch`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Checking release batch readiness..." },
    }));
    try {
      const payload = await postJSON<ReleaseBatchEnvelope>(`/api/projects/${snapshot.project.id}/integration-applies/${encodeURIComponent(applyID)}/release-batch`, {
        min_items: 3,
        requested_by: "console",
      });
      const releaseBatch = payload.release_batch;
      if (!releaseBatch) {
        throw new Error(payload.error ?? "Release batch returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: {
          status: releaseBatch.status === "suggested" ? "completed" : "blocked",
          id: releaseBatch.id,
          message: `${releaseBatch.decision ?? releaseBatch.status}${releaseBatch.ready_item_count ? ` / ${releaseBatch.ready_item_count} ready` : ""}`,
        },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Release batch check failed." },
      }));
    }
  }

  async function planReleaseCandidate(releaseBatchID: string) {
    const actionKey = `${releaseBatchID}:candidate`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Planning release candidate..." },
    }));
    try {
      const payload = await postJSON<ReleaseCandidateEnvelope>(`/api/projects/${snapshot.project.id}/release-batches/${encodeURIComponent(releaseBatchID)}/candidate`, {
        deployment_targets: ["test_dev"],
        requested_by: "console",
      });
      const candidate = payload.release_candidate;
      if (!candidate) {
        throw new Error(payload.error ?? "Release candidate returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: candidate.status === "ready" ? "completed" : "blocked", id: candidate.id, message: candidate.decision ?? candidate.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Release candidate plan failed." },
      }));
    }
  }

  async function dryRunReleaseCandidateApply(candidateID: string) {
    const actionKey = `${candidateID}:release-branch-apply`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Planning release branch apply..." },
    }));
    try {
      const payload = await postJSON<ReleaseCandidateApplyEnvelope>(`/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/apply`, {
        mode: "dry_run",
        requested_by: "console",
      });
      const apply = payload.release_candidate_apply;
      if (!apply) {
        throw new Error(payload.error ?? "Release branch apply returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: apply.status === "planned" || apply.status === "applied" ? "completed" : "blocked", id: apply.id, message: apply.decision ?? apply.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Release branch apply failed." },
      }));
    }
  }

  async function previewReleaseCandidateProvider(candidateID: string) {
    const actionKey = `${candidateID}:provider-preview`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Creating provider preview..." },
    }));
    try {
      const payload = await postJSON<ReleaseCandidateProviderPreviewEnvelope>(
        `/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/provider-preview`,
        {},
      );
      const preview = payload.release_candidate_provider_preview;
      if (!preview) {
        throw new Error(payload.error ?? "Release candidate provider preview returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: preview.status === "completed" ? "completed" : "blocked", id: preview.id, message: preview.decision ?? preview.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Provider preview failed." },
      }));
    }
  }

  async function createCandidateDeploymentPlan(candidateID: string) {
    const actionKey = `${candidateID}:deployment-plan`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Creating deployment handoff..." },
    }));
    try {
      const payload = await postJSON<DeploymentPlanEnvelope>(`/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/deployment-plan`, {
        environment: "test_dev",
        approved: true,
      });
      const deployment = payload.deployment;
      if (!deployment) {
        throw new Error(payload.error ?? "Deployment handoff returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: deployment.status === "planned" ? "completed" : "blocked", id: deployment.id, message: deployment.decision ?? deployment.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Deployment handoff failed." },
      }));
    }
  }

  async function publishReleaseCandidateProvider(candidateID: string) {
    const actionKey = `${candidateID}:provider-publish`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Checking publish gate..." },
    }));
    try {
      const payload = await postJSON<ReleaseProviderExecutionEnvelope>(
        `/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/provider-publish`,
        {},
      );
      const execution = payload.release_provider_execution;
      if (!execution) {
        throw new Error(payload.error ?? "Candidate publish returned no execution.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: execution.status === "completed" ? "completed" : "blocked", id: execution.id, message: execution.decision ?? execution.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Candidate publish failed." },
      }));
    }
  }

  async function planCandidatePRMR(candidateID: string) {
    const actionKey = `${candidateID}:pr-mr-plan`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Planning PR/MR..." },
    }));
    try {
      const payload = await postJSON<GitProviderActionEnvelope>(`/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/pr-mr-plan`, {});
      const plan = payload.git_provider_plan;
      if (!plan) {
        throw new Error(payload.error ?? "Candidate PR/MR plan returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: plan.status === "pr_mr_plan_ready" ? "completed" : "blocked", id: plan.id, message: plan.decision ?? plan.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Candidate PR/MR plan failed." },
      }));
    }
  }

  async function runCandidateDeploymentDryRun(candidateID: string) {
    const actionKey = `${candidateID}:deployment-execution`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Running deploy dry-run..." },
    }));
    try {
      const payload = await postJSON<DeploymentExecutionEnvelope>(
        `/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/deployment-execution`,
        { mode: "dry_run", environment: "test_dev" },
      );
      const execution = payload.execution;
      if (!execution) {
        throw new Error(payload.error ?? "Candidate deployment execution returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: execution.status === "completed" ? "completed" : "blocked", id: execution.id, message: execution.decision ?? execution.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Candidate deployment dry-run failed." },
      }));
    }
  }

  return (
    <main className="shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brandMark">
            <Layers3 size={20} />
          </div>
          <div>
            <strong>Moyuan</strong>
            <span>Control Console</span>
          </div>
        </div>

        <nav className="navList">
          {nav.map((item) => (
            <button className={`navItem ${activeView === item.label ? "active" : ""}`} key={item.label} onClick={() => setActiveView(item.label)} type="button">
              <item.icon size={17} />
              <span>{item.label}</span>
            </button>
          ))}
        </nav>

        <div className="sideCard">
          <div className="sideCardTop">
            <span>Runtime</span>
            <StatusPill tone={snapshot.backendStatus} label={snapshot.mode === "live" ? "live" : "demo"} />
          </div>
          <strong>3000 &gt; 8080</strong>
          <small>Next.js 16 / Go Gin</small>
        </div>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">AI Engineering Ops</p>
            <h1>{snapshot.project.name}</h1>
          </div>
          <div className="topActions">
            <div className="searchBox">
              <Search size={16} />
              <span>Jump to issue, run, provider...</span>
            </div>
            <button className="iconButton" type="button" aria-label="Run selected issue">
              <Play size={18} />
            </button>
          </div>
        </header>

        <section className="heroGrid">
          <MetricCard label="Issues" value={snapshot.stats.issues} tone="neutral" detail={`${snapshot.stats.accepted} accepted`} />
          <MetricCard label="Blocked" value={snapshot.stats.blocked} tone={snapshot.stats.blocked > 0 ? "warning" : "ok"} detail="dependency / approval" />
          <MetricCard label="Providers" value={snapshot.stats.providers} tone="running" detail="Claude / Codex / API" />
          <MetricCard label="Deploys" value={snapshot.stats.executions} tone="neutral" detail={`${snapshot.stats.deployments} plans`} />
        </section>

        <section className="opsGrid" hidden={!viewVisible(activeView, ["Projects", "Deployments"])}>
          <div className="panel requirementPanel">
            <PanelTitle icon={<Sparkles size={18} />} title="Requirement Intake" meta="plan to issue graph" />
            <form className="requirementForm" onSubmit={submitRequirement}>
              <textarea
                aria-label="Requirement text"
                onChange={(event) => setRequirementText(event.target.value)}
                placeholder="Describe a feature, fix, or operational change with verification expectations..."
                rows={3}
                value={requirementText}
              />
              <div className="formFooter">
                <button className="primaryButton" disabled={requirementState.status === "planning"} type="submit">
                  <Play size={16} />
                  <span>{requirementState.status === "planning" ? "Planning" : "Plan Issues"}</span>
                </button>
                {requirementState.status !== "idle" ? (
                  <div className={`formResult ${requirementState.status}`}>
                    <strong>{requirementState.status.replaceAll("_", " ")}</strong>
                    <span>{requirementState.message}</span>
                    {requirementState.epic ? <code>{requirementState.epic}</code> : null}
                  </div>
                ) : null}
              </div>
              <SchemaFeedback errors={schemaErrors.requirement} />
            </form>
          </div>

          <div className="panel executionPanel">
            <PanelTitle icon={<Rocket size={18} />} title="Deployment Executions" meta={`${snapshot.executions.length} recent`} />
            <div className="deploymentControls">
              <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} onClick={() => void suggestRelease()} type="button">
                <GitBranch size={13} />
                <span>Suggest Release</span>
              </button>
              <button
                className="inlineActionButton"
                disabled={deploymentActionState.status === "running" || !latestDeployment}
                onClick={() => void runDeploymentDryRun(latestDeployment?.id)}
                type="button"
              >
                <Rocket size={13} />
                <span>Dry Run</span>
              </button>
              <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} onClick={() => void runResourceHealthScan()} type="button">
                <Server size={13} />
                <span>Health Scan</span>
              </button>
              <button
                className="inlineActionButton"
                disabled={deploymentActionState.status === "running" || !latestRollbackCandidate}
                onClick={() => void previewRollbackExecution(latestRollbackCandidate?.id)}
                type="button"
              >
                <AlertTriangle size={13} />
                <span>Rollback Preview</span>
              </button>
              <button
                className="inlineActionButton"
                disabled={deploymentActionState.status === "running"}
                onClick={() => void summarizeDeploymentMonitor()}
                type="button"
              >
                <Activity size={13} />
                <span>Monitor Summary</span>
              </button>
              <button
                className="inlineActionButton"
                disabled={deploymentActionState.status === "running" || snapshot.executions.length === 0}
                onClick={() => void createPostDeploymentVerification()}
                type="button"
              >
                <ShieldCheck size={13} />
                <span>Verify</span>
              </button>
              <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} onClick={() => void createDeploymentRehearsal()} type="button">
                <CircleDotDashed size={13} />
                <span>Rehearsal</span>
              </button>
              <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} onClick={() => void runRehearsalScheduler()} type="button">
                <RefreshCw size={13} />
                <span>Scheduler</span>
              </button>
              <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} onClick={() => void createReleaseAdmission()} type="button">
                <ShieldCheck size={13} />
                <span>Admission</span>
              </button>
              <button
                className="inlineActionButton"
                disabled={deploymentActionState.status === "running" || !latestAdmission}
                onClick={() => void createDeploymentRiskHandoff()}
                type="button"
              >
                <Wrench size={13} />
                <span>Risk Handoff</span>
              </button>
              {deploymentActionState.message ? (
                <small className={`actionMessage ${deploymentActionState.status}`}>
                  {deploymentActionState.id ? `${compactID(deploymentActionState.id)} / ` : ""}
                  {deploymentActionState.message}
                </small>
              ) : null}
            </div>
            <div className="executionList">
              {snapshot.executions.length > 0 ? (
                snapshot.executions.map((execution) => (
                  <div className="executionItem" key={execution.id}>
                    <div>
                      <strong>{execution.mode}</strong>
                      <span>
                        {[
                          execution.decision,
                          execution.smoke_status ? `smoke ${execution.smoke_status}` : "",
                          execution.monitor_status ? `monitor ${execution.monitor_status}` : "",
                          execution.rollback_required ? "rollback suggested" : "",
                          execution.approval_id ? `approval ${compactID(execution.approval_id)}` : "",
                          execution.approval_consumed ? "approval consumed" : "",
                        ]
                          .filter(Boolean)
                          .join(" / ")}
                      </span>
                    </div>
                    <StatusPill tone={toneForStatus(execution.status)} label={execution.status} />
                  </div>
                ))
              ) : snapshot.deployments.length > 0 ? (
                snapshot.deployments.map((deployment) => (
                  <div className="executionItem" key={deployment.id}>
                    <div>
                      <strong>{deployment.environment}</strong>
                      <span>{deployment.decision}</span>
                    </div>
                    <StatusPill tone={toneForStatus(deployment.status)} label={deployment.status} />
                  </div>
                ))
              ) : (
                <div className="emptyState">No deployment executions yet</div>
              )}
            </div>
            <div className="maintenanceList">
              {latestMonitorSummary ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{latestMonitorSummary.environment || "all environments"}</strong>
                    <span>{`${latestMonitorSummary.decision} / ${latestMonitorSummary.history_count} histories / ${latestMonitorSummary.failed_count} failed`}</span>
                    <span>{`${latestMonitorSummary.rollback_count} rollback / window ${latestMonitorSummary.window_size}`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestMonitorSummary.status)} label={latestMonitorSummary.status} />
                </div>
              ) : null}
              {latestVerification ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestVerification.execution_id || latestVerification.id)}</strong>
                    <span>{`${latestVerification.decision} / ${latestVerification.monitor_decision || "monitor pending"}`}</span>
                    <span>
                      {latestVerification.risk_handoff_recommended
                        ? `${latestVerification.risk_source_type || "risk"} ${compactID(latestVerification.risk_source_id || "")}`
                        : "risk handoff not required"}
                    </span>
                  </div>
                  <StatusPill tone={toneForStatus(latestVerification.risk_handoff_recommended ? "warning" : latestVerification.status)} label={latestVerification.status} />
                </div>
              ) : null}
              {latestRehearsal ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestRehearsal.id)}</strong>
                    <span>{`${latestRehearsal.decision} / ${latestRehearsal.timeline.length} steps`}</span>
                    <span>{`${latestRehearsal.monitor_status || "monitor pending"} / ${latestRehearsal.rollback_decision || "rollback pending"}`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestRehearsal.status)} label={latestRehearsal.status} />
                </div>
              ) : null}
              {latestAdmission ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestAdmission.id)}</strong>
                    <span>{`${latestAdmission.decision} / ${latestAdmission.signals.length} signals`}</span>
                    <span>{`${latestAdmission.policy_id || snapshot.release_admission_policy?.id || "policy pending"} / ${latestAdmission.matched_rules.length} matched rules`}</span>
                    <span>{latestAdmission.policy_decision?.reasons[0] || latestAdmission.reasons[0] || "reason pending"}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestAdmission.status)} label={latestAdmission.status} />
                </div>
              ) : null}
              {latestSchedulerRun ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestSchedulerRun.id)}</strong>
                    <span>{`${latestSchedulerRun.decision} / created ${latestSchedulerRun.created_count} / skipped ${latestSchedulerRun.skipped_count}`}</span>
                    <span>{`${latestSchedulerRun.blocked_count} blocked / ${latestSchedulerRun.manual_count} manual / ${latestSchedulerRun.targets[0]?.reason || "target reason pending"}`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestSchedulerRun.status)} label={latestSchedulerRun.status} />
                </div>
              ) : null}
              {latestRiskHandoff ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestRiskHandoff.id)}</strong>
                    <span>{`${latestRiskHandoff.decision} / ${latestRiskHandoff.failure_class}`}</span>
                    <span>{`${latestRiskHandoff.review_decision || (latestRiskHandoff.review_required ? "pending review" : "review not required")} / ${
                      latestRiskHandoff.repair_plan_id ? compactID(latestRiskHandoff.repair_plan_id) : "repair not required"
                    }`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestRiskHandoff.status)} label={latestRiskHandoff.status} />
                </div>
              ) : null}
              {latestRiskReviewQueueItem ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestRiskReviewQueueItem.handoff_id)}</strong>
                    <span>{`${latestRiskReviewQueueItem.decision} / ${latestRiskReviewQueueItem.failure_class}`}</span>
                    <span>{`${latestRiskReviewQueueItem.review_decision || "pending"} / ${latestRiskReviewQueueItem.review_next_step || latestRiskReviewQueueItem.reasons[0] || "next step pending"}`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestRiskReviewQueueItem.status)} label={latestRiskReviewQueueItem.status} />
                </div>
              ) : null}
              {latestRiskReview ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestRiskReview.id)}</strong>
                    <span>{`${latestRiskReview.decision} / ${latestRiskReview.next_step || "next step pending"}`}</span>
                    <span>{latestRiskReview.reason || latestRiskReview.failure_class || "review reason pending"}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestRiskReview.status)} label={latestRiskReview.status} />
                </div>
              ) : null}
              {snapshot.rollback_executions.slice(0, 2).map((rollback) => (
                <div className="maintenanceItem" key={rollback.id}>
                  <div>
                    <strong>{compactID(rollback.execution_id)}</strong>
                    <span>{`${rollback.decision} / ${rollback.mode} / ${rollback.step_count} steps`}</span>
                    <span>{rollback.approval_id ? `approval ${compactID(rollback.approval_id)}` : "approval not consumed"}</span>
                  </div>
                  <StatusPill tone={toneForStatus(rollback.status)} label={rollback.status} />
                </div>
              ))}
              {snapshot.post_deployment_histories.length > 0 ? (
                snapshot.post_deployment_histories.slice(0, 3).map((history) => (
                  <div className="maintenanceItem" key={history.id}>
                    <div>
                      <strong>{compactID(history.execution_id)}</strong>
                      <span>{`${history.decision} / ${history.checks.length} checks / rollback ${history.rollback.status}`}</span>
                      {history.checks[0]?.template_id ? (
                        <span>{`${history.checks[0].template_id}${history.severity ? ` / ${history.severity}` : ""}`}</span>
                      ) : null}
                    </div>
                    <StatusPill tone={toneForStatus(history.failure_class === "none" ? history.status : history.failure_class)} label={history.failure_class} />
                  </div>
                ))
              ) : (
                <div className="emptyState compact">
                  {hasDeploymentOpsHistory ? "No post deployment history" : "No deployment ops history"}
                </div>
              )}
            </div>
          </div>
        </section>

        <section className="operationGrid" hidden={!viewVisible(activeView, ["Projects", "Deployments", "Operations"])}>
          <div className="panel operationHistoryPanel">
            <PanelTitle icon={<ScrollText size={18} />} title="Operation History" meta={`${snapshot.operation_history.length} traced`} />
            <div className="operationList">
              {snapshot.operation_history.length > 0 ? (
                snapshot.operation_history.map((operation) => (
                  <button
                    className={`operationItem ${selectedOperation?.id === operation.id ? "selected" : ""}`}
                    key={operation.id}
                    onClick={() => setSelectedOperationID(operation.id)}
                    type="button"
                  >
                    <StatusDot tone={operation.tone} />
                    <div>
                      <strong>{operation.title}</strong>
                      <span>{operation.detail}</span>
                    </div>
                    <time>{operation.time}</time>
                  </button>
                ))
              ) : (
                <div className="emptyState">No operation history</div>
              )}
            </div>
          </div>

          <div className="panel operationDetailPanel">
            <PanelTitle icon={<Activity size={18} />} title="Execution Detail" meta={selectedOperationDetail?.operation_type ?? selectedOperation?.type ?? "operation"} />
            {selectedOperation ? (
              <div className="operationDetail">
                <div className="detailHeader">
                  <div>
                    <strong>{selectedOperation.title}</strong>
                    <span>{selectedOperationDetail?.operation || selectedOperation.id}</span>
                  </div>
                  <div className="detailHeaderActions">
                    <StatusPill tone={toneForStatus(selectedOperationDetail?.status || selectedOperation.status)} label={selectedOperationDetail?.status || selectedOperation.status} />
                    <button aria-label="Refresh operation detail" className="iconActionButton" onClick={() => router.refresh()} type="button">
                      <RefreshCw size={14} />
                    </button>
                  </div>
                </div>
                <dl>
                  <div>
                    <dt>Decision</dt>
                    <dd>{selectedOperationDetail?.decision || selectedOperation.decision}</dd>
                  </div>
                  <div>
                    <dt>Primary Ref</dt>
                    <dd>{selectedOperationDetail?.primary_ref || selectedOperation.primary_ref || "none"}</dd>
                  </div>
                  <div>
                    <dt>Secondary Ref</dt>
                    <dd>{selectedOperationDetail?.secondary_ref || selectedOperation.secondary_ref || "none"}</dd>
                  </div>
                  <div>
                    <dt>Evidence</dt>
                    <dd>{selectedEvidenceRecords.length > 0 ? selectedEvidenceRecords.map((record) => compactID(record.id)).join(", ") : "none"}</dd>
                  </div>
                </dl>
                {(selectedOperationDetail?.reasons.length ? selectedOperationDetail.reasons : selectedOperation.reasons).length > 0 ? (
                  <div className="detailChips">
                    {(selectedOperationDetail?.reasons.length ? selectedOperationDetail.reasons : selectedOperation.reasons).slice(0, 3).map((reason) => (
                      <code key={reason}>{reason}</code>
                    ))}
                  </div>
                ) : null}
                {selectedOperation.metadata.length > 0 ? (
                  <div className="detailChips subtle">
                    {selectedOperationDetail ? <code>detail api</code> : null}
                    {selectedOperationDetail?.summary.evidence_count ? <code>{selectedOperationDetail.summary.evidence_count} evidence</code> : null}
                    {selectedOperationDetail?.summary.artifact_count ? <code>{selectedOperationDetail.summary.artifact_count} artifacts</code> : null}
                    {selectedOperation.metadata.map((item) => (
                      <code key={item}>{item}</code>
                    ))}
                  </div>
                ) : null}
                <div className="evidenceDrilldown">
                  <div className="detailSectionTitle">
                    <strong>Evidence Chain</strong>
                    <span>{selectedEvidenceRecords.length} records</span>
                  </div>
                  {selectedEvidenceRecords.length > 0 ? (
                    selectedEvidenceRecords.map((record) => (
                      <div className="evidenceCard" key={record.id}>
                        <div className="evidenceCardHeader">
                          <div>
                            <strong>{record.operation}</strong>
                            <span>{record.id}</span>
                          </div>
                          <StatusPill tone={toneForStatus(record.status)} label={record.status} />
                        </div>
                        <dl>
                          <div>
                            <dt>Decision</dt>
                            <dd>{record.decision}</dd>
                          </div>
                          <div>
                            <dt>Artifacts</dt>
                            <dd>{record.artifact_count}</dd>
                          </div>
                        </dl>
                        {record.reasons.length > 0 ? (
                          <div className="detailChips">
                            {record.reasons.slice(0, 3).map((reason) => (
                              <code key={`${record.id}-${reason}`}>{reason}</code>
                            ))}
                          </div>
                        ) : null}
                        {record.artifacts.length > 0 ? (
                          <div className="artifactList">
                            {record.artifacts.map((artifact, index) => (
                              <code key={`${record.id}-${artifact.kind}-${index}`}>
                                {artifact.kind}
                                {artifact.path ? ` / ${artifact.path}` : ""}
                              </code>
                            ))}
                          </div>
                        ) : null}
                      </div>
                    ))
                  ) : (
                    <div className="emptyState compact">No linked evidence records</div>
                  )}
                </div>
              </div>
            ) : (
              <div className="emptyState">Select an operation</div>
            )}
          </div>
        </section>

        <section className="observabilityGrid" hidden={!viewVisible(activeView, ["Projects", "Operations", "Audit"])}>
          <div className="panel">
            <PanelTitle
              icon={<ScrollText size={18} />}
              title="Audit Export"
              meta={operationsAuditExport ? `${operationsAuditExport.timeline_item_count} timeline` : "not generated"}
            />
            {operationsAuditExport ? (
              <div className="signalList">
                <div className="signalItem">
                  <div className="signalHeader">
                    <strong>{compactID(operationsAuditExport.id)}</strong>
                    <StatusPill tone={operationsAuditExport.attention_item_count > 0 ? "warning" : "ok"} label={operationsAuditExport.format} />
                  </div>
                  <span>
                    evidence {operationsAuditExport.evidence_ref_count} / verification {operationsAuditExport.post_deployment_verification_count} / refs{" "}
                    {operationsAuditExport.resource_deployment_ref_count}
                  </span>
                  <div className="signalMeta">
                    <code>{operationsAuditExport.redaction_applied ? "redaction applied" : "redaction clear"}</code>
                    <code>{operationsAuditExport.risk_handoff_recommended_count} risk handoff</code>
                    <code>{shortTimestamp(operationsAuditExport.generated_at)}</code>
                  </div>
                  <div className="routeCandidateGrid compact">
                    {Object.entries(operationsAuditExport.by_type)
                      .slice(0, 4)
                      .map(([type, count]) => (
                        <div className="routeCandidate" key={type}>
                          <strong>{type}</strong>
                          <span>{count} records</span>
                        </div>
                      ))}
                  </div>
                </div>
              </div>
            ) : (
              <div className="emptyState">No audit export</div>
            )}
          </div>

          <div className="panel">
            <PanelTitle icon={<ShieldCheck size={18} />} title="Decision Ledger" meta={decisionLedger ? `${decisionLedger.entry_count} entries` : "not generated"} />
            <div className="signalList">
              {decisionLedger ? (
                <>
                  <div className="signalItem">
                    <div className="signalHeader">
                      <strong>{compactID(decisionLedger.id)}</strong>
                      <StatusPill tone={decisionLedger.attention_count > 0 ? "warning" : "ok"} label={`${decisionLedger.attention_count} attention`} />
                    </div>
                    <span>
                      evidence {decisionLedger.evidence_ref_count} / redaction {decisionLedger.redaction_applied ? "applied" : "clear"}
                    </span>
                    <div className="signalMeta">
                      {Object.entries(decisionLedger.by_source_type)
                        .slice(0, 4)
                        .map(([type, count]) => (
                          <code key={type}>
                            {type}:{count}
                          </code>
                        ))}
                    </div>
                  </div>
                  {decisionEntries.slice(0, 3).map((entry) => (
                    <div className="signalItem" key={entry.id}>
                      <div className="signalHeader">
                        <strong>{entry.source_type}</strong>
                        <StatusPill tone={toneForStatus(entry.status)} label={entry.decision} />
                      </div>
                      <span>
                        {compactID(entry.source_id)} / {entry.environment || "all"}
                      </span>
                      <div className="signalMeta">
                        {entry.rule_refs[0] ? <code>{entry.rule_refs[0]}</code> : null}
                        {entry.evidence_refs.length ? <code>{entry.evidence_refs.length} evidence</code> : null}
                        {entry.parent_ref ? <code>{compactID(entry.parent_ref)}</code> : null}
                      </div>
                      {entry.reasons[0] ? <small>{entry.reasons[0]}</small> : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">No decision ledger</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<Lock size={18} />} title="Write Proof" meta={writeProofReport ? `${writeProofReport.proof_count} proofs` : "not generated"} />
            <div className="signalList">
              {writeProofReport ? (
                <>
                  <div className="signalItem">
                    <div className="signalHeader">
                      <strong>{compactID(writeProofReport.id)}</strong>
                      <StatusPill tone={writeProofReport.blocked_count > 0 ? "blocked" : "ok"} label={`${writeProofReport.blocked_count} blocked`} />
                    </div>
                    <span>
                      manual {writeProofReport.manual_required_count} / redaction {writeProofReport.redaction_applied ? "applied" : "clear"}
                    </span>
                    <div className="signalMeta">
                      {Object.entries(writeProofReport.by_operation_type)
                        .slice(0, 4)
                        .map(([type, count]) => (
                          <code key={type}>
                            {type}:{count}
                          </code>
                        ))}
                    </div>
                  </div>
                  {writeProofs.slice(0, 4).map((proof) => (
                    <div className="signalItem" key={proof.id}>
                      <div className="signalHeader">
                        <strong>{proof.operation_type}</strong>
                        <StatusPill tone={toneForStatus(proof.status)} label={proof.decision} />
                      </div>
                      <span>
                        {proof.provider || "provider"} / {proof.mode || "mode"} / {compactID(proof.operation_id)}
                      </span>
                      <div className="signalMeta">
                        <code>{proof.write_enabled ? "write enabled" : "write disabled"}</code>
                        <code>{proof.dry_run ? "dry-run" : "write path"}</code>
                        <code>{proof.approval_satisfied ? "approval ok" : proof.approval_required ? "approval required" : "approval n/a"}</code>
                        {proof.secret_ref_status ? <code>{proof.secret_ref_status}</code> : null}
                        {proof.provider_evidence_refs.length ? <code>{proof.provider_evidence_refs.length} evidence</code> : null}
                      </div>
                      {proof.least_privilege ? <small>{proof.least_privilege}</small> : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">No write proofs</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<RefreshCw size={18} />} title="Control Runner" meta={latestControlLoopRun ? compactID(latestControlLoopRun.id) : "no runs"} />
            <div className="signalList">
              {latestControlLoopRun ? (
                <div className="signalItem">
                  <div className="signalHeader">
                    <strong>{latestControlLoopRun.trigger}</strong>
                    <StatusPill tone={toneForStatus(latestControlLoopRun.status)} label={latestControlLoopRun.decision} />
                  </div>
                  <span>
                    {latestControlLoopRun.steps.length} steps / {shortTimestamp(latestControlLoopRun.finished_at || latestControlLoopRun.started_at)}
                  </span>
                  <div className="routeCandidateGrid compact">
                    {latestControlLoopRun.steps.slice(0, 4).map((step) => (
                      <div className="routeCandidate" key={step.id}>
                        <strong>{step.type}</strong>
                        <span>{step.summary || step.decision}</span>
                        <div className="signalMeta">
                          <code>{step.status}</code>
                          {step.evidence_count ? <code>{step.evidence_count} evidence</code> : null}
                        </div>
                      </div>
                    ))}
                  </div>
                  {latestControlLoopRun.reasons[0] ? <small>{latestControlLoopRun.reasons[0]}</small> : null}
                </div>
              ) : (
                <div className="emptyState">No control loop runs</div>
              )}
            </div>
          </div>
        </section>

        <section className="mainGrid" hidden={!viewVisible(activeView, ["Projects", "Issue Graph"])}>
          <div className="panel graphPanel">
            <PanelTitle icon={<Network size={18} />} title="Issue Graph" meta="dependency aware" />
            <div className="graphCanvas">
              {Object.entries(groupedIssues).map(([lane, issues]) => (
                <div className="lane" key={lane}>
                  <div className="laneTitle">{laneLabels[lane as IssueNode["lane"]]}</div>
                  <div className="laneNodes">
                    {issues.map((issue) => (
                      <button
                        className={`issueNode ${statusClass(issue.status)} ${selectedIssue?.id === issue.id ? "selected" : ""}`}
                        key={issue.id}
                        onClick={() => setSelectedIssueID(issue.id)}
                        type="button"
                      >
                        <span>{issue.title}</span>
                        <small>{issue.role}</small>
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <aside className="panel inspector">
            <PanelTitle icon={<CircleDotDashed size={18} />} title="Inspector" meta={selectedIssue?.id ?? "issue"} />
            {selectedIssue ? (
              <div className="inspectorBody">
                <h2>{selectedIssue.title}</h2>
                <StatusPill tone={toneForStatus(selectedIssue.status)} label={selectedIssue.status} />
                <dl>
                  <div>
                    <dt>Run</dt>
                    <dd>{selectedIssue.run_id ?? "not started"}</dd>
                  </div>
                  <div>
                    <dt>Subagent</dt>
                    <dd>{selectedIssue.subagent_id ?? "not assigned"}</dd>
                  </div>
                  <div>
                    <dt>Role</dt>
                    <dd>{selectedIssue.role}</dd>
                  </div>
                  <div>
                    <dt>Runtime</dt>
                    <dd>{selectedIssue.runtime ?? "pending"}</dd>
                  </div>
                  <div>
                    <dt>Runtime Status</dt>
                    <dd>{selectedIssue.runtime_status ?? "pending"}</dd>
                  </div>
                  <div>
                    <dt>Provider</dt>
                    <dd>{selectedIssue.provider ?? "route pending"}</dd>
                  </div>
                  <div>
                    <dt>Quality</dt>
                    <dd>{selectedIssue.quality ?? "not started"}</dd>
                  </div>
                  <div>
                    <dt>Review</dt>
                    <dd>{selectedIssue.review_status ?? "not reviewed"}</dd>
                  </div>
                  <div>
                    <dt>Quality Report</dt>
                    <dd>{selectedIssue.quality_report_id ?? "none"}</dd>
                  </div>
                </dl>
                {selectedIssue.quality_decision ? (
                  <div className="decisionStrip">
                    <ShieldCheck size={16} />
                    <div>
                      <strong>{selectedIssue.quality_decision}</strong>
                      <span>{selectedIssue.quality_reasons?.[0] ?? "quality explanation available"}</span>
                    </div>
                  </div>
                ) : null}
                {selectedIssue.skills && selectedIssue.skills.length > 0 ? (
                  <div className="chipSection">
                    <span>Skills</span>
                    <div className="chipList">
                      {selectedIssue.skills.map((skill) => (
                        <code key={skill}>{skill}</code>
                      ))}
                    </div>
                  </div>
                ) : null}
                {selectedIssue.output_contract && selectedIssue.output_contract.length > 0 ? (
                  <div className="chipSection">
                    <span>Output Contract</span>
                    <div className="chipList">
                      {selectedIssue.output_contract.map((item) => (
                        <code key={item}>{item}</code>
                      ))}
                    </div>
                  </div>
                ) : null}
                {selectedIssue.depends_on && selectedIssue.depends_on.length > 0 ? (
                  <div className="dependencyList">
                    {selectedIssue.depends_on.map((dependency) => (
                      <span key={dependency}>
                        <ChevronRight size={13} />
                        {dependency}
                      </span>
                    ))}
                  </div>
                ) : null}
                {selectedIssue.quality_reasons && selectedIssue.quality_reasons.length > 1 ? (
                  <div className="reasonList">
                    {selectedIssue.quality_reasons.slice(1).map((reason) => (
                      <span key={reason}>{reason}</span>
                    ))}
                  </div>
                ) : null}
                {selectedIssue.blocking_findings && selectedIssue.blocking_findings.length > 0 ? (
                  <div className="findingList">
                    {selectedIssue.blocking_findings.map((finding) => (
                      <div key={finding.id}>
                        <strong>
                          {finding.severity} / {finding.category}
                        </strong>
                        <span>{finding.message}</span>
                        {finding.path ? <code>{finding.path}</code> : null}
                      </div>
                    ))}
                  </div>
                ) : null}
                {selectedIssue.blocked_reason ? <div className="warningLine">{selectedIssue.blocked_reason}</div> : null}
              </div>
            ) : null}
          </aside>
        </section>

        <section className="lowerGrid" hidden={!viewVisible(activeView, ["Projects", "Runs", "Quality", "Memory"])}>
          <div className="panel">
            <PanelTitle icon={<Activity size={18} />} title="Run Timeline" meta={`${snapshot.runs.length} runs`} />
            <div className="timeline">
              {snapshot.timeline.map((event) => (
                <div className="timelineItem" key={event.id}>
                  <StatusDot tone={event.tone} />
                  <div>
                    <strong>{event.title}</strong>
                    <span>{event.detail}</span>
                  </div>
                  <time>{event.time}</time>
                </div>
              ))}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<ShieldCheck size={18} />} title="Quality Gates" meta="diff-first" />
            <div className="qualityList">
              {snapshot.quality.map((item) => (
                <div className="qualityItem" key={item.id}>
                  {item.severity === "ok" ? <CheckCircle2 size={18} /> : <AlertTriangle size={18} />}
                  <div>
                    <strong>{item.title}</strong>
                    <span>{item.detail}</span>
                  </div>
                  <StatusPill tone={item.severity} label={item.status} />
                </div>
              ))}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<MemoryStick size={18} />} title="Memory" meta="compact aware" />
            <div className="memoryList">
              {snapshot.memory.map((record) => (
                <div className="memoryItem" key={record.id}>
                  <span>{record.kind}</span>
                  <strong>{record.summary}</strong>
                  <meter value={record.score} min="0" max="1" />
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="lowerGrid" hidden={!viewVisible(activeView, ["Projects", "Batches", "Runs", "Quality"])}>
          <div className="panel">
            <PanelTitle icon={<Layers3 size={18} />} title="Batch Plans" meta={`${snapshot.batch_plans.length} plans`} />
            <div className="signalList">
              {snapshot.batch_plans.length > 0 ? (
                snapshot.batch_plans.map((plan) => {
                  const dryRunState = batchActionState[plan.id];
                  const mergeState = batchActionState[`${plan.id}:merge`];
                  return (
                    <div className="signalItem" key={plan.id}>
                      <div className="signalHeader">
                        <strong>{compactID(plan.epic_id || plan.id)}</strong>
                        <StatusPill tone={toneForStatus(plan.status)} label={plan.decision} />
                      </div>
                      <span>
                        dispatch {plan.dispatch_count} / waiting {plan.waiting_count} / blocked {plan.blocked_count}
                      </span>
                      <div className="signalMeta">
                        <code>{plan.mode}</code>
                        <code>{plan.max_parallel} parallel</code>
                        <code>{plan.runtime_slots} slots</code>
                        {plan.write_scope_conflict_count ? <code>{plan.write_scope_conflict_count} conflicts</code> : null}
                      </div>
                      {plan.reasons[0] ? <small>{plan.reasons[0]}</small> : null}
                      <div className="signalActions">
                        <button className="inlineActionButton" disabled={dryRunState?.status === "running"} onClick={() => void runBatchDryRun(plan.id)} type="button">
                          <Play size={13} />
                          <span>{dryRunState?.status === "running" ? "Running" : "Dry Run"}</span>
                        </button>
                        <button className="inlineActionButton" disabled={mergeState?.status === "running"} onClick={() => void buildMergeQueue(plan.id)} type="button">
                          <ShieldCheck size={13} />
                          <span>{mergeState?.status === "running" ? "Building" : "Merge Queue"}</span>
                        </button>
                      </div>
                      <ActionFeedback state={dryRunState} />
                      <ActionFeedback state={mergeState} />
                    </div>
                  );
                })
              ) : (
                <div className="emptyState">No batch plans</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<TerminalSquare size={18} />} title="Batch Runs" meta={`${snapshot.batch_runs.length} runs`} />
            <div className="signalList">
              {snapshot.batch_runs.length > 0 ? (
                snapshot.batch_runs.map((run) => (
                  <div className="signalItem" key={run.id}>
                    <div className="signalHeader">
                      <strong>{compactID(run.id)}</strong>
                      <StatusPill tone={toneForStatus(run.status)} label={run.decision} />
                    </div>
                    <span>
                      {run.mode} / {run.item_count} items / parallel {run.parallelism || 1} / accepted {run.accepted_count}
                    </span>
                    <div className="signalMeta">
                      <code>{compactID(run.batch_id)}</code>
                      <code>rework {run.needs_rework_count}</code>
                      <code>blocked {run.blocked_count}</code>
                      {run.requested_by ? <code>{run.requested_by}</code> : null}
                    </div>
                    <div className="routeCandidateGrid compact">
                      {run.items.slice(0, 3).map((item) => (
                        <div className="routeCandidate" key={`${run.id}-${item.issue_id}`}>
                          <strong>{compactID(item.issue_id)}</strong>
                          <span>{item.decision}</span>
                          <div className="signalMeta">
                            {item.worker_slot ? <code>slot {item.worker_slot}</code> : null}
                            {item.runtime_id ? <code>{item.runtime_id}</code> : null}
                            {item.worktree_id ? <code>{compactID(item.worktree_id)}</code> : null}
                            {item.quality_report_id ? <code>{compactID(item.quality_report_id)}</code> : null}
                            {item.canceled_reason ? <code>{item.canceled_reason}</code> : null}
                          </div>
                        </div>
                      ))}
                    </div>
                    {run.reasons[0] ? <small>{run.reasons[0]}</small> : null}
                  </div>
                ))
              ) : (
                <div className="emptyState">No batch runs</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<GitBranch size={18} />} title="Worktrees & Merge" meta={`${snapshot.worktrees.length} worktrees / ${snapshot.merge_queues.length} queues`} />
            <div className="signalList">
              {snapshot.merge_queues.length > 0 ? (
                snapshot.merge_queues.map((queue) => {
                  const previewState = batchActionState[`${queue.id}:integration-preview`];
                  return (
                    <div className="signalItem" key={queue.id}>
                      <div className="signalHeader">
                        <strong>{compactID(queue.id)}</strong>
                        <StatusPill tone={toneForStatus(queue.status)} label={queue.decision} />
                      </div>
                      <span>
                        ready {queue.ready_count} / rework {queue.needs_rework_count} / blocked {queue.blocked_count}
                      </span>
                      <div className="signalMeta">
                        <code>{compactID(queue.batch_id)}</code>
                        {queue.batch_run_id ? <code>{compactID(queue.batch_run_id)}</code> : null}
                        {queue.reasons[0] ? <code>{queue.reasons[0]}</code> : null}
                      </div>
                      <div className="routeCandidateGrid compact">
                        {queue.items.slice(0, 3).map((item) => (
                          <div className="routeCandidate" key={`${queue.id}-${item.issue_id}`}>
                            <strong>{compactID(item.issue_id)}</strong>
                            <span>{item.reason || item.decision}</span>
                            <div className="signalMeta">
                              <code>{item.status}</code>
                              {item.worktree_id ? <code>{compactID(item.worktree_id)}</code> : null}
                            </div>
                          </div>
                        ))}
                      </div>
                      <div className="signalActions">
                        <button
                          className="inlineActionButton"
                          disabled={previewState?.status === "running"}
                          onClick={() => void buildIntegrationPreview(queue.id)}
                          type="button"
                        >
                          <GitBranch size={13} />
                          <span>{previewState?.status === "running" ? "Previewing" : "Integration Preview"}</span>
                        </button>
                      </div>
                      <ActionFeedback state={previewState} />
                    </div>
                  );
                })
              ) : (
                <div className="emptyState compact">No merge queues</div>
              )}
              {snapshot.worktrees.slice(0, 3).map((record) => (
                <div className="signalItem" key={record.id}>
                  <div className="signalHeader">
                    <strong>{compactID(record.issue_id || record.id)}</strong>
                    <StatusPill tone={toneForStatus(record.status)} label={record.decision} />
                  </div>
                  <span>{record.branch || record.base_ref || "branch pending"}</span>
                  <div className="signalMeta">
                    {record.batch_id ? <code>{compactID(record.batch_id)}</code> : null}
                    {record.worktree_path ? <code>{shortPath(record.worktree_path)}</code> : null}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="panel">
            <PanelTitle
              icon={<Rocket size={18} />}
              title="Integration & Release"
              meta={`${snapshot.integration_previews.length} previews / ${snapshot.release_batches.length} batches`}
            />
            <div className="signalList">
              {snapshot.integration_previews.length > 0 ? (
                snapshot.integration_previews.slice(0, 3).map((preview) => {
                  const applyState = batchActionState[`${preview.id}:apply-dry-run`];
                  return (
                    <div className="signalItem" key={preview.id}>
                      <div className="signalHeader">
                        <strong>{compactID(preview.id)}</strong>
                        <StatusPill tone={toneForStatus(preview.status)} label={preview.decision} />
                      </div>
                      <span>
                        ready {preview.ready_count} / conflict {preview.conflict_count} / blocked {preview.blocked_count}
                      </span>
                      <div className="signalMeta">
                        {preview.merge_queue_id ? <code>{compactID(preview.merge_queue_id)}</code> : null}
                        {preview.integration_branch ? <code>{compactID(preview.integration_branch)}</code> : null}
                        {preview.base_ref ? <code>base {preview.base_ref}</code> : null}
                      </div>
                      <div className="routeCandidateGrid compact">
                        {preview.items.slice(0, 2).map((item) => (
                          <div className="routeCandidate" key={`${preview.id}-${item.issue_id}`}>
                            <strong>{compactID(item.issue_id)}</strong>
                            <span>{item.reason || item.decision}</span>
                            <div className="signalMeta">
                              <code>{item.status}</code>
                              {item.changed_files.length > 0 ? <code>{item.changed_files.length} files</code> : null}
                              {item.conflicted_files.length > 0 ? <code>{item.conflicted_files.length} conflicts</code> : null}
                            </div>
                          </div>
                        ))}
                      </div>
                      <div className="signalActions">
                        <button
                          className="inlineActionButton"
                          disabled={applyState?.status === "running"}
                          onClick={() => void dryRunIntegrationApply(preview.id)}
                          type="button"
                        >
                          <ShieldCheck size={13} />
                          <span>{applyState?.status === "running" ? "Planning" : "Apply Dry Run"}</span>
                        </button>
                      </div>
                      <ActionFeedback state={applyState} />
                    </div>
                  );
                })
              ) : (
                <div className="emptyState compact">No integration previews</div>
              )}
              {snapshot.integration_applies.slice(0, 2).map((apply) => {
                const releaseState = batchActionState[`${apply.id}:release-batch`];
                return (
                  <div className="signalItem" key={apply.id}>
                    <div className="signalHeader">
                      <strong>{compactID(apply.id)}</strong>
                      <StatusPill tone={toneForStatus(apply.status)} label={apply.decision} />
                    </div>
                    <span>
                      {apply.mode} / {apply.write_enabled ? "write enabled" : "guarded"} / {apply.action_count} actions
                    </span>
                    <div className="signalMeta">
                      {apply.preview_id ? <code>{compactID(apply.preview_id)}</code> : null}
                      {apply.target_branch ? <code>{compactID(apply.target_branch)}</code> : null}
                      {apply.requested_by ? <code>{apply.requested_by}</code> : null}
                    </div>
                    <div className="signalActions">
                      <button
                        className="inlineActionButton"
                        disabled={releaseState?.status === "running"}
                        onClick={() => void planReleaseBatch(apply.id)}
                        type="button"
                      >
                        <Rocket size={13} />
                        <span>{releaseState?.status === "running" ? "Checking" : "Release Batch"}</span>
                      </button>
                    </div>
                    <ActionFeedback state={releaseState} />
                  </div>
                );
              })}
              {snapshot.release_batches.slice(0, 2).map((releaseBatch) => {
                const candidateState = batchActionState[`${releaseBatch.id}:candidate`];
                return (
                  <div className="signalItem" key={releaseBatch.id}>
                    <div className="signalHeader">
                      <strong>{compactID(releaseBatch.version || releaseBatch.id)}</strong>
                      <StatusPill tone={toneForStatus(releaseBatch.status)} label={releaseBatch.decision} />
                    </div>
                    <span>
                      ready {releaseBatch.ready_item_count}/{releaseBatch.min_items} / {releaseBatch.release_branch || "release branch pending"}
                    </span>
                    <div className="signalMeta">
                      {releaseBatch.integration_apply_id ? <code>{compactID(releaseBatch.integration_apply_id)}</code> : null}
                      {releaseBatch.source_branch ? <code>{compactID(releaseBatch.source_branch)}</code> : null}
                      {releaseBatch.reasons[0] ? <code>{releaseBatch.reasons[0]}</code> : null}
                    </div>
                    <div className="signalActions">
                      <button
                        className="inlineActionButton"
                        disabled={candidateState?.status === "running"}
                        onClick={() => void planReleaseCandidate(releaseBatch.id)}
                        type="button"
                      >
                        <Rocket size={13} />
                        <span>{candidateState?.status === "running" ? "Planning" : "Candidate"}</span>
                      </button>
                    </div>
                    <ActionFeedback state={candidateState} />
                    {releaseBatch.commands[0] ? <small>{releaseBatch.commands[0]}</small> : null}
                  </div>
                );
              })}
              {snapshot.release_candidates.slice(0, 2).map((candidate) => {
                const applyState = batchActionState[`${candidate.id}:release-branch-apply`];
                const providerState = batchActionState[`${candidate.id}:provider-preview`];
                const deployState = batchActionState[`${candidate.id}:deployment-plan`];
                const publishState = batchActionState[`${candidate.id}:provider-publish`];
                const prmrState = batchActionState[`${candidate.id}:pr-mr-plan`];
                const executionState = batchActionState[`${candidate.id}:deployment-execution`];
                const feedback = snapshot.deployment_feedback.find((item) => item.candidate_id === candidate.id);
                return (
                  <div className="signalItem" key={candidate.id}>
                    <div className="signalHeader">
                      <strong>{compactID(candidate.version || candidate.id)}</strong>
                      <StatusPill tone={toneForStatus(candidate.status)} label={candidate.decision} />
                    </div>
                    <span>
                      {candidate.provider || "provider pending"} / {candidate.release_branch || "release branch pending"}
                    </span>
                    <div className="signalMeta">
                      {candidate.release_batch_id ? <code>{compactID(candidate.release_batch_id)}</code> : null}
                      {candidate.source_branch ? <code>{compactID(candidate.source_branch)}</code> : null}
                      {candidate.deployment_targets.length > 0 ? <code>{candidate.deployment_targets.join(",")}</code> : null}
                    </div>
                    <div className="signalActions">
                      <button
                        className="inlineActionButton"
                        disabled={applyState?.status === "running"}
                        onClick={() => void dryRunReleaseCandidateApply(candidate.id)}
                        type="button"
                      >
                        <GitBranch size={13} />
                        <span>{applyState?.status === "running" ? "Planning" : "Branch Dry Run"}</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={providerState?.status === "running"}
                        onClick={() => void previewReleaseCandidateProvider(candidate.id)}
                        type="button"
                      >
                        <ShieldCheck size={13} />
                        <span>{providerState?.status === "running" ? "Previewing" : "Provider Preview"}</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={deployState?.status === "running"}
                        onClick={() => void createCandidateDeploymentPlan(candidate.id)}
                        type="button"
                      >
                        <Server size={13} />
                        <span>{deployState?.status === "running" ? "Planning" : "Deploy Plan"}</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={publishState?.status === "running"}
                        onClick={() => void publishReleaseCandidateProvider(candidate.id)}
                        type="button"
                      >
                        <Rocket size={13} />
                        <span>{publishState?.status === "running" ? "Checking" : "Publish Gate"}</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={prmrState?.status === "running"}
                        onClick={() => void planCandidatePRMR(candidate.id)}
                        type="button"
                      >
                        <GitBranch size={13} />
                        <span>{prmrState?.status === "running" ? "Planning" : "PR/MR Plan"}</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={executionState?.status === "running"}
                        onClick={() => void runCandidateDeploymentDryRun(candidate.id)}
                        type="button"
                      >
                        <Play size={13} />
                        <span>{executionState?.status === "running" ? "Running" : "Deploy Dry Run"}</span>
                      </button>
                    </div>
                    <ActionFeedback state={applyState} />
                    <ActionFeedback state={providerState} />
                    <ActionFeedback state={deployState} />
                    <ActionFeedback state={publishState} />
                    <ActionFeedback state={prmrState} />
                    <ActionFeedback state={executionState} />
                    {feedback ? (
                      <div className="signalMeta">
                        <code>{feedback.decision}</code>
                        {feedback.environment ? <code>{feedback.environment}</code> : null}
                        {feedback.rollback_required ? <code>rollback suggested</code> : null}
                      </div>
                    ) : null}
                  </div>
                );
              })}
              {snapshot.release_candidate_applies.slice(0, 2).map((apply) => (
                <div className="signalItem" key={apply.id}>
                  <div className="signalHeader">
                    <strong>{compactID(apply.candidate_id || apply.id)}</strong>
                    <StatusPill tone={toneForStatus(apply.status)} label={apply.decision} />
                  </div>
                  <span>
                    {apply.mode} / {apply.write_enabled ? "write enabled" : "guarded"} / {apply.action_count} actions
                  </span>
                  <div className="signalMeta">
                    {apply.release_branch ? <code>{compactID(apply.release_branch)}</code> : null}
                    {apply.source_branch ? <code>{compactID(apply.source_branch)}</code> : null}
                    {apply.reasons[0] ? <code>{apply.reasons[0]}</code> : null}
                  </div>
                </div>
              ))}
              {snapshot.release_candidate_provider_previews.slice(0, 2).map((preview) => (
                <div className="signalItem" key={preview.id}>
                  <div className="signalHeader">
                    <strong>{compactID(preview.candidate_id || preview.id)}</strong>
                    <StatusPill tone={toneForStatus(preview.status)} label={preview.decision} />
                  </div>
                  <span>
                    {preview.provider || "provider"} / {preview.remote_action_count} actions / {preview.pr_mr_type || "pr/mr"}
                  </span>
                  <div className="signalMeta">
                    {preview.pr_mr_decision ? <code>{preview.pr_mr_decision}</code> : null}
                    {preview.pr_mr_head_branch ? <code>{compactID(preview.pr_mr_head_branch)}</code> : null}
                    {preview.reasons[0] ? <code>{preview.reasons[0]}</code> : null}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="observabilityGrid" hidden={!viewVisible(activeView, ["Projects", "Runs", "Providers"])}>
          <div className="panel">
            <PanelTitle icon={<TerminalSquare size={18} />} title="Runtime Recoveries" meta={`${snapshot.runtime_recoveries.length} archived`} />
            <div className="signalList">
              {snapshot.runtime_recoveries.length > 0 ? (
                snapshot.runtime_recoveries.map((recovery) => {
                  const artifactState = recoveryArtifactState[recovery.id];
                  return (
                    <div className="signalItem" key={recovery.id}>
                      <div className="signalHeader">
                        <strong>{compactID(recovery.issue_id || recovery.run_id || recovery.id)}</strong>
                        <StatusPill tone={toneForStatus(recovery.status)} label={recovery.status} />
                      </div>
                      <span>
                        {recovery.failure_category} / {recovery.runtime_id || "runtime pending"}
                      </span>
                      <div className="signalMeta">
                        {recovery.fallback_candidate ? <code>fallback {recovery.fallback_candidate}</code> : null}
                        {recovery.native_session_id ? <code>{compactID(recovery.native_session_id)}</code> : null}
                        {recovery.diff_summary_path ? <code>{shortPath(recovery.diff_summary_path)}</code> : null}
                      </div>
                      <div className="signalActions">
                        <button
                          className="inlineActionButton"
                          disabled={artifactState?.status === "loading"}
                          onClick={() => void loadRecoveryArtifacts(recovery.id)}
                          type="button"
                        >
                          <TerminalSquare size={13} />
                          <span>{artifactState?.status === "loading" ? "Loading" : "Artifacts"}</span>
                        </button>
                        {artifactState?.message ? <small className={`actionMessage ${artifactState.status}`}>{artifactState.message}</small> : null}
                      </div>
                      {artifactState?.artifacts && artifactState.artifacts.length > 0 ? (
                        <div className="artifactPreviewList">
                          {artifactState.artifacts.map((artifact) => (
                            <div className="artifactPreview" key={`${recovery.id}-${artifact.kind}`}>
                              <div className="artifactPreviewHeader">
                                <strong>{artifact.kind}</strong>
                                <code>{shortPath(artifact.path)}</code>
                                <span>{artifact.truncated ? "truncated" : artifact.status}</span>
                              </div>
                              <pre>{artifact.content || artifact.status}</pre>
                            </div>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  );
                })
              ) : (
                <div className="emptyState">No runtime recovery archived</div>
              )}
            </div>
            <div className="controlForm compact">
              <label>
                <span>Reviewer</span>
                <input
                  onChange={(event) => setRepairReviewForm((current) => ({ ...current, reviewerID: event.target.value }))}
                  value={repairReviewForm.reviewerID}
                />
              </label>
              <label>
                <span>Reason</span>
                <input
                  onChange={(event) => setRepairReviewForm((current) => ({ ...current, reason: event.target.value }))}
                  value={repairReviewForm.reason}
                />
              </label>
            </div>
            <SchemaFeedback errors={schemaErrors.repairReview} />
            <div className="signalList">
              {snapshot.operation_repair_candidates.length > 0 ? (
                snapshot.operation_repair_candidates.slice(0, 3).map((candidate) => (
                  <div className="signalItem" key={candidate.id}>
                    <div className="signalHeader">
                      <strong>{compactID(candidate.operation_id || candidate.id)}</strong>
                      <StatusPill tone={toneForStatus(candidate.failure_class)} label={candidate.failure_class} />
                    </div>
                    <span>{`${candidate.decision} / ${candidate.signal_type}`}</span>
                    <div className="signalMeta">
                      {candidate.repair_plan_id ? <code>{compactID(candidate.repair_plan_id)}</code> : null}
                      {candidate.evidence_refs.length > 0 ? <code>{candidate.evidence_refs.length} evidence</code> : null}
                      {candidate.review_required ? <code>review required</code> : null}
                      {candidate.review_decision ? <code>{candidate.review_decision}</code> : null}
                      {candidate.issue_id ? <code>{compactID(candidate.issue_id)}</code> : null}
                      {candidate.repair_attempt_id ? <code>{compactID(candidate.repair_attempt_id)}</code> : null}
                    </div>
                    {candidate.review_reason ? <small>{candidate.review_reason}</small> : null}
                    {candidate.status === "review_required" || candidate.review_required ? (
                      <div className="signalActions">
                        <button
                          className="inlineActionButton"
                          disabled={repairActionState[candidate.id]?.status === "running"}
                          onClick={() => void reviewOperationRepairCandidate(candidate.id, "approved")}
                          type="button"
                        >
                          <CheckCircle2 size={13} />
                          <span>Approve</span>
                        </button>
                        <button
                          className="inlineActionButton danger"
                          disabled={repairActionState[candidate.id]?.status === "running"}
                          onClick={() => void reviewOperationRepairCandidate(candidate.id, "rejected")}
                          type="button"
                        >
                          <AlertTriangle size={13} />
                          <span>Reject</span>
                        </button>
                        {repairActionState[candidate.id]?.message ? <ActionFeedback state={repairActionState[candidate.id]} /> : null}
                      </div>
                    ) : null}
                  </div>
                ))
              ) : (
                <div className="emptyState compact">No operation repair candidates</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<CircleDotDashed size={18} />} title="Subagent Backlog" meta={`${snapshot.subagent_backlog.length} waiting`} />
            <div className="signalList">
              {snapshot.subagent_backlog.length > 0 ? (
                snapshot.subagent_backlog.map((item) => (
                  <div className="signalItem" key={`${item.issue_id}-${item.subagent_id}`}>
                    <div className="signalHeader">
                      <strong>{compactID(item.issue_id)}</strong>
                      <StatusPill tone={toneForStatus(item.status)} label={item.status} />
                    </div>
                    <span>{item.reason || item.failure_category || "waiting for scheduler decision"}</span>
                    <div className="signalMeta">
                      <code>{compactID(item.subagent_id)}</code>
                      <code>
                        retry {item.retry_count}/{item.max_retries}
                      </code>
                      {item.recovery_id ? <code>{compactID(item.recovery_id)}</code> : null}
                    </div>
                  </div>
                ))
              ) : (
                <div className="emptyState">No subagent backlog</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<RefreshCw size={18} />} title="Control Loop Runs" meta={`${snapshot.control_loop_runs.length} runs`} />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={controlLoopActionState.status === "running"} onClick={() => void runControlLoop()} type="button">
                <RefreshCw size={13} />
                <span>{controlLoopActionState.status === "running" ? "Running" : "Run Loop"}</span>
              </button>
              <ActionFeedback state={controlLoopActionState} />
            </div>
            <div className="signalList">
              {snapshot.control_loop_runs.length > 0 ? (
                snapshot.control_loop_runs.map((run) => (
                  <div className="signalItem" key={run.id}>
                    <div className="signalHeader">
                      <strong>{compactID(run.id)}</strong>
                      <StatusPill tone={toneForStatus(run.status)} label={run.decision} />
                    </div>
                    <span>
                      {run.trigger} / {run.steps.length} steps / {shortTimestamp(run.finished_at || run.started_at)}
                    </span>
                    <div className="signalMeta">
                      {run.requested_by ? <code>{run.requested_by}</code> : null}
                      {run.reasons[0] ? <code>{run.reasons[0]}</code> : null}
                    </div>
                    <div className="routeCandidateGrid compact">
                      {run.steps.slice(0, 3).map((step) => (
                        <div className="routeCandidate" key={step.id}>
                          <strong>{step.type}</strong>
                          <span>{step.summary || step.decision}</span>
                          <div className="signalMeta">
                            <code>{step.status}</code>
                            <code>{step.duration_ms}ms</code>
                            {step.evidence_count ? <code>{step.evidence_count} evidence</code> : null}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ))
              ) : (
                <div className="emptyState">No control loop runs</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle
              icon={<Sparkles size={18} />}
              title="Visual Assets"
              meta={`${snapshot.visual_assets.length} plans / ${snapshot.visual_render_executions.length} renders`}
            />
            <div className="signalList">
              {snapshot.visual_assets.length > 0 || snapshot.visual_render_executions.length > 0 ? (
                <>
                  {snapshot.visual_assets.map((asset) => {
                    const actionState = visualActionState[asset.id];
                    return (
                      <div className="signalItem" key={asset.id}>
                        <div className="signalHeader">
                          <strong>{asset.title}</strong>
                          <StatusPill tone={toneForStatus(asset.status)} label={asset.status} />
                        </div>
                        <span>
                          {asset.diagram_type} / {asset.size}
                        </span>
                        <div className="signalMeta">
                          {asset.provider_id ? <code>{asset.provider_id}</code> : null}
                          {asset.model_id ? <code>{asset.model_id}</code> : null}
                          <code>{shortPath(asset.prompt_path || asset.spec_path)}</code>
                        </div>
                        {asset.route_reason ? <small>{asset.route_reason}</small> : null}
                        <div className="signalActions">
                          <button
                            className="inlineActionButton"
                            disabled={actionState?.status === "running"}
                            onClick={() => void runVisualDryRun(asset.id)}
                            type="button"
                          >
                            <Play size={13} />
                            <span>{actionState?.status === "running" ? "Running" : "Dry Run"}</span>
                          </button>
                          {actionState ? (
                            <small className={`actionMessage ${actionState.status}`}>
                              {actionState.executionID ? `${compactID(actionState.executionID)} / ` : ""}
                              {actionState.message}
                            </small>
                          ) : null}
                        </div>
                      </div>
                    );
                  })}
                  {snapshot.visual_render_executions.map((execution) => (
                    <div className="signalItem" key={execution.id}>
                      <div className="signalHeader">
                        <strong>{execution.title || compactID(execution.asset_id || execution.id)}</strong>
                        <StatusPill tone={toneForStatus(execution.status)} label={execution.mode} />
                      </div>
                      <span>
                        {execution.decision} / {execution.step_count} steps
                      </span>
                      <div className="signalMeta">
                        <code>{execution.status}</code>
                        {execution.provider_id ? <code>{execution.provider_id}</code> : null}
                        {execution.script_path ? <code>{shortPath(execution.script_path)}</code> : null}
                        {execution.image_path ? <code>{shortPath(execution.image_path)}</code> : null}
                      </div>
                      {execution.reasons[0] ? <small>{execution.reasons[0]}</small> : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">No visual assets planned</div>
              )}
            </div>
          </div>
        </section>

        <section className="auditGrid" hidden={!viewVisible(activeView, ["Projects", "Audit"])}>
          <div className="panel">
            <PanelTitle icon={<Lock size={18} />} title="Approval Queue" meta={`${snapshot.approvals.length} records`} />
            <div className="controlForm compact">
              <label>
                <span>Decider</span>
                <input
                  onChange={(event) => setApprovalForm((current) => ({ ...current, decidedBy: event.target.value }))}
                  value={approvalForm.decidedBy}
                />
              </label>
              <label>
                <span>Reason</span>
                <input onChange={(event) => setApprovalForm((current) => ({ ...current, reason: event.target.value }))} value={approvalForm.reason} />
              </label>
            </div>
            <SchemaFeedback errors={schemaErrors.approval} />
            <div className="signalList">
              {snapshot.approvals.length > 0 ? (
                snapshot.approvals.map((approval) => (
                  <div className="signalItem" key={approval.id}>
                    <div className="signalHeader">
                      <strong>{approval.action}</strong>
                      <StatusPill tone={toneForStatus(approval.status)} label={approval.status} />
                    </div>
                    <span>
                      {approval.target_type} / {compactID(approval.target_id)}
                    </span>
                    <div className="signalMeta">
                      <code>{approval.risk_level}</code>
                      <code>{approval.decision}</code>
                      <code>{shortTimestamp(approval.requested_at)}</code>
                    </div>
                    <small>{approval.request_reason || approval.decision_reason || `requested by ${approval.requested_by}`}</small>
                    {approval.status === "pending" ? (
                      <div className="signalActions">
                        <button
                          className="inlineActionButton"
                          disabled={approvalActionState[approval.id]?.status === "running"}
                          onClick={() => void decideApproval(approval.id, "approved")}
                          type="button"
                        >
                          <CheckCircle2 size={13} />
                          <span>Approve</span>
                        </button>
                        <button
                          className="inlineActionButton danger"
                          disabled={approvalActionState[approval.id]?.status === "running"}
                          onClick={() => void decideApproval(approval.id, "rejected")}
                          type="button"
                        >
                          <AlertTriangle size={13} />
                          <span>Reject</span>
                        </button>
                        {approvalActionState[approval.id]?.message ? (
                          <small className={`actionMessage ${approvalActionState[approval.id]?.status}`}>{approvalActionState[approval.id]?.message}</small>
                        ) : null}
                      </div>
                    ) : null}
                  </div>
                ))
              ) : (
                <div className="emptyState">No approval records</div>
              )}
            </div>
          </div>
          <div className="panel">
            <PanelTitle icon={<ScrollText size={18} />} title="Audit Trail" meta={`${snapshot.audit_events.length} core events`} />
            <div className="signalList auditList">
              {snapshot.audit_events.length > 0 ? (
                snapshot.audit_events.map((event) => (
                  <div className="signalItem auditItem" key={event.id}>
                    <div className="signalHeader">
                      <strong>{event.event}</strong>
                      <StatusPill tone={toneForStatus(event.status || event.decision || event.channel)} label={event.channel} />
                    </div>
                    <span>
                      {event.decision || event.status || "recorded"} / {shortTimestamp(event.ts)}
                    </span>
                    <div className="signalMeta">
                      {event.issue_id ? <code>issue {compactID(event.issue_id)}</code> : null}
                      {event.run_id ? <code>run {compactID(event.run_id)}</code> : null}
                      {event.subagent_id ? <code>subagent {compactID(event.subagent_id)}</code> : null}
                      {event.trace_id ? <code>trace {compactID(event.trace_id)}</code> : null}
                    </div>
                    {event.reason ? <small>{event.reason}</small> : null}
                  </div>
                ))
              ) : (
                <div className="emptyState">No audit events recorded</div>
              )}
            </div>
          </div>
        </section>

        <section className="accessGrid" hidden={!viewVisible(activeView, ["Projects", "Audit"])}>
          <div className="panel">
            <PanelTitle
              icon={<ShieldCheck size={18} />}
              title="Access Baseline"
              meta={`${activeSessions.length} sessions / ${activeTokens.length} tokens / ${activeServiceAccounts.length} service accounts`}
            />
            <div className="accessForms">
              <form className="controlForm" onSubmit={createSession}>
                <div className="controlFormTitle">
                  <UserPlus size={15} />
                  <strong>Session</strong>
                </div>
                <label>
                  <span>User</span>
                  <input onChange={(event) => setSessionForm((current) => ({ ...current, userID: event.target.value }))} value={sessionForm.userID} />
                </label>
                <label>
                  <span>Display</span>
                  <input
                    onChange={(event) => setSessionForm((current) => ({ ...current, displayName: event.target.value }))}
                    value={sessionForm.displayName}
                  />
                </label>
                <label>
                  <span>Roles</span>
                  <input onChange={(event) => setSessionForm((current) => ({ ...current, roles: event.target.value }))} value={sessionForm.roles} />
                </label>
                <div className="buttonRow">
                  <button className="inlineActionButton" disabled={accessActionState.session?.status === "running"} type="submit">
                    <UserPlus size={13} />
                    <span>Create</span>
                  </button>
                  <ActionFeedback state={accessActionState.session} />
                </div>
                <SchemaFeedback errors={schemaErrors.session} />
              </form>

              <form className="controlForm" onSubmit={createAPIToken}>
                <div className="controlFormTitle">
                  <KeyRound size={15} />
                  <strong>API Token</strong>
                </div>
                <label>
                  <span>Name</span>
                  <input onChange={(event) => setTokenForm((current) => ({ ...current, name: event.target.value }))} value={tokenForm.name} />
                </label>
                <label>
                  <span>Actor</span>
                  <input onChange={(event) => setTokenForm((current) => ({ ...current, actorID: event.target.value }))} value={tokenForm.actorID} />
                </label>
                <label>
                  <span>Scopes</span>
                  <input onChange={(event) => setTokenForm((current) => ({ ...current, scopes: event.target.value }))} value={tokenForm.scopes} />
                </label>
                <div className="buttonRow">
                  <button className="inlineActionButton" disabled={accessActionState.token?.status === "running"} type="submit">
                    <KeyRound size={13} />
                    <span>Create</span>
                  </button>
                  <ActionFeedback state={accessActionState.token} />
                </div>
                <SchemaFeedback errors={schemaErrors.token} />
              </form>

              <form className="controlForm" onSubmit={createServiceAccount}>
                <div className="controlFormTitle">
                  <ShieldCheck size={15} />
                  <strong>Service Account</strong>
                </div>
                <label>
                  <span>ID</span>
                  <input onChange={(event) => setServiceAccountForm((current) => ({ ...current, id: event.target.value }))} value={serviceAccountForm.id} />
                </label>
                <label>
                  <span>Name</span>
                  <input
                    onChange={(event) => setServiceAccountForm((current) => ({ ...current, name: event.target.value }))}
                    value={serviceAccountForm.name}
                  />
                </label>
                <label>
                  <span>Roles</span>
                  <input
                    onChange={(event) => setServiceAccountForm((current) => ({ ...current, roles: event.target.value }))}
                    value={serviceAccountForm.roles}
                  />
                </label>
                <div className="buttonRow">
                  <button className="inlineActionButton" disabled={accessActionState.service?.status === "running"} type="submit">
                    <ShieldCheck size={13} />
                    <span>Upsert</span>
                  </button>
                  <ActionFeedback state={accessActionState.service} />
                </div>
                <SchemaFeedback errors={schemaErrors.service} />
              </form>
            </div>
            <div className="accessList">
              <div className="accessCard">
                <div className="accessCardHeader">
                  <strong>Sessions</strong>
                  <StatusPill tone={activeSessions.length > 0 ? "ok" : "neutral"} label={`${activeSessions.length} active`} />
                </div>
                {snapshot.auth_sessions.slice(0, 3).map((session) => (
                  <div className="accessRow" key={session.id}>
                    <div>
                      <strong>{session.display_name || session.user_id}</strong>
                      <span>{session.roles.join(", ") || "role pending"}</span>
                    </div>
                    <code>{shortTimestamp(session.created_at)}</code>
                  </div>
                ))}
                {snapshot.auth_sessions.length === 0 ? <div className="emptyState">No sessions</div> : null}
              </div>

              <div className="accessCard">
                <div className="accessCardHeader">
                  <strong>API Tokens</strong>
                  <StatusPill tone={activeTokens.length > 0 ? "ok" : "neutral"} label={`${activeTokens.length} active`} />
                </div>
                {snapshot.api_tokens.slice(0, 3).map((token) => (
                  <div className="accessRow" key={token.id}>
                    <div>
                      <strong>{token.name}</strong>
                      <span>{token.scopes.join(", ") || "scope pending"}</span>
                    </div>
                    <code>{token.token_prefix || compactID(token.id)}</code>
                    {token.status === "active" ? (
                      <button
                        className="inlineActionButton danger compactButton"
                        disabled={accessActionState[token.id]?.status === "running"}
                        onClick={() => void revokeAPIToken(token.id)}
                        type="button"
                      >
                        <span>Revoke</span>
                      </button>
                    ) : null}
                  </div>
                ))}
                {snapshot.api_tokens.map((token) =>
                  accessActionState[token.id]?.message ? <ActionFeedback key={`${token.id}-feedback`} state={accessActionState[token.id]} /> : null,
                )}
                {snapshot.api_tokens.length === 0 ? <div className="emptyState">No API tokens</div> : null}
              </div>

              <div className="accessCard">
                <div className="accessCardHeader">
                  <strong>Service Accounts</strong>
                  <StatusPill tone={activeServiceAccounts.length > 0 ? "ok" : "neutral"} label={`${activeServiceAccounts.length} active`} />
                </div>
                {snapshot.service_accounts.slice(0, 3).map((account) => (
                  <div className="accessRow" key={account.id}>
                    <div>
                      <strong>{account.name}</strong>
                      <span>{account.roles.join(", ") || "role pending"}</span>
                    </div>
                    <code>{compactID(account.id)}</code>
                  </div>
                ))}
                {snapshot.service_accounts.length === 0 ? <div className="emptyState">No service accounts</div> : null}
              </div>
            </div>
          </div>
        </section>

        <section className="bottomGrid" hidden={!viewVisible(activeView, ["Projects", "Providers", "Deployments"])}>
          <div className="panel">
            <PanelTitle icon={<Sparkles size={18} />} title="Providers & Runtimes" meta={`${snapshot.providers.length} registered`} />
            <div className="providerMatrix">
              {snapshot.providers.map((provider) => (
                <div className="providerRow" key={provider.id}>
                  <StatusDot tone={provider.enabled ? "ok" : "neutral"} />
                  <div>
                    <strong>{provider.name}</strong>
                    <span>
                      {provider.vendor} / {provider.runtime_id || provider.api_type}
                    </span>
                  </div>
                  <code>{provider.model || provider.id}</code>
                  <StatusPill tone={toneForStatus(provider.health_status || (provider.enabled ? "ok" : "unknown"))} label={provider.health_status || "unknown"} />
                </div>
              ))}
            </div>
            <div className="routePreviewBox">
              <div className="controlForm compact">
                <label>
                  <span>Role</span>
                  <select onChange={(event) => setProviderRouteForm((current) => ({ ...current, role: event.target.value }))} value={providerRouteForm.role}>
                    <option value="frontend">frontend</option>
                    <option value="backend">backend</option>
                    <option value="devops">devops</option>
                    <option value="review">review</option>
                  </select>
                </label>
                <label>
                  <span>Task</span>
                  <select
                    onChange={(event) => setProviderRouteForm((current) => ({ ...current, taskType: event.target.value }))}
                    value={providerRouteForm.taskType}
                  >
                    <option value="requirement_planning">requirement_planning</option>
                    <option value="architecture_planning">architecture_planning</option>
                    <option value="memory_extraction">memory_extraction</option>
                    <option value="image_generation">image_generation</option>
                  </select>
                </label>
                <label>
                  <span>Output</span>
                  <select
                    onChange={(event) => setProviderRouteForm((current) => ({ ...current, outputType: event.target.value }))}
                    value={providerRouteForm.outputType}
                  >
                    <option value="code">code</option>
                    <option value="markdown">markdown</option>
                    <option value="architecture_diagram">architecture_diagram</option>
                    <option value="image">image</option>
                  </select>
                </label>
                <label className="checkboxLine">
                  <input
                    checked={providerRouteForm.requiresRepoEdit}
                    onChange={(event) => setProviderRouteForm((current) => ({ ...current, requiresRepoEdit: event.target.checked }))}
                    type="checkbox"
                  />
                  <span>Repo edit</span>
                </label>
                <label className="checkboxLine">
                  <input
                    checked={providerRouteForm.includesProjectMemory}
                    onChange={(event) => setProviderRouteForm((current) => ({ ...current, includesProjectMemory: event.target.checked }))}
                    type="checkbox"
                  />
                  <span>Memory</span>
                </label>
              </div>
              <SchemaFeedback errors={schemaErrors.providerRoute} />
              <div className="rowActions panelActionRow">
                <button className="inlineActionButton" disabled={providerRouteState.status === "running"} onClick={() => void previewProviderRoute()} type="button">
                  <Search size={13} />
                  <span>{providerRouteState.status === "running" ? "Routing" : "Route Preview"}</span>
                </button>
                <ActionFeedback state={providerRouteState} />
              </div>
              {providerRoute ? (
                <>
                  <div className="routeSummary">
                    <strong>{providerRoute.provider_id || "no provider selected"}</strong>
                    <span>{providerRoute.explanation?.summary || providerRoute.reason || providerRoute.decision}</span>
                  </div>
                  <div className="routeCandidateGrid">
                    {(providerRoute.candidates ?? []).slice(0, 6).map((candidate, index) => (
                      <div className="routeCandidate" key={candidate.provider_id || `route-candidate-${index}`}>
                        <div className="signalHeader">
                          <strong>{candidate.provider_id}</strong>
                          <StatusPill tone={toneForStatus(candidate.status ?? "")} label={candidate.status ?? "candidate"} />
                        </div>
                        <span>{candidate.reason}</span>
                        <div className="signalMeta">
                          {candidate.runtime_id ? <code>{candidate.runtime_id}</code> : null}
                          {candidate.model_id ? <code>{candidate.model_id}</code> : null}
                          <code>score {candidate.score ?? 0}</code>
                        </div>
                      </div>
                    ))}
                  </div>
                </>
              ) : null}
            </div>
            <div className="telemetryList">
              {snapshot.provider_telemetry.length > 0 ? (
                snapshot.provider_telemetry.slice(0, 4).map((record) => (
                  <div className="telemetryItem" key={record.id}>
                    <div>
                      <strong>{record.provider_id}</strong>
                      <span>
                        {record.source}
                        {record.total_tokens ? ` / ${record.total_tokens} tokens` : ""}
                      </span>
                    </div>
                    <StatusPill tone={toneForStatus(record.quality_status || record.health_status || record.decision)} label={record.quality_status || record.health_status || record.decision} />
                    <code>{record.cost_status || record.quota_status || record.runtime_status || "ops"}</code>
                  </div>
                ))
              ) : (
                <div className="emptyState">No provider telemetry</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<Server size={18} />} title="Server Resources" meta={`${snapshot.lifecycle_alerts.length} alerts / ${snapshot.maintenance_records.length} maintenance`} />
            <div className="controlForm compact">
              <label>
                <span>Actor</span>
                <input onChange={(event) => setResourceForm((current) => ({ ...current, actorID: event.target.value }))} value={resourceForm.actorID} />
              </label>
              <label>
                <span>Expires</span>
                <input
                  onChange={(event) => setResourceForm((current) => ({ ...current, expiresAt: event.target.value }))}
                  type="date"
                  value={resourceForm.expiresAt}
                />
              </label>
              <label>
                <span>Reason</span>
                <input onChange={(event) => setResourceForm((current) => ({ ...current, reason: event.target.value }))} value={resourceForm.reason} />
              </label>
            </div>
            <SchemaFeedback errors={schemaErrors.resource} />
            <div className="resourceList">
              {snapshot.resources.length > 0 ? (
                snapshot.resources.map((resource) => (
                  <div className="resourceItem" key={resource.id}>
                    <div>
                      <strong>{resource.id}</strong>
                      <span>
                        {resource.host}
                        {resource.health ? ` / ${resource.health}` : ""}
                        {resource.expiration_state ? ` / ${resource.expiration_state}` : ""}
                      </span>
                      {resource.last_deployment ? (
                        <span>{`last ${resource.last_deployment.kind} / ${compactID(resource.last_deployment.execution_id || resource.last_deployment.deployment_id || resource.last_deployment.id)}`}</span>
                      ) : null}
                    </div>
                    <StatusPill tone={toneForStatus(resource.expiration_state || (resource.environment === "production" ? "warning" : "ok"))} label={resource.environment} />
                    <div className="rowActions">
                      <button
                        className="inlineActionButton"
                        disabled={resourceActionState[resource.id]?.status === "running"}
                        onClick={() => void runResourceAction(resource.id, "renew")}
                        type="button"
                      >
                        <RefreshCw size={13} />
                        <span>Renew</span>
                      </button>
                      <button
                        className="inlineActionButton danger"
                        disabled={resourceActionState[resource.id]?.status === "running"}
                        onClick={() => void runResourceAction(resource.id, "retire")}
                        type="button"
                      >
                        <Wrench size={13} />
                        <span>Retire</span>
                      </button>
                    </div>
                    {resourceActionState[resource.id]?.message ? <ActionFeedback state={resourceActionState[resource.id]} /> : null}
                  </div>
                ))
              ) : (
                <div className="emptyState">No resources registered</div>
              )}
            </div>
            <div className="maintenanceList">
              {latestResourceDeploymentRef ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{latestResourceDeploymentRef.resource_id}</strong>
                    <span>{`${latestResourceDeploymentRef.kind} / ${latestResourceDeploymentRef.decision}`}</span>
                    <span>{compactID(latestResourceDeploymentRef.execution_id || latestResourceDeploymentRef.deployment_id || latestResourceDeploymentRef.id)}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestResourceDeploymentRef.status)} label={latestResourceDeploymentRef.environment || latestResourceDeploymentRef.status} />
                </div>
              ) : null}
              {snapshot.resource_deployment_refs.slice(1, 3).map((ref) => (
                <div className="maintenanceItem" key={ref.id}>
                  <div>
                    <strong>{ref.resource_id}</strong>
                    <span>{`${ref.kind} / ${ref.decision}`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(ref.status)} label={ref.mode || ref.status} />
                </div>
              ))}
            </div>
            <div className="maintenanceList">
              {snapshot.lifecycle_alerts.length > 0 ? (
                snapshot.lifecycle_alerts.slice(0, 3).map((alert) => (
                  <div className="maintenanceItem" key={alert.id}>
                    <div>
                      <strong>{alert.resource_id || compactID(alert.id)}</strong>
                      <span>{alert.reason || alert.type}</span>
                    </div>
                    <StatusPill tone={toneForStatus(alert.severity || alert.status)} label={alert.expiration_state || alert.health_status || alert.type} />
                  </div>
                ))
              ) : (
                <div className="emptyState compact">No lifecycle alerts</div>
              )}
            </div>
            <div className="maintenanceList">
              {snapshot.maintenance_records.length > 0 ? (
                snapshot.maintenance_records.slice(0, 3).map((record) => (
                  <div className="maintenanceItem" key={record.id}>
                    <div>
                      <strong>{record.resource_id || compactID(record.id)}</strong>
                      <span>{record.reason || record.type}</span>
                    </div>
                    <StatusPill tone={toneForStatus(record.status)} label={record.expiration_state || record.health_status || record.status} />
                  </div>
                ))
              ) : (
                <div className="emptyState">No maintenance records</div>
              )}
            </div>
          </div>

          <div className="panel releasePanel">
            <PanelTitle icon={<GitBranch size={18} />} title="Release Pipeline" meta={`${snapshot.git_provider_plans.length} PR/MR plans`} />
            <div className="releaseSteps">
              <span>accepted issues</span>
              <ChevronRight size={15} />
              <span>release branch</span>
              <ChevronRight size={15} />
              <span>tag + PR/MR</span>
              <ChevronRight size={15} />
              <span>deploy plan</span>
            </div>
            <div className="releaseProviderBox">
              <div className="controlForm compact">
                <label>
                  <span>Release ID</span>
                  <input
                    onChange={(event) => setReleaseProviderForm((current) => ({ ...current, releaseID: event.target.value }))}
                    value={releaseProviderForm.releaseID}
                  />
                </label>
                <label className="checkboxLine">
                  <input
                    checked={releaseProviderForm.approved}
                    onChange={(event) => setReleaseProviderForm((current) => ({ ...current, approved: event.target.checked }))}
                    type="checkbox"
                  />
                  <span>Approved publish</span>
                </label>
                <label>
                  <span>Approval ID</span>
                  <input
                    onChange={(event) => setReleaseProviderForm((current) => ({ ...current, approvalID: event.target.value }))}
                    value={releaseProviderForm.approvalID}
                  />
                </label>
              </div>
              <div className="rowActions wide">
                <button
                  className="inlineActionButton"
                  disabled={releaseProviderActionState.status === "running"}
                  onClick={() => void runReleaseProviderAction("preview")}
                  type="button"
                >
                  <Search size={13} />
                  <span>Provider Preview</span>
                </button>
                <button
                  className="inlineActionButton"
                  disabled={releaseProviderActionState.status === "running"}
                  onClick={() => void runReleaseProviderAction("publish")}
                  type="button"
                >
                  <Rocket size={13} />
                  <span>Provider Publish</span>
                </button>
                <ActionFeedback state={releaseProviderActionState} />
              </div>
              <SchemaFeedback errors={schemaErrors.releaseProvider} />
            </div>
            <div className="releaseControls">
              <label className="checkboxLine">
                <input checked={gitCreateApproved} onChange={(event) => setGitCreateApproved(event.target.checked)} type="checkbox" />
                <span>Approved create</span>
              </label>
              <label className="approvalIDInput">
                <span>Approval ID</span>
                <input onChange={(event) => setGitCreateApprovalID(event.target.value)} value={gitCreateApprovalID} />
              </label>
            </div>
            <SchemaFeedback errors={schemaErrors.gitCreate} />
            <div className="prmrList">
              {snapshot.git_provider_plans.length > 0 ? (
                snapshot.git_provider_plans.slice(0, 3).map((plan) => (
                  <div className="prmrItem" key={plan.id}>
                    <div>
                      <strong>{plan.issue_id || compactID(plan.id)}</strong>
                      <span>
                        {plan.provider} / {plan.target_branch || "branch pending"}
                      </span>
                    </div>
                    <StatusPill tone={toneForStatus(plan.status)} label={plan.remote_status || plan.status} />
                    <code>{plan.create_decision || plan.preview_decision || plan.sync_decision || plan.decision}</code>
                    <div className="rowActions wide">
                      <button
                        className="inlineActionButton"
                        disabled={gitActionState[plan.id]?.status === "running"}
                        onClick={() => void runGitProviderAction(plan.id, "preview")}
                        type="button"
                      >
                        <Search size={13} />
                        <span>Preview</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={gitActionState[plan.id]?.status === "running"}
                        onClick={() => void runGitProviderAction(plan.id, "sync")}
                        type="button"
                      >
                        <RefreshCw size={13} />
                        <span>Sync</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={gitActionState[plan.id]?.status === "running"}
                        onClick={() => void runGitProviderAction(plan.id, "create")}
                        type="button"
                      >
                        <GitBranch size={13} />
                        <span>Create</span>
                      </button>
                      {gitActionState[plan.id]?.message ? <ActionFeedback state={gitActionState[plan.id]} /> : null}
                    </div>
                  </div>
                ))
              ) : (
                <div className="emptyState">No PR/MR plans recorded</div>
              )}
            </div>
            <div className="approvalStrip">
              <Lock size={16} />
              <span>production deploy requires approval, smoke, monitor and rollback plan</span>
            </div>
          </div>
        </section>
      </section>
    </main>
  );
}

type RequirementSubmitState = {
  status: "idle" | "planning" | "planned" | "needs_user_input" | "error";
  id?: string;
  epic?: string;
  message?: string;
};

type RequirementPlanEnvelope = {
  error?: string;
  requirement?: {
    id?: string;
    epic_id?: string;
    issues?: unknown[];
    clarification_decision?: {
      required?: boolean;
      questions?: string[];
    };
  };
};

type RecoveryArtifactPreview = {
  kind: string;
  path: string;
  status: string;
  content?: string;
  truncated?: boolean;
};

type RecoveryArtifactState = {
  status: "loading" | "loaded" | "error";
  artifacts?: RecoveryArtifactPreview[];
  message?: string;
};

type RecoveryArtifactsEnvelope = {
  error?: string;
  runtime_recovery_artifacts?: {
    artifacts?: RecoveryArtifactPreview[];
  };
};

type VisualActionState = {
  status: "running" | "completed" | "blocked" | "error";
  executionID?: string;
  message?: string;
};

type VisualRenderEnvelope = {
  error?: string;
  visual_render_execution?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type DeploymentActionState = {
  status: "idle" | "running" | "completed" | "blocked" | "error";
  id?: string;
  message?: string;
};

type ReleaseSuggestEnvelope = {
  error?: string;
  release?: {
    id?: string;
    status?: string;
    decision?: string;
    reasons?: string[];
  };
};

type ReleaseProviderExecutionEnvelope = {
  error?: string;
  release_provider_execution?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type DeploymentExecutionEnvelope = {
  error?: string;
  execution?: {
    id?: string;
    status?: string;
    decision?: string;
    reasons?: string[];
  };
};

type RollbackExecutionEnvelope = {
  error?: string;
  rollback_execution?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type MonitorSummaryEnvelope = {
  error?: string;
  monitor_summary?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type PostDeploymentVerificationEnvelope = {
  error?: string;
  post_deployment_verification?: {
    id?: string;
    status?: string;
    decision?: string;
    risk_handoff_recommended?: boolean;
  };
};

type DeploymentRehearsalEnvelope = {
  error?: string;
  deployment_rehearsal?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type RehearsalSchedulerEnvelope = {
  error?: string;
  rehearsal_scheduler_run?: {
    id?: string;
    status?: string;
    decision?: string;
    blocked_count?: number;
  };
};

type ReleaseAdmissionEnvelope = {
  error?: string;
  release_admission?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type DeploymentRiskHandoffEnvelope = {
  error?: string;
  deployment_risk_handoff?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ResourceHealthScanEnvelope = {
  error?: string;
  health_scan?: {
    id?: string;
    status?: string;
    decision?: string;
    results?: unknown[];
  };
};

type ActionStatus = "idle" | "running" | "completed" | "blocked" | "error";

type ActionState = {
  status: ActionStatus;
  id?: string;
  message?: string;
  secretPreview?: string;
};

type ApprovalDecisionEnvelope = {
  error?: string;
  approval?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type AuthSessionEnvelope = {
  error?: string;
  session?: {
    id?: string;
    status?: string;
  };
};

type APITokenCreateEnvelope = {
  error?: string;
  api_token?: {
    id?: string;
    status?: string;
  };
  token_value?: string;
};

type APITokenRevokeEnvelope = {
  error?: string;
  api_token?: {
    id?: string;
    status?: string;
  };
};

type ServiceAccountEnvelope = {
  error?: string;
  service_account?: {
    id?: string;
    status?: string;
  };
};

type GitProviderActionEnvelope = {
  error?: string;
  git_provider_plan?: {
    id?: string;
    status?: string;
    decision?: string;
    pr_mr?: {
      remote_status?: string;
      approval_id?: string;
      preview_decision?: string;
      create_decision?: string;
      sync_decision?: string;
    };
  };
};

type ResourceActionEnvelope = {
  error?: string;
  maintenance_record?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ProviderRouteCandidate = {
  provider_id?: string;
  runtime_id?: string;
  vendor?: string;
  api_type?: string;
  model_id?: string;
  status?: string;
  reason?: string;
  score?: number;
  signals?: Array<{ type?: string; status?: string; reason?: string }>;
};

type ProviderRouteDecision = {
  decision?: string;
  blocked?: boolean;
  strategy?: string;
  provider_id?: string;
  runtime_id?: string;
  model_id?: string;
  reason?: string;
  explanation?: {
    summary?: string;
    selected_provider_id?: string;
    selected_reason?: string;
    candidate_count?: number;
    selected_count?: number;
    skipped_count?: number;
    blocked_count?: number;
  };
  candidates?: ProviderRouteCandidate[];
};

type ProviderRouteEnvelope = {
  error?: string;
  route?: ProviderRouteDecision;
};

type ControlLoopRunEnvelope = {
  error?: string;
  control_loop_run?: {
    id?: string;
    status?: string;
    decision?: string;
    steps?: unknown[];
  };
};

type BatchRunEnvelope = {
  error?: string;
  batch_run?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type MergeQueueEnvelope = {
  error?: string;
  merge_queue?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type IntegrationPreviewEnvelope = {
  error?: string;
  integration_preview?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type IntegrationApplyEnvelope = {
  error?: string;
  integration_apply?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ReleaseBatchEnvelope = {
  error?: string;
  release_batch?: {
    id?: string;
    status?: string;
    decision?: string;
    ready_item_count?: number;
  };
};

type ReleaseCandidateEnvelope = {
  error?: string;
  release_candidate?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ReleaseCandidateApplyEnvelope = {
  error?: string;
  release_candidate_apply?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ReleaseCandidateProviderPreviewEnvelope = {
  error?: string;
  release_candidate_provider_preview?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type DeploymentPlanEnvelope = {
  error?: string;
  deployment?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type OperationRepairReviewEnvelope = {
  error?: string;
  operation_repair_review?: {
    id?: string;
    decision?: string;
    status?: string;
  };
  operation_repair_candidate?: {
    id?: string;
    status?: string;
    decision?: string;
    issue_id?: string;
  };
  repair_attempt?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

async function postJSON<T extends { error?: string }>(path: string, body: unknown): Promise<T> {
  const response = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const payload = (await response.json().catch(() => ({}))) as T;
  if (!response.ok) {
    throw new Error(payload.error ?? `Request failed with status ${response.status}`);
  }
  return payload;
}

function splitCSV(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function viewVisible(activeView: ConsoleView, views: ConsoleView[]) {
  return views.includes(activeView);
}

function isGitActionCompleted(plan: NonNullable<GitProviderActionEnvelope["git_provider_plan"]>) {
  const decision = plan.pr_mr?.create_decision || plan.pr_mr?.preview_decision || plan.pr_mr?.sync_decision || plan.decision || "";
  const remoteStatus = plan.pr_mr?.remote_status || "";
  if (decision.includes("FAILED") || decision.includes("REQUIRED") || remoteStatus.includes("required") || remoteStatus.includes("missing")) {
    return false;
  }
  return Boolean(decision || remoteStatus || plan.status);
}

function ActionFeedback({ state }: { state?: ActionState }) {
  if (!state?.message) return null;
  return (
    <small className={`actionMessage ${state.status}`}>
      {state.id ? `${compactID(state.id)} / ` : ""}
      {state.message}
      {state.secretPreview ? ` / ${state.secretPreview}` : ""}
    </small>
  );
}

function SchemaFeedback({ errors }: { errors?: string[] }) {
  if (!errors || errors.length === 0) return null;
  return (
    <div className="schemaFeedback">
      <AlertTriangle size={14} />
      <div>
        {errors.map((error) => (
          <span key={error}>{error}</span>
        ))}
      </div>
    </div>
  );
}

function PanelTitle({ icon, title, meta }: { icon: React.ReactNode; title: string; meta: string }) {
  return (
    <div className="panelTitle">
      <div>
        {icon}
        <strong>{title}</strong>
      </div>
      <span>{meta}</span>
    </div>
  );
}

function MetricCard({
  label,
  value,
  tone,
  detail,
}: {
  label: string;
  value: number;
  tone: StatusTone;
  detail: string;
}) {
  return (
    <div className={`metricCard ${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{detail}</small>
    </div>
  );
}

function StatusPill({ tone, label }: { tone: StatusTone; label: string }) {
  return <span className={`statusPill ${tone}`}>{label}</span>;
}

function StatusDot({ tone }: { tone: StatusTone }) {
  return <span className={`statusDot ${tone}`} />;
}

function groupIssues(issues: IssueNode[]) {
  return issues.reduce<Record<IssueNode["lane"], IssueNode[]>>(
    (acc, issue) => {
      acc[issue.lane].push(issue);
      return acc;
    },
    { plan: [], backend: [], frontend: [], quality: [], release: [] },
  );
}

function toneForStatus(status: string): StatusTone {
  if (
    status === "accepted" ||
    status === "approved" ||
    status === "selected" ||
    status === "passed" ||
    status === "ready" ||
    status === "ready_to_merge" ||
    status === "completed" ||
    status === "planned" ||
    status === "applied" ||
    status === "allowed" ||
    status === "ok" ||
    status === "healthy"
  )
    return "ok";
  if (status === "running" || status === "dispatch" || status === "retrying") return "running";
  if (
    status === "blocked" ||
    status === "rejected" ||
    status === "failed" ||
    status === "route_blocked" ||
    status === "unhealthy" ||
    status === "down" ||
    status === "expired" ||
    status === "critical" ||
    status === "smoke_failed" ||
    status === "monitor_failed" ||
    status === "execution_failed" ||
    status === "execution_blocked" ||
    status === "operation_failed" ||
    status === "operation_blocked" ||
    status === "check_failed" ||
    status === "conflict"
  )
    return "blocked";
  if (
    status === "needs_rework" ||
    status === "dry_run" ||
    status === "waiting" ||
    status === "pending" ||
    status === "archived" ||
    status === "open" ||
    status === "warning" ||
    status === "degraded" ||
    status === "attention_required" ||
    status === "manual_required" ||
    status === "manual_check_required" ||
    status === "review_required" ||
    status === "suggested" ||
    status === "not_ready"
  )
    return "warning";
  return "neutral";
}

function statusClass(status: string) {
  return toneForStatus(status);
}

function compactID(value: string) {
  if (!value) return "unknown";
  if (value.length <= 28) return value;
  return `${value.slice(0, 18)}...${value.slice(-7)}`;
}

function shortPath(value?: string) {
  if (!value) return "path pending";
  const parts = value.split("/").filter(Boolean);
  return parts.slice(-3).join("/");
}

function shortTimestamp(value?: string) {
  if (!value) return "time pending";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "time pending";
  return new Intl.DateTimeFormat("en", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}
