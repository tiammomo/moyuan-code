# Phase 1 下一步开发任务规划

状态：ready
责任角色：orchestrator_owner + core_engineer + qa_owner
最后更新：2026-05-04

本文是 Phase 1 当前 Go CLI 骨架之后的开发计划。它不替代 [Phase 1 实现 Issue Graph](./phase1-issue-graph.md)，只把下一批任务的执行顺序、依赖、验收标准和 git 同步规则收敛到一个入口。

## 1. 当前基线

已完成最小骨架：

- `workspace`、`auth`、`logging`、`git`、`comprehension`、`issue graph`、`runtime adapter`、`orchestrator`、`scheduler`、`quality`、`memory`、`repair` 都已有 Go package 和 CLI 入口。
- `local_shell` runtime 已能执行并落盘结果。
- `phase1-013 e2e-smoke` 已覆盖本地项目和本地 bare remote 模拟远程项目的端到端 CLI 链路。
- `phase1-014 runtime-diff-capture` 已捕获 before/after git snapshot、changed files、diff summary，并阻断脏工作区和保护路径变更。
- `phase1-015 native-runtime-adapters` 已补齐 Claude CLI / Codex CLI 的 prompt file、cwd、env allowlist、stdout/stderr、result contract 和失败分类。
- `phase1-016 orchestrator-state-machine` 已持久化 issue/run 状态、转移历史、accepted/needs_rework 结论，并同步 issue graph/schedule。
- 当前测试覆盖已经包含 package unit test、CLI smoke 和 Phase 1 e2e smoke。

下一步目标不是继续铺更多模块，而是把“能跑通、能审计、能复核、能失败恢复”的 MVP 闭环做实。

## 2. 下一批任务总览

| 优先级 | ID | 任务 | 目标 | 依赖 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| Done | `phase1-013` | `e2e-smoke` | 用本地仓库和远程仓库样例验证完整 CLI 闭环 | `phase1-001`~`phase1-012` | 已由 Go e2e smoke 覆盖本地项目和本地 bare remote 模拟远程项目 |
| Done | `phase1-014` | `runtime-diff-capture` | 为 runtime run 捕获 before/after git 状态、changed files 和 diff summary | `phase1-005`,`phase1-006`,`phase1-008` | runtime result 已包含 git snapshot、changed files、diff summary，dirty worktree 和 protected path 可被阻断 |
| Done | `phase1-015` | `native-runtime-adapters` | 补齐 Claude CLI / Codex CLI 的真实调用契约和失败降级 | `phase1-014` | fake CLI 测试通过，CLI 缺失和非零退出有明确分类 |
| Done | `phase1-016` | `orchestrator-state-machine` | 持久化 issue/run 状态流转，连接 quality、review 和 rework | `phase1-014`,`phase1-015` | issue/run 状态可查询，accepted/needs_rework 可追踪，issue graph/schedule 可同步 |
| P1 | `phase1-017` | `quality-review-hardening` | 强化质量复核：diff review、secret scan、重复/复杂度/保护路径检查 | `phase1-014`,`phase1-016` | 不合格 diff 不能进入 accepted |
| P1 | `phase1-018` | `memory-record-gate` | 将当前 memory stub 升级为 record gate、staging、dedup、compact 最小闭环 | `phase1-007`,`phase1-016` | 可记录项目事实和运行经验，compact 可自动产生摘要 |
| P2 | `phase1-019` | `repair-controlled-loop` | 将 runtime signal、bug candidate、repair attempt 接入受控修复闭环 | `phase1-016`,`phase1-017`,`phase1-018` | 修复必须补回归测试并重新通过 quality gate |
| P2 | `phase1-020` | `docs-release-readiness` | 更新 README、CLI help、e2e 说明和 Phase 1 验收记录 | `phase1-013`~`phase1-019` | 用户可按文档复现 Phase 1 MVP |

## 3. 推荐执行顺序

1. `phase1-013 e2e-smoke` 已完成，当前 CLI 骨架有可重复 e2e 基线。
2. `phase1-014 runtime-diff-capture` 已完成，后续 Native Runtime 修改代码时可以进入 diff 复核。
3. `phase1-015 native-runtime-adapters` 已完成，Claude CLI / Codex CLI 已从 health check 占位推进到可执行契约。
4. `phase1-016 orchestrator-state-machine` 已完成，issue/run 状态不再只停留在单次命令输出。
5. 下一步做 `phase1-017 quality-review-hardening`，随后推进 `phase1-018 memory-record-gate`。
6. 最后做 `phase1-019 repair-controlled-loop` 和 `phase1-020 docs-release-readiness`。

## 4. 任务详情

### `phase1-013 e2e-smoke`

状态：已完成。

范围：

- 新增本地 e2e smoke 脚本或 Go integration test。
- 覆盖 `project add --local`、`project add --remote`、`comprehend`、`issue graph`、`orchestrator plan/run`、`quality check`、`memory add/search/compact`、`repair signal`、`logs tail`。
- 远程仓库优先使用本地 bare git repo 模拟，避免网络和外部权限影响测试稳定性。

验收：

- 脚本可在干净临时目录重复运行。
- 能断言 `.moyuan/project.yaml`、`.moyuan/repository.yaml`、`.moyuan/comprehension/*`、`.moyuan/runtime/*`、`.moyuan/quality/*`、`.moyuan/logs/*` 存在。
- 失败时输出具体命令、stdout、stderr 和临时目录路径。

实现：

- e2e smoke 位于 `internal/cli/cli_test.go`。
- 远程项目通过本地 bare git repo 模拟，不依赖 GitHub/Gitee 网络。
- 验证命令为 `go test ./...`。

### `phase1-014 runtime-diff-capture`

状态：已完成。

范围：

- runtime 执行前记录 git branch、HEAD、dirty status。
- runtime 执行后记录 changed files、diff summary、exit code、duration 和 artifact path。
- 在受保护路径或脏工作区风险存在时返回 blocked/needs_approval。

验收：

- `RuntimeResult` 包含 `changed_files`、`diff_summary_path`、`git_before`、`git_after`。
- `orchestrator run` 能把 diff 信息传给 quality gate。
- 没有 git 仓库时明确降级为 `diff_unavailable`，不能伪造结果。

实现：

- Git snapshot 和 diff summary 位于 `internal/git/diff.go`。
- Runtime result 已包含 `git_before`、`git_after`、`diff`、`changed_files` 和 `diff_summary_path`。
- Runtime 会过滤 `.moyuan/` 控制区改动，阻断 pre-existing user dirty worktree，并在保护路径变更时返回 blocked。
- 回归测试位于 `internal/cli/cli_test.go`。

### `phase1-015 native-runtime-adapters`

状态：已完成。

范围：

- 定义 Claude CLI / Codex CLI 的 prompt file、cwd、env allowlist、timeout、stdout/stderr、result contract。
- 支持 CLI 不存在、超时、非零退出、无 diff、输出不可解析等错误分类。
- 测试使用 fake `claude` / fake `codex` 二进制放入临时 `PATH`。

验收：

- `runtime health` 能区分 available/unavailable/degraded。
- `runtime invoke claude_cli` 和 `runtime invoke codex_cli` 能落盘结构化结果。
- 真实 CLI 不存在时不影响 `local_shell` 和其他 CLI 测试。

实现：

- Prompt 编译落盘在 `.moyuan/runtime/prompts/`。
- Claude CLI 以 `claude -p <prompt>` 的最小 headless 形态调用。
- Codex CLI 以 `codex exec --skip-git-repo-check --cd <worktree> -` 调用，并通过 stdin 传入 prompt。
- Native CLI metadata、stdout、stderr、exit code 和 command contract 落盘到 `.moyuan/runtime/*-native.json`。
- fake `claude` / fake `codex` 回归测试覆盖成功、CLI unavailable 和 CLI failed。

### `phase1-016 orchestrator-state-machine`

状态：已完成。

范围：

- 持久化 epic、issue、run 的状态快照。
- 支持 `ready`、`blocked`、`running`、`quality_check`、`review`、`accepted`、`needs_rework`、`failed`。
- quality fail、runtime fail、review finding 都必须转成明确下一步。

验收：

- `orchestrator plan` 输出 ready queue、blocked queue 和并发度。
- `orchestrator run <issue-id>` 后状态可查询、可恢复、可重试。
- 下游 issue 只有在前置 accepted 后才能进入 ready。

实现：

- Issue state 写入 `.moyuan/orchestrator/issue-states/<issue-id>.json`。
- Run state 写入 `.moyuan/orchestrator/run-states/<run-id>.json`。
- `orchestrator status <issue-id>` 和 `orchestrator issue status <issue-id>` 可查询 issue 状态。
- `orchestrator run status <run-id>` 可查询 run 状态。
- 状态转移会同步 issue graph 和 schedule，并写入 run 日志。

### `phase1-017 quality-review-hardening`

范围：

- 增加 secret scan、protected path、diff size、重复代码、复杂度和测试覆盖提示。
- 将 review finding 作为结构化质量结果，而不是自然语言备注。
- 明确 `accepted`、`needs_rework`、`rejected` 的判定门槛。

验收：

- 发现疑似密钥时强制 fail。
- 触碰保护路径时至少进入 `needs_approval`。
- quality report 同时输出 JSON 和 Markdown。

### `phase1-018 memory-record-gate`

范围：

- 实现 memory candidate 的 score、staging、dedup、record gate 和 compact。
- 支持项目事实、技术决策、失败模式、修复模式和偏好规则。
- 与 [Agent Memory 系统方案](./agent-memory-system.md) 保持一致，当前阶段不实现完整向量检索。

验收：

- 低分或敏感候选不会进入长期 memory。
- compact 能基于新增 memory 自动生成摘要。
- 每条 memory 都有 source、scope、confidence、created_by 和 trace。

### `phase1-019 repair-controlled-loop`

范围：

- 将 runtime signal 分类为 non_bug、bug_candidate、confirmed_bug、enhancement。
- 低风险 confirmed bug 才允许进入 repair attempt。
- repair 后必须补充或运行回归测试，再回到 quality/review。

验收：

- repair attempt 有最大重试次数和失败转人工规则。
- 自动修复不能直接提交或合入。
- 成功修复会生成 memory candidate。

### `phase1-020 docs-release-readiness`

范围：

- 更新 README、CLI help、Phase 1 验收说明和 e2e 运行说明。
- 记录 Phase 1 已实现能力、未实现能力和风险。
- 补充从本地安装到执行 smoke 的最短路径。

验收：

- 新用户可按 README 跑通 smoke。
- 文档不再保留“仍在规划中”的 Phase 1 入口表述。
- 未实现能力明确标注为 Beta/Production。

## 5. 并发策略

当前阶段推荐低并发：

- `phase1-013` 已完成，作为后续实现的回归基线。
- `phase1-014` 已完成，runtime、git、orchestrator 和 quality 的共同字段已有回归基线。
- `phase1-015` 已完成，后续可在真实 CLI 安装环境中继续增强 session resume 和结构化输出解析。
- `phase1-017` 和 `phase1-018` 可以并行，但当前建议先做 `phase1-017`，让 quality/review 的阻断能力先接入状态机。
- `phase1-019` 等 `phase1-016`、`phase1-017`、`phase1-018` 完成后再做。

## 6. 每轮实现门禁

每个任务完成前必须执行：

```bash
go test ./...
git diff --check
```

如果改动涉及 shell 调用、runtime、Git 或日志，还必须检查：

- 不把 API key、token、SSH key、`.env` 明文写入日志、Memory、prompt 或测试 fixture。
- 不破坏用户已有 dirty worktree。
- 失败路径能落盘结构化错误。
- 文档、CLI help 和测试描述与实际命令一致。

## 7. Git 同步规则

- 每个 Phase 1 任务独立 commit。
- commit message 使用 `feat:`、`fix:`、`test:`、`docs:` 或 `chore:` 前缀。
- commit 前必须查看 `git status --short`，确认没有无关文件混入。
- 提交后推送 `origin/main`。
- 如果任务产生未完成风险，必须在对应文档或 TODO 状态文件中记录，不允许只留在对话中。

## 8. 暂不做

以下内容继续留在 Beta/Production，不进入本轮：

- Web Console。
- team_server。
- 多用户组织级协作。
- GitHub/Gitee/GitLab PR/MR 自动创建。
- 生产服务器部署和监控。
- 完整向量检索 Memory。
- 完整 Provider Registry 和模型成本统计。
