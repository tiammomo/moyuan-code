# Moyuan Code 文档

当前阶段：规划设计。

`moyuan-code` 目标是构建一个面向代码开发全生命周期的多 Agent 开发框架。框架需要统一调度 Claude Code、Codex、本地/远程命令行工具以及多种国产大模型 API，并支持按项目隔离工作空间、配置、记忆、skills 和迭代记录。

## 文档索引

- [基础规范](./foundations/README.md)：术语、核心数据对象、权限模型、失败恢复和文档维护规则。
- [设计就绪门禁](./design-readiness-checklist.md)：进入代码实现前必须满足的文档完整性和风险收敛标准。
- [总体规划与生命周期路线图](./lifecycle-roadmap.md)：产品定位、端到端流程、CLI 路线、Phase 和近期任务。
- [参考架构](./reference-architecture.md)：系统分层、核心模块、运行链路和安全边界。
- [项目工作空间规范](./project-workspace-spec.md)：`.moyuan/` schema 索引、目录职责和配置归属。
- [完整配置方案](./configuration-guide.md)：项目接入、模型、gpt-image-2 架构可视化、Agent、编排、质量、Memory、核心日志、服务器资源、发布投产和环境配置。
- [配置 Schema 规则](./configuration-schema-spec.md)：所有 `.moyuan/` 配置字段的必填、可选、可为空和必须为空规则。
- [Issues 编排与并发调度](./issue-orchestration.md)：需求自动拆分、依赖图、ready queue、并发度和 issue 执行编排。
- [仓库接入、Git 与项目理解](./repository-onboarding-git-management.md)：本地/远程仓库接入、任务分支、项目阅读理解。
- [GitHub 接入配置](./github-integration.md)：GitHub 仓库连接、认证、token 权限、必填和可空字段。
- [代码生命周期质量门禁](./code-lifecycle-quality-gates.md)：AI 生成代码后的验证、审查、复杂度、重复度和返工机制。
- [Agent、Skills 与编排](./agent-skills-memory.md)：角色体系、team、skills、memory scope 和输出契约。
- [Agent Memory 系统方案](./agent-memory-system.md)：记忆判断、抽取、暂存去重、异步写入、分层存储和动态维护。
- [模型与工具适配规划](./model-tool-adapters.md)：Claude Code、Codex、GPT、Claude、GLM、MiniMax、gpt-image-2、第三方模型 API 和工具执行适配层。

## 设计原则

1. 项目隔离：每个被管理项目都拥有独立工作空间、配置、记忆、任务状态和审计记录。
2. 编排优先：框架不把自己绑定到某一个模型或 CLI，而是通过统一 Agent Runtime 调度不同执行后端。
3. 可替换适配：Claude Code、Codex、国产大模型、MCP、命令行工具都通过 adapter 接入。
4. Issue 编排：开发目标自动拆成 issues，按依赖图和冲突检测决定串行或并发执行。
5. 生命周期管理：需求、设计、开发、评审、测试、发布、回归和复盘都进入可追踪流程。
6. 仓库可接入：被管理项目可以来自本地路径或远程 Git 仓库，远程仓库可覆盖 GitHub、Gitee 和通用 Git URL。
7. 分支可治理：每个任务默认在独立工作分支中执行，框架自动创建、同步、检查和收尾分支。
8. 自动理解：每次项目接入后、每次远程分支拉取后，都要执行项目阅读理解并更新项目画像。
9. 记忆可治理：记忆先判断价值，再结构化抽取，经暂存去重后异步写入，并定期维护。
10. 日志可追踪：run、agent、model、Git、质量门禁、Memory、发布部署和错误都进入统一核心日志，敏感信息默认脱敏。
11. 资源可治理：测试开发机和生产机进入统一资源清单，环境部署只引用资源组，生产操作必须走 release/deploy pipeline。
12. 质量门禁：AI 生成代码必须通过验证、审查、重复度、复杂度和测试缺口检查，失败则进入返工。
13. 人在关键环节确认：权限提升、跨目录写入、发布、删除、密钥访问和高风险命令需要策略控制。
