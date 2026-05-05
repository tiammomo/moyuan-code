# Phase 8 Release Readiness

状态：ready_for_next_phase
最后更新：2026-05-05

本文记录 Phase 8 受控外部执行 Beta 的收口验证。Phase 8 的执行图见 [Phase 8 实现 Issue Graph](./phase8-issue-graph.md)，实施记录见 [Phase 8 实施记录](./phase8-next-development-plan.md)。

## 1. 验证范围

已完成能力：

- Release provider adapter 已支持 GitHub/Gitee create release 的最小真实写入，受 `MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE=1`、approval、secret resolver 和 replay guard 共同约束。
- Deployment `ssh_execute` 已支持受控 SSH 命令执行，受 `MOYUAN_ALLOW_SSH_EXECUTE=1`、资源校验、命令 allowlist、timeout 和输出脱敏约束。
- 部署后会生成 execution、smoke、monitor 和 rollback evidence；失败场景会生成 rollback suggestion 和人工审查 runbook。
- Console Operation Detail 已可展开 evidence chain、artifact path，并支持刷新当前 snapshot。
- Provider telemetry 已接入 runtime token 估算、quota 扣减、可配置 token 单价成本估算和 quality feedback signal。

不在本次收口内：

- 不自动执行 branch push、tag push 或 workflow dispatch。
- 不默认打开 production 真实部署。
- 不自动执行生产回滚命令；rollback runbook 仍要求人工审查。
- 不调用云厂商账单 API 或供应商额度 API；当前成本和额度以本地配置、runtime feedback 和 ops snapshot 为准。
- Console 仍缺少单个 operation detail 聚合 API，当前由 snapshot 组合展示。

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

- Phase 8 issue graph 中 `phase8-001` 到 `phase8-006` 均为 `completed`。
- 本轮收口中 `go test ./...`、`npm run typecheck`、`npm run build` 和 `git diff --check` 均已通过。
- 当前 main 已推送到 GitHub。
- 外部真实写入能力仍默认关闭，必须显式打开环境开关并满足 approval/authz/secret 条件。

## 4. 新增运行入口

Release provider：

```bash
MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE=1 moyuan release provider publish <release-id> --approved --approval-id <approval-id>
```

Deployment：

```bash
MOYUAN_ALLOW_SSH_EXECUTE=1 moyuan deploy execute <deployment-id> --mode ssh_execute --approved --approval-id <approval-id>
```

Evidence：

```bash
moyuan evidence list --parent-type deployment_execution --limit 20
moyuan evidence show <evidence-id>
```

Provider telemetry：

```bash
moyuan model provider ops codex_cli --limit-tokens 100000 --input-token-cost-per-1k 0.01 --output-token-cost-per-1k 0.03
moyuan model provider telemetry --provider codex_cli --limit 20
```

Console：

```bash
cd apps/console
npm run dev -- --port 3000
```

## 5. 产物位置

- `.moyuan/lifecycle/releases/provider-executions/`
- `.moyuan/lifecycle/deployments/executions/`
- `.moyuan/lifecycle/deployments/rollback-runbooks/`
- `.moyuan/lifecycle/evidence/`
- `.moyuan/models/provider-telemetry.jsonl`
- `.moyuan/logs/`

## 6. 剩余风险

- Console operation detail 仍缺少后端聚合 API，复杂执行链需要前端从多个列表中拼装。
- 生产真实部署仍未开放，当前默认阻断 production real execution。
- 服务器资源到期、维护窗口和生产监控仍需要更主动的后台检查与提醒。
- Provider cost/quota 仍是本地估算和配置驱动，不等同于供应商真实账单。
- Release provider 仍只完成 create release，branch/tag/workflow 自动化需要后续分阶段扩展。

## 7. 下一阶段入口

Phase 9 建议进入“生产运维控制面增强”：

1. 新增 operation detail 聚合 API，让 Console 能按 operation id 读取 execution、evidence、artifact 和后续日志摘要。
2. 增强服务器资源生命周期管理，包括到期时间、维护窗口、资源健康和续期提醒。
3. 增强部署后的 smoke/monitor 运行记录和生产监控摘要。
4. 把 provider budget、quota 和 quality feedback 纳入更明确的路由策略解释。
5. 推进 self-repair 使用反馈，让运行过程发现的问题能形成可审查修复任务。

执行入口见 [Phase 9 实现 Issue Graph](./phase9-issue-graph.md) 和 [Phase 9 实施记录](./phase9-next-development-plan.md)。
