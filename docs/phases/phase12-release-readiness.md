# Phase 12 Release Readiness

状态：ready
责任角色：orchestrator_owner + backend_owner + git_owner + release_owner + frontend_owner + qa_owner
最后更新：2026-05-05

Phase 12 已完成“真实并发执行与集成合入准备”的第一批能力。系统已经可以基于 batch plan 启动受控并发 worker，在独立 worktree 中执行 issue，聚合 merge queue 后生成 integration merge preview，再通过受控 apply 固化本地 integration branch，并根据 ready item 数量生成 release batch readiness；Console 已能展示和触发这条链路的受控动作。

## 1. 完成范围

- `phase12-001 parallel-batch-worker-executor`：`local_shell` batch run 已支持 bounded worker pool、worker slot、fail-fast cancel 和稳定 item ordering。
- `phase12-002 integration-merge-preview`：ready merge queue 可创建独立 integration worktree 和 preview branch，逐项检测 clean merge、conflict、protected path 和 source branch 缺失。
- `phase12-003 controlled-merge-apply`：integration apply 默认 dry-run；真实 apply 需要审批和 `MOYUAN_ALLOW_INTEGRATION_APPLY=1`，且只更新本地 integration branch。
- `phase12-004 release-batch-readiness`：completed integration apply 可生成 release batch plan，记录版本、release branch、source branch、阈值和建议命令。
- `phase12-005 console-parallel-merge-surface`：Console 已展示 batch run parallelism、worker slot、integration preview、integration apply 和 release batch readiness，并只调用后端受控 API。

## 2. 验证结论

最新收口门禁：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```

结论：通过。

最近提交：

```text
61f2a17 feat: surface integration release flow
bb3fa93 feat: add release batch readiness
cb9267a feat: add controlled integration apply
fd1d12a feat: add integration merge preview
6c3b7b1 feat: add parallel batch worker executor
bab2a34 docs: open phase 12 parallel execution
```

## 3. 保留边界

- integration preview 会创建 preview worktree 和 preview branch，但不合入 main。
- integration apply 的真实写入只更新本地 integration branch，不 push、不 PR/MR、不 tag、不 publish。
- release batch readiness 只生成建议和命令预览，不创建 release branch、不推送远程、不打 tag。
- Console 不自行计算并发、合入或发版结论，只展示后端事实源和后端返回状态。
- GitHub/Gitee 远程写入、release branch 推送、tag 推送、PR/MR 创建和生产部署仍需要 approval/authz、secret resolver、执行开关和审计。

## 4. 进入 Phase 13 的理由

Phase 12 已经把“多 Agent 并发开发到本地 integration/release batch 准备”的链路打通。下一阶段应把本地准备结果推进到可控的远程发布候选链路：

- release batch plan 生成后，应能受控创建 release branch、关联 Git Provider PR/MR plan，并保留远程写入证据。
- GitHub/Gitee 发布动作需要从 preview/skipped 推进到 approval-gated execution。
- release candidate 应能衔接部署计划、线上冒烟和生产监控，但默认仍以 dry-run / guarded execution 起步。
- Console 需要能看到 release candidate 从 integration branch 到远程分支、PR/MR、tag、部署计划的完整状态。

Phase 13 应聚焦“Release Candidate 远程发布与部署交接”，继续保持生产写入默认关闭。
