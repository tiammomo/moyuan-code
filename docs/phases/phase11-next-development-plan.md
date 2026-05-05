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
| P1 | `phase11-003` | `parallel-worktree-isolation` | completed | 并发隔离 | 并发 issue 使用独立 worktree/branch，不共享写入目录 |
| P1 | `phase11-004` | `quality-review-merge-queue` | completed | 质量复核合入队列 | issue 通过 quality + review 后进入 merge ready |
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

## 5. 执行规划：`phase11-003 parallel-worktree-isolation`

实现状态：completed。

范围：

- 新增 issue worktree manager，负责创建、记录、查询和清理 Git worktree。
- worktree branch 使用 `moyuan/<epic>/<issue>/<worktree-id>` 命名，实际目录位于 `.moyuan/worktrees/`。
- worktree 记录写入 `.moyuan/orchestrator/worktrees/` 和 `.moyuan/orchestrator/worktrees.jsonl`。
- 创建 worktree 前检查主仓库是否为 Git repo，并用 user dirty 口径阻断用户改动；`.moyuan` 控制文件不视为用户改动。
- `batch_run local_shell` 为每个 issue 分配独立 worktree，再在 worktree 内运行 Runtime、diff capture 和 quality checks。
- `orchestrator.RunIssueWithOptions` 支持 `worktree_path` 和 `branch`，质量检查也切到 issue worktree 内执行。
- API 支持查看 worktree 列表和详情。

非目标：

- 不在本阶段启动 goroutine 并发执行。
- 不自动删除 task branch。
- 不自动合入 integration branch。

验收：

- 可为 issue 创建独立 Git worktree 和 branch。
- dirty user worktree 会阻断创建。
- cleanup 可移除 worktree 并记录状态。
- batch run item 会记录 `worktree_id`、`worktree_path` 和 `branch`。
- Console 可通过 API 读取 worktree 事实源。

落地结果：

- 新增 `internal/worktree`。
- API 新增 `GET /v1/projects/:project_id/worktrees`、`GET /v1/projects/:project_id/worktrees/:worktree_id`。
- `batch.Run(local_shell)` 不再共享主工作区执行 issue。

## 6. 执行规划：`phase11-004 quality-review-merge-queue`

实现状态：completed。

范围：

- 新增 batch merge queue，读取 batch run 结果并聚合每个 issue 的质量复核和合入状态。
- 对 accepted batch item 调用现有 `review.DecideMerge`，复用单 issue merge gate。
- 每个 queue item 输出 issue、run、subagent、quality report、worktree、branch 和 merge decision。
- queue 聚合为 `ready_to_merge`、`needs_rework`、`blocked` 三类，并统计对应数量。
- merge queue 写入 `.moyuan/lifecycle/merge-reports/queues/` 和 `.moyuan/lifecycle/merge-reports/merge-queues.jsonl`。
- API 支持生成、列表和详情查询 merge queue。

非目标：

- 不执行真实 `git merge`。
- 不创建 PR/MR。
- 不推进 release suggestion。

验收：

- dry-run batch item 不允许进入 ready merge queue。
- accepted issue + passed quality + accepted review 可进入 ready merge queue。
- failed/rejected item 进入 needs rework。
- 缺少 batch run 或缺少 merge facts 时 queue blocked。

落地结果：

- `review.BuildMergeQueue`、`LoadMergeQueue`、`ListMergeQueues` 已实现。
- API 新增 `POST /v1/projects/:project_id/batches/:batch_id/merge-queue`、`GET /v1/projects/:project_id/merge-queues`、`GET /v1/projects/:project_id/merge-queues/:queue_id`。
- 后续 `phase11-005` Console 可直接展示 merge readiness，不需要前端自行计算。

## 7. 验证要求

每完成一个 Phase 11 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
