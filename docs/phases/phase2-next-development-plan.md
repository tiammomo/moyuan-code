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
| P1 | `phase2-005` | `provider-health-quota-usage` | completed | Provider 健康、额度、用量、成本和数据策略扩展 | 路由可解释地考虑健康、额度和成本 |
| P1 | `phase2-006` | `task-model-strategy-switch` | completed | 同一任务支持模型策略切换和 fallback | 切换过程可审计，不能绕过质量门禁 |
| P1 | `phase2-007` | `native-runtime-session-recovery` | completed | Claude/Codex CLI session resume 和失败降级 | runtime 失败可恢复、归档或安全降级 |
| P2 | `phase2-008` | `gpt-image-2-diagram-pipeline` | completed | gpt-image-2 架构图、流程图和部署拓扑图流水线 | 资产、prompt、diagram spec 和说明可追踪 |
| P1 | `phase2-009` | `subagent-scheduler-retry-archive` | completed | Subagent 调度、重试、归档和输出收敛增强 | Subagent 生命周期可执行、可复盘、可审计 |

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

## 7. 已完成任务：`phase2-005 provider-health-quota-usage`

范围草案：

- 扩展 Provider Registry 的健康、额度、用量、成本和状态快照。
- 提供 provider health/usage update API、CLI 和本地 refresh。
- 路由解释中能展示 provider disabled、quota risk、health risk 的阻断原因。
- 为后续 task-model-strategy-switch 提供 provider 可用性输入。

非目标：

- 不真实调用云厂商账单 API。
- 不自动扣费或结算。
- 不改变现有 Claude/Codex runtime 执行流程。

退出条件：

- Provider 可记录 health、quota、usage 和 cost snapshot。
- Provider 可基于本地可验证信号 refresh ops snapshot。
- Provider route 可基于 disabled/health/quota 给出明确阻断原因。
- `go test ./...` 通过。

完成记录：

- Provider Registry 已新增 `health`、`quota`、`usage` 和 `cost` 运行期快照字段。
- 已支持 CLI：`moyuan model provider ops <provider-id> [--health ok] [--quota-status ok] [--used-tokens 1000]`、`moyuan model provider refresh [--provider <provider-id>]`。
- 已支持 API：`POST /v1/projects/:project_id/providers/:provider_id/ops`、`POST /v1/projects/:project_id/providers/ops/refresh`。
- 路由会阻断 `health.status == unhealthy/down`、`quota.status == exhausted`、`cost.status == exceeded` 的 provider。
- API provider 选择会跳过不可用 provider；直接命中的 image provider 会返回明确 blocked decision。
- Ops refresh 会检查 native runtime 是否可发现、API provider 的 `auth_ref/base_url` 是否齐全，并按 quota/cost 阈值刷新状态；当前不真实调用云厂商账单或模型 API。

## 8. 已完成任务：`phase2-006 task-model-strategy-switch`

范围草案：

- 新增任务级 model strategy 对象。
- 支持 `default`、`frontend-first`、`backend-safe`、`low-cost-memory`、`image-diagram` 等策略名。
- Provider route 可接收策略输入，决定 runtime/provider/fallback。
- 策略切换必须保留审计记录，不能绕过 data policy、quality gate 和 review。

非目标：

- 不重写 Orchestrator 调度器。
- 不真实执行 fallback runtime。
- 不自动修改用户配置。

退出条件：

- 同一任务可通过 CLI/API 指定模型策略并得到可解释 route。
- 策略不会绕过 secret、sensitive code、project memory 和 provider availability 阻断。
- `go test ./...` 通过。

完成记录：

- `RouteRequest` 已新增 `model_strategy`，`RouteDecision` 会返回归一化后的 `strategy`。
- 已支持策略：`frontend-first`、`backend-safe`、`low-cost-memory`、`image-diagram`、`planning`。
- 已支持 CLI：`moyuan model route --strategy <strategy>`。
- 已支持 API：`POST /v1/projects/:project_id/provider-route` 传入 `model_strategy`。
- 策略不会绕过 secret、data policy、provider health、quota 或 cost 阻断。

## 9. 已完成任务：`phase2-007 native-runtime-session-recovery`

范围草案：

- 为 Claude CLI/Codex CLI runtime session 建立 resume metadata。
- Runtime invocation 失败时归档 session、stdout/stderr、diff snapshot 和 fallback reason。
- 支持 CLI/API 查询 session recovery status。
- 为后续 Subagent retry/archive 提供 runtime recovery 输入。

非目标：

- 不真实调用 Claude/Codex resume 命令。
- 不改变 issue merge/quality gate。
- 不自动 fallback 到另一 runtime 执行代码。

退出条件：

- Native runtime session 可记录 resume id、failure category、fallback candidate 和 archived status。
- 失败不能丢失 diff snapshot 和审计日志。
- `go test ./...` 通过。

完成记录：

- Native Runtime 失败时会生成 `native_session_id` 和 `recovery_id`。
- Recovery 记录写入 `.moyuan/runtimes/recoveries/<recovery-id>.json`，事件写入 `.moyuan/runtimes/recoveries/events.jsonl`。
- Session 归档写入 `.moyuan/runtimes/sessions/<session-id>/stdout.txt` 和 `stderr.txt`。
- Runtime result 会保留 diff summary、changed files、risk、fallback candidate 和 resume hint。
- 已支持 CLI：`moyuan runtime recovery list`、`moyuan runtime recovery show <recovery-id>`。
- 已支持 API：`GET /v1/projects/:project_id/runtime-recoveries`、`GET /runtime-recoveries/:recovery_id`。
- 当前不自动执行真实 resume，不自动切换 runtime，避免恢复过程绕过质量门禁。

## 10. 已完成任务：`phase2-009 subagent-scheduler-retry-archive`

范围草案：

- 将 runtime recovery、skill binding 和 provider route 接入 Subagent 调度输入。
- 为 Subagent 增加 retry policy、archive reason、blocked reason 和 output convergence 状态。
- 支持 Subagent 失败后按风险决定 retry、needs_rework、archive 或等待人工处理。
- 控制并发时考虑 write scope、runtime slot、provider availability 和 recovery backlog。

非目标：

- 不自动合入失败后重试产生的代码。
- 不绕过 review/quality gate。
- 不实现生产级分布式队列。

退出条件：

- Subagent lifecycle 可表达 retrying、archived、waiting_runtime、needs_rework。
- Scheduler 可读取 Subagent retry/archive 状态，并避免无限重试。
- CLI/API 能查询 Subagent retry/archive 结果。
- `go test ./...` 通过。

完成记录：

- Subagent Instance 已新增 `retry_policy`、`retry_count`、`max_retries`、`blocked_reason`、`archive_reason`、`recovery_id`、`failure_category` 和 `output_converged`。
- Orchestrator 在 Native Runtime recovery 出现时会把 Subagent 标记为 `archived`，并保留 recovery 和 failure category。
- Scheduler plan 已新增 `subagent_backlog`，dispatch decision 会带上 subagent/recovery/retry 字段。
- Scheduler 会因 `subagent_waiting_runtime`、`subagent_needs_rework`、`subagent_retry_exhausted` 阻止继续 dispatch，避免无限重试。
- CLI/API 通过现有 Subagent list/show 和 schedule endpoint 查询 retry/archive 状态。

## 11. 已完成任务：`phase2-008 gpt-image-2-diagram-pipeline`

范围草案：

- 建立 diagram spec 数据对象，描述架构图、Issue Graph 图、部署拓扑图和多 Agent 编排图。
- 将项目理解、Provider 路由、Subagent/Scheduler 状态和部署资源转换为可脱敏的图像生成输入。
- 复用现有 `scripts/` 图像生成脚本，输出资产索引、prompt、说明文档和版本记录。
- 支持 CLI/API 生成 diagram plan，当前先不自动提交图片到 release。

非目标：

- 不把密钥、私网 IP、生产事故原文或完整项目 memory dump 传给图像模型。
- 不要求图片作为质量门禁必需项。
- 不实现复杂在线图片编辑器。

退出条件：

- 可生成可审计的 diagram spec 和 asset record。
- 生成输入经过脱敏和范围裁剪。
- CLI/API 能查询图像资产索引。
- `go test ./...` 通过。

完成记录：

- 已新增 Visual Diagram plan 能力，生成 `.moyuan/visuals/specs/`、`.moyuan/visuals/prompts/` 和 `.moyuan/visuals/assets/`。
- 已支持 diagram type：`architecture`、`issue_graph`、`multi_agent`、`deployment_topology`、`release_flow`。
- 已对 prompt/spec 输入做 secret、API key 和私网 IP 脱敏。
- 已接入 Provider Route，使用 `image-diagram` 策略判断 `gpt_image_2` 是否可用；不可用时 asset status 为 `route_blocked`。
- 已支持 CLI：`moyuan visuals diagram plan`、`moyuan visuals assets`、`moyuan visuals asset show <asset-id>`、`moyuan visuals asset render <asset-id>`、`moyuan visuals renders`。
- 已支持 API：`POST /v1/projects/:project_id/visuals/diagrams/plan`、`GET /visuals/assets`、`GET /visuals/assets/:asset_id`、`POST /visuals/assets/:asset_id/render`、`GET /visuals/render-executions`。
- 已新增受控 render execution：默认 dry-run 不调用图像 API；script mode 必须显式 approval、`MOYUAN_ALLOW_IMAGE_SCRIPT=1`、`OPENAI_API_KEY` 和脚本文件同时满足。

## 12. Phase 2 收口与下一步

Phase 2 第一批 issue 已完成：

- Skills registry/recommendation/binding/effectiveness。
- Provider health/quota/usage/cost 和 model strategy。
- Native Runtime recovery。
- Subagent retry/archive/scheduler backlog。
- Visual diagram plan 和 asset index。

下一步建议：

- 做一次 Phase 2 release readiness 检查，确认文档、API、CLI、Console 和测试状态一致。
- 将 visual assets、runtime recoveries、subagent backlog 接入 Web Console。
- 将 `scripts/` 图像生成执行接入受控 CLI/API，但仍保持不阻塞代码质量门禁。
- 规划下一批 issue：真实 provider 健康检查、schema validator、state store 索引深化、Console 操作流。

## 13. Phase 2 收口规则

- 每完成一个 Phase 2 issue，必须同步本实施记录和 issue graph。
- 若实现产生稳定对象、状态机、配置字段或错误码，必须回写到对应专题、策略或契约文档。
- Phase 2 仍保持“AI 生成代码必须通过质量门禁和 review 后才能合入”的主规则。
