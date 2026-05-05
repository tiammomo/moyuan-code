# Phase 13 Release Readiness

状态：ready
责任角色：release_owner + git_owner + devops_owner + backend_owner + frontend_owner + security_owner + qa_owner
最后更新：2026-05-05

Phase 13 已完成“Release Candidate 远程发布与部署交接”的第一批能力。系统已经可以把 Phase 12 的 release batch readiness 转成可审计的 release candidate，按审批和执行开关受控准备本地 release branch，生成 GitHub/Gitee provider preview 和 PR/MR handoff，并交接到 deployment plan；Console 已能展示和触发这条链路的受控动作。

## 1. 完成范围

- `phase13-001 release-candidate-plan-from-batch`：suggested release batch 可生成 release candidate plan，记录版本、source branch、release branch、provider、remote 和部署目标。
- `phase13-002 guarded-local-release-branch-apply`：release branch apply 默认 dry-run；真实本地 branch 更新需要审批和 `MOYUAN_ALLOW_RELEASE_BRANCH_APPLY=1`。
- `phase13-003 release-candidate-provider-preview`：已完成本地 release branch apply 的 candidate 可生成 Git Provider preview、PR/MR plan、tag/release/workflow guarded action 摘要。
- `phase13-004 deployment-handoff-from-release-candidate`：release candidate 可生成 deployment plan，衔接环境、服务器资源、smoke 和 monitor 模板。
- `phase13-005 console-release-candidate-surface`：Console 已展示 release candidate、branch apply、provider preview 和 deployment handoff，并只调用后端受控 API。

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
4b8bbe2 feat: surface release candidate flow
7b010af feat: add release candidate deployment handoff
6792ce6 feat: add release candidate provider preview
e4cda30 feat: add guarded release branch apply
46dd66c feat: add release candidate planning
2d2dd5e docs: open phase 13 release candidates
```

## 3. 保留边界

- release candidate plan 不执行 Git、Provider 或部署写入。
- release branch apply 的真实写入只更新本地 release branch，不 push、不创建 PR/MR、不 tag、不 publish。
- provider preview 只生成远程动作计划和 PR/MR handoff，不执行 GitHub/Gitee 写入。
- deployment handoff 只生成 deployment plan，不执行部署、不运行线上冒烟、不更新生产监控。
- Console 不自行计算 release、provider 或 deploy readiness，只展示后端事实源和后端返回状态。
- GitHub/Gitee 远程写入、tag/release 发布、deployment execution、线上 smoke 和生产 monitor 仍需要 approval/authz、secret resolver、执行开关和审计。

## 4. 进入 Phase 14 的理由

Phase 13 已经把“本地 release batch 准备”推进到 release candidate、release branch、Git Provider preview 和 deployment plan 的完整可见链路。下一阶段应把这条链路推进到更接近真实发布，但仍保持强门禁：

- GitHub/Gitee PR/MR 创建需要从 preview/handoff 推进到 approval-gated execution。
- release tag、release note 和 provider publish 需要有独立审批、replay guard、审计和回滚记录。
- deployment plan 需要推进到受控 dry-run/execution、线上 smoke 和生产 monitor 状态回写。
- Console 需要展示发布执行流水线、审批阻断、远程结果、部署结果和回退入口。

Phase 14 应聚焦“受控远程发布与部署执行”，继续保持生产写入默认关闭。
