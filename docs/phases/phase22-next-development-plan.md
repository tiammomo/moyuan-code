# Phase 22 实施记录

状态：planned
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

本文记录 Phase 22 的实际执行顺序。Phase 22 的入口以 [Phase 22 实现 Issue Graph](./phase22-issue-graph.md) 为准。

## 1. 阶段入口

Phase 22 只能在 Phase 21 完成后进入。唯一前置输入是 write review packet。

## 2. Phase 22 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase22-001` | `guarded-write-execution-plan` | planned | 生成 preview/apply 执行计划契约 | Apply 受 packet、approval、write switch 保护 |
| P0 | `phase22-002` | `write-execution-api-cli` | planned | API/CLI create/list | 可查询、可审计、可测试 |
| P1 | `phase22-003` | `console-write-execution-plan` | planned | Console 展示 execution plan | 前端只读展示事实源 |
| P1 | `phase22-004` | `phase22-readiness` | planned | Phase 22 收口 | 全量门禁和后续入口完成 |

## 3. 执行规划：`phase22-001 guarded-write-execution-plan`

实现状态：planned。

范围：

- 新增 write execution plan，支持 `preview` 和 `apply` mode。
- Preview mode 输出可审查计划，不要求真实写入开关。
- Apply mode 需要 ready review packet、approval id、`MOYUAN_ALLOW_REAL_WRITE=1`，并仍记录 `external_write_performed=false`。
- Plan 持久化到 `.moyuan/lifecycle/deployments/write-execution-plans/`，追加 JSONL，并写入 evidence。

非目标：

- 不调用 GitHub/Gitee/SSH/cloud 外部写入。
- 不消费 secret 明文。
- 不替代后续 provider adapter。

## 4. 执行规划：`phase22-002 write-execution-api-cli`

实现状态：planned。

范围：

- API 增加 `POST/GET /v1/projects/:project_id/operations/write-execution-plans`。
- CLI 增加 `moyuan operations write-execution-plans create|list ...`。
- 单测覆盖 preview、apply switch disabled、apply ready 但 external write 未执行。

## 5. 执行规划：`phase22-003 console-write-execution-plan`

实现状态：planned。

范围：

- Console Operations 面板读取 `write_execution_plans`。
- 展示 plan mode、status、decision、review packet id、approval id、reasons、rule refs、evidence refs 和 external write 标记。

## 6. 执行规划：`phase22-004 phase22-readiness`

实现状态：planned。

范围：

- 运行全量门禁。
- 回写 README、docs 入口、Phase 22 issue graph、实施记录和 readiness。
- 明确 Phase 23+ 真实 adapter 的入口与保留边界。

## 7. 验证要求

每完成一个 Phase 22 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
