# Moyuan Code 文档

当前阶段：规划设计。

`moyuan-code` 是一个面向代码开发全生命周期的多 Agent 开发框架。它的核心目标是：在理解项目代码的基础上，把用户开发需求自动完善、拆分为 Issue Graph，调度 Claude CLI、Codex CLI 和多种模型 API 分工开发，并通过用户鉴权、质量门禁、自我修复、Git 分支、发布流水线、Memory 和日志持续管理项目迭代。

本文是 docs 入口，只负责导航和边界说明，不承载完整方案。

## 推荐阅读顺序

1. [总体规划与生命周期路线图](./lifecycle-roadmap.md)：先看产品定位、端到端流程、CLI 路线、Phase 和近期落地范围。

2. [参考架构](./reference-architecture.md)：理解系统分层、核心模块、运行链路、上下文装配和安全边界。

3. [主线文档](./mainlines/README.md)：按平台用户与访问控制、项目接入、需求规划、代码开发、运行反馈与自我修复、代码管理、服务器资源和 DevOps 发布投产 8 条主线理解真实执行流程。

4. [策略决策树](./policies/README.md)：理解鉴权、澄清、阅读理解、并发调度、质量合入、Bug 判断、自我修复、Git、服务器、发布、Provider 和 Memory 的判断规则。

5. [项目工作空间规范](./project-workspace-spec.md)：理解每个被管理项目独立 `.moyuan/` 工作空间、目录职责和 schema 索引。

6. [配置方案](./configuration-guide.md) 与 [配置 Schema 规则](./configuration-schema-spec.md)：前者说明配置组合方式和关键样例，后者维护字段必填、可空、必须为空和条件必填规则。

7. [契约文档](./contracts/README.md)：理解身份会话、Subagent 与 Skill、自我修复、schema 校验、Runtime Adapter、日志审计事件和 Workspace 迁移的实现契约。

8. [Agent Memory 系统方案](./agent-memory-system.md)：理解记忆判断、抽取、暂存去重、自动 compact、分层存储、检索和长期维护。

## 核心设计文档

| 文档 | 作用 |
| --- | --- |
| [lifecycle-roadmap.md](./lifecycle-roadmap.md) | 产品定位、生命周期、CLI、Phase 和路线图 |
| [reference-architecture.md](./reference-architecture.md) | 系统架构、模块职责、状态机和上下文链路 |
| [mainlines/](./mainlines/README.md) | 按真实生命周期组织的 8 条主线流程 |
| [policies/](./policies/README.md) | 可实现为规则引擎或状态机的策略决策树 |
| [contracts/](./contracts/README.md) | 面向实现的 auth、subagent/skill、self-repair、schema、runtime、logging 和 migration 契约 |
| [project-workspace-spec.md](./project-workspace-spec.md) | `.moyuan/` 工作空间目录和 schema 索引 |
| [configuration-guide.md](./configuration-guide.md) | 配置总览、关键配置组合和最小/投产闭环 |
| [configuration-schema-spec.md](./configuration-schema-spec.md) | 配置字段规则、必填/可空/必须为空约束 |
| [agent-roles-overview.md](./agent-roles-overview.md) | Agent role、team、默认 Runtime 和 memory scope 概览 |
| [subagents-skills-system.md](./subagents-skills-system.md) | Subagent 生命周期、Skill Registry、推荐、绑定和效果反馈 |
| [agent-memory-system.md](./agent-memory-system.md) | Agent Memory 唯一详细方案 |

## 主线文档

| 主线 | 文档 | 作用 |
| --- | --- | --- |
| 平台用户与访问控制 | [mainlines/platform-user-access.md](./mainlines/platform-user-access.md) | 用户、组织、会话、API Token、角色、审批和审计 |
| 项目接入与阅读理解 | [mainlines/project-comprehension.md](./mainlines/project-comprehension.md) | 本地/远程仓库接入、full/incremental/diff comprehension、项目画像和 memory candidates |
| 需求规划与 Issue 编排 | [mainlines/requirement-planning.md](./mainlines/requirement-planning.md) | 需求完善、澄清判断、Issue Graph、依赖、schedule 和 ready/blocked 队列 |
| 代码开发 | [mainlines/code-development.md](./mainlines/code-development.md) | 消费 ready issue，完成多 Agent 开发、测试、质量复核和返工 |
| 运行反馈与自我修复 | [mainlines/runtime-feedback-self-repair.md](./mainlines/runtime-feedback-self-repair.md) | 运行信号、Bug 判断、自动修复、回归测试和能力增强 |
| 代码管理 | [mainlines/code-management.md](./mainlines/code-management.md) | branch、worktree、integration branch、PR/MR 和用户改动保护 |
| 服务器资源管理 | [mainlines/server-resource-management.md](./mainlines/server-resource-management.md) | 测试开发机、生产机、云资产、到期、巡检和资源组 |
| DevOps 发布投产 | [mainlines/devops-release-deployment.md](./mainlines/devops-release-deployment.md) | release branch、tag、部署、线上冒烟、监控、回滚和复盘 |

## 策略决策树

策略文档把流程中的判断节点整理成接近可实现的决策树。后续代码实现时，策略应优先转为规则引擎、状态机或 runtime validator。

| 策略 | 文档 |
| --- | --- |
| 鉴权与访问控制策略 | [policies/auth-access-policy.md](./policies/auth-access-policy.md) |
| 项目阅读理解策略 | [policies/project-comprehension-policy.md](./policies/project-comprehension-policy.md) |
| Issue 调度策略 | [policies/issue-scheduling-policy.md](./policies/issue-scheduling-policy.md) |
| 质量与合入策略 | [policies/quality-merge-policy.md](./policies/quality-merge-policy.md) |
| Bug 判断与自我修复策略 | [policies/bug-detection-self-repair-policy.md](./policies/bug-detection-self-repair-policy.md) |
| Git 分支策略 | [policies/git-branch-policy.md](./policies/git-branch-policy.md) |
| 服务器资源策略 | [policies/server-resource-policy.md](./policies/server-resource-policy.md) |
| 发布投产策略 | [policies/release-deployment-policy.md](./policies/release-deployment-policy.md) |
| Provider 路由策略 | [policies/provider-routing-policy.md](./policies/provider-routing-policy.md) |
| Memory 决策策略 | [policies/memory-decision-policy.md](./policies/memory-decision-policy.md) |

## 契约文档

| 契约 | 文档 | 作用 |
| --- | --- | --- |
| 身份会话契约 | [contracts/auth-session-contract.md](./contracts/auth-session-contract.md) | 统一用户身份、会话、API Token、服务账号和鉴权决策接口 |
| Subagent 与 Skill 契约 | [contracts/subagent-skill-contract.md](./contracts/subagent-skill-contract.md) | 定义 Subagent、Skill Registry、Skill 绑定和效果反馈接口 |
| 自我修复契约 | [contracts/self-repair-contract.md](./contracts/self-repair-contract.md) | 定义 Runtime Signal、Bug Candidate、Repair Attempt 和能力增强接口 |
| Schema 校验契约 | [contracts/schema-validation-contract.md](./contracts/schema-validation-contract.md) | 将配置规则转成机器可校验 schema 和 runtime validator |
| Runtime Adapter 契约 | [contracts/runtime-adapter-contract.md](./contracts/runtime-adapter-contract.md) | 统一 Claude CLI、Codex CLI 等 Runtime 的调用边界 |
| 日志与审计事件契约 | [contracts/logging-audit-event-contract.md](./contracts/logging-audit-event-contract.md) | 定义核心日志、审计事件、状态变化和 trace 关联 |
| Workspace 迁移契约 | [contracts/workspace-migration-contract.md](./contracts/workspace-migration-contract.md) | 管理 `.moyuan/` schema_version、迁移、回滚和兼容 |

## 专题设计文档

| 文档 | 作用 |
| --- | --- |
| [repository-onboarding-git-management.md](./repository-onboarding-git-management.md) | 本地/远程仓库接入、Git Provider Adapter 和远程同步触发 |
| [github-integration.md](./github-integration.md) | GitHub 连接、认证、token 权限、必填和可空字段 |
| [agent-roles-overview.md](./agent-roles-overview.md) | Agent role、team、memory scope 和输出契约概要 |
| [subagents-skills-system.md](./subagents-skills-system.md) | Subagent 和 skills 的唯一详细方案 |
| [engineering-process-standards.md](./engineering-process-standards.md) | commit、issue、回退后 fix、发版和测试覆盖率规范 |
| [model-tool-adapters.md](./model-tool-adapters.md) | Claude CLI、Codex CLI、GPT、Claude、GLM、MiniMax、第三方 API、gpt-image-2 和工具适配 |
| [issue-orchestration.md](./issue-orchestration.md) | Issue Graph、并发调度和等待模型的专题参考 |
| [code-lifecycle-quality-gates.md](./code-lifecycle-quality-gates.md) | 质量门禁、审核、返工和合入前检查的专题参考 |

## 基础规范

基础规范集中在 [foundations/](./foundations/README.md)，用于统一术语、对象、用户鉴权、权限、失败恢复和文档维护规则。

| 文档 | 作用 |
| --- | --- |
| [foundations/glossary.md](./foundations/glossary.md) | 核心术语 |
| [foundations/core-data-objects.md](./foundations/core-data-objects.md) | User、Organization、Project、Issue、Run、Subagent、Skill、Bug Candidate、Repair Attempt、Memory、Server、Release、Deployment 等核心对象 |
| [foundations/permission-model.md](./foundations/permission-model.md) | 用户身份、文件、命令、网络、密钥、Git、部署和模型权限边界 |
| [foundations/failure-recovery.md](./foundations/failure-recovery.md) | 失败分类、恢复策略、人工介入和审计要求 |
| [foundations/state-machine-catalog.md](./foundations/state-machine-catalog.md) | User、Project、Issue、Run、Subagent、Skill、Bug Candidate、Repair Attempt、Release、Deployment 等状态机总表 |
| [foundations/documentation-governance.md](./foundations/documentation-governance.md) | 文档权威来源、重复控制、资产管理和维护规则 |
| [design-readiness-checklist.md](./design-readiness-checklist.md) | 进入实现前的设计就绪门禁 |

## 辅助资产

[assets/](./assets/) 目录不是当前文档体系的核心设计来源。这里保存的是 `gpt-image-2` 生成的辅助可视化产物，例如架构图图片和讲解文档。

规则：

- 核心设计以 Markdown 文档为准，不以图片为准。
- 图片只用于辅助讲解和评审沟通。
- 生成 prompt 默认放在 `.moyuan/visuals/prompts/`，不参与 docs 正文巡检。
- 当图片与核心文档不一致时，以核心文档为准，并更新或废弃旧图。

## 设计原则

1. 项目隔离：每个被管理项目都拥有独立工作空间、配置、记忆、任务状态和审计记录。
2. 身份先行：任何项目、Git、Runtime、服务器、发布和模型操作都先建立 `auth_context`，再判断权限和审批。
3. 编排优先：框架不绑定某一个模型或 CLI，而是通过统一 Agent Runtime 调度不同执行后端，并把 Subagent 作为显式执行实例管理。
4. 任务图驱动：开发目标先拆成 Issue Graph，再按依赖、风险、写入范围和资源决定串行或并发。
5. 前后端分工：前端默认交给 Claude CLI，后端和后端调优默认交给 Codex CLI，最终统一回到质量门禁和 review。
6. 质量先于合入：AI 生成代码必须通过可运行性、测试覆盖率、重复度、复杂度、架构边界和独立审查。
7. 运行中自我修复：运行失败、测试失败、冒烟失败和用户反馈先判断是否 bug，再按风险自动修复或升级人工。
8. 记忆可治理：记忆先判断价值，再结构化抽取，经暂存去重、自动 compact 和维护后进入长期记忆。
9. 仓库可治理：支持本地路径、GitHub、Gitee 和通用 Git URL，任务分支、集成分支和版本分支都受系统管理。
10. 发布可控制：版本分支、tag、PR/MR、部署、线上冒烟、监控和回滚进入统一流水线。
11. 资源可维护：测试开发机和生产机统一登记，云服务器到期、健康检查、备份和维护记录可追踪。
12. 敏感信息不落盘：API key、token、SSH key、云凭证和 `.env` 明文只能以引用形式出现。

## 进入实现前

正式进入代码实现前，需要完成：

- 所有核心设计文档通过 [设计就绪门禁](./design-readiness-checklist.md)。
- 配置规则已能转换为 JSON Schema、Zod schema 或 TypeScript runtime validator。
- 契约文档已覆盖 auth、subagent/skill、self-repair、schema、runtime、logging 和 workspace migration。
- 核心对象、权限模型、失败恢复和文档维护规则没有互相冲突。
- README 只保留导航和边界，不重复专题方案细节。
