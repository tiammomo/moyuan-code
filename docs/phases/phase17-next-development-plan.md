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
| P0 | `phase17-001` | `release-admission-policy-pack` | completed | 发布准入策略包 | policy 决策可解释、可测试、可审计 |
| P0 | `phase17-002` | `bounded-rehearsal-scheduler` | completed | 有界演练调度 | 可重复触发 rehearsal/admission，不常驻 |
| P1 | `phase17-003` | `risk-review-queue` | planned | 风险复核队列 | handoff 可 approved/rejected/defer |
| P1 | `phase17-004` | `console-policy-risk-drilldown` | planned | Console drill-down | 展示 policy、scheduler 和 risk review |
| P2 | `phase17-005` | `phase17-readiness` | planned | Phase 17 收口 | 全量门禁和自动化边界完成 |

## 3. 执行规划：`phase17-001 release-admission-policy-pack`

实现状态：completed。

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

完成记录：

- 新增内置 `release-admission-default-v1` policy pack，覆盖 rehearsal、monitor、rollback、candidate feedback 和 resource status。
- 支持 `.moyuan/policies/release.yaml` 中的 `release_admission_policy_pack` 扩展；自定义规则只能追加更严格规则，不移除内置安全门禁。
- Release admission 输出 `policy_id`、`policy_version`、`policy_source`、`matched_rules` 和 `policy_decision`。
- CLI 增加 `moyuan release admission policy [--environment <env>]`。
- API 增加 `GET /v1/projects/:project_id/release-admission-policy?environment=<env>`。
- 已补后端单测覆盖 monitor critical、execution failed、rollback required、healthy path 和 production strict policy。

## 4. 执行规划：`phase17-002 bounded-rehearsal-scheduler`

实现状态：completed。

范围：

- 新增 bounded rehearsal scheduler，基于 release candidate、deployment 或最新 execution 触发 rehearsal/admission。
- scheduler 只执行一次有界 run，不启动常驻后台任务。
- 输出 scheduler run 记录，包含 trigger、selected targets、skipped reason、created rehearsal/admission id。
- 默认仍然不执行真实生产命令，只复用 rehearsal/admission 的既有受控入口。

验收：

- 可重复运行，重复目标会给出 skipped/linked 解释。
- candidate/deployment/execution 三类触发入口都有明确选择规则。
- 单测覆盖无目标、已有 rehearsal/admission、创建成功和 blocked admission。

完成记录：

- 新增 bounded rehearsal scheduler，一次 run 最多处理 10 个目标，默认 3 个目标。
- 支持 execution、deployment、candidate 和 latest executions 四类受控触发。
- 重复运行会复用已有 rehearsal/admission，并输出 `admission_already_exists` 或 `rehearsal_already_exists`。
- Scheduler run 写入 `.moyuan/lifecycle/deployments/rehearsal-scheduler-runs/`、JSONL、release log 和 evidence。
- CLI 增加 `moyuan deploy rehearsal schedule ...`、`moyuan deploy rehearsal-scheduler <run-id>` 和 `moyuan deploy rehearsal-schedulers`。
- API 增加 `POST/GET /v1/projects/:project_id/deployment-rehearsal-scheduler-runs` 和详情查询。
- 已补后端单测覆盖无目标、重复跳过、execution/deployment/candidate 触发和 blocked admission。

## 5. 执行规划：`phase17-003 risk-review-queue`

实现状态：next。

范围：

- 将 deployment risk handoff 接入 review queue，支持 `approved`、`rejected`、`deferred`。
- review 决策只记录治理状态和下一步建议，不直接执行 repair attempt 或生产命令。
- 风险复核需要保留 source admission/monitor、signal、bug candidate、repair plan 和 evidence 关联。
- API/CLI 可查看待复核风险、提交复核结果和审计记录。

验收：

- allowed admission 不产生必处理 review。
- blocked/manual admission 对应 handoff 可进入 pending review。
- approved/rejected/deferred 都会写入审计、日志和可查询记录。
- 单测覆盖重复 review、防止非法状态跳转和 evidence 关联。

## 6. 验证要求

每完成一个 Phase 17 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
