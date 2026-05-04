# 身份会话契约

本文定义用户身份、会话、API Token、服务账号和鉴权决策在实现层的接口契约。具体权限规则由 [鉴权与访问控制策略](../policies/auth-access-policy.md) 和 [权限模型](../foundations/permission-model.md) 维护。

## 1. 目标

- 为 CLI、API Server、Web Console、Orchestrator 和 Adapter 提供统一身份上下文。
- 保证所有操作能关联到 actor、resource、action、decision 和 trace。
- 禁止 Token、密码、Secret 明文进入日志、Memory、配置或图像 prompt。

## 2. 核心接口

```ts
export type AuthMethod =
  | "local_identity"
  | "session"
  | "api_token"
  | "service_account";

export type AuthDecision = "ALLOW" | "DENY" | "REQUIRE_APPROVAL";

export interface UserIdentity {
  id: string;
  username?: string;
  email?: string;
  displayName?: string;
  status: "invited" | "active" | "suspended" | "disabled" | "archived";
  createdAt: string;
  updatedAt: string;
}

export interface ServiceAccountIdentity {
  id: string;
  name: string;
  status: "active" | "disabled" | "revoked";
  ownerUserId?: string;
  organizationId?: string;
  projectId?: string;
  roles: string[];
  createdAt: string;
  updatedAt: string;
}

export interface AuthSession {
  id: string;
  userId: string;
  status: "active" | "expired" | "revoked";
  createdAt: string;
  expiresAt: string;
  lastSeenAt?: string;
  client?: {
    type: "cli" | "api" | "web";
    deviceId?: string;
    ipHash?: string;
  };
}

export interface ApiTokenRef {
  id: string;
  ownerType: "user" | "service_account";
  ownerId: string;
  status: "active" | "expired" | "revoked" | "rotated";
  scopes: string[];
  tokenHashRef: string;
  tokenPrefix?: string;
  tokenSuffix?: string;
  createdAt: string;
  expiresAt?: string;
  lastUsedAt?: string;
}

export interface Membership {
  id: string;
  subjectType: "user" | "service_account";
  subjectId: string;
  organizationId?: string;
  projectId?: string;
  roles: string[];
  status: "invited" | "active" | "suspended" | "removed";
  createdAt: string;
  updatedAt: string;
}

export interface AuthContext {
  actorType: "user" | "service_account" | "system";
  actorId: string;
  authMethod: AuthMethod;
  organizationId?: string;
  projectId?: string;
  roles: string[];
  sessionId?: string;
  apiTokenId?: string;
  traceId: string;
  requestId?: string;
}

export interface AuthzRequest {
  context: AuthContext;
  resourceType: string;
  resourceId?: string;
  action: string;
  riskLevel: "low" | "medium" | "high" | "critical";
  environment?: "local" | "test_dev" | "staging" | "production";
  reason?: string;
}

export interface AuthzResult {
  decision: AuthDecision;
  reason: string;
  requiredApprovalRole?: string;
  approvalId?: string;
  blockedReason?: string;
  auditEventId: string;
}
```

## 3. Provider 接口

```ts
export interface AuthProvider {
  resolveContext(input: {
    authMethod: AuthMethod;
    credentialRef?: string;
    projectId?: string;
    organizationId?: string;
    traceId: string;
  }): Promise<AuthContext>;

  authorize(request: AuthzRequest): Promise<AuthzResult>;

  revokeSession(sessionId: string, reason: string): Promise<void>;

  revokeApiToken(tokenId: string, reason: string): Promise<void>;
}
```

实现要求：

- `resolveContext` 不能返回密码、Token 明文或 Secret 明文。
- `authorize` 必须写入审计事件，即使结果是 `DENY`。
- `revokeSession` 和 `revokeApiToken` 必须使后续请求立即失效。
- API Server、CLI 和 Orchestrator 不能各自实现分叉鉴权逻辑，必须调用同一契约。

## 3.1 当前 Approval Record 接口

Phase 4 第一批已提供 approval record store/API，作为后续团队鉴权和 approver role 校验的落点：

```http
GET /v1/projects/:project_id/approvals?status=pending&limit=20
POST /v1/projects/:project_id/approvals
GET /v1/projects/:project_id/approvals/:approval_id
POST /v1/projects/:project_id/approvals/:approval_id/decide
```

审批记录结构：

```ts
interface ApprovalRecord {
  id: string;
  target_type: string;
  target_id: string;
  action: string;
  risk_level: "low" | "medium" | "high" | "critical";
  status: "pending" | "approved" | "rejected";
  decision: "APPROVAL_PENDING" | "APPROVAL_APPROVED" | "APPROVAL_REJECTED";
  requested_by: string;
  request_reason?: string;
  decided_by?: string;
  decision_reason?: string;
  metadata?: Record<string, unknown>;
  requested_at: string;
  decided_at?: string;
}
```

当前接入的高风险动作：

- production deployment plan。
- 非 dry-run deployment execution。
- Visual script render。
- Provider probe。

当前边界：

- 已能创建、查询、通过和拒绝 approval record，并写入 audit log。
- 未实现 approver role、禁止自审批和审批过期校验；这些由后续 RBAC middleware 任务接入。
- 审批 reason/metadata 禁止携带 token、API key、password、secret、credential 和 private key。

## 3.2 当前 Team Auth Baseline 接口

Phase 4 已提供本地团队身份对象基线，供 Console、CLI、CI 和后续 RBAC middleware 复用：

```http
GET /v1/projects/:project_id/auth/sessions
POST /v1/projects/:project_id/auth/sessions
POST /v1/projects/:project_id/auth/sessions/:session_id/revoke

GET /v1/projects/:project_id/auth/api-tokens
POST /v1/projects/:project_id/auth/api-tokens
POST /v1/projects/:project_id/auth/api-tokens/:token_id/revoke

GET /v1/projects/:project_id/auth/service-accounts
POST /v1/projects/:project_id/auth/service-accounts
```

当前存储：

- 状态文件：`.moyuan/auth/team.json`。
- Session：保存 user、display name、roles、status、created/revoked 信息。
- API Token：创建时返回一次 `token_value`；落盘只保存 `token_hash` 与 `token_prefix`，列表接口不返回 hash。
- Service Account：保存 id、name、roles、status，用于 release bot、deploy bot、CI bot。

当前审计事件：

- `auth.session.created`
- `auth.session.revoked`
- `auth.token.created`
- `auth.token.revoked`
- `auth.service_account.upserted`

当前边界：

- 这是 local team baseline，不是完整登录系统。
- API 还未强制解析 `Authorization` header，也未按 role/scope 拦截请求。
- 后续 RBAC middleware 必须基于这些对象输出统一 `AuthContext` 和 `AuthzResult`。

## 4. 错误类型

| 错误 | 含义 |
| --- | --- |
| `AUTH_MISSING_CREDENTIAL` | 缺少身份凭证 |
| `AUTH_INVALID_CREDENTIAL` | 凭证无效 |
| `AUTH_SESSION_EXPIRED` | 会话过期 |
| `AUTH_SESSION_REVOKED` | 会话撤销 |
| `AUTH_TOKEN_EXPIRED` | API Token 过期 |
| `AUTH_TOKEN_REVOKED` | API Token 撤销 |
| `AUTH_TOKEN_SCOPE_MISMATCH` | Token scope 不匹配 |
| `AUTH_USER_DISABLED` | 用户禁用 |
| `AUTH_MEMBERSHIP_MISSING` | 缺少成员关系 |
| `AUTH_PERMISSION_DENIED` | 权限拒绝 |
| `AUTH_APPROVAL_REQUIRED` | 需要审批 |
| `AUTH_APPROVAL_REJECTED` | 审批被拒绝 |

## 5. 日志事件

必须产生的审计事件：

- `auth.login`
- `auth.logout`
- `auth.session.revoked`
- `auth.token.created`
- `auth.token.revoked`
- `auth.token.rotated`
- `auth.membership.changed`
- `auth.role.changed`
- `auth.decision.allow`
- `auth.decision.deny`
- `auth.decision.require_approval`
- `auth.approval.created`
- `auth.approval.approved`
- `auth.approval.rejected`
- `auth.approval.expired`

事件字段至少包含：

- `event_id`
- `trace_id`
- `actor_id`
- `actor_type`
- `auth_method`
- `resource_type`
- `resource_id`
- `action`
- `decision`
- `reason`
- `timestamp`

## 6. 存储规则

- 用户密码如后续支持，只能保存强哈希，不进入 `.moyuan/`。
- API Token 只能保存哈希或密钥管理系统引用。
- local_single_user 可以保存本地 owner identity，但不能保存云凭证明文。
- team_server 模式的 User、Membership、Session、Token 和 Approval 应进入控制面数据库。
- 项目 `.moyuan/` 只保存项目访问策略、角色映射和审计产物，不保存身份凭证明文。

## 7. 验收标准

- 任意 Orchestrator 操作都能通过 `AuthContext` 找到 actor。
- 任意 `DENY` 和 `REQUIRE_APPROVAL` 都有审计事件。
- 撤销会话或 Token 后，新请求立即失败。
- API Token 明文只在创建时返回一次，之后不可查询。
- 身份契约能被 CLI、API Server、Web Console 和自动化服务账号复用。
