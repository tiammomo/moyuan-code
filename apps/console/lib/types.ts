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
  runtime_status?: string;
  quality_status?: string;
  input_tokens?: number;
  output_tokens?: number;
  total_tokens?: number;
  usage_tokens?: number;
  incremental_cost?: number;
  estimated_cost?: number;
  feedback_status?: string;
  created_at?: string;
};

export type ResourceSummary = {
  id: string;
  environment: string;
  host: string;
  provider?: string;
  owner?: string;
  expires_at?: string;
  expiration_state?: string;
  maintenance_window?: string;
  health?: string;
};

export type LifecycleAlertSummary = {
  id: string;
  resource_id: string;
  environment: string;
  type: string;
  severity: string;
  status: string;
  decision: string;
  reason?: string;
  expiration_state?: string;
  expires_at?: string;
  maintenance_window?: string;
  health_status?: string;
  created_at?: string;
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
  release_id?: string;
  environment: string;
  mode: string;
  status: string;
  decision: string;
  reasons: string[];
  step_count: number;
  smoke_status?: string;
  monitor_status?: string;
  rollback_required?: boolean;
  approval_id?: string;
  approval_consumed?: boolean;
  started_at?: string;
  finished_at?: string;
};

export type PostDeploymentHistorySummary = {
  id: string;
  execution_id: string;
  deployment_id: string;
  release_id?: string;
  environment?: string;
  status: string;
  decision: string;
  failure_class: string;
  severity?: string;
  checks: PostDeploymentCheckSummary[];
  rollback: RollbackHistorySummary;
  evidence_ids: string[];
  artifacts: EvidenceArtifactSummary[];
  reasons: string[];
  created_at?: string;
};

export type PostDeploymentCheckSummary = {
  type: string;
  status: string;
  decision: string;
  template_id?: string;
  severity?: string;
  failure_class?: string;
  result_count: number;
  reasons: string[];
  checked_at?: string;
};

export type RollbackHistorySummary = {
  required: boolean;
  status: string;
  decision: string;
  reason?: string;
  runbook_status?: string;
  runbook_decision?: string;
  runbook_path?: string;
  step_count: number;
  actions: string[];
};

export type RollbackExecutionSummary = {
  id: string;
  execution_id: string;
  deployment_id?: string;
  release_id?: string;
  environment?: string;
  mode: string;
  status: string;
  decision: string;
  reasons: string[];
  step_count: number;
  approval_id?: string;
  approval_consumed: boolean;
  execution_enabled: boolean;
  started_at?: string;
  finished_at?: string;
};

export type DeploymentMonitorHistorySummary = {
  id: string;
  execution_id: string;
  deployment_id: string;
  release_id?: string;
  environment?: string;
  status: string;
  decision: string;
  failure_class?: string;
  severity?: string;
  rollback: boolean;
  created_at?: string;
};

export type DeploymentMonitorSummary = {
  id: string;
  environment?: string;
  status: string;
  decision: string;
  reasons: string[];
  window_size: number;
  history_count: number;
  failed_count: number;
  blocked_count: number;
  manual_count: number;
  rollback_count: number;
  failure_classes: Record<string, number>;
  latest: DeploymentMonitorHistorySummary[];
  evidence_ids: string[];
  created_at?: string;
};

export type ReleaseProviderExecutionSummary = {
  id: string;
  release_id: string;
  candidate_id?: string;
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

export type OperationRepairCandidateSummary = {
  id: string;
  status: string;
  decision: string;
  operation_type: string;
  operation_id: string;
  operation?: string;
  operation_status?: string;
  operation_decision?: string;
  failure_class: string;
  signal_type: string;
  signal_id?: string;
  bug_candidate_id?: string;
  repair_plan_id?: string;
  evidence_refs: string[];
  reasons: string[];
  review_required: boolean;
  reviewed_at?: string;
  reviewed_by?: string;
  review_decision?: string;
  review_reason?: string;
  issue_id?: string;
  repair_attempt_id?: string;
  created_at?: string;
};

export type ControlLoopStepSummary = {
  id: string;
  type: string;
  status: string;
  decision: string;
  summary?: string;
  reasons: string[];
  artifact_count: number;
  evidence_count: number;
  duration_ms: number;
  started_at?: string;
  finished_at?: string;
};

export type ControlLoopRunSummary = {
  id: string;
  status: string;
  decision: string;
  trigger: string;
  requested_by?: string;
  max_steps: number;
  step_timeout_ms: number;
  steps: ControlLoopStepSummary[];
  reasons: string[];
  started_at?: string;
  finished_at?: string;
};

export type BatchPlanSummary = {
  id: string;
  epic_id: string;
  mode: string;
  status: string;
  decision: string;
  max_parallel: number;
  dispatch_count: number;
  waiting_count: number;
  blocked_count: number;
  write_scope_conflict_count: number;
  runtime_slots: number;
  reasons: string[];
  item_count: number;
  created_at?: string;
};

export type BatchRunItemSummary = {
  issue_id: string;
  status: string;
  decision: string;
  reason?: string;
  runtime_id?: string;
  provider_id?: string;
  model_id?: string;
  worktree_id?: string;
  worktree_path?: string;
  branch?: string;
  worker_slot?: number;
  run_id?: string;
  subagent_id?: string;
  quality_report_id?: string;
  canceled_reason?: string;
};

export type BatchRunSummary = {
  id: string;
  batch_id: string;
  epic_id?: string;
  mode: string;
  status: string;
  decision: string;
  requested_by?: string;
  max_issues: number;
  parallelism: number;
  item_count: number;
  accepted_count: number;
  blocked_count: number;
  needs_rework_count: number;
  reasons: string[];
  items: BatchRunItemSummary[];
  started_at?: string;
  finished_at?: string;
};

export type WorktreeSummary = {
  id: string;
  epic_id?: string;
  batch_id?: string;
  issue_id: string;
  status: string;
  decision: string;
  worktree_path?: string;
  branch?: string;
  base_ref?: string;
  reasons: string[];
  created_at?: string;
  removed_at?: string;
};

export type MergeQueueItemSummary = {
  issue_id: string;
  status: string;
  decision: string;
  reason?: string;
  run_id?: string;
  quality_report_id?: string;
  worktree_id?: string;
  branch?: string;
};

export type MergeQueueSummary = {
  id: string;
  batch_id: string;
  epic_id?: string;
  batch_run_id?: string;
  status: string;
  decision: string;
  ready_count: number;
  needs_rework_count: number;
  blocked_count: number;
  reasons: string[];
  items: MergeQueueItemSummary[];
  created_at?: string;
};

export type IntegrationPreviewItemSummary = {
  issue_id: string;
  status: string;
  decision: string;
  reason?: string;
  branch?: string;
  worktree_id?: string;
  commit?: string;
  changed_files: string[];
  conflicted_files: string[];
  protected_files: string[];
};

export type IntegrationPreviewSummary = {
  id: string;
  merge_queue_id: string;
  batch_id?: string;
  epic_id?: string;
  status: string;
  decision: string;
  reasons: string[];
  base_ref?: string;
  integration_branch?: string;
  ready_count: number;
  conflict_count: number;
  blocked_count: number;
  item_count: number;
  items: IntegrationPreviewItemSummary[];
  created_at?: string;
};

export type IntegrationApplySummary = {
  id: string;
  preview_id: string;
  merge_queue_id?: string;
  batch_id?: string;
  epic_id?: string;
  mode: string;
  status: string;
  decision: string;
  reasons: string[];
  approved: boolean;
  requested_by?: string;
  source_branch?: string;
  target_branch?: string;
  write_enabled: boolean;
  action_count: number;
  started_at?: string;
  finished_at?: string;
};

export type ReleaseBatchSummary = {
  id: string;
  integration_apply_id: string;
  integration_preview_id?: string;
  merge_queue_id?: string;
  batch_id?: string;
  epic_id?: string;
  status: string;
  decision: string;
  version: string;
  release_branch: string;
  source_branch?: string;
  ready_item_count: number;
  min_items: number;
  reasons: string[];
  commands: string[];
  requested_by?: string;
  created_at?: string;
};

export type ReleaseCandidateSummary = {
  id: string;
  release_batch_id: string;
  integration_apply_id?: string;
  status: string;
  decision: string;
  version: string;
  release_branch: string;
  source_branch?: string;
  provider?: string;
  remote_name?: string;
  ready_item_count: number;
  deployment_targets: string[];
  reasons: string[];
  created_at?: string;
};

export type ReleaseCandidateApplySummary = {
  id: string;
  candidate_id: string;
  release_batch_id?: string;
  mode: string;
  status: string;
  decision: string;
  reasons: string[];
  release_branch?: string;
  source_branch?: string;
  write_enabled: boolean;
  action_count: number;
  started_at?: string;
  finished_at?: string;
};

export type ReleaseCandidateProviderPreviewSummary = {
  id: string;
  candidate_id: string;
  release_batch_id?: string;
  version?: string;
  provider?: string;
  status: string;
  decision: string;
  reasons: string[];
  remote_action_count: number;
  pr_mr_type?: string;
  pr_mr_decision?: string;
  pr_mr_head_branch?: string;
  created_at?: string;
};

export type CandidateDeploymentFeedbackSummary = {
  id: string;
  candidate_id: string;
  status: string;
  decision: string;
  failure_class?: string;
  severity?: string;
  latest_execution_id?: string;
  latest_deployment_id?: string;
  environment?: string;
  history_count: number;
  rollback_required: boolean;
  evidence_count: number;
  reasons: string[];
  created_at?: string;
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

export type OperationDetailSummary = {
  id: string;
  operation_type: string;
  operation: string;
  status: string;
  decision: string;
  reasons: string[];
  primary_ref?: string;
  secondary_ref?: string;
  started_at?: string;
  finished_at?: string;
  created_at?: string;
  summary: {
    mode?: string;
    release_id?: string;
    version?: string;
    provider?: string;
    deployment_id?: string;
    environment?: string;
    action_count?: number;
    step_count?: number;
    resource_count?: number;
    evidence_count?: number;
    artifact_count?: number;
    remote_status?: string;
    smoke_decision?: string;
    monitor_decision?: string;
    rollback_decision?: string;
    approval_id?: string;
    approval_consumed?: boolean;
    write_enabled?: boolean;
    remote_exec_enabled?: boolean;
  };
  evidence: EvidenceSummary[];
  artifacts: EvidenceArtifactSummary[];
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
  candidate_id?: string;
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
    control_loop_runs: number;
    batch_plans: number;
    batch_runs: number;
    worktrees: number;
    merge_queues: number;
    integration_previews: number;
    integration_applies: number;
    release_batches: number;
    release_candidates: number;
  };
  issues: IssueNode[];
  schedule: ScheduleItem[];
  subagent_backlog: SubagentBacklogItem[];
  providers: ProviderSummary[];
  provider_telemetry: ProviderTelemetrySummary[];
  resources: ResourceSummary[];
  lifecycle_alerts: LifecycleAlertSummary[];
  maintenance_records: MaintenanceRecordSummary[];
  deployments: DeploymentSummary[];
  executions: DeploymentExecutionSummary[];
  post_deployment_histories: PostDeploymentHistorySummary[];
  rollback_executions: RollbackExecutionSummary[];
  monitor_summaries: DeploymentMonitorSummary[];
  release_provider_executions: ReleaseProviderExecutionSummary[];
  evidence: EvidenceSummary[];
  operation_history: OperationHistoryItem[];
  operation_details: OperationDetailSummary[];
  runs: RunSummary[];
  subagents: SubagentSummary[];
  runtime_recoveries: RuntimeRecoverySummary[];
  operation_repair_candidates: OperationRepairCandidateSummary[];
  control_loop_runs: ControlLoopRunSummary[];
  batch_plans: BatchPlanSummary[];
  batch_runs: BatchRunSummary[];
  worktrees: WorktreeSummary[];
  merge_queues: MergeQueueSummary[];
  integration_previews: IntegrationPreviewSummary[];
  integration_applies: IntegrationApplySummary[];
  release_batches: ReleaseBatchSummary[];
  release_candidates: ReleaseCandidateSummary[];
  release_candidate_applies: ReleaseCandidateApplySummary[];
  release_candidate_provider_previews: ReleaseCandidateProviderPreviewSummary[];
  deployment_feedback: CandidateDeploymentFeedbackSummary[];
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
