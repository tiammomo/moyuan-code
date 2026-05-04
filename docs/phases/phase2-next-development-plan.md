# Phase 2 实施记录

状态：in_progress
责任角色：orchestrator_owner + adapter_owner + skills_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 2 阶段从规划到执行的实际顺序。稳定设计结论需要回写到 `subagents-skills-system.md`、`model-tool-adapters.md`、策略、契约或配置文档；本文件只记录阶段执行事实。

## 1. 当前基线

Phase 1 本地 CLI MVP 已完成，Beta 第一批控制面能力已完成。当前可复用能力：

- Gin + GORM API 控制面和 Next.js 16 Console。
- Provider Registry、runtime route、Claude CLI/Codex CLI native runtime contract。
- Requirement -> Issue Graph -> Schedule -> Run -> Quality -> Review 的基础闭环。
- Subagent Instance 模型、run visibility 和 Issue Inspector。
- Quality Policy、Quality Reports 和 Quality Explanation API。
- 服务器资源 registry、release/deploy plan、controlled deploy executor 和 health scan 基线。

## 2. Phase 2 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase2-001` | `skill-registry-store-api` | completed | 建立 Skill Definition、启用状态、风险和适配 role 的存储/API/CLI | 可登记、查询、禁用 skill，并被项目 workspace 持久化 |
| P0 | `phase2-002` | `find-skills-recommendation-adapter` | completed | 生成项目、role、issue 级 skills 推荐结果 | 推荐结果有理由、来源、风险和落盘记录 |
| P0 | `phase2-003` | `role-skill-binding-policy` | completed | 将 skills 绑定到 role、issue、subagent | Agent 执行前能解析 skills，冲突和禁用项会阻断 |
| P1 | `phase2-004` | `skill-effectiveness-feedback` | completed | Skill 效果反馈进入质量和 memory 闭环 | 低效果或高风险 skill 可降权或禁用 |
| P1 | `phase2-005` | `provider-health-quota-usage` | planned | Provider 健康、额度、用量、成本和数据策略扩展 | 路由可解释地考虑健康、额度和成本 |
| P1 | `phase2-006` | `task-model-strategy-switch` | planned | 同一任务支持模型策略切换和 fallback | 切换过程可审计，不能绕过质量门禁 |
| P1 | `phase2-007` | `native-runtime-session-recovery` | planned | Claude/Codex CLI session resume 和失败降级 | runtime 失败可恢复、归档或安全降级 |
| P2 | `phase2-008` | `gpt-image-2-diagram-pipeline` | planned | gpt-image-2 架构图、流程图和部署拓扑图流水线 | 资产、prompt、diagram spec 和说明可追踪 |
| P1 | `phase2-009` | `subagent-scheduler-retry-archive` | planned | Subagent 调度、重试、归档和输出收敛增强 | Subagent 生命周期可执行、可复盘、可审计 |

## 3. 已完成任务：`phase2-001 skill-registry-store-api`

范围草案：

- 新增 Skill Definition 数据对象。
- 新增 project workspace 下的 skills registry 文件。
- 新增 CLI：`moyuan skills list`、`moyuan skills add`、`moyuan skills disable`。
- 新增 API：`GET /v1/projects/:project_id/skills`、`POST /skills`、`POST /skills/:skill_id/disable`。
- 新增最小测试，覆盖登记、去重、禁用和不保存明文密钥。

非目标：

- 不真实安装第三方 skill。
- 不执行 skill prompt。
- 不接入效果评分。
- 不修改 Subagent 调度逻辑。

退出条件：

- Skill Registry 可落盘、查询、禁用。
- 明文 secret 被拒绝。
- `go test ./...` 通过。

完成记录：

- 已新增 `internal/skills`，Skill Registry 写入 `.moyuan/skills/registry.json`，事件写入 `.moyuan/skills/events.jsonl`。
- 已支持 CLI：`moyuan skills add`、`moyuan skills list`、`moyuan skills disable <skill-id>`。
- 已支持 API：`GET /v1/projects/:project_id/skills`、`POST /v1/projects/:project_id/skills`、`POST /v1/projects/:project_id/skills/:skill_id/disable`。
- 已覆盖登记、去重、禁用、明文 secret 阻断、API 路由和 CLI smoke。
- 当前不安装第三方 skill、不执行 skill prompt、不修改 Subagent 调度逻辑。

## 4. 已完成任务：`phase2-002 find-skills-recommendation-adapter`

范围草案：

- 基于 Project Comprehension、role、issue 类型、风险和现有 Skill Registry 生成推荐候选。
- 预留外部 `find-skills` adapter 入口，当前可先实现本地规则 fallback。
- 推荐结果写入 `.moyuan/skills/recommendations.jsonl`。
- 推荐结果包含 skill id、score、reasons、risks 和 target role/issue。

非目标：

- 不真实安装外部 skill。
- 不自动启用高风险 skill。
- 不把推荐结果直接绑定到 Subagent。

退出条件：

- CLI/API 可生成 skills recommendation。
- 推荐结果可审计落盘。
- 禁用 skill 不会被推荐为 enabled candidate。
- `go test ./...` 通过。

完成记录：

- 已新增 `skills.Recommend` 本地规则 fallback，基于 role、task type、risk 和 Skill Registry 评分。
- 推荐结果写入 `.moyuan/skills/recommendations.jsonl`，并记录 `skill.recommendation.created` 审计日志。
- 已支持 CLI：`moyuan skills recommend --role backend [--task-type testing] [--risk medium]`。
- 已支持 API：`POST /v1/projects/:project_id/skills/recommend`。
- 禁用 skill 不会进入推荐候选；当前不真实调用外部 `find-skills` 网络服务。

## 5. 已完成任务：`phase2-003 role-skill-binding-policy`

范围草案：

- 新增 Skill Binding 数据对象和 `.moyuan/skills/bindings.json`。
- 支持 project、role、issue、subagent 四级绑定目标。
- 绑定时校验 skill 存在、enabled、risk 和 target type。
- 提供 CLI/API 创建、查询和禁用绑定。
- 后续 Orchestrator 可在创建 Subagent 前解析绑定，但本任务先不改变执行调度。

非目标：

- 不自动把 recommendation 绑定为 enabled。
- 不执行 skill prompt。
- 不改变 Subagent 写入范围。

退出条件：

- Skill Binding 可落盘、查询、禁用。
- 禁用或不存在的 skill 不能被绑定为 enabled。
- `go test ./...` 通过。

完成记录：

- 已新增 Skill Binding 存储：`.moyuan/skills/bindings.json`。
- Binding 事件写入 `.moyuan/skills/bindings.events.jsonl`，并记录 audit log。
- 已支持 CLI：`moyuan skills bind`、`moyuan skills bindings`、`moyuan skills binding disable <binding-id>`。
- 已支持 API：`GET /v1/projects/:project_id/skills/bindings`、`POST /skills/bindings`、`POST /skills/bindings/:binding_id/disable`。
- 已阻断缺失 skill、disabled skill、高风险 project 级绑定和包含明文 secret 的 binding config。
- 当前不改变 Subagent 调度，仅为后续 Orchestrator binding resolution 提供稳定状态。

## 6. 已完成任务：`phase2-004 skill-effectiveness-feedback`

范围草案：

- 新增 Skill Effectiveness 数据对象和 `.moyuan/skills/effectiveness/` 记录。
- 记录 skill 对 quality、review、rework、耗时和风险的影响。
- 提供 CLI/API 创建和查询 effectiveness report。
- 后续推荐阶段可读取 effectiveness 做加权，本任务先写入与查询。

非目标：

- 不自动降权或禁用 skill。
- 不修改 recommendation score。
- 不改 Orchestrator 执行链路。

退出条件：

- Skill Effectiveness 可落盘、查询。
- 记录必须关联 skill id 和 subagent/run/issue 之一。
- `go test ./...` 通过。

完成记录：

- 已新增 Skill Effectiveness 记录，单条记录写入 `.moyuan/skills/effectiveness/<id>.json`。
- 汇总事件写入 `.moyuan/skills/effectiveness/effectiveness.jsonl`，并记录 audit log。
- 已支持 CLI：`moyuan skills effectiveness add`、`moyuan skills effectiveness list`。
- 已支持 API：`GET /v1/projects/:project_id/skills/effectiveness`、`POST /skills/effectiveness`。
- 已阻断缺失 skill、缺失 issue/run/subagent 引用、非法 outcome/quality impact，并会过滤 finding 里的明显 secret。
- 当前不自动影响推荐分数，不自动降权或禁用 skill。

## 7. 当前任务：`phase2-005 provider-health-quota-usage`

范围草案：

- 扩展 Provider Registry 的健康、额度、用量、成本和状态快照。
- 提供 provider health/usage update API 和 CLI。
- 路由解释中能展示 provider disabled、quota risk、health risk 的阻断原因。
- 为后续 task-model-strategy-switch 提供 provider 可用性输入。

非目标：

- 不真实调用云厂商账单 API。
- 不自动扣费或结算。
- 不改变现有 Claude/Codex runtime 执行流程。

退出条件：

- Provider 可记录 health、quota、usage 和 cost snapshot。
- Provider route 可基于 disabled/health/quota 给出明确阻断原因。
- `go test ./...` 通过。

## 8. Phase 2 收口规则

- 每完成一个 Phase 2 issue，必须同步本实施记录和 issue graph。
- 若实现产生稳定对象、状态机、配置字段或错误码，必须回写到对应专题、策略或契约文档。
- Phase 2 仍保持“AI 生成代码必须通过质量门禁和 review 后才能合入”的主规则。
