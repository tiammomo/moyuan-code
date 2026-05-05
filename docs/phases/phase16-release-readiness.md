# Phase 16 Release Readiness

状态：ready
责任角色：devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 16 已完成“部署演练、运行风险闭环与发布准入增强”的第一批能力。系统已经可以把 deployment execution、monitor summary、rollback preview 和 evidence 聚合为 deployment rehearsal；再基于 rehearsal 生成 release admission；最后把 blocked/manual 风险交接到 self-repair 的 deployment risk handoff。Console 已能展示和触发这条受控链路。

## 1. 完成范围

- `phase16-001 deployment-rehearsal-controller`：新增 rehearsal 事实对象，串联 candidate、deployment、execution、post-deployment history、monitor summary、rollback preview 和 evidence。
- `phase16-002 release-admission-risk-gate`：新增 release admission，输出 `allowed`、`manual_required` 或 `blocked`，并保留 signals、reasons 和 evidence。
- `phase16-003 monitor-risk-repair-bridge`：新增 deployment risk handoff，把 blocked/manual admission 或 monitor risk 转入 signal、bug candidate 和 approval-gated repair plan。
- `phase16-004 console-rehearsal-risk-surface`：Console 已展示 rehearsal、admission 和 risk handoff，并只调用后端受控 API。
- `phase16-005 phase16-readiness`：状态、门禁结论、保留边界和下一阶段入口完成收口。

## 2. 验证结论

最新收口门禁：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```

结论：通过。

最近提交：

```text
d99bf47 feat: surface rehearsal risk flow
11cf077 feat: add deployment risk repair handoff
aa0d7f0 feat: add release admission risk gate
53182fd feat: add deployment rehearsal controller
0338699 docs: open phase 16 deployment rehearsal
```

## 3. 保留边界

- rehearsal 只聚合事实源并触发低风险 rollback preview，不执行真实部署或真实 rollback。
- release admission 只输出准入结论，不修改 release candidate、Git、部署状态或服务器。
- deployment risk handoff 只生成 review-required repair plan，不自动执行 repair attempt。
- Console 不自行计算准入结论，不消费 approval，不执行生产命令。
- 生产真实写入仍默认关闭。

## 4. 进入 Phase 17 的理由

Phase 16 已经把部署演练、准入判断和风险修复入口串成闭环。下一阶段应聚焦“策略自动化与演练调度”：

- release admission 可以进入可配置 policy pack，而不是固定在代码分支判断里。
- deployment rehearsal 可以由 scheduler 或 release pipeline 自动创建，但仍保持真实执行默认关闭。
- risk handoff 需要进一步连接 review queue、repair attempt dry-run 和 Console drill-down。
- Console 可以把 rehearsal/admission/handoff 拆成更清晰的 timeline drill-down，而不是只在摘要区展示。

Phase 17 建议聚焦“发布准入策略包、演练调度与风险修复 drill-down”，继续保持生产写入默认关闭。
