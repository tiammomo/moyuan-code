# Phase 18 实现 Issue Graph

状态：in_progress
责任角色：devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 18 的目标是把 Phase 17 的 policy、scheduler、review queue 和 deployment monitor 进一步收敛为“生产运维闭环与策略化维护控制面”。生产真实写入继续默认关闭，所有自动化动作都必须保留 approval、authz、quality、provider gate 和 evidence。

## 1. Phase 18 目标

- Operations timeline 能统一展示 release、deployment、admission、rehearsal、scheduler、risk review、resource health 和 rollback 事实。
- Maintenance policy 能表达维护窗口、冻结期、环境级允许动作和人工复核要求。
- Post-deployment smoke/monitor 能形成受控更新后的线上验证记录。
- Server resource lifecycle 能和部署、监控、续费、退役、健康扫描长期关联。
- Console 提供生产运维 dashboard，但只消费后端事实源和受控动作。

## 2. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase18-001` | `operations-timeline` | completed | 聚合 release/deployment/admission/scheduler/risk/resource 事实为 timeline | Phase 17 readiness | `backend_owner` + `devops_owner` | 统一查询、排序、过滤、可追溯 evidence |
| `phase18-002` | `maintenance-policy-pack` | completed | 维护窗口、冻结期、环境级动作许可和人工复核策略 | `phase18-001` | `security_owner` + `release_owner` | policy 不降低已有门禁，可解释 |
| `phase18-003` | `post-deployment-smoke-monitor-loop` | completed | 发布后 smoke、monitor、rollback suggestion 和 risk review 形成闭环 | `phase18-001` | `qa_owner` + `devops_owner` | 线上验证可审计，失败不自动生产修复 |
| `phase18-004` | `server-resource-lifecycle-control` | completed | 服务器资源到期、续费、退役、健康扫描和部署关系长期维护 | `phase18-001` | `devops_owner` | 测试开发机/生产机区分清晰 |
| `phase18-005` | `console-operations-dashboard` | next | Console 展示 operations timeline、维护策略、资源风险和受控动作 | `phase18-001`,`phase18-004` | `frontend_owner` | 前端只展示事实源，不重新决策 |
| `phase18-006` | `phase18-readiness` | planned | 收口验证、文档回写、剩余风险和 Phase 19 入口 | `phase18-005` | `release_owner` + `security_owner` | 全量门禁通过，生产边界清晰 |

## 3. 建议执行顺序

1. 先做 `phase18-001`，把生产运维事实聚合为统一 timeline。
2. 再做 `phase18-002`，将维护窗口和环境动作许可策略化。
3. `phase18-003` 接线上 smoke/monitor loop，保证投产后反馈能形成复核对象。
4. `phase18-004` 把服务器资源生命周期接入长期维护。
5. `phase18-005` 做 Console dashboard。
6. `phase18-006` 做 readiness 收口。

## 4. 强制边界

- Operations timeline 是事实聚合，不改变任何 release、deployment、resource 或 repair 状态。
- Maintenance policy 不能绕过 approval、authz、quality、review、secret 和 provider gate。
- Smoke/monitor 失败只能生成 post-deployment verification 风险事实和复核入口，不能自动执行生产修复。
- Console 不能自行计算准入、维护窗口、风险复核或回滚结论。
