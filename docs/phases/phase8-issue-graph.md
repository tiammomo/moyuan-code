# Phase 8 实现 Issue Graph

状态：in_progress
责任角色：release_manager + devops_owner + security_owner + provider_owner + frontend_owner + qa_owner
最后更新：2026-05-05

Phase 8 的目标是从“受控真实外部执行准备”进入“受控外部执行 Beta”。本阶段可以实现最小真实外部写入和远程命令执行，但必须继续由写开关、authz、approval、secret resolver、allowlist、audit、evidence 和回滚策略共同约束。

## 1. Phase 8 目标

- GitHub/Gitee release provider adapter 支持最小真实写入：create release 可受控执行，branch/tag/workflow 先显式跳过。
- SSH runner 支持受控远程命令执行：命令 allowlist、timeout、stdout/stderr 脱敏、evidence 和失败恢复。
- 部署后形成 smoke、monitor 和 rollback suggestion 的结构化 evidence。
- Console 能从 operation history drill down 到单个 operation detail 和 evidence chain。
- Provider telemetry 逐步接入真实 quota/cost/quality feedback，而不是只依赖本地状态。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase8-001` | `release-provider-real-adapter-beta` | completed | GitHub/Gitee release provider adapter 最小真实写入 | Phase 7 readiness | `release_manager` + `security_owner` | 写开关、approval、secret resolver 和 replay guard 全部满足后才执行远程写入 |
| `phase8-002` | `ssh-runner-controlled-execution` | completed | SSH runner 真实连接、命令执行、timeout、脱敏和 evidence | `phase8-001` 可并行准备 | `devops_owner` + `security_owner` | allowlist 命令能执行，非 allowlist 阻断，stdout/stderr 不泄密 |
| `phase8-003` | `post-deploy-smoke-monitor-evidence` | planned | smoke、monitor、health check 和结果 evidence | `phase8-002` | `qa_owner` + `devops_owner` | 部署后能生成 smoke/monitor evidence，失败能阻断发布完成 |
| `phase8-004` | `rollback-suggestion-and-runbook` | planned | rollback suggestion、runbook、手动确认和回滚 evidence | `phase8-003` | `release_manager` + `devops_owner` | 失败部署能生成可审查回滚建议，不默认自动回滚生产 |
| `phase8-005` | `console-operation-drilldown` | planned | Console operation detail 独立 API、evidence drill-down 和刷新 | `phase7-005` | `frontend_owner` | 用户能从 operation history 打开完整 execution/evidence detail |
| `phase8-006` | `provider-real-quota-cost-feedback` | planned | Provider quota/cost/quality feedback 接入真实或半真实来源 | `phase7-004` | `provider_owner` | route decision 能读取更可信的 quota/cost/quality signals |

## 3. 建议执行顺序

1. `phase8-001` 已完成，release provider adapter 的 create release 真实写入边界已最小可验证。
2. `phase8-002` 已完成，真实 SSH runner 已接入 secret resolver、allowlist、timeout、脱敏和 smoke/monitor/rollback evidence。
3. `phase8-003` 在 SSH runner 可控后推进，形成部署后的自动验证。
4. `phase8-004` 在 smoke/monitor 有结果后推进，不提前自动回滚。
5. `phase8-005` 在 operation 数据模型稳定后增强 Console drill-down。
6. `phase8-006` 可穿插推进，但不能阻塞核心 release/deploy Beta。

## 4. 收口规则

- 所有真实外部写入都必须默认关闭，并通过显式环境开关打开。
- 所有高风险真实执行都必须绑定 approval record，且 approval 不可重放。
- 所有 secret value 只能在 adapter 执行时短暂注入，不能写入日志、Memory、prompt、execution 或 Console。
- 所有真实执行必须生成 evidence，且 evidence 只保存引用和摘要。
- Console 仍不自行判定成功，只展示后端事实源。
