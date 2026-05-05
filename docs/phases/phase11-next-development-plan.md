# Phase 11 实施记录

状态：in_progress
责任角色：orchestrator_owner + backend_owner + frontend_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 11 的实际执行顺序。Phase 11 的入口以 [Phase 11 实现 Issue Graph](./phase11-issue-graph.md) 为准。

## 1. 当前基线

Phase 10 已完成并通过 readiness：

- Control loop 已能手动 bounded run。
- Operation repair candidate 已有 review flow。
- Provider route candidates 可由后端解释，并已能在 Console 触发预览。
- Scheduler 已能生成 dispatch queue 和 waiting queue，但还没有 batch execution 事实源。
- Orchestrator 已能执行单个 issue，并串接 runtime、subagent、quality 和 review。

## 2. Phase 11 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase11-001` | `issue-batch-dispatch-preview` | completed | 批量执行预览 | 可生成 batch plan，解释 dispatch/wait/block、并发槽和 write scope 冲突 |
| P0 | `phase11-002` | `bounded-issue-batch-run` | completed | 受控批量执行 | 审批/安全模式下可执行一批 issue，并记录每个 issue 结果 |
| P1 | `phase11-003` | `parallel-worktree-isolation` | planned | 并发隔离 | 并发 issue 使用独立 worktree/branch，不共享写入目录 |
| P1 | `phase11-004` | `quality-review-merge-queue` | planned | 质量复核合入队列 | issue 通过 quality + review 后进入 merge ready |
| P2 | `phase11-005` | `console-batch-execution-surface` | planned | Console 批量执行面 | Console 可查看 batch plan/run 和 merge readiness |

## 3. 执行规划：`phase11-001 issue-batch-dispatch-preview`

实现状态：completed。

范围：

- 新增 batch plan 结构，基于 `scheduler.Build` 的 dispatch/waiting/backlog 结果生成事实源。
- batch plan 记录 `epic_id`、`mode=dry_run`、`max_parallel`、`dispatch_count`、`waiting_count`、`blocked_count`、`write_scope_conflict_count`。
- 每个 issue item 记录 role、runtime_id、provider route preview、write_scopes、dependency_ids、decision 和 reason。
- 输出到 `.moyuan/orchestrator/batches/` 和 `.moyuan/orchestrator/batches.jsonl`。
- API 支持创建 batch plan、列表和详情。

非目标：

- 不运行 runtime。
- 不修改 issue 状态。
- 不创建 worktree。
- 不合入分支。

验收：

- `POST /v1/projects/:project_id/epics/:epic_id/batches/plan` 可生成 dry-run batch plan。
- `GET /v1/projects/:project_id/epics/:epic_id/batches` 可查看最近 batch plans。
- `GET /v1/projects/:project_id/batches/:batch_id` 可查看详情。
- plan 能解释 ready issue 的 dispatch/waiting 原因和 write scope 冲突。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

落地结果：

- 新增 `internal/batch`，生成 dry-run `batch_plan`。
- 新增 API：`POST /v1/projects/:project_id/epics/:epic_id/batches/plan`、`GET /v1/projects/:project_id/epics/:epic_id/batches`、`GET /v1/projects/:project_id/batches/:batch_id`。
- batch plan 写入 `.moyuan/orchestrator/batches/` 和 `.moyuan/orchestrator/batches.jsonl`。
- 每个 ready/waiting issue 会附带 provider route preview；blocked issue 保留 dependency reason。

## 4. 执行规划：`phase11-002 bounded-issue-batch-run`

实现状态：completed。

范围：

- 新增 `batch_run` 事实源，基于已存在 `batch_plan` 执行，不允许跳过 plan。
- 默认 `dry_run`，只记录将执行的 issue，不运行 runtime、不修改 issue 状态。
- 支持受控 `local_shell` 执行，必须满足 `approved=true`、`MOYUAN_ALLOW_BATCH_RUN=1` 和 prompt 安全白名单。
- 在 `phase11-003` worktree isolation 完成前，真实执行自动收敛为单 issue 串行执行，并记录 `shared_worktree_serial_limit`。
- 每个 run item 记录 issue、runtime、provider、model、run_id、subagent_id、quality_report_id 和执行结论。
- API 支持触发 batch run、查看 batch run 列表和详情。

非目标：

- 不做真实多 worktree 并发。
- 不自动合入分支。
- 不替代后续 `phase11-004` 的 merge queue。

验收：

- `POST /v1/projects/:project_id/batches/:batch_id/run` 可触发 dry-run batch run。
- `GET /v1/projects/:project_id/batch-runs` 可查看最近 batch runs。
- `GET /v1/projects/:project_id/batch-runs/:run_id` 可查看详情。
- 未审批或未开启环境开关时，真实 local shell run 必须被阻断并记录原因。
- local shell run 通过时，issue state 必须回写到 batch plan 所属 epic。

落地结果：

- 新增 `batch.Run`、`LoadRun`、`ListRuns`。
- batch run 写入 `.moyuan/orchestrator/batch-runs/` 和 `.moyuan/orchestrator/batch-runs.jsonl`。
- `orchestrator.RunIssueWithOptions` 支持 `epic_id`，避免自定义 issue graph 回写到默认 Phase1 epic。
- API 新增 `POST /v1/projects/:project_id/batches/:batch_id/run`、`GET /v1/projects/:project_id/batch-runs`、`GET /v1/projects/:project_id/batch-runs/:run_id`。

## 5. 验证要求

每完成一个 Phase 11 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
