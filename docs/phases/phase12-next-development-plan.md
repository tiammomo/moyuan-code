# Phase 12 实施记录

状态：in_progress
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
| P0 | `phase12-001` | `parallel-batch-worker-executor` | planned | 真实受控并发执行 | batch run 可按安全并发度执行多个 issue，并记录 worker slot 和 fail-fast |
| P0 | `phase12-002` | `integration-merge-preview` | planned | 集成合入预览 | ready merge queue 可生成 merge dry-run 和冲突报告 |
| P1 | `phase12-003` | `controlled-merge-apply` | planned | 受控真实合入 | 审批和开关满足后可合入 integration branch |
| P1 | `phase12-004` | `release-batch-readiness` | planned | 发版批次建议 | 根据合入量、风险和版本策略生成 release batch plan |
| P2 | `phase12-005` | `console-parallel-merge-surface` | planned | Console 并发与合入面 | Console 可见 worker slot、merge preview 和 release batch readiness |

## 3. 执行规划：`phase12-001 parallel-batch-worker-executor`

实现状态：planned。

范围：

- 为 `batch.Run(local_shell)` 增加 bounded worker pool。
- 并发度取 `min(batch_plan.runtime_slots, requested max_issues, system cap)`。
- 每个 issue worker 独立调用 worktree manager 和 orchestrator。
- `RunItem` 增加 worker slot、cancellation reason 或 parallel execution metadata。
- `continue_on_failure=false` 时，首个失败取消未开始任务，已开始任务允许自然收口。
- run items 按 batch plan issue 顺序稳定输出，避免前端和审计抖动。

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

## 4. 验证要求

每完成一个 Phase 12 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
