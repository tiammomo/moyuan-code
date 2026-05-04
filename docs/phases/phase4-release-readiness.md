# Phase 4 Release Readiness

状态：ready_for_next_phase
最后更新：2026-05-05

本文记录 Phase 4 团队协作、审计查询、审批记录、Git PR/MR 计划同步和服务器维护能力的收口验证。稳定结论已回写到身份、日志、代码管理、Git Provider 和服务器资源主线文档。

## 1. 验证范围

已完成能力：

- 核心日志查询 API 和 Console Audit Trail，支持按 channel、issue、run、event 和 limit 查询脱敏日志。
- Approval record store/API/Console，覆盖 production deploy plan、非 dry-run deployment execute、Visual script render 和 Provider probe。
- Local team auth baseline，支持 session、API token、service account 的创建、查询和撤销；API token 明文只在创建时返回一次。
- Git Provider PR/MR plan list/sync，支持 GitHub/Gitee/GitLab remote link、manual/API auth missing 降级状态和 Console 可见。
- Server resource maintenance，支持维护扫描、续费记录、退役记录、维护队列和 Console 可见。

不在本次收口内：

- 完整登录页、SSO/OIDC、refresh token 和组织成员 UI。
- 全局 RBAC middleware 和按 role/scope 的请求拦截。
- `secret:` 引用的 secret manager 解析。
- 真实 GitHub/Gitee PR/MR 创建、评论、review 状态读取和 merge。
- 真实 SSH/云厂商资源续费、生产远程命令、线上烟测和监控告警联动。

## 2. 验证命令

后端：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
```

前端：

```bash
cd apps/console
npm run typecheck
npm run build
```

Git：

```bash
git diff --check
git status --short
```

## 3. 验证结论

- Phase 4 issue graph 中 `phase4-001` 到 `phase4-005` 均为 `completed`。
- 高风险操作已有 approval record 和 audit trail，不再只依赖临时返回值。
- Team auth 具备最小可审计身份对象，但尚未强制保护所有 API。
- GitHub/Gitee 当前停在 plan、remote link、status sync record，不会真实创建 PR/MR。
- Server resource 当前停在 inventory、maintenance record、renew/retire record，不会真实修改云资源。

## 4. 新增运行入口

Audit：

```bash
GET /v1/projects/:project_id/audit-events?channel=all&limit=20
```

Approval：

```bash
GET /v1/projects/:project_id/approvals
POST /v1/projects/:project_id/approvals/:approval_id/decide
```

Team Auth：

```bash
POST /v1/projects/:project_id/auth/sessions
POST /v1/projects/:project_id/auth/api-tokens
POST /v1/projects/:project_id/auth/service-accounts
```

Git Provider：

```bash
moyuan git provider list
moyuan git provider sync <plan-id>
```

Server Resources：

```bash
moyuan resources maintenance scan
moyuan resources maintenance list
moyuan resources renew <resource-id> --expires-at YYYY-MM-DD
moyuan resources retire <resource-id>
```

## 5. 产物位置

- `.moyuan/logs/*.jsonl`
- `.moyuan/lifecycle/approvals/`
- `.moyuan/auth/team.json`
- `.moyuan/lifecycle/pull-requests/`
- `.moyuan/resources/maintenance/`
- `.moyuan/resources/maintenance.jsonl`

## 6. 剩余风险

- API token/session 已可记录，但还没有统一 middleware 强制鉴权。
- Approval record 已可记录，但 approver role、自审批禁止、过期策略还未接入。
- Git Provider PR/MR 仍是本地 plan/sync record，真实平台 API adapter 未启用。
- Server resource 维护记录不等于真实云厂商续费或 SSH 巡检。
- Console 能展示状态，但还没有用户管理和审批操作表单的完整交互。

## 7. 下一阶段入口

Phase 5 进入“真实外部执行前的强制门禁与 adapter 接入”：

1. 全局 auth context + RBAC middleware。
2. Secret resolver，支持 `secret:` 引用并统一脱敏。
3. GitHub/Gitee PR/MR API adapter，默认 dry-run/preview。
4. 部署烟测和监控 adapter，生产执行仍需 approval。
5. Console 增加用户/审批/PR/MR/维护操作表单。

执行入口见 [Phase 5 实现 Issue Graph](./phase5-issue-graph.md) 和 [Phase 5 实施记录](./phase5-next-development-plan.md)。
