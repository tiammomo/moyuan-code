# 配置 Schema 规则

本文定义 `.moyuan/` 配置文件的字段必填、可选、可为空和必须为空规则。配置分层、闭环和关键片段由 [配置方案](./configuration-guide.md) 维护；本文只维护 schema 约束。

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

## 2. 配置域索引

本文是配置字段规则的唯一详细表。为避免实现阶段查找困难，配置域按以下顺序维护：

| 配置域 | 章节 | 配置文件 |
| --- | --- | --- |
| Project | [3](#3-projectyaml) | `project.yaml` |
| Repository / Git | [4](#4-repositoryyaml) | `repository.yaml` |
| Access | [5](#5-policiesaccessyaml) | `policies/access.yaml` |
| Provider Registry | [6](#6-modelsprovidersyaml) | `models/providers.yaml` |
| Provider Routing | [7](#7-modelsroutingyaml) | `models/routing.yaml` |
| Visuals | [8](#8-visualsarchitecture-visualsyaml) | `visuals/architecture-visuals.yaml` |
| Agent Runtime | [9](#9-runtimesagent-runtimesyaml) | `runtimes/agent-runtimes.yaml` |
| Agent Role / Team / Subagent | [10-12](#10-agentsrolesyaml) | `agents/*.yaml` |
| Skill Registry | [13](#13-skillsregistryyamlskillsenabledyamlskillsbindingsyaml) | `skills/*.yaml` |
| Permission / Secret | [14-15](#14-policiespermissionsyaml) | `policies/permissions.yaml`、`policies/secrets.yaml` |
| Orchestration / Quality / Comprehension | [16-18](#16-policiesorchestrationyaml) | `policies/orchestration.yaml`、`policies/code-quality.yaml`、`policies/comprehension.yaml` |
| Memory / Logging | [19-20](#19-policiesmemoryyaml) | `policies/memory.yaml`、`policies/logging.yaml` |
| Server / Environment / Release | [21-23](#21-policiesserver-resourcesyaml) | `policies/server-resources.yaml`、`policies/environments.yaml`、`policies/release.yaml` |
| Budget / Engineering | [24-25](#24-policiesbudgetyaml) | `policies/budget.yaml`、`policies/engineering.yaml` |
| Validation / MVP | [26-28](#26-配置校验顺序) | 校验顺序、最小配置、机器校验 |

配置分层和样例只在 [配置方案](./configuration-guide.md) 展开，`.moyuan/` 目录位置只在 [项目工作空间规范](./project-workspace-spec.md) 展开。

## 3. project.yaml

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

## 4. repository.yaml

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
| `source.type = local_path` | `repository.provider_config` | must_be_null_when |
| `source.type = remote_git` | `repository.provider_config` | conditional_required |

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
| `git.commit_policy.enabled` | required | `true` | 是否启用 commit 规范 |
| `git.commit_policy.format` | required | `conventional_commits` | MVP 固定 |
| `git.commit_policy.require_issue_ref` | required | `true` | 必须关联 issue |
| `git.commit_policy.require_quality_ref` | required | `true` | 必须关联质量报告 |

Provider 专项字段规则：

- 由 [Git Provider 接入配置](./git-provider-integration.md) 维护。
- `repository.provider_config` 只有在 `repository.source.type = remote_git` 时允许出现。
- `repository.github`、`repository.gitee`、`repository.gitlab` 这类旧式 provider 专项字段进入实现时应迁移到 `repository.provider_config`。

## 5. policies/access.yaml

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

## 6. models/providers.yaml

Beta 当前实现先写入 `.moyuan/models/providers.json` 作为运行期 registry 快照，字段应与本节目标 schema 对齐。进入 schema validator 阶段后，`providers.yaml` 作为用户可维护配置，`providers.json` 作为运行审计快照。

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
| `runtime_id` | optional, nullable | 绑定 Native Runtime 时填写，例如 `claude_cli` |
| `allowed_use_cases` | optional | 可为空；代码 provider 建议声明 `frontend`、`backend`、`review` 等 |
| `models` | optional | 绑定 API 或兼容 API 的 CLI profile 时可填写模型 id |

必须为空：

| 条件 | 字段 | 规则 |
| --- | --- | --- |
| `api_type = claude-code` | `base_url` | must_be_null_when |
| `api_type = codex` 且使用本地 CLI | `base_url` | must_be_null_when |
| `vendor != third_party` | `upstream_vendor` | must_be_null_when |

Native Runtime profile 特例：

- `api_type = anthropic-compatible` 且 `runtime_id = claude_cli` 时，`base_url` 必填，`auth_ref` 必须引用环境变量或 secret manager，`models` 至少包含一个模型 id。
- `runtime_id = claude_cli` 的 MiniMax profile 推荐模型 id 为 `MiniMax-M2.7`。
- `runtime_id = claude_cli` 且需要处理前端代码时，`allowed_use_cases` 必须包含 `frontend`，并显式声明数据策略是否允许代码上下文和项目 memory。
- 运行期只允许把 `auth_ref` 解析成子进程环境变量；`.moyuan/models/providers.json`、`.moyuan/runtime/*-native.json` 和日志中必须不出现 token 值。

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

## 7. models/routing.yaml

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

## 8. visuals/architecture-visuals.yaml

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

## 9. runtimes/agent-runtimes.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `agent_runtimes.enabled` | required | `true` | 是否启用 Runtime |
| `agent_runtimes.default_runtime` | required | 无 | 默认 Runtime id |
| `agent_runtimes.session_store` | required | 无 | 会话目录 |
| `agent_runtimes.output_store` | required | 无 | 输出目录 |
| `agent_runtimes.runtimes` | required | 无 | 至少一个 runtime |
| `agent_runtimes.routing.task_modes.frontend` | required | 无 | 前端 Runtime 选择策略，至少允许 Claude CLI 或 Codex CLI 之一 |
| `agent_runtimes.routing.task_modes.backend` | required | 无 | 默认绑定 Codex CLI |
| `agent_runtimes.role_runtime_defaults.frontend` | required | `claude_cli` | 前端复杂 UI 首版默认 Runtime，可按 issue 策略改用 Codex CLI |
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
| `provider_env_profile.enabled` | optional | 是否允许从 Provider Registry 注入运行环境 |
| `provider_env_profile.allowed_env_keys` | conditional_required | 允许注入的环境变量白名单 |
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
| `provider_env_profile.enabled = false` | `provider_env_profile.allowed_env_keys` | must_be_empty_when |
| `provider = claude_code` | `invocation.ask` | must_be_null_when |
| `provider = codex` | `invocation.one_shot` | must_be_null_when |

默认 provider env profile 白名单：

- `claude_cli`：`ANTHROPIC_BASE_URL`、`ANTHROPIC_AUTH_TOKEN`、`ANTHROPIC_MODEL`、`ANTHROPIC_DEFAULT_SONNET_MODEL`、`ANTHROPIC_DEFAULT_OPUS_MODEL`、`ANTHROPIC_DEFAULT_HAIKU_MODEL`、`API_TIMEOUT_MS`、`CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`。
- `codex_cli`：`OPENAI_BASE_URL`、`OPENAI_API_KEY`、`OPENAI_MODEL`。

## 10. agents/roles.yaml

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

## 11. agents/teams.yaml

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

## 12. agents/subagents.yaml

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

## 13. skills/registry.yaml、skills/enabled.yaml、skills/bindings.yaml

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

## 14. policies/permissions.yaml

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

## 15. policies/secrets.yaml

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

## 16. policies/orchestration.yaml

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
| `orchestration.issue_spec.required_fields` | required | 无 | Issue 必填字段 |

必须为空：

- `max_parallel_issues` 不能为 null。
- `max_parallel_subagents` 不能为 null。
- `merge_gate` 不能为空对象。
- `waiting_policy.queues` 不能缺少 `blocked_queue`、`ready_queue`、`running_queue`、`review_queue`。
- `waiting_policy.frontend_runtime` 必须引用已启用的前端 Runtime 策略，允许 `claude_cli`、`codex_cli` 或二者组成的候选列表。
- `waiting_policy.backend_runtime` 必须引用 `codex_cli`。
- `issue_spec.required_fields` 不能缺少 `acceptance_criteria`、`test_plan`、`write_scopes`、`rollback_or_fix_plan`。

## 17. policies/code-quality.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `quality.enabled` | required | `true` | 是否启用 |
| `quality.required_for_all_code_tasks` | required | `true` | 必须为 true |
| `quality.max_rework_rounds` | required | `3` | 返工上限 |
| `gates` | required | 无 | 至少包含 runnable、test_gap、coverage |
| `gates.coverage.enabled` | required | `true` | 是否启用覆盖率门禁 |
| `gates.coverage.thresholds.line` | required | `80` | 行覆盖率 |
| `gates.coverage.thresholds.branch` | required | `70` | 分支覆盖率 |
| `gates.coverage.thresholds.changed_files` | required | `85` | 变更文件覆盖率 |
| `coverage.exemptions` | optional | `[]` | 覆盖率豁免，必须可审计 |
| `self_repair.enabled` | required | `true` | 是否启用运行反馈和自我修复 |
| `self_repair.mode` | required | `candidate_only` | observe_only、candidate_only、issue_only、auto_repair_low_risk |
| `self_repair.max_attempts_per_bug` | required | `2` | 单个 bug 自动修复上限 |
| `self_repair.require_regression_test` | required | `true` | 自动修复必须补回归测试 |
| `self_repair.require_approval_for` | required | 无 | 高风险修复审批触发器 |

必须为空：

- `gates.runnable` 不允许为空。
- `gates.test_gap` 不允许为空。
- `gates.coverage` 不允许为空。
- `quality.max_rework_rounds` 不能为 null。
- `self_repair.mode` 不能为 null。
- `self_repair.require_approval_for` 不能为空数组。
- 覆盖率低于阈值时，`coverage.exemptions.*.approval_id` 条件必填。

## 18. policies/comprehension.yaml

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

## 19. policies/memory.yaml

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

## 20. policies/logging.yaml

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

## 21. policies/server-resources.yaml

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

## 22. policies/environments.yaml

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

## 23. policies/release.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `release.auto_suggest` | required | `true` | 是否自动建议 |
| `release.mode` | required | `branch_only` | `branch_only` 或 `deploy_to_environment` |
| `release.remote_providers` | required | 无 | 至少一个远程 provider |
| `release.default_batch` | required | 无 | 发布批次建议 |
| `release.gates` | required | 无 | 发布门禁 |
| `release.gates.require_release_note` | required | `true` | 必须为 true |
| `release.gates.require_coverage_passed` | required | `true` | 必须为 true |
| `release.gates.require_rollback_plan` | required | `true` | 必须为 true |
| `release.git` | required | 无 | release branch 和 tag 策略 |
| `release.deployment` | conditional_required | null | `deploy_to_environment` 时必填 |

必须为空：

- `mode = branch_only` 时，`release.deployment.enabled` 必须为 false 或 `release.deployment` 为 null。
- `mode = deploy_to_environment` 时，`release.deployment` 不能为 null。
- `release.gates.require_release_note`、`release.gates.require_coverage_passed`、`release.gates.require_rollback_plan` 必须为 true。

## 24. policies/budget.yaml

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

## 25. policies/engineering.yaml

| 字段 | 规则 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | required | 无 | MVP 固定为 `1` |
| `engineering.commit.enabled` | required | `true` | 启用 commit 规范 |
| `engineering.commit.format` | required | `conventional_commits` | MVP 固定 |
| `engineering.issue.required_fields` | required | 无 | Issue 必填字段 |
| `engineering.fix.require_regression_test` | required | `true` | 回退后 fix 必须补回归测试 |
| `engineering.release.require_release_note` | required | `true` | 发版必须有 release note |
| `engineering.release.require_rollback_plan` | required | `true` | 发版必须有回滚计划 |
| `engineering.coverage.default_thresholds` | required | 无 | 默认覆盖率阈值 |

必须为空：

- `engineering.issue.required_fields` 不允许为空数组。
- `engineering.coverage.default_thresholds.changed_files` 不允许为 null。
- `engineering.release.require_release_note`、`engineering.release.require_rollback_plan` 必须为 true。

## 26. 配置校验顺序

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

## 27. MVP 最小配置

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
- `policies/engineering.yaml`
- `policies/orchestration.yaml`
- `policies/comprehension.yaml`
- `policies/logging.yaml`

可以为空或延后：

- `policies/secrets.yaml`，仅本地公开仓库且无外部 API 时可为空对象。
- `policies/server-resources.yaml`，不启用部署时 `hosts` 可为空。
- `policies/environments.yaml`，不启用部署时可为空对象。
- `skills/enabled.yaml`，skills 未启用时可为空数组，但文件仍应存在。

## 28. 进入实现前必须补的机器校验

本文是人类可读 schema 规则。机器校验需要逐步转换为：

- JSON Schema，或
- Zod schema，或
- TypeScript 类型 + runtime validator。

Phase 3 当前落地：

- `moyuan workspace validate` 已开始读取用户可编辑的 `.moyuan/project.yaml`、`.moyuan/repository.yaml` 和 `.moyuan/policies/access.yaml`。
- 当前 validator 会检查 YAML 解析错误、`schema_version`、核心必填字段、`local_path`/`remote_git` 互斥、`local_single_user`/`team_server` 条件必填、`workspace.json` 与 YAML 的关键字段漂移。
- `.moyuan/models/providers.yaml` 已纳入可选校验：当文件存在时会校验 provider 管理开关、accounts、providers、`auth_ref` 引用、API 型 `base_url` 和明文密钥禁用。
- `.moyuan/models/routing.yaml` 已纳入可选校验：当文件存在时会校验 policies、primary provider、fallback provider 和明文密钥禁用。
- 后续 `phase3-002` 继续扩展到 provider、routing、visual、runtime、server、release 和 budget 配置域。

机器校验必须能输出：

- 缺失字段。
- 不允许为 null。
- 条件必填未满足。
- 条件必须为空未满足。
- secret 明文泄露。
- 引用对象不存在。
