
你是资深软件架构图设计师和技术信息图设计师。请根据下面的设计规格，生成一张适合技术评审会议展示的横版 2K 技术调用逻辑图。


图名：Moyuan Code Multi-Agent SDLC 调用逻辑
目标：生成一张横版 2K 技术信息流图，用编号层级、主流程箭头、辅助流程虚线、数据/工作空间沉淀层和底部治理层，精炼展示 Moyuan Code 的核心调用逻辑。参考用户给出的优秀横版流程图风格：顶部强标题、分层编号模块、深蓝标题条、浅色卡片、数据库圆柱、右上图例、底部调度/控制层。不要照搬参考图的业务内容，只参考版式组织方式。
受众：技术负责人、架构师、后端/前端/测试/运维 Agent 配置人员、后续实现工程师。

画面布局：
请生成一张横版 2K 宽屏技术流程图，不是竖版海报。整体结构必须像工程调用逻辑图：

顶部：大标题居中
- 标题：Moyuan Code Multi-Agent SDLC 调用逻辑
- 右上角图例：实线 = Main Flow，虚线 = Control / Feedback，圆柱 = Workspace / Data Store，圆角框 = Processing Module

第一行：从左到右 7 个主流程层，每层一个编号卡片，卡片顶部使用深蓝标题条。

1. 输入与权限层
   - 用户入口 / CLI/API
   - Auth Context / RBAC
   - 审批 / 审计
   - 密钥引用

2. 仓库接入层
   - 本地仓库 / GitHub / Gitee
   - Git Adapter
   - clone / fetch / branch
   - 初始化 .moyuan

3. 项目理解层
   - Project Comprehension
   - 项目画像
   - 模块地图
   - 命令 / 风险文件

4. 需求规划层
   - Requirement Refiner
   - Clarification Gate
   - Issue Planner
   - Issue Graph
   - 调度器

5. Multi-Agent 执行层
   - Subagent Manager
   - Skills Registry / find-skills
   - Claude CLI / Codex CLI
   - Model Routing

6. 质量合入层
   - Issue Worktree
   - Build / Lint / Test
   - Quality Gate
   - Review
   - 集成分支

7. 发布投产层
   - Release Branch / Tag
   - 推送 GitHub/Gitee
   - Deployment
   - 冒烟 / 监控
   - 回滚

第二行：Workspace / Data Store 沉淀层，用一条虚线边框包起来，放 6 个数据库圆柱或文件库图标，从左到右：
- repository.yaml / project.yaml
- comprehension/
- lifecycle/issue-graphs/
- agents/subagents/
- memory/
- logs/

中部横向总结条：
- “规划层决定做什么 = 第 1-4 层”
- “执行层决定怎么做 = 第 5-7 层”
- “Workspace 记录配置、状态、证据和审计”

右侧竖向补充层：
8. Server Resources
   - test_dev / production
   - 云元数据 / 到期时间
   - 健康检查 / 负责人
9. Provider & Runtime
   - GPT / Claude / GLM / MiniMax
   - 第三方 API 策略
   - Runtime Adapter

底部：治理与反馈层，横向时间线，使用浅棕或深蓝标题条：
10. Runtime Signals
    - 错误 / 测试失败 / 冒烟失败
11. Self Repair
    - Bug Candidate -> Repair Attempt
12. Agent Memory
    - Record Gate / Retrieve / Memory Compact
13. Documentation & Contracts
    - Config Schema / Failure Recovery / 审计日志

箭头规则：
- 主流程用粗实线从 1 -> 7。
- 数据沉淀用向下虚线连到 Workspace / Data Store。
- 反馈层用虚线从 10-13 回到 4 需求规划层和 5 Multi-Agent 执行层。

视觉要求：
- 版式参考用户提供的横版调用逻辑图：编号模块、深蓝标题条、浅色内容区、箭头清晰、数据存储圆柱、底部调度/治理层、右上图例。
- 不要出现人物肖像，不要出现参考图里的股票、交易、时间、业务名或任何无关内容。
- 普通动作、说明和中文业务语义必须用中文；英文专有名词必须保留：Auth Context、RBAC、Git Adapter、Project Comprehension、Issue Graph、Subagent、Skills Registry、Model Routing、Quality Gate、Review、Release Branch、Deployment、Agent Memory、Memory Compact、Runtime Adapter。
- 每个卡片只保留 3-5 个核心技术点，不要大段文字。
- 横版宽屏布局，信息密度适中但不拥挤。
- 使用白底、浅灰卡片、深色文字，模块标题条以深蓝为主，少量绿、橙、青区分分区。
- 图标风格统一，使用线性或半扁平图标：仓库、DAG、Agent、拼图、测试瓶、服务器、监控、数据库、扳手、日志、盾牌。
- 文字必须清晰可读，不要密集小字。
- 整体是技术调用逻辑图，不是宣传海报，不要夸张视觉效果。
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
- 只生成一张完整横版技术调用逻辑图。
- 图片中不要出现说明性长段落，必须用编号模块、箭头、数据存储层、图标和短要点表达。
- 普通说明尽量使用中文；英文专有名词必须原样保留，不要翻译成中文。
- 不要把规格里的长句原样放进图里。
- 每个主题只放核心技术点，详细设计保留在配套讲解文档中。
- 图需要让工程师一眼看懂当前 Moyuan Code 项目的端到端执行链路、并发编排、质量控制、发布投产和长期反馈闭环。
