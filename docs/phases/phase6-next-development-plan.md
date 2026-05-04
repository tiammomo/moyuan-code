# Phase 6 实施记录

状态：in_progress
责任角色：orchestrator_owner + security_owner + devops_owner + git_owner + provider_owner + frontend_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 6 的实际执行顺序。Phase 6 的入口以 [Phase 6 实现 Issue Graph](./phase6-issue-graph.md) 为准。

## 1. 当前基线

Phase 5 已完成并通过 readiness：

- API authz middleware 已保护高风险写操作。
- Secret Resolver 已作为 adapter 获取凭证的唯一入口。
- GitHub/Gitee PR/MR create 已要求 approval proof、secret resolver 和写开关。
- Deployment 已能记录 smoke、monitor 和 rollback suggestion。
- Console 已具备受控操作表单。

## 2. Phase 6 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase6-001` | `approval-consumption-replay-guard` | completed | approval record 消费和重放防护 | 已消费 approval 不能再次触发真实外部写入 |
| P1 | `phase6-002` | `deployment-ssh-preview-adapter` | completed | 部署 adapter preview/dry-run/execute 状态模型 | 生产真实执行继续默认关闭 |
| P1 | `phase6-003` | `ci-cd-release-provider-adapter` | completed | release/tag/workflow provider adapter | 远程发布动作可审计且可降级 |
| P1 | `phase6-004` | `provider-cost-health-telemetry` | planned | Provider quota/cost/health 反馈 | 路由能读取 provider 健康和预算信号 |
| P2 | `phase6-005` | `console-routes-schema-forms` | planned | Console 多页面和 schema-aware forms | 表单错误和 execution 状态以后端为准 |

## 3. 执行规划：`phase6-001 approval-consumption-replay-guard`

范围：

- `approvals` 增加 consume 能力，已批准 record 可被真实外部写操作消费。
- 消费后 approval status 进入 `consumed`，后续 `VerifyApproved` 不再通过。
- Git Provider 远程 PR/MR create 在调用 provider API 前消费 approval record。
- 消费事件写入 audit log。

非目标：

- 不实现 approval 自动过期策略。
- 不实现审批多人会签。
- 不改变 dry-run、preview-only 或 production deployment 的默认阻断边界。

验收：

- 同一 approval id 无法重复创建远程 PR/MR。
- preview-only 不消费 approval。
- `go test ./internal/approvals ./internal/gitprovider ./internal/cli ./internal/api` 通过。
- `go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

## 4. 已完成任务：`phase6-001 approval-consumption-replay-guard`

范围：

- `approvals` 增加 `ConsumeApproved`，将已批准 record 标记为 `consumed` 并写入 `approval.consumed` audit event。
- `VerifyApproved` 对 `consumed` approval 不再通过。
- Git Provider 远程 PR/MR create 在真正调用 provider API 前消费 approval record。
- preview-only 和写开关未开启场景不消费 approval。
- 回归测试覆盖 approval 消费和同一 approval id 重放阻断。

非目标：

- 不实现 approval 自动过期和多人会签。
- 不改变 deployment 生产真实执行默认阻断策略。

验证：

- `go test ./internal/approvals ./internal/gitprovider ./internal/cli ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 5. 执行规划：`phase6-002 deployment-ssh-preview-adapter`

范围：

- Deployment execute 增加 `ssh_preview` 模式，读取 deployment plan 引用的 server resources，生成远程目标、host、provider、auth_ref 和预览命令。
- `ssh_preview` 不触发真实 SSH、不解析 secret 明文、不要求 approval，可用于 `test_dev`、`staging` 和 `production` 的投产前审阅。
- Deployment execute 增加 `ssh_execute` 状态入口，但真实 SSH 执行继续默认阻断。
- 预览结果写入 deployment execution，并记录 `deployment.ssh.previewed` 日志事件。

非目标：

- 不实现真实 SSH command runner。
- 不解析 SSH 私钥或服务器凭证明文。
- 不改变 production real execution 默认阻断策略。
- 不替代 smoke、monitor 和 rollback 的后续闭环。

验收：

- `ssh_preview` 生成 `remote_plan`，每个 target 必须包含 resource id、environment、host、provider、auth_ref、status 和 commands。
- `ssh_execute` 返回 blocked，并给出 `ssh_real_execution_not_enabled`。
- `deployment.ssh.previewed` 写入 release 日志。
- `go test ./internal/deployment` 通过。
- `go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

## 6. 已完成任务：`phase6-002 deployment-ssh-preview-adapter`

范围：

- `deployment.Execute` 支持 `ssh_preview` 和 `ssh_execute` 两种部署 adapter 状态。
- `ssh_preview` 产出 `remote_plan`，仅记录 server resource reference 和预览命令。
- `ssh_execute` 保留为 blocked 状态入口，避免提前开启真实远程执行。
- CLI usage 已同步新的 deploy execute mode。
- 测试覆盖 SSH preview 和真实 SSH execution blocked。

非目标：

- 不引入 SSH client 依赖。
- 不访问远程服务器。
- 不把 secret value 写入 execution 或 log。

验证：

- `go test ./internal/deployment` 通过。
- `go test ./internal/deployment ./internal/cli ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 7. 执行规划：`phase6-003 ci-cd-release-provider-adapter`

范围：

- Release 增加 provider execution，用于记录 GitHub/Gitee release、tag 和 workflow dispatch 的 preview/publish 状态。
- `release provider preview` 生成 `push_branch`、`create_tag`、`push_tag`、`create_release` 和 `trigger_workflow` action plan。
- `release provider publish` 必须要求 approval proof；默认不真实写远程，返回 preview-only 降级结果。
- API 增加 release provider preview/publish/execution 查询入口，publish 受 `release:write` scope 保护。
- 发布 provider execution 写入 `.moyuan/lifecycle/releases/provider-executions/`，并记录 release 日志。

非目标：

- 不在本任务中真实调用 GitHub/Gitee release 或 workflow API。
- 不执行 `git push`、`git tag` 或 workflow dispatch。
- 不消费 approval record，因为本任务默认不触发真实远程写入。

验收：

- preview 能形成远程 release/tag/workflow action plan。
- publish 未审批时生成 approval record。
- publish 已审批但写开关未开启时返回 `RELEASE_PROVIDER_PUBLISH_PREVIEW_ONLY`。
- API publish 入口要求 `release:write`。
- `go test ./internal/release ./internal/cli ./internal/api` 通过。
- `go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

## 8. 已完成任务：`phase6-003 ci-cd-release-provider-adapter`

范围：

- `internal/release` 增加 `ProviderExecution`，记录 release provider preview/publish 执行结果。
- CLI 增加 `moyuan release provider preview|publish|execution`。
- API 增加 `/releases/:release_id/provider-preview`、`/provider-publish` 和 `/release-provider-executions/:execution_id`。
- Authz middleware 将 provider publish 归入 `release.provider.publish`，要求 `release:write`。
- 测试覆盖 provider preview、approval required 和 preview-only 降级。

非目标：

- 不打开真实 release provider write。
- 不把 token 或 secret value 写入 execution/log。

验证：

- `go test ./internal/release ./internal/cli ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 9. 验证要求

每完成一个 Phase 6 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
