# Phase 11 实现 Issue Graph

状态：in_progress
责任角色：orchestrator_owner + backend_owner + frontend_owner + qa_owner
最后更新：2026-05-05

Phase 11 的目标是把当前 scheduler、orchestrator、subagent、quality 和 Console 能力收敛成“Issue Graph 批量执行控制器”。系统需要能基于已理解项目和完善后的需求，生成可见 issue graph，判断依赖和写入冲突，决定并发度，并以受控方式推进开发、复核和合入。

## 1. Phase 11 目标

- 从 scheduler dispatch queue 生成 batch execution plan，解释 ready、waiting、blocked 和并发槽位。
- 支持 batch dry-run，先把任务拆分、依赖、write scope、runtime/provider route 和质量门禁展示清楚。
- 为后续真实多 agent 执行引入 worktree isolation、write scope lock 和 merge queue。
- 每个 issue 完成后进入质量门禁、review、merge decision，不能绕过复核。
- Console 展示 batch plan、运行状态、waiting reason、review queue 和 merge readiness。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase11-001` | `issue-batch-dispatch-preview` | planned | batch plan、dispatch snapshot、并发槽、write scope 冲突、runtime/provider route 摘要 | Phase 10 readiness | `orchestrator_owner` + `backend_owner` | 可对 epic 生成 dry-run batch execution plan，并解释每个 issue 为什么 dispatch/wait/block |
| `phase11-002` | `bounded-issue-batch-run` | planned | 受控 batch run、串行/有限并发执行、run artifact、失败停止策略 | `phase11-001` | `backend_owner` + `qa_owner` | 可在审批和安全模式下执行一批 issue，记录每个 issue 的 runtime、quality 和 review 结果 |
| `phase11-003` | `parallel-worktree-isolation` | planned | 每个并发 issue 的隔离 worktree、branch 命名、cleanup、冲突检测 | `phase11-002` | `backend_owner` | 并发 issue 不共享写入目录，冲突只在 merge queue 暴露 |
| `phase11-004` | `quality-review-merge-queue` | planned | batch 质量聚合、review queue、merge decision queue、失败返工 | `phase11-002` | `qa_owner` | issue 只有 quality + review 通过后才能进入 merge ready |
| `phase11-005` | `console-batch-execution-surface` | planned | batch plan、batch runs、waiting reason、merge readiness Console 展示 | `phase11-001`,`phase11-004` | `frontend_owner` | Console 可查看 batch plan/run，并触发 dry-run preview |

## 3. 建议执行顺序

1. 先做 `phase11-001`，只生成 batch execution preview，不运行 runtime。
2. 再做 `phase11-002`，开放受控 batch run，但默认仍可 dry-run。
3. `phase11-003` 在真实并发前必须完成，避免多个 agent 共用同一 worktree。
4. `phase11-004` 把质量、复核和合入队列收敛成统一结果。
5. `phase11-005` 最后把后端事实源接到 Console。

## 4. 收口规则

- 没有 batch plan，不允许 batch run。
- 没有 write scope，不允许真实并发。
- 没有 worktree isolation，不允许多个写入型 issue 同时执行。
- 没有 quality report 和 review decision，不允许进入 merge ready。
- Console 只能展示后端 batch plan/run 事实源，不自行计算 merge 结论。
