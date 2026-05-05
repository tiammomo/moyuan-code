# Phase 17 Release Readiness

状态：ready
责任角色：release_owner + devops_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 17 已完成“发布准入策略包、演练调度与风险修复 drill-down”的第一批能力。系统已经可以把 release admission 从固定判断升级为可解释 policy pack，再用 bounded scheduler 创建 deployment rehearsal/admission，最后把 blocked/manual 风险进入 review queue，并在 Console 展示 policy、scheduler 和 risk review 的事实源。

## 1. 完成范围

- `phase17-001 release-admission-policy-pack`：新增内置 release admission policy pack，支持环境规则、matched rules、policy decision 和 CLI/API 查询。
- `phase17-002 bounded-rehearsal-scheduler`：新增一次性有界 scheduler，可基于 release candidate、deployment 或 execution 创建 rehearsal/admission，并记录 created/skipped/blocked/manual 结果。
- `phase17-003 risk-review-queue`：新增 deployment risk review queue，支持 `approved`、`rejected`、`deferred`，并写入 evidence、run log 和审计事实。
- `phase17-004 console-policy-risk-drilldown`：Console 展示 release admission policy、scheduler run、risk handoff、review queue 和 latest review，且只消费后端事实源。
- `phase17-005 phase17-readiness`：状态、门禁结论、保留边界和下一阶段入口完成收口。

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
bad1e6c feat: add console policy risk drilldown
f9414fe feat: add deployment risk review queue
12f7e14 feat: add bounded rehearsal scheduler
d76010f feat: add release admission policy pack
d2a48c9 docs: open phase 17 policy automation
```

## 3. 保留边界

- Policy pack 只能追加或解释准入规则，不能降低 approval、authz、quality、review、secret 和 protected path 门禁。
- Bounded scheduler 只执行一次 run，不启动常驻后台任务，不执行真实生产命令。
- Rehearsal/admission 仍只聚合事实源和输出结论，不修改 release candidate、Git、服务器或部署状态。
- Risk review 只记录人工复核结论和下一步建议，不直接执行 repair attempt 或生产命令。
- Console 只展示后端事实源并触发受控 API，不在前端重新计算 policy、scheduler 或 review 决策。
- 生产真实写入仍默认关闭，必须由显式环境配置、权限、审批和 provider gate 同时放行。

## 4. 进入 Phase 18 的理由

Phase 17 已经把发布准入、演练调度和风险复核的事实链路打通。下一阶段应聚焦“生产运行控制面与长期维护闭环”：

- 将 release admission、scheduler、review queue 和 deployment monitor 串成可审计的 operations dashboard。
- 增强 server resource、deployment execution、monitor summary 和 repair review 之间的长期维护关系。
- 补齐生产环境的受控更新、线上冒烟、监控告警、维护窗口和异常回滚建议。
- 将 Phase 17 的自动化边界沉淀为可配置 policy，而不是散落在单个命令或面板动作中。

Phase 18 建议聚焦“生产运维闭环与策略化维护控制面”，继续保持生产写入默认关闭。
