# Phase 10 实施记录

状态：completed
责任角色：orchestrator_owner + backend_owner + frontend_owner + devops_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 10 的实际执行顺序。Phase 10 的入口以 [Phase 10 实现 Issue Graph](./phase10-issue-graph.md) 为准。

## 1. 当前基线

Phase 9 已完成并通过 release readiness：

- Operation detail 聚合 API 已支持 release provider、deployment 和 evidence operation。
- 服务器资源 lifecycle scan 已支持到期、维护和健康提醒。
- Deployment post-deployment history 已记录 smoke/monitor/rollback 结果和失败分类。
- Provider route explanation v2 已返回 selected、skipped、blocked candidates 和 signals。
- Operation repair candidate 可从失败 operation 生成，但默认只进入 review-only 状态。

## 2. Phase 10 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase10-001` | `background-control-loop-scheduler` | completed | 控制循环调度底座 | 一次 run 可触发多个受控 step，并写入状态、日志和审计 |
| P0 | `phase10-002` | `operation-repair-candidate-review-flow` | completed | 修复候选复核流转 | approve/reject 后可生成 issue 或受控 repair attempt |
| P1 | `phase10-003` | `release-provider-branch-tag-workflow-preview` | completed | release provider 扩展预览 | branch/tag/workflow 有 preview plan 和 guardrail |
| P1 | `phase10-004` | `deployment-check-template-policy` | completed | 部署检查模板策略 | smoke/monitor 失败分级可配置、可追踪 |
| P2 | `phase10-005` | `console-route-repair-operator-surfaces` | completed | Console 操作面增强 | route candidates、repair review、control loop history 可见 |

## 3. 执行规划：`phase10-001 background-control-loop-scheduler`

实现状态：completed。

范围：

- 新增 project-scoped control loop run 结构，记录 `pending`、`running`、`succeeded`、`failed` 和 `skipped` step。
- 第一批 step 支持资源 lifecycle scan、Provider telemetry/ops refresh 的受控刷新入口，以及 project comprehension refresh hook。
- API 支持手动触发和查询最近 control loop run。
- 每个 step 必须有 timeout、错误摘要、artifact/evidence 引用和 audit log。
- Console 可先通过 snapshot 或后续 `phase10-005` 展示最近运行摘要。

非目标：

- 不新增生产真实写入。
- 不自动批准 repair candidate。
- 不默认后台常驻定时器；第一批先提供可手动触发的 bounded run，为后续 scheduler 做事实源。
- 不保存完整 stdout/stderr、secret、token 或模型响应正文。

验收：

- `POST /v1/projects/:project_id/control-loop/run` 可创建一次 bounded run。
- `GET /v1/projects/:project_id/control-loop/runs` 可查看最近 runs。
- run 中每个 step 都有 status、started_at、finished_at、summary 和 evidence/artifact references。
- 资源 lifecycle scan 可以作为 control loop step 被调用。
- `go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

落地结果：

- 新增 `internal/controlloop`，默认安全 step 为 `resource_lifecycle_scan`、`provider_ops_refresh` 和 `project_comprehension_refresh`。
- 新增 API：`POST /v1/projects/:project_id/control-loop/run`、`GET /v1/projects/:project_id/control-loop/runs`、`GET /v1/projects/:project_id/control-loop/runs/:run_id`。
- 每个 step 都写入 `control_loop` evidence，并记录 artifact references；run 写入 `.moyuan/control-loop/runs/` 和 `.moyuan/control-loop/runs.jsonl`。
- `provider_ops_refresh` 默认不 probe 外部服务；如果启用 probe 且未审批，会沿用 provider approval guard。
- 第一批只提供手动 bounded run，不启动后台常驻定时器，不新增生产真实写入。

## 4. 执行规划：`phase10-002 operation-repair-candidate-review-flow`

实现状态：completed。

范围：

- Operation repair candidate 支持 approve/reject 复核。
- approve 后默认创建 repair issue，并写入 `repair-epic` issue graph。
- approve 且 `next_step=repair_attempt` 时，只创建 `review_ready` repair attempt，不执行 runtime。
- reject 会关闭候选，并记录 reviewer、reason 和审计日志。
- API 支持候选 review，列表接口返回去重后的最新候选状态。

非目标：

- 不自动运行修复 runtime。
- 不绕过 repair plan、issue graph、quality gate、review 和 approval。
- 不把 operation candidate 直接合入开发分支。

落地结果：

- 新增 API：`POST /v1/projects/:project_id/repair/operation-candidates/:candidate_id/review`。
- 新增产物：`.moyuan/repair/operation-candidate-reviews/`、`.moyuan/repair/issues/`、`.moyuan/repair/repair-issues.jsonl`。
- `repair_attempt.status=review_ready` 表示进入人工复核后的受控准备态，仍不代表代码已修改。
- `operation-candidates` 列表按 candidate id 去重，展示最新 review 状态。

## 5. 执行规划：`phase10-003 release-provider-branch-tag-workflow-preview`

实现状态：completed。

范围：

- Release provider remote plan 中的 `push_branch`、`create_tag`、`push_tag`、`create_release` 和 `trigger_workflow` 均带结构化 `risk_level`、`execution_mode` 和 `guardrails`。
- Preview 能明确标记 branch/tag/workflow 动作仍是受控计划，不代表真实远程写入。
- Publish 中被跳过的 branch/tag/workflow action result 会保留 guardrails，方便审计和 Console 展示。

非目标：

- 不执行真实 branch push、tag push 或 workflow dispatch。
- 不扩大 `MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE=1` 的真实写入范围。
- 不消费 approval 来执行未启用的 branch/tag/workflow 动作。

落地结果：

- `ProviderAction` 增加 `risk_level`、`execution_mode` 和 `guardrails`。
- `ProviderActionResult` 增加 `guardrails`。
- branch/tag/workflow 预览固定包含 approval、write switch、replay guard、secret ref 和动作特定检查项。

## 6. 执行规划：`phase10-004 deployment-check-template-policy`

实现状态：completed。

范围：

- Deployment plan 的 smoke/monitor step 引用默认检查模板，包含 `template_id`、`severity`、`window` 和 `failure_classes`。
- Deployment execution 的 smoke/monitor report 继承模板信息，并在失败时写入 `failure_class`。
- Post-deployment history 保留每个检查项的模板、severity 和 failure class；失败 history 会汇总失败 severity。
- release 日志和 evidence reasons 能追踪检查模板、严重度和失败归类。

非目标：

- 不引入外部监控系统适配器。
- 不自动执行生产 rollback。
- 不改变 SSH execute、local shell 和 healthcheck 的执行安全边界。

落地结果：

- 新增 `CheckTemplate`，默认模板为 `deploy-smoke-<env>-v1` 和 `deploy-monitor-<env>-v1`。
- 非生产 smoke severity 为 `high`，monitor severity 为 `medium`；production 默认为 `critical`。
- `smoke_failed`、`monitor_failed` 和 `manual_check_required` 可在 report/history 中被审计。

## 7. 执行规划：`phase10-005 console-route-repair-operator-surfaces`

实现状态：completed。

范围：

- Console Providers 面板增加 Provider route preview，可查看 selected、skipped、blocked candidate、score、runtime/model 和 route reason。
- Runtime Recoveries 面板中的 operation repair candidates 增加 approve/reject 操作，approve 默认进入 `repair_attempt` 准备态。
- Console 新增 Control Loop Runs 面板，可触发一次 bounded control loop，并展示每个 step 的状态、summary、duration 和 evidence 数量。
- Deployment post-deployment history 在 Console 中展示 smoke/monitor template 和 severity 摘要。

非目标：

- 不在前端自行计算 route、repair 或 control loop 结论。
- 不绕过后端 approval、review、repair attempt 和 production 写入控制。
- 不启动后台常驻 scheduler；Console 只触发一次 bounded run。

落地结果：

- `ConsoleSnapshot` 增加 `control_loop_runs`。
- Console 的 Provider route preview、repair review 和 control loop run 均调用后端受控 API。
- 前端 demo snapshot 同步补充 control loop 示例，保证无后端时仍能展示结构。

## 8. 验证要求

每完成一个 Phase 10 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
