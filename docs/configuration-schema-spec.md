# 配置 Schema 规则

本文定义 `.moyuan/` 配置文件的字段必填、可选、可为空和必须为空规则。完整 YAML 示例由 [完整配置方案](./configuration-guide.md) 维护；本文只维护 schema 约束。

## 1. 规则定义

| 标记 | 含义 |
| --- | --- |
| `required` | 字段必须出现，且不能为 `null` |
| `optional` | 字段可以不出现 |
| `nullable` | 字段可以出现且值为 `null` |
| `must_be_null_when` | 满足某条件时字段必须为 `null` |
| `must_be_empty_when` | 满足某条件时字段必须为空数组或空对象 |
| `default` | 字段缺失时系统使用默认值 |
| `conditional_required` | 满足某条件时必填 |

通用规则：

- 所有配置文件必须有 `schema_version`。
- `schema_version` 必须为整数，MVP 只支持 `1`。
- 所有 secret 只能保存引用，不能保存明文。
- `null` 只表示“显式无值”，不等同于空字符串。
- 空字符串不得用于表示未配置；未配置应使用 `null` 或省略字段。

## 2. project.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `project.id` | required | 无 | 项目唯一 id |
| `project.name` | required | 无 | 项目名称 |
| `project.root` | required | `.` | 项目根路径 |
| `project.type` | required | `single-repo` | `single-repo` 或后续 `multi-repo` |
| `project.description` | optional, nullable | null | 项目描述 |
| `stack.languages` | optional | `[]` | 可为空数组 |
| `stack.frameworks` | optional | `[]` | 可为空数组 |
| `stack.package_managers` | optional | `[]` | 可为空数组 |
| `stack.build_commands` | optional | `[]` | 可为空数组 |
| `stack.test_commands` | optional | `[]` | 可为空数组 |
| `stack.lint_commands` | optional | `[]` | 可为空数组 |
| `workspace.protected_paths` | required | 无 | 至少包含 `.env`、`.env.*` |
| `workspace.writable_paths` | required | 无 | 至少一个可写路径 |

必须为空：

- `project.root` 不允许为空字符串。
- `workspace.protected_paths` 不允许为空数组。
- `workspace.writable_paths` 不允许为空数组。

## 3. repository.yaml

通用字段：

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `repository.source.type` | required | 无 | `local_path` 或 `remote_git` |
| `repository.source.provider` | required | 无 | `local`、`github`、`gitee`、`gitlab`、`generic_git` |
| `repository.source.url` | conditional_required | null | `remote_git` 必填 |
| `repository.source.local_path` | conditional_required | null | `local_path` 必填 |
| `repository.source.clone_path` | optional, nullable | null | 为空时自动生成 |
| `repository.default_remote` | optional | `origin` | 本地仓库无 remote 时可为 null |
| `repository.default_branch` | optional, nullable | null | 为空时自动探测 |

必须为空：

| 条件 | 字段 | 规则 |
| --- | --- | --- |
| `source.type = local_path` | `repository.source.url` | must_be_null_when |
| `source.type = remote_git` | `repository.source.local_path` | must_be_null_when |
| `source.provider != github` | `repository.github` | must_be_null_when |

Git 策略：

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `git.branch_policy.mode` | required | `task_branch` | MVP 固定 `task_branch` |
| `git.branch_policy.naming` | required | 无 | 任务分支模板 |
| `git.branch_policy.base` | optional | `default_branch` | 分支基线 |
| `git.branch_policy.sync_before_run` | optional | `true` | 执行前同步 |
| `git.branch_policy.require_clean_worktree` | optional | `true` | 保护用户改动 |
| `git.branch_policy.allow_auto_commit` | optional | `false` | 默认不自动 commit |
| `git.branch_policy.allow_auto_push` | optional | `false` | 默认不自动 push |
| `git.branch_policy.allow_auto_pr` | optional | `false` | 默认不自动 PR |

GitHub 字段规则：

- 由 [GitHub 接入配置](./github-integration.md) 维护。
- `repository.github` 只有在 `repository.source.provider = github` 时允许出现。

## 4. policies/access.yaml

`policies/access.yaml` 只保存项目级访问策略和角色映射，不保存用户密码、API Token 明文、session secret 或云凭证明文。用户、组织、会话、Token 和成员关系的控制面对象见 [平台用户与访问控制主线](./mainlines/platform-user-access.md)。

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `access.mode` | required | `local_single_user` | `local_single_user` 或 `team_server` |
| `access.local_owner_id` | conditional_required | null | `local_single_user` 必填 |
| `access.organization_id` | conditional_required | null | `team_server` 必填 |
| `access.project_roles` | required | 无 | 项目角色到能力的映射 |
| `access.approval_policy` | required | 无 | 高风险操作审批入口 |
| `access.audit.enabled` | required | `true` | 必须启用 |

必须为空：

| 条件 | 字段 | 规则 |
| --- | --- | --- |
| `access.mode = local_single_user` | `access.organization_id` | must_be_null_when |
| `access.mode = team_server` | `access.local_owner_id` | must_be_null_when |
| 任意条件 | `password`、`api_token`、`session_secret` | must_be_null_when |

## 5. models/providers.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `model_provider_management.enabled` | required | `true` | 是否启用 provider 管理 |
| `model_provider_management.registry_path` | required | 无 | provider 快照路径 |
| `model_provider_management.usage_path` | required | 无 | 用量日志路径 |
| `accounts` | required | 无 | 至少一个账号 |
| `providers` | required | 无 | 至少一个 provider |
| `quotas.default` | required | 无 | 默认预算和限流 |
| `health_checks.enabled` | required | `true` | 是否启用健康检查 |
| `security.forbid_plaintext_api_key` | required | `true` | 必须为 true |

Account 字段：

| 字段 | 规则 | 说明 |
| --- | --- | --- |
| `vendor` | required | 厂商 |
| `api_type` | required | API 类型 |
| `base_url` | conditional_required | API 型账号必填；CLI 型账号可为空 |
| `auth_ref` | required | 必须是 secret 或 env 引用 |
| `enabled` | required | 是否启用 |
| `data_policy` | required | 数据策略 |

必须为空：

| 条件 | 字段 | 规则 |
| --- | --- | --- |
| `api_type = claude-code` | `base_url` | must_be_null_when |
| `api_type = codex` 且使用本地 CLI | `base_url` | must_be_null_when |
| `vendor != third_party` | `upstream_vendor` | must_be_null_when |

Provider 字段：

| 字段 | 规则 | 说明 |
| --- | --- | --- |
| `type` | required | `llm-api`、`image-generation-api`、`codex`、`claude-code` 等 |
| `adapter` | conditional_required | API provider 必填 |
| `account` | required | 引用 accounts |
| `enabled` | required | 是否启用 |
| `models` | conditional_required | `llm-api` 和 `image-generation-api` 必填 |
| `capabilities` | conditional_required | CLI provider 必填 |

必须为空：

| 条件 | 字段 | 规则 |
| --- | --- | --- |
| `type = codex` | `models` | must_be_empty_when |
| `type = claude-code` | `models` | must_be_empty_when |
| `type = image-generation-api` | `capabilities.code_edit` | must_be_null_when |

## 6. models/routing.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `policies` | required | 无 | 至少包含 planning、coding、review |
| `policies.*.primary.provider` | required | 无 | 必须引用 provider |
| `policies.*.primary.model` | conditional_required | null | API provider 必填；CLI provider 可为 `default` |
| `policies.*.fallback` | optional | `[]` | 可为空数组 |
| `policies.*.constraints` | optional, nullable | null | 可为空 |

必须为空：

- 第三方安全文本策略中 `allow_code_context` 必须为 `false`。
- 第三方安全文本策略中 `allow_project_memory` 必须为 `false`。

## 7. visuals/architecture-visuals.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `architecture_visuals.enabled` | required | `true` | 是否启用 |
| `provider_policy.diagram_planning` | required | 无 | 文本规划策略 |
| `provider_policy.image_generation` | required | 无 | 图像生成策略 |
| `output.base_dir` | required | `.moyuan/visuals` | 输出目录 |
| `diagram_types` | required | 无 | 至少一种图类型 |
| `pipeline.steps` | required | 无 | 生成流程 |
| `diagram_spec.required_fields` | required | 无 | 图 spec 必填字段 |
| `gpt_image_2.model` | required | `gpt-image-2` | 图像模型 |
| `safety.strip_secrets` | required | `true` | 必须为 true |

必须为空：

- `gpt_image_2.prompt_template` 不允许为空字符串；可省略或填路径。
- 图像 prompt 中必须不包含 secret、私网 IP、token、`.env` 明文。

## 8. runtimes/agent-runtimes.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `agent_runtimes.enabled` | required | `true` | 是否启用 Runtime |
| `agent_runtimes.default_runtime` | required | 无 | 默认 Runtime id |
| `agent_runtimes.session_store` | required | 无 | 会话目录 |
| `agent_runtimes.output_store` | required | 无 | 输出目录 |
| `agent_runtimes.runtimes` | required | 无 | 至少一个 runtime |
| `agent_runtimes.routing.task_modes.frontend` | required | 无 | 默认绑定 Claude CLI |
| `agent_runtimes.routing.task_modes.backend` | required | 无 | 默认绑定 Codex CLI |
| `agent_runtimes.role_runtime_defaults.frontend` | required | `claude_cli` | 前端默认 Runtime |
| `agent_runtimes.role_runtime_defaults.backend` | required | `codex_cli` | 后端默认 Runtime |
| `agent_runtimes.role_runtime_defaults.backend_tuning` | required | `codex_cli` | 后端调优默认 Runtime |
| `agent_runtimes.isolation.require_issue_worktree` | required | `true` | 必须为 true |
| `agent_runtimes.require_quality_gate_after_run` | required | `true` | 必须为 true |

Runtime 字段：

| 字段 | 规则 | 说明 |
| --- | --- | --- |
| `type` | required | MVP 为 `native_agent_cli` |
| `provider` | required | `claude_code` 或 `codex` |
| `enabled` | required | 是否启用 |
| `command` | required | CLI 命令 |
| `auth.mode` | required | `env` 或 `local_cli_login` |
| `auth.auth_ref` | optional, nullable | 本地已登录可为空 |
| `health_check.command` | required | 健康检查命令 |
| `invocation` | required | 调用参数 |
| `context` | required | 上下文注入策略 |
| `tools` | required | 工具策略 |
| `session.enable_resume` | required | 是否支持恢复 |
| `audit.capture_diff_before_after` | required | 必须为 true |

必须为空：

| 条件 | 字段 | 规则 |
| --- | --- | --- |
| `auth.mode = local_cli_login` 且依赖本地登录 | `auth.auth_ref` | nullable |
| `provider = claude_code` | `invocation.ask` | must_be_null_when |
| `provider = codex` | `invocation.one_shot` | must_be_null_when |

## 9. agents/roles.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `roles` | required | 无 | 至少一个 role |
| `roles.*.default_model_policy` | required | 无 | 模型策略 |
| `roles.*.skills` | optional | `[]` | 可为空数组 |
| `roles.*.memory_scopes` | optional | `[]` | 可为空数组 |
| `roles.*.tools` | required | 无 | 可用工具 |

必须为空：

- `tools` 不允许为空数组。
- `default_model_policy` 不允许为空字符串。

## 10. agents/teams.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `teams` | required | 无 | 至少一个 team |
| `teams.*.planners` | optional | `[]` | 可为空 |
| `teams.*.implementers` | optional | `[]` | 可为空 |
| `teams.*.verifiers` | optional | `[]` | 可为空 |

必须为空：

- 一个可执行 team 不能同时让 `planners`、`implementers`、`verifiers` 都为空。
- release team 可以让 `implementers` 为空，但 `verifiers` 不应为空。

## 11. agents/subagents.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `subagents.enabled` | required | `true` | 是否启用显式 Subagent |
| `subagents.max_parallel_subagents` | required | `1` | 至少为 1 |
| `subagents.require_parent` | required | `true` | 必须为 true |
| `subagents.require_output_contract` | required | `true` | 必须为 true |
| `subagents.require_skill_compatibility_check` | required | `true` | 必须为 true |
| `subagents.lifecycle` | required | 无 | 状态集合和失败恢复入口 |
| `subagents.allowed_parent_types` | required | 无 | epic、issue、run、repair_attempt、release、deployment、memory_job |

必须为空：

- `allowed_parent_types` 不允许为空数组。
- `max_parallel_subagents` 不能为 null。
- `require_parent`、`require_output_contract`、`require_skill_compatibility_check` 必须为 true。

## 12. skills/registry.yaml、skills/enabled.yaml、skills/bindings.yaml

Skill Registry 字段：

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `skills` | required | 无 | 可为空数组，但 registry 文件必须存在 |
| `skills.*.id` | required | 无 | skill id |
| `skills.*.version` | required | 无 | semver 或内部版本 |
| `skills.*.source` | required | 无 | builtin、project、organization、marketplace、manual |
| `skills.*.supported_roles` | required | 无 | 至少一个 role |
| `skills.*.task_types` | required | 无 | 至少一个任务类型 |
| `skills.*.required_tools` | optional | `[]` | 可为空 |
| `skills.*.memory_scopes` | optional | `[]` | 可为空 |
| `skills.*.risk_level` | required | `low` | low、medium、high |
| `skills.*.enabled` | required | `false` | 默认不自动启用外部 skill |

Skill Binding 字段：

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `bindings` | optional | `[]` | 可为空数组 |
| `bindings.*.skill_id` | required | 无 | 必须引用 registry |
| `bindings.*.target_type` | required | 无 | project、role、issue、subagent |
| `bindings.*.target_id` | required | 无 | 绑定目标 |
| `bindings.*.priority` | optional | `100` | 越小优先级越高 |
| `bindings.*.status` | required | `enabled` | candidate、enabled、disabled、deprecated |

必须为空：

- 外部 marketplace skill 未审批时，`enabled` 必须为 false。
- `supported_roles` 和 `task_types` 不允许为空数组。
- `target_type = subagent` 时，`target_id` 必须引用已存在 subagent。

## 13. policies/permissions.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `permissions.filesystem.writable_paths` | required | 无 | 可写范围 |
| `permissions.filesystem.protected_paths` | required | 无 | 保护范围 |
| `permissions.commands.allow` | required | 无 | 允许命令 |
| `permissions.commands.require_approval` | optional | `[]` | 需审批命令 |
| `permissions.commands.deny` | optional | `[]` | 禁止命令 |
| `permissions.network.enabled` | required | `false` | 是否允许网络 |

必须为空：

- `protected_paths` 不允许为空数组。
- 生产部署启用时，`commands.require_approval` 不能为空。
- secret 访问不能出现在 `commands.allow` 中。

## 14. policies/secrets.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `secrets` | optional | `{}` | 无密钥场景可为空对象 |
| `secrets.*.type` | required | 无 | token、ssh_key、private_key 等 |
| `secrets.*.ref` | required | 无 | env 或 secret manager 引用 |
| `secrets.*.usage` | required | 无 | 用途 |

必须为空：

- `ref` 不得为明文 secret。
- `usage` 不允许为空数组。
- 不需要投产、远程私有仓库、registry、第三方 API 时，`secrets` 可以为空对象。

## 15. policies/orchestration.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `orchestration.enabled` | required | `true` | 是否启用 |
| `orchestration.issue_graph` | required | `true` | 是否生成 issue graph |
| `orchestration.auto_parallelism` | required | `true` | 是否自动并发 |
| `orchestration.max_parallel_issues` | required | `1` | 至少为 1 |
| `orchestration.max_parallel_subagents` | required | `1` | 至少为 1 |
| `orchestration.concurrency_guards` | required | 无 | 并发保护 |
| `orchestration.waiting_policy` | required | 无 | 编排等待策略 |
| `orchestration.merge_gate` | required | 无 | 合入门禁 |

必须为空：

- `max_parallel_issues` 不能为 null。
- `max_parallel_subagents` 不能为 null。
- `merge_gate` 不能为空对象。
- `waiting_policy.queues` 不能缺少 `blocked_queue`、`ready_queue`、`running_queue`、`review_queue`。
- `waiting_policy.frontend_runtime` 必须引用 `claude_cli`。
- `waiting_policy.backend_runtime` 必须引用 `codex_cli`。

## 16. policies/code-quality.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `quality.enabled` | required | `true` | 是否启用 |
| `quality.required_for_all_code_tasks` | required | `true` | 必须为 true |
| `quality.max_rework_rounds` | required | `3` | 返工上限 |
| `gates` | required | 无 | 至少包含 runnable、test_gap |
| `self_repair.enabled` | required | `true` | 是否启用运行反馈和自我修复 |
| `self_repair.mode` | required | `candidate_only` | observe_only、candidate_only、issue_only、auto_repair_low_risk |
| `self_repair.max_attempts_per_bug` | required | `2` | 单个 bug 自动修复上限 |
| `self_repair.require_regression_test` | required | `true` | 自动修复必须补回归测试 |
| `self_repair.require_approval_for` | required | 无 | 高风险修复审批触发器 |

必须为空：

- `gates.runnable` 不允许为空。
- `gates.test_gap` 不允许为空。
- `quality.max_rework_rounds` 不能为 null。
- `self_repair.mode` 不能为 null。
- `self_repair.require_approval_for` 不能为空数组。

## 17. policies/comprehension.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `comprehension.enabled` | required | `true` | 是否启用 |
| `run_after_project_add` | required | `true` | 必须为 true |
| `run_after_remote_pull` | required | `true` | 必须为 true |
| `mode.initial` | required | `full` | 首次理解 |
| `mode.after_pull` | required | `incremental` | 拉取后理解 |

必须为空：

- `mode.initial` 不允许为空。
- `mode.after_pull` 不允许为空。

## 18. policies/memory.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `memory.enabled` | required | `true` | 是否启用 |
| `record_gate` | required | 无 | 记录判断 |
| `extraction` | required | 无 | 抽取策略 |
| `staging` | required | 无 | 暂存策略 |
| `retrieval` | required | 无 | 检索策略 |
| `compact.enabled` | required | `true` | 自动 compact |
| `maintenance.enabled` | required | `true` | 维护策略 |

必须为空：

- `record_gate.threshold` 不能为 null。
- `retrieval.top_k` 不能为 null。
- `compact.triggers` 不能为空对象。

## 19. policies/logging.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `logging.enabled` | required | `true` | 是否启用 |
| `logging.format` | required | `jsonl` | MVP 固定 JSONL |
| `logging.storage.base_dir` | required | `.moyuan/logs` | 日志目录 |
| `logging.streams` | required | 无 | 至少 run、audit、error |
| `logging.redaction.enabled` | required | `true` | 必须为 true |

必须为空：

- `redaction.redact_patterns` 不允许为空数组。
- `streams.audit` 不允许为空。
- `streams.error` 不允许为空。

## 20. policies/server-resources.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `server_resources.enabled` | required | `false` | 是否启用 |
| `registry` | conditional_required | 无 | 启用时必填 |
| `categories` | conditional_required | 无 | 启用时必填 |
| `hosts` | optional | `[]` | 可为空数组 |
| `groups` | optional | `{}` | 可为空对象 |
| `access_policy` | conditional_required | 无 | 启用时必填 |
| `inventory_checks` | conditional_required | 无 | 启用时必填 |

必须为空：

- 不启用投产时，`hosts` 可以为空数组。
- `category = production` 的 host 不能缺少 `owner`、`auth_ref`、`lifecycle.expires_at`。
- `cloud.enabled = false` 时，`cloud.account`、`cloud.instance_id` 可以为 null。

## 21. policies/environments.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `environments` | optional | `{}` | 不启用部署时可为空 |
| `environments.*.resource_group` | conditional_required | 无 | 部署环境必填 |
| `environments.*.approval_required` | required | `true` | 生产必须 true |
| `environments.*.artifact` | conditional_required | 无 | 自动部署必填 |
| `environments.*.deploy` | conditional_required | 无 | 自动部署必填 |
| `environments.*.healthcheck` | conditional_required | 无 | 自动部署必填 |
| `environments.*.smoke_tests` | conditional_required | 无 | 生产必填 |
| `environments.*.rollback` | conditional_required | 无 | 生产必填 |

必须为空：

- 不启用 deployment 时，`environments` 可以为空对象。
- `production.approval_required` 必须为 true。
- `production.rollback` 不允许为空。

## 22. policies/release.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `release.auto_suggest` | required | `true` | 是否自动建议 |
| `release.mode` | required | `branch_only` | `branch_only` 或 `deploy_to_environment` |
| `release.remote_providers` | required | 无 | 至少一个远程 provider |
| `release.default_batch` | required | 无 | 发布批次建议 |
| `release.gates` | required | 无 | 发布门禁 |
| `release.git` | required | 无 | release branch 和 tag 策略 |
| `release.deployment` | conditional_required | null | `deploy_to_environment` 时必填 |

必须为空：

- `mode = branch_only` 时，`release.deployment.enabled` 必须为 false 或 `release.deployment` 为 null。
- `mode = deploy_to_environment` 时，`release.deployment` 不能为 null。

## 23. policies/budget.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `budget.max_parallel_issues` | required | `1` | issue 并发 |
| `budget.max_parallel_model_calls` | required | `1` | 模型并发 |
| `budget.max_daily_model_cost_usd` | optional, nullable | null | null 表示不按金额限制 |
| `budget.max_task_runtime_minutes` | required | `60` | 单任务超时 |
| `budget.fallback_to_low_cost_model` | required | `true` | 是否降级低成本模型 |

必须为空：

- 无预算限制时，`max_daily_model_cost_usd` 必须为 null，不使用 0 表示无限制。

## 24. 配置校验顺序

系统校验配置时按以下顺序：

1. `schema_version`。
2. 文件是否存在。
3. 必填字段。
4. null 和空值规则。
5. 条件必填。
6. 必须为空规则。
7. 引用完整性。
8. 权限和敏感信息规则。
9. provider/runtime/secret/resource 可用性。

## 25. MVP 最小配置

必须存在且通过 schema 校验：

- `project.yaml`
- `repository.yaml`
- `models/providers.yaml`
- `models/routing.yaml`
- `runtimes/agent-runtimes.yaml`
- `agents/roles.yaml`
- `agents/teams.yaml`
- `agents/subagents.yaml`
- `skills/registry.yaml`
- `skills/bindings.yaml`
- `policies/access.yaml`
- `policies/permissions.yaml`
- `policies/code-quality.yaml`
- `policies/orchestration.yaml`
- `policies/comprehension.yaml`
- `policies/logging.yaml`

可以为空或延后：

- `policies/secrets.yaml`，仅本地公开仓库且无外部 API 时可为空对象。
- `policies/server-resources.yaml`，不启用部署时 `hosts` 可为空。
- `policies/environments.yaml`，不启用部署时可为空对象。
- `skills/enabled.yaml`，skills 未启用时可为空数组，但文件仍应存在。

## 26. 进入实现前必须补的机器校验

本文是人类可读 schema 规则。进入实现前必须转换为：

- JSON Schema，或
- Zod schema，或
- TypeScript 类型 + runtime validator。

机器校验必须能输出：

- 缺失字段。
- 不允许为 null。
- 条件必填未满足。
- 条件必须为空未满足。
- secret 明文泄露。
- 引用对象不存在。
