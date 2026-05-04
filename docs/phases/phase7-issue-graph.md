# Phase 7 实现 Issue Graph

状态：in_progress
责任角色：release_manager + devops_owner + security_owner + provider_owner + frontend_owner + qa_owner
最后更新：2026-05-05

Phase 7 的目标是在 Phase 6 的 preview、approval、telemetry 和 Console 操作基础上，推进“受控真实外部执行准备”。本阶段仍不默认打开生产真实执行，而是把远程写入开关、审批消费、执行证据、回滚证据和 Console 追踪补成可验证闭环。

## 1. Phase 7 目标

- Release provider publish 在真实远程写入前必须消费 approval，并具备 replay guard、写开关和 preview-only 降级。
- Deployment SSH executor 具备命令 allowlist、secret resolver 注入边界、执行记录和默认阻断策略。
- 发布和部署流水线形成统一 post-action evidence，覆盖 smoke、monitor、rollback suggestion 和失败恢复。
- Provider telemetry 能吸收 runtime execution、quality gate 和 route decision 的结果，影响后续调度。
- Console 从多视图工作台推进到 execution detail、operation history 和 schema-driven forms。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase7-001` | `release-provider-approval-consumption` | planned | release provider publish 的 approval consumption、replay guard 和写开关策略 | Phase 6 readiness | `release_manager` + `security_owner` | 真实 publish 路径不能重复使用 approval，默认仍可 preview-only 降级 |
| `phase7-002` | `ssh-executor-guarded-runner` | planned | SSH executor 的 allowlist、secret resolver 注入、执行记录和 blocked-by-default 策略 | `phase7-001` | `devops_owner` + `security_owner` | 未启用真实执行时只记录 blocked，启用后只允许白名单命令 |
| `phase7-003` | `post-action-evidence-model` | planned | release/deploy/smoke/monitor/rollback evidence 统一数据模型和查询入口 | `phase7-001`,`phase7-002` | `qa_owner` + `devops_owner` | 每次 publish/deploy 都能查询 evidence chain |
| `phase7-004` | `runtime-telemetry-feedback-loop` | planned | runtime、quality、provider route 结果写入 provider telemetry 反馈 | Phase 6 readiness | `provider_owner` + `orchestrator_owner` | 失败率、成本和健康信号可影响后续 provider route |
| `phase7-005` | `console-execution-detail-history` | planned | Console execution detail、operation history 和 schema metadata 接入 | `phase7-003` | `frontend_owner` | 用户能追踪一次操作从 preview 到 evidence 的完整链路 |

## 3. 建议执行顺序

1. 先做 `phase7-001`，release provider 是最小外部写入闭环，范围比 SSH 低。
2. `phase7-002` 在 release 写入策略稳定后推进，避免 SSH executor 提前扩大风险面。
3. `phase7-003` 把 release/deploy/smoke/monitor 证据统一，成为后续生产门禁的事实源。
4. `phase7-004` 把执行结果反哺 provider routing，让系统越用越能调整模型和 runtime。
5. `phase7-005` 最后接 Console detail/history，避免前端先固化未稳定的数据模型。

## 4. 收口规则

- 任何真实外部写入必须同时满足 authz、approval consumption、secret resolver、write switch、audit event 和 replay guard。
- 未启用真实写入时，系统必须返回 preview-only 或 blocked，并给出可解释 reason。
- 所有 execution 必须能追溯到 project、issue/release/deployment、actor、approval、provider、resource 和 evidence。
- Console 只展示和提交后端状态，不自行判定真实执行成功。
