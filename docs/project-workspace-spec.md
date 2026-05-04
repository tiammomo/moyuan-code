# 项目工作空间规范

## 1. 目标

每个被管理项目都拥有独立 `.moyuan/` 工作空间。本文只维护 schema 索引、目录职责和跨模块引用，不重复各能力模块的完整配置。

详细规则请看：

- 配置方案：[配置方案](./configuration-guide.md)
- 配置字段规则：[配置 Schema 规则](./configuration-schema-spec.md)
- 仓库和 Git Adapter：[仓库接入与 Git Provider Adapter](./repository-onboarding-git-management.md)
- 项目阅读理解：[项目接入与阅读理解主线](./mainlines/project-comprehension.md)
- Agent、team、skills：[Agent 角色与团队概览](./agent-roles-overview.md)
- Memory：[Agent Memory 系统方案](./agent-memory-system.md)
- 质量门禁：[代码生命周期质量门禁](./code-lifecycle-quality-gates.md)
- 模型与工具：[模型与工具适配规划](./model-tool-adapters.md)
- 生命周期和 CLI：[总体规划与生命周期路线图](./lifecycle-roadmap.md)

## 2. 目录结构

```text
.moyuan/
  project.yaml
  repository.yaml
  state.db
  agents/
    roles.yaml
    teams.yaml
    subagents.yaml
    subagents/
  models/
    providers.yaml
    providers.json
    routing.yaml
  visuals/
    architecture-visuals.yaml
    specs/
    prompts/
    diagrams/
    explanations/
    index.jsonl
  runtimes/
    agent-runtimes.yaml
    sessions/
    outputs/
    context/
  model-ops/
    providers.snapshot.json
    usage/
    health/
    cost/
    incidents/
  skills/
    enabled.yaml
    registry.json
    bindings.json
    events.jsonl
    bindings.events.jsonl
    effectiveness/
    recommendations.jsonl
  memory/
    facts.jsonl
    decisions.md
    preferences.yaml
    lessons.jsonl
    candidates.jsonl
    staging.jsonl
    audit.jsonl
    indexes/
    compacted/
    archive/
    runtime/
    reflections/
  resources/
    inventory.json
    events.jsonl
    checks/
    maintenance/
  logs/
    runs/
    agents/
    models/
    git/
    quality/
    releases/
    memory/
    audit/
    errors/
  comprehension/
    project-profile.md
    module-map.md
    dependency-map.md
    commands.md
    risks.md
    events.jsonl
    snapshots/
  resources/
    inventory.yaml
    events.jsonl
    checks/
    maintenance/
    renewals/
    cost/
  lifecycle/
    epics/
    issues/
    issue-graphs/
    schedules/
    requirements/
    designs/
    tasks/
    runs/
    quality/
    reviews/
    signals/
    bug-candidates/
    repair-attempts/
    improvements/
    releases/
    deployments/
    retrospectives/
  policies/
    access.yaml
    permissions.yaml
    secrets.yaml
    budget.yaml
    orchestration.yaml
    engineering.yaml
    code-quality.yaml
    comprehension.yaml
    memory.yaml
    logging.yaml
    server-resources.yaml
    release.yaml
    environments.yaml
  reports/
  tmp/
    transactions/
  .locks/
```

## 3. Schema 索引

| 文件/目录 | 职责 | 权威文档 |
| --- | --- | --- |
| `project.yaml` | 项目基础信息、技术栈摘要、生命周期指针 | 本文 |
| `state.db` | GORM SQLite 查询索引，当前索引项目来源、服务商、owner、状态和注册时间 | [持久化与并发一致性](./persistence-concurrency-consistency.md) |
| 配置索引和关键片段 | 初始化项目所需配置分层、闭环和关键片段 | [配置方案](./configuration-guide.md) |
| 配置字段规则 | 必填、可选、可为空、必须为空、默认值和条件必填 | [配置 Schema 规则](./configuration-schema-spec.md) |
| `repository.yaml` | 仓库来源、remote、分支策略、PR/MR 策略 | [仓库接入与 Git Provider Adapter](./repository-onboarding-git-management.md) |
| `agents/roles.yaml` | Agent role、工具权限、skills、memory scope | [Agent 角色与团队概览](./agent-roles-overview.md) |
| `agents/teams.yaml` | 默认 team、任务类型 team、验证链路 | [Agent 角色与团队概览](./agent-roles-overview.md) |
| `agents/subagents.yaml` | Subagent 创建、并发、生命周期和委派策略 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| `agents/subagents/` | Subagent 实例、父对象、状态、输出和审计索引 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| `models/providers.yaml` | 模型服务商 API、账号、模型能力、额度、健康检查和第三方网关的目标 schema | [模型与工具适配规划](./model-tool-adapters.md) |
| `models/providers.json` | Beta 运行期 Provider Registry 快照，字段与目标 schema 对齐 | [模型与工具适配规划](./model-tool-adapters.md) |
| `models/routing.yaml` | 模型路由、fallback、成本策略 | [模型与工具适配规划](./model-tool-adapters.md) |
| `visuals/architecture-visuals.yaml` | gpt-image-2 架构流程图生成、编辑、讲解和复核策略 | [配置方案](./configuration-guide.md) |
| `visuals/` | 架构图 spec、prompt、图片资产、讲解文档和索引 | [配置方案](./configuration-guide.md) |
| `resources/inventory.json` | Beta 运行期服务器资源 registry，记录测试机、预发和生产机 | [服务器资源管理主线](./mainlines/server-resource-management.md) |
| `resources/events.jsonl` | 服务器资源登记、禁用、维护和到期扫描事件 | [服务器资源管理主线](./mainlines/server-resource-management.md) |
| `lifecycle/deployments/` | Beta 运行期 deploy/smoke/monitor/rollback plan | [DevOps 发布投产主线](./mainlines/devops-release-deployment.md) |
| `runtimes/agent-runtimes.yaml` | Claude CLI、Codex CLI 等原生 Agent Runtime 调用、会话、隔离和审计 | [模型与工具适配规划](./model-tool-adapters.md) |
| `runtimes/` | 原生 Agent Runtime 会话、输出和上下文文件 | [配置方案](./configuration-guide.md) |
| `model-ops/` | 模型服务商快照、用量、健康检查、成本和故障记录 | [配置方案](./configuration-guide.md) |
| `skills/enabled.yaml` | 启用 skills、skill source | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| `skills/registry.json` | Phase 2 运行期 Skill Registry，记录版本、来源、适配 role、风险和启用状态 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| `skills/events.jsonl` | Skill Registry 变更审计流 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| `skills/bindings.json` | Phase 2 运行期 project、role、issue、subagent 级 skill 绑定 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| `skills/bindings.events.jsonl` | Skill Binding 变更审计流 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| `skills/effectiveness/` | skill 使用效果、质量影响、返工和降权依据 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| `memory/` | 长期记忆、候选、暂存、索引、审计 | [Agent Memory 系统方案](./agent-memory-system.md) |
| `logs/` | run、agent、model、git、quality、release、memory、audit 和 error 核心日志 | [配置方案](./configuration-guide.md) |
| `comprehension/` | 项目画像、模块地图、理解事件 | [项目接入与阅读理解主线](./mainlines/project-comprehension.md) |
| `resources/` | 服务器资产清单、巡检、续费、成本和变更事件 | [配置方案](./configuration-guide.md) |
| `lifecycle/epics/` | 用户开发目标和总体计划 | [Issues 编排与并发调度](./issue-orchestration.md) |
| `lifecycle/issues/` | issue 定义、依赖、写入范围、验收标准 | [Issues 编排与并发调度](./issue-orchestration.md) |
| `lifecycle/issue-graphs/` | issue DAG、ready queue、blocked/running 状态 | [Issues 编排与并发调度](./issue-orchestration.md) |
| `lifecycle/schedules/` | 调度计划、并发度、worktree 分配 | [Issues 编排与并发调度](./issue-orchestration.md) |
| `lifecycle/` | 需求、设计、run、quality、review、release、retro | [总体规划与生命周期路线图](./lifecycle-roadmap.md) |
| `lifecycle/signals/` | 运行、测试、冒烟、监控、用户反馈和 review 异常信号 | [运行反馈与自我修复主线](./mainlines/runtime-feedback-self-repair.md) |
| `lifecycle/bug-candidates/` | 疑似 bug、证据、分类结果和阻断原因 | [运行反馈与自我修复主线](./mainlines/runtime-feedback-self-repair.md) |
| `lifecycle/repair-attempts/` | 自动修复计划、执行记录、质量结果和 review 结论 | [自我修复契约](./contracts/self-repair-contract.md) |
| `lifecycle/improvements/` | bug signature、fix pattern、测试策略和能力增强候选 | [自我修复契约](./contracts/self-repair-contract.md) |
| `lifecycle/deployments/` | 投产记录、服务器部署结果、冒烟和监控报告 | [DevOps 发布投产主线](./mainlines/devops-release-deployment.md) |
| `policies/orchestration.yaml` | issue graph、自动并发、冲突保护、replan 策略 | [Issues 编排与并发调度](./issue-orchestration.md) |
| `policies/engineering.yaml` | commit、issue、fix、release 和 coverage 规范入口 | [工程流程规范](./engineering-process-standards.md) |
| `policies/access.yaml` | 项目级角色、成员访问边界和审批入口；不保存身份凭证明文 | [平台用户与访问控制主线](./mainlines/platform-user-access.md) |
| `policies/permissions.yaml` | 文件、命令、网络、密钥权限 | [权限模型](./foundations/permission-model.md) |
| `policies/code-quality.yaml` | 质量门禁策略 | [代码生命周期质量门禁](./code-lifecycle-quality-gates.md) |
| `policies/comprehension.yaml` | 项目理解触发策略 | [项目接入与阅读理解策略](./policies/project-comprehension-policy.md) |
| `policies/memory.yaml` | Record Gate、抽取、暂存、检索、维护策略 | [Agent Memory 系统方案](./agent-memory-system.md) |
| `policies/logging.yaml` | 核心日志、审计日志、脱敏、保留和导出策略 | [配置方案](./configuration-guide.md) |
| `policies/server-resources.yaml` | 测试开发机、生产机、云资产、到期时间、资源组、连接、权限和健康检查 | [服务器资源管理主线](./mainlines/server-resource-management.md) |
| `policies/release.yaml` | 版本分支、tag、PR/MR、投产和回滚策略 | [DevOps 发布投产主线](./mainlines/devops-release-deployment.md) |
| `policies/environments.yaml` | 服务器、环境、部署方式、健康检查、监控配置引用 | [DevOps 发布投产主线](./mainlines/devops-release-deployment.md) |
| `reports/` | 人类可读报告 | [总体规划与生命周期路线图](./lifecycle-roadmap.md) |
| `tmp/transactions/` | 跨文件状态变更的事务 journal、崩溃恢复和回滚依据 | [持久化与并发一致性](./persistence-concurrency-consistency.md) |
| `.locks/` | project、issue、graph、run、memory、release 等 scoped lock | [持久化与并发一致性](./persistence-concurrency-consistency.md) |

## 4. Schema Validator 与状态索引

当前实现：

- CLI：`moyuan workspace validate`、`moyuan workspace doctor`。
- Validator 输出 `status=passed|warning|failed` 和结构化 `issues[]`。
- 校验范围包括 `.moyuan/workspace.json`、`project.yaml`、`repository.yaml`、`policies/access.yaml` 的存在性、YAML 解析、核心必填字段、条件必填、必须为空和关键字段漂移。
- `Load` 优先读取用户可编辑的 YAML 配置，再回退到 `workspace.json` 或默认值；`workspace.json` 保持运行期状态索引职责。
- `workspace doctor` 同时输出 `.moyuan/state.db` 路径、可用性和项目索引数量。
- `.moyuan/state.db` 的项目表已对 `source_type`、`provider`、`owner_id`、`status`、`registered_at` 建索引，用于后续控制台和 API 查询。

## 5. project.yaml 最小结构

```yaml
schema_version: 1
project:
  id: moyuan-demo
  name: Moyuan Demo
  root: .
  type: single-repo
  description: 多 Agent 代码开发框架示例项目

repository:
  source_type: local_path
  provider: local
  local_path: .
  default_remote: origin
  default_branch: main

stack:
  languages: []
  frameworks: []
  package_managers: []
  build_commands: []
  test_commands: []
  lint_commands: []

workspace:
  protected_paths:
    - .env
    - .env.*
    - secrets/**
  writable_paths:
    - src/**
    - tests/**
    - docs/**

lifecycle:
  current_phase: planning
  current_iteration: 0
```

## 6. Run 记录最小结构

Run 是任务执行审计的核心对象。详细字段可以由各模块扩展，但最小结构必须包含：

```json
{
  "id": "run-20260503-001",
  "task_id": "task-20260503-001",
  "agents": ["planner", "backend", "tester", "quality_guard", "reviewer"],
  "model_policy": "coding_strong",
  "status": "completed",
  "changed_files": [],
  "git": {
    "base_branch": "main",
    "base_commit": "abc123",
    "task_branch": "moyuan/task-20260503-001-login-api"
  },
  "comprehension": {
    "profile_version": "comp-20260503-001",
    "mode": "incremental"
  },
  "quality_gates": {
    "status": "passed",
    "review_decision": "accepted"
  },
  "commands": [],
  "tests": [],
  "risks": [],
  "memory_candidates": []
}
```

## 7. 配置归属原则

- 只能有一个权威文档展开某类配置。
- `project-workspace-spec.md` 只维护目录和 schema 索引。
- CLI、MVP 和 Phase 只在 [总体规划与生命周期路线图](./lifecycle-roadmap.md) 维护。
- Memory 机制只在 [Agent Memory 系统方案](./agent-memory-system.md) 展开。
- 仓库接入只在 [仓库接入与 Git Provider Adapter](./repository-onboarding-git-management.md) 展开，项目阅读理解只在 [项目接入与阅读理解主线](./mainlines/project-comprehension.md) 展开。
