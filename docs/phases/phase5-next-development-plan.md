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
| P1 | `phase5-004` | `deployment-smoke-monitor-adapters` | planned | smoke/monitor 结果记录和 rollback 建议 | production 必须 approval，结果可审计 |
| P1 | `phase5-005` | `console-controlled-forms` | planned | Console 增加受控操作表单 | 表单只调用后端受控 API，状态以后端为准 |

## 3. 执行规划：`phase5-001 auth-context-rbac-middleware`

范围：

- 增加 API auth middleware，可从 header 解析 API token 或使用 local owner fallback。
- 输出统一 AuthContext：actor、auth method、roles/scopes、project、trace。
- 对高风险 API 做最小 RBAC：provider probe、approval decision、deployment execute、visual script、resource renew/retire、git sync。
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
- API 增加 authz middleware，保护 provider refresh、approval decide、deployment execute、visual render、resource renew/retire 和 git provider sync。
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
- CLI 增加 `moyuan git provider preview <plan-id>` 和 `moyuan git provider create <plan-id> [--approved]`。
- API 增加 `/git-provider-plans/:plan_id/preview` 和 `/git-provider-plans/:plan_id/create`。
- `create` 真实远程写入必须同时满足 approval、authz `git:write`、Secret Resolver `pull_request.create` 用途、`MOYUAN_ALLOW_GIT_PROVIDER_WRITE=1`。
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

## 7. 验证要求

每完成一个 Phase 5 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```

涉及 Secret、GitHub/Gitee、部署或生产资源的 issue 必须有单元测试或 API 测试覆盖拒绝路径和脱敏路径。
