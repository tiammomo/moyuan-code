# Phase 11 Release Readiness

状态：ready
责任角色：orchestrator_owner + backend_owner + frontend_owner + qa_owner
最后更新：2026-05-05

Phase 11 已完成“Issue Graph 批量执行控制器”的第一批能力。系统已经可以从 scheduler 生成 batch execution plan，解释依赖、并发槽、write scope 冲突和 provider route；可以在受控条件下运行 batch；可以为 issue 创建独立 worktree；可以聚合质量复核并生成 merge queue；Console 已能展示并触发 batch dry-run / merge queue build。

## 1. 完成范围

- `phase11-001 issue-batch-dispatch-preview`：已实现 batch plan，解释 dispatch、waiting、blocked、runtime/provider route 和 write scope conflict。
- `phase11-002 bounded-issue-batch-run`：已实现 batch run 事实源，默认 dry-run；真实 `local_shell` run 必须审批、环境开关和 prompt allowlist 同时满足。
- `phase11-003 parallel-worktree-isolation`：已实现 issue worktree manager，batch run 会为 issue 分配隔离 worktree 和 branch。
- `phase11-004 quality-review-merge-queue`：已实现 batch merge queue，复用单 issue quality/review gate，聚合 ready、needs_rework 和 blocked。
- `phase11-005 console-batch-execution-surface`：Console 已展示 batch plan/run、worktree、merge readiness，并支持受控 dry-run 和 merge queue build。

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
19d871b feat: add console batch execution surface
742bc71 feat: add batch merge queue
cab84d6 feat: isolate batch runs with issue worktrees
3f980ca feat: add bounded batch run records
854bb64 feat: add issue batch dispatch preview
```

## 3. 保留边界

- 真实并发 worker 还没有启动；`local_shell` 仍以受控串行或有限范围执行为主。
- merge queue 只生成合入决策和队列，不执行真实 `git merge`。
- 不自动创建 PR/MR，不自动发版，不自动 publish。
- Console 不计算 dependency、write scope、quality/review 或 merge readiness，只展示后端事实源。
- 生产写入、远程发布和服务器部署仍需要 approval/authz、secret resolver、执行开关和审计。

## 4. 进入 Phase 12 的理由

Phase 11 已经把“可见的批量执行控制面”补齐，但距离用户期望的完整多 Agent 自动开发还差三类执行能力：

- 真正按 issue graph 自动决定并发度，并启动多个 worker 在独立 worktree 中执行。
- merge queue 从只读决策推进到受控集成分支合入、冲突检测和返工回流。
- release batching 从建议推进到可控的版本分支、tag、PR/MR 和后续发布流水线入口。

Phase 12 应聚焦“真实并发执行与集成合入准备”，继续保持 production write 默认关闭。
