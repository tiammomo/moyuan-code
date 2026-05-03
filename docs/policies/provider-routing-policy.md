# Provider 路由策略

## 1. 目标

决定不同任务使用 Claude CLI、Codex CLI、GPT、Claude API、国产模型 API、第三方 API 或 `gpt-image-2`，并控制降级、敏感数据边界和成本。

## 2. 输入事实

- task type。
- agent role。
- code sensitivity。
- memory sensitivity。
- provider health。
- model capabilities。
- context size。
- tool requirement。
- budget remaining。
- user/provider policy。
- target output type。

## 3. 决策结果

- `USE_CLAUDE_CLI`
- `USE_CODEX_CLI`
- `USE_GPT_API`
- `USE_CLAUDE_API`
- `USE_DOMESTIC_MODEL_API`
- `USE_THIRD_PARTY_API`
- `USE_GPT_IMAGE_2`
- `ROUTE_BLOCKED`
- `FALLBACK_PROVIDER`

## 4. Runtime 路由树

```text
if task requires repository edits:
  if role == frontend:
    USE_CLAUDE_CLI
  else if role == backend or backend_tuning:
    USE_CODEX_CLI
  else if role == reviewer or tester:
    USE_CODEX_CLI
else if task is architecture planning:
  USE_CLAUDE_API or USE_GPT_API
else if task is memory extraction light:
  USE_DOMESTIC_MODEL_API if data policy allows
else if task is architecture diagram image:
  USE_GPT_IMAGE_2
```

## 5. 敏感数据路由树

```text
if context includes secret or .env value:
  ROUTE_BLOCKED
else if context includes proprietary code and provider disallows sensitive code:
  ROUTE_BLOCKED
else if context includes project memory and provider disallows project memory:
  ROUTE_BLOCKED
else if provider is third_party and task needs code context:
  ROUTE_BLOCKED
else:
  route allowed
```

## 6. 降级树

```text
if primary provider unhealthy:
  try fallback provider
else if rate limited:
  retry with backoff
else if budget exceeded:
  fallback to lower-cost provider if sensitivity allows
else:
  ROUTE_BLOCKED
```

降级不能绕过：

- secret 禁止外发。
- 第三方 API 禁止敏感代码。
- Memory scope 限制。
- 工具执行权限。

## 7. 图像生成路由

```text
if output type == architecture_diagram:
  require diagram spec
  strip secrets/private IP/env values
  USE_GPT_IMAGE_2
```

图像生成不参与代码合入，只生成辅助资产。

## 8. 产物和日志

产物：

- provider selection record。
- fallback record。
- model usage record。
- visual asset record。

日志：

- `model`
- `agent`
- `run`
- `audit`
- `error`

## 9. 关联配置

- `.moyuan/models/providers.yaml`
- `.moyuan/models/routing.yaml`
- `.moyuan/runtimes/agent-runtimes.yaml`
- `.moyuan/policies/permissions.yaml`
- `.moyuan/policies/budget.yaml`
- `.moyuan/visuals/architecture-visuals.yaml`

## 10. 验收用例

- 前端代码任务默认路由 Claude CLI。
- 后端代码任务默认路由 Codex CLI。
- 第三方 API 不能接收敏感代码。
- provider unhealthy 时尝试 fallback。
- `gpt-image-2` 只用于架构可视化，不参与代码生成。
