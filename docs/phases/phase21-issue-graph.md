# Phase 21 实现 Issue Graph

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 21 的目标是把 Phase 20 的 write admission、provider proof requirement、remote execution rehearsal 和 control queue 收敛为“真实写入前 review packet”。本阶段仍不执行 Git/provider/SSH/cloud 真实写入，只生成可审查、可导出、可被队列绑定的准入材料。

## 1. Phase 21 目标

- 将 proof、admission、provider requirement、remote rehearsal、queue item 聚合为 write review packet。
- Review packet 需要给出 `ready`、`blocked` 或 `manual_required` 结论，并保留 rule refs、evidence refs、source refs。
- Control queue 执行前能绑定 review packet、admission 和 rehearsal 结果，未通过时进入 manual handoff。
- Console 能展示 review packet 与 queue gate 事实源，不在前端重新计算策略。
- Phase 21 完成后为 Phase 22 的 guarded write execution plan 提供唯一前置输入。

## 2. Issue Graph

| Issue | 名称 | 状态 | 范围 | 前置 | 责任角色 | 验收重点 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase21-001` | `write-review-packet` | completed | 新增 review packet 聚合器，汇总 admission、provider proof、remote rehearsal、queue snapshot 和 evidence | Phase 20 readiness | `backend_owner` + `security_owner` | Packet 可持久化、可导出、可审计，不执行写入 |
| `phase21-002` | `control-queue-review-gate` | completed | Queue item 增加 admission/rehearsal/review packet 绑定，执行前校验 gate | `phase21-001` | `orchestrator_owner` + `devops_owner` | 未 ready 的 packet 不能进入 runner，只能 waiting/manual |
| `phase21-003` | `console-review-packet` | completed | Console Operations 面板展示 review packet 与 queue gate 关联 | `phase21-001`,`phase21-002` | `frontend_owner` | 只读展示后端事实源 |
| `phase21-004` | `phase21-readiness` | completed | 收口验证、文档回写、提交记录和 Phase 22 入口 | `phase21-003` | `release_owner` + `qa_owner` | 全量门禁通过，Phase 22 输入明确 |

## 3. 建议执行顺序

1. 先做 `phase21-001`，让系统有统一的写入前 review packet。
2. 再做 `phase21-002`，把 queue runner 的执行前门禁绑定到 packet/admission/rehearsal。
3. `phase21-003` 将 review packet 暴露到 Console。
4. `phase21-004` 做 readiness 收口。

## 4. 强制边界

- Review packet 只能聚合事实、生成审查材料和输出 gate decision，不能执行外部写入。
- Queue gate 不能绕过 authz、approval、secret、provider proof、write admission、remote rehearsal、maintenance window 或 retry budget。
- Console 不负责重新计算 packet status、queue decision 或 evidence 结论。
- Phase 22 之前不存在真实 provider apply、SSH apply 或 cloud apply。
