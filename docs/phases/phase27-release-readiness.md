# Phase 27 Release Readiness

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 27 已完成 adapter recovery control queue binding。系统现在可以把 write adapter recovery record 绑定到 control queue item，并在 queue run 时进入人工复核门禁。该阶段不执行 repair、retry、handoff 或真实外部写入，只提供后续恢复编排的可审计入口。

## 1. 完成范围

- `phase27-001 adapter-recovery-queue-binding`：control queue item 支持 `adapter_recovery_id`。
- `phase27-002 adapter-recovery-review-gate`：绑定 recovery 的 queue item run 时进入 `CONTROL_QUEUE_RECOVERY_REVIEW_REQUIRED`。
- `phase27-003 console-queue-recovery-ref`：Console 展示 queue item 的 recovery ref。
- `phase27-004 phase27-readiness`：阶段文档、门禁和后续入口完成收口。

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

- Recovery queue 只进入人工复核，不自动执行恢复动作。
- 尚未实现 recovery approval consumption。
- 尚未实现 adapter repair runner 或 retry runner。
- GitHub/Gitee release provider adapter 仍未接入真实 publish。

## 4. 下一阶段入口建议

Phase 28+ 可以从以下入口继续：

- 为 adapter recovery queue 增加 approval consumption 和显式执行许可。
- 增加 adapter repair/retry runner 的 dry-run contract。
- 为 GitHub/Gitee release provider adapter 接回真实 provider publish，并写入统一 adapter execution result。
