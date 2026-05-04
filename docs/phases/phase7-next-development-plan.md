# Phase 7 实施记录

状态：in_progress
责任角色：release_manager + devops_owner + security_owner + provider_owner + frontend_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 7 的实际执行顺序。Phase 7 的入口以 [Phase 7 实现 Issue Graph](./phase7-issue-graph.md) 为准。

## 1. 当前基线

Phase 6 已完成并通过 release readiness：

- Approval record 已支持消费和重放防护。
- Git Provider PR/MR create 已在真实写入路径前消费 approval。
- Deployment 已具备 SSH preview 状态模型，真实 SSH 仍默认阻断。
- Release provider 已具备 preview/publish execution，真实 release provider write 仍默认关闭。
- Provider telemetry 已进入 ops update、refresh 和 route decision。
- Console 已具备多视图、schema-aware 必填预检和 release/provider 操作入口。

## 2. Phase 7 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase7-001` | `release-provider-approval-consumption` | planned | release provider 真实 publish 的 approval consumption 和 replay guard | 真实 publish 路径不能重复使用 approval |
| P0 | `phase7-002` | `ssh-executor-guarded-runner` | planned | SSH executor 受控执行边界 | 默认阻断真实 SSH，启用后只执行白名单命令 |
| P1 | `phase7-003` | `post-action-evidence-model` | planned | 发布/部署/烟测/监控/回滚证据链 | 每次操作能查询统一 evidence |
| P1 | `phase7-004` | `runtime-telemetry-feedback-loop` | planned | runtime/quality 结果反哺 provider telemetry | route decision 可读取执行反馈 |
| P2 | `phase7-005` | `console-execution-detail-history` | planned | Console execution detail 和 operation history | 用户能追踪 preview、approval、publish、evidence |

## 3. 执行规划：`phase7-001 release-provider-approval-consumption`

范围：

- Release provider publish 增加真实写入开关语义，不因 `approved=true` 直接视为可远程写入。
- 当真实写入开关开启且 approval 已通过时，publish 必须消费 approval record。
- 已消费 approval 不能再次用于同一或其他 release provider publish。
- 写开关未开启时继续返回 preview-only，不消费 approval。
- execution 中明确记录 `approval_consumed`、`write_enabled` 和 replay guard reason。

非目标：

- 不在本任务中调用 GitHub/Gitee release、tag 或 workflow API。
- 不执行 `git push`、`git tag` 或 workflow dispatch。
- 不改变 API authz middleware 和 Secret Resolver 的既有规则。

验收：

- 缺少 approval 时 publish 仍生成 approval record。
- 写开关关闭时，即使 approval 已批准也返回 preview-only 且不消费 approval。
- 写开关开启时，approval 被消费，重复 publish 使用同一 approval 会被阻断。
- `go test ./internal/release ./internal/approvals ./internal/cli ./internal/api` 通过。
- `go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

## 4. 验证要求

每完成一个 Phase 7 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
