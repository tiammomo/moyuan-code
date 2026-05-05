# Phase 7 Release Readiness

状态：ready_for_next_phase
最后更新：2026-05-05

本文记录 Phase 7 受控真实外部执行准备的收口验证。Phase 7 的执行图见 [Phase 7 实现 Issue Graph](./phase7-issue-graph.md)，实施记录见 [Phase 7 实施记录](./phase7-next-development-plan.md)。

## 1. 验证范围

已完成能力：

- Release provider publish 已具备真实写入开关、approval consumption 和 replay guard；默认仍停在 preview-only。
- Deployment `ssh_execute` 已具备真实执行开关、资源校验、`auth_ref` 边界和命令 allowlist；默认仍阻断真实 SSH。
- Release provider execution 和 deployment execution 已写入统一 evidence chain。
- Provider telemetry 已接入 runtime execution、quality gate 和 provider route feedback。
- Console 已展示 operation history 和 execution detail，可追踪 deployment、release provider、visual render 与 evidence 关联。

不在本次收口内：

- 不真实调用 GitHub/Gitee release、tag、workflow dispatch API。
- 不建立真实 SSH session。
- 不执行生产部署、线上冒烟、生产监控或回滚命令。
- 不接入外部 observability vendor 或云厂商账单/额度 API。

## 2. 验证命令

后端：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
```

前端：

```bash
cd apps/console
npm run typecheck
npm run build
```

Git：

```bash
git diff --check
git status --short
```

## 3. 验证结论

- Phase 7 issue graph 中 `phase7-001` 到 `phase7-005` 均为 `completed`。
- 本轮收口中 `go test ./...`、`npm run typecheck`、`npm run build` 和 `git diff --check` 均已通过。
- 当前 main 已推送到 GitHub。
- Console dev server 已在 `http://127.0.0.1:3000` 可访问，页面可渲染 `Operation History` 和 `Execution Detail`。

## 4. 新增运行入口

Evidence：

```bash
moyuan evidence list --parent-type deployment_execution --limit 20
moyuan evidence show <evidence-id>
```

Provider telemetry：

```bash
moyuan model provider telemetry --provider codex_cli --limit 20
```

Release provider executions：

```http
GET /v1/projects/:project_id/release-provider-executions?limit=10
GET /v1/projects/:project_id/release-provider-executions/:execution_id
```

Console：

```bash
cd apps/console
npm run dev -- --port 3000
```

## 5. 产物位置

- `.moyuan/lifecycle/evidence/`
- `.moyuan/lifecycle/releases/provider-executions/`
- `.moyuan/lifecycle/deployments/executions/`
- `.moyuan/models/provider-telemetry.jsonl`
- `.moyuan/logs/`

## 6. 剩余风险

- Release provider 真实远程 adapter 仍未实现。
- SSH runner 仍未建立真实远程 session。
- 部署后的 smoke、monitor 和 rollback 仍停在 evidence/策略规划层。
- Provider telemetry 仍未接入真实模型账单、额度和外部质量采样。
- Console operation detail 目前使用 snapshot 汇总数据，尚未提供每个 operation 的独立 detail API 聚合视图。

## 7. 下一阶段入口

Phase 8 建议进入“受控外部执行 Beta”：

1. GitHub/Gitee release provider adapter 在写开关、approval 和 secret resolver 下执行最小真实写入。
2. SSH runner 在 allowlist、timeout、audit 和 evidence 约束下执行受控命令。
3. 部署后接入 smoke、monitor 和 rollback suggestion evidence。
4. Console 增加 operation detail 独立刷新和 evidence drill-down。
5. Provider telemetry 接入更真实的 quota/cost/quality feedback。

执行入口见 [Phase 8 实现 Issue Graph](./phase8-issue-graph.md) 和 [Phase 8 实施记录](./phase8-next-development-plan.md)。
