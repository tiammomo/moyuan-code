# Moyuan Code 文档

当前阶段：Phase 19 已启动，聚焦受控自动化执行增强与生产可观测性深化；Phase 18 operations timeline、maintenance policy pack、post-deployment verification、服务器生命周期控制和 Console 运维 dashboard 已完成 readiness。

`moyuan-code` 是面向代码开发全生命周期的多 Agent 开发框架。系统在理解项目代码的基础上，把用户需求完善为可执行的 Issue Graph，调度 Claude CLI、Codex CLI 和多种模型 Provider 分工开发，并通过鉴权、质量门禁、Git、发布投产、Memory、日志和自我修复持续管理项目迭代。

本文只作为 docs 入口，不承载完整方案。

## 推荐阅读顺序

1. [总体规划与生命周期路线图](./lifecycle-roadmap.md)：产品定位、端到端流程、CLI、Phase 和近期落地范围。
2. [Phase 规划与执行记录](./phases/README.md)：查看当前验收状态、完成范围、issue graph 和剩余边界。
3. [参考架构](./reference-architecture.md)：系统分层、核心模块、运行链路、上下文装配和安全边界。
4. [主线文档](./mainlines/README.md)：按真实生命周期阅读平台用户、项目接入、需求规划、代码开发、运行反馈、代码管理、服务器资源和 DevOps 发布投产。
5. [策略决策树](./policies/README.md)：阅读鉴权、阅读理解、调度、质量、Bug 判断、Git、服务器、发布、Provider 和 Memory 的判断规则。
6. [契约文档](./contracts/README.md)：确认 auth、secret resolver、subagent/skill、self-repair、schema、runtime、logging 和 workspace migration 的接口边界。
7. [项目工作空间规范](./project-workspace-spec.md)、[配置方案](./configuration-guide.md)、[配置 Schema 规则](./configuration-schema-spec.md)：理解每个被管理项目的 `.moyuan/` 工作空间和配置校验边界。
8. [Agent Memory 系统方案](./agent-memory-system.md)、[Subagent 与 Skills 系统方案](./subagents-skills-system.md)、[模型与工具适配规划](./model-tool-adapters.md)、[后端技术栈与本地环境](./backend-tech-stack.md)、[前端控制台文档](./frontend/README.md)：阅读关键横切能力和前后端职责边界。
9. [实现模块拆分](./implementation-module-map.md)、[框架自身测试策略](./framework-testing-strategy.md)、[持久化与并发一致性](./persistence-concurrency-consistency.md)：确认模块、测试、状态和锁。
10. [安全威胁模型](./threat-model.md)、[设计就绪门禁](./design-readiness-checklist.md)、[ADR](./adr/README.md)：确认生产级实现前的安全、评审和架构决策。

## 文档分层

| 层级 | 入口 | 作用 |
| --- | --- | --- |
| 基础规范 | [foundations/](./foundations/README.md) | 术语、核心对象、权限、失败恢复、状态机和文档治理 |
| 总体规划 | [lifecycle-roadmap.md](./lifecycle-roadmap.md)、[reference-architecture.md](./reference-architecture.md) | 产品生命周期、系统架构和阶段计划 |
| 实施入口 | [phases/](./phases/README.md) | Phase 验收、完成状态、依赖和剩余边界 |
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
| Web Console、Next.js 16、前端端口和控制台设计模式 | [前端控制台文档](./frontend/README.md) |
| Issue Graph、并发调度和等待模型 | [Issues 编排与并发调度](./issue-orchestration.md) |
| AI 代码质量门禁、review 和返工闭环 | [代码生命周期质量门禁](./code-lifecycle-quality-gates.md) |
| commit、issue、fix、release 和覆盖率规范 | [工程流程规范](./engineering-process-standards.md) |
| Git Provider 能力、认证和降级 | [Git Provider 接入配置](./git-provider-integration.md) |
| 仓库接入与 Git Provider Adapter 触发点 | [仓库接入与 Git Provider Adapter](./repository-onboarding-git-management.md) |
| Secret 引用、用途校验、注入和审计 | [Secret Resolver 契约](./contracts/secret-resolver-contract.md) |

## 设计边界

1. 每个被管理项目都有独立 `.moyuan/` 工作空间、配置、任务状态、Memory 和审计记录。
2. 所有项目、Git、Runtime、Provider、服务器、发布和模型操作都先建立 `auth_context`。
3. 用户需求必须先进入需求完善、澄清判断和 Issue Graph，再进入代码开发。
4. 前端复杂 UI 首版可优先交给 Claude CLI，样式稳定后的前端代码修改、测试、修复和重构可由 Codex CLI 参与或主导；后端和后端调优默认交给 Codex CLI，最终统一回到质量门禁和 review；Web Console 技术栈冻结为 Next.js 16，前端端口 `3000`，后端 API 端口 `8080`。
5. AI 生成代码必须通过可运行性、测试覆盖率、重复度、复杂度、架构边界、安全和独立审查。
6. Memory 只以 [Agent Memory 系统方案](./agent-memory-system.md) 为唯一详细方案，其他文档只引用。
7. Provider Registry、Skill Registry、Project Registry、Server Resource Registry 和 Agent Runtime Registry 是配置注册表，不是服务注册发现系统。
8. gpt-image-2 只用于架构图、流程图和讲解资产，不作为代码事实来源。
9. API key、token、SSH key、云凭证和 `.env` 明文只能以引用形式出现，不能写入日志、Memory、prompt 或文档。

## 当前实施入口

Phase 1 本地 CLI MVP、Beta 控制面能力、Phase 2 到 Phase 18 已完成主要闭环。当前验收和状态入口：

- [Phase 19 实现 Issue Graph](./phases/phase19-issue-graph.md)：受控自动化执行增强与生产可观测性深化的依赖图。
- [Phase 19 实施记录](./phases/phase19-next-development-plan.md)：Phase 19 当前任务、验收标准和执行入口。
- [Phase 18 Release Readiness](./phases/phase18-release-readiness.md)：生产运维闭环与策略化维护控制面的收口验证。
- [Phase 18 实现 Issue Graph](./phases/phase18-issue-graph.md)：生产运维闭环与策略化维护控制面的依赖图。
- [Phase 18 实施记录](./phases/phase18-next-development-plan.md)：Phase 18 完成任务、验收标准和执行记录。
- [Phase 17 实现 Issue Graph](./phases/phase17-issue-graph.md)：发布准入策略包、演练调度与风险修复 drill-down 的依赖图。
- [Phase 17 实施记录](./phases/phase17-next-development-plan.md)：Phase 17 当前任务、验收标准和执行入口。
- [Phase 17 Release Readiness](./phases/phase17-release-readiness.md)：Phase 17 完成范围、门禁结论、保留边界和 Phase 18 入口。
- [Phase 16 Release Readiness](./phases/phase16-release-readiness.md)：部署演练、运行风险闭环与发布准入增强的收口验证。
- [Phase 16 实现 Issue Graph](./phases/phase16-issue-graph.md)：部署演练、运行风险闭环与发布准入增强的依赖图。
- [Phase 16 实施记录](./phases/phase16-next-development-plan.md)：Phase 16 完成任务、验收标准和执行入口。
- [Phase 15 Release Readiness](./phases/phase15-release-readiness.md)：部署审批加固、回退执行与生产可观测性增强的收口验证。
- [Phase 15 实现 Issue Graph](./phases/phase15-issue-graph.md)：部署审批加固、回退执行与生产可观测性的依赖图。
- [Phase 15 实施记录](./phases/phase15-next-development-plan.md)：Phase 15 完成任务、验收标准和执行入口。
- [Phase 14 Release Readiness](./phases/phase14-release-readiness.md)：受控远程发布与部署执行的收口验证。
- [Phase 14 实现 Issue Graph](./phases/phase14-issue-graph.md)：受控远程发布与部署执行的依赖图。
- [Phase 14 实施记录](./phases/phase14-next-development-plan.md)：Phase 14 完成任务、验收标准和执行入口。
- [Phase 13 Release Readiness](./phases/phase13-release-readiness.md)：Release Candidate 远程发布与部署交接的收口验证。
- [Phase 13 实现 Issue Graph](./phases/phase13-issue-graph.md)：Release Candidate 远程发布与部署交接的依赖图。
- [Phase 13 实施记录](./phases/phase13-next-development-plan.md)：Phase 13 完成任务、验收标准和执行入口。
- [Phase 12 Release Readiness](./phases/phase12-release-readiness.md)：真实并发执行与集成合入准备的收口验证。
- [Phase 12 实现 Issue Graph](./phases/phase12-issue-graph.md)：真实并发执行与集成合入准备的依赖图。
- [Phase 12 实施记录](./phases/phase12-next-development-plan.md)：Phase 12 完成任务、验收标准和执行入口。
- [Phase 11 Release Readiness](./phases/phase11-release-readiness.md)：Issue Graph 批量执行控制器的收口验证。
- [Phase 11 实现 Issue Graph](./phases/phase11-issue-graph.md)：batch plan、batch run、worktree isolation、merge queue 和 Console 操作面的依赖图。
- [Phase 11 实施记录](./phases/phase11-next-development-plan.md)：Phase 11 完成任务、验收标准和边界。
- [Phase 规划与执行记录](./phases/README.md)：历史 Phase、release readiness 和执行记录总入口。
- [设计就绪门禁](./design-readiness-checklist.md)：设计风险仍按 `READY_WITH_RISKS` 跟踪，不能绕过实现门禁。
