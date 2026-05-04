# Moyuan Code 文档驱动全生命周期流程图讲解

这张图基于当前 docs 文档结构重新梳理 Moyuan Code 的端到端流程。它强调文档如何约束后续实现：从用户入口、仓库接入、Project Comprehension、需求规划、Issue Graph、Scheduler、多 Agent 执行、质量合入，到发布投产、运行反馈、自我修复、Memory、测试、安全威胁模型、ADR 和设计就绪门禁。

## 1. 主流程

主流程从 CLI/API/Web Console 进入，先建立 Auth Context、RBAC、Approval 和 Audit，再进入 Git Adapter 接入本地仓库、GitHub 或 Gitee。项目初始化 `.moyuan` 后触发 Project Comprehension，产出 Project Profile、Module Map、Commands、Risk Files 和 Memory Candidates。

用户需求不会直接进入编码，而是先经过 Requirement Refiner、Clarification Gate、Issue Planner 和 Issue Graph。Scheduler 再根据 ready、blocked、running、review 队列、并发预算、worktree/runtime slots 和 write scope conflict 控制执行顺序。

## 2. Multi-Agent 执行

Multi-Agent 执行层以 Subagent Plan 为入口，绑定 role、skills、memory scope、read/write scope 和输出契约。前端和设计类任务优先 Claude CLI，后端、测试、review、quality_guard 和修复类任务优先 Codex CLI，普通模型 API 通过 Model Routing 和 Provider Policy 参与规划、总结、抽取和降级。

所有 Runtime 输出都必须回到 Output Contract、Quality Gate 和 Reviewer，不允许直接 accepted、merge、push、tag 或 deploy。

## 3. Workspace / State

图中间的 Workspace / State 是系统事实沉淀层。关键状态包括 project/repository 配置、comprehension 结果、lifecycle issue graph、subagent 实例、runtime session、memory、server resources 和 logs/audit。

这些状态由 Workspace、State Store、持久化与并发一致性规则保护，必须支持原子写、锁、版本、事务 journal、append-only log 和崩溃恢复。

## 4. 右侧控制面

Server Resources 管理 test_dev 和 production 资源，记录云元数据、到期时间、健康检查和 owner。Provider & Runtime 管理 GPT、Claude、GLM、MiniMax、第三方 API 和 Runtime Adapter。

Security Model 来自安全威胁模型，关注 Threat Model、protected paths、data policy 和 secret redaction。ADR & Readiness 记录关键架构决策、实现模块拆分、测试策略和一致性规则，确保后续实现不偏离文档。

## 5. 底部反馈治理

Runtime Signals 汇聚运行错误、测试失败、冒烟失败和监控异常。Self Repair 将信号归类为 Bug Candidate，再按风险创建 Repair Attempt，并要求 regression、Quality Gate 和 Review。

Agent Memory 通过 Record Gate、Retrieve 和 Memory Compact 管理长期记忆。Framework Tests 使用 fake runtime、golden fixtures 和 recovery tests 验证 Moyuan 本体。Documentation Governance 则用 schema index、contracts 和 design readiness 控制文档权威边界。

生成文件：

- 图片：moyuan-code-architecture-2026-05-04T02-40-50-228Z.png
- Prompt：.moyuan/visuals/prompts/moyuan-code-architecture-2026-05-04T02-40-50-228Z.prompt.md
- 讲解：moyuan-code-architecture-2026-05-04T02-40-50-228Z.explanation.md
