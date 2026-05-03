# Subagent 与 Skills 系统方案

本文定义 Moyuan Code 中 Subagent 和 Skill 的显式设计。它不是 Agent 角色表的补充说明，而是后续实现多 Agent 作业、动态委派、能力复用和持续优化的权威方案。

## 1. 目标

- 把 Subagent 从隐式 Runtime 执行实例提升为可配置、可观测、可审计的对象。
- 让 Orchestrator 能按 Issue Graph、风险、写入范围、技能需求和资源状态动态创建 Subagent。
- 让 Skills 成为可发现、可评分、可绑定、可版本化和可复盘的能力组件。
- 让项目越使用，Subagent 分工、skills 推荐、模型路由和质量规则越准确。
- 保证所有 Subagent 输出都回到统一质量门禁、review、Memory 和日志链路。

## 2. 核心概念

### Agent Role

Agent Role 是职责模板，例如 `backend`、`frontend`、`tester`、`reviewer`、`bug_triager`。

Role 定义：

- 默认模型策略。
- 默认 Runtime。
- 可用工具。
- 默认 skills。
- Memory scope。
- 输出契约。

### Subagent

Subagent 是 Orchestrator 为一个具体任务创建的 Agent 执行实例。

Subagent 不是模型，也不是长期角色。它是一次受控委派，具备：

- 明确父任务或父 Run。
- 明确 role。
- 明确 issue、worktree、读写范围。
- 明确 runtime、skills、memory scope 和工具权限。
- 明确输出契约和完成条件。
- 明确生命周期、失败处理和审计记录。

### Skill

Skill 是可复用能力单元，包含任务流程、提示模板、工具使用规范、检查清单、领域知识或外部工具适配说明。

Skill 不直接执行任务。Skill 只能被 Role 或 Subagent 引用，再由 Runtime 执行。

## 3. Subagent 类型

| 类型 | 说明 | 是否可写代码 |
| --- | --- | --- |
| planning_subagent | 需求澄清、拆分、依赖规划、风险判断 | 否 |
| discovery_subagent | 项目阅读理解、模块定位、技术栈识别 | 否 |
| implementation_subagent | 前端、后端、调优、测试补齐 | 是 |
| verification_subagent | 测试、质量门禁、review、安全检查 | 默认否 |
| repair_subagent | 低风险 confirmed bug 修复 | 是 |
| release_subagent | release note、版本分支、tag、PR/MR、部署检查 | 条件允许 |
| memory_subagent | memory candidate、compact、去重、过期、冲突处理 | 否 |

## 4. Subagent 创建流程

```text
Issue Graph / Runtime Signal / Release Task
  -> resolve required roles
  -> resolve candidate skills
  -> check auth_context and policy
  -> compute read_scope / write_scope
  -> select runtime and model policy
  -> assemble memory scope
  -> create subagent instance
  -> execute in isolated run
  -> validate output contract
  -> quality gate / review / merge decision
  -> record memory and skill effectiveness
```

## 5. 委派决策

Orchestrator 只有在满足以下条件时才创建 Subagent：

- 任务能被明确描述为一个可验收工作单元。
- 输入上下文足够，或可以由 discovery/planning subagent 先补充。
- role 和 skills 可解析。
- `auth_context` 有效。
- 文件读写范围明确。
- Runtime 健康。
- 预算、并发槽位和 worktree 可用。
- 没有未满足的硬依赖、审批或用户澄清。

不能创建 Subagent 的情况：

- 用户目标不可验证。
- 写入范围无法限制。
- 需要未授权 secret 或生产数据。
- 目标依赖未完成的高风险设计。
- 当前 issue 处于 blocked、cancelled 或 failed 且没有恢复策略。

## 6. Subagent 生命周期

```text
planned -> context_assembled -> dispatched -> running -> output_collected -> validated -> completed -> archived
```

失败或返工状态：

```text
blocked
failed
timeout
needs_rework
needs_user_input
cancelled
superseded
```

状态变化必须写入：

- `agent` log。
- `run` log。
- `audit` log，若涉及写权限、审批、密钥引用或 protected path。
- `error` log，若失败。

## 7. 父子关系

Subagent 必须挂在以下父对象之一：

- Epic。
- Issue。
- Run。
- Repair Attempt。
- Release。
- Deployment。
- Memory Maintenance Job。

父对象负责：

- 提供目标和验收标准。
- 提供依赖状态。
- 提供权限边界。
- 接收 Subagent 输出。
- 决定是否进入下一阶段。

Subagent 不能自行创建无限层级子任务。需要继续拆分时，必须把拆分建议交回 Orchestrator，由 Orchestrator 更新 Issue Graph 或创建新的 Subagent。

## 8. 并发控制

Subagent 并发度由 Scheduler 统一计算：

```text
subagent_parallelism = min(
  policy.max_parallel_subagents,
  ready_subagent_count,
  issue_parallelism_budget,
  available_runtime_slots,
  available_worktrees,
  model_budget_slots,
  non_conflicting_write_sets
)
```

默认禁止并发：

- 两个 Subagent 写同一文件。
- 两个 Subagent 写同一模块核心入口。
- 涉及数据库迁移、鉴权、安全、支付或公共 API。
- 其中一个 Subagent 需要等待另一个输出契约。
- 任一 Subagent 的质量门禁或 review 未通过。

允许并发：

- discovery 和 planning 只读任务。
- 不同模块的测试补齐。
- 契约 accepted 后的前端和后端实现。
- 文档、测试、低风险修复且写入范围不冲突。

## 9. Skill Registry

Skill Registry 负责保存可用 skills、来源、版本、能力标签、适配 role、风险和效果数据。

每个 Skill 必须声明：

- `id`
- `name`
- `version`
- `source`
- `description`
- `supported_roles`
- `task_types`
- `required_tools`
- `memory_scopes`
- `risk_level`
- `input_contract`
- `output_contract`
- `validation`

Skill 来源：

- 内置 skill。
- 项目本地 skill。
- 组织共享 skill。
- 外部 marketplace 或 `find-skills` 推荐结果。
- 用户手动绑定 skill。

## 10. Skill 推荐流程

```text
Project Comprehension
  -> stack detection
  -> module map
  -> task intent classification
  -> issue type and risk
  -> role requirements
  -> find-skills query
  -> candidate skill scoring
  -> compatibility check
  -> bind to role or subagent
  -> execute
  -> record effectiveness
```

评分维度：

| 维度 | 说明 |
| --- | --- |
| stack_match | 是否匹配技术栈 |
| role_match | 是否匹配 role |
| task_match | 是否匹配 issue type |
| tool_match | 需要的工具是否允许 |
| memory_match | 需要的 memory scope 是否可用 |
| risk_fit | 风险等级是否适合当前任务 |
| historical_success | 历史成功率 |
| quality_effect | 是否降低返工、重复代码和 bug |
| cost_effect | 是否显著增加成本或延迟 |

## 11. Skill 绑定规则

绑定层级从低到高：

```text
system default
  -> organization default
  -> project default
  -> role default
  -> issue override
  -> subagent runtime override
```

冲突处理：

- 高层级覆盖低层级。
- 高风险 skill 需要审批。
- 需要写权限的 skill 必须经过权限策略。
- 外部 skill 不能默认读取敏感代码或项目长期 Memory。
- 与项目规范冲突的 skill 不启用。

## 12. Skill 效果反馈

每次 Subagent 执行完成后，系统要记录 Skill 使用效果：

- 是否帮助任务完成。
- 是否减少返工。
- 是否降低重复代码或复杂度。
- 是否补齐测试。
- 是否产生 bug。
- 是否被 reviewer 或 quality_guard 否决。
- 是否需要版本升级、禁用或降权。

效果记录进入：

- `.moyuan/skills/effectiveness/`
- `.moyuan/memory/candidates/`
- `Improvement Record`，如果是能力增强建议。

## 13. 输出收敛

多个 Subagent 的输出不能直接拼接合入。必须由 Orchestrator 收敛：

```text
collect subagent outputs
  -> validate output contracts
  -> compare changed files
  -> detect conflicts
  -> run quality gates
  -> request reviewer / quality_guard
  -> merge accepted output
  -> update issue / run / memory
```

当输出冲突时：

- 写入冲突：阻断后续合入，要求 rebase、重新执行或人工处理。
- 需求冲突：交回 planner 或用户澄清。
- 质量冲突：交给 quality_guard 生成返工项。
- 架构冲突：交给 architect 判断。

## 14. 与 Memory 的关系

Subagent 执行前只检索与 role 和 task 相关的 Memory，避免上下文污染。

Subagent 执行后只能产生 memory candidate，不能直接写入长期 Memory。长期 Memory 写入仍由 [Agent Memory 系统方案](./agent-memory-system.md) 的 Record Gate、抽取、暂存去重、compact 和维护流程决定。

可写入候选：

- 成功的实现经验。
- 失败的误区。
- bug signature。
- fix pattern。
- 测试命令。
- 模块边界事实。
- 用户偏好。
- skill 效果。

## 15. 安全边界

Subagent 必须遵守：

- 继承 `auth_context`。
- 不能自行提升权限。
- 不能读取未授权 secret。
- 不能修改 protected paths。
- 不能直接 push、merge、deploy。
- 不能绕过质量门禁。
- 不能把完整项目 Memory dump 发给低信任 Provider。
- 不能把外部 skill 当成可信代码执行。

## 16. 配置位置

| 配置 | 作用 |
| --- | --- |
| `.moyuan/agents/roles.yaml` | Agent Role 定义 |
| `.moyuan/agents/teams.yaml` | Team 与验证链路 |
| `.moyuan/agents/subagents.yaml` | Subagent 创建、并发、生命周期和委派策略 |
| `.moyuan/skills/enabled.yaml` | 项目启用 skills |
| `.moyuan/skills/registry.yaml` | skill registry 索引 |
| `.moyuan/skills/effectiveness/` | skill 使用效果 |
| `.moyuan/policies/orchestration.yaml` | issue/subagent 并发和等待规则 |
| `.moyuan/runtimes/agent-runtimes.yaml` | Runtime 调用和隔离 |

## 17. 验收标准

- 一个 Epic 能生成多个 Subagent 执行实例。
- Subagent 有独立生命周期、日志、输出契约和父对象关联。
- Orchestrator 能根据依赖、写入范围、Runtime 和预算决定 Subagent 并发度。
- Skills 能被发现、评分、绑定、执行和复盘。
- Skill 效果能影响后续推荐、降权或禁用。
- Subagent 输出必须经过质量门禁和 review 后才能合入。
- Subagent 不能绕过权限、审批、Git、Memory 和发布策略。
