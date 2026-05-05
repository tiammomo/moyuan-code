# Phase 21 实施记录

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

本文记录 Phase 21 的实际执行顺序。Phase 21 的入口以 [Phase 21 实现 Issue Graph](./phase21-issue-graph.md) 为准。

## 1. 阶段入口

Phase 20 已完成：

- Write admission 可以把 write proof 转成真实写入前准入结论。
- Provider proof requirements 已声明 GitHub/Gitee/SSH/cloud/local registry 的最小权限、secret ref、evidence 和 replay guard。
- Remote execution rehearsal 可以验证目标、auth ref、command allowlist 和 rollback readiness，且不执行真实写入。
- Control queue 已具备维护窗口、retry budget、handoff 和 durable runner。
- Console 已展示 proof、admission、provider requirement、remote rehearsal 和 queue。

Phase 21 聚焦“写入前审查包”和“队列执行前强绑定”，不改变真实写入默认关闭原则。

## 2. Phase 21 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase21-001` | `write-review-packet` | completed | 生成写入前 review packet | Packet 包含 admission、requirement、rehearsal、queue、evidence |
| P0 | `phase21-002` | `control-queue-review-gate` | completed | Queue 执行前校验 packet/admission/rehearsal | 未 ready 不执行 runner |
| P1 | `phase21-003` | `console-review-packet` | completed | Console 展示 review packet | 前端只读展示事实源 |
| P1 | `phase21-004` | `phase21-readiness` | completed | Phase 21 收口 | 全量门禁和 Phase 22 入口完成 |

## 3. 执行规划：`phase21-001 write-review-packet`

实现状态：completed。

范围：

- 新增 write review packet report，读取 write admission、provider proof requirement、remote execution rehearsal 和 control queue snapshot。
- 每个 packet 输出 `ready`、`blocked` 或 `manual_required`，并保留 source admission、source proof、operation、provider、environment、evidence refs、rule refs 和 markdown 摘要。
- Packet 持久化到 `.moyuan/lifecycle/deployments/write-review-packets/`，追加 JSONL，并写入 evidence。

非目标：

- 不执行 Git/provider/SSH/cloud 写入。
- 不消费 approval。
- 不读取 secret 明文。

完成记录：

- `internal/operations` 新增 write review packet create/list/load，聚合 write admission、provider requirement refs、remote rehearsal、queue snapshot、rule refs 和 evidence refs。
- Packet 持久化 `.moyuan/lifecycle/deployments/write-review-packets/*.json`，追加 `write-review-packets.jsonl`，写入 evidence，并进入 operations timeline。
- API 增加 `POST/GET /v1/projects/:project_id/operations/write-review-packets`。
- CLI 增加 `moyuan operations write-review-packets create|list ...`。
- 单测覆盖 ready resource maintenance packet、持久化、列表、timeline、API 和 CLI。

## 4. 执行规划：`phase21-002 control-queue-review-gate`

实现状态：completed。

范围：

- Queue item 增加 `admission_id`、`remote_rehearsal_id`、`review_packet_id`。
- Queue runner 在维护窗口通过后、执行 control runner 前校验绑定事实。
- Review packet 未 ready、rehearsal 未 completed、admission blocked/manual 时，queue item 进入 `manual_required`。

完成记录：

- Queue item 新增 `admission_id`、`remote_rehearsal_id`、`review_packet_id`。
- Queue runner 在 maintenance window 通过后、执行 durable control runner 前校验绑定事实。
- 缺失或未 ready 的 review packet、admission、remote rehearsal 会让 queue item 进入 `manual_required`，decision 为 `CONTROL_QUEUE_REVIEW_GATE_MANUAL_REQUIRED`。
- API/CLI queue add 支持绑定 ID，单测覆盖缺失 review packet 的 manual handoff。

## 5. 执行规划：`phase21-003 console-review-packet`

实现状态：completed。

范围：

- Console Operations 面板读取 `write_review_packets`。
- 展示 packet status、decision、source refs、evidence refs、rule refs、queue binding 和 markdown 摘要。
- 前端只展示后端结论，不重新计算 gate。

完成记录：

- Console 数据层新增 `write_review_packets` 拉取、类型和 normalize。
- Operations 视图新增 Write Review Packet 面板，展示 packet status、decision、operation、provider、remote rehearsal、provider requirement、queue count、evidence 和 rule count。
- Control Queue 面板展示绑定的 review packet、admission 和 remote rehearsal ID。

## 6. 执行规划：`phase21-004 phase21-readiness`

实现状态：completed。

范围：

- 运行全量门禁。
- 回写 README、docs 入口、Phase 21 issue graph、实施记录和 readiness。
- 明确 Phase 22 只能基于 ready review packet 进入 guarded write execution plan。

完成记录：

- 新增 [Phase 21 Release Readiness](./phase21-release-readiness.md)。
- Phase 21 issue graph、实施记录和 docs phase 入口已更新为 ready。
- Phase 22 入口保持 planned，唯一前置输入为 ready review packet。

## 7. 验证要求

每完成一个 Phase 21 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
