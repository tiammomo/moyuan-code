
你是资深软件架构图设计师和技术信息图设计师。请根据下面的设计规格，生成一张适合技术评审会议展示的技术可视化流程图。


图名：Moyuan Code Multi-Agent SDLC Technical Map
目标：用中文说明、英文专有名词原样保留、图标和节点辅助表达的方式，展示当前项目从 User Request、Repository Onboarding、Project Comprehension、Issue Graph、Subagent 并发执行、Quality Gate、Git Merge、Release/Deployment 到 Memory/Logs 反馈的完整技术流程。
受众：技术负责人、架构师、后端/前端/测试/运维 Agent 配置人员、后续实现工程师。

画面布局：
请生成一张横向技术可视化地图，不要做纯文本表格。采用“主流程从左到右 + 底部反馈环”的结构。每个主模块用大号中文标题、英文专有名词、小型流程节点、图标和短标签表达。技术密度要高于普通信息图，但比纯文本架构图更好看。

主流程 7 个大模块：

1. 入口与权限
   - 图标建议：用户头像、终端窗口、API 插头、盾牌、审批勾选
   - 必须出现的术语：Platform User、CLI/API、Auth Context、RBAC、Approval、Audit、Secret Ref

2. 仓库接入与项目理解
   - 图标建议：代码仓库、Git 分支、云端仓库、放大镜、项目地图
   - 必须出现的术语：Local Path、GitHub/Gitee、Generic Git、Git Adapter、.moyuan Workspace、Project Comprehension、Project Profile、Module Map

3. 需求规划与 Issue Graph
   - 图标建议：便签需求、问号气泡、DAG 节点图、队列看板
   - 必须出现的术语：Requirement Refiner、Clarification Gate、Issue Planner、Dependency Planner、Scheduler、Issue Graph、ready_queue、blocked_queue

4. Multi-Agent 执行
   - 图标建议：多个 Agent 节点、工具箱、技能拼图、Claude/Codex 运行器、模型云
   - 必须出现的术语：Agent Roles、Subagent Manager、Skills Registry、find-skills、Skill Binding、Runtime Adapter、Claude CLI、Codex CLI、Model Routing、GPT/Claude/GLM/MiniMax

5. 质量门禁与合入
   - 图标建议：代码文件、测试烧杯、仪表盘、锁门、合并箭头
   - 必须出现的术语：Issue Branch、Worktree、Build/Lint/Test、Coverage、Duplication、Complexity、Security Scan、Quality Gate、Review、Epic Integration Branch

6. 发布与服务器
   - 图标建议：版本标签、GitHub/Gitee 云、服务器机柜、火箭/部署箭头、监控波形、回滚按钮
   - 必须出现的术语：Release Suggestion、Release Branch、Tag、PR/MR、GitHub/Gitee Push、test_dev、production、Deployment、Smoke Test、Monitor、Rollback

7. 反馈与长期治理
   - 图标建议：环形箭头、大脑/记忆库、日志卷轴、Bug 修复扳手、文档书本
   - 必须出现的术语：Runtime Signals、Bug Candidate、Repair Attempt、Improvement Record、Agent Memory、Memory Compact、Unified Logs、Contracts、Config Schema、Failure Recovery

底部反馈环：
从 Runtime Signals / Logs / Quality Findings / User Feedback 流向 Bug Candidate -> Repair Attempt -> Improvement Record -> Memory Compact -> Policy/Skills/Docs Update，再虚线回到 Requirement Planning 和 Multi-Agent Execution。

视觉要求：
- 中文用于模块标题和说明性短语；英文专有名词必须原样使用，不能翻译成中文。必须保留：Auth Context、Git Adapter、Project Comprehension、Issue Graph、Subagent、Skills Registry、Runtime Adapter、Claude CLI、Codex CLI、Model Routing、Quality Gate、Review、Release Branch、Deployment、Agent Memory、Memory Compact、Unified Logs。
- 必须使用合适的图标、简化设备图、节点图、箭头和小型可视化元素，不要纯文本框堆叠。
- 生成清晰的工程技术流程图，不要营销风格，不要卡通人物，不要 3D，不要抽象插画。
- 白底或极浅灰底，深灰文字，蓝/绿/橙/紫/青作为模块区分色。
- 每个模块允许 6-9 个短标签或小节点，以提高技术密度，但不能变成说明段落。
- 标题必须大、粗、清晰；小字不能密集。
- 用实线箭头表示主流程，用虚线箭头表示反馈闭环和治理回路。
- 画面中必须能看出“User Request -> Issue Graph -> Subagent Parallel Runs -> Quality Gate -> Git Merge -> Release/Deployment -> Memory/Logs/Self Repair Feedback”的主流程。
- 右下角放一个小型图例：实线 = Main Flow，虚线 = Feedback Loop。
- 整体要比普通架构图更好看，但仍然是技术向，不要变成宣传海报。
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
- 图片中不要出现说明性段落，必须用图标、节点、箭头和短标签表达。
- 中文用于模块标题和说明性短语；英文专有名词必须原样保留，不要翻译成中文。
- 不要把规格里的长句原样放进图里。
- 图需要让工程师一眼看懂当前 Moyuan Code 项目的端到端执行链路、并发编排、质量控制、发布投产和长期反馈闭环。
