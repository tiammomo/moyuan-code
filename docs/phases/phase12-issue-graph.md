# Phase 12 实现 Issue Graph

状态：in_progress
责任角色：orchestrator_owner + backend_owner + git_owner + release_owner + frontend_owner + qa_owner
最后更新：2026-05-05

Phase 12 的目标是把 Phase 11 的“可见批量执行控制器”推进到“真实并发执行与集成合入准备”。本阶段继续保持 production write 默认关闭，但要让系统能在独立 worktree 中受控并发执行 issue，聚合结果后进入 integration merge preview，并为后续 release batching 建立事实源。

## 1. Phase 12 目标

- batch run 根据 batch plan、runtime slots、write scope 和用户上限决定真实并发度。
- 每个并发 issue 必须使用独立 worktree，不允许共享主工作区写入。
- 并发执行要记录 worker slot、started/finished、失败原因和是否因 fail-fast 被取消。
- merge queue 之后增加 integration merge preview，能检测冲突、保护路径和需要返工的 issue。
- release batching 先生成建议和阈值判断，不自动 publish。
- Console 能看到 parallel run、worker slot、merge preview 和 release batch readiness。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase12-001` | `parallel-batch-worker-executor` | completed | goroutine worker pool、bounded parallelism、worker slot、fail-fast cancel、稳定 run item ordering | Phase 11 readiness | `backend_owner` + `orchestrator_owner` | `local_shell` batch run 可按安全并发度执行多个 issue，并记录每个 worker 的状态 |
| `phase12-002` | `integration-merge-preview` | completed | integration branch/worktree、merge dry-run、conflict detection、protected path guard | `phase12-001`,`phase11-004` | `git_owner` + `qa_owner` | ready merge queue 可生成 integration merge preview，不执行真实合入 |
| `phase12-003` | `controlled-merge-apply` | completed | approval/env gated merge apply、audit/evidence、失败回滚到 merge preview 状态 | `phase12-002` | `git_owner` + `security_owner` | 只有通过审批和开关后才能把 ready item 合入 integration branch |
| `phase12-004` | `release-batch-readiness` | completed | release batch 阈值、版本分支建议、tag/PR/MR preview、累积量建议 | `phase12-002` | `release_owner` | 系统能建议“积累到多少改动后发版”，并生成 release batch plan |
| `phase12-005` | `console-parallel-merge-surface` | planned | parallel run、worker slot、merge preview、release batch readiness Console 展示 | `phase12-001`,`phase12-004` | `frontend_owner` | Console 可见并发执行和集成合入准备状态 |

## 3. 建议执行顺序

1. 先做 `phase12-001`，因为后续 merge preview 必须依赖真实并发 run 产物。
2. 再做 `phase12-002`，把 merge queue 的 ready item 变成可验证的 integration preview。
3. `phase12-003` 必须在 preview 稳定后做，且默认关闭真实合入。
4. `phase12-004` 在 merge preview 后做，避免 release batch 基于不稳定合入状态。
5. `phase12-005` 最后接 Console，前端只展示后端事实源。

## 4. 收口规则

- 没有独立 worktree，不允许真实并发。
- 并发度不得超过 batch plan runtime slots、用户 `max_issues` 和系统安全上限。
- `continue_on_failure=false` 时，首个失败必须取消尚未开始的 worker，并记录 cancellation reason。
- integration preview 只能基于 ready merge queue，不接受 failed、dry-run 或 needs_rework item。
- 真实 merge apply、PR/MR create、tag/push 和 release publish 必须继续走 approval/authz 和执行开关。
