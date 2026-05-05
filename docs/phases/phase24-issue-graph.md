# Phase 24 实现 Issue Graph

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 24 的目标是为 `server_resource_registry_adapter` 增加最小 apply receipt。该能力只对本地 `.moyuan` server resource registry 相关的 `resource_maintenance` 生效，用于证明 adapter contract 可以完成 apply 闭环；不执行 SSH、Git provider、cloud 或真实服务器写入。

## 1. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase24-001` | `server-resource-registry-apply-receipt` | completed | 对 `server_resource_registry_adapter` apply-ready plan 生成 completed adapter execution | Phase 23 readiness | `backend_owner` + `security_owner` | 本地 receipt 可审计，外部写入仍为 false |
| `phase24-002` | `phase24-readiness` | completed | 收口验证、文档回写和后续真实 adapter 入口 | `phase24-001` | `release_owner` + `qa_owner` | 全量门禁通过，边界清晰 |

## 2. 强制边界

- 只支持 `server_resource_registry_adapter`。
- 只处理 apply-ready write execution plan。
- 必须要求 `MOYUAN_ENABLE_WRITE_ADAPTERS=1`。
- 必须保持 `external_write_attempted=false` 和 `external_write_performed=false`。
- SSH/GitHub/Gitee/cloud adapter 继续 manual handoff。
