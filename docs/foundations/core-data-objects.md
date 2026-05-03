# 核心数据对象

本文定义 Moyuan Code 的核心数据对象。配置示例仍由 [完整配置方案](../configuration-guide.md) 维护，本文只定义对象语义、关键字段、生命周期和关联关系。

## 设计原则

- 每个对象必须有稳定 `id`。
- 每个对象必须能追踪来源、版本和更新时间。
- 跨对象引用使用 id，不直接复制完整对象。
- 运行产物和配置对象分离。
- 敏感信息只保存引用，不保存明文。

## Project

职责：表示一个被 Moyuan 管理的软件项目。

关键字段：

- `id`
- `name`
- `organization_id`
- `owner_user_id`
- `root`
- `type`
- `description`
- `workspace_path`
- `repository_id`
- `schema_version`

生命周期：

```text
created -> onboarded -> comprehended -> active -> archived
```

落盘位置：

- `.moyuan/project.yaml`

关联对象：

- Repository
- Organization
- User
- Workspace
- Project Comprehension
- Epic
- Memory Record

## User

职责：表示使用 Moyuan 平台的人类用户。

关键字段：

- `id`
- `username`
- `email`
- `display_name`
- `status`
- `auth_methods`
- `created_at`
- `updated_at`

生命周期：

```text
invited -> active -> suspended -> disabled -> archived
```

落盘位置：

- local_single_user：本地身份文件和本地审计日志。
- team_server：控制面数据库。

关联对象：

- Organization
- Membership
- Auth Session
- API Token
- Approval

## Organization

职责：表示 Moyuan 平台内的团队、租户或组织边界。

关键字段：

- `id`
- `name`
- `status`
- `owner_user_id`
- `policy_refs`
- `created_at`
- `updated_at`

生命周期：

```text
created -> active -> suspended -> archived
```

落盘位置：

- team_server：控制面数据库。

关联对象：

- User
- Membership
- Project
- Service Account
- Audit Log

## Membership

职责：表示 User 或 Service Account 在组织或项目中的角色绑定。

关键字段：

- `id`
- `subject_type`
- `subject_id`
- `organization_id`
- `project_id`
- `roles`
- `status`
- `created_at`
- `updated_at`

生命周期：

```text
invited -> active -> suspended -> removed
```

落盘位置：

- team_server：控制面数据库。
- 项目级角色策略可有 `.moyuan/policies/access.yaml` 引用，但不保存凭证明文。

关联对象：

- User
- Service Account
- Organization
- Project

## Service Account

职责：表示 CI、发布、部署或外部系统使用的非人类 actor。

关键字段：

- `id`
- `name`
- `status`
- `owner_user_id`
- `organization_id`
- `project_id`
- `roles`
- `token_refs`
- `created_at`
- `updated_at`

生命周期：

```text
created -> active -> disabled -> revoked -> archived
```

落盘位置：

- team_server：控制面数据库。

关联对象：

- Membership
- API Token
- Run
- Release
- Deployment

## API Token

职责：表示调用 Moyuan API 或自动化任务的受限凭证引用。

关键字段：

- `id`
- `owner_type`
- `owner_id`
- `status`
- `scopes`
- `token_hash_ref`
- `token_prefix`
- `token_suffix`
- `expires_at`
- `last_used_at`

生命周期：

```text
created -> active -> rotated -> revoked
```

终止状态：

```text
expired
```

落盘位置：

- team_server：控制面数据库或密钥管理系统引用。
- 不允许落入 `.moyuan/` 明文配置。

关联对象：

- User
- Service Account
- Auth Session
- Audit Log

## Auth Session

职责：表示一次用户登录、本地身份或 API 会话的短期访问状态。

关键字段：

- `id`
- `user_id`
- `status`
- `created_at`
- `expires_at`
- `last_seen_at`
- `client`

生命周期：

```text
created -> active -> idle -> expired
```

失败或终止状态：

```text
revoked
invalid
```

落盘位置：

- local_single_user：本地会话状态或进程上下文。
- team_server：控制面数据库或会话存储。

关联对象：

- User
- API Token
- Audit Log

## Approval

职责：表示高风险操作的人工确认、拒绝、过期或取消记录。

关键字段：

- `id`
- `requester_id`
- `approver_id`
- `operation`
- `resource_type`
- `resource_id`
- `risk_level`
- `decision`
- `reason`
- `expires_at`
- `decided_at`

生命周期：

```text
requested -> approved -> consumed -> archived
```

失败或终止状态：

```text
rejected
expired
cancelled
```

落盘位置：

- `.moyuan/logs/audit/`
- `.moyuan/lifecycle/releases/`，如果与发布相关。
- `.moyuan/lifecycle/deployments/`，如果与部署相关。
- team_server：控制面数据库。

关联对象：

- User
- Run
- Issue
- Release
- Deployment
- Audit Log

## Repository

职责：描述被管理项目的 Git 仓库来源、remote、默认分支和分支策略。

关键字段：

- `id`
- `project_id`
- `source.type`
- `source.url`
- `source.local_path`
- `default_remote`
- `default_branch`
- `branch_policy`
- `release_branch`

生命周期：

```text
configured -> cloned_or_bound -> synced -> active -> disconnected
```

落盘位置：

- `.moyuan/repository.yaml`

关联对象：

- Project
- Task Branch
- Epic Branch
- Release Branch

## Project Comprehension

职责：保存项目阅读理解的结果和版本。

关键字段：

- `id`
- `project_id`
- `mode`
- `base_commit`
- `head_commit`
- `profile_version`
- `module_map_version`
- `risk_summary`
- `generated_at`

生命周期：

```text
pending -> running -> completed -> stale -> refreshed
```

落盘位置：

- `.moyuan/comprehension/project-profile.md`
- `.moyuan/comprehension/module-map.md`
- `.moyuan/comprehension/dependency-map.md`
- `.moyuan/comprehension/events.jsonl`

关联对象：

- Project
- Repository
- Memory Record
- Issue

## Epic

职责：表示用户提出的开发目标，是多个 issues 的上层容器。

关键字段：

- `id`
- `project_id`
- `title`
- `original_request`
- `refined_requirement`
- `clarification_status`
- `status`
- `issue_graph_id`
- `integration_branch`
- `created_at`
- `updated_at`

生命周期：

```text
created -> planning -> ready -> running -> completed -> released -> archived
```

落盘位置：

- `.moyuan/lifecycle/epics/`

关联对象：

- Issue
- Issue Graph
- Schedule
- Run
- Release

## Issue

职责：表示最小可执行开发单元。

关键字段：

- `id`
- `epic_id`
- `project_id`
- `title`
- `type`
- `description`
- `dependencies`
- `write_scope`
- `acceptance_criteria`
- `test_plan`
- `risk_level`
- `assigned_roles`
- `status`

生命周期：

```text
created -> blocked -> ready -> running -> quality_checking -> reviewing -> accepted -> merged
```

失败状态：

```text
failed
needs_rework
needs_user_input
cancelled
```

落盘位置：

- `.moyuan/lifecycle/issues/`

关联对象：

- Epic
- Issue Graph
- Schedule
- Run
- Quality Report
- Review

## Issue Graph

职责：描述 issues 的依赖 DAG、阻塞原因和 ready queue。

关键字段：

- `id`
- `epic_id`
- `nodes`
- `edges`
- `ready_queue`
- `blocked`
- `parallelism_plan`
- `generated_at`

生命周期：

```text
draft -> user_visible -> accepted -> executing -> completed -> replanned
```

落盘位置：

- `.moyuan/lifecycle/issue-graphs/`

关联对象：

- Epic
- Issue
- Schedule

## Schedule

职责：描述 issue 执行顺序、并发度、worktree 分配和资源限制。

关键字段：

- `id`
- `epic_id`
- `issue_graph_id`
- `max_parallel_issues`
- `ready_queue`
- `worktree_assignments`
- `resource_limits`
- `blocked_reason`

生命周期：

```text
planned -> executing -> paused -> completed -> replanned
```

落盘位置：

- `.moyuan/lifecycle/schedules/`

关联对象：

- Issue Graph
- Issue
- Run
- Runtime Session

## Run

职责：表示一次任务执行实例，是审计和恢复的核心对象。

关键字段：

- `id`
- `project_id`
- `epic_id`
- `issue_id`
- `agents`
- `runtime_id`
- `model_policy`
- `status`
- `started_at`
- `completed_at`
- `changed_files`
- `commands`
- `tests`
- `quality_gates`
- `runtime_signals`
- `memory_candidates`
- `logs`

生命周期：

```text
created -> running -> quality_checking -> verifying -> reviewing -> completed -> archived
```

失败状态：

```text
failed
cancelled
needs_rework
needs_user_input
```

落盘位置：

- `.moyuan/lifecycle/runs/`
- `.moyuan/logs/runs/`

关联对象：

- Issue
- Agent Role
- Runtime Session
- Quality Report
- Runtime Signal
- Repair Attempt
- Memory Record

## Agent Role

职责：定义 Agent 的职责、模型策略、skills、工具权限和 memory scope。

关键字段：

- `id`
- `name`
- `default_model_policy`
- `skills`
- `memory_scopes`
- `tools`
- `output_contract`

生命周期：

```text
defined -> enabled -> disabled -> deprecated
```

落盘位置：

- `.moyuan/agents/roles.yaml`

关联对象：

- Agent Team
- Run
- Skill
- Memory Record

## Agent Team

职责：定义某类任务使用的角色集合和验证链路。

关键字段：

- `id`
- `planners`
- `implementers`
- `verifiers`
- `release_roles`

生命周期：

```text
defined -> enabled -> disabled
```

落盘位置：

- `.moyuan/agents/teams.yaml`

关联对象：

- Agent Role
- Epic
- Issue
- Run

## Runtime Session

职责：保存 Claude CLI、Codex CLI 等原生 Agent Runtime 的会话和输出。

关键字段：

- `id`
- `runtime_id`
- `provider`
- `native_session_id`
- `project_id`
- `issue_id`
- `run_id`
- `worktree_path`
- `status`
- `created_at`
- `last_used_at`

生命周期：

```text
created -> active -> idle -> resumed -> closed -> archived
```

落盘位置：

- `.moyuan/runtimes/sessions/`
- `.moyuan/runtimes/outputs/`

关联对象：

- Run
- Issue
- Agent Role
- Model Provider

## Model Provider

职责：描述模型服务商 API、账号、模型能力、额度、健康状态和数据策略。

关键字段：

- `id`
- `vendor`
- `api_type`
- `base_url`
- `auth_ref`
- `enabled`
- `models`
- `quotas`
- `health_checks`
- `data_policy`

生命周期：

```text
configured -> healthy -> degraded -> disabled -> retired
```

落盘位置：

- `.moyuan/models/providers.yaml`
- `.moyuan/model-ops/`

关联对象：

- Model Policy
- Runtime Session
- Run
- Unified Logs

## Model Policy

职责：按任务类型、角色、成本、能力和敏感等级选择模型 provider。

关键字段：

- `id`
- `primary`
- `fallback`
- `constraints`
- `max_data_sensitivity`

生命周期：

```text
defined -> active -> overridden -> deprecated
```

落盘位置：

- `.moyuan/models/routing.yaml`

关联对象：

- Model Provider
- Agent Role
- Run

## Memory Record

职责：保存长期可检索、可维护、可审计的项目记忆。

关键字段：

- `id`
- `project_id`
- `type`
- `scope`
- `content`
- `entities`
- `tags`
- `confidence`
- `source`
- `status`
- `last_accessed_at`

生命周期：

```text
candidate -> staged -> committed -> hot -> warm -> cold -> archived
```

落盘位置：

- `.moyuan/memory/`

关联对象：

- Project
- Run
- Agent Role
- Project Comprehension
- Bug Candidate
- Repair Attempt
- Improvement Record

## Quality Report

职责：保存质量门禁和 review 的结构化结论。

关键字段：

- `id`
- `run_id`
- `issue_id`
- `status`
- `gates`
- `findings`
- `rework_items`
- `review_decision`
- `generated_at`

生命周期：

```text
created -> checking -> passed -> failed -> superseded
```

落盘位置：

- `.moyuan/lifecycle/quality/`

关联对象：

- Run
- Issue
- Review
- Bug Candidate
- Repair Attempt

## Runtime Signal

职责：表示测试、运行、冒烟、监控、用户反馈或 review 中产生的异常信号。

关键字段：

- `id`
- `project_id`
- `signal_type`
- `source_type`
- `source_id`
- `summary`
- `evidence_refs`
- `environment`
- `trace_id`
- `occurred_at`

生命周期：

```text
captured -> normalized -> correlated -> classified -> archived
```

失败状态：

```text
invalid
insufficient_context
```

落盘位置：

- `.moyuan/lifecycle/signals/`
- `.moyuan/logs/errors/`

关联对象：

- Run
- Issue
- Release
- Deployment
- Bug Candidate

## Bug Candidate

职责：表示由一个或多个 Runtime Signal 聚合出的疑似 bug。

关键字段：

- `id`
- `project_id`
- `signal_ids`
- `title`
- `affected_scope`
- `suspected_root_cause`
- `reproducible`
- `reproduction_commands`
- `classification`
- `confidence`
- `risk_level`
- `status`

生命周期：

```text
detected -> classifying -> confirmed -> issue_created -> repairing -> repaired -> archived
```

失败或分流状态：

```text
not_bug
needs_evidence
enhancement_candidate
blocked
```

落盘位置：

- `.moyuan/lifecycle/bug-candidates/`

关联对象：

- Runtime Signal
- Issue
- Repair Attempt
- Quality Report
- Memory Record

## Repair Attempt

职责：表示一次自动或半自动修复尝试。

关键字段：

- `id`
- `bug_candidate_id`
- `project_id`
- `issue_id`
- `run_id`
- `repair_branch`
- `write_scope`
- `strategy`
- `status`
- `regression_tests`
- `quality_report_id`
- `review_decision`

生命周期：

```text
planned -> branch_created -> running -> quality_checking -> reviewing -> accepted -> merged
```

失败或终止状态：

```text
blocked
failed
needs_rework
escalated
cancelled
```

落盘位置：

- `.moyuan/lifecycle/repair-attempts/`

关联对象：

- Bug Candidate
- Issue
- Run
- Quality Report
- Review
- Improvement Record

## Improvement Record

职责：表示由成功修复或重复问题产生的能力增强候选。

关键字段：

- `id`
- `project_id`
- `source_repair_attempt_id`
- `type`
- `summary`
- `confidence`
- `status`
- `target`
- `created_at`

生命周期：

```text
candidate -> approved -> applied -> archived
```

失败或终止状态：

```text
rejected
superseded
```

落盘位置：

- `.moyuan/lifecycle/improvements/`
- `.moyuan/memory/candidates/`，如果作为 Memory 候选。

关联对象：

- Repair Attempt
- Memory Record
- Agent Role
- Skill
- Model Policy
- Module Map

## Server Resource

职责：表示被纳管的服务器资产。

关键字段：

- `id`
- `category`
- `status`
- `owner`
- `cloud`
- `lifecycle.expires_at`
- `spec`
- `network`
- `system`
- `services`
- `healthcheck`

生命周期：

```text
registered -> active -> maintenance -> expiring -> retired
```

落盘位置：

- `.moyuan/resources/inventory.yaml`
- `.moyuan/resources/events.jsonl`
- `.moyuan/policies/server-resources.yaml`

关联对象：

- Resource Group
- Environment
- Deployment

## Resource Group

职责：表示可被环境部署引用的一组服务器资源。

关键字段：

- `id`
- `category`
- `host_ids`
- `deployment_order`
- `max_parallel_hosts`
- `requirements`

生命周期：

```text
defined -> active -> degraded -> retired
```

落盘位置：

- `.moyuan/policies/server-resources.yaml`

关联对象：

- Server Resource
- Environment
- Deployment

## Release

职责：表示版本分支、回归、release note、tag、PR/MR 和发布审批过程。

关键字段：

- `id`
- `project_id`
- `version`
- `source_branch`
- `release_branch`
- `accepted_issues`
- `risk_level`
- `status`
- `approval`
- `tag`

生命周期：

```text
suggested -> planned -> branch_created -> verified -> approved -> published -> archived
```

落盘位置：

- `.moyuan/lifecycle/releases/`

关联对象：

- Epic
- Issue
- Deployment
- Memory Record

## Deployment

职责：表示一次部署投产执行。

关键字段：

- `id`
- `release_id`
- `environment`
- `resource_group`
- `strategy`
- `artifact`
- `status`
- `smoke_result`
- `monitor_result`
- `rollback_plan`

生命周期：

```text
created -> precheck -> deploying -> smoke_testing -> monitoring -> healthy -> completed
```

失败状态：

```text
failed
rollback_required
rolled_back
needs_human_intervention
```

落盘位置：

- `.moyuan/lifecycle/deployments/`

关联对象：

- Release
- Environment
- Resource Group
- Unified Logs

## Visual Diagram

职责：表示由 gpt-image-2 辅助生成的架构流程图和讲解产物。

关键字段：

- `id`
- `diagram_type`
- `project_id`
- `source_refs`
- `diagram_spec_path`
- `prompt_path`
- `image_path`
- `explanation_path`
- `review_status`

生命周期：

```text
requested -> spec_generated -> image_generated -> reviewed -> published -> archived
```

落盘位置：

- `.moyuan/visuals/`
- `docs/assets/` 可保存对外展示图

关联对象：

- Project Comprehension
- Issue Graph
- Server Resource
- Release

## 对象引用规则

- 对象之间只通过 `id` 和路径引用。
- 不在 Run 中复制完整 Issue，只记录 `issue_id`。
- 不在 Memory 中保存完整 secret、prompt 或源码文件。
- 不在 Visual Diagram prompt 中保存密钥、私网 IP、token 或环境变量值。
- 对象状态变化必须写入对应事件日志或审计日志。
