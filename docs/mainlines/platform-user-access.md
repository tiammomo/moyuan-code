# 平台用户与访问控制主线

本文定义 Moyuan Code 框架自身的用户、身份认证、会话、API Token、组织/项目成员关系、角色、审批和审计流程。

本主线不设计被管理项目的业务用户系统。被管理项目自身的登录、会员、租户、权限代码仍属于被管理项目的业务开发范围，由需求规划和代码开发主线处理。

## 1. 目标

- 支持单用户本地 CLI 模式，也能平滑升级到多人团队协作模式。
- 所有高风险操作都能回答“谁触发、以什么身份、对什么资源、为什么允许、是否审批”。
- 支持组织、项目、成员、角色、服务账号和 API Token 的统一管理。
- 支持会话失效、Token 轮换、用户禁用、权限收敛和审计查询。
- 让 Agent、Runtime、Provider、Git、服务器和发布操作都继承同一套访问控制上下文。

## 2. 边界

本主线负责：

- Moyuan 平台用户和组织。
- 本地身份、服务端登录态和 API Token。
- 项目成员关系和角色。
- 高风险操作审批。
- 鉴权审计事件。

本主线不负责：

- 被管理项目的业务账号体系。
- 第三方 OAuth 的完整产品细节。
- 企业 SSO 的具体厂商适配。
- GitHub/Gitee token 的字段清单，字段规则由接入文档维护。
- Secret 明文保存，Secret 只能保存引用或哈希。

## 3. 运行模式

| 模式 | 适用阶段 | 身份来源 | 说明 |
| --- | --- | --- | --- |
| local_single_user | Phase 1 MVP | 本地初始化的 owner identity | 不要求登录服务，但所有操作仍带 `actor_id` 和审计记录 |
| team_server | Phase 4 | 用户登录会话、组织成员关系 | 支持多人共享项目、角色和审批 |
| service_account | Phase 4+ | API Token 或 CI 凭证引用 | 用于 CI、自动化发布和外部系统调用 |
| enterprise_sso | Phase 5 | 企业 SSO / OIDC / LDAP | 只作为未来扩展，不进入 MVP 必选范围 |

## 4. 输入和输出

输入：

- 用户命令、API 请求或 Web 操作。
- 当前身份凭证：local identity、session、API Token 或 service account。
- 目标资源：Project、Issue、Run、Git、Server Resource、Release、Deployment、Provider、Memory。
- 操作类型：read、write、execute、approve、publish、deploy、admin。
- 当前项目成员关系、角色、策略和风险等级。

输出：

- `auth_context`：身份、组织、项目、角色、来源和 trace 信息。
- `auth_decision`：`ALLOW`、`DENY` 或 `REQUIRE_APPROVAL`。
- `approval_request`：需要人工审批时的结构化请求。
- `audit_event`：不可丢失的审计事件。
- `blocked_reason`：拒绝或等待审批的用户可见原因。

## 5. 端到端流程

```text
用户/系统触发操作
  -> 解析身份凭证
  -> 建立 auth_context
  -> 校验用户、会话或 API Token 状态
  -> 解析组织、项目和成员关系
  -> 计算平台角色 + 项目角色 + 服务账号权限
  -> 调用鉴权策略
  -> ALLOW / DENY / REQUIRE_APPROVAL
  -> 写入审计事件
  -> 执行或阻断后续主线
```

关键规则：

- Orchestrator 在进入项目接入、Issue 编排、代码开发、Git 合入、服务器操作、发布投产前都必须先建立 `auth_context`。
- Native Agent Runtime 不能自行提升权限，只能继承 Orchestrator 下发的受限上下文。
- 高风险操作必须产生审批对象，审批通过后才能继续执行。
- 审批者和请求者是否允许同人，由 [鉴权与访问控制策略](../policies/auth-access-policy.md) 决定。

## 6. 用户功能

MVP 必须具备的用户功能：

- 初始化本地 owner。
- 查看当前身份。
- 切换当前项目上下文。
- 查看项目权限摘要。
- 对高风险操作进行确认或拒绝。
- 查询本地审计记录。

团队模式需要补充：

- 邀请用户。
- 禁用或恢复用户。
- 创建组织。
- 添加或移除项目成员。
- 分配项目角色。
- 创建、撤销和轮换 API Token。
- 创建 service account。
- 查看和撤销会话。
- 查询审批记录和审计日志。

## 7. 角色模型

平台角色：

| 角色 | 能力 |
| --- | --- |
| platform_owner | 初始化系统、管理组织、全局策略和审计 |
| org_admin | 管理组织成员、项目访问和组织级策略 |
| auditor | 只读审计日志、审批记录和运行报告 |

项目角色：

| 角色 | 能力 |
| --- | --- |
| project_owner | 管理项目配置、成员、策略、发布审批 |
| maintainer | 管理需求、Issue、分支、质量门禁和普通发布 |
| developer | 提交需求、执行授权范围内的开发任务 |
| reviewer | 审核 diff、质量报告和合入建议 |
| operator | 操作测试开发机、预发、生产部署和回滚 |
| viewer | 只读项目画像、计划、报告和日志摘要 |

服务账号角色：

- `ci_runner`：执行 CI、回归和只读报告上传。
- `release_bot`：按审批结果推送 release branch、tag 或 PR/MR。
- `deploy_bot`：按 deployment plan 执行受控投产。

## 8. 关键决策点

必须调用 [鉴权与访问控制策略](../policies/auth-access-policy.md) 的节点：

- 项目接入远程仓库。
- 读取或写入项目配置。
- 创建、更新或取消 Epic/Issue。
- 调用 Claude CLI、Codex CLI 等 Native Agent Runtime。
- 访问模型 Provider、Memory、Secret 引用或服务器资源。
- 创建任务分支、提交、push、tag、PR/MR。
- 执行部署、回滚、线上冒烟和生产监控查询。
- 创建、撤销、轮换 API Token。
- 修改权限策略、用户角色和组织成员。

## 9. 数据与配置

控制面数据：

- User、Organization、Membership、Service Account、API Token、Auth Session 和 Approval 是 Moyuan 控制面对象。
- team_server 模式下应进入数据库。
- local_single_user 模式下可使用本地身份文件和本地审计日志。

项目访问策略：

- 项目级策略只保存角色和访问边界，不保存用户密码、Token 明文或云凭证。
- Secret、Git token、SSH key、模型 API key、服务器登录凭证都只能保存引用。

权威定义：

- 核心对象见 [核心数据对象](../foundations/core-data-objects.md)。
- 权限边界见 [权限模型](../foundations/permission-model.md)。
- 会话接口见 [身份会话契约](../contracts/auth-session-contract.md)。

当前实现基线：

- Phase 4 已提供 local team session、API token 和 service account 的创建、查询、撤销 API。
- API token 明文只在创建时返回一次；后续列表只展示 token id、prefix、actor 和 scopes。
- Phase 5 已启用最小 authz middleware，先保护 provider refresh、approval decide、deployment execute、visual render、resource renew/retire、git provider sync 和 PR/MR create。
- 完整登录、组织成员 UI 和 read-only API 全量拦截仍在后续阶段推进。

## 10. 日志与审计

必须审计：

- 登录、登出、会话撤销。
- API Token 创建、使用、撤销、轮换和过期。
- 成员添加、移除、角色变更。
- 权限策略变更。
- 鉴权拒绝。
- 审批创建、通过、拒绝、过期和取消。
- 所有 `REQUIRE_APPROVAL` 操作的最终执行结果。

日志要求：

- 审计日志必须包含 `actor_id`、`auth_method`、`resource_type`、`resource_id`、`action`、`decision`、`trace_id`。
- 不记录密码、Token 明文、Secret 明文、完整 `.env` 或未脱敏 prompt。
- API Token 只能记录前后缀摘要、哈希引用和 token id。

## 11. 阻断和失败恢复

立即阻断：

- 身份无效。
- 用户被禁用。
- 会话过期或撤销。
- API Token 已过期、撤销或 scope 不匹配。
- 目标项目成员关系不存在。
- 操作命中显式 deny。
- 高风险操作缺少审批。

可恢复：

- 重新登录。
- 切换项目或组织上下文。
- 申请补充权限。
- 等待审批。
- 轮换 Token。
- 使用权限更小的替代操作。

不可自动恢复：

- 试图绕过审批。
- 试图扩大自身权限。
- 使用被撤销的 Token 重试。
- 在生产资源上降级为本地人工命令执行。

## 12. 验收标准

- 任意用户命令都能生成稳定 `auth_context`。
- 任意高风险操作都能得到 `ALLOW`、`DENY` 或 `REQUIRE_APPROVAL`。
- 用户禁用、Token 撤销、会话过期后不能继续执行任务。
- API Token 不以明文写入配置、日志、Memory 或图像 prompt。
- 项目角色能限制 Issue、Git、Server、Release、Deployment 的操作范围。
- 审批记录能关联到 run、issue、release 或 deployment。
- 本地主线可以单用户运行，团队模式可以平滑接入数据库和 API Server。

## 13. 相关文档

- [鉴权与访问控制策略](../policies/auth-access-policy.md)
- [身份会话契约](../contracts/auth-session-contract.md)
- [权限模型](../foundations/permission-model.md)
- [核心数据对象](../foundations/core-data-objects.md)
- [日志与审计事件契约](../contracts/logging-audit-event-contract.md)
