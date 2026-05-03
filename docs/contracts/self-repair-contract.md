# 自我修复契约

本文定义运行信号、Bug Candidate、Bug 分类、Repair Attempt 和能力增强记录的实现接口。判断规则由 [Bug 判断与自我修复策略](../policies/bug-detection-self-repair-policy.md) 维护。

## 1. 目标

- 将运行过程中的异常信号转成结构化对象。
- 让 Orchestrator 能判断是否为 bug、是否可自动修复。
- 让自动修复过程可审计、可验证、可回滚。
- 让成功修复经验进入 Memory 和后续任务上下文。

## 2. 核心接口

```ts
export type RuntimeSignalType =
  | "test_failure"
  | "runtime_error"
  | "smoke_failure"
  | "monitor_alert"
  | "user_feedback"
  | "review_finding"
  | "repeated_pattern";

export interface RuntimeSignal {
  id: string;
  projectId: string;
  signalType: RuntimeSignalType;
  sourceType: "run" | "issue" | "commit" | "release" | "deployment" | "user" | "monitor";
  sourceId?: string;
  summary: string;
  evidenceRefs: string[];
  environment?: "local" | "test_dev" | "staging" | "production";
  occurredAt: string;
  traceId: string;
}

export type BugClassification =
  | "CONFIRMED_BUG"
  | "NOT_BUG"
  | "NEEDS_EVIDENCE"
  | "ENHANCEMENT_CANDIDATE";

export interface BugCandidate {
  id: string;
  projectId: string;
  signalIds: string[];
  title: string;
  affectedScope: string[];
  suspectedRootCause?: string;
  reproducible: boolean;
  reproductionCommands: string[];
  classification: BugClassification;
  confidence: number;
  riskLevel: "low" | "medium" | "high" | "critical";
  status:
    | "detected"
    | "classifying"
    | "confirmed"
    | "not_bug"
    | "needs_evidence"
    | "issue_created"
    | "repairing"
    | "repaired"
    | "archived";
}

export interface RepairPlan {
  id: string;
  bugCandidateId: string;
  projectId: string;
  issueId?: string;
  writeScope: string[];
  strategy: "minimal_fix" | "test_first_fix" | "rollback" | "issue_only";
  regressionTestRequired: boolean;
  commands: string[];
  requiresApproval: boolean;
  approvalId?: string;
}

export interface RepairAttemptResult {
  id: string;
  repairPlanId: string;
  runId: string;
  status: "passed" | "failed" | "needs_rework" | "blocked" | "escalated";
  changedFiles: string[];
  regressionTests: string[];
  qualityReportId?: string;
  reviewDecision?: "accepted" | "needs_rework" | "rejected";
  memoryCandidateIds: string[];
}

export interface ImprovementRecord {
  id: string;
  projectId: string;
  sourceRepairAttemptId?: string;
  type:
    | "bug_signature"
    | "fix_pattern"
    | "regression_test"
    | "quality_rule_suggestion"
    | "skill_recommendation"
    | "model_routing_suggestion"
    | "module_map_update";
  summary: string;
  confidence: number;
  status: "candidate" | "approved" | "applied" | "rejected" | "archived";
}
```

## 3. Engine 接口

```ts
export interface SelfRepairEngine {
  captureSignal(signal: RuntimeSignal): Promise<RuntimeSignal>;
  classify(candidateId: string): Promise<BugCandidate>;
  planRepair(candidateId: string): Promise<RepairPlan>;
  runRepair(planId: string): Promise<RepairAttemptResult>;
  recordImprovement(resultId: string): Promise<ImprovementRecord[]>;
}
```

实现要求：

- `captureSignal` 不保存 secret、完整 `.env` 或未脱敏生产数据。
- `classify` 必须输出 confidence 和 evidence refs。
- `planRepair` 必须遵守 write scope、auth context 和审批策略。
- `runRepair` 必须经过 Runtime Adapter、质量门禁和 review。
- `recordImprovement` 只能生成 Memory candidate，不能绕过 Memory Record Gate。

## 4. 错误类型

| 错误 | 含义 |
| --- | --- |
| `SELF_REPAIR_SIGNAL_INVALID` | 信号缺少必要字段 |
| `SELF_REPAIR_NOT_BUG` | 分类结果不是 bug |
| `SELF_REPAIR_NEEDS_EVIDENCE` | 证据不足 |
| `SELF_REPAIR_SCOPE_UNSAFE` | 写入范围不安全 |
| `SELF_REPAIR_APPROVAL_REQUIRED` | 需要审批 |
| `SELF_REPAIR_REPRODUCTION_FAILED` | 复现步骤无法执行 |
| `SELF_REPAIR_QUALITY_FAILED` | 修复后质量门禁失败 |
| `SELF_REPAIR_REVIEW_REJECTED` | review 拒绝 |
| `SELF_REPAIR_MAX_ATTEMPTS_EXCEEDED` | 超过自动修复上限 |

## 5. 日志事件

必须产生：

- `self_repair.signal.captured`
- `self_repair.bug.classified`
- `self_repair.repair.planned`
- `self_repair.repair.started`
- `self_repair.repair.completed`
- `self_repair.repair.failed`
- `self_repair.improvement.candidate_created`
- `self_repair.improvement.applied`

事件必须包含：

- `trace_id`
- `project_id`
- `bug_candidate_id`
- `repair_attempt_id`，如适用
- `run_id`，如适用
- `decision`
- `reason`

## 6. 验收标准

- 任意运行异常可以转成 Runtime Signal。
- Bug Candidate 能保留证据、风险、复现步骤和分类结果。
- Repair Plan 能被转换成受控 issue/run。
- Repair Attempt 结果能关联质量报告和 review 结论。
- Improvement Record 只能作为候选进入 Memory 或策略建议，不能直接改写核心策略。
