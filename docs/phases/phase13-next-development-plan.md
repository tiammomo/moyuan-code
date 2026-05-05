# Phase 13 实施记录

状态：completed
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
| P0 | `phase13-001` | `release-candidate-plan-from-batch` | completed | Release Candidate 事实源 | suggested release batch 可生成 release candidate plan |
| P0 | `phase13-002` | `guarded-local-release-branch-apply` | completed | 本地 release branch 受控 apply | 审批和开关满足后可更新本地 release branch |
| P1 | `phase13-003` | `release-candidate-provider-preview` | completed | 远程发布预览 | Candidate 可生成 PR/MR、tag、release 和 workflow guarded preview |
| P1 | `phase13-004` | `deployment-handoff-from-release-candidate` | completed | 部署交接 | Candidate 可生成 deployment dry-run plan |
| P2 | `phase13-005` | `console-release-candidate-surface` | completed | Console 发布候选面 | Console 可见 candidate 到 provider/deploy 的完整链路 |

## 3. 执行规划：`phase13-001 release-candidate-plan-from-batch`

实现状态：completed。

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

落地结果：

- 新增 `release.PlanCandidate`、`LoadCandidate`、`ListCandidates`。
- Release candidate 写入 `.moyuan/lifecycle/releases/candidates/` 和 `release-candidates.jsonl`。
- API 新增 `POST /v1/projects/:project_id/release-batches/:batch_id/candidate`、`GET /v1/projects/:project_id/release-candidates`、`GET /v1/projects/:project_id/release-candidates/:candidate_id`。
- Candidate 仍是计划事实源，不创建 release branch、不执行远程写入、不创建 deployment execution。

## 4. 执行规划：`phase13-002 guarded-local-release-branch-apply`

实现状态：completed。

范围：

- 新增 release candidate apply 事实源，基于 ready release candidate 执行。
- 默认 `dry_run`，只验证 candidate ready，不更新 Git ref。
- 真实 `apply` 必须满足 `approved=true` 和 `MOYUAN_ALLOW_RELEASE_BRANCH_APPLY=1`。
- 真实 apply 只把 source integration branch 固化为本地 release branch，不 push、不 tag、不 PR/MR、不 publish。
- Apply 写入 `.moyuan/lifecycle/releases/candidate-applies/` 和 `release-candidate-applies.jsonl`。
- API 支持 apply、列表和详情查询。

非目标：

- 不推送远程 release branch。
- 不创建 tag、PR/MR、release 或 workflow dispatch。
- 不创建 deployment execution。

验收：

- Candidate 未 ready 时 apply blocked。
- 未审批或未开启环境开关时真实 apply blocked。
- dry-run 不更新 Git ref。
- 真实 apply 只更新本地 release branch。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

落地结果：

- 新增 `release.ApplyCandidate`、`LoadCandidateApply`、`ListCandidateApplies`。
- API 新增 `POST /v1/projects/:project_id/release-candidates/:candidate_id/apply`、`GET /v1/projects/:project_id/release-candidate-applies`、`GET /v1/projects/:project_id/release-candidate-applies/:apply_id`。
- release candidate apply 是本地 Git ref 层的受控动作，远程发布仍留给后续 provider preview/publish。

## 5. 执行规划：`phase13-003 release-candidate-provider-preview`

实现状态：completed。

范围：

- 新增 release candidate provider preview 事实源。
- Preview 必须基于 ready release candidate，且要求已有 completed local release branch apply。
- 生成 release provider remote plan，包含 push branch、create tag、push tag、create release、workflow dispatch 的 guarded action。
- 生成 PR/MR preview 摘要，记录 base branch、head branch、title、body、provider type 和 preview decision。
- Preview 写入 `.moyuan/lifecycle/releases/candidate-provider-previews/` 和 `release-candidate-provider-previews.jsonl`。
- API 支持生成、列表和详情查询。

非目标：

- 不执行 push branch、tag、release、workflow dispatch 或 PR/MR create。
- 不消费 approval。
- 不创建 deployment execution。

验收：

- Candidate 未 ready 时 provider preview blocked。
- Candidate 没有 completed release branch apply 时 provider preview blocked。
- Provider preview ready 时包含 release provider guarded actions 和 PR/MR preview。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

落地结果：

- 新增 `release.ProviderPreviewForCandidate`、`LoadCandidateProviderPreview`、`ListCandidateProviderPreviews`。
- API 新增 `POST /v1/projects/:project_id/release-candidates/:candidate_id/provider-preview`、`GET /v1/projects/:project_id/release-candidate-provider-previews`、`GET /v1/projects/:project_id/release-candidate-provider-previews/:preview_id`。
- Candidate provider preview 仍是远程发布预览层，不进行远程写入。

## 6. 执行规划：`phase13-004 deployment-handoff-from-release-candidate`

实现状态：completed。

范围：

- 在 deployment 模块新增基于 release candidate 的 deployment plan 创建入口。
- Candidate 必须 ready，否则 deployment plan blocked 并记录 candidate reason。
- 复用现有 server resource、environment、smoke/monitor template、production approval 和 rollback plan 规则。
- API 支持从 release candidate 创建 deployment plan。

非目标：

- 不执行 deployment execution。
- 不执行 SSH、local shell deploy、线上 smoke 或 monitor。
- 不改变现有 deployment execution 的生产写入边界。

验收：

- Ready candidate + active server resource 可生成 `DEPLOY_PLAN_READY`。
- Candidate 未 ready 时生成 blocked deployment plan。
- 生产环境仍需要 approval。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

落地结果：

- 新增 `deployment.CreatePlanFromCandidate`。
- API 新增 `POST /v1/projects/:project_id/release-candidates/:candidate_id/deployment-plan`。
- Release candidate 到 deployment plan 的交接只生成部署计划，不执行部署。

## 7. 执行规划：`phase13-005 console-release-candidate-surface`

实现状态：completed。

范围：

- Console snapshot 接入 release candidate、candidate apply 和 candidate provider preview。
- `Integration & Release` 面板展示 release batch、release candidate、branch apply、provider preview 和 deployment handoff。
- Console 可触发 release candidate plan、branch apply dry-run、provider preview 和 deployment plan。
- 所有状态以后端 API 返回为准，前端不自行计算 release/provider/deploy readiness。

非目标：

- 不在前端执行真实 Git 或远程发布命令。
- 不在前端开启 release branch apply 写开关。
- 不在前端执行 deployment execution。

验收：

- Console live snapshot 能读取并展示 release candidate 链路。
- 无后端时 demo snapshot 能展示完整 release candidate 示例。
- 新增按钮只调用后端受控 API，成功后刷新 server snapshot。
- 门禁通过：`go test ./...`、`npm run typecheck`、`npm run build`、`git diff --check`。

落地结果：

- `ConsoleSnapshot` 增加 `release_candidates`、`release_candidate_applies` 和 `release_candidate_provider_previews`。
- Console 可见 Phase 13 从 release batch 到 candidate、branch、provider 和 deployment handoff 的链路。

## 8. 后续执行占位

Phase 13 第一批任务完成后，应进入 release readiness 收口，确认远程写入仍默认关闭、Console 不计算权威结论，并规划后续真实 PR/MR 或 deployment execution 的审批边界。

## 9. 验证要求

每完成一个 Phase 13 issue，至少运行：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```
