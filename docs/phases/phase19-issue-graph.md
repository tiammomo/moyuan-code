# Phase 19 实现 Issue Graph

状态：in_progress
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 19 的目标是把 Phase 18 已形成的运维事实源继续推进到“受控自动化执行增强与生产可观测性深化”。生产真实写入继续默认关闭，所有自动化动作都必须保留 approval、authz、quality、secret、provider gate、evidence 和 audit trail。

## 1. Phase 19 目标

- Audit export 能把 operations timeline、release/deployment/maintenance/resource 事实打包为可审查报告。
- Decision ledger 能统一记录 policy、readiness、verification、review 和 risk handoff 的可解释结论。
- Durable control runner 能为受控自动化任务提供幂等、重试、失败恢复和审计状态。
- Provider write proof 能让真实 Git/部署/云资源写入继续保持 dry-run、approval proof 和最小权限契约。
- Console 提供生产可观测性 drill-down，但只消费后端事实源和受控动作。

## 2. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase19-001` | `operations-audit-export` | next | 导出 operations timeline、verification、resource refs 和关键 evidence 为 JSON/Markdown 报告 | Phase 18 readiness | `backend_owner` + `devops_owner` | 可过滤、可复盘、无 secret 泄漏 |
| `phase19-002` | `decision-ledger` | planned | 统一记录 release admission、maintenance policy、resource readiness、verification 和 risk review decision | `phase19-001` | `release_owner` + `security_owner` | 结论结构一致，可追溯规则和来源 |
| `phase19-003` | `durable-control-runner` | planned | 为 bounded scheduler、resource scan、verification 和 audit export 增加幂等 run、retry budget 和失败恢复状态 | `phase19-002` | `orchestrator_owner` + `devops_owner` | 重试可控，重复触发不产生重复写入 |
| `phase19-004` | `provider-write-proof-contract` | planned | 加固真实 Git/部署/云资源写入的 dry-run、approval proof、provider evidence 和最小权限契约 | `phase19-003` | `security_owner` + `release_owner` | 生产写入仍默认关闭，放行条件可审计 |
| `phase19-005` | `console-observability-drilldown` | planned | Console 展示 audit export、decision ledger、control runner 和 provider write proof | `phase19-001`,`phase19-004` | `frontend_owner` | 前端只展示事实源，不重新决策 |
| `phase19-006` | `phase19-readiness` | planned | 收口验证、文档回写、剩余风险和 Phase 20 入口 | `phase19-005` | `release_owner` + `security_owner` | 全量门禁通过，生产边界清晰 |

## 3. 建议执行顺序

1. 先做 `phase19-001`，把 Phase 18 的运维事实打包成可审计报告。
2. 再做 `phase19-002`，统一 decision ledger，避免各模块 decision 语义分散。
3. `phase19-003` 增强长期控制任务的幂等、重试和失败恢复。
4. `phase19-004` 收紧真实写入 proof contract，为生产写入打基础。
5. `phase19-005` 做 Console drill-down。
6. `phase19-006` 做 readiness 收口。

## 4. 强制边界

- Audit export 只能读取事实和生成报告，不能修改 release、deployment、resource、repair 或 approval 状态。
- Decision ledger 是统一记录和解释，不替代原模块的权威状态机。
- Durable control runner 不能绕过原有 approval、authz、quality、secret、provider 和 protected path 门禁。
- Provider write proof contract 不能把生产真实写入改成默认开启。
- Console 不能自行计算准入、维护窗口、资源就绪、verification 或真实写入结论。
