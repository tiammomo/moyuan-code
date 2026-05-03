# 需求规划与 Issue 编排主线

## 1. 目标

需求规划与 Issue 编排主线负责把用户原始开发需求转成可执行、可审查、可调度的 Issue Graph。

这条主线独立出来的原因：

- 它发生在代码写入之前，决定后续开发是否可以开始。
- 它有独立产物：clarified requirement、clarification decision、issues、dependencies、schedule。
- 它有独立风险门禁：意图不清、验收不可验证、生产/安全/迁移风险未确认时必须阻断。
- 它决定并发拓扑，不应该和具体代码实现混在同一主线里。

## 2. 输入与输出

输入：

- 用户原始需求。
- 最新项目画像、模块地图和依赖图。
- 历史需求、用户偏好和项目决策 memory。
- 当前 Git 状态和写入范围约束。
- Agent role、skills、runtime 健康状态和预算。

输出：

- clarified requirement。
- clarification decision。
- issue list。
- issue dependency graph。
- acceptance criteria。
- read/write scopes。
- test plan。
- style constraints。
- schedule。
- blocked reason。
- user-visible execution plan。

## 3. 端到端流程

```text
user request
  -> retrieve project context and relevant memory
  -> requirement refiner
  -> clarification gate
  -> acceptance criteria generation
  -> issue planner
  -> dependency planner
  -> write scope analysis
  -> scheduler
  -> user-visible issue graph
  -> ready/blocked queue output
```

## 4. 主线判定策略

一条能力是否成为主线，需要满足至少三项：

- 有明确生命周期阶段，不只是一个工具或配置项。
- 有独立输入、输出和持久化产物。
- 有会阻断后续流程的关键决策点。
- 有独立责任角色或 owner。
- 会被多个横切能力引用。
- 出错后需要独立失败恢复路径。

按这个标准：

- 需求规划与 Issue 编排是主线。
- Memory 不是主线，是横切能力。
- Provider 路由不是主线，是策略能力。
- 日志不是主线，是审计基础设施。
- gpt-image-2 架构图不是主线，是辅助可视化能力。

## 5. 决策点

调用策略：

- [Issue 调度策略](../policies/issue-scheduling-policy.md)
- [项目阅读理解策略](../policies/project-comprehension-policy.md)
- [Provider 路由策略](../policies/provider-routing-policy.md)
- [Memory 决策策略](../policies/memory-decision-policy.md)

核心决策：

- 是否需要向用户澄清。
- 是否可以基于项目事实自动补全缺失信息。
- 是否需要先新增 design/contract issue。
- 如何拆分 issue 粒度。
- 哪些 issue 是 hard dependency。
- 哪些 issue 可以并发。
- 哪些 issue 因用户确认、权限、资源或契约缺失而 blocked。

## 6. 阻断条件

必须阻断并追问用户：

- 目标结果不可验证。
- 存在多个互斥实现方向。
- 涉及破坏性数据库迁移。
- 涉及鉴权、支付、安全、生产投产。
- 需要用户选择兼容策略或 UI 行为。
- 缺失信息不能从项目画像、memory 或代码事实中可靠补齐。

必须阻断并等待系统条件：

- 项目理解过期。
- 远程分支拉取后未完成 incremental comprehension。
- Git worktree 状态不可用。
- Runtime 健康检查失败。
- 权限策略拒绝读取必要上下文。

## 7. 配置入口

- `.moyuan/policies/orchestration.yaml`
- `.moyuan/policies/comprehension.yaml`
- `.moyuan/policies/memory.yaml`
- `.moyuan/agents/roles.yaml`
- `.moyuan/agents/teams.yaml`
- `.moyuan/runtimes/agent-runtimes.yaml`
- `.moyuan/models/routing.yaml`

## 8. Workspace 产物

```text
.moyuan/lifecycle/
  epics/
  issues/
  issue-graphs/
  schedules/
  clarification/
```

## 9. 日志与审计

必须记录：

- 原始需求摘要。
- 检索到的项目上下文和 memory scope。
- clarified requirement。
- clarification decision。
- 拆分后的 issue graph 版本。
- blocked reason。
- schedule 和并发计划。
- 人工确认问题和用户回答。

日志流：

- `run`
- `agent`
- `model`
- `memory`
- `audit`
- `error`

## 10. 验收标准

- 用户需求能转成可见 Issue Graph。
- 每个 issue 有验收标准、读写范围和测试计划。
- 前置依赖能被表达为 graph edge。
- 需要用户确认的风险不会进入开发。
- ready issue 和 blocked issue 可以被清晰区分。
- 后续代码开发主线只消费 ready issue，不重新做需求拆分。
