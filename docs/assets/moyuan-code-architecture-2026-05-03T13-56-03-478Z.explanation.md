# Moyuan Code 技术流程图讲解

这张图展示当前项目的端到端技术流程：用户通过 CLI/API/Web Console 提交项目接入、开发任务、发布或维护请求，系统先建立 Auth Context 和权限边界，再进入仓库接入、项目理解、需求规划、Issue Graph、Subagent 执行、质量门禁、Git 合入、发布投产和长期反馈闭环。

## 1. 入口与控制面

入口层承接 Platform User、CLI、API 和 Web Console。任何操作进入项目之前，都必须先形成 Auth Context，再经过 RBAC、Approval 和 Audit 判断。高风险动作，例如生产部署、Git push、tag、密钥访问、服务器命令和策略变更，不能绕过审批与审计。

## 2. 仓库接入与项目理解

仓库接入支持 Local Path、GitHub、Gitee 和 Generic Git。Git Adapter 负责 clone、fetch、branch、worktree、push、PR/MR 能力声明和用户改动保护。项目接入后会初始化独立 .moyuan 工作空间，并立即触发 Full Project Comprehension；每次远程同步、rebase、merge 或任务完成后触发 Incremental/Diff Comprehension。

阅读理解产物包括 Project Profile、Module Map、Commands、Risk Files 和 Memory Candidates。这些产物不是完整源码复制，而是后续需求规划、Agent 上下文装配、质量判断和记忆检索的稳定事实来源。

## 3. 需求规划与 Issue Graph

用户提出开发任务后，不直接进入编码。Requirement Refiner 会补齐背景、范围、约束、验收和风险；Clarification Gate 判断是否必须追问用户。信息足够后，Issue Planner 拆分 issues，Dependency Planner 构建 DAG，Scheduler 计算 ready_queue、blocked_queue、running_queue 和 review_queue。

Issue Graph 是系统调度的核心。它控制哪些 issue 可以并发、哪些必须等待契约、后端、前端、Runtime slot、worktree、质量门禁或用户审批。用户可以看到 issue graph、blocked reason 和并发计划。

## 4. Subagent 执行平面

Issue 被调度后，Orchestrator 不直接调用模型，而是创建 Subagent Plan。Subagent 绑定父对象、role、skills、memory scope、read/write scope、Runtime 和输出契约。

默认分工是：前端和复杂 UI/架构任务优先使用 Claude CLI；后端、测试、review、quality_guard、repair 和后端调优优先使用 Codex CLI。GPT、Claude、GLM、MiniMax、DeepSeek、DashScope 和第三方 API 通过 Provider Registry 和 Model Routing 参与规划、审查、摘要、Memory record gate、抽取和降级 fallback。

Skills Registry 和 find-skills 负责推荐能力包。Skill Binding 可按 project、role、issue 或 subagent 绑定，并通过 Skill Effectiveness 记录效果，避免长期使用低质量技能。

## 5. 代码质量与合入

每个 issue 使用独立 issue branch 或 issue worktree。Subagent 完成代码修改后，系统必须执行 build、lint、typecheck、unit tests、integration tests、coverage、重复度、复杂度、架构边界和安全检查。

Quality Gate 和 Reviewer 都通过后，issue 才能 accepted，并合入 epic integration branch。失败时进入 needs_rework 或 replan，不允许把未复核的 AI 代码直接合入主分支。

## 6. 版本发布与服务器 DevOps

当 integration branch 累积足够 accepted issues，Release Manager 根据风险、issue 数量、变更范围、迁移、安全和公共 API 变更生成 release suggestion。发布流程包括 release branch、release note、tag、push 到 GitHub/Gitee、PR/MR、回归测试和审批。

投产阶段读取 environments 和 server resources。服务器资源区分 test_dev 和 production，记录云厂商、规格、到期时间、续费 owner、健康检查、备份和维护策略。生产部署必须执行备份、线上冒烟、监控窗口和回滚判断。

## 7. 反馈闭环和长期治理

运行失败、测试失败、冒烟失败、监控异常、review finding 或用户反馈会进入 Runtime Signals。系统先判断是否为 Bug Candidate，再决定是否自动 Repair Attempt。成功修复会生成 Improvement Record，并可能进入 Memory、Skill 效果反馈或质量策略增强。

Agent Memory 通过 Record Gate、Extraction、Staging Dedup、Async Write、Retrieval、Automatic Compact 和 Reflection 管理长期记忆。统一日志记录 run、agent、model、git、quality、release、deployment、memory、audit 和 error，保证每一次自动化行为可追踪。

## 8. .moyuan 工作空间

每个被管理项目都有独立 .moyuan 工作空间。核心目录包括 project、repository、agents、models、runtimes、skills、memory、logs、comprehension、resources、lifecycle 和 policies。配置 Schema、契约文档、状态机和文档治理共同保证后续实现不会把对象字段、流程规则和策略判断散落到多个不一致的位置。

## 9. gpt-image-2 的角色

gpt-image-2 只用于架构图、流程图、部署拓扑图和讲解资产生成。它接收脱敏后的 diagram spec 和视觉 prompt，不参与代码生成、代码审查、质量合入或发布决策。

生成文件：

- 图片：moyuan-code-architecture-2026-05-03T13-56-03-478Z.png
- Prompt：.moyuan/visuals/prompts/moyuan-code-architecture-2026-05-03T13-56-03-478Z.prompt.md
- 讲解：moyuan-code-architecture-2026-05-03T13-56-03-478Z.explanation.md
