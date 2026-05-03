# Moyuan Code 总体结构设计图讲解

这张图展示当前项目的规划结构：用户通过 CLI/API/Web Console 接入本地或远程仓库，系统先进行项目理解，再由 Orchestrator 完成需求完善、澄清判断、Issue 拆分、依赖图构建和并发调度。

执行层由 Native Agent Runtime 和模型 API 共同组成。Claude CLI 与 Codex CLI 作为强 Agent 后端，负责复杂代码任务；GPT、Claude、GLM、MiniMax、DeepSeek、DashScope 和第三方 API 通过 Provider Registry 统一管理，并按 routing 策略参与规划、审查、摘要和记忆抽取。

每个项目拥有独立的 .moyuan 工作空间，保存项目配置、仓库策略、Agent 角色、模型配置、Runtime 会话、Memory、Logs、服务器资源、生命周期记录和质量策略。

代码生命周期从 Issue Graph 进入独立 worktree 或任务分支，经过代码生成、测试、lint、build、质量门禁、独立 review 后，才能合入 epic integration branch。发布阶段创建 release branch，推送 GitHub/Gitee，并按服务器资源组部署到测试开发机或生产机，随后执行线上冒烟、监控和回滚判断。

底部反馈闭环包括 Agent Memory、统一日志、服务器资源长期维护和 gpt-image-2 架构可视化。它们共同保证系统能持续理解项目、追踪决策、控制质量并辅助讲解架构。

生成文件：

- 图片：moyuan-code-architecture-2026-05-03T07-01-24-041Z.png
- Prompt：../../.moyuan/visuals/prompts/moyuan-code-architecture-2026-05-03T07-01-24-041Z.prompt.md
- 讲解：moyuan-code-architecture-2026-05-03T07-01-24-041Z.explanation.md
