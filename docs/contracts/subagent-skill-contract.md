# Subagent 与 Skill 契约

本文定义 Subagent、Skill Registry、Skill 绑定和 Skill 效果反馈的实现接口。完整设计见 [Subagent 与 Skills 系统方案](../subagents-skills-system.md)。

## 1. 目标

- 让 Subagent 成为可创建、可调度、可审计的执行对象。
- 让 Skill 成为可发现、可绑定、可验证、可复盘的能力对象。
- 让 Orchestrator 能统一管理父子任务、并发、输出收敛和失败恢复。

## 2. Subagent 接口

```ts
export type SubagentType =
  | "planning_subagent"
  | "discovery_subagent"
  | "implementation_subagent"
  | "verification_subagent"
  | "repair_subagent"
  | "release_subagent"
  | "memory_subagent";

export type SubagentStatus =
  | "planned"
  | "context_assembled"
  | "dispatched"
  | "running"
  | "output_collected"
  | "validated"
  | "completed"
  | "archived"
  | "blocked"
  | "failed"
  | "timeout"
  | "needs_rework"
  | "needs_user_input"
  | "cancelled"
  | "superseded";

export interface SubagentInstance {
  id: string;
  projectId: string;
  parentType: "epic" | "issue" | "run" | "repair_attempt" | "release" | "deployment" | "memory_job";
  parentId: string;
  issueId?: string;
  runId?: string;
  roleId: string;
  type: SubagentType;
  runtimeId: string;
  modelPolicyId: string;
  skillBindingIds: string[];
  memoryScopes: string[];
  readScope: string[];
  writeScope: string[];
  allowedTools: string[];
  status: SubagentStatus;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  traceId: string;
}
```

## 3. Subagent 计划接口

```ts
export interface SubagentPlan {
  id: string;
  projectId: string;
  parentType: SubagentInstance["parentType"];
  parentId: string;
  objective: string;
  acceptanceCriteria: string[];
  requiredRoles: string[];
  candidateSkills: string[];
  dependencies: string[];
  readScope: string[];
  writeScope: string[];
  riskLevel: "low" | "medium" | "high" | "critical";
  requiresApproval: boolean;
  blockedReason?: string;
}
```

## 4. Subagent 输出接口

```ts
export interface SubagentResult {
  subagentId: string;
  runId: string;
  status: "completed" | "failed" | "needs_rework" | "needs_user_input" | "blocked";
  summary: string;
  changedFiles: Array<{
    path: string;
    changeType: "added" | "modified" | "deleted" | "renamed";
    reason: string;
  }>;
  commands: Array<{
    command: string;
    status: "passed" | "failed" | "skipped";
    exitCode?: number;
  }>;
  tests: Array<{
    name: string;
    status: "passed" | "failed" | "skipped";
  }>;
  risks: Array<{
    severity: "low" | "medium" | "high" | "blocker";
    message: string;
  }>;
  memoryCandidateIds: string[];
  skillEffectivenessIds: string[];
}
```

## 5. Skill Registry 接口

```ts
export interface SkillDefinition {
  id: string;
  name: string;
  version: string;
  source: "builtin" | "project" | "organization" | "marketplace" | "manual";
  description: string;
  supportedRoles: string[];
  taskTypes: string[];
  requiredTools: string[];
  memoryScopes: string[];
  riskLevel: "low" | "medium" | "high";
  inputContractRef?: string;
  outputContractRef?: string;
  validationCommands: string[];
  enabled: boolean;
}

export interface SkillBinding {
  id: string;
  projectId: string;
  skillId: string;
  targetType: "project" | "role" | "issue" | "subagent";
  targetId: string;
  priority: number;
  config: Record<string, unknown>;
  status: "candidate" | "enabled" | "disabled" | "deprecated";
}

export interface SkillRecommendation {
  id: string;
  projectId: string;
  issueId?: string;
  roleId?: string;
  query: string;
  candidates: Array<{
    skillId: string;
    score: number;
    reasons: string[];
    risks: string[];
  }>;
  decision: "bind" | "reject" | "needs_approval";
}
```

## 6. Skill 效果接口

```ts
export interface SkillEffectiveness {
  id: string;
  projectId: string;
  skillId: string;
  subagentId: string;
  issueId?: string;
  outcome: "helped" | "neutral" | "harmful" | "blocked";
  qualityImpact: "improved" | "unchanged" | "worsened";
  reworkReduced: boolean;
  bugIntroduced: boolean;
  reviewerDecision?: "accepted" | "needs_rework" | "rejected";
  notes: string;
  createdAt: string;
}
```

## 7. Engine 接口

```ts
export interface SubagentSkillEngine {
  planSubagents(parentType: string, parentId: string): Promise<SubagentPlan[]>;
  createSubagent(planId: string): Promise<SubagentInstance>;
  dispatchSubagent(subagentId: string): Promise<SubagentResult>;
  validateSubagentResult(result: SubagentResult): Promise<SubagentResult>;
  recommendSkills(input: {
    projectId: string;
    issueId?: string;
    roleId?: string;
    taskType: string;
    stackTags: string[];
  }): Promise<SkillRecommendation>;
  bindSkill(binding: SkillBinding): Promise<SkillBinding>;
  recordSkillEffectiveness(effectiveness: SkillEffectiveness): Promise<SkillEffectiveness>;
}
```

实现要求：

- `planSubagents` 不能绕过 Issue Graph 和权限策略。
- `createSubagent` 必须绑定父对象、role、runtime、skills、scope 和 trace。
- `dispatchSubagent` 必须通过 Runtime Adapter。
- `validateSubagentResult` 必须校验输出契约、写入范围和质量入口。
- `recommendSkills` 不能把外部 skill 默认用于敏感上下文。
- `recordSkillEffectiveness` 只能生成推荐、降权或 memory candidate，不能直接改写核心策略。

## 8. 错误类型

| 错误 | 含义 |
| --- | --- |
| `SUBAGENT_PARENT_INVALID` | 父对象不存在或状态不允许创建 |
| `SUBAGENT_ROLE_UNRESOLVED` | 无法解析 role |
| `SUBAGENT_SCOPE_UNSAFE` | 读写范围不安全 |
| `SUBAGENT_RUNTIME_UNAVAILABLE` | Runtime 不可用 |
| `SUBAGENT_OUTPUT_INVALID` | 输出不符合契约 |
| `SUBAGENT_CONFLICT_DETECTED` | 与其他 Subagent 写入冲突 |
| `SKILL_NOT_FOUND` | skill 不存在 |
| `SKILL_INCOMPATIBLE` | skill 与 role、task 或工具权限不兼容 |
| `SKILL_APPROVAL_REQUIRED` | skill 风险较高，需要审批 |
| `SKILL_EFFECTIVENESS_INVALID` | skill 效果记录无效 |

## 9. 日志事件

必须产生：

- `subagent.planned`
- `subagent.created`
- `subagent.dispatched`
- `subagent.completed`
- `subagent.failed`
- `subagent.output_validated`
- `skill.recommended`
- `skill.bound`
- `skill.effectiveness.recorded`

事件必须包含：

- `trace_id`
- `project_id`
- `parent_type`
- `parent_id`
- `subagent_id`，如适用
- `skill_id`，如适用
- `run_id`，如适用
- `decision`
- `reason`

## 10. 验收标准

- Orchestrator 能为一个 Issue 创建多个 Subagent。
- 每个 Subagent 能关联父对象、role、runtime、skills、memory scope 和读写范围。
- Subagent 输出能被 Runtime Adapter、质量门禁和 review 消费。
- Skill 能被推荐、绑定、禁用和记录效果。
- 高风险 skill 和写权限 Subagent 必须触发权限或审批策略。
- Subagent 和 Skill 事件能进入统一日志与审计。
