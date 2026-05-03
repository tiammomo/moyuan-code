# 服务器资源策略

## 1. 目标

决定服务器能否添加到系统、能否进入资源组、是否可以用于投产、是否需要生成维护 issue，以及生产机远程操作是否需要审批。

## 2. 输入事实

- host id。
- category。
- provider。
- auth_ref。
- owner。
- cloud metadata。
- lifecycle expires_at。
- spec。
- healthcheck。
- backup policy。
- resource group。
- latest check result。
- target environment。

## 3. 决策结果

- `HOST_ACCEPTED`
- `HOST_REJECTED`
- `GROUP_ACCEPTED`
- `DEPLOY_RESOURCE_READY`
- `DEPLOY_RESOURCE_BLOCKED`
- `MAINTENANCE_ISSUE_REQUIRED`
- `APPROVAL_REQUIRED`

## 4. 主机登记决策树

```text
if host id is missing or duplicated:
  HOST_REJECTED
else if auth_ref is missing:
  HOST_REJECTED
else if owner is missing:
  HOST_REJECTED
else if category == production and expires_at missing:
  HOST_REJECTED
else if cloud.enabled and cloud metadata missing:
  HOST_REJECTED
else if healthcheck missing:
  HOST_REJECTED
else:
  HOST_ACCEPTED
```

## 5. 生产资源就绪树

```text
if category == production:
  if approval policy missing:
    DEPLOY_RESOURCE_BLOCKED
  else if backup required and backup unavailable:
    DEPLOY_RESOURCE_BLOCKED
  else if latest healthcheck failed:
    DEPLOY_RESOURCE_BLOCKED
  else if expires_at within critical window:
    DEPLOY_RESOURCE_BLOCKED
  else:
    DEPLOY_RESOURCE_READY
```

测试开发机：

```text
if category == test_dev:
  if healthcheck failed:
    DEPLOY_RESOURCE_BLOCKED
  else:
    DEPLOY_RESOURCE_READY
```

## 6. 到期维护树

```text
if expires_at < today:
  MAINTENANCE_ISSUE_REQUIRED(blocker)
else if expires_at within 7 days:
  MAINTENANCE_ISSUE_REQUIRED(critical)
else if expires_at within 30 days:
  MAINTENANCE_ISSUE_REQUIRED(warning)
else:
  no issue
```

## 7. 人工确认条件

- 生产机新增。
- 生产机 owner 或 auth_ref 变更。
- 生产机执行远程命令。
- 生产机备份缺失但仍请求投产。
- 安全组或网络配置变化。

## 8. 产物和日志

产物：

- `.moyuan/resources/inventory.yaml`
- `.moyuan/resources/events.jsonl`
- `.moyuan/resources/checks/`
- `.moyuan/resources/maintenance/`

日志：

- `run`
- `audit`
- `error`
- `release`

## 9. 关联配置

- `.moyuan/policies/server-resources.yaml`
- `.moyuan/policies/environments.yaml`
- `.moyuan/policies/secrets.yaml`
- `.moyuan/policies/permissions.yaml`

## 10. 验收用例

- 生产机无 owner 时不能登记。
- 生产机无到期时间时不能登记。
- 生产机健康检查失败时不能投产。
- 生产机到期 7 天内生成 critical 维护 issue。
- 测试开发机不能访问生产数据。
