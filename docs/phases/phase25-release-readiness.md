# Phase 25 Release Readiness

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 25 已完成 `ssh_deployment_adapter` 执行前 sandbox 和 rollback binding。系统现在可以在 write adapter execution 中记录 SSH target、AuthRef、command sandbox、preview-only/no-remote-write 结论和 rollback plan/runbook 绑定结果。该阶段不执行真实 SSH，也不执行 GitHub/Gitee、cloud 或服务器写入。

## 1. 完成范围

- `phase25-001 ssh-adapter-sandbox`：`ssh_deployment_adapter` 会加载 deployment execution 的 `RemotePlan`，生成 `sandbox_results`。
- `phase25-002 ssh-rollback-binding`：adapter execution 会绑定 deployment rollback plan；已有 rollback suggestion 时要求 runbook。
- `phase25-003 console-adapter-sandbox-view`：Console 展示 sandbox 和 rollback binding 摘要。
- `phase25-004 phase25-readiness`：阶段文档、门禁和后续真实 adapter 入口完成收口。

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

- Phase 25 不执行真实 SSH，`external_write_attempted=false` 和 `external_write_performed=false` 仍保持。
- SSH command sandbox 只负责准入判断，不负责命令编排和重试。
- Production rollback binding 会进入 manual review，不自动执行 rollback。
- GitHub/Gitee release provider adapter 仍未接入真实 publish。
- Adapter failure recovery record 仍需在后续阶段补齐。

## 4. 下一阶段入口建议

Phase 26+ 可以从以下入口继续：

- 增加 adapter failure recovery record，将失败转入 repair/retry/handoff。
- 为 GitHub/Gitee release provider adapter 接回真实 provider publish，并写入统一 adapter execution result。
- 为 SSH adapter 增加 preview-to-apply 的审批消费、replay guard 和执行后 smoke/monitor 绑定。
