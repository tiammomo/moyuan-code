# 术语表

本文统一 Moyuan Code 的核心术语。后续文档和实现代码应优先使用本文命名，避免同一概念多种叫法。

## 术语总表

| 术语 | 英文 | 定义 | 不是 | 权威文档 |
| --- | --- | --- | --- | --- |
| Moyuan | Moyuan Code | 多 Agent 代码开发框架本体 | 不是被管理项目 | [总体规划](../lifecycle-roadmap.md) |
| 被管理项目 | Project | Moyuan 接入和管理的软件项目 | 不是 Moyuan 自身仓库 | [项目工作空间规范](../project-workspace-spec.md) |
| 工作空间 | Workspace | 每个项目独立的 `.moyuan/` 配置、状态和产物目录 | 不是源代码目录本身 | [项目工作空间规范](../project-workspace-spec.md) |
| 仓库 | Repository | 被管理项目的 Git 仓库，可以来自本地路径或远程 URL | 不等同于 Workspace | [仓库接入、Git 与项目理解](../repository-onboarding-git-management.md) |
| 项目理解 | Project Comprehension | 对项目结构、模块、命令、依赖和风险的阅读理解结果 | 不是一次性摘要 | [仓库接入、Git 与项目理解](../repository-onboarding-git-management.md) |
| 项目画像 | Project Profile | 项目理解后的稳定画像，包括技术栈、模块、命令和风险 | 不是完整源码复制 | [仓库接入、Git 与项目理解](../repository-onboarding-git-management.md) |
| 模块地图 | Module Map | 项目模块边界、职责和依赖关系 | 不是文件树的简单罗列 | [仓库接入、Git 与项目理解](../repository-onboarding-git-management.md) |
| Epic | Epic | 用户提出的开发目标，会被拆成多个 issues | 不是单个开发任务 | [Issues 编排与并发调度](../issue-orchestration.md) |
| Issue | Issue | 最小可执行开发单元，具备依赖、写入范围、验收标准和测试计划 | 不等同于 GitHub Issue | [Issues 编排与并发调度](../issue-orchestration.md) |
| Issue Graph | Issue Graph | Issues 之间的依赖 DAG，用于判断串行、并行和阻塞关系 | 不是简单任务列表 | [Issues 编排与并发调度](../issue-orchestration.md) |
| Ready Queue | Ready Queue | 当前依赖已满足、可以被调度执行的 issue 队列 | 不是所有未完成 issue | [Issues 编排与并发调度](../issue-orchestration.md) |
| Schedule | Schedule | Orchestrator 生成的执行排期、并发度和 worktree 分配 | 不是最终执行结果 | [Issues 编排与并发调度](../issue-orchestration.md) |
| Run | Run | 一次任务执行实例，记录 Agent、模型、工具、Git、质量、测试和 Memory 信息 | 不是 Issue 本身 | [项目工作空间规范](../project-workspace-spec.md) |
| Agent | Agent | 角色、工具权限、Memory scope、skills、模型策略和输出契约的组合 | 不等同于某个模型 | [Agent、Skills 与编排](../agent-skills-memory.md) |
| Role | Role | Agent 的职责定义，例如 backend、tester、reviewer | 不是具体执行进程 | [Agent、Skills 与编排](../agent-skills-memory.md) |
| Team | Team | 一组 Agent role 的协作编排配置 | 不是组织团队 | [Agent、Skills 与编排](../agent-skills-memory.md) |
| Skill | Skill | 可被 Agent 引用的专门能力、提示模板或工具知识 | 不是 Agent 本身 | [Agent、Skills 与编排](../agent-skills-memory.md) |
| Orchestrator | Orchestrator | 核心编排层，负责需求、Issue Graph、调度、状态、权限和合入决策 | 不是模型调用封装 | [参考架构](../reference-architecture.md) |
| Agent Runtime | Agent Runtime | 执行 Agent 的运行时后端，可以是 CLI、API 或本地工具链 | 不是普通模型 provider | [模型与工具适配规划](../model-tool-adapters.md) |
| Native Agent Runtime | Native Agent Runtime | Claude CLI、Codex CLI 这类能直接读写仓库和执行工具的强 Agent 后端 | 不是纯文本 LLM API | [模型与工具适配规划](../model-tool-adapters.md) |
| Adapter | Adapter | 外部能力的统一封装，例如 Git、Shell、Codex、Claude、模型 API、MCP | 不是业务编排层 | [模型与工具适配规划](../model-tool-adapters.md) |
| Provider | Provider | 模型或远程服务的服务商账号和 API 能力登记 | 不是路由策略 | [模型与工具适配规划](../model-tool-adapters.md) |
| Model Policy | Model Policy | 针对任务类型选择模型 provider 和 fallback 的路由规则 | 不是具体模型账号 | [完整配置方案](../configuration-guide.md) |
| Third-party API | Third-party API | 非官方或聚合型 OpenAI-compatible 模型网关 | 不能默认处理敏感上下文 | [模型与工具适配规划](../model-tool-adapters.md) |
| Memory | Memory | 可检索、可维护、可审计的长期项目记忆 | 不是聊天上下文缓存 | [Agent Memory 系统方案](../agent-memory-system.md) |
| Record Gate | Record Gate | 判断信息是否值得进入 Memory 的决策环节 | 不是信息抽取 | [Agent Memory 系统方案](../agent-memory-system.md) |
| Retrieve | Retrieve | 任务执行前从 Memory 检索相关历史信息 | 不是全文注入 | [Agent Memory 系统方案](../agent-memory-system.md) |
| Compact | Compact | 对 Memory 自动压缩、合并、去重和整理 | 不是简单删除 | [Agent Memory 系统方案](../agent-memory-system.md) |
| Quality Gate | Quality Gate | 对 AI 生成代码执行的可运行性、测试、重复、复杂度、架构和安全检查 | 不是人工 review 的替代 | [代码生命周期质量门禁](../code-lifecycle-quality-gates.md) |
| Review | Review | 对 diff、风险、测试缺口和可维护性的独立审核 | 不是测试命令 | [代码生命周期质量门禁](../code-lifecycle-quality-gates.md) |
| Worktree | Worktree | Git worktree，用于隔离并行 issue 开发 | 不是长期分支策略 | [Issues 编排与并发调度](../issue-orchestration.md) |
| Task Branch | Task Branch | 单个 issue 或 task 的开发分支 | 不是 release branch | [仓库接入、Git 与项目理解](../repository-onboarding-git-management.md) |
| Epic Branch | Epic Branch | 一个 Epic 的集成分支，用于合并已验收 issues | 不是默认主分支 | [Issues 编排与并发调度](../issue-orchestration.md) |
| Release Branch | Release Branch | 发布候选分支 | 不是任务开发分支 | [Issues 编排与并发调度](../issue-orchestration.md) |
| Release | Release | 从 accepted issues 到版本分支、回归、tag、PR/MR 和发布记录的过程 | 不等同于部署 | [Issues 编排与并发调度](../issue-orchestration.md) |
| Deployment | Deployment | 将发布版本部署到目标环境和资源组的过程 | 不等同于 Git push | [Issues 编排与并发调度](../issue-orchestration.md) |
| Environment | Environment | test、staging、production 等部署环境配置 | 不是单台机器 | [完整配置方案](../configuration-guide.md) |
| Server Resource | Server Resource | 被登记和维护的服务器资产，包括云信息、规格、到期和健康检查 | 不是部署环境 | [完整配置方案](../configuration-guide.md) |
| Resource Group | Resource Group | 一组服务器资源，用于环境部署引用 | 不是 Agent 分组 | [完整配置方案](../configuration-guide.md) |
| Unified Logs | Unified Logs | run、agent、model、git、quality、release、memory、audit、error 等核心日志 | 不是业务应用日志 | [完整配置方案](../configuration-guide.md) |
| Audit Log | Audit Log | 审批、密钥访问、高风险命令和保护路径访问的不可变审计事件 | 不是普通 debug log | [完整配置方案](../configuration-guide.md) |
| Visual Diagram | Visual Diagram | 由 gpt-image-2 辅助生成的架构流程图、部署拓扑图或讲解资产 | 不是代码事实来源 | [完整配置方案](../configuration-guide.md) |
| Diagram Spec | Diagram Spec | 生成架构图前的结构化图定义，包括节点、边、层级和敏感信息省略项 | 不是图片文件 | [完整配置方案](../configuration-guide.md) |

## 命名约定

- 文档中使用英文对象名时首字母大写，例如 `Project`、`Issue`、`Run`。
- 配置字段使用 snake_case。
- CLI 命令使用 kebab 或空格分组，例如 `moyuan issue graph`。
- 文件和目录使用 kebab-case 或既有生态约定。

## 避免混淆

- `Issue` 是 Moyuan 内部开发单元，不默认等同于 GitHub/Gitee/GitLab issue。
- `Runtime` 是执行后端，`Provider` 是模型或服务商账号能力。
- `Environment` 是部署环境，`Resource Group` 是服务器集合。
- `Memory` 是长期记忆，`Runtime State` 是当前执行状态。
- `Visual Diagram` 是辅助讲解产物，不作为架构事实的唯一来源。
