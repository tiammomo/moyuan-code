import { connection } from "next/server";
import { demoSnapshot } from "./demo-data";
import type {
  ConsoleSnapshot,
  AuditEventSummary,
  ApprovalRecordSummary,
  DeploymentExecutionSummary,
  DeploymentSummary,
  IssueNode,
  ProjectSummary,
  ProviderSummary,
  QualityExplanation,
  QualitySignal,
  RuntimeRecoverySummary,
  RunSummary,
  ResourceSummary,
  ScheduleItem,
  SubagentBacklogItem,
  SubagentSummary,
  VisualAssetSummary,
  VisualRenderExecutionSummary,
} from "./types";

const apiBase = process.env.MOYUAN_API_BASE_URL ?? "http://127.0.0.1:8080/v1";

type ApiEnvelope<T> = T & { error?: string };

async function apiGet<T>(path: string): Promise<T | null> {
  try {
    const response = await fetch(`${apiBase}${path}`, {
      next: { revalidate: 3 },
    });

    if (!response.ok) {
      return null;
    }

    return (await response.json()) as T;
  } catch {
    return null;
  }
}

export async function getConsoleSnapshot(): Promise<ConsoleSnapshot> {
  await connection();

  const projectsResponse = await apiGet<ApiEnvelope<{ projects: ProjectSummary[] }>>("/projects");
  const projects = projectsResponse?.projects ?? [];
  const project = projects[0];

  if (!project) {
    return {
      ...demoSnapshot,
      generatedAt: new Date().toISOString(),
    };
  }

  const [
    graphResponse,
    scheduleResponse,
    providersResponse,
    resourcesResponse,
    deploymentsResponse,
    executionsResponse,
    runsResponse,
    subagentsResponse,
    recoveriesResponse,
    visualAssetsResponse,
    visualRenderExecutionsResponse,
    qualityReportsResponse,
    memoryResponse,
    auditEventsResponse,
    approvalsResponse,
  ] = await Promise.all([
    apiGet<ApiEnvelope<{ issue_graph: { issues?: unknown[] } }>>(`/projects/${project.id}/epics/phase1-epic/issue-graph`),
    apiGet<ApiEnvelope<{ schedule: { dispatch_queue?: unknown[]; waiting_queue?: unknown[]; subagent_backlog?: unknown[] } }>>(
      `/projects/${project.id}/epics/phase1-epic/schedule?limit=4`,
    ),
    apiGet<ApiEnvelope<{ providers: unknown[] }>>(`/projects/${project.id}/providers`),
    apiGet<ApiEnvelope<{ resources: unknown[] }>>(`/projects/${project.id}/resources`),
    apiGet<ApiEnvelope<{ deployments: unknown[] }>>(`/projects/${project.id}/deployments?limit=4`),
    apiGet<ApiEnvelope<{ executions: unknown[] }>>(`/projects/${project.id}/deployment-executions?limit=4`),
    apiGet<ApiEnvelope<{ runs: unknown[] }>>(`/projects/${project.id}/runs?limit=12`),
    apiGet<ApiEnvelope<{ subagents: unknown[] }>>(`/projects/${project.id}/subagents?limit=12`),
    apiGet<ApiEnvelope<{ runtime_recoveries: unknown[] }>>(`/projects/${project.id}/runtime-recoveries?limit=6`),
    apiGet<ApiEnvelope<{ visual_assets: unknown[] }>>(`/projects/${project.id}/visuals/assets?limit=6`),
    apiGet<ApiEnvelope<{ visual_render_executions: unknown[] }>>(`/projects/${project.id}/visuals/render-executions?limit=6`),
    apiGet<ApiEnvelope<{ quality_reports: unknown[] }>>(`/projects/${project.id}/quality-reports?limit=8`),
    apiGet<ApiEnvelope<{ candidates: unknown[] }>>(`/projects/${project.id}/memory/candidates?limit=3`),
    apiGet<ApiEnvelope<{ audit_events: unknown[] }>>(`/projects/${project.id}/audit-events?channel=all&limit=10`),
    apiGet<ApiEnvelope<{ approvals: unknown[] }>>(`/projects/${project.id}/approvals?limit=6`),
  ]);

  const schedule = [
    ...normalizeScheduleItems(scheduleResponse?.schedule.dispatch_queue ?? [], "dispatch"),
    ...normalizeScheduleItems(scheduleResponse?.schedule.waiting_queue ?? [], "waiting"),
  ];
  const subagentBacklog = normalizeSubagentBacklog(scheduleResponse?.schedule.subagent_backlog ?? []);
  const providers = normalizeProviders(providersResponse?.providers ?? []);
  const resources = normalizeResources(resourcesResponse?.resources ?? []);
  const deployments = normalizeDeployments(deploymentsResponse?.deployments ?? []);
  const executions = normalizeExecutions(executionsResponse?.executions ?? []);
  const runs = normalizeRuns(runsResponse?.runs ?? []);
  const subagents = normalizeSubagents(subagentsResponse?.subagents ?? []);
  const recoveries = normalizeRuntimeRecoveries(recoveriesResponse?.runtime_recoveries ?? []);
  const visualAssets = normalizeVisualAssets(visualAssetsResponse?.visual_assets ?? []);
  const visualRenderExecutions = normalizeVisualRenderExecutions(visualRenderExecutionsResponse?.visual_render_executions ?? []);
  const qualityReports = normalizeQualityReports(qualityReportsResponse?.quality_reports ?? []);
  const auditEvents = normalizeAuditEvents(auditEventsResponse?.audit_events ?? []);
  const approvals = normalizeApprovals(approvalsResponse?.approvals ?? []);
  const qualityExplanations = await fetchQualityExplanations(project.id, runs, qualityReports);
  const issues = normalizeIssues(graphResponse?.issue_graph?.issues ?? [], runs, subagents, qualityExplanations);
  const timeline = liveTimeline(runs, executions, deployments);
  const qualitySignals = normalizeQualitySignals(qualityExplanations, qualityReports);

  return {
    mode: "live",
    backendStatus: "ok",
    generatedAt: new Date().toISOString(),
    project,
    stats: {
      issues: issues.length,
      accepted: issues.filter((issue) => issue.status === "accepted").length,
      blocked: issues.filter((issue) => issue.status === "blocked" || issue.status === "waiting").length,
      providers: providers.length,
      resources: resources.length,
      deployments: deployments.length,
      executions: executions.length,
      runs: runs.length,
      recoveries: recoveries.length,
      visual_assets: visualAssets.length,
      visual_render_executions: visualRenderExecutions.length,
    },
    issues: issues.length > 0 ? issues : demoSnapshot.issues,
    schedule: schedule.length > 0 ? schedule : demoSnapshot.schedule,
    subagent_backlog: subagentBacklog,
    providers: providers.length > 0 ? providers : demoSnapshot.providers,
    resources,
    deployments,
    executions,
    runs,
    subagents,
    runtime_recoveries: recoveries,
    visual_assets: visualAssets,
    visual_render_executions: visualRenderExecutions,
    quality_explanations: qualityExplanations,
    approvals,
    audit_events: auditEvents,
    timeline: timeline.length > 0 ? timeline : demoSnapshot.timeline,
    quality: qualitySignals.length > 0 ? qualitySignals : demoSnapshot.quality,
    memory:
      memoryResponse?.candidates?.map((candidate, index) => ({
        id: `candidate-${index + 1}`,
        kind: readString(candidate, "kind", "candidate"),
        summary: readString(candidate, "summary", "Memory candidate"),
        score: Number(readUnknown(candidate, "score") ?? 0.72),
      })) ?? demoSnapshot.memory,
  };
}

function normalizeScheduleItems(rawItems: unknown[], fallbackStatus: string): ScheduleItem[] {
  return rawItems.map((raw, index) => ({
    issue_id: readString(raw, "issue_id", `issue-${index + 1}`),
    status: readString(raw, "decision", readString(raw, "status", fallbackStatus)),
    runtime_id: readString(raw, "runtime_id", ""),
    reason: readString(raw, "reason", ""),
    blocked_reason: readString(raw, "blocked_reason", ""),
    subagent_id: readString(raw, "subagent_id", ""),
    subagent_status: readString(raw, "subagent_status", ""),
    recovery_id: readString(raw, "recovery_id", ""),
    retry_count: readNumber(raw, "retry_count"),
    max_retries: readNumber(raw, "max_retries"),
  }));
}

function normalizeSubagentBacklog(rawItems: unknown[]): SubagentBacklogItem[] {
  return rawItems.map((raw, index) => ({
    issue_id: readString(raw, "issue_id", `issue-${index + 1}`),
    subagent_id: readString(raw, "subagent_id", `subagent-${index + 1}`),
    status: readString(raw, "status", "archived"),
    reason: readString(raw, "reason", ""),
    recovery_id: readString(raw, "recovery_id", ""),
    failure_category: readString(raw, "failure_category", ""),
    retry_count: readNumber(raw, "retry_count"),
    max_retries: readNumber(raw, "max_retries"),
  }));
}

function normalizeIssues(rawIssues: unknown[], runs: RunSummary[], subagents: SubagentSummary[], explanations: QualityExplanation[]): IssueNode[] {
  const runsByIssue = new Map<string, RunSummary>();
  for (const run of runs) {
    if (run.issue_id && !runsByIssue.has(run.issue_id)) {
      runsByIssue.set(run.issue_id, run);
    }
  }
  const subagentsByID = new Map(subagents.map((item) => [item.id, item]));
  const subagentsByIssue = new Map<string, SubagentSummary>();
  for (const subagent of subagents) {
    if (subagent.issue_id && !subagentsByIssue.has(subagent.issue_id)) {
      subagentsByIssue.set(subagent.issue_id, subagent);
    }
  }
  const explanationsByReport = new Map(explanations.map((item) => [item.report_id, item]));

  return rawIssues.map((raw, index) => {
    const id = readString(raw, "id", `issue-${index + 1}`);
    const role = readString(raw, "role", index % 2 === 0 ? "backend" : "frontend");
    const status = readString(raw, "status", "ready");
    const run = runsByIssue.get(id);
    const subagent = run?.subagent_id ? subagentsByID.get(run.subagent_id) : subagentsByIssue.get(id);
    const qualityReportID = run?.quality_report_id || readString(raw, "quality_report_id", "");
    const explanation = qualityReportID ? explanationsByReport.get(qualityReportID) : undefined;

    return {
      id,
      title: readString(raw, "title", `Issue ${index + 1}`),
      role,
      status,
      depends_on: readArray(raw, "depends_on"),
      run_id: run?.run_id || undefined,
      subagent_id: run?.subagent_id || subagent?.id || undefined,
      runtime: run?.runtime_id || subagent?.runtime_id || (role === "frontend" ? "claude_cli" : "codex_cli"),
      runtime_status: run?.runtime_status || subagent?.status || undefined,
      provider: subagent?.provider_id || (role === "frontend" ? "claude_cli" : "codex_cli"),
      quality: explanation?.status || run?.quality_status || (status === "accepted" ? "passed" : "pending"),
      quality_report_id: qualityReportID || undefined,
      quality_decision: explanation?.decision,
      quality_reasons: explanation?.reasons ?? [],
      review_status: explanation?.review_status,
      blocking_findings: explanation?.findings.filter((finding) => finding.blocking) ?? [],
      skills: subagent?.skills ?? [],
      output_contract: subagent?.output_contract ?? [],
      blocked_reason: readString(raw, "blocked_reason", ""),
      lane: laneFor(role, status),
    };
  });
}

function normalizeProviders(rawProviders: unknown[]): ProviderSummary[] {
  return rawProviders.map((raw, index) => {
    const models = readUnknown(raw, "models");
    const model = Array.isArray(models) && models[0] && typeof models[0] === "object" ? readString(models[0], "id", "") : "";

    return {
      id: readString(raw, "id", `provider-${index + 1}`),
      name: readString(raw, "name", `Provider ${index + 1}`),
      vendor: readString(raw, "vendor", "unknown"),
      api_type: readString(raw, "api_type", "unknown"),
      enabled: Boolean(readUnknown(raw, "enabled") ?? false),
      runtime_id: readString(raw, "runtime_id", ""),
      model,
      use_cases: readArray(raw, "allowed_use_cases"),
    };
  });
}

function normalizeResources(rawResources: unknown[]): ResourceSummary[] {
  return rawResources.map((raw, index) => ({
    id: readString(raw, "id", `resource-${index + 1}`),
    environment: readString(raw, "environment", "test_dev"),
    host: readString(raw, "host", "unknown"),
    provider: readString(raw, "provider", ""),
    owner: readString(raw, "owner", ""),
    expires_at: readString(raw, "expires_at", ""),
    health: readString(readUnknown(raw, "healthcheck"), "last_status", "unknown"),
  }));
}

function normalizeDeployments(rawDeployments: unknown[]): DeploymentSummary[] {
  return rawDeployments.map((raw, index) => ({
    id: readString(raw, "id", `deployment-${index + 1}`),
    release_id: readString(raw, "release_id", ""),
    environment: readString(raw, "environment", "test_dev"),
    status: readString(raw, "status", "unknown"),
    decision: readString(raw, "decision", "unknown"),
    reasons: readArray(raw, "reasons"),
    resource_count: readObjectArray(raw, "resources").length,
    created_at: readString(raw, "created_at", ""),
  }));
}

function normalizeExecutions(rawExecutions: unknown[]): DeploymentExecutionSummary[] {
  return rawExecutions.map((raw, index) => ({
    id: readString(raw, "id", `execution-${index + 1}`),
    deployment_id: readString(raw, "deployment_id", ""),
    environment: readString(raw, "environment", "test_dev"),
    mode: readString(raw, "mode", "dry_run"),
    status: readString(raw, "status", "unknown"),
    decision: readString(raw, "decision", "unknown"),
    reasons: readArray(raw, "reasons"),
    step_count: readObjectArray(raw, "steps").length,
    started_at: readString(raw, "started_at", ""),
  }));
}

function normalizeRuns(rawRuns: unknown[]): RunSummary[] {
  return rawRuns.map((raw, index) => ({
    run_id: readString(raw, "run_id", `run-${index + 1}`),
    issue_id: readString(raw, "issue_id", ""),
    status: readString(raw, "status", "unknown"),
    subagent_id: readString(raw, "subagent_id", ""),
    runtime_id: readString(raw, "runtime_id", ""),
    runtime_status: readString(raw, "runtime_status", ""),
    recovery_id: readString(raw, "recovery_id", ""),
    quality_status: readString(raw, "quality_status", ""),
    quality_report_id: readString(raw, "quality_report_id", ""),
    updated_at: readString(raw, "updated_at", ""),
  }));
}

function normalizeSubagents(rawSubagents: unknown[]): SubagentSummary[] {
  return rawSubagents.map((raw, index) => ({
    id: readString(raw, "id", `subagent-${index + 1}`),
    issue_id: readString(raw, "issue_id", ""),
    run_id: readString(raw, "run_id", ""),
    status: readString(raw, "status", "unknown"),
    role: readString(raw, "role", "unknown"),
    runtime_id: readString(raw, "runtime_id", ""),
    provider_id: readString(raw, "provider_id", ""),
    model_id: readString(raw, "model_id", ""),
    skills: readArray(raw, "skills"),
    memory_scope: readArray(raw, "memory_scope"),
    read_scope: readArray(raw, "read_scope"),
    write_scope: readArray(raw, "write_scope"),
    output_contract: readArray(raw, "output_contract"),
    retry_policy: readString(raw, "retry_policy", ""),
    retry_count: readNumber(raw, "retry_count"),
    max_retries: readNumber(raw, "max_retries"),
    blocked_reason: readString(raw, "blocked_reason", ""),
    archive_reason: readString(raw, "archive_reason", ""),
    recovery_id: readString(raw, "recovery_id", ""),
    failure_category: readString(raw, "failure_category", ""),
    output_converged: readBoolean(raw, "output_converged"),
    updated_at: readString(raw, "updated_at", ""),
  }));
}

function normalizeRuntimeRecoveries(rawRecoveries: unknown[]): RuntimeRecoverySummary[] {
  return rawRecoveries.map((raw, index) => ({
    id: readString(raw, "id", `recovery-${index + 1}`),
    run_id: readString(raw, "run_id", ""),
    subagent_id: readString(raw, "subagent_id", ""),
    issue_id: readString(raw, "issue_id", ""),
    runtime_id: readString(raw, "runtime_id", ""),
    provider_id: readString(raw, "provider_id", ""),
    model_id: readString(raw, "model_id", ""),
    native_session_id: readString(raw, "native_session_id", ""),
    status: readString(raw, "status", "unknown"),
    failure_category: readString(raw, "failure_category", "unknown"),
    fallback_candidate: readString(raw, "fallback_candidate", ""),
    fallback_reason: readString(raw, "fallback_reason", ""),
    resume_hint: readString(raw, "resume_hint", ""),
    prompt_path: readString(raw, "prompt_path", ""),
    metadata_path: readString(raw, "metadata_path", ""),
    stdout_path: readString(raw, "stdout_path", ""),
    stderr_path: readString(raw, "stderr_path", ""),
    diff_summary_path: readString(raw, "diff_summary_path", ""),
    changed_files: readArray(raw, "changed_files"),
    risks: readArray(raw, "risks"),
    created_at: readString(raw, "created_at", ""),
    updated_at: readString(raw, "updated_at", ""),
  }));
}

function normalizeVisualAssets(rawAssets: unknown[]): VisualAssetSummary[] {
  return rawAssets.map((raw, index) => {
    const routeDecision = readUnknown(raw, "route_decision");
    return {
      id: readString(raw, "id", `visual-${index + 1}`),
      diagram_spec_id: readString(raw, "diagram_spec_id", ""),
      diagram_type: readString(raw, "diagram_type", "architecture"),
      title: readString(raw, "title", `Visual asset ${index + 1}`),
      status: readString(raw, "status", "planned"),
      provider_id: readString(raw, "provider_id", ""),
      model_id: readString(raw, "model_id", ""),
      size: readString(raw, "size", "3072x2048"),
      image_path: readString(raw, "image_path", ""),
      prompt_path: readString(raw, "prompt_path", ""),
      spec_path: readString(raw, "spec_path", ""),
      explanation_path: readString(raw, "explanation_path", ""),
      route_reason: readString(routeDecision, "reason", ""),
      created_at: readString(raw, "created_at", ""),
      updated_at: readString(raw, "updated_at", ""),
    };
  });
}

function normalizeVisualRenderExecutions(rawExecutions: unknown[]): VisualRenderExecutionSummary[] {
  return rawExecutions.map((raw, index) => ({
    id: readString(raw, "id", `visual-render-${index + 1}`),
    asset_id: readString(raw, "asset_id", ""),
    diagram_spec_id: readString(raw, "diagram_spec_id", ""),
    diagram_type: readString(raw, "diagram_type", ""),
    title: readString(raw, "title", ""),
    mode: readString(raw, "mode", "dry_run"),
    status: readString(raw, "status", "unknown"),
    decision: readString(raw, "decision", "unknown"),
    reasons: readArray(raw, "reasons"),
    provider_id: readString(raw, "provider_id", ""),
    model_id: readString(raw, "model_id", ""),
    size: readString(raw, "size", ""),
    prompt_path: readString(raw, "prompt_path", ""),
    spec_path: readString(raw, "spec_path", ""),
    image_path: readString(raw, "image_path", ""),
    script_path: readString(raw, "script_path", ""),
    step_count: readObjectArray(raw, "steps").length,
    started_at: readString(raw, "started_at", ""),
    finished_at: readString(raw, "finished_at", ""),
  }));
}

function normalizeQualityReports(rawReports: unknown[]) {
  return rawReports.map((raw, index) => ({
    id: readString(raw, "id", `quality-${index + 1}`),
    task_id: readString(raw, "task_id", ""),
    status: readString(raw, "status", "unknown"),
    review_status: readString(raw, "review_status", ""),
    findings_count: readObjectArray(raw, "findings").length,
  }));
}

function normalizeAuditEvents(rawEvents: unknown[]): AuditEventSummary[] {
  return rawEvents.map((raw, index) => ({
    id: readString(raw, "id", `audit-event-${index + 1}`),
    channel: readString(raw, "channel", readString(raw, "stream", "audit")),
    stream: readString(raw, "stream", readString(raw, "channel", "audit")),
    event: readString(raw, "event", "unknown.event"),
    ts: readString(raw, "ts", ""),
    issue_id: readString(raw, "issue_id", ""),
    run_id: readString(raw, "run_id", ""),
    subagent_id: readString(raw, "subagent_id", ""),
    trace_id: readString(raw, "trace_id", ""),
    status: readString(raw, "status", ""),
    decision: readString(raw, "decision", ""),
    reason: readString(raw, "reason", ""),
  }));
}

function normalizeApprovals(rawApprovals: unknown[]): ApprovalRecordSummary[] {
  return rawApprovals.map((raw, index) => ({
    id: readString(raw, "id", `approval-${index + 1}`),
    target_type: readString(raw, "target_type", "unknown"),
    target_id: readString(raw, "target_id", ""),
    action: readString(raw, "action", "unknown.action"),
    risk_level: readString(raw, "risk_level", "high"),
    status: readString(raw, "status", "pending"),
    decision: readString(raw, "decision", "APPROVAL_PENDING"),
    requested_by: readString(raw, "requested_by", "system"),
    request_reason: readString(raw, "request_reason", ""),
    decided_by: readString(raw, "decided_by", ""),
    decision_reason: readString(raw, "decision_reason", ""),
    requested_at: readString(raw, "requested_at", ""),
    decided_at: readString(raw, "decided_at", ""),
  }));
}

async function fetchQualityExplanations(projectID: string, runs: RunSummary[], reports: ReturnType<typeof normalizeQualityReports>) {
  const ids = unique([
    ...runs.map((run) => run.quality_report_id ?? ""),
    ...reports.map((report) => report.id),
  ]).slice(0, 8);
  const responses = await Promise.all(
    ids.map((id) => apiGet<ApiEnvelope<{ quality_explanation: unknown }>>(`/projects/${projectID}/quality/${encodeURIComponent(id)}/explain`)),
  );
  return responses
    .map((response) => response?.quality_explanation)
    .filter((value): value is unknown => Boolean(value))
    .map(normalizeQualityExplanation);
}

function normalizeQualityExplanation(raw: unknown): QualityExplanation {
  return {
    report_id: readString(raw, "report_id", ""),
    task_id: readString(raw, "task_id", ""),
    status: readString(raw, "status", "unknown"),
    review_status: readString(raw, "review_status", "unknown"),
    decision: readString(raw, "decision", "QUALITY_UNKNOWN"),
    reasons: readArray(raw, "reasons"),
    checks: readObjectArray(raw, "checks").map((check) => ({
      type: readString(check, "type", "check"),
      command: readString(check, "command", ""),
      status: readString(check, "status", "unknown"),
      reason: readString(check, "reason", ""),
    })),
    findings: readObjectArray(raw, "findings").map((finding, index) => ({
      id: readString(finding, "id", `finding-${index + 1}`),
      severity: readString(finding, "severity", "unknown"),
      category: readString(finding, "category", "unknown"),
      message: readString(finding, "message", ""),
      path: readString(finding, "path", ""),
      blocking: readBoolean(finding, "blocking"),
    })),
  };
}

function normalizeQualitySignals(explanations: QualityExplanation[], reports: ReturnType<typeof normalizeQualityReports>): QualitySignal[] {
  if (explanations.length > 0) {
    return explanations.slice(0, 4).map((explanation) => ({
      id: explanation.report_id,
      title: explanation.task_id || explanation.report_id,
      detail: explanation.reasons[0] ?? `${explanation.checks.length} checks / ${explanation.findings.length} findings`,
      status: explanation.decision,
      severity: toneFromQualityDecision(explanation.decision, explanation.status),
    }));
  }
  return reports.slice(0, 4).map((report) => ({
    id: report.id,
    title: report.task_id || report.id,
    detail: `${report.review_status || "review unknown"} / ${report.findings_count} findings`,
    status: report.status,
    severity: toneFromStatus(report.status),
  }));
}

function liveTimeline(runs: RunSummary[], executions: DeploymentExecutionSummary[], deployments: DeploymentSummary[]) {
  return [
    ...runs.map((run) => ({
      id: run.run_id,
      title: `Run ${run.issue_id || run.run_id}`,
      detail: `${run.runtime_id || "runtime pending"} / quality ${run.quality_status || "pending"}`,
      tone: toneFromStatus(run.status),
      time: shortTime(run.updated_at),
    })),
    ...executions.map((execution) => ({
      id: execution.id,
      title: `Deploy execution ${execution.mode}`,
      detail: `${execution.decision} / ${execution.step_count} steps`,
      tone: toneFromStatus(execution.status),
      time: shortTime(execution.started_at),
    })),
    ...deployments.map((deployment) => ({
      id: deployment.id,
      title: `Deployment plan ${deployment.environment}`,
      detail: `${deployment.decision} / ${deployment.resource_count} resources`,
      tone: toneFromStatus(deployment.status),
      time: shortTime(deployment.created_at),
    })),
  ].slice(0, 5);
}

function laneFor(role: string, status: string): IssueNode["lane"] {
  if (role.includes("frontend")) return "frontend";
  if (role.includes("quality") || status === "waiting") return "quality";
  if (role.includes("release") || role.includes("devops")) return "release";
  if (role.includes("backend")) return "backend";
  return "plan";
}

function readUnknown(value: unknown, key: string): unknown {
  if (!value || typeof value !== "object") return undefined;
  return (value as Record<string, unknown>)[key];
}

function readString(value: unknown, key: string, fallback: string): string {
  const field = readUnknown(value, key);
  return typeof field === "string" && field.trim() !== "" ? field : fallback;
}

function readArray(value: unknown, key: string): string[] {
  const field = readUnknown(value, key);
  if (!Array.isArray(field)) return [];
  return field.filter((item): item is string => typeof item === "string");
}

function readObjectArray(value: unknown, key: string): Record<string, unknown>[] {
  const field = readUnknown(value, key);
  if (!Array.isArray(field)) return [];
  return field.filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === "object");
}

function readNumber(value: unknown, key: string): number {
  const field = readUnknown(value, key);
  return typeof field === "number" && Number.isFinite(field) ? field : 0;
}

function readBoolean(value: unknown, key: string): boolean {
  return readUnknown(value, key) === true;
}

function unique(values: string[]): string[] {
  return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean)));
}

function toneFromStatus(status: string) {
  if (status === "completed" || status === "planned" || status === "accepted" || status === "passed" || status === "ready") return "ok" as const;
  if (status === "running" || status === "dispatch" || status === "retrying") return "running" as const;
  if (status === "blocked" || status === "failed" || status === "rejected" || status === "route_blocked") return "blocked" as const;
  if (status === "waiting" || status === "pending" || status === "archived") return "warning" as const;
  return "neutral" as const;
}

function toneFromQualityDecision(decision: string, status: string) {
  if (decision.includes("ACCEPTED") || status === "passed") return "ok" as const;
  if (decision.includes("BLOCKED") || status === "failed") return "blocked" as const;
  if (decision.includes("PENDING") || status === "pending") return "warning" as const;
  return toneFromStatus(status);
}

function shortTime(value?: string) {
  if (!value) return "live";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "live";
  return new Intl.DateTimeFormat("en", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}
