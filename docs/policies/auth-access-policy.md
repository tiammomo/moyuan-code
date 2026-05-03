# 鉴权与访问控制策略

本文定义 Moyuan Code 在用户身份、会话、API Token、组织成员关系、项目角色和高风险审批上的判断规则。

权限模型定义“哪些资源和动作有风险”，本文定义“当前 actor 是否能执行这次操作”。

## 1. 目标

- 所有操作先鉴权，再进入主线执行。
- 本地单用户模式和团队服务端模式使用同一套决策结果。
- 高风险操作必须能升级为审批，而不是静默放行。
- Token、会话、成员、角色和审批变更必须可审计。

## 2. 输入事实

| 输入 | 说明 |
| --- | --- |
| `actor` | User 或 Service Account |
| `auth_method` | local identity、session、api_token、service account |
| `session_status` | active、expired、revoked、missing |
| `token_status` | active、expired、revoked、rotated、scope_mismatch |
| `organization_id` | 当前组织 |
| `project_id` | 当前项目 |
| `membership` | actor 在组织和项目中的成员关系 |
| `roles` | 平台角色、组织角色、项目角色和服务账号角色 |
| `resource` | Project、Issue、Run、Git、Server、Release、Deployment、Provider、Memory 等 |
| `action` | read、write、execute、approve、publish、deploy、admin |
| `risk_level` | low、medium、high、critical |
| `environment` | local、test_dev、staging、production |
| `policy` | 项目策略、组织策略和全局策略 |
| `approval` | 已有审批记录，如适用 |

## 3. 决策结果

鉴权策略输出两层结果：

| 结果 | 含义 |
| --- | --- |
| `AUTHENTICATED` | 身份有效，可以继续做授权判断 |
| `UNAUTHENTICATED` | 身份无效，必须阻断 |
| `ALLOW` | 允许执行 |
| `DENY` | 禁止执行 |
| `REQUIRE_APPROVAL` | 需要指定审批者确认 |

最终执行层只消费 `ALLOW`、`DENY`、`REQUIRE_APPROVAL` 三种授权结果。

## 4. 身份认证树

```text
if mode == local_single_user:
  if local owner identity exists and not disabled:
    AUTHENTICATED
  else:
    UNAUTHENTICATED

if auth_method == session:
  if session missing or expired or revoked:
    UNAUTHENTICATED
  if user disabled:
    UNAUTHENTICATED
  else:
    AUTHENTICATED

if auth_method == api_token:
  if token missing or hash mismatch:
    UNAUTHENTICATED
  if token expired or revoked:
    UNAUTHENTICATED
  if token scope does not cover requested resource/action:
    UNAUTHENTICATED
  else:
    AUTHENTICATED

if auth_method == service_account:
  if service account disabled:
    UNAUTHENTICATED
  if token or secret reference invalid:
    UNAUTHENTICATED
  else:
    AUTHENTICATED
```

## 5. 授权决策树

```text
if authentication_result == UNAUTHENTICATED:
  DENY

if actor is suspended or disabled:
  DENY

if resource belongs to a project:
  require active project membership or service account binding

if policy has explicit deny:
  DENY

if action is read:
  allow viewer or higher unless resource is secret/protected

if action is write:
  require developer or higher and write scope match

if action is review:
  require reviewer or higher

if action is approve:
  require approver role and approval policy match

if action is publish:
  require maintainer or release_bot and release policy pass

if action is deploy:
  require operator or deploy_bot and environment policy pass

if action is admin:
  require project_owner, org_admin or platform_owner

if risk_level in high/critical:
  REQUIRE_APPROVAL unless policy explicitly pre-approved

otherwise:
  ALLOW
```

## 6. 高风险审批树

需要审批：

- `git push`、创建 PR/MR、创建 tag。
- 修改权限、Secret、CI/CD、部署脚本、数据库迁移和生产配置。
- 调用第三方模型处理内部高敏上下文。
- 连接生产服务器。
- 部署、回滚、线上冒烟和生产监控窗口变更。
- 创建高权限 API Token。
- 撤销用户、组织管理员或项目 owner。

审批者选择：

```text
if environment == production:
  require project_owner or operator approver

if action changes auth or permission policy:
  require project_owner or org_admin

if action creates platform-wide token:
  require platform_owner

if requester == approver and policy.disallow_self_approval:
  DENY or wait for another approver

if approval expired:
  REQUIRE_APPROVAL
```

## 7. API Token 策略

Token 必须：

- 只保存哈希，不保存明文。
- 有 owner、scope、expires_at、created_by 和 last_used_at。
- 能单独撤销。
- 能轮换并保留旧 token 的撤销记录。
- 默认最小权限，不继承用户全部权限。

Token 禁止：

- 写入 `.moyuan/` 明文。
- 写入日志、Memory、图像 prompt 或报告。
- 在过期后自动延长。
- 被 service account 用来创建更高权限 token。

## 8. 会话策略

会话必须：

- 有 `session_id`、`user_id`、`created_at`、`expires_at`、`last_seen_at`。
- 支持撤销。
- 支持空闲超时。
- 支持设备或客户端来源记录。

会话失效后：

- 已经运行中的低风险只读操作可以结束并记录警告。
- 写入、Git、Secret、服务器、发布和部署操作必须停止或转入等待重新鉴权。
- 不允许用过期会话继续审批。

## 9. 阻断条件

必须 `DENY`：

- 身份无效。
- 用户、服务账号或 Token 被禁用。
- Token scope 不匹配。
- 成员关系不存在或已过期。
- 目标资源不属于当前组织或项目上下文。
- 操作命中显式 deny。
- 审批者无审批权限。
- 需要审批但审批被拒绝。

必须 `REQUIRE_APPROVAL`：

- 高风险操作没有有效审批。
- 策略无法判断风险但资源属于生产、Secret、权限或发布域。
- 第三方模型外发上下文超过项目数据策略。
- 需要跨项目访问但没有预授权。

## 10. 产物和日志

每次鉴权必须产出：

- `auth_context`
- `auth_decision`
- `trace_id`

以下情况必须写审计：

- 登录、登出、会话撤销。
- Token 创建、撤销、轮换、过期使用。
- 成员和角色变化。
- `DENY`。
- `REQUIRE_APPROVAL`。
- 审批通过、拒绝、过期、取消。
- 生产、Secret、权限和发布相关 `ALLOW`。

## 11. 验收用例

- local_single_user 初始化后可以接入本地项目，但不能绕过 Git push 审批。
- 被禁用用户不能继续运行已排队的 issue。
- 过期 API Token 不能调用 release publish。
- developer 可以执行授权写入范围内的代码开发，但不能修改权限策略。
- reviewer 可以给出 review 结论，但不能发布生产部署。
- deploy_bot 没有审批记录时不能部署 production。
- org_admin 可以管理成员，但不能读取项目 Secret 明文。
- 审批拒绝后同一 trace 不能继续执行原操作。
