# 完整配置方案

本文定义 Moyuan 项目配置的组织方式、最小可运行闭环、生产投产闭环和关键组合示例。

字段的必填、可选、可为空和必须为空规则由 [配置 Schema 规则](./configuration-schema-spec.md) 维护。本文不重复 schema 表，只说明配置意图、文件边界和跨模块组合方式。

## 1. 目标

配置目标：

- 每个被管理项目拥有独立 `.moyuan/` 工作空间。
- 本地开发、远程仓库、Issue 编排、多 Agent 执行、质量门禁、自我修复、Memory、日志、版本分支和投产部署都通过配置驱动。
- 敏感信息只保存 `env:` 或 `secret:` 引用，不保存明文。
- MVP 可以用最小配置运行；投产能力通过 release、server resources 和 environments 显式启用。
- Orchestrator、Agent Runtime、Git Adapter、Memory Engine、Release Manager 和 Resource Manager 读取同一套项目配置。

## 2. 配置分层

```text
.moyuan/
  project.yaml
  repository.yaml
  agents/
    roles.yaml
    teams.yaml
  models/
    providers.yaml
    routing.yaml
  runtimes/
    agent-runtimes.yaml
  visuals/
    architecture-visuals.yaml
  skills/
    enabled.yaml
  policies/
    access.yaml
    permissions.yaml
    orchestration.yaml
    code-quality.yaml
    comprehension.yaml
    memory.yaml
    logging.yaml
    secrets.yaml
    budget.yaml
    release.yaml
    server-resources.yaml
    environments.yaml
```

配置边界：

| 文件 | 负责内容 | 详细规则 |
| --- | --- | --- |
| `project.yaml` | 项目基础信息、技术栈、工作区边界 | [配置 Schema 规则](./configuration-schema-spec.md) |
| `repository.yaml` | 本地/远程仓库、GitHub/Gitee、分支策略 | [仓库接入与 Git 管理](./repository-onboarding-git-management.md)、[GitHub 接入配置](./github-integration.md) |
| `models/providers.yaml` | GPT、Claude、GLM、MiniMax、第三方 API、Codex/Claude Code provider | [模型与工具适配规划](./model-tool-adapters.md) |
| `models/routing.yaml` | 规划、编码、审查、Memory、图像生成的模型路由 | [模型与工具适配规划](./model-tool-adapters.md) |
| `runtimes/agent-runtimes.yaml` | Claude CLI、Codex CLI 原生 Agent Runtime | [模型与工具适配规划](./model-tool-adapters.md) |
| `visuals/architecture-visuals.yaml` | `gpt-image-2` 架构图和流程图生成 | [模型与工具适配规划](./model-tool-adapters.md) |
| `agents/roles.yaml`、`agents/teams.yaml` | Agent role、team、skills、memory scope | [Agent、Skills 与编排](./agent-skills-memory.md) |
| `policies/access.yaml` | 项目级访问边界、角色映射和审批入口；不保存用户密码或 Token 明文 | [平台用户与访问控制主线](./mainlines/platform-user-access.md) |
| `policies/orchestration.yaml` | Issue Graph、并发调度、等待队列、合入门禁 | [Issues 编排与并发调度](./issue-orchestration.md) |
| `policies/code-quality.yaml` | 可运行性、测试、重复度、复杂度、review 门禁和低风险自我修复入口 | [代码生命周期质量门禁](./code-lifecycle-quality-gates.md)、[运行反馈与自我修复主线](./mainlines/runtime-feedback-self-repair.md) |
| `policies/memory.yaml` | Memory record、retrieve、compact、维护策略入口 | [Agent Memory 系统方案](./agent-memory-system.md) |
| `policies/release.yaml` | 版本分支、tag、PR/MR、发布批次和投产策略 | [Issues 编排与并发调度](./issue-orchestration.md) |
| `policies/server-resources.yaml` | 测试开发机、生产机、云资产、到期和巡检 | [核心数据对象](./foundations/core-data-objects.md) |
| `policies/environments.yaml` | 部署、线上冒烟、监控、回滚 | [配置 Schema 规则](./configuration-schema-spec.md) |

## 3. 最小闭环

本地代码开发闭环必须配置：

- `project.yaml`
- `repository.yaml`
- `models/providers.yaml`
- `models/routing.yaml`
- `runtimes/agent-runtimes.yaml`
- `agents/roles.yaml`
- `agents/teams.yaml`
- `policies/access.yaml`
- `policies/permissions.yaml`
- `policies/orchestration.yaml`
- `policies/code-quality.yaml`
- `policies/comprehension.yaml`
- `policies/logging.yaml`

投产闭环额外需要：

- `policies/secrets.yaml`
- `policies/release.yaml`
- `policies/server-resources.yaml`
- `policies/environments.yaml`

可以延后或为空：

- `skills/enabled.yaml`：未启用 skills 时可为空。
- `policies/budget.yaml`：MVP 可使用系统默认预算。
- `visuals/architecture-visuals.yaml`：不需要架构图生成时可以关闭。

## 4. 项目与仓库

`project.yaml` 只保存项目基础事实和工作区边界：

```yaml
schema_version: 1
project:
  id: order-service
  name: Order Service
  root: .
  type: single-repo
  description: 订单服务

stack:
  languages: [typescript]
  frameworks: [nestjs]
  package_managers: [pnpm]
  build_commands: [pnpm build]
  test_commands: [pnpm test]
  lint_commands: [pnpm lint]

workspace:
  protected_paths:
    - .env
    - .env.*
    - secrets/**
  writable_paths:
    - src/**
    - tests/**
    - docs/**
```

`repository.yaml` 支持本地路径和远程仓库两种接入方式。GitHub 字段、token/SSH 规则和必填项由 [GitHub 接入配置](./github-integration.md) 维护。

```yaml
schema_version: 1
repository:
  source:
    type: remote_git
    provider: github
    url: https://github.com/org/order-service.git
    clone_path: ~/.moyuan/workspaces/github.com/org/order-service
  default_remote: origin
  default_branch: main

git:
  branch_policy:
    mode: task_branch
    naming: moyuan/{issue_id}-{slug}
    base: default_branch
    sync_before_run: true
    require_clean_worktree: true
    allow_auto_commit: false
    allow_auto_push: false
    allow_auto_pr: false
  commit_policy:
    enabled: true
    format: conventional_commits
    require_issue_ref: true
    require_run_ref: true
    require_quality_ref: true
    allowed_types: [feat, fix, perf, refactor, test, docs, build, ci, chore, revert, hotfix]
  epic_branch:
    enabled: true
    naming: moyuan/{epic_id}
  release_branch:
    enabled: true
    naming: release/{version}
```

## 5. 模型、Provider 与 Runtime

`models/providers.yaml` 统一管理：

- 官方 API：GPT/OpenAI、Claude/Anthropic。
- 国产模型 API：GLM、MiniMax、DeepSeek、DashScope 等。
- 第三方 OpenAI-compatible API 网关。
- 原生 Agent 后端：Codex CLI、Claude CLI/Claude Code。
- 图像生成：`gpt-image-2`。

```yaml
schema_version: 1
accounts:
  openai_main:
    vendor: openai
    api_type: openai
    base_url: https://api.openai.com/v1
    auth_ref: env:OPENAI_API_KEY
    enabled: true
    data_policy:
      allow_sensitive_code: true
      allow_project_memory: true

  third_party_llm_gateway:
    vendor: third_party
    api_type: openai-compatible
    base_url: ${THIRD_PARTY_LLM_BASE_URL}
    auth_ref: env:THIRD_PARTY_LLM_API_KEY
    enabled: false
    upstream_vendor: unknown
    data_policy:
      allow_sensitive_code: false
      allow_project_memory: false
      allow_secret_context: false

providers:
  codex:
    type: codex
    account: openai_main
    enabled: true
    capabilities:
      code_edit: true
      shell_exec: true
      repository_context: true
      review: true

  claude_code:
    type: claude-code
    account: anthropic_main
    enabled: true
    capabilities:
      code_edit: true
      shell_exec: true
      repository_context: true
      long_task: true

  gpt_image_2:
    type: image-generation-api
    adapter: openai-image
    account: openai_main
    enabled: true
    models:
      - id: gpt-image-2
        alias: architecture_image
```

`models/routing.yaml` 负责按任务选择 provider：

```yaml
schema_version: 1
policies:
  planning:
    primary: {provider: claude, model: claude_strong}
    fallback:
      - {provider: gpt, model: gpt_strong}

  coding_strong:
    primary: {provider: codex, model: default}
    fallback:
      - {provider: claude_code, model: default}

  review_reasoning:
    primary: {provider: gpt, model: gpt_strong}

  memory_record_gate:
    primary: {provider: gpt, model: gpt_fast}

  memory_extraction_light:
    primary: {provider: deepseek, model: deepseek_default}
    fallback:
      - {provider: minimax, model: minimax_text}

  architecture_visual_generation:
    primary: {provider: gpt_image_2, model: architecture_image}
    constraints:
      allow_secret_context: false
      require_diagram_spec: true
```

`runtimes/agent-runtimes.yaml` 只描述原生 Agent Runtime 的调用方式和隔离策略：

```yaml
schema_version: 1
agent_runtimes:
  enabled: true
  default_runtime: codex_cli
  session_store: .moyuan/runtimes/sessions
  output_store: .moyuan/runtimes/outputs
  require_diff_capture: true
  require_quality_gate_after_run: true

  runtimes:
    claude_cli:
      type: native_agent_cli
      provider: claude_code
      enabled: true
      command: claude
      auth:
        mode: local_cli_login
        auth_ref: null
      health_check:
        command: claude --version

    codex_cli:
      type: native_agent_cli
      provider: codex
      enabled: true
      command: codex
      auth:
        mode: env
        auth_ref: env:OPENAI_API_KEY
      health_check:
        command: codex --version

  role_runtime_defaults:
    frontend: claude_cli
    backend: codex_cli
    backend_tuning: codex_cli
    tester: codex_cli
    reviewer: codex_cli
    architect: claude_cli
    planner: claude_cli

  isolation:
    require_issue_worktree: true
    require_clean_worktree_before_start: true
    block_if_untracked_user_files: true
    capture_git_diff_before_start: true
    capture_git_diff_after_finish: true
```

## 6. Agent、Subagent、Team 与 Skills

Agent role、team 和 memory scope 的概要见 [Agent、Skills 与编排](./agent-skills-memory.md)。Subagent 生命周期、Skill Registry、Skill Binding 和效果反馈的完整设计见 [Subagent 与 Skills 系统方案](./subagents-skills-system.md)。配置中只保留可执行映射。

```yaml
schema_version: 1
roles:
  requirement_refiner:
    default_model_policy: planning
    skills: [requirement-enrichment]
    memory_scopes: [user_preferences, project_facts]
    tools: [read_project, search_memory]

  backend:
    default_model_policy: coding_strong
    skills: [backend-development, api-design]
    memory_scopes: [project_facts, lessons]
    tools: [read_project, edit_code, run_tests]

  frontend:
    default_model_policy: coding_strong
    skills: [component-design, frontend-performance]
    memory_scopes: [project_facts, user_preferences]
    tools: [read_project, edit_code, run_tests]

  quality_guard:
    default_model_policy: review_reasoning
    skills: [code-quality-review, duplication-check, complexity-analysis]
    tools: [read_project, run_quality_checks]
```

```yaml
schema_version: 1
teams:
  feature_team:
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

`agents/subagents.yaml` 控制 Subagent 创建、父子关系、并发和输出契约：

```yaml
schema_version: 1
subagents:
  enabled: true
  max_parallel_subagents: 4
  require_parent: true
  require_output_contract: true
  require_skill_compatibility_check: true
  allowed_parent_types:
    - epic
    - issue
    - run
    - repair_attempt
    - release
    - deployment
    - memory_job
  lifecycle:
    retry_on:
      - runtime_unavailable
      - timeout
    max_retries: 1
    require_orchestrator_for_child_tasks: true
```

`skills/registry.yaml` 和 `skills/bindings.yaml` 让 skills 可发现、可绑定、可审计：

```yaml
schema_version: 1
skills:
  - id: backend-development
    name: Backend Development
    version: 1.0.0
    source: builtin
    supported_roles: [backend, repair_agent]
    task_types: [backend, repair]
    required_tools: [read_project, edit_code, run_tests]
    memory_scopes: [project_facts, lessons]
    risk_level: medium
    enabled: true

bindings:
  - skill_id: backend-development
    target_type: role
    target_id: backend
    priority: 100
    status: enabled
```

## 7. Issue 编排与等待策略

Issue Graph、Subagent 并发度、前端 Claude / 后端 Codex 的等待模型由 [Issues 编排与并发调度](./issue-orchestration.md) 和 [Subagent 与 Skills 系统方案](./subagents-skills-system.md) 展开。配置只负责启用和限制。

```yaml
schema_version: 1
orchestration:
  enabled: true
  issue_graph: true
  auto_parallelism: true
  max_parallel_issues: 3
  max_parallel_subagents: 4
  require_clean_worktree: true
  use_epic_integration_branch: true
  use_issue_worktrees: true

  concurrency_guards:
    disallow_same_file_writes: true
    disallow_same_module_core_writes: true
    serialize_database_migrations: true
    serialize_auth_security_payment: true
    require_design_acceptance_for_public_api: true

  waiting_policy:
    event_driven: true
    poll_interval_seconds: 10
    require_contract_before_frontend_backend_parallel: true
    frontend_runtime: claude_cli
    backend_runtime: codex_cli
    queues:
      - blocked_queue
      - ready_queue
      - running_queue
      - review_queue

  merge_gate:
    require_quality_passed: true
    require_review_accepted: true
    require_style_check: true
    require_integration_checks: true
    merge_into_epic_branch_only: true

  issue_spec:
    required_fields:
      - clarified_requirement
      - acceptance_criteria
      - test_plan
      - write_scopes
      - subagent_plan
      - rollback_or_fix_plan
```

## 8. 质量、阅读理解与 Memory

工程流程规范配置：

```yaml
schema_version: 1
engineering:
  commit:
    enabled: true
    format: conventional_commits
    require_issue_ref: true
    require_run_ref: true
    require_quality_ref: true
  issue:
    required_fields:
      - clarified_requirement
      - acceptance_criteria
      - test_plan
      - write_scopes
      - rollback_or_fix_plan
  fix:
    require_regression_test: true
    require_root_cause: true
  release:
    require_release_note: true
    require_rollback_plan: true
  coverage:
    default_thresholds:
      line: 80
      branch: 70
      function: 80
      statement: 80
      changed_files: 85
      new_code: 85
```

代码质量门禁：

```yaml
schema_version: 1
quality:
  enabled: true
  required_for_all_code_tasks: true
  fail_task_on_blocker: true
  max_rework_rounds: 3
  require_independent_review: true
  allow_self_review: false

gates:
  runnable: {enabled: true, severity: blocker}
  test_gap: {enabled: true, severity: blocker}
  coverage:
    enabled: true
    severity: blocker
    thresholds:
      line: 80
      branch: 70
      function: 80
      statement: 80
      changed_files: 85
      new_code: 85
    high_risk_thresholds:
      auth_security_payment:
        line: 90
        branch: 85
        changed_files: 90
  duplication:
    enabled: true
    max_new_duplicate_ratio: 0.08
  complexity:
    enabled: true
    thresholds:
      max_function_lines: 80
      max_cyclomatic_complexity: 10
      max_nesting_depth: 4

self_repair:
  enabled: true
  mode: auto_repair_low_risk
  max_attempts_per_bug: 2
  require_regression_test: true
  auto_create_bug_candidate: true
  auto_repair_allowed_for:
    - test_failure
    - review_finding
  require_approval_for:
    - production
    - auth
    - security
    - payment
    - migration
    - public_api
  learning:
    record_bug_signature: true
    record_fix_pattern: true
    suggest_quality_rules: true
```

项目阅读理解：

```yaml
schema_version: 1
comprehension:
  enabled: true
  run_after_project_add: true
  run_after_remote_pull: true
  run_before_task_branch: true
  run_after_task_complete: true
  mode:
    initial: full
    after_pull: incremental
    before_task_branch: incremental
    after_task_complete: diff
```

Memory 配置只保留开关和策略入口。评分、抽取、暂存去重、自动 compact、reflection 和分层存储由 [Agent Memory 系统方案](./agent-memory-system.md) 维护。

```yaml
schema_version: 1
memory:
  enabled: true
  record_gate:
    model_policy: memory_record_gate
    threshold: 3.5
  extraction:
    model_policy: memory_extraction_light
    require_structured_output: true
  retrieval:
    top_k: 8
    min_score: 0.55
    role_scoped: true
  compact:
    enabled: true
    mode: automatic
    triggers:
      max_run_context_tokens: 24000
      staging_max_items: 100
      after_remote_pull: true
      after_task_complete: true
  maintenance:
    enabled: true
    compact_interval: daily
    merge_duplicates: true
```

## 9. 访问控制、日志与权限

访问控制配置只定义项目级角色、审批入口和审计开关。用户、会话、API Token 和服务账号是 Moyuan 控制面对象，不在 `.moyuan/` 保存密码、Token 明文或 session secret。

```yaml
schema_version: 1
access:
  mode: local_single_user
  local_owner_id: local-owner
  organization_id: null
  project_roles:
    project_owner:
      can_manage_members: true
      can_approve_release: true
    maintainer:
      can_run_issues: true
      can_merge_to_integration: true
    developer:
      can_run_issues: true
      can_modify_policies: false
  approval_policy:
    disallow_self_approval: false
    require_for:
      - git.push
      - git.tag
      - release.publish
      - deployment.production
      - policy.auth_change
  audit:
    enabled: true
```

权限配置只定义执行边界；完整权限模型见 [权限模型](./foundations/permission-model.md)。

```yaml
schema_version: 1
permissions:
  filesystem:
    writable_paths:
      - src/**
      - tests/**
      - docs/**
    protected_paths:
      - .env
      - .env.*
      - secrets/**
  commands:
    allow:
      - pnpm test
      - pnpm lint
      - pnpm build
      - git status
      - git diff
    require_approval:
      - git push
      - kubectl *
      - docker compose up -d
      - ssh *
  network:
    enabled: true
    require_approval_for_external: true
```

日志配置保留核心事件流，避免保存完整 prompt、response、secret 或 `.env` 内容。

```yaml
schema_version: 1
logging:
  enabled: true
  format: jsonl
  timezone: Asia/Shanghai
  level: info
  storage:
    base_dir: .moyuan/logs
    retention:
      run_logs_days: 30
      audit_logs_days: 180
      model_logs_days: 14
      error_logs_days: 90
  streams:
    run: {enabled: true, path: .moyuan/logs/runs/{date}.jsonl}
    agent: {enabled: true, path: .moyuan/logs/agents/{date}.jsonl}
    model:
      enabled: true
      path: .moyuan/logs/models/{date}.jsonl
      record_prompt: false
      record_response: false
      record_token_usage: true
      record_cost: true
    git: {enabled: true, path: .moyuan/logs/git/{date}.jsonl}
    quality: {enabled: true, path: .moyuan/logs/quality/{date}.jsonl}
    release: {enabled: true, path: .moyuan/logs/releases/{date}.jsonl}
    memory: {enabled: true, path: .moyuan/logs/memory/{date}.jsonl}
    audit:
      enabled: true
      path: .moyuan/logs/audit/{date}.jsonl
      immutable_append_only: true
    error: {enabled: true, path: .moyuan/logs/errors/{date}.jsonl}
  redaction:
    enabled: true
    redact_secret_refs: true
    redact_env_values: true
```

核心日志最小要求：

- `run`、`issue`、`agent`、`command`、`quality gate`、`merge`、`release` 和 `deploy` 都必须可追踪。
- 审批、密钥访问、高风险命令和保护路径拒绝必须进入 `audit`。
- 默认只记录模型、token、成本、耗时、状态和错误摘要，不保存完整提示词和响应。

## 10. 版本分支与投产

Release Manager 的流程见 [Issues 编排与并发调度](./issue-orchestration.md)。配置只定义策略入口。

```yaml
schema_version: 1
release:
  auto_suggest: true
  mode: deploy_to_environment
  default_environment: production
  remote_providers:
    - github
    - gitee
  default_batch:
    low_risk_issue_count: 5
    medium_risk_issue_count: 3
    high_risk_issue_count: 1
  force_single_release_for:
    - database_migration
    - auth_change
    - payment_change
    - security_change
    - public_api_breaking_change
  gates:
    require_user_approval: true
    require_full_regression: true
    require_release_note: true
    require_coverage_passed: true
    require_smoke_tests: true
    require_monitor_window: true
    require_rollback_plan: true
  git:
    create_release_branch: true
    release_branch_naming: release/{version}
    create_tag: true
    tag_naming: v{version}
    push_release_branch: true
    create_pr_or_mr: true
  deployment:
    enabled: true
    require_server_config: true
    monitor_window_minutes: 30
```

服务器资源统一登记在 `policies/server-resources.yaml`，环境只引用资源组，不重复维护主机字段。

```yaml
schema_version: 1
server_resources:
  enabled: true
  hosts:
    - id: test-dev-1
      category: test_dev
      provider: ssh
      host: test-dev-1.example.com
      user: deploy
      auth_ref: secret:test_dev_ssh_key
      app_path: /srv/order-service
      owner: {team: backend, primary: zhangsan, backup: lisi}
      lifecycle:
        expires_at: "2026-12-31"
        renewal_owner: ops
      spec: {cpu_cores: 4, memory_gb: 8, disk_gb: 100}
      healthcheck:
        command: systemctl is-active order-service

    - id: prod-app-1
      category: production
      provider: ssh
      host: app1.example.com
      user: deploy
      auth_ref: secret:prod_ssh_key
      app_path: /srv/order-service
      owner: {team: ops, primary: ops-owner, backup: backend-owner}
      lifecycle:
        expires_at: "2027-01-10"
        renewal_owner: ops
        auto_renewal: true
      spec: {cpu_cores: 8, memory_gb: 16, disk_gb: 200}
      backup:
        required: true
      healthcheck:
        command: systemctl is-active order-service

  groups:
    test_dev_app_servers:
      category: test_dev
      host_ids: [test-dev-1]
    production_app_servers:
      category: production
      host_ids: [prod-app-1]
      deployment_order: rolling
      require_pre_deploy_backup: true
      require_post_deploy_smoke: true
      require_monitor_window: true
```

```yaml
schema_version: 1
environments:
  test:
    resource_group: test_dev_app_servers
    approval_required: false
    artifact:
      type: docker_image
      image: registry.example.com/order-service
      tag: "{version}"
    deploy:
      strategy: recreate
      apply:
        - docker compose pull
        - docker compose up -d
    healthcheck:
      url: https://test-order.example.com/health
      retries: 3

  production:
    resource_group: production_app_servers
    approval_required: true
    artifact:
      type: docker_image
      image: registry.example.com/order-service
      tag: "{version}"
      registry_auth_ref: secret:registry_token
    deploy:
      strategy: rolling
      pre_deploy: [./scripts/backup.sh]
      apply:
        - docker compose pull
        - docker compose up -d
    healthcheck:
      url: https://order.example.com/health
      retries: 5
    smoke_tests:
      commands:
        - curl -f https://order.example.com/health
    observability:
      logs:
        type: ssh_tail
        path: /srv/order-service/logs/app.log
      metrics:
        type: http
        url: https://order.example.com/metrics
    rollback:
      enabled: true
      strategy: previous_release
      keep_releases: 5
```

## 11. 架构可视化

`visuals/architecture-visuals.yaml` 只服务于架构流程设计图和讲解，不参与代码合入。

```yaml
schema_version: 1
architecture_visuals:
  enabled: true
  provider_policy:
    diagram_planning: architecture_visual_planning
    image_generation: architecture_visual_generation
  output:
    base_dir: .moyuan/visuals
    diagrams_dir: .moyuan/visuals/diagrams
    explanations_dir: .moyuan/visuals/explanations
    prompts_dir: .moyuan/visuals/prompts
  diagram_types:
    project_architecture:
      inputs:
        - comprehension.project_profile
        - comprehension.module_map
        - repository.branch_policy
    lifecycle_flow:
      inputs:
        - lifecycle.issue_graph
        - quality.gates
        - release.pipeline
  safety:
    strip_secrets: true
    strip_private_ips: true
    strip_env_values: true
```

生成流程：

```text
collect project context
  -> strip sensitive details
  -> generate diagram spec
  -> generate visual prompt
  -> call gpt-image-2
  -> save image/explanation/prompt
  -> review readability and consistency
```

## 12. 配置校验清单

开发闭环：

- 项目、仓库、模型、Runtime、Agent team、权限、编排、质量、阅读理解和日志配置存在。
- `policies/access.yaml` 存在，且不包含密码、API Token 明文或 session secret。
- 前端默认 Runtime 为 `claude_cli`，后端和后端调优默认 Runtime 为 `codex_cli`。
- Issue Graph、等待队列、合入门禁和质量门禁已启用。
- 自我修复启用，但只允许低风险 confirmed bug 自动修复。
- Memory 自动 compact 已启用，但详细策略以 [Agent Memory 系统方案](./agent-memory-system.md) 为准。
- 日志启用 run、agent、model、git、quality、memory、audit、error 核心流。

投产闭环：

- GitHub/Gitee 远程发布策略配置完成。
- release branch、tag、PR/MR、回归、审批和发布批次策略配置完成。
- 服务器资源区分测试开发机和生产机。
- 每台线上机器有 owner、auth_ref、基础规格、到期时间、健康检查和维护策略。
- 环境配置引用资源组，不重复维护服务器资产。
- 生产投产启用备份、线上冒烟、监控窗口和回滚。

安全底线：

- 所有 API key、SSH key、token、registry 凭证和云厂商凭证都只能保存引用。
- 第三方 API 默认不得接收敏感代码、项目 Memory、secret 和生产事故上下文。
- 原生 Agent Runtime 的所有写入必须经过 diff 捕获、质量门禁和 review。
