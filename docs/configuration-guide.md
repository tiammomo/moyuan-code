# 配置方案

本文定义 Moyuan 项目配置的分层、组合方式、最小闭环和投产闭环。

权威边界：

| 内容 | 权威文档 |
| --- | --- |
| 字段必填、可选、可为空、必须为空、条件必填 | [配置 Schema 规则](./configuration-schema-spec.md) |
| `.moyuan/` 目录和 schema 索引 | [项目工作空间规范](./project-workspace-spec.md) |
| 模型、Provider、Claude CLI、Codex CLI、gpt-image-2 | [模型与工具适配规划](./model-tool-adapters.md) |
| Agent role 和默认 team | [Agent 角色与团队概览](./agent-roles-overview.md) |
| Subagent 与 Skills | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| Memory 配置语义 | [Agent Memory 系统方案](./agent-memory-system.md) |
| 服务器与发布投产 | [DevOps 发布投产主线](./mainlines/devops-release-deployment.md)、[服务器资源管理主线](./mainlines/server-resource-management.md) |

## 1. 配置目标

- 每个被管理项目拥有独立 `.moyuan/` 工作空间。
- 本地开发、远程仓库、多 Agent、质量门禁、Memory、日志、发布和部署都由配置驱动。
- 敏感信息只保存 `env:` 或 `secret:` 引用，不保存明文。
- MVP 可以用最小闭环运行，投产能力通过 release、server resources 和 environments 显式启用。
- Orchestrator、Runtime Adapter、Git Adapter、Memory Engine、Release Manager 和 Resource Manager 读取同一套项目配置。

## 2. 配置分层

```text
.moyuan/
  project.yaml
  repository.yaml
  agents/
    roles.yaml
    teams.yaml
    subagents.yaml
  models/
    providers.yaml
    routing.yaml
  runtimes/
    agent-runtimes.yaml
  skills/
    enabled.yaml
    registry.json
    bindings.json
    events.jsonl
    bindings.events.jsonl
  visuals/
    architecture-visuals.yaml
  policies/
    access.yaml
    permissions.yaml
    orchestration.yaml
    engineering.yaml
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

## 3. 配置索引

| 文件 | 职责 | 是否最小闭环必需 | 投产必需 | 详细规则 |
| --- | --- | --- | --- | --- |
| `project.yaml` | 项目基础信息、技术栈、工作区边界 | 是 | 是 | [配置 Schema 规则](./configuration-schema-spec.md) |
| `repository.yaml` | 本地/远程仓库、remote、分支策略 | 是 | 是 | [仓库接入与 Git Provider Adapter](./repository-onboarding-git-management.md) |
| `models/providers.yaml` | GPT、Claude、GLM、MiniMax、第三方 API、gpt-image-2 账号引用 | 是 | 是 | [模型与工具适配规划](./model-tool-adapters.md) |
| `models/routing.yaml` | planning、coding、review、memory、image generation 路由 | 是 | 是 | [模型与工具适配规划](./model-tool-adapters.md) |
| `runtimes/agent-runtimes.yaml` | Claude CLI、Codex CLI 等原生 Runtime | 是 | 是 | [Runtime Adapter 契约](./contracts/runtime-adapter-contract.md) |
| `agents/roles.yaml` | Agent role 到模型策略、工具和 memory scope 的映射 | 是 | 是 | [Agent 角色与团队概览](./agent-roles-overview.md) |
| `agents/teams.yaml` | feature、repair、release 等 team 编排 | 是 | 是 | [Agent 角色与团队概览](./agent-roles-overview.md) |
| `agents/subagents.yaml` | Subagent 创建、并发、父对象和输出契约 | 是 | 是 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| `skills/enabled.yaml` | 启用的 skills | 否 | 否 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| `skills/registry.json` | Phase 2 运行期 Skill Registry | 否 | 否 | [Subagent 与 Skill 契约](./contracts/subagent-skill-contract.md) |
| `skills/bindings.json` | Phase 2 运行期 Skill 绑定 | 否 | 否 | [Subagent 与 Skills 系统方案](./subagents-skills-system.md) |
| `policies/access.yaml` | 项目级角色、审批入口、审计开关 | 是 | 是 | [平台用户与访问控制主线](./mainlines/platform-user-access.md) |
| `policies/permissions.yaml` | 文件、命令、网络、密钥、Git 和部署权限边界 | 是 | 是 | [权限模型](./foundations/permission-model.md) |
| `policies/orchestration.yaml` | Issue Graph、并发、等待队列和合入门禁 | 是 | 是 | [Issues 编排与并发调度](./issue-orchestration.md) |
| `policies/engineering.yaml` | commit、issue、fix、release、coverage 规范入口 | 是 | 是 | [工程流程规范](./engineering-process-standards.md) |
| `policies/code-quality.yaml` | 可运行性、测试、重复度、复杂度、review 和自我修复入口 | 是 | 是 | [代码生命周期质量门禁](./code-lifecycle-quality-gates.md) |
| `policies/comprehension.yaml` | full/incremental/diff 项目阅读理解触发 | 是 | 是 | [项目接入与阅读理解主线](./mainlines/project-comprehension.md) |
| `policies/memory.yaml` | Memory record、retrieve、compact 和维护策略入口 | 否 | 建议 | [Agent Memory 系统方案](./agent-memory-system.md) |
| `policies/logging.yaml` | run、agent、model、git、quality、release、memory、audit、error 日志 | 是 | 是 | [日志与审计事件契约](./contracts/logging-audit-event-contract.md) |
| `policies/secrets.yaml` | secret provider、引用规则、用途校验和轮换策略 | 否 | 是 | [Secret Resolver 契约](./contracts/secret-resolver-contract.md) |
| `policies/budget.yaml` | 模型、Runtime、并发和成本预算 | 否 | 建议 | [模型与工具适配规划](./model-tool-adapters.md) |
| `policies/release.yaml` | release branch、tag、PR/MR、发布批次和审批 | 否 | 是 | [DevOps 发布投产主线](./mainlines/devops-release-deployment.md) |
| `policies/server-resources.yaml` | 测试开发机、生产机、云资产、到期和巡检 | 否 | 是 | [服务器资源管理主线](./mainlines/server-resource-management.md) |
| `policies/environments.yaml` | 环境、部署方式、冒烟、监控、回滚 | 否 | 是 | [DevOps 发布投产主线](./mainlines/devops-release-deployment.md) |
| `visuals/architecture-visuals.yaml` | gpt-image-2 架构图和流程图生成 | 否 | 否 | [模型与工具适配规划](./model-tool-adapters.md) |

## 4. 最小开发闭环

最小闭环用于本地或远程仓库的 AI 代码开发。

必须配置：

- 项目与仓库：`project.yaml`、`repository.yaml`。
- 模型与 Runtime：`models/providers.yaml`、`models/routing.yaml`、`runtimes/agent-runtimes.yaml`。
- Agent 编排：`agents/roles.yaml`、`agents/teams.yaml`、`agents/subagents.yaml`。
- 策略：`policies/access.yaml`、`policies/permissions.yaml`、`policies/orchestration.yaml`、`policies/engineering.yaml`、`policies/code-quality.yaml`、`policies/comprehension.yaml`、`policies/logging.yaml`。

最小闭环必须满足：

- 新项目接入后自动 full comprehension。
- 每次远程同步后自动 incremental comprehension。
- 用户需求先经过澄清判断和 Issue Graph 拆分。
- 前端复杂 UI 首版可默认 Claude CLI，样式稳定后的前端工程修改可路由 Codex CLI；后端和后端调优默认 Codex CLI。
- Web Console 默认 Next.js 16，前端端口 `3000`，Go/Gin API 后端端口 `8080`。
- 每个 issue 使用独立分支或 worktree。
- 每个 issue 都经过测试、质量门禁和 review。
- 日志和审计记录可追踪 run、agent、model、git、quality、memory、error。

## 5. 投产闭环

投产闭环在最小开发闭环基础上额外启用：

- `policies/secrets.yaml`
- `policies/release.yaml`
- `policies/server-resources.yaml`
- `policies/environments.yaml`

投产闭环必须满足：

- GitHub/Gitee 或通用 Git remote 发布策略已配置。
- release branch、tag、PR/MR、release note、审批和回滚策略已配置。
- 服务器资源区分 `test_dev` 和 `production`。
- 每台线上机器有 owner、auth_ref、基础规格、到期时间、健康检查和维护策略。
- 环境配置只引用资源组，不重复维护服务器字段。
- 生产投产启用备份、线上冒烟、监控窗口和回滚。

## 6. 敏感信息规则

配置文件禁止保存：

- API key、token、SSH 私钥、registry 凭证、云厂商密钥。
- 用户密码、session secret。
- `.env` 明文内容。
- 生产数据库连接串明文。

允许保存：

```yaml
auth_ref: env:OPENAI_API_KEY
ssh_key_ref: secret:prod_ssh_key
registry_auth_ref: secret:registry_token
```

`secret:` 引用必须在 `.moyuan/policies/secrets.yaml` 登记，并声明允许用途：

```yaml
schema_version: 1
secrets:
  minimax_runtime_token:
    type: token
    ref: env:MINIMAX_API_KEY
    usage:
      - runtime.invoke
      - model.provider.*
```

第三方 API 默认不得接收敏感代码、项目 Memory、secret 和生产事故上下文，除非项目策略显式批准。

## 7. 配置片段

本文只保留必要片段，完整字段以 [配置 Schema 规则](./configuration-schema-spec.md) 为准。

最小项目片段：

```yaml
schema_version: 1
project:
  id: order-service
  name: Order Service
  root: .
repository:
  source:
    type: remote_git
    provider: github
    url: https://github.com/org/order-service.git
```

最小 Runtime 片段：

```yaml
agent_runtimes:
  enabled: true
  default_runtime: codex_cli
  runtimes:
    claude_cli:
      type: native_agent_cli
      command: claude
      provider_env_profile:
        enabled: true
        allowed_env_keys:
          - ANTHROPIC_BASE_URL
          - ANTHROPIC_AUTH_TOKEN
          - ANTHROPIC_MODEL
    codex_cli:
      type: native_agent_cli
      command: codex
  role_runtime_defaults:
    frontend: claude_cli
    backend: codex_cli
    backend_tuning: codex_cli
```

MiniMax-M2.7 作为 Claude CLI 前端 profile 的最小登记：

```bash
export MINIMAX_API_KEY="<local-only>"

moyuan model provider add \
  --id minimax-m27-claude \
  --vendor minimax \
  --api-type anthropic-compatible \
  --base-url https://api.minimaxi.com/anthropic \
  --auth-ref env:MINIMAX_API_KEY \
  --runtime claude_cli \
  --model MiniMax-M2.7 \
  --use-case frontend \
  --allow-sensitive-code \
  --allow-project-memory
```

最小编排片段：

```yaml
orchestration:
  enabled: true
  issue_graph: true
  auto_parallelism: true
  max_parallel_issues: 3
  max_parallel_subagents: 4
  require_clean_worktree: true
```

最小 Memory 片段：

```yaml
memory:
  enabled: true
  record_gate:
    threshold: 3.5
  retrieval:
    top_k: 8
    role_scoped: true
  compact:
    enabled: true
    mode: automatic
```

最小发布片段：

```yaml
release:
  auto_suggest: true
  create_release_branch: true
  create_tag: true
  push_release_branch: true
  create_pr_or_mr: true
```

## 8. 校验清单

开发闭环：

- 配置文件存在且 schema_version 匹配。
- 必填字段通过 schema 校验。
- 敏感信息只有引用，没有明文。
- Claude CLI 和 Codex CLI 健康检查可执行。
- Issue Graph、Subagent、质量门禁和日志已启用。
- Memory compact 已启用或明确关闭。

投产闭环：

- release、server resources、environments 和 secrets 配置齐全。
- 生产机资源组和测试开发机资源组已区分。
- 生产部署需要审批、备份、冒烟、监控窗口和回滚策略。
- GitHub/Gitee token 或 SSH 权限满足 push/tag/PR/MR 需要。

维护要求：

- 新增配置文件时，必须同步更新本文索引、[配置 Schema 规则](./configuration-schema-spec.md) 和 [项目工作空间规范](./project-workspace-spec.md)。
- 新增字段时，不在多个文档重复维护字段规则。
- 示例只能展示关键组合，不承载完整 schema。
