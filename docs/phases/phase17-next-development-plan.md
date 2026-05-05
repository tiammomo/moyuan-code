# Phase 17 实施记录

状态：in_progress
责任角色：release_owner + devops_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

本文记录 Phase 17 的实际执行顺序。Phase 17 的入口以 [Phase 17 实现 Issue Graph](./phase17-issue-graph.md) 为准。

## 1. 阶段入口

Phase 16 已完成并通过 readiness：

- Deployment rehearsal 已能串联 execution、monitor、rollback 和 evidence。
- Release admission 已能输出 allowed、manual_required 或 blocked。
- Deployment risk handoff 已能进入 self-repair 的 signal、bug candidate 和 repair plan。
- Console 已展示 rehearsal、admission 和 risk handoff。

Phase 17 不改变生产真实写入默认关闭的原则，重点补策略包、bounded scheduler 和风险 drill-down。

## 2. Phase 17 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase17-001` | `release-admission-policy-pack` | planned | 发布准入策略包 | policy 决策可解释、可测试、可审计 |
| P0 | `phase17-002` | `bounded-rehearsal-scheduler` | planned | 有界演练调度 | 可重复触发 rehearsal/admission，不常驻 |
| P1 | `phase17-003` | `risk-review-queue` | planned | 风险复核队列 | handoff 可 approved/rejected/defer |
| P1 | `phase17-004` | `console-policy-risk-drilldown` | planned | Console drill-down | 展示 policy、scheduler 和 risk review |
| P2 | `phase17-005` | `phase17-readiness` | planned | Phase 17 收口 | 全量门禁和自动化边界完成 |

## 3. 执行规划：`phase17-001 release-admission-policy-pack`

实现状态：planned。

范围：

- 新增 release admission policy pack，先提供内置默认策略。
- policy rule 覆盖 monitor status、rehearsal status、rollback required、candidate feedback 和 resource status。
- admission 输出附带 policy id、matched rules 和 decision reasons。
- CLI/API 可查看默认 policy 和最近 admission 的 policy decision。

非目标：

- 不开放生产真实执行。
- 不允许 policy 降低已有 approval/authz/quality/review 门禁。
- 不做复杂规则 DSL，先用结构化 rule。

验收：

- 默认 policy 与 Phase 16 现有 decision 行为兼容。
- 每条 block/manual/allow 结论都能解释匹配的 rule。
- 单测覆盖 monitor critical、execution failed、rollback required、healthy path。

## 4. 验证要求

每完成一个 Phase 17 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
