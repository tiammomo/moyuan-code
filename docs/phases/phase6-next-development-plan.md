# Phase 6 实施记录

状态：planned
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
| P0 | `phase6-001` | `approval-consumption-replay-guard` | planned | approval record 消费和重放防护 | 已消费 approval 不能再次触发真实外部写入 |
| P1 | `phase6-002` | `deployment-ssh-preview-adapter` | planned | 部署 adapter preview/dry-run/execute 状态模型 | 生产真实执行继续默认关闭 |
| P1 | `phase6-003` | `ci-cd-release-provider-adapter` | planned | release/tag/workflow provider adapter | 远程发布动作可审计且可降级 |
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

## 4. 验证要求

每完成一个 Phase 6 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
