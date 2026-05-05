# Phase 22 Release Readiness

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 22 已完成“受保护真实写入执行计划契约”的能力收口。系统现在可以基于 ready write review packet 生成 write execution plan，区分 `preview` 和 `apply` mode，并在 apply mode 中强制校验 approval id 与 `MOYUAN_ALLOW_REAL_WRITE=1`。本阶段仍不直接执行外部 provider、SSH、cloud 或服务器写入。

## 1. 完成范围

- `phase22-001 guarded-write-execution-plan`：新增 write execution plan preview/apply 契约，唯一前置输入为 review packet。
- `phase22-002 write-execution-api-cli`：新增 API/CLI create/list 入口，计划持久化 JSON、追加 JSONL、写入 evidence，并进入 operations timeline。
- `phase22-003 console-write-execution-plan`：Console Operations 视图新增 Write Execution Plan 面板。
- `phase22-004 phase22-readiness`：阶段文档、门禁和后续真实 adapter 入口完成收口。

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

- Write execution plan 是执行契约和审计事实，不是外部写入执行器。
- `preview` mode 不要求真实写入开关，但只生成可审查计划。
- `apply` mode 必须同时满足 ready review packet、approval id 和 `MOYUAN_ALLOW_REAL_WRITE=1`。
- 即使 `apply` mode 达到 `WRITE_EXECUTION_APPLY_READY`，Phase 22 仍记录 `external_write_performed=false`。
- GitHub/Gitee publish、SSH command、cloud operation、server mutation 必须在后续真实 adapter 阶段单独实现。

## 4. 下一阶段入口建议

Phase 23+ 可以从以下入口继续：

- 基于 write execution plan 接入 provider adapter preview/apply。
- 为 GitHub/Gitee release publish、SSH deployment、cloud operation 和 server resource mutation 分别实现 adapter。
- 在真实执行前消费 approval、解析 secret ref、执行 replay guard，并将结果写回 evidence 和 operations timeline。
- 增加失败恢复、回滚和线上 smoke/monitor 的强绑定。
