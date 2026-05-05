# Phase 18 实施记录

状态：in_progress
责任角色：devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

本文记录 Phase 18 的实际执行顺序。Phase 18 的入口以 [Phase 18 实现 Issue Graph](./phase18-issue-graph.md) 为准。

## 1. 阶段入口

Phase 17 已完成并通过 readiness：

- Release admission 已升级为可解释 policy pack。
- Bounded rehearsal scheduler 已能一次性创建 rehearsal/admission。
- Deployment risk handoff 已进入 review queue。
- Console 已展示 policy、scheduler、risk review 的后端事实源。

Phase 18 不改变生产真实写入默认关闭的原则，重点补生产运维 timeline、维护策略、线上验证和服务器资源长期维护。

## 2. Phase 18 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase18-001` | `operations-timeline` | next | 统一运维事实查询 | timeline 可过滤、可排序、可追溯 evidence |
| P0 | `phase18-002` | `maintenance-policy-pack` | planned | 维护策略包 | 窗口/冻结期/人工复核可解释 |
| P1 | `phase18-003` | `post-deployment-smoke-monitor-loop` | planned | 线上验证闭环 | smoke/monitor 失败进入风险复核 |
| P1 | `phase18-004` | `server-resource-lifecycle-control` | planned | 服务器生命周期控制 | 到期、续费、退役、健康与部署关联 |
| P1 | `phase18-005` | `console-operations-dashboard` | planned | Console 运维 dashboard | 展示 timeline 和资源风险 |
| P2 | `phase18-006` | `phase18-readiness` | planned | Phase 18 收口 | 全量门禁和生产边界完成 |

## 3. 执行规划：`phase18-001 operations-timeline`

实现状态：next。

范围：

- 在 `internal/operations` 增加 timeline 聚合能力。
- 聚合 release provider execution、deployment execution、monitor summary、deployment rehearsal、release admission、scheduler run、risk handoff/review、resource health scan 和 rollback execution。
- 支持 `limit`、`type`、`status`、`decision`、`environment` 过滤。
- API 增加 `GET /v1/projects/:project_id/operations/timeline`。
- CLI 增加 `moyuan operations timeline [--type <type>] [--environment <env>] [--limit 20]`。

非目标：

- 不改写任何业务状态。
- 不启动后台调度。
- 不执行生产命令、Git 写入或 repair attempt。

验收：

- timeline 按时间倒序，缺失时间的记录稳定排序。
- 每条 item 至少包含 `id`、`type`、`status`、`decision`、`primary_ref`、`environment`、`evidence_refs`。
- API、CLI 和单测覆盖 release/deployment/admission/risk/resource 的代表性记录。

## 4. 验证要求

每完成一个 Phase 18 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
