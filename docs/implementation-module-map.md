# 实现模块拆分

状态：ready
责任角色：architect + core_engineer
最后更新：2026-05-03

本文定义 Moyuan Code 后续代码实现的模块边界。它把现有设计文档映射为可落地的代码包、服务接口和测试责任，不重复主线流程、策略决策树和契约字段。

## 1. 目标

- 让后续实现 issue 可以直接按模块拆分。
- 避免 Orchestrator、Runtime、Workspace、Memory、Quality 等职责混在一起。
- 明确每个模块的输入、输出、依赖方向和测试边界。
- 保证文档中的对象、策略和契约能映射为稳定代码结构。

## 2. 边界

本文只回答“代码模块如何拆、谁依赖谁、最小交付是什么”。

不在本文展开：

- CLI 命令列表，见 [总体规划与生命周期路线图](./lifecycle-roadmap.md)。
- 完整配置字段，见 [配置 Schema 规则](./configuration-schema-spec.md)。
- 端到端流程，见 [主线文档](./mainlines/README.md)。
- 策略判断树，见 [策略决策树](./policies/README.md)。
- 实现接口字段，见 [契约文档](./contracts/README.md)。

## 3. 模块分层

建议实现分为 9 层：

| 层级 | 模块 | 主要职责 |
| --- | --- | --- |
| Entry | `cli`、`api` | 接收用户命令、解析参数、建立 Auth Context |
| Control | `orchestrator` | 驱动 Epic、Issue Graph、Run、合入和恢复 |
| Planning | `requirement`、`scheduler` | 需求完善、澄清、issue 拆分、依赖和并发调度 |
| Execution | `subagent`、`runtime-adapters` | 创建 Subagent、调用 Claude CLI、Codex CLI、模型 API |
| State | `workspace`、`state-store` | `.moyuan/` 读写、锁、事务、迁移和恢复 |
| Knowledge | `comprehension`、`memory`、`skills` | 项目理解、长期记忆、skills 推荐和效果反馈 |
| Quality | `quality`、`review`、`self-repair` | 测试、质量门禁、review、bug 判断和修复 |
| External | `git`、`providers`、`server-resources`、`release` | Git、模型 Provider、服务器资源、发布部署 |
| Cross-cutting | `auth`、`policy`、`logging`、`secrets` | 鉴权、策略、日志、审计、密钥引用 |

依赖方向：

```text
cli/api
  -> orchestrator
  -> requirement / scheduler / subagent / quality / release
  -> runtime-adapters / git / providers / server-resources
  -> workspace / store / logging / auth / policy
```

禁止方向：

- `runtime-adapters` 不能直接修改 Issue Graph。
- `quality` 不能直接绕过 Orchestrator 合入分支。
- `memory` 不能直接读取 secret 明文。
- `providers` 不能直接读取完整 workspace，只能接收脱敏后的 request。
- `cli` 不能直接写 `.moyuan/lifecycle/` 状态，必须通过 Orchestrator 或 Workspace API。
- `api` 统一使用 Gin router。
- `store` 统一使用 GORM，业务模块不能自行迁移数据库表。

## 4. 模块职责

### `auth`

职责：

- 解析 local user、API Token、service account。
- 生成 `auth_context`。
- 调用权限策略判断 ALLOW、DENY、REQUIRE_APPROVAL。
- 写入鉴权和审批审计事件。

权威文档：

- [平台用户与访问控制主线](./mainlines/platform-user-access.md)
- [权限模型](./foundations/permission-model.md)
- [身份会话契约](./contracts/auth-session-contract.md)

### `workspace`

职责：

- 初始化 `.moyuan/`。
- 读写配置、生命周期状态、日志、Memory、项目理解产物。
- 管理 schema_version 和迁移。
- 提供原子写、锁和崩溃恢复能力。

权威文档：

- [项目工作空间规范](./project-workspace-spec.md)
- [Workspace 迁移契约](./contracts/workspace-migration-contract.md)
- [持久化与并发一致性](./persistence-concurrency-consistency.md)

### `api`

职责：

- 使用 Gin 暴露控制面 HTTP API。
- 管理 health、version、project、issue、run、quality、memory 等 API 入口。
- 只做参数绑定、响应结构和中间件编排，不承载领域状态机。

权威文档：

- [后端技术栈与本地环境](./backend-tech-stack.md)
- [持久化与并发一致性](./persistence-concurrency-consistency.md)

### `store`

职责：

- 使用 GORM 管理本地 SQLite State Store。
- 维护 project、issue、run、quality、memory 等查询型索引。
- 提供迁移入口，保证后续 PostgreSQL 替换不改变对象语义。

权威文档：

- [后端技术栈与本地环境](./backend-tech-stack.md)
- [持久化与并发一致性](./persistence-concurrency-consistency.md)

### `orchestrator`

职责：

- 接收用户目标并创建 Epic。
- 调用需求完善、Issue Planner、Scheduler。
- 创建 Run、Subagent Plan、质量检查和合入决策。
- 处理失败恢复、replan、needs_rework 和 retry。

权威文档：

- [参考架构](./reference-architecture.md)
- [Issues 编排与并发调度](./issue-orchestration.md)
- [Subagent 与 Skills 系统方案](./subagents-skills-system.md)

### `scheduler`

职责：

- 维护 blocked、ready、running、review 队列。
- 基于依赖、写入范围、worktree、Runtime slot 和预算计算并发度。
- 记录 blocked reason。
- 解锁下游 issue。

权威文档：

- [Issues 编排与并发调度](./issue-orchestration.md)
- [Issue 调度策略](./policies/issue-scheduling-policy.md)

### `runtime-adapters`

职责：

- 统一 Claude CLI、Codex CLI、模型 API 和 Shell 的调用。
- 管理 Runtime Session。
- 从 Provider Registry 解析 provider env profile，并向 Native Runtime 子进程注入允许的环境变量。
- 捕获 stdout、stderr、exit code、diff、输出契约和错误类型。
- 支持降级、超时、resume 和审计。

权威文档：

- [模型与工具适配规划](./model-tool-adapters.md)
- [Runtime Adapter 契约](./contracts/runtime-adapter-contract.md)

### `subagent`

职责：

- 根据 Issue 和 Role 创建 Subagent 实例。
- 解析 role、skills、memory scope、read/write scope 和输出契约。
- 收敛 Runtime 输出，并交回 Orchestrator。

权威文档：

- [Agent 角色与团队概览](./agent-roles-overview.md)
- [Subagent 与 Skills 系统方案](./subagents-skills-system.md)
- [Subagent 与 Skill 契约](./contracts/subagent-skill-contract.md)

### `quality`

职责：

- 执行 build、lint、typecheck、test、coverage、重复度、复杂度、架构边界和安全检查。
- 生成 Quality Report。
- 阻断不合格 diff 合入。

权威文档：

- [代码生命周期质量门禁](./code-lifecycle-quality-gates.md)
- [质量与合入策略](./policies/quality-merge-policy.md)
- [工程流程规范](./engineering-process-standards.md)

### `memory`

职责：

- Record Gate、Extraction、Staging Dedup、Retrieve、Automatic Compact、Reflection。
- 维护 memory candidates、staging、长期记忆、索引和审计。
- 为 planning、implementation、review、self-repair 提供检索上下文。

权威文档：

- [Agent Memory 系统方案](./agent-memory-system.md)
- [Memory 决策策略](./policies/memory-decision-policy.md)

### `comprehension`

职责：

- 项目首次 full comprehension。
- 远程拉取、rebase、merge、任务完成后的 incremental/diff comprehension。
- 输出 Project Profile、Module Map、Commands、Risk Files 和 Memory Candidates。

权威文档：

- [项目接入与阅读理解主线](./mainlines/project-comprehension.md)
- [项目阅读理解策略](./policies/project-comprehension-policy.md)

### `git`

职责：

- 本地路径和远程仓库接入。
- clone、fetch、branch、worktree、diff、merge、push、PR/MR。
- 保护用户 dirty worktree。

权威文档：

- [仓库接入与 Git Provider Adapter](./repository-onboarding-git-management.md)
- [Git 分支策略](./policies/git-branch-policy.md)

### `release` 和 `server-resources`

职责：

- 版本分支、tag、release note、GitHub/Gitee 发布。
- 部署资源组、测试开发机、生产机、冒烟、监控、回滚。

权威文档：

- [DevOps 发布投产主线](./mainlines/devops-release-deployment.md)
- [服务器资源管理主线](./mainlines/server-resource-management.md)
- [发布投产策略](./policies/release-deployment-policy.md)

## 5. 建议代码目录

后续代码实现建议使用以下逻辑目录。控制面主实现语言为 `Go`，模型邻接 worker 以 `Python` 为辅；具体语言边界见 [后端技术栈与本地环境](./backend-tech-stack.md) 和 [ADR-0005](./adr/0005-go-control-plane-python-worker.md)。

```text
cmd/
  moyuan/
internal/
  cli/
  api/
  store/
  auth/
  orchestrator/
  requirement/
  scheduler/
  subagent/
  runtime/
    claude/
    codex/
    model/
    shell/
  workspace/
  state/
  policy/
  logging/
  secrets/
  git/
  comprehension/
  memory/
  skills/
  providers/
  quality/
  review/
  self-repair/
  release/
  server-resources/
  visuals/
  testing/
workers/
  python/
    src/
      moyuan_worker/
        memory/
        review/
        prompts/
        analysis/
scripts/
```

## 6. 最小实现顺序

第一批必须先实现：

1. `workspace`：初始化、配置读取、原子写、日志目录。
2. `auth`：local_single_user、Auth Context、基础权限判断。
3. `logging`：run、audit、error 事件。
4. `git`：本地仓库接入、状态、diff、分支。
5. `runtime-adapters`：Codex CLI 和 Claude CLI 最小调用封装。
6. `orchestrator`：Epic、Issue、Run、状态流转。
7. `scheduler`：ready/blocked/running/review 队列和串行执行。
8. `quality`：基础 build/lint/test gate。

第二批扩展：

- 并发 worktree。
- Skill Registry。
- Memory Retrieve / Record Gate。
- Provider Registry。
- GitHub/Gitee push 和 PR/MR。
- self-repair。
- release/deployment。

## 7. 测试边界

每个模块必须有独立测试替身：

| 模块 | 测试替身 |
| --- | --- |
| `runtime-adapters` | fake Claude CLI、fake Codex CLI、fake model API |
| `git` | fake repo、temporary git repo |
| `workspace` | temporary `.moyuan/` workspace |
| `scheduler` | golden Issue Graph fixtures |
| `quality` | fake command runner 和 sample project |
| `providers` | fake provider server |
| `server-resources` | fake SSH/cloud inventory |

完整测试策略见 [框架自身测试策略](./framework-testing-strategy.md)。

## 8. 权限和安全

- 所有入口必须先生成 `auth_context`。
- 文件写入必须经过 Workspace API 和权限策略。
- Native Runtime 的写入范围必须由 Subagent Plan 限定。
- 外部 Provider 只接收脱敏后的上下文。
- 生产服务器、密钥引用、Git push、tag、部署必须进入审批和审计。

## 9. 失败恢复

模块失败必须映射到统一失败类型：

- 配置失败：schema validation error。
- 状态失败：lock conflict、transaction interrupted、migration failed。
- Runtime 失败：timeout、nonzero exit、invalid output contract。
- Git 失败：dirty worktree、merge conflict、push rejected。
- 质量失败：gate failed、review rejected。

恢复规则见 [失败恢复设计](./foundations/failure-recovery.md) 和 [持久化与并发一致性](./persistence-concurrency-consistency.md)。

## 10. Phase 1 实现拆分验收标准

进入 Phase 1 实现拆分时，本文必须满足：

- 每个核心能力都能映射到一个代码模块。
- 模块依赖方向清楚，没有循环依赖要求。
- 每个模块有权威文档和测试替身。
- 最小实现顺序可以直接转成开发 issues。
- 新模块不会绕过 Auth、Policy、Workspace、Logging 和 Quality Gate。

## 11. 相关文档

- [参考架构](./reference-architecture.md)
- [项目工作空间规范](./project-workspace-spec.md)
- [框架自身测试策略](./framework-testing-strategy.md)
- [持久化与并发一致性](./persistence-concurrency-consistency.md)
- [设计就绪门禁](./design-readiness-checklist.md)
