# Phase 23 实施记录

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

本文记录 Phase 23 的实际执行顺序。Phase 23 的入口以 [Phase 23 实现 Issue Graph](./phase23-issue-graph.md) 为准。

## 1. 阶段入口

Phase 22 已完成 guarded write execution plan。Phase 23 只做 adapter dispatch scaffold，把 execution plan 转成 adapter execution 事实，不打开真实外部写入。

## 2. Phase 23 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase23-001` | `write-adapter-dispatch-scaffold` | completed | 生成 adapter execution | 可推导 adapter、记录 guard、无外部写入 |
| P0 | `phase23-002` | `write-adapter-api-cli` | completed | API/CLI create/list | 可查询、可审计、可测试 |
| P1 | `phase23-003` | `console-write-adapter-execution` | completed | Console 展示 adapter execution | 前端只读展示事实源 |
| P1 | `phase23-004` | `phase23-readiness` | completed | Phase 23 收口 | 全量门禁和后续入口完成 |

## 3. 完成记录

- `internal/operations` 新增 write adapter execution create/list/load。
- Adapter dispatch 可从 execution plan 推导 `server_resource_registry_adapter`、`ssh_deployment_adapter`、Git provider release adapter 和 generic adapter。
- API 增加 `POST/GET /v1/projects/:project_id/operations/write-adapter-executions`。
- CLI 增加 `moyuan operations write-adapter-executions create|list ...`。
- Operations timeline 聚合 `write_adapter_execution`。
- Console Operations 视图新增 Write Adapter Execution 面板。
- 单测覆盖 preview dispatch、apply handoff、持久化、列表、timeline、API 和 CLI。

## 4. 验证要求

每完成一个 Phase 23 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
