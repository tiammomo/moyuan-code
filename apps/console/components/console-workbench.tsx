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
import type { ConsoleSnapshot, IssueNode, StatusTone } from "@/lib/types";

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
  { label: "Runs", icon: TerminalSquare },
  { label: "Quality", icon: ShieldCheck },
  { label: "Memory", icon: Brain },
  { label: "Providers", icon: Sparkles },
  { label: "Deployments", icon: Rocket },
  { label: "Audit", icon: Lock },
];

export function ConsoleWorkbench({ snapshot }: { snapshot: ConsoleSnapshot }) {
  const router = useRouter();
  const [selectedIssueID, setSelectedIssueID] = useState(snapshot.issues[0]?.id ?? "");
  const [requirementText, setRequirementText] = useState("");
  const [requirementState, setRequirementState] = useState<RequirementSubmitState>({ status: "idle" });
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
  const selectedIssue = snapshot.issues.find((issue) => issue.id === selectedIssueID) ?? snapshot.issues[0];
  const groupedIssues = useMemo(() => groupIssues(snapshot.issues), [snapshot.issues]);
  const latestDeployment = snapshot.deployments[0];
  const activeSessions = snapshot.auth_sessions.filter((session) => session.status === "active");
  const activeTokens = snapshot.api_tokens.filter((token) => token.status === "active");
  const activeServiceAccounts = snapshot.service_accounts.filter((account) => account.status === "active");

  async function submitRequirement(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const text = requirementText.trim();
    if (!text) {
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
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Release suggestion failed." });
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

  async function decideApproval(approvalID: string, decision: "approved" | "rejected") {
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

  async function runResourceAction(resourceID: string, action: "renew" | "retire") {
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
            <button className="navItem" key={item.label} type="button">
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

        <section className="opsGrid">
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
                      <span>{execution.decision}</span>
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
          </div>
        </section>

        <section className="mainGrid">
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

        <section className="lowerGrid">
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

        <section className="observabilityGrid">
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

        <section className="auditGrid">
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

        <section className="accessGrid">
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

        <section className="bottomGrid">
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
                </div>
              ))}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<Server size={18} />} title="Server Resources" meta={`${snapshot.maintenance_records.length} maintenance`} />
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
            <div className="resourceList">
              {snapshot.resources.length > 0 ? (
                snapshot.resources.map((resource) => (
                  <div className="resourceItem" key={resource.id}>
                    <div>
                      <strong>{resource.id}</strong>
                      <span>
                        {resource.host}
                        {resource.health ? ` / ${resource.health}` : ""}
                      </span>
                    </div>
                    <StatusPill tone={resource.environment === "production" ? "warning" : "ok"} label={resource.environment} />
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

type DeploymentExecutionEnvelope = {
  error?: string;
  execution?: {
    id?: string;
    status?: string;
    decision?: string;
    reasons?: string[];
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
  if (status === "accepted" || status === "passed" || status === "ready" || status === "completed" || status === "planned") return "ok";
  if (status === "running" || status === "dispatch" || status === "retrying") return "running";
  if (status === "blocked" || status === "rejected" || status === "failed" || status === "route_blocked") return "blocked";
  if (status === "waiting" || status === "pending" || status === "archived" || status === "open") return "warning";
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
