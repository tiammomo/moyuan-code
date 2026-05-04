# Beta 实施记录

状态：in_progress
责任角色：orchestrator_owner + backend_owner + qa_owner
最后更新：2026-05-04

本文记录 Beta 阶段从规划到执行的实际顺序。稳定设计结论需要回写到对应主线、策略、契约或配置文档；本文件只记录阶段执行事实。

## 1. 当前基线

Phase 1 本地 CLI MVP 已完成，验收入口见 [Phase 1 Release Readiness](./phase1-release-readiness.md)。

当前可复用能力：

- `.moyuan/` 项目工作空间、项目接入、阅读理解和 Git 绑定。
- Issue graph、schedule、orchestrator issue/run 状态机。
- Runtime adapter、Claude CLI/Codex CLI 调用契约和 local shell fallback。
- Quality review gate、Memory record gate、repair controlled loop。
- Gin + GORM 基线，项目注册会同步 `.moyuan/state.db`。

## 2. Beta 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `beta-001` | `state-query-api` | completed | 控制面 API 可查询项目核心状态 | API + 测试 + 文档同步 |
| P0 | `beta-002` | `issue-graph-api` | in_progress | API 可展示 issue graph、schedule 和队列 | issue graph 可被前端可视化读取 |
| P0 | `beta-003` | `requirement-to-issues` | planned | 需求丰富、澄清判断和 issue graph 生成 | 用户需求可转为 issues DAG |
| P1 | `beta-004` | `parallel-orchestration-engine` | planned | 自动并发、等待和 replan | 并发度由系统决策且可审计 |
| P1 | `beta-005` | `review-merge-pipeline` | planned | 复核通过后合入任务分支 | review gate 阻断未达标代码 |

## 3. 已完成任务：`beta-001 state-query-api`

范围：

- `GET /v1/projects`
- `GET /v1/projects/:project_id`
- `GET /v1/projects/:project_id/issues/:issue_id`
- `GET /v1/projects/:project_id/runs/:run_id`
- `GET /v1/projects/:project_id/quality/:report_id`
- `GET /v1/projects/:project_id/memory/search?q=&limit=`
- `GET /v1/projects/:project_id/memory/candidates?limit=`
- `GET /v1/projects/:project_id/repair/attempts/:attempt_id`

非目标：

- 不做写操作 API。
- 不做 Web Console。
- 不做自动 push、merge、deploy。
- 不改变 `.moyuan/` 文件状态作为当前事实来源的原则。

验收：

- 缺失项目和缺失状态返回 404。
- 查询接口使用 Gin router 测试覆盖。
- GORM Store 支持按 project id 查询。
- `go test ./...` 通过。

完成记录：

- 已新增 project、issue state、run state、quality report、memory search、memory candidates 和 repair attempt 只读 API。
- 已新增 Store `FindProject` 查询能力。
- 已覆盖 GORM Store、controlplane fallback、状态读取和 404 行为。
- 验证命令：`PATH=/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH go test ./...`。

## 4. 当前任务：`beta-002 issue-graph-api`

范围：

- `GET /v1/projects/:project_id/epics/:epic_id/issue-graph`
- `GET /v1/projects/:project_id/epics/:epic_id/schedule`
- 统一返回 ready queue、blocked queue、running/review 占位队列和 blocked reason。

非目标：

- 不生成新 issue graph。
- 不执行调度。
- 不修改 issue 状态。

验收：

- 已有 Phase 1 issue graph 可通过 API 读取。
- 缺失项目返回 404。
- 缺失 epic 返回 404。
- `go test ./...` 通过。
