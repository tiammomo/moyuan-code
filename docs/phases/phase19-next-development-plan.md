# Phase 19 实施记录

状态：in_progress
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

本文记录 Phase 19 的实际执行顺序。Phase 19 的入口以 [Phase 19 实现 Issue Graph](./phase19-issue-graph.md) 为准。

## 1. 阶段入口

Phase 18 已完成并通过 readiness：

- Operations timeline 已统一聚合 release、deployment、admission、scheduler、risk、resource 和 verification 事实。
- Maintenance policy pack 已能输出环境级维护窗口、冻结期、人工复核和可解释 decision。
- Post-deployment verification 已把线上验证失败收敛为风险事实和复核建议。
- Server resource lifecycle 已接入部署关联、健康、到期、续费和退役风险。
- Console operations dashboard 已展示后端事实源和受控 API 动作。

Phase 19 不改变生产真实写入默认关闭的原则，重点补审计导出、统一决策账本、长期控制 runner、真实写入 proof contract 和 Console observability drill-down。

## 2. Phase 19 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase19-001` | `operations-audit-export` | completed | 运维审计报告导出 | JSON/Markdown 可复盘且无 secret 泄漏 |
| P0 | `phase19-002` | `decision-ledger` | completed | 统一决策账本 | policy/readiness/verification/review 结论结构一致 |
| P1 | `phase19-003` | `durable-control-runner` | next | 长期控制任务 runner | 幂等、重试、失败恢复可审计 |
| P1 | `phase19-004` | `provider-write-proof-contract` | planned | 真实写入 proof contract | dry-run、approval proof、provider evidence 完整 |
| P1 | `phase19-005` | `console-observability-drilldown` | planned | Console 可观测性 drill-down | 展示 audit、decision、runner 和 proof |
| P2 | `phase19-006` | `phase19-readiness` | planned | Phase 19 收口 | 全量门禁和生产边界完成 |

## 3. 执行规划：`phase19-001 operations-audit-export`

实现状态：completed。

范围：

- 在后端增加 operations audit export 能力，基于 timeline 和关键运维事实生成 JSON/Markdown 报告。
- 支持按 environment、type、status、decision、limit 过滤。
- 报告包含导出元数据、摘要、timeline items、verification 摘要、resource deployment refs 和 evidence refs。
- API 增加受控导出入口，CLI 增加本地导出命令。

非目标：

- 不修改任何业务状态。
- 不读取 secret 明文，不把 auth token、API key、SSH key 写入报告。
- 不启动后台任务，不执行生产命令、Git 写入或 repair attempt。

验收：

- JSON export 可被机器读取，Markdown export 可供 release/ops review。
- 报告中的 evidence refs 保留引用，不内联敏感内容。
- API、CLI 和单测覆盖过滤、空结果、Markdown 渲染和 secret-like 文本脱敏。

完成记录：

- `internal/operations` 新增 operations audit export，复用 timeline 过滤参数，并汇总 timeline、post-deployment verification、resource deployment refs 和 evidence refs。
- API 增加 `GET /v1/projects/:project_id/operations/audit-export`，支持 `format=json|markdown`、`type`、`status`、`decision`、`environment` 和 `limit`。
- CLI 增加 `moyuan operations audit-export ...`。
- 导出文本统一使用 secret redaction，Markdown 只作为报告字段返回，不执行外部写入或生产动作。
- 单测覆盖 Markdown 导出、类型过滤、API 查询、CLI 查询和 secret-like 文本脱敏。

## 4. 执行规划：`phase19-002 decision-ledger`

实现状态：completed。

范围：

- 设计统一 decision ledger，将 release admission、maintenance policy、resource readiness、post-deployment verification、risk handoff/review 的结论转为同一类可审计 decision entry。
- 每条 entry 保留 source type、source id、environment、status、decision、reasons、rule/policy/ref 信息和 evidence refs。
- ledger 只记录和解释，不反向改写原模块权威状态。

验收：

- 可按 source type、decision、environment 查询 decision entries。
- 现有 policy/readiness/verification/review 结论能被聚合到 ledger。
- 单测覆盖无数据、重复来源去重、过滤和 evidence refs 保留。

完成记录：

- `internal/operations` 新增 decision ledger，统一聚合 release admission、maintenance policy、resource readiness、post-deployment verification、deployment risk handoff 和 deployment risk review。
- API 增加 `GET /v1/projects/:project_id/operations/decision-ledger`，支持 `source_type`、`status`、`decision`、`environment` 和 `limit`。
- CLI 增加 `moyuan operations decision-ledger ...`。
- 修正 secret redaction 对 `risk-review` 类普通 ID 的误伤，避免审计对象 ID 被错误脱敏，同时继续脱敏真实 `sk-...` token。
- 单测覆盖 ledger 聚合、过滤、API/CLI 查询和 redaction 边界。

## 5. 执行规划：`phase19-003 durable-control-runner`

实现状态：next。

范围：

- 增加 durable control runner 的最小只读/受控执行底座，统一记录 control run、step、idempotency key、retry budget、started/finished 状态和失败原因。
- 首批接入低风险目标：operations audit export、decision ledger refresh、resource health/lifecycle scan、post-deployment verification dry-run 触发。
- runner 不绕过原模块权限、审批、secret、provider 和 quality gate；失败只进入可审计状态和后续复核入口。

验收：

- 同一个 idempotency key 重复触发不会产生重复 run。
- retry budget 耗尽后状态进入 failed/manual_required，不能无限重试。
- API、CLI 和单测覆盖 plan/run/list/show、幂等、失败恢复和禁止生产写入边界。

## 6. 验证要求

每完成一个 Phase 19 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
