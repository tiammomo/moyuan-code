# Phase 15 实施记录

状态：in_progress
责任角色：devops_owner + release_owner + backend_owner + frontend_owner + security_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 15 的实际执行顺序。Phase 15 的入口以 [Phase 15 实现 Issue Graph](./phase15-issue-graph.md) 为准。

## 1. 阶段入口

Phase 14 已完成并通过 readiness：

- Release candidate 可以触发受控 provider publish、PR/MR plan 和 deployment execution。
- Deployment execution 已支持 dry-run、ssh preview、local shell、ssh execute、smoke、monitor 和 rollback suggestion。
- Candidate feedback 已能聚合 deployment post-deployment history。
- Console 已展示 release candidate 的发布和部署执行流水线。

Phase 15 不改变生产真实写入默认关闭的原则，重点补齐执行门禁、回退执行和持续监控。

## 2. Phase 15 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase15-001` | `deployment-execution-approval-proof` | completed | 部署真实执行审批凭证化 | 真实执行必须携带并消费匹配 approval |
| P0 | `phase15-002` | `rollback-execution-controller` | completed | 回退执行控制器 | rollback 默认 preview，真实执行 gated |
| P1 | `phase15-003` | `production-monitor-loop` | completed | 持续 monitor 与风险摘要 | 最近窗口 monitor 可查询并聚合到反馈 |
| P1 | `phase15-004` | `console-deployment-ops-surface` | planned | Console 运维操作面 | 展示 approval、rollback、monitor 事实源 |
| P2 | `phase15-005` | `phase15-readiness` | planned | Phase 15 收口 | 全量门禁和风险清单完成 |

## 3. 执行规划：`phase15-001 deployment-execution-approval-proof`

实现状态：completed。

范围：

- `deployment.ExecuteOptions`、API 和 CLI 增加 `approval_id`。
- deployment execution 对真实执行 mode 校验 approval scope。
- approval target 使用稳定 deployment/mode scope，避免每次 execution id 变化导致无法验证。
- local_shell 和 ssh_execute 在真正执行命令前消费 approval。
- candidate deployment execution 透传 `approval_id`。
- 单测覆盖未传 approval、scope mismatch、消费成功和复用阻断。

非目标：

- 不开放 production real execution。
- 不新增 rollback execution。
- 不改变 dry-run 或 ssh preview 的低风险行为。

验收：

- `approved=true` 但无 `approval_id` 时阻断。
- 已批准且 scope 匹配的 approval 在真实执行前被消费。
- 已消费 approval 再次执行被阻断。
- CLI/API/candidate bridge 行为一致。

落地结果：

- `deployment.ExecuteOptions`、`CandidateExecuteOptions`、API request 和 CLI 均支持 `approval_id`。
- approval target 固定为 deployment plan id，action 固定为 `deploy.execute.<mode>` 的归一化形式，避免瞬时 execution id 造成 scope 不稳定。
- `local_shell` 和 `ssh_execute` 只在命令 allowlist、资源和执行开关通过后消费 approval，阻断路径不消耗审批。
- `Execution` 增加 `approval_consumed`，execution evidence 和 post-deployment history 可追溯审批消费结果。
- 单测覆盖 approval request、missing proof、消费成功、复用阻断、CLI 审批执行和 unsafe command 不消费审批。

## 4. 执行规划：`phase15-002 rollback-execution-controller`

实现状态：completed。

范围：

- 新增 `RollbackExecution`，从 failed deployment execution 的 rollback runbook 进入独立回退执行对象。
- 支持 `preview` 和 `local_shell` 两种 mode；默认 `preview`，只展示 runbook 步骤并写 evidence。
- `local_shell` 真实回退必须满足 approval、`approval_id`、`MOYUAN_ALLOW_ROLLBACK_EXECUTE=1`、命令 allowlist 和 approval consumption。
- CLI 增加 `moyuan deploy rollback <execution-id>` 和 `moyuan deploy rollback-execution <rollback-execution-id>`。
- API 增加 `POST /deployment-executions/:execution_id/rollback`、rollback execution list/show。

非目标：

- 不接入 SSH rollback。
- 不开放 production 自动回退。
- 不从前端直接执行回退命令。

验收：

- 失败部署可生成 rollback preview。
- 未审批时生成 approval request。
- 审批通过但执行开关未开时 preview-only 且不消费 approval。
- 执行开关开启后，安全命令执行前消费 approval；复用已消费 approval 会阻断。
- CLI/API 路由、日志和 evidence 均可追溯 rollback execution。

落地结果：

- rollback execution 写入 `.moyuan/lifecycle/deployments/rollback-executions/` 和 `rollback-executions.jsonl`。
- evidence parent type 为 `deployment_rollback_execution`，operation 为 `deployment.rollback.execute.<mode>`。
- `ROLLBACK_EXECUTION_PREVIEW_ONLY` 明确表达“已审批但执行开关未打开”，不消耗审批。
- 单测覆盖 preview、approval request、preview-only、approval consumption、replay guard、CLI missing path 和 API list/show path。

## 5. 执行规划：`phase15-003 production-monitor-loop`

实现状态：completed。

范围：

- 新增 `MonitorSummary`，从 post-deployment history 聚合最近窗口的部署健康状态。
- 输出 `healthy`、`attention_required`、`critical`、`unknown` 状态和对应 decision。
- 统计 history count、failed、blocked、manual、rollback、failure class 和 latest histories。
- CLI 增加 `moyuan deploy monitor summarize [--environment <env>] [--limit <n>]` 和 `moyuan deploy monitor-summary <id>`。
- API 增加 `POST /deployment-monitor-summary` 和 `GET /deployment-monitor-summaries`。

非目标：

- 不接入真实 APM/Prometheus/日志平台。
- 不在本 issue 做定时调度器。
- 不让 monitor summary 绕过 release/deploy/review 门禁。

验收：

- 无 history 时输出 `DEPLOYMENT_MONITOR_NO_HISTORY`。
- 最近窗口存在失败或 rollback 时输出 attention/critical。
- production 环境失败时输出 `PRODUCTION_MONITOR_CRITICAL`。
- summary 写入 `.moyuan/lifecycle/deployments/monitor-summaries/`、JSONL、日志和 evidence。
- CLI/API 可生成和查询 monitor summary。

落地结果：

- monitor summary 成为 Console 和后续 release readiness 的统一生产风险事实源。
- `deployment_monitor_summary` evidence 记录 summary artifact、decision 和 reasons。
- 单测覆盖窗口聚合、失败分类、rollback 计数、evidence、CLI summarize/show 和 API list path。

## 6. 后续执行占位

完成 `phase15-003` 后，应进入 `phase15-004 console-deployment-ops-surface`，让 Console 展示 approval、rollback preview、monitor summary 和生产风险摘要，但继续只调用后端受控 API。

## 7. 验证要求

每完成一个 Phase 15 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```

涉及真实或模拟外部写入时，还必须补充 replay guard、approval consumption、secret redaction 和 write switch 的单测。
