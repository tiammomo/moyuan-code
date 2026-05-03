
你是资深软件架构图设计师。请根据下面的设计规格，生成一张适合技术评审会议展示的架构流程设计图。


图名：Moyuan Code 多 Agent 代码开发框架总体结构设计图
目标：展示当前项目规划中的核心模块、执行流程、外部系统和反馈闭环。
受众：技术负责人、架构师、后端/前端/测试/运维 Agent 配置人员。

画面布局：
1. 左侧：用户入口和项目接入
   - User
   - CLI / API / Web Console
   - 本地仓库 / GitHub / Gitee
   - 项目理解 Project Comprehension

2. 中央顶部：Orchestrator 编排核心
   - Requirement Refiner
   - Clarification Gate
   - Issue Planner
   - Dependency Planner
   - Scheduler
   - Issue Graph / Ready Queue

3. 中央中部：Agent Runtime 和执行后端
   - Native Agent Runtime: Claude CLI, Codex CLI
   - Model API Providers: GPT, Claude, GLM, MiniMax, DeepSeek, DashScope, Third-party API
   - Agent Roles: planner, architect, backend, frontend, tester, reviewer, quality_guard, release_manager
   - Skills Engine / find-skills

4. 中央底部：项目工作空间 .moyuan
   - project.yaml / repository.yaml
   - agents / models / runtimes / visuals
   - memory / logs / resources / lifecycle
   - policies: permissions, quality, orchestration, release, environments

5. 右侧：代码生命周期流水线
   - issue worktree / task branch
   - code generation and edits
   - tests / lint / build
   - quality gates
   - review
   - merge into epic integration branch
   - release branch
   - publish to GitHub/Gitee
   - deploy to test_dev / production resource groups
   - online smoke / monitor / rollback

6. 底部反馈闭环：
   - Agent Memory: record gate, extraction, staging dedup, compact, retrieval
   - Unified Logs: run, agent, model, git, quality, release, memory, audit, error
   - Server Resources: cloud metadata, expiration, renewal, checks, maintenance
   - gpt-image-2 Visuals: architecture diagrams, flow explanations

视觉要求：
- 生成清晰的工程架构图，不要营销风格，不要卡通，不要 3D。
- 使用分层架构图 + 流程箭头，节点少而清楚。
- 中文标签要大、短、清晰，避免小字密集。
- 使用冷静专业配色：白底、深灰文字、蓝/绿/橙作为模块区分色。
- 画面中要能看出“需求 -> Issue Graph -> 多 Agent 并发执行 -> 质量门禁 -> 发布部署 -> Memory/Logs 反馈”的主流程。
- 不要出现任何 API Key、token、私网 IP、真实账号或密码。


当前 docs 目录的文档结构摘要：
## README.md
# Moyuan Code 文档
## 文档索引
## 设计原则

## reference-architecture.md
# 参考架构
## 1. 总体架构
## 2. 模块职责
### CLI / API / Web Console
### Orchestrator
### Agent Runtime
### Project Workspace Manager
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
### Phase 4：团队协作与审计
### Phase 5：Web Console 与企业化
## 8. 技术选型建议
## 9. 近期任务拆分

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
## 2. 核心抽象
### Epic
### Issue
### Issue Graph
## 3. Issue 类型
## 4. 自动拆分流程
## 4.1 需求完善与意图澄清
## 5. 依赖类型
## 6. 并发决策
## 7. 调度状态机
## 8. 执行模型
### 串行依赖
### 并行执行
### 集成分支
## 8.1 合入门禁
## 9. Worktree 策略
## 10. 冲突处理
## 11. 配置
## 12. 版本分支、投产与维护流水线
### 版本批次建议
### 投产流水线
### 服务器资源与环境配置
### 投产策略
## 13. Workspace 产物
## 14. 验收标准

## agent-skills-memory.md
# Agent、Skills 与编排
## 1. Agent 角色体系
## 2. 核心角色
## 3. Team 配置
## 4. Agent 输出契约
## 5. Skills 体系
## 6. Role 与 Skill 绑定
## 7. Memory Scope

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
# 完整配置方案
## 1. 目标
## 2. 配置分层
## 3. 最小必填配置
## 4. project.yaml
## 5. repository.yaml
## 6. models/providers.yaml
## 7. models/routing.yaml
## 8. visuals/architecture-visuals.yaml
## 9. runtimes/agent-runtimes.yaml
## 10. agents/roles.yaml
## 11. agents/teams.yaml
## 12. policies/orchestration.yaml
## 13. policies/permissions.yaml
## 14. policies/secrets.yaml
## 15. policies/code-quality.yaml
## 16. policies/comprehension.yaml
## 17. policies/memory.yaml
## 18. policies/release.yaml
## 19. policies/logging.yaml
## 20. policies/server-resources.yaml
## 21. policies/environments.yaml
## 22. policies/budget.yaml
## 23. skills/enabled.yaml
## 24. 配置校验清单

## repository-onboarding-git-management.md
# 仓库接入、Git 与项目理解
## 1. 目标
## 2. 接入流程
### 本地路径接入
### 远程仓库接入
## 3. 远程 Provider
## 4. 项目阅读理解
### Full Comprehension
### Incremental Comprehension
## 5. 阅读理解触发点
## 6. 阅读理解产物
## 7. Git 分支策略
## 8. 用户改动保护
## 9. PR/MR 策略
## 10. Memory 联动
## 11. 审计记录

## code-lifecycle-quality-gates.md
# 代码生命周期质量门禁
## 1. 目标
## 2. 生命周期位置
## 3. 必须执行的门禁
### 可运行性门禁
### 测试缺口门禁
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

输出要求：
- 只生成一张完整架构流程图。
- 图片中不要出现说明性段落，保留必要短标签即可。
- 所有文字使用中文，英文技术名可以保留，例如 Claude CLI、Codex CLI、Issue Graph、Memory、Logs。
- 架构图需要让人一眼看懂当前 Moyuan Code 项目总体设计。
