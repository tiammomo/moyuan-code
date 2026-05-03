# Agent Memory 系统方案

## 1. 目标

当前 memory 规划已经覆盖项目事实、决策、偏好、经验和检索注入。升级后的目标是把 memory 从“文件化记录”提升为“可判断、可抽取、可去重、可检索、可维护、可审计”的长期系统。

核心原则：

- 先判断是否值得记，再决定怎么记。
- 判断“要不要记”使用强模型，抽取“怎么记”使用轻模型。
- 所有候选先进入暂存区，先去重和合并，再异步写入长期存储。
- 长期存储采用结构化库、向量库和关系图组合。
- Memory 需要冷/热分层、自动 compact 压缩、定期总结、过期和冲突处理。
- Prompt 需要明确 record/retrieve 的触发场景，而不是让 Agent 随机决定。

## 2. 六环节流水线

```text
User / Agent Event
  -> Record Gate
  -> Extraction & Classification
  -> Staging Dedup & Merge
  -> Async Commit
  -> Layered Storage
  -> Compact & Reflection
  -> Maintenance
```

### 环节一：Record Gate

Record Gate 负责判断一条信息“要不要进入记忆系统”。

建议使用强模型或规则 + 强模型组合。强模型只做 Yes/No 或评分，不做抽取。

判断维度：

- 持久性价值：未来任务是否长期有用。
- 结构化程度：是否能抽成实体、关系、约束、偏好、决策。
- 个性化价值：是否反映用户、项目或团队偏好。
- 可靠性：是否来自用户明确陈述、代码事实、测试结果或已确认结论。
- 风险等级：是否涉及安全、发布、架构、数据、权限等高影响领域。

代码开发场景下建议评分：

```yaml
record_gate:
  score_dimensions:
    persistence: 0.35
    structure: 0.20
    personalization: 0.15
    reliability: 0.20
    risk_impact: 0.10
  threshold: 3.5
```

必须进入候选记忆的场景：

- 用户明确说“记住”“以后都按这个”“这个项目约定是”。
- 项目阅读理解发现稳定事实，例如构建命令、测试命令、模块职责。
- Agent 完成重要推理并被测试、review 或用户确认。
- 用户纠正 Agent 的错误理解。
- 质量门禁发现重复出现的问题。
- 发布、回滚、安全或架构策略被确认。
- 远程分支拉取后发现项目结构、模块边界或命令发生变化。

禁止进入候选记忆的场景：

- 一次性操作，例如“帮我看一下今天日期”。
- 未确认猜测。
- 密钥、token、密码、个人敏感信息。
- 临时错误日志，且没有复盘价值。
- 已被代码删除或后续结论推翻的信息。

### 环节二：Extraction & Classification

轻模型负责“怎么记”，不判断“要不要记”。

抽取目标：

- 实体：项目、模块、文件、接口、数据库表、Agent、用户、依赖、命令。
- 关系：依赖、调用、归属、替代、冲突、影响、约束。
- 关键句：用户原话、项目约定、结论、风险。
- 时间：创建、确认、过期、关联 commit/run/task。
- 分类：fact、decision、preference、lesson、quality、comprehension、release、security。
- 标签：backend、frontend、test、performance、auth、database、deployment 等。

统一输出结构：

```yaml
candidate_id: string
source:
  type: user_message | project_comprehension | run_result | review | quality_gate | release | manual
  ref: string
memory_type: fact | decision | preference | lesson | quality | security | release | comprehension
scope: project | workspace | user | organization
confidence: 0-1
importance: low | medium | high
ttl: null | duration
entities:
  - type: module | file | command | dependency | api | table | user | agent
    name: string
relations:
  - subject: string
    predicate: string
    object: string
summary: string
evidence: string
tags: []
requires_approval: boolean
dedup_keys: []
```

### 环节三：Staging Dedup & Merge

抽取后的候选不直接进入长期存储，而是先进入暂存区。

暂存区职责：

- 短时间窗口内去重。
- 合并相同实体的补充信息。
- 区分“重复”和“更新”。
- 标记冲突。
- 批量写入长期存储。

去重策略：

- 精确键：scope + memory_type + entity + relation + normalized summary。
- 语义相似：向量相似度超过阈值。
- 图关系相同：相同 subject/predicate/object。
- 来源优先级：用户确认 > 测试结果 > review 结论 > 项目阅读理解 > Agent 推测。

合并策略：

- 相同事实重复出现：增加访问次数和置信度。
- 同一实体属性变化：记录为 update，不当作重复。
- 旧事实被新代码推翻：标记 stale。
- 冲突无法自动判断：进入 approval queue。

暂存区配置：

```yaml
staging:
  enabled: true
  max_items: 100
  time_window_seconds: 30
  semantic_similarity_threshold: 0.90
  force_flush_after_seconds: 300
```

### 环节四：Async Commit

长期写入应异步进行，不阻塞主任务。

写入流程：

```text
staging ready
  -> check cache
  -> write relational metadata
  -> write vector embedding
  -> write graph relation
  -> update memory indexes
  -> append audit event
```

设计要求：

- 主任务只等待候选生成，不等待长期写入完成。
- 写入失败需要重试。
- 每次写入保留审计记录。
- 对需要用户确认的候选不自动 commit。

### 环节五：Layered Storage

长期存储分三类。

#### 结构化关系库

保存可查询元数据：

- memory id。
- type。
- scope。
- source。
- confidence。
- status。
- timestamps。
- access count。
- ttl。
- tags。
- approval state。

MVP 可用 SQLite，生产版用 PostgreSQL。

#### 向量库

保存原文片段、摘要、证据和 embedding。

用途：

- 语义检索。
- 相似记忆去重。
- 找历史讨论和经验。
- 给 Agent 注入相关背景。

MVP 可用 SQLite + 本地向量索引，生产版可选 pgvector、Qdrant、Milvus。

#### 关系图

保存实体关系：

- 模块依赖模块。
- 接口属于模块。
- 文件实现接口。
- 决策影响模块。
- 质量问题反复出现在文件。
- 用户偏好适用于项目。

MVP 可用 JSONL 或 SQLite edge table，生产版可选 Neo4j 或 PostgreSQL graph-style 表。

## 3. 记忆类型

建议升级为以下类型：

| Type | 含义 | 示例 |
| --- | --- | --- |
| fact | 稳定项目事实 | `npm test` 是项目测试命令 |
| decision | 已确认决策 | controller 不直接访问数据库 |
| preference | 用户或团队偏好 | 文档使用中文，测试优先补核心路径 |
| lesson | 复盘经验 | 修改鉴权中间件后必须跑集成测试 |
| quality | 质量经验 | 订单模块多次出现重复 DTO 转换逻辑 |
| comprehension | 项目阅读理解结果 | user 模块负责登录和会话管理 |
| release | 发布和回滚信息 | v1.3 发布需要执行迁移脚本 |
| security | 安全约束 | 禁止在日志中打印 access token |
| runtime | 当前会话或任务状态 | 本次任务正在修改 src/auth |

## 4. 冷热分层

Memory 需要冷热分层，避免所有历史都同等检索。

热记忆：

- 最近访问。
- 当前任务相关。
- 高置信度。
- 高风险模块相关。
- 用户明确要求遵守。

温记忆：

- 项目稳定事实。
- 历史 lessons。
- 中频访问的模块信息。

冷记忆：

- 长期未访问。
- 低置信度。
- 旧版本相关。
- 已被新事实替代但仍需审计保留。

检索排序建议：

```yaml
retrieval_ranking:
  weights:
    semantic_similarity: 0.35
    recency: 0.15
    confidence: 0.20
    access_count: 0.10
    risk_impact: 0.10
    role_relevance: 0.10
```

## 5. 自动 Compact 与整理

Memory 系统必须具备自动化 compact 压缩和整理能力，避免长期运行后检索噪声变大、上下文膨胀、重复经验堆积。

### Compact 目标

- 将多条相似 memory 合并为更稳定的 summary。
- 将多次 run 的临时上下文压缩为 task summary。
- 将多次任务经验压缩为 lesson 或 project rule。
- 将过时 facts 标记为 stale，并从默认检索结果中降权。
- 将低价值、低置信度、长期未访问的 memory 转入冷层或归档层。
- 保留原始证据和审计链，不直接丢弃可追溯来源。

### Compact 类型

| 类型 | 输入 | 输出 |
| --- | --- | --- |
| session_compact | 单次会话、Agent handoff、临时上下文 | run summary |
| task_compact | task 下多个 run、diff、测试、review | task summary、lessons |
| topic_compact | 同一模块/主题的多条 facts 和 lessons | topic summary |
| project_compact | 一段时间内的项目变化 | project memory summary |
| stale_compact | 与最新项目理解冲突的 memory | stale 标记、替代关系 |
| duplicate_compact | 相似或重复 memory | merged memory |

### 自动触发条件

- 单个 run context 超过 token 或条目阈值。
- `staging.jsonl` 达到最大条数或等待时间。
- 同一 dedup key 在短时间内重复出现。
- 项目阅读理解发现 facts 与新代码冲突。
- 拉取远程分支后模块地图变化。
- 任务完成后进入 retrospective。
- 定时任务触发 daily/weekly reflection。
- 检索结果中重复项比例过高。

### Compact 流程

```text
memory events
  -> detect compact trigger
  -> group by scope/type/entity/topic
  -> retrieve source evidence
  -> summarize and merge
  -> mark stale / superseded / archived
  -> write compacted memory
  -> update vector and graph indexes
  -> append audit event
```

### Compact 输出结构

```yaml
compact_id: string
type: session_compact | task_compact | topic_compact | project_compact | stale_compact | duplicate_compact
scope: project | workspace | user | organization
source_memory_ids: []
source_run_ids: []
summary: string
preserved_facts: []
discarded_noise: []
stale_memory_ids: []
supersedes: []
confidence: 0-1
requires_approval: boolean
created_at: datetime
```

### 安全规则

- Compact 不删除原始审计记录，只改变检索权重、状态或归档位置。
- 用户明确偏好、架构决策、安全策略不能自动覆盖，只能生成候选等待确认。
- compact summary 必须保留来源 id。
- 如果两个 memory 冲突且无法判断新旧，进入 approval queue。
- 低置信度 summary 不进入热记忆。

## 6. Record / Retrieve Prompt 策略

Prompt 分两层。

### System Prompt

长期有效，告诉 Agent 通用规则：

- 什么时候要 record。
- 什么时候要 retrieve。
- 不要记录什么。
- 如何处理敏感信息。
- 如何输出 memory candidates。

### Runtime Prompt

在特定节点动态触发：

- 用户发来新消息后，问是否需要 record。
- 任务开始前，问是否需要 retrieve。
- 项目接入后，强制 record comprehension candidates。
- 拉取远程分支后，强制 retrieve 旧项目画像并 record 变化。
- Review 完成后，强制 record 被确认的质量经验。
- 自我修复完成后，强制 record 被验证的 bug signature、root cause、fix pattern 和 regression test。

Record Prompt 核心规则：

```text
你需要判断当前内容中是否存在值得长期保留的信息。
只有当信息具备长期价值、可靠来源，并可能影响未来任务时，才输出 memory candidate。
不要记录一次性指令、敏感信息、未确认猜测或与项目无关内容。
如果信息是用户纠正、项目事实、架构决策、测试命令、质量经验或偏好，优先生成候选。
```

Retrieve Prompt 核心规则：

```text
在执行任务前，判断是否需要从记忆中检索历史信息。
当任务涉及已有模块、历史决策、用户偏好、测试策略、发布流程、安全约束或用户提到“之前/上次/按原方案”时，必须检索。
如果检索为空或不相关，继续基于当前上下文执行，不要编造历史。
```

## 7. 触发场景

### Record 触发

- 用户明确要求记住。
- 用户纠正 Agent。
- 项目接入后完成 full comprehension。
- 拉取远程分支后完成 incremental comprehension。
- 质量门禁发现高价值经验。
- Review Agent 给出被接受的结论。
- 自我修复成功并通过质量门禁和 review。
- 同类 bug 重复出现并被确认。
- 测试、构建、发布命令被确认。
- 任务完成后产生可复用经验。

### Retrieve 触发

- 新任务开始。
- 任务涉及已有模块。
- 用户提到“之前”“上次”“按我们约定”。
- Agent 要修改高风险文件。
- Agent 要做架构、测试、发布、安全相关决策。
- 拉取远程分支后需要判断旧 memory 是否过时。
- Review 或 quality_guard 需要参考项目规范。

## 8. 运行时状态层

Runtime State 不等同于长期 memory，但它是 memory 系统的控制面板。

保存内容：

- 当前 task id。
- 当前 run id。
- 当前 branch。
- 当前修改文件。
- 当前参与 Agent。
- 已检索 memory id。
- 已生成 memory candidates。
- 暂存区状态。
- 质量门禁状态。
- 错误和重试信息。

用途：

- 防止重复检索。
- 防止重复记录。
- 支持 Agent handoff。
- 支持失败恢复。
- 支持审计。

## 9. 配置

`.moyuan/policies/memory.yaml`：

```yaml
schema_version: 1

memory:
  enabled: true
  record_gate:
    model_policy: memory_record_gate
    threshold: 3.5
    dimensions:
      persistence: 0.35
      structure: 0.20
      personalization: 0.15
      reliability: 0.20
      risk_impact: 0.10

  extraction:
    model_policy: memory_extraction_light
    require_structured_output: true
    classify_types:
      - fact
      - decision
      - preference
      - lesson
      - quality
      - comprehension
      - release
      - security

  staging:
    enabled: true
    max_items: 100
    time_window_seconds: 30
    semantic_similarity_threshold: 0.90
    force_flush_after_seconds: 300

  storage:
    relational: sqlite
    vector: local
    graph: sqlite_edges
    audit_log: .moyuan/memory/audit.jsonl

  retrieval:
    top_k: 8
    min_score: 0.55
    include_sources: true
    role_scoped: true

  compact:
    enabled: true
    mode: automatic
    triggers:
      max_run_context_tokens: 24000
      staging_max_items: 100
      duplicate_ratio_threshold: 0.30
      after_remote_pull: true
      after_task_complete: true
      schedule:
        daily: true
        weekly_project_summary: true
    outputs:
      summaries_path: .moyuan/memory/compacted/
      archive_path: .moyuan/memory/archive/
    require_approval_for:
      - decision
      - security
      - organization_preference

  maintenance:
    enabled: true
    reflection_interval: daily
    compact_interval: daily
    mark_stale_on_comprehension_change: true
    merge_duplicates: true
    require_approval_for_conflicts: true
```

## 10. Workspace 结构

```text
.moyuan/memory/
  facts.jsonl
  decisions.md
  preferences.yaml
  lessons.jsonl
  candidates.jsonl
  staging.jsonl
  audit.jsonl
  indexes/
    vector/
    graph/
    relational.sqlite
  compacted/
    session/
    task/
    topic/
    project/
  archive/
  runtime/
    current.json
  reflections/
    daily/
    weekly/
```

## 11. 与项目阅读理解联动

项目阅读理解是 memory 的主要来源之一。

项目接入后：

```text
full comprehension
  -> memory candidates
  -> record gate
  -> staging
  -> user approval if needed
  -> async commit
```

拉取远程分支后：

```text
incremental comprehension
  -> retrieve existing project facts
  -> compare changed code
  -> mark stale memory
  -> create new candidates
  -> staging dedup
```

## 12. 与多 Agent 协作联动

每个 Agent 的 memory 行为不同：

| Agent | Retrieve | Record |
| --- | --- | --- |
| planner | 历史需求、决策、偏好 | 新需求约束、验收标准 |
| architect | 架构事实、模块边界、ADR | 新架构决策候选 |
| backend | 模块事实、API 约定、lessons | 已验证实现经验 |
| backend_tuning | 性能历史、测试命令、瓶颈记录 | 优化结论和指标 |
| tester | 测试策略、历史缺口 | 新测试命令和回归经验 |
| quality_guard | 质量规范、重复问题 | 质量经验和坏味道模式 |
| reviewer | 历史风险、决策、规范 | 被接受的 review 结论 |
| memory_curator | 全部候选和冲突 | 合并、过期、降权结果 |

## 13. 落地范围

CLI 和 Phase 以 [总体规划与生命周期路线图](./lifecycle-roadmap.md) 为唯一权威来源。本模块先实现轻量版本：

- Record Gate：规则 + 强模型评分。
- Extraction：轻模型结构化抽取。
- Staging：JSONL 暂存 + 简单去重。
- Storage：SQLite + JSONL + 本地向量索引。
- Retrieval：关键词 + 向量混合检索。
- Compact：自动 session/task compact + daily reflection。
- Maintenance：自动 compact、过期标记和冲突队列。
- Prompt：内置 record/retrieve system prompt 和 runtime prompt。
