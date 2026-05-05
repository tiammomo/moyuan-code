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
| P0 | `phase18-001` | `operations-timeline` | completed | 统一运维事实查询 | timeline 可过滤、可排序、可追溯 evidence |
| P0 | `phase18-002` | `maintenance-policy-pack` | completed | 维护策略包 | 窗口/冻结期/人工复核可解释 |
| P1 | `phase18-003` | `post-deployment-smoke-monitor-loop` | completed | 线上验证闭环 | smoke/monitor 失败进入风险复核 |
| P1 | `phase18-004` | `server-resource-lifecycle-control` | next | 服务器生命周期控制 | 到期、续费、退役、健康与部署关联 |
| P1 | `phase18-005` | `console-operations-dashboard` | planned | Console 运维 dashboard | 展示 timeline 和资源风险 |
| P2 | `phase18-006` | `phase18-readiness` | planned | Phase 18 收口 | 全量门禁和生产边界完成 |

## 3. 执行规划：`phase18-001 operations-timeline`

实现状态：completed。

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

完成记录：

- `internal/operations` 新增 timeline 聚合能力，覆盖 release provider execution、deployment execution、rollback execution、monitor summary、deployment rehearsal、release admission、scheduler run、risk handoff/review、resource health scan、maintenance、lifecycle alert 和 server resource。
- API 增加 `GET /v1/projects/:project_id/operations/timeline`，支持 `type`、`status`、`decision`、`environment` 和 `limit`。
- CLI 增加 `moyuan operations timeline ...`。
- `serverresources` 增加 health scan 列表读取能力，供 timeline 聚合使用。
- 单测覆盖 timeline 聚合、过滤、API 查询和 CLI 查询。

## 4. 执行规划：`phase18-002 maintenance-policy-pack`

实现状态：completed。

范围：

- 增加 maintenance policy pack，表达环境级维护窗口、冻结期、允许动作和人工复核要求。
- policy 只输出 explainable decision，不直接执行部署、回滚、repair 或资源变更。
- API/CLI 可查询当前环境的 maintenance policy 和最近 policy decision。

验收：

- production/test_dev 可配置不同维护窗口和冻结期。
- policy 不能降低现有 approval、authz、quality、review、secret 和 provider gate。
- 单测覆盖窗口内、窗口外、冻结期、人工复核 required 和未知环境。

完成记录：

- `internal/serverresources` 新增 maintenance policy pack，默认策略区分 `test_dev` 和 `production`。
- 支持在 `.moyuan/policies/server-resources.yaml` 配置 `maintenance_policy_pack`，包含 `maintenance_windows`、`freeze_windows`、`allowed_actions`、`manual_required_actions`、`blocked_actions` 和 `outside_window_effect`。
- API 增加 `GET /v1/projects/:project_id/resources/maintenance-policy`。
- CLI 增加 `moyuan resources maintenance policy ...`。
- policy 只输出 explainable decision，不执行部署、回滚、repair 或资源变更。
- 单测覆盖默认策略、配置策略、冻结期、窗口外和 API/CLI 查询。

## 5. 执行规划：`phase18-003 post-deployment-smoke-monitor-loop`

实现状态：completed。

范围：

- 将 deployment execution 的 smoke report、monitor report、post-deployment history 和 risk handoff 进一步串联。
- 增加 post-deployment verification run 事实对象，聚合 smoke、monitor、rollback suggestion 和 evidence。
- verification 失败只生成风险事实和复核入口，不自动执行生产修复。

验收：

- 可按 execution/deployment/environment 查询 verification run。
- smoke/monitor failure 能生成可审计 risk handoff 或复核建议。
- 单测覆盖 healthy、smoke failed、monitor attention、rollback required。

完成记录：

- `internal/deployment` 新增 post-deployment verification 事实对象，串联 deployment execution、post-deployment history、monitor summary、rollback suggestion 和 evidence。
- verification 输出 `completed`、`attention_required`、`failed` 或 `blocked`；失败和关注态只设置 `risk_handoff_recommended`，不自动创建 repair attempt 或执行生产修复。
- API 增加 `POST /v1/projects/:project_id/post-deployment-verifications`、`GET /v1/projects/:project_id/post-deployment-verifications` 和 `GET /v1/projects/:project_id/post-deployment-verifications/:verification_id`。
- CLI 增加 `moyuan deploy verify create/list/show`。
- Operations timeline 增加 `post_deployment_verification` item，便于 Console 和运维视图统一展示线上验证结论。
- 单测覆盖健康 deployment verification、失败 verification、API 创建/查询和 CLI create/list/show。

## 6. 执行规划：`phase18-004 server-resource-lifecycle-control`

实现状态：next。

范围：

- 增强 server resource 生命周期事实对象，统一表达到期时间、续费记录、退役计划、健康扫描和部署关联。
- 区分 `test_dev`、`staging`、`production` 的资源约束和维护策略。
- 资源状态只能作为部署、维护和告警判断输入，不直接触发真实云厂商写操作。

验收：

- 可查询资源到期、续费、退役、健康扫描和最近部署引用。
- production 资源过期、未知健康或退役中时，部署计划必须给出阻断或人工复核原因。
- Operations timeline 能展示资源生命周期风险。
- 单测覆盖测试开发机、生产机、到期、续费、退役和健康扫描边界。

## 7. 验证要求

每完成一个 Phase 18 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
