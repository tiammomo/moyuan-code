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
| P0 | `phase20-002` | `provider-specific-proof-pack` | next | provider 最小权限要求 | GitHub/Gitee/SSH/cloud 要求可配置 |
| P1 | `phase20-003` | `remote-execution-rehearsal-runner` | planned | 远程执行演练 | rehearsal 不执行生产变更 |
| P1 | `phase20-004` | `control-runner-queue-window` | planned | 长期任务队列与窗口 | 维护窗口、retry、handoff 可审计 |
| P1 | `phase20-005` | `console-proof-admission-drilldown` | planned | Console 单条钻取 | 展示 proof/admission/runner step |
| P2 | `phase20-006` | `phase20-readiness` | planned | Phase 20 收口 | 全量门禁和生产边界完成 |

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

实现状态：next。

范围：

- 新增 provider-specific proof requirements，覆盖 `github`、`gitee`、`ssh`、`cloud` 和 `local_registry` 的默认要求。
- 每个 provider requirement 需要声明 allowed operation types、required secret ref status、required evidence、required approval、least privilege scopes 和 replay guard。
- write admission 后续可引用 provider requirement 的 rule refs，但本任务不执行真实写入。

验收：

- 可按 provider 和 operation type 查询 provider proof requirements。
- GitHub/Gitee/SSH/cloud/local_registry 默认规则可测试。
- API、CLI 和单测覆盖未知 provider、provider 不支持 operation、缺 required evidence/secret/approval 的解释。

## 5. 验证要求

每完成一个 Phase 20 issue，至少运行：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```
