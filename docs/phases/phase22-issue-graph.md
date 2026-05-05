# Phase 22 实现 Issue Graph

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 22 的目标是基于 Phase 21 的 ready review packet 建立“受保护的真实写入执行计划契约”。本阶段先实现 preview/apply contract、审批/开关校验、审计和 Console 展示；即使进入 apply-ready，也不在本阶段直接执行外部 provider、SSH 或 cloud 命令。

## 1. Phase 22 目标

- 新增 guarded write execution plan，唯一输入为 review packet。
- Preview 模式生成可审查执行计划，不需要真实写入开关。
- Apply 模式必须同时满足 ready packet、approval id、显式写开关和 replay guard。
- 所有 plan 都必须持久化、写入 evidence，并明确 `external_write_performed=false`。
- Console 能展示 execution plan 的 mode、status、decision、approval、gate reasons 和 external write 标记。

## 2. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase22-001` | `guarded-write-execution-plan` | completed | 新增 write execution plan preview/apply 契约 | Phase 21 readiness | `backend_owner` + `security_owner` | Apply 需要 packet、approval、write switch，且不执行外部写入 |
| `phase22-002` | `write-execution-api-cli` | completed | 增加 API/CLI create/list 入口 | `phase22-001` | `backend_owner` | 可审计、可测试、失败可解释 |
| `phase22-003` | `console-write-execution-plan` | completed | Console 展示 execution plan | `phase22-002` | `frontend_owner` | 只读展示后端事实源 |
| `phase22-004` | `phase22-readiness` | completed | 收口验证、文档回写和后续真实 adapter 入口 | `phase22-003` | `release_owner` + `qa_owner` | 全量门禁通过，真实写入边界清晰 |

## 3. 建议执行顺序

1. 先做 `phase22-001`，定义执行计划契约和安全门禁。
2. 再做 `phase22-002`，暴露 API/CLI。
3. `phase22-003` 做 Console 只读展示。
4. `phase22-004` 做 readiness 收口。

## 4. 强制边界

- Phase 22 不直接执行 GitHub/Gitee publish、SSH command、cloud operation 或 server mutation。
- Apply 模式缺少 ready packet、approval id 或 `MOYUAN_ALLOW_REAL_WRITE=1` 时必须 blocked/manual。
- Plan 必须记录 `external_write_performed=false`，避免误导用户以为生产变更已发生。
- 后续真实 adapter 必须在 Phase 23+ 单独实现，并复用本阶段契约。
