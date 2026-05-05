# Phase 15 实现 Issue Graph

状态：in_progress
责任角色：devops_owner + release_owner + backend_owner + frontend_owner + security_owner + qa_owner
最后更新：2026-05-05

Phase 15 的目标是把 Phase 14 已接通的发布部署执行链路继续加固到“可进入生产演练”的水平。本阶段继续保持生产真实写入默认关闭，优先补齐 deployment execution 的 approval proof、approval consumption、rollback execution、持续 monitor 和生产风险摘要。

## 1. Phase 15 目标

- Deployment execution 不再接受裸 `approved=true` 作为真实执行凭证，必须携带匹配 scope 的 `approval_id`，并在真实执行前消费。
- 回退从 suggestion/runbook 进入受控 rollback execution，默认 preview，真实回退必须审批、写开关和证据齐备。
- 生产 monitor 从一次性反馈扩展为可记录的持续检查摘要，为 release candidate 和 deployment 提供统一风险视图。
- Console 展示审批、执行、回退和 monitor 的后端事实源，不在前端自行判断生产结论。
- Phase 15 完成后给出 readiness 结论，明确哪些部署能力可以进入受控演练，哪些仍保持阻断。

## 2. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase15-001` | `deployment-execution-approval-proof` | planned | deployment execute/candidate execute 增加 `approval_id`、scope 校验、消费和 replay guard | Phase 14 readiness | `security_owner` + `devops_owner` + `backend_owner` | local_shell/ssh_execute 真实执行前必须消费已批准 approval，复用失败会阻断 |
| `phase15-002` | `rollback-execution-controller` | planned | rollback runbook preview、approval request、真实回退开关、执行记录和 evidence | `phase15-001` | `devops_owner` + `release_owner` | 失败部署可生成受控 rollback execution，默认不真实执行 |
| `phase15-003` | `production-monitor-loop` | planned | deployment monitor history、连续失败归因、风险摘要和 candidate feedback 聚合 | `phase15-001` | `devops_owner` + `qa_owner` | monitor 不只停留在单次报告，可查询最近窗口结论 |
| `phase15-004` | `console-deployment-ops-surface` | planned | Console 展示 approval id、rollback preview、monitor window 和生产风险摘要 | `phase15-002`,`phase15-003` | `frontend_owner` | Console 只提交受控请求并展示后端事实源 |
| `phase15-005` | `phase15-readiness` | planned | 收口验证、文档回写、剩余风险和下一阶段入口 | `phase15-004` | `release_owner` + `security_owner` | 全量门禁通过，生产写入默认关闭的边界清晰 |

## 3. 建议执行顺序

1. 先做 `phase15-001`，把真实部署执行的 approval proof 与 consumption 补齐，避免 Phase 14 的裸 `approved` 继续向后扩散。
2. 再做 `phase15-002`，让 rollback 从建议和 runbook 变成可审计、可审批、可阻断的执行对象。
3. `phase15-003` 补充持续 monitor 和风险摘要，为是否发版、是否回退提供事实输入。
4. `phase15-004` 最后接 Console，确保前端只消费后端事实源。
5. `phase15-005` 做 readiness 收口，确认测试、文档和安全边界。

## 4. 强制边界

- 真实 deployment execution 必须同时满足 approval、scope match、approval consumption、环境策略和执行开关。
- `approval_id` 只能用于一次真实执行，已消费 approval 不能重放。
- production real execution 继续默认阻断，除非后续策略明确开放并有独立开关。
- rollback execution 不能直接复用 deploy command，必须从 rollback runbook 或受控模板进入。
- monitor 和 risk summary 只能作为事实输入，不能绕过 release/deploy/review 门禁。
