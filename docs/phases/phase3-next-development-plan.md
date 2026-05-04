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
| P1 | `phase3-003` | `console-operation-actions` | planned | Console 增加受控操作入口和后端 preview/dry-run 对齐 | 高风险动作不能绕过 approval/authz |
| P1 | `phase3-004` | `runtime-log-diff-viewer` | planned | Console 展开 runtime 日志、diff summary 和 resume hint | 失败排查证据链可见 |
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

## 4. Phase 3 收口规则

- 每完成一个 Phase 3 issue，必须同步本实施记录和 issue graph。
- 配置 validator 新增 issue code 时，必须能追溯到 [配置 Schema 规则](../configuration-schema-spec.md)。
- Console 操作流只能调用后端受控 API，不允许在前端直接改变权威状态。
