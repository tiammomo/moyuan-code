# Phase 4 实施记录

状态：in_progress
责任角色：orchestrator_owner + api_owner + frontend_owner + security_owner + devops_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 4 的实际执行顺序。Phase 4 已进入实现期，第一批入口以 [Phase 4 实现 Issue Graph](./phase4-issue-graph.md) 为准。

## 1. 当前基线

Phase 3 已完成并通过 release readiness：

- Workspace YAML validator 覆盖核心配置域。
- Provider refresh 支持可选轻量 probe。
- Visual script mode 支持 auth ref、脱敏审计、质量检查和预览索引。
- Console 支持 Visual dry-run、Runtime artifacts、Release suggest、Deploy dry-run 和 Health scan。
- Release/deploy 当前以受控计划、dry-run 和状态记录为主。

## 2. Phase 4 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase4-001` | `audit-log-query-api-console` | completed | 统一核心日志查询 API 和 Console Audit 面板 | 脱敏后的 run/audit/error 日志可按 channel、issue、run、limit 查询 |
| P0 | `phase4-002` | `approval-record-store-api` | completed | 高风险操作审批记录落盘、查询和审计 | release/deploy/visual/provider 高风险动作有完整 approval lifecycle |
| P1 | `phase4-003` | `team-auth-session-token-baseline` | completed | 本地团队模式的 session、API token 和 service account 基线 | session/token/service account 可创建、查询、撤销并写入 audit log |
| P1 | `phase4-004` | `git-pr-mr-plan-sync` | completed | GitHub/Gitee PR/MR 计划、远程链接和状态同步 | PR/MR 状态可记录，不绕过 review 与质量门禁 |
| P2 | `phase4-005` | `server-resource-maintenance` | planned | 服务器到期、续费、巡检、退役和环境引用维护 | 测试开发机和生产机生命周期可查询、可提醒、可审计 |

## 3. 已完成任务：`phase4-001 audit-log-query-api-console`

范围：

- 新增 `logging.List`，从 `.moyuan/logs/*.jsonl` 聚合核心日志。
- 支持按 `channel`/`stream`、`issue_id`、`run_id`、`event` 和 `limit` 查询。
- 查询结果会按时间倒序返回，并输出统一的 `audit_events` 视图。
- 日志查询会脱敏 token、API key、password、secret、credential 和 private key。
- API 新增 `GET /v1/projects/:project_id/audit-events`。
- Console 新增 `Audit Trail` 面板，展示核心审计事件、channel、状态/决策、issue/run/subagent/trace 关联。

非目标：

- 不在本任务中实现 approval record。
- 不引入团队登录、RBAC session 或 API token。
- 不把 JSONL 日志迁移为集中式日志系统。

验证：

- `go test ./internal/logging ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 4. 已完成任务：`phase4-002 approval-record-store-api`

范围：

- 新增 `internal/approvals`，审批记录写入 `.moyuan/lifecycle/approvals/` 和 `approvals.jsonl`。
- 新增 API：`GET /approvals`、`POST /approvals`、`GET /approvals/:id`、`POST /approvals/:id/decide`。
- 审批记录包含 target、action、risk、status、decision、requester、decider、reason 和 metadata。
- 高风险动作已接入 approval record：production deploy plan、非 dry-run deployment execute、Visual script render、Provider probe。
- Provider probe 未批准时不外呼上游，返回 `provider_probe_approval_required` 和 `approval_id`。
- Console 新增 `Approval Queue` 面板，展示审批 action、target、risk、decision 和 reason。
- 审批 reason/metadata 禁止携带 token、API key、password、secret、credential 和 private key。

非目标：

- 不在本任务中实现团队登录和 approver role 校验。
- 不自动用已批准 record 继续执行原动作；后续由 Phase 4 team auth/session 接入有效审批校验。
- 不替代 GitHub/Gitee PR/MR review。

验证：

- `go test ./internal/approvals ./internal/api ./internal/providers ./internal/visuals ./internal/cli` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 5. 已完成任务：`phase4-003 team-auth-session-token-baseline`

范围：

- 新增 `internal/auth` 团队身份基线，状态写入 `.moyuan/auth/team.json`。
- 支持 local team session 创建、查询和撤销。
- 支持 API token 创建、查询和撤销；Token 明文只在创建时返回一次，落盘只保存 hash 与 prefix。
- 支持 service account 创建/覆盖和查询，用于后续 CI、release bot、deploy bot。
- 新增 API：`GET/POST /auth/sessions`、`POST /auth/sessions/:id/revoke`、`GET/POST /auth/api-tokens`、`POST /auth/api-tokens/:id/revoke`、`GET/POST /auth/service-accounts`。
- Console 新增 `Access Baseline` 面板，展示 active session、API token 和 service account 摘要。
- session/token/service account 的创建与撤销写入 `auth.*` audit event。

非目标：

- 不实现真实登录页、密码认证、SSO/OIDC 或 refresh token。
- 不在本任务中启用完整 RBAC middleware；当前是可审计的身份对象基线。
- 不把 API token 明文写入日志、Memory、配置或 Console 列表。

验证：

- `go test ./internal/auth ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 6. 已完成任务：`phase4-004 git-pr-mr-plan-sync`

范围：

- 扩展 `internal/gitprovider`，支持 PR/MR plan 列表和远程状态同步记录。
- PR/MR plan 增加 `remote_link`、`remote_status`、`sync_decision`、`sync_reason`、`synced_at` 字段。
- GitHub/Gitee/GitLab plan 会生成远程 compare/new PR/MR 链接；缺少 API auth 时降级为 manual status。
- 新增 API：`GET /git-provider-plans`、`POST /git-provider-plans/:id/sync`。
- 新增 CLI：`moyuan git provider list`、`moyuan git provider sync <plan-id>`。
- Console 的 Release Pipeline 展示 PR/MR plan、branch、provider、remote status 和 sync decision。
- 状态同步写入 `git_provider.status.synced` 日志。

非目标：

- 不真实 push、创建 PR/MR、合并 PR/MR 或读取远程保护分支。
- 不绕过 review merge decision 和质量门禁。
- 不读取 GitHub/Gitee token 明文；API auth adapter 进入后续任务。

验证：

- `go test ./internal/gitprovider ./internal/api ./internal/cli` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 7. 验证要求

每完成一个 Phase 4 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```

涉及权限、审批、Git remote 或服务器资源的 issue 必须补充对应单元测试或 API 测试。
