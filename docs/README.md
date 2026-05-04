# Moyuan Code 文档

当前阶段：规划设计。

`moyuan-code` 是面向代码开发全生命周期的多 Agent 开发框架。系统在理解项目代码的基础上，把用户需求完善为可执行的 Issue Graph，调度 Claude CLI、Codex CLI 和多种模型 Provider 分工开发，并通过鉴权、质量门禁、Git、发布投产、Memory、日志和自我修复持续管理项目迭代。

本文只作为 docs 入口，不承载完整方案。

## 推荐阅读顺序

1. [总体规划与生命周期路线图](./lifecycle-roadmap.md)：产品定位、端到端流程、CLI、Phase 和近期落地范围。
2. [参考架构](./reference-architecture.md)：系统分层、核心模块、运行链路、上下文装配和安全边界。
3. [主线文档](./mainlines/README.md)：按真实生命周期阅读平台用户、项目接入、需求规划、代码开发、运行反馈、代码管理、服务器资源和 DevOps 发布投产。
4. [策略决策树](./policies/README.md)：阅读鉴权、阅读理解、调度、质量、Bug 判断、Git、服务器、发布、Provider 和 Memory 的判断规则。
5. [契约文档](./contracts/README.md)：进入实现前确认 auth、subagent/skill、self-repair、schema、runtime、logging 和 workspace migration 的接口边界。
6. [项目工作空间规范](./project-workspace-spec.md)、[配置方案](./configuration-guide.md)、[配置 Schema 规则](./configuration-schema-spec.md)：理解每个被管理项目的 `.moyuan/` 工作空间和配置校验边界。
7. [Agent Memory 系统方案](./agent-memory-system.md)、[Subagent 与 Skills 系统方案](./subagents-skills-system.md)、[模型与工具适配规划](./model-tool-adapters.md)、[后端技术栈与本地环境](./backend-tech-stack.md)：阅读关键横切能力和 Go/Python 职责边界。
8. [实现模块拆分](./implementation-module-map.md)、[框架自身测试策略](./framework-testing-strategy.md)、[持久化与并发一致性](./persistence-concurrency-consistency.md)：进入代码实现前确认模块、测试、状态和锁。
9. [安全威胁模型](./threat-model.md)、[设计就绪门禁](./design-readiness-checklist.md)、[ADR](./adr/README.md)：确认生产级实现前的安全、评审和架构决策。

## 文档分层

| 层级 | 入口 | 作用 |
| --- | --- | --- |
| 基础规范 | [foundations/](./foundations/README.md) | 术语、核心对象、权限、失败恢复、状态机和文档治理 |
| 总体规划 | [lifecycle-roadmap.md](./lifecycle-roadmap.md)、[reference-architecture.md](./reference-architecture.md) | 产品生命周期、系统架构和阶段计划 |
| 主线流程 | [mainlines/](./mainlines/README.md) | 真实执行链路和每条主线的输入、输出、阻断点 |
| 策略决策 | [policies/](./policies/README.md) | 可实现为规则引擎、状态机或 validator 的判断规则 |
| 实现契约 | [contracts/](./contracts/README.md) | 模块接口、事件、错误和迁移边界 |
| 专题设计 | 本目录专题文档 | 单一能力的权威展开 |
| 决策记录 | [adr/](./adr/README.md) | 关键技术决策、替代方案和影响 |
| 辅助资产 | [assets/](./assets/) | gpt-image-2 生成的辅助图和说明，不是核心设计来源 |

## 核心专题入口

| 能力 | 文档 |
| --- | --- |
| Agent role、team 和默认 Runtime | [Agent 角色与团队概览](./agent-roles-overview.md) |
| Subagent 生命周期、Skill Registry、推荐、绑定和效果反馈 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| Agent Memory record、retrieve、compact 和维护 | [Agent Memory 系统方案](./agent-memory-system.md) |
| 后端技术栈、Go 本地环境和 Python 本地环境 | [后端技术栈与本地环境](./backend-tech-stack.md) |
| Issue Graph、并发调度和等待模型 | [Issues 编排与并发调度](./issue-orchestration.md) |
| AI 代码质量门禁、review 和返工闭环 | [代码生命周期质量门禁](./code-lifecycle-quality-gates.md) |
| commit、issue、fix、release 和覆盖率规范 | [工程流程规范](./engineering-process-standards.md) |
| Git Provider 能力、认证和降级 | [Git Provider 接入配置](./git-provider-integration.md) |
| 仓库接入与 Git Provider Adapter 触发点 | [仓库接入与 Git Provider Adapter](./repository-onboarding-git-management.md) |

## 设计边界

1. 每个被管理项目都有独立 `.moyuan/` 工作空间、配置、任务状态、Memory 和审计记录。
2. 所有项目、Git、Runtime、Provider、服务器、发布和模型操作都先建立 `auth_context`。
3. 用户需求必须先进入需求完善、澄清判断和 Issue Graph，再进入代码开发。
4. 前端默认交给 Claude CLI，后端和后端调优默认交给 Codex CLI，最终统一回到质量门禁和 review。
5. AI 生成代码必须通过可运行性、测试覆盖率、重复度、复杂度、架构边界、安全和独立审查。
6. Memory 只以 [Agent Memory 系统方案](./agent-memory-system.md) 为唯一详细方案，其他文档只引用。
7. Provider Registry、Skill Registry、Project Registry、Server Resource Registry 和 Agent Runtime Registry 是配置注册表，不是服务注册发现系统。
8. gpt-image-2 只用于架构图、流程图和讲解资产，不作为代码事实来源。
9. API key、token、SSH key、云凭证和 `.env` 明文只能以引用形式出现，不能写入日志、Memory、prompt 或文档。

## 进入实现前

正式进入代码实现前，需要完成：

- 所有核心设计文档通过 [设计就绪门禁](./design-readiness-checklist.md)。
- 配置规则可以转换为 JSON Schema、Zod schema 或 TypeScript runtime validator。
- 契约文档覆盖 auth、subagent/skill、self-repair、schema、runtime、logging 和 workspace migration。
- 实现模块拆分、框架测试策略、持久化与并发一致性已经通过设计评审。
- 安全威胁模型和关键 ADR 已通过评审。
- README 只保留导航和边界，不重复专题方案细节。
