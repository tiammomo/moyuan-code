# 服务器资源管理主线

## 1. 目标

服务器资源管理主线负责把测试开发机、预发机和生产机纳入统一资源清单，长期维护云资产、到期时间、规格、owner、资源组、健康检查、巡检和维护 issue。

这条主线只管理资源事实和维护，不直接执行发布投产。发布投产由 [DevOps 发布投产主线](./devops-release-deployment.md) 负责。

## 2. 输入与输出

输入：

- 服务器连接信息引用。
- 云厂商账号引用。
- 主机基础规格。
- 服务路径和健康检查方式。
- 资源用途分类。
- owner、backup owner 和续费负责人。

输出：

- server resource inventory。
- resource groups。
- resource check report。
- expiration alerts。
- maintenance issues。
- resource change events。

## 3. 资源分类

| 类型 | 用途 | 默认风险 |
| --- | --- | --- |
| `test_dev` | 开发联调、测试验证、部署演练、问题复现 | medium |
| `staging` | 预发布验证、接近生产的回归和冒烟 | high |
| `production` | 正式线上流量 | critical |

生产机必须满足：

- 有 owner 和 backup owner。
- 有 auth_ref，不允许明文密码。
- 有到期时间和续费负责人。
- 有健康检查。
- 有备份或明确说明不能备份的原因。
- 只能通过 release/deploy pipeline 操作。

## 4. 端到端流程

```text
server add
  -> validate secret refs
  -> validate category and owner
  -> validate cloud metadata if cloud host
  -> validate expiration date
  -> validate healthcheck
  -> add host to inventory
  -> assign resource group
  -> run connectivity and health checks
  -> write resource event
```

日常维护：

```text
scheduled maintenance
  -> connectivity check
  -> disk usage check
  -> service health check
  -> backup status check
  -> cloud expiration scan
  -> cost snapshot
  -> create maintenance issue when needed
```

## 5. 决策点

调用策略：

- [服务器资源策略](../policies/server-resource-policy.md)
- [质量与合入策略](../policies/quality-merge-policy.md)
- [Memory 决策策略](../policies/memory-decision-policy.md)

核心决策：

- 新增机器是否允许进入资源清单。
- 资源类别是否为生产。
- 是否缺少 owner、到期时间、健康检查或备份。
- 到期提醒是否生成维护 issue。
- 巡检失败是否阻断投产。
- 生产机远程操作是否必须走审批。

## 6. 配置入口

- `.moyuan/policies/secrets.yaml`
- `.moyuan/policies/server-resources.yaml`
- `.moyuan/policies/environments.yaml`
- `.moyuan/policies/permissions.yaml`
- `.moyuan/policies/logging.yaml`

## 7. Workspace 产物

```text
.moyuan/resources/
  inventory.yaml
  events.jsonl
  checks/
  maintenance/
```

## 8. 日志与审计

必须记录：

- host added/updated/retired。
- owner changed。
- expiration changed。
- resource group changed。
- connectivity check。
- healthcheck result。
- backup status。
- cloud expiration scan。
- production remote command approval。

日志流：

- `run`
- `audit`
- `error`
- `release`，仅发布流程引用资源时记录。

## 9. 验收标准

- 测试开发机和生产机能明确区分。
- 每台服务器有唯一 host id。
- 云服务器有到期时间和续费负责人。
- 生产机缺失备份、健康检查或 owner 时不能投产。
- 资源组可以被环境配置引用。
- 巡检失败能生成维护 issue。
