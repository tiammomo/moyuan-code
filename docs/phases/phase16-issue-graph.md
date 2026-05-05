# Phase 16 实现 Issue Graph

状态：in_progress
责任角色：devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 16 的目标是把 Phase 15 已完成的 deployment execution、rollback execution 和 monitor summary 组织成“部署演练、运行风险闭环与发布准入增强”。本阶段继续保持生产真实写入默认关闭，优先让一次发布或部署可以形成可复现 rehearsal record，并把 monitor risk 回流到 release admission 和 self-repair。

## 1. Phase 16 目标

- Deployment rehearsal 串联 release candidate、deployment plan、deployment execution、rollback preview、monitor summary 和 evidence。
- Release admission 能消费 monitor summary、candidate feedback、resource health 和 rollback signal，给出是否允许继续发布/部署的后端事实源。
- Monitor critical 或 rollback required 能生成 repair candidate 或维护建议，进入自我修复/人工复核队列。
- Console 展示 rehearsal timeline、risk gate、repair handoff 和 evidence，不在前端自行计算准入结论。
- Phase 16 完成后给出 readiness，明确哪些能力可以进入受控投产演练，哪些仍保持 preview-only。

## 2. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase16-001` | `deployment-rehearsal-controller` | completed | 新增 deployment rehearsal 记录，串联 candidate、deployment、execution、rollback preview、monitor summary 和 evidence | Phase 15 readiness | `devops_owner` + `backend_owner` | 可通过 CLI/API 创建和查询一次 rehearsal，真实写入仍不发生 |
| `phase16-002` | `release-admission-risk-gate` | completed | release/deploy 准入读取 monitor summary、candidate feedback、resource health、rollback signal | `phase16-001` | `release_owner` + `qa_owner` | 输出 allow/block/manual 的后端准入结论和原因 |
| `phase16-003` | `monitor-risk-repair-bridge` | planned | critical monitor 或 rollback required 生成 repair candidate/maintenance handoff | `phase16-001`,`phase16-002` | `qa_owner` + `backend_owner` | 风险能进入自修复或人工复核队列，不自动改生产 |
| `phase16-004` | `console-rehearsal-risk-surface` | planned | Console 展示 rehearsal timeline、admission gate 和 repair handoff | `phase16-001`,`phase16-003` | `frontend_owner` | Console 只展示后端事实源和触发低风险 preview |
| `phase16-005` | `phase16-readiness` | planned | 收口验证、文档回写、剩余风险和下一阶段入口 | `phase16-004` | `release_owner` + `security_owner` | 全量门禁通过，投产演练边界清晰 |

## 3. 建议执行顺序

1. 先做 `phase16-001`，建立 rehearsal record 这个贯穿发布、部署、回退和 monitor 的事实对象。
2. 再做 `phase16-002`，让 release admission 不再只看单点结果，而是消费 rehearsal 与风险摘要。
3. `phase16-003` 把风险转为 repair/maintenance handoff，接入项目越用越完善的能力闭环。
4. `phase16-004` 最后接 Console，避免前端先固化未稳定的数据模型。
5. `phase16-005` 做 readiness 收口。

## 4. 强制边界

- rehearsal 只编排已有事实源和低风险 preview，不直接执行生产部署或真实 rollback。
- release admission 只能阻断、允许或要求人工复核，不能绕过 approval、authz、quality 或 review。
- repair handoff 默认需要 review，不自动修改生产环境或主分支。
- Console 不能自行计算生产准入、不能消费 approval、不能执行服务器命令。
