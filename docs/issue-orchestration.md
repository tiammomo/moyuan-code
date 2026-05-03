# Issues 编排与并发调度

## 1. 目标

用户提出一个开发任务后，系统需要自动拆分为多个可执行 issues，并根据依赖关系、代码写入范围、风险、资源和质量门禁决定哪些 issues 可以并发执行，哪些必须串行等待。

核心目标：

- 自动丰富任务需求描述，补齐背景、范围、约束、验收标准和风险。
- 判断是否需要向用户做意图澄清，关键不确定项未确认时不进入开发。
- 自动把需求拆成 issue graph。
- 将 issue graph、依赖关系、blocked reason 和并发计划展示给用户。
- 明确每个 issue 的前置依赖、验收标准、写入范围和测试要求。
- 自动判断并发执行数量。
- 为每个 issue 分配合适的 Agent team。
- 依赖未完成的 issue 不启动。
- 并发执行的 issue 必须避免写入冲突。
- 每个 issue 都经过质量门禁和独立 review。
- 每个 issue 必须满足项目代码风格和架构边界。
- 复核通过后才允许合入 integration branch。
- 下游 issue 在上游完成后自动解锁。

## 2. 核心抽象

### Epic

用户提出的原始开发目标。一个 Epic 会被拆成多个 issues。

### Issue

最小可执行开发单元。每个 issue 必须有：

- title。
- description。
- clarified requirement。
- type。
- priority。
- dependencies。
- read scopes。
- write scopes。
- acceptance criteria。
- test plan。
- assigned team。
- model policy。
- branch/worktree。
- quality gate。
- style constraints。

### Issue Graph

Issue 之间的有向无环图。边代表前置依赖。

```text
issue-a
  -> issue-b
  -> issue-d

issue-a
  -> issue-c
  -> issue-d
```

`issue-b` 和 `issue-c` 可以在 `issue-a` 完成后并发，`issue-d` 必须等待两者都完成。

## 3. Issue 类型

| Type | 说明 | 默认并发策略 |
| --- | --- | --- |
| discovery | 补充阅读理解、调研、定位模块 | 可并发 |
| design | 接口、数据模型、架构设计 | 通常串行 |
| backend | 后端实现 | 可并发，但需检查写入范围 |
| frontend | 前端实现 | 可并发 |
| migration | 数据库迁移、schema 变更 | 默认串行 |
| test | 测试补齐、回归测试 | 可并发 |
| quality | 质量修复、重复度/复杂度修复 | 依赖实现完成 |
| security | 权限、安全、敏感信息 | 默认串行或人工确认 |
| release | 版本分支、release note、部署投产、冒烟、监控、后续维护 | 最后执行 |

## 4. 自动拆分流程

```text
User Request
  -> Requirement Refiner 丰富任务描述
  -> Clarification Gate 判断是否需要追问用户
  -> Planner Agent 生成验收标准
  -> Project Reader 检索项目画像和模块地图
  -> Architect 判断模块边界和技术方案
  -> Issue Planner 拆分 issues
  -> Dependency Planner 构建 issue graph
  -> Scheduler 计算可并发队列
  -> 用户可见 issue graph / schedule
  -> Orchestrator 执行 issue DAG
```

## 4.1 需求完善与意图澄清

Requirement Refiner 需要把用户输入补成可执行需求：

- 背景：为什么做。
- 目标：完成后用户能做什么。
- 范围：包含什么、不包含什么。
- 约束：技术、权限、兼容性、性能、风格。
- 验收：可验证的成功条件。
- 风险：影响模块、数据、安全、发布。

Clarification Gate 判断是否需要追问用户。

必须澄清的情况：

- 目标结果不可验证。
- 存在多个互斥实现方向。
- 涉及破坏性数据迁移。
- 涉及鉴权、支付、安全、版本分支发布或生产投产。
- 需要用户选择兼容策略或 UI 行为。
- 验收标准不清楚。

可以直接进入拆分的情况：

- 项目已有明确规范。
- 用户需求与现有模块和历史决策一致。
- 缺失信息可从项目画像、memory 或代码事实中可靠补齐。

拆分时必须输出：

```yaml
epic_id: epic-20260503-001
issues:
  - id: issue-001
    title: 定义登录接口契约
    clarified_requirement: 为现有用户模块补充登录 API 契约
    type: design
    depends_on: []
    read_scopes:
      - src/auth/**
    write_scopes:
      - docs/**
    acceptance_criteria:
      - 明确请求、响应、错误码
    test_plan:
      - review API contract
    style_constraints:
      - 遵守现有 auth 模块命名和错误码风格

  - id: issue-002
    title: 实现登录接口后端逻辑
    type: backend
    depends_on:
      - issue-001
    read_scopes:
      - src/auth/**
      - src/user/**
    write_scopes:
      - src/auth/**
      - tests/auth/**
    acceptance_criteria:
      - 登录成功返回 token
      - 密码错误返回业务错误
    test_plan:
      - unit tests
      - integration tests
    style_constraints:
      - 不新增重复 token helper
      - 不绕过现有 service 层
```

## 5. 依赖类型

依赖不只是顺序关系，需要分类：

| Dependency | 含义 | 调度规则 |
| --- | --- | --- |
| blocks | 上游未完成，下游不能启动 | 强阻塞 |
| data_contract | API/schema/类型契约依赖 | 上游 design accepted 后可启动 |
| code_dependency | 依赖上游代码实现 | 上游 merged 或 integration passed 后可启动 |
| test_dependency | 依赖测试工具或 fixture | 上游测试基础完成后可启动 |
| resource_conflict | 写入范围冲突 | 不能并发 |
| review_dependency | 依赖 review 结论 | review accepted 后可启动 |

## 6. 并发决策

系统需要自己决定并发数，不是固定全开。

并发度由 Scheduler 计算：

```text
parallelism = min(
  project_policy.max_parallel_issues,
  ready_issue_count,
  available_worktrees,
  model_budget_slots,
  tool_execution_slots,
  non_conflicting_write_sets
)
```

默认阻止并发的情况：

- 两个 issue 写同一文件或同一模块核心文件。
- 涉及数据库迁移。
- 涉及鉴权、支付、安全策略。
- 涉及公共 API 或基础类型变更。
- 上游设计未 accepted。
- 项目 dirty worktree。
- quality gate 或 review 未通过。

允许并发的情况：

- 写入范围无交集。
- 一个前端 issue 和一个后端 issue 依赖同一已确认契约。
- 多个测试补齐 issue 覆盖不同模块。
- 文档、测试、低风险重构互不冲突。

## 6.1 前端 Claude / 后端 Codex 的编排等待设计

本项目以代码开发效果为核心目标。默认执行策略：

- 前端开发优先分配给 `frontend` role，并使用 `claude_cli`。
- 后端开发优先分配给 `backend` role，并使用 `codex_cli`。
- 后端调优优先分配给 `backend_tuning` role，并使用 `codex_cli`。
- 跨端接口契约、架构方案和复杂 UI/交互设计可以先由 `architect` 或 `planner` 产出设计，再分派给前后端实现。

### 等待队列

Scheduler 不直接把所有 ready issues 同时启动，而是维护四类队列：

```text
blocked_queue      # 依赖、契约、权限或用户输入未满足
ready_queue        # 依赖已满足，可以调度
running_queue      # 正在由 Runtime 执行
review_queue       # 已执行完成，等待质量门禁或 review
```

Issue 从 `blocked_queue` 进入 `ready_queue` 的条件：

- 所有 `hard` 依赖已 accepted 或 merged。
- 如果依赖 API/schema/类型契约，契约 issue 已 accepted。
- 写入范围没有与 running issue 冲突。
- 目标 Runtime 健康检查通过。
- 当前预算、worktree 和模型并发槽位可用。
- 没有等待用户澄清或审批。

### 前后端依赖规则

典型拆分：

```text
issue-a api_contract
  -> issue-b backend implementation  # codex_cli
  -> issue-c frontend integration     # claude_cli
  -> issue-d integration tests
```

调度规则：

- `api_contract` 未 accepted 前，后端和前端都必须等待。
- 后端和前端写入范围不冲突时，可以在契约 accepted 后并行。
- 如果前端依赖后端真实接口和 mock 不足，则前端进入 `blocked_queue`，等待后端 accepted。
- 如果后端依赖前端交互确认或字段展示规则，则后端进入 `needs_user_input` 或等待设计 issue accepted。
- 集成测试必须等待后端和前端都 accepted。

### Runtime 选择

| Issue 类型 | 默认 Role | 默认 Runtime | 等待条件 |
| --- | --- | --- | --- |
| `frontend` | `frontend` | `claude_cli` | UI 需求、接口契约、写入范围 ready |
| `backend` | `backend` | `codex_cli` | API 契约、数据模型、写入范围 ready |
| `backend_tuning` | `backend_tuning` | `codex_cli` | 性能目标、基线指标、测试命令 ready |
| `contract` | `architect` | `claude_cli` 或 `codex_cli` | 用户需求和项目理解 ready |
| `test` | `tester` | `codex_cli` | 被测实现 accepted |
| `review` | `reviewer` | `codex_cli` | diff 和 quality report ready |

如果默认 Runtime 不可用：

```text
preferred runtime unavailable
  -> mark runtime slot unavailable
  -> keep issue in ready_queue
  -> try fallback runtime if policy allows
  -> otherwise mark issue BLOCKED(runtime_unavailable)
```

### 等待状态细分

`BLOCKED` 必须记录具体原因：

| blocked reason | 含义 | 解除条件 |
| --- | --- | --- |
| `waiting_contract` | 等待 API/schema/UI 契约 | contract issue accepted |
| `waiting_backend` | 前端等待后端实现或 mock | backend issue accepted 或 mock ready |
| `waiting_frontend_decision` | 后端等待前端交互或字段规则 | frontend/design issue accepted |
| `waiting_runtime_slot` | Runtime 并发槽位不足 | 有可用 `claude_cli` 或 `codex_cli` slot |
| `waiting_worktree` | worktree 资源不足 | worktree 可用 |
| `waiting_quality` | 上游质量门禁未通过 | quality passed |
| `waiting_review` | 上游 review 未通过 | review accepted |
| `waiting_user_input` | 用户澄清或审批缺失 | 用户补充或批准 |
| `waiting_merge` | 上游已 accepted 但未合入 integration branch | merge 完成 |

### 编排等待循环

Orchestrator 按事件驱动方式循环：

```text
issue status changed
  -> update issue graph
  -> release dependency locks
  -> refresh ready_queue
  -> calculate runtime slots
  -> schedule frontend issues to claude_cli
  -> schedule backend issues to codex_cli
  -> start runs
  -> wait for run events
  -> quality/review
  -> merge accepted issue
  -> unlock downstream issues
```

等待期间系统必须做的事：

- 周期性检查 Runtime 健康状态。
- 检查 running issue 是否超时。
- 将 blocked reason 写入用户可见 schedule。
- 对可解决的阻塞自动 replan。
- 不重复启动同一个 issue。
- 不在上游未 accepted 时提前启动下游代码写入。

### 合流点

前后端并行开发后必须在集成分支合流：

```text
backend accepted
frontend accepted
  -> merge both issue branches into epic integration branch
  -> run integration checks
  -> run API/UI compatibility tests
  -> unlock integration test issue
```

如果合流失败：

- 标记相关 issue 为 `NEEDS_REWORK` 或生成新的 integration fix issue。
- 下游测试和 release 必须等待。
- 不允许只合入前端或只合入后端后直接 release，除非该 Epic 明确是单端变更。

## 7. 调度状态机

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

失败路径：
  -> NEEDS_REWORK
  -> FAILED
  -> CANCELLED
```

Epic 状态由 issue graph 汇总：

- 全部 DONE：epic completed。
- 有 FAILED：epic failed 或需要 replan。
- 有 BLOCKED：等待依赖或用户输入。
- 有 NEEDS_REWORK：暂停下游依赖。

## 8. 执行模型

### 串行依赖

```text
issue-a design
  -> accepted
  -> issue-b backend
  -> accepted
  -> issue-c tests
```

### 并行执行

```text
issue-a API contract
  -> accepted
  -> issue-b backend implementation
  -> issue-c frontend integration
  -> issue-d tests
```

`issue-b`、`issue-c`、`issue-d` 只有在写入范围不冲突时才可并发。

### 集成分支

建议每个 Epic 有一个 integration branch：

```text
moyuan/epic-20260503-001
```

每个 issue 有独立 issue branch：

```text
moyuan/issue-20260503-001-login-api
```

完成并通过质量门禁后，issue branch 合入 epic integration branch。下游 issue 默认基于 integration branch 最新状态启动。

## 8.1 合入门禁

issue branch 合入 integration branch 前必须满足：

- issue acceptance criteria 全部通过。
- quality gates 通过。
- test/lint/build/typecheck 通过。
- reviewer accepted。
- quality_guard accepted。
- style constraints 通过。
- 没有未解决 blocker/high findings。
- 没有写入范围冲突。

合入后动作：

```text
issue accepted
  -> merge issue branch into epic integration branch
  -> run integration checks
  -> update issue graph
  -> unlock downstream issues
  -> recalculate ready queue and parallelism
```

代码风格控制来源：

- 项目已有 lint/formatter/typecheck。
- `.moyuan/policies/code-quality.yaml`。
- 项目阅读理解生成的 style facts。
- review 和 quality_guard 的 accepted 结论。

## 9. Worktree 策略

并发执行时建议为每个 running issue 创建独立 worktree，避免互相覆盖。

```text
.moyuan/worktrees/
  issue-001/
  issue-002/
  issue-003/
```

每个 worktree 绑定：

- issue id。
- branch。
- assigned agents。
- write scopes。
- run id。
- lock 文件。

## 10. 冲突处理

冲突来源：

- Git merge conflict。
- 写入范围重叠。
- 设计契约变化。
- 下游基于过时接口实现。
- quality gate 失败。

处理策略：

1. 暂停受影响下游 issues。
2. 生成 conflict report。
3. 由 planner/architect 判断是否需要 replan。
4. 更新 issue graph。
5. 重新计算 ready queue。

## 11. 配置

`.moyuan/policies/orchestration.yaml`：

```yaml
schema_version: 1

orchestration:
  enabled: true
  issue_graph: true
  auto_parallelism: true
  max_parallel_issues: 3
  require_clean_worktree: true
  use_epic_integration_branch: true
  use_issue_worktrees: true

  concurrency_guards:
    disallow_same_file_writes: true
    disallow_same_module_core_writes: true
    serialize_database_migrations: true
    serialize_auth_security_payment: true
    require_design_acceptance_for_public_api: true

  replan:
    enabled: true
    on_conflict: true
    on_quality_failure: true
    on_dependency_change: true

  graph_visibility:
    write_user_visible_graph: true
    graph_path: .moyuan/lifecycle/issue-graphs
    include_blocked_reason: true
    include_parallelism_plan: true

  merge_gate:
    require_quality_passed: true
    require_review_accepted: true
    require_style_check: true
    require_integration_checks: true
    merge_into_epic_branch_only: true
```

## 12. 版本分支、投产与维护流水线

Release Manager 负责版本分支管理、发布到 GitHub/Gitee、结合目标服务器自动化部署投产，以及投产后的冒烟、监控和更新维护。

配置入口：

- 版本分支、tag、PR/MR 和投产门禁：`.moyuan/policies/release.yaml`。
- 服务器资产、资源组、到期时间和巡检：`.moyuan/policies/server-resources.yaml`。
- 环境级部署、线上冒烟、监控和回滚：`.moyuan/policies/environments.yaml`。
- 字段必填、可空和必须为空规则：[配置 Schema 规则](./configuration-schema-spec.md)。
- 服务器资源对象语义：[核心数据对象](./foundations/core-data-objects.md)。

### 版本批次建议

默认建议小批量合并到版本分支并投产：

- 低风险功能：累计 3-7 个 accepted issues 后建议创建/更新 release branch 并准备投产。
- 中风险功能：累计 2-4 个 accepted issues 后建议创建/更新 release branch 并准备投产。
- 高风险变更：单独 release branch 和单独投产窗口，例如数据库迁移、鉴权、安全、支付、公共 API 变更。
- hotfix/security：不等待批次，立即创建 hotfix/release branch 并进入投产流水线。

实际阈值由风险和变更规模动态调整：

```text
release_batch_score = issue_count
  + changed_module_count * 1.5
  + migration_count * 3
  + public_api_change * 2
  + security_change * 3
  + unresolved_risk_count * 2
```

超过项目阈值时，Release Manager 生成 release/deploy suggestion。

### 投产流水线

```text
integration branch ready
  -> release suggestion
  -> release planning
  -> create release candidate branch
  -> full quality gates
  -> full regression tests
  -> migration and config checks
  -> release notes
  -> user approval if required
  -> create tag if configured
  -> push release branch to GitHub/Gitee
  -> create PR/MR if configured
  -> prepare server environment
  -> backup current version
  -> deploy to target servers
  -> run online smoke tests
  -> observe metrics and logs
  -> mark release healthy or rollback required
  -> release retrospective
  -> release memory
```

发布必须记录：

- release id。
- included issues。
- excluded issues。
- risk summary。
- test evidence。
- migration checklist。
- release branch。
- tag。
- remote provider。
- PR/MR url。
- target environment。
- target servers。
- deploy strategy。
- server config refs。
- backup artifact。
- smoke test result。
- monitor window result。
- rollback plan。
- approval record。

### 服务器资源与环境引用

投产流水线不直接维护服务器字段，只读取资源组和环境配置：

```text
release.target_environment
  -> environments.<environment>.resource_group
  -> server_resources.groups.<group>.host_ids
  -> server_resources.hosts
```

设计约束：

- 服务器只在 `server-resources.yaml` 登记一次。
- `environments.yaml` 只描述部署方式、健康检查、冒烟、监控和回滚。
- 生产机必须经过 release/deploy pipeline 操作，不能由单个 issue 绕过审批直接执行远程命令。
- 云服务器到期、巡检失败、备份缺失和生产健康检查失败会生成维护 issue。
- 敏感值只保存引用，不写入主机密码、token、SSH 私钥或云厂商密钥。

### 投产策略

支持策略：

- `manual_push`：只推送 release branch/tag，不部署。
- `ssh_script`：通过 SSH 执行部署脚本。
- `docker_compose`：更新镜像并重启 compose 服务。
- `kubernetes`：更新 deployment 镜像或 manifest。
- `ci_trigger`：触发 GitHub Actions、Gitee Go、Jenkins 等外部流水线。

高风险投产必须要求用户确认：

- 数据库迁移。
- 回滚不可自动完成。
- 涉及多台服务器。
- 健康检查缺失。
- 没有可用备份。

发布策略配置示例统一维护在 [完整配置方案](./configuration-guide.md)，字段约束统一维护在 [配置 Schema 规则](./configuration-schema-spec.md)。

## 13. Workspace 产物

```text
.moyuan/lifecycle/
  epics/
    epic-20260503-001.yaml
  issues/
    issue-20260503-001.yaml
  issue-graphs/
    epic-20260503-001.graph.json
  schedules/
    schedule-20260503-001.json
  releases/
    release-20260503-001.yaml
```

Issue graph 示例：

```json
{
  "epic_id": "epic-20260503-001",
  "nodes": [
    {"id": "issue-001", "status": "accepted"},
    {"id": "issue-002", "status": "running"},
    {"id": "issue-003", "status": "blocked"}
  ],
  "edges": [
    {"from": "issue-001", "to": "issue-002", "type": "blocks"},
    {"from": "issue-002", "to": "issue-003", "type": "code_dependency"}
  ],
  "ready_queue": ["issue-002"],
  "running": ["issue-002"],
  "blocked": ["issue-003"]
}
```

## 14. 验收标准

- 用户输入一个开发目标后，系统能自动生成 epic 和 issues。
- 每个 issue 有明确验收标准、写入范围和测试计划。
- 系统能生成 issue dependency graph。
- 系统能自动判断 ready queue。
- 系统能根据依赖和写入范围决定并发数量。
- 依赖未完成的 issue 不会启动。
- 并发 issue 不会写同一文件。
- issue 失败会阻塞下游并触发 replan。
- 每个 issue 都经过质量门禁和 review 后才可解锁下游。
- issue 复核通过后才能合入 epic integration branch。
- 系统能基于 integration branch 累计 issue 数、风险和变更范围给出 release/deploy 建议。
- 发布流水线包含 release branch、回归测试、release notes、审批、tag、push 到 GitHub/Gitee、PR/MR、服务器部署、线上冒烟、监控窗口、回滚策略和复盘。
