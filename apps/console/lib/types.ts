export type StatusTone = "ok" | "running" | "blocked" | "warning" | "neutral";

export type ProjectSummary = {
  id: string;
  name: string;
  root: string;
  status: string;
  source?: Record<string, unknown>;
};

export type IssueNode = {
  id: string;
  title: string;
  role: string;
  status: string;
  depends_on?: string[];
  run_id?: string;
  subagent_id?: string;
  runtime?: string;
  runtime_status?: string;
  provider?: string;
  quality?: string;
  quality_report_id?: string;
  quality_decision?: string;
  quality_reasons?: string[];
  review_status?: string;
  blocking_findings?: QualityFinding[];
  skills?: string[];
  output_contract?: string[];
  blocked_reason?: string;
  lane: "plan" | "backend" | "frontend" | "quality" | "release";
};

export type ScheduleItem = {
  issue_id: string;
  status: string;
  runtime_id?: string;
  reason?: string;
  blocked_reason?: string;
  subagent_id?: string;
  subagent_status?: string;
  recovery_id?: string;
  retry_count?: number;
  max_retries?: number;
};

export type SubagentBacklogItem = {
  issue_id: string;
  subagent_id: string;
  status: string;
  reason?: string;
  recovery_id?: string;
  failure_category?: string;
  retry_count: number;
  max_retries: number;
};

export type ProviderSummary = {
  id: string;
  name: string;
  vendor: string;
  api_type: string;
  enabled: boolean;
  runtime_id?: string;
  model?: string;
  use_cases: string[];
  health_status?: string;
  quota_status?: string;
  cost_status?: string;
};

export type ProviderTelemetrySummary = {
  id: string;
  provider_id: string;
  source: string;
  decision: string;
  reason?: string;
  health_status?: string;
  quota_status?: string;
  cost_status?: string;
  usage_tokens?: number;
  estimated_cost?: number;
  created_at?: string;
};

export type ResourceSummary = {
  id: string;
  environment: string;
  host: string;
  provider?: string;
  owner?: string;
  expires_at?: string;
  health?: string;
};

export type MaintenanceRecordSummary = {
  id: string;
  resource_id: string;
  environment: string;
  type: string;
  status: string;
  decision: string;
  expiration_state?: string;
  expires_at?: string;
  health_status?: string;
  reason?: string;
  created_at?: string;
};

export type DeploymentSummary = {
  id: string;
  release_id: string;
  environment: string;
  status: string;
  decision: string;
  reasons: string[];
  resource_count: number;
  created_at?: string;
};

export type DeploymentExecutionSummary = {
  id: string;
  deployment_id: string;
  environment: string;
  mode: string;
  status: string;
  decision: string;
  reasons: string[];
  step_count: number;
  started_at?: string;
  finished_at?: string;
};

export type ReleaseProviderExecutionSummary = {
  id: string;
  release_id: string;
  version?: string;
  provider?: string;
  mode: string;
  status: string;
  decision: string;
  reasons: string[];
  approval_id?: string;
  approval_consumed: boolean;
  write_enabled: boolean;
  action_count: number;
  remote_status?: string;
  started_at?: string;
  finished_at?: string;
};

export type EvidenceSummary = {
  id: string;
  parent_type: string;
  parent_id: string;
  subject_type: string;
  subject_id: string;
  operation: string;
  status: string;
  decision: string;
  reasons: string[];
  artifacts: EvidenceArtifactSummary[];
  artifact_count: number;
  created_at?: string;
};

export type EvidenceArtifactSummary = {
  kind: string;
  id?: string;
  path?: string;
};

export type RunSummary = {
  run_id: string;
  issue_id: string;
  status: string;
  subagent_id?: string;
  runtime_id?: string;
  runtime_status?: string;
  recovery_id?: string;
  quality_status?: string;
  quality_report_id?: string;
  updated_at?: string;
};

export type SubagentSummary = {
  id: string;
  issue_id: string;
  run_id: string;
  status: string;
  role: string;
  runtime_id: string;
  provider_id?: string;
  model_id?: string;
  skills: string[];
  memory_scope: string[];
  read_scope: string[];
  write_scope: string[];
  output_contract: string[];
  retry_policy?: string;
  retry_count?: number;
  max_retries?: number;
  blocked_reason?: string;
  archive_reason?: string;
  recovery_id?: string;
  failure_category?: string;
  output_converged?: boolean;
  updated_at?: string;
};

export type RuntimeRecoverySummary = {
  id: string;
  run_id: string;
  subagent_id?: string;
  issue_id?: string;
  runtime_id: string;
  provider_id?: string;
  model_id?: string;
  native_session_id?: string;
  status: string;
  failure_category: string;
  fallback_candidate?: string;
  fallback_reason?: string;
  resume_hint?: string;
  prompt_path?: string;
  metadata_path?: string;
  stdout_path?: string;
  stderr_path?: string;
  diff_summary_path?: string;
  changed_files: string[];
  risks: string[];
  created_at?: string;
  updated_at?: string;
};

export type VisualAssetSummary = {
  id: string;
  diagram_spec_id: string;
  diagram_type: string;
  title: string;
  status: string;
  provider_id?: string;
  model_id?: string;
  size: string;
  image_path?: string;
  prompt_path: string;
  spec_path: string;
  explanation_path?: string;
  route_reason?: string;
  created_at?: string;
  updated_at?: string;
};

export type VisualRenderExecutionSummary = {
  id: string;
  asset_id: string;
  diagram_spec_id?: string;
  diagram_type?: string;
  title?: string;
  mode: string;
  status: string;
  decision: string;
  reasons: string[];
  provider_id?: string;
  model_id?: string;
  size?: string;
  prompt_path?: string;
  spec_path?: string;
  image_path?: string;
  script_path?: string;
  step_count: number;
  started_at?: string;
  finished_at?: string;
};

export type QualityCheck = {
  type: string;
  command?: string;
  status: string;
  reason?: string;
};

export type QualityFinding = {
  id: string;
  severity: string;
  category: string;
  message: string;
  path?: string;
  blocking: boolean;
};

export type QualityExplanation = {
  report_id: string;
  task_id: string;
  status: string;
  review_status: string;
  decision: string;
  reasons: string[];
  checks: QualityCheck[];
  findings: QualityFinding[];
};

export type TimelineEvent = {
  id: string;
  title: string;
  detail: string;
  tone: StatusTone;
  time: string;
};

export type OperationHistoryItem = {
  id: string;
  type: "release_provider" | "deployment" | "visual_render" | "evidence";
  title: string;
  detail: string;
  status: string;
  decision: string;
  tone: StatusTone;
  time: string;
  occurred_at?: string;
  primary_ref?: string;
  secondary_ref?: string;
  evidence_ids: string[];
  reasons: string[];
  metadata: string[];
};

export type QualitySignal = {
  id: string;
  title: string;
  detail: string;
  status: string;
  severity: StatusTone;
};

export type MemorySignal = {
  id: string;
  summary: string;
  kind: string;
  score: number;
};

export type AuditEventSummary = {
  id: string;
  channel: string;
  stream: string;
  event: string;
  ts?: string;
  issue_id?: string;
  run_id?: string;
  subagent_id?: string;
  trace_id?: string;
  status?: string;
  decision?: string;
  reason?: string;
};

export type ApprovalRecordSummary = {
  id: string;
  target_type: string;
  target_id: string;
  action: string;
  risk_level: string;
  status: string;
  decision: string;
  requested_by: string;
  request_reason?: string;
  decided_by?: string;
  decision_reason?: string;
  requested_at?: string;
  decided_at?: string;
};

export type GitProviderPlanSummary = {
  id: string;
  issue_id: string;
  status: string;
  decision: string;
  provider: string;
  remote_name?: string;
  base_branch?: string;
  target_branch?: string;
  pr_mr_type?: string;
  create_mode?: string;
  remote_link?: string;
  remote_status?: string;
  preview_decision?: string;
  create_decision?: string;
  sync_decision?: string;
  sync_reason?: string;
  manual_required: boolean;
  created_at?: string;
};

export type AuthSessionSummary = {
  id: string;
  user_id: string;
  display_name?: string;
  roles: string[];
  status: string;
  created_at?: string;
};

export type APITokenSummary = {
  id: string;
  name: string;
  actor_id: string;
  scopes: string[];
  token_prefix: string;
  status: string;
  created_at?: string;
};

export type ServiceAccountSummary = {
  id: string;
  name: string;
  roles: string[];
  status: string;
  created_at?: string;
};

export type ConsoleSnapshot = {
  mode: "live" | "demo";
  backendStatus: StatusTone;
  generatedAt: string;
  project: ProjectSummary;
  stats: {
    issues: number;
    accepted: number;
    blocked: number;
    providers: number;
    resources: number;
    deployments: number;
    executions: number;
    runs: number;
    recoveries: number;
    visual_assets: number;
    visual_render_executions: number;
  };
  issues: IssueNode[];
  schedule: ScheduleItem[];
  subagent_backlog: SubagentBacklogItem[];
  providers: ProviderSummary[];
  provider_telemetry: ProviderTelemetrySummary[];
  resources: ResourceSummary[];
  maintenance_records: MaintenanceRecordSummary[];
  deployments: DeploymentSummary[];
  executions: DeploymentExecutionSummary[];
  release_provider_executions: ReleaseProviderExecutionSummary[];
  evidence: EvidenceSummary[];
  operation_history: OperationHistoryItem[];
  runs: RunSummary[];
  subagents: SubagentSummary[];
  runtime_recoveries: RuntimeRecoverySummary[];
  visual_assets: VisualAssetSummary[];
  visual_render_executions: VisualRenderExecutionSummary[];
  quality_explanations: QualityExplanation[];
  approvals: ApprovalRecordSummary[];
  audit_events: AuditEventSummary[];
  git_provider_plans: GitProviderPlanSummary[];
  auth_sessions: AuthSessionSummary[];
  api_tokens: APITokenSummary[];
  service_accounts: ServiceAccountSummary[];
  timeline: TimelineEvent[];
  quality: QualitySignal[];
  memory: MemorySignal[];
};
