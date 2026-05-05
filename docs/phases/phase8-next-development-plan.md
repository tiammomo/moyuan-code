# Phase 8 实施记录

状态：planned
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
| P0 | `phase8-001` | `release-provider-real-adapter-beta` | planned | GitHub/Gitee release provider adapter 最小真实写入 | approval、secret resolver、write switch 和 replay guard 全部满足 |
| P0 | `phase8-002` | `ssh-runner-controlled-execution` | planned | SSH runner 真实受控执行 | allowlist 命令可执行，非 allowlist 阻断，输出脱敏 |
| P1 | `phase8-003` | `post-deploy-smoke-monitor-evidence` | planned | 部署后 smoke/monitor evidence | 失败能阻断发布完成并生成 evidence |
| P1 | `phase8-004` | `rollback-suggestion-and-runbook` | planned | 回滚建议和 runbook | 失败部署能生成可审查回滚建议 |
| P2 | `phase8-005` | `console-operation-drilldown` | planned | Console operation detail 独立 drill-down | 用户能刷新并查看单个 operation/evidence detail |
| P2 | `phase8-006` | `provider-real-quota-cost-feedback` | planned | Provider quota/cost/quality feedback 更真实 | route decision 读取可信 signals |

## 3. 执行规划：`phase8-001 release-provider-real-adapter-beta`

范围：

- 定义 release provider adapter interface，区分 GitHub、Gitee 和 unsupported provider。
- 真实写入仍必须同时满足 `MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE=1`、已批准且未消费 approval、secret resolver 成功、release plan ready。
- 第一版优先支持最小可验证动作：创建 tag/release 或生成 workflow dispatch request；无法安全执行的 action 必须显式 skipped。
- 远程请求只记录 provider、endpoint category、status code category 和 artifact reference，不记录 token 或响应正文。
- execution、audit 和 evidence 必须能串联。

非目标：

- 不自动提升到 production release。
- 不绕过人工 approval。
- 不支持所有 Git provider API 差异，只先做 GitHub/Gitee 的最小公共路径。

验收：

- 缺少写开关、approval 或 secret 时仍阻断。
- 写开关开启且 approval 通过时，adapter 到达真实请求边界或受控 skipped。
- 真实/模拟远程响应都写入脱敏 execution 和 evidence。
- `go test ./internal/release ./internal/approvals ./internal/secrets ./internal/api ./internal/cli` 通过。
- `go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

## 4. 验证要求

每完成一个 Phase 8 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
