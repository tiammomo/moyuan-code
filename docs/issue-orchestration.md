# Issues 编排与并发调度

本文定义从用户开发目标到 Issue Graph、ready queue、Subagent 调度、质量复核和 integration branch 合入的编排规则。

权威边界：

| 内容 | 权威文档 |
| --- | --- |
| Issue 字段、命名、commit、fix、release、coverage | [工程流程规范](./engineering-process-standards.md) |
| Subagent 生命周期、输出汇聚和 Skill 绑定 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| 质量门禁、重复度、复杂度和 review | [代码生命周期质量门禁](./code-lifecycle-quality-gates.md) |
| Git 分支、worktree 和用户改动保护 | [Git 分支策略](./policies/git-branch-policy.md) |
| 发布批次、投产、服务器、冒烟和监控 | [DevOps 发布投产主线](./mainlines/devops-release-deployment.md)、[发布投产策略](./policies/release-deployment-policy.md) |
| 配置字段必填、可空和必须为空 | [配置 Schema 规则](./configuration-schema-spec.md) |

## 1. 目标

用户提出开发需求后，系统必须先理解和规划，再开发。

编排目标：

- 基于项目画像和 Memory 丰富需求描述。
- 判断是否需要意图澄清。
- 将完善后的需求拆成可执行 issues。
- 构建用户可见的 Issue Graph。
- 根据依赖、写入范围、风险、worktree、Runtime 和预算自动决定并发度。
- 为每个 issue 创建显式 Subagent Plan。
- 每个 issue 完成后经过质量门禁和独立 review。
- 复核通过后才合入 epic integration branch。
- 下游 issue 只在前置依赖满足后解锁。

## 2. 端到端流程

```text
User Request
  -> Requirement Refiner
  -> Clarification Gate
  -> Project Context Assembly
  -> Issue Planner
  -> Dependency Planner
  -> Scheduler
  -> User-visible Issue Graph
  -> Subagent Dispatch
  -> Quality Gate / Review
  -> Merge to Epic Integration Branch
  -> Unlock Downstream Issues
```

## 3. 需求完善与澄清

Requirement Refiner 必须补齐：

- 背景：为什么做。
- 目标：完成后用户能做什么。
- 范围：包含什么、不包含什么。
- 约束：技术、权限、兼容性、性能、风格。
- 验收：可验证的成功条件。
- 风险：影响模块、数据、安全、发布。

Clarification Gate 必须追问的情况：

- 目标不可验证。
- 存在多个互斥实现方向。
- 涉及破坏性数据迁移、鉴权、支付、安全、生产投产或版本发布。
- 需要用户选择兼容策略、交互行为或审批边界。
- 项目画像、Memory 和代码事实无法可靠补齐关键缺口。

## 4. Issue Graph

Issue Graph 是 issues 的依赖 DAG。边表示前置依赖、契约依赖、代码依赖、测试依赖、资源冲突或 review 依赖。

```text
issue-a api_contract
  -> issue-b backend implementation
  -> issue-c frontend integration
  -> issue-d integration tests
```

规则：

- `issue-b` 和 `issue-c` 只有在 `issue-a` accepted 后才能启动。
- `issue-d` 必须等待后端和前端都 accepted。
- 写入范围冲突会把可并行 issue 改为串行。
- 下游 issue 不能绕过上游质量门禁。

Issue 最小内容以 [工程流程规范](./engineering-process-standards.md) 为准。编排层额外要求：

```yaml
id: issue-001
type: backend | frontend | design | test | quality | security | release
depends_on: []
dependency_type: blocks | data_contract | code_dependency | test_dependency | review_dependency
read_scopes: []
write_scopes: []
assigned_team: feature_team
subagent_plan_ref: .moyuan/agents/subagents/issue-001.yaml
blocked_reason: null
```

## 5. 并发决策

Scheduler 不固定并发数，按可用条件动态计算。

```text
parallelism = min(
  project_policy.max_parallel_issues,
  ready_issue_count,
  available_worktrees,
  model_budget_slots,
  runtime_slots,
  non_conflicting_write_sets
)
```

Subagent 并发度：

```text
subagent_parallelism = min(
  project_policy.max_parallel_subagents,
  ready_subagent_count,
  issue_parallelism_budget,
  available_runtime_slots,
  available_worktrees,
  model_budget_slots
)
```

阻止并发：

- 两个 issue 写同一文件或同一核心模块。
- 涉及数据库迁移、鉴权、支付、安全、公共 API 或基础类型变更。
- 上游设计或契约未 accepted。
- dirty worktree。
- quality gate 或 review 未通过。
- Runtime、预算或 worktree 不可用。

允许并发：

- 写入范围无交集。
- 前后端依赖同一已确认接口契约。
- 多个测试 issue 覆盖不同模块。
- 文档、测试、低风险重构互不冲突。

## 6. 队列与等待

Scheduler 维护四类队列：

```text
blocked_queue
ready_queue
running_queue
review_queue
subagent_backlog
```

`BLOCKED` 必须记录原因：

| blocked reason | 解除条件 |
| --- | --- |
| `waiting_contract` | 契约 issue accepted |
| `waiting_backend` | 后端 accepted 或 mock ready |
| `waiting_frontend_decision` | 前端/design issue accepted |
| `waiting_runtime_slot` | Runtime slot 可用 |
| `waiting_worktree` | worktree 可用 |
| `waiting_quality` | quality passed |
| `waiting_review` | review accepted |
| `waiting_user_input` | 用户补充或批准 |
| `dirty_worktree` | 用户处理本地改动 |
| `subagent_waiting_runtime` | Runtime 恢复或 provider 可用 |
| `subagent_needs_rework` | 返工 issue 重新规划或用户确认 |
| `subagent_retry_exhausted` | 人工复核后创建新 run 或调整 retry policy |

Phase 2 当前实现中，Scheduler 会读取 Subagent 的 retry/archive 状态：

- `archived + retry_count >= max_retries`：进入 waiting，原因 `subagent_retry_exhausted`。
- `waiting_runtime`：进入 waiting，原因 `subagent_waiting_runtime`。
- `needs_rework`：进入 waiting，原因 `subagent_needs_rework`。
- `retrying`：保留 retry metadata，继续按 runtime slot 和写入范围调度。

Phase 11 到 Phase 12 当前实现中，Orchestrator 已新增 batch dispatch preview、受控 batch run、issue worktree isolation 和 integration merge preview：

- `batch_plan` 基于 Scheduler 的 dispatch/waiting/backlog 结果生成，不运行 Runtime。
- 每个 dispatch/waiting issue 记录 role、runtime、write scopes、dependency ids、conflict reason 和 provider route preview。
- blocked issue 保留 dependency reason。
- batch plan 输出到 `.moyuan/orchestrator/batches/` 和 `.moyuan/orchestrator/batches.jsonl`。
- `batch_run` 必须基于已生成的 batch plan，输出到 `.moyuan/orchestrator/batch-runs/` 和 `.moyuan/orchestrator/batch-runs.jsonl`。
- batch run 默认 `dry_run`，只记录将要执行的 issue，不运行 Runtime、不修改 issue 状态。
- 真实 `local_shell` batch run 需要 `approved=true`、`MOYUAN_ALLOW_BATCH_RUN=1`，并且 prompt 只能使用受控安全前缀。
- 真实 `local_shell` batch run 会为每个 issue 创建独立 Git worktree 和 branch，Runtime、diff capture 和 quality check 都在 issue worktree 内执行。
- worktree 记录输出到 `.moyuan/orchestrator/worktrees/` 和 `.moyuan/orchestrator/worktrees.jsonl`，实际 worktree 位于 `.moyuan/worktrees/`。
- batch executor 支持受控 bounded parallelism，`RunRecord.parallelism` 和 `RunItem.worker_slot` 记录真实 worker 槽位；默认未显式提高 `max_issues` 时仍保持保守执行。
- `merge_queue` 基于 batch run 聚合每个 issue 的 quality report、review status 和 merge decision，输出 ready、needs_rework、blocked 三类队列。
- merge queue 输出到 `.moyuan/lifecycle/merge-reports/queues/` 和 `.moyuan/lifecycle/merge-reports/merge-queues.jsonl`。
- `integration_preview` 基于 ready merge queue 创建独立 integration worktree，执行 merge dry-run、冲突检测和 protected path guard，输出到 `.moyuan/lifecycle/merge-reports/integration-previews/` 和 `.moyuan/lifecycle/merge-reports/integration-previews.jsonl`。
- 当前 integration preview 不执行真实合入、PR/MR、tag、push 或 publish。

## 7. 前端 Runtime 选择 / 后端 Codex

默认分工：

| Issue 类型 | 默认 Role | 默认 Runtime | 等待条件 |
| --- | --- | --- | --- |
| `frontend` | `frontend` | `claude_cli` 或 `codex_cli` | UI 需求、接口契约、写入范围 ready |
| `backend` | `backend` | `codex_cli` | API 契约、数据模型、写入范围 ready |
| `backend_tuning` | `backend_tuning` | `codex_cli` | 性能目标、基线指标、测试命令 ready |
| `contract` | `architect` | `claude_cli` 或 `codex_cli` | 用户需求和项目理解 ready |
| `test` | `tester` | `codex_cli` | 被测实现 accepted |
| `review` | `reviewer` | `codex_cli` | diff 和 quality report ready |

前端 Runtime 选择规则：

- 复杂 UI 首版、交互探索、视觉系统调整优先 `claude_cli`。
- 样式基线已经稳定后的组件修改、状态修复、测试补齐和重构可以使用 `codex_cli`。
- 无论使用哪个 Runtime，前端 issue 都必须回到同一套 typecheck、build、UI smoke、quality gate 和 review。

前后端合流：

```text
api contract accepted
  -> backend issue and frontend issue may run in parallel
  -> backend accepted + frontend accepted
  -> merge both into epic integration branch
  -> integration checks
  -> unlock integration test issue
```

合流失败时，生成 integration fix issue 或把相关 issue 标记为 `NEEDS_REWORK`，下游测试和 release 必须等待。

## 8. Subagent 调度

Issue 被调度时先生成 Subagent Plan，再调用 Runtime。

```text
Ready Issue
  -> resolve roles
  -> resolve skills
  -> assemble memory scope
  -> create subagent instances
  -> dispatch runtime
  -> validate output contract
  -> quality gate / review
  -> issue accepted or needs_rework
```

典型映射：

| Issue 类型 | 默认 Subagent |
| --- | --- |
| discovery | `project_reader`、`module_mapper` |
| design | `requirement_refiner`、`architect` |
| backend | `backend`、`tester`、`quality_guard`、`reviewer` |
| frontend | `frontend`、`tester`、`quality_guard`、`reviewer` |
| quality | `quality_guard`、`repair_agent`、`reviewer` |
| release | `release_manager`、`tester`、`reviewer` |

Subagent 不能自行无限拆分任务。需要继续拆分时，必须把建议交回 Orchestrator 更新 Issue Graph。

## 9. 状态机

Issue 状态：

```text
CREATED
  -> PLANNED
  -> BLOCKED
  -> READY
  -> RUNNING
  -> QUALITY_CHECKING
  -> VERIFYING
  -> REVIEWING
  -> ACCEPTED
  -> MERGED
  -> DONE
```

失败路径：

```text
NEEDS_REWORK | FAILED | CANCELLED
```

Epic 状态由 issue graph 汇总：

- 全部 `DONE`：epic completed。
- 存在 `FAILED`：epic failed 或需要 replan。
- 存在 `BLOCKED`：等待依赖、资源或用户输入。
- 存在 `NEEDS_REWORK`：暂停受影响下游。

完整状态来源见 [状态机总表](./foundations/state-machine-catalog.md)。

## 10. 合入门禁

issue branch 合入 epic integration branch 前必须满足：

- acceptance criteria 全部通过。
- build/test/lint/typecheck 通过。
- coverage、重复度、复杂度和架构边界通过。
- `quality_guard` accepted。
- `reviewer` accepted。
- 没有未解决 blocker/high findings。
- 没有写入范围冲突。

合入后：

```text
issue accepted
  -> merge issue branch into epic integration branch
  -> run integration checks
  -> update issue graph
  -> unlock downstream issues
  -> recalculate ready queue and parallelism
```

## 11. 配置入口

`.moyuan/policies/orchestration.yaml` 只保留编排开关和限制；字段规则见 [配置 Schema 规则](./configuration-schema-spec.md)。

```yaml
schema_version: 1
orchestration:
  enabled: true
  issue_graph: true
  auto_parallelism: true
  max_parallel_issues: 3
  max_parallel_subagents: 4
  require_clean_worktree: true
  use_epic_integration_branch: true
  use_issue_worktrees: true
  concurrency_guards:
    disallow_same_file_writes: true
    disallow_same_module_core_writes: true
    serialize_database_migrations: true
    serialize_auth_security_payment: true
    require_design_acceptance_for_public_api: true
  graph_visibility:
    write_user_visible_graph: true
    include_blocked_reason: true
    include_parallelism_plan: true
  merge_gate:
    require_quality_passed: true
    require_review_accepted: true
    require_style_check: true
    require_integration_checks: true
```

## 12. Workspace 产物

```text
.moyuan/lifecycle/
  epics/
  issues/
  issue-graphs/
  schedules/
  runs/
```

Issue graph 最小结构：

```json
{
  "epic_id": "epic-20260503-001",
  "nodes": [{"id": "issue-001", "status": "accepted"}],
  "edges": [{"from": "issue-001", "to": "issue-002", "type": "blocks"}],
  "ready_queue": ["issue-002"],
  "running": [],
  "blocked": []
}
```

## 13. 发布衔接

Issue 编排只负责把 accepted issues 合入 epic integration branch，并在达到条件时生成 release suggestion。

发布批次、release branch、tag、PR/MR、GitHub/Gitee 发布、服务器部署、线上冒烟、监控窗口、回滚和复盘由 [DevOps 发布投产主线](./mainlines/devops-release-deployment.md)、[发布投产策略](./policies/release-deployment-policy.md) 和 [工程流程规范](./engineering-process-standards.md) 维护。

## 14. 验收标准

- 用户输入开发目标后，系统能生成 clarified requirement。
- 需要澄清时不会直接进入开发。
- 系统能生成用户可见 Issue Graph。
- Scheduler 能计算 ready queue、blocked reason 和并发度。
- 依赖未完成的 issue 不会启动。
- 并发 issue 不会写同一文件或冲突模块。
- 每个 issue 都经过质量门禁和 review。
- 复核通过后才合入 epic integration branch。
- 上游合入后自动解锁下游 issue。
