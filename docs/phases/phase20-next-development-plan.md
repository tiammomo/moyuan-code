# Phase 20 实施记录

状态：in_progress
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

本文记录 Phase 20 的实际执行顺序。Phase 20 的入口以 [Phase 20 实现 Issue Graph](./phase20-issue-graph.md) 为准。

## 1. 阶段入口

Phase 19 已完成并通过 readiness：

- Operations audit export 已能导出 JSON/Markdown 审计报告。
- Decision ledger 已统一 policy/readiness/verification/review 结论。
- Durable control runner 已具备幂等、retry budget、manual required 和低风险 step。
- Provider write proof 已能解释 release provider、deployment execution 和 resource maintenance 的真实写入条件。
- Console 已展示 audit export、decision ledger、write proof 和 control runner 摘要。

Phase 20 不改变生产真实写入默认关闭的原则，重点补 write admission、provider-specific proof、remote rehearsal、control runner queue/window 和 Console 单条 drill-down。

## 2. Phase 20 第一批任务

| 优先级 | Issue | 名称 | 状态 | 目标 | 验收 |
| --- | --- | --- | --- | --- | --- |
| P0 | `phase20-001` | `write-proof-admission-policy` | completed | 写入准入报告 | proof -> admission 可解释且不执行写入 |
| P0 | `phase20-002` | `provider-specific-proof-pack` | completed | provider 最小权限要求 | GitHub/Gitee/SSH/cloud 要求可配置 |
| P1 | `phase20-003` | `remote-execution-rehearsal-runner` | completed | 远程执行演练 | rehearsal 不执行生产变更 |
| P1 | `phase20-004` | `control-runner-queue-window` | completed | 长期任务队列与窗口 | 维护窗口、retry、handoff 可审计 |
| P1 | `phase20-005` | `console-proof-admission-drilldown` | completed | Console 单条钻取 | 展示 proof/admission/runner step |
| P2 | `phase20-006` | `phase20-readiness` | next | Phase 20 收口 | 全量门禁和生产边界完成 |

## 3. 执行规划：`phase20-001 write-proof-admission-policy`

实现状态：completed。

范围：

- 新增 write admission report，读取 write proof 并按默认 policy 输出 `ready`、`blocked`、`manual_required` 或 `rehearsal_only`。
- 每条 admission 保留 proof id、operation、provider、environment、write flag、approval、secret/auth ref、evidence、rule refs 和 reasons。
- 增加 API/CLI 查询入口，支持按 provider、operation、environment、status、decision 过滤。

非目标：

- 不执行 Git/provider/SSH/cloud 写入。
- 不消费 approval。
- 不读取 secret 明文。
- 不修改原 write proof、release、deployment 或 resource 状态。

验收：

- 可由 release provider execution、deployment execution、resource maintenance 的 proof 生成 admission。
- write disabled、approval missing、secret missing、production blocked、evidence missing 都有明确 decision。
- API、CLI 和单测覆盖过滤、空结果和生产边界。

完成记录：

- `internal/operations` 新增 write admission report，读取 write proof 后输出 `ready`、`blocked`、`manual_required` 或 `rehearsal_only`。
- 每条 admission 保留 proof id、operation、provider、environment、write flag、approval、secret/auth ref、evidence、rule refs、least privilege 和 replay guard。
- API 增加 `GET /v1/projects/:project_id/operations/write-admissions`。
- CLI 增加 `moyuan operations write-admissions ...`。
- 单测覆盖 release provider、deployment dry-run、resource maintenance、过滤、API 和 CLI。

## 4. 执行规划：`phase20-002 provider-specific-proof-pack`

实现状态：completed。

范围：

- 新增 provider-specific proof requirements，覆盖 `github`、`gitee`、`ssh`、`cloud` 和 `local_registry` 的默认要求。
- 每个 provider requirement 需要声明 allowed operation types、required secret ref status、required evidence、required approval、least privilege scopes 和 replay guard。
- write admission 后续可引用 provider requirement 的 rule refs，但本任务不执行真实写入。

验收：

- 可按 provider 和 operation type 查询 provider proof requirements。
- GitHub/Gitee/SSH/cloud/local_registry 默认规则可测试。
- API、CLI 和单测覆盖未知 provider、provider 不支持 operation、缺 required evidence/secret/approval 的解释。

完成记录：

- `internal/operations` 新增 provider proof requirements report，默认覆盖 `generic_git`、`github`、`gitee`、`local`、`ssh`、`cloud`、`aliyun`、`tencent_cloud` 和 `local_registry`。
- 每条 requirement 显式声明 secret ref、evidence、approval、write switch、production review、least privilege scopes 和 replay guard。
- Write admission 现在会引用 provider requirement id、policy version 和 rule refs，并在缺少 provider requirement 时阻断真实写入准入。
- API 增加 `GET /v1/projects/:project_id/operations/provider-proof-requirements`。
- CLI 增加 `moyuan operations provider-proof-requirements ...`。
- 单测覆盖 provider/operation 过滤、未知 provider 空结果、API、CLI 和 admission provider requirement 引用。

## 5. 执行规划：`phase20-003 remote-execution-rehearsal-runner`

实现状态：completed。

范围：

- 新增 remote execution rehearsal report，读取 write admission 和 provider requirement，验证目标、命令 allowlist、auth ref、证据引用和回滚准备。
- rehearsal 只输出 preview、blocked 或 manual handoff，不执行 SSH、cloud、Git provider 或生产命令。
- 产物需要能被 operations timeline、decision ledger 或 control runner 后续引用。

验收：

- 可按 environment、provider、status、decision 查询 remote execution rehearsal。
- blocked/manual/ready rehearsal 都有 reasons、rule refs、evidence refs 和 source admission。
- API、CLI 和单测覆盖 dry-run、缺 provider requirement、缺 auth ref 和生产边界。

完成记录：

- `internal/operations` 新增 remote execution rehearsal runner，读取 deployment write admission 并验证 remote target、auth ref、command allowlist、rollback readiness。
- 每次 run 都持久化 `.moyuan/lifecycle/deployments/remote-execution-rehearsals/*.json`，追加 JSONL，写入 evidence，并进入 operations timeline。
- API 增加 `POST/GET /v1/projects/:project_id/operations/remote-execution-rehearsals`。
- CLI 增加 `moyuan operations remote-execution-rehearsals run|list ...`。
- 单测覆盖 ssh preview 演练通过、missing target/remote plan 阻断、API、CLI 和 timeline 聚合。

## 6. 执行规划：`phase20-004 control-runner-queue-window`

实现状态：completed。

范围：

- Durable control runner 增加轻量任务队列和维护窗口校验。
- 队列任务需要记录 trigger、step、environment、requested_by、window、attempt、status、decision 和 handoff reason。
- Runner 执行前必须判断维护窗口、retry budget、idempotency key 和 write/rehearsal admission 相关事实。

验收：

- 可创建/list control runner queue item，并能执行 due item。
- 不在维护窗口内的任务必须进入 waiting 或 manual handoff，不能直接执行。
- API、CLI 和单测覆盖队列、窗口、retry/handoff 和幂等 replay。

完成记录：

- `internal/controlloop` 新增 queue item、queue list 和 queue runner，持久化 `.moyuan/control-loop/queue/*.json` 与 `queue.jsonl`。
- 维护窗口支持 `always`、`due:YYYY-MM-DD`、`after:RFC3339` 和 `between:HH:MM-HH:MM`；未到窗口保持 `waiting`，非法窗口进入 `manual_required`。
- Queue runner 会把到期任务转换为 durable control loop run，并复用 idempotency key、retry budget、environment、resource 和 deployment execution 参数。
- API 增加 `POST/GET /v1/projects/:project_id/control-loop/queue` 和 `POST /v1/projects/:project_id/control-loop/queue/run`。
- CLI 增加 `moyuan control-loop queue add|list|run ...`。
- 单测覆盖 due item 执行、future window 等待、非法窗口 handoff、API 和 CLI。

## 7. 执行规划：`phase20-005 console-proof-admission-drilldown`

实现状态：completed。

范围：

- Console operations 面板新增 proof/admission/provider requirement/remote rehearsal/control queue 的 drill-down 数据读取。
- 前端只展示 API 返回的事实源，不重新计算 admission、proof 或 queue decision。
- 提供只读导出入口，便于用户把单条 proof/admission/rehearsal/queue 复制到 issue 或 release review。

验收：

- Console 能展示 write proof、write admission、provider requirement、remote rehearsal 和 control queue 摘要/详情。
- drill-down 展示 reasons、rule refs、evidence refs、source refs、provider requirement refs 和 queue/run 关联。
- `npm run typecheck` 和 `npm run build` 通过。

完成记录：

- Console 数据层新增 `write_admissions`、`provider_proof_requirements`、`remote_execution_rehearsals` 和 `control_loop_queue` 拉取与规范化。
- Operations 视图新增 Write Admission、Provider Proof Pack、Remote Rehearsal 和 Control Queue 面板。
- 面板只展示后端返回的 status、decision、reasons、rule refs、evidence refs、provider requirement refs、queue/run 关联和 check count，不在前端重新计算准入结论。
- TypeScript 类型覆盖新增 report/entry/requirement/rehearsal/queue item。

## 8. 执行规划：`phase20-006 phase20-readiness`

实现状态：next。

范围：

- 运行全量门禁并记录 Phase 20 完成范围。
- 回写 README、docs 入口、issue graph、实施记录和 readiness 文档。
- 明确 Phase 20 后仍然不默认开启生产真实写入，Phase 21 只能基于 readiness 入口继续。

验收：

- Go、Console typecheck/build 和 diff check 全部通过。
- Phase 20 readiness 文档说明完成项、提交记录、保留边界和下一阶段入口。
- 完成后停止，不进入 Phase 21。

## 9. 验证要求

每完成一个 Phase 20 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
