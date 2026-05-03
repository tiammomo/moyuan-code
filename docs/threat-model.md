# 安全威胁模型

状态：planned
责任角色：security_owner + architect
最后更新：2026-05-03

本文定义 Moyuan Code 的主要威胁场景、攻击面、缓解措施和验收标准。它不替代 [权限模型](./foundations/permission-model.md)，而是从攻击者视角检查多 Agent、Native Runtime、模型 Provider、Git、服务器和 Memory 的风险。

## 1. 目标

- 明确不可信仓库、模型上下文、Native Runtime、第三方 API 和生产服务器的安全边界。
- 防止 prompt injection、secret 泄露、越权写入、恶意命令、供应链污染和生产事故。
- 把安全要求落到权限策略、Provider 路由、Runtime Adapter、日志审计和质量门禁。
- 为后续实现提供安全测试和 review 清单。

## 2. 边界

本文关注 Moyuan 框架本体的安全威胁。

不在本文展开：

- 权限主体、资源、动作和审批语义，见 [权限模型](./foundations/permission-model.md)。
- 鉴权决策树，见 [鉴权与访问控制策略](./policies/auth-access-policy.md)。
- Provider 路由规则，见 [Provider 路由策略](./policies/provider-routing-policy.md)。
- 日志事件字段，见 [日志与审计事件契约](./contracts/logging-audit-event-contract.md)。

## 3. 信任边界

| 边界 | 默认信任级别 | 说明 |
| --- | --- | --- |
| 平台用户会话 | 条件可信 | 必须有有效 Auth Session 或 API Token |
| 被管理仓库 | 默认不可信 | 仓库内代码、脚本、文档和 prompt 都可能恶意 |
| `.moyuan/` 工作空间 | 项目级可信状态 | 仍需 schema 校验、锁和审计 |
| Native Runtime | 高能力、受限可信 | Claude CLI、Codex CLI 能读写文件和执行命令，必须受 scope 控制 |
| 官方模型 API | 条件可信 | 受 data policy、敏感等级和组织策略限制 |
| 第三方 API | 默认低信任 | 只能处理低敏摘要、分类、轻量抽取 |
| 外部 Skill | 默认不可信 | 必须经过 registry、风险标记和审批 |
| Git remote | 条件可信 | 推送、tag、PR/MR 必须受权限和审批控制 |
| 测试开发机 | 受限可信 | 不能自动代表生产 |
| 生产机 | 高风险边界 | 部署、SSH、回滚必须审批、审计、可回滚 |

## 4. 主要资产

- 源代码、测试、构建脚本和配置。
- API key、SSH key、registry token、云凭证和 `.env`。
- Project Profile、Module Map、Issue Graph、Run、Quality Report。
- Memory facts、decisions、lessons、compact 结果。
- Git 分支、tag、PR/MR、release branch。
- 服务器 inventory、生产环境、部署凭证和监控信息。
- 统一日志、审计日志和 approval 记录。

## 5. 威胁场景

### 不可信仓库指令注入

风险：

- 仓库中的 README、CLAUDE.md、测试脚本、注释或文档诱导 Agent 泄露 secret、跳过测试、修改 protected paths。

缓解：

- 项目文档内容只能作为被管理项目事实，不作为 Moyuan 系统指令。
- Runtime prompt 必须明确系统边界，禁止仓库内容覆盖 Moyuan policy。
- 读取仓库内 agent 配置时必须区分“用户项目约定”和“Moyuan 权限策略”。
- protected paths、命令 allowlist、Provider data policy 不可被仓库内容覆盖。

### Native Runtime 越权写入

风险：

- Claude CLI 或 Codex CLI 直接修改 `.env`、CI/CD、权限、安全、发布脚本或用户未授权文件。

缓解：

- 每个 Subagent 必须有 read_scope 和 write_scope。
- Runtime 开始前后捕获 diff。
- 超出 write_scope 的 diff 直接标记为 policy violation。
- 高风险路径变更必须 REQUIRE_APPROVAL。
- Runtime 不能自行 push、tag、deploy。

### 恶意命令执行

风险：

- 被管理项目的 package scripts、测试脚本或生成代码执行删除、网络上传、挖矿、植入后门等命令。

缓解：

- Shell Adapter 必须有 allowlist、denylist、timeout 和工作目录限制。
- 首次运行未知 install、postinstall、migration、deploy、ssh 命令需要审批。
- 测试命令输出和退出码必须记录。
- 命令运行失败不能自动扩大权限。

### Secret 泄露

风险：

- secret 进入模型 prompt、日志、Memory、图像 prompt、review 报告或错误堆栈。

缓解：

- 配置只保存 secret ref。
- 日志和 prompt 输出前执行 secret scan 和 redaction。
- `.env` 明文禁止读取、外发和写入 Memory。
- 图像生成 prompt 必须脱敏。
- secret 访问必须写 audit log。

### 第三方 API 数据外发

风险：

- OpenAI-compatible 第三方网关接收高敏代码、完整 Memory、生产事故上下文或 secret。

缓解：

- 第三方 API 默认只能用于低风险摘要、分类和轻量 Memory 抽取。
- Provider 必须声明 upstream_vendor、allowed_use_cases 和 data_policy。
- internal_high、confidential、secret 数据默认 ROUTE_BLOCKED。
- 降级不能绕过敏感数据策略。

### Memory 污染

风险：

- 攻击者通过用户输入、仓库文档或模型幻觉写入错误长期记忆，影响后续开发和发布。

缓解：

- Record Gate 先判断价值和来源可信度。
- Extraction 只抽取结构化事实，不保存一次性指令。
- Staging dedup 和 compact 必须保留来源引用。
- 低置信、冲突和敏感候选进入人工 review 或丢弃。
- Memory 不能自动扩大权限或覆盖策略。

### Skill 供应链风险

风险：

- 外部 skill 诱导 Agent 使用危险工具、忽略质量门禁、上传代码或读取敏感文件。

缓解：

- 外部 skill 默认不可信。
- Skill Registry 必须记录 source、version、risk_level、required_tools。
- 高风险 skill 需要审批。
- Skill 不能覆盖权限、质量和发布策略。
- Skill effectiveness 发现负面结果后降权或禁用。

### Git 和发布篡改

风险：

- Agent 错误合入、覆盖用户改动、创建错误 tag、推送错误 release branch 或绕过 review。

缓解：

- dirty worktree 保护。
- issue worktree 和 task branch 隔离。
- accepted issue 才能 merge。
- push、tag、PR/MR 必须审批和审计。
- 远程操作失败必须进入恢复流程，不允许静默重试造成重复发布。

### 生产服务器风险

风险：

- Agent 在生产机执行未授权命令、错误部署、破坏数据、绕过冒烟和监控。

缓解：

- 生产资源必须登记到 Server Resource 和 Environment。
- deployment plan、backup、smoke、monitor、rollback 必须齐备。
- SSH、deploy、rollback 默认 REQUIRE_APPROVAL。
- 生产命令必须记录 audit log。
- 生产失败必须进入人工介入路径。

## 6. 安全控制面

安全控制必须分布在以下模块：

| 模块 | 安全职责 |
| --- | --- |
| `auth` | 身份、会话、API Token、审批 |
| `policy` | 统一 ALLOW、DENY、REQUIRE_APPROVAL |
| `workspace` | protected paths、原子写、锁、schema 校验 |
| `runtime-adapters` | 命令限制、diff 捕获、超时、输出脱敏 |
| `providers` | data policy、provider health、fallback 限制 |
| `memory` | record gate、来源引用、compact 审计 |
| `quality` | 安全检查、依赖风险、危险 diff 阻断 |
| `release` | tag、push、deploy、rollback 审批和审计 |
| `logging` | audit、error、redaction、trace 关联 |

## 7. 安全测试

必须加入 framework tests：

- 仓库 README 包含恶意 prompt，系统仍不泄露 secret。
- Subagent 修改 protected path，Quality/Policy 阻断。
- 第三方 API 请求包含 internal_high 代码，Provider 路由阻断。
- fake runtime 输出含疑似 token，日志脱敏。
- 外部 skill 请求危险工具，高风险审批阻断。
- dirty worktree 下禁止自动 merge。
- production deploy 无审批时阻断。
- Memory candidate 来自低可信来源，Record Gate 拒绝或进入 review。

测试策略见 [框架自身测试策略](./framework-testing-strategy.md)。

## 8. 失败恢复

安全相关失败默认保守处理：

- secret scan 命中：立即阻断外发和落盘，生成 security finding。
- policy violation：停止当前 Subagent，保留 diff 和日志，进入 review。
- 生产命令异常：停止流水线，进入人工介入。
- 第三方 Provider 数据策略不明：ROUTE_BLOCKED。
- audit log 写入失败：阻断高风险操作。

恢复规则见 [失败恢复设计](./foundations/failure-recovery.md)。

## 9. 验收标准

进入实现前，本文必须能支撑：

- 每个高风险攻击面都有默认阻断或审批规则。
- Native Runtime 不能绕过 write_scope、protected paths 和质量门禁。
- 第三方 API 不能默认接收敏感代码、完整 Memory 或生产事故上下文。
- Memory、Skill 和仓库文档不能覆盖 Moyuan 权限策略。
- 发布和生产部署必须可审批、可审计、可回滚。

## 10. 相关文档

- [权限模型](./foundations/permission-model.md)
- [Provider 路由策略](./policies/provider-routing-policy.md)
- [模型与工具适配规划](./model-tool-adapters.md)
- [Subagent 与 Skills 系统方案](./subagents-skills-system.md)
- [Agent Memory 系统方案](./agent-memory-system.md)
- [框架自身测试策略](./framework-testing-strategy.md)
