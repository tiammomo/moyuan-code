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
  X,
  type LucideIcon,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState, type FormEvent } from "react";
import type {
  BatchPlanSummary,
  ConsoleSnapshot,
  EvidenceSummary,
  IssueNode,
  MemoryRecordSummary,
  ProjectSummary,
  QualityReportSummary,
  RequirementSummary,
  RunSummary,
  SkillRecommendationSummary,
  StatusTone,
} from "@/lib/types";

const detailedViews = [
  "项目",
  "需求登记",
  "Issue Graph",
  "批量执行",
  "运行",
  "质量",
  "测试验证",
  "Memory",
  "Provider",
  "技能",
  "部署",
  "执行适配器",
  "操作",
  "审计",
] as const;

type ConsoleView = (typeof detailedViews)[number];

const viewTabLabels: Record<ConsoleView, string> = {
  项目: "项目接入",
  需求登记: "需求登记",
  "Issue Graph": "Issue Graph",
  批量执行: "批量执行",
  运行: "运行时间线",
  质量: "代码质量",
  测试验证: "测试验证",
  Memory: "Memory",
  Provider: "Provider",
  技能: "技能",
  部署: "发布部署",
  执行适配器: "执行安全",
  操作: "操作证据",
  审计: "权限审计",
};

const navGroups = [
  { label: "项目工作台", icon: Boxes, views: ["项目"] },
  { label: "需求与 Issue", icon: Network, views: ["需求登记", "Issue Graph", "批量执行"] },
  { label: "执行与恢复", icon: TerminalSquare, views: ["运行", "操作"] },
  { label: "质量与验证", icon: ShieldCheck, views: ["质量", "测试验证"] },
  { label: "发布与部署", icon: Rocket, views: ["部署", "执行适配器"] },
  { label: "AI 能力", icon: Sparkles, views: ["Provider", "技能", "Memory"] },
  { label: "权限与审计", icon: Lock, views: ["审计"] },
] as const satisfies readonly { label: string; icon: LucideIcon; views: readonly ConsoleView[] }[];

type ConsoleNavGroup = (typeof navGroups)[number];

export function ConsoleWorkbench({ snapshot }: { snapshot: ConsoleSnapshot }) {
  const router = useRouter();
  const [activeView, setActiveView] = useState<ConsoleView>("项目");
  const [selectedIssueID, setSelectedIssueID] = useState(snapshot.issues[0]?.id ?? "");
  const [selectedOperationID, setSelectedOperationID] = useState(snapshot.operation_history[0]?.id ?? "");
  const [requirementText, setRequirementText] = useState("");
  const [requirementState, setRequirementState] = useState<RequirementSubmitState>({ status: "idle" });
  const [clarificationAnswer, setClarificationAnswer] = useState("");
  const [clarificationModalOpen, setClarificationModalOpen] = useState(false);
  const [projectForm, setProjectForm] = useState({
    mode: "local",
    localPath: snapshot.project.root,
    remoteURL: "",
    destPath: "",
    provider: "",
  });
  const [projectModalOpen, setProjectModalOpen] = useState(false);
  const [projectActionState, setProjectActionState] = useState<ActionState>({ status: "idle" });
  const [schemaErrors, setSchemaErrors] = useState<Record<string, string[]>>({});
  const [recoveryArtifactState, setRecoveryArtifactState] = useState<Record<string, RecoveryArtifactState>>({});
  const [visualActionState, setVisualActionState] = useState<Record<string, VisualActionState>>({});
  const [deploymentActionState, setDeploymentActionState] = useState<DeploymentActionState>({ status: "idle" });
  const [visualPlanForm, setVisualPlanForm] = useState({
    diagramType: "architecture",
    title: "系统架构图",
    scope: "当前项目控制台与后端执行链路",
    size: "3072x2048",
  });
  const [visualPlanModalOpen, setVisualPlanModalOpen] = useState(false);
  const [approvalForm, setApprovalForm] = useState({ decidedBy: "console-owner", reason: "已在控制台复核" });
  const [approvalCreateForm, setApprovalCreateForm] = useState({
    targetType: "deployment_execution",
    targetID: snapshot.executions[0]?.id ?? "",
    action: "production_execute",
    riskLevel: "high",
    requestedBy: "console-owner",
    reason: "生产动作需要人工批准",
    metadata: "",
  });
  const [approvalCreateModalOpen, setApprovalCreateModalOpen] = useState(false);
  const [approvalActionState, setApprovalActionState] = useState<Record<string, ActionState>>({});
  const [approvalDecisionModal, setApprovalDecisionModal] = useState<{ approvalID: string; decision: "approved" | "rejected" } | null>(null);
  const [sessionForm, setSessionForm] = useState({ userID: "developer", displayName: "Developer", roles: "developer" });
  const [sessionModalOpen, setSessionModalOpen] = useState(false);
  const [sessionRevokeForm, setSessionRevokeForm] = useState({ actorID: "security-owner", reason: "控制台撤销访问" });
  const [sessionRevokeModalID, setSessionRevokeModalID] = useState<string | null>(null);
  const [tokenForm, setTokenForm] = useState({ name: "console-token", actorID: "developer", scopes: "project:read" });
  const [tokenModalOpen, setTokenModalOpen] = useState(false);
  const [tokenRevokeForm, setTokenRevokeForm] = useState({ actorID: "security-owner", reason: "控制台撤销 Token" });
  const [tokenRevokeModalID, setTokenRevokeModalID] = useState<string | null>(null);
  const [serviceAccountForm, setServiceAccountForm] = useState({ id: "", name: "Release Bot", roles: "release_bot,deploy_executor" });
  const [serviceAccountModalOpen, setServiceAccountModalOpen] = useState(false);
  const [accessActionState, setAccessActionState] = useState<Record<string, ActionState>>({});
  const [resourceForm, setResourceForm] = useState({ actorID: "ops-owner", expiresAt: "2099-01-01", reason: "控制台维护" });
  const [resourceCreateForm, setResourceCreateForm] = useState({
    id: "",
    environment: "test_dev",
    host: "",
    provider: "local_vm",
    owner: "dev",
    authRef: "env:DEV_SERVER_SSH_KEY",
    purpose: "",
    expiresAt: "",
    maintenanceWindow: "",
    cpu: "",
    memoryGB: "",
    diskGB: "",
    os: "linux",
    healthType: "manual",
    healthTarget: "",
  });
  const [resourceCreateModalOpen, setResourceCreateModalOpen] = useState(false);
  const [resourceCreateActionState, setResourceCreateActionState] = useState<ActionState>({ status: "idle" });
  const [resourceActionModal, setResourceActionModal] = useState<{ resourceID: string; action: "renew" | "retire" } | null>(null);
  const [resourceActionState, setResourceActionState] = useState<Record<string, ActionState>>({});
  const [gitActionState, setGitActionState] = useState<Record<string, ActionState>>({});
  const [gitCreateApproved, setGitCreateApproved] = useState(false);
  const [gitCreateApprovalID, setGitCreateApprovalID] = useState("");
  const [gitCreateModalPlanID, setGitCreateModalPlanID] = useState<string | null>(null);
  const [providerRouteForm, setProviderRouteForm] = useState({
    role: "frontend",
    taskType: "requirement_planning",
    outputType: "code",
    modelStrategy: "default",
    requiresRepoEdit: true,
    includesSensitiveCode: false,
    includesProjectMemory: true,
  });
  const [providerRoute, setProviderRoute] = useState<ProviderRouteDecision | null>(null);
  const [providerRouteState, setProviderRouteState] = useState<ActionState>({ status: "idle" });
  const [controlLoopActionState, setControlLoopActionState] = useState<ActionState>({ status: "idle" });
  const [repairReviewForm, setRepairReviewForm] = useState({ reviewerID: "qa-owner", reason: "已在控制台复核" });
  const [repairReviewModal, setRepairReviewModal] = useState<{ candidateID: string; decision: "approved" | "rejected" } | null>(null);
  const [repairActionState, setRepairActionState] = useState<Record<string, ActionState>>({});
  const [operationRepairCreateState, setOperationRepairCreateState] = useState<ActionState>({ status: "idle" });
  const [issueActionState, setIssueActionState] = useState<Record<string, ActionState>>({});
  const [batchPlanForm, setBatchPlanForm] = useState({
    epicID: defaultBatchEpicID(snapshot),
    mode: "dry_run",
    maxParallel: "2",
    requestedBy: "console-owner",
  });
  const [batchPlanModalOpen, setBatchPlanModalOpen] = useState(false);
  const [batchActionState, setBatchActionState] = useState<Record<string, ActionState>>({});
  const [releaseProviderForm, setReleaseProviderForm] = useState({
    releaseID: snapshot.deployments[0]?.release_id ?? "",
    approved: false,
    approvalID: "",
  });
  const [releaseProviderModalOpen, setReleaseProviderModalOpen] = useState(false);
  const [releaseProviderActionState, setReleaseProviderActionState] = useState<ActionState>({ status: "idle" });
  const [deploymentPlanForm, setDeploymentPlanForm] = useState({
    releaseID: snapshot.deployments[0]?.release_id ?? snapshot.release_candidates[0]?.version ?? snapshot.release_candidates[0]?.id ?? "",
    environment: "test_dev",
    resourceIDs: snapshot.resources.map((resource) => resource.id).join(","),
    approved: false,
  });
  const [deploymentPlanModalOpen, setDeploymentPlanModalOpen] = useState(false);
  const [deploymentExecuteForm, setDeploymentExecuteForm] = useState({
    deploymentID: snapshot.deployments[0]?.id ?? "",
    environment: snapshot.deployments[0]?.environment ?? "test_dev",
    mode: "dry_run",
    approved: false,
    approvalID: "",
    commands: "",
  });
  const [deploymentExecuteModalOpen, setDeploymentExecuteModalOpen] = useState(false);
  const [resourceScanForm, setResourceScanForm] = useState({ environment: "test_dev", resourceIDs: "", approved: false });
  const [resourceScanModal, setResourceScanModal] = useState<"maintenance" | "lifecycle" | "health" | null>(null);
  const [resourceDisableModalID, setResourceDisableModalID] = useState<string | null>(null);
  const [deploymentRiskReviewForm, setDeploymentRiskReviewForm] = useState({
    reviewerID: "ops-owner",
    reason: "控制台风险复核",
    nextStep: "repair_attempt",
  });
  const [deploymentRiskReviewModal, setDeploymentRiskReviewModal] = useState<{ handoffID: string; decision: "approved" | "rejected" } | null>(null);
  const [controlQueueForm, setControlQueueForm] = useState({
    trigger: "console_manual",
    requestedBy: "console-owner",
    idempotencyKey: "",
    retryBudget: "1",
    steps: "remote_rehearsal,write_review_packet,write_execution_plan,write_adapter_execution",
    environment: "test_dev",
    resourceIDs: "",
    deploymentExecutionID: snapshot.executions[0]?.id ?? "",
    maintenanceWindow: "always",
    dueAt: "",
    priority: "50",
    admissionID: snapshot.release_admissions[0]?.id ?? "",
    remoteRehearsalID: snapshot.remote_execution_rehearsals?.rehearsals[0]?.id ?? "",
    reviewPacketID: snapshot.write_review_packets?.packets[0]?.id ?? "",
    adapterRecoveryID: snapshot.write_adapter_recoveries?.recoveries[0]?.id ?? "",
  });
  const [controlQueueModalOpen, setControlQueueModalOpen] = useState(false);
  const [controlQueueRunForm, setControlQueueRunForm] = useState({ status: "queued", environment: "", maxItems: "3" });
  const [controlQueueRunModalOpen, setControlQueueRunModalOpen] = useState(false);
  const [adapterActionState, setAdapterActionState] = useState<ActionState>({ status: "idle" });
  const [writePipelineModal, setWritePipelineModal] = useState<"remoteRehearsal" | "reviewPacket" | "executionPlan" | "adapterExecution" | null>(null);
  const [remoteRehearsalForm, setRemoteRehearsalForm] = useState({
    admissionID: snapshot.write_admissions?.entries[0]?.id ?? snapshot.release_admissions[0]?.id ?? "",
    executionID: snapshot.executions[0]?.id ?? "",
    provider: "",
    environment: "test_dev",
    status: "",
    decision: "",
    limit: "20",
  });
  const [writeReviewPacketForm, setWriteReviewPacketForm] = useState({
    admissionID: snapshot.write_admissions?.entries[0]?.id ?? "",
    operationType: "deployment_execution",
    operationID: snapshot.executions[0]?.id ?? "",
    provider: "",
    environment: "test_dev",
    status: "",
    decision: "",
    limit: "20",
  });
  const [writeExecutionPlanForm, setWriteExecutionPlanForm] = useState({
    reviewPacketID: snapshot.write_review_packets?.packets[0]?.id ?? "",
    mode: "preview",
    approvalID: "",
    requestedBy: "console-owner",
    status: "",
    decision: "",
    limit: "20",
  });
  const [writeAdapterExecutionForm, setWriteAdapterExecutionForm] = useState({
    executionPlanID: snapshot.write_execution_plans?.plans[0]?.id ?? "",
    mode: "preview",
    adapterID: "ssh_production_preview",
    status: "",
    decision: "",
    limit: "20",
  });
  const [providerForm, setProviderForm] = useState({
    id: "",
    name: "",
    vendor: "openai",
    apiType: "openai",
    baseURL: "",
    authRef: "",
    runtimeID: "",
    model: "",
    useCases: "frontend,backend,review",
    enabled: true,
    nativeRuntime: false,
    allowSensitiveCode: false,
    allowProjectMemory: true,
    allowProductionContext: false,
  });
  const [providerModalOpen, setProviderModalOpen] = useState(false);
  const [providerOpsForm, setProviderOpsForm] = useState({ providerID: "", includeDisabled: false, probe: true, approved: false, probeTimeoutMS: "1200" });
  const [providerOpsModalOpen, setProviderOpsModalOpen] = useState(false);
  const [providerOpsSnapshotForm, setProviderOpsSnapshotForm] = useState({
    healthStatus: "ok",
    healthReason: "manual console update",
    quotaStatus: "ok",
    limitTokens: "",
    usedTokens: "",
    remainingTokens: "",
    costStatus: "ok",
    estimatedAmount: "",
    budgetAmount: "",
    usageWindow: "daily",
    requests: "",
    inputTokens: "",
    outputTokens: "",
    totalTokens: "",
  });
  const [providerOpsSnapshotModalID, setProviderOpsSnapshotModalID] = useState<string | null>(null);
  const [providerDisableModalID, setProviderDisableModalID] = useState<string | null>(null);
  const [providerActionState, setProviderActionState] = useState<Record<string, ActionState>>({});
  const [skillForm, setSkillForm] = useState({
    id: "",
    name: "",
    source: "local",
    version: "",
    description: "",
    enabled: true,
    riskLevel: "medium",
    compatibleRoles: "frontend,backend,quality",
    tags: "",
    requiredTools: "",
    authRef: "",
  });
  const [skillModalOpen, setSkillModalOpen] = useState(false);
  const [skillRecommendationForm, setSkillRecommendationForm] = useState({
    issueID: snapshot.issues[0]?.id ?? "",
    role: "frontend",
    taskType: "coding",
    riskLevel: "medium",
    limit: "5",
  });
  const [skillRecommendationModalOpen, setSkillRecommendationModalOpen] = useState(false);
  const [skillRecommendation, setSkillRecommendation] = useState<SkillRecommendationSummary | null>(null);
  const [skillBindingForm, setSkillBindingForm] = useState({
    id: "",
    skillID: snapshot.skills[0]?.id ?? "",
    targetType: "role",
    targetID: "frontend",
    priority: "80",
    status: "active",
    config: "",
  });
  const [skillBindingModalOpen, setSkillBindingModalOpen] = useState(false);
  const [skillEffectivenessForm, setSkillEffectivenessForm] = useState({
    id: "",
    skillID: snapshot.skills[0]?.id ?? "",
    bindingID: snapshot.skill_bindings[0]?.id ?? "",
    subagentID: "",
    runID: "",
    issueID: snapshot.issues[0]?.id ?? "",
    outcome: "accepted",
    qualityImpact: "positive",
    reworkReduced: true,
    durationSeconds: "",
    findings: "",
  });
  const [skillEffectivenessModalOpen, setSkillEffectivenessModalOpen] = useState(false);
  const [skillDisableModalID, setSkillDisableModalID] = useState<string | null>(null);
  const [skillBindingDisableModalID, setSkillBindingDisableModalID] = useState<string | null>(null);
  const [skillActionState, setSkillActionState] = useState<Record<string, ActionState>>({});
  const [memorySearchForm, setMemorySearchForm] = useState({ query: "", limit: "10" });
  const [memorySearchModalOpen, setMemorySearchModalOpen] = useState(false);
  const [memorySearchResults, setMemorySearchResults] = useState<MemoryRecordSummary[]>([]);
  const [memoryActionState, setMemoryActionState] = useState<ActionState>({ status: "idle" });
  const [qualityDetailReportID, setQualityDetailReportID] = useState<string | null>(null);
  const selectedIssue = snapshot.issues.find((issue) => issue.id === selectedIssueID) ?? snapshot.issues[0];
  const selectedOperation = snapshot.operation_history.find((operation) => operation.id === selectedOperationID) ?? snapshot.operation_history[0];
  const selectedQualityReport = qualityDetailReportID ? snapshot.quality_reports.find((report) => report.id === qualityDetailReportID) : undefined;
  const selectedQualityExplanation = qualityDetailReportID ? snapshot.quality_explanations.find((explanation) => explanation.report_id === qualityDetailReportID) : undefined;
  const operationDetailByID = useMemo(() => new Map(snapshot.operation_details.map((detail) => [detail.id, detail])), [snapshot.operation_details]);
  const selectedOperationDetail = selectedOperation ? operationDetailByID.get(selectedOperation.id) : undefined;
  const evidenceByID = useMemo(() => new Map(snapshot.evidence.map((record) => [record.id, record])), [snapshot.evidence]);
  const selectedEvidenceRecords = useMemo(
    () =>
      selectedOperationDetail?.evidence.length
        ? selectedOperationDetail.evidence
        : selectedOperation?.evidence_ids.map((id) => evidenceByID.get(id)).filter((record): record is EvidenceSummary => Boolean(record)) ?? [],
    [evidenceByID, selectedOperation, selectedOperationDetail],
  );
  const issueGraphLayout = useMemo(() => layoutIssueGraph(snapshot.issues), [snapshot.issues]);
  const graphRelations = useMemo(() => relatedIssueSets(snapshot.issues, selectedIssueID), [selectedIssueID, snapshot.issues]);
  const issueByID = useMemo(() => new Map(snapshot.issues.map((issue) => [issue.id, issue])), [snapshot.issues]);
  const requirementByEpicID = useMemo(() => new Map(snapshot.requirements.map((requirement) => [requirement.epic_id, requirement])), [snapshot.requirements]);
  const selectedDependencyIDs = useMemo(() => {
    if (!selectedIssue) return [];
    const directDependencies = selectedIssue.depends_on ?? [];
    if (directDependencies.length > 0) {
      return directDependencies;
    }
    return issueGraphLayout.edges.filter((edge) => edge.to === selectedIssue.id).map((edge) => edge.from);
  }, [issueGraphLayout.edges, selectedIssue]);
  const runsByIssue = useMemo(() => groupRunsByIssue(snapshot.runs), [snapshot.runs]);
  const orderedRequirements = useMemo(() => orderRequirements(snapshot.requirements), [snapshot.requirements]);
  const completedRequirements = useMemo(() => snapshot.requirements.filter((requirement) => requirement.status === "completed"), [snapshot.requirements]);
  const latestDeployment = snapshot.deployments[0];
  const latestVerification = snapshot.post_deployment_verifications[0];
  const latestResourceDeploymentRef = snapshot.resource_deployment_refs[0];
  const latestRollbackCandidate = snapshot.executions.find((execution) => execution.rollback_required);
  const latestMonitorSummary = snapshot.monitor_summaries[0];
  const latestRehearsal = snapshot.deployment_rehearsals[0];
  const latestAdmission = snapshot.release_admissions[0];
  const latestSchedulerRun = snapshot.rehearsal_scheduler_runs[0];
  const latestRiskHandoff = snapshot.deployment_risk_handoffs[0];
  const latestRiskReviewQueueItem = snapshot.deployment_risk_review_queue[0];
  const latestRiskReview = snapshot.deployment_risk_reviews[0];
  const hasDeploymentOpsHistory = Boolean(
    latestMonitorSummary ||
      latestRehearsal ||
      latestAdmission ||
      latestSchedulerRun ||
      latestRiskHandoff ||
      latestRiskReviewQueueItem ||
      latestRiskReview ||
      latestVerification ||
      snapshot.rollback_executions.length > 0,
  );

  useEffect(() => {
    setSelectedIssueID((currentIssueID) => {
      if (currentIssueID && snapshot.issues.some((issue) => issue.id === currentIssueID)) {
        return currentIssueID;
      }
      return snapshot.issues[0]?.id ?? "";
    });
  }, [snapshot.issues]);

  useEffect(() => {
    const nextEpicID = defaultBatchEpicID(snapshot);
    setBatchPlanForm((current) => {
      if (current.epicID === nextEpicID) {
        return current;
      }
      if (current.epicID.trim() && snapshot.requirements.some((requirement) => requirement.epic_id === current.epicID.trim() && isActiveBatchRequirement(requirement))) {
        return current;
      }
      return { ...current, epicID: nextEpicID };
    });
  }, [snapshot]);
  const operationsAuditExport = snapshot.operations_audit_export;
  const decisionLedger = snapshot.decision_ledger;
  const writeProofReport = snapshot.write_proofs;
  const writeProofs = writeProofReport?.proofs ?? [];
  const writeAdmissionReport = snapshot.write_admissions;
  const writeAdmissions = writeAdmissionReport?.entries ?? [];
  const providerProofRequirementReport = snapshot.provider_proof_requirements;
  const providerProofRequirements = providerProofRequirementReport?.requirements ?? [];
  const remoteRehearsalReport = snapshot.remote_execution_rehearsals;
  const remoteRehearsals = remoteRehearsalReport?.rehearsals ?? [];
  const writeReviewPacketReport = snapshot.write_review_packets;
  const writeReviewPackets = writeReviewPacketReport?.packets ?? [];
  const writeExecutionPlanReport = snapshot.write_execution_plans;
  const writeExecutionPlans = writeExecutionPlanReport?.plans ?? [];
  const writeAdapterExecutionReport = snapshot.write_adapter_executions;
  const writeAdapterExecutions = writeAdapterExecutionReport?.executions ?? [];
  const writeAdapterRecoveryReport = snapshot.write_adapter_recoveries;
  const writeAdapterRecoveries = writeAdapterRecoveryReport?.recoveries ?? [];
  const controlLoopQueue = snapshot.control_loop_queue ?? [];
  const decisionEntries = decisionLedger?.entries ?? [];
  const latestControlLoopRun = snapshot.control_loop_runs[0];
  const activeSessions = snapshot.auth_sessions.filter((session) => session.status === "active");
  const activeTokens = snapshot.api_tokens.filter((token) => token.status === "active");
  const activeServiceAccounts = snapshot.service_accounts.filter((account) => account.status === "active");
  const activeNavGroup = navGroupForView(activeView);
  const orderedProjects = useMemo(() => orderProjects(snapshot.projects, snapshot.project.id), [snapshot.project.id, snapshot.projects]);

  useEffect(() => {
    function applyHashView() {
      const raw = decodeURIComponent(window.location.hash.replace(/^#/, ""));
      const view = resolveConsoleView(raw);
      if (view) {
        setActiveView(view);
      }
    }
    applyHashView();
    window.addEventListener("hashchange", applyHashView);
    return () => window.removeEventListener("hashchange", applyHashView);
  }, []);

  function selectGroup(group: ConsoleNavGroup) {
    const nextView = group.views[0] ?? "项目";
    setActiveView(nextView);
    window.history.replaceState(null, "", `#${encodeURIComponent(group.label)}`);
  }

  function selectView(view: ConsoleView) {
    setActiveView(view);
    window.history.replaceState(null, "", `#${encodeURIComponent(view)}`);
  }

  function selectProject(projectID: string) {
    const nextProjectID = projectID.trim();
    if (!nextProjectID) {
      return;
    }
    if (nextProjectID === snapshot.project.id) {
      router.refresh();
      return;
    }
    const params = new URLSearchParams(window.location.search);
    params.set("project", nextProjectID);
    const query = params.toString();
    router.replace(`${window.location.pathname}${query ? `?${query}` : ""}${window.location.hash}`);
  }

  function setSchemaResult(key: string, errors: string[]) {
    setSchemaErrors((current) => ({ ...current, [key]: errors }));
  }

  function requireFields(key: string, fields: Array<[string, string]>) {
    const errors = fields.filter(([, value]) => value.trim() === "").map(([label]) => `${label}必填`);
    setSchemaResult(key, errors);
    return errors.length === 0;
  }

  function submitApprovalDecision(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!approvalDecisionModal) return;
    void decideApproval(approvalDecisionModal.approvalID, approvalDecisionModal.decision);
  }

  function submitRepairReview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!repairReviewModal) return;
    void reviewOperationRepairCandidate(repairReviewModal.candidateID, repairReviewModal.decision);
  }

  function submitResourceAction(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!resourceActionModal) return;
    void runResourceAction(resourceActionModal.resourceID, resourceActionModal.action);
  }

  function submitResourceDisable(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!resourceDisableModalID) return;
    void disableResource(resourceDisableModalID);
  }

  function submitGitCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!gitCreateModalPlanID) return;
    void runGitProviderAction(gitCreateModalPlanID, "create");
  }

  function submitDeploymentPlan(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void createDeploymentPlan();
  }

  function submitDeploymentExecute(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void executeDeployment();
  }

  function submitResourceScan(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!resourceScanModal) return;
    void runResourceScan(resourceScanModal);
  }

  function submitDeploymentRiskReview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!deploymentRiskReviewModal) return;
    void reviewDeploymentRiskHandoff(deploymentRiskReviewModal.handoffID, deploymentRiskReviewModal.decision);
  }

  function submitControlQueue(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void enqueueControlLoop();
  }

  function submitControlQueueRun(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void runControlQueue();
  }

  function submitProviderDisable(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!providerDisableModalID) return;
    void disableProvider(providerDisableModalID);
  }

  function submitSkillDisable(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!skillDisableModalID) return;
    void disableSkill(skillDisableModalID);
  }

  function submitSkillBindingDisable(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!skillBindingDisableModalID) return;
    void disableSkillBinding(skillBindingDisableModalID);
  }

  function submitApprovalCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void createApproval();
  }

  function submitSessionRevoke(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!sessionRevokeModalID) return;
    void revokeSession(sessionRevokeModalID);
  }

  function submitTokenRevoke(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!tokenRevokeModalID) return;
    void revokeAPIToken(tokenRevokeModalID);
  }

  function submitProviderOpsSnapshot(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!providerOpsSnapshotModalID) return;
    void updateProviderOpsSnapshot(providerOpsSnapshotModalID);
  }

  function submitBatchPlan(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void createBatchPlan();
  }

  function submitMemorySearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void searchMemory();
  }

  function submitVisualPlan(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void createVisualPlan();
  }

  async function submitRequirement(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const text = requirementText.trim();
    if (!requireFields("requirement", [["需求描述", text]])) {
      setRequirementState({ status: "error", message: "请先填写需求描述。" });
      return;
    }
    setRequirementState({ status: "planning", message: "正在规划 Issue Graph..." });
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/requirements/plan`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text }),
      });
      const payload = (await response.json()) as RequirementPlanEnvelope;
      if (!payload.requirement) {
        throw new Error(payload.error ?? "需求规划没有返回结果。");
      }
      const decision = payload.requirement.clarification_decision;
      const needsInput = Boolean(decision?.required);
      setRequirementState({
        status: needsInput ? "needs_user_input" : "planned",
        id: payload.requirement.id,
        epic: payload.requirement.epic_id,
        message: needsInput
          ? decision?.questions?.[0] ?? "这个需求还需要进一步澄清。"
          : `已生成 ${payload.requirement.issues?.length ?? 0} 个 issue。`,
      });
      router.refresh();
    } catch (error) {
      setRequirementState({ status: "error", message: error instanceof Error ? error.message : "需求规划失败。" });
    }
  }

  async function submitRequirementClarification(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const answer = clarificationAnswer.trim();
    if (!requireFields("requirementClarification", [["补充回答", answer]])) {
      setRequirementState({ status: "error", message: "请先填写补充回答。" });
      return;
    }
    const question = requirementState.message || "需求澄清";
    const text = `${requirementText.trim()}\n\n澄清问题：${question}\n补充回答：${answer}`;
    setRequirementState({ status: "planning", message: "正在带补充回答重新规划..." });
    try {
      const payload = await postJSON<RequirementPlanEnvelope>(`/api/projects/${snapshot.project.id}/requirements/plan`, { text });
      const requirement = payload.requirement;
      if (!requirement) {
        throw new Error(payload.error ?? "补充澄清没有返回规划结果。");
      }
      const decision = requirement.clarification_decision;
      const needsInput = Boolean(decision?.required);
      setRequirementText(text);
      setRequirementState({
        status: needsInput ? "needs_user_input" : "planned",
        id: requirement.id,
        epic: requirement.epic_id,
        message: needsInput ? decision?.questions?.[0] ?? "仍需进一步澄清。" : `已生成 ${requirement.issues?.length ?? 0} 个 issue。`,
      });
      setClarificationAnswer("");
      setClarificationModalOpen(false);
      router.refresh();
    } catch (error) {
      setRequirementState({ status: "error", message: error instanceof Error ? error.message : "补充澄清失败。" });
    }
  }

  async function submitProjectOnboarding(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const mode = projectForm.mode === "remote" ? "remote" : "local";
    const fields: Array<[string, string]> =
      mode === "local" ? [["本地路径", projectForm.localPath]] : [["Git 地址", projectForm.remoteURL]];
    if (!requireFields("projectOnboarding", fields)) {
      setProjectActionState({ status: "error", message: "请先补齐项目接入信息。" });
      return;
    }
    setProjectActionState({ status: "running", message: mode === "local" ? "正在接入本地项目..." : "正在 clone 并接入 Git 项目..." });
    try {
      const payload = await postJSON<ProjectCreateEnvelope>("/api/projects", {
        mode,
        local_path: mode === "local" ? projectForm.localPath.trim() : "",
        remote_url: mode === "remote" ? projectForm.remoteURL.trim() : "",
        dest_path: mode === "remote" ? projectForm.destPath.trim() : "",
        provider: mode === "remote" ? projectForm.provider.trim() : "",
      });
      const project = payload.project;
      if (!project) {
        throw new Error(payload.error ?? "项目接入没有返回项目记录。");
      }
      setProjectActionState({
        status: project.status === "active" ? "completed" : "blocked",
        id: project.id,
        message: `已接入 ${project.name || project.id}`,
      });
      setProjectModalOpen(false);
      if (project.id) {
        selectProject(project.id);
      } else {
        router.refresh();
      }
    } catch (error) {
      setProjectActionState({ status: "error", message: error instanceof Error ? error.message : "项目接入失败。" });
    }
  }

  async function loadRecoveryArtifacts(recoveryID: string) {
    setRecoveryArtifactState((current) => ({
      ...current,
      [recoveryID]: { status: "loading", message: "正在加载归档产物..." },
    }));
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/runtime-recoveries/${encodeURIComponent(recoveryID)}/artifacts`);
      const payload = (await response.json()) as RecoveryArtifactsEnvelope;
      const artifacts = payload.runtime_recovery_artifacts?.artifacts;
      if (!response.ok || !artifacts) {
        throw new Error(payload.error ?? "Runtime recovery 产物加载失败。");
      }
      setRecoveryArtifactState((current) => ({
        ...current,
        [recoveryID]: { status: "loaded", artifacts, message: `${artifacts.length} 个产物` },
      }));
    } catch (error) {
      setRecoveryArtifactState((current) => ({
        ...current,
        [recoveryID]: {
          status: "error",
          message: error instanceof Error ? error.message : "Runtime recovery 产物加载失败。",
        },
      }));
    }
  }

  async function runVisualDryRun(assetID: string) {
    setVisualActionState((current) => ({
      ...current,
      [assetID]: { status: "running", message: "正在创建渲染 dry-run..." },
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
        throw new Error(payload.error ?? "视觉渲染 dry-run 失败。");
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
          message: error instanceof Error ? error.message : "视觉渲染 dry-run 失败。",
        },
      }));
    }
  }

  async function createVisualPlan() {
    if (!requireFields("visualPlan", [["图类型", visualPlanForm.diagramType], ["标题", visualPlanForm.title]])) {
      setVisualActionState((current) => ({ ...current, plan: { status: "error", message: "请先补齐图计划信息。" } }));
      return;
    }
    setVisualActionState((current) => ({ ...current, plan: { status: "running", message: "正在创建视觉图计划..." } }));
    try {
      const payload = await postJSON<VisualPlanEnvelope>(`/api/projects/${snapshot.project.id}/visuals/diagrams/plan`, {
        diagram_type: visualPlanForm.diagramType.trim(),
        title: visualPlanForm.title.trim(),
        scope: visualPlanForm.scope.trim(),
        size: visualPlanForm.size.trim(),
      });
      const asset = payload.visual_plan?.asset;
      if (!asset) {
        throw new Error(payload.error ?? "视觉图计划没有返回资产记录。");
      }
      setVisualActionState((current) => ({
        ...current,
        plan: { status: asset.status === "route_blocked" ? "blocked" : "completed", id: asset.id, message: `${asset.diagram_type ?? "diagram"} / ${asset.status}` },
      }));
      setVisualPlanModalOpen(false);
      router.refresh();
    } catch (error) {
      setVisualActionState((current) => ({
        ...current,
        plan: { status: "error", message: error instanceof Error ? error.message : "视觉图计划创建失败。" },
      }));
    }
  }

  async function suggestRelease() {
    setDeploymentActionState({ status: "running", message: "正在生成 Release 建议..." });
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/releases/suggest`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ min_issues: 1 }),
      });
      const payload = (await response.json()) as ReleaseSuggestEnvelope;
      const release = payload.release;
      if (!response.ok || !release) {
        throw new Error(payload.error ?? "Release 建议生成失败。");
      }
      setDeploymentActionState({
        status: release.status === "suggested" ? "completed" : "blocked",
        id: release.id,
        message: `${release.decision ?? release.status ?? "Release 决策已记录"}${release.reasons?.[0] ? ` / ${release.reasons[0]}` : ""}`,
      });
      if (release.id) {
        setReleaseProviderForm((current) => ({ ...current, releaseID: release.id ?? current.releaseID }));
      }
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Release 建议生成失败。" });
    }
  }

  async function runReleaseProviderAction(action: "preview" | "publish") {
    const releaseID = releaseProviderForm.releaseID.trim();
    const schemaKey = "releaseProvider";
    const fields: Array<[string, string]> = [["Release ID", releaseID]];
    if (action === "publish" && releaseProviderForm.approved) {
      fields.push(["Approval ID", releaseProviderForm.approvalID]);
    }
    if (!requireFields(schemaKey, fields)) {
      setReleaseProviderActionState({ status: "error", message: "表单校验失败。" });
      return;
    }
    setReleaseProviderActionState({ status: "running", message: action === "publish" ? "正在发布到 Release Provider..." : "正在预览 Release Provider..." });
    try {
      const payload = await postJSON<ReleaseProviderExecutionEnvelope>(
        `/api/projects/${snapshot.project.id}/releases/${encodeURIComponent(releaseID)}/provider-${action}`,
        action === "publish"
          ? { approved: releaseProviderForm.approved, approval_id: releaseProviderForm.approvalID }
          : {},
      );
      const execution = payload.release_provider_execution;
      if (!execution) {
        throw new Error(payload.error ?? "Release Provider 操作没有返回执行记录。");
      }
      setReleaseProviderActionState({
        status: execution.status === "completed" ? "completed" : "blocked",
        id: execution.id,
        message: execution.decision ?? execution.status,
      });
      router.refresh();
    } catch (error) {
      setReleaseProviderActionState({
        status: "error",
        message: error instanceof Error ? error.message : "Release Provider 操作失败。",
      });
    }
  }

  async function createDeploymentPlan() {
    if (!requireFields("deploymentPlan", [["Release ID", deploymentPlanForm.releaseID], ["环境", deploymentPlanForm.environment]])) {
      setDeploymentActionState({ status: "error", message: "请先补齐部署计划信息。" });
      return;
    }
    setDeploymentActionState({ status: "running", message: "正在创建部署计划..." });
    try {
      const payload = await postJSON<DeploymentPlanEnvelope>(`/api/projects/${snapshot.project.id}/deployments/plan`, {
        release_id: deploymentPlanForm.releaseID.trim(),
        environment: deploymentPlanForm.environment.trim(),
        resource_ids: splitCSV(deploymentPlanForm.resourceIDs),
        approved: deploymentPlanForm.approved,
      });
      const deployment = payload.deployment;
      if (!deployment) {
        throw new Error(payload.error ?? "部署计划没有返回记录。");
      }
      setDeploymentActionState({
        status: deployment.status === "planned" ? "completed" : "blocked",
        id: deployment.id,
        message: deployment.decision ?? deployment.status ?? "部署计划已记录",
      });
      setDeploymentPlanModalOpen(false);
      setDeploymentExecuteForm((current) => ({
        ...current,
        deploymentID: deployment.id ?? current.deploymentID,
        environment: deploymentPlanForm.environment,
      }));
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "部署计划创建失败。" });
    }
  }

  async function executeDeployment() {
    const fields: Array<[string, string]> = [
      ["Deployment ID", deploymentExecuteForm.deploymentID],
      ["模式", deploymentExecuteForm.mode],
    ];
    if (deploymentExecuteForm.approved) {
      fields.push(["Approval ID", deploymentExecuteForm.approvalID]);
    }
    if (!requireFields("deploymentExecute", fields)) {
      setDeploymentActionState({ status: "error", message: "请先补齐部署执行信息。" });
      return;
    }
    setDeploymentActionState({ status: "running", message: deploymentExecuteForm.mode === "dry_run" ? "正在执行部署 Dry Run..." : "正在提交部署执行..." });
    try {
      const payload = await postJSON<DeploymentExecutionEnvelope>(
        `/api/projects/${snapshot.project.id}/deployments/${encodeURIComponent(deploymentExecuteForm.deploymentID.trim())}/execute`,
        {
          environment: deploymentExecuteForm.environment.trim(),
          mode: deploymentExecuteForm.mode.trim(),
          approved: deploymentExecuteForm.approved,
          approval_id: deploymentExecuteForm.approvalID.trim(),
          commands: splitLines(deploymentExecuteForm.commands),
        },
      );
      const execution = payload.execution;
      if (!execution) {
        throw new Error(payload.error ?? "部署执行没有返回记录。");
      }
      setDeploymentActionState({
        status: execution.status === "completed" ? "completed" : "blocked",
        id: execution.id,
        message: `${execution.decision ?? execution.status ?? "部署执行已记录"}${execution.reasons?.[0] ? ` / ${execution.reasons[0]}` : ""}`,
      });
      setDeploymentExecuteModalOpen(false);
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "部署执行失败。" });
    }
  }

  async function runDeploymentDryRun(deploymentID?: string) {
    if (!deploymentID) {
      setDeploymentActionState({ status: "error", message: "当前没有可用于 dry-run 的部署计划。" });
      return;
    }
    setDeploymentActionState({ status: "running", message: "正在创建部署 dry-run..." });
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/deployments/${encodeURIComponent(deploymentID)}/execute`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mode: "dry_run" }),
      });
      const payload = (await response.json()) as DeploymentExecutionEnvelope;
      const execution = payload.execution;
      if (!response.ok || !execution) {
        throw new Error(payload.error ?? "部署 dry-run 失败。");
      }
      setDeploymentActionState({
        status: execution.status === "completed" ? "completed" : "blocked",
        id: execution.id,
        message: `${execution.decision ?? execution.status ?? "部署决策已记录"}${execution.reasons?.[0] ? ` / ${execution.reasons[0]}` : ""}`,
      });
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "部署 dry-run 失败。" });
    }
  }

  async function runResourceHealthScan() {
    setDeploymentActionState({ status: "running", message: "正在执行 test_dev 健康扫描..." });
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/resources/health-scan`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ environment: "test_dev" }),
      });
      const payload = (await response.json()) as ResourceHealthScanEnvelope;
      const scan = payload.health_scan;
      if (!response.ok || !scan) {
        throw new Error(payload.error ?? "资源健康扫描失败。");
      }
      setDeploymentActionState({
        status: scan.status === "healthy" || scan.status === "completed" ? "completed" : "blocked",
        id: scan.id,
        message: `${scan.decision ?? scan.status ?? "健康扫描已记录"}${scan.results?.length ? ` / ${scan.results.length} 个资源` : ""}`,
      });
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "资源健康扫描失败。" });
    }
  }

  async function previewRollbackExecution(executionID?: string) {
    if (!executionID) {
      setDeploymentActionState({ status: "error", message: "当前没有可回滚的候选执行。" });
      return;
    }
    setDeploymentActionState({ status: "running", message: "正在创建回滚预览..." });
    try {
      const payload = await postJSON<RollbackExecutionEnvelope>(
        `/api/projects/${snapshot.project.id}/deployment-executions/${encodeURIComponent(executionID)}/rollback`,
        { mode: "preview" },
      );
      const rollback = payload.rollback_execution;
      if (!rollback) {
        throw new Error(payload.error ?? "回滚预览没有返回执行记录。");
      }
      setDeploymentActionState({
        status: rollback.status === "completed" ? "completed" : "blocked",
        id: rollback.id,
        message: rollback.decision ?? rollback.status ?? "回滚预览已记录",
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "回滚预览失败。" });
    }
  }

  async function summarizeDeploymentMonitor() {
    setDeploymentActionState({ status: "running", message: "正在生成监控摘要..." });
    try {
      const payload = await postJSON<MonitorSummaryEnvelope>(`/api/projects/${snapshot.project.id}/deployment-monitor-summary`, { limit: 10 });
      const summary = payload.monitor_summary;
      if (!summary) {
        throw new Error(payload.error ?? "监控摘要没有返回结果。");
      }
      setDeploymentActionState({
        status: summary.status === "healthy" || summary.status === "completed" ? "completed" : "blocked",
        id: summary.id,
        message: summary.decision ?? summary.status ?? "监控摘要已记录",
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "监控摘要生成失败。" });
    }
  }

  async function createPostDeploymentVerification() {
    const execution = snapshot.executions[0];
    if (!execution) {
      setDeploymentActionState({ status: "error", message: "当前没有可用于验证的执行记录。" });
      return;
    }
    setDeploymentActionState({ status: "running", message: "正在创建部署后验证..." });
    try {
      const payload = await postJSON<PostDeploymentVerificationEnvelope>(`/api/projects/${snapshot.project.id}/post-deployment-verifications`, {
        execution_id: execution.id,
        environment: execution.environment,
        monitor_limit: 10,
      });
      const verification = payload.post_deployment_verification;
      if (!verification) {
        throw new Error(payload.error ?? "部署后验证没有返回记录。");
      }
      setDeploymentActionState({
        status: verification.status === "completed" ? "completed" : "blocked",
        id: verification.id,
        message: `${verification.decision ?? verification.status}${verification.risk_handoff_recommended ? " / 建议风险交接" : ""}`,
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "部署后验证失败。" });
    }
  }

  async function createDeploymentRehearsal() {
    const execution = snapshot.executions[0];
    if (!latestDeployment && !execution) {
      setDeploymentActionState({ status: "error", message: "当前没有可用于演练的部署或执行记录。" });
      return;
    }
    setDeploymentActionState({ status: "running", message: "正在创建部署演练..." });
    try {
      const payload = await postJSON<DeploymentRehearsalEnvelope>(`/api/projects/${snapshot.project.id}/deployment-rehearsals`, {
        deployment_id: latestDeployment?.id,
        execution_id: execution?.id,
        environment: latestDeployment?.environment || execution?.environment,
      });
      const rehearsal = payload.deployment_rehearsal;
      if (!rehearsal) {
        throw new Error(payload.error ?? "部署演练没有返回记录。");
      }
      setDeploymentActionState({
        status: rehearsal.status === "blocked" ? "blocked" : "completed",
        id: rehearsal.id,
        message: rehearsal.decision ?? rehearsal.status ?? "部署演练已记录",
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "部署演练失败。" });
    }
  }

  async function runRehearsalScheduler() {
    const execution = snapshot.executions[0];
    if (!latestDeployment && !execution && !snapshot.release_candidates[0]) {
      setDeploymentActionState({ status: "error", message: "当前没有可用于调度的候选、部署或执行记录。" });
      return;
    }
    setDeploymentActionState({ status: "running", message: "正在运行有界演练调度器..." });
    try {
      const payload = await postJSON<RehearsalSchedulerEnvelope>(`/api/projects/${snapshot.project.id}/deployment-rehearsal-scheduler-runs`, {
        candidate_id: snapshot.release_candidates[0]?.id,
        deployment_id: latestDeployment?.id,
        execution_id: execution?.id,
        environment: latestDeployment?.environment || execution?.environment,
        max_targets: 3,
      });
      const run = payload.rehearsal_scheduler_run;
      if (!run) {
        throw new Error(payload.error ?? "演练调度器没有返回运行记录。");
      }
      setDeploymentActionState({
        status: run.status === "blocked" || run.status === "attention_required" ? "blocked" : "completed",
        id: run.id,
        message: `${run.decision ?? run.status ?? "调度运行已记录"}${run.blocked_count ? ` / 阻断 ${run.blocked_count}` : ""}`,
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "演练调度失败。" });
    }
  }

  async function createReleaseAdmission() {
    setDeploymentActionState({ status: "running", message: "正在创建 Release 准入..." });
    try {
      const payload = await postJSON<ReleaseAdmissionEnvelope>(`/api/projects/${snapshot.project.id}/release-admissions`, {
        rehearsal_id: latestRehearsal?.id,
        deployment_id: latestDeployment?.id,
        execution_id: snapshot.executions[0]?.id,
        environment: latestDeployment?.environment || snapshot.executions[0]?.environment,
      });
      const admission = payload.release_admission;
      if (!admission) {
        throw new Error(payload.error ?? "Release admission returned no record.");
      }
      setDeploymentActionState({
        status: admission.status === "blocked" ? "blocked" : "completed",
        id: admission.id,
        message: admission.decision ?? admission.status ?? "Release 准入已记录",
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "Release 准入失败。" });
    }
  }

  async function createDeploymentRiskHandoff() {
    if (!latestAdmission) {
      setDeploymentActionState({ status: "error", message: "当前没有可用于风险交接的 Release 准入记录。" });
      return;
    }
    setDeploymentActionState({ status: "running", message: "正在创建部署风险交接..." });
    try {
      const payload = await postJSON<DeploymentRiskHandoffEnvelope>(`/api/projects/${snapshot.project.id}/repair/deployment-risk-handoffs`, {
        admission_id: latestAdmission.id,
      });
      const handoff = payload.deployment_risk_handoff;
      if (!handoff) {
        throw new Error(payload.error ?? "部署风险交接没有返回记录。");
      }
      setDeploymentActionState({
        status: handoff.status === "blocked" ? "blocked" : "completed",
        id: handoff.id,
        message: handoff.decision ?? handoff.status ?? "部署风险交接已记录",
      });
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "部署风险交接失败。" });
    }
  }

  async function createApproval() {
    if (
      !requireFields("approvalCreate", [
        ["目标类型", approvalCreateForm.targetType],
        ["目标 ID", approvalCreateForm.targetID],
        ["动作", approvalCreateForm.action],
        ["请求人", approvalCreateForm.requestedBy],
        ["原因", approvalCreateForm.reason],
      ])
    ) {
      setApprovalActionState((current) => ({ ...current, create: { status: "error", message: "请先补齐审批信息。" } }));
      return;
    }
    setApprovalActionState((current) => ({ ...current, create: { status: "running", message: "正在创建审批..." } }));
    try {
      const payload = await postJSON<ApprovalCreateEnvelope>(`/api/projects/${snapshot.project.id}/approvals`, {
        target_type: approvalCreateForm.targetType.trim(),
        target_id: approvalCreateForm.targetID.trim(),
        action: approvalCreateForm.action.trim(),
        risk_level: approvalCreateForm.riskLevel.trim(),
        requested_by: approvalCreateForm.requestedBy.trim(),
        reason: approvalCreateForm.reason.trim(),
        metadata: parseKeyValuePairs(approvalCreateForm.metadata),
      });
      const approval = payload.approval;
      if (!approval) {
        throw new Error(payload.error ?? "审批创建没有返回记录。");
      }
      setApprovalActionState((current) => ({
        ...current,
        create: { status: approval.status === "pending" ? "completed" : "blocked", id: approval.id, message: approval.decision ?? approval.status },
      }));
      setApprovalCreateModalOpen(false);
      router.refresh();
    } catch (error) {
      setApprovalActionState((current) => ({ ...current, create: { status: "error", message: error instanceof Error ? error.message : "审批创建失败。" } }));
    }
  }

  async function decideApproval(approvalID: string, decision: "approved" | "rejected") {
    if (!requireFields("approval", [["决策人", approvalForm.decidedBy], ["原因", approvalForm.reason]])) {
      return;
    }
    setApprovalActionState((current) => ({
      ...current,
      [approvalID]: { status: "running", message: decision === "approved" ? "正在批准审批..." : "正在拒绝审批..." },
    }));
    try {
      const payload = await postJSON<ApprovalDecisionEnvelope>(`/api/projects/${snapshot.project.id}/approvals/${encodeURIComponent(approvalID)}/decide`, {
        decision,
        decided_by: approvalForm.decidedBy,
        reason: approvalForm.reason,
      });
      const approval = payload.approval;
      if (!approval) {
        throw new Error(payload.error ?? "审批决策没有返回记录。");
      }
      setApprovalActionState((current) => ({
        ...current,
        [approvalID]: { status: approval.status === "approved" ? "completed" : "blocked", id: approval.id, message: approval.decision ?? approval.status },
      }));
      setApprovalDecisionModal(null);
      router.refresh();
    } catch (error) {
      setApprovalActionState((current) => ({
        ...current,
        [approvalID]: { status: "error", message: error instanceof Error ? error.message : "审批决策失败。" },
      }));
    }
  }

  async function createSession(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("session", [["用户", sessionForm.userID], ["显示名", sessionForm.displayName], ["角色", sessionForm.roles]])) {
      return;
    }
    setAccessActionState((current) => ({ ...current, session: { status: "running", message: "正在创建会话..." } }));
    try {
      const payload = await postJSON<AuthSessionEnvelope>(`/api/projects/${snapshot.project.id}/auth/sessions`, {
        user_id: sessionForm.userID,
        display_name: sessionForm.displayName,
        roles: splitCSV(sessionForm.roles),
      });
      if (!payload.session) {
        throw new Error(payload.error ?? "会话创建没有返回记录。");
      }
      setAccessActionState((current) => ({
        ...current,
        session: { status: "completed", id: payload.session?.id, message: "SESSION_CREATED" },
      }));
      setSessionModalOpen(false);
      router.refresh();
    } catch (error) {
      setAccessActionState((current) => ({
        ...current,
        session: { status: "error", message: error instanceof Error ? error.message : "会话创建失败。" },
      }));
    }
  }

  async function createAPIToken(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("token", [["名称", tokenForm.name], ["主体", tokenForm.actorID], ["Scopes", tokenForm.scopes]])) {
      return;
    }
    setAccessActionState((current) => ({ ...current, token: { status: "running", message: "正在创建 API Token..." } }));
    try {
      const payload = await postJSON<APITokenCreateEnvelope>(`/api/projects/${snapshot.project.id}/auth/api-tokens`, {
        name: tokenForm.name,
        actor_id: tokenForm.actorID,
        scopes: splitCSV(tokenForm.scopes),
      });
      if (!payload.api_token) {
        throw new Error(payload.error ?? "API Token 创建没有返回记录。");
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
      setTokenModalOpen(false);
      router.refresh();
    } catch (error) {
      setAccessActionState((current) => ({
        ...current,
        token: { status: "error", message: error instanceof Error ? error.message : "API Token 创建失败。" },
      }));
    }
  }

  async function revokeSession(sessionID: string) {
    if (!requireFields("sessionRevoke", [["执行人", sessionRevokeForm.actorID], ["原因", sessionRevokeForm.reason]])) {
      return;
    }
    setAccessActionState((current) => ({ ...current, [sessionID]: { status: "running", message: "正在撤销会话..." } }));
    try {
      const payload = await postJSON<AuthSessionEnvelope>(`/api/projects/${snapshot.project.id}/auth/sessions/${encodeURIComponent(sessionID)}/revoke`, {
        actor_id: sessionRevokeForm.actorID.trim(),
        reason: sessionRevokeForm.reason.trim(),
      });
      if (!payload.session) {
        throw new Error(payload.error ?? "会话撤销没有返回记录。");
      }
      setAccessActionState((current) => ({
        ...current,
        [sessionID]: { status: payload.session?.status === "revoked" ? "completed" : "blocked", id: payload.session?.id, message: "SESSION_REVOKED" },
      }));
      setSessionRevokeModalID(null);
      router.refresh();
    } catch (error) {
      setAccessActionState((current) => ({
        ...current,
        [sessionID]: { status: "error", message: error instanceof Error ? error.message : "会话撤销失败。" },
      }));
    }
  }

  async function revokeAPIToken(tokenID: string) {
    if (!requireFields("tokenRevoke", [["执行人", tokenRevokeForm.actorID], ["原因", tokenRevokeForm.reason]])) {
      return;
    }
    setAccessActionState((current) => ({ ...current, [tokenID]: { status: "running", message: "正在撤销 Token..." } }));
    try {
      const payload = await postJSON<APITokenRevokeEnvelope>(`/api/projects/${snapshot.project.id}/auth/api-tokens/${encodeURIComponent(tokenID)}/revoke`, {
        actor_id: tokenRevokeForm.actorID.trim(),
        reason: tokenRevokeForm.reason.trim(),
      });
      if (!payload.api_token) {
        throw new Error(payload.error ?? "API Token 撤销没有返回记录。");
      }
      setAccessActionState((current) => ({
        ...current,
        [tokenID]: { status: "completed", id: payload.api_token?.id, message: "API_TOKEN_REVOKED" },
      }));
      setTokenRevokeModalID(null);
      router.refresh();
    } catch (error) {
      setAccessActionState((current) => ({
        ...current,
        [tokenID]: { status: "error", message: error instanceof Error ? error.message : "API Token 撤销失败。" },
      }));
    }
  }

  async function createServiceAccount(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("service", [["名称", serviceAccountForm.name], ["角色", serviceAccountForm.roles]])) {
      return;
    }
    setAccessActionState((current) => ({ ...current, service: { status: "running", message: "正在保存服务账号..." } }));
    try {
      const payload = await postJSON<ServiceAccountEnvelope>(`/api/projects/${snapshot.project.id}/auth/service-accounts`, {
        id: serviceAccountForm.id,
        name: serviceAccountForm.name,
        roles: splitCSV(serviceAccountForm.roles),
      });
      if (!payload.service_account) {
        throw new Error(payload.error ?? "服务账号保存没有返回记录。");
      }
      setAccessActionState((current) => ({
        ...current,
        service: { status: "completed", id: payload.service_account?.id, message: "SERVICE_ACCOUNT_UPSERTED" },
      }));
      setServiceAccountModalOpen(false);
      router.refresh();
    } catch (error) {
      setAccessActionState((current) => ({
        ...current,
        service: { status: "error", message: error instanceof Error ? error.message : "服务账号保存失败。" },
      }));
    }
  }

  async function runGitProviderAction(planID: string, action: "preview" | "sync" | "create") {
    if (action === "create" && gitCreateApproved && !requireFields("gitCreate", [["Approval ID", gitCreateApprovalID]])) {
      return;
    }
    setGitActionState((current) => ({ ...current, [planID]: { status: "running", message: `正在${action === "preview" ? "预览" : action === "sync" ? "同步" : "创建"} PR/MR...` } }));
    try {
      const payload = await postJSON<GitProviderActionEnvelope>(`/api/projects/${snapshot.project.id}/git-provider-plans/${encodeURIComponent(planID)}/${action}`, {
        approved: action === "create" ? gitCreateApproved : undefined,
        approval_id: action === "create" ? gitCreateApprovalID : undefined,
      });
      const plan = payload.git_provider_plan;
      if (!plan) {
        throw new Error(payload.error ?? "PR/MR 操作没有返回计划。");
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
      if (action === "create") {
        setGitCreateModalPlanID(null);
      }
      router.refresh();
    } catch (error) {
      setGitActionState((current) => ({
        ...current,
        [planID]: { status: "error", message: error instanceof Error ? error.message : "PR/MR 操作失败。" },
      }));
    }
  }

  async function previewProviderRoute() {
    if (!requireFields("providerRoute", [["角色", providerRouteForm.role], ["任务类型", providerRouteForm.taskType]])) {
      setProviderRouteState({ status: "error", message: "表单校验失败。" });
      return;
    }
    setProviderRouteState({ status: "running", message: "正在评估路由候选..." });
    try {
      const payload = await postJSON<ProviderRouteEnvelope>(`/api/projects/${snapshot.project.id}/provider-route`, {
        role: providerRouteForm.role,
        model_strategy: providerRouteForm.modelStrategy === "default" ? "" : providerRouteForm.modelStrategy,
        task_type: providerRouteForm.taskType,
        output_type: providerRouteForm.outputType,
        requires_repo_edit: providerRouteForm.requiresRepoEdit,
        includes_sensitive_code: providerRouteForm.includesSensitiveCode,
        includes_project_memory: providerRouteForm.includesProjectMemory,
        includes_secrets: false,
      });
      if (!payload.route) {
        throw new Error(payload.error ?? "Provider 路由没有返回决策。");
      }
      setProviderRoute(payload.route);
      setProviderRouteState({
        status: payload.route.blocked ? "blocked" : "completed",
        id: payload.route.provider_id,
        message: `${payload.route.decision} / ${payload.route.candidates?.length ?? 0} 个候选`,
      });
      router.refresh();
    } catch (error) {
      setProviderRouteState({ status: "error", message: error instanceof Error ? error.message : "Provider 路由预览失败。" });
    }
  }

  async function upsertProvider(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("provider", [["Provider ID", providerForm.id], ["名称", providerForm.name], ["Vendor", providerForm.vendor], ["API Type", providerForm.apiType]])) {
      setProviderActionState((current) => ({ ...current, upsert: { status: "error", message: "请先补齐 Provider 信息。" } }));
      return;
    }
    setProviderActionState((current) => ({ ...current, upsert: { status: "running", message: "正在保存 Provider..." } }));
    try {
      const payload = await postJSON<ProviderEnvelope>(`/api/projects/${snapshot.project.id}/providers`, {
        id: providerForm.id.trim(),
        name: providerForm.name.trim(),
        vendor: providerForm.vendor.trim(),
        api_type: providerForm.apiType.trim(),
        base_url: providerForm.baseURL.trim(),
        auth_ref: providerForm.authRef.trim(),
        enabled: providerForm.enabled,
        native_runtime: providerForm.nativeRuntime,
        runtime_id: providerForm.runtimeID.trim(),
        data_policy: {
          allow_sensitive_code: providerForm.allowSensitiveCode,
          allow_project_memory: providerForm.allowProjectMemory,
          allow_production_context: providerForm.allowProductionContext,
        },
        models: providerForm.model.trim() ? [{ id: providerForm.model.trim() }] : [],
        allowed_use_cases: splitCSV(providerForm.useCases),
      });
      const provider = payload.provider;
      if (!provider) {
        throw new Error(payload.error ?? "Provider 保存没有返回记录。");
      }
      setProviderActionState((current) => ({
        ...current,
        upsert: { status: provider.enabled ? "completed" : "blocked", id: provider.id, message: provider.enabled ? "已启用" : "已保存但未启用" },
      }));
      setProviderModalOpen(false);
      router.refresh();
    } catch (error) {
      setProviderActionState((current) => ({
        ...current,
        upsert: { status: "error", message: error instanceof Error ? error.message : "Provider 保存失败。" },
      }));
    }
  }

  async function refreshProviderOps(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setProviderActionState((current) => ({ ...current, ops: { status: "running", message: "正在刷新 Provider Ops..." } }));
    try {
      const payload = await postJSON<ProviderOpsRefreshEnvelope>(`/api/projects/${snapshot.project.id}/providers/ops/refresh`, {
        provider_id: providerOpsForm.providerID.trim(),
        include_disabled: providerOpsForm.includeDisabled,
        probe: providerOpsForm.probe,
        probe_timeout_ms: parseOptionalInt(providerOpsForm.probeTimeoutMS),
        approved: providerOpsForm.approved,
      });
      const result = payload.provider_ops_refresh;
      if (!result) {
        throw new Error(payload.error ?? "Provider Ops 没有返回结果。");
      }
      setProviderActionState((current) => ({
        ...current,
        ops: { status: result.status === "completed" || result.status === "ok" ? "completed" : "blocked", id: result.id, message: result.decision ?? result.status ?? "刷新完成" },
      }));
      setProviderOpsModalOpen(false);
      router.refresh();
    } catch (error) {
      setProviderActionState((current) => ({
        ...current,
        ops: { status: "error", message: error instanceof Error ? error.message : "Provider Ops 刷新失败。" },
      }));
    }
  }

  async function updateProviderOpsSnapshot(providerID: string) {
    if (!requireFields("providerOpsSnapshot", [["健康状态", providerOpsSnapshotForm.healthStatus], ["配额状态", providerOpsSnapshotForm.quotaStatus], ["成本状态", providerOpsSnapshotForm.costStatus]])) {
      setProviderActionState((current) => ({ ...current, [`${providerID}:ops`]: { status: "error", message: "请先补齐 Ops 快照。" } }));
      return;
    }
    setProviderActionState((current) => ({ ...current, [`${providerID}:ops`]: { status: "running", message: "正在保存 Provider Ops..." } }));
    try {
      const payload = await postJSON<ProviderEnvelope>(`/api/projects/${snapshot.project.id}/providers/${encodeURIComponent(providerID)}/ops`, {
        health: {
          status: providerOpsSnapshotForm.healthStatus.trim(),
          reason: providerOpsSnapshotForm.healthReason.trim(),
        },
        quota: {
          status: providerOpsSnapshotForm.quotaStatus.trim(),
          limit_tokens: parseOptionalInt(providerOpsSnapshotForm.limitTokens),
          used_tokens: parseOptionalInt(providerOpsSnapshotForm.usedTokens),
          remaining_tokens: parseOptionalInt(providerOpsSnapshotForm.remainingTokens),
        },
        usage: {
          window: providerOpsSnapshotForm.usageWindow.trim(),
          requests: parseOptionalInt(providerOpsSnapshotForm.requests),
          input_tokens: parseOptionalInt(providerOpsSnapshotForm.inputTokens),
          output_tokens: parseOptionalInt(providerOpsSnapshotForm.outputTokens),
          total_tokens: parseOptionalInt(providerOpsSnapshotForm.totalTokens),
        },
        cost: {
          status: providerOpsSnapshotForm.costStatus.trim(),
          estimated_amount: parseOptionalFloat(providerOpsSnapshotForm.estimatedAmount),
          budget_amount: parseOptionalFloat(providerOpsSnapshotForm.budgetAmount),
        },
      });
      const provider = payload.provider;
      if (!provider) {
        throw new Error(payload.error ?? "Provider Ops 保存没有返回记录。");
      }
      setProviderActionState((current) => ({
        ...current,
        [`${providerID}:ops`]: { status: provider.enabled ? "completed" : "blocked", id: provider.id, message: "PROVIDER_OPS_UPDATED" },
      }));
      setProviderOpsSnapshotModalID(null);
      router.refresh();
    } catch (error) {
      setProviderActionState((current) => ({
        ...current,
        [`${providerID}:ops`]: { status: "error", message: error instanceof Error ? error.message : "Provider Ops 保存失败。" },
      }));
    }
  }

  async function disableProvider(providerID: string) {
    setProviderActionState((current) => ({ ...current, [providerID]: { status: "running", message: "正在禁用 Provider..." } }));
    try {
      const payload = await postJSON<ProviderEnvelope>(`/api/projects/${snapshot.project.id}/providers/${encodeURIComponent(providerID)}/disable`, {});
      const provider = payload.provider;
      if (!provider) {
        throw new Error(payload.error ?? "Provider 禁用没有返回记录。");
      }
      setProviderActionState((current) => ({
        ...current,
        [providerID]: { status: provider.enabled ? "blocked" : "completed", id: provider.id, message: provider.enabled ? "仍处于启用状态" : "已禁用" },
      }));
      setProviderDisableModalID(null);
      router.refresh();
    } catch (error) {
      setProviderActionState((current) => ({
        ...current,
        [providerID]: { status: "error", message: error instanceof Error ? error.message : "Provider 禁用失败。" },
      }));
    }
  }

  async function upsertSkill(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("skill", [["Skill ID", skillForm.id], ["名称", skillForm.name], ["来源", skillForm.source], ["风险等级", skillForm.riskLevel]])) {
      setSkillActionState((current) => ({ ...current, upsert: { status: "error", message: "请先补齐 Skill 信息。" } }));
      return;
    }
    setSkillActionState((current) => ({ ...current, upsert: { status: "running", message: "正在保存 Skill..." } }));
    try {
      const payload = await postJSON<SkillEnvelope>(`/api/projects/${snapshot.project.id}/skills`, {
        id: skillForm.id.trim(),
        name: skillForm.name.trim(),
        source: skillForm.source.trim(),
        version: skillForm.version.trim(),
        description: skillForm.description.trim(),
        enabled: skillForm.enabled,
        risk_level: skillForm.riskLevel.trim(),
        compatible_roles: splitCSV(skillForm.compatibleRoles),
        tags: splitCSV(skillForm.tags),
        required_tools: splitCSV(skillForm.requiredTools),
        auth_ref: skillForm.authRef.trim(),
      });
      const skill = payload.skill;
      if (!skill) {
        throw new Error(payload.error ?? "Skill 保存没有返回记录。");
      }
      setSkillActionState((current) => ({
        ...current,
        upsert: { status: skill.enabled ? "completed" : "blocked", id: skill.id, message: skill.enabled ? "已启用" : "已保存但未启用" },
      }));
      setSkillModalOpen(false);
      router.refresh();
    } catch (error) {
      setSkillActionState((current) => ({ ...current, upsert: { status: "error", message: error instanceof Error ? error.message : "Skill 保存失败。" } }));
    }
  }

  async function recommendSkills(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("skillRecommend", [["角色", skillRecommendationForm.role]])) {
      setSkillActionState((current) => ({ ...current, recommend: { status: "error", message: "请先填写角色。" } }));
      return;
    }
    setSkillActionState((current) => ({ ...current, recommend: { status: "running", message: "正在生成 Skill 推荐..." } }));
    try {
      const payload = await postJSON<SkillRecommendationEnvelope>(`/api/projects/${snapshot.project.id}/skills/recommend`, {
        issue_id: skillRecommendationForm.issueID.trim(),
        role: skillRecommendationForm.role.trim(),
        task_type: skillRecommendationForm.taskType.trim(),
        risk_level: skillRecommendationForm.riskLevel.trim(),
        limit: parseOptionalInt(skillRecommendationForm.limit),
      });
      if (!payload.skill_recommendation) {
        throw new Error(payload.error ?? "Skill 推荐没有返回报告。");
      }
      setSkillRecommendation(payload.skill_recommendation);
      setSkillActionState((current) => ({
        ...current,
        recommend: {
          status: payload.skill_recommendation?.candidates.length ? "completed" : "blocked",
          id: payload.skill_recommendation?.id,
          message: `${payload.skill_recommendation?.candidates.length ?? 0} 个候选`,
        },
      }));
      setSkillRecommendationModalOpen(false);
    } catch (error) {
      setSkillActionState((current) => ({ ...current, recommend: { status: "error", message: error instanceof Error ? error.message : "Skill 推荐失败。" } }));
    }
  }

  async function upsertSkillBinding(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("skillBinding", [["Skill ID", skillBindingForm.skillID], ["目标类型", skillBindingForm.targetType], ["目标 ID", skillBindingForm.targetID]])) {
      setSkillActionState((current) => ({ ...current, binding: { status: "error", message: "请先补齐绑定信息。" } }));
      return;
    }
    setSkillActionState((current) => ({ ...current, binding: { status: "running", message: "正在保存 Skill 绑定..." } }));
    try {
      const payload = await postJSON<SkillBindingEnvelope>(`/api/projects/${snapshot.project.id}/skills/bindings`, {
        id: skillBindingForm.id.trim(),
        skill_id: skillBindingForm.skillID.trim(),
        target_type: skillBindingForm.targetType.trim(),
        target_id: skillBindingForm.targetID.trim(),
        priority: parseOptionalInt(skillBindingForm.priority),
        status: skillBindingForm.status.trim() || "active",
        config: parseKeyValuePairs(skillBindingForm.config),
      });
      const binding = payload.skill_binding;
      if (!binding) {
        throw new Error(payload.error ?? "Skill 绑定没有返回记录。");
      }
      setSkillActionState((current) => ({
        ...current,
        binding: { status: binding.status === "active" ? "completed" : "blocked", id: binding.id, message: binding.status ?? "已保存" },
      }));
      setSkillBindingModalOpen(false);
      router.refresh();
    } catch (error) {
      setSkillActionState((current) => ({ ...current, binding: { status: "error", message: error instanceof Error ? error.message : "Skill 绑定保存失败。" } }));
    }
  }

  async function recordSkillEffectiveness(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("skillEffectiveness", [["Skill ID", skillEffectivenessForm.skillID], ["结果", skillEffectivenessForm.outcome], ["质量影响", skillEffectivenessForm.qualityImpact]])) {
      setSkillActionState((current) => ({ ...current, effectiveness: { status: "error", message: "请先补齐效果记录。" } }));
      return;
    }
    setSkillActionState((current) => ({ ...current, effectiveness: { status: "running", message: "正在记录 Skill 效果..." } }));
    try {
      const payload = await postJSON<SkillEffectivenessEnvelope>(`/api/projects/${snapshot.project.id}/skills/effectiveness`, {
        id: skillEffectivenessForm.id.trim(),
        skill_id: skillEffectivenessForm.skillID.trim(),
        binding_id: skillEffectivenessForm.bindingID.trim(),
        subagent_id: skillEffectivenessForm.subagentID.trim(),
        run_id: skillEffectivenessForm.runID.trim(),
        issue_id: skillEffectivenessForm.issueID.trim(),
        outcome: skillEffectivenessForm.outcome.trim(),
        quality_impact: skillEffectivenessForm.qualityImpact.trim(),
        rework_reduced: skillEffectivenessForm.reworkReduced,
        duration_seconds: parseOptionalInt(skillEffectivenessForm.durationSeconds),
        findings: splitLines(skillEffectivenessForm.findings),
      });
      const record = payload.skill_effectiveness;
      if (!record) {
        throw new Error(payload.error ?? "Skill 效果没有返回记录。");
      }
      setSkillActionState((current) => ({
        ...current,
        effectiveness: { status: "completed", id: record.id, message: `${record.outcome ?? "recorded"} / ${record.quality_impact ?? "quality"}` },
      }));
      setSkillEffectivenessModalOpen(false);
      router.refresh();
    } catch (error) {
      setSkillActionState((current) => ({
        ...current,
        effectiveness: { status: "error", message: error instanceof Error ? error.message : "Skill 效果记录失败。" },
      }));
    }
  }

  async function disableSkill(skillID: string) {
    setSkillActionState((current) => ({ ...current, [skillID]: { status: "running", message: "正在禁用 Skill..." } }));
    try {
      const payload = await postJSON<SkillEnvelope>(`/api/projects/${snapshot.project.id}/skills/${encodeURIComponent(skillID)}/disable`, {});
      const skill = payload.skill;
      if (!skill) {
        throw new Error(payload.error ?? "Skill 禁用没有返回记录。");
      }
      setSkillActionState((current) => ({
        ...current,
        [skillID]: { status: skill.enabled ? "blocked" : "completed", id: skill.id, message: skill.enabled ? "仍处于启用状态" : "已禁用" },
      }));
      setSkillDisableModalID(null);
      router.refresh();
    } catch (error) {
      setSkillActionState((current) => ({ ...current, [skillID]: { status: "error", message: error instanceof Error ? error.message : "Skill 禁用失败。" } }));
    }
  }

  async function disableSkillBinding(bindingID: string) {
    setSkillActionState((current) => ({ ...current, [bindingID]: { status: "running", message: "正在禁用绑定..." } }));
    try {
      const payload = await postJSON<SkillBindingEnvelope>(`/api/projects/${snapshot.project.id}/skills/bindings/${encodeURIComponent(bindingID)}/disable`, {});
      const binding = payload.skill_binding;
      if (!binding) {
        throw new Error(payload.error ?? "Skill 绑定禁用没有返回记录。");
      }
      setSkillActionState((current) => ({
        ...current,
        [bindingID]: { status: binding.status === "disabled" ? "completed" : "blocked", id: binding.id, message: binding.status ?? "已禁用" },
      }));
      setSkillBindingDisableModalID(null);
      router.refresh();
    } catch (error) {
      setSkillActionState((current) => ({
        ...current,
        [bindingID]: { status: "error", message: error instanceof Error ? error.message : "Skill 绑定禁用失败。" },
      }));
    }
  }

  async function runControlLoop() {
    setControlLoopActionState({ status: "running", message: "正在运行有界控制循环..." });
    try {
      const payload = await postJSON<ControlLoopRunEnvelope>(`/api/projects/${snapshot.project.id}/control-loop/run`, {
        trigger: "console_manual",
        requested_by: "console-owner",
      });
      const run = payload.control_loop_run;
      if (!run) {
        throw new Error(payload.error ?? "控制循环没有返回运行记录。");
      }
      setControlLoopActionState({
        status: run.status === "completed" && !(run.decision ?? "").includes("ATTENTION") ? "completed" : "blocked",
        id: run.id,
        message: `${run.decision ?? run.status} / ${run.steps?.length ?? 0} 个步骤`,
      });
      router.refresh();
    } catch (error) {
      setControlLoopActionState({ status: "error", message: error instanceof Error ? error.message : "控制循环运行失败。" });
    }
  }

  async function createRemoteExecutionRehearsals(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAdapterActionState({ status: "running", message: "正在生成远程执行演练..." });
    try {
      const payload = await postJSON<RemoteExecutionRehearsalEnvelope>(`/api/projects/${snapshot.project.id}/operations/remote-execution-rehearsals`, {
        admission_id: remoteRehearsalForm.admissionID.trim(),
        execution_id: remoteRehearsalForm.executionID.trim(),
        provider: remoteRehearsalForm.provider.trim(),
        environment: remoteRehearsalForm.environment.trim(),
        status: remoteRehearsalForm.status.trim(),
        decision: remoteRehearsalForm.decision.trim(),
        limit: parseOptionalInt(remoteRehearsalForm.limit),
      });
      const report = payload.remote_execution_rehearsals;
      if (!report) {
        throw new Error(payload.error ?? "远程执行演练没有返回报告。");
      }
      setAdapterActionState({
        status: report.summary?.blocked_count ? "blocked" : "completed",
        id: report.id,
        message: `${report.summary?.rehearsal_count ?? 0} 次演练`,
      });
      setWritePipelineModal(null);
      router.refresh();
    } catch (error) {
      setAdapterActionState({ status: "error", message: error instanceof Error ? error.message : "远程执行演练失败。" });
    }
  }

  async function createWriteReviewPackets(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("writeReviewPacket", [["Operation Type", writeReviewPacketForm.operationType]])) {
      setAdapterActionState({ status: "error", message: "请先补齐复核包信息。" });
      return;
    }
    setAdapterActionState({ status: "running", message: "正在生成写入复核包..." });
    try {
      const payload = await postJSON<WriteReviewPacketEnvelope>(`/api/projects/${snapshot.project.id}/operations/write-review-packets`, {
        admission_id: writeReviewPacketForm.admissionID.trim(),
        operation_type: writeReviewPacketForm.operationType.trim(),
        operation_id: writeReviewPacketForm.operationID.trim(),
        provider: writeReviewPacketForm.provider.trim(),
        environment: writeReviewPacketForm.environment.trim(),
        status: writeReviewPacketForm.status.trim(),
        decision: writeReviewPacketForm.decision.trim(),
        limit: parseOptionalInt(writeReviewPacketForm.limit),
      });
      const report = payload.write_review_packets;
      if (!report) {
        throw new Error(payload.error ?? "写入复核包没有返回报告。");
      }
      setAdapterActionState({
        status: report.summary?.blocked_count ? "blocked" : "completed",
        id: report.id,
        message: `${report.summary?.packet_count ?? 0} 个 packet`,
      });
      setWritePipelineModal(null);
      router.refresh();
    } catch (error) {
      setAdapterActionState({ status: "error", message: error instanceof Error ? error.message : "写入复核包失败。" });
    }
  }

  async function createWriteExecutionPlans(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("writeExecutionPlan", [["Review Packet ID", writeExecutionPlanForm.reviewPacketID], ["模式", writeExecutionPlanForm.mode]])) {
      setAdapterActionState({ status: "error", message: "请先补齐执行计划信息。" });
      return;
    }
    setAdapterActionState({ status: "running", message: "正在生成写入执行计划..." });
    try {
      const payload = await postJSON<WriteExecutionPlanEnvelope>(`/api/projects/${snapshot.project.id}/operations/write-execution-plans`, {
        review_packet_id: writeExecutionPlanForm.reviewPacketID.trim(),
        mode: writeExecutionPlanForm.mode.trim(),
        approval_id: writeExecutionPlanForm.approvalID.trim(),
        requested_by: writeExecutionPlanForm.requestedBy.trim(),
        status: writeExecutionPlanForm.status.trim(),
        decision: writeExecutionPlanForm.decision.trim(),
        limit: parseOptionalInt(writeExecutionPlanForm.limit),
      });
      const report = payload.write_execution_plans;
      if (!report) {
        throw new Error(payload.error ?? "写入执行计划没有返回报告。");
      }
      setAdapterActionState({
        status: report.summary?.blocked_count ? "blocked" : "completed",
        id: report.id,
        message: `${report.summary?.plan_count ?? 0} 个计划`,
      });
      setWritePipelineModal(null);
      router.refresh();
    } catch (error) {
      setAdapterActionState({ status: "error", message: error instanceof Error ? error.message : "写入执行计划失败。" });
    }
  }

  async function createWriteAdapterExecutions(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!requireFields("writeAdapterExecution", [["Execution Plan ID", writeAdapterExecutionForm.executionPlanID], ["Adapter ID", writeAdapterExecutionForm.adapterID]])) {
      setAdapterActionState({ status: "error", message: "请先补齐 Adapter 执行信息。" });
      return;
    }
    setAdapterActionState({ status: "running", message: "正在生成 Adapter 执行..." });
    try {
      const payload = await postJSON<WriteAdapterExecutionEnvelope>(`/api/projects/${snapshot.project.id}/operations/write-adapter-executions`, {
        execution_plan_id: writeAdapterExecutionForm.executionPlanID.trim(),
        mode: writeAdapterExecutionForm.mode.trim(),
        adapter_id: writeAdapterExecutionForm.adapterID.trim(),
        status: writeAdapterExecutionForm.status.trim(),
        decision: writeAdapterExecutionForm.decision.trim(),
        limit: parseOptionalInt(writeAdapterExecutionForm.limit),
      });
      const report = payload.write_adapter_executions;
      if (!report) {
        throw new Error(payload.error ?? "Adapter 执行没有返回报告。");
      }
      setAdapterActionState({
        status: report.summary?.blocked_count ? "blocked" : "completed",
        id: report.id,
        message: `${report.summary?.execution_count ?? 0} 条执行`,
      });
      setWritePipelineModal(null);
      router.refresh();
    } catch (error) {
      setAdapterActionState({ status: "error", message: error instanceof Error ? error.message : "Adapter 执行失败。" });
    }
  }

  async function reviewOperationRepairCandidate(candidateID: string, decision: "approved" | "rejected") {
    if (!requireFields("repairReview", [["复核人", repairReviewForm.reviewerID], ["原因", repairReviewForm.reason]])) {
      return;
    }
    setRepairActionState((current) => ({
      ...current,
      [candidateID]: { status: "running", message: decision === "approved" ? "正在批准修复候选..." : "正在拒绝修复候选..." },
    }));
    try {
      const payload = await postJSON<OperationRepairReviewEnvelope>(
        `/api/projects/${snapshot.project.id}/repair/operation-candidates/${encodeURIComponent(candidateID)}/review`,
        {
          decision,
          reviewer_id: repairReviewForm.reviewerID,
          reason: repairReviewForm.reason,
          next_step: decision === "approved" ? "repair_attempt" : "",
        },
      );
      const review = payload.operation_repair_review;
      const candidate = payload.operation_repair_candidate;
      if (!review || !candidate) {
        throw new Error(payload.error ?? "修复复核没有返回记录。");
      }
      setRepairActionState((current) => ({
        ...current,
        [candidateID]: {
          status: candidate.status === "approved" ? "completed" : "blocked",
          id: payload.repair_attempt?.id ?? candidate.issue_id ?? candidate.id,
          message: `${review.decision ?? candidate.decision}${payload.repair_attempt?.status ? ` / ${payload.repair_attempt.status}` : ""}`,
        },
      }));
      setRepairReviewModal(null);
      router.refresh();
    } catch (error) {
      setRepairActionState((current) => ({
        ...current,
        [candidateID]: { status: "error", message: error instanceof Error ? error.message : "修复候选复核失败。" },
      }));
    }
  }

  async function reviewDeploymentRiskHandoff(handoffID: string, decision: "approved" | "rejected") {
    if (!requireFields("deploymentRiskReview", [["复核人", deploymentRiskReviewForm.reviewerID], ["原因", deploymentRiskReviewForm.reason]])) {
      return;
    }
    setDeploymentActionState((current) => ({ ...current, status: "running", message: "正在提交风险复核..." }));
    try {
      const payload = await postJSON<DeploymentRiskReviewEnvelope>(
        `/api/projects/${snapshot.project.id}/repair/deployment-risk-handoffs/${encodeURIComponent(handoffID)}/review`,
        {
          decision,
          reviewer_id: deploymentRiskReviewForm.reviewerID.trim(),
          reason: deploymentRiskReviewForm.reason.trim(),
          next_step: deploymentRiskReviewForm.nextStep.trim(),
        },
      );
      const review = payload.deployment_risk_review;
      if (!review) {
        throw new Error(payload.error ?? "部署风险复核没有返回记录。");
      }
      setDeploymentActionState({
        status: review.status === "approved" || review.decision === "approved" ? "completed" : "blocked",
        id: review.id,
        message: `${review.decision ?? review.status}${review.next_step ? ` / ${review.next_step}` : ""}`,
      });
      setDeploymentRiskReviewModal(null);
      router.refresh();
    } catch (error) {
      setDeploymentActionState({ status: "error", message: error instanceof Error ? error.message : "部署风险复核失败。" });
    }
  }

  async function enqueueControlLoop() {
    if (!requireFields("controlQueue", [["触发器", controlQueueForm.trigger], ["请求人", controlQueueForm.requestedBy], ["步骤", controlQueueForm.steps]])) {
      setAdapterActionState({ status: "error", message: "请先补齐控制队列信息。" });
      return;
    }
    setAdapterActionState({ status: "running", message: "正在写入控制队列..." });
    try {
      const payload = await postJSON<ControlQueueEnvelope>(`/api/projects/${snapshot.project.id}/control-loop/queue`, {
        trigger: controlQueueForm.trigger.trim(),
        requested_by: controlQueueForm.requestedBy.trim(),
        idempotency_key: controlQueueForm.idempotencyKey.trim(),
        retry_budget: parseOptionalInt(controlQueueForm.retryBudget),
        steps: splitCSV(controlQueueForm.steps),
        environment: controlQueueForm.environment.trim(),
        resource_ids: splitCSV(controlQueueForm.resourceIDs),
        deployment_execution_id: controlQueueForm.deploymentExecutionID.trim(),
        maintenance_window: controlQueueForm.maintenanceWindow.trim(),
        due_at: controlQueueForm.dueAt.trim(),
        priority: parseOptionalInt(controlQueueForm.priority),
        admission_id: controlQueueForm.admissionID.trim(),
        remote_rehearsal_id: controlQueueForm.remoteRehearsalID.trim(),
        review_packet_id: controlQueueForm.reviewPacketID.trim(),
        adapter_recovery_id: controlQueueForm.adapterRecoveryID.trim(),
      });
      const item = payload.control_loop_queue_item;
      if (!item) {
        throw new Error(payload.error ?? "控制队列没有返回条目。");
      }
      setAdapterActionState({
        status: item.status === "queued" || item.status === "pending" ? "completed" : "blocked",
        id: item.id,
        message: item.decision ?? item.status ?? "已入队",
      });
      setControlQueueModalOpen(false);
      router.refresh();
    } catch (error) {
      setAdapterActionState({ status: "error", message: error instanceof Error ? error.message : "控制队列写入失败。" });
    }
  }

  async function runControlQueue() {
    setAdapterActionState({ status: "running", message: "正在消费控制队列..." });
    try {
      const payload = await postJSON<ControlQueueRunEnvelope>(`/api/projects/${snapshot.project.id}/control-loop/queue/run`, {
        status: controlQueueRunForm.status.trim(),
        environment: controlQueueRunForm.environment.trim(),
        max_items: parseOptionalInt(controlQueueRunForm.maxItems),
      });
      const run = payload.control_loop_queue_run;
      if (!run) {
        throw new Error(payload.error ?? "控制队列运行没有返回报告。");
      }
      setAdapterActionState({
        status: run.status === "completed" ? "completed" : "blocked",
        id: run.id,
        message: `${run.decision ?? run.status ?? "队列已消费"}${typeof run.processed_count === "number" ? ` / ${run.processed_count} 条` : ""}`,
      });
      setControlQueueRunModalOpen(false);
      router.refresh();
    } catch (error) {
      setAdapterActionState({ status: "error", message: error instanceof Error ? error.message : "控制队列运行失败。" });
    }
  }

  async function submitResourceCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const fields: Array<[string, string]> = [
      ["资源 ID", resourceCreateForm.id],
      ["环境", resourceCreateForm.environment],
      ["Host", resourceCreateForm.host],
      ["Provider", resourceCreateForm.provider],
      ["Owner", resourceCreateForm.owner],
      ["Auth Ref", resourceCreateForm.authRef],
    ];
    if (resourceCreateForm.environment === "production") {
      fields.push(["到期日", resourceCreateForm.expiresAt]);
    }
    if (!requireFields("resourceCreate", fields)) {
      setResourceCreateActionState({ status: "error", message: "请先补齐服务器资源信息。" });
      return;
    }
    setResourceCreateActionState({ status: "running", message: "正在登记服务器资源..." });
    try {
      const payload = await postJSON<ResourceCreateEnvelope>(`/api/projects/${snapshot.project.id}/resources`, {
        id: resourceCreateForm.id.trim(),
        environment: resourceCreateForm.environment.trim(),
        host: resourceCreateForm.host.trim(),
        provider: resourceCreateForm.provider.trim(),
        owner: resourceCreateForm.owner.trim(),
        purpose: resourceCreateForm.purpose.trim(),
        auth_ref: resourceCreateForm.authRef.trim(),
        expires_at: resourceCreateForm.expiresAt.trim(),
        maintenance_window: resourceCreateForm.maintenanceWindow.trim(),
        spec: {
          cpu: parseOptionalInt(resourceCreateForm.cpu),
          memory_gb: parseOptionalInt(resourceCreateForm.memoryGB),
          disk_gb: parseOptionalInt(resourceCreateForm.diskGB),
          os: resourceCreateForm.os.trim(),
        },
        healthcheck: {
          type: resourceCreateForm.healthType.trim() || "manual",
          target: resourceCreateForm.healthTarget.trim(),
          last_status: "unknown",
        },
      });
      const resource = payload.resource;
      if (!resource) {
        throw new Error(payload.error ?? "服务器资源登记没有返回记录。");
      }
      setResourceCreateActionState({
        status: resource.status === "active" ? "completed" : "blocked",
        id: resource.id,
        message: `已登记 ${resource.id}`,
      });
      setResourceCreateForm((current) => ({ ...current, id: "", host: "", purpose: "", healthTarget: "" }));
      setResourceCreateModalOpen(false);
      router.refresh();
    } catch (error) {
      setResourceCreateActionState({ status: "error", message: error instanceof Error ? error.message : "服务器资源登记失败。" });
    }
  }

  async function runResourceAction(resourceID: string, action: "renew" | "retire") {
    const fields: Array<[string, string]> = [["执行人", resourceForm.actorID], ["原因", resourceForm.reason]];
    if (action === "renew") {
      fields.push(["到期日", resourceForm.expiresAt]);
    }
    if (!requireFields("resource", fields)) {
      return;
    }
    setResourceActionState((current) => ({ ...current, [resourceID]: { status: "running", message: action === "renew" ? "正在续期资源..." : "正在退役资源..." } }));
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
      setResourceActionModal(null);
      router.refresh();
    } catch (error) {
      setResourceActionState((current) => ({
        ...current,
        [resourceID]: { status: "error", message: error instanceof Error ? error.message : "Resource action failed." },
      }));
    }
  }

  async function runResourceScan(kind: "maintenance" | "lifecycle" | "health") {
    const actionKey = `scan:${kind}`;
    if (kind === "health" && !requireFields("resourceScan", [["环境", resourceScanForm.environment]])) {
      setResourceActionState((current) => ({ ...current, [actionKey]: { status: "error", message: "请先补齐健康扫描环境。" } }));
      return;
    }
    setResourceActionState((current) => ({ ...current, [actionKey]: { status: "running", message: "正在触发资源扫描..." } }));
    try {
      const body =
        kind === "health"
          ? {
              environment: resourceScanForm.environment.trim(),
              resource_ids: splitCSV(resourceScanForm.resourceIDs),
              approved: resourceScanForm.approved,
            }
          : {};
      const payload = await postJSON<ResourceScanEnvelope>(
        `/api/projects/${snapshot.project.id}/resources/${kind === "maintenance" ? "maintenance/scan" : kind === "lifecycle" ? "lifecycle/scan" : "health-scan"}`,
        body,
      );
      const id =
        payload.lifecycle_scan?.id ||
        payload.health_scan?.id ||
        payload.maintenance_records?.[0]?.id ||
        `${kind}-scan`;
      const status = payload.lifecycle_scan?.status || payload.health_scan?.status || "completed";
      const decision = payload.lifecycle_scan?.decision || payload.health_scan?.decision || `${kind}_scan_completed`;
      setResourceActionState((current) => ({
        ...current,
        [actionKey]: {
          status: status === "completed" || status === "healthy" ? "completed" : "blocked",
          id,
          message: decision,
        },
      }));
      setResourceScanModal(null);
      router.refresh();
    } catch (error) {
      setResourceActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "资源扫描失败。" },
      }));
    }
  }

  async function disableResource(resourceID: string) {
    const actionKey = `${resourceID}:disable`;
    setResourceActionState((current) => ({ ...current, [actionKey]: { status: "running", message: "正在禁用资源..." } }));
    try {
      const payload = await postJSON<ResourceDisableEnvelope>(`/api/projects/${snapshot.project.id}/resources/${encodeURIComponent(resourceID)}/disable`, {});
      if (!payload.resource) {
        throw new Error(payload.error ?? "资源禁用没有返回记录。");
      }
      setResourceActionState((current) => ({
        ...current,
        [actionKey]: { status: payload.resource?.status === "disabled" ? "completed" : "blocked", id: payload.resource?.id, message: payload.resource?.status ?? "已禁用" },
      }));
      setResourceDisableModalID(null);
      router.refresh();
    } catch (error) {
      setResourceActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "资源禁用失败。" },
      }));
    }
  }

  async function createBatchPlan() {
    if (!requireFields("batchPlan", [["Epic ID", batchPlanForm.epicID], ["模式", batchPlanForm.mode], ["请求人", batchPlanForm.requestedBy]])) {
      setBatchActionState((current) => ({ ...current, plan: { status: "error", message: "请先补齐批量计划信息。" } }));
      return;
    }
    setBatchActionState((current) => ({ ...current, plan: { status: "running", message: "正在创建批量计划..." } }));
    try {
      const payload = await postJSON<BatchPlanEnvelope>(
        `/api/projects/${snapshot.project.id}/epics/${encodeURIComponent(batchPlanForm.epicID.trim())}/batches/plan`,
        {
          mode: batchPlanForm.mode.trim(),
          max_parallel: parseOptionalInt(batchPlanForm.maxParallel),
          requested_by: batchPlanForm.requestedBy.trim(),
        },
      );
      const plan = payload.batch_plan;
      if (!plan) {
        throw new Error(payload.error ?? "批量计划没有返回记录。");
      }
      setBatchActionState((current) => ({
        ...current,
        plan: { status: plan.status === "planned" ? "completed" : "blocked", id: plan.id, message: decisionLabel(plan.decision ?? plan.status ?? "planned") },
      }));
      setBatchPlanModalOpen(false);
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({ ...current, plan: { status: "error", message: error instanceof Error ? error.message : "批量计划创建失败。" } }));
    }
  }

  async function runBatchDryRun(batchID: string) {
    setBatchActionState((current) => ({
      ...current,
      [batchID]: { status: "running", message: "正在创建 dry-run 运行..." },
    }));
    try {
      const payload = await postJSON<BatchRunEnvelope>(`/api/projects/${snapshot.project.id}/batches/${encodeURIComponent(batchID)}/run`, {
        mode: "dry_run",
        requested_by: "console",
      });
      const run = payload.batch_run;
      if (!run) {
        throw new Error(payload.error ?? "Batch dry run returned no run.");
      }
      setBatchActionState((current) => ({
        ...current,
        [batchID]: {
          status: run.status === "completed" ? "completed" : "blocked",
          id: run.id,
          message: decisionLabel(run.decision ?? run.status ?? "completed"),
        },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [batchID]: { status: "error", message: error instanceof Error ? error.message : "批量 dry-run 失败。" },
      }));
    }
  }

  async function buildMergeQueue(batchID: string) {
    const actionKey = `${batchID}:merge`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Building merge queue..." },
    }));
    try {
      const payload = await postJSON<MergeQueueEnvelope>(`/api/projects/${snapshot.project.id}/batches/${encodeURIComponent(batchID)}/merge-queue`, {});
      const queue = payload.merge_queue;
      if (!queue) {
        throw new Error(payload.error ?? "Merge queue returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: {
          status: queue.status === "ready_to_merge" ? "completed" : queue.status === "needs_rework" ? "blocked" : "blocked",
          id: queue.id,
          message: queue.decision ?? queue.status,
        },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Merge queue failed." },
      }));
    }
  }

  async function buildIntegrationPreview(queueID: string) {
    const actionKey = `${queueID}:integration-preview`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Creating integration preview..." },
    }));
    try {
      const payload = await postJSON<IntegrationPreviewEnvelope>(
        `/api/projects/${snapshot.project.id}/merge-queues/${encodeURIComponent(queueID)}/integration-preview`,
        {},
      );
      const preview = payload.integration_preview;
      if (!preview) {
        throw new Error(payload.error ?? "Integration preview returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: {
          status: preview.status === "ready" ? "completed" : "blocked",
          id: preview.id,
          message: preview.decision ?? preview.status,
        },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Integration preview failed." },
      }));
    }
  }

  async function dryRunIntegrationApply(previewID: string) {
    const actionKey = `${previewID}:apply-dry-run`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Planning integration apply..." },
    }));
    try {
      const payload = await postJSON<IntegrationApplyEnvelope>(`/api/projects/${snapshot.project.id}/integration-previews/${encodeURIComponent(previewID)}/apply`, {
        mode: "dry_run",
        requested_by: "console",
      });
      const apply = payload.integration_apply;
      if (!apply) {
        throw new Error(payload.error ?? "Integration apply returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: {
          status: apply.status === "planned" || apply.status === "applied" ? "completed" : "blocked",
          id: apply.id,
          message: apply.decision ?? apply.status,
        },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Integration apply failed." },
      }));
    }
  }

  async function planReleaseBatch(applyID: string) {
    const actionKey = `${applyID}:release-batch`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Checking release batch readiness..." },
    }));
    try {
      const payload = await postJSON<ReleaseBatchEnvelope>(`/api/projects/${snapshot.project.id}/integration-applies/${encodeURIComponent(applyID)}/release-batch`, {
        min_items: 3,
        requested_by: "console",
      });
      const releaseBatch = payload.release_batch;
      if (!releaseBatch) {
        throw new Error(payload.error ?? "Release batch returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: {
          status: releaseBatch.status === "suggested" ? "completed" : "blocked",
          id: releaseBatch.id,
          message: `${releaseBatch.decision ?? releaseBatch.status}${releaseBatch.ready_item_count ? ` / ${releaseBatch.ready_item_count} ready` : ""}`,
        },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Release batch check failed." },
      }));
    }
  }

  async function planReleaseCandidate(releaseBatchID: string) {
    const actionKey = `${releaseBatchID}:candidate`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Planning release candidate..." },
    }));
    try {
      const payload = await postJSON<ReleaseCandidateEnvelope>(`/api/projects/${snapshot.project.id}/release-batches/${encodeURIComponent(releaseBatchID)}/candidate`, {
        deployment_targets: ["test_dev"],
        requested_by: "console",
      });
      const candidate = payload.release_candidate;
      if (!candidate) {
        throw new Error(payload.error ?? "Release candidate returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: candidate.status === "ready" ? "completed" : "blocked", id: candidate.id, message: candidate.decision ?? candidate.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Release candidate plan failed." },
      }));
    }
  }

  async function dryRunReleaseCandidateApply(candidateID: string) {
    const actionKey = `${candidateID}:release-branch-apply`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Planning release branch apply..." },
    }));
    try {
      const payload = await postJSON<ReleaseCandidateApplyEnvelope>(`/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/apply`, {
        mode: "dry_run",
        requested_by: "console",
      });
      const apply = payload.release_candidate_apply;
      if (!apply) {
        throw new Error(payload.error ?? "Release branch apply returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: apply.status === "planned" || apply.status === "applied" ? "completed" : "blocked", id: apply.id, message: apply.decision ?? apply.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Release branch apply failed." },
      }));
    }
  }

  async function previewReleaseCandidateProvider(candidateID: string) {
    const actionKey = `${candidateID}:provider-preview`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Creating provider preview..." },
    }));
    try {
      const payload = await postJSON<ReleaseCandidateProviderPreviewEnvelope>(
        `/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/provider-preview`,
        {},
      );
      const preview = payload.release_candidate_provider_preview;
      if (!preview) {
        throw new Error(payload.error ?? "Release candidate provider preview returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: preview.status === "completed" ? "completed" : "blocked", id: preview.id, message: preview.decision ?? preview.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Provider preview failed." },
      }));
    }
  }

  async function createCandidateDeploymentPlan(candidateID: string) {
    const actionKey = `${candidateID}:deployment-plan`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Creating deployment handoff..." },
    }));
    try {
      const payload = await postJSON<DeploymentPlanEnvelope>(`/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/deployment-plan`, {
        environment: "test_dev",
        approved: true,
      });
      const deployment = payload.deployment;
      if (!deployment) {
        throw new Error(payload.error ?? "Deployment handoff returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: deployment.status === "planned" ? "completed" : "blocked", id: deployment.id, message: deployment.decision ?? deployment.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Deployment handoff failed." },
      }));
    }
  }

  async function publishReleaseCandidateProvider(candidateID: string) {
    const actionKey = `${candidateID}:provider-publish`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Checking publish gate..." },
    }));
    try {
      const payload = await postJSON<ReleaseProviderExecutionEnvelope>(
        `/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/provider-publish`,
        {},
      );
      const execution = payload.release_provider_execution;
      if (!execution) {
        throw new Error(payload.error ?? "Candidate publish returned no execution.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: execution.status === "completed" ? "completed" : "blocked", id: execution.id, message: execution.decision ?? execution.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Candidate publish failed." },
      }));
    }
  }

  async function planCandidatePRMR(candidateID: string) {
    const actionKey = `${candidateID}:pr-mr-plan`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Planning PR/MR..." },
    }));
    try {
      const payload = await postJSON<GitProviderActionEnvelope>(`/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/pr-mr-plan`, {});
      const plan = payload.git_provider_plan;
      if (!plan) {
        throw new Error(payload.error ?? "Candidate PR/MR plan returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: plan.status === "pr_mr_plan_ready" ? "completed" : "blocked", id: plan.id, message: plan.decision ?? plan.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Candidate PR/MR plan failed." },
      }));
    }
  }

  async function runCandidateDeploymentDryRun(candidateID: string) {
    const actionKey = `${candidateID}:deployment-execution`;
    setBatchActionState((current) => ({
      ...current,
      [actionKey]: { status: "running", message: "Running deploy dry-run..." },
    }));
    try {
      const payload = await postJSON<DeploymentExecutionEnvelope>(
        `/api/projects/${snapshot.project.id}/release-candidates/${encodeURIComponent(candidateID)}/deployment-execution`,
        { mode: "dry_run", environment: "test_dev" },
      );
      const execution = payload.execution;
      if (!execution) {
        throw new Error(payload.error ?? "Candidate deployment execution returned no result.");
      }
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: execution.status === "completed" ? "completed" : "blocked", id: execution.id, message: execution.decision ?? execution.status },
      }));
      router.refresh();
    } catch (error) {
      setBatchActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Candidate deployment dry-run failed." },
      }));
    }
  }

  async function createIssueMergeDecision(issueID: string) {
    const actionKey = `${issueID}:merge-decision`;
    setIssueActionState((current) => ({ ...current, [actionKey]: { status: "running", message: "正在生成合并决策..." } }));
    try {
      const payload = await postJSON<MergeDecisionEnvelope>(`/api/projects/${snapshot.project.id}/issues/${encodeURIComponent(issueID)}/merge-decision`, {});
      const decision = payload.merge_decision;
      if (!decision) {
        throw new Error(payload.error ?? "合并决策没有返回记录。");
      }
      setIssueActionState((current) => ({
        ...current,
        [actionKey]: { status: decision.status === "ready_to_merge" ? "completed" : "blocked", id: decision.id, message: decision.decision ?? decision.status },
      }));
      router.refresh();
    } catch (error) {
      setIssueActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "合并决策生成失败。" },
      }));
    }
  }

  async function createIssueGitProviderPlan(issueID: string) {
    const actionKey = `${issueID}:git-provider-plan`;
    setIssueActionState((current) => ({ ...current, [actionKey]: { status: "running", message: "正在创建 Git Provider 计划..." } }));
    try {
      const payload = await postJSON<GitProviderActionEnvelope>(`/api/projects/${snapshot.project.id}/issues/${encodeURIComponent(issueID)}/git-provider-plan`, {});
      const plan = payload.git_provider_plan;
      if (!plan) {
        throw new Error(payload.error ?? "Git Provider 计划没有返回记录。");
      }
      setIssueActionState((current) => ({
        ...current,
        [actionKey]: { status: isGitActionCompleted(plan) ? "completed" : "blocked", id: plan.id, message: plan.decision ?? plan.status },
      }));
      router.refresh();
    } catch (error) {
      setIssueActionState((current) => ({
        ...current,
        [actionKey]: { status: "error", message: error instanceof Error ? error.message : "Git Provider 计划创建失败。" },
      }));
    }
  }

  async function createOperationRepairCandidate() {
    if (!selectedOperation) {
      setOperationRepairCreateState({ status: "error", message: "请先选择一个操作。" });
      return;
    }
    const operationType = selectedOperationDetail?.operation_type || selectedOperation.type;
    const operationID = selectedOperationDetail?.id || selectedOperation.id;
    if (!operationType || !operationID) {
      setOperationRepairCreateState({ status: "error", message: "当前操作缺少类型或 ID。" });
      return;
    }
    setOperationRepairCreateState({ status: "running", message: "正在生成修复候选..." });
    try {
      const payload = await postJSON<OperationRepairCandidateEnvelope>(
        `/api/projects/${snapshot.project.id}/operations/${encodeURIComponent(operationType)}/${encodeURIComponent(operationID)}/repair-candidate`,
        {},
      );
      const candidate = payload.operation_repair_candidate;
      if (!candidate) {
        throw new Error(payload.error ?? "修复候选没有返回记录。");
      }
      setOperationRepairCreateState({
        status: candidate.status === "review_required" ? "completed" : "blocked",
        id: candidate.id,
        message: `${candidate.decision ?? candidate.status}${candidate.failure_class ? ` / ${candidate.failure_class}` : ""}`,
      });
      router.refresh();
    } catch (error) {
      setOperationRepairCreateState({ status: "error", message: error instanceof Error ? error.message : "修复候选创建失败。" });
    }
  }

  async function searchMemory() {
    const query = memorySearchForm.query.trim();
    if (!requireFields("memorySearch", [["查询", query]])) {
      setMemoryActionState({ status: "error", message: "请先填写查询内容。" });
      return;
    }
    setMemoryActionState({ status: "running", message: "正在搜索 Memory..." });
    try {
      const response = await fetch(
        `/api/projects/${snapshot.project.id}/memory/search?q=${encodeURIComponent(query)}&limit=${encodeURIComponent(memorySearchForm.limit.trim() || "10")}`,
      );
      const payload = (await response.json()) as MemorySearchEnvelope;
      if (!response.ok || !payload.records) {
        throw new Error(payload.error ?? "Memory 搜索失败。");
      }
      const records = payload.records.map(normalizeMemoryRecord);
      setMemorySearchResults(records);
      setMemoryActionState({ status: records.length > 0 ? "completed" : "blocked", message: `${records.length} 条记录` });
      setMemorySearchModalOpen(false);
    } catch (error) {
      setMemoryActionState({ status: "error", message: error instanceof Error ? error.message : "Memory 搜索失败。" });
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
            <span>控制台</span>
          </div>
        </div>

        <nav className="navList">
          {navGroups.map((item) => {
            const active = groupHasView(item, activeView);
            return (
              <button className={`navItem ${active ? "active" : ""}`} key={item.label} onClick={() => selectGroup(item)} type="button">
                <item.icon size={17} />
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>

        <div className="sideCard">
          <div className="sideCardTop">
            <span>控制台服务</span>
            <StatusPill tone={snapshot.backendStatus} label={snapshot.mode === "live" ? "live" : "demo"} />
          </div>
          <strong>3000 / 8080</strong>
          <small>控制台前端 / 控制平面 API</small>
        </div>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">AI 工程控制台</p>
            <h1>{snapshot.project.name}</h1>
            <span className="projectRoot">{shortPath(snapshot.project.root)}</span>
          </div>
          <div className="topActions">
            <details className="projectMenu">
              <summary aria-label="切换当前项目">
                <span className="projectMenuIcon">
                  <Boxes size={15} />
                </span>
                <span className="projectMenuText">
                  <span>当前项目</span>
                  <strong>{snapshot.project.name || snapshot.project.id}</strong>
                </span>
                <ChevronRight className="projectMenuChevron" size={15} />
              </summary>
              <div className="projectMenuPanel">
                {orderedProjects.map((project) => {
                  const active = project.id === snapshot.project.id;
                  return (
                    <button className={`projectMenuOption ${active ? "active" : ""}`} key={project.id} onClick={() => selectProject(project.id)} type="button">
                      <span className="projectMenuOptionText">
                        <strong>{project.name || project.id}</strong>
                        <span>{shortPath(project.root)}</span>
                      </span>
                      {active ? <CheckCircle2 size={15} /> : <StatusPill tone={toneForStatus(project.status)} label={project.status || "unknown"} />}
                    </button>
                  );
                })}
              </div>
            </details>
            <div className="searchBox">
              <Search size={16} />
              <span>跳转到 issue、run、provider...</span>
            </div>
            <button className="iconButton" type="button" aria-label="运行选中的 issue">
              <Play size={18} />
            </button>
          </div>
        </header>

        {activeNavGroup.views.length > 1 ? (
          <div className="viewTabs" role="tablist" aria-label={`${activeNavGroup.label}功能`}>
            {activeNavGroup.views.map((view) => (
              <button
                aria-selected={activeView === view}
                className={`viewTab ${activeView === view ? "active" : ""}`}
                key={view}
                onClick={() => selectView(view)}
                role="tab"
                type="button"
              >
                {viewTabLabels[view]}
              </button>
            ))}
          </div>
        ) : null}

        <section className="heroGrid" hidden={activeView !== "项目"}>
          <MetricCard label="Issues" value={snapshot.stats.issues} tone="neutral" detail={`${snapshot.stats.accepted} 已接受`} />
          <MetricCard label="阻断项" value={snapshot.stats.blocked} tone={snapshot.stats.blocked > 0 ? "warning" : "ok"} detail="依赖 / 审批" />
          <MetricCard label="Provider" value={snapshot.stats.providers} tone="running" detail="Claude / Codex / API" />
          <MetricCard label="部署" value={snapshot.stats.executions} tone="neutral" detail={`${snapshot.stats.deployments} 个计划`} />
        </section>

        <section className="opsGrid singlePanelGrid" hidden={!viewVisible(activeView, ["项目", "需求登记", "部署", "测试验证"])}>
          <div className="panel requirementPanel" hidden={activeView !== "需求登记"}>
            <PanelTitle icon={<Sparkles size={18} />} title="需求登记" meta="登记后拆分为受控 Issue" />
            <form className="requirementForm" onSubmit={submitRequirement}>
              <label className="textareaField">
                <FieldLabel required>需求描述</FieldLabel>
                <textarea
                  aria-label="需求描述"
                  onChange={(event) => setRequirementText(event.target.value)}
                  placeholder="描述一个功能、修复或运维变更，并写清期望的验证方式..."
                  required
                  rows={3}
                  value={requirementText}
                />
              </label>
              <div className="formFooter">
                <button className="primaryButton" disabled={requirementState.status === "planning"} type="submit">
                  <Play size={16} />
                  <span>{requirementState.status === "planning" ? "规划中" : "规划 Issues"}</span>
                </button>
                {requirementState.status !== "idle" ? (
                  <div className={`formResult ${requirementState.status}`}>
                    <strong>{requirementState.status.replaceAll("_", " ")}</strong>
                    <span>{requirementState.message}</span>
                    {requirementState.epic ? <code>{requirementState.epic}</code> : null}
                    {requirementState.status === "needs_user_input" ? (
                      <button className="inlineActionButton compactButton" onClick={() => setClarificationModalOpen(true)} type="button">
                        <ScrollText size={13} />
                        <span>补充回答</span>
                      </button>
                    ) : null}
                  </div>
                ) : null}
              </div>
              <SchemaFeedback errors={schemaErrors.requirement} />
            </form>
          </div>

          <div className="panel requirementLedgerPanel" hidden={activeView !== "需求登记"}>
            <PanelTitle icon={<ScrollText size={18} />} title="需求记录" meta={`${completedRequirements.length} 已完成 / ${snapshot.requirements.length} 总数`} />
            <div className="signalList compactList">
              {snapshot.requirements.length > 0 ? (
                orderedRequirements.slice(0, 6).map((requirement) => {
                  const managedRuns = requirement.issues.flatMap((issue) => runsByIssue.get(issue.id) ?? []);
                  const latestCommitRun = managedRuns.find((run) => run.commit_after);
                  return (
                    <div className="signalItem compactSignal requirementRecord" key={requirement.id}>
                      <div className="signalHeader">
                        <strong>{requirement.title}</strong>
                        <StatusPill tone={toneForStatus(requirement.status)} label={statusLabel(requirement.status)} />
                      </div>
                      <span>{requirement.raw_text || requirement.clarified_requirement}</span>
                      <div className="signalMeta">
                        <code>{requirement.issue_count} 个 issue</code>
                        <code>{requirement.accepted_count} 已接受</code>
                        {requirement.blocked_count > 0 ? <code>{requirement.blocked_count} 阻断</code> : null}
                        <code>{requirement.commit_count} commits</code>
                      </div>
                      {latestCommitRun ? (
                        <div className="commitLine">
                          <GitBranch size={13} />
                          <span>{latestCommitRun.commit_after ? compactCommit(latestCommitRun.commit_after) : "未产生 commit"}</span>
                          {latestCommitRun.changed_files?.length ? <code>{latestCommitRun.changed_files.length} files</code> : null}
                        </div>
                      ) : null}
                      <div className="requirementIssueList">
                        {requirement.issues.slice(0, 4).map((issue) => (
                          <span key={issue.id} title={cleanDisplayID(issue.id)}>
                            <StatusDot tone={toneForStatus(issue.status)} />
                            {issue.title || compactID(issue.id)}
                          </span>
                        ))}
                      </div>
                    </div>
                  );
                })
              ) : (
                <div className="emptyState">暂无 Moyuan 登记的需求</div>
              )}
            </div>
          </div>

          <div className="panel projectOnboardingPanel" hidden={activeView !== "项目"}>
            <PanelTitle
              icon={<Boxes size={18} />}
              title="项目接入"
              meta="本地目录 / GitHub / Gitee"
              action={
                <button className="inlineActionButton" disabled={projectActionState.status === "running"} onClick={() => setProjectModalOpen(true)} type="button">
                  <Boxes size={13} />
                  <span>{projectActionState.status === "running" ? "接入中" : "接入项目"}</span>
                </button>
              }
            />
            <div className="signalList compactList">
              {orderedProjects.slice(0, 4).map((project) => (
                <div className="signalItem compactSignal" key={project.id}>
                  <div className="signalHeader">
                    <strong>{project.name || project.id}</strong>
                    <div className="signalActions">
                      <StatusPill tone={project.id === snapshot.project.id ? "ok" : toneForStatus(project.status)} label={project.status || "unknown"} />
                      {project.id !== snapshot.project.id ? (
                        <button className="inlineActionButton" onClick={() => selectProject(project.id)} type="button">
                          切换
                        </button>
                      ) : null}
                    </div>
                  </div>
                  <div className="projectCardMeta">
                    <span>项目名称</span>
                    <strong>{project.name || project.id}</strong>
                    <span>Git 地址</span>
                    <code>{projectGitAddress(project)}</code>
                    <span>本机路径</span>
                    <code>{projectLocalPath(project)}</code>
                    <span>技术栈</span>
                    <code>{projectTechStack(project)}</code>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="panel executionPanel" hidden={!viewVisible(activeView, ["部署", "测试验证"])}>
            <PanelTitle
              icon={activeView === "测试验证" ? <ShieldCheck size={18} /> : <Rocket size={18} />}
              title={activeView === "测试验证" ? "测试与验证执行" : "部署执行"}
              meta={`${snapshot.executions.length} 条最近记录`}
            />
            <div className="deploymentControls">
              <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} onClick={() => void suggestRelease()} type="button">
                <GitBranch size={13} />
                <span>建议 Release</span>
              </button>
              <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} onClick={() => setDeploymentPlanModalOpen(true)} type="button">
                <Layers3 size={13} />
                <span>部署计划</span>
              </button>
              <button
                className="inlineActionButton"
                disabled={deploymentActionState.status === "running" || !deploymentExecuteForm.deploymentID}
                onClick={() => setDeploymentExecuteModalOpen(true)}
                type="button"
              >
                <Play size={13} />
                <span>执行部署</span>
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
                <span>健康扫描</span>
              </button>
              <button
                className="inlineActionButton"
                disabled={deploymentActionState.status === "running" || !latestRollbackCandidate}
                onClick={() => void previewRollbackExecution(latestRollbackCandidate?.id)}
                type="button"
              >
                <AlertTriangle size={13} />
                <span>回滚预览</span>
              </button>
              <button
                className="inlineActionButton"
                disabled={deploymentActionState.status === "running"}
                onClick={() => void summarizeDeploymentMonitor()}
                type="button"
              >
                <Activity size={13} />
                <span>监控摘要</span>
              </button>
              <button
                className="inlineActionButton"
                disabled={deploymentActionState.status === "running" || snapshot.executions.length === 0}
                onClick={() => void createPostDeploymentVerification()}
                type="button"
              >
                <ShieldCheck size={13} />
                <span>验证</span>
              </button>
              <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} onClick={() => void createDeploymentRehearsal()} type="button">
                <CircleDotDashed size={13} />
                <span>演练</span>
              </button>
              <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} onClick={() => void runRehearsalScheduler()} type="button">
                <RefreshCw size={13} />
                <span>调度</span>
              </button>
              <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} onClick={() => void createReleaseAdmission()} type="button">
                <ShieldCheck size={13} />
                <span>准入</span>
              </button>
              <button
                className="inlineActionButton"
                disabled={deploymentActionState.status === "running" || !latestAdmission}
                onClick={() => void createDeploymentRiskHandoff()}
                type="button"
              >
                <Wrench size={13} />
                <span>风险交接</span>
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
                      <span>
                        {[
                          execution.decision,
                          execution.smoke_status ? `smoke ${execution.smoke_status}` : "",
                          execution.monitor_status ? `monitor ${execution.monitor_status}` : "",
                          execution.rollback_required ? "建议回滚" : "",
                          execution.approval_id ? `审批 ${compactID(execution.approval_id)}` : "",
                          execution.approval_consumed ? "审批已消费" : "",
                        ]
                          .filter(Boolean)
                          .join(" / ")}
                      </span>
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
                <div className="emptyState">还没有部署执行记录</div>
              )}
            </div>
            <div className="maintenanceList">
              {latestMonitorSummary ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{latestMonitorSummary.environment || "全部环境"}</strong>
                    <span>{`${latestMonitorSummary.decision} / ${latestMonitorSummary.history_count} 条历史 / ${latestMonitorSummary.failed_count} 失败`}</span>
                    <span>{`${latestMonitorSummary.rollback_count} 个回滚 / 窗口 ${latestMonitorSummary.window_size}`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestMonitorSummary.status)} label={latestMonitorSummary.status} />
                </div>
              ) : null}
              {latestVerification ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestVerification.execution_id || latestVerification.id)}</strong>
                    <span>{`${latestVerification.decision} / ${latestVerification.monitor_decision || "监控待生成"}`}</span>
                    <span>
                      {latestVerification.risk_handoff_recommended
                        ? `${latestVerification.risk_source_type || "risk"} ${compactID(latestVerification.risk_source_id || "")}`
                        : "不需要风险交接"}
                    </span>
                  </div>
                  <StatusPill tone={toneForStatus(latestVerification.risk_handoff_recommended ? "warning" : latestVerification.status)} label={latestVerification.status} />
                </div>
              ) : null}
              {latestRehearsal ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestRehearsal.id)}</strong>
                    <span>{`${latestRehearsal.decision} / ${latestRehearsal.timeline.length} 个步骤`}</span>
                    <span>{`${latestRehearsal.monitor_status || "监控待生成"} / ${latestRehearsal.rollback_decision || "回滚待生成"}`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestRehearsal.status)} label={latestRehearsal.status} />
                </div>
              ) : null}
              {latestAdmission ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestAdmission.id)}</strong>
                    <span>{`${latestAdmission.decision} / ${latestAdmission.signals.length} 个信号`}</span>
                    <span>{`${latestAdmission.policy_id || snapshot.release_admission_policy?.id || "策略待生成"} / ${latestAdmission.matched_rules.length} 条命中规则`}</span>
                    <span>{latestAdmission.policy_decision?.reasons[0] || latestAdmission.reasons[0] || "原因待生成"}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestAdmission.status)} label={latestAdmission.status} />
                </div>
              ) : null}
              {latestSchedulerRun ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestSchedulerRun.id)}</strong>
                    <span>{`${latestSchedulerRun.decision} / 创建 ${latestSchedulerRun.created_count} / 跳过 ${latestSchedulerRun.skipped_count}`}</span>
                    <span>{`${latestSchedulerRun.blocked_count} 阻断 / ${latestSchedulerRun.manual_count} 人工 / ${latestSchedulerRun.targets[0]?.reason || "目标原因待生成"}`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestSchedulerRun.status)} label={latestSchedulerRun.status} />
                </div>
              ) : null}
              {latestRiskHandoff ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestRiskHandoff.id)}</strong>
                    <span>{`${latestRiskHandoff.decision} / ${latestRiskHandoff.failure_class}`}</span>
                    <span>{`${latestRiskHandoff.review_decision || (latestRiskHandoff.review_required ? "待复核" : "无需复核")} / ${
                      latestRiskHandoff.repair_plan_id ? compactID(latestRiskHandoff.repair_plan_id) : "无需修复"
                    }`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestRiskHandoff.status)} label={latestRiskHandoff.status} />
                  {latestRiskHandoff.review_required && !latestRiskHandoff.review_id ? (
                    <div className="rowActions">
                      <button
                        className="inlineActionButton"
                        disabled={deploymentActionState.status === "running"}
                        onClick={() => setDeploymentRiskReviewModal({ handoffID: latestRiskHandoff.id, decision: "approved" })}
                        type="button"
                      >
                        <CheckCircle2 size={13} />
                        <span>批准</span>
                      </button>
                      <button
                        className="inlineActionButton danger"
                        disabled={deploymentActionState.status === "running"}
                        onClick={() => setDeploymentRiskReviewModal({ handoffID: latestRiskHandoff.id, decision: "rejected" })}
                        type="button"
                      >
                        <AlertTriangle size={13} />
                        <span>拒绝</span>
                      </button>
                    </div>
                  ) : null}
                </div>
              ) : null}
              {latestRiskReviewQueueItem ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestRiskReviewQueueItem.handoff_id)}</strong>
                    <span>{`${latestRiskReviewQueueItem.decision} / ${latestRiskReviewQueueItem.failure_class}`}</span>
                    <span>{`${latestRiskReviewQueueItem.review_decision || "待处理"} / ${latestRiskReviewQueueItem.review_next_step || latestRiskReviewQueueItem.reasons[0] || "下一步待定"}`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestRiskReviewQueueItem.status)} label={latestRiskReviewQueueItem.status} />
                  {latestRiskReviewQueueItem.review_required && !latestRiskReviewQueueItem.review_id ? (
                    <div className="rowActions">
                      <button
                        className="inlineActionButton"
                        disabled={deploymentActionState.status === "running"}
                        onClick={() => setDeploymentRiskReviewModal({ handoffID: latestRiskReviewQueueItem.handoff_id, decision: "approved" })}
                        type="button"
                      >
                        <CheckCircle2 size={13} />
                        <span>批准</span>
                      </button>
                      <button
                        className="inlineActionButton danger"
                        disabled={deploymentActionState.status === "running"}
                        onClick={() => setDeploymentRiskReviewModal({ handoffID: latestRiskReviewQueueItem.handoff_id, decision: "rejected" })}
                        type="button"
                      >
                        <AlertTriangle size={13} />
                        <span>拒绝</span>
                      </button>
                    </div>
                  ) : null}
                </div>
              ) : null}
              {latestRiskReview ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{compactID(latestRiskReview.id)}</strong>
                    <span>{`${latestRiskReview.decision} / ${latestRiskReview.next_step || "下一步待定"}`}</span>
                    <span>{latestRiskReview.reason || latestRiskReview.failure_class || "复核原因待生成"}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestRiskReview.status)} label={latestRiskReview.status} />
                </div>
              ) : null}
              {snapshot.rollback_executions.slice(0, 2).map((rollback) => (
                <div className="maintenanceItem" key={rollback.id}>
                  <div>
                    <strong>{compactID(rollback.execution_id)}</strong>
                    <span>{`${rollback.decision} / ${rollback.mode} / ${rollback.step_count} 个步骤`}</span>
                    <span>{rollback.approval_id ? `审批 ${compactID(rollback.approval_id)}` : "未消费审批"}</span>
                  </div>
                  <StatusPill tone={toneForStatus(rollback.status)} label={rollback.status} />
                </div>
              ))}
              {snapshot.post_deployment_histories.length > 0 ? (
                snapshot.post_deployment_histories.slice(0, 3).map((history) => (
                  <div className="maintenanceItem" key={history.id}>
                    <div>
                      <strong>{compactID(history.execution_id)}</strong>
                      <span>{`${history.decision} / ${history.checks.length} 个检查 / 回滚 ${history.rollback.status}`}</span>
                      {history.checks[0]?.template_id ? (
                        <span>{`${history.checks[0].template_id}${history.severity ? ` / ${history.severity}` : ""}`}</span>
                      ) : null}
                    </div>
                    <StatusPill tone={toneForStatus(history.failure_class === "none" ? history.status : history.failure_class)} label={history.failure_class} />
                  </div>
                ))
              ) : (
                <div className="emptyState compact">
                  {hasDeploymentOpsHistory ? "暂无部署后历史" : "暂无部署运维历史"}
                </div>
              )}
            </div>
          </div>
        </section>

        <section className="operationGrid" hidden={activeView !== "操作"}>
          <div className="panel operationHistoryPanel">
            <PanelTitle icon={<ScrollText size={18} />} title="操作历史" meta={`${snapshot.operation_history.length} 条追踪`} />
            <div className="operationList">
              {snapshot.operation_history.length > 0 ? (
                snapshot.operation_history.map((operation) => (
                  <button
                    className={`operationItem ${selectedOperation?.id === operation.id ? "selected" : ""}`}
                    key={operation.id}
                    onClick={() => setSelectedOperationID(operation.id)}
                    type="button"
                  >
                    <StatusDot tone={operation.tone} />
                    <div>
                      <strong>{operation.title}</strong>
                      <span>{operation.detail}</span>
                    </div>
                    <time>{operation.time}</time>
                  </button>
                ))
              ) : (
                <div className="emptyState">暂无操作历史</div>
              )}
            </div>
          </div>

          <div className="panel operationDetailPanel">
            <PanelTitle icon={<Activity size={18} />} title="执行详情" meta={selectedOperationDetail?.operation_type ?? selectedOperation?.type ?? "operation"} />
            {selectedOperation ? (
              <div className="operationDetail">
                <div className="detailHeader">
                  <div>
                    <strong>{selectedOperation.title}</strong>
                    <span>{selectedOperationDetail?.operation || selectedOperation.id}</span>
                  </div>
                  <div className="detailHeaderActions">
                    <StatusPill tone={toneForStatus(selectedOperationDetail?.status || selectedOperation.status)} label={selectedOperationDetail?.status || selectedOperation.status} />
                    <button
                      className="inlineActionButton compactButton"
                      disabled={operationRepairCreateState.status === "running"}
                      onClick={() => void createOperationRepairCandidate()}
                      type="button"
                    >
                      <Wrench size={13} />
                      <span>{operationRepairCreateState.status === "running" ? "生成中" : "修复候选"}</span>
                    </button>
                    <button aria-label="刷新操作详情" className="iconActionButton" onClick={() => router.refresh()} type="button">
                      <RefreshCw size={14} />
                    </button>
                  </div>
                </div>
                <ActionFeedback state={operationRepairCreateState} />
                <dl>
                  <div>
                    <dt>决策</dt>
                    <dd>{selectedOperationDetail?.decision || selectedOperation.decision}</dd>
                  </div>
                  <div>
                    <dt>主引用</dt>
                    <dd>{selectedOperationDetail?.primary_ref || selectedOperation.primary_ref || "无"}</dd>
                  </div>
                  <div>
                    <dt>次引用</dt>
                    <dd>{selectedOperationDetail?.secondary_ref || selectedOperation.secondary_ref || "无"}</dd>
                  </div>
                  <div>
                    <dt>证据</dt>
                    <dd>{selectedEvidenceRecords.length > 0 ? selectedEvidenceRecords.map((record) => compactID(record.id)).join(", ") : "无"}</dd>
                  </div>
                </dl>
                {(selectedOperationDetail?.reasons.length ? selectedOperationDetail.reasons : selectedOperation.reasons).length > 0 ? (
                  <div className="detailChips">
                    {(selectedOperationDetail?.reasons.length ? selectedOperationDetail.reasons : selectedOperation.reasons).slice(0, 3).map((reason) => (
                      <code key={reason}>{reason}</code>
                    ))}
                  </div>
                ) : null}
                {selectedOperation.metadata.length > 0 ? (
                  <div className="detailChips subtle">
                    {selectedOperationDetail ? <code>详情 API</code> : null}
                    {selectedOperationDetail?.summary.evidence_count ? <code>{selectedOperationDetail.summary.evidence_count} evidence</code> : null}
                    {selectedOperationDetail?.summary.artifact_count ? <code>{selectedOperationDetail.summary.artifact_count} artifacts</code> : null}
                    {selectedOperation.metadata.map((item) => (
                      <code key={item}>{item}</code>
                    ))}
                  </div>
                ) : null}
                <div className="evidenceDrilldown">
                  <div className="detailSectionTitle">
                    <strong>证据链</strong>
                    <span>{selectedEvidenceRecords.length} 条记录</span>
                  </div>
                  {selectedEvidenceRecords.length > 0 ? (
                    selectedEvidenceRecords.map((record) => (
                      <div className="evidenceCard" key={record.id}>
                        <div className="evidenceCardHeader">
                          <div>
                            <strong>{record.operation}</strong>
                            <span>{record.id}</span>
                          </div>
                          <StatusPill tone={toneForStatus(record.status)} label={record.status} />
                        </div>
                        <dl>
                          <div>
                            <dt>决策</dt>
                            <dd>{record.decision}</dd>
                          </div>
                          <div>
                            <dt>产物</dt>
                            <dd>{record.artifact_count}</dd>
                          </div>
                        </dl>
                        {record.reasons.length > 0 ? (
                          <div className="detailChips">
                            {record.reasons.slice(0, 3).map((reason) => (
                              <code key={`${record.id}-${reason}`}>{reason}</code>
                            ))}
                          </div>
                        ) : null}
                        {record.artifacts.length > 0 ? (
                          <div className="artifactList">
                            {record.artifacts.map((artifact, index) => (
                              <code key={`${record.id}-${artifact.kind}-${index}`}>
                                {artifact.kind}
                                {artifact.path ? ` / ${artifact.path}` : ""}
                              </code>
                            ))}
                          </div>
                        ) : null}
                      </div>
                    ))
                  ) : (
                    <div className="emptyState compact">暂无关联证据记录</div>
                  )}
                </div>
              </div>
            ) : (
              <div className="emptyState">请选择一个操作</div>
            )}
          </div>
        </section>

        <section className="observabilityGrid" hidden={!viewVisible(activeView, ["审计", "执行适配器"])}>
          <div className="panel" hidden={activeView !== "审计"}>
            <PanelTitle
              icon={<ScrollText size={18} />}
              title="审计导出"
              meta={operationsAuditExport ? `${operationsAuditExport.timeline_item_count} 条时间线` : "未生成"}
            />
            {operationsAuditExport ? (
              <div className="signalList">
                <div className="signalItem">
                  <div className="signalHeader">
                    <strong>{compactID(operationsAuditExport.id)}</strong>
                    <StatusPill tone={operationsAuditExport.attention_item_count > 0 ? "warning" : "ok"} label={operationsAuditExport.format} />
                  </div>
                  <span>
                    证据 {operationsAuditExport.evidence_ref_count} / 验证 {operationsAuditExport.post_deployment_verification_count} / 引用{" "}
                    {operationsAuditExport.resource_deployment_ref_count}
                  </span>
                  <div className="signalMeta">
                    <code>{operationsAuditExport.redaction_applied ? "已脱敏" : "无脱敏项"}</code>
                    <code>{operationsAuditExport.risk_handoff_recommended_count} 个风险交接</code>
                    <code>{shortTimestamp(operationsAuditExport.generated_at)}</code>
                  </div>
                  <div className="routeCandidateGrid compact">
                    {Object.entries(operationsAuditExport.by_type)
                      .slice(0, 4)
                      .map(([type, count]) => (
                        <div className="routeCandidate" key={type}>
                          <strong>{type}</strong>
                          <span>{count} 条记录</span>
                        </div>
                      ))}
                  </div>
                </div>
              </div>
            ) : (
              <div className="emptyState">暂无审计导出</div>
            )}
          </div>

          <div className="panel" hidden={activeView !== "审计"}>
            <PanelTitle icon={<ShieldCheck size={18} />} title="决策账本" meta={decisionLedger ? `${decisionLedger.entry_count} 条记录` : "未生成"} />
            <div className="signalList">
              {decisionLedger ? (
                <>
                  <div className="signalItem">
                    <div className="signalHeader">
                      <strong>{compactID(decisionLedger.id)}</strong>
                      <StatusPill tone={decisionLedger.attention_count > 0 ? "warning" : "ok"} label={`${decisionLedger.attention_count} 个需关注`} />
                    </div>
                    <span>
                      证据 {decisionLedger.evidence_ref_count} / 脱敏 {decisionLedger.redaction_applied ? "已应用" : "无"}
                    </span>
                    <div className="signalMeta">
                      {Object.entries(decisionLedger.by_source_type)
                        .slice(0, 4)
                        .map(([type, count]) => (
                          <code key={type}>
                            {type}:{count}
                          </code>
                        ))}
                    </div>
                  </div>
                  {decisionEntries.slice(0, 3).map((entry) => (
                    <div className="signalItem" key={entry.id}>
                      <div className="signalHeader">
                        <strong>{entry.source_type}</strong>
                        <StatusPill tone={toneForStatus(entry.status)} label={entry.decision} />
                      </div>
                      <span>
                        {compactID(entry.source_id)} / {entry.environment || "all"}
                      </span>
                      <div className="signalMeta">
                        {entry.rule_refs[0] ? <code>{entry.rule_refs[0]}</code> : null}
                        {entry.evidence_refs.length ? <code>{entry.evidence_refs.length} 个证据</code> : null}
                        {entry.parent_ref ? <code>{compactID(entry.parent_ref)}</code> : null}
                      </div>
                      {entry.reasons[0] ? <small>{entry.reasons[0]}</small> : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">暂无决策账本</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "执行适配器"}>
            <PanelTitle icon={<Lock size={18} />} title="写入证明" meta={writeProofReport ? `${writeProofReport.proof_count} 条证明` : "未生成"} />
            <div className="signalList">
              {writeProofReport ? (
                <>
                  <div className="signalItem">
                    <div className="signalHeader">
                      <strong>{compactID(writeProofReport.id)}</strong>
                      <StatusPill tone={writeProofReport.blocked_count > 0 ? "blocked" : "ok"} label={`${writeProofReport.blocked_count} 阻断`} />
                    </div>
                    <span>
                      人工 {writeProofReport.manual_required_count} / 脱敏 {writeProofReport.redaction_applied ? "已应用" : "无"}
                    </span>
                    <div className="signalMeta">
                      {Object.entries(writeProofReport.by_operation_type)
                        .slice(0, 4)
                        .map(([type, count]) => (
                          <code key={type}>
                            {type}:{count}
                          </code>
                        ))}
                    </div>
                  </div>
                  {writeProofs.slice(0, 4).map((proof) => (
                    <div className="signalItem" key={proof.id}>
                      <div className="signalHeader">
                        <strong>{proof.operation_type}</strong>
                        <StatusPill tone={toneForStatus(proof.status)} label={proof.decision} />
                      </div>
                      <span>
                        {proof.provider || "provider"} / {proof.mode || "mode"} / {compactID(proof.operation_id)}
                      </span>
                      <div className="signalMeta">
                        <code>{proof.write_enabled ? "写入已启用" : "写入未启用"}</code>
                        <code>{proof.dry_run ? "dry-run" : "write path"}</code>
                        <code>{proof.approval_satisfied ? "审批满足" : proof.approval_required ? "需要审批" : "无需审批"}</code>
                        {proof.secret_ref_status ? <code>{proof.secret_ref_status}</code> : null}
                        {proof.provider_evidence_refs.length ? <code>{proof.provider_evidence_refs.length} 个证据</code> : null}
                      </div>
                      {proof.least_privilege ? <small>{proof.least_privilege}</small> : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">暂无写入证明</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "执行适配器"}>
            <PanelTitle
              icon={<ShieldCheck size={18} />}
              title="写入准入"
              meta={writeAdmissionReport ? `${writeAdmissionReport.entry_count} 条记录` : "未生成"}
            />
            <div className="signalList">
              {writeAdmissionReport ? (
                <>
                  <div className="signalItem">
                    <div className="signalHeader">
                      <strong>{writeAdmissionReport.policy_id}</strong>
                      <StatusPill tone={writeAdmissionReport.blocked_count > 0 ? "blocked" : "ok"} label={writeAdmissionReport.target} />
                    </div>
                    <span>
                      就绪 {writeAdmissionReport.ready_count} / 演练 {writeAdmissionReport.rehearsal_only_count} / 人工{" "}
                      {writeAdmissionReport.manual_required_count}
                    </span>
                    <div className="signalMeta">
                      <code>{writeAdmissionReport.redaction_applied ? "已脱敏" : "无脱敏项"}</code>
                      {Object.entries(writeAdmissionReport.by_status)
                        .slice(0, 3)
                        .map(([status, count]) => (
                          <code key={status}>
                            {status}:{count}
                          </code>
                        ))}
                    </div>
                  </div>
                  {writeAdmissions.slice(0, 4).map((admission) => (
                    <div className="signalItem" key={admission.id}>
                      <div className="signalHeader">
                        <strong>{admission.operation_type}</strong>
                        <StatusPill tone={toneForStatus(admission.status)} label={admission.decision} />
                      </div>
                      <span>
                        {admission.provider || "provider"} / {admission.mode || "mode"} / {compactID(admission.operation_id)}
                      </span>
                      <div className="signalMeta">
                        <code>{admission.write_enabled ? "写入已启用" : "写入未启用"}</code>
                        <code>{admission.rehearsal_allowed ? "允许演练" : "演练阻断"}</code>
                        {admission.provider_requirement_id ? <code>{compactID(admission.provider_requirement_id)}</code> : null}
                        {admission.rule_refs.length ? <code>{admission.rule_refs.length} 条规则</code> : null}
                        {admission.provider_evidence_refs.length ? <code>{admission.provider_evidence_refs.length} 个证据</code> : null}
                      </div>
                      {admission.reasons[0] ? <small>{admission.reasons[0]}</small> : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">暂无写入准入</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "执行适配器"}>
            <PanelTitle
              icon={<KeyRound size={18} />}
              title="Provider 证明包"
              meta={providerProofRequirementReport ? `${providerProofRequirementReport.requirement_count} 条要求` : "未生成"}
            />
            <div className="signalList">
              {providerProofRequirementReport ? (
                <>
                  <div className="signalItem">
                    <div className="signalHeader">
                      <strong>{providerProofRequirementReport.policy_id}</strong>
                      <StatusPill tone="ok" label={providerProofRequirementReport.policy_version || "active"} />
                    </div>
                    <div className="signalMeta">
                      {Object.entries(providerProofRequirementReport.by_provider)
                        .slice(0, 5)
                        .map(([provider, count]) => (
                          <code key={provider}>
                            {provider}:{count}
                          </code>
                        ))}
                    </div>
                  </div>
                  {providerProofRequirements.slice(0, 5).map((requirement) => (
                    <div className="signalItem" key={requirement.id}>
                      <div className="signalHeader">
                        <strong>{requirement.provider}</strong>
                        <StatusPill tone={toneForStatus(requirement.status)} label={requirement.operation_type} />
                      </div>
                      <span>{requirement.decision}</span>
                      <div className="signalMeta">
                        <code>{requirement.require_write_switch ? "需要写入开关" : "只读"}</code>
                        <code>{requirement.require_approval ? "需要审批" : "无需审批"}</code>
                        <code>{requirement.require_evidence ? "需要证据" : "无需证据"}</code>
                        {requirement.required_secret_ref_status ? <code>{requirement.required_secret_ref_status}</code> : null}
                        {requirement.least_privilege_scopes.length ? <code>{requirement.least_privilege_scopes.length} 个 scope</code> : null}
                      </div>
                      {requirement.replay_guard ? <small>{requirement.replay_guard}</small> : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">暂无 Provider proof 要求</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "执行适配器"}>
            <PanelTitle
              icon={<Server size={18} />}
              title="远程执行演练"
              meta={remoteRehearsalReport ? `${remoteRehearsalReport.rehearsal_count} 次演练` : "未生成"}
            />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={adapterActionState.status === "running"} onClick={() => setWritePipelineModal("remoteRehearsal")} type="button">
                <Play size={13} />
                <span>创建演练</span>
              </button>
              <ActionFeedback state={adapterActionState} />
            </div>
            <div className="signalList">
              {remoteRehearsalReport ? (
                <>
                  <div className="signalItem">
                    <div className="signalHeader">
                      <strong>{compactID(remoteRehearsalReport.id)}</strong>
                      <StatusPill tone={remoteRehearsalReport.blocked_count > 0 ? "blocked" : "ok"} label={`${remoteRehearsalReport.completed_count} 已完成`} />
                    </div>
                    <span>
                      阻断 {remoteRehearsalReport.blocked_count} / 人工 {remoteRehearsalReport.manual_count}
                    </span>
                  </div>
                  {remoteRehearsals.slice(0, 4).map((rehearsal) => (
                    <div className="signalItem" key={rehearsal.id}>
                      <div className="signalHeader">
                        <strong>{rehearsal.provider || "provider"}</strong>
                        <StatusPill tone={toneForStatus(rehearsal.status)} label={rehearsal.decision} />
                      </div>
                      <span>
                        {rehearsal.mode || "mode"} / {compactID(rehearsal.operation_id)}
                      </span>
                      <div className="signalMeta">
                        <code>{rehearsal.target_check_count} 个目标</code>
                        <code>{rehearsal.auth_ref_check_count} auth</code>
                        <code>{rehearsal.command_check_count} 条命令</code>
                        <code>{rehearsal.rollback_required ? "需要回滚" : "无需回滚"}</code>
                        {rehearsal.evidence_refs.length ? <code>{rehearsal.evidence_refs.length} 个证据</code> : null}
                      </div>
                      {rehearsal.reasons[0] ? <small>{rehearsal.reasons[0]}</small> : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">暂无远程执行演练</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "执行适配器"}>
            <PanelTitle
              icon={<ScrollText size={18} />}
              title="写入复核包"
              meta={writeReviewPacketReport ? `${writeReviewPacketReport.packet_count} 个 packet` : "未生成"}
            />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={adapterActionState.status === "running"} onClick={() => setWritePipelineModal("reviewPacket")} type="button">
                <ScrollText size={13} />
                <span>生成复核包</span>
              </button>
            </div>
            <div className="signalList">
              {writeReviewPacketReport ? (
                <>
                  <div className="signalItem">
                    <div className="signalHeader">
                      <strong>{compactID(writeReviewPacketReport.id)}</strong>
                      <StatusPill tone={writeReviewPacketReport.blocked_count > 0 ? "blocked" : writeReviewPacketReport.manual_required_count > 0 ? "warning" : "ok"} label={`${writeReviewPacketReport.ready_count} 就绪`} />
                    </div>
                    <span>
                      阻断 {writeReviewPacketReport.blocked_count} / 人工 {writeReviewPacketReport.manual_required_count}
                    </span>
                    <div className="signalMeta">
                      {Object.entries(writeReviewPacketReport.by_status)
                        .slice(0, 4)
                        .map(([status, count]) => (
                          <code key={status}>
                            {status}:{count}
                          </code>
                        ))}
                    </div>
                  </div>
                  {writeReviewPackets.slice(0, 4).map((packet) => (
                    <div className="signalItem" key={packet.id}>
                      <div className="signalHeader">
                        <strong>{packet.operation_type}</strong>
                        <StatusPill tone={toneForStatus(packet.status)} label={packet.decision} />
                      </div>
                      <span>
                        {packet.provider || "provider"} / {packet.environment || "env"} / {compactID(packet.operation_id)}
                      </span>
                      <div className="signalMeta">
                        {packet.remote_rehearsal_id ? <code>{compactID(packet.remote_rehearsal_id)}</code> : null}
                        {packet.provider_requirement_id ? <code>{compactID(packet.provider_requirement_id)}</code> : null}
                        {packet.queue_item_ids.length ? <code>{packet.queue_item_ids.length} 个队列项</code> : null}
                        {packet.evidence_refs.length ? <code>{packet.evidence_refs.length} 个证据</code> : null}
                        {packet.rule_refs.length ? <code>{packet.rule_refs.length} 条规则</code> : null}
                      </div>
                      {packet.reasons[0] ? <small>{packet.reasons[0]}</small> : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">暂无写入复核包</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "执行适配器"}>
            <PanelTitle
              icon={<Play size={18} />}
              title="写入执行计划"
              meta={writeExecutionPlanReport ? `${writeExecutionPlanReport.plan_count} 个计划` : "未生成"}
            />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={adapterActionState.status === "running"} onClick={() => setWritePipelineModal("executionPlan")} type="button">
                <Play size={13} />
                <span>生成计划</span>
              </button>
            </div>
            <div className="signalList">
              {writeExecutionPlanReport ? (
                <>
                  <div className="signalItem">
                    <div className="signalHeader">
                      <strong>{compactID(writeExecutionPlanReport.id)}</strong>
                      <StatusPill tone={writeExecutionPlanReport.external_write_count > 0 ? "blocked" : writeExecutionPlanReport.blocked_count > 0 ? "blocked" : "ok"} label={`${writeExecutionPlanReport.planned_count} 个预览`} />
                    </div>
                    <span>
                      就绪 {writeExecutionPlanReport.ready_count} / 人工 {writeExecutionPlanReport.manual_required_count} / 外部写入{" "}
                      {writeExecutionPlanReport.external_write_count}
                    </span>
                    <div className="signalMeta">
                      {Object.entries(writeExecutionPlanReport.by_mode)
                        .slice(0, 3)
                        .map(([mode, count]) => (
                          <code key={mode}>
                            {mode}:{count}
                          </code>
                        ))}
                    </div>
                  </div>
                  {writeExecutionPlans.slice(0, 4).map((plan) => (
                    <div className="signalItem" key={plan.id}>
                      <div className="signalHeader">
                        <strong>{plan.mode}</strong>
                        <StatusPill tone={toneForStatus(plan.status)} label={plan.decision} />
                      </div>
                      <span>
                        {plan.provider || "provider"} / {plan.environment || "env"} / {compactID(plan.operation_id || plan.review_packet_id || plan.id)}
                      </span>
                      <div className="signalMeta">
                        {plan.review_packet_id ? <code>{compactID(plan.review_packet_id)}</code> : null}
                        {plan.approval_id ? <code>{compactID(plan.approval_id)}</code> : null}
                        <code>{plan.apply_allowed ? "允许 apply" : "apply 锁定"}</code>
                        <code>{plan.external_write_performed ? "已外部写入" : "无外部写入"}</code>
                        {plan.evidence_refs.length ? <code>{plan.evidence_refs.length} 个证据</code> : null}
                      </div>
                      {plan.reasons[0] ? <small>{decisionLabel(plan.reasons[0])}</small> : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">暂无写入执行计划</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "执行适配器"}>
            <PanelTitle
              icon={<Wrench size={18} />}
              title="写入 Adapter 执行"
              meta={writeAdapterExecutionReport ? `${writeAdapterExecutionReport.execution_count} 条执行` : "未生成"}
            />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={adapterActionState.status === "running"} onClick={() => setWritePipelineModal("adapterExecution")} type="button">
                <Wrench size={13} />
                <span>创建执行</span>
              </button>
            </div>
            <div className="signalList">
              {writeAdapterExecutionReport ? (
                <>
                  <div className="signalItem">
                    <div className="signalHeader">
                      <strong>{compactID(writeAdapterExecutionReport.id)}</strong>
                      <StatusPill tone={writeAdapterExecutionReport.external_write_count > 0 ? "blocked" : writeAdapterExecutionReport.blocked_count > 0 ? "blocked" : "ok"} label={`${writeAdapterExecutionReport.completed_count} 已完成`} />
                    </div>
                    <span>
                      人工 {writeAdapterExecutionReport.manual_required_count} / 尝试 {writeAdapterExecutionReport.external_attempt_count} / 写入{" "}
                      {writeAdapterExecutionReport.external_write_count}
                    </span>
                    <span>
                      sandbox {writeAdapterExecutionReport.sandbox_result_count} / 回滚 {writeAdapterExecutionReport.rollback_bound_count}
                    </span>
                    <div className="signalMeta">
                      {Object.entries(writeAdapterExecutionReport.by_adapter)
                        .slice(0, 3)
                        .map(([adapter, count]) => (
                          <code key={adapter}>
                            {adapter}:{count}
                          </code>
                        ))}
                    </div>
                  </div>
                  {writeAdapterExecutions.slice(0, 4).map((execution) => (
                    <div className="signalItem" key={execution.id}>
                      <div className="signalHeader">
                        <strong>{execution.adapter_id || "adapter"}</strong>
                        <StatusPill tone={toneForStatus(execution.status)} label={execution.decision} />
                      </div>
                      <span>
                        {execution.mode} / {execution.provider || "provider"} / {compactID(execution.operation_id || execution.execution_plan_id || execution.id)}
                      </span>
                      <div className="signalMeta">
                        {execution.execution_plan_id ? <code>{compactID(execution.execution_plan_id)}</code> : null}
                        <code>{execution.guard_results.length} 个 guard</code>
                        {execution.sandbox_results.length ? <code>{execution.sandbox_results.length} sandbox</code> : null}
                        {execution.rollback_binding?.decision ? <code>{execution.rollback_binding.decision}</code> : null}
                        <code>{execution.external_write_attempted ? "已尝试" : "未尝试"}</code>
                        <code>{execution.external_write_performed ? "已外部写入" : "无外部写入"}</code>
                        {execution.evidence_refs.length ? <code>{execution.evidence_refs.length} 个证据</code> : null}
                      </div>
                      {execution.sandbox_results[0] ? (
                        <small>
                          {execution.sandbox_results[0].decision} / {execution.sandbox_results[0].no_remote_write ? "无远程写入" : "远程写入受控"}
                        </small>
                      ) : execution.guard_results[0] ? (
                        <small>{execution.guard_results[0].decision}</small>
                      ) : execution.reasons[0] ? (
                        <small>{execution.reasons[0]}</small>
                      ) : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">暂无写入 Adapter 执行</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "执行适配器"}>
            <PanelTitle
              icon={<Wrench size={18} />}
              title="写入 Adapter 恢复"
              meta={writeAdapterRecoveryReport ? `${writeAdapterRecoveryReport.recovery_count} 条恢复记录` : "未生成"}
            />
            <div className="signalList">
              {writeAdapterRecoveryReport ? (
                <>
                  <div className="signalItem">
                    <div className="signalHeader">
                      <strong>{compactID(writeAdapterRecoveryReport.id)}</strong>
                      <StatusPill tone={writeAdapterRecoveryReport.open_count > 0 ? "warning" : "ok"} label={`${writeAdapterRecoveryReport.open_count} 待处理`} />
                    </div>
                    <span>
                      修复 {writeAdapterRecoveryReport.repair_count} / 重试 {writeAdapterRecoveryReport.retry_count} / 交接{" "}
                      {writeAdapterRecoveryReport.handoff_count}
                    </span>
                    <div className="signalMeta">
                      {Object.entries(writeAdapterRecoveryReport.by_failure)
                        .slice(0, 3)
                        .map(([failure, count]) => (
                          <code key={failure}>
                            {failure}:{count}
                          </code>
                        ))}
                    </div>
                  </div>
                  {writeAdapterRecoveries.slice(0, 4).map((recovery) => (
                    <div className="signalItem" key={recovery.id}>
                      <div className="signalHeader">
                        <strong>{recovery.failure_class}</strong>
                        <StatusPill tone={recovery.handoff_required ? "warning" : recovery.repair_allowed ? "blocked" : "ok"} label={recovery.decision} />
                      </div>
                      <span>
                        {recovery.adapter_id || "adapter"} / {recovery.recovery_action} / {compactID(recovery.execution_id)}
                      </span>
                      <div className="signalMeta">
                        <code>{recovery.repair_allowed ? "允许修复" : "无需修复"}</code>
                        <code>{recovery.retry_allowed ? "允许重试" : "无需重试"}</code>
                        <code>{recovery.handoff_required ? "需要交接" : "无需交接"}</code>
                        {recovery.evidence_refs.length ? <code>{recovery.evidence_refs.length} 个证据</code> : null}
                      </div>
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">暂无写入 Adapter 恢复记录</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "执行适配器"}>
            <PanelTitle icon={<RefreshCw size={18} />} title="控制循环" meta={latestControlLoopRun ? compactID(latestControlLoopRun.id) : "暂无运行"} />
            <div className="signalList">
              {latestControlLoopRun ? (
                <div className="signalItem">
                  <div className="signalHeader">
                    <strong>{latestControlLoopRun.trigger}</strong>
                    <StatusPill tone={toneForStatus(latestControlLoopRun.status)} label={latestControlLoopRun.decision} />
                  </div>
                  <span>
                    {latestControlLoopRun.steps.length} 个步骤 / {shortTimestamp(latestControlLoopRun.finished_at || latestControlLoopRun.started_at)}
                  </span>
                  <div className="routeCandidateGrid compact">
                    {latestControlLoopRun.steps.slice(0, 4).map((step) => (
                      <div className="routeCandidate" key={step.id}>
                        <strong>{step.type}</strong>
                        <span>{step.summary || step.decision}</span>
                        <div className="signalMeta">
                          <code>{step.status}</code>
                          {step.evidence_count ? <code>{step.evidence_count} 个证据</code> : null}
                        </div>
                      </div>
                    ))}
                  </div>
                  {latestControlLoopRun.reasons[0] ? <small>{latestControlLoopRun.reasons[0]}</small> : null}
                </div>
              ) : (
                <div className="emptyState">暂无控制循环运行</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "执行适配器"}>
            <PanelTitle icon={<CircleDotDashed size={18} />} title="控制队列" meta={`${controlLoopQueue.length} 个条目`} />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={adapterActionState.status === "running"} onClick={() => setControlQueueModalOpen(true)} type="button">
                <CircleDotDashed size={13} />
                <span>入队</span>
              </button>
              <button className="inlineActionButton" disabled={adapterActionState.status === "running" || controlLoopQueue.length === 0} onClick={() => setControlQueueRunModalOpen(true)} type="button">
                <RefreshCw size={13} />
                <span>消费队列</span>
              </button>
              <ActionFeedback state={adapterActionState} />
            </div>
            <div className="signalList">
              {controlLoopQueue.length > 0 ? (
                controlLoopQueue.slice(0, 5).map((item) => (
                  <div className="signalItem" key={item.id}>
                    <div className="signalHeader">
                      <strong>{compactID(item.id)}</strong>
                      <StatusPill tone={toneForStatus(item.status)} label={item.decision} />
                    </div>
                    <span>
                      {item.steps.length} 个步骤 / {item.environment || "全部"} / 尝试 {item.attempt_count}
                    </span>
                    <div className="signalMeta">
                      <code>{item.maintenance_window || "窗口开启"}</code>
                      {item.review_packet_id ? <code>{compactID(item.review_packet_id)}</code> : null}
                      {item.admission_id ? <code>{compactID(item.admission_id)}</code> : null}
                      {item.remote_rehearsal_id ? <code>{compactID(item.remote_rehearsal_id)}</code> : null}
                      {item.adapter_recovery_id ? <code>{compactID(item.adapter_recovery_id)}</code> : null}
                      {item.due_at ? <code>{shortTimestamp(item.due_at)}</code> : null}
                      {item.run_id ? <code>{compactID(item.run_id)}</code> : null}
                    </div>
                    {item.reasons[0] ? <small>{item.reasons[0]}</small> : null}
                  </div>
                ))
              ) : (
                <div className="emptyState">暂无控制队列条目</div>
              )}
            </div>
          </div>
        </section>

        <section className="mainGrid" hidden={activeView !== "Issue Graph"}>
          <div className="panel graphPanel">
            <PanelTitle icon={<Network size={18} />} title="Issue Graph" meta={`待执行 ${issueGraphLayout.nodes.length} 节点 / ${issueGraphLayout.edges.length} 依赖`} />
            <div className="graphCanvas">
              {snapshot.issues.length === 0 ? (
                <div className="emptyState graphEmptyState">当前没有待执行的 Issue Graph；已完成需求可在需求记录中查看。</div>
              ) : (
                <div className="issueDagViewport">
                  <div className="issueDagSurface" style={{ height: issueGraphLayout.height, width: issueGraphLayout.width }}>
                    <svg className="issueDagEdges" height={issueGraphLayout.height} viewBox={`0 0 ${issueGraphLayout.width} ${issueGraphLayout.height}`} width={issueGraphLayout.width}>
                      <defs>
                        <marker id="issueDagArrow" markerHeight="8" markerWidth="8" orient="auto" refX="7" refY="4">
                          <path d="M0,0 L8,4 L0,8 Z" />
                        </marker>
                      </defs>
                      {issueGraphLayout.columns.map((column) => (
                        <text className="issueDagColumnLabel" key={column.level} x={column.x} y="24">
                          {column.label}
                        </text>
                      ))}
                      {issueGraphLayout.edges.map((edge) => (
                        <path
                          className={`issueDagEdge ${graphEdgeClass(edge, graphRelations, selectedIssueID)}`}
                          d={edge.path}
                          key={`${edge.from}-${edge.to}`}
                          markerEnd="url(#issueDagArrow)"
                        />
                      ))}
                    </svg>
                    {issueGraphLayout.nodes.map((node) => {
                      const issue = node.issue;
                      const selected = selectedIssue?.id === issue.id;
                      const related = graphRelations.upstream.has(issue.id) || graphRelations.downstream.has(issue.id);
                      const dimmed = Boolean(selectedIssueID) && !selected && !related;
                      return (
                        <button
                          aria-label={`查看 Issue ${issue.title}`}
                          className={`issueDagNode ${statusClass(issue.status)} ${selected ? "selected" : ""} ${graphRelations.upstream.has(issue.id) ? "upstream" : ""} ${
                            graphRelations.downstream.has(issue.id) ? "downstream" : ""
                          } ${dimmed ? "dimmed" : ""}`}
                          key={issue.id}
                          onClick={() => setSelectedIssueID(issue.id)}
                          style={{ height: node.height, left: node.x, top: node.y, width: node.width }}
                          type="button"
                        >
                          <span>{issue.title}</span>
                          <small>{issue.role}</small>
                          <div className="issueDagNodeMeta">
                            <StatusDot tone={toneForStatus(issue.status)} />
                            <code>{compactID(issue.id)}</code>
                            {issue.commit_after ? <code>{compactCommit(issue.commit_after)}</code> : null}
                          </div>
                        </button>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          </div>

          <aside className="panel inspector">
            <PanelTitle icon={<CircleDotDashed size={18} />} title="检查器" meta={selectedIssue?.id ? compactID(selectedIssue.id) : "issue"} />
            {selectedIssue ? (
              <div className="inspectorBody">
                <h2>{selectedIssue.title}</h2>
                <StatusPill tone={toneForStatus(selectedIssue.status)} label={statusLabel(selectedIssue.status)} />
                <div className="rowActions wide">
                  <button
                    className="inlineActionButton"
                    disabled={issueActionState[`${selectedIssue.id}:merge-decision`]?.status === "running"}
                    onClick={() => void createIssueMergeDecision(selectedIssue.id)}
                    type="button"
                  >
                    <ShieldCheck size={13} />
                    <span>合并决策</span>
                  </button>
                  <button
                    className="inlineActionButton"
                    disabled={issueActionState[`${selectedIssue.id}:git-provider-plan`]?.status === "running"}
                    onClick={() => void createIssueGitProviderPlan(selectedIssue.id)}
                    type="button"
                  >
                    <GitBranch size={13} />
                    <span>PR/MR 计划</span>
                  </button>
                </div>
                <ActionFeedback state={issueActionState[`${selectedIssue.id}:merge-decision`]} />
                <ActionFeedback state={issueActionState[`${selectedIssue.id}:git-provider-plan`]} />
                <dl>
                  <div>
                    <dt>运行</dt>
                    <dd>{selectedIssue.run_id ?? "未开始"}</dd>
                  </div>
                  <div>
                    <dt>Subagent</dt>
                    <dd>{selectedIssue.subagent_id ?? "未分配"}</dd>
                  </div>
                  <div>
                    <dt>角色</dt>
                    <dd>{selectedIssue.role}</dd>
                  </div>
                  <div>
                    <dt>Runtime</dt>
                    <dd>{selectedIssue.runtime ?? "待定"}</dd>
                  </div>
                  <div>
                    <dt>Runtime 状态</dt>
                    <dd>{selectedIssue.runtime_status ?? "待定"}</dd>
                  </div>
                  <div>
                    <dt>Provider</dt>
                    <dd>{selectedIssue.provider ?? "路由待定"}</dd>
                  </div>
                  <div>
                    <dt>质量</dt>
                    <dd>{selectedIssue.quality ?? "未开始"}</dd>
                  </div>
                  <div>
                    <dt>复核</dt>
                    <dd>{selectedIssue.review_status ?? "未复核"}</dd>
                  </div>
                  <div>
                    <dt>质量报告</dt>
                    <dd>{selectedIssue.quality_report_id ?? "无"}</dd>
                  </div>
                  <div>
                    <dt>Commit</dt>
                    <dd>{selectedIssue.commit_after ? compactCommit(selectedIssue.commit_after) : "未由 Moyuan 产生"}</dd>
                  </div>
                  <div>
                    <dt>变更文件</dt>
                    <dd>{selectedIssue.changed_files?.length ?? 0}</dd>
                  </div>
                </dl>
                {selectedIssue.commit_before || selectedIssue.diff_summary_path ? (
                  <div className="commitStrip">
                    <GitBranch size={16} />
                    <div>
                      <strong>{selectedIssue.commit_changed ? "受控 commit 已记录" : "受控运行已记录"}</strong>
                      <span>
                        {selectedIssue.commit_before ? compactCommit(selectedIssue.commit_before) : "无 before"} →{" "}
                        {selectedIssue.commit_after ? compactCommit(selectedIssue.commit_after) : "无 after"}
                      </span>
                      {selectedIssue.diff_summary_path ? <code>{shortPath(selectedIssue.diff_summary_path)}</code> : null}
                    </div>
                  </div>
                ) : null}
                {selectedIssue.quality_decision ? (
                  <div className="decisionStrip">
                    <ShieldCheck size={16} />
                    <div>
                      <strong>{selectedIssue.quality_decision}</strong>
                      <span>{selectedIssue.quality_reasons?.[0] ?? "质量解释可用"}</span>
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
                    <span>输出契约</span>
                    <div className="chipList">
                      {selectedIssue.output_contract.map((item) => (
                        <code key={item}>{item}</code>
                      ))}
                    </div>
                  </div>
                ) : null}
                {selectedDependencyIDs.length > 0 ? (
                  <div className="dependencyList">
                    <div className="dependencyListTitle">上游依赖</div>
                    {selectedDependencyIDs.map((dependency) => {
                      const dependencyIssue = issueByID.get(dependency);
                      return (
                        <span key={dependency}>
                          <ChevronRight size={13} />
                          <strong>{dependencyIssue?.title ?? compactID(dependency)}</strong>
                          <code>{compactID(dependency)}</code>
                        </span>
                      );
                    })}
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
            ) : (
              <div className="emptyState inspectorEmptyState">请选择一个 Issue 查看执行、质量和依赖信息。</div>
            )}
          </aside>
        </section>

        <section className="lowerGrid singlePanelGrid" hidden={!viewVisible(activeView, ["运行", "质量", "测试验证", "Memory"])}>
          <div className="panel" hidden={activeView !== "运行"}>
            <PanelTitle icon={<Activity size={18} />} title="运行时间线" meta={`${snapshot.runs.length} 次运行`} />
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

          <div className="panel" hidden={!viewVisible(activeView, ["质量", "测试验证"])}>
            <PanelTitle
              icon={activeView === "测试验证" ? <ShieldCheck size={18} /> : <CheckCircle2 size={18} />}
              title={activeView === "测试验证" ? "测试与质量信号" : "质量门禁"}
              meta={activeView === "测试验证" ? "test / lint / smoke / monitor" : "diff 优先"}
            />
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
            {activeView === "质量" ? (
              <div className="signalList qualityReportList">
                {snapshot.quality_reports.length > 0 ? (
                  snapshot.quality_reports.slice(0, 5).map((report) => (
                    <div className="signalItem" key={report.id}>
                      <div className="signalHeader">
                        <strong>{report.task_id || report.id}</strong>
                        <StatusPill tone={toneForStatus(report.status)} label={report.review_status || report.status} />
                      </div>
                      <span>
                        {report.check_count} 个检查 / {report.findings_count} 条发现 / {report.changed_files.length} 个文件
                      </span>
                      <div className="signalActions">
                        <button className="inlineActionButton" onClick={() => setQualityDetailReportID(report.id)} type="button">
                          <ScrollText size={13} />
                          <span>详情</span>
                        </button>
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="emptyState compact">暂无质量报告详情</div>
                )}
              </div>
            ) : null}
          </div>

          <div className="panel" hidden={activeView !== "Memory"}>
            <PanelTitle icon={<MemoryStick size={18} />} title="Memory" meta="支持 compact" />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={memoryActionState.status === "running"} onClick={() => setMemorySearchModalOpen(true)} type="button">
                <Search size={13} />
                <span>搜索 Memory</span>
              </button>
              <ActionFeedback state={memoryActionState} />
            </div>
            <div className="memoryList">
              {(memorySearchResults.length > 0 ? memorySearchResults : snapshot.memory).map((record) => (
                <div className="memoryItem" key={record.id}>
                  <span>{record.kind}</span>
                  <strong>{record.summary}</strong>
                  <meter value={record.score} min="0" max="1" />
                  {isMemoryRecord(record) && record.tags.length ? <small>{record.tags.slice(0, 4).join(", ")}</small> : null}
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="lowerGrid" hidden={activeView !== "批量执行"}>
          <div className="panel">
            <PanelTitle icon={<Layers3 size={18} />} title="批量计划" meta={`${snapshot.batch_plans.length} 个计划`} />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={batchActionState.plan?.status === "running"} onClick={() => setBatchPlanModalOpen(true)} type="button">
                <Layers3 size={13} />
                <span>{batchActionState.plan?.status === "running" ? "创建中" : "创建计划"}</span>
              </button>
              <ActionFeedback state={batchActionState.plan} />
            </div>
            <div className="signalList">
              {snapshot.batch_plans.length > 0 ? (
                snapshot.batch_plans.map((plan) => {
                  const dryRunState = batchActionState[plan.id];
                  const mergeState = batchActionState[`${plan.id}:merge`];
                  return (
                    <div className="signalItem" key={plan.id}>
                      <div className="signalHeader">
                        <strong title={plan.epic_id || plan.id}>{batchPlanTitle(plan, requirementByEpicID)}</strong>
                        <StatusPill tone={toneForStatus(plan.status)} label={decisionLabel(plan.decision || plan.status)} />
                      </div>
                      <small>{batchPlanSubtitle(plan, requirementByEpicID)}</small>
                      <span>
                        派发 {plan.dispatch_count} / 等待 {plan.waiting_count} / 阻断 {plan.blocked_count}
                      </span>
                      <div className="signalMeta">
                        <code>{modeLabel(plan.mode)}</code>
                        <code>{plan.max_parallel} 并行</code>
                        <code>{plan.runtime_slots} 个槽位</code>
                        {plan.write_scope_conflict_count ? <code>{plan.write_scope_conflict_count} 个冲突</code> : null}
                      </div>
                      {plan.reasons[0] ? <small>{decisionLabel(plan.reasons[0])}</small> : null}
                      <div className="signalActions">
                        <button className="inlineActionButton" disabled={dryRunState?.status === "running"} onClick={() => void runBatchDryRun(plan.id)} type="button">
                          <Play size={13} />
                          <span>{dryRunState?.status === "running" ? "运行中" : "试运行"}</span>
                        </button>
                        <button className="inlineActionButton" disabled={mergeState?.status === "running"} onClick={() => void buildMergeQueue(plan.id)} type="button">
                          <ShieldCheck size={13} />
                          <span>{mergeState?.status === "running" ? "构建中" : "合并队列"}</span>
                        </button>
                      </div>
                      <ActionFeedback state={dryRunState} />
                      <ActionFeedback state={mergeState} />
                    </div>
                  );
                })
              ) : (
                <div className="emptyState">暂无批量计划</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<TerminalSquare size={18} />} title="批量运行" meta={`${snapshot.batch_runs.length} 次运行`} />
            <div className="signalList">
              {snapshot.batch_runs.length > 0 ? (
                snapshot.batch_runs.map((run) => (
                  <div className="signalItem" key={run.id}>
                    <div className="signalHeader">
                      <strong>{compactID(run.id)}</strong>
                      <StatusPill tone={toneForStatus(run.status)} label={decisionLabel(run.decision || run.status)} />
                    </div>
                    <span>
                      {modeLabel(run.mode)} / {run.item_count} 项 / 并行 {run.parallelism || 1} / 接受 {run.accepted_count}
                    </span>
                    <div className="signalMeta">
                      <code>{compactID(run.batch_id)}</code>
                      <code>返工 {run.needs_rework_count}</code>
                      <code>阻断 {run.blocked_count}</code>
                      {run.requested_by ? <code>{run.requested_by}</code> : null}
                    </div>
                    <div className="routeCandidateGrid compact">
                      {run.items.slice(0, 3).map((item) => (
                        <div className="routeCandidate" key={`${run.id}-${item.issue_id}`}>
                          <strong>{compactID(item.issue_id)}</strong>
                          <span>{decisionLabel(item.decision || item.status)}</span>
                          <div className="signalMeta">
                            {item.worker_slot ? <code>槽位 {item.worker_slot}</code> : null}
                            {item.runtime_id ? <code>{item.runtime_id}</code> : null}
                            {item.worktree_id ? <code>{compactID(item.worktree_id)}</code> : null}
                            {item.quality_report_id ? <code>{compactID(item.quality_report_id)}</code> : null}
                            {item.canceled_reason ? <code>{decisionLabel(item.canceled_reason)}</code> : null}
                          </div>
                        </div>
                      ))}
                    </div>
                    {run.reasons[0] ? <small>{decisionLabel(run.reasons[0])}</small> : null}
                  </div>
                ))
              ) : (
                <div className="emptyState">暂无批量运行</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<GitBranch size={18} />} title="Worktree 与合并" meta={`${snapshot.worktrees.length} 个 worktree / ${snapshot.merge_queues.length} 个队列`} />
            <div className="signalList">
              {snapshot.merge_queues.length > 0 ? (
                snapshot.merge_queues.map((queue) => {
                  const previewState = batchActionState[`${queue.id}:integration-preview`];
                  return (
                    <div className="signalItem" key={queue.id}>
                      <div className="signalHeader">
                        <strong>{compactID(queue.id)}</strong>
                        <StatusPill tone={toneForStatus(queue.status)} label={queue.decision} />
                      </div>
                      <span>
                        就绪 {queue.ready_count} / 返工 {queue.needs_rework_count} / 阻断 {queue.blocked_count}
                      </span>
                      <div className="signalMeta">
                        <code>{compactID(queue.batch_id)}</code>
                        {queue.batch_run_id ? <code>{compactID(queue.batch_run_id)}</code> : null}
                        {queue.reasons[0] ? <code>{queue.reasons[0]}</code> : null}
                      </div>
                      <div className="routeCandidateGrid compact">
                        {queue.items.slice(0, 3).map((item) => (
                          <div className="routeCandidate" key={`${queue.id}-${item.issue_id}`}>
                            <strong>{compactID(item.issue_id)}</strong>
                            <span>{item.reason || item.decision}</span>
                            <div className="signalMeta">
                              <code>{item.status}</code>
                              {item.worktree_id ? <code>{compactID(item.worktree_id)}</code> : null}
                            </div>
                          </div>
                        ))}
                      </div>
                      <div className="signalActions">
                        <button
                          className="inlineActionButton"
                          disabled={previewState?.status === "running"}
                          onClick={() => void buildIntegrationPreview(queue.id)}
                          type="button"
                        >
                          <GitBranch size={13} />
                          <span>{previewState?.status === "running" ? "预览中" : "Integration Preview"}</span>
                        </button>
                      </div>
                      <ActionFeedback state={previewState} />
                    </div>
                  );
                })
              ) : (
                <div className="emptyState compact">暂无合并队列</div>
              )}
              {snapshot.worktrees.slice(0, 3).map((record) => (
                <div className="signalItem" key={record.id}>
                  <div className="signalHeader">
                    <strong>{compactID(record.issue_id || record.id)}</strong>
                    <StatusPill tone={toneForStatus(record.status)} label={record.decision} />
                  </div>
                  <span>{record.branch || record.base_ref || "分支待定"}</span>
                  <div className="signalMeta">
                    {record.batch_id ? <code>{compactID(record.batch_id)}</code> : null}
                    {record.worktree_path ? <code>{shortPath(record.worktree_path)}</code> : null}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="panel">
            <PanelTitle
              icon={<Rocket size={18} />}
              title="集成与 Release"
              meta={`${snapshot.integration_previews.length} 个预览 / ${snapshot.release_batches.length} 个批次`}
            />
            <div className="signalList">
              {snapshot.integration_previews.length > 0 ? (
                snapshot.integration_previews.slice(0, 3).map((preview) => {
                  const applyState = batchActionState[`${preview.id}:apply-dry-run`];
                  return (
                    <div className="signalItem" key={preview.id}>
                      <div className="signalHeader">
                        <strong>{compactID(preview.id)}</strong>
                        <StatusPill tone={toneForStatus(preview.status)} label={preview.decision} />
                      </div>
                      <span>
                        就绪 {preview.ready_count} / 冲突 {preview.conflict_count} / 阻断 {preview.blocked_count}
                      </span>
                      <div className="signalMeta">
                        {preview.merge_queue_id ? <code>{compactID(preview.merge_queue_id)}</code> : null}
                        {preview.integration_branch ? <code>{compactID(preview.integration_branch)}</code> : null}
                        {preview.base_ref ? <code>基线 {preview.base_ref}</code> : null}
                      </div>
                      <div className="routeCandidateGrid compact">
                        {preview.items.slice(0, 2).map((item) => (
                          <div className="routeCandidate" key={`${preview.id}-${item.issue_id}`}>
                            <strong>{compactID(item.issue_id)}</strong>
                            <span>{item.reason || item.decision}</span>
                            <div className="signalMeta">
                              <code>{item.status}</code>
                              {item.changed_files.length > 0 ? <code>{item.changed_files.length} 个文件</code> : null}
                              {item.conflicted_files.length > 0 ? <code>{item.conflicted_files.length} 个冲突</code> : null}
                            </div>
                          </div>
                        ))}
                      </div>
                      <div className="signalActions">
                        <button
                          className="inlineActionButton"
                          disabled={applyState?.status === "running"}
                          onClick={() => void dryRunIntegrationApply(preview.id)}
                          type="button"
                        >
                          <ShieldCheck size={13} />
                          <span>{applyState?.status === "running" ? "规划中" : "Apply Dry Run"}</span>
                        </button>
                      </div>
                      <ActionFeedback state={applyState} />
                    </div>
                  );
                })
              ) : (
                <div className="emptyState compact">暂无集成预览</div>
              )}
              {snapshot.integration_applies.slice(0, 2).map((apply) => {
                const releaseState = batchActionState[`${apply.id}:release-batch`];
                return (
                  <div className="signalItem" key={apply.id}>
                    <div className="signalHeader">
                      <strong>{compactID(apply.id)}</strong>
                      <StatusPill tone={toneForStatus(apply.status)} label={apply.decision} />
                    </div>
                    <span>
                      {apply.mode} / {apply.write_enabled ? "写入已启用" : "受保护"} / {apply.action_count} 个动作
                    </span>
                    <div className="signalMeta">
                      {apply.preview_id ? <code>{compactID(apply.preview_id)}</code> : null}
                      {apply.target_branch ? <code>{compactID(apply.target_branch)}</code> : null}
                      {apply.requested_by ? <code>{apply.requested_by}</code> : null}
                    </div>
                    <div className="signalActions">
                      <button
                        className="inlineActionButton"
                        disabled={releaseState?.status === "running"}
                        onClick={() => void planReleaseBatch(apply.id)}
                        type="button"
                      >
                        <Rocket size={13} />
                        <span>{releaseState?.status === "running" ? "检查中" : "Release Batch"}</span>
                      </button>
                    </div>
                    <ActionFeedback state={releaseState} />
                  </div>
                );
              })}
              {snapshot.release_batches.slice(0, 2).map((releaseBatch) => {
                const candidateState = batchActionState[`${releaseBatch.id}:candidate`];
                return (
                  <div className="signalItem" key={releaseBatch.id}>
                    <div className="signalHeader">
                      <strong>{compactID(releaseBatch.version || releaseBatch.id)}</strong>
                      <StatusPill tone={toneForStatus(releaseBatch.status)} label={releaseBatch.decision} />
                    </div>
                    <span>
                      就绪 {releaseBatch.ready_item_count}/{releaseBatch.min_items} / {releaseBatch.release_branch || "Release 分支待定"}
                    </span>
                    <div className="signalMeta">
                      {releaseBatch.integration_apply_id ? <code>{compactID(releaseBatch.integration_apply_id)}</code> : null}
                      {releaseBatch.source_branch ? <code>{compactID(releaseBatch.source_branch)}</code> : null}
                      {releaseBatch.reasons[0] ? <code>{releaseBatch.reasons[0]}</code> : null}
                    </div>
                    <div className="signalActions">
                      <button
                        className="inlineActionButton"
                        disabled={candidateState?.status === "running"}
                        onClick={() => void planReleaseCandidate(releaseBatch.id)}
                        type="button"
                      >
                        <Rocket size={13} />
                        <span>{candidateState?.status === "running" ? "规划中" : "Candidate"}</span>
                      </button>
                    </div>
                    <ActionFeedback state={candidateState} />
                    {releaseBatch.commands[0] ? <small>{releaseBatch.commands[0]}</small> : null}
                  </div>
                );
              })}
              {snapshot.release_candidates.slice(0, 2).map((candidate) => {
                const applyState = batchActionState[`${candidate.id}:release-branch-apply`];
                const providerState = batchActionState[`${candidate.id}:provider-preview`];
                const deployState = batchActionState[`${candidate.id}:deployment-plan`];
                const publishState = batchActionState[`${candidate.id}:provider-publish`];
                const prmrState = batchActionState[`${candidate.id}:pr-mr-plan`];
                const executionState = batchActionState[`${candidate.id}:deployment-execution`];
                const feedback = snapshot.deployment_feedback.find((item) => item.candidate_id === candidate.id);
                return (
                  <div className="signalItem" key={candidate.id}>
                    <div className="signalHeader">
                      <strong>{compactID(candidate.version || candidate.id)}</strong>
                      <StatusPill tone={toneForStatus(candidate.status)} label={candidate.decision} />
                    </div>
                    <span>
                      {candidate.provider || "Provider 待定"} / {candidate.release_branch || "Release 分支待定"}
                    </span>
                    <div className="signalMeta">
                      {candidate.release_batch_id ? <code>{compactID(candidate.release_batch_id)}</code> : null}
                      {candidate.source_branch ? <code>{compactID(candidate.source_branch)}</code> : null}
                      {candidate.deployment_targets.length > 0 ? <code>{candidate.deployment_targets.join(",")}</code> : null}
                    </div>
                    <div className="signalActions">
                      <button
                        className="inlineActionButton"
                        disabled={applyState?.status === "running"}
                        onClick={() => void dryRunReleaseCandidateApply(candidate.id)}
                        type="button"
                      >
                        <GitBranch size={13} />
                        <span>{applyState?.status === "running" ? "规划中" : "分支 Dry Run"}</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={providerState?.status === "running"}
                        onClick={() => void previewReleaseCandidateProvider(candidate.id)}
                        type="button"
                      >
                        <ShieldCheck size={13} />
                        <span>{providerState?.status === "running" ? "预览中" : "Provider 预览"}</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={deployState?.status === "running"}
                        onClick={() => void createCandidateDeploymentPlan(candidate.id)}
                        type="button"
                      >
                        <Server size={13} />
                        <span>{deployState?.status === "running" ? "规划中" : "部署计划"}</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={publishState?.status === "running"}
                        onClick={() => void publishReleaseCandidateProvider(candidate.id)}
                        type="button"
                      >
                        <Rocket size={13} />
                        <span>{publishState?.status === "running" ? "检查中" : "发布门禁"}</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={prmrState?.status === "running"}
                        onClick={() => void planCandidatePRMR(candidate.id)}
                        type="button"
                      >
                        <GitBranch size={13} />
                        <span>{prmrState?.status === "running" ? "规划中" : "PR/MR 计划"}</span>
                      </button>
                      <button
                        className="inlineActionButton"
                        disabled={executionState?.status === "running"}
                        onClick={() => void runCandidateDeploymentDryRun(candidate.id)}
                        type="button"
                      >
                        <Play size={13} />
                        <span>{executionState?.status === "running" ? "运行中" : "部署 Dry Run"}</span>
                      </button>
                    </div>
                    <ActionFeedback state={applyState} />
                    <ActionFeedback state={providerState} />
                    <ActionFeedback state={deployState} />
                    <ActionFeedback state={publishState} />
                    <ActionFeedback state={prmrState} />
                    <ActionFeedback state={executionState} />
                    {feedback ? (
                      <div className="signalMeta">
                        <code>{feedback.decision}</code>
                        {feedback.environment ? <code>{feedback.environment}</code> : null}
                        {feedback.rollback_required ? <code>建议回滚</code> : null}
                      </div>
                    ) : null}
                  </div>
                );
              })}
              {snapshot.release_candidate_applies.slice(0, 2).map((apply) => (
                <div className="signalItem" key={apply.id}>
                  <div className="signalHeader">
                    <strong>{compactID(apply.candidate_id || apply.id)}</strong>
                    <StatusPill tone={toneForStatus(apply.status)} label={apply.decision} />
                  </div>
                  <span>
                    {apply.mode} / {apply.write_enabled ? "写入已启用" : "受保护"} / {apply.action_count} 个动作
                  </span>
                  <div className="signalMeta">
                    {apply.release_branch ? <code>{compactID(apply.release_branch)}</code> : null}
                    {apply.source_branch ? <code>{compactID(apply.source_branch)}</code> : null}
                    {apply.reasons[0] ? <code>{apply.reasons[0]}</code> : null}
                  </div>
                </div>
              ))}
              {snapshot.release_candidate_provider_previews.slice(0, 2).map((preview) => (
                <div className="signalItem" key={preview.id}>
                  <div className="signalHeader">
                    <strong>{compactID(preview.candidate_id || preview.id)}</strong>
                    <StatusPill tone={toneForStatus(preview.status)} label={preview.decision} />
                  </div>
                  <span>
                    {preview.provider || "provider"} / {preview.remote_action_count} 个动作 / {preview.pr_mr_type || "pr/mr"}
                  </span>
                  <div className="signalMeta">
                    {preview.pr_mr_decision ? <code>{preview.pr_mr_decision}</code> : null}
                    {preview.pr_mr_head_branch ? <code>{compactID(preview.pr_mr_head_branch)}</code> : null}
                    {preview.reasons[0] ? <code>{preview.reasons[0]}</code> : null}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="observabilityGrid" hidden={!viewVisible(activeView, ["运行", "Provider", "操作"])}>
          <div className="panel" hidden={activeView !== "运行"}>
            <PanelTitle icon={<TerminalSquare size={18} />} title="Runtime 恢复" meta={`${snapshot.runtime_recoveries.length} 条归档`} />
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
                        {recovery.failure_category} / {recovery.runtime_id || "Runtime 待定"}
                      </span>
                      <div className="signalMeta">
                        {recovery.fallback_candidate ? <code>降级 {recovery.fallback_candidate}</code> : null}
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
                          <span>{artifactState?.status === "loading" ? "加载中" : "产物"}</span>
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
                                <span>{artifact.truncated ? "已截断" : artifact.status}</span>
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
                <div className="emptyState">暂无 Runtime recovery 归档</div>
              )}
            </div>
            <div className="signalList">
              {snapshot.operation_repair_candidates.length > 0 ? (
                snapshot.operation_repair_candidates.slice(0, 3).map((candidate) => (
                  <div className="signalItem" key={candidate.id}>
                    <div className="signalHeader">
                      <strong>{compactID(candidate.operation_id || candidate.id)}</strong>
                      <StatusPill tone={toneForStatus(candidate.failure_class)} label={candidate.failure_class} />
                    </div>
                    <span>{`${candidate.decision} / ${candidate.signal_type}`}</span>
                    <div className="signalMeta">
                      {candidate.repair_plan_id ? <code>{compactID(candidate.repair_plan_id)}</code> : null}
                      {candidate.evidence_refs.length > 0 ? <code>{candidate.evidence_refs.length} 个证据</code> : null}
                      {candidate.review_required ? <code>需要复核</code> : null}
                      {candidate.review_decision ? <code>{candidate.review_decision}</code> : null}
                      {candidate.issue_id ? <code>{compactID(candidate.issue_id)}</code> : null}
                      {candidate.repair_attempt_id ? <code>{compactID(candidate.repair_attempt_id)}</code> : null}
                    </div>
                    {candidate.review_reason ? <small>{candidate.review_reason}</small> : null}
                    {candidate.status === "review_required" || candidate.review_required ? (
                      <div className="signalActions">
                        <button
                          className="inlineActionButton"
                          disabled={repairActionState[candidate.id]?.status === "running"}
                          onClick={() => setRepairReviewModal({ candidateID: candidate.id, decision: "approved" })}
                          type="button"
                        >
                          <CheckCircle2 size={13} />
                          <span>批准</span>
                        </button>
                        <button
                          className="inlineActionButton danger"
                          disabled={repairActionState[candidate.id]?.status === "running"}
                          onClick={() => setRepairReviewModal({ candidateID: candidate.id, decision: "rejected" })}
                          type="button"
                        >
                          <AlertTriangle size={13} />
                          <span>拒绝</span>
                        </button>
                        {repairActionState[candidate.id]?.message ? <ActionFeedback state={repairActionState[candidate.id]} /> : null}
                      </div>
                    ) : null}
                  </div>
                ))
              ) : (
                <div className="emptyState compact">暂无操作修复候选</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "运行"}>
            <PanelTitle icon={<CircleDotDashed size={18} />} title="Subagent 积压" meta={`${snapshot.subagent_backlog.length} 个等待中`} />
            <div className="signalList">
              {snapshot.subagent_backlog.length > 0 ? (
                snapshot.subagent_backlog.map((item) => (
                  <div className="signalItem" key={`${item.issue_id}-${item.subagent_id}`}>
                    <div className="signalHeader">
                      <strong>{compactID(item.issue_id)}</strong>
                      <StatusPill tone={toneForStatus(item.status)} label={item.status} />
                    </div>
                    <span>{item.reason || item.failure_category || "等待调度器决策"}</span>
                    <div className="signalMeta">
                      <code>{compactID(item.subagent_id)}</code>
                      <code>
                        重试 {item.retry_count}/{item.max_retries}
                      </code>
                      {item.recovery_id ? <code>{compactID(item.recovery_id)}</code> : null}
                    </div>
                  </div>
                ))
              ) : (
                <div className="emptyState">暂无 Subagent 积压</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "操作"}>
            <PanelTitle icon={<RefreshCw size={18} />} title="控制循环运行" meta={`${snapshot.control_loop_runs.length} 次运行`} />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={controlLoopActionState.status === "running"} onClick={() => void runControlLoop()} type="button">
                <RefreshCw size={13} />
                <span>{controlLoopActionState.status === "running" ? "运行中" : "运行循环"}</span>
              </button>
              <ActionFeedback state={controlLoopActionState} />
            </div>
            <div className="signalList">
              {snapshot.control_loop_runs.length > 0 ? (
                snapshot.control_loop_runs.map((run) => (
                  <div className="signalItem" key={run.id}>
                    <div className="signalHeader">
                      <strong>{compactID(run.id)}</strong>
                      <StatusPill tone={toneForStatus(run.status)} label={run.decision} />
                    </div>
                    <span>
                      {run.trigger} / {run.steps.length} 个步骤 / {shortTimestamp(run.finished_at || run.started_at)}
                    </span>
                    <div className="signalMeta">
                      {run.requested_by ? <code>{run.requested_by}</code> : null}
                      {run.reasons[0] ? <code>{run.reasons[0]}</code> : null}
                    </div>
                    <div className="routeCandidateGrid compact">
                      {run.steps.slice(0, 3).map((step) => (
                        <div className="routeCandidate" key={step.id}>
                          <strong>{step.type}</strong>
                          <span>{step.summary || step.decision}</span>
                          <div className="signalMeta">
                            <code>{step.status}</code>
                            <code>{step.duration_ms}ms</code>
                            {step.evidence_count ? <code>{step.evidence_count} 个证据</code> : null}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ))
              ) : (
                <div className="emptyState">暂无控制循环运行</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "Provider"}>
            <PanelTitle
              icon={<Sparkles size={18} />}
              title="视觉资产"
              meta={`${snapshot.visual_assets.length} 个计划 / ${snapshot.visual_render_executions.length} 次渲染`}
            />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={visualActionState.plan?.status === "running"} onClick={() => setVisualPlanModalOpen(true)} type="button">
                <Sparkles size={13} />
                <span>{visualActionState.plan?.status === "running" ? "创建中" : "创建图计划"}</span>
              </button>
              <ActionFeedback state={visualActionState.plan} />
            </div>
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
                            <span>{actionState?.status === "running" ? "运行中" : "Dry Run"}</span>
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
                        {execution.decision} / {execution.step_count} 个步骤
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
                <div className="emptyState">暂无视觉资产计划</div>
              )}
            </div>
          </div>
        </section>

        <section className="auditGrid" hidden={activeView !== "审计"}>
          <div className="panel">
            <PanelTitle icon={<Lock size={18} />} title="审批队列" meta={`${snapshot.approvals.length} 条记录`} />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={approvalActionState.create?.status === "running"} onClick={() => setApprovalCreateModalOpen(true)} type="button">
                <Lock size={13} />
                <span>{approvalActionState.create?.status === "running" ? "创建中" : "发起审批"}</span>
              </button>
              <ActionFeedback state={approvalActionState.create} />
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
                    <small>{approval.request_reason || approval.decision_reason || `请求人 ${approval.requested_by}`}</small>
                    {approval.status === "pending" ? (
                      <div className="signalActions">
                        <button
                          className="inlineActionButton"
                          disabled={approvalActionState[approval.id]?.status === "running"}
                          onClick={() => setApprovalDecisionModal({ approvalID: approval.id, decision: "approved" })}
                          type="button"
                        >
                          <CheckCircle2 size={13} />
                          <span>批准</span>
                        </button>
                        <button
                          className="inlineActionButton danger"
                          disabled={approvalActionState[approval.id]?.status === "running"}
                          onClick={() => setApprovalDecisionModal({ approvalID: approval.id, decision: "rejected" })}
                          type="button"
                        >
                          <AlertTriangle size={13} />
                          <span>拒绝</span>
                        </button>
                        {approvalActionState[approval.id]?.message ? (
                          <small className={`actionMessage ${approvalActionState[approval.id]?.status}`}>{approvalActionState[approval.id]?.message}</small>
                        ) : null}
                      </div>
                    ) : null}
                  </div>
                ))
              ) : (
                <div className="emptyState">暂无审批记录</div>
              )}
            </div>
          </div>
          <div className="panel">
            <PanelTitle icon={<ScrollText size={18} />} title="审计轨迹" meta={`${snapshot.audit_events.length} 个核心事件`} />
            <div className="signalList auditList">
              {snapshot.audit_events.length > 0 ? (
                snapshot.audit_events.map((event) => (
                  <div className="signalItem auditItem" key={event.id}>
                    <div className="signalHeader">
                      <strong>{event.event}</strong>
                      <StatusPill tone={toneForStatus(event.status || event.decision || event.channel)} label={event.channel} />
                    </div>
                    <span>
                      {event.decision || event.status || "已记录"} / {shortTimestamp(event.ts)}
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
                <div className="emptyState">暂无审计事件</div>
              )}
            </div>
          </div>
        </section>

        <section className="accessGrid" hidden={activeView !== "审计"}>
          <div className="panel">
            <PanelTitle
              icon={<ShieldCheck size={18} />}
              title="访问基线"
              meta={`${activeSessions.length} 个会话 / ${activeTokens.length} 个 token / ${activeServiceAccounts.length} 个服务账号`}
            />
            <div className="accessCommandGrid">
              <div className="accessCommand">
                <div className="controlFormTitle">
                  <UserPlus size={15} />
                  <strong>会话</strong>
                </div>
                <span>录入控制台用户、显示名和角色。</span>
                <div className="buttonRow">
                  <button className="inlineActionButton" disabled={accessActionState.session?.status === "running"} onClick={() => setSessionModalOpen(true)} type="button">
                    <UserPlus size={13} />
                    <span>{accessActionState.session?.status === "running" ? "创建中" : "创建会话"}</span>
                  </button>
                  <ActionFeedback state={accessActionState.session} />
                </div>
              </div>

              <div className="accessCommand">
                <div className="controlFormTitle">
                  <KeyRound size={15} />
                  <strong>API Token</strong>
                </div>
                <span>生成可撤销的 API Token，并回显一次性密钥片段。</span>
                <div className="buttonRow">
                  <button className="inlineActionButton" disabled={accessActionState.token?.status === "running"} onClick={() => setTokenModalOpen(true)} type="button">
                    <KeyRound size={13} />
                    <span>{accessActionState.token?.status === "running" ? "创建中" : "创建 Token"}</span>
                  </button>
                  <ActionFeedback state={accessActionState.token} />
                </div>
              </div>

              <div className="accessCommand">
                <div className="controlFormTitle">
                  <ShieldCheck size={15} />
                  <strong>服务账号</strong>
                </div>
                <span>保存 Release Bot、部署执行等服务账号基线。</span>
                <div className="buttonRow">
                  <button className="inlineActionButton" disabled={accessActionState.service?.status === "running"} onClick={() => setServiceAccountModalOpen(true)} type="button">
                    <ShieldCheck size={13} />
                    <span>{accessActionState.service?.status === "running" ? "保存中" : "保存服务账号"}</span>
                  </button>
                  <ActionFeedback state={accessActionState.service} />
                </div>
              </div>
            </div>
            <div className="accessList">
              <div className="accessCard">
                <div className="accessCardHeader">
                  <strong>会话</strong>
                  <StatusPill tone={activeSessions.length > 0 ? "ok" : "neutral"} label={`${activeSessions.length} 活跃`} />
                </div>
                {snapshot.auth_sessions.slice(0, 3).map((session) => (
                  <div className="accessRow" key={session.id}>
                    <div>
                      <strong>{session.display_name || session.user_id}</strong>
                      <span>{session.roles.join(", ") || "角色待定"}</span>
                    </div>
                    <code>{shortTimestamp(session.created_at)}</code>
                    {session.status === "active" ? (
                      <button
                        className="inlineActionButton danger compactButton"
                        disabled={accessActionState[session.id]?.status === "running"}
                        onClick={() => setSessionRevokeModalID(session.id)}
                        type="button"
                      >
                        <span>撤销</span>
                      </button>
                    ) : null}
                  </div>
                ))}
                {snapshot.auth_sessions.map((session) =>
                  accessActionState[session.id]?.message ? <ActionFeedback key={`${session.id}-feedback`} state={accessActionState[session.id]} /> : null,
                )}
                {snapshot.auth_sessions.length === 0 ? <div className="emptyState">暂无会话</div> : null}
              </div>

              <div className="accessCard">
                <div className="accessCardHeader">
                  <strong>API Tokens</strong>
                  <StatusPill tone={activeTokens.length > 0 ? "ok" : "neutral"} label={`${activeTokens.length} 活跃`} />
                </div>
                {snapshot.api_tokens.slice(0, 3).map((token) => (
                  <div className="accessRow" key={token.id}>
                    <div>
                      <strong>{token.name}</strong>
                      <span>{token.scopes.join(", ") || "scope 待定"}</span>
                    </div>
                    <code>{token.token_prefix || compactID(token.id)}</code>
                    {token.status === "active" ? (
                      <button
                        className="inlineActionButton danger compactButton"
                        disabled={accessActionState[token.id]?.status === "running"}
                        onClick={() => setTokenRevokeModalID(token.id)}
                        type="button"
                      >
                        <span>撤销</span>
                      </button>
                    ) : null}
                  </div>
                ))}
                {snapshot.api_tokens.map((token) =>
                  accessActionState[token.id]?.message ? <ActionFeedback key={`${token.id}-feedback`} state={accessActionState[token.id]} /> : null,
                )}
                {snapshot.api_tokens.length === 0 ? <div className="emptyState">暂无 API Token</div> : null}
              </div>

              <div className="accessCard">
                <div className="accessCardHeader">
                  <strong>服务账号</strong>
                  <StatusPill tone={activeServiceAccounts.length > 0 ? "ok" : "neutral"} label={`${activeServiceAccounts.length} 活跃`} />
                </div>
                {snapshot.service_accounts.slice(0, 3).map((account) => (
                  <div className="accessRow" key={account.id}>
                    <div>
                      <strong>{account.name}</strong>
                      <span>{account.roles.join(", ") || "角色待定"}</span>
                    </div>
                    <code>{compactID(account.id)}</code>
                  </div>
                ))}
                {snapshot.service_accounts.length === 0 ? <div className="emptyState">暂无服务账号</div> : null}
              </div>
            </div>
          </div>
        </section>

        <section className="observabilityGrid" hidden={activeView !== "技能"}>
          <div className="panel">
            <PanelTitle icon={<MemoryStick size={18} />} title="Skill 注册" meta={`${snapshot.skills.length} 个 Skill`} />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={skillActionState.upsert?.status === "running"} onClick={() => setSkillModalOpen(true)} type="button">
                <MemoryStick size={13} />
                <span>新增 Skill</span>
              </button>
              <button className="inlineActionButton" disabled={skillActionState.recommend?.status === "running"} onClick={() => setSkillRecommendationModalOpen(true)} type="button">
                <Search size={13} />
                <span>推荐</span>
              </button>
              <ActionFeedback state={skillActionState.upsert} />
              <ActionFeedback state={skillActionState.recommend} />
            </div>
            <div className="signalList">
              {skillRecommendation ? (
                <div className="signalItem">
                  <div className="signalHeader">
                    <strong>{compactID(skillRecommendation.id)}</strong>
                    <StatusPill tone={skillRecommendation.candidates.length > 0 ? "ok" : "warning"} label={`${skillRecommendation.candidates.length} 个候选`} />
                  </div>
                  <span>
                    {skillRecommendation.role} / {skillRecommendation.task_type || "task"} / {skillRecommendation.risk_level}
                  </span>
                  <div className="signalMeta">
                    {skillRecommendation.candidates.slice(0, 4).map((candidate) => (
                      <code key={candidate.skill_id}>
                        {candidate.name}:{candidate.score.toFixed(2)}
                      </code>
                    ))}
                  </div>
                </div>
              ) : null}
              {snapshot.skills.length > 0 ? (
                snapshot.skills.map((skill) => (
                  <div className="signalItem" key={skill.id}>
                    <div className="signalHeader">
                      <strong>{skill.name}</strong>
                      <StatusPill tone={skill.enabled ? "ok" : "neutral"} label={skill.enabled ? "enabled" : "disabled"} />
                    </div>
                    <span>
                      {skill.source} / {skill.risk_level}
                      {skill.version ? ` / ${skill.version}` : ""}
                    </span>
                    <div className="signalMeta">
                      <code>{skill.id}</code>
                      {skill.compatible_roles.slice(0, 3).map((role) => (
                        <code key={`${skill.id}-${role}`}>{role}</code>
                      ))}
                      {skill.tags.slice(0, 3).map((tag) => (
                        <code key={`${skill.id}-${tag}`}>{tag}</code>
                      ))}
                    </div>
                    {skill.description ? <small>{skill.description}</small> : null}
                    <div className="signalActions">
                      <button
                        className="inlineActionButton danger"
                        disabled={!skill.enabled || skillActionState[skill.id]?.status === "running"}
                        onClick={() => setSkillDisableModalID(skill.id)}
                        type="button"
                      >
                        <span>禁用</span>
                      </button>
                      <ActionFeedback state={skillActionState[skill.id]} />
                    </div>
                  </div>
                ))
              ) : (
                <div className="emptyState">暂无 Skill</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<Layers3 size={18} />} title="Skill 绑定" meta={`${snapshot.skill_bindings.length} 条绑定`} />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={skillActionState.binding?.status === "running"} onClick={() => setSkillBindingModalOpen(true)} type="button">
                <Layers3 size={13} />
                <span>新增绑定</span>
              </button>
              <ActionFeedback state={skillActionState.binding} />
            </div>
            <div className="signalList">
              {snapshot.skill_bindings.length > 0 ? (
                snapshot.skill_bindings.map((binding) => (
                  <div className="signalItem" key={binding.id}>
                    <div className="signalHeader">
                      <strong>{binding.skill_id}</strong>
                      <StatusPill tone={toneForStatus(binding.status)} label={binding.status} />
                    </div>
                    <span>
                      {binding.target_type} / {binding.target_id} / 优先级 {binding.priority}
                    </span>
                    <div className="signalMeta">
                      <code>{compactID(binding.id)}</code>
                      {Object.entries(binding.config).map(([key, value]) => (
                        <code key={`${binding.id}-${key}`}>
                          {key}:{value}
                        </code>
                      ))}
                    </div>
                    <div className="signalActions">
                      <button
                        className="inlineActionButton danger"
                        disabled={binding.status === "disabled" || skillActionState[binding.id]?.status === "running"}
                        onClick={() => setSkillBindingDisableModalID(binding.id)}
                        type="button"
                      >
                        <span>禁用</span>
                      </button>
                      <ActionFeedback state={skillActionState[binding.id]} />
                    </div>
                  </div>
                ))
              ) : (
                <div className="emptyState">暂无 Skill 绑定</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<CheckCircle2 size={18} />} title="Skill 效果" meta={`${snapshot.skill_effectiveness.length} 条记录`} />
            <div className="rowActions panelActionRow">
              <button
                className="inlineActionButton"
                disabled={skillActionState.effectiveness?.status === "running"}
                onClick={() => setSkillEffectivenessModalOpen(true)}
                type="button"
              >
                <CheckCircle2 size={13} />
                <span>记录效果</span>
              </button>
              <ActionFeedback state={skillActionState.effectiveness} />
            </div>
            <div className="signalList">
              {snapshot.skill_effectiveness.length > 0 ? (
                snapshot.skill_effectiveness.map((record) => (
                  <div className="signalItem" key={record.id}>
                    <div className="signalHeader">
                      <strong>{record.skill_id}</strong>
                      <StatusPill tone={record.quality_impact === "positive" ? "ok" : toneForStatus(record.outcome)} label={record.outcome} />
                    </div>
                    <span>
                      {record.quality_impact} / {record.rework_reduced ? "减少返工" : "未减少返工"} / {record.duration_seconds || 0}s
                    </span>
                    <div className="signalMeta">
                      {record.issue_id ? <code>{compactID(record.issue_id)}</code> : null}
                      {record.binding_id ? <code>{compactID(record.binding_id)}</code> : null}
                      {record.run_id ? <code>{compactID(record.run_id)}</code> : null}
                      {record.findings.length ? <code>{record.findings.length} 条发现</code> : null}
                    </div>
                  </div>
                ))
              ) : (
                <div className="emptyState">暂无 Skill 效果记录</div>
              )}
            </div>
          </div>
        </section>

        <section className="bottomGrid" hidden={!viewVisible(activeView, ["Provider", "部署"])}>
          <div className="panel" hidden={activeView !== "Provider"}>
            <PanelTitle icon={<Sparkles size={18} />} title="Provider 与 Runtime" meta={`${snapshot.providers.length} 个已注册`} />
            <div className="rowActions panelActionRow">
              <button className="inlineActionButton" disabled={providerActionState.upsert?.status === "running"} onClick={() => setProviderModalOpen(true)} type="button">
                <Sparkles size={13} />
                <span>新增 Provider</span>
              </button>
              <button className="inlineActionButton" disabled={providerActionState.ops?.status === "running"} onClick={() => setProviderOpsModalOpen(true)} type="button">
                <RefreshCw size={13} />
                <span>刷新 Ops</span>
              </button>
              <ActionFeedback state={providerActionState.upsert} />
              <ActionFeedback state={providerActionState.ops} />
            </div>
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
                  <StatusPill tone={toneForStatus(provider.health_status || (provider.enabled ? "ok" : "unknown"))} label={provider.health_status || "unknown"} />
                  <button
                    className="inlineActionButton compactButton"
                    disabled={providerActionState[`${provider.id}:ops`]?.status === "running"}
                    onClick={() => setProviderOpsSnapshotModalID(provider.id)}
                    type="button"
                  >
                    <span>Ops</span>
                  </button>
                  <button
                    className="inlineActionButton danger compactButton"
                    disabled={!provider.enabled || providerActionState[provider.id]?.status === "running"}
                    onClick={() => setProviderDisableModalID(provider.id)}
                    type="button"
                  >
                    <span>禁用</span>
                  </button>
                  {providerActionState[`${provider.id}:ops`]?.message ? <ActionFeedback state={providerActionState[`${provider.id}:ops`]} /> : null}
                  {providerActionState[provider.id]?.message ? <ActionFeedback state={providerActionState[provider.id]} /> : null}
                </div>
              ))}
            </div>
            <div className="routePreviewBox">
              <div className="controlForm compact">
                <label>
                  <FieldLabel required>角色</FieldLabel>
                  <select onChange={(event) => setProviderRouteForm((current) => ({ ...current, role: event.target.value }))} value={providerRouteForm.role}>
                    <option value="frontend">前端</option>
                    <option value="backend">后端</option>
                    <option value="devops">DevOps</option>
                    <option value="review">复核</option>
                  </select>
                </label>
                <label>
                  <FieldLabel required>任务</FieldLabel>
                  <select
                    onChange={(event) => setProviderRouteForm((current) => ({ ...current, taskType: event.target.value }))}
                    value={providerRouteForm.taskType}
                  >
                    <option value="requirement_planning">需求规划</option>
                    <option value="architecture_planning">架构规划</option>
                    <option value="memory_extraction">Memory 提取</option>
                    <option value="image_generation">图像生成</option>
                  </select>
                </label>
                <label>
                  <span>输出</span>
                  <select
                    onChange={(event) => setProviderRouteForm((current) => ({ ...current, outputType: event.target.value }))}
                    value={providerRouteForm.outputType}
                  >
                    <option value="code">代码</option>
                    <option value="markdown">Markdown</option>
                    <option value="architecture_diagram">架构图</option>
                    <option value="image">图像</option>
                  </select>
                </label>
                <label className="checkboxLine">
                  <input
                    checked={providerRouteForm.requiresRepoEdit}
                    onChange={(event) => setProviderRouteForm((current) => ({ ...current, requiresRepoEdit: event.target.checked }))}
                    type="checkbox"
                  />
                  <span>需要改仓库</span>
                </label>
                <label className="checkboxLine">
                  <input
                    checked={providerRouteForm.includesProjectMemory}
                    onChange={(event) => setProviderRouteForm((current) => ({ ...current, includesProjectMemory: event.target.checked }))}
                    type="checkbox"
                  />
                  <span>Memory</span>
                </label>
              </div>
              <SchemaFeedback errors={schemaErrors.providerRoute} />
              <div className="rowActions panelActionRow">
                <button className="inlineActionButton" disabled={providerRouteState.status === "running"} onClick={() => void previewProviderRoute()} type="button">
                  <Search size={13} />
                  <span>{providerRouteState.status === "running" ? "路由中" : "路由预览"}</span>
                </button>
                <ActionFeedback state={providerRouteState} />
              </div>
              {providerRoute ? (
                <>
                  <div className="routeSummary">
                    <strong>{providerRoute.provider_id || "未选择 Provider"}</strong>
                    <span>{providerRoute.explanation?.summary || providerRoute.reason || providerRoute.decision}</span>
                  </div>
                  <div className="routeCandidateGrid">
                    {(providerRoute.candidates ?? []).slice(0, 6).map((candidate, index) => (
                      <div className="routeCandidate" key={candidate.provider_id || `route-candidate-${index}`}>
                        <div className="signalHeader">
                          <strong>{candidate.provider_id}</strong>
                          <StatusPill tone={toneForStatus(candidate.status ?? "")} label={candidate.status ?? "候选"} />
                        </div>
                        <span>{candidate.reason}</span>
                        <div className="signalMeta">
                          {candidate.runtime_id ? <code>{candidate.runtime_id}</code> : null}
                          {candidate.model_id ? <code>{candidate.model_id}</code> : null}
                          <code>评分 {candidate.score ?? 0}</code>
                        </div>
                      </div>
                    ))}
                  </div>
                </>
              ) : null}
            </div>
            <div className="telemetryList">
              {snapshot.provider_telemetry.length > 0 ? (
                snapshot.provider_telemetry.slice(0, 4).map((record) => (
                  <div className="telemetryItem" key={record.id}>
                    <div>
                      <strong>{record.provider_id}</strong>
                      <span>
                        {record.source}
                        {record.total_tokens ? ` / ${record.total_tokens} tokens` : ""}
                      </span>
                    </div>
                    <StatusPill tone={toneForStatus(record.quality_status || record.health_status || record.decision)} label={record.quality_status || record.health_status || record.decision} />
                    <code>{record.cost_status || record.quota_status || record.runtime_status || "ops"}</code>
                  </div>
                ))
              ) : (
                <div className="emptyState">暂无 Provider 遥测</div>
              )}
            </div>
          </div>

          <div className="panel" hidden={activeView !== "部署"}>
            <PanelTitle icon={<Server size={18} />} title="服务器资源" meta={`${snapshot.lifecycle_alerts.length} 个告警 / ${snapshot.maintenance_records.length} 条维护记录`} />
            <div className="rowActions wide panelActionRow">
              <button className="inlineActionButton" disabled={resourceCreateActionState.status === "running"} onClick={() => setResourceCreateModalOpen(true)} type="button">
                <Server size={13} />
                <span>{resourceCreateActionState.status === "running" ? "登记中" : "新增服务器"}</span>
              </button>
              <button className="inlineActionButton" disabled={resourceActionState["scan:maintenance"]?.status === "running"} onClick={() => setResourceScanModal("maintenance")} type="button">
                <RefreshCw size={13} />
                <span>维护扫描</span>
              </button>
              <button className="inlineActionButton" disabled={resourceActionState["scan:lifecycle"]?.status === "running"} onClick={() => setResourceScanModal("lifecycle")} type="button">
                <Activity size={13} />
                <span>生命周期扫描</span>
              </button>
              <button className="inlineActionButton" disabled={resourceActionState["scan:health"]?.status === "running"} onClick={() => setResourceScanModal("health")} type="button">
                <ShieldCheck size={13} />
                <span>健康扫描</span>
              </button>
              <ActionFeedback state={resourceCreateActionState} />
              <ActionFeedback state={resourceActionState["scan:maintenance"]} />
              <ActionFeedback state={resourceActionState["scan:lifecycle"]} />
              <ActionFeedback state={resourceActionState["scan:health"]} />
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
                        {resource.expiration_state ? ` / ${resource.expiration_state}` : ""}
                      </span>
                      {resource.last_deployment ? (
                        <span>{`最近 ${resource.last_deployment.kind} / ${compactID(resource.last_deployment.execution_id || resource.last_deployment.deployment_id || resource.last_deployment.id)}`}</span>
                      ) : null}
                    </div>
                    <StatusPill tone={toneForStatus(resource.expiration_state || (resource.environment === "production" ? "warning" : "ok"))} label={resource.environment} />
                    <div className="rowActions">
                      <button
                        className="inlineActionButton"
                        disabled={resourceActionState[resource.id]?.status === "running"}
                        onClick={() => setResourceActionModal({ resourceID: resource.id, action: "renew" })}
                        type="button"
                      >
                        <RefreshCw size={13} />
                        <span>续期</span>
                      </button>
                      <button
                        className="inlineActionButton danger"
                        disabled={resourceActionState[resource.id]?.status === "running"}
                        onClick={() => setResourceActionModal({ resourceID: resource.id, action: "retire" })}
                        type="button"
                      >
                        <Wrench size={13} />
                        <span>退役</span>
                      </button>
                      <button
                        className="inlineActionButton danger"
                        disabled={resourceActionState[`${resource.id}:disable`]?.status === "running"}
                        onClick={() => setResourceDisableModalID(resource.id)}
                        type="button"
                      >
                        <X size={13} />
                        <span>禁用</span>
                      </button>
                    </div>
                    {resourceActionState[resource.id]?.message ? <ActionFeedback state={resourceActionState[resource.id]} /> : null}
                    {resourceActionState[`${resource.id}:disable`]?.message ? <ActionFeedback state={resourceActionState[`${resource.id}:disable`]} /> : null}
                  </div>
                ))
              ) : (
                <div className="emptyState">暂无已登记资源</div>
              )}
            </div>
            <div className="maintenanceList">
              {latestResourceDeploymentRef ? (
                <div className="maintenanceItem">
                  <div>
                    <strong>{latestResourceDeploymentRef.resource_id}</strong>
                    <span>{`${latestResourceDeploymentRef.kind} / ${latestResourceDeploymentRef.decision}`}</span>
                    <span>{compactID(latestResourceDeploymentRef.execution_id || latestResourceDeploymentRef.deployment_id || latestResourceDeploymentRef.id)}</span>
                  </div>
                  <StatusPill tone={toneForStatus(latestResourceDeploymentRef.status)} label={latestResourceDeploymentRef.environment || latestResourceDeploymentRef.status} />
                </div>
              ) : null}
              {snapshot.resource_deployment_refs.slice(1, 3).map((ref) => (
                <div className="maintenanceItem" key={ref.id}>
                  <div>
                    <strong>{ref.resource_id}</strong>
                    <span>{`${ref.kind} / ${ref.decision}`}</span>
                  </div>
                  <StatusPill tone={toneForStatus(ref.status)} label={ref.mode || ref.status} />
                </div>
              ))}
            </div>
            <div className="maintenanceList">
              {snapshot.lifecycle_alerts.length > 0 ? (
                snapshot.lifecycle_alerts.slice(0, 3).map((alert) => (
                  <div className="maintenanceItem" key={alert.id}>
                    <div>
                      <strong>{alert.resource_id || compactID(alert.id)}</strong>
                      <span>{alert.reason || alert.type}</span>
                    </div>
                    <StatusPill tone={toneForStatus(alert.severity || alert.status)} label={alert.expiration_state || alert.health_status || alert.type} />
                  </div>
                ))
              ) : (
                <div className="emptyState compact">暂无生命周期告警</div>
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
                <div className="emptyState">暂无维护记录</div>
              )}
            </div>
          </div>

          <div className="panel releasePanel" hidden={activeView !== "部署"}>
            <PanelTitle icon={<GitBranch size={18} />} title="Release 流水线" meta={`${snapshot.git_provider_plans.length} 个 PR/MR 计划`} />
            <div className="releaseSteps">
              <span>已接受 issues</span>
              <ChevronRight size={15} />
              <span>release 分支</span>
              <ChevronRight size={15} />
              <span>tag + PR/MR</span>
              <ChevronRight size={15} />
              <span>部署计划</span>
            </div>
            <div className="panelCommandBody">
              <div>
                <strong>{releaseProviderForm.releaseID || "Release Provider"}</strong>
                <span>{releaseProviderForm.approved ? `已配置审批 ${compactID(releaseProviderForm.approvalID || "待填写")}` : "预览或发布前补充 Release 信息"}</span>
              </div>
              <div className="rowActions wide">
                <button className="inlineActionButton" disabled={releaseProviderActionState.status === "running"} onClick={() => setReleaseProviderModalOpen(true)} type="button">
                  <Rocket size={13} />
                  <span>{releaseProviderActionState.status === "running" ? "执行中" : "Release Provider"}</span>
                </button>
                <ActionFeedback state={releaseProviderActionState} />
              </div>
            </div>
            <div className="prmrList">
              {snapshot.git_provider_plans.length > 0 ? (
                snapshot.git_provider_plans.slice(0, 3).map((plan) => (
                  <div className="prmrItem" key={plan.id}>
                    <div>
                      <strong>{plan.issue_id || compactID(plan.id)}</strong>
                      <span>
                        {plan.provider} / {plan.target_branch || "分支待定"}
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
                        <span>预览</span>
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
                        onClick={() => setGitCreateModalPlanID(plan.id)}
                        type="button"
                      >
                        <GitBranch size={13} />
                        <span>创建</span>
                      </button>
                      {gitActionState[plan.id]?.message ? <ActionFeedback state={gitActionState[plan.id]} /> : null}
                    </div>
                  </div>
                ))
              ) : (
                <div className="emptyState">暂无 PR/MR 计划记录</div>
              )}
            </div>
            <div className="approvalStrip">
              <Lock size={16} />
              <span>生产部署需要审批、smoke、监控和回滚计划</span>
            </div>
          </div>
        </section>

        {projectModalOpen ? (
          <ModalShell title="接入项目" onClose={() => setProjectModalOpen(false)}>
            <form className="controlForm projectOnboardingForm modalForm" onSubmit={submitProjectOnboarding}>
              <div className="modeSwitch formWide" role="group" aria-label="项目来源">
                <button
                  className={projectForm.mode === "local" ? "active" : ""}
                  onClick={() => setProjectForm((current) => ({ ...current, mode: "local" }))}
                  type="button"
                >
                  <Boxes size={14} />
                  <span>本地项目</span>
                </button>
                <button
                  className={projectForm.mode === "remote" ? "active" : ""}
                  onClick={() => setProjectForm((current) => ({ ...current, mode: "remote" }))}
                  type="button"
                >
                  <GitBranch size={14} />
                  <span>Git 项目</span>
                </button>
              </div>
              {projectForm.mode === "local" ? (
                <label className="formWide">
                  <FieldLabel required>本地路径</FieldLabel>
                  <input
                    onChange={(event) => setProjectForm((current) => ({ ...current, localPath: event.target.value }))}
                    placeholder="/path/to/repo"
                    required
                    value={projectForm.localPath}
                  />
                </label>
              ) : (
                <>
                  <label className="formWide">
                    <FieldLabel required>Git 地址</FieldLabel>
                    <input
                      onChange={(event) => setProjectForm((current) => ({ ...current, remoteURL: event.target.value }))}
                      placeholder="git@github.com:owner/repo.git"
                      required
                      value={projectForm.remoteURL}
                    />
                  </label>
                  <label>
                    <span>Clone 目录</span>
                    <input
                      onChange={(event) => setProjectForm((current) => ({ ...current, destPath: event.target.value }))}
                      placeholder="留空自动生成"
                      value={projectForm.destPath}
                    />
                  </label>
                  <label>
                    <span>Provider</span>
                    <select onChange={(event) => setProjectForm((current) => ({ ...current, provider: event.target.value }))} value={projectForm.provider}>
                      <option value="">自动识别</option>
                      <option value="github">GitHub</option>
                      <option value="gitee">Gitee</option>
                      <option value="gitlab">GitLab</option>
                      <option value="generic_git">Generic Git</option>
                    </select>
                  </label>
                </>
              )}
              <SchemaFeedback errors={schemaErrors.projectOnboarding} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setProjectModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={projectActionState.status === "running"} type="submit">
                  {projectForm.mode === "local" ? <Boxes size={13} /> : <GitBranch size={13} />}
                  <span>{projectActionState.status === "running" ? "接入中" : "接入项目"}</span>
                </button>
                <ActionFeedback state={projectActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {clarificationModalOpen ? (
          <ModalShell title="补充需求澄清" onClose={() => setClarificationModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitRequirementClarification}>
              <div className="modalContext formWide">
                <span>澄清问题</span>
                <strong>{requirementState.message || "请补充需求细节"}</strong>
              </div>
              <label className="formWide">
                <FieldLabel required>补充回答</FieldLabel>
                <textarea
                  onChange={(event) => setClarificationAnswer(event.target.value)}
                  placeholder="补充业务边界、验收标准、限制条件或优先级..."
                  required
                  value={clarificationAnswer}
                />
              </label>
              <SchemaFeedback errors={schemaErrors.requirementClarification} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setClarificationModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={requirementState.status === "planning"} type="submit">
                  <ScrollText size={13} />
                  <span>{requirementState.status === "planning" ? "提交中" : "提交回答"}</span>
                </button>
              </div>
            </form>
          </ModalShell>
        ) : null}

        {deploymentPlanModalOpen ? (
          <ModalShell title="创建部署计划" onClose={() => setDeploymentPlanModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitDeploymentPlan}>
              <label>
                <FieldLabel required>Release ID</FieldLabel>
                <input onChange={(event) => setDeploymentPlanForm((current) => ({ ...current, releaseID: event.target.value }))} required value={deploymentPlanForm.releaseID} />
              </label>
              <label>
                <FieldLabel required>环境</FieldLabel>
                <select onChange={(event) => setDeploymentPlanForm((current) => ({ ...current, environment: event.target.value }))} required value={deploymentPlanForm.environment}>
                  <option value="test_dev">test_dev</option>
                  <option value="staging">staging</option>
                  <option value="production">production</option>
                </select>
              </label>
              <label className="formWide">
                <span>Resource IDs</span>
                <input
                  onChange={(event) => setDeploymentPlanForm((current) => ({ ...current, resourceIDs: event.target.value }))}
                  placeholder="dev-1,prod-1"
                  value={deploymentPlanForm.resourceIDs}
                />
              </label>
              <label className="checkboxLine">
                <input checked={deploymentPlanForm.approved} onChange={(event) => setDeploymentPlanForm((current) => ({ ...current, approved: event.target.checked }))} type="checkbox" />
                <span>已批准创建计划</span>
              </label>
              <SchemaFeedback errors={schemaErrors.deploymentPlan} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setDeploymentPlanModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} type="submit">
                  <Layers3 size={13} />
                  <span>{deploymentActionState.status === "running" ? "创建中" : "创建计划"}</span>
                </button>
                <ActionFeedback state={deploymentActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {deploymentExecuteModalOpen ? (
          <ModalShell title="执行部署" onClose={() => setDeploymentExecuteModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitDeploymentExecute}>
              <label>
                <FieldLabel required>Deployment ID</FieldLabel>
                <input onChange={(event) => setDeploymentExecuteForm((current) => ({ ...current, deploymentID: event.target.value }))} required value={deploymentExecuteForm.deploymentID} />
              </label>
              <label>
                <FieldLabel required>模式</FieldLabel>
                <select onChange={(event) => setDeploymentExecuteForm((current) => ({ ...current, mode: event.target.value }))} required value={deploymentExecuteForm.mode}>
                  <option value="dry_run">dry_run</option>
                  <option value="ssh_execute">ssh_execute</option>
                  <option value="local_execute">local_execute</option>
                </select>
              </label>
              <label>
                <span>环境</span>
                <select onChange={(event) => setDeploymentExecuteForm((current) => ({ ...current, environment: event.target.value }))} value={deploymentExecuteForm.environment}>
                  <option value="test_dev">test_dev</option>
                  <option value="staging">staging</option>
                  <option value="production">production</option>
                </select>
              </label>
              <label className="checkboxLine">
                <input checked={deploymentExecuteForm.approved} onChange={(event) => setDeploymentExecuteForm((current) => ({ ...current, approved: event.target.checked }))} type="checkbox" />
                <span>已批准执行</span>
              </label>
              <label>
                <FieldLabel required={deploymentExecuteForm.approved}>Approval ID</FieldLabel>
                <input
                  onChange={(event) => setDeploymentExecuteForm((current) => ({ ...current, approvalID: event.target.value }))}
                  required={deploymentExecuteForm.approved}
                  value={deploymentExecuteForm.approvalID}
                />
              </label>
              <label className="formWide">
                <span>Commands</span>
                <textarea
                  onChange={(event) => setDeploymentExecuteForm((current) => ({ ...current, commands: event.target.value }))}
                  placeholder="每行一条命令；dry_run 可留空"
                  value={deploymentExecuteForm.commands}
                />
              </label>
              <SchemaFeedback errors={schemaErrors.deploymentExecute} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setDeploymentExecuteModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={deploymentActionState.status === "running"} type="submit">
                  <Play size={13} />
                  <span>{deploymentActionState.status === "running" ? "执行中" : "提交执行"}</span>
                </button>
                <ActionFeedback state={deploymentActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {resourceCreateModalOpen ? (
          <ModalShell title="新增服务器" onClose={() => setResourceCreateModalOpen(false)}>
            <form className="controlForm resourceCreateForm modalForm" onSubmit={submitResourceCreate}>
              <label>
                <FieldLabel required>资源 ID</FieldLabel>
                <input onChange={(event) => setResourceCreateForm((current) => ({ ...current, id: event.target.value }))} required value={resourceCreateForm.id} />
              </label>
              <label>
                <FieldLabel required>环境</FieldLabel>
                <select onChange={(event) => setResourceCreateForm((current) => ({ ...current, environment: event.target.value }))} required value={resourceCreateForm.environment}>
                  <option value="test_dev">test_dev</option>
                  <option value="staging">staging</option>
                  <option value="production">production</option>
                </select>
              </label>
              <label>
                <FieldLabel required>Host</FieldLabel>
                <input onChange={(event) => setResourceCreateForm((current) => ({ ...current, host: event.target.value }))} required value={resourceCreateForm.host} />
              </label>
              <label>
                <FieldLabel required>Provider</FieldLabel>
                <input onChange={(event) => setResourceCreateForm((current) => ({ ...current, provider: event.target.value }))} required value={resourceCreateForm.provider} />
              </label>
              <label>
                <FieldLabel required>Owner</FieldLabel>
                <input onChange={(event) => setResourceCreateForm((current) => ({ ...current, owner: event.target.value }))} required value={resourceCreateForm.owner} />
              </label>
              <label>
                <FieldLabel required>Auth Ref</FieldLabel>
                <input onChange={(event) => setResourceCreateForm((current) => ({ ...current, authRef: event.target.value }))} required value={resourceCreateForm.authRef} />
              </label>
              <label>
                <FieldLabel required={resourceCreateForm.environment === "production"}>到期日</FieldLabel>
                <input
                  onChange={(event) => setResourceCreateForm((current) => ({ ...current, expiresAt: event.target.value }))}
                  required={resourceCreateForm.environment === "production"}
                  type="date"
                  value={resourceCreateForm.expiresAt}
                />
              </label>
              <label>
                <span>维护窗口</span>
                <input
                  onChange={(event) => setResourceCreateForm((current) => ({ ...current, maintenanceWindow: event.target.value }))}
                  placeholder="always / Sun 02:00"
                  value={resourceCreateForm.maintenanceWindow}
                />
              </label>
              <label>
                <span>用途</span>
                <input onChange={(event) => setResourceCreateForm((current) => ({ ...current, purpose: event.target.value }))} value={resourceCreateForm.purpose} />
              </label>
              <label>
                <span>OS</span>
                <input onChange={(event) => setResourceCreateForm((current) => ({ ...current, os: event.target.value }))} value={resourceCreateForm.os} />
              </label>
              <label>
                <span>CPU</span>
                <input onChange={(event) => setResourceCreateForm((current) => ({ ...current, cpu: event.target.value }))} inputMode="numeric" value={resourceCreateForm.cpu} />
              </label>
              <label>
                <span>内存 GB</span>
                <input
                  onChange={(event) => setResourceCreateForm((current) => ({ ...current, memoryGB: event.target.value }))}
                  inputMode="numeric"
                  value={resourceCreateForm.memoryGB}
                />
              </label>
              <label>
                <span>磁盘 GB</span>
                <input
                  onChange={(event) => setResourceCreateForm((current) => ({ ...current, diskGB: event.target.value }))}
                  inputMode="numeric"
                  value={resourceCreateForm.diskGB}
                />
              </label>
              <label>
                <span>健康检查</span>
                <select onChange={(event) => setResourceCreateForm((current) => ({ ...current, healthType: event.target.value }))} value={resourceCreateForm.healthType}>
                  <option value="manual">manual</option>
                  <option value="http">http</option>
                  <option value="tcp">tcp</option>
                </select>
              </label>
              <label>
                <span>检查目标</span>
                <input onChange={(event) => setResourceCreateForm((current) => ({ ...current, healthTarget: event.target.value }))} value={resourceCreateForm.healthTarget} />
              </label>
              <SchemaFeedback errors={schemaErrors.resourceCreate} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setResourceCreateModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={resourceCreateActionState.status === "running"} type="submit">
                  <Server size={13} />
                  <span>{resourceCreateActionState.status === "running" ? "登记中" : "登记服务器"}</span>
                </button>
                <ActionFeedback state={resourceCreateActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {approvalDecisionModal ? (
          <ModalShell title={approvalDecisionModal.decision === "approved" ? "批准审批" : "拒绝审批"} onClose={() => setApprovalDecisionModal(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitApprovalDecision}>
              <div className="modalContext formWide">
                <span>审批</span>
                <strong>{compactID(approvalDecisionModal.approvalID)}</strong>
              </div>
              <label>
                <FieldLabel required>决策人</FieldLabel>
                <input
                  onChange={(event) => setApprovalForm((current) => ({ ...current, decidedBy: event.target.value }))}
                  required
                  value={approvalForm.decidedBy}
                />
              </label>
              <label>
                <FieldLabel required>原因</FieldLabel>
                <input onChange={(event) => setApprovalForm((current) => ({ ...current, reason: event.target.value }))} required value={approvalForm.reason} />
              </label>
              <SchemaFeedback errors={schemaErrors.approval} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setApprovalDecisionModal(null)} type="button">
                  <span>取消</span>
                </button>
                <button className={approvalDecisionModal.decision === "approved" ? "inlineActionButton" : "inlineActionButton danger"} disabled={approvalActionState[approvalDecisionModal.approvalID]?.status === "running"} type="submit">
                  {approvalDecisionModal.decision === "approved" ? <CheckCircle2 size={13} /> : <AlertTriangle size={13} />}
                  <span>{approvalDecisionModal.decision === "approved" ? "批准" : "拒绝"}</span>
                </button>
                <ActionFeedback state={approvalActionState[approvalDecisionModal.approvalID]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {approvalCreateModalOpen ? (
          <ModalShell title="发起审批" onClose={() => setApprovalCreateModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitApprovalCreate}>
              <label>
                <FieldLabel required>目标类型</FieldLabel>
                <input onChange={(event) => setApprovalCreateForm((current) => ({ ...current, targetType: event.target.value }))} required value={approvalCreateForm.targetType} />
              </label>
              <label>
                <FieldLabel required>目标 ID</FieldLabel>
                <input onChange={(event) => setApprovalCreateForm((current) => ({ ...current, targetID: event.target.value }))} required value={approvalCreateForm.targetID} />
              </label>
              <label>
                <FieldLabel required>动作</FieldLabel>
                <input onChange={(event) => setApprovalCreateForm((current) => ({ ...current, action: event.target.value }))} required value={approvalCreateForm.action} />
              </label>
              <label>
                <span>风险等级</span>
                <select onChange={(event) => setApprovalCreateForm((current) => ({ ...current, riskLevel: event.target.value }))} value={approvalCreateForm.riskLevel}>
                  <option value="low">low</option>
                  <option value="medium">medium</option>
                  <option value="high">high</option>
                  <option value="critical">critical</option>
                </select>
              </label>
              <label>
                <FieldLabel required>请求人</FieldLabel>
                <input onChange={(event) => setApprovalCreateForm((current) => ({ ...current, requestedBy: event.target.value }))} required value={approvalCreateForm.requestedBy} />
              </label>
              <label className="formWide">
                <FieldLabel required>原因</FieldLabel>
                <textarea onChange={(event) => setApprovalCreateForm((current) => ({ ...current, reason: event.target.value }))} required value={approvalCreateForm.reason} />
              </label>
              <label className="formWide">
                <span>Metadata</span>
                <textarea
                  onChange={(event) => setApprovalCreateForm((current) => ({ ...current, metadata: event.target.value }))}
                  placeholder="key=value，每行一条"
                  value={approvalCreateForm.metadata}
                />
              </label>
              <SchemaFeedback errors={schemaErrors.approvalCreate} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setApprovalCreateModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={approvalActionState.create?.status === "running"} type="submit">
                  <Lock size={13} />
                  <span>{approvalActionState.create?.status === "running" ? "创建中" : "创建审批"}</span>
                </button>
                <ActionFeedback state={approvalActionState.create} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {repairReviewModal ? (
          <ModalShell title={repairReviewModal.decision === "approved" ? "批准修复候选" : "拒绝修复候选"} onClose={() => setRepairReviewModal(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitRepairReview}>
              <div className="modalContext formWide">
                <span>候选</span>
                <strong>{compactID(repairReviewModal.candidateID)}</strong>
              </div>
              <label>
                <FieldLabel required>复核人</FieldLabel>
                <input
                  onChange={(event) => setRepairReviewForm((current) => ({ ...current, reviewerID: event.target.value }))}
                  required
                  value={repairReviewForm.reviewerID}
                />
              </label>
              <label>
                <FieldLabel required>原因</FieldLabel>
                <input
                  onChange={(event) => setRepairReviewForm((current) => ({ ...current, reason: event.target.value }))}
                  required
                  value={repairReviewForm.reason}
                />
              </label>
              <SchemaFeedback errors={schemaErrors.repairReview} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setRepairReviewModal(null)} type="button">
                  <span>取消</span>
                </button>
                <button className={repairReviewModal.decision === "approved" ? "inlineActionButton" : "inlineActionButton danger"} disabled={repairActionState[repairReviewModal.candidateID]?.status === "running"} type="submit">
                  {repairReviewModal.decision === "approved" ? <CheckCircle2 size={13} /> : <AlertTriangle size={13} />}
                  <span>{repairReviewModal.decision === "approved" ? "批准" : "拒绝"}</span>
                </button>
                <ActionFeedback state={repairActionState[repairReviewModal.candidateID]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {deploymentRiskReviewModal ? (
          <ModalShell title={deploymentRiskReviewModal.decision === "approved" ? "批准部署风险" : "拒绝部署风险"} onClose={() => setDeploymentRiskReviewModal(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitDeploymentRiskReview}>
              <div className="modalContext formWide">
                <span>风险交接</span>
                <strong>{compactID(deploymentRiskReviewModal.handoffID)}</strong>
              </div>
              <label>
                <FieldLabel required>复核人</FieldLabel>
                <input
                  onChange={(event) => setDeploymentRiskReviewForm((current) => ({ ...current, reviewerID: event.target.value }))}
                  required
                  value={deploymentRiskReviewForm.reviewerID}
                />
              </label>
              <label>
                <FieldLabel required>原因</FieldLabel>
                <input
                  onChange={(event) => setDeploymentRiskReviewForm((current) => ({ ...current, reason: event.target.value }))}
                  required
                  value={deploymentRiskReviewForm.reason}
                />
              </label>
              <label>
                <span>下一步</span>
                <select onChange={(event) => setDeploymentRiskReviewForm((current) => ({ ...current, nextStep: event.target.value }))} value={deploymentRiskReviewForm.nextStep}>
                  <option value="repair_attempt">repair_attempt</option>
                  <option value="manual_handoff">manual_handoff</option>
                  <option value="retry_rehearsal">retry_rehearsal</option>
                  <option value="">无</option>
                </select>
              </label>
              <SchemaFeedback errors={schemaErrors.deploymentRiskReview} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setDeploymentRiskReviewModal(null)} type="button">
                  <span>取消</span>
                </button>
                <button className={deploymentRiskReviewModal.decision === "approved" ? "inlineActionButton" : "inlineActionButton danger"} disabled={deploymentActionState.status === "running"} type="submit">
                  {deploymentRiskReviewModal.decision === "approved" ? <CheckCircle2 size={13} /> : <AlertTriangle size={13} />}
                  <span>{deploymentRiskReviewModal.decision === "approved" ? "批准" : "拒绝"}</span>
                </button>
                <ActionFeedback state={deploymentActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {resourceActionModal ? (
          <ModalShell title={resourceActionModal.action === "renew" ? "续期服务器" : "退役服务器"} onClose={() => setResourceActionModal(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitResourceAction}>
              <div className="modalContext formWide">
                <span>资源</span>
                <strong>{resourceActionModal.resourceID}</strong>
              </div>
              <label>
                <FieldLabel required>执行人</FieldLabel>
                <input onChange={(event) => setResourceForm((current) => ({ ...current, actorID: event.target.value }))} required value={resourceForm.actorID} />
              </label>
              {resourceActionModal.action === "renew" ? (
                <label>
                  <FieldLabel required>到期日</FieldLabel>
                  <input
                    onChange={(event) => setResourceForm((current) => ({ ...current, expiresAt: event.target.value }))}
                    required
                    type="date"
                    value={resourceForm.expiresAt}
                  />
                </label>
              ) : null}
              <label>
                <FieldLabel required>原因</FieldLabel>
                <input onChange={(event) => setResourceForm((current) => ({ ...current, reason: event.target.value }))} required value={resourceForm.reason} />
              </label>
              <SchemaFeedback errors={schemaErrors.resource} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setResourceActionModal(null)} type="button">
                  <span>取消</span>
                </button>
                <button className={resourceActionModal.action === "renew" ? "inlineActionButton" : "inlineActionButton danger"} disabled={resourceActionState[resourceActionModal.resourceID]?.status === "running"} type="submit">
                  {resourceActionModal.action === "renew" ? <RefreshCw size={13} /> : <Wrench size={13} />}
                  <span>{resourceActionModal.action === "renew" ? "续期" : "退役"}</span>
                </button>
                <ActionFeedback state={resourceActionState[resourceActionModal.resourceID]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {resourceScanModal ? (
          <ModalShell
            title={resourceScanModal === "maintenance" ? "维护扫描" : resourceScanModal === "lifecycle" ? "生命周期扫描" : "健康扫描"}
            onClose={() => setResourceScanModal(null)}
          >
            <form className="controlForm modalGridForm modalForm" onSubmit={submitResourceScan}>
              {resourceScanModal === "health" ? (
                <>
                  <label>
                    <FieldLabel required>环境</FieldLabel>
                    <select onChange={(event) => setResourceScanForm((current) => ({ ...current, environment: event.target.value }))} required value={resourceScanForm.environment}>
                      <option value="test_dev">test_dev</option>
                      <option value="staging">staging</option>
                      <option value="production">production</option>
                    </select>
                  </label>
                  <label>
                    <span>Resource IDs</span>
                    <input
                      onChange={(event) => setResourceScanForm((current) => ({ ...current, resourceIDs: event.target.value }))}
                      placeholder="留空扫描该环境全部资源"
                      value={resourceScanForm.resourceIDs}
                    />
                  </label>
                  <label className="checkboxLine">
                    <input checked={resourceScanForm.approved} onChange={(event) => setResourceScanForm((current) => ({ ...current, approved: event.target.checked }))} type="checkbox" />
                    <span>允许生产扫描</span>
                  </label>
                </>
              ) : (
                <div className="modalContext formWide">
                  <span>扫描类型</span>
                  <strong>{resourceScanModal === "maintenance" ? "生成维护记录" : "生成生命周期告警"}</strong>
                </div>
              )}
              <SchemaFeedback errors={schemaErrors.resourceScan} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setResourceScanModal(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={resourceActionState[`scan:${resourceScanModal}`]?.status === "running"} type="submit">
                  <RefreshCw size={13} />
                  <span>{resourceActionState[`scan:${resourceScanModal}`]?.status === "running" ? "扫描中" : "开始扫描"}</span>
                </button>
                <ActionFeedback state={resourceActionState[`scan:${resourceScanModal}`]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {resourceDisableModalID ? (
          <ModalShell title="禁用服务器资源" onClose={() => setResourceDisableModalID(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitResourceDisable}>
              <div className="modalContext formWide">
                <span>资源</span>
                <strong>{resourceDisableModalID}</strong>
              </div>
              <SchemaFeedback errors={schemaErrors.resourceDisable} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setResourceDisableModalID(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton danger" disabled={resourceActionState[`${resourceDisableModalID}:disable`]?.status === "running"} type="submit">
                  <X size={13} />
                  <span>禁用</span>
                </button>
                <ActionFeedback state={resourceActionState[`${resourceDisableModalID}:disable`]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {batchPlanModalOpen ? (
          <ModalShell title="创建批量计划" onClose={() => setBatchPlanModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitBatchPlan}>
              <label className="formWide">
                <FieldLabel required>Epic ID</FieldLabel>
                <input onChange={(event) => setBatchPlanForm((current) => ({ ...current, epicID: event.target.value }))} required value={batchPlanForm.epicID} />
              </label>
              <label>
                <FieldLabel required>模式</FieldLabel>
                <select onChange={(event) => setBatchPlanForm((current) => ({ ...current, mode: event.target.value }))} required value={batchPlanForm.mode}>
                  <option value="dry_run">dry_run</option>
                  <option value="dispatch">dispatch</option>
                  <option value="parallel">parallel</option>
                </select>
              </label>
              <label>
                <span>最大并行</span>
                <input inputMode="numeric" onChange={(event) => setBatchPlanForm((current) => ({ ...current, maxParallel: event.target.value }))} value={batchPlanForm.maxParallel} />
              </label>
              <label>
                <FieldLabel required>请求人</FieldLabel>
                <input onChange={(event) => setBatchPlanForm((current) => ({ ...current, requestedBy: event.target.value }))} required value={batchPlanForm.requestedBy} />
              </label>
              <SchemaFeedback errors={schemaErrors.batchPlan} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setBatchPlanModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={batchActionState.plan?.status === "running"} type="submit">
                  <Layers3 size={13} />
                  <span>{batchActionState.plan?.status === "running" ? "创建中" : "创建计划"}</span>
                </button>
                <ActionFeedback state={batchActionState.plan} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {providerModalOpen ? (
          <ModalShell title="新增 Provider" onClose={() => setProviderModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={upsertProvider}>
              <label>
                <FieldLabel required>Provider ID</FieldLabel>
                <input onChange={(event) => setProviderForm((current) => ({ ...current, id: event.target.value }))} required value={providerForm.id} />
              </label>
              <label>
                <FieldLabel required>名称</FieldLabel>
                <input onChange={(event) => setProviderForm((current) => ({ ...current, name: event.target.value }))} required value={providerForm.name} />
              </label>
              <label>
                <FieldLabel required>Vendor</FieldLabel>
                <input onChange={(event) => setProviderForm((current) => ({ ...current, vendor: event.target.value }))} required value={providerForm.vendor} />
              </label>
              <label>
                <FieldLabel required>API Type</FieldLabel>
                <input onChange={(event) => setProviderForm((current) => ({ ...current, apiType: event.target.value }))} required value={providerForm.apiType} />
              </label>
              <label>
                <span>Base URL</span>
                <input onChange={(event) => setProviderForm((current) => ({ ...current, baseURL: event.target.value }))} value={providerForm.baseURL} />
              </label>
              <label>
                <span>Auth Ref</span>
                <input onChange={(event) => setProviderForm((current) => ({ ...current, authRef: event.target.value }))} placeholder="env:OPENAI_API_KEY" value={providerForm.authRef} />
              </label>
              <label>
                <span>Runtime ID</span>
                <input onChange={(event) => setProviderForm((current) => ({ ...current, runtimeID: event.target.value }))} value={providerForm.runtimeID} />
              </label>
              <label>
                <span>Model</span>
                <input onChange={(event) => setProviderForm((current) => ({ ...current, model: event.target.value }))} value={providerForm.model} />
              </label>
              <label className="formWide">
                <span>Use Cases</span>
                <input onChange={(event) => setProviderForm((current) => ({ ...current, useCases: event.target.value }))} value={providerForm.useCases} />
              </label>
              <label className="checkboxLine">
                <input checked={providerForm.enabled} onChange={(event) => setProviderForm((current) => ({ ...current, enabled: event.target.checked }))} type="checkbox" />
                <span>启用</span>
              </label>
              <label className="checkboxLine">
                <input checked={providerForm.nativeRuntime} onChange={(event) => setProviderForm((current) => ({ ...current, nativeRuntime: event.target.checked }))} type="checkbox" />
                <span>Native Runtime</span>
              </label>
              <label className="checkboxLine">
                <input
                  checked={providerForm.allowSensitiveCode}
                  onChange={(event) => setProviderForm((current) => ({ ...current, allowSensitiveCode: event.target.checked }))}
                  type="checkbox"
                />
                <span>允许敏感代码</span>
              </label>
              <label className="checkboxLine">
                <input
                  checked={providerForm.allowProjectMemory}
                  onChange={(event) => setProviderForm((current) => ({ ...current, allowProjectMemory: event.target.checked }))}
                  type="checkbox"
                />
                <span>允许项目 Memory</span>
              </label>
              <label className="checkboxLine">
                <input
                  checked={providerForm.allowProductionContext}
                  onChange={(event) => setProviderForm((current) => ({ ...current, allowProductionContext: event.target.checked }))}
                  type="checkbox"
                />
                <span>允许生产上下文</span>
              </label>
              <SchemaFeedback errors={schemaErrors.provider} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setProviderModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={providerActionState.upsert?.status === "running"} type="submit">
                  <Sparkles size={13} />
                  <span>{providerActionState.upsert?.status === "running" ? "保存中" : "保存 Provider"}</span>
                </button>
                <ActionFeedback state={providerActionState.upsert} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {providerOpsModalOpen ? (
          <ModalShell title="刷新 Provider Ops" onClose={() => setProviderOpsModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={refreshProviderOps}>
              <label>
                <span>Provider ID</span>
                <input
                  onChange={(event) => setProviderOpsForm((current) => ({ ...current, providerID: event.target.value }))}
                  placeholder="留空刷新全部"
                  value={providerOpsForm.providerID}
                />
              </label>
              <label>
                <span>Probe Timeout MS</span>
                <input
                  inputMode="numeric"
                  onChange={(event) => setProviderOpsForm((current) => ({ ...current, probeTimeoutMS: event.target.value }))}
                  value={providerOpsForm.probeTimeoutMS}
                />
              </label>
              <label className="checkboxLine">
                <input checked={providerOpsForm.includeDisabled} onChange={(event) => setProviderOpsForm((current) => ({ ...current, includeDisabled: event.target.checked }))} type="checkbox" />
                <span>包含禁用项</span>
              </label>
              <label className="checkboxLine">
                <input checked={providerOpsForm.probe} onChange={(event) => setProviderOpsForm((current) => ({ ...current, probe: event.target.checked }))} type="checkbox" />
                <span>执行 Probe</span>
              </label>
              <label className="checkboxLine">
                <input checked={providerOpsForm.approved} onChange={(event) => setProviderOpsForm((current) => ({ ...current, approved: event.target.checked }))} type="checkbox" />
                <span>已批准 Probe</span>
              </label>
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setProviderOpsModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={providerActionState.ops?.status === "running"} type="submit">
                  <RefreshCw size={13} />
                  <span>{providerActionState.ops?.status === "running" ? "刷新中" : "刷新 Ops"}</span>
                </button>
                <ActionFeedback state={providerActionState.ops} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {providerOpsSnapshotModalID ? (
          <ModalShell title="更新 Provider Ops" onClose={() => setProviderOpsSnapshotModalID(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitProviderOpsSnapshot}>
              <div className="modalContext formWide">
                <span>Provider</span>
                <strong>{providerOpsSnapshotModalID}</strong>
              </div>
              <label>
                <FieldLabel required>健康状态</FieldLabel>
                <select onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, healthStatus: event.target.value }))} required value={providerOpsSnapshotForm.healthStatus}>
                  <option value="ok">ok</option>
                  <option value="degraded">degraded</option>
                  <option value="blocked">blocked</option>
                  <option value="unknown">unknown</option>
                </select>
              </label>
              <label>
                <span>健康原因</span>
                <input onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, healthReason: event.target.value }))} value={providerOpsSnapshotForm.healthReason} />
              </label>
              <label>
                <FieldLabel required>配额状态</FieldLabel>
                <select onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, quotaStatus: event.target.value }))} required value={providerOpsSnapshotForm.quotaStatus}>
                  <option value="ok">ok</option>
                  <option value="near_limit">near_limit</option>
                  <option value="exhausted">exhausted</option>
                  <option value="unknown">unknown</option>
                </select>
              </label>
              <label>
                <FieldLabel required>成本状态</FieldLabel>
                <select onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, costStatus: event.target.value }))} required value={providerOpsSnapshotForm.costStatus}>
                  <option value="ok">ok</option>
                  <option value="watch">watch</option>
                  <option value="over_budget">over_budget</option>
                  <option value="unknown">unknown</option>
                </select>
              </label>
              <label>
                <span>Token 上限</span>
                <input inputMode="numeric" onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, limitTokens: event.target.value }))} value={providerOpsSnapshotForm.limitTokens} />
              </label>
              <label>
                <span>已用 Token</span>
                <input inputMode="numeric" onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, usedTokens: event.target.value }))} value={providerOpsSnapshotForm.usedTokens} />
              </label>
              <label>
                <span>剩余 Token</span>
                <input inputMode="numeric" onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, remainingTokens: event.target.value }))} value={providerOpsSnapshotForm.remainingTokens} />
              </label>
              <label>
                <span>Usage Window</span>
                <input onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, usageWindow: event.target.value }))} value={providerOpsSnapshotForm.usageWindow} />
              </label>
              <label>
                <span>请求数</span>
                <input inputMode="numeric" onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, requests: event.target.value }))} value={providerOpsSnapshotForm.requests} />
              </label>
              <label>
                <span>输入 Token</span>
                <input inputMode="numeric" onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, inputTokens: event.target.value }))} value={providerOpsSnapshotForm.inputTokens} />
              </label>
              <label>
                <span>输出 Token</span>
                <input inputMode="numeric" onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, outputTokens: event.target.value }))} value={providerOpsSnapshotForm.outputTokens} />
              </label>
              <label>
                <span>总 Token</span>
                <input inputMode="numeric" onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, totalTokens: event.target.value }))} value={providerOpsSnapshotForm.totalTokens} />
              </label>
              <label>
                <span>预估金额</span>
                <input inputMode="decimal" onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, estimatedAmount: event.target.value }))} value={providerOpsSnapshotForm.estimatedAmount} />
              </label>
              <label>
                <span>预算金额</span>
                <input inputMode="decimal" onChange={(event) => setProviderOpsSnapshotForm((current) => ({ ...current, budgetAmount: event.target.value }))} value={providerOpsSnapshotForm.budgetAmount} />
              </label>
              <SchemaFeedback errors={schemaErrors.providerOpsSnapshot} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setProviderOpsSnapshotModalID(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={providerActionState[`${providerOpsSnapshotModalID}:ops`]?.status === "running"} type="submit">
                  <RefreshCw size={13} />
                  <span>{providerActionState[`${providerOpsSnapshotModalID}:ops`]?.status === "running" ? "保存中" : "保存 Ops"}</span>
                </button>
                <ActionFeedback state={providerActionState[`${providerOpsSnapshotModalID}:ops`]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {providerDisableModalID ? (
          <ModalShell title="禁用 Provider" onClose={() => setProviderDisableModalID(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitProviderDisable}>
              <div className="modalContext formWide">
                <span>Provider</span>
                <strong>{providerDisableModalID}</strong>
              </div>
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setProviderDisableModalID(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton danger" disabled={providerActionState[providerDisableModalID]?.status === "running"} type="submit">
                  <X size={13} />
                  <span>禁用</span>
                </button>
                <ActionFeedback state={providerActionState[providerDisableModalID]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {releaseProviderModalOpen ? (
          <ModalShell title="Release Provider" onClose={() => setReleaseProviderModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={(event) => event.preventDefault()}>
              <label>
                <FieldLabel required>Release ID</FieldLabel>
                <input
                  onChange={(event) => setReleaseProviderForm((current) => ({ ...current, releaseID: event.target.value }))}
                  required
                  value={releaseProviderForm.releaseID}
                />
              </label>
              <label className="checkboxLine">
                <input
                  checked={releaseProviderForm.approved}
                  onChange={(event) => setReleaseProviderForm((current) => ({ ...current, approved: event.target.checked }))}
                  type="checkbox"
                />
                <span>已批准发布</span>
              </label>
              <label>
                <FieldLabel required={releaseProviderForm.approved}>Approval ID</FieldLabel>
                <input
                  onChange={(event) => setReleaseProviderForm((current) => ({ ...current, approvalID: event.target.value }))}
                  required={releaseProviderForm.approved}
                  value={releaseProviderForm.approvalID}
                />
              </label>
              <SchemaFeedback errors={schemaErrors.releaseProvider} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setReleaseProviderModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={releaseProviderActionState.status === "running"} onClick={() => void runReleaseProviderAction("preview")} type="button">
                  <Search size={13} />
                  <span>Provider 预览</span>
                </button>
                <button className="inlineActionButton" disabled={releaseProviderActionState.status === "running"} onClick={() => void runReleaseProviderAction("publish")} type="button">
                  <Rocket size={13} />
                  <span>Provider 发布</span>
                </button>
                <ActionFeedback state={releaseProviderActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {gitCreateModalPlanID ? (
          <ModalShell title="创建 PR/MR" onClose={() => setGitCreateModalPlanID(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitGitCreate}>
              <div className="modalContext formWide">
                <span>计划</span>
                <strong>{compactID(gitCreateModalPlanID)}</strong>
              </div>
              <label className="checkboxLine">
                <input checked={gitCreateApproved} onChange={(event) => setGitCreateApproved(event.target.checked)} type="checkbox" />
                <span>已批准创建</span>
              </label>
              <label>
                <FieldLabel required={gitCreateApproved}>Approval ID</FieldLabel>
                <input onChange={(event) => setGitCreateApprovalID(event.target.value)} required={gitCreateApproved} value={gitCreateApprovalID} />
              </label>
              <SchemaFeedback errors={schemaErrors.gitCreate} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setGitCreateModalPlanID(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={gitActionState[gitCreateModalPlanID]?.status === "running"} type="submit">
                  <GitBranch size={13} />
                  <span>{gitActionState[gitCreateModalPlanID]?.status === "running" ? "创建中" : "创建 PR/MR"}</span>
                </button>
                <ActionFeedback state={gitActionState[gitCreateModalPlanID]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {visualPlanModalOpen ? (
          <ModalShell title="创建视觉图计划" onClose={() => setVisualPlanModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitVisualPlan}>
              <label>
                <FieldLabel required>图类型</FieldLabel>
                <select onChange={(event) => setVisualPlanForm((current) => ({ ...current, diagramType: event.target.value }))} required value={visualPlanForm.diagramType}>
                  <option value="architecture">architecture</option>
                  <option value="multi_agent">multi_agent</option>
                  <option value="deployment">deployment</option>
                  <option value="data_flow">data_flow</option>
                </select>
              </label>
              <label>
                <FieldLabel required>标题</FieldLabel>
                <input onChange={(event) => setVisualPlanForm((current) => ({ ...current, title: event.target.value }))} required value={visualPlanForm.title} />
              </label>
              <label>
                <span>Size</span>
                <select onChange={(event) => setVisualPlanForm((current) => ({ ...current, size: event.target.value }))} value={visualPlanForm.size}>
                  <option value="3072x2048">3072x2048</option>
                  <option value="2048x2048">2048x2048</option>
                  <option value="1536x1024">1536x1024</option>
                </select>
              </label>
              <label className="formWide">
                <span>Scope</span>
                <textarea onChange={(event) => setVisualPlanForm((current) => ({ ...current, scope: event.target.value }))} value={visualPlanForm.scope} />
              </label>
              <SchemaFeedback errors={schemaErrors.visualPlan} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setVisualPlanModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={visualActionState.plan?.status === "running"} type="submit">
                  <Sparkles size={13} />
                  <span>{visualActionState.plan?.status === "running" ? "创建中" : "创建计划"}</span>
                </button>
                <ActionFeedback state={visualActionState.plan} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {writePipelineModal === "remoteRehearsal" ? (
          <ModalShell title="创建远程执行演练" onClose={() => setWritePipelineModal(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={createRemoteExecutionRehearsals}>
              <label>
                <span>Admission ID</span>
                <input onChange={(event) => setRemoteRehearsalForm((current) => ({ ...current, admissionID: event.target.value }))} value={remoteRehearsalForm.admissionID} />
              </label>
              <label>
                <span>Execution ID</span>
                <input onChange={(event) => setRemoteRehearsalForm((current) => ({ ...current, executionID: event.target.value }))} value={remoteRehearsalForm.executionID} />
              </label>
              <label>
                <span>Provider</span>
                <input onChange={(event) => setRemoteRehearsalForm((current) => ({ ...current, provider: event.target.value }))} value={remoteRehearsalForm.provider} />
              </label>
              <label>
                <span>环境</span>
                <input onChange={(event) => setRemoteRehearsalForm((current) => ({ ...current, environment: event.target.value }))} value={remoteRehearsalForm.environment} />
              </label>
              <label>
                <span>Status</span>
                <input onChange={(event) => setRemoteRehearsalForm((current) => ({ ...current, status: event.target.value }))} value={remoteRehearsalForm.status} />
              </label>
              <label>
                <span>Decision</span>
                <input onChange={(event) => setRemoteRehearsalForm((current) => ({ ...current, decision: event.target.value }))} value={remoteRehearsalForm.decision} />
              </label>
              <label>
                <span>Limit</span>
                <input inputMode="numeric" onChange={(event) => setRemoteRehearsalForm((current) => ({ ...current, limit: event.target.value }))} value={remoteRehearsalForm.limit} />
              </label>
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setWritePipelineModal(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={adapterActionState.status === "running"} type="submit">
                  <Play size={13} />
                  <span>{adapterActionState.status === "running" ? "创建中" : "创建演练"}</span>
                </button>
                <ActionFeedback state={adapterActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {writePipelineModal === "reviewPacket" ? (
          <ModalShell title="生成写入复核包" onClose={() => setWritePipelineModal(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={createWriteReviewPackets}>
              <label>
                <span>Admission ID</span>
                <input onChange={(event) => setWriteReviewPacketForm((current) => ({ ...current, admissionID: event.target.value }))} value={writeReviewPacketForm.admissionID} />
              </label>
              <label>
                <FieldLabel required>Operation Type</FieldLabel>
                <input onChange={(event) => setWriteReviewPacketForm((current) => ({ ...current, operationType: event.target.value }))} required value={writeReviewPacketForm.operationType} />
              </label>
              <label>
                <span>Operation ID</span>
                <input onChange={(event) => setWriteReviewPacketForm((current) => ({ ...current, operationID: event.target.value }))} value={writeReviewPacketForm.operationID} />
              </label>
              <label>
                <span>Provider</span>
                <input onChange={(event) => setWriteReviewPacketForm((current) => ({ ...current, provider: event.target.value }))} value={writeReviewPacketForm.provider} />
              </label>
              <label>
                <span>环境</span>
                <input onChange={(event) => setWriteReviewPacketForm((current) => ({ ...current, environment: event.target.value }))} value={writeReviewPacketForm.environment} />
              </label>
              <label>
                <span>Status</span>
                <input onChange={(event) => setWriteReviewPacketForm((current) => ({ ...current, status: event.target.value }))} value={writeReviewPacketForm.status} />
              </label>
              <label>
                <span>Decision</span>
                <input onChange={(event) => setWriteReviewPacketForm((current) => ({ ...current, decision: event.target.value }))} value={writeReviewPacketForm.decision} />
              </label>
              <label>
                <span>Limit</span>
                <input inputMode="numeric" onChange={(event) => setWriteReviewPacketForm((current) => ({ ...current, limit: event.target.value }))} value={writeReviewPacketForm.limit} />
              </label>
              <SchemaFeedback errors={schemaErrors.writeReviewPacket} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setWritePipelineModal(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={adapterActionState.status === "running"} type="submit">
                  <ScrollText size={13} />
                  <span>{adapterActionState.status === "running" ? "生成中" : "生成复核包"}</span>
                </button>
                <ActionFeedback state={adapterActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {writePipelineModal === "executionPlan" ? (
          <ModalShell title="生成写入执行计划" onClose={() => setWritePipelineModal(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={createWriteExecutionPlans}>
              <label className="formWide">
                <FieldLabel required>Review Packet ID</FieldLabel>
                <input onChange={(event) => setWriteExecutionPlanForm((current) => ({ ...current, reviewPacketID: event.target.value }))} required value={writeExecutionPlanForm.reviewPacketID} />
              </label>
              <label>
                <FieldLabel required>模式</FieldLabel>
                <select onChange={(event) => setWriteExecutionPlanForm((current) => ({ ...current, mode: event.target.value }))} required value={writeExecutionPlanForm.mode}>
                  <option value="preview">preview</option>
                  <option value="dry_run">dry_run</option>
                  <option value="apply">apply</option>
                </select>
              </label>
              <label>
                <span>Approval ID</span>
                <input onChange={(event) => setWriteExecutionPlanForm((current) => ({ ...current, approvalID: event.target.value }))} value={writeExecutionPlanForm.approvalID} />
              </label>
              <label>
                <span>请求人</span>
                <input onChange={(event) => setWriteExecutionPlanForm((current) => ({ ...current, requestedBy: event.target.value }))} value={writeExecutionPlanForm.requestedBy} />
              </label>
              <label>
                <span>Status</span>
                <input onChange={(event) => setWriteExecutionPlanForm((current) => ({ ...current, status: event.target.value }))} value={writeExecutionPlanForm.status} />
              </label>
              <label>
                <span>Decision</span>
                <input onChange={(event) => setWriteExecutionPlanForm((current) => ({ ...current, decision: event.target.value }))} value={writeExecutionPlanForm.decision} />
              </label>
              <label>
                <span>Limit</span>
                <input inputMode="numeric" onChange={(event) => setWriteExecutionPlanForm((current) => ({ ...current, limit: event.target.value }))} value={writeExecutionPlanForm.limit} />
              </label>
              <SchemaFeedback errors={schemaErrors.writeExecutionPlan} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setWritePipelineModal(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={adapterActionState.status === "running"} type="submit">
                  <Play size={13} />
                  <span>{adapterActionState.status === "running" ? "生成中" : "生成计划"}</span>
                </button>
                <ActionFeedback state={adapterActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {writePipelineModal === "adapterExecution" ? (
          <ModalShell title="创建写入 Adapter 执行" onClose={() => setWritePipelineModal(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={createWriteAdapterExecutions}>
              <label className="formWide">
                <FieldLabel required>Execution Plan ID</FieldLabel>
                <input
                  onChange={(event) => setWriteAdapterExecutionForm((current) => ({ ...current, executionPlanID: event.target.value }))}
                  required
                  value={writeAdapterExecutionForm.executionPlanID}
                />
              </label>
              <label>
                <FieldLabel required>Adapter ID</FieldLabel>
                <input onChange={(event) => setWriteAdapterExecutionForm((current) => ({ ...current, adapterID: event.target.value }))} required value={writeAdapterExecutionForm.adapterID} />
              </label>
              <label>
                <span>模式</span>
                <select onChange={(event) => setWriteAdapterExecutionForm((current) => ({ ...current, mode: event.target.value }))} value={writeAdapterExecutionForm.mode}>
                  <option value="preview">preview</option>
                  <option value="dry_run">dry_run</option>
                  <option value="apply">apply</option>
                </select>
              </label>
              <label>
                <span>Status</span>
                <input onChange={(event) => setWriteAdapterExecutionForm((current) => ({ ...current, status: event.target.value }))} value={writeAdapterExecutionForm.status} />
              </label>
              <label>
                <span>Decision</span>
                <input onChange={(event) => setWriteAdapterExecutionForm((current) => ({ ...current, decision: event.target.value }))} value={writeAdapterExecutionForm.decision} />
              </label>
              <label>
                <span>Limit</span>
                <input inputMode="numeric" onChange={(event) => setWriteAdapterExecutionForm((current) => ({ ...current, limit: event.target.value }))} value={writeAdapterExecutionForm.limit} />
              </label>
              <SchemaFeedback errors={schemaErrors.writeAdapterExecution} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setWritePipelineModal(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={adapterActionState.status === "running"} type="submit">
                  <Wrench size={13} />
                  <span>{adapterActionState.status === "running" ? "创建中" : "创建执行"}</span>
                </button>
                <ActionFeedback state={adapterActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {controlQueueModalOpen ? (
          <ModalShell title="控制队列入队" onClose={() => setControlQueueModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitControlQueue}>
              <label>
                <FieldLabel required>触发器</FieldLabel>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, trigger: event.target.value }))} required value={controlQueueForm.trigger} />
              </label>
              <label>
                <FieldLabel required>请求人</FieldLabel>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, requestedBy: event.target.value }))} required value={controlQueueForm.requestedBy} />
              </label>
              <label className="formWide">
                <FieldLabel required>步骤</FieldLabel>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, steps: event.target.value }))} required value={controlQueueForm.steps} />
              </label>
              <label>
                <span>环境</span>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, environment: event.target.value }))} value={controlQueueForm.environment} />
              </label>
              <label>
                <span>Retry Budget</span>
                <input inputMode="numeric" onChange={(event) => setControlQueueForm((current) => ({ ...current, retryBudget: event.target.value }))} value={controlQueueForm.retryBudget} />
              </label>
              <label>
                <span>Priority</span>
                <input inputMode="numeric" onChange={(event) => setControlQueueForm((current) => ({ ...current, priority: event.target.value }))} value={controlQueueForm.priority} />
              </label>
              <label>
                <span>Maintenance Window</span>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, maintenanceWindow: event.target.value }))} value={controlQueueForm.maintenanceWindow} />
              </label>
              <label>
                <span>Due At</span>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, dueAt: event.target.value }))} placeholder="RFC3339，可留空" value={controlQueueForm.dueAt} />
              </label>
              <label>
                <span>Resource IDs</span>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, resourceIDs: event.target.value }))} value={controlQueueForm.resourceIDs} />
              </label>
              <label>
                <span>Deployment Execution ID</span>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, deploymentExecutionID: event.target.value }))} value={controlQueueForm.deploymentExecutionID} />
              </label>
              <label>
                <span>Admission ID</span>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, admissionID: event.target.value }))} value={controlQueueForm.admissionID} />
              </label>
              <label>
                <span>Remote Rehearsal ID</span>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, remoteRehearsalID: event.target.value }))} value={controlQueueForm.remoteRehearsalID} />
              </label>
              <label>
                <span>Review Packet ID</span>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, reviewPacketID: event.target.value }))} value={controlQueueForm.reviewPacketID} />
              </label>
              <label>
                <span>Adapter Recovery ID</span>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, adapterRecoveryID: event.target.value }))} value={controlQueueForm.adapterRecoveryID} />
              </label>
              <label className="formWide">
                <span>Idempotency Key</span>
                <input onChange={(event) => setControlQueueForm((current) => ({ ...current, idempotencyKey: event.target.value }))} value={controlQueueForm.idempotencyKey} />
              </label>
              <SchemaFeedback errors={schemaErrors.controlQueue} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setControlQueueModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={adapterActionState.status === "running"} type="submit">
                  <CircleDotDashed size={13} />
                  <span>{adapterActionState.status === "running" ? "入队中" : "入队"}</span>
                </button>
                <ActionFeedback state={adapterActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {controlQueueRunModalOpen ? (
          <ModalShell title="消费控制队列" onClose={() => setControlQueueRunModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitControlQueueRun}>
              <label>
                <span>Status</span>
                <select onChange={(event) => setControlQueueRunForm((current) => ({ ...current, status: event.target.value }))} value={controlQueueRunForm.status}>
                  <option value="queued">queued</option>
                  <option value="pending">pending</option>
                  <option value="all">all</option>
                </select>
              </label>
              <label>
                <span>环境</span>
                <input onChange={(event) => setControlQueueRunForm((current) => ({ ...current, environment: event.target.value }))} placeholder="留空全部" value={controlQueueRunForm.environment} />
              </label>
              <label>
                <span>Max Items</span>
                <input inputMode="numeric" onChange={(event) => setControlQueueRunForm((current) => ({ ...current, maxItems: event.target.value }))} value={controlQueueRunForm.maxItems} />
              </label>
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setControlQueueRunModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={adapterActionState.status === "running"} type="submit">
                  <RefreshCw size={13} />
                  <span>{adapterActionState.status === "running" ? "运行中" : "消费队列"}</span>
                </button>
                <ActionFeedback state={adapterActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {skillModalOpen ? (
          <ModalShell title="新增 Skill" onClose={() => setSkillModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={upsertSkill}>
              <label>
                <FieldLabel required>Skill ID</FieldLabel>
                <input onChange={(event) => setSkillForm((current) => ({ ...current, id: event.target.value }))} required value={skillForm.id} />
              </label>
              <label>
                <FieldLabel required>名称</FieldLabel>
                <input onChange={(event) => setSkillForm((current) => ({ ...current, name: event.target.value }))} required value={skillForm.name} />
              </label>
              <label>
                <FieldLabel required>来源</FieldLabel>
                <input onChange={(event) => setSkillForm((current) => ({ ...current, source: event.target.value }))} required value={skillForm.source} />
              </label>
              <label>
                <FieldLabel required>风险等级</FieldLabel>
                <select onChange={(event) => setSkillForm((current) => ({ ...current, riskLevel: event.target.value }))} required value={skillForm.riskLevel}>
                  <option value="low">low</option>
                  <option value="medium">medium</option>
                  <option value="high">high</option>
                </select>
              </label>
              <label>
                <span>版本</span>
                <input onChange={(event) => setSkillForm((current) => ({ ...current, version: event.target.value }))} value={skillForm.version} />
              </label>
              <label>
                <span>Auth Ref</span>
                <input onChange={(event) => setSkillForm((current) => ({ ...current, authRef: event.target.value }))} value={skillForm.authRef} />
              </label>
              <label className="formWide">
                <span>描述</span>
                <textarea onChange={(event) => setSkillForm((current) => ({ ...current, description: event.target.value }))} value={skillForm.description} />
              </label>
              <label>
                <span>兼容角色</span>
                <input onChange={(event) => setSkillForm((current) => ({ ...current, compatibleRoles: event.target.value }))} value={skillForm.compatibleRoles} />
              </label>
              <label>
                <span>Tags</span>
                <input onChange={(event) => setSkillForm((current) => ({ ...current, tags: event.target.value }))} value={skillForm.tags} />
              </label>
              <label>
                <span>Required Tools</span>
                <input onChange={(event) => setSkillForm((current) => ({ ...current, requiredTools: event.target.value }))} value={skillForm.requiredTools} />
              </label>
              <label className="checkboxLine">
                <input checked={skillForm.enabled} onChange={(event) => setSkillForm((current) => ({ ...current, enabled: event.target.checked }))} type="checkbox" />
                <span>启用</span>
              </label>
              <SchemaFeedback errors={schemaErrors.skill} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setSkillModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={skillActionState.upsert?.status === "running"} type="submit">
                  <MemoryStick size={13} />
                  <span>{skillActionState.upsert?.status === "running" ? "保存中" : "保存 Skill"}</span>
                </button>
                <ActionFeedback state={skillActionState.upsert} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {skillRecommendationModalOpen ? (
          <ModalShell title="Skill 推荐" onClose={() => setSkillRecommendationModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={recommendSkills}>
              <label>
                <span>Issue ID</span>
                <input onChange={(event) => setSkillRecommendationForm((current) => ({ ...current, issueID: event.target.value }))} value={skillRecommendationForm.issueID} />
              </label>
              <label>
                <FieldLabel required>角色</FieldLabel>
                <input onChange={(event) => setSkillRecommendationForm((current) => ({ ...current, role: event.target.value }))} required value={skillRecommendationForm.role} />
              </label>
              <label>
                <span>Task Type</span>
                <input onChange={(event) => setSkillRecommendationForm((current) => ({ ...current, taskType: event.target.value }))} value={skillRecommendationForm.taskType} />
              </label>
              <label>
                <span>风险等级</span>
                <select onChange={(event) => setSkillRecommendationForm((current) => ({ ...current, riskLevel: event.target.value }))} value={skillRecommendationForm.riskLevel}>
                  <option value="low">low</option>
                  <option value="medium">medium</option>
                  <option value="high">high</option>
                </select>
              </label>
              <label>
                <span>Limit</span>
                <input inputMode="numeric" onChange={(event) => setSkillRecommendationForm((current) => ({ ...current, limit: event.target.value }))} value={skillRecommendationForm.limit} />
              </label>
              <SchemaFeedback errors={schemaErrors.skillRecommend} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setSkillRecommendationModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={skillActionState.recommend?.status === "running"} type="submit">
                  <Search size={13} />
                  <span>{skillActionState.recommend?.status === "running" ? "推荐中" : "生成推荐"}</span>
                </button>
                <ActionFeedback state={skillActionState.recommend} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {skillBindingModalOpen ? (
          <ModalShell title="新增 Skill 绑定" onClose={() => setSkillBindingModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={upsertSkillBinding}>
              <label>
                <span>绑定 ID</span>
                <input onChange={(event) => setSkillBindingForm((current) => ({ ...current, id: event.target.value }))} placeholder="留空自动生成" value={skillBindingForm.id} />
              </label>
              <label>
                <FieldLabel required>Skill ID</FieldLabel>
                <input onChange={(event) => setSkillBindingForm((current) => ({ ...current, skillID: event.target.value }))} required value={skillBindingForm.skillID} />
              </label>
              <label>
                <FieldLabel required>目标类型</FieldLabel>
                <select onChange={(event) => setSkillBindingForm((current) => ({ ...current, targetType: event.target.value }))} required value={skillBindingForm.targetType}>
                  <option value="role">role</option>
                  <option value="issue">issue</option>
                  <option value="provider">provider</option>
                  <option value="runtime">runtime</option>
                </select>
              </label>
              <label>
                <FieldLabel required>目标 ID</FieldLabel>
                <input onChange={(event) => setSkillBindingForm((current) => ({ ...current, targetID: event.target.value }))} required value={skillBindingForm.targetID} />
              </label>
              <label>
                <span>优先级</span>
                <input inputMode="numeric" onChange={(event) => setSkillBindingForm((current) => ({ ...current, priority: event.target.value }))} value={skillBindingForm.priority} />
              </label>
              <label>
                <span>Status</span>
                <select onChange={(event) => setSkillBindingForm((current) => ({ ...current, status: event.target.value }))} value={skillBindingForm.status}>
                  <option value="active">active</option>
                  <option value="disabled">disabled</option>
                </select>
              </label>
              <label className="formWide">
                <span>Config</span>
                <textarea
                  onChange={(event) => setSkillBindingForm((current) => ({ ...current, config: event.target.value }))}
                  placeholder="每行 key=value"
                  value={skillBindingForm.config}
                />
              </label>
              <SchemaFeedback errors={schemaErrors.skillBinding} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setSkillBindingModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={skillActionState.binding?.status === "running"} type="submit">
                  <Layers3 size={13} />
                  <span>{skillActionState.binding?.status === "running" ? "保存中" : "保存绑定"}</span>
                </button>
                <ActionFeedback state={skillActionState.binding} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {skillEffectivenessModalOpen ? (
          <ModalShell title="记录 Skill 效果" onClose={() => setSkillEffectivenessModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={recordSkillEffectiveness}>
              <label>
                <span>记录 ID</span>
                <input onChange={(event) => setSkillEffectivenessForm((current) => ({ ...current, id: event.target.value }))} placeholder="留空自动生成" value={skillEffectivenessForm.id} />
              </label>
              <label>
                <FieldLabel required>Skill ID</FieldLabel>
                <input onChange={(event) => setSkillEffectivenessForm((current) => ({ ...current, skillID: event.target.value }))} required value={skillEffectivenessForm.skillID} />
              </label>
              <label>
                <span>Binding ID</span>
                <input onChange={(event) => setSkillEffectivenessForm((current) => ({ ...current, bindingID: event.target.value }))} value={skillEffectivenessForm.bindingID} />
              </label>
              <label>
                <span>Issue ID</span>
                <input onChange={(event) => setSkillEffectivenessForm((current) => ({ ...current, issueID: event.target.value }))} value={skillEffectivenessForm.issueID} />
              </label>
              <label>
                <span>Run ID</span>
                <input onChange={(event) => setSkillEffectivenessForm((current) => ({ ...current, runID: event.target.value }))} value={skillEffectivenessForm.runID} />
              </label>
              <label>
                <span>Subagent ID</span>
                <input onChange={(event) => setSkillEffectivenessForm((current) => ({ ...current, subagentID: event.target.value }))} value={skillEffectivenessForm.subagentID} />
              </label>
              <label>
                <FieldLabel required>结果</FieldLabel>
                <select onChange={(event) => setSkillEffectivenessForm((current) => ({ ...current, outcome: event.target.value }))} required value={skillEffectivenessForm.outcome}>
                  <option value="accepted">accepted</option>
                  <option value="needs_rework">needs_rework</option>
                  <option value="blocked">blocked</option>
                </select>
              </label>
              <label>
                <FieldLabel required>质量影响</FieldLabel>
                <select
                  onChange={(event) => setSkillEffectivenessForm((current) => ({ ...current, qualityImpact: event.target.value }))}
                  required
                  value={skillEffectivenessForm.qualityImpact}
                >
                  <option value="positive">positive</option>
                  <option value="neutral">neutral</option>
                  <option value="negative">negative</option>
                </select>
              </label>
              <label>
                <span>耗时秒</span>
                <input
                  inputMode="numeric"
                  onChange={(event) => setSkillEffectivenessForm((current) => ({ ...current, durationSeconds: event.target.value }))}
                  value={skillEffectivenessForm.durationSeconds}
                />
              </label>
              <label className="checkboxLine">
                <input
                  checked={skillEffectivenessForm.reworkReduced}
                  onChange={(event) => setSkillEffectivenessForm((current) => ({ ...current, reworkReduced: event.target.checked }))}
                  type="checkbox"
                />
                <span>减少返工</span>
              </label>
              <label className="formWide">
                <span>Findings</span>
                <textarea
                  onChange={(event) => setSkillEffectivenessForm((current) => ({ ...current, findings: event.target.value }))}
                  placeholder="每行一条发现"
                  value={skillEffectivenessForm.findings}
                />
              </label>
              <SchemaFeedback errors={schemaErrors.skillEffectiveness} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setSkillEffectivenessModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={skillActionState.effectiveness?.status === "running"} type="submit">
                  <CheckCircle2 size={13} />
                  <span>{skillActionState.effectiveness?.status === "running" ? "记录中" : "记录效果"}</span>
                </button>
                <ActionFeedback state={skillActionState.effectiveness} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {skillDisableModalID ? (
          <ModalShell title="禁用 Skill" onClose={() => setSkillDisableModalID(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitSkillDisable}>
              <div className="modalContext formWide">
                <span>Skill</span>
                <strong>{skillDisableModalID}</strong>
              </div>
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setSkillDisableModalID(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton danger" disabled={skillActionState[skillDisableModalID]?.status === "running"} type="submit">
                  <X size={13} />
                  <span>禁用</span>
                </button>
                <ActionFeedback state={skillActionState[skillDisableModalID]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {skillBindingDisableModalID ? (
          <ModalShell title="禁用 Skill 绑定" onClose={() => setSkillBindingDisableModalID(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitSkillBindingDisable}>
              <div className="modalContext formWide">
                <span>绑定</span>
                <strong>{skillBindingDisableModalID}</strong>
              </div>
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setSkillBindingDisableModalID(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton danger" disabled={skillActionState[skillBindingDisableModalID]?.status === "running"} type="submit">
                  <X size={13} />
                  <span>禁用</span>
                </button>
                <ActionFeedback state={skillActionState[skillBindingDisableModalID]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {sessionModalOpen ? (
          <ModalShell title="创建会话" onClose={() => setSessionModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={createSession}>
              <label>
                <FieldLabel required>用户</FieldLabel>
                <input onChange={(event) => setSessionForm((current) => ({ ...current, userID: event.target.value }))} required value={sessionForm.userID} />
              </label>
              <label>
                <FieldLabel required>显示名</FieldLabel>
                <input
                  onChange={(event) => setSessionForm((current) => ({ ...current, displayName: event.target.value }))}
                  required
                  value={sessionForm.displayName}
                />
              </label>
              <label>
                <FieldLabel required>角色</FieldLabel>
                <input onChange={(event) => setSessionForm((current) => ({ ...current, roles: event.target.value }))} required value={sessionForm.roles} />
              </label>
              <SchemaFeedback errors={schemaErrors.session} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setSessionModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={accessActionState.session?.status === "running"} type="submit">
                  <UserPlus size={13} />
                  <span>{accessActionState.session?.status === "running" ? "创建中" : "创建会话"}</span>
                </button>
                <ActionFeedback state={accessActionState.session} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {sessionRevokeModalID ? (
          <ModalShell title="撤销会话" onClose={() => setSessionRevokeModalID(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitSessionRevoke}>
              <div className="modalContext formWide">
                <span>会话</span>
                <strong>{sessionRevokeModalID}</strong>
              </div>
              <label>
                <FieldLabel required>执行人</FieldLabel>
                <input onChange={(event) => setSessionRevokeForm((current) => ({ ...current, actorID: event.target.value }))} required value={sessionRevokeForm.actorID} />
              </label>
              <label className="formWide">
                <FieldLabel required>原因</FieldLabel>
                <textarea onChange={(event) => setSessionRevokeForm((current) => ({ ...current, reason: event.target.value }))} required value={sessionRevokeForm.reason} />
              </label>
              <SchemaFeedback errors={schemaErrors.sessionRevoke} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setSessionRevokeModalID(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton danger" disabled={accessActionState[sessionRevokeModalID]?.status === "running"} type="submit">
                  <X size={13} />
                  <span>{accessActionState[sessionRevokeModalID]?.status === "running" ? "撤销中" : "撤销会话"}</span>
                </button>
                <ActionFeedback state={accessActionState[sessionRevokeModalID]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {tokenModalOpen ? (
          <ModalShell title="创建 API Token" onClose={() => setTokenModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={createAPIToken}>
              <label>
                <FieldLabel required>名称</FieldLabel>
                <input onChange={(event) => setTokenForm((current) => ({ ...current, name: event.target.value }))} required value={tokenForm.name} />
              </label>
              <label>
                <FieldLabel required>主体</FieldLabel>
                <input onChange={(event) => setTokenForm((current) => ({ ...current, actorID: event.target.value }))} required value={tokenForm.actorID} />
              </label>
              <label>
                <FieldLabel required>Scopes</FieldLabel>
                <input onChange={(event) => setTokenForm((current) => ({ ...current, scopes: event.target.value }))} required value={tokenForm.scopes} />
              </label>
              <SchemaFeedback errors={schemaErrors.token} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setTokenModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={accessActionState.token?.status === "running"} type="submit">
                  <KeyRound size={13} />
                  <span>{accessActionState.token?.status === "running" ? "创建中" : "创建 Token"}</span>
                </button>
                <ActionFeedback state={accessActionState.token} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {tokenRevokeModalID ? (
          <ModalShell title="撤销 API Token" onClose={() => setTokenRevokeModalID(null)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitTokenRevoke}>
              <div className="modalContext formWide">
                <span>API Token</span>
                <strong>{tokenRevokeModalID}</strong>
              </div>
              <label>
                <FieldLabel required>执行人</FieldLabel>
                <input onChange={(event) => setTokenRevokeForm((current) => ({ ...current, actorID: event.target.value }))} required value={tokenRevokeForm.actorID} />
              </label>
              <label className="formWide">
                <FieldLabel required>原因</FieldLabel>
                <textarea onChange={(event) => setTokenRevokeForm((current) => ({ ...current, reason: event.target.value }))} required value={tokenRevokeForm.reason} />
              </label>
              <SchemaFeedback errors={schemaErrors.tokenRevoke} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setTokenRevokeModalID(null)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton danger" disabled={accessActionState[tokenRevokeModalID]?.status === "running"} type="submit">
                  <X size={13} />
                  <span>{accessActionState[tokenRevokeModalID]?.status === "running" ? "撤销中" : "撤销 Token"}</span>
                </button>
                <ActionFeedback state={accessActionState[tokenRevokeModalID]} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {serviceAccountModalOpen ? (
          <ModalShell title="保存服务账号" onClose={() => setServiceAccountModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={createServiceAccount}>
              <label>
                <span>ID</span>
                <input onChange={(event) => setServiceAccountForm((current) => ({ ...current, id: event.target.value }))} value={serviceAccountForm.id} />
              </label>
              <label>
                <FieldLabel required>名称</FieldLabel>
                <input
                  onChange={(event) => setServiceAccountForm((current) => ({ ...current, name: event.target.value }))}
                  required
                  value={serviceAccountForm.name}
                />
              </label>
              <label>
                <FieldLabel required>角色</FieldLabel>
                <input
                  onChange={(event) => setServiceAccountForm((current) => ({ ...current, roles: event.target.value }))}
                  required
                  value={serviceAccountForm.roles}
                />
              </label>
              <SchemaFeedback errors={schemaErrors.service} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setServiceAccountModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={accessActionState.service?.status === "running"} type="submit">
                  <ShieldCheck size={13} />
                  <span>{accessActionState.service?.status === "running" ? "保存中" : "保存服务账号"}</span>
                </button>
                <ActionFeedback state={accessActionState.service} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {memorySearchModalOpen ? (
          <ModalShell title="搜索 Memory" onClose={() => setMemorySearchModalOpen(false)}>
            <form className="controlForm modalGridForm modalForm" onSubmit={submitMemorySearch}>
              <label className="formWide">
                <FieldLabel required>查询</FieldLabel>
                <input
                  onChange={(event) => setMemorySearchForm((current) => ({ ...current, query: event.target.value }))}
                  placeholder="输入关键词、模块名或决策原因"
                  required
                  value={memorySearchForm.query}
                />
              </label>
              <label>
                <span>Limit</span>
                <input inputMode="numeric" onChange={(event) => setMemorySearchForm((current) => ({ ...current, limit: event.target.value }))} value={memorySearchForm.limit} />
              </label>
              <SchemaFeedback errors={schemaErrors.memorySearch} />
              <div className="modalActions formWide">
                <button className="inlineActionButton" onClick={() => setMemorySearchModalOpen(false)} type="button">
                  <span>取消</span>
                </button>
                <button className="inlineActionButton" disabled={memoryActionState.status === "running"} type="submit">
                  <Search size={13} />
                  <span>{memoryActionState.status === "running" ? "搜索中" : "搜索"}</span>
                </button>
                <ActionFeedback state={memoryActionState} />
              </div>
            </form>
          </ModalShell>
        ) : null}

        {selectedQualityReport ? (
          <ModalShell title="质量报告详情" onClose={() => setQualityDetailReportID(null)}>
            <div className="operationDetail modalDetail">
              <div className="detailHeader">
                <div>
                  <strong>{selectedQualityReport.task_id || selectedQualityReport.id}</strong>
                  <span>{selectedQualityReport.id}</span>
                </div>
                <StatusPill tone={toneForStatus(selectedQualityReport.status)} label={selectedQualityReport.review_status || selectedQualityReport.status} />
              </div>
              <dl>
                <div>
                  <dt>检查</dt>
                  <dd>{selectedQualityReport.check_count}</dd>
                </div>
                <div>
                  <dt>发现</dt>
                  <dd>{selectedQualityReport.findings_count}</dd>
                </div>
                <div>
                  <dt>变更文件</dt>
                  <dd>{selectedQualityReport.changed_files.length}</dd>
                </div>
                <div>
                  <dt>解释</dt>
                  <dd>{selectedQualityExplanation?.decision || "未生成"}</dd>
                </div>
              </dl>
              {selectedQualityExplanation?.reasons.length ? (
                <div className="detailChips">
                  {selectedQualityExplanation.reasons.slice(0, 4).map((reason) => (
                    <code key={reason}>{reason}</code>
                  ))}
                </div>
              ) : null}
              <div className="signalList">
                {selectedQualityReport.checks.map((check, index) => (
                  <div className="signalItem" key={`${selectedQualityReport.id}-check-${index}`}>
                    <div className="signalHeader">
                      <strong>{check.type}</strong>
                      <StatusPill tone={toneForStatus(check.status)} label={check.status} />
                    </div>
                    {check.command ? <span>{check.command}</span> : null}
                    {check.reason ? <small>{check.reason}</small> : null}
                  </div>
                ))}
                {selectedQualityReport.findings.map((finding) => (
                  <div className="signalItem" key={finding.id}>
                    <div className="signalHeader">
                      <strong>{finding.category}</strong>
                      <StatusPill tone={finding.blocking ? "blocked" : toneForStatus(finding.severity)} label={finding.severity} />
                    </div>
                    <span>{finding.message}</span>
                    {finding.path ? <code>{finding.path}</code> : null}
                  </div>
                ))}
              </div>
              {selectedQualityReport.diff_summary_path ? <code>{shortPath(selectedQualityReport.diff_summary_path)}</code> : null}
            </div>
          </ModalShell>
        ) : null}
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
  id?: string;
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

type VisualPlanEnvelope = {
  error?: string;
  visual_plan?: {
    asset?: {
      id?: string;
      diagram_type?: string;
      status?: string;
    };
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

type ReleaseProviderExecutionEnvelope = {
  error?: string;
  release_provider_execution?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ProjectCreateEnvelope = {
  error?: string;
  project?: {
    id?: string;
    name?: string;
    root?: string;
    status?: string;
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

type RollbackExecutionEnvelope = {
  error?: string;
  rollback_execution?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type MonitorSummaryEnvelope = {
  error?: string;
  monitor_summary?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type PostDeploymentVerificationEnvelope = {
  error?: string;
  post_deployment_verification?: {
    id?: string;
    status?: string;
    decision?: string;
    risk_handoff_recommended?: boolean;
  };
};

type DeploymentRehearsalEnvelope = {
  error?: string;
  deployment_rehearsal?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type RehearsalSchedulerEnvelope = {
  error?: string;
  rehearsal_scheduler_run?: {
    id?: string;
    status?: string;
    decision?: string;
    blocked_count?: number;
  };
};

type ReleaseAdmissionEnvelope = {
  error?: string;
  release_admission?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type DeploymentRiskHandoffEnvelope = {
  error?: string;
  deployment_risk_handoff?: {
    id?: string;
    status?: string;
    decision?: string;
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

type ApprovalCreateEnvelope = ApprovalDecisionEnvelope;

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

type ResourceCreateEnvelope = {
  error?: string;
  resource?: {
    id?: string;
    status?: string;
    environment?: string;
    host?: string;
  };
};

type ProviderRouteCandidate = {
  provider_id?: string;
  runtime_id?: string;
  vendor?: string;
  api_type?: string;
  model_id?: string;
  status?: string;
  reason?: string;
  score?: number;
  signals?: Array<{ type?: string; status?: string; reason?: string }>;
};

type ProviderRouteDecision = {
  decision?: string;
  blocked?: boolean;
  strategy?: string;
  provider_id?: string;
  runtime_id?: string;
  model_id?: string;
  reason?: string;
  explanation?: {
    summary?: string;
    selected_provider_id?: string;
    selected_reason?: string;
    candidate_count?: number;
    selected_count?: number;
    skipped_count?: number;
    blocked_count?: number;
  };
  candidates?: ProviderRouteCandidate[];
};

type ProviderRouteEnvelope = {
  error?: string;
  route?: ProviderRouteDecision;
};

type ControlLoopRunEnvelope = {
  error?: string;
  control_loop_run?: {
    id?: string;
    status?: string;
    decision?: string;
    steps?: unknown[];
  };
};

type BatchRunEnvelope = {
  error?: string;
  batch_run?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type BatchPlanEnvelope = {
  error?: string;
  batch_plan?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type MergeQueueEnvelope = {
  error?: string;
  merge_queue?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type IntegrationPreviewEnvelope = {
  error?: string;
  integration_preview?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type IntegrationApplyEnvelope = {
  error?: string;
  integration_apply?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ReleaseBatchEnvelope = {
  error?: string;
  release_batch?: {
    id?: string;
    status?: string;
    decision?: string;
    ready_item_count?: number;
  };
};

type ReleaseCandidateEnvelope = {
  error?: string;
  release_candidate?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ReleaseCandidateApplyEnvelope = {
  error?: string;
  release_candidate_apply?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ReleaseCandidateProviderPreviewEnvelope = {
  error?: string;
  release_candidate_provider_preview?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type DeploymentPlanEnvelope = {
  error?: string;
  deployment?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type OperationRepairReviewEnvelope = {
  error?: string;
  operation_repair_review?: {
    id?: string;
    decision?: string;
    status?: string;
  };
  operation_repair_candidate?: {
    id?: string;
    status?: string;
    decision?: string;
    issue_id?: string;
  };
  repair_attempt?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type OperationRepairCandidateEnvelope = {
  error?: string;
  operation_repair_candidate?: {
    id?: string;
    status?: string;
    decision?: string;
    failure_class?: string;
  };
};

type MergeDecisionEnvelope = {
  error?: string;
  merge_decision?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ResourceScanEnvelope = {
  error?: string;
  maintenance_records?: Array<{ id?: string; status?: string; decision?: string }>;
  lifecycle_scan?: {
    id?: string;
    status?: string;
    decision?: string;
  };
  health_scan?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ResourceDisableEnvelope = {
  error?: string;
  resource?: {
    id?: string;
    status?: string;
  };
};

type DeploymentRiskReviewEnvelope = {
  error?: string;
  deployment_risk_review?: {
    id?: string;
    status?: string;
    decision?: string;
    next_step?: string;
  };
  deployment_risk_handoff?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ControlQueueEnvelope = {
  error?: string;
  control_loop_queue_item?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type ControlQueueRunEnvelope = {
  error?: string;
  control_loop_queue_run?: {
    id?: string;
    status?: string;
    decision?: string;
    processed_count?: number;
  };
};

type OperationReportEnvelope = {
  id?: string;
  summary?: Record<string, number>;
};

type RemoteExecutionRehearsalEnvelope = {
  error?: string;
  remote_execution_rehearsals?: OperationReportEnvelope;
};

type WriteReviewPacketEnvelope = {
  error?: string;
  write_review_packets?: OperationReportEnvelope;
};

type WriteExecutionPlanEnvelope = {
  error?: string;
  write_execution_plans?: OperationReportEnvelope;
};

type WriteAdapterExecutionEnvelope = {
  error?: string;
  write_adapter_executions?: OperationReportEnvelope;
};

type ProviderEnvelope = {
  error?: string;
  provider?: {
    id?: string;
    enabled?: boolean;
    status?: string;
  };
};

type ProviderOpsRefreshEnvelope = {
  error?: string;
  provider_ops_refresh?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

type SkillEnvelope = {
  error?: string;
  skill?: {
    id?: string;
    enabled?: boolean;
    status?: string;
  };
};

type SkillRecommendationEnvelope = {
  error?: string;
  skill_recommendation?: SkillRecommendationSummary;
};

type SkillBindingEnvelope = {
  error?: string;
  skill_binding?: {
    id?: string;
    status?: string;
  };
};

type SkillEffectivenessEnvelope = {
  error?: string;
  skill_effectiveness?: {
    id?: string;
    outcome?: string;
    quality_impact?: string;
  };
};

type MemorySearchEnvelope = {
  error?: string;
  records?: unknown[];
};

async function postJSON<T extends { error?: string }>(path: string, body: unknown): Promise<T> {
  const response = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const payload = (await response.json().catch(() => ({}))) as T;
  if (!response.ok) {
    if (payload.error) {
      throw new Error(payload.error);
    }
    if (response.status === 404) {
      throw new Error("后端没有找到这个接口，请确认 API 服务已重启到最新版本。");
    }
    throw new Error(`请求失败，状态码 ${response.status}`);
  }
  return payload;
}

function splitCSV(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function splitLines(value: string) {
  return value
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseKeyValuePairs(value: string) {
  return Object.fromEntries(
    splitLines(value)
      .map((line) => {
        const [key, ...rest] = line.split("=");
        return [key.trim(), rest.join("=").trim()];
      })
      .filter(([key, item]) => key && item),
  );
}

function parseOptionalInt(value: string) {
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

function parseOptionalFloat(value: string) {
  const parsed = Number.parseFloat(value);
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0;
}

function readField(value: unknown, key: string) {
  return value && typeof value === "object" ? (value as Record<string, unknown>)[key] : undefined;
}

function readString(value: unknown, key: string, fallback: string) {
  const raw = readField(value, key);
  return typeof raw === "string" ? raw : fallback;
}

function readArray(value: unknown, key: string) {
  const raw = readField(value, key);
  return Array.isArray(raw) ? raw.filter((item): item is string => typeof item === "string") : [];
}

function readNumber(value: unknown, key: string) {
  const raw = readField(value, key);
  return typeof raw === "number" && Number.isFinite(raw) ? raw : 0;
}

function normalizeMemoryRecord(raw: unknown, index: number): MemoryRecordSummary {
  return {
    id: readString(raw, "id", `memory-${index + 1}`),
    kind: readString(raw, "kind", "record"),
    summary: readString(raw, "summary", "Memory record"),
    tags: readArray(raw, "tags"),
    source: readString(raw, "source", ""),
    scope: readString(raw, "scope", ""),
    scopes: readArray(raw, "scopes"),
    confidence: readNumber(raw, "confidence"),
    score: Math.min(1, readNumber(raw, "confidence") || readNumber(raw, "score") / 100 || 0.5),
    created_by: readString(raw, "created_by", ""),
    created_at: readString(raw, "created_at", ""),
  };
}

function isMemoryRecord(record: { id: string; kind: string; summary: string; score: number } | MemoryRecordSummary): record is MemoryRecordSummary {
  return "tags" in record && Array.isArray(record.tags);
}

function viewVisible(activeView: ConsoleView, views: ConsoleView[]) {
  return views.includes(activeView);
}

function isConsoleView(value: string): value is ConsoleView {
  return (detailedViews as readonly string[]).includes(value);
}

function groupHasView(group: ConsoleNavGroup, view: ConsoleView) {
  return (group.views as readonly ConsoleView[]).includes(view);
}

function navGroupForView(view: ConsoleView): ConsoleNavGroup {
  return navGroups.find((group) => groupHasView(group, view)) ?? navGroups[0];
}

function resolveConsoleView(value: string): ConsoleView | null {
  if (isConsoleView(value)) {
    return value;
  }
  return navGroups.find((group) => group.label === value)?.views[0] ?? null;
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

function SchemaFeedback({ errors }: { errors?: string[] }) {
  if (!errors || errors.length === 0) return null;
  return (
    <div className="schemaFeedback">
      <AlertTriangle size={14} />
      <div>
        {errors.map((error) => (
          <span key={error}>{error}</span>
        ))}
      </div>
    </div>
  );
}

function PanelTitle({ icon, title, meta, action }: { icon: React.ReactNode; title: string; meta: string; action?: React.ReactNode }) {
  return (
    <div className="panelTitle">
      <div className="panelTitleMain">
        {icon}
        <strong>{title}</strong>
      </div>
      <div className="panelTitleMeta">
        <span>{meta}</span>
        {action}
      </div>
    </div>
  );
}

function ModalShell({ title, children, onClose }: { title: string; children: React.ReactNode; onClose: () => void }) {
  return (
    <div
      className="modalBackdrop"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
      role="presentation"
    >
      <section aria-labelledby={`modal-${title}`} aria-modal="true" className="modalPanel" role="dialog">
        <div className="modalHeader">
          <h2 id={`modal-${title}`}>{title}</h2>
          <button aria-label="关闭" className="iconActionButton" onClick={onClose} type="button">
            <X size={15} />
          </button>
        </div>
        {children}
      </section>
    </div>
  );
}

function FieldLabel({ children, required = false }: { children: string; required?: boolean }) {
  return (
    <span>
      {children}
      {required ? <b className="requiredMark"> *</b> : null}
    </span>
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

type IssueGraphLayoutNode = {
  issue: IssueNode;
  x: number;
  y: number;
  width: number;
  height: number;
  level: number;
};

type IssueGraphLayoutEdge = {
  from: string;
  to: string;
  path: string;
};

type IssueGraphLayout = {
  nodes: IssueGraphLayoutNode[];
  edges: IssueGraphLayoutEdge[];
  columns: { level: number; label: string; x: number }[];
  width: number;
  height: number;
};

function layoutIssueGraph(issues: IssueNode[]): IssueGraphLayout {
  const nodeWidth = 190;
  const nodeHeight = 98;
  const columnGap = 56;
  const rowGap = 24;
  const paddingX = 20;
  const paddingTop = 54;
  const paddingBottom = 20;
  const byID = new Map(issues.map((issue) => [issue.id, issue]));
  const levelCache = new Map<string, number>();

  function levelFor(issue: IssueNode, visiting = new Set<string>()): number {
    const cached = levelCache.get(issue.id);
    if (cached !== undefined) return cached;
    if (visiting.has(issue.id)) return 0;
    visiting.add(issue.id);
    const dependencyLevels = (issue.depends_on ?? [])
      .map((dependencyID) => byID.get(dependencyID))
      .filter((dependency): dependency is IssueNode => Boolean(dependency))
      .map((dependency) => levelFor(dependency, new Set(visiting)) + 1);
    const level = dependencyLevels.length > 0 ? Math.max(...dependencyLevels) : 0;
    levelCache.set(issue.id, level);
    return level;
  }

  const columns = new Map<number, IssueNode[]>();
  for (const issue of issues) {
    const level = levelFor(issue);
    const column = columns.get(level) ?? [];
    column.push(issue);
    columns.set(level, column);
  }

  const sortedLevels = Array.from(columns.keys()).sort((left, right) => left - right);
  const sortedColumns = sortedLevels.map((level) => {
    const items = [...(columns.get(level) ?? [])].sort(compareIssueNodes);
    return { level, items };
  });
  const layoutNodes: IssueGraphLayoutNode[] = [];
  for (const column of sortedColumns) {
    column.items.forEach((issue, row) => {
      layoutNodes.push({
        issue,
        x: paddingX + column.level * (nodeWidth + columnGap),
        y: paddingTop + row * (nodeHeight + rowGap),
        width: nodeWidth,
        height: nodeHeight,
        level: column.level,
      });
    });
  }

  const layoutByID = new Map(layoutNodes.map((node) => [node.issue.id, node]));
  const edges: IssueGraphLayoutEdge[] = [];
  for (const node of layoutNodes) {
    for (const dependencyID of node.issue.depends_on ?? []) {
      const dependency = layoutByID.get(dependencyID);
      if (!dependency) continue;
      edges.push({
        from: dependency.issue.id,
        to: node.issue.id,
        path: issueEdgePath(dependency, node),
      });
    }
  }

  const maxLevel = sortedLevels.length > 0 ? Math.max(...sortedLevels) : 0;
  const maxRows = Math.max(1, ...sortedColumns.map((column) => column.items.length));
  return {
    nodes: layoutNodes,
    edges,
    columns: sortedColumns.map((column) => ({
      level: column.level,
      label: `第 ${column.level + 1} 层`,
      x: paddingX + column.level * (nodeWidth + columnGap),
    })),
    width: Math.max(720, paddingX * 2 + (maxLevel + 1) * nodeWidth + maxLevel * columnGap),
    height: paddingTop + maxRows * nodeHeight + Math.max(0, maxRows - 1) * rowGap + paddingBottom,
  };
}

function compareIssueNodes(left: IssueNode, right: IssueNode) {
  const laneOrder: Record<IssueNode["lane"], number> = { plan: 0, backend: 1, frontend: 2, quality: 3, release: 4 };
  const laneDiff = laneOrder[left.lane] - laneOrder[right.lane];
  if (laneDiff !== 0) return laneDiff;
  return left.id.localeCompare(right.id);
}

function issueEdgePath(from: IssueGraphLayoutNode, to: IssueGraphLayoutNode) {
  const startX = from.x + from.width;
  const startY = from.y + from.height / 2;
  const endX = to.x;
  const endY = to.y + to.height / 2;
  const curve = Math.max(44, (endX - startX) / 2);
  return `M ${startX} ${startY} C ${startX + curve} ${startY}, ${endX - curve} ${endY}, ${endX} ${endY}`;
}

function relatedIssueSets(issues: IssueNode[], selectedIssueID: string) {
  const byID = new Map(issues.map((issue) => [issue.id, issue]));
  const downstreamByID = new Map<string, string[]>();
  for (const issue of issues) {
    for (const dependencyID of issue.depends_on ?? []) {
      const items = downstreamByID.get(dependencyID) ?? [];
      items.push(issue.id);
      downstreamByID.set(dependencyID, items);
    }
  }
  return {
    upstream: collectRelated(selectedIssueID, (id) => byID.get(id)?.depends_on ?? []),
    downstream: collectRelated(selectedIssueID, (id) => downstreamByID.get(id) ?? []),
  };
}

function collectRelated(startID: string, nextIDs: (id: string) => string[]) {
  const visited = new Set<string>();
  const stack = nextIDs(startID);
  while (stack.length > 0) {
    const id = stack.pop();
    if (!id || visited.has(id)) continue;
    visited.add(id);
    stack.push(...nextIDs(id));
  }
  return visited;
}

function graphEdgeClass(edge: IssueGraphLayoutEdge, relations: { upstream: Set<string>; downstream: Set<string> }, selectedIssueID: string) {
  if (!selectedIssueID) return "";
  const upstreamEdge =
    (edge.to === selectedIssueID || relations.upstream.has(edge.to)) && (edge.from === selectedIssueID || relations.upstream.has(edge.from));
  if (upstreamEdge) return "upstream";
  const downstreamEdge =
    (edge.from === selectedIssueID || relations.downstream.has(edge.from)) && (edge.to === selectedIssueID || relations.downstream.has(edge.to));
  if (downstreamEdge) return "downstream";
  return "dimmed";
}

function groupRunsByIssue(runs: RunSummary[]) {
  const records = new Map<string, RunSummary[]>();
  for (const run of runs) {
    if (!run.issue_id) continue;
    const items = records.get(run.issue_id) ?? [];
    items.push(run);
    records.set(run.issue_id, items);
  }
  return records;
}

function toneForStatus(status: string): StatusTone {
  if (
    status === "accepted" ||
    status === "approved" ||
    status === "selected" ||
    status === "passed" ||
    status === "ready" ||
    status === "ready_to_merge" ||
    status === "completed" ||
    status === "planned" ||
    status === "applied" ||
    status === "allowed" ||
    status === "ok" ||
    status === "healthy"
  )
    return "ok";
  if (status === "running" || status === "dispatch" || status === "retrying") return "running";
  if (
    status === "blocked" ||
    status === "rejected" ||
    status === "failed" ||
    status === "route_blocked" ||
    status === "unhealthy" ||
    status === "down" ||
    status === "expired" ||
    status === "critical" ||
    status === "smoke_failed" ||
    status === "monitor_failed" ||
    status === "execution_failed" ||
    status === "execution_blocked" ||
    status === "operation_failed" ||
    status === "operation_blocked" ||
    status === "check_failed" ||
    status === "conflict"
  )
    return "blocked";
  if (
    status === "needs_rework" ||
    status === "dry_run" ||
    status === "waiting" ||
    status === "pending" ||
    status === "archived" ||
    status === "open" ||
    status === "warning" ||
    status === "degraded" ||
    status === "attention_required" ||
    status === "manual_required" ||
    status === "manual_check_required" ||
    status === "review_required" ||
    status === "suggested" ||
    status === "not_ready"
  )
    return "warning";
  return "neutral";
}

function statusClass(status: string) {
  return toneForStatus(status);
}

function compactID(value: string) {
  const displayValue = cleanDisplayID(value);
  if (!displayValue) return "unknown";
  const chars = Array.from(displayValue);
  if (chars.length <= 28) return displayValue;
  return `${chars.slice(0, 18).join("")}...${chars.slice(-7).join("")}`;
}

function cleanDisplayID(value: string) {
  return value
    .replace(/\uFFFD+/g, "")
    .replace(/-{2,}/g, "-")
    .replace(/-\./g, ".")
    .replace(/\.-/g, ".")
    .replace(/^-+|-+$/g, "")
    .trim();
}

function compactCommit(value: string) {
  if (!value) return "无";
  return value.slice(0, 10);
}

function shortPath(value?: string) {
  if (!value) return "path pending";
  const parts = value.split("/").filter(Boolean);
  return parts.slice(-3).join("/");
}

function projectGitAddress(project: ProjectSummary) {
  const source = project.source ?? {};
  return project.remote_url || readSourceString(source, ["url", "remote_url", "git_url", "repository_url", "html_url"]) || "未绑定 Git 远程";
}

function projectLocalPath(project: ProjectSummary) {
  const source = project.source ?? {};
  return project.root || readSourceString(source, ["path", "clone_path", "local_path"]) || "本机路径待登记";
}

function projectTechStack(project: ProjectSummary) {
  const items = [...(project.languages ?? []), ...(project.frameworks ?? []), ...(project.package_managers ?? [])]
    .map((item) => item.trim())
    .filter((item) => item && item !== "unknown");
  return items.length > 0 ? Array.from(new Set(items)).join(" / ") : "待识别";
}

function readSourceString(source: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = source[key];
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
  }
  return "";
}

function orderProjects(projects: ProjectSummary[], activeProjectID: string) {
  return [...projects].sort((left, right) => {
    if (left.id === activeProjectID) {
      return -1;
    }
    if (right.id === activeProjectID) {
      return 1;
    }
    return 0;
  });
}

function orderRequirements(requirements: ConsoleSnapshot["requirements"]) {
  return [...requirements].sort((left, right) => {
    const leftPhase = phaseNumberForRequirement(left);
    const rightPhase = phaseNumberForRequirement(right);
    if (leftPhase !== rightPhase) {
      return leftPhase - rightPhase;
    }
    const leftCreatedAt = Date.parse(left.created_at ?? "");
    const rightCreatedAt = Date.parse(right.created_at ?? "");
    return (Number.isNaN(leftCreatedAt) ? Number.MAX_SAFE_INTEGER : leftCreatedAt) - (Number.isNaN(rightCreatedAt) ? Number.MAX_SAFE_INTEGER : rightCreatedAt);
  });
}

function defaultBatchEpicID(snapshot: ConsoleSnapshot) {
  return [...snapshot.requirements].filter(isActiveBatchRequirement).sort(compareActiveBatchRequirements)[0]?.epic_id ?? "";
}

function isActiveBatchRequirement(requirement: ConsoleSnapshot["requirements"][number]) {
  if (finishedRequirementStatuses.has(requirement.status.trim().toLowerCase())) {
    return false;
  }
  return requirement.issues.some((issue) => !finishedIssueStatuses.has(issue.status.trim().toLowerCase()));
}

function compareActiveBatchRequirements(left: ConsoleSnapshot["requirements"][number], right: ConsoleSnapshot["requirements"][number]) {
  const leftPhase = phaseNumberForRequirement(left);
  const rightPhase = phaseNumberForRequirement(right);
  if (leftPhase !== rightPhase) {
    return leftPhase - rightPhase;
  }
  const leftCreatedAt = Date.parse(left.created_at ?? "");
  const rightCreatedAt = Date.parse(right.created_at ?? "");
  return (Number.isNaN(leftCreatedAt) ? Number.MAX_SAFE_INTEGER : leftCreatedAt) - (Number.isNaN(rightCreatedAt) ? Number.MAX_SAFE_INTEGER : rightCreatedAt);
}

function phaseNumberForRequirement(requirement: ConsoleSnapshot["requirements"][number]) {
  const match = `${requirement.raw_text} ${requirement.title} ${requirement.epic_id}`.match(/phase\s*(\d+)/i);
  return match ? Number(match[1]) : Number.MAX_SAFE_INTEGER;
}

const finishedRequirementStatuses = new Set(["accepted", "archived", "closed", "completed", "done"]);
const finishedIssueStatuses = new Set(["accepted", "archived", "closed", "completed", "done", "passed", "success", "succeeded"]);

function batchPlanTitle(plan: BatchPlanSummary, requirementByEpicID: Map<string, RequirementSummary>) {
  const requirement = requirementByEpicID.get(plan.epic_id);
  if (!requirement) {
    return compactID(plan.epic_id || plan.id);
  }
  const phase = phaseNumberForRequirement(requirement);
  if (Number.isFinite(phase)) {
    return `Phase ${phase} 批量执行`;
  }
  return requirement.title || compactID(plan.epic_id || plan.id);
}

function batchPlanSubtitle(plan: BatchPlanSummary, requirementByEpicID: Map<string, RequirementSummary>) {
  const requirement = requirementByEpicID.get(plan.epic_id);
  const pieces = [compactID(plan.id)];
  if (requirement?.title) {
    pieces.push(requirement.title.replace(/^Phase\s*\d+\s*[:：-]?\s*/i, ""));
  }
  if (plan.created_at) {
    pieces.push(shortTimestamp(plan.created_at));
  }
  return pieces.join(" / ");
}

function modeLabel(mode: string) {
  const normalized = mode.trim().toLowerCase();
  const labels: Record<string, string> = {
    apply: "应用",
    dry_run: "dry-run",
    execute: "执行",
    preview: "预览",
  };
  return labels[normalized] ?? mode;
}

function decisionLabel(value: string) {
  const normalized = value.trim().toLowerCase();
  const labels: Record<string, string> = {
    batch_item_dry_run: "dry-run 项",
    batch_plan_ready: "批量计划就绪",
    batch_run_dry_run: "dry-run 已完成",
    dispatch_ready: "可派发",
    no_runtime_executed: "未执行真实运行时",
  };
  return labels[normalized] ?? statusLabel(value);
}

function statusLabel(status: string) {
  const normalized = status.trim().toLowerCase();
  const labels: Record<string, string> = {
    accepted: "已接受",
    archived: "已归档",
    blocked: "阻断",
    closed: "已关闭",
    completed: "已完成",
    dispatch: "待派发",
    done: "已完成",
    failed: "失败",
    needs_rework: "需返工",
    pending: "待处理",
    planned: "已规划",
    ready: "就绪",
    reviewing: "复核中",
    running: "运行中",
    waiting: "等待中",
  };
  return labels[normalized] ?? status;
}

function shortTimestamp(value?: string) {
  if (!value) return "时间待定";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "时间待定";
  return new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}
