# Phase 26 实现 Issue Graph

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 26 的目标是为 write adapter 增加 failure recovery record。系统需要在 adapter execution 进入 blocked、manual_required 或 failed 后自动生成恢复事实，明确 failure class、repair/retry/handoff 建议，并进入 Timeline、API、CLI 和 Console。

## 1. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase26-001` | `write-adapter-recovery-record` | completed | blocked/manual/failed adapter execution 自动生成 recovery record | Phase 25 readiness | `backend_owner` + `security_owner` | failure class 和恢复动作可解释 |
| `phase26-002` | `write-adapter-recovery-api-cli` | completed | 增加 recovery list API/CLI 和 Timeline 集成 | `phase26-001` | `backend_owner` | 可查询、可审计、可测试 |
| `phase26-003` | `console-write-adapter-recovery` | completed | Console 展示 recovery 摘要 | `phase26-002` | `frontend_owner` | 前端只读展示事实源 |
| `phase26-004` | `phase26-readiness` | completed | 收口验证、文档回写和下一阶段入口 | `phase26-001` + `phase26-002` + `phase26-003` | `qa_owner` | 全量门禁通过，边界清晰 |

## 2. 强制边界

- Recovery record 只记录恢复建议，不直接执行 repair、retry 或 handoff。
- Recovery record 必须保留源 adapter execution 的 status、decision、reasons、外部写入标记和 evidence refs。
- `external_write_attempted` 和 `external_write_performed` 不得被 recovery record 改写。
- 后续自动 retry/repair 必须消费 recovery record，而不能直接读取失败文本后自行决策。
