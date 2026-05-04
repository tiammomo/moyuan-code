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
  runtime?: string;
  provider?: string;
  quality?: string;
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
  };
  issues: IssueNode[];
  schedule: ScheduleItem[];
  providers: ProviderSummary[];
  resources: ResourceSummary[];
  deployments: DeploymentSummary[];
  executions: DeploymentExecutionSummary[];
  timeline: TimelineEvent[];
  quality: QualitySignal[];
  memory: MemorySignal[];
};
