# Phase 19 Release Readiness

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 19 已完成“受控自动化执行增强与生产可观测性深化”的第一批能力。系统现在可以把 operations timeline 导出为审计报告，把 policy/readiness/verification/review 结论沉淀为 decision ledger，用 durable control runner 执行低风险长期任务，并通过 provider write proof 和 Console drill-down 展示真实写入的放行条件。

## 1. 完成范围

- `phase19-001 operations-audit-export`：新增 operations audit export，支持 JSON/Markdown、过滤、verification/resource refs/evidence refs 聚合和 secret redaction。
- `phase19-002 decision-ledger`：新增 decision ledger，统一聚合 release admission、maintenance policy、resource readiness、post-deployment verification、deployment risk handoff/review。
- `phase19-003 durable-control-runner`：新增 durable control runner 字段、idempotency index、retry budget、manual required 状态和低风险 step。
- `phase19-004 provider-write-proof-contract`：新增 write proof report，统一解释 release provider execution、deployment execution、resource maintenance 的 dry-run、write flag、approval、secret/auth ref、evidence、least privilege 和 replay guard。
- `phase19-005 console-observability-drilldown`：Console snapshot 和 Operations/Audit 视图展示 audit export、decision ledger、write proof 和最新 control runner 摘要。
- `phase19-006 phase19-readiness`：状态、门禁结论、保留边界和 Phase 20 入口完成收口。

## 2. 验证结论

最新收口门禁：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```

结论：通过。

最近提交：

```text
91a6dde feat: add console observability drilldown
8d04cdb feat: add provider write proof contract
a739007 feat: add durable control runner
b8a7d9b feat: add operations decision ledger
56301d8 feat: add operations audit export
3f358d9 docs: open phase 19 automation observability
```

## 3. 保留边界

- Audit export、decision ledger 和 write proof 都是事实聚合与解释层，不修改 release、deployment、resource、repair、approval 或 provider 状态。
- Durable control runner 只能调用已接入的低风险 step；不能绕过原模块的 authz、approval、secret、provider、quality 和 protected path 门禁。
- Provider write proof 只解释放行条件，不开启真实写入开关，不读取 secret 明文，不执行 Git/provider/SSH/cloud 写入。
- Console 只展示后端事实源和后端返回状态，不在前端重新计算准入、维护窗口、资源就绪、verification 或 write proof 结论。
- 生产真实写入仍默认关闭，必须由显式写开关、授权、审批、secret 引用、provider gate、质量门禁和 evidence 同时满足后才允许进入后续真实执行阶段。

## 4. Phase 20 入口建议

Phase 19 已经把长期自动化、审计、决策账本、写入证明和 Console 可观测性打通。后续 Phase 20 建议聚焦“受控生产写入演练与远程运维执行增强”：

- 将 write proof 与具体 release/deployment/resource 操作的 admission policy 串联，形成可执行前置准入。
- 为真实 Git/provider/SSH/cloud 写入增加更细的 least-privilege 配置和 provider-specific proof。
- 增强 control runner 的任务队列、调度窗口、失败恢复和人工复核 handoff。
- 扩展 Console 的单条 proof/detail drill-down 和只读导出入口。

当前收口不启动 Phase 20 实现。
