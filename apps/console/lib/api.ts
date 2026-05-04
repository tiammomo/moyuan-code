import { connection } from "next/server";
import { demoSnapshot } from "./demo-data";
import type {
  ConsoleSnapshot,
  DeploymentExecutionSummary,
  DeploymentSummary,
  IssueNode,
  ProjectSummary,
  ProviderSummary,
  QualityExplanation,
  QualitySignal,
  RunSummary,
  ResourceSummary,
  ScheduleItem,
  SubagentSummary,
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
    qualityReportsResponse,
    memoryResponse,
  ] = await Promise.all([
    apiGet<ApiEnvelope<{ issue_graph: { issues?: unknown[] } }>>(`/projects/${project.id}/epics/phase1-epic/issue-graph`),
    apiGet<ApiEnvelope<{ schedule: { dispatch_queue?: ScheduleItem[]; waiting_queue?: ScheduleItem[] } }>>(
      `/projects/${project.id}/epics/phase1-epic/schedule?limit=4`,
    ),
    apiGet<ApiEnvelope<{ providers: unknown[] }>>(`/projects/${project.id}/providers`),
    apiGet<ApiEnvelope<{ resources: unknown[] }>>(`/projects/${project.id}/resources`),
    apiGet<ApiEnvelope<{ deployments: unknown[] }>>(`/projects/${project.id}/deployments?limit=4`),
    apiGet<ApiEnvelope<{ executions: unknown[] }>>(`/projects/${project.id}/deployment-executions?limit=4`),
    apiGet<ApiEnvelope<{ runs: unknown[] }>>(`/projects/${project.id}/runs?limit=12`),
    apiGet<ApiEnvelope<{ subagents: unknown[] }>>(`/projects/${project.id}/subagents?limit=12`),
    apiGet<ApiEnvelope<{ quality_reports: unknown[] }>>(`/projects/${project.id}/quality-reports?limit=8`),
    apiGet<ApiEnvelope<{ candidates: unknown[] }>>(`/projects/${project.id}/memory/candidates?limit=3`),
  ]);

  const schedule = [
    ...(scheduleResponse?.schedule.dispatch_queue ?? []),
    ...(scheduleResponse?.schedule.waiting_queue ?? []),
  ];
  const providers = normalizeProviders(providersResponse?.providers ?? []);
  const resources = normalizeResources(resourcesResponse?.resources ?? []);
  const deployments = normalizeDeployments(deploymentsResponse?.deployments ?? []);
  const executions = normalizeExecutions(executionsResponse?.executions ?? []);
  const runs = normalizeRuns(runsResponse?.runs ?? []);
  const subagents = normalizeSubagents(subagentsResponse?.subagents ?? []);
  const qualityReports = normalizeQualityReports(qualityReportsResponse?.quality_reports ?? []);
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
    },
    issues: issues.length > 0 ? issues : demoSnapshot.issues,
    schedule: schedule.length > 0 ? schedule : demoSnapshot.schedule,
    providers: providers.length > 0 ? providers : demoSnapshot.providers,
    resources,
    deployments,
    executions,
    runs,
    subagents,
    quality_explanations: qualityExplanations,
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
    updated_at: readString(raw, "updated_at", ""),
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

function readBoolean(value: unknown, key: string): boolean {
  return readUnknown(value, key) === true;
}

function unique(values: string[]): string[] {
  return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean)));
}

function toneFromStatus(status: string) {
  if (status === "completed" || status === "planned" || status === "accepted" || status === "passed" || status === "ready") return "ok" as const;
  if (status === "running" || status === "dispatch") return "running" as const;
  if (status === "blocked" || status === "failed" || status === "rejected") return "blocked" as const;
  if (status === "waiting" || status === "pending") return "warning" as const;
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
