# Phase 10 Release Readiness

状态：ready
责任角色：orchestrator_owner + backend_owner + frontend_owner + devops_owner + qa_owner
最后更新：2026-05-05

Phase 10 已完成“控制面自动化闭环增强”的第一批能力。真实生产写入默认仍关闭，但系统已经可以触发 bounded control loop、复核 operation repair candidate、预览 release provider 高风险动作、解释部署检查模板，并在 Console 中形成可操作闭环。

## 1. 完成范围

- `phase10-001 background-control-loop-scheduler`：已实现手动 bounded control loop run，覆盖资源生命周期扫描、Provider ops refresh 和项目理解刷新 hook。
- `phase10-002 operation-repair-candidate-review-flow`：repair candidate 必须 approve/reject 后才能进入 repair issue 或 `review_ready` attempt。
- `phase10-003 release-provider-branch-tag-workflow-preview`：branch、tag、workflow dispatch 动作具备 risk、execution mode 和 guardrails 预览。
- `phase10-004 deployment-check-template-policy`：smoke/monitor plan、report 和 post-deployment history 已携带 template、severity 和 failure class。
- `phase10-005 console-route-repair-operator-surfaces`：Console 支持 provider route preview、repair candidate review、control loop history 和手动 run。

## 2. 验证结论

最新收口门禁：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```

结论：通过。

## 3. 保留边界

- 不启动后台常驻 scheduler。
- 不自动批准 repair candidate。
- 不自动执行生产部署或生产 rollback。
- branch push、tag push 和 workflow dispatch 仍保持 preview/skipped，不做真实远程写入。
- Console 不自行计算高风险结论，只展示后端事实源并调用受控 API。

## 4. 进入 Phase 11 的理由

当前系统已经能展示和触发控制面动作，但“用户提出一个开发任务后，系统自动拆分 issues、判断依赖、决定并发、编排开发、复核后合入”的执行控制器还需要进一步加强。

Phase 11 进入 Issue Graph 批量执行控制器：

- 从 scheduler dispatch queue 生成 batch execution plan。
- 解释哪些 issue 可并发、哪些因为依赖或 write scope 冲突等待。
- 先提供 dry-run/preview，避免多个 agent 直接在同一 worktree 写入。
- 后续再引入 worktree isolation、真实 bounded execution、质量聚合和 merge queue。
