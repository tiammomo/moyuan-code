# Phase 18 Release Readiness

状态：ready
责任角色：devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 18 已完成“生产运维闭环与策略化维护控制面”的第一批能力。系统已经可以把 release、deployment、admission、scheduler、risk、resource 和 post-deployment verification 聚合为统一运维事实源，并在 Console 中展示后端计算后的结果。

## 1. 完成范围

- `phase18-001 operations-timeline`：新增 operations timeline，统一聚合 release provider execution、deployment execution、rollback execution、monitor summary、deployment rehearsal、release admission、scheduler run、risk handoff/review、resource health scan、maintenance、lifecycle alert 和 server resource。
- `phase18-002 maintenance-policy-pack`：新增 server resource maintenance policy pack，支持维护窗口、冻结期、环境级允许动作、人工复核动作和可解释 decision。
- `phase18-003 post-deployment-smoke-monitor-loop`：新增 post-deployment verification，将 deployment execution、post-deployment history、monitor summary、rollback suggestion 和 evidence 串成线上验证事实。
- `phase18-004 server-resource-lifecycle-control`：增强 server resource 生命周期管理，记录 deployment readiness、最近部署引用、资源健康、到期、续费和退役风险。
- `phase18-005 console-operations-dashboard`：Console 展示 operations timeline、post-deployment verification、maintenance policy、resource lifecycle 和 deployment refs，并通过受控 API 触发低风险动作。
- `phase18-006 phase18-readiness`：状态、门禁结论、保留边界和下一阶段入口完成收口。

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
d7b4ef6 feat: add console operations dashboard
ffac8d8 feat: track server resource deployment lifecycle
4abed9d feat: add post deployment verification
9e2bedd feat: add maintenance policy pack
b5f2ac2 feat: add operations timeline
0e1f45a docs: open phase 18 operations control
```

## 3. 保留边界

- Operations timeline 是事实聚合，不改变 release、deployment、resource、repair 或 approval 状态。
- Maintenance policy 只输出 explainable decision，不能降低 approval、authz、quality、review、secret、provider 和 protected path 门禁。
- Post-deployment verification 失败只生成风险事实和复核建议，不自动执行生产修复、回滚或 repair attempt。
- Server resource readiness 是部署、维护和告警判断输入，不直接触发云厂商写操作、续费、关机、重启或退役。
- Console 只展示后端事实源并调用受控 API，不在前端重新计算 release admission、maintenance policy、deployment readiness 或 risk handoff。
- 生产真实写入仍默认关闭，必须由显式环境配置、权限、审批、质量门禁、secret gate 和 provider gate 同时放行。

## 4. 进入 Phase 19 的理由

Phase 18 已经把生产运维事实、维护策略、线上验证和服务器生命周期控制接入统一控制面。下一阶段应聚焦“受控自动化执行增强与生产可观测性深化”：

- 将 policy、readiness、verification 和 review queue 进一步沉淀为统一 rule evaluation 和 audit export。
- 增强长期运行任务的调度、重试、幂等和失败恢复，不依赖手动 CLI 串联。
- 为真实 Git/部署/云资源写入继续补充 dry-run、approval proof、provider evidence 和最小权限执行契约。
- 扩展 Console 的运维 drill-down、审计导出和异常处置入口，但保持所有决策来自后端事实源。

Phase 19 建议继续保持生产写入默认关闭，先完善执行可靠性、可观测性和审计闭环。
