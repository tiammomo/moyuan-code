# Phase 17 实现 Issue Graph

状态：in_progress
责任角色：release_owner + devops_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 17 的目标是把 Phase 16 的 rehearsal、release admission 和 deployment risk handoff 从固定代码判断推进到可配置策略包、受控演练调度和更清晰的风险 drill-down。生产真实写入继续默认关闭。

## 1. Phase 17 目标

- Release admission policy pack 可配置化，支持不同环境、risk class 和 evidence 要求。
- Deployment rehearsal 可由 release pipeline 或 bounded scheduler 触发，但不执行真实生产命令。
- Risk handoff 能进入 review queue，并支持 Console drill-down 到 signal、bug candidate 和 repair plan。
- Console 展示 policy decision、scheduler trigger、risk drill-down 和 review handoff。
- Phase 17 完成后给出 readiness，明确哪些自动化可以进入受控长期运行。

## 2. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase17-001` | `release-admission-policy-pack` | completed | admission 规则从固定判断抽成 policy pack，支持环境级阈值和 signal 规则 | Phase 16 readiness | `release_owner` + `backend_owner` | policy 决策可解释、可测试、可审计 |
| `phase17-002` | `bounded-rehearsal-scheduler` | completed | bounded scheduler 可按 release candidate 或 deployment 自动创建 rehearsal/admission | `phase17-001` | `devops_owner` | 不常驻、不真实执行生产命令、可重复运行 |
| `phase17-003` | `risk-review-queue` | completed | deployment risk handoff 进入 review queue，支持 approved/rejected/defer | `phase16-003` | `qa_owner` | 风险处理有人工决策和审计记录 |
| `phase17-004` | `console-policy-risk-drilldown` | planned | Console 展示 policy rules、scheduler run、risk handoff drill-down | `phase17-001`,`phase17-003` | `frontend_owner` | 前端只展示后端事实源和受控动作 |
| `phase17-005` | `phase17-readiness` | planned | 收口验证、文档回写、剩余风险和下一阶段入口 | `phase17-004` | `release_owner` + `security_owner` | 全量门禁通过，自动化边界清晰 |

## 3. 建议执行顺序

1. 先做 `phase17-001`，把 release admission 的硬编码判断收敛为 policy pack。
2. 再做 `phase17-002`，基于 policy 触发 bounded rehearsal scheduler。
3. `phase17-003` 补风险 review queue，避免自动化只生成对象没人处理。
4. `phase17-004` 接 Console drill-down。
5. `phase17-005` 做 readiness 收口。

## 4. 强制边界

- policy pack 不能降低 approval、authz、quality 和 review 门禁。
- scheduler 只能 bounded run，不启动常驻后台任务。
- risk review 不能直接执行 repair attempt 或生产命令。
- Console 不能自行计算 policy 结论。
