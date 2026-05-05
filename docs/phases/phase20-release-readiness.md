# Phase 20 Release Readiness

状态：ready
责任角色：orchestrator_owner + devops_owner + release_owner + backend_owner + frontend_owner + qa_owner + security_owner
最后更新：2026-05-05

Phase 20 已完成“受控生产写入演练与远程运维执行增强”的能力收口。系统现在可以把 write proof 转换为 write admission，按 provider 解释最小权限和 evidence 要求，运行不执行真实写入的 remote execution rehearsal，并用 control runner queue/window 管理长期任务。Console 已能展示 proof、admission、provider requirement、remote rehearsal 和 queue 的事实详情。

## 1. 完成范围

- `phase20-001 write-proof-admission-policy`：新增 write admission report，读取 write proof 后输出 `ready`、`blocked`、`manual_required` 或 `rehearsal_only`。
- `phase20-002 provider-specific-proof-pack`：新增 provider proof requirements，覆盖 `generic_git`、`github`、`gitee`、`local`、`ssh`、`cloud`、`aliyun`、`tencent_cloud` 和 `local_registry`。
- `phase20-003 remote-execution-rehearsal-runner`：新增 remote execution rehearsal runner，验证 target、auth ref、command allowlist、rollback readiness，并写入 evidence 和 operations timeline。
- `phase20-004 control-runner-queue-window`：新增 control loop queue、queue runner 和 maintenance window 判断，支持 `always`、`due:YYYY-MM-DD`、`after:RFC3339`、`between:HH:MM-HH:MM`。
- `phase20-005 console-proof-admission-drilldown`：Console Operations 视图展示 Write Admission、Provider Proof Pack、Remote Rehearsal 和 Control Queue 的后端事实源。
- `phase20-006 phase20-readiness`：状态、门禁结论、保留边界和下一阶段入口完成收口。

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
25cc995 feat: add console proof admission drilldowns
59e2f32 feat: add control loop queue
fb251a2 feat: add remote execution rehearsals
eabc52e feat: add provider proof requirements
ca7656b feat: add write proof admission policy
8d87675 docs: open phase 20 production rehearsal
```

## 3. 保留边界

- Write admission、provider proof requirement 和 remote rehearsal 都是准入解释、演练和事实记录层，不执行 Git/provider/SSH/cloud 真实写入。
- Remote execution rehearsal 可以验证 target、auth ref、command allowlist、rollback readiness，但不能执行生产命令、读取 secret 明文或消费 approval。
- Control loop queue 只能调度已接入的 control runner step；不能绕过 authz、approval、secret、provider、quality、maintenance window、write admission 或 protected path 门禁。
- Console 只展示后端事实源，不在前端重新计算 proof、admission、provider requirement、remote rehearsal 或 queue decision。
- 生产真实写入仍默认关闭，必须由显式写开关、授权、审批、secret 引用、provider gate、质量门禁、maintenance window、evidence 和 replay guard 同时满足后才可进入后续真实执行阶段。

## 4. 下一阶段入口建议

Phase 21 尚未启动。后续如果进入 Phase 21，建议只从以下入口继续：

- 把 remote execution rehearsal 的成功结果转化为更严格的真实写入前置审批包。
- 为 GitHub/Gitee release publish、SSH/cloud deployment 和 server resource maintenance 增加 provider adapter 的真实写入 preview/apply 分层。
- 把 control queue 的 maintenance window 与 approval、write admission、remote rehearsal 结果做更强绑定。
- 增加只读导出和 review packet，支持把 proof/admission/rehearsal/queue 作为 release review 附件。

当前收口按用户要求停止，不进入 Phase 21。
