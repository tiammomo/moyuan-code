# 权限模型

本文定义 Moyuan Code 的权限主体、资源、动作、决策和审批规则。用户身份、会话和 API Token 的接口契约由 [身份会话契约](../contracts/auth-session-contract.md) 维护，具体鉴权决策树由 [鉴权与访问控制策略](../policies/auth-access-policy.md) 维护。

## 目标

- 防止 Agent 和 Runtime 越权修改项目。
- 防止密钥、生产服务器和敏感代码被误用。
- 防止第三方 API 接收超出策略允许的数据。
- 防止未授权用户、过期会话或失效 Token 执行项目操作。
- 让高风险操作可审批、可审计、可回滚。

## 身份与鉴权边界

权限判断分两步：

1. Authentication：确认当前 actor 是谁，身份是否有效。
2. Authorization：确认当前 actor 是否能对目标资源执行目标动作。

Moyuan 支持的 actor：

- User：人类用户。
- Service Account：CI、发布、部署或外部系统使用的非人类 actor。
- Orchestrator：系统编排层，只能代表已鉴权 actor 执行。
- Agent Role：任务角色，不能独立获得平台身份。

关键规则：

- Orchestrator 必须先建立 `auth_context`，再调度 Agent、Runtime、Git、Server 或 Provider。
- Agent Role 和 Native Runtime 的权限不能超过触发它们的 actor 和 issue write scope。
- API Token 只能代表其 owner 和 scope，不能继承用户全部权限。
- 会话过期、用户禁用、Token 撤销后，新操作必须阻断。

## 权限主体

| 主体 | 定义 | 典型能力 |
| --- | --- | --- |
| User | 使用 Moyuan 的人类用户 | 提需求、审批、取消、调整策略 |
| Service Account | 自动化调用 Moyuan 的非人类 actor | CI、发布、部署、报告同步 |
| API Token | User 或 Service Account 的受限凭证 | 调用 API、触发自动化任务 |
| Orchestrator | 核心编排层 | 调度、状态流转、权限判断、合入决策 |
| Subagent | Orchestrator 创建的具体执行实例 | 在父对象授权范围内执行规划、实现、验证或修复 |
| Agent Role | 具备职责和工具权限的角色 | 规划、实现、测试、review、发布 |
| Native Agent Runtime | Claude CLI、Codex CLI 等强执行后端 | 读写文件、运行命令、生成 diff |
| Adapter | Shell、Git、模型、MCP、Image 等外部能力封装 | 执行受控操作 |
| Release Manager | 发布和部署编排角色 | release branch、tag、PR/MR、部署 |
| Memory Curator | Memory 维护角色 | 候选审批、compact、过期、冲突处理 |

## 平台与项目角色

平台角色：

| 角色 | 典型能力 |
| --- | --- |
| `platform_owner` | 初始化系统、管理全局策略、组织和审计 |
| `org_admin` | 管理组织成员、项目访问和组织级策略 |
| `auditor` | 只读审计日志、审批记录和运行报告 |

项目角色：

| 角色 | 典型能力 |
| --- | --- |
| `project_owner` | 管理项目配置、成员、策略、发布审批 |
| `maintainer` | 管理需求、Issue、分支、质量门禁和普通发布 |
| `developer` | 提交需求、执行授权范围内的开发任务 |
| `reviewer` | 审核 diff、质量报告和合入建议 |
| `operator` | 操作测试开发机、预发、生产部署和回滚 |
| `viewer` | 只读项目画像、计划、报告和日志摘要 |

服务账号角色：

| 角色 | 典型能力 |
| --- | --- |
| `ci_runner` | 执行 CI、回归和只读报告上传 |
| `release_bot` | 按审批结果推送 release branch、tag 或 PR/MR |
| `deploy_bot` | 按 deployment plan 执行受控投产 |

## 权限资源

| 资源 | 示例 | 风险 |
| --- | --- | --- |
| 身份和成员 | User、Organization、Membership、Session、API Token | 越权、账号接管 |
| 文件系统 | `src/**`、`tests/**`、`.env` | 泄露、破坏用户改动 |
| Shell 命令 | `pnpm test`、`rm -rf`、`ssh` | 破坏性操作 |
| Git 操作 | branch、commit、push、merge、tag | 覆盖、发布错误 |
| 模型 API | GPT、Claude、GLM、MiniMax、第三方网关 | 数据外发 |
| Native Runtime | Claude CLI、Codex CLI | 越权写入、命令执行 |
| Skill | 内置、项目、组织或外部技能 | 不兼容能力、恶意提示、越权工具 |
| Memory | facts、decisions、lessons | 错误长期记忆 |
| Secret | API key、SSH key、registry token | 凭证泄露 |
| Server Resource | 测试开发机、生产机 | 线上事故 |
| Deployment | 部署、回滚、冒烟、监控 | 生产风险 |
| Visual Asset | 架构图 prompt、图片 | 敏感信息暴露 |

## 决策结果

权限判断只能输出三类结果：

| 决策 | 含义 |
| --- | --- |
| `ALLOW` | 可直接执行 |
| `DENY` | 禁止执行，不允许自动绕过 |
| `REQUIRE_APPROVAL` | 必须等待用户或指定审批者确认 |

## 默认策略

默认允许：

- 查看自己的身份、会话和权限摘要。
- 读取非敏感项目文件。
- 写入任务授权范围内的 `src/**`、`tests/**`、`docs/**`。
- 执行 allowlist 中的测试、lint、build、git status、git diff。
- 读取已脱敏的项目理解和 Memory 检索结果。

默认需要审批：

- 创建高权限 API Token。
- 修改成员、角色、鉴权策略或权限策略。
- 自动修复涉及生产、权限、安全、支付、数据库迁移、公共 API 或跨模块写入。
- `git push`、创建 PR/MR、创建 tag。
- 发布、部署、回滚。
- SSH 连接服务器。
- 修改 CI/CD、数据库迁移、权限、安全、支付相关代码。
- 访问 secret 引用。
- 向第三方模型发送内部代码上下文。
- 删除文件或大范围重构。

默认禁止：

- 保存明文 secret。
- 保存 API Token、会话密钥或密码明文。
- 自动修复时删除失败测试、降低质量门禁或扩大写入范围。
- 读取或发送 `.env` 明文内容。
- 将密钥、token、私网 IP、账号密码写入日志、Memory 或图像 prompt。
- 在生产机临时改代码。
- 生产机绕过 release/deploy pipeline 执行部署。
- 第三方模型 API 处理生产事故、完整项目 Memory dump、高敏私有算法或带密钥代码。
- Runtime 绕过 Moyuan 的 protected paths 和命令策略。

## 数据敏感等级

| 等级 | 定义 | 可发送对象 |
| --- | --- | --- |
| `public` | 公开信息 | 任意启用 provider |
| `internal_low` | 低敏内部信息，如普通摘要 | 官方 API、低风险第三方 API |
| `internal_high` | 项目代码、架构决策、内部接口 | 官方 API 或可信 Runtime |
| `confidential` | 私有算法、生产事故、关键业务逻辑 | 仅可信 Runtime 或明确授权 provider |
| `secret` | key、token、密码、`.env` 明文 | 禁止发送 |

## Agent 权限继承

Agent 的有效权限由以下来源合成：

```text
project workspace policy
  + auth context
  + platform role
  + project membership
  + role tools
  + subagent scope
  + skill required tools
  + team policy
  + issue write_scope
  + runtime capability
  + data sensitivity policy
  + current lifecycle phase
```

合成规则：

- 身份无效时最终为 `DENY`。
- 任意来源 `DENY`，最终为 `DENY`。
- 任意来源 `REQUIRE_APPROVAL`，最终为 `REQUIRE_APPROVAL`。
- 只有全部允许时才是 `ALLOW`。
- Issue 的 `write_scope` 不能扩大 workspace 的 `writable_paths`。
- Subagent 的 `write_scope` 不能扩大 Issue 的 `write_scope`。
- Skill 需要的工具权限不能扩大 Subagent 或 Agent Role 权限。
- Runtime 能力不能扩大 Agent Role 权限。
- API Token scope 不能扩大 User 或 Service Account 角色权限。

## API Token 权限边界

API Token 必须遵守：

- 只保存哈希或 secret 引用，不保存明文。
- scope 必须显式声明。
- expires_at 默认必填，长期 token 必须有更高审批。
- token owner 被禁用后，token 自动失效。
- token 不能创建权限更高的新 token。
- token 使用必须记录审计。

API Token 不允许：

- 出现在 `.moyuan/` 明文配置。
- 出现在日志、Memory、图像 prompt 或报告。
- 被模型 Provider 接收。
- 用于绕过用户会话、审批或项目角色限制。

## 会话权限边界

会话必须遵守：

- 有过期时间和撤销能力。
- 只代表一个 User。
- 不保存密码或 Token 明文。
- 高风险审批必须使用有效会话。
- 会话过期后，写入、Git、服务器、发布、部署和权限变更必须停止或等待重新鉴权。

## Runtime 权限边界

Claude CLI 和 Codex CLI 必须遵守：

- 只能在 issue worktree 或任务分支内运行。
- 运行前必须捕获 baseline diff。
- 运行后必须捕获 final diff。
- 写入范围受 `write_scope` 和 `protected_paths` 限制。
- 命令执行受 allowlist/denylist 限制。
- 输出必须脱敏后写入日志。
- 不能自行 push、tag、deploy，除非权限策略明确允许并完成审批。

## Subagent 与 Skill 权限边界

Subagent 必须遵守：

- 必须继承 Orchestrator 下发的 `auth_context`。
- 必须绑定父对象，不能脱离 Epic、Issue、Run、Repair Attempt、Release、Deployment 或 Memory Job 独立执行。
- 读写范围必须小于或等于父对象授权范围。
- 不允许自行创建无限层级子任务。
- 输出必须回到 Orchestrator 收敛，不允许直接合入、push 或 deploy。

Skill 必须遵守：

- 外部 skill 默认不可信。
- 高风险 skill 需要审批后才能启用。
- Skill 不能要求读取 secret 明文。
- Skill 不能覆盖权限策略、质量门禁或发布策略。
- Skill 效果只能影响推荐、降权或禁用，不能直接扩大权限。

## 模型 Provider 权限

官方 API：

- 可按 `data_policy` 接收项目代码、需求、架构摘要。
- 不允许接收 secret 明文。
- 请求和响应默认不写完整日志。

第三方 API：

- 必须声明 `upstream_vendor`。
- 必须声明 `allowed_use_cases`。
- 默认只能处理低风险文本、摘要、分类和轻量 Memory 抽取。
- 不允许处理 `internal_high` 以上数据，除非项目策略显式放行。

图像模型：

- `gpt-image-2` 只接收脱敏后的 `diagram_spec` 和视觉 prompt。
- 不允许接收 key、token、私网 IP、账号密码、`.env` 内容。
- 图像产物不能作为代码事实来源。

## 服务器权限

测试开发机：

- 可用于联调、部署演练、冒烟、日志查看。
- 不允许访问生产数据。
- 高风险命令仍需审批。

生产机：

- 只能通过 release/deploy pipeline 操作。
- 需要 release id 和审批记录。
- 必须具备回滚方案。
- 远程命令必须记录审计日志。
- 禁止临时改代码、临时装包、导出 secret。

## Git 权限

默认允许：

- `git status`
- `git diff`
- 创建 task branch
- 创建 issue worktree

默认需要审批：

- `git push`
- 创建 PR/MR
- 创建 tag
- 合并到默认分支
- 删除远程分支

默认禁止：

- `git reset --hard`
- 覆盖用户未提交改动
- 强推默认分支

## 审批对象

审批记录必须包含：

- `approval_id`
- `requester`
- `operation`
- `resource`
- `risk_level`
- `reason`
- `diff_or_command`
- `approver`
- `decision`
- `decided_at`

审批记录落盘：

- `.moyuan/logs/audit/`
- `.moyuan/lifecycle/deployments/`，如果与部署相关
- `.moyuan/lifecycle/releases/`，如果与发布相关

## 权限审计

必须审计：

- secret 访问。
- protected path 访问。
- 高风险命令。
- Runtime 写入文件。
- Git push、tag、PR/MR。
- 生产部署和回滚。
- 第三方 API 调用。
- 权限策略修改。

不应记录：

- 明文 secret。
- 完整 `.env`。
- 完整 prompt/response，除非策略明确允许且已脱敏。
