# Phase 14 实施记录

状态：in_progress
责任角色：release_owner + git_owner + devops_owner + backend_owner + frontend_owner + security_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 14 的实际执行顺序。Phase 14 的入口以 [Phase 14 实现 Issue Graph](./phase14-issue-graph.md) 为准。

## 1. 阶段入口

Phase 13 已完成并通过 readiness：

- Release batch readiness 可生成 release candidate。
- Release candidate 可受控准备本地 release branch。
- Release candidate 可生成 provider preview 和 PR/MR handoff。
- Release candidate 可交接 deployment plan。
- Console 已能展示并触发 candidate、branch apply、provider preview 和 deployment handoff。

Phase 14 只推进受控执行层，不改变“生产写入默认关闭”的原则。

## 2. Phase 14 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase14-001` | `release-candidate-provider-publish-bridge` | completed | Candidate 接入 release provider publish | 默认 approval/preview-only 阻断，满足门禁后复用 provider adapter |
| P0 | `phase14-002` | `release-candidate-pr-mr-create-bridge` | completed | Candidate 接入 PR/MR create | 远程 PR/MR 创建由审批和写开关控制 |
| P1 | `phase14-003` | `candidate-deployment-execution-bridge` | completed | Candidate 接入 deployment execution | 支持 dry-run、ssh preview、local shell、ssh execute 的受控触发 |
| P1 | `phase14-004` | `post-deploy-smoke-monitor-feedback` | completed | 线上检查回写 | smoke/monitor/rollback 结果成为可查询证据 |
| P2 | `phase14-005` | `console-release-execution-surface` | completed | Console 执行流水线 | Console 展示发布、部署和线上检查状态 |

## 3. 执行规划：`phase14-001 release-candidate-provider-publish-bridge`

实现状态：completed。

范围：

- 新增 release candidate provider publish 后端入口。
- 复用现有 release provider publish 的 approval、write switch、secret resolver、provider adapter、evidence 和 replay guard。
- publish 前检查 candidate ready、本地 release branch apply completed、provider preview completed。
- API 提供 `POST /v1/projects/:project_id/release-candidates/:candidate_id/provider-publish`。
- execution 结果仍归入 release provider execution 事实源，并用 `candidate_id` 标记来源。

非目标：

- 不默认打开 `MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE`。
- 不在本 issue 新增 PR/MR create。
- 不执行 deployment。
- 不在 Console 做前端入口。

验收：

- 未审批时返回 approval required。
- 已审批但写开关未开时返回 preview-only 阻断，approval 不被消费。
- 未完成 provider preview 时阻断。
- 已有 release provider adapter 测试仍通过。

落地结果：

- 新增 `release.ProviderPublishForCandidate`，从 release candidate 构造受控 provider publish 执行上下文。
- publish 前强制检查 candidate ready、本地 release branch apply completed、provider preview completed。
- `ProviderExecution` 增加 `candidate_id`，相关 evidence 的 subject 指向 `release_candidate`。
- API 新增 `POST /v1/projects/:project_id/release-candidates/:candidate_id/provider-publish`。
- 真实远程写入仍复用 `MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE`、approval、secret resolver 和 replay guard。

## 4. 执行规划：`phase14-002 release-candidate-pr-mr-create-bridge`

实现状态：completed。

范围：

- 新增 release candidate -> Git Provider PR/MR plan 桥接。
- 生成稳定 `git_provider_plan` ID，确保 approval target 可复用。
- PR/MR 创建仍复用既有 `gitprovider.Create` 的 approval、`MOYUAN_ALLOW_GIT_PROVIDER_WRITE`、secret resolver 和 replay guard。
- API 新增 `POST /v1/projects/:project_id/release-candidates/:candidate_id/pr-mr-plan`。

非目标：

- 不在 candidate endpoint 里直接执行远程 PR/MR create。
- 不默认打开 Git Provider 远程写入。
- 不绕过 `git-provider-plans/:plan_id/create` 的既有高风险门禁。

验收：

- candidate 未 ready 或 release branch apply 未完成时生成 blocked plan。
- ready candidate 可生成 `pr_mr_plan_ready`。
- PR/MR create 未审批时要求 approval，已审批但写开关未开时 preview-only 且 approval 不被消费。

落地结果：

- `gitprovider.PlanReleaseCandidate` 负责生成候选发布分支到 default branch 的 PR/MR plan。
- `gitprovider.Plan` 增加 `candidate_id`，审批 metadata 带上 candidate 来源。
- API 可从 release candidate 生成 Git Provider plan，后续真实创建继续走已有高风险 create endpoint。

## 5. 执行规划：`phase14-003 candidate-deployment-execution-bridge`

实现状态：completed。

范围：

- 新增 release candidate -> deployment execution 桥接。
- 支持通过 candidate 自动选择最近 deployment plan，也支持显式传入 deployment id。
- 继续复用 `deployment.Execute` 的 dry-run、ssh preview、local shell、ssh execute 和线上检查逻辑。
- API 新增 `POST /v1/projects/:project_id/release-candidates/:candidate_id/deployment-execution`。

非目标：

- 不新增新的部署执行器。
- 不默认放开 production real execution。
- 不绕过已有 deployment plan readiness。

验收：

- candidate 未 ready 时 execution 阻断。
- candidate 已有 ready deployment plan 时可触发 dry-run execution。
- deployment id 与 candidate 不匹配时阻断。

落地结果：

- `deployment.ExecuteFromCandidate` 负责 candidate 到 deployment execution 的受控桥接。
- `deployment.LatestPlanForCandidate` 支持按 candidate 和环境选择最近 plan。
- API 可从 candidate 触发 deployment execution，执行结论仍以 deployment 模块事实源为准。

## 6. 执行规划：`phase14-004 post-deploy-smoke-monitor-feedback`

实现状态：completed。

范围：

- 聚合 candidate 相关 deployment post-deployment history。
- 输出最新 execution、deployment、environment、status、failure class、severity、rollback 和 evidence 摘要。
- API 新增 `GET /v1/projects/:project_id/release-candidates/:candidate_id/deployment-feedback`。

非目标：

- 不重复执行 smoke 或 monitor。
- 不修改 deployment execution 的判定逻辑。
- 不自动执行 rollback。

验收：

- candidate 无 execution 时返回 pending 反馈。
- candidate 有通过的 smoke/monitor 时返回 healthy 反馈。
- failed、blocked、manual_required 和 skipped 状态有明确 candidate 级 decision。

落地结果：

- `deployment.FeedbackForCandidate` 提供 candidate 级部署反馈事实源。
- Candidate feedback 复用 deployment post-deployment history 和 evidence，不产生第二套线上检查记录。
- API 可直接查询 release candidate 的投产反馈，供 Console 和后续策略使用。

## 7. 执行规划：`phase14-005 console-release-execution-surface`

实现状态：completed。

范围：

- Console snapshot 接入 candidate deployment feedback。
- Candidate 卡片可触发 provider publish gate、PR/MR plan 和 deployment dry-run。
- Console 展示 candidate deployment feedback、rollback suggested 和环境摘要。
- Provider publish、PR/MR plan 和 deployment execution 均只调用后端受控 API。

非目标：

- 不在前端执行真实 Git、Provider 或部署命令。
- 不在前端计算发布、PR/MR 或部署 readiness。
- 不在前端消费 approval。

验收：

- Console live snapshot 能读取 candidate feedback。
- Demo snapshot 能展示 Phase 14 执行流水线样例。
- 新增按钮只调用后端 API，成功后刷新 server snapshot。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

落地结果：

- `ConsoleSnapshot` 增加 `deployment_feedback`。
- Console 可见 release candidate 从 provider publish、PR/MR plan 到 deployment execution/feedback 的受控链路。

## 8. 后续执行占位

Phase 14 第一批任务完成后，应进入 release readiness 收口，确认远程写入仍默认关闭、Console 不计算权威结论，并规划后续部署 approval hardening、release rollback execution 或生产可观测性增强。

## 9. 验证要求

每完成一个 Phase 14 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```

涉及真实或模拟远程写入时，还必须补充对应 adapter 单测，证明 approval、secret、write switch 和 replay guard 均未被绕过。
