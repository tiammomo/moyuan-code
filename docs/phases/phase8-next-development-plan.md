# Phase 8 实施记录

状态：in_progress
责任角色：release_manager + devops_owner + security_owner + provider_owner + frontend_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 8 的实际执行顺序。Phase 8 的入口以 [Phase 8 实现 Issue Graph](./phase8-issue-graph.md) 为准。

## 1. 当前基线

Phase 7 已完成并通过 release readiness：

- Release provider publish 具备写开关、approval consumption 和 replay guard。
- Deployment `ssh_execute` 具备写开关、资源校验、命令 allowlist 和默认阻断。
- Release/deployment execution 已写入 evidence。
- Provider telemetry 已接入 runtime、quality 和 route feedback。
- Console 已具备 operation history 和 execution detail 汇总视图。

## 2. Phase 8 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase8-001` | `release-provider-real-adapter-beta` | completed | GitHub/Gitee release provider adapter 最小真实写入 | approval、secret resolver、write switch 和 replay guard 全部满足 |
| P0 | `phase8-002` | `ssh-runner-controlled-execution` | completed | SSH runner 真实受控执行 | allowlist 命令可执行，非 allowlist 阻断，输出脱敏 |
| P1 | `phase8-003` | `post-deploy-smoke-monitor-evidence` | planned | 部署后 smoke/monitor evidence | 失败能阻断发布完成并生成 evidence |
| P1 | `phase8-004` | `rollback-suggestion-and-runbook` | planned | 回滚建议和 runbook | 失败部署能生成可审查回滚建议 |
| P2 | `phase8-005` | `console-operation-drilldown` | planned | Console operation detail 独立 drill-down | 用户能刷新并查看单个 operation/evidence detail |
| P2 | `phase8-006` | `provider-real-quota-cost-feedback` | planned | Provider quota/cost/quality feedback 更真实 | route decision 读取可信 signals |

## 3. 执行规划：`phase8-001 release-provider-real-adapter-beta`

实现状态：completed。

范围：

- 定义 release provider adapter interface，区分 GitHub、Gitee 和 unsupported provider。
- 真实写入仍必须同时满足 `MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE=1`、已批准且未消费 approval、secret resolver 成功、release plan ready。
- 第一版支持最小可验证动作：GitHub/Gitee create release HTTP request；branch push、tag push 和 workflow dispatch 仍显式 `skipped`。
- 远程请求只记录 provider、endpoint category、status code category 和 artifact reference，不记录 token 或响应正文。
- execution、audit 和 evidence 必须能串联。

非目标：

- 不自动提升到 production release。
- 不绕过人工 approval。
- 不支持所有 Git provider API 差异，只先做 GitHub/Gitee 的最小公共路径。

验收：

- 缺少写开关、approval 或 secret 时仍阻断。
- 写开关开启且 approval 通过、secret resolver 通过时，adapter 可调用 create release；无法安全自动执行的 action 受控 skipped。
- 真实/模拟远程响应都写入脱敏 execution 和 evidence。
- `go test ./internal/release ./internal/approvals ./internal/secrets ./internal/api ./internal/cli` 通过。
- `go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

落地结果：

- `ProviderExecution` 记录 `remote_results` 和 `adapter_status`。
- unsupported provider 或缺少 create release endpoint 时返回 `RELEASE_PROVIDER_PUBLISH_UNSUPPORTED`，并保持 approval 未消费。
- 缺少 secret policy 或用途不匹配时返回 `RELEASE_PROVIDER_PUBLISH_AUTH_REQUIRED`，并保持 approval 未消费。
- secret 解析成功后才消费 approval；消费后执行 create release，approval replay 会被阻断。
- GitHub token 只进入 Authorization header；Gitee token 只进入请求体 `access_token`，但请求体不写入日志、execution 或 evidence。
- 测试使用 `httptest` 模拟 GitHub release API，断言 remote request、approval consumption、replay guard 和 secret 脱敏。

## 4. 执行规划：`phase8-002 ssh-runner-controlled-execution`

实现状态：completed。

范围：

- `ssh_execute` 默认仍由 `MOYUAN_ALLOW_SSH_EXECUTE=1` 显式开启，production real execution 继续阻断。
- 开启后先校验 deployment plan、server resource、`auth_ref`、命令 allowlist 和 command timeout。
- `auth_ref` 通过 Secret Resolver 以 `server.ssh.execute` purpose 临时解析；明文值只作为 `ssh -i` 参数使用。
- 真实执行通过本机 `ssh` 二进制完成，参数包含 `BatchMode=yes`、`StrictHostKeyChecking=no` 和 `ConnectTimeout=10`。
- stdout/stderr 在写入 `ExecutionStep` 前统一脱敏和截断。
- SSH 命令成功后自动串接 smoke、monitor 和 rollback suggestion；SSH 命令失败直接生成 rollback suggestion。

落地结果：

- `SSH_EXECUTION_READY` 取代 Phase 7 的 guarded ready 状态。
- 成功执行远程命令后记录 `deployment.ssh.commands.completed` 日志；如果后续 smoke/monitor 通过，则最终返回 `DEPLOY_EXECUTION_COMPLETED`。
- SSH 命令失败返回 `DEPLOY_SSH_EXECUTION_FAILED`，并生成 `ssh_command_failed` rollback suggestion。
- secret 缺失、用途不匹配、资源缺失或命令不在 allowlist 时保持 blocked/failed，不执行远程命令。
- 测试使用 fake `ssh` 二进制模拟成功和失败，断言 secret 不进入 execution、jsonl 或 audit log。

## 5. 验证要求

每完成一个 Phase 8 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
