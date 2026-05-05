# Phase 26 实施记录

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + qa_owner + security_owner
最后更新：2026-05-05

本文记录 Phase 26 的实际执行顺序。Phase 26 的入口以 [Phase 26 实现 Issue Graph](./phase26-issue-graph.md) 为准。

## 1. 阶段入口

Phase 25 已完成 SSH adapter sandbox 和 rollback binding。Phase 26 在此基础上补齐失败恢复事实：任何 write adapter execution 若进入 blocked、manual_required 或 failed，都必须生成结构化 recovery record，为后续 repair、retry 和 handoff 编排提供单一事实源。

## 2. Phase 26 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase26-001` | `write-adapter-recovery-record` | completed | 自动生成 adapter recovery record | failure class、动作建议、外部写入标记完整 |
| P0 | `phase26-002` | `write-adapter-recovery-api-cli` | completed | API/CLI/Timeline 查询 recovery | 可查询、可审计、可测试 |
| P1 | `phase26-003` | `console-write-adapter-recovery` | completed | Console 展示 recovery 摘要 | 前端只读展示事实源 |
| P1 | `phase26-004` | `phase26-readiness` | completed | Phase 26 收口 | 全量门禁通过 |

## 3. 完成记录

- 新增 `WriteAdapterRecoveryRecord`、list/load、summary 和持久化目录。
- `finishWriteAdapterExecution` 会自动为 blocked、manual_required、failed adapter execution 记录 recovery。
- Recovery 分类覆盖 SSH sandbox、rollback binding、adapter switch、execution plan、adapter implementation 和 generic handoff。
- 增加 `GET /v1/projects/:project_id/operations/write-adapter-recoveries`。
- 增加 `moyuan operations write-adapter-recoveries list ...`。
- Operations Timeline 增加 `write_adapter_recovery` 类型。
- Console Operations 面板新增 Write Adapter Recovery 摘要。

## 4. 验证要求

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
