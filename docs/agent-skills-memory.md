# Agent、Skills 与编排

本文保留 Agent role、team、输出契约和 memory scope 的概要。Subagent 生命周期、Skill Registry、Skill 推荐、Skill 绑定和效果反馈的完整设计由 [Subagent 与 Skills 系统方案](./subagents-skills-system.md) 维护。

## 1. Agent 角色体系

Agent 不是模型本身，而是以下配置的组合：

```text
角色 + 工具权限 + memory scope + skills + 输出契约 + 模型策略
```

详细 Memory 机制由 [Agent Memory 系统方案](./agent-memory-system.md) 维护，本文件只定义 Agent 如何引用 memory。

Agent Role 是职责模板。真正被 Orchestrator 调度执行的是 Subagent。Subagent 是带有父对象、role、runtime、skills、memory scope、读写范围和生命周期的一次执行实例。

## 2. 核心角色

| Role | 职责 | 典型输入 | 典型输出 |
| --- | --- | --- | --- |
| planner | 需求澄清、任务拆解、验收标准 | 用户需求、项目画像 | task plan、验收标准、风险 |
| requirement_refiner | 丰富任务需求描述并识别缺失信息 | 用户原始需求、项目画像、memory | clarified requirement、clarification questions |
| clarification_gate | 判断是否必须追问用户 | clarified requirement、风险、验收标准 | proceed 或 needs_user_input |
| issue_planner | 将 Epic 拆分为可执行 issues | 用户目标、项目画像、模块地图 | issues、验收标准、写入范围 |
| dependency_planner | 构建 issue 依赖图 | issues、模块边界、技术方案 | issue graph、前置依赖、阻塞关系 |
| scheduler | 计算 ready queue 和并发度 | issue graph、资源、写入范围、策略 | schedule、parallelism、blocked reason |
| architect | 架构设计、模块边界、技术方案 | 需求、代码结构、历史决策 | 设计文档、接口约定、迁移方案 |
| project_reader | 项目阅读理解 | 仓库、配置、文档、diff | 项目画像、模块地图、memory candidates |
| module_mapper | 模块边界和依赖维护 | 项目画像、代码结构、diff | module map、边界说明 |
| backend | 后端功能实现 | task plan、设计文档、相关代码 | 代码变更、测试说明 |
| backend_tuning | 后端性能调优 | 性能问题、日志、指标、代码 | 优化方案、代码变更、指标对比 |
| frontend | 前端功能和交互实现 | 设计、接口约定、UI 需求 | 页面变更、组件、交互验证 |
| tester | 测试设计和执行 | 需求、代码变更、风险 | 测试用例、测试结果、缺口 |
| quality_guard | 质量门禁、重复度、复杂度和可维护性审核 | diff、quality report、项目规范 | 质量结论、返工项、阻断原因 |
| bug_triager | 运行信号和疑似 bug 分类 | runtime signal、日志、测试失败、用户反馈 | bug classification、evidence、blocked reason |
| repair_agent | 低风险 bug 的最小修复和回归测试 | confirmed bug、repair plan、write scope | 修复 diff、回归测试、风险说明 |
| improvement_curator | 修复经验和能力增强候选整理 | repair attempts、quality reports、memory candidates | improvement record、skill/model/quality 建议 |
| reviewer | 代码审查和风险识别 | diff、设计、测试结果 | review findings、修复建议 |
| security | 安全审计 | 代码、依赖、配置、权限 | 漏洞风险、修复建议 |
| release_manager | 版本分支建议、release notes、tag、push/PR、服务器部署投产和维护 | integration branch、accepted issues、测试记录、服务器配置 | release/deploy suggestion、release note、tag、PR/MR、deployment record |
| memory_curator | 记忆候选审批、compact 压缩、合并、过期和冲突处理 | memory candidates、run 记录、用户反馈 | compact summary、写入建议、过期建议 |

## 3. Team 配置

默认 feature team：

```yaml
planner: planner
requirement_refiner: requirement_refiner
clarification_gate: clarification_gate
issue_planner: issue_planner
dependency_planner: dependency_planner
scheduler: scheduler
implementers:
  - backend
  - frontend
verifiers:
  - tester
  - quality_guard
  - reviewer
self_repair:
  triager: bug_triager
  repairer: repair_agent
  curator: improvement_curator
```

默认 Runtime 绑定：

```yaml
role_runtime_defaults:
  frontend: claude_cli
  backend: codex_cli
  backend_tuning: codex_cli
  tester: codex_cli
  bug_triager: codex_cli
  repair_agent: codex_cli
  improvement_curator: codex_cli
  reviewer: codex_cli
  architect: claude_cli
  planner: claude_cli
```

设计意图：

- 前端任务默认交给 Claude CLI，利用其长上下文和 UI/交互实现能力。
- 后端和后端调优默认交给 Codex CLI，强调代码生成、测试补齐、审查修复和自动化返工。
- Runtime 绑定只是默认值，Orchestrator 仍可根据健康状态、权限、预算和 fallback 策略调整。
- 无论使用哪个 Runtime，最终都必须回到 Moyuan 的 diff 审计、质量门禁和 review。

后端调优 team：

```yaml
planner: planner
implementers:
  - backend_tuning
verifiers:
  - tester
  - quality_guard
  - reviewer
```

项目接入和远程同步 team：

```yaml
readers:
  - project_reader
  - module_mapper
curators:
  - memory_curator
```

## 4. Agent 输出契约

所有 Agent 输出都需要结构化，便于 Orchestrator 做审计和调度。

```yaml
status: completed | failed | needs_input | needs_rework
summary: string
changed_files:
  - path: string
    reason: string
commands:
  - command: string
    result: passed | failed | skipped
tests:
  - name: string
    result: passed | failed | skipped
risks:
  - severity: low | medium | high | blocker
    description: string
    mitigation: string
next_actions:
  - string
memory_candidates:
  - type: fact | decision | preference | lesson | quality | comprehension | release | security
    text: string
    confidence: number
```

## 5. Subagent 引用

Subagent 的权威定义见 [Subagent 与 Skills 系统方案](./subagents-skills-system.md)。

最小创建链路：

```text
Issue / Run / Repair Attempt
  -> resolve Agent Role
  -> select skills
  -> assemble memory scope
  -> create Subagent
  -> dispatch Runtime
  -> validate output
  -> quality gate / review
```

Subagent 不能绕过 Issue Graph、权限、Runtime Adapter、质量门禁、review 或 Memory Record Gate。

## 6. Skills 体系

Skill 是可复用能力包，用于让 Agent 在特定任务上有稳定流程和知识。完整 Skill Registry、推荐、绑定和效果反馈见 [Subagent 与 Skills 系统方案](./subagents-skills-system.md)。

Skill 类型：

- 语言框架类：Spring Boot、FastAPI、React、Vue、Go、Rust。
- 工程流程类：代码审查、测试补全、重构、迁移。
- 性能类：SQL 优化、缓存分析、并发调优、前端性能。
- 安全类：依赖安全、权限模型、敏感信息检查。
- 运维类：Docker、Kubernetes、CI/CD、日志分析。
- 领域类：电商订单、支付、会员、风控、数据报表。
- 工具类：Git、数据库、搜索、文档生成、Mermaid 图。

`find-skills` 推荐流程：

```text
Project Comprehension
  -> stack detection
  -> task intent classification
  -> role requirements
  -> find-skills query
  -> score skills
  -> recommend enable/bind
```

## 7. Role 与 Skill 绑定

默认绑定：

- planner：requirement-analysis、task-breakdown、risk-analysis。
- requirement_refiner：requirement-enrichment、acceptance-criteria、scope-definition。
- clarification_gate：intent-clarification、risk-questioning、approval-gate。
- issue_planner：issue-breakdown、acceptance-criteria、write-scope-planning。
- dependency_planner：dependency-analysis、dag-planning、blocking-analysis。
- scheduler：parallel-scheduling、resource-planning、conflict-detection。
- architect：architecture-design、adr-writing、dependency-analysis。
- project_reader：project-comprehension、dependency-analysis、command-detection。
- module_mapper：module-boundary-analysis、dependency-map。
- backend：api-design、database-migration、backend-development。
- backend_tuning：profiling、sql-optimization、cache-analysis、benchmarking。
- frontend：component-design、accessibility-check、frontend-performance。
- tester：unit-test-generation、integration-test-plan、regression-test。
- quality_guard：code-quality-review、duplication-check、complexity-analysis、architecture-boundary-check。
- reviewer：code-review、test-gap-analysis、security-review。
- release_manager：release-branch-planning、release-note、tag-planning、remote-publish-checklist、deployment-checklist、rollback-plan。
- memory_curator：memory-dedup、memory-compact、memory-reflection、conflict-resolution。

动态绑定：

- 生成或修改代码时必须追加 `quality_guard`。
- 复杂开发目标必须先启用 `issue_planner` 和 `dependency_planner`。
- 用户原始需求进入拆分前必须先启用 `requirement_refiner` 和 `clarification_gate`。
- 存在多个 ready issues 时必须启用 `scheduler` 判断并发度。
- 新增大量代码或跨模块修改时追加 `complexity-analysis` 和 `duplication-check`。
- 涉及鉴权时追加 `security-review`。
- 涉及迁移时追加 `migration-planning`。
- 涉及 UI 回归时追加 `visual-regression`。
- 运行失败、测试失败、冒烟失败或用户反馈异常时追加 `bug_triager`。
- 低风险 confirmed bug 追加 `repair_agent`，修复完成后追加 `improvement_curator`。
- 项目接入和远程分支同步后追加 `project_reader`、`module_mapper` 和 `memory_curator`。

## 8. Memory Scope

不同 Agent 只检索与职责相关的 memory，避免上下文污染。

| Agent | Retrieve Scope | Record Scope |
| --- | --- | --- |
| planner | 历史需求、决策、偏好 | 新需求约束、验收标准 |
| requirement_refiner | 项目画像、用户偏好、历史需求约束 | clarified requirement |
| clarification_gate | 历史追问、风险规则、审批偏好 | clarification decision |
| issue_planner | 项目画像、模块地图、历史拆分经验 | issues、验收标准、写入范围 |
| dependency_planner | 模块边界、依赖图、历史阻塞关系 | issue graph、依赖关系 |
| scheduler | issue graph、资源限制、冲突历史 | schedule、并发决策 |
| architect | 架构事实、模块边界、ADR | 新架构决策候选 |
| project_reader | 项目 facts、旧项目画像、旧模块地图 | comprehension candidates |
| module_mapper | 模块事实、依赖关系、边界约束 | 模块地图更新 |
| backend | 模块事实、API 约定、lessons | 已验证实现经验 |
| backend_tuning | 性能历史、测试命令、瓶颈记录 | 优化结论和指标 |
| tester | 测试策略、历史缺口 | 新测试命令和回归经验 |
| quality_guard | 质量规范、重复问题 | 质量经验和坏味道模式 |
| bug_triager | 历史 bug、错误 signature、模块风险 | bug classification、证据和误报经验 |
| repair_agent | fix pattern、回归测试、模块约束 | 已验证修复经验和回归测试 |
| improvement_curator | repair history、质量经验、skills 效果 | improvement record、能力增强建议 |
| reviewer | 历史风险、决策、规范 | 被接受的 review 结论 |
| release_manager | 发布历史、版本批次偏好、远程仓库策略、服务器和回滚策略 | release/deploy suggestion、deployment memory、release memory |
| memory_curator | 全部候选、暂存区、冲突和低频记忆 | compact summary、合并、过期、降权结果 |
