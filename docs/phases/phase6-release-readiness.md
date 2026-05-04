# Phase 6 Release Readiness

状态：ready_for_next_phase
最后更新：2026-05-05

本文记录 Phase 6 approval consumption、部署 SSH preview、release provider adapter、provider telemetry 和 Console schema forms 的收口验证。Phase 6 的执行图见 [Phase 6 实现 Issue Graph](./phase6-issue-graph.md)，实施记录见 [Phase 6 实施记录](./phase6-next-development-plan.md)。

## 1. 验证范围

已完成能力：

- Approval record 已支持消费和重放防护，真实外部写操作不能重复使用同一 approval。
- Git Provider PR/MR create 已在真实写入路径前消费 approval；preview-only 不消费。
- Deployment execute 已支持 `ssh_preview`，能基于 server resources 生成远程目标、命令和 auth reference 预览；`ssh_execute` 仍默认阻断。
- Release provider adapter 已支持 preview/publish execution，能记录 branch/tag/release/workflow action plan；publish 缺少 approval 时会生成 approval record。
- Provider ops 已记录 health/quota/usage/cost telemetry，路由决策会返回 `signals`。
- Console 已支持多视图切换、受控表单必填字段预检、provider telemetry 展示和 release provider preview/publish 操作入口。

不在本次收口内：

- 不真实调用 GitHub/Gitee release、tag、workflow dispatch API。
- 不执行真实 SSH command、云厂商部署、云资源变更或生产投产。
- 不接入云厂商账单、模型服务商真实用量 API 或 observability vendor。
- Console 当前仍是工作台多视图，不是完整多 URL 页面和组织级权限 UI。

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

- Phase 6 issue graph 中 `phase6-001` 到 `phase6-005` 均为 `completed`。
- 本轮收口中 `go test ./...`、`npm run typecheck`、`npm run build` 和 `git diff --check` 均已通过。
- 所有新增外部执行入口仍保持 preview/dry-run/approval/authz/secret resolver 的安全边界。
- Console 只做字段预检和后端 API 调用，不自行构造权威状态。
- 当前 main 已推送到 GitHub。

## 4. 新增运行入口

Approval：

```bash
moyuan approval decide <approval-id> --decision approved --decided-by release-manager --reason "release gates passed"
```

Deployment：

```bash
moyuan deploy execute <deployment-id> --mode ssh_preview
moyuan deploy execute <deployment-id> --mode ssh_execute
```

Release provider：

```bash
moyuan release provider preview <release-id>
moyuan release provider publish <release-id> --approved --approval-id <approval-id>
moyuan release provider execution <execution-id>
```

Provider telemetry：

```bash
moyuan model provider telemetry --limit 20
moyuan model route --strategy backend-safe
```

Console：

```bash
cd apps/console
npm run dev
```

## 5. 产物位置

- `.moyuan/lifecycle/approvals/`
- `.moyuan/lifecycle/releases/provider-executions/`
- `.moyuan/deployments/`
- `.moyuan/models/provider-telemetry.jsonl`
- `.moyuan/models/providers.json`
- `.moyuan/logs/`

## 6. 剩余风险

- Release provider publish 仍停在 preview-only 降级，没有真实远程写入 adapter。
- Deployment SSH 仍只做 preview，尚未实现受控 SSH runner、远程烟测和回滚命令执行。
- Provider telemetry 仍依赖本地 ops update/refresh，未接入真实账单、额度和模型质量采样。
- Console schema-aware forms 先落地必填字段预检，尚未接入后端导出的完整 schema metadata。

## 7. 下一阶段入口

Phase 7 进入“受控真实外部执行准备”：

1. Release provider publish 增加真实写入开关、approval consumption 和 replay guard。
2. Deployment SSH executor 增加命令 allowlist、secret resolver 注入、远程执行记录和默认阻断策略。
3. 发布/部署流水线增加 post-action smoke、monitor 和 rollback evidence 的统一结果模型。
4. Provider telemetry 与 runtime execution 结果联动，形成失败、成本和降级反馈闭环。
5. Console 增加 execution detail、operation history 和更完整的 schema-driven forms。

执行入口见 [Phase 7 实现 Issue Graph](./phase7-issue-graph.md) 和 [Phase 7 实施记录](./phase7-next-development-plan.md)。
