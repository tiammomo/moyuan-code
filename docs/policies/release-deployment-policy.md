# 发布投产策略

## 1. 目标

决定什么时候建议发版、什么时候创建 release branch、是否允许发布到 GitHub/Gitee、是否允许部署到目标环境，以及线上冒烟或监控异常时是否回滚。

Release note、发版前置条件、批次规则、禁止发版条件和覆盖率门禁由 [工程流程规范](../engineering-process-standards.md) 维护。

当前 Beta 实现只做 `RELEASE_SUGGESTED` / `RELEASE_NOT_READY` / `RELEASE_BLOCKED` 的计划层判断，不执行真实 branch、tag、push 或部署。

## 2. 输入事实

- accepted integration branch。
- included issue count。
- changed modules。
- risk level。
- migration count。
- public API changes。
- security changes。
- quality reports。
- coverage reports。
- target environment。
- resource group health。
- rollback availability。
- user approval。

## 3. 决策结果

- `RELEASE_NOT_READY`
- `RELEASE_SUGGESTED`
- `RELEASE_REQUIRED_SINGLE`
- `RELEASE_BRANCH_ALLOWED`
- `REMOTE_PUBLISH_ALLOWED`
- `DEPLOY_ALLOWED`
- `DEPLOY_BLOCKED`
- `ROLLBACK_REQUIRED`
- `RELEASE_BLOCKED`
- `MANUAL_INTERVENTION_REQUIRED`

## 4. 发版建议树

```text
if integration branch not accepted:
  RELEASE_NOT_READY
else if security/hotfix:
  RELEASE_REQUIRED_SINGLE
else if database migration or breaking API:
  RELEASE_REQUIRED_SINGLE
else if release_batch_score >= threshold:
  RELEASE_SUGGESTED
else:
  RELEASE_NOT_READY
```

评分：

```text
release_batch_score =
  issue_count
  + changed_module_count * 1.5
  + migration_count * 3
  + public_api_change * 2
  + security_change * 3
  + unresolved_risk_count * 2
```

## 5. 创建 release branch 决策树

```text
if source branch != accepted integration branch:
  block
else if full quality gates not passed:
  block
else if coverage gates not passed and exemption missing:
  block
else if release note missing:
  block
else if rollback plan missing:
  block
else if release approval required and missing:
  block
else:
  RELEASE_BRANCH_ALLOWED
```

## 6. 部署决策树

```text
if release.mode == branch_only:
  REMOTE_PUBLISH_ALLOWED
else if target environment missing:
  DEPLOY_BLOCKED
else if resource group not healthy:
  DEPLOY_BLOCKED
else if production and approval missing:
  DEPLOY_BLOCKED
else if rollback required and unavailable:
  DEPLOY_BLOCKED
else:
  DEPLOY_ALLOWED
```

## 7. 冒烟和监控决策树

```text
if smoke tests failed:
  if rollback available:
    ROLLBACK_REQUIRED
  else:
    MANUAL_INTERVENTION_REQUIRED
else if monitor window has critical alerts:
  if rollback available:
    ROLLBACK_REQUIRED
  else:
    MANUAL_INTERVENTION_REQUIRED
else:
  mark release healthy
```

## 8. 人工确认条件

- 生产发布。
- 数据库迁移。
- 不可自动回滚。
- 多台生产服务器。
- 健康检查缺失。
- 监控缺失。
- 安全或支付相关变更。

## 9. 产物和日志

产物：

- `.moyuan/lifecycle/releases/`
- `.moyuan/lifecycle/deployments/`
- `.moyuan/lifecycle/rollback/`
- `.moyuan/lifecycle/retrospectives/`

日志：

- `release`
- `git`
- `quality`
- `audit`
- `memory`
- `error`

## 10. 关联配置

- `.moyuan/policies/release.yaml`
- `.moyuan/policies/environments.yaml`
- `.moyuan/policies/server-resources.yaml`
- `.moyuan/policies/secrets.yaml`
- [工程流程规范](../engineering-process-standards.md)

## 11. 验收用例

- 未 accepted 的 integration branch 不能发版。
- breaking API 必须单独发版。
- release note、rollback plan 或覆盖率门禁缺失时不能发版。
- 生产发布缺少审批时不能部署。
- 资源组不健康时不能部署。
- 冒烟失败且可回滚时必须回滚。
