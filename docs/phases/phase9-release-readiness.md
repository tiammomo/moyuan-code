# Phase 9 Release Readiness

状态：ready_for_next_phase
最后更新：2026-05-05

本文记录 Phase 9 生产运维控制面增强的收口验证。Phase 9 的执行图见 [Phase 9 实现 Issue Graph](./phase9-issue-graph.md)，实施记录见 [Phase 9 实施记录](./phase9-next-development-plan.md)。

## 1. 验证范围

已完成能力：

- Operation detail 聚合 API 可按 operation type/id 返回 execution、evidence chain、artifact references 和状态摘要。
- 服务器资源生命周期 scan 可记录 expiring、expired、maintenance due 和 health attention，并写入 audit。
- Deployment post-deployment history 可按 execution 查询 smoke/monitor/rollback 结果、失败分类和 evidence/artifact 引用。
- Provider route explanation v2 可返回 selected/skipped/blocked candidates、selection signal 和候选计数。
- 失败 operation 可生成 review-only self-repair candidate，默认创建 `candidate_review_required` repair plan，不自动运行修复。
- Console 已展示 operation detail、server lifecycle alerts、deployment monitor history、provider telemetry 和 operation repair candidates。

不在本次收口内：

- 不自动执行生产 rollback。
- 不默认打开 production real execution。
- 不自动批准 operation repair candidate。
- 不调用云厂商真实账单、资源续费或监控平台写接口。
- 不扩大 release provider 的 branch push、tag push 或 workflow dispatch 权限。

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

- Phase 9 issue graph 中 `phase9-001` 到 `phase9-005` 均为 `completed`。
- 本轮收口中 `go test ./...`、`npm run typecheck`、`npm run build` 和 `git diff --check` 均已通过。
- 当前 main 已推送到 GitHub。
- 所有新增能力都是可观测、可审查或受控建议；没有新增默认真实生产写入。

## 4. 新增运行入口

Operation detail：

```bash
GET /v1/projects/:project_id/operations/:operation_type/:operation_id
```

Server lifecycle：

```bash
POST /v1/projects/:project_id/resources/lifecycle/scan
GET /v1/projects/:project_id/resources/lifecycle-alerts
```

Deployment monitor history：

```bash
GET /v1/projects/:project_id/deployment-monitor-history
GET /v1/projects/:project_id/deployment-executions/:execution_id/post-deployment-history
```

Provider route explanation：

```bash
POST /v1/projects/:project_id/provider-route
```

Operation repair candidate：

```bash
POST /v1/projects/:project_id/operations/:operation_type/:operation_id/repair-candidate
GET /v1/projects/:project_id/repair/operation-candidates
```

## 5. 产物位置

- `.moyuan/lifecycle/evidence/`
- `.moyuan/lifecycle/deployments/post-deployment-history/`
- `.moyuan/resources/lifecycle-alerts.jsonl`
- `.moyuan/resources/lifecycle-scans/`
- `.moyuan/models/provider-telemetry.jsonl`
- `.moyuan/repair/operation-candidates/`
- `.moyuan/logs/`

## 6. 剩余风险

- 资源 lifecycle scan、provider ops refresh 和 operation repair candidate 仍需要后台调度或用户触发。
- Operation repair candidate 还没有审批后流转为 issue/run 的完整编排。
- Release provider 仍只支持 create release，branch/tag/workflow 动作仍受控 skipped。
- Deployment smoke/monitor 配置还偏轻，需要更清晰的检查模板、窗口和失败分级。
- Provider route explanation 已有后端字段，但 Console 还没有专门的候选矩阵视图。

## 7. 下一阶段入口

Phase 10 建议进入“控制面自动化闭环增强”：

1. 增加后台调度入口，自动运行 resource lifecycle scan、provider ops refresh 和必要的 project comprehension refresh。
2. 给 operation repair candidate 增加 review/approval 流转，允许创建 issue 或受控 repair attempt。
3. 扩展 release provider 的 branch/tag/workflow 动作，但继续保持 approval、secret 和 replay guard。
4. 抽象 deployment smoke/monitor 配置模板和失败分级。
5. 在 Console 增加 provider route explanation 和 repair candidate 的可操作视图。

执行入口见 [Phase 10 实现 Issue Graph](./phase10-issue-graph.md) 和 [Phase 10 实施记录](./phase10-next-development-plan.md)。
