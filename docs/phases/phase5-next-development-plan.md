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
| P0 | `phase5-001` | `auth-context-rbac-middleware` | planned | API middleware 解析身份并执行最小 RBAC | 受保护 API 有 allow/deny/require approval 决策和 audit event |
| P0 | `phase5-002` | `secret-ref-resolver` | planned | 安全解析 `secret:`/`env:` 引用并注入 adapter | Secret 明文不出现在响应、日志、Memory、prompt 或测试 fixture |
| P1 | `phase5-003` | `github-gitee-pr-mr-adapter` | planned | PR/MR preview/create/status adapter | 默认 preview，真实 create 需 authz + approval |
| P1 | `phase5-004` | `deployment-smoke-monitor-adapters` | planned | smoke/monitor 结果记录和 rollback 建议 | production 必须 approval，结果可审计 |
| P1 | `phase5-005` | `console-controlled-forms` | planned | Console 增加受控操作表单 | 表单只调用后端受控 API，状态以后端为准 |

## 3. 即将执行：`phase5-001 auth-context-rbac-middleware`

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

## 4. 验证要求

每完成一个 Phase 5 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```

涉及 Secret、GitHub/Gitee、部署或生产资源的 issue 必须有单元测试或 API 测试覆盖拒绝路径和脱敏路径。
