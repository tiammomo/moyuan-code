# 代码开发主线

## 1. 目标

代码开发主线负责消费已经通过规划和调度的 ready issues，创建显式 Subagent，调度 Claude CLI、Codex CLI 和相关 Agent 完成代码实现、测试、质量复核和返工。

需求完善、意图澄清、Issue Graph 和并发计划由 [需求规划与 Issue 编排主线](./requirement-planning.md) 负责。代码开发主线不重新拆分需求，只执行已经进入 `ready_queue` 的 issue。

默认分工：

- 前端开发默认使用 `frontend` role。复杂 UI 首版和设计探索可优先 `claude_cli`；样式基线稳定后的前端代码修改、测试、修复和重构可使用 `codex_cli`。
- 后端开发优先使用 `backend` role 和 `codex_cli`。
- 后端调优优先使用 `backend_tuning` role 和 `codex_cli`。
- 复杂契约、架构方案和跨端依赖必须在进入本主线前形成 design/contract issue 或 accepted contract。
- 当项目启用 `minimax-m27-claude` 等绑定 `claude_cli` 的 provider profile 时，前端 issue 可通过 Claude CLI 使用 MiniMax-M2.7；当 issue 更偏工程修改、测试修复或重构时，也可以路由到 Codex CLI。Moyuan 仍负责 provider route、diff 捕获、质量门禁和合入判断。

## 2. 输入与输出

输入：

- ready issue。
- issue graph 和 schedule。
- 最新项目画像、模块地图和历史决策。
- 当前 Git 状态、分支策略和写入权限。
- Agent role、team、Subagent plan、runtime、provider routing 和 skills。

输出：

- issue worktrees。
- subagent instances。
- code diff。
- test report。
- quality report。
- review report。
- accepted issue 或 rework issue。

## 3. 端到端流程

```text
ready issue
  -> load issue spec and accepted contracts
  -> load project context and scoped memory
  -> select role, team, skills, runtime and provider
  -> inject provider env profile into native runtime
  -> create subagent instances
  -> prepare issue branch/worktree
  -> dispatch Claude/Codex runtime
  -> collect subagent outputs, diff and command results
  -> validate subagent output contracts
  -> run local checks
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
- [Git 分支策略](../policies/git-branch-policy.md)

核心决策：

- 当前 issue 应使用 Claude CLI、Codex CLI 还是普通模型 API。
- 当前 issue 应创建哪些 Subagent，以及是否允许并发。
- 当前 issue 应绑定哪些 skills。
- 当前 issue 是否满足启动条件。
- Agent 生成的 diff 是否可接受。
- 质量失败后返工几轮，什么时候升级人工处理。
- 代码是否允许进入合入门禁。

## 5. 启动条件

Issue 进入代码开发主线前必须满足：

- 上游 hard dependency 已 accepted 或 merged。
- API/schema/UI 契约已 accepted。
- Issue 已进入 `ready_queue`。
- Issue 已来自已生成的 issue graph，且 issue graph 本身已接受。
- 写入范围不与 running issue 冲突。
- Runtime 健康且有可用 slot。
- Subagent role、skills、scope 和输出契约可解析。
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
- `.moyuan/agents/subagents.yaml`
- `.moyuan/skills/registry.yaml`
- `.moyuan/skills/bindings.yaml`
- `.moyuan/runtimes/agent-runtimes.yaml`
- `.moyuan/models/routing.yaml`
- `.moyuan/policies/orchestration.yaml`
- `.moyuan/policies/code-quality.yaml`
- `.moyuan/policies/permissions.yaml`
- `.moyuan/policies/budget.yaml`

## 8. Workspace 产物

```text
.moyuan/lifecycle/
  issues/
  quality/
  reviews/
  runs/
```

## 9. 日志与审计

必须记录：

- clarified requirement。
- issue id 和 issue spec 版本。
- assigned role/runtime/provider。
- worktree path 和 branch。
- changed files。
- commands、tests、quality gates。
- provider telemetry feedback：`runtime_execution`、`quality_gate`、`provider_route`。
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

- 只执行 `ready_queue` 中的 issue。
- 依赖未满足的 issue 不会进入本主线。
- 前后端实现都基于 accepted contract。
- 每个代码 issue 都经过质量门禁和独立 review。
- 复核未通过不会合入。
- 失败会生成 rework 或 replan，而不是继续推进下游。
