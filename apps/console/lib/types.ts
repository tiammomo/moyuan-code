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
  blocked_reason?: string;
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
};

export type RunSummary = {
  run_id: string;
  issue_id: string;
  status: string;
  subagent_id?: string;
  runtime_id?: string;
  runtime_status?: string;
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
  updated_at?: string;
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
  };
  issues: IssueNode[];
  schedule: ScheduleItem[];
  providers: ProviderSummary[];
  resources: ResourceSummary[];
  deployments: DeploymentSummary[];
  executions: DeploymentExecutionSummary[];
  runs: RunSummary[];
  subagents: SubagentSummary[];
  quality_explanations: QualityExplanation[];
  timeline: TimelineEvent[];
  quality: QualitySignal[];
  memory: MemorySignal[];
};
