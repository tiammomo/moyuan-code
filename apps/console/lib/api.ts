import { connection } from "next/server";
import { demoSnapshot } from "./demo-data";
import type {
  ConsoleSnapshot,
  AuditEventSummary,
  APITokenSummary,
  ApprovalRecordSummary,
  AuthSessionSummary,
  BatchPlanSummary,
  BatchRunSummary,
  ControlLoopRunSummary,
  DeploymentExecutionSummary,
  DeploymentSummary,
  EvidenceSummary,
  GitProviderPlanSummary,
  IssueNode,
  LifecycleAlertSummary,
  MaintenanceRecordSummary,
  OperationDetailSummary,
  OperationHistoryItem,
  OperationRepairCandidateSummary,
  PostDeploymentHistorySummary,
  ProjectSummary,
  ProviderSummary,
  ProviderTelemetrySummary,
  QualityExplanation,
  QualitySignal,
  ReleaseProviderExecutionSummary,
  RuntimeRecoverySummary,
  RunSummary,
  ResourceSummary,
  ScheduleItem,
  ServiceAccountSummary,
  SubagentBacklogItem,
  SubagentSummary,
  VisualAssetSummary,
  VisualRenderExecutionSummary,
  WorktreeSummary,
  MergeQueueSummary,
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
    providerTelemetryResponse,
    resourcesResponse,
    lifecycleAlertsResponse,
    maintenanceResponse,
    deploymentsResponse,
    executionsResponse,
    postDeploymentHistoriesResponse,
    releaseProviderExecutionsResponse,
    evidenceResponse,
    runsResponse,
    subagentsResponse,
    recoveriesResponse,
    operationRepairCandidatesResponse,
    visualAssetsResponse,
    visualRenderExecutionsResponse,
    qualityReportsResponse,
    memoryResponse,
    auditEventsResponse,
    approvalsResponse,
    gitProviderPlansResponse,
    controlLoopRunsResponse,
    batchPlansResponse,
    batchRunsResponse,
    worktreesResponse,
    mergeQueuesResponse,
    sessionsResponse,
    apiTokensResponse,
    serviceAccountsResponse,
  ] = await Promise.all([
    apiGet<ApiEnvelope<{ issue_graph: { issues?: unknown[] } }>>(`/projects/${project.id}/epics/phase1-epic/issue-graph`),
    apiGet<ApiEnvelope<{ schedule: { dispatch_queue?: unknown[]; waiting_queue?: unknown[]; subagent_backlog?: unknown[] } }>>(
      `/projects/${project.id}/epics/phase1-epic/schedule?limit=4`,
    ),
    apiGet<ApiEnvelope<{ providers: unknown[] }>>(`/projects/${project.id}/providers`),
    apiGet<ApiEnvelope<{ provider_telemetry: unknown[] }>>(`/projects/${project.id}/providers/telemetry?limit=6`),
    apiGet<ApiEnvelope<{ resources: unknown[] }>>(`/projects/${project.id}/resources`),
    apiGet<ApiEnvelope<{ lifecycle_alerts: unknown[] }>>(`/projects/${project.id}/resources/lifecycle-alerts?limit=5`),
    apiGet<ApiEnvelope<{ maintenance_records: unknown[] }>>(`/projects/${project.id}/resources/maintenance?limit=5`),
    apiGet<ApiEnvelope<{ deployments: unknown[] }>>(`/projects/${project.id}/deployments?limit=4`),
    apiGet<ApiEnvelope<{ executions: unknown[] }>>(`/projects/${project.id}/deployment-executions?limit=4`),
    apiGet<ApiEnvelope<{ post_deployment_histories: unknown[] }>>(`/projects/${project.id}/deployment-monitor-history?limit=5`),
    apiGet<ApiEnvelope<{ release_provider_executions: unknown[] }>>(`/projects/${project.id}/release-provider-executions?limit=6`),
    apiGet<ApiEnvelope<{ evidence: unknown[] }>>(`/projects/${project.id}/evidence?limit=30`),
    apiGet<ApiEnvelope<{ runs: unknown[] }>>(`/projects/${project.id}/runs?limit=12`),
    apiGet<ApiEnvelope<{ subagents: unknown[] }>>(`/projects/${project.id}/subagents?limit=12`),
    apiGet<ApiEnvelope<{ runtime_recoveries: unknown[] }>>(`/projects/${project.id}/runtime-recoveries?limit=6`),
    apiGet<ApiEnvelope<{ operation_repair_candidates: unknown[] }>>(`/projects/${project.id}/repair/operation-candidates?limit=4`),
    apiGet<ApiEnvelope<{ visual_assets: unknown[] }>>(`/projects/${project.id}/visuals/assets?limit=6`),
    apiGet<ApiEnvelope<{ visual_render_executions: unknown[] }>>(`/projects/${project.id}/visuals/render-executions?limit=6`),
    apiGet<ApiEnvelope<{ quality_reports: unknown[] }>>(`/projects/${project.id}/quality-reports?limit=8`),
    apiGet<ApiEnvelope<{ candidates: unknown[] }>>(`/projects/${project.id}/memory/candidates?limit=3`),
    apiGet<ApiEnvelope<{ audit_events: unknown[] }>>(`/projects/${project.id}/audit-events?channel=all&limit=10`),
    apiGet<ApiEnvelope<{ approvals: unknown[] }>>(`/projects/${project.id}/approvals?limit=6`),
    apiGet<ApiEnvelope<{ git_provider_plans: unknown[] }>>(`/projects/${project.id}/git-provider-plans?limit=5`),
    apiGet<ApiEnvelope<{ control_loop_runs: unknown[] }>>(`/projects/${project.id}/control-loop/runs?limit=5`),
    apiGet<ApiEnvelope<{ batch_plans: unknown[] }>>(`/projects/${project.id}/batches?limit=5`),
    apiGet<ApiEnvelope<{ batch_runs: unknown[] }>>(`/projects/${project.id}/batch-runs?limit=5`),
    apiGet<ApiEnvelope<{ worktrees: unknown[] }>>(`/projects/${project.id}/worktrees?limit=5`),
    apiGet<ApiEnvelope<{ merge_queues: unknown[] }>>(`/projects/${project.id}/merge-queues?limit=5`),
    apiGet<ApiEnvelope<{ sessions: unknown[] }>>(`/projects/${project.id}/auth/sessions`),
    apiGet<ApiEnvelope<{ api_tokens: unknown[] }>>(`/projects/${project.id}/auth/api-tokens`),
    apiGet<ApiEnvelope<{ service_accounts: unknown[] }>>(`/projects/${project.id}/auth/service-accounts`),
  ]);

  const schedule = [
    ...normalizeScheduleItems(scheduleResponse?.schedule.dispatch_queue ?? [], "dispatch"),
    ...normalizeScheduleItems(scheduleResponse?.schedule.waiting_queue ?? [], "waiting"),
  ];
  const subagentBacklog = normalizeSubagentBacklog(scheduleResponse?.schedule.subagent_backlog ?? []);
  const providers = normalizeProviders(providersResponse?.providers ?? []);
  const providerTelemetry = normalizeProviderTelemetry(providerTelemetryResponse?.provider_telemetry ?? []);
  const resources = normalizeResources(resourcesResponse?.resources ?? []);
  const lifecycleAlerts = normalizeLifecycleAlerts(lifecycleAlertsResponse?.lifecycle_alerts ?? []);
  const maintenanceRecords = normalizeMaintenanceRecords(maintenanceResponse?.maintenance_records ?? []);
  const deployments = normalizeDeployments(deploymentsResponse?.deployments ?? []);
  const executions = normalizeExecutions(executionsResponse?.executions ?? []);
  const postDeploymentHistories = normalizePostDeploymentHistories(postDeploymentHistoriesResponse?.post_deployment_histories ?? []);
  const releaseProviderExecutions = normalizeReleaseProviderExecutions(releaseProviderExecutionsResponse?.release_provider_executions ?? []);
  const evidence = normalizeEvidence(evidenceResponse?.evidence ?? []);
  const runs = normalizeRuns(runsResponse?.runs ?? []);
  const subagents = normalizeSubagents(subagentsResponse?.subagents ?? []);
  const recoveries = normalizeRuntimeRecoveries(recoveriesResponse?.runtime_recoveries ?? []);
  const operationRepairCandidates = normalizeOperationRepairCandidates(operationRepairCandidatesResponse?.operation_repair_candidates ?? []);
  const visualAssets = normalizeVisualAssets(visualAssetsResponse?.visual_assets ?? []);
  const visualRenderExecutions = normalizeVisualRenderExecutions(visualRenderExecutionsResponse?.visual_render_executions ?? []);
  const qualityReports = normalizeQualityReports(qualityReportsResponse?.quality_reports ?? []);
  const auditEvents = normalizeAuditEvents(auditEventsResponse?.audit_events ?? []);
  const approvals = normalizeApprovals(approvalsResponse?.approvals ?? []);
  const gitProviderPlans = normalizeGitProviderPlans(gitProviderPlansResponse?.git_provider_plans ?? []);
  const controlLoopRuns = normalizeControlLoopRuns(controlLoopRunsResponse?.control_loop_runs ?? []);
  const batchPlans = normalizeBatchPlans(batchPlansResponse?.batch_plans ?? []);
  const batchRuns = normalizeBatchRuns(batchRunsResponse?.batch_runs ?? []);
  const worktrees = normalizeWorktrees(worktreesResponse?.worktrees ?? []);
  const mergeQueues = normalizeMergeQueues(mergeQueuesResponse?.merge_queues ?? []);
  const authSessions = normalizeAuthSessions(sessionsResponse?.sessions ?? []);
  const apiTokens = normalizeAPITokens(apiTokensResponse?.api_tokens ?? []);
  const serviceAccounts = normalizeServiceAccounts(serviceAccountsResponse?.service_accounts ?? []);
  const qualityExplanations = await fetchQualityExplanations(project.id, runs, qualityReports);
  const issues = normalizeIssues(graphResponse?.issue_graph?.issues ?? [], runs, subagents, qualityExplanations);
  const operationHistory = buildOperationHistory(releaseProviderExecutions, executions, visualRenderExecutions, evidence);
  const operationDetails = await fetchOperationDetails(project.id, operationHistory);
  const timeline = liveTimeline(runs, executions, deployments, releaseProviderExecutions);
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
      control_loop_runs: controlLoopRuns.length,
      batch_plans: batchPlans.length,
      batch_runs: batchRuns.length,
      worktrees: worktrees.length,
      merge_queues: mergeQueues.length,
    },
    issues: issues.length > 0 ? issues : demoSnapshot.issues,
    schedule: schedule.length > 0 ? schedule : demoSnapshot.schedule,
    subagent_backlog: subagentBacklog,
    providers: providers.length > 0 ? providers : demoSnapshot.providers,
    provider_telemetry: providerTelemetry,
    resources,
    lifecycle_alerts: lifecycleAlerts,
    maintenance_records: maintenanceRecords,
    deployments,
    executions,
    post_deployment_histories: postDeploymentHistories,
    release_provider_executions: releaseProviderExecutions,
    evidence,
    operation_history: operationHistory,
    operation_details: operationDetails,
    runs,
    subagents,
    runtime_recoveries: recoveries,
    operation_repair_candidates: operationRepairCandidates,
    control_loop_runs: controlLoopRuns,
    batch_plans: batchPlans,
    batch_runs: batchRuns,
    worktrees,
    merge_queues: mergeQueues,
    visual_assets: visualAssets,
    visual_render_executions: visualRenderExecutions,
    quality_explanations: qualityExplanations,
    approvals,
    audit_events: auditEvents,
    git_provider_plans: gitProviderPlans,
    auth_sessions: authSessions,
    api_tokens: apiTokens,
    service_accounts: serviceAccounts,
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
      health_status: readString(readUnknown(raw, "health"), "status", ""),
      quota_status: readString(readUnknown(raw, "quota"), "status", ""),
      cost_status: readString(readUnknown(raw, "cost"), "status", ""),
    };
  });
}

function normalizeProviderTelemetry(rawRecords: unknown[]): ProviderTelemetrySummary[] {
  return rawRecords.map((raw, index) => ({
    id: readString(raw, "id", `provider-telemetry-${index + 1}`),
    provider_id: readString(raw, "provider_id", ""),
    source: readString(raw, "source", "unknown"),
    decision: readString(raw, "decision", "PROVIDER_TELEMETRY_UNKNOWN"),
    reason: readString(raw, "reason", ""),
    health_status: readString(raw, "health_status", ""),
    quota_status: readString(raw, "quota_status", ""),
    cost_status: readString(raw, "cost_status", ""),
    runtime_status: readString(raw, "runtime_status", ""),
    quality_status: readString(raw, "quality_status", ""),
    input_tokens: readNumber(raw, "input_tokens"),
    output_tokens: readNumber(raw, "output_tokens"),
    total_tokens: readNumber(raw, "total_tokens"),
    usage_tokens: readNumber(raw, "usage_tokens"),
    incremental_cost: readNumber(raw, "incremental_cost"),
    estimated_cost: readNumber(raw, "estimated_cost"),
    feedback_status: readString(raw, "feedback_status", ""),
    created_at: readString(raw, "created_at", ""),
  }));
}

function normalizeResources(rawResources: unknown[]): ResourceSummary[] {
  return rawResources.map((raw, index) => ({
    id: readString(raw, "id", `resource-${index + 1}`),
    environment: readString(raw, "environment", "test_dev"),
    host: readString(raw, "host", "unknown"),
    provider: readString(raw, "provider", ""),
    owner: readString(raw, "owner", ""),
    expires_at: readString(raw, "expires_at", ""),
    expiration_state: readString(raw, "expiration_state", ""),
    maintenance_window: readString(raw, "maintenance_window", ""),
    health: readString(readUnknown(raw, "healthcheck"), "last_status", "unknown"),
  }));
}

function normalizeLifecycleAlerts(rawAlerts: unknown[]): LifecycleAlertSummary[] {
  return rawAlerts.map((raw, index) => ({
    id: readString(raw, "id", `lifecycle-alert-${index + 1}`),
    resource_id: readString(raw, "resource_id", ""),
    environment: readString(raw, "environment", "test_dev"),
    type: readString(raw, "type", "lifecycle"),
    severity: readString(raw, "severity", "warning"),
    status: readString(raw, "status", "open"),
    decision: readString(raw, "decision", "RESOURCE_LIFECYCLE_ATTENTION_REQUIRED"),
    reason: readString(raw, "reason", ""),
    expiration_state: readString(raw, "expiration_state", ""),
    expires_at: readString(raw, "expires_at", ""),
    maintenance_window: readString(raw, "maintenance_window", ""),
    health_status: readString(raw, "health_status", ""),
    created_at: readString(raw, "created_at", ""),
  }));
}

function normalizeMaintenanceRecords(rawRecords: unknown[]): MaintenanceRecordSummary[] {
  return rawRecords.map((raw, index) => ({
    id: readString(raw, "id", `maintenance-${index + 1}`),
    resource_id: readString(raw, "resource_id", ""),
    environment: readString(raw, "environment", "test_dev"),
    type: readString(raw, "type", "maintenance"),
    status: readString(raw, "status", "open"),
    decision: readString(raw, "decision", "MAINTENANCE_REQUIRED"),
    expiration_state: readString(raw, "expiration_state", ""),
    expires_at: readString(raw, "expires_at", ""),
    health_status: readString(raw, "health_status", ""),
    reason: readString(raw, "reason", ""),
    created_at: readString(raw, "created_at", ""),
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
  return rawExecutions.map((raw, index) => {
    const smokeReport = readUnknown(raw, "smoke_report");
    const monitorReport = readUnknown(raw, "monitor_report");
    const rollbackSuggestion = readUnknown(raw, "rollback_suggestion");
    return {
      id: readString(raw, "id", `execution-${index + 1}`),
      deployment_id: readString(raw, "deployment_id", ""),
      environment: readString(raw, "environment", "test_dev"),
      mode: readString(raw, "mode", "dry_run"),
      status: readString(raw, "status", "unknown"),
      decision: readString(raw, "decision", "unknown"),
      reasons: readArray(raw, "reasons"),
      step_count: readObjectArray(raw, "steps").length,
      smoke_status: readString(smokeReport, "status", ""),
      monitor_status: readString(monitorReport, "status", ""),
      rollback_required: readBoolean(rollbackSuggestion, "required"),
      started_at: readString(raw, "started_at", ""),
      finished_at: readString(raw, "finished_at", ""),
    };
  });
}

function normalizePostDeploymentHistories(rawHistories: unknown[]): PostDeploymentHistorySummary[] {
  return rawHistories.map((raw, index) => {
    const rollback = readUnknown(raw, "rollback");
    return {
      id: readString(raw, "id", `post-deployment-history-${index + 1}`),
      execution_id: readString(raw, "execution_id", ""),
      deployment_id: readString(raw, "deployment_id", ""),
      release_id: readString(raw, "release_id", ""),
      environment: readString(raw, "environment", ""),
      status: readString(raw, "status", "unknown"),
      decision: readString(raw, "decision", "unknown"),
      failure_class: readString(raw, "failure_class", "unknown"),
      severity: readString(raw, "severity", ""),
      checks: readObjectArray(raw, "checks").map((check) => ({
        type: readString(check, "type", "check"),
        status: readString(check, "status", "unknown"),
        decision: readString(check, "decision", "unknown"),
        template_id: readString(check, "template_id", ""),
        severity: readString(check, "severity", ""),
        failure_class: readString(check, "failure_class", ""),
        result_count: readObjectArray(check, "results").length,
        reasons: readArray(check, "reasons"),
        checked_at: readString(check, "checked_at", ""),
      })),
      rollback: {
        required: readBoolean(rollback, "required"),
        status: readString(rollback, "status", "not_applicable"),
        decision: readString(rollback, "decision", ""),
        reason: readString(rollback, "reason", ""),
        runbook_status: readString(rollback, "runbook_status", ""),
        runbook_decision: readString(rollback, "runbook_decision", ""),
        runbook_path: readString(rollback, "runbook_path", ""),
        step_count: Number(readUnknown(rollback, "step_count") ?? 0),
        actions: readArray(rollback, "actions"),
      },
      evidence_ids: readArray(raw, "evidence_ids"),
      artifacts: normalizeEvidenceArtifacts(readObjectArray(raw, "artifacts")),
      reasons: readArray(raw, "reasons"),
      created_at: readString(raw, "created_at", ""),
    };
  });
}

function normalizeReleaseProviderExecutions(rawExecutions: unknown[]): ReleaseProviderExecutionSummary[] {
  return rawExecutions.map((raw, index) => {
    const remotePlan = readUnknown(raw, "remote_plan");
    return {
      id: readString(raw, "id", `release-provider-execution-${index + 1}`),
      release_id: readString(raw, "release_id", ""),
      version: readString(raw, "version", ""),
      provider: readString(raw, "provider", ""),
      mode: readString(raw, "mode", "preview"),
      status: readString(raw, "status", "unknown"),
      decision: readString(raw, "decision", "unknown"),
      reasons: readArray(raw, "reasons"),
      approval_id: readString(raw, "approval_id", ""),
      approval_consumed: readBoolean(raw, "approval_consumed"),
      write_enabled: readBoolean(raw, "write_enabled"),
      action_count: readObjectArray(remotePlan, "actions").length,
      remote_status: readString(remotePlan, "status", ""),
      started_at: readString(raw, "started_at", ""),
      finished_at: readString(raw, "finished_at", ""),
    };
  });
}

function normalizeEvidence(rawRecords: unknown[]): EvidenceSummary[] {
  return rawRecords.map((raw, index) => ({
    id: readString(raw, "id", `evidence-${index + 1}`),
    parent_type: readString(raw, "parent_type", ""),
    parent_id: readString(raw, "parent_id", ""),
    subject_type: readString(raw, "subject_type", ""),
    subject_id: readString(raw, "subject_id", ""),
    operation: readString(raw, "operation", "operation"),
    status: readString(raw, "status", "unknown"),
    decision: readString(raw, "decision", "unknown"),
    reasons: readArray(raw, "reasons"),
    artifacts: readObjectArray(raw, "artifacts").map((artifact) => ({
      kind: readString(artifact, "kind", "artifact"),
      id: readString(artifact, "id", ""),
      path: readString(artifact, "path", ""),
    })),
    artifact_count: readObjectArray(raw, "artifacts").length,
    created_at: readString(raw, "created_at", ""),
  }));
}

function normalizeOperationDetails(rawDetails: unknown[]): OperationDetailSummary[] {
  return rawDetails.map((raw) => ({
    id: readString(raw, "id", ""),
    operation_type: readString(raw, "operation_type", ""),
    operation: readString(raw, "operation", ""),
    status: readString(raw, "status", ""),
    decision: readString(raw, "decision", ""),
    reasons: readArray(raw, "reasons"),
    primary_ref: readString(raw, "primary_ref", ""),
    secondary_ref: readString(raw, "secondary_ref", ""),
    started_at: readString(raw, "started_at", ""),
    finished_at: readString(raw, "finished_at", ""),
    created_at: readString(raw, "created_at", ""),
    summary: normalizeOperationSummary(readUnknown(raw, "summary")),
    evidence: normalizeEvidence(readObjectArray(raw, "evidence")),
    artifacts: normalizeEvidenceArtifacts(readObjectArray(raw, "artifacts")),
  }));
}

function normalizeOperationSummary(raw: unknown): OperationDetailSummary["summary"] {
  return {
    mode: readString(raw, "mode", ""),
    release_id: readString(raw, "release_id", ""),
    version: readString(raw, "version", ""),
    provider: readString(raw, "provider", ""),
    deployment_id: readString(raw, "deployment_id", ""),
    environment: readString(raw, "environment", ""),
    action_count: readNumber(raw, "action_count"),
    step_count: readNumber(raw, "step_count"),
    resource_count: readNumber(raw, "resource_count"),
    evidence_count: readNumber(raw, "evidence_count"),
    artifact_count: readNumber(raw, "artifact_count"),
    remote_status: readString(raw, "remote_status", ""),
    smoke_decision: readString(raw, "smoke_decision", ""),
    monitor_decision: readString(raw, "monitor_decision", ""),
    rollback_decision: readString(raw, "rollback_decision", ""),
    approval_id: readString(raw, "approval_id", ""),
    approval_consumed: readBoolean(raw, "approval_consumed"),
    write_enabled: readBoolean(raw, "write_enabled"),
    remote_exec_enabled: readBoolean(raw, "remote_exec_enabled"),
  };
}

function normalizeEvidenceArtifacts(rawArtifacts: Record<string, unknown>[]): EvidenceSummary["artifacts"] {
  return rawArtifacts.map((raw) => ({
    kind: readString(raw, "kind", "artifact"),
    id: readString(raw, "id", ""),
    path: readString(raw, "path", ""),
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

function normalizeOperationRepairCandidates(rawCandidates: unknown[]): OperationRepairCandidateSummary[] {
  return rawCandidates.map((raw, index) => ({
    id: readString(raw, "id", `operation-repair-candidate-${index + 1}`),
    status: readString(raw, "status", "unknown"),
    decision: readString(raw, "decision", "unknown"),
    operation_type: readString(raw, "operation_type", ""),
    operation_id: readString(raw, "operation_id", ""),
    operation: readString(raw, "operation", ""),
    operation_status: readString(raw, "operation_status", ""),
    operation_decision: readString(raw, "operation_decision", ""),
    failure_class: readString(raw, "failure_class", "unknown"),
    signal_type: readString(raw, "signal_type", "runtime_error"),
    signal_id: readString(raw, "signal_id", ""),
    bug_candidate_id: readString(raw, "bug_candidate_id", ""),
    repair_plan_id: readString(raw, "repair_plan_id", ""),
    evidence_refs: readArray(raw, "evidence_refs"),
    reasons: readArray(raw, "reasons"),
    review_required: readBoolean(raw, "review_required"),
    reviewed_at: readString(raw, "reviewed_at", ""),
    reviewed_by: readString(raw, "reviewed_by", ""),
    review_decision: readString(raw, "review_decision", ""),
    review_reason: readString(raw, "review_reason", ""),
    issue_id: readString(raw, "issue_id", ""),
    repair_attempt_id: readString(raw, "repair_attempt_id", ""),
    created_at: readString(raw, "created_at", ""),
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

function normalizeGitProviderPlans(rawPlans: unknown[]): GitProviderPlanSummary[] {
  return rawPlans.map((raw, index) => {
    const prmr = readUnknown(raw, "pr_mr");
    return {
      id: readString(raw, "id", `git-plan-${index + 1}`),
      issue_id: readString(raw, "issue_id", ""),
      status: readString(raw, "status", "unknown"),
      decision: readString(raw, "decision", "unknown"),
      provider: readString(raw, "provider", "generic_git"),
      remote_name: readString(raw, "remote_name", ""),
      base_branch: readString(raw, "base_branch", ""),
      target_branch: readString(raw, "target_branch", ""),
      pr_mr_type: readString(prmr, "type", ""),
      create_mode: readString(prmr, "create_mode", ""),
      remote_link: readString(prmr, "remote_link", ""),
      remote_status: readString(prmr, "remote_status", ""),
      preview_decision: readString(prmr, "preview_decision", ""),
      create_decision: readString(prmr, "create_decision", ""),
      sync_decision: readString(prmr, "sync_decision", ""),
      sync_reason: readString(prmr, "sync_reason", ""),
      manual_required: readBoolean(raw, "manual_required"),
      created_at: readString(raw, "created_at", ""),
    };
  });
}

function normalizeControlLoopRuns(rawRuns: unknown[]): ControlLoopRunSummary[] {
  return rawRuns.map((raw, index) => ({
    id: readString(raw, "id", `control-loop-${index + 1}`),
    status: readString(raw, "status", "unknown"),
    decision: readString(raw, "decision", "unknown"),
    trigger: readString(raw, "trigger", "manual"),
    requested_by: readString(raw, "requested_by", ""),
    max_steps: readNumber(raw, "max_steps"),
    step_timeout_ms: readNumber(raw, "step_timeout_ms"),
    steps: readObjectArray(raw, "steps").map((step, stepIndex) => ({
      id: readString(step, "id", `control-loop-step-${stepIndex + 1}`),
      type: readString(step, "type", "unknown"),
      status: readString(step, "status", "unknown"),
      decision: readString(step, "decision", "unknown"),
      summary: readString(step, "summary", ""),
      reasons: readArray(step, "reasons"),
      artifact_count: readObjectArray(step, "artifacts").length,
      evidence_count: readArray(step, "evidence_ids").length,
      duration_ms: readNumber(step, "duration_ms"),
      started_at: readString(step, "started_at", ""),
      finished_at: readString(step, "finished_at", ""),
    })),
    reasons: readArray(raw, "reasons"),
    started_at: readString(raw, "started_at", ""),
    finished_at: readString(raw, "finished_at", ""),
  }));
}

function normalizeBatchPlans(rawPlans: unknown[]): BatchPlanSummary[] {
  return rawPlans.map((raw, index) => ({
    id: readString(raw, "id", `batch-plan-${index + 1}`),
    epic_id: readString(raw, "epic_id", ""),
    mode: readString(raw, "mode", "dry_run"),
    status: readString(raw, "status", "unknown"),
    decision: readString(raw, "decision", "unknown"),
    max_parallel: readNumber(raw, "max_parallel"),
    dispatch_count: readNumber(raw, "dispatch_count"),
    waiting_count: readNumber(raw, "waiting_count"),
    blocked_count: readNumber(raw, "blocked_count"),
    write_scope_conflict_count: readNumber(raw, "write_scope_conflict_count"),
    runtime_slots: readNumber(raw, "runtime_slots"),
    reasons: readArray(raw, "reasons"),
    item_count: readObjectArray(raw, "items").length,
    created_at: readString(raw, "created_at", ""),
  }));
}

function normalizeBatchRuns(rawRuns: unknown[]): BatchRunSummary[] {
  return rawRuns.map((raw, index) => {
    const items = readObjectArray(raw, "items").map((item) => ({
      issue_id: readString(item, "issue_id", ""),
      status: readString(item, "status", "unknown"),
      decision: readString(item, "decision", "unknown"),
      reason: readString(item, "reason", ""),
      runtime_id: readString(item, "runtime_id", ""),
      provider_id: readString(item, "provider_id", ""),
      model_id: readString(item, "model_id", ""),
      worktree_id: readString(item, "worktree_id", ""),
      worktree_path: readString(item, "worktree_path", ""),
      branch: readString(item, "branch", ""),
      run_id: readString(item, "run_id", ""),
      subagent_id: readString(item, "subagent_id", ""),
      quality_report_id: readString(item, "quality_report_id", ""),
    }));
    return {
      id: readString(raw, "id", `batch-run-${index + 1}`),
      batch_id: readString(raw, "batch_id", ""),
      epic_id: readString(raw, "epic_id", ""),
      mode: readString(raw, "mode", "dry_run"),
      status: readString(raw, "status", "unknown"),
      decision: readString(raw, "decision", "unknown"),
      requested_by: readString(raw, "requested_by", ""),
      max_issues: readNumber(raw, "max_issues"),
      item_count: items.length,
      accepted_count: items.filter((item) => item.decision === "BATCH_ITEM_ACCEPTED").length,
      blocked_count: items.filter((item) => item.status === "blocked" || item.decision.includes("BLOCKED")).length,
      needs_rework_count: items.filter((item) => item.status === "failed" || item.decision.includes("REWORK")).length,
      reasons: readArray(raw, "reasons"),
      items,
      started_at: readString(raw, "started_at", ""),
      finished_at: readString(raw, "finished_at", ""),
    };
  });
}

function normalizeWorktrees(rawRecords: unknown[]): WorktreeSummary[] {
  return rawRecords.map((raw, index) => ({
    id: readString(raw, "id", `worktree-${index + 1}`),
    epic_id: readString(raw, "epic_id", ""),
    batch_id: readString(raw, "batch_id", ""),
    issue_id: readString(raw, "issue_id", ""),
    status: readString(raw, "status", "unknown"),
    decision: readString(raw, "decision", "unknown"),
    worktree_path: readString(raw, "worktree_path", ""),
    branch: readString(raw, "branch", ""),
    base_ref: readString(raw, "base_ref", ""),
    reasons: readArray(raw, "reasons"),
    created_at: readString(raw, "created_at", ""),
    removed_at: readString(raw, "removed_at", ""),
  }));
}

function normalizeMergeQueues(rawQueues: unknown[]): MergeQueueSummary[] {
  return rawQueues.map((raw, index) => ({
    id: readString(raw, "id", `merge-queue-${index + 1}`),
    batch_id: readString(raw, "batch_id", ""),
    epic_id: readString(raw, "epic_id", ""),
    batch_run_id: readString(raw, "batch_run_id", ""),
    status: readString(raw, "status", "unknown"),
    decision: readString(raw, "decision", "unknown"),
    ready_count: readNumber(raw, "ready_count"),
    needs_rework_count: readNumber(raw, "needs_rework_count"),
    blocked_count: readNumber(raw, "blocked_count"),
    reasons: readArray(raw, "reasons"),
    items: readObjectArray(raw, "items").map((item) => ({
      issue_id: readString(item, "issue_id", ""),
      status: readString(item, "status", "unknown"),
      decision: readString(item, "decision", "unknown"),
      reason: readString(item, "reason", ""),
      run_id: readString(item, "run_id", ""),
      quality_report_id: readString(item, "quality_report_id", ""),
      worktree_id: readString(item, "worktree_id", ""),
      branch: readString(item, "branch", ""),
    })),
    created_at: readString(raw, "created_at", ""),
  }));
}

function normalizeAuthSessions(rawSessions: unknown[]): AuthSessionSummary[] {
  return rawSessions.map((raw, index) => ({
    id: readString(raw, "id", `session-${index + 1}`),
    user_id: readString(raw, "user_id", "unknown"),
    display_name: readString(raw, "display_name", ""),
    roles: readArray(raw, "roles"),
    status: readString(raw, "status", "unknown"),
    created_at: readString(raw, "created_at", ""),
  }));
}

function normalizeAPITokens(rawTokens: unknown[]): APITokenSummary[] {
  return rawTokens.map((raw, index) => ({
    id: readString(raw, "id", `api-token-${index + 1}`),
    name: readString(raw, "name", "token"),
    actor_id: readString(raw, "actor_id", "unknown"),
    scopes: readArray(raw, "scopes"),
    token_prefix: readString(raw, "token_prefix", ""),
    status: readString(raw, "status", "unknown"),
    created_at: readString(raw, "created_at", ""),
  }));
}

function normalizeServiceAccounts(rawAccounts: unknown[]): ServiceAccountSummary[] {
  return rawAccounts.map((raw, index) => ({
    id: readString(raw, "id", `svc-${index + 1}`),
    name: readString(raw, "name", "service account"),
    roles: readArray(raw, "roles"),
    status: readString(raw, "status", "unknown"),
    created_at: readString(raw, "created_at", ""),
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

async function fetchOperationDetails(projectID: string, operations: OperationHistoryItem[]) {
  const supported = operations
    .filter((operation) => operation.type === "release_provider" || operation.type === "deployment" || operation.type === "evidence")
    .slice(0, 8);
  const responses = await Promise.all(
    supported.map((operation) =>
      apiGet<ApiEnvelope<{ operation_detail: unknown }>>(
        `/projects/${projectID}/operations/${operation.type}/${encodeURIComponent(operation.id)}`,
      ),
    ),
  );
  return normalizeOperationDetails(responses.map((response) => response?.operation_detail).filter((value): value is unknown => Boolean(value)));
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

function buildOperationHistory(
  releaseProviderExecutions: ReleaseProviderExecutionSummary[],
  executions: DeploymentExecutionSummary[],
  visualExecutions: VisualRenderExecutionSummary[],
  evidence: EvidenceSummary[],
): OperationHistoryItem[] {
  const evidenceByParent = new Map<string, EvidenceSummary[]>();
  for (const record of evidence) {
    const key = `${record.parent_type}:${record.parent_id}`;
    evidenceByParent.set(key, [...(evidenceByParent.get(key) ?? []), record]);
  }

  const items: OperationHistoryItem[] = [
    ...releaseProviderExecutions.map((execution) => {
      const linkedEvidence = evidenceByParent.get(`release_provider_execution:${execution.id}`) ?? [];
      return {
        id: execution.id,
        type: "release_provider" as const,
        title: `Release provider ${execution.mode}`,
        detail: `${execution.decision} / ${execution.action_count} actions`,
        status: execution.status,
        decision: execution.decision,
        tone: toneFromStatus(execution.status),
        time: shortTime(execution.finished_at || execution.started_at),
        occurred_at: execution.finished_at || execution.started_at,
        primary_ref: execution.release_id,
        secondary_ref: execution.provider || execution.version,
        evidence_ids: linkedEvidence.map((record) => record.id),
        reasons: execution.reasons,
        metadata: [
          execution.write_enabled ? "write enabled" : "preview guarded",
          execution.approval_consumed ? "approval consumed" : "approval not consumed",
          execution.remote_status ? `remote ${execution.remote_status}` : "",
        ].filter(Boolean),
      };
    }),
    ...executions.map((execution) => {
      const linkedEvidence = evidenceByParent.get(`deployment_execution:${execution.id}`) ?? [];
      return {
        id: execution.id,
        type: "deployment" as const,
        title: `Deployment ${execution.mode}`,
        detail: `${execution.decision} / ${execution.step_count} steps`,
        status: execution.status,
        decision: execution.decision,
        tone: toneFromStatus(execution.status),
        time: shortTime(execution.finished_at || execution.started_at),
        occurred_at: execution.finished_at || execution.started_at,
        primary_ref: execution.deployment_id,
        secondary_ref: execution.environment,
        evidence_ids: linkedEvidence.map((record) => record.id),
        reasons: execution.reasons,
        metadata: [`environment ${execution.environment}`, `${execution.step_count} steps`],
      };
    }),
    ...visualExecutions.map((execution) => ({
      id: execution.id,
      type: "visual_render" as const,
      title: `Visual render ${execution.mode}`,
      detail: `${execution.decision} / ${execution.step_count} steps`,
      status: execution.status,
      decision: execution.decision,
      tone: toneFromStatus(execution.status),
      time: shortTime(execution.finished_at || execution.started_at),
      occurred_at: execution.finished_at || execution.started_at,
      primary_ref: execution.asset_id,
      secondary_ref: execution.provider_id,
      evidence_ids: [],
      reasons: execution.reasons,
      metadata: [execution.image_path ? "image artifact" : "", execution.script_path ? "script artifact" : ""].filter(Boolean),
    })),
    ...evidence
      .filter((record) => !record.parent_id)
      .map((record) => ({
        id: record.id,
        type: "evidence" as const,
        title: record.operation,
        detail: `${record.decision} / ${record.artifact_count} artifacts`,
        status: record.status,
        decision: record.decision,
        tone: toneFromStatus(record.status),
        time: shortTime(record.created_at),
        occurred_at: record.created_at,
        primary_ref: record.subject_id,
        secondary_ref: record.subject_type,
        evidence_ids: [record.id],
        reasons: record.reasons,
        metadata: [`${record.artifact_count} artifacts`],
      })),
  ];

  return items.sort((a, b) => timestampOf(b.occurred_at) - timestampOf(a.occurred_at)).slice(0, 12);
}

function liveTimeline(
  runs: RunSummary[],
  executions: DeploymentExecutionSummary[],
  deployments: DeploymentSummary[],
  releaseProviderExecutions: ReleaseProviderExecutionSummary[],
) {
  const items = [
    ...runs.map((run) => ({
      id: run.run_id,
      title: `Run ${run.issue_id || run.run_id}`,
      detail: `${run.runtime_id || "runtime pending"} / quality ${run.quality_status || "pending"}`,
      tone: toneFromStatus(run.status),
      occurred_at: run.updated_at,
      time: shortTime(run.updated_at),
    })),
    ...executions.map((execution) => ({
      id: execution.id,
      title: `Deploy execution ${execution.mode}`,
      detail: `${execution.decision} / ${execution.step_count} steps`,
      tone: toneFromStatus(execution.status),
      occurred_at: execution.finished_at || execution.started_at,
      time: shortTime(execution.started_at),
    })),
    ...releaseProviderExecutions.map((execution) => ({
      id: execution.id,
      title: `Release provider ${execution.mode}`,
      detail: `${execution.decision} / ${execution.action_count} actions`,
      tone: toneFromStatus(execution.status),
      occurred_at: execution.finished_at || execution.started_at,
      time: shortTime(execution.finished_at || execution.started_at),
    })),
    ...deployments.map((deployment) => ({
      id: deployment.id,
      title: `Deployment plan ${deployment.environment}`,
      detail: `${deployment.decision} / ${deployment.resource_count} resources`,
      tone: toneFromStatus(deployment.status),
      occurred_at: deployment.created_at,
      time: shortTime(deployment.created_at),
    })),
  ].sort((a, b) => timestampOf(b.occurred_at) - timestampOf(a.occurred_at));

  return items.slice(0, 5).map(({ occurred_at: _, ...item }) => item);
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

function timestampOf(value?: string) {
  if (!value || value === "live") return 0;
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? 0 : timestamp;
}
