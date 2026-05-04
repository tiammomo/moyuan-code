# Phase 5 实施记录

状态：planned
责任角色：orchestrator_owner + security_owner + api_owner + adapter_owner + frontend_owner + devops_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 5 的实际执行顺序。Phase 5 的入口以 [Phase 5 实现 Issue Graph](./phase5-issue-graph.md) 为准。

## 1. 当前基线

Phase 4 已完成并通过 readiness：

- Audit Trail、Approval Record、Team Auth Baseline 已落地。
- Git Provider plan 支持 list/sync 和 remote link，但不真实创建 PR/MR。
- Server Resource 支持 maintenance record、renew 和 retire，但不修改真实云资源。
- Console 能展示审计、审批、身份、PR/MR plan 和维护队列。

## 2. Phase 5 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase5-001` | `auth-context-rbac-middleware` | completed | API middleware 解析身份并执行最小 RBAC | 受保护 API 有 allow/deny 决策和 audit event |
| P0 | `phase5-002` | `secret-ref-resolver` | completed | 安全解析 `secret:`/`env:` 引用并注入 adapter | Secret 明文不出现在响应、日志、Memory、prompt 或测试 fixture |
| P1 | `phase5-003` | `github-gitee-pr-mr-adapter` | completed | PR/MR preview/create/status adapter | 默认 preview，真实 create 需 authz + approval |
| P1 | `phase5-004` | `deployment-smoke-monitor-adapters` | completed | smoke/monitor 结果记录和 rollback 建议 | production 必须 approval，结果可审计 |
| P1 | `phase5-005` | `console-controlled-forms` | completed | Console 增加受控操作表单 | 表单只调用后端受控 API，状态以后端为准 |
| P0 | `phase5-006` | `approval-proof-enforcement` | completed | PR/MR 真实创建校验 approved approval record | 不能只靠请求体 `approved: true` 执行远程写入 |

## 3. 执行规划：`phase5-001 auth-context-rbac-middleware`

范围：

- 增加 API auth middleware，可从 header 解析 API token 或使用 local owner fallback。
- 输出统一 AuthContext：actor、auth method、roles/scopes、project、trace。
- 对高风险 API 做最小 RBAC：provider probe、approval decision、auth session/token/service account 写操作、deployment execute、visual script、resource renew/retire、git sync。
- allow/deny/require approval 决策必须写入 audit log。

非目标：

- 不实现登录 UI、密码认证、SSO/OIDC。
- 不实现完整组织成员管理。
- 不强制所有 read-only API 认证，先保护写操作和高风险操作。

验收：

- 缺少或无效 API token 访问受保护写操作时被拒绝或要求审批。
- service account scope 不足时返回 deny。
- local owner fallback 仅在 local_single_user 模式可用。
- `go test ./internal/auth ./internal/api ./...` 通过。

## 4. 已完成任务：`phase5-001 auth-context-rbac-middleware`

范围：

- `internal/auth` 增加 RequestContext、AuthzResult、Bearer token/session/local owner 解析和最小 scope/role 授权。
- API 增加 authz middleware，保护 provider refresh、approval decide、auth session/token/service account 写操作、deployment execute、visual render、resource renew/retire、git provider sync 和 PR/MR create。
- local_single_user 模式保留 owner fallback；team_server 模式必须提供有效 Bearer token 或 session。
- API token 按 scope 校验，scope 不足返回 `AUTH_TOKEN_SCOPE_MISMATCH`。
- allow/deny 写入 `auth.decision.allow` / `auth.decision.deny` audit event。

非目标：

- 不实现登录 UI、SSO/OIDC 或组织成员管理。
- 不拦截所有 read-only API。
- 不在 middleware 中自动创建 approval record；高风险业务动作仍由各模块创建 approval record。

验证：

- `go test ./internal/auth ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 5. 已完成任务：`phase5-002 secret-ref-resolver`

范围：

- 新增 `internal/secrets`，统一解析 `env:` 和 `secret:` 引用。
- `secret:` 通过 `.moyuan/policies/secrets.yaml` 间接指向 `env:KEY`，并按 `usage` 校验用途。
- 每次真实取值写入 `secret.access.granted` / `secret.access.denied` audit event。
- Provider ops probe、Native Runtime env profile 和 Visual script render 已改用 Secret Resolver。
- Runtime metadata、provider registry、render execution 和审计日志只保存 `auth_ref`、`env_keys`、secret id、用途和状态，不保存明文值。
- `.gitignore` 收窄为只忽略仓库根部 `/secrets/`，避免忽略 `internal/secrets` 代码包。

非目标：

- 不实现 Vault/KMS 真实 backend。
- 不支持 `secret:` 嵌套解析。
- 不在 Console 暴露 secret 明文查看或编辑能力。

验证：

- `go test ./internal/secrets ./internal/providers ./internal/runtime ./internal/visuals` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 6. 已完成任务：`phase5-003 github-gitee-pr-mr-adapter`

范围：

- Git Provider plan 增加 PR/MR preview 和 create 结果字段。
- CLI 增加 `moyuan git provider preview <plan-id>` 和 `moyuan git provider create <plan-id> [--approved] [--approval-id <approval-id>]`。
- API 增加 `/git-provider-plans/:plan_id/preview` 和 `/git-provider-plans/:plan_id/create`。
- `create` 真实远程写入必须同时满足已批准 approval record、authz `git:write`、Secret Resolver `pull_request.create` 用途、`MOYUAN_ALLOW_GIT_PROVIDER_WRITE=1`。
- GitHub/Gitee adapter 支持构造远程 PR 请求；默认保持 preview-only。
- 未审批、manual mode、缺少 token、关闭写开关和远程 API 失败都会写回本地 plan，不重复执行 push 或泄露 token。

非目标：

- 不自动 merge PR/MR。
- 不读取 GitHub/Gitee review 审批状态。
- 不绕过现有 review merge decision 和质量门禁。

验证：

- `go test ./internal/gitprovider ./internal/cli ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 7. 已完成任务：`phase5-004 deployment-smoke-monitor-adapters`

范围：

- Deployment execution 增加 `smoke_report`、`monitor_report` 和 `rollback_suggestion`。
- `local_shell` 部署命令成功后自动执行资源 healthcheck 作为 smoke，再执行 monitor 检查。
- HTTP/HTTPS healthcheck 仅允许 `127.0.0.1` 和 `localhost`，避免误扫外部或生产内网。
- smoke 或 monitor 失败时 execution 标记失败，并写入 rollback suggestion。
- 记录 `deployment.smoke.completed`、`deployment.monitor.completed`、`deployment.rollback.suggested` 日志。
- production 真实执行仍保持阻断；test_dev 可执行 dry-run 或受限 local_shell。

非目标：

- 不实现 SSH/云厂商真实部署。
- 不读取外部监控系统。
- 不自动执行 rollback command，只给出受控建议和审计记录。

验证：

- `go test ./internal/deployment ./internal/cli ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 8. 已完成任务：`phase5-005 console-controlled-forms`

范围：

- Console `Approval Queue` 增加审批决定输入和 `Approve` / `Reject` 操作，结果以后端 approval API 为准。
- Console `Access Baseline` 增加 session、API token 和 service account 受控表单；API token 列表支持 revoke。
- Console `Release Pipeline` 增加 Git Provider PR/MR `Preview`、`Sync`、`Create` 操作，真实 create 仍受后端 authz、approval、secret resolver 和写开关约束。
- Console `Server Resources` 增加资源续期与退役操作，维护记录以后端返回为权威状态。
- 所有 mutation 成功后刷新 server snapshot；前端不自行伪造权威状态。

非目标：

- 不在 Console 展示 secret 明文；API token 创建只显示短 preview。
- 不绕过 team_server 下的 API token/session 鉴权。
- 不实现生产 SSH/云厂商真实操作。

验证：

- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 9. 已完成任务：`phase5-006 approval-proof-enforcement`

范围：

- `approvals.VerifyApproved` 增加 approval record 校验，确认 approval 已批准且 target/action 匹配。
- Git Provider `create` 不再接受裸 `approved: true` 作为远程写入凭证；必须同时传入 `approval_id`。
- API `POST /git-provider-plans/:plan_id/create` 支持 `approval_id`。
- CLI `moyuan git provider create` 支持 `--approval-id <approval-id>`。
- Console PR/MR create 控制区增加 Approval ID 输入。

非目标：

- 不自动消费、归档或锁定 approval record。
- 不改变 production deployment 的真实执行阻断策略。
- 不把 approval 校验扩展到所有历史 `--approved` CLI，本任务先封住真实 PR/MR create。

验证：

- `go test ./internal/gitprovider ./internal/cli ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 10. 验证要求

每完成一个 Phase 5 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```

涉及 Secret、GitHub/Gitee、部署或生产资源的 issue 必须有单元测试或 API 测试覆盖拒绝路径和脱敏路径。
