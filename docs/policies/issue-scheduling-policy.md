# Issue 调度策略

## 1. 目标

决定用户需求是否需要澄清、如何拆分为 issues、如何构建 Issue Graph、哪些 issue 可以并发、哪些必须等待，以及每个 issue 应创建哪些 Subagent、绑定哪些 skills、由哪个 Agent Runtime 执行。

Issue 字段、命名、粒度和 accepted 条件由 [工程流程规范](../engineering-process-standards.md) 维护。

## 2. 输入事实

- 用户原始需求。
- clarified requirement。
- 项目画像和模块地图。
- 历史决策和 memory。
- issue 类型、依赖、读写范围。
- Subagent 并发策略。
- Skill Registry 和已启用 bindings。
- Runtime 健康状态。
- worktree 可用数量。
- 模型预算。
- 权限策略。
- 当前 running issues。

## 3. 决策结果

- `NEEDS_CLARIFICATION`
- `CREATE_ISSUE_GRAPH`
- `ISSUE_BLOCKED`
- `ISSUE_READY`
- `ISSUE_RUNNING`
- `SCHEDULE_PARALLEL`
- `SCHEDULE_SERIAL`
- `CREATE_SUBAGENT_PLAN`
- `SUBAGENT_BLOCKED`
- `REPLAN_REQUIRED`
- `ISSUE_SPEC_INVALID`

## 4. 澄清决策树

```text
if acceptance criteria cannot be verified:
  NEEDS_CLARIFICATION
else if multiple mutually exclusive implementations exist:
  NEEDS_CLARIFICATION
else if destructive data migration is possible:
  NEEDS_CLARIFICATION
else if auth/payment/security/production deploy is involved:
  NEEDS_CLARIFICATION
else if project facts can safely fill missing detail:
  CREATE_ISSUE_GRAPH
else:
  CREATE_ISSUE_GRAPH
```

如果生成的 issue 缺少 clarified requirement、acceptance criteria、test plan、write scope 或 rollback_or_fix_plan，则返回 `ISSUE_SPEC_INVALID`，不能进入 ready queue。

## 5. 并发决策树

```text
parallelism = min(
  max_parallel_issues,
  ready_issue_count,
  available_worktrees,
  runtime_slots,
  model_budget_slots,
  non_conflicting_write_sets
)
```

```text
subagent_parallelism = min(
  max_parallel_subagents,
  ready_subagent_count,
  runtime_slots,
  model_budget_slots,
  non_conflicting_write_sets
)
```

```text
if issue has unmet hard dependency:
  ISSUE_BLOCKED(waiting_dependency)
else if API/schema/UI contract is required and not accepted:
  ISSUE_BLOCKED(waiting_contract)
else if write scope conflicts with running issue:
  ISSUE_BLOCKED(resource_conflict)
else if runtime unhealthy or no slot:
  ISSUE_BLOCKED(waiting_runtime_slot)
else if dirty worktree blocks branch creation:
  ISSUE_BLOCKED(waiting_worktree)
else:
  ISSUE_READY
```

## 6. Subagent 计划树

```text
if issue is not ready:
  do not create subagent
else if role cannot be resolved:
  SUBAGENT_BLOCKED(role_unresolved)
else if required skill is incompatible:
  SUBAGENT_BLOCKED(skill_incompatible)
else if write scope unsafe:
  SUBAGENT_BLOCKED(scope_unsafe)
else:
  CREATE_SUBAGENT_PLAN
```

## 7. Runtime 分派树

```text
if issue.type == frontend:
  preferred_runtime = claude_cli
else if issue.type == backend:
  preferred_runtime = codex_cli
else if issue.type == backend_tuning:
  preferred_runtime = codex_cli
else if issue.type == test:
  preferred_runtime = codex_cli
else if issue.type == design or contract:
  preferred_runtime = claude_cli
else:
  preferred_runtime = codex_cli

if preferred runtime unavailable and fallback allowed:
  use fallback runtime
else if preferred runtime unavailable:
  ISSUE_BLOCKED(runtime_unavailable)
```

## 8. 阻断条件

- 用户澄清未完成。
- 上游 issue 未 accepted。
- 写入范围冲突。
- 生产、数据库、安全变更缺少审批。
- Runtime 不健康。
- Subagent role、skill 或输出契约无法解析。
- worktree 不可用。
- 预算不足。
- 权限策略拒绝。

## 9. 人工确认条件

- 需求范围影响生产。
- 涉及破坏性数据库迁移。
- 需要新增第三方依赖。
- 需要扩大写入范围。
- 需要执行高风险命令。
- 多次 replan 仍无法解除阻塞。

## 10. 产物和日志

产物：

- `.moyuan/lifecycle/epics/`
- `.moyuan/lifecycle/issues/`
- `.moyuan/lifecycle/issue-graphs/`
- `.moyuan/lifecycle/schedules/`
- `.moyuan/agents/subagents/`

日志：

- `run`
- `agent`
- `model`
- `audit`
- `error`

## 11. 关联配置

- `.moyuan/policies/orchestration.yaml`
- `.moyuan/runtimes/agent-runtimes.yaml`
- `.moyuan/agents/roles.yaml`
- `.moyuan/agents/teams.yaml`
- `.moyuan/agents/subagents.yaml`
- `.moyuan/skills/registry.yaml`
- `.moyuan/skills/bindings.yaml`
- `.moyuan/policies/budget.yaml`
- [工程流程规范](../engineering-process-standards.md)

## 12. 验收用例

- API 契约未 accepted 时，前后端实现不启动。
- Issue 缺少必填字段时不能进入 ready queue。
- 前后端写入范围不冲突时，可以在契约 accepted 后并发。
- 后端默认使用 Codex CLI。
- 前端默认使用 Claude CLI。
- ready issue 能生成 Subagent Plan。
- Skill 不兼容时不创建 Subagent。
- dirty worktree 时不启动自动写入。
