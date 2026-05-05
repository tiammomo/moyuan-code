# Phase 13 实现 Issue Graph

状态：in_progress
责任角色：release_owner + git_owner + devops_owner + backend_owner + frontend_owner + security_owner + qa_owner
最后更新：2026-05-05

Phase 13 的目标是把 Phase 12 的 release batch readiness 推进到“Release Candidate 远程发布与部署交接”。本阶段继续保持 production write 默认关闭，先把本地 integration/release batch 准备结果转成可审计的 release candidate，再按审批、权限和执行开关逐步衔接 release branch、Git Provider、tag、部署计划和 Console 可见性。

## 1. Phase 13 目标

- release batch plan 可以生成 release candidate 事实源，保留 integration apply、source branch、release branch、version、provider、风险和部署目标。
- release candidate 可以预览 release branch、PR/MR、tag、release provider 和 workflow dispatch，不默认远程写入。
- local release branch apply 必须受审批和环境开关控制，并与 integration apply 一样只做本地 Git ref 更新。
- release candidate 可以生成 deployment handoff plan，默认 dry-run，后续再进入服务器部署、线上冒烟和监控。
- Console 能看见 release candidate 从 batch readiness 到 branch/provider/deployment 的完整链路。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase13-001` | `release-candidate-plan-from-batch` | completed | release candidate 事实源、从 release batch plan 读取 source/release branch/version、provider/remote/deploy target 摘要 | Phase 12 readiness | `release_owner` + `backend_owner` | suggested release batch 可生成 release candidate plan，不执行 Git 或远程写入 |
| `phase13-002` | `guarded-local-release-branch-apply` | planned | approval/env gated local release branch update、write evidence、blocked reason | `phase13-001` | `git_owner` + `security_owner` | 只有审批和开关满足时，才把 source integration branch 固化为本地 release branch |
| `phase13-003` | `release-candidate-provider-preview` | planned | GitHub/Gitee provider preview、PR/MR plan、tag/release/workflow guarded actions | `phase13-001`,`phase13-002` | `git_owner` + `release_owner` | release candidate 可生成远程发布预览和 PR/MR handoff，不默认 push/tag/publish |
| `phase13-004` | `deployment-handoff-from-release-candidate` | planned | 根据 release candidate 生成 deployment plan、环境/服务器资源引用、smoke/monitor 模板引用 | `phase13-001` | `devops_owner` | release candidate 可进入部署 dry-run 和后续线上检查链路 |
| `phase13-005` | `console-release-candidate-surface` | planned | Console 展示 candidate、branch apply、provider preview、deployment handoff 和阻断原因 | `phase13-001`,`phase13-004` | `frontend_owner` | Console 可见 release candidate 全链路并只调用后端受控 API |

## 3. 建议执行顺序

1. 先做 `phase13-001`，因为后续 branch、provider 和 deployment 都需要统一的 release candidate 事实源。
2. `phase13-002` 只处理本地 release branch apply，避免一开始混入远程 push/tag。
3. `phase13-003` 在本地 release branch 准备后接 Git Provider 和 release provider preview。
4. `phase13-004` 与 provider preview 可局部并行，但 deployment handoff 只消费 candidate，不直接依赖远程 publish 完成。
5. `phase13-005` 最后接 Console，前端只展示后端事实源和受控操作结果。

## 4. 收口规则

- 没有 suggested release batch，不允许创建 release candidate。
- 没有 approval/authz 和 `MOYUAN_ALLOW_RELEASE_BRANCH_APPLY=1`，不允许更新本地 release branch。
- 没有 release branch readiness，不允许远程 push/tag/publish。
- GitHub/Gitee 写入必须继续走 provider secret resolver、approval consumption、write switch 和审计日志。
- deployment handoff 默认 dry-run；生产执行必须继续走 deployment approval、服务器资源策略、线上 smoke/monitor 和 rollback suggestion。
- Console 不自行计算 release readiness、provider readiness 或 deploy readiness。
