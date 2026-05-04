# Phase 3 实施记录

状态：in_progress
责任角色：orchestrator_owner + config_owner + frontend_owner + adapter_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 3 的实际执行顺序。稳定设计结论需要回写到配置、策略、契约、Console 或相关主线文档；本文件只记录阶段执行事实。

## 1. 当前基线

Phase 2 第一批能力已完成并通过 release readiness：

- Skills registry、recommendation、binding、effectiveness。
- Provider health、quota、usage、cost 和 model strategy。
- Native Runtime recovery。
- Subagent retry/archive/scheduler backlog。
- Visual diagram plan、asset index 和受控 render execution。
- Console 可展示 runtime recoveries、subagent backlog、visual assets 和 visual render executions。

## 2. Phase 3 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase3-001` | `workspace-yaml-schema-validator` | completed | 让 `.moyuan/project.yaml`、`repository.yaml`、`policies/access.yaml` 成为可读取、可校验的配置事实源 | `workspace validate` 能发现 YAML 解析错误、条件必填、必须为空和 state drift |
| P0 | `phase3-002` | `workspace-schema-coverage-expansion` | planned | 扩展到 providers、routing、visuals、runtimes、server、release 和 budget | 核心配置域均有字段级 issue code |
| P0 | `phase3-002a` | `providers-yaml-schema-validator` | completed | 将 `models/providers.yaml` 纳入 workspace validate | provider schema、auth_ref 引用和明文密钥禁用可被阻断 |
| P0 | `phase3-002b` | `routing-yaml-schema-validator` | completed | 将 `models/routing.yaml` 纳入 workspace validate | 路由 primary/fallback provider 缺失可被阻断 |
| P0 | `phase3-002c` | `visuals-yaml-schema-validator` | completed | 将 `visuals/architecture-visuals.yaml` 纳入 workspace validate | 图像流水线策略、安全和 gpt-image-2 配置错误可被阻断 |
| P0 | `phase3-002d` | `agent-runtimes-yaml-schema-validator` | completed | 将 `runtimes/agent-runtimes.yaml` 纳入 workspace validate | Claude/Codex Runtime 配置错误可被阻断 |
| P1 | `phase3-003` | `console-operation-actions` | planned | Console 增加受控操作入口和后端 preview/dry-run 对齐 | 高风险动作不能绕过 approval/authz |
| P1 | `phase3-003a` | `visual-render-dry-run-console-action` | completed | Visual Assets 面板触发后端 dry-run render | dry-run action 可见、可反馈 execution id，不调用真实图片 API |
| P1 | `phase3-004` | `runtime-log-diff-viewer` | completed | Console 展开 runtime 日志、diff summary 和 resume hint | 失败排查证据链可见 |
| P1 | `phase3-005` | `provider-probe-adapters` | planned | Provider refresh 接入可选轻量探测 adapter | 探测失败可解释，密钥不落盘 |
| P2 | `phase3-006` | `visual-script-auth-quality` | planned | Visual script mode 接入 auth ref、审计和图片质量检查 | 图片生成可执行且可复核 |
| P2 | `phase3-007` | `release-deploy-control-actions` | planned | Release/deploy/smoke/monitor 动作在 Console 可控 | 发布与部署流水线状态可见 |

## 3. 已完成任务：`phase3-001 workspace-yaml-schema-validator`

范围：

- 为 `ProjectConfig`、`RepositoryConfig`、`AccessConfig` 增加 YAML 字段映射。
- `Load` 优先读取用户可编辑的 `.moyuan/*.yaml`，再回退到 `workspace.json` 或默认值。
- `Validate` 同时校验 runtime state 和 YAML 文件。
- 增加 YAML 解析错误、条件必填、必须为空和 state drift 的 issue code。
- 补充测试，覆盖 YAML 覆盖读取、remote_git/local_path 互斥和 YAML 解析错误。

非目标：

- 不在本任务中校验全部 provider/server/release 配置域。
- 不引入远程配置中心。
- 不改变高风险操作审批规则。

退出条件：

- `go test ./internal/workspace` 通过。
- `go test ./...` 通过。
- 文档入口更新到 Phase 3。

完成记录：

- `ProjectConfig`、`RepositoryConfig`、`AccessConfig` 已增加 YAML 字段映射。
- `Load` 会优先读取用户可编辑的 `.moyuan/project.yaml`、`.moyuan/repository.yaml`、`.moyuan/policies/access.yaml`，再回退到 `workspace.json` 或默认值。
- `Validate` 会同时检查 runtime state 和 YAML 配置，并输出 YAML 解析错误、条件必填、必须为空和 state drift 的 issue code。
- 已补充测试覆盖 YAML 覆盖读取、`remote_git` 与 `local_path` 互斥、 malformed YAML。
- 验证通过：`go test ./internal/workspace`、`go test ./...`、`npm run typecheck`、`npm run build`。

## 4. 已完成增量：`phase3-002a providers-yaml-schema-validator`

范围：

- 新增 `.moyuan/models/providers.yaml` 路径索引。
- `workspace validate` 会在文件存在时校验 provider 配置。
- 校验内容包括 `schema_version`、`model_provider_management`、`accounts`、`providers`、`security.forbid_plaintext_api_key`。
- Account 校验覆盖 `vendor`、`api_type`、`auth_ref`、API 型 `base_url`、`enabled`、`data_policy`。
- Provider 校验覆盖 `type`、`account`、`enabled`、API 型 `models` 和 CLI 型 `capabilities/models` 互斥。
- 发现 `sk-`、非 `env:`/`secret:` auth_ref、疑似 token/api_key 明文时阻断。

验证：

- `go test ./internal/workspace` 通过。
- `go test ./...` 通过。

## 5. 已完成增量：`phase3-002b routing-yaml-schema-validator`

范围：

- 新增 `.moyuan/models/routing.yaml` 路径索引。
- `workspace validate` 会在文件存在时校验 routing 配置。
- 校验内容包括 `schema_version`、`policies`、`policies.*.primary.provider` 和 `policies.*.fallback[].provider`。
- `primary.model` 为空时输出 warning，提示 CLI provider 应显式使用 `default`。
- 发现疑似明文 token/API key 时阻断。

验证：

- `go test ./internal/workspace` 通过。
- `go test ./...` 通过。

## 6. 已完成增量：`phase3-002c visuals-yaml-schema-validator`

范围：

- 新增 `.moyuan/visuals/architecture-visuals.yaml` 路径索引。
- `workspace validate` 会在文件存在时校验 visuals 配置。
- 校验内容包括 `architecture_visuals.enabled`、`provider_policy.diagram_planning`、`provider_policy.image_generation`、`output.base_dir`、`diagram_types`、`pipeline.steps`、`diagram_spec.required_fields`、`gpt_image_2.model`。
- `safety.strip_secrets` 必须为 true；发现疑似明文 token、API key 或 `.env` 内容时阻断。

验证：

- `go test ./internal/workspace` 通过。
- `go test ./...` 通过。

## 7. 已完成增量：`phase3-002d agent-runtimes-yaml-schema-validator`

范围：

- 新增 `.moyuan/runtimes/agent-runtimes.yaml` 路径索引。
- `workspace validate` 会在文件存在时校验 Native Runtime 配置。
- 校验内容包括 `agent_runtimes.enabled`、`default_runtime`、`session_store`、`output_store`、`runtimes`、`role_runtime_defaults`、`isolation.require_issue_worktree` 和 `require_quality_gate_after_run`。
- Runtime entry 校验覆盖 `type`、`provider`、`enabled`、`command`、`auth.mode`、`provider_env_profile.allowed_env_keys`、`health_check.command`、`invocation/context/tools/session/audit`。
- `audit.capture_diff_before_after` 必须为 true；`provider_env_profile.enabled=false` 时 env key 白名单必须为空；provider 专属 invocation 字段必须为空。

验证：

- `go test ./internal/workspace` 通过。
- `go test ./...` 通过。

## 8. 已完成增量：`phase3-003a visual-render-dry-run-console-action`

范围：

- 在 Console Visual Assets 面板增加 `Dry Run` 操作按钮。
- 前端调用后端 `POST /v1/projects/:project_id/visuals/assets/:asset_id/render`，请求体固定为 `mode=dry_run`。
- UI 展示运行中、完成、阻断和错误状态，并回显 execution id 和 decision。
- 该动作只触发后端 dry-run，不调用真实图像 API，不绕过后端审批和安全开关。

验证：

- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。

## 9. 已完成任务：`phase3-004 runtime-log-diff-viewer`

范围：

- 新增后端 artifact preview：`GET /v1/projects/:project_id/runtime-recoveries/:recovery_id/artifacts`。
- 后端只读取 recovery 记录中归档的 stdout、stderr 和 diff summary，且路径必须位于 `.moyuan/` 下。
- artifact 内容会复用 runtime 输出脱敏逻辑，并按 limit 截断，避免任意文件读取和大日志压垮 Console。
- Console Runtime Recoveries 面板增加 `Artifacts` 操作，可展开 stdout/stderr/diff summary 预览。

验证：

- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。

## 10. Phase 3 收口规则

- 每完成一个 Phase 3 issue，必须同步本实施记录和 issue graph。
- 配置 validator 新增 issue code 时，必须能追溯到 [配置 Schema 规则](../configuration-schema-spec.md)。
- Console 操作流只能调用后端受控 API，不允许在前端直接改变权威状态。
