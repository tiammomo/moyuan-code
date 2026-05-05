# Phase 12 实施记录

状态：completed
责任角色：orchestrator_owner + backend_owner + git_owner + release_owner + frontend_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 12 的实际执行顺序。Phase 12 的入口以 [Phase 12 实现 Issue Graph](./phase12-issue-graph.md) 为准。

## 1. 当前基线

Phase 11 已完成并通过 readiness：

- Batch plan 能解释 dispatch、waiting、blocked、write scope conflict 和 provider route。
- Batch run 已有 dry-run 和受控 `local_shell` 执行。
- Issue worktree isolation 已落地，每个执行 issue 有独立 worktree 和 branch。
- Merge queue 已能聚合 quality/review 结论。
- Console 已能展示 batch plan/run、worktree 和 merge readiness。

## 2. Phase 12 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase12-001` | `parallel-batch-worker-executor` | completed | 真实受控并发执行 | batch run 可按安全并发度执行多个 issue，并记录 worker slot 和 fail-fast |
| P0 | `phase12-002` | `integration-merge-preview` | completed | 集成合入预览 | ready merge queue 可生成 merge dry-run 和冲突报告 |
| P1 | `phase12-003` | `controlled-merge-apply` | completed | 受控真实合入 | 审批和开关满足后可合入 integration branch |
| P1 | `phase12-004` | `release-batch-readiness` | completed | 发版批次建议 | 根据合入量、风险和版本策略生成 release batch plan |
| P2 | `phase12-005` | `console-parallel-merge-surface` | completed | Console 并发与合入面 | Console 可见 worker slot、merge preview 和 release batch readiness |

## 3. 执行规划：`phase12-001 parallel-batch-worker-executor`

实现状态：completed。

范围：

- 为 `batch.Run(local_shell)` 增加 bounded worker pool。
- 并发度取 `min(batch_plan.runtime_slots, requested max_issues, system cap)`。
- 每个 issue worker 独立调用 worktree manager 和 orchestrator；worktree 创建保持受控顺序，runtime/orchestrator 执行进入 worker pool。
- `RunRecord` 增加 `parallelism`，`RunItem` 增加 `worker_slot` 和 `canceled_reason`。
- `continue_on_failure=false` 时，首个失败取消未开始任务，已开始任务允许自然收口。
- run items 按 batch plan issue 顺序稳定输出，避免前端和审计抖动。
- orchestrator issue graph 状态回写增加串行保护，避免并发 issue 同时写 graph 时互相覆盖。

非目标：

- 不做 integration merge preview。
- 不做真实 `git merge`。
- 不引入后台常驻 scheduler。

验收：

- `local_shell` batch run 在 `max_issues > 1` 时能处理多个 dispatch issue。
- 每个 issue 都有独立 worktree。
- fail-fast 会阻止后续未开始 issue 并记录 blocked/canceled item。
- `continue_on_failure=true` 时失败不阻断其他 issue。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

落地结果：

- `batch.Run(local_shell)` 在 `max_issues > 1` 时会按安全并发度执行多个 issue。
- batch run artifact 记录 `parallelism`、每个 item 的 `worker_slot`、`canceled_reason`、worktree 和 quality report。
- Console 的 Batch Runs 面板可展示 parallelism 和 worker slot。

## 4. 执行规划：`phase12-002 integration-merge-preview`

实现状态：completed。

范围：

- 新增 integration preview 事实源，读取 ready merge queue 后创建独立 integration worktree 和 `moyuan/integration/...` 分支。
- 对每个 ready merge queue item 执行 `git merge --no-commit --no-ff` 预览。
- 记录 clean、conflict、protected path blocked、source branch missing 等 item 级状态。
- merge 成功且有变更时只提交到 integration preview branch，不影响主工作区和生产分支。
- preview 写入 `.moyuan/lifecycle/merge-reports/integration-previews/` 和 `integration-previews.jsonl`。
- API 支持生成、列表和详情查询 integration preview。

非目标：

- 不执行真实 integration branch apply。
- 不创建 PR/MR。
- 不自动 tag、push 或 publish。

验收：

- ready merge queue 可生成 `INTEGRATION_PREVIEW_READY`。
- unready merge queue 会生成 blocked preview，并记录 `merge_queue_not_ready`。
- merge conflict 会被记录为 `INTEGRATION_ITEM_CONFLICT`，并保留 conflicted files。
- protected path 变更会阻断 preview item。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

落地结果：

- 新增 `review.BuildIntegrationPreview`、`LoadIntegrationPreview`、`ListIntegrationPreviews`。
- API 新增 `POST /v1/projects/:project_id/merge-queues/:queue_id/integration-preview`、`GET /v1/projects/:project_id/integration-previews`、`GET /v1/projects/:project_id/integration-previews/:preview_id`。
- integration preview 仍是 dry-run / preview 层，不会执行真实合入。

## 5. 执行规划：`phase12-003 controlled-merge-apply`

实现状态：completed。

范围：

- 新增 integration apply 事实源，基于 ready integration preview 执行。
- 默认 `dry_run`，只验证 preview ready，不更新 Git ref。
- 真实 `apply` 必须满足 `approved=true` 和 `MOYUAN_ALLOW_INTEGRATION_APPLY=1`。
- 真实 apply 只把 preview branch 固化为本地 integration branch，不 push、不 PR/MR、不 tag。
- apply 写入 `.moyuan/lifecycle/merge-reports/integration-applies/` 和 `integration-applies.jsonl`。
- API 支持 apply、列表和详情查询。

非目标：

- 不合入 main。
- 不推送远程。
- 不创建 PR/MR 或 release。

验收：

- preview 未 ready 时 apply blocked。
- 未审批或未开启环境开关时真实 apply blocked。
- dry-run 不更新 Git ref。
- 真实 apply 只更新本地 target integration branch。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

落地结果：

- 新增 `review.ApplyIntegrationPreview`、`LoadIntegrationApply`、`ListIntegrationApplies`。
- API 新增 `POST /v1/projects/:project_id/integration-previews/:preview_id/apply`、`GET /v1/projects/:project_id/integration-applies`、`GET /v1/projects/:project_id/integration-applies/:apply_id`。
- integration apply 是本地 Git ref 层的受控动作，远程发布仍留给后续 release/provider 流水线。

## 6. 执行规划：`phase12-004 release-batch-readiness`

实现状态：completed。

范围：

- 新增 release batch plan 事实源，基于 completed integration apply 生成。
- 根据 ready integration items 数量和 `min_items` 阈值判断 `suggested` 或 `not_ready`。
- 记录 source integration branch、release branch、version、commands 和 readiness reason。
- release batch 写入 `.moyuan/lifecycle/releases/batches/` 和 `release-batches.jsonl`。
- API 支持生成、列表和详情查询 release batch。

非目标：

- 不创建 release branch。
- 不 tag、不 push、不 PR/MR、不 publish。
- 不替代已有 release provider preview/publish 流程。

验收：

- completed integration apply + ready item count 达标时生成 `RELEASE_BATCH_SUGGESTED`。
- ready item count 未达标时生成 `RELEASE_BATCH_NOT_READY`。
- integration apply 未完成时 blocked。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

落地结果：

- 新增 `release.PlanBatch`、`LoadBatchPlan`、`ListBatchPlans`。
- API 新增 `POST /v1/projects/:project_id/integration-applies/:apply_id/release-batch`、`GET /v1/projects/:project_id/release-batches`、`GET /v1/projects/:project_id/release-batches/:batch_id`。
- release batch 仍是建议层，不执行 Git 或远程发布写入。

## 7. 执行规划：`phase12-005 console-parallel-merge-surface`

实现状态：completed。

范围：

- Console snapshot 接入 `integration_previews`、`integration_applies` 和 `release_batches`。
- `Batches` 视图在 batch plan/run、worktree/merge queue 之后展示 Integration & Release 链路。
- merge queue 可触发后端 `integration-preview` 受控预览。
- ready integration preview 可触发 `apply dry-run`，只生成 apply 计划，不更新 Git ref。
- integration apply 可触发 release batch readiness 检查，输出建议或未达阈值原因。
- demo snapshot 同步补齐 integration preview、integration apply 和 release batch 示例。

非目标：

- 不在前端执行真实 Git 命令。
- 不在前端推导 merge readiness 或 release readiness。
- 不默认触发真实 integration apply、release branch、tag、push 或 publish。

验收：

- Console live snapshot 能读取并展示最近 integration preview、integration apply 和 release batch。
- Console 所有新增操作都只调用后端受控 API，成功后刷新 server snapshot。
- 无后端时 demo snapshot 仍能展示完整并发到发版建议链路。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

落地结果：

- `ConsoleSnapshot` 增加 `integration_previews`、`integration_applies` 和 `release_batches`。
- Console `Integration & Release` 面板展示 preview/apply/release batch 的状态、reason、branch、ready count 和命令预览。
- Phase 12 从真实并发执行到集成预览、受控应用、发版批次建议和 Console 可见性形成闭环。

## 8. 验证要求

每完成一个 Phase 12 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
