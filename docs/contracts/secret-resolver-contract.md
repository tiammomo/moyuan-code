# Secret Resolver 契约

本文定义 `env:` / `secret:` 引用如何被解析、注入和审计。它是 Phase 5 后所有外部 adapter 使用凭证的唯一入口。

## 1. 目标

- 配置、日志、Memory、prompt、Console 和 runtime metadata 都只保存 secret reference，不保存明文值。
- Adapter 需要凭证时通过 resolver 临时取值，并只注入到受控子进程环境变量或 HTTP header。
- 每次真实取值都写入 audit log。
- `secret:` 必须声明用途，不能被任意 adapter 复用。

## 2. 引用类型

支持：

```yaml
auth_ref: env:OPENAI_API_KEY
auth_ref: secret:gpt_image_token
```

当前 Phase 5 可执行链路：

```text
secret:gpt_image_token
  -> .moyuan/policies/secrets.yaml
  -> secrets.gpt_image_token.ref = env:OPENAI_IMAGE_API_KEY
  -> resolver 临时读取环境变量
  -> adapter env/header 注入
```

当前不支持：

- secret 明文落盘。
- `secret:` 嵌套指向另一个 `secret:`。
- 未配置 backend 的 Vault/KMS 真实读取。

## 3. policies/secrets.yaml

示例：

```yaml
schema_version: 1

secrets:
  minimax_runtime_token:
    type: token
    ref: env:MINIMAX_API_KEY
    usage:
      - runtime.invoke
      - model.provider.*

  gpt_image_token:
    type: token
    ref: env:OPENAI_IMAGE_API_KEY
    usage:
      - visual.render.script
```

字段规则：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `schema_version` | 是 | 当前固定为 `1` |
| `secrets.<id>.type` | 是 | `token`、`ssh_key`、`private_key` 等 |
| `secrets.<id>.ref` | 是 | 当前可执行 backend 为 `env:KEY` |
| `secrets.<id>.usage` | 是 | 允许用途，可使用 `model.provider.*` 这种前缀通配 |

## 4. Resolver 输出

运行时对象：

```go
type Resolution struct {
  Reference string
  Source    string
  Name      string
  SecretID  string
  Type      string
  Purpose   string
  AdapterID string
  Status    string
  Reason    string
  EnvKey    string
}
```

明文值只能通过进程内 `Value()` 获取，不参与 JSON 序列化，不写入 metadata。

状态：

| 状态 | 说明 |
| --- | --- |
| `ok` | 已解析，可临时注入 |
| `missing` | 引用为空、secret 未登记、policy 缺失或环境变量缺失 |
| `denied` | secret 已登记，但用途不允许 |
| `invalid` | 引用格式、secret id、env key 或 backend 不合法 |

## 5. 已接入 Adapter

Phase 5 第一批接入：

- Provider ops probe：`model.provider.probe`。
- Provider local status：`model.provider.status`。
- Native Runtime env profile：`runtime.invoke`。
- Visual script render：`visual.render.script`。
- Git provider PR/MR create：`pull_request.create`。
- Release provider publish：`release.provider.publish`。

注入规则：

- Claude CLI profile 注入 `ANTHROPIC_AUTH_TOKEN`。
- Codex CLI profile 注入 `OPENAI_API_KEY`。
- gpt-image-2 script 注入 `OPENAI_API_KEY`。
- GitHub release provider publish 注入 HTTP `Authorization` header。
- Gitee release provider publish 注入 API request body 的 `access_token` 字段；请求体不得写入日志、execution、evidence 或 Memory。
- Metadata 只记录 `env_keys`、`auth_ref`、provider id 和 model id。

## 6. 审计和脱敏

每次 `Resolve` 真实取值都记录：

```text
audit event = secret.access.granted | secret.access.denied
```

审计字段只包含：

- `ref`
- `source`
- `secret_id`
- `type`
- `purpose`
- `adapter_id`
- `status`
- `reason`
- `env_key`

禁止记录：

- secret value。
- HTTP Authorization header 完整值。
- `.env` 文件内容。
- SSH 私钥内容。

## 7. 测试要求

Secret resolver 或 adapter 改动必须覆盖：

- direct `env:` 正常解析。
- `secret:` 经 `policies/secrets.yaml` 间接解析。
- usage 不匹配时拒绝。
- JSON、日志、runtime metadata、render execution 不包含 secret value。
