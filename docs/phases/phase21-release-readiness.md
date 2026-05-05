# Phase 21 Release Readiness

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 21 已完成“写入前 Review Packet 与队列准入绑定”的能力收口。系统现在可以把 write admission、provider proof requirement、remote execution rehearsal、control queue snapshot 和 evidence 聚合为 write review packet，并让 control queue 在执行前校验 admission、rehearsal 和 review packet 绑定。

## 1. 完成范围

- `phase21-001 write-review-packet`：新增 write review packet report，支持 create/list/load，持久化 JSON、追加 JSONL、写入 evidence，并进入 operations timeline。
- `phase21-002 control-queue-review-gate`：Control queue item 新增 `admission_id`、`remote_rehearsal_id`、`review_packet_id`，runner 执行前校验绑定事实，未 ready 时进入 manual handoff。
- `phase21-003 console-review-packet`：Console Operations 视图新增 Write Review Packet 面板，并在 Control Queue 面板展示绑定 ID。
- `phase21-004 phase21-readiness`：阶段文档、门禁和 Phase 22 入口完成收口。

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

- Write review packet 只聚合事实、生成审查材料和输出 gate decision，不执行 Git/provider/SSH/cloud 真实写入。
- Control queue 的 review gate 只能阻断或放行本地 control runner，不能绕过 authz、approval、secret、provider proof、write admission、remote rehearsal、maintenance window 或 retry budget。
- Console 只展示后端事实源，不重新计算 packet status、queue gate 或 evidence 结论。
- 真实写入仍默认关闭，必须进入 Phase 22 的 guarded write execution plan 后继续收敛。

## 4. 下一阶段入口建议

Phase 22 从以下入口继续：

- 基于 ready review packet 生成 guarded write execution plan。
- 区分 `preview` 和 `apply` mode，apply 必须要求 approval id 和显式写开关。
- 所有 execution plan 都必须持久化、写入 evidence，并明确记录 `external_write_performed=false`。
- Phase 22 仍不直接调用 GitHub/Gitee publish、SSH command、cloud operation 或 server mutation。
