
你是资深软件架构图设计师。请根据下面的设计规格，生成一张适合技术评审会议展示的技术流程图。


图名：Moyuan Code 多 Agent 研发全生命周期技术流程图
目标：展示当前项目从用户需求、仓库接入、项目理解、Issue Graph、Subagent 并发执行、质量复核、Git 合入、发布投产到 Memory/日志反馈的完整技术流程。
受众：技术负责人、架构师、后端/前端/测试/运维 Agent 配置人员、后续实现工程师。

画面布局：
请用横向泳道图或分层流程图。必须包含 7 条技术主线，并用箭头展示主流程和反馈回路。

1. 入口与控制面
   - Platform User / CLI / API / Web Console
   - Auth Context / RBAC / Approval / Audit
   - Secret refs, no plaintext secrets

2. 仓库接入与项目理解
   - Local Path / GitHub / Gitee / Generic Git
   - Git Adapter: clone, fetch, branch, worktree, PR/MR capability
   - .moyuan Workspace initialization
   - Full / Incremental / Diff Project Comprehension
   - Project Profile, Module Map, Commands, Risk Files, Memory Candidates

3. 需求规划与 Issue Graph
   - Requirement Refiner
   - Clarification Gate: needs_user_input or proceed
   - Issue Planner
   - Dependency Planner
   - Scheduler
   - User-visible Issue Graph
   - ready_queue / blocked_queue / running_queue / review_queue

4. Subagent 执行平面
   - Agent Roles Overview: frontend, backend, backend_tuning, tester, quality_guard, reviewer, release_manager
   - Subagent Manager: parent issue/run, role, skills, memory scope, write scope
   - Skills Registry / find-skills / Skill Binding / Effectiveness
   - Runtime Adapter
   - Claude CLI for frontend and architecture-heavy work
   - Codex CLI for backend, tests, review, quality and repair
   - Model Providers: GPT, Claude, GLM, MiniMax, DeepSeek, DashScope, Third-party API

5. 代码质量与合入
   - issue branch / issue worktree
   - code edits and generated tests
   - build / lint / typecheck / unit tests / integration tests
   - coverage, duplication, complexity, architecture boundary, security
   - independent review
   - accepted -> merge to epic integration branch
   - failed -> needs_rework -> replan

6. 版本发布与服务器 DevOps
   - release suggestion and release batch policy
   - release branch, release notes, tag
   - push to GitHub/Gitee, PR/MR
   - environments: test, staging, production
   - server resources: test_dev, production, cloud metadata, expiration, renewal owner
   - deployment, backup, online smoke tests, monitoring window, rollback

7. 反馈闭环和长期治理
   - Runtime Signals -> Bug Candidate -> Repair Attempt -> Improvement Record
   - Agent Memory: Record Gate, Extraction, Staging Dedup, Async Write, Retrieval, Automatic Compact, Reflection
   - Unified Logs: run, agent, model, git, quality, release, deployment, memory, audit, error
   - Documentation Governance, Config Schema, Contracts, Failure Recovery
   - gpt-image-2 Visuals: diagram spec, prompt, image, explanation

视觉要求：
- 生成清晰的工程技术流程图，不要营销风格，不要卡通，不要 3D，不要抽象插画。
- 使用白底、深灰文字、蓝/绿/橙/紫作为泳道或模块区分色。
- 标签要短、粗、清晰；不要大量小字。
- 用实线箭头表示主流程，用虚线箭头表示反馈闭环和治理回路。
- 画面中必须能看出“需求 -> Issue Graph -> Subagent 并发执行 -> 质量门禁 -> Git 合入 -> 发布部署 -> Memory/Logs/自我修复反馈”的主流程。
- 右下角放一个小型图例：solid arrow = main flow, dashed arrow = feedback.
- 不要出现任何 API Key、token、私网 IP、真实账号或密码。


当前 docs 目录的文档结构摘要：
## README.md
# Moyuan Code 文档
## 推荐阅读顺序
## 核心设计文档
## 主线文档
## 策略决策树
## 契约文档
## 专题设计文档
## 基础规范
## 辅助资产
## 设计原则
## 进入实现前

## reference-architecture.md
# 参考架构
## 1. 总体架构
## 2. 模块职责
### CLI / API / Web Console
### Identity & Access Control
### Orchestrator
### Subagent Manager
### Agent Runtime
### Project Workspace Manager
### Self Repair Engine
### Quality Engine
### Skills Engine
### Memory Engine
### Adapter Layer
## 3. 执行状态机
### 状态说明
## 4. 上下文装配链路
## 5. 安全与权限边界
## 6. 数据存储建议

## lifecycle-roadmap.md
# 总体规划与生命周期路线图
## 1. 产品定位
## 2. 核心能力
## 2.1 主线映射
## 2.2 策略层映射
## 3. 关键抽象
## 4. 端到端流程
## 5. 项目生命周期
### DISCOVERY
### PLANNING
### DESIGN
### IMPLEMENTATION
### QUALITY_CHECK
### VERIFICATION
### REVIEW
### RELEASE
### OPERATION
### RETROSPECTIVE
## 6. CLI 路线
### MVP 命令
### Beta 命令
### Production 命令
## 7. 落地阶段
### Phase 0：规划与规格
### Phase 1：本地 CLI MVP
### Phase 2：多模型与 Skills
### Phase 3：Memory 强化
### Phase 3.5：运行反馈与自我修复
### Phase 4：团队协作与审计
### Phase 5：Web Console 与企业化
## 8. 技术选型建议
## 9. 文档迭代计划

## project-workspace-spec.md
# 项目工作空间规范
## 1. 目标
## 2. 目录结构
## 3. Schema 索引
## 4. project.yaml 最小结构
## 5. Run 记录最小结构
## 6. 配置归属原则

## issue-orchestration.md
# Issues 编排与并发调度
## 1. 目标
## 2. 端到端流程
## 3. 需求完善与澄清
## 4. Issue Graph
## 5. 并发决策
## 6. 队列与等待
## 7. 前端 Claude / 后端 Codex
## 8. Subagent 调度
## 9. 状态机
## 10. 合入门禁
## 11. 配置入口
## 12. Workspace 产物
## 13. 发布衔接
## 14. 验收标准

## agent-roles-overview.md
# Agent 角色与团队概览
## 1. Agent 与 Subagent
## 2. 角色目录
## 3. 默认 Team
## 4. 默认 Runtime 分工
## 5. 输出契约
## 6. Memory Scope
## 7. 变更规则

## subagents-skills-system.md
# Subagent 与 Skills 系统方案
## 1. 目标
## 2. 核心概念
### Agent Role
### Subagent
### Skill
## 3. Subagent 类型
## 4. Subagent 创建流程
## 5. 委派决策
## 6. Subagent 生命周期
## 7. 父子关系
## 8. 并发控制
## 9. Skill Registry
## 10. Skill 推荐流程
## 11. Skill 绑定规则
## 12. Skill 效果反馈
## 13. 输出收敛
## 14. 与 Memory 的关系
## 15. 安全边界
## 16. 配置位置
## 17. 验收标准

## agent-memory-system.md
# Agent Memory 系统方案
## 1. 目标
## 2. 六环节流水线
### 环节一：Record Gate
### 环节二：Extraction & Classification
### 环节三：Staging Dedup & Merge
### 环节四：Async Commit
### 环节五：Layered Storage
#### 结构化关系库
#### 向量库
#### 关系图
## 3. 记忆类型
## 4. 冷热分层
## 5. 自动 Compact 与整理
### Compact 目标
### Compact 类型
### 自动触发条件
### Compact 流程
### Compact 输出结构
### 安全规则
## 6. Record / Retrieve Prompt 策略
### System Prompt
### Runtime Prompt
## 7. 触发场景
### Record 触发
### Retrieve 触发
## 8. 运行时状态层
## 9. 配置
## 10. Workspace 结构
## 11. 与项目阅读理解联动
## 12. 与多 Agent 协作联动
## 13. 落地范围

## model-tool-adapters.md
# 模型与工具适配规划
## 1. 适配层目标
## 2. Provider Registry
## 3. Adapter 能力声明
## 4. Claude Code Adapter
## 5. Codex Adapter
## 6. 国产大模型 API Adapter
### 统一接口
### 能力差异处理
## 7. Image Adapter
## 8. Tool Adapter
### Shell Adapter
### Git Adapter
### Test Adapter
### MCP Adapter
## 9. 路由策略
## 10. 错误分类
## 11. 外部能力基线

## configuration-guide.md
# 配置方案
## 1. 配置目标
## 2. 配置分层
## 3. 配置索引
## 4. 最小开发闭环
## 5. 投产闭环
## 6. 敏感信息规则
## 7. 配置片段
## 8. 校验清单

## configuration-schema-spec.md
# 配置 Schema 规则
## 1. 规则定义
## 2. project.yaml
## 3. repository.yaml
## 4. policies/access.yaml
## 5. models/providers.yaml
## 6. models/routing.yaml
## 7. visuals/architecture-visuals.yaml
## 8. runtimes/agent-runtimes.yaml
## 9. agents/roles.yaml
## 10. agents/teams.yaml
## 11. agents/subagents.yaml
## 12. skills/registry.yaml、skills/enabled.yaml、skills/bindings.yaml
## 13. policies/permissions.yaml
## 14. policies/secrets.yaml
## 15. policies/orchestration.yaml
## 16. policies/code-quality.yaml
## 17. policies/comprehension.yaml
## 18. policies/memory.yaml
## 19. policies/logging.yaml
## 20. policies/server-resources.yaml
## 21. policies/environments.yaml
## 22. policies/release.yaml
## 23. policies/budget.yaml
## 24. policies/engineering.yaml
## 25. 配置校验顺序
## 26. MVP 最小配置
## 27. 进入实现前必须补的机器校验

## repository-onboarding-git-management.md
# 仓库接入与 Git Adapter
## 1. 接入目标
## 2. 接入流程
## 3. Provider 能力
## 4. Git 触发点
## 5. 工作空间产物
## 6. 用户改动保护
## 7. 验收标准

## code-lifecycle-quality-gates.md
# 代码生命周期质量门禁
## 1. 目标
## 2. 生命周期位置
## 3. 必须执行的门禁
### 可运行性门禁
### 测试缺口门禁
### 覆盖率门禁
### 重复代码门禁
### 复杂度门禁
### 架构边界门禁
### 依赖和安全门禁
## 4. 审核 Agent
## 5. 质量门禁配置
## 6. 执行流程
## 7. Review 输出契约
## 8. Run 记录
## 9. 落地范围

## engineering-process-standards.md
# 工程流程规范
## 1. 目标
## 2. Commit 规范
### 2.1 提交格式
### 2.2 必填信息
### 2.3 禁止事项
### 2.4 自动 commit 条件
## 3. Issue 规范
### 3.1 Issue 最小字段
### 3.2 Issue 命名
### 3.3 Issue 粒度
### 3.4 Issue 状态要求
## 4. 功能回退后的 Fix 规范
### 4.1 回退触发条件
### 4.2 回退后修复流程
### 4.3 Fix Issue 必填字段
### 4.4 修复验收
## 5. 发版要求
### 5.1 发版前置条件
### 5.2 发版批次
### 5.3 Release Note 必填
### 5.4 禁止发版
## 6. 测试覆盖率要求
### 6.1 默认阈值
### 6.2 覆盖率策略
### 6.3 覆盖率报告
## 7. 配置入口
## 8. 验收标准

## mainlines/project-comprehension.md
# 项目接入与阅读理解主线
## 1. 目标
## 2. 输入与输出
## 3. 端到端流程
## 4. 决策点
## 5. 配置入口
## 6. Workspace 产物
## 7. 日志与审计
## 8. 验收标准

## mainlines/requirement-planning.md
# 需求规划与 Issue 编排主线
## 1. 目标
## 2. 输入与输出
## 3. 端到端流程
## 4. 主线判定策略
## 5. 决策点
## 6. 阻断条件
## 7. 配置入口
## 8. Workspace 产物
## 9. 日志与审计
## 10. 验收标准

## mainlines/code-development.md
# 代码开发主线
## 1. 目标
## 2. 输入与输出
## 3. 端到端流程
## 4. 关键决策点
## 5. 启动条件
## 6. 质量要求
## 7. 配置入口
## 8. Workspace 产物
## 9. 日志与审计
## 10. 验收标准

## mainlines/code-management.md
# 代码管理主线
## 1. 目标
## 2. 输入与输出
## 3. 端到端流程
## 4. 决策点
## 5. 用户改动保护
## 6. 配置入口
## 7. Workspace 产物
## 8. 日志与审计
## 9. 验收标准

## mainlines/runtime-feedback-self-repair.md
# 运行反馈与自我修复主线
## 1. 目标
## 2. 边界
## 3. 输入信号
## 4. 端到端流程
## 5. Bug 判断标准
## 6. 自动修复模式
## 7. 自我增强机制
## 8. 产物
## 9. 关联策略
## 10. 阻断条件
## 11. 验收标准

## mainlines/server-resource-management.md
# 服务器资源管理主线
## 1. 目标
## 2. 输入与输出
## 3. 资源分类
## 4. 端到端流程
## 5. 决策点
## 6. 配置入口
## 7. Workspace 产物
## 8. 日志与审计
## 9. 验收标准

## mainlines/devops-release-deployment.md
# DevOps 发布投产主线
## 1. 目标
## 2. 输入与输出
## 3. 端到端流程
## 4. 发布批次建议
## 5. 决策点
## 6. 配置入口
## 7. Workspace 产物
## 8. 日志与审计
## 9. 验收标准

输出要求：
- 只生成一张完整架构流程图。
- 图片中不要出现说明性段落，保留必要短标签即可。
- 所有文字使用中文，英文技术名可以保留，例如 Claude CLI、Codex CLI、Issue Graph、Memory、Logs。
- 图需要让工程师一眼看懂当前 Moyuan Code 项目的端到端执行链路、并发编排、质量控制、发布投产和长期反馈闭环。
