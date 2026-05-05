# Phase 20 实现 Issue Graph

状态：in_progress
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 20 的目标是把 Phase 19 的 write proof、decision ledger、durable control runner 和 Console drill-down 推进到“受控生产写入演练与远程运维执行增强”。本阶段仍不默认开启生产真实写入，所有真实写入路径必须先通过 write admission、approval、secret、provider、quality、evidence 和 replay guard。

## 1. Phase 20 目标

- Write proof 能被转换为可执行前置准入结论，明确 real write、rehearsal 和 manual review 边界。
- Provider-specific proof pack 能描述 GitHub/Gitee/SSH/cloud 操作需要的最小权限、secret 引用和证据要求。
- Remote execution rehearsal 能在不执行生产写入的前提下验证目标、命令、凭证引用和回滚准备。
- Control runner 能从一次性 bounded run 逐步升级到队列、维护窗口和人工 handoff。
- Console 能钻取单条 proof/admission/runner step，并提供只读导出入口。

## 2. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase20-001` | `write-proof-admission-policy` | completed | 将 write proof 转为 write admission report，输出 ready/blocked/manual/rehearsal 结论 | Phase 19 readiness | `security_owner` + `backend_owner` | 不执行写入，准入规则可解释、可测试、可审计 |
| `phase20-002` | `provider-specific-proof-pack` | completed | 为 GitHub/Gitee/SSH/cloud 维护 provider-specific proof requirements | `phase20-001` | `release_owner` + `devops_owner` | 最小权限、secret ref、evidence 和 replay guard 按 provider 可配置 |
| `phase20-003` | `remote-execution-rehearsal-runner` | completed | 新增 remote execution rehearsal，验证目标、命令 allowlist、auth ref 和回滚准备 | `phase20-002` | `devops_owner` + `backend_owner` | rehearsal 不执行生产写入，失败有 evidence 和 handoff |
| `phase20-004` | `control-runner-queue-window` | completed | durable control runner 增加任务队列、维护窗口、retry/handoff 和幂等调度 | `phase20-003` | `orchestrator_owner` + `devops_owner` | 长期任务不无限重试，不绕过维护窗口和审批 |
| `phase20-005` | `console-proof-admission-drilldown` | next | Console 展示单条 proof、admission、runner step 和导出入口 | `phase20-001`,`phase20-004` | `frontend_owner` | 前端只展示事实源，不重新决策 |
| `phase20-006` | `phase20-readiness` | planned | 收口验证、文档回写、剩余风险和 Phase 21 入口 | `phase20-005` | `release_owner` + `security_owner` | 全量门禁通过，生产写入边界清晰 |

## 3. 建议执行顺序

1. 先做 `phase20-001`，把 Phase 19 的 proof contract 变成后续真实执行前的 admission gate。
2. 再做 `phase20-002`，把 provider-specific 最小权限和证据要求显式化。
3. `phase20-003` 基于 provider proof 做 remote execution rehearsal。
4. `phase20-004` 增强长期控制任务的队列、窗口和 handoff。
5. `phase20-005` 做 Console drill-down。
6. `phase20-006` 做 readiness 收口。

## 4. 强制边界

- Write admission 只能读取 proof 和事实源，不执行 Git、provider、SSH 或 cloud 写入。
- Provider-specific proof pack 只能声明要求，不保存 secret 明文。
- Remote rehearsal 只做 preview、dry-run、连接/命令/回滚准备校验，不执行生产变更。
- Control runner 不能绕过 authz、approval、secret、provider、quality、maintenance window 或 protected path。
- Console 不能自行计算 admission 或 proof 结论。
