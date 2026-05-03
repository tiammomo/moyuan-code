# 代码开发主线

## 1. 目标

代码开发主线负责把用户提出的开发需求转为可执行 Issue Graph，并调度多个 Agent 完成开发、测试、质量复核和返工。

默认分工：

- 前端开发优先使用 `frontend` role 和 `claude_cli`。
- 后端开发优先使用 `backend` role 和 `codex_cli`。
- 后端调优优先使用 `backend_tuning` role 和 `codex_cli`。
- 复杂契约、架构方案和跨端依赖先由 planner/architect 产出设计。

## 2. 输入与输出

输入：

- 用户原始开发需求。
- 最新项目画像、模块地图和历史决策。
- 当前 Git 状态、分支策略和写入权限。
- Agent role、team、runtime、provider routing 和 skills。

输出：

- clarified requirement。
- clarification decision。
- issue graph。
- schedule。
- issue worktrees。
- code diff。
- test report。
- quality report。
- review report。
- accepted issue 或 rework issue。

## 3. 端到端流程

```text
user request
  -> requirement refiner
  -> clarification gate
  -> issue planner
  -> dependency planner
  -> scheduler
  -> user-visible issue graph
  -> dispatch ready issues
  -> run Claude/Codex agents
  -> collect diff and command results
  -> quality gates
  -> independent review
  -> rework or accepted
  -> merge gate
```

## 4. 关键决策点

调用策略：

- [Issue 调度策略](../policies/issue-scheduling-policy.md)
- [质量与合入策略](../policies/quality-merge-policy.md)
- [Provider 路由策略](../policies/provider-routing-policy.md)
- [Memory 决策策略](../policies/memory-decision-policy.md)

核心决策：

- 是否需要向用户澄清。
- 如何拆分 issues。
- 哪些 issue 可以并发，哪些必须等待。
- 当前 issue 应使用 Claude CLI、Codex CLI 还是普通模型 API。
- 质量失败后返工几轮，什么时候升级人工处理。
- 代码是否允许进入合入门禁。

## 5. 调度队列

```text
blocked_queue
ready_queue
running_queue
review_queue
```

Issue 进入 `ready_queue` 的条件：

- 上游 hard dependency 已 accepted 或 merged。
- API/schema/UI 契约已 accepted。
- 写入范围不与 running issue 冲突。
- Runtime 健康且有可用 slot。
- worktree、预算、权限和模型路由满足要求。
- 没有等待用户澄清或审批。

## 6. 质量要求

所有代码 issue 必须通过：

- 可运行性检查。
- lint、format、typecheck、build。
- 单元测试或合理测试缺口说明。
- 重复代码检查。
- 复杂度检查。
- 架构边界检查。
- 依赖和安全检查。
- independent review。

详细规则见 [代码生命周期质量门禁](../code-lifecycle-quality-gates.md)。

## 7. 配置入口

- `.moyuan/agents/roles.yaml`
- `.moyuan/agents/teams.yaml`
- `.moyuan/runtimes/agent-runtimes.yaml`
- `.moyuan/models/routing.yaml`
- `.moyuan/policies/orchestration.yaml`
- `.moyuan/policies/code-quality.yaml`
- `.moyuan/policies/permissions.yaml`
- `.moyuan/policies/budget.yaml`

## 8. Workspace 产物

```text
.moyuan/lifecycle/
  epics/
  issues/
  issue-graphs/
  schedules/
  quality/
  reviews/
  runs/
```

## 9. 日志与审计

必须记录：

- clarified requirement。
- clarification decision。
- issue graph 版本。
- schedule 和 blocked reason。
- assigned role/runtime/provider。
- worktree path 和 branch。
- changed files。
- commands、tests、quality gates。
- review findings。
- rework decision。
- memory candidates。

日志流：

- `run`
- `agent`
- `model`
- `quality`
- `memory`
- `audit`
- `error`

## 10. 验收标准

- 用户需求可以自动变成可见 Issue Graph。
- 系统能自行判断并发度。
- 依赖未满足的 issue 不会启动。
- 前后端可以在契约 accepted 后并发。
- 每个代码 issue 都经过质量门禁和独立 review。
- 复核未通过不会合入。
- 失败会生成 rework 或 replan，而不是继续推进下游。
