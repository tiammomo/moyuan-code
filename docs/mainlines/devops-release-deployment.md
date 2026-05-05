# DevOps 发布投产主线

## 1. 目标

DevOps 发布投产主线负责从已通过质量门禁的 integration branch 创建 release branch，完成版本说明、tag、远程推送、PR/MR、部署投产、线上冒烟、监控窗口、回滚和复盘。

这条主线的目标不是替代 CI/CD 平台，而是统一控制发布决策、服务器资源引用、审批、冒烟、监控和回滚状态。

Release note、发版批次、覆盖率门禁、禁止发版条件和回退后 fix 规范见 [工程流程规范](../engineering-process-standards.md)。

当前 Beta 实现已落地 release suggestion 最小闭环：

- CLI：`moyuan release suggest [--version <version>] [--min-issues <n>]`、`moyuan release show <release-id>`、`moyuan release provider preview <release-id>`、`moyuan release provider publish <release-id> [--approved] [--approval-id <approval-id>]`、`moyuan evidence list/show`。
- CLI：`moyuan deploy plan <release-id> --environment <env> [--resource <host-id>]`、`moyuan deploy execute <deployment-id> [--mode dry_run|ssh_preview|ssh_execute|local_shell]`、`moyuan deploy show <deployment-id>`。
- API：`POST /v1/projects/:project_id/releases/suggest`、`GET /v1/projects/:project_id/releases/:release_id`、`POST /v1/projects/:project_id/releases/:release_id/provider-preview`、`POST /v1/projects/:project_id/releases/:release_id/provider-publish`、`GET /v1/projects/:project_id/release-provider-executions`、`GET /v1/projects/:project_id/release-provider-executions/:execution_id`、`POST /v1/projects/:project_id/deployments/plan`、`GET /v1/projects/:project_id/deployments/:deployment_id`、`POST /v1/projects/:project_id/deployments/:deployment_id/execute`、`GET /v1/projects/:project_id/deployment-monitor-history`、`GET /v1/projects/:project_id/deployment-executions/:execution_id/post-deployment-history`、`GET /v1/projects/:project_id/evidence`、`GET /v1/projects/:project_id/operations/:operation_type/:operation_id`。
- Console：Deployment Executions 面板可触发 `Suggest Release`、最新 deployment `Dry Run` 和 `test_dev` `Health Scan`；Release Pipeline 面板可触发 release provider `Preview`/`Publish`，所有动作都走后端受控 API。
- 输出位置：`.moyuan/lifecycle/releases/` 和 `.moyuan/lifecycle/deployments/`。
- 当前生成 release suggestion、release branch plan、tag suggestion、release notes draft、provider release/tag/workflow action preview、deploy/smoke/monitor/rollback plan；smoke/monitor plan 会引用检查模板并携带 severity、window 和 failure classes。受控 `local_shell` 执行后自动记录 smoke/monitor 结果；`ssh_preview` 可生成远程目标执行预览，`ssh_execute` 默认不开启真实 SSH 连接或生产部署。
- Release provider publish 已具备受控真实写入 Beta：默认返回 preview-only 且不消费 approval；设置 `MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE=1` 后，仍必须通过 approval、secret resolver 和 release plan ready 检查。secret 缺失时阻断且不消费 approval；secret 通过后消费 approval 并调用 GitHub/Gitee create release API。branch push、tag push 和 workflow dispatch 暂时显式 `skipped`，但 preview/result 会结构化展示 `risk_level`、`execution_mode` 和 guardrails，同一 approval 不能重复使用。
- SSH execute 已具备受控真实执行 Beta：默认返回 blocked；设置 `MOYUAN_ALLOW_SSH_EXECUTE=1` 后校验 server resource、`auth_ref`、命令 allowlist 和超时，再通过本机 `ssh` 二进制执行。stdout/stderr 写入 execution 前脱敏，成功后自动进入 smoke/monitor，失败后生成 rollback suggestion；production real execution 仍继续阻断。
- Release provider execution 和 deployment execution 会自动写入 `.moyuan/lifecycle/evidence/`；deployment 会额外拆出 smoke、monitor 和 rollback/not-required evidence，作为发布、部署、烟测、监控和回滚证据链的统一索引。
- Operation detail 聚合 API 会按 operation type/id 返回 execution 摘要、evidence chain 和 artifact references；该接口不返回 secret、SSH key、完整 stdout/stderr 或 provider response body。
- 失败部署会生成结构化 rollback runbook artifact，位置为 `.moyuan/lifecycle/deployments/rollback-runbooks/<execution-id>.json`；runbook 默认要求人工审查，不自动回滚生产。
- 每次 deployment execution 完成后会生成 post-deployment history，位置为 `.moyuan/lifecycle/deployments/post-deployment-history/<execution-id>.json`；其中固定包含 smoke/monitor 检查摘要、检查模板、severity、失败分类、rollback runbook 状态、evidence ids 和 artifact references。
- 已接入门禁：dirty worktree、remote 缺失、无 accepted issue、存在 unresolved issue 时阻断。
- production deployment plan 缺少 approval 时阻断；test_dev 可生成演练计划。

## 2. 输入与输出

输入：

- accepted integration branch。
- included issues 和 excluded issues。
- quality reports。
- review reports。
- coverage reports。
- migration checklist。
- release policy。
- target environment。
- server resource group。

输出：

- release suggestion。
- release branch。
- tag。
- release notes。
- PR/MR。
- deployment record。
- smoke test result。
- monitor window result。
- rollback record。
- release retrospective。
- release memory candidates。

## 3. 端到端流程

```text
integration branch accepted
  -> calculate release batch score
  -> create release suggestion
  -> user approval if required
  -> create release branch
  -> full quality gates
  -> regression tests
  -> migration and config checks
  -> generate release notes
  -> create tag if configured
  -> push release branch to GitHub/Gitee
  -> create PR/MR if configured
  -> resolve target environment
  -> resolve resource group
  -> pre-deploy backup
  -> deploy
  -> online smoke tests
  -> monitor window
  -> healthy, rollback, or manual intervention
  -> retrospective and memory candidates
```

## 4. 发布批次建议

默认建议：

- 低风险功能：累计 3-7 个 accepted issues 后建议发版。
- 中风险功能：累计 2-4 个 accepted issues 后建议发版。
- 高风险变更：单独发版窗口。
- hotfix/security：立即进入 hotfix/release 流水线。

高风险包括：

- 数据库迁移。
- 鉴权、安全、支付。
- 公共 API breaking change。
- 多服务联动。
- 无法自动回滚。
- 生产监控缺失。

## 5. 决策点

调用策略：

- [发布投产策略](../policies/release-deployment-policy.md)
- [服务器资源策略](../policies/server-resource-policy.md)
- [Git 分支策略](../policies/git-branch-policy.md)
- [质量与合入策略](../policies/quality-merge-policy.md)
- [Memory 决策策略](../policies/memory-decision-policy.md)

核心决策：

- 是否达到发版批次。
- 是否必须单独发版。
- 是否允许创建 release branch。
- 是否允许 tag 和 push。
- 是否允许部署到目标环境。
- 目标服务器资源是否健康。
- 线上冒烟失败是否回滚。
- 监控窗口异常是否回滚或人工介入。

## 6. 配置入口

- `.moyuan/policies/release.yaml`
- `.moyuan/policies/server-resources.yaml`
- `.moyuan/policies/environments.yaml`
- `.moyuan/policies/secrets.yaml`
- `.moyuan/policies/permissions.yaml`
- `.moyuan/policies/logging.yaml`

## 7. Workspace 产物

```text
.moyuan/lifecycle/releases/
.moyuan/lifecycle/deployments/
.moyuan/lifecycle/deployments/post-deployment-history/
.moyuan/lifecycle/deployments/rollback-runbooks/
.moyuan/lifecycle/rollback/
.moyuan/lifecycle/retrospectives/
```

## 8. 日志与审计

必须记录：

- release suggested。
- release branch created。
- tag created。
- push attempted/completed。
- PR/MR created。
- approval requested/granted/rejected。
- deployment started/completed/failed。
- smoke test result。
- monitor window result。
- rollback suggested/started/completed/failed。
- release retrospective。

日志流：

- `release`
- `git`
- `quality`
- `audit`
- `memory`
- `error`

## 9. 验收标准

- release branch 只能从 accepted integration branch 创建。
- 发版前必须满足 release note、覆盖率、回归测试和 rollback plan 要求。
- 高风险发布必须要求用户确认。
- 生产投产必须引用生产资源组。
- 生产投产前必须有备份、健康检查、冒烟和回滚策略。
- 冒烟失败或监控异常会进入回滚或人工介入。
- 发布完成后生成 release memory candidates。
