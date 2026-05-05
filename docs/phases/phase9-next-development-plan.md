# Phase 9 实施记录

状态：in_progress
责任角色：devops_owner + backend_owner + frontend_owner + provider_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 9 的实际执行顺序。Phase 9 的入口以 [Phase 9 实现 Issue Graph](./phase9-issue-graph.md) 为准。

## 1. 当前基线

Phase 8 已完成并通过 release readiness：

- Release provider adapter 支持 GitHub/Gitee create release 的最小受控真实写入。
- SSH runner 支持受控命令执行、命令 allowlist、timeout、输出脱敏和 smoke/monitor/rollback evidence。
- Rollback suggestion 已生成结构化 runbook。
- Console 已支持 Operation Detail 的 evidence chain 展示。
- Provider telemetry 已支持 runtime token 估算、quota 扣减、成本估算和 quality signal。

## 2. Phase 9 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase9-001` | `operation-detail-aggregation-api` | planned | Operation detail 后端聚合 | Console/API 可按 operation id 获取 execution/evidence/artifact detail |
| P0 | `phase9-002` | `server-resource-lifecycle-alerts` | planned | 服务器资源生命周期提醒 | expiring/expired/maintenance_due 可查询并写入 audit |
| P1 | `phase9-003` | `deployment-monitor-history` | planned | 部署后检查历史和失败分类 | smoke/monitor/rollback 可按 execution 查询历史 |
| P1 | `phase9-004` | `provider-route-explanation-v2` | planned | Provider 路由解释增强 | selected/skipped provider signals 可解释 |
| P2 | `phase9-005` | `self-repair-candidate-from-operations` | planned | 从失败 operation 生成修复候选 | repair candidate 可审查，不自动越权 |

## 3. 执行规划：`phase9-001 operation-detail-aggregation-api`

实现状态：planned。

范围：

- 新增 operation detail 聚合结构，统一支持 release provider execution、deployment execution 和 evidence chain。
- API 支持按 operation type/id 查询 detail；找不到时返回 404。
- 聚合结果只返回状态、decision、reasons、artifact references 和脱敏摘要。
- Console 详情区优先使用聚合 API；API 不可用时继续 fallback 到 snapshot。
- 测试覆盖 release provider execution、deployment execution、evidence artifact 和缺失资源。

非目标：

- 不读取完整 stdout/stderr。
- 不暴露 secret、token、SSH key 或 provider response body。
- 不新增真实外部执行能力。

验收：

- `GET /v1/projects/:project_id/operations/:operation_type/:operation_id` 可返回 detail。
- detail 中包含 operation、evidence、artifacts、status、decision 和 reason。
- Console Operation Detail 能展示聚合 detail。
- `go test ./internal/api ./internal/evidence ./internal/release ./internal/deployment`、`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

## 4. 验证要求

每完成一个 Phase 9 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
