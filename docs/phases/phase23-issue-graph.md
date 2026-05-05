# Phase 23 实现 Issue Graph

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 23 的目标是把 Phase 22 的 guarded write execution plan 接入 adapter dispatch scaffold。系统需要能根据 operation/provider 选择目标 adapter，记录执行前 guard，生成 adapter execution 事实，并进入 evidence 和 operations timeline。本阶段仍不直接执行 GitHub/Gitee、SSH、cloud 或 server mutation。

## 1. Phase 23 目标

- 基于 write execution plan 生成 write adapter execution。
- 按 operation/provider 推导 adapter id，例如 release provider、SSH deployment、resource registry。
- Preview mode 输出 adapter preview ready，不执行外部写入。
- Apply mode 只做 guard 检查和人工 handoff；真实 adapter 实现留给后续阶段。
- Console 能展示 adapter execution、guard result 和 external write 标记。

## 2. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase23-001` | `write-adapter-dispatch-scaffold` | completed | 新增 write adapter execution create/list/load 和 adapter 推导 | Phase 22 readiness | `backend_owner` + `security_owner` | 不执行外部写入，guard 可解释 |
| `phase23-002` | `write-adapter-api-cli` | completed | 增加 API/CLI create/list 入口 | `phase23-001` | `backend_owner` | 可审计、可测试、失败可解释 |
| `phase23-003` | `console-write-adapter-execution` | completed | Console 展示 adapter execution | `phase23-002` | `frontend_owner` | 前端只读展示事实源 |
| `phase23-004` | `phase23-readiness` | completed | 收口验证、文档回写和后续真实 adapter 入口 | `phase23-003` | `release_owner` + `qa_owner` | 全量门禁通过，真实写入边界清晰 |

## 3. 强制边界

- Phase 23 不直接执行外部 provider、SSH、cloud 或 server mutation。
- Adapter execution 必须记录 `external_write_attempted=false` 和 `external_write_performed=false`，除非后续真实 adapter 阶段显式改变。
- Apply mode 缺少 ready execution plan、adapter 开关或真实 adapter 实现时必须 blocked/manual。
- 后续真实 adapter 必须复用本阶段 adapter execution contract。
