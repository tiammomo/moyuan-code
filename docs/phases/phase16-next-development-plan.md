# Phase 16 实施记录

状态：in_progress
责任角色：devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

本文记录 Phase 16 的实际执行顺序。Phase 16 的入口以 [Phase 16 实现 Issue Graph](./phase16-issue-graph.md) 为准。

## 1. 阶段入口

Phase 15 已完成并通过 readiness：

- Deployment execution 已具备 approval proof、scope 校验、approval consumption 和 replay guard。
- Rollback execution 已从 suggestion/runbook 推进为受控 preview/gated execution 对象。
- Monitor summary 已能聚合最近窗口的部署运行风险。
- Console 已展示 deployment approval、rollback 和 monitor 事实源。

Phase 16 不改变生产真实写入默认关闭的原则，重点把这些能力串成部署演练和运行风险闭环。

## 2. Phase 16 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase16-001` | `deployment-rehearsal-controller` | completed | 部署演练记录 | rehearsal 可串联 deployment/execution/rollback/monitor/evidence |
| P0 | `phase16-002` | `release-admission-risk-gate` | completed | 发布准入风险门禁 | 输出 allow/block/manual 和 reasons |
| P1 | `phase16-003` | `monitor-risk-repair-bridge` | planned | 风险到修复队列 | critical/rollback risk 进入 repair/maintenance handoff |
| P1 | `phase16-004` | `console-rehearsal-risk-surface` | planned | Console 演练风险面 | 可见 rehearsal timeline 和 admission gate |
| P2 | `phase16-005` | `phase16-readiness` | planned | Phase 16 收口 | 全量门禁和风险边界完成 |

## 3. 执行规划：`phase16-001 deployment-rehearsal-controller`

实现状态：completed。

范围：

- 新增 `DeploymentRehearsal`，以 release candidate、deployment 或 execution 为入口创建演练记录。
- rehearsal 聚合 deployment plan、最近 execution、rollback preview 或 rollback state、monitor summary、post-deployment histories 和 evidence。
- CLI 增加 rehearsal create/show/list。
- API 增加 rehearsal create/list/show。
- rehearsal 默认 preview-only，不触发真实部署、真实 rollback 或远程写入。

非目标：

- 不实现新的 scheduler。
- 不执行生产真实命令。
- 不修改 release candidate 状态。

验收：

- 无 deployment/execution 时生成 blocked rehearsal，并记录原因。
- 有 execution 和 monitor summary 时能生成完整 timeline。
- rollback required 时可引用或生成 rollback preview，但不执行真实 rollback。
- rehearsal 写入 `.moyuan/lifecycle/deployments/rehearsals/`、JSONL、日志和 evidence。

落地结果：

- 新增 `DeploymentRehearsal`、timeline、CLI create/show/list 和 API create/list/show。
- rehearsal 从 candidate/deployment/execution 解析上下文，聚合 deployment plan、post-deployment history、monitor summary、rollback preview 和 evidence。
- rollback required 只会生成或复用 preview，不触发真实 rollback。
- 单测覆盖 rehearsal 关联 execution、monitor、rollback 和 evidence；CLI/API 测试覆盖 create/list/show。

## 4. 执行规划：`phase16-002 release-admission-risk-gate`

实现状态：completed。

范围：

- 新增 `ReleaseAdmission`，以 rehearsal 为主输入，也可从 candidate、deployment 或 execution 自动创建 rehearsal。
- admission 读取 deployment rehearsal、monitor summary、rollback preview、candidate deployment feedback 和 deployment resource status。
- 输出 `allowed`、`manual_required` 或 `blocked`，并记录 signals、reasons 和 evidence。
- CLI 增加 `moyuan release admission create/show/list`。
- API 增加 release admission create/list/show。

非目标：

- 不改变 release candidate、deployment 或 git provider 的真实状态。
- 不绕过 approval、authz、quality 或 review。
- 不主动执行新的服务器健康探测。

验收：

- healthy rehearsal 能输出 `RELEASE_ADMISSION_ALLOWED`。
- blocked/failed execution、critical monitor、rollback required 或不可用 resource 会输出 block/manual。
- admission 写入 `.moyuan/lifecycle/deployments/release-admissions/`、JSONL、日志和 evidence。

落地结果：

- release admission 成为 release/deploy 准入的后端事实源。
- 单测覆盖 risky rehearsal 产生 blocked admission；CLI/API 测试覆盖 create/list/show。

## 5. 验证要求

每完成一个 Phase 16 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
