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
| P0 | `phase14-002` | `release-candidate-pr-mr-create-bridge` | planned | Candidate 接入 PR/MR create | 远程 PR/MR 创建由审批和写开关控制 |
| P1 | `phase14-003` | `candidate-deployment-execution-bridge` | planned | Candidate 接入 deployment execution | 支持 dry-run、ssh preview、local shell、ssh execute 的受控触发 |
| P1 | `phase14-004` | `post-deploy-smoke-monitor-feedback` | planned | 线上检查回写 | smoke/monitor/rollback 结果成为可查询证据 |
| P2 | `phase14-005` | `console-release-execution-surface` | planned | Console 执行流水线 | Console 展示发布、部署和线上检查状态 |

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

## 4. 后续执行占位

`phase14-002` 之后的实际落地结果在对应 issue 完成后补充，稳定设计会回写到 release、git provider、deployment 和 Console 相关主线文档。

## 5. 验证要求

每完成一个 Phase 14 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```

涉及真实或模拟远程写入时，还必须补充对应 adapter 单测，证明 approval、secret、write switch 和 replay guard 均未被绕过。
