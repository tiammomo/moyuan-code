# Phase 13 实施记录

状态：in_progress
责任角色：release_owner + git_owner + devops_owner + backend_owner + frontend_owner + security_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 13 的实际执行顺序。Phase 13 的入口以 [Phase 13 实现 Issue Graph](./phase13-issue-graph.md) 为准。

## 1. 当前基线

Phase 12 已完成并通过 readiness：

- Batch run 已支持受控并发 worker、worker slot 和 fail-fast cancel。
- 每个 issue 可以在独立 worktree 中执行，避免共享主工作区写入。
- Merge queue 可生成 integration merge preview，检测 conflict、protected path 和 blocked item。
- Integration apply 可在审批和开关满足后固化本地 integration branch。
- Release batch readiness 可根据 ready item 数量生成版本、release branch 和命令预览。
- Console 已能展示并触发 batch、integration preview、apply dry-run 和 release batch readiness。

## 2. Phase 13 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase13-001` | `release-candidate-plan-from-batch` | planned | Release Candidate 事实源 | suggested release batch 可生成 release candidate plan |
| P0 | `phase13-002` | `guarded-local-release-branch-apply` | planned | 本地 release branch 受控 apply | 审批和开关满足后可更新本地 release branch |
| P1 | `phase13-003` | `release-candidate-provider-preview` | planned | 远程发布预览 | Candidate 可生成 PR/MR、tag、release 和 workflow guarded preview |
| P1 | `phase13-004` | `deployment-handoff-from-release-candidate` | planned | 部署交接 | Candidate 可生成 deployment dry-run plan |
| P2 | `phase13-005` | `console-release-candidate-surface` | planned | Console 发布候选面 | Console 可见 candidate 到 provider/deploy 的完整链路 |

## 3. 执行规划：`phase13-001 release-candidate-plan-from-batch`

实现状态：planned。

范围：

- 新增 release candidate 事实源，读取 Phase 12 的 release batch plan。
- 只有 `RELEASE_BATCH_SUGGESTED` 才能生成 ready candidate；blocked/not_ready batch 生成 blocked candidate 并记录原因。
- Candidate 记录 release batch id、integration apply id、integration preview id、merge queue id、source branch、release branch、version、ready item count、provider、remote、deployment targets 和 commands preview。
- Candidate 写入 `.moyuan/lifecycle/releases/candidates/` 和 `release-candidates.jsonl`。
- API 支持生成、列表和详情查询。

非目标：

- 不创建 release branch。
- 不 push、不 tag、不 PR/MR、不 publish。
- 不创建 deployment execution。

验收：

- suggested release batch 可生成 `RELEASE_CANDIDATE_READY`。
- not_ready/blocked release batch 会生成 blocked candidate，并保留 batch reason。
- 非 Git 仓库或远程缺失时 candidate blocked。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

## 4. 后续执行占位

`phase13-002` 之后的实际落地结果在对应 issue 完成后补充，稳定设计会回写到 release、git provider、deployment 和 Console 相关主线文档。

## 5. 验证要求

每完成一个 Phase 13 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
