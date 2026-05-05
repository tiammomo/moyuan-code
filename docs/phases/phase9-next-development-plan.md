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
| P0 | `phase9-001` | `operation-detail-aggregation-api` | completed | Operation detail 后端聚合 | Console/API 可按 operation id 获取 execution/evidence/artifact detail |
| P0 | `phase9-002` | `server-resource-lifecycle-alerts` | completed | 服务器资源生命周期提醒 | expiring/expired/maintenance_due 可查询并写入 audit |
| P1 | `phase9-003` | `deployment-monitor-history` | completed | 部署后检查历史和失败分类 | smoke/monitor/rollback 可按 execution 查询历史 |
| P1 | `phase9-004` | `provider-route-explanation-v2` | completed | Provider 路由解释增强 | selected/skipped provider signals 可解释 |
| P2 | `phase9-005` | `self-repair-candidate-from-operations` | planned | 从失败 operation 生成修复候选 | repair candidate 可审查，不自动越权 |

## 3. 执行规划：`phase9-001 operation-detail-aggregation-api`

实现状态：completed。

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

落地结果：

- 新增 `internal/operations` 聚合层，支持 `release_provider`、`deployment` 和 `evidence` operation detail。
- 新增 API：`GET /v1/projects/:project_id/operations/:operation_type/:operation_id`。
- 聚合结果返回 execution 摘要、evidence chain 和 artifact references，不返回完整 stdout/stderr、secret、SSH key 或 provider response body。
- Console snapshot 会拉取近期 operation detail，详情区优先展示 detail API 的 evidence chain，API 不可用时回退到原 snapshot。

## 4. 执行规划：`phase9-002 server-resource-lifecycle-alerts`

实现状态：completed。

范围：

- 增加资源生命周期 scan，统一识别 `RESOURCE_EXPIRING`、`RESOURCE_EXPIRED`、`RESOURCE_MAINTENANCE_DUE` 和 `RESOURCE_HEALTH_ATTENTION`。
- 生命周期 scan 会更新资源 `expiration_state`，写入 lifecycle alert JSONL、scan report、maintenance record 和 audit log。
- API 支持 `POST /v1/projects/:project_id/resources/lifecycle/scan` 和 `GET /v1/projects/:project_id/resources/lifecycle-alerts`。
- Console Server Resources 面板展示 lifecycle alerts、expiration state 和维护记录。

非目标：

- 不连接云厂商 API，不自动续费，不修改云资源。
- 不执行 SSH 或生产部署。

落地结果：

- 新增 `LifecycleScan`、`ListLifecycleAlerts`、`LifecycleScanReport` 和 `LifecycleAlert`。
- `maintenance_window` 支持 `due:YYYY-MM-DD` 或 `YYYY-MM-DD` 形式的 due 判断；其他自然语言窗口只保留为信息，不强行判断。
- 生命周期提醒会沉淀到 `.moyuan/resources/lifecycle-alerts.jsonl` 和 `.moyuan/resources/lifecycle-scans/`。

## 5. 执行规划：`phase9-003 deployment-monitor-history`

实现状态：completed。

范围：

- 每次 deployment execution 完成后生成 post-deployment history，统一记录 smoke/monitor 检查、失败分类、rollback 状态、evidence ids 和 artifact references。
- API 支持查询最近 history 和按 execution id 查询单个 history。
- Console Deployment Executions 面板展示 smoke/monitor 状态、rollback suggested 和 post-deployment history 摘要。
- history 不读取完整 stdout/stderr，不包含 secret、SSH key 或生产远程响应正文。

非目标：

- 不自动执行 rollback。
- 不扩大 production real execution 权限。
- 不替代监控平台，只沉淀本系统可审计的 post-deployment 结果。

落地结果：

- 新增 `PostDeploymentHistory`、`PostDeploymentCheck` 和 `RollbackHistory`，产物路径为 `.moyuan/lifecycle/deployments/post-deployment-history/<execution-id>.json`。
- 新增 API：`GET /v1/projects/:project_id/deployment-monitor-history` 和 `GET /v1/projects/:project_id/deployment-executions/:execution_id/post-deployment-history`。
- 失败分类包括 `smoke_failed`、`monitor_failed`、`execution_failed`、`execution_blocked`、`manual_check_required` 和 `none`。

## 6. 执行规划：`phase9-004 provider-route-explanation-v2`

实现状态：completed。

范围：

- `provider-route` 保持原有 `route.decision/provider_id/runtime_id/signals`，新增 `route.explanation` 和 `route.candidates`。
- 候选 provider 按 `selected`、`skipped`、`blocked` 分类，给出 reason、score 和 health/quota/cost/quality/selection signals。
- explanation 汇总 selected reason、strategy、candidate count、selected/skipped/blocked count。
- API 测试覆盖 route explanation 字段；provider 单测覆盖 selected、skipped 和 blocked candidate。

非目标：

- 不改变 route policy 的安全边界。
- 不让前端自行重算 provider 决策。
- 不记录 prompt、模型响应、secret、token 或 provider 原始响应正文。

落地结果：

- 新增 `RouteExplanation` 和 `RouteCandidate`。
- 路由候选理由覆盖 disabled、repo/runtime mismatch、memory API provider mismatch、health/quota/cost/data policy 阻断和低优先级跳过。
- `candidate.signals` 追加 `selection` signal，便于 Console/日志解释本次路由为什么选或跳过。

## 7. 验证要求

每完成一个 Phase 9 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
