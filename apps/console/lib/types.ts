export type StatusTone = "ok" | "running" | "blocked" | "warning" | "neutral";

export type ProjectSummary = {
  id: string;
  name: string;
  root: string;
  status: string;
  remote_url?: string;
  languages?: string[];
  frameworks?: string[];
  package_managers?: string[];
  source_type?: string;
  provider?: string;
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
  commit_before?: string;
  commit_after?: string;
  commit_changed?: boolean;
  changed_files?: string[];
  diff_summary_path?: string;
  skills?: string[];
  output_contract?: string[];
  blocked_reason?: string;
  lane: "plan" | "backend" | "frontend" | "quality" | "release";
};

export type RequirementIssueSummary = {
  id: string;
  title: string;
  status: string;
  role?: string;
  depends_on: string[];
};

export type RequirementSummary = {
  id: string;
  epic_id: string;
  title: string;
  status: string;
  clarified_requirement: string;
  raw_text: string;
  issue_count: number;
  accepted_count: number;
  blocked_count: number;
  commit_count: number;
  issues: RequirementIssueSummary[];
  created_at?: string;
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

export type SkillSummary = {
  id: string;
  name: string;
  source: string;
  version?: string;
  description?: string;
  enabled: boolean;
  risk_level: string;
  compatible_roles: string[];
  tags: string[];
  required_tools: string[];
  auth_ref?: string;
  created_at?: string;
  updated_at?: string;
};

export type SkillBindingSummary = {
  id: string;
  skill_id: string;
  target_type: string;
  target_id: string;
  priority: number;
  status: string;
  config: Record<string, string>;
  created_at?: string;
  updated_at?: string;
};

export type SkillEffectivenessSummary = {
  id: string;
  skill_id: string;
  binding_id?: string;
  subagent_id?: string;
  run_id?: string;
  issue_id?: string;
  outcome: string;
  quality_impact: string;
  rework_reduced: boolean;
  duration_seconds: number;
  findings: string[];
  created_at?: string;
};

export type SkillRecommendationCandidateSummary = {
  skill_id: string;
  name: string;
  source: string;
  score: number;
  reasons: string[];
  risks: string[];
};

export type SkillRecommendationSummary = {
  id: string;
  issue_id?: string;
  role: string;
  task_type?: string;
  risk_level: string;
  candidates: SkillRecommendationCandidateSummary[];
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
  last_deployment?: ResourceDeploymentRefSummary;
};

export type ResourceDeploymentRefSummary = {
  id: string;
  resource_id: string;
  kind: string;
  deployment_id?: string;
  execution_id?: string;
  release_id?: string;
  environment?: string;
  mode?: string;
  status: string;
  decision: string;
  recorded_at?: string;
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

export type PostDeploymentVerificationSummary = {
  id: string;
  execution_id?: string;
  deployment_id?: string;
  release_id?: string;
  environment?: string;
  status: string;
  decision: string;
  reasons: string[];
  history_id?: string;
  history_decision?: string;
  monitor_summary_id?: string;
  monitor_decision?: string;
  smoke_decision?: string;
  rollback_required: boolean;
  risk_handoff_recommended: boolean;
  risk_source_type?: string;
  risk_source_id?: string;
  evidence_ids: string[];
  created_at?: string;
};

export type DeploymentRehearsalSummary = {
  id: string;
  candidate_id?: string;
  deployment_id?: string;
  execution_id?: string;
  release_id?: string;
  environment?: string;
  status: string;
  decision: string;
  reasons: string[];
  timeline: DeploymentRehearsalTimelineItem[];
  monitor_summary_id?: string;
  monitor_status?: string;
  monitor_decision?: string;
  rollback_execution_id?: string;
  rollback_status?: string;
  rollback_decision?: string;
  evidence_ids: string[];
  created_at?: string;
};

export type DeploymentRehearsalTimelineItem = {
  type: string;
  id: string;
  status: string;
  decision: string;
  detail?: string;
  evidence_ids: string[];
  created_at?: string;
};

export type ReleaseAdmissionSummary = {
  id: string;
  rehearsal_id?: string;
  candidate_id?: string;
  deployment_id?: string;
  execution_id?: string;
  environment?: string;
  status: string;
  decision: string;
  reasons: string[];
  signals: ReleaseAdmissionSignalSummary[];
  policy_id?: string;
  policy_version?: string;
  policy_source?: string;
  matched_rules: AdmissionRuleMatchSummary[];
  policy_decision?: ReleaseAdmissionPolicyDecisionSummary;
  evidence_ids: string[];
  created_at?: string;
};

export type ReleaseAdmissionSignalSummary = {
  type: string;
  id?: string;
  status: string;
  decision: string;
  severity?: string;
  reason?: string;
};

export type ReleaseAdmissionPolicyPackSummary = {
  id: string;
  version?: string;
  source?: string;
  default_environment?: string;
  environment_count: number;
  rule_count: number;
  rules: ReleaseAdmissionPolicyRuleSummary[];
};

export type ReleaseAdmissionPolicyRuleSummary = {
  id: string;
  signal_type?: string;
  effect: string;
  reason: string;
};

export type ReleaseAdmissionPolicyDecisionSummary = {
  policy_id: string;
  policy_version?: string;
  policy_source?: string;
  environment?: string;
  status: string;
  decision: string;
  reasons: string[];
  matched_rule_count: number;
  blocked: boolean;
  manual_required: boolean;
};

export type AdmissionRuleMatchSummary = {
  policy_id: string;
  rule_id: string;
  signal_type?: string;
  signal_id?: string;
  status?: string;
  decision?: string;
  effect: string;
  reason: string;
};

export type RehearsalSchedulerRunSummary = {
  id: string;
  trigger: string;
  requested_by?: string;
  candidate_id?: string;
  deployment_id?: string;
  execution_id?: string;
  environment?: string;
  status: string;
  decision: string;
  reasons: string[];
  max_targets: number;
  skip_admission: boolean;
  created_count: number;
  skipped_count: number;
  blocked_count: number;
  manual_count: number;
  targets: RehearsalSchedulerTargetSummary[];
  rehearsal_ids: string[];
  admission_ids: string[];
  evidence_ids: string[];
  started_at?: string;
  finished_at?: string;
};

export type RehearsalSchedulerTargetSummary = {
  type: string;
  candidate_id?: string;
  deployment_id?: string;
  execution_id?: string;
  environment?: string;
  status: string;
  decision: string;
  reason?: string;
  rehearsal_id?: string;
  admission_id?: string;
};

export type DeploymentRiskHandoffSummary = {
  id: string;
  source_type: string;
  source_id: string;
  status: string;
  decision: string;
  failure_class: string;
  signal_id?: string;
  bug_candidate_id?: string;
  repair_plan_id?: string;
  evidence_refs: string[];
  reasons: string[];
  review_required: boolean;
  review_id?: string;
  reviewed_at?: string;
  reviewed_by?: string;
  review_decision?: string;
  review_reason?: string;
  review_next_step?: string;
  created_at?: string;
};

export type DeploymentRiskReviewQueueItemSummary = {
  handoff_id: string;
  source_type: string;
  source_id: string;
  status: string;
  decision: string;
  failure_class: string;
  review_required: boolean;
  review_id?: string;
  review_decision?: string;
  review_next_step?: string;
  signal_id?: string;
  bug_candidate_id?: string;
  repair_plan_id?: string;
  evidence_refs: string[];
  reasons: string[];
  created_at?: string;
  reviewed_at?: string;
};

export type DeploymentRiskReviewSummary = {
  id: string;
  handoff_id: string;
  source_type: string;
  source_id: string;
  decision: string;
  status: string;
  reviewer_id?: string;
  reason?: string;
  next_step?: string;
  failure_class?: string;
  signal_id?: string;
  bug_candidate_id?: string;
  repair_plan_id?: string;
  evidence_refs: string[];
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
  commit_before?: string;
  commit_after?: string;
  commit_changed?: boolean;
  changed_files?: string[];
  diff_summary_path?: string;
  managed_by?: string;
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

export type QualityReportSummary = {
  id: string;
  task_id: string;
  status: string;
  review_status: string;
  findings_count: number;
  check_count: number;
  changed_files: string[];
  diff_summary_path?: string;
  checks: QualityCheck[];
  findings: QualityFinding[];
  created_at?: string;
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
  type: string;
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

export type MemoryRecordSummary = {
  id: string;
  kind: string;
  summary: string;
  tags: string[];
  source?: string;
  scope?: string;
  scopes: string[];
  confidence: number;
  score: number;
  created_by?: string;
  created_at?: string;
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

export type OperationsAuditExportSummary = {
  id: string;
  generated_at?: string;
  format: string;
  timeline_item_count: number;
  post_deployment_verification_count: number;
  resource_deployment_ref_count: number;
  evidence_ref_count: number;
  attention_item_count: number;
  risk_handoff_recommended_count: number;
  redaction_applied: boolean;
  by_type: Record<string, number>;
  evidence_refs: string[];
};

export type DecisionLedgerEntrySummary = {
  id: string;
  source_type: string;
  source_id: string;
  parent_ref?: string;
  environment?: string;
  status: string;
  decision: string;
  reasons: string[];
  rule_refs: string[];
  evidence_refs: string[];
  created_at?: string;
};

export type DecisionLedgerReportSummary = {
  id: string;
  generated_at?: string;
  entry_count: number;
  evidence_ref_count: number;
  attention_count: number;
  redaction_applied: boolean;
  by_source_type: Record<string, number>;
  by_status: Record<string, number>;
  by_decision: Record<string, number>;
  entries: DecisionLedgerEntrySummary[];
};

export type ProviderWriteProofSummary = {
  id: string;
  operation_type: string;
  operation_id: string;
  provider?: string;
  environment?: string;
  mode?: string;
  status: string;
  decision: string;
  reasons: string[];
  source_ref?: string;
  dry_run: boolean;
  write_enabled: boolean;
  approval_id?: string;
  approval_consumed: boolean;
  approval_required: boolean;
  approval_satisfied: boolean;
  secret_ref_status?: string;
  provider_evidence_refs: string[];
  least_privilege?: string;
  replay_guard?: string;
  created_at?: string;
};

export type ProviderWriteProofReportSummary = {
  id: string;
  generated_at?: string;
  proof_count: number;
  blocked_count: number;
  manual_required_count: number;
  redaction_applied: boolean;
  by_operation_type: Record<string, number>;
  by_provider: Record<string, number>;
  by_environment: Record<string, number>;
  by_status: Record<string, number>;
  by_decision: Record<string, number>;
  proofs: ProviderWriteProofSummary[];
};

export type WriteAdmissionSummary = {
  id: string;
  proof_id?: string;
  proof_decision?: string;
  operation_type: string;
  operation_id: string;
  provider?: string;
  environment?: string;
  mode?: string;
  status: string;
  decision: string;
  reasons: string[];
  rule_refs: string[];
  source_ref?: string;
  dry_run: boolean;
  write_enabled: boolean;
  rehearsal_allowed: boolean;
  approval_required: boolean;
  approval_satisfied: boolean;
  secret_ref_status?: string;
  provider_evidence_refs: string[];
  provider_requirement_id?: string;
  provider_requirement_version?: string;
  provider_requirement_refs: string[];
  least_privilege?: string;
  replay_guard?: string;
  created_at?: string;
};

export type WriteAdmissionReportSummary = {
  id: string;
  generated_at?: string;
  policy_id: string;
  policy_version?: string;
  target: string;
  entry_count: number;
  ready_count: number;
  blocked_count: number;
  manual_required_count: number;
  rehearsal_only_count: number;
  redaction_applied: boolean;
  by_operation_type: Record<string, number>;
  by_provider: Record<string, number>;
  by_status: Record<string, number>;
  by_decision: Record<string, number>;
  entries: WriteAdmissionSummary[];
};

export type ProviderProofRequirementSummary = {
  id: string;
  provider: string;
  operation_type: string;
  status: string;
  decision: string;
  required_secret_ref_status?: string;
  require_evidence: boolean;
  require_approval: boolean;
  require_write_switch: boolean;
  production_review_required: boolean;
  least_privilege_scopes: string[];
  replay_guard?: string;
  rule_refs: string[];
};

export type ProviderProofRequirementReportSummary = {
  id: string;
  generated_at?: string;
  policy_id: string;
  policy_version?: string;
  requirement_count: number;
  by_provider: Record<string, number>;
  by_operation_type: Record<string, number>;
  requirements: ProviderProofRequirementSummary[];
};

export type RemoteExecutionRehearsalSummary = {
  id: string;
  source_admission_id?: string;
  source_proof_id?: string;
  operation_type: string;
  operation_id: string;
  provider?: string;
  environment?: string;
  mode?: string;
  status: string;
  decision: string;
  reasons: string[];
  rule_refs: string[];
  evidence_refs: string[];
  provider_requirement_id?: string;
  target_check_count: number;
  command_check_count: number;
  auth_ref_check_count: number;
  rollback_required: boolean;
  rollback_decision?: string;
  created_at?: string;
  finished_at?: string;
};

export type RemoteExecutionRehearsalReportSummary = {
  id: string;
  generated_at?: string;
  rehearsal_count: number;
  completed_count: number;
  blocked_count: number;
  manual_count: number;
  by_provider: Record<string, number>;
  by_status: Record<string, number>;
  by_decision: Record<string, number>;
  rehearsals: RemoteExecutionRehearsalSummary[];
};

export type WriteReviewPacketSummary = {
  id: string;
  admission_id?: string;
  proof_id?: string;
  operation_type: string;
  operation_id: string;
  provider?: string;
  environment?: string;
  mode?: string;
  status: string;
  decision: string;
  reasons: string[];
  rule_refs: string[];
  evidence_refs: string[];
  provider_requirement_id?: string;
  remote_rehearsal_id?: string;
  remote_rehearsal_status?: string;
  remote_rehearsal_decision?: string;
  queue_item_ids: string[];
  queue_decisions: string[];
  markdown?: string;
  created_at?: string;
};

export type WriteReviewPacketReportSummary = {
  id: string;
  generated_at?: string;
  packet_count: number;
  ready_count: number;
  blocked_count: number;
  manual_required_count: number;
  by_operation_type: Record<string, number>;
  by_provider: Record<string, number>;
  by_status: Record<string, number>;
  by_decision: Record<string, number>;
  packets: WriteReviewPacketSummary[];
};

export type WriteExecutionPlanSummary = {
  id: string;
  review_packet_id?: string;
  operation_type?: string;
  operation_id?: string;
  provider?: string;
  environment?: string;
  mode: string;
  status: string;
  decision: string;
  reasons: string[];
  rule_refs: string[];
  evidence_refs: string[];
  approval_id?: string;
  requested_by?: string;
  apply_allowed: boolean;
  external_write_performed: boolean;
  created_at?: string;
};

export type WriteExecutionPlanReportSummary = {
  id: string;
  generated_at?: string;
  plan_count: number;
  ready_count: number;
  planned_count: number;
  blocked_count: number;
  manual_required_count: number;
  external_write_count: number;
  by_mode: Record<string, number>;
  by_status: Record<string, number>;
  by_decision: Record<string, number>;
  plans: WriteExecutionPlanSummary[];
};

export type WriteAdapterGuardSummary = {
  name: string;
  status: string;
  decision: string;
  reason?: string;
};

export type WriteAdapterSandboxSummary = {
  resource_id?: string;
  environment?: string;
  provider?: string;
  host_status?: string;
  command?: string;
  allowlist: string[];
  status: string;
  decision: string;
  reason?: string;
  preview_only: boolean;
  no_remote_write: boolean;
};

export type WriteAdapterRollbackBindingSummary = {
  deployment_id?: string;
  required: boolean;
  status?: string;
  decision?: string;
  reason?: string;
  plan_ref?: string;
  runbook_ref?: string;
  action_count?: number;
  step_count?: number;
};

export type WriteAdapterExecutionSummary = {
  id: string;
  execution_plan_id?: string;
  review_packet_id?: string;
  operation_type?: string;
  operation_id?: string;
  provider?: string;
  environment?: string;
  adapter_id?: string;
  mode: string;
  status: string;
  decision: string;
  reasons: string[];
  rule_refs: string[];
  evidence_refs: string[];
  guard_results: WriteAdapterGuardSummary[];
  sandbox_results: WriteAdapterSandboxSummary[];
  rollback_binding?: WriteAdapterRollbackBindingSummary;
  apply_allowed: boolean;
  external_write_attempted: boolean;
  external_write_performed: boolean;
  created_at?: string;
  finished_at?: string;
};

export type WriteAdapterExecutionReportSummary = {
  id: string;
  generated_at?: string;
  execution_count: number;
  completed_count: number;
  blocked_count: number;
  manual_required_count: number;
  sandbox_result_count: number;
  rollback_bound_count: number;
  external_attempt_count: number;
  external_write_count: number;
  by_adapter: Record<string, number>;
  by_mode: Record<string, number>;
  by_status: Record<string, number>;
  by_decision: Record<string, number>;
  executions: WriteAdapterExecutionSummary[];
};

export type WriteAdapterRecoverySummary = {
  id: string;
  execution_id: string;
  execution_plan_id?: string;
  operation_type?: string;
  operation_id?: string;
  provider?: string;
  environment?: string;
  adapter_id?: string;
  mode?: string;
  source_status: string;
  source_decision: string;
  status: string;
  decision: string;
  failure_class: string;
  recovery_action: string;
  repair_allowed: boolean;
  retry_allowed: boolean;
  handoff_required: boolean;
  review_required: boolean;
  reasons: string[];
  evidence_refs: string[];
  created_at?: string;
};

export type WriteAdapterRecoveryReportSummary = {
  id: string;
  generated_at?: string;
  recovery_count: number;
  open_count: number;
  repair_count: number;
  retry_count: number;
  handoff_count: number;
  by_adapter: Record<string, number>;
  by_status: Record<string, number>;
  by_decision: Record<string, number>;
  by_failure: Record<string, number>;
  by_action: Record<string, number>;
  recoveries: WriteAdapterRecoverySummary[];
};

export type ControlLoopQueueItemSummary = {
  id: string;
  status: string;
  decision: string;
  trigger: string;
  requested_by?: string;
  retry_budget: number;
  attempt_count: number;
  steps: string[];
  environment?: string;
  maintenance_window?: string;
  due_at?: string;
  admission_id?: string;
  remote_rehearsal_id?: string;
  review_packet_id?: string;
  adapter_recovery_id?: string;
  run_id?: string;
  reasons: string[];
  created_at?: string;
  updated_at?: string;
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
  projects: ProjectSummary[];
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
    skills: number;
  };
  requirements: RequirementSummary[];
  issues: IssueNode[];
  schedule: ScheduleItem[];
  subagent_backlog: SubagentBacklogItem[];
  providers: ProviderSummary[];
  provider_telemetry: ProviderTelemetrySummary[];
  skills: SkillSummary[];
  skill_bindings: SkillBindingSummary[];
  skill_effectiveness: SkillEffectivenessSummary[];
  resources: ResourceSummary[];
  lifecycle_alerts: LifecycleAlertSummary[];
  maintenance_records: MaintenanceRecordSummary[];
  resource_deployment_refs: ResourceDeploymentRefSummary[];
  deployments: DeploymentSummary[];
  executions: DeploymentExecutionSummary[];
  post_deployment_histories: PostDeploymentHistorySummary[];
  post_deployment_verifications: PostDeploymentVerificationSummary[];
  rollback_executions: RollbackExecutionSummary[];
  monitor_summaries: DeploymentMonitorSummary[];
  deployment_rehearsals: DeploymentRehearsalSummary[];
  release_admissions: ReleaseAdmissionSummary[];
  release_admission_policy?: ReleaseAdmissionPolicyPackSummary;
  rehearsal_scheduler_runs: RehearsalSchedulerRunSummary[];
  deployment_risk_handoffs: DeploymentRiskHandoffSummary[];
  deployment_risk_review_queue: DeploymentRiskReviewQueueItemSummary[];
  deployment_risk_reviews: DeploymentRiskReviewSummary[];
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
  quality_reports: QualityReportSummary[];
  quality_explanations: QualityExplanation[];
  approvals: ApprovalRecordSummary[];
  audit_events: AuditEventSummary[];
  operations_audit_export?: OperationsAuditExportSummary;
  decision_ledger?: DecisionLedgerReportSummary;
  write_proofs?: ProviderWriteProofReportSummary;
  write_admissions?: WriteAdmissionReportSummary;
  provider_proof_requirements?: ProviderProofRequirementReportSummary;
  remote_execution_rehearsals?: RemoteExecutionRehearsalReportSummary;
  write_review_packets?: WriteReviewPacketReportSummary;
  write_execution_plans?: WriteExecutionPlanReportSummary;
  write_adapter_executions?: WriteAdapterExecutionReportSummary;
  write_adapter_recoveries?: WriteAdapterRecoveryReportSummary;
  control_loop_queue?: ControlLoopQueueItemSummary[];
  git_provider_plans: GitProviderPlanSummary[];
  auth_sessions: AuthSessionSummary[];
  api_tokens: APITokenSummary[];
  service_accounts: ServiceAccountSummary[];
  timeline: TimelineEvent[];
  quality: QualitySignal[];
  memory: MemorySignal[];
};
