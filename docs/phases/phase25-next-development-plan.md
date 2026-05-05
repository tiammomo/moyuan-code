# Phase 25 实施记录

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + qa_owner + security_owner
最后更新：2026-05-05

本文记录 Phase 25 的实际执行顺序。Phase 25 的入口以 [Phase 25 实现 Issue Graph](./phase25-issue-graph.md) 为准。

## 1. 阶段入口

Phase 24 已完成 `server_resource_registry_adapter` apply receipt。Phase 25 将 write adapter contract 扩展到 SSH 部署执行前审查：读取 deployment execution 的 `RemotePlan`，检查 target、AuthRef 和 command sandbox，并把 rollback plan/runbook 绑定进 adapter execution。

## 2. Phase 25 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase25-001` | `ssh-adapter-sandbox` | completed | SSH adapter execution 前置 sandbox | target/auth/command 可审计，危险命令阻断 |
| P0 | `phase25-002` | `ssh-rollback-binding` | completed | 绑定 deployment rollback plan/runbook | 缺失 rollback 被阻断或人工 |
| P1 | `phase25-003` | `console-adapter-sandbox-view` | completed | Console 展示 sandbox/rollback 摘要 | 前端只读展示事实源 |
| P1 | `phase25-004` | `phase25-readiness` | completed | Phase 25 收口 | 全量门禁通过 |

## 3. 完成记录

- `WriteAdapterExecution` 增加 `sandbox_results` 和 `rollback_binding`。
- `ssh_deployment_adapter` 会加载 deployment execution，并消费 `RemotePlan.Targets`。
- Sandbox 检查覆盖 target host、AuthRef 引用安全性、preview command 控制字符和真实命令 allowlist。
- Rollback binding 优先绑定 deployment rollback plan；若已有 rollback suggestion，则要求 structured runbook。
- Console Operations 面板展示 sandbox 数量、rollback decision 和 no remote write 标记。

## 4. 验证要求

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
