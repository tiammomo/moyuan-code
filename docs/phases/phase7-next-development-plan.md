# Phase 7 实施记录

状态：completed
责任角色：release_manager + devops_owner + security_owner + provider_owner + frontend_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 7 的实际执行顺序。Phase 7 的入口以 [Phase 7 实现 Issue Graph](./phase7-issue-graph.md) 为准。

## 1. 当前基线

Phase 6 已完成并通过 release readiness：

- Approval record 已支持消费和重放防护。
- Git Provider PR/MR create 已在真实写入路径前消费 approval。
- Deployment 已具备 SSH preview 状态模型，真实 SSH 仍默认阻断。
- Release provider 已具备 preview/publish execution，真实 release provider write 仍默认关闭。
- Provider telemetry 已进入 ops update、refresh、route decision、runtime execution 和 quality gate。
- Console 已具备多视图、schema-aware 必填预检和 release/provider 操作入口。

## 2. Phase 7 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase7-001` | `release-provider-approval-consumption` | completed | release provider 真实 publish 的 approval consumption 和 replay guard | 真实 publish 路径不能重复使用 approval |
| P0 | `phase7-002` | `ssh-executor-guarded-runner` | completed | SSH executor 受控执行边界 | 默认阻断真实 SSH，启用后只执行白名单命令 |
| P1 | `phase7-003` | `post-action-evidence-model` | completed | 发布/部署/烟测/监控/回滚证据链 | 每次操作能查询统一 evidence |
| P1 | `phase7-004` | `runtime-telemetry-feedback-loop` | completed | runtime/quality 结果反哺 provider telemetry | route decision 可读取执行反馈 |
| P2 | `phase7-005` | `console-execution-detail-history` | completed | Console execution detail 和 operation history | 用户能追踪 preview、approval、publish、evidence |

## 3. 执行规划：`phase7-001 release-provider-approval-consumption`

范围：

- Release provider publish 增加真实写入开关语义，不因 `approved=true` 直接视为可远程写入。
- 当真实写入开关开启且 approval 已通过时，publish 必须消费 approval record。
- 已消费 approval 不能再次用于同一或其他 release provider publish。
- 写开关未开启时继续返回 preview-only，不消费 approval。
- execution 中明确记录 `approval_consumed`、`write_enabled` 和 replay guard reason。

非目标：

- 不在本任务中调用 GitHub/Gitee release、tag 或 workflow API。
- 不执行 `git push`、`git tag` 或 workflow dispatch。
- 不改变 API authz middleware 和 Secret Resolver 的既有规则。

验收：

- 缺少 approval 时 publish 仍生成 approval record。
- 写开关关闭时，即使 approval 已批准也返回 preview-only 且不消费 approval。
- 写开关开启时，approval 被消费，重复 publish 使用同一 approval 会被阻断。
- `go test ./internal/release ./internal/approvals ./internal/cli ./internal/api` 通过。
- `go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

## 4. 已完成任务：`phase7-001 release-provider-approval-consumption`

范围：

- `ProviderExecution` 新增 `write_enabled` 和 `approval_consumed`，明确记录 release provider publish 的写开关和审批消费状态。
- `MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE=1` 作为真实 release provider write 的显式开关；默认关闭时继续返回 `RELEASE_PROVIDER_PUBLISH_PREVIEW_ONLY`，且不消费 approval。
- 写开关开启且 approval 已通过时，publish 会先消费 approval，再进入远程 adapter 边界。
- 已消费 approval 再次用于 publish 会被 `approval_not_approved` 阻断。
- 测试覆盖 preview-only 不消费 approval、写开关开启消费 approval、重复使用 approval 被阻断。

非目标：

- 不真实调用 GitHub/Gitee release、tag 或 workflow API。
- 不执行 `git push`、`git tag` 或 workflow dispatch。
- 不把 token、secret 或远程响应写入 execution。

验证：

- `go test ./internal/release ./internal/approvals ./internal/cli ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 5. 执行规划：`phase7-002 ssh-executor-guarded-runner`

范围：

- `ssh_execute` 增加真实执行开关语义，默认未启用时只记录 blocked execution 和 `SSH_EXECUTION_NOT_ENABLED` remote plan。
- 设置 `MOYUAN_ALLOW_SSH_EXECUTE=1` 后，不直接连接远程 SSH，而是先校验 server resource、`auth_ref` 引用和命令 allowlist。
- 通过校验时生成 `SSH_EXECUTION_GUARDED_READY` remote plan，明确记录 `remote_ssh_command_runner_not_enabled`，等待后续真实 runner。
- 不安全命令必须被 allowlist 阻断，并进入 execution step。

非目标：

- 不解析 SSH 私钥或 secret value。
- 不启动真实 SSH session。
- 不执行生产环境真实部署。

验收：

- 默认 `ssh_execute` 仍返回 blocked，并记录 `ssh_real_execution_not_enabled`。
- 开启 `MOYUAN_ALLOW_SSH_EXECUTE=1` 且命令在 allowlist 内时，remote plan 进入 guarded ready。
- 开启写开关但命令不在 allowlist 内时，execution 被阻断并记录 `command_not_allowed`。
- `go test ./internal/deployment ./internal/cli ./internal/api` 通过。
- `go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

## 6. 已完成任务：`phase7-002 ssh-executor-guarded-runner`

范围：

- `Execution` 新增 `remote_exec_enabled`，明确记录本次远程执行开关是否开启。
- `ssh_execute` 默认仍返回 `DEPLOY_EXECUTION_BLOCKED` 和 `SSH_EXECUTION_NOT_ENABLED`。
- `MOYUAN_ALLOW_SSH_EXECUTE=1` 开启后，会生成 guarded remote plan，但仍不真实连接 SSH。
- SSH 命令复用受限 allowlist 和 shell metacharacter 阻断规则；不安全命令进入 blocked step。
- 新增 `deployment.ssh.execution.guarded` release log，用于审计 guarded ready 状态。

非目标：

- 不执行真实远程命令。
- 不读取、解析或打印 secret value。
- 不改变 production real execution 默认阻断策略。

验证：

- `go test ./internal/deployment ./internal/cli ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 7. 执行规划：`phase7-003 post-action-evidence-model`

范围：

- 新增统一 evidence record，用于记录 release provider execution、deployment execution、smoke、monitor、rollback 等后续动作的证据链。
- Release provider execution 和 deployment execution 完成时自动写入 evidence。
- CLI 增加 `moyuan evidence list/show`。
- API 增加 `GET /v1/projects/:project_id/evidence` 和 `GET /v1/projects/:project_id/evidence/:evidence_id`。
- evidence record 写入 audit log，后续 Console 和自修复可按 parent/subject 检索。

非目标：

- 不替代原始 execution JSON；evidence 只做统一索引和摘要。
- 不读取任意文件内容，只记录 artifact reference。
- 不接入外部 observability vendor。

验收：

- release provider execution 完成后能查到 evidence。
- deployment execution 完成后能查到 evidence。
- CLI/API 能按 parent type/id 查询 evidence。
- `go test ./internal/evidence ./internal/release ./internal/deployment ./internal/cli ./internal/api` 通过。
- `go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

## 8. 已完成任务：`phase7-003 post-action-evidence-model`

范围：

- 新增 `internal/evidence`，支持 `Add`、`List` 和 `Load`。
- Evidence 存储位置为 `.moyuan/lifecycle/evidence/`，并追加 `evidence.jsonl`。
- Release provider execution 和 deployment execution 都会写入 evidence artifact reference。
- CLI 增加 `moyuan evidence list` 和 `moyuan evidence show`。
- API 增加 evidence list/show 查询入口。
- 审计日志新增 `evidence.recorded`。

非目标：

- 不复制 execution 原文内容。
- 不记录 secret value 或远程响应正文。
- 不改变 release/deploy 原有执行状态。

验证：

- `go test ./internal/evidence ./internal/release ./internal/deployment ./internal/cli ./internal/api` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 9. 执行规划：`phase7-004 runtime-telemetry-feedback-loop`

范围：

- Provider telemetry record 增加 runtime、model、run、issue、quality report、runtime status、quality status 和 route decision 上下文。
- Native Runtime 完成后写入 `runtime_execution` telemetry，并更新 provider health 与 usage request 计数。
- Orchestrator quality gate 完成后写入 `quality_gate` telemetry，质量通过可恢复 provider health，质量失败会降级。
- Provider route decision 写入 `provider_route` telemetry，用于解释后续调度看到的 health/quota/cost signals。

非目标：

- 不外呼模型服务商账单、额度或真实质量采样接口。
- 不把 prompt、secret、stdout、stderr 或 diff 原文写入 provider telemetry。
- 不因 telemetry 写入失败阻断 runtime 主流程。

验收：

- runtime execution 能产生 `runtime_execution` telemetry。
- quality gate 能产生 `quality_gate` telemetry。
- route decision 能产生 `provider_route` telemetry。
- degraded provider 不被直接阻断，但 route signal 必须可见。
- `go test ./internal/providers ./internal/runtime ./internal/orchestrator` 通过。
- `go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check` 通过。

## 10. 已完成任务：`phase7-004 runtime-telemetry-feedback-loop`

范围：

- `TelemetryRecord` 增加 execution feedback 上下文字段。
- 新增 `RecordExecutionFeedback` 和 `RecordQualityFeedback`，统一写入 provider telemetry。
- Runtime native CLI 执行结束后记录 `runtime_execution` 反馈。
- Orchestrator quality gate 结束后记录 `quality_gate` 反馈。
- Provider route decision 记录 `provider_route` 反馈。
- 反馈会更新 provider health 和 usage request，route decision 可读取降级后的 health signal。

验证：

- `go test ./internal/providers ./internal/runtime ./internal/orchestrator` 通过。
- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过。
- `git diff --check` 通过。

## 11. 执行规划：`phase7-005 console-execution-detail-history`

范围：

- Release provider execution 增加 list API，支持 Console 拉取近期 preview/publish 记录。
- Console snapshot 增加 release provider executions、evidence 和 operation history。
- Deployments 工作区增加 operation history 列表和 execution detail 面板。
- Operation detail 展示 status、decision、primary ref、secondary ref、evidence ids、reasons 和 metadata。

非目标：

- 不在 Console 里判定执行成功，仍以后端 execution/evidence 为事实源。
- 不展示 secret、stdout/stderr 原文或远程响应正文。
- 不在本任务中实现真实 release provider write 或真实 SSH runner。

验收：

- `GET /v1/projects/:project_id/release-provider-executions` 可查询 release provider execution 列表。
- Console 能展示 deployment execution、release provider execution、visual render 和 evidence 关联历史。
- `go test ./internal/release ./internal/api` 通过。
- `npm run typecheck` 通过。

## 12. 已完成任务：`phase7-005 console-execution-detail-history`

范围：

- `internal/release` 增加 `ListProviderExecutions`。
- API 增加 `GET /v1/projects/:project_id/release-provider-executions`。
- Console live snapshot 拉取 release provider execution list 和 evidence list。
- Console 新增 operation history/detail 区域，展示 deployment、release provider、visual render 和 evidence 关联。
- Demo snapshot 补齐 release provider execution、evidence 和 operation history 示例。

验证：

- `go test ./internal/release ./internal/api` 通过。
- `npm run typecheck` 通过。

## 13. 验证要求

每完成一个 Phase 7 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
