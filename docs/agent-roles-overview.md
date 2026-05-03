# Agent 角色与团队概览

本文只定义 Agent role、team、默认 Runtime 绑定和 memory scope 的使用边界。

权威边界：

| 内容 | 权威文档 |
| --- | --- |
| Subagent 生命周期、父对象、并发、输出汇聚 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| Skill Registry、推荐、绑定和效果反馈 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| Memory record、retrieve、compact、维护 | [Agent Memory 系统方案](./agent-memory-system.md) |
| Runtime Adapter、Claude CLI、Codex CLI、模型服务商 | [模型与工具适配规划](./model-tool-adapters.md) |
| Issue Graph、ready queue、并发和等待 | [Issues 编排与并发调度](./issue-orchestration.md) |

## 1. Agent 与 Subagent

Agent 是职责模板，不直接等同于模型或执行进程。

```text
Agent Role = 职责 + 工具权限 + memory scope + 默认 skills + 输出契约 + 模型策略
Subagent = Orchestrator 为某个 issue/run/repair/release 创建的一次执行实例
```

约束：

- Agent role 只描述“该由谁做”。
- Subagent 描述“这次任务由哪个执行实例做”。
- Runtime 描述“通过 Claude CLI、Codex CLI 或模型 API 怎么执行”。
- Orchestrator 是唯一能创建、停止、重试和合并 Subagent 输出的模块。

## 2. 角色目录

| Role | 职责 | 默认 Runtime | 典型输出 |
| --- | --- | --- | --- |
| `requirement_refiner` | 丰富用户需求，补齐背景、范围、验收和风险 | `claude_cli` | clarified requirement |
| `clarification_gate` | 判断是否必须向用户追问 | `claude_cli` | proceed / needs_user_input |
| `issue_planner` | 拆分 issues | `claude_cli` | issue list |
| `dependency_planner` | 构建 issue graph | `claude_cli` | dependency graph |
| `scheduler` | 计算 ready queue、并发度和等待原因 | `codex_cli` | schedule |
| `project_reader` | 项目阅读理解 | `codex_cli` | project profile、module map |
| `module_mapper` | 维护模块边界和依赖 | `codex_cli` | module map patch |
| `architect` | 架构方案、接口契约、跨模块设计 | `claude_cli` | design / ADR |
| `backend` | 后端功能实现 | `codex_cli` | backend diff、tests |
| `backend_tuning` | 后端性能调优 | `codex_cli` | benchmark、optimization diff |
| `frontend` | 前端功能、组件和交互 | `claude_cli` | frontend diff、UI validation |
| `tester` | 测试设计、补齐和执行 | `codex_cli` | test report |
| `quality_guard` | 重复度、复杂度、架构边界和质量门禁 | `codex_cli` | quality report |
| `reviewer` | 独立代码审查 | `codex_cli` | review findings |
| `security` | 鉴权、安全、敏感信息和依赖风险审查 | `codex_cli` | security report |
| `bug_triager` | 判断运行信号是否为 bug | `codex_cli` | bug classification |
| `repair_agent` | 低风险 confirmed bug 的最小修复 | `codex_cli` | repair diff、regression result |
| `improvement_curator` | 从修复和反馈中提炼能力增强候选 | `codex_cli` | improvement record |
| `release_manager` | 版本分支、release note、tag、PR/MR、部署建议 | `codex_cli` | release/deploy suggestion |
| `memory_curator` | Memory 候选、compact、合并、过期和冲突整理 | `codex_cli` | memory maintenance report |

## 3. 默认 Team

Feature 开发 team：

```yaml
planners:
  - requirement_refiner
  - clarification_gate
  - issue_planner
  - dependency_planner
  - scheduler
implementers:
  - backend
  - frontend
verifiers:
  - tester
  - quality_guard
  - reviewer
```

项目接入 team：

```yaml
readers:
  - project_reader
  - module_mapper
curators:
  - memory_curator
```

运行反馈与自我修复 team：

```yaml
triage:
  - bug_triager
repair:
  - repair_agent
verify:
  - tester
  - quality_guard
  - reviewer
curate:
  - improvement_curator
  - memory_curator
```

发布 team：

```yaml
release:
  - release_manager
verify:
  - tester
  - quality_guard
  - reviewer
```

## 4. 默认 Runtime 分工

| 场景 | 默认分配 |
| --- | --- |
| 前端开发、UI 交互、复杂页面实现 | `frontend` + `claude_cli` |
| 后端开发、测试补齐、质量修复 | `backend` / `tester` / `quality_guard` + `codex_cli` |
| 后端调优、性能定位、回归验证 | `backend_tuning` + `codex_cli` |
| 需求拆分和架构方案 | `requirement_refiner` / `architect` + `claude_cli` |
| 审查、修复、自我修复和发布建议 | `codex_cli` 优先 |

默认绑定可以被项目配置覆盖，但覆盖后仍必须通过 Runtime Adapter、权限、预算、质量门禁和审计。

## 5. 输出契约

所有 Agent 输出都必须能被 Orchestrator 校验。最小结构：

```yaml
status: completed | failed | needs_input | needs_rework
summary: string
changed_files: []
commands: []
tests: []
risks: []
next_actions: []
memory_candidates: []
```

Subagent 级字段、错误码和效果反馈见 [Subagent 与 Skill 契约](./contracts/subagent-skill-contract.md)。

## 6. Memory Scope

Agent 只声明 memory 使用范围，不定义 Memory 机制。

| Role | Retrieve Scope | Record Scope |
| --- | --- | --- |
| planning roles | 项目画像、历史需求、用户偏好、架构决策 | clarified requirement、issue graph、决策候选 |
| project reader roles | 旧项目画像、模块地图、构建命令、过期 facts | comprehension candidates |
| implementation roles | 模块事实、API 约定、测试命令、历史 lessons | 已验证实现经验 |
| quality roles | 质量规范、历史坏味道、覆盖率要求 | review 结论、质量经验 |
| self-repair roles | bug signature、fix pattern、回归测试经验 | 修复经验、能力增强候选 |
| release roles | 发布历史、回滚策略、服务器资源和风险记录 | release/deployment memory |
| memory curator | 候选、暂存区、冲突、低频记忆 | compact summary、合并/过期结果 |

Memory 是否记录、如何抽取、何时 compact 和如何维护，只以 [Agent Memory 系统方案](./agent-memory-system.md) 为准。

## 7. 变更规则

- 新增角色时，必须补充职责、默认 Runtime、可用工具和 memory scope。
- 新增 Skill 绑定规则时，更新 [Subagent 与 Skills 系统方案](./subagents-skills-system.md)，本文只保留角色引用。
- 新增 Runtime 或模型服务商时，更新 [模型与工具适配规划](./model-tool-adapters.md)。
- 新增 issue 调度规则时，更新 [Issues 编排与并发调度](./issue-orchestration.md) 和对应策略文档。
