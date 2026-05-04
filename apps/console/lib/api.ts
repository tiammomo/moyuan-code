import { connection } from "next/server";
import { demoSnapshot } from "./demo-data";
import type {
  ConsoleSnapshot,
  DeploymentExecutionSummary,
  DeploymentSummary,
  IssueNode,
  ProjectSummary,
  ProviderSummary,
  RunSummary,
  ResourceSummary,
  ScheduleItem,
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
    apiGet<ApiEnvelope<{ runs: unknown[] }>>(`/projects/${project.id}/runs?limit=6`),
    apiGet<ApiEnvelope<{ candidates: unknown[] }>>(`/projects/${project.id}/memory/candidates?limit=3`),
  ]);

  const issues = normalizeIssues(graphResponse?.issue_graph?.issues ?? []);
  const schedule = [
    ...(scheduleResponse?.schedule.dispatch_queue ?? []),
    ...(scheduleResponse?.schedule.waiting_queue ?? []),
  ];
  const providers = normalizeProviders(providersResponse?.providers ?? []);
  const resources = normalizeResources(resourcesResponse?.resources ?? []);
  const deployments = normalizeDeployments(deploymentsResponse?.deployments ?? []);
  const executions = normalizeExecutions(executionsResponse?.executions ?? []);
  const runs = normalizeRuns(runsResponse?.runs ?? []);
  const timeline = liveTimeline(runs, executions, deployments);

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
    timeline: timeline.length > 0 ? timeline : demoSnapshot.timeline,
    quality: demoSnapshot.quality,
    memory:
      memoryResponse?.candidates?.map((candidate, index) => ({
        id: `candidate-${index + 1}`,
        kind: readString(candidate, "kind", "candidate"),
        summary: readString(candidate, "summary", "Memory candidate"),
        score: Number(readUnknown(candidate, "score") ?? 0.72),
      })) ?? demoSnapshot.memory,
  };
}

function normalizeIssues(rawIssues: unknown[]): IssueNode[] {
  return rawIssues.map((raw, index) => {
    const role = readString(raw, "role", index % 2 === 0 ? "backend" : "frontend");
    const status = readString(raw, "status", "ready");

    return {
      id: readString(raw, "id", `issue-${index + 1}`),
      title: readString(raw, "title", `Issue ${index + 1}`),
      role,
      status,
      depends_on: readArray(raw, "depends_on"),
      runtime: role === "frontend" ? "claude_cli" : "codex_cli",
      provider: role === "frontend" ? "claude_cli" : "codex_cli",
      quality: status === "accepted" ? "passed" : "pending",
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
    runtime_id: readString(raw, "runtime_id", ""),
    runtime_status: readString(raw, "runtime_status", ""),
    quality_status: readString(raw, "quality_status", ""),
    quality_report_id: readString(raw, "quality_report_id", ""),
    updated_at: readString(raw, "updated_at", ""),
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

function toneFromStatus(status: string) {
  if (status === "completed" || status === "planned") return "ok" as const;
  if (status === "running") return "running" as const;
  if (status === "blocked" || status === "failed") return "blocked" as const;
  return "neutral" as const;
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
