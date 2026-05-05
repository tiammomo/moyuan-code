# Phase 25 实现 Issue Graph

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 25 的目标是为 `ssh_deployment_adapter` 增加执行前 sandbox 和 rollback binding。该阶段只消费已有 deployment execution 的 `RemotePlan`、命令、AuthRef 和 rollback plan，生成可审计的 adapter execution 事实；仍不执行真实 SSH、Git provider、cloud 或服务器写入。

## 1. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase25-001` | `ssh-adapter-sandbox` | completed | 对 `ssh_deployment_adapter` 增加 target/auth/command sandbox | Phase 24 readiness | `backend_owner` + `security_owner` | 危险命令阻断，preview-only 不触发远程写入 |
| `phase25-002` | `ssh-rollback-binding` | completed | 将 deployment rollback plan/runbook 绑定到 adapter execution | `phase25-001` | `devops_owner` + `release_owner` | 缺失 rollback 被阻断或转人工 |
| `phase25-003` | `console-adapter-sandbox-view` | completed | Console 展示 sandbox 和 rollback binding 摘要 | `phase25-001` | `frontend_owner` | 前端只读展示事实源 |
| `phase25-004` | `phase25-readiness` | completed | 收口验证、文档回写和下一阶段入口 | `phase25-001` + `phase25-002` + `phase25-003` | `qa_owner` | 全量门禁通过，边界清晰 |

## 2. 强制边界

- `ssh_deployment_adapter` 只生成 sandbox、guard、rollback binding 和 evidence，不执行 SSH。
- `sandbox_results` 必须记录 target、command、allowlist、preview_only 和 no_remote_write。
- `rollback_binding` 必须绑定 deployment rollback plan；已有 rollback suggestion 时必须绑定 runbook。
- `external_write_attempted=false` 和 `external_write_performed=false` 必须保持。
- 后续真实 SSH adapter 必须复用 Phase 25 输出的 sandbox 与 rollback binding。
