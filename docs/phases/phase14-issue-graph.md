# Phase 14 实现 Issue Graph

状态：in_progress
责任角色：release_owner + git_owner + devops_owner + backend_owner + frontend_owner + security_owner + qa_owner
最后更新：2026-05-05

Phase 14 的目标是把 Phase 13 的 release candidate 链路推进到“受控远程发布与部署执行”。本阶段继续保持生产写入默认关闭，所有 GitHub/Gitee 写入、release publish、deployment execution、线上 smoke 和 monitor 都必须经过 approval/authz、secret resolver、执行开关、证据和审计。

## 1. Phase 14 目标

- Release candidate 可以触发 approval-gated release provider publish，复用既有 release provider adapter、write switch、secret resolver 和 replay guard。
- Release candidate 可以衔接 PR/MR create 执行路径，远程创建默认关闭，先输出审批和阻断原因。
- Deployment plan 可以进入受控执行，区分 dry-run、ssh preview、local shell 和 ssh execute。
- 线上 smoke、monitor 和 rollback suggestion 结果能回写到 candidate/release 流水线。
- Console 能看见 candidate publish、PR/MR、deployment execution、smoke/monitor 和 rollback 的完整状态。

## 2. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase14-001` | `release-candidate-provider-publish-bridge` | completed | candidate -> release provider publish、approval request、write switch、secret resolver、provider execution evidence | Phase 13 readiness | `release_owner` + `security_owner` | release candidate 可调用受控 provider publish，默认只生成 approval/preview-only 阻断 |
| `phase14-002` | `release-candidate-pr-mr-create-bridge` | completed | candidate PR/MR plan -> Git Provider create、approval、remote write switch、replay guard | `phase14-001` | `git_owner` + `security_owner` | release branch 到 default branch 的 PR/MR 创建可受控执行，不绕过审批 |
| `phase14-003` | `candidate-deployment-execution-bridge` | completed | candidate deployment plan -> deployment execution、dry-run/ssh preview/local shell/ssh execute、approval | `phase13-004` | `devops_owner` + `backend_owner` | deployment execution 可从 candidate 链路触发，生产真实执行仍默认阻断 |
| `phase14-004` | `post-deploy-smoke-monitor-feedback` | planned | smoke、monitor、rollback suggestion、post deployment history 与 candidate/release 状态关联 | `phase14-003` | `devops_owner` + `qa_owner` | 线上检查结果能回写为证据，失败时给出 rollback 建议 |
| `phase14-005` | `console-release-execution-surface` | planned | Console 展示 provider publish、PR/MR create、deployment execution、smoke/monitor 和 rollback | `phase14-001`,`phase14-004` | `frontend_owner` | Console 可见执行流水线，但不自行计算发布或部署结论 |

## 3. 建议执行顺序

1. 先做 `phase14-001`，把 release candidate 接入已有 release provider publish 能力，确认审批、写开关、secret 和 replay guard 不被绕过。
2. 再做 `phase14-002`，把 release branch 的 PR/MR 创建放到独立门禁中，避免与 tag/release publish 混成一个动作。
3. `phase14-003` 接 deployment execution，只允许后端执行受控 mode，不在 Console 或脚本里拼接生产命令。
4. `phase14-004` 把执行后的 smoke、monitor 和 rollback suggestion 回写为证据。
5. `phase14-005` 最后接 Console，前端只展示后端事实源和受控操作结果。

## 4. 强制边界

- 没有 release candidate ready，不允许进入 provider publish、PR/MR create 或 deployment execution。
- 没有本地 release branch apply，不允许远程 push/tag/publish。
- 没有 completed provider preview，不允许执行 provider publish。
- 没有 approval、secret resolver 和对应写开关，不允许真实远程写入。
- production deployment execution 默认阻断，必须由环境策略、approval 和执行开关共同放行。
- Console 不自行计算 release readiness、provider readiness 或 deploy readiness。
