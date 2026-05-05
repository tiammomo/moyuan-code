# Provider 路由策略

## 1. 目标

决定不同任务使用 Claude CLI、Codex CLI、GPT、Claude API、国产模型 API、第三方 API 或 `gpt-image-2`，并控制降级、敏感数据边界和成本。

## 2. 输入事实

- task type。
- agent role。
- subagent type。
- skill requirements。
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
    if an enabled provider profile is bound to claude_cli and allows frontend:
      inject provider env profile, for example MiniMax-M2.7
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

Subagent 路由补充：

```text
if subagent.type == implementation_subagent and role == frontend:
  USE_CLAUDE_CLI
else if subagent.type == implementation_subagent and role == backend:
  USE_CODEX_CLI
else if subagent.type == verification_subagent:
  USE_CODEX_CLI or trusted API
else if skill requires shell_exec or file_write:
  require Native Agent Runtime
```

Native Runtime profile 选择规则：

```text
if runtime_id == claude_cli and role == frontend:
  prefer enabled provider where runtime_id == claude_cli and allowed_use_cases contains frontend
  require data_policy to allow sensitive code and project memory when repository context is included
  inject only whitelisted ANTHROPIC_* variables
else if runtime_id == codex_cli:
  prefer enabled provider where runtime_id == codex_cli and allowed_use_cases matches role/task
  inject only whitelisted OPENAI_* variables
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

Phase 2 当前实现：

- `provider.health.status == unhealthy/down` 时阻断该 provider，原因形如 `provider_unhealthy:<provider_id>:<status>`。
- `provider.quota.status == exhausted` 时阻断该 provider，原因形如 `provider_quota_exhausted:<provider_id>`。
- `provider.cost.status == exceeded` 时阻断该 provider，原因形如 `provider_budget_exceeded:<provider_id>`。
- API provider 选择会跳过不可用 provider；如果没有可用 API provider，会回到默认 runtime 路由。
- 直接命中的 provider，例如 `gpt_image_2` 图像路由，会返回明确 blocked decision。

Phase 6 当前实现：

- Provider ops update/refresh 会追加 `.moyuan/models/provider-telemetry.jsonl`。
- `model provider telemetry` 和 `GET /providers/telemetry` 可查询 provider health/quota/usage/cost/feedback 历史。
- `provider-route` 返回 `signals`，包含参与决策的 health、quota、cost 和 quality 状态。
- Runtime execution feedback 会累计本地 token 估算；配置 token 单价后会同步更新成本估算，配置 token limit 后会同步扣减额度。
- Telemetry 只记录状态、数值型 token 估算、成本估算和 reason，不记录 prompt、模型响应或 secret。

Phase 9 route explanation v2 当前实现：

- `route.explanation`：包含 summary、selected provider、selected reason、strategy、candidate count、selected/skipped/blocked count。
- `route.candidates[]`：每个 provider 输出 `provider_id/runtime_id/vendor/api_type/model_id/status/reason/score/signals`。
- `candidate.status`：`selected` 表示本次路由命中；`skipped` 表示能力或优先级不匹配；`blocked` 表示 disabled、secret/data policy、health、quota 或 cost 阻断。
- `candidate.signals`：继承 provider 的 health/quota/cost/quality signals，并追加 `selection` signal。该字段只解释路由，不授权绕过门禁。
- 不记录 prompt、模型响应、token 值、API key、SSH key 或完整 stdout/stderr。

模型策略当前实现：

- `frontend-first`：将任务按前端代码路径路由，默认进入 `claude_cli` 或其 provider profile。
- `backend-safe`：将任务按后端代码路径路由，默认进入 `codex_cli`。
- `low-cost-memory`：将任务路由到允许项目 Memory 的低成本国产/API provider。
- `image-diagram`：将任务路由到 `gpt_image_2` 图像生成 provider。
- `planning`：将任务路由到架构/需求规划 provider 或 `claude_cli` fallback。
- 策略只改变候选方向，不绕过 secret、sensitive code、project memory、provider health、quota 或 cost 阻断。

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

- 前端复杂 UI 首版和视觉探索默认路由 Claude CLI；样式稳定后的前端代码修改、测试、修复和重构可以路由 Codex CLI。
- 后端代码任务默认路由 Codex CLI。
- 第三方 API 不能接收敏感代码。
- provider unhealthy 时尝试 fallback。
- `gpt-image-2` 只用于架构可视化，不参与代码生成。
