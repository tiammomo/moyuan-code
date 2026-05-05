# Phase 15 Release Readiness

状态：ready
责任角色：devops_owner + release_owner + backend_owner + frontend_owner + security_owner + qa_owner
最后更新：2026-05-05

Phase 15 已完成“部署审批加固、回退执行与生产可观测性增强”的第一批能力。系统已经把 deployment execution 的真实执行凭证从裸 `approved` 收紧为 `approval_id` proof 和 approval consumption；把 rollback suggestion 推进为受控 rollback execution；把 post-deployment history 聚合为 monitor summary；Console 已能展示 approval、rollback 和 monitor 事实源并触发低风险受控动作。

## 1. 完成范围

- `phase15-001 deployment-execution-approval-proof`：真实部署执行必须携带匹配 scope 的 `approval_id`，并在真实命令执行前消费 approval；已消费 approval 不能重放。
- `phase15-002 rollback-execution-controller`：rollback execution 支持 preview 和 gated `local_shell`，默认 preview-only，真实回退需要 approval、写开关、安全命令和 evidence。
- `phase15-003 production-monitor-loop`：monitor summary 可按最近窗口聚合 post-deployment history，输出 healthy、attention_required、critical 或 unknown 结论。
- `phase15-004 console-deployment-ops-surface`：Console snapshot 已接入 rollback executions 和 monitor summaries；Deployment 面板可触发 rollback preview 和 monitor summary。
- `phase15-005 phase15-readiness`：状态、门禁结论、保留边界和下一阶段入口完成收口。

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
511a3e2 feat: surface deployment ops signals
d23483e feat: add deployment monitor summaries
f7edeb7 feat: add rollback execution controller
c6bff2c feat: require deployment execution approval proof
5ff4bcd docs: open phase 15 deployment hardening
```

## 3. 保留边界

- production real deployment execution 继续默认阻断，除非后续策略明确开放并配置独立执行开关。
- rollback execution 不会自动真实执行；Console 只触发 preview，不消费 approval，不执行命令。
- `local_shell` rollback 只允许安全命令 allowlist，并要求 `MOYUAN_ALLOW_ROLLBACK_EXECUTE=1`。
- monitor summary 只作为事实输入，不能绕过 release、deployment、quality、review 或 approval 门禁。
- Console 不自行计算生产健康结论，不直接执行 Git、Provider、部署或回退命令。

## 4. 进入 Phase 16 的理由

Phase 15 已经把部署执行、回退执行和 monitor 摘要补齐到可审计、可审批、可回放的控制面。下一阶段应把这些离散能力组织成“部署演练和生产运行维护闭环”：

- deployment rehearsal 需要把 plan、execution、rollback preview、monitor summary 和 evidence 串成一次可复现演练记录。
- monitor summary 需要进入 release readiness、repair candidate 和后续告警/维护建议，而不是只停留在列表展示。
- 服务器资源、部署目标和 release candidate 之间需要更明确的环境级健康门禁。
- Console 需要从操作入口继续推进到 deployment rehearsal timeline 和 risk decision drill-down。

Phase 16 建议聚焦“部署演练、运行风险闭环与发布准入增强”，继续保持真实生产写入默认关闭。
