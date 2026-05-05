# Phase 27 实施记录

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + qa_owner + security_owner
最后更新：2026-05-05

本文记录 Phase 27 的实际执行顺序。Phase 27 的入口以 [Phase 27 实现 Issue Graph](./phase27-issue-graph.md) 为准。

## 1. 阶段入口

Phase 26 已完成 write adapter failure recovery record。Phase 27 将 recovery record 接入 control queue，但保持人工复核：系统能排队、展示、审计 recovery 任务，同时 queue run 不会自动执行 repair、retry 或 handoff。

## 2. Phase 27 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase27-001` | `adapter-recovery-queue-binding` | completed | queue item 支持 `adapter_recovery_id` | 可持久化、API/CLI 可传入 |
| P0 | `phase27-002` | `adapter-recovery-review-gate` | completed | recovery queue run 进入 manual review | 不自动执行恢复动作 |
| P1 | `phase27-003` | `console-queue-recovery-ref` | completed | Console 展示 recovery ref | 前端只读展示事实源 |
| P1 | `phase27-004` | `phase27-readiness` | completed | Phase 27 收口 | 全量门禁通过 |

## 3. 完成记录

- `QueueOptions` 和 `QueueItem` 增加 `adapter_recovery_id`。
- API `POST /v1/projects/:project_id/control-loop/queue` 支持 `adapter_recovery_id`。
- CLI `moyuan control-loop queue add` 支持 `--adapter-recovery-id`。
- `queueReviewGateAllows` 会加载 recovery record，确认 open 后仍返回 `CONTROL_QUEUE_RECOVERY_REVIEW_REQUIRED`。
- Console Control Runner 队列展示 recovery ref。

## 4. 验证要求

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
