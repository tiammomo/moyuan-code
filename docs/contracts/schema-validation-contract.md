# Schema 校验契约

## 1. 目标

把 [配置 Schema 规则](../configuration-schema-spec.md) 转成机器可执行的校验契约，保证 `.moyuan/` 配置在进入任意主线前都能被稳定验证。

## 2. 适用范围

必须校验：

- `.moyuan/project.yaml`
- `.moyuan/repository.yaml`
- `.moyuan/models/providers.yaml`
- `.moyuan/models/routing.yaml`
- `.moyuan/runtimes/agent-runtimes.yaml`
- `.moyuan/agents/roles.yaml`
- `.moyuan/agents/teams.yaml`
- `.moyuan/policies/access.yaml`
- `.moyuan/policies/*.yaml`
- `.moyuan/skills/enabled.yaml`

## 3. 校验接口

```ts
type ValidationSeverity = "error" | "warning";

interface ValidationIssue {
  path: string;
  code:
    | "missing_required"
    | "invalid_type"
    | "null_not_allowed"
    | "empty_not_allowed"
    | "conditional_required"
    | "must_be_null_when"
    | "must_be_empty_when"
    | "reference_not_found"
    | "secret_plaintext"
    | "permission_denied"
    | "unknown_field";
  message: string;
  severity: ValidationSeverity;
}

interface ValidationResult {
  ok: boolean;
  schema_version: number;
  file: string;
  issues: ValidationIssue[];
}

interface ConfigValidator {
  validateFile(path: string): Promise<ValidationResult>;
  validateWorkspace(root: string): Promise<ValidationResult[]>;
}
```

## 4. 校验顺序

```text
read yaml
  -> parse schema_version
  -> validate known file
  -> validate required fields
  -> validate type
  -> validate null/empty rules
  -> validate conditional required
  -> validate must_be_null_when / must_be_empty_when
  -> validate references
  -> validate secret references
  -> validate permission constraints
  -> return structured result
```

## 5. Secret 校验

必须拒绝：

- 明文 API key。
- 明文 token。
- 明文 session secret。
- 明文 SSH private key。
- 明文 password。
- `.env` 内容复制到 YAML。

允许：

- `env:OPENAI_API_KEY`
- `secret:github_token`
- `vault:path/to/secret`

## 6. 引用校验

必须检查：

- provider 引用的 account 存在。
- routing 引用的 provider 和 model alias 存在。
- role 引用的 model policy 存在。
- team 引用的 role 存在。
- access policy 引用的 project role 存在。
- self_repair require_approval_for 只能引用已定义风险触发器。
- environment 引用的 resource group 存在。
- resource group 引用的 host id 存在。
- release 引用的 remote provider 已配置。

## 7. 输出要求

- 校验错误必须包含字段路径。
- 阻断级错误使用 `severity = error`。
- 可自动补默认值的问题使用 `severity = warning`。
- 不允许只输出自然语言错误。
- 所有校验结果可写入 `run` 或 `audit` 日志。

## 8. 验收用例

- 缺少 `schema_version` 返回 `missing_required`。
- secret 明文返回 `secret_plaintext`。
- `source.type = local_path` 时 `repository.source.url` 非空返回 `must_be_null_when`。
- routing 引用不存在 provider 返回 `reference_not_found`。
- production environment 缺少 rollback 返回 `conditional_required`。
