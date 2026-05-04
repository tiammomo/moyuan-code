# Phase 1 实现 Issue Graph

状态：ready
责任角色：orchestrator_owner + architect + core_engineer
最后更新：2026-05-04

本文把冻结后的 Phase 1 执行入口拆成可执行 issue graph。它只覆盖本地 CLI MVP，不覆盖 team_server、Web Console、生产部署和 Beta 能力。

## 0. 当前实现进度

当前进入 Phase 1 实施阶段，控制面已切换为 Go。

已完成骨架：

- `phase1-001 workspace-core`：已实现 `.moyuan/` 初始化、核心目录、`project.yaml`、`repository.yaml`、`policies/access.yaml` 和内部 `workspace.json` 状态缓存。
- `phase1-002 auth-context`：已实现 local owner、`whoami` 和基础 `auth_context` 审计事件。
- `phase1-003 logging-audit`：已实现 run、audit、quality、git JSONL 日志。
- `phase1-004 cli-bootstrap`：已实现 Go CLI 入口和 `bin/moyuan` wrapper。
- `phase1-005 git-adapter-basics`：已实现本地绑定、远程 clone、status、branch list、fetch sync。
- `phase1-006 runtime-adapters-core`：已实现 `local_shell` 调用、Claude CLI/Codex CLI 健康检查占位、timeout/exit code/status/result 落盘。
- `phase1-007 project-comprehension`：已实现 full/incremental comprehension 的启发式项目画像、模块地图、命令识别和 memory candidate。
- `phase1-008 orchestrator-core`：已实现 issue run 的 auth context、run、runtime、quality gate 和 accepted/needs_rework 状态收敛。
- `phase1-009 scheduler-core`：已实现 ready/blocked queue、blocked reason 和最小并发度计算。
- `phase1-010 quality-gates-core`：已实现 build、lint、test 命令执行和 quality report。
- `phase1-011 memory-basics`：已实现 memory add/search/compact 的最小闭环。
- `phase1-012 repair-basics`：已实现 runtime signal、bug candidate 分类和 repair plan 生成。
- `phase1-013 e2e-smoke`：已实现本地项目和本地 bare remote 模拟远程项目的端到端 CLI smoke，覆盖项目接入、阅读理解、issue graph、schedule、runtime、orchestrator、quality、memory、repair、logs 和关键 `.moyuan/` 产物断言。
- `phase1-014 runtime-diff-capture`：已实现 runtime before/after git snapshot、changed files、diff summary Markdown、`.moyuan/` 控制区过滤、脏工作区阻断和保护路径变更阻断。
- `phase1-015 native-runtime-adapters`：已实现 Claude CLI / Codex CLI 的 prompt file、cwd、env allowlist、stdout/stderr、result contract、fake CLI 回归测试和 unavailable/failed 分类。
- `phase1-016 orchestrator-state-machine`：已实现 issue/run 状态文件、状态转移历史、accepted/needs_rework 持久化、状态查询 CLI、issue graph/schedule 同步和回归测试。
- `phase1-017 quality-review-hardening`：已实现结构化 findings、review_status、protected path、敏感文件、runtime risk、大 diff 阻断和 Markdown/JSON 报告同步。
- `phase1-017a gin-gorm-baseline`：已切换后端框架口径为 Gin + GORM，新增 `internal/api` Gin router、`internal/store` GORM SQLite store，并让项目注册同步 `.moyuan/state.db`。
- `phase1-018 memory-record-gate`：已实现 candidate score、staging、dedup、敏感信息阻断、record 元数据、`memory candidates` 和 compact 自动摘要。

下一轮进入 `phase1-019 repair-controlled-loop`，将 runtime signal、bug candidate 和 repair attempt 接入受控修复闭环。下一批任务的执行顺序、验收标准和 git 同步规则见 [Phase 1 下一步开发任务规划](./phase1-next-development-plan.md)。

## 1. 目标

- 让 Phase 1 的实现顺序、依赖关系和并发边界可见。
- 让后续 issue 直接按模块和责任角色分配。
- 让 `ready_queue`、blocked reason 和验收标准都能从同一张图出发。
- 让最小闭环先跑通，再逐步补齐 memory 和 self-repair。

## 2. Epic

```text
phase1-epic: local-cli-mvp
```

Epic 目标：

- 本地路径接入。
- 远程 Git 接入。
- 本地 owner identity 和 `auth_context`。
- full comprehension / incremental comprehension。
- issue graph、schedule、quality gate、review。
- 基础 memory 和低风险 repair attempt。

## 3. Issue Graph

| ID | Issue | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| `phase1-001` | `workspace-core` | `.moyuan/` 初始化、配置读写、原子写、锁、workspace doctor | 无 | `core_engineer` | 能创建并维护最小 workspace |
| `phase1-002` | `auth-context` | local owner identity、Auth Context、ALLOW/DENY/REQUIRE_APPROVAL | `phase1-001` | `identity_owner` | 所有命令都能带 actor 和审计轨迹 |
| `phase1-003` | `logging-audit` | run/audit/error JSONL、redaction、日志目录 | `phase1-001` | `logging_owner` | 每次执行都能落日志且可追踪 |
| `phase1-004` | `cli-bootstrap` | CLI 入口、命令注册、项目加载、错误输出、命令分发 | `phase1-001`,`phase1-002`,`phase1-003` | `core_engineer` | `moyuan` 能启动并分发基础命令 |
| `phase1-005` | `git-adapter-basics` | 本地绑定、远程 clone、status、branch、diff、dirty worktree 保护 | `phase1-001`,`phase1-002`,`phase1-003` | `git_owner` | 能接入本地 repo 和 remote URL |
| `phase1-006` | `runtime-adapters-core` | Codex CLI / Claude CLI 最小封装、session、timeout、exit code、output contract | `phase1-001`,`phase1-003` | `adapter_owner` | 能稳定调用并捕获结构化结果 |
| `phase1-007` | `project-comprehension` | full/incremental/diff comprehension、project profile、module map、commands、risk files、memory candidates | `phase1-005` | `project_owner` | project add 和 git sync 能触发理解产物 |
| `phase1-008` | `orchestrator-core` | epic/issue/run 状态流转、上下文装配、issue 创建、dispatch flow | `phase1-004`,`phase1-005`,`phase1-006`,`phase1-007` | `orchestrator_owner` | 能创建并驱动一个 issue 生命周期 |
| `phase1-009` | `scheduler-core` | blocked/ready/running/review 队列、依赖计算、blocked reason | `phase1-008` | `orchestrator_owner` | 能从 DAG 计算 ready queue |
| `phase1-010` | `quality-gates-core` | build/lint/test/typecheck、diff review 摘要、quality report | `phase1-003`,`phase1-005`,`phase1-006` | `quality_owner` | 能对 sample issue 给出 pass/fail 结论 |
| `phase1-011` | `memory-basics` | record gate、retrieve、staging dedup、compact stub、candidates | `phase1-007`,`phase1-008` | `memory_owner` | 项目事实和运行事实可记录可检索 |
| `phase1-012` | `repair-basics` | runtime signal -> bug candidate -> repair attempt -> regression test | `phase1-010`,`phase1-011` | `quality_owner` | 低风险 bug 可回到修复闭环 |
| `phase1-013` | `e2e-smoke` | local/remote repo add、comprehension、schedule、run、quality、review、report | `phase1-004`~`phase1-012` | `qa_owner` + `core_engineer` | 一个本地项目和一个远程项目可跑通端到端 |

## 4. 依赖图

```text
phase1-001
  -> phase1-002
  -> phase1-003

phase1-002 + phase1-003
  -> phase1-004
  -> phase1-005
  -> phase1-006

phase1-005
  -> phase1-007

phase1-004 + phase1-005 + phase1-006 + phase1-007
  -> phase1-008

phase1-008
  -> phase1-009
  -> phase1-011

phase1-003 + phase1-005 + phase1-006
  -> phase1-010

phase1-010 + phase1-011
  -> phase1-012

phase1-004 + phase1-005 + phase1-006 + phase1-007 + phase1-008 + phase1-009 + phase1-010 + phase1-011 + phase1-012
  -> phase1-013
```

## 5. 推荐并发顺序

1. 先独立完成 `phase1-001`。
2. `phase1-002` 和 `phase1-003` 在 `workspace` 落地后可以并行。
3. `phase1-004`、`phase1-005`、`phase1-006` 在基础能力就绪后可以并行启动。
4. `phase1-007` 依赖 git 接入，适合在 Git 稳定后单独推进。
5. `phase1-008` 需要前面的事实、runtime 和理解产物齐备后再启动。
6. `phase1-009` 依赖 orchestrator core。
7. `phase1-010` 可以与 `phase1-007`、`phase1-008` 交错推进，只要 git/runtime 基础已稳定。
8. `phase1-011` 依赖 comprehension 和 orchestrator 的基本事实模型。
9. `phase1-012` 依赖 quality 和 memory。
10. `phase1-013` 放在最后做端到端验证。

## 6. Phase 1 边界

本图只覆盖 Phase 1 本地 CLI MVP。

暂不纳入第一批的能力：

- team_server。
- Web Console。
- 并发 worktree。
- GitHub/Gitee/GitLab PR/MR 自动创建。
- release/deployment 投产流水线。
- 更完整的 Skill 自动推荐闭环。

这些能力可以在 Phase 1 基础模块稳定后，按 [实现模块拆分](./implementation-module-map.md) 的第二批扩展继续拆分。
