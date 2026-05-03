# 参考架构

## 1. 总体架构

```text
┌─────────────────────────────────────────────────────────────┐
│ CLI / API / Web Console                                      │
├─────────────────────────────────────────────────────────────┤
│ Identity & Access Control                                    │
│ - users / organizations / sessions / API tokens              │
│ - auth context / approval / audit                            │
├─────────────────────────────────────────────────────────────┤
│ Orchestrator                                                 │
│ - task planning                                              │
│ - agent scheduling                                           │
│ - context assembly                                           │
│ - policy enforcement                                         │
│ - run state machine                                          │
├───────────────────────┬─────────────────────────────────────┤
│ Agent Runtime          │ Project Workspace Manager           │
│ - role execution       │ - config                            │
│ - handoff              │ - memory                            │
│ - tool calls           │ - skills                            │
│ - output contracts     │ - lifecycle records                 │
│                        │ - repository state                   │
│                        │ - branch policy                      │
│                        │ - project comprehension              │
│                        │ - server resource inventory           │
├───────────────────────┼─────────────────────────────────────┤
│ Skills Engine          │ Memory Engine                        │
│ - find-skills adapter  │ - short-term context                 │
│ - recommendation       │ - long-term project memory           │
│ - role binding         │ - vector/graph/structured store      │
│                        │ - record gate / staging              │
├───────────────────────┴─────────────────────────────────────┤
│ Adapter Layer                                                │
│ - Claude Code Adapter                                        │
│ - Codex Adapter                                              │
│ - Domestic LLM Adapters                                      │
│ - Shell/Git/Test/MCP Adapters                                │
├─────────────────────────────────────────────────────────────┤
│ External Systems                                             │
│ - code repositories                                          │
│ - GitHub / Gitee / GitLab                                    │
│ - model providers                                            │
│ - issue trackers                                             │
│ - CI/CD                                                      │
│ - cloud providers                                            │
│ - observability                                              │
└─────────────────────────────────────────────────────────────┘
```

## 2. 模块职责

### CLI / API / Web Console

首阶段优先实现 CLI，后续暴露 API 和 Web Console。

CLI 需要覆盖：

- 本地 owner 初始化。
- 当前身份查看。
- 审批确认或拒绝。
- 项目初始化。
- 本地路径和远程 Git 仓库接入。
- Git 状态和任务分支查看。
- 任务创建。
- 任务执行。
- Agent 配置。
- 模型配置。
- memory 管理。
- skills 推荐。
- 生命周期报告。
- 服务器资源添加、巡检、续费提醒和退役。

API 预留：

- 用户、组织、会话、API Token 和 service account 管理。
- 面向 Web Console。
- 面向 CI/CD。
- 面向企业内部平台。

### Identity & Access Control

Identity & Access Control 负责 Moyuan 平台用户、组织、会话、API Token、服务账号、角色、审批和审计。

职责：

- 初始化本地 owner identity。
- 解析 CLI、API 和 Web 操作的身份凭证。
- 生成 `auth_context`。
- 校验会话、Token、用户状态和成员关系。
- 调用鉴权策略，输出 `ALLOW`、`DENY` 或 `REQUIRE_APPROVAL`。
- 管理审批记录。
- 写入身份、权限和审批审计事件。

### Orchestrator

Orchestrator 是核心编排层，负责把用户目标转成可追踪、可恢复、可审计的执行流。

职责：

- 解析用户请求。
- 建立或接收 `auth_context`。
- 选择工作项目和 workspace。
- 装配上下文。
- 选择 Agent 和模型。
- 拆解任务。
- 调度串行/并行执行。
- 处理失败、重试和回滚建议。
- 执行权限策略。
- 执行鉴权与审批策略。
- 写入 Run 记录。

### Agent Runtime

Agent Runtime 负责执行一个具体 Agent 的任务。

一个 Agent Runtime 输入：

- role 定义。
- task 说明。
- 项目上下文。
- 可用 memory。
- 可用 skills。
- 可用工具权限。
- 模型策略。
- 输出契约。

输出：

- 结构化结果。
- 文件变更摘要。
- 测试结果。
- 风险列表。
- 下一步建议。
- 需要写入 memory 的候选内容。

### Project Workspace Manager

管理每个项目的 `.moyuan/` 目录。

职责：

- 读取和校验项目配置。
- 管理 project profile。
- 管理 project comprehension 产物。
- 管理任务和运行记录。
- 管理项目 memory。
- 管理 skills 启用状态。
- 管理权限策略。
- 管理模型预算。
- 管理仓库来源、remote、默认分支和任务分支状态。
- 管理任务分支策略和 Git 审计记录。

### Skills Engine

职责：

- 调用或封装 `find-skills`。
- 根据项目画像推荐 skills。
- 根据任务类型推荐 skills。
- 根据 Agent role 绑定 skills。
- 评估 skill 使用效果。
- 支持本地 skill、团队 skill、远程 skill marketplace。

### Memory Engine

职责：

- 在任务开始前检索相关 memory。
- 在任务结束后抽取可沉淀 memory。
- 使用 Record Gate 判断信息是否值得长期保存。
- 使用轻量模型抽取结构化 memory candidates。
- 使用暂存区进行去重、合并和冲突标记。
- 异步写入关系库、向量库和关系图。
- 区分事实、决策、偏好和经验。
- 支持过期、置信度和来源追踪。
- 防止把临时错误、敏感信息或过时方案写入长期记忆。

### Adapter Layer

Adapter Layer 让上层不依赖具体厂商或工具。

每个 adapter 需要声明：

- provider 名称。
- 支持的调用模式：chat、responses、agentic coding、CLI、SDK、OpenAI-compatible。
- 支持的工具能力。
- 上下文限制。
- 文件读写能力。
- 流式输出能力。
- 成本估算能力。
- 错误类型。
- 鉴权方式。

## 3. 执行状态机

```text
CREATED
  -> PLANNING
  -> WAITING_APPROVAL
  -> RUNNING
  -> QUALITY_CHECKING
  -> VERIFYING
  -> REVIEWING
  -> COMPLETED
  -> ARCHIVED

任意阶段可进入：
  -> FAILED
  -> CANCELLED
  -> NEEDS_USER_INPUT
  -> NEEDS_REWORK
```

### 状态说明

- `CREATED`：任务已创建。
- `PLANNING`：正在拆解方案。
- `WAITING_APPROVAL`：等待用户确认方案、权限或预算。
- `RUNNING`：Agent 正在执行。
- `QUALITY_CHECKING`：正在检查重复代码、复杂度、架构边界、测试缺口和依赖安全。
- `VERIFYING`：正在运行测试、lint、构建或其他验证。
- `REVIEWING`：正在审查输出。
- `COMPLETED`：任务完成。
- `ARCHIVED`：任务进入历史归档。
- `FAILED`：执行失败并记录原因。
- `CANCELLED`：用户取消。
- `NEEDS_USER_INPUT`：缺少必要信息。
- `NEEDS_REWORK`：质量门禁或审核未通过，需要回到实现阶段返工。

## 4. 上下文装配链路

```text
User Request
  -> Auth Context
  -> Project Config
  -> Project Profile
  -> Project Comprehension
  -> Module Map
  -> Relevant Files
  -> Related Tasks
  -> Memory Search
  -> Skill Recommendations
  -> Role System Prompt
  -> Tool Policy
  -> Model Policy
  -> Adapter Request
```

上下文装配必须遵守：

- 优先提供当前任务相关内容。
- 长期 memory 只注入命中的片段。
- 大文件用摘要和引用，不直接全量塞入。
- 敏感文件默认不注入。
- 每次注入需要记录来源，方便审计。

## 5. 安全与权限边界

必须实现的策略：

- 身份、会话、API Token 和成员关系校验。
- 文件系统范围限制。
- 命令 allowlist/denylist。
- 网络访问策略。
- 密钥读取策略。
- Git 操作策略。
- 远程仓库认证策略。
- 自动分支创建、push 和 PR/MR 策略。
- 发布操作确认。
- 生产环境操作确认。
- 跨项目 memory 隔离。

高风险操作示例：

- 删除文件或目录。
- 修改锁文件和迁移脚本。
- 执行部署命令。
- 访问 `.env`、密钥、凭证。
- 修改 CI/CD 配置。
- 执行数据库写操作。
- 开启外网访问。

## 6. 数据存储建议

MVP：

- 配置：YAML。
- 任务状态：JSONL 或 SQLite。
- 核心日志：按 run、agent、model、git、quality、release、memory、audit、error 分流写入 JSONL。
- memory：SQLite + 本地向量索引。
- 大文本产物：Markdown。

生产版：

- 元数据：PostgreSQL。
- 向量：pgvector、Milvus、Qdrant 或 Elasticsearch 向量能力。
- 对象存储：S3/MinIO。
- 队列：Redis Stream、NATS 或 Kafka。
- 核心日志：OpenTelemetry、Loki、Elasticsearch 或 ClickHouse。
- 审计日志：不可变追加日志，长期保留。

日志配置见 [完整配置方案](./configuration-guide.md) 的 `policies/logging.yaml`。生产应用自身日志、指标和健康检查见 `policies/environments.yaml`。
