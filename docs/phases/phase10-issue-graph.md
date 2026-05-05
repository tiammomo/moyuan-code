# Phase 10 实现 Issue Graph

状态：in_progress
责任角色：orchestrator_owner + backend_owner + frontend_owner + devops_owner + qa_owner
最后更新：2026-05-05

Phase 10 的目标是把 Phase 9 已具备的观测、解释和候选能力推进到“控制面自动化闭环增强”。本阶段仍保持真实生产写入默认关闭，但要让系统能按策略自动触发巡检、生成候选、进入人工复核，并在 Console 中形成可操作闭环。

## 1. Phase 10 目标

- 增加后台/手动控制循环入口，按项目配置触发资源生命周期扫描、Provider 运维刷新和必要的项目理解刷新。
- Operation repair candidate 进入审批、转换 issue 或创建受控 repair attempt 的流转。
- Release provider 扩展 branch、tag 和 workflow dispatch 的预览/受控执行计划。
- Deployment smoke、monitor 和 rollback 的检查模板、失败分级和环境策略更明确。
- Console 展示 Provider route candidates、repair candidate review 和控制循环运行历史。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase10-001` | `background-control-loop-scheduler` | completed | 控制循环 run、手动触发、资源 lifecycle scan、provider ops refresh、理解刷新 hook、审计日志 | Phase 9 readiness | `orchestrator_owner` + `backend_owner` | 可按项目触发一次 bounded control loop，并记录每个 step 的状态、证据和错误 |
| `phase10-002` | `operation-repair-candidate-review-flow` | completed | repair candidate approve/reject、转换 issue、创建 review-only repair attempt、审批约束 | `phase9-005` | `backend_owner` + `qa_owner` | 候选必须复核后才能进入 issue/run，不能默认自动修复 |
| `phase10-003` | `release-provider-branch-tag-workflow-preview` | completed | release branch、tag、workflow dispatch 的 provider plan 和 guardrail | Phase 8 release provider、`phase9-001` | `devops_owner` + `backend_owner` | branch/tag/workflow 动作可生成 preview，真实写入仍受 approval、secret、replay guard 控制 |
| `phase10-004` | `deployment-check-template-policy` | completed | smoke/monitor 模板、失败 severity、环境策略、rollback 建议条件 | `phase9-003` | `devops_owner` + `qa_owner` | deployment plan 可引用检查模板，history 能解释失败分级 |
| `phase10-005` | `console-route-repair-operator-surfaces` | planned | Provider route 候选矩阵、repair candidate 操作面、control loop 历史 | `phase9-004`,`phase10-001`,`phase10-002` | `frontend_owner` | Console 可查看候选、复核 repair candidate，并追踪控制循环执行历史 |

## 3. 建议执行顺序

1. 先做 `phase10-001`，把可重复运行的控制循环作为 Phase 10 的统一执行底座。
2. 再做 `phase10-002`，让 operation repair candidate 从“可看”进入“可治理流转”。
3. `phase10-003` 和 `phase10-004` 可并行推进，前者增强 release provider，后者增强部署检查策略。
4. `phase10-005` 最后收敛 Console 操作面，避免前端先行推导后端尚未稳定的状态机。

## 4. 收口规则

- 控制循环必须 bounded：每次 run 有 step 上限、timeout、去重和 audit trace。
- 自动循环只允许创建建议、候选、报告或低风险本地记录；生产写入、Git push、tag、workflow dispatch、部署执行仍需显式审批。
- 不记录 secret value、SSH key、API token、完整 stdout/stderr、完整模型响应或 provider 原始响应正文。
- Repair candidate 不能绕过 issue graph、write scope、quality gate、review 和 approval。
- Console 只调用后端事实源，不自行计算高风险执行结论。
