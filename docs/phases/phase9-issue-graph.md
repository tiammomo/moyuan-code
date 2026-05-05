# Phase 9 实现 Issue Graph

状态：in_progress
责任角色：devops_owner + backend_owner + frontend_owner + provider_owner + qa_owner
最后更新：2026-05-05

Phase 9 的目标是把 Phase 8 的受控外部执行 Beta 推进到“生产运维控制面增强”。本阶段继续保持真实生产动作默认关闭，但要让 operation detail、服务器资源、部署后检查、provider 反馈和 self-repair 更可观测、更可追踪。

## 1. Phase 9 目标

- Console 和 API 能按 operation id 聚合 release/deployment execution、evidence、artifact 和状态摘要。
- 服务器资源生命周期管理覆盖到期时间、维护窗口、环境分层、健康状态和续期提醒。
- 部署后的 smoke、monitor 和 rollback runbook 能形成更完整的生产运维记录。
- Provider budget、quota、cost 和 quality feedback 能进入更明确的 route explanation。
- Runtime failure、quality failure 和线上 smoke failure 能触发可审查 self-repair candidate。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase9-001` | `operation-detail-aggregation-api` | completed | 聚合 operation、execution、evidence、artifact 和状态摘要 | Phase 8 readiness | `backend_owner` + `frontend_owner` | Console 可按 operation id 获取完整 detail，而不是只依赖 snapshot 拼装 |
| `phase9-002` | `server-resource-lifecycle-alerts` | completed | 服务器到期、维护窗口、环境分层、续期提醒和资源健康摘要 | `phase9-001` 可并行 | `devops_owner` | 资源列表可标记 expiring/expired/maintenance_due 并写入 audit |
| `phase9-003` | `deployment-monitor-history` | planned | smoke/monitor 历史、失败分类、rollback runbook 状态追踪 | `phase8-003`,`phase8-004` | `qa_owner` + `devops_owner` | 部署后检查可按 execution 查询历史与失败原因 |
| `phase9-004` | `provider-route-explanation-v2` | planned | provider budget/quota/cost/quality feedback 的路由解释增强 | `phase8-006` | `provider_owner` | route decision 给出候选 provider 的 skipped/selected signals |
| `phase9-005` | `self-repair-candidate-from-operations` | planned | 从 runtime/quality/deploy failure 生成 repair candidate | `phase9-001`,`phase9-003` | `backend_owner` + `qa_owner` | 失败事件能生成可审查 repair plan，不自动越权修改代码 |

## 3. 建议执行顺序

1. 先做 `phase9-001`，因为 Console、部署历史和 self-repair 都需要稳定的 operation detail 聚合入口。
2. `phase9-002` 可以与 `phase9-001` 并行准备，但资源提醒最终也应接入 operation/detail 或日志视图。
3. `phase9-003` 扩展部署后检查历史，为生产监控和 rollback runbook 状态追踪打基础。
4. `phase9-004` 在 provider feedback 已可累计后，增强 route explanation 的可解释性。
5. `phase9-005` 最后接入 self-repair candidate，避免在 evidence/detail 不完整时过早自动修复。

## 4. 收口规则

- Phase 9 仍不默认打开 production real execution。
- 所有自动提醒、repair candidate 和 route explanation 都只能生成建议或受控记录，不能绕过 approval、quality gate、review 或 deployment gate。
- 服务器资源、provider telemetry 和 operation detail 不能记录 secret value、SSH key、API token、完整 stdout/stderr 或模型响应正文。
- Console 只展示后端事实源，不自行推导高风险执行结论。
