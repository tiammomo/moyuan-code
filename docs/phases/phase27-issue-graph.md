# Phase 27 实现 Issue Graph

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 27 的目标是把 write adapter recovery record 接入 control queue。系统需要允许 queue item 绑定 `adapter_recovery_id`，并在 queue run 时通过 review gate 阻断自动执行，让 recovery 进入可见、可审计、可人工批准的编排入口。

## 1. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase27-001` | `adapter-recovery-queue-binding` | completed | control queue item 支持 `adapter_recovery_id` | Phase 26 readiness | `backend_owner` | queue item 可持久化绑定 recovery |
| `phase27-002` | `adapter-recovery-review-gate` | completed | queue run 时 recovery 绑定进入 manual review gate | `phase27-001` | `security_owner` + `qa_owner` | 不自动执行 repair/retry/handoff |
| `phase27-003` | `console-queue-recovery-ref` | completed | Console 展示 queue item 的 recovery ref | `phase27-001` | `frontend_owner` | 前端只读展示事实源 |
| `phase27-004` | `phase27-readiness` | completed | 收口验证、文档回写和下一阶段入口 | `phase27-001` + `phase27-002` + `phase27-003` | `qa_owner` | 全量门禁通过 |

## 2. 强制边界

- 绑定 `adapter_recovery_id` 的 queue item 不会自动执行 control run。
- Review gate 必须返回 `CONTROL_QUEUE_RECOVERY_REVIEW_REQUIRED`。
- Queue item 只记录 recovery ref、recovery action 和人工复核原因。
- 后续显式 approval/repair runner 未实现前，不允许 recovery queue 自动触发 repair、retry 或 handoff。
