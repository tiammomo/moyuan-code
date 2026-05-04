# 模型与工具适配规划

## 1. 适配层目标

模型和工具接入必须通过 Adapter Layer，避免上层 Orchestrator 直接依赖某一个厂商 SDK、CLI 参数或响应格式。

Adapter 统一解决：

- 鉴权。
- 服务商账号登记。
- 请求构造。
- 响应解析。
- 流式输出。
- 工具调用。
- 文件变更。
- 错误分类。
- 重试和降级。
- 成本和用量统计。
- 能力声明。
- 健康检查和故障降级。

## 2. Provider Registry

模型服务商必须先进入 Provider Registry，再被路由策略引用。目标权威配置是 `models/providers.yaml`，运行记录落在 `.moyuan/model-ops/`。

当前实现已落地 Provider Registry 和 Phase 2 ops snapshot：

- 运行期文件：`.moyuan/models/providers.json`。
- CLI：`moyuan model provider add/list/show/ops/refresh/disable`、`moyuan model route`。
- API：`GET/POST /v1/projects/:project_id/providers`、`GET /v1/projects/:project_id/providers/:provider_id`、`POST /v1/projects/:project_id/providers/:provider_id/ops`、`POST /v1/projects/:project_id/providers/ops/refresh`、`POST /v1/projects/:project_id/providers/:provider_id/disable`、`POST /v1/projects/:project_id/provider-route`。
- 已实现约束：`auth_ref` 只能是 `env:` 或 `secret:` 引用；不会保存明文 API key。
- 已实现默认路由：前端和架构类代码任务路由到 `claude_cli`，后端、调优、测试、review 和修复类任务路由到 `codex_cli`，启用后的 API provider 可承担 memory 抽取、规划或图像类任务。
- 已实现 provider env profile：绑定到 `claude_cli` 或 `codex_cli` 的 provider 可通过 Secret Resolver 注入 `base_url`、模型名和 `auth_ref` 对应的环境变量；runtime metadata 只记录 `env_keys`，不记录 token 值。
- 已实现 ops snapshot：provider 可记录 `health`、`quota`、`usage` 和 `cost`，路由会因 `unhealthy/down`、`quota.exhausted`、`cost.exceeded` 给出明确阻断原因。
- 已实现 ops refresh：自动检查 native runtime 是否可发现、API provider 的 `auth_ref/base_url` 配置完整性，并按 quota/cost 阈值刷新状态；默认不外呼云厂商账单或模型 API。传入 `probe=true` 或 CLI `--probe --approved` 时，才通过轻量 HTTP probe 检查 API provider 可达性和鉴权状态，探测过程不落盘 token；未带 approval 时会生成 approval record 并返回 `provider_probe_approval_required`。
- 已实现 task model strategy：`model route --strategy <strategy>` 和 `provider-route` API 可指定 `frontend-first`、`backend-safe`、`low-cost-memory`、`image-diagram`、`planning` 策略。
- 已实现 Native Runtime recovery：Claude/Codex CLI 失败后会生成 `recovery_id`、`native_session_id`、stdout/stderr 归档、diff summary 引用和 fallback candidate。

`providers.json` 是 Beta 运行状态快照；后续 schema validator 完成后，再把同字段收敛到 `models/providers.yaml`，并保留 snapshot 用于审计。

纳管对象：

- 官方 API：OpenAI/GPT、Anthropic/Claude、智谱 GLM、MiniMax、DeepSeek、DashScope 等。
- 图像生成 API：OpenAI `gpt-image-2`，用于架构流程图和可视化讲解资产。
- Agent CLI：Codex、Claude Code。
- 第三方 API：OpenAI-compatible 聚合网关、企业内部代理、私有模型网关。

每个服务商账号必须声明：

- `vendor`：真实厂商或第三方网关。
- `api_type`：`openai`、`anthropic`、`openai-compatible`、`minimax`、`dashscope`、`codex`、`claude-code` 等。
- `base_url`：官方或第三方网关地址。
- `auth_ref`：密钥引用，不保存明文。
- `enabled`：是否启用。
- `data_policy`：是否允许敏感代码、项目记忆、生产事故上下文。
- `models`：模型 id、alias、能力声明和限制。
- `quotas`：限流、日预算、超时。
- `health_checks`：可用性检测和自动降级。

Phase 2 当前运行期 ops 字段：

- `health.status`：`ok`、`healthy`、`degraded`、`unhealthy`、`down`、`unknown`。
- `quota.status`：`ok`、`warning`、`exhausted`、`unknown`。
- `usage`：请求数、输入/输出/总 token、统计窗口和更新时间。
- `cost`：币种、预估成本、预算和 `ok`、`warning`、`exceeded` 状态。

Phase 2 当前模型策略：

- `frontend-first`：优先按前端代码任务选择 `claude_cli` 或绑定到 `claude_cli` 的 provider profile。
- `backend-safe`：优先按后端代码任务选择 `codex_cli`。
- `low-cost-memory`：优先选择允许项目 Memory 的低成本国产/API provider。
- `image-diagram`：强制走 `gpt_image_2` 图像生成路由。
- `planning`：按架构/需求规划任务选择 Claude/GPT/可信第三方 provider 或 `claude_cli` fallback。

第三方 API 必须额外声明：

- `upstream_vendor`，如果无法确定则标记 `unknown`。
- `require_provider_label: true`。
- `allowed_use_cases`，默认只能用于低风险文本、摘要、分类和轻量 memory 抽取。
- 禁止处理密钥上下文、生产事故、完整项目 memory dump 和高敏私有代码。

## 3. Adapter 能力声明

每个 adapter 必须实现 `capabilities`：

```yaml
name: claude_code
type: cli-agent
capabilities:
  chat: true
  code_edit: true
  file_read: true
  file_write: true
  shell_exec: true
  mcp: true
  streaming: true
  session_resume: true
  structured_output: partial
  native_agent_runtime: true
limits:
  max_context_tokens: provider-defined
  max_output_tokens: provider-defined
auth:
  methods:
    - env
    - local_cli_login
```

## 4. Claude Code Adapter

Claude Code 适合承担复杂代码理解、跨文件修改、长任务执行和已有 Claude Code 工作流复用。

Claude Code 在 Moyuan 中作为 Native Agent Runtime 接入，不是普通文本模型。它可以被分配到 issue worktree，接收完整任务上下文，直接完成代码修改，并把结果交回 Moyuan 进行 diff 审计、质量门禁和合入。

规划能力：

- 支持 headless/print 模式执行一次性任务。
- 支持继续或恢复会话。
- 支持读取项目内 `.claude/settings.json`、`.claude/settings.local.json` 和 `CLAUDE.md` 相关约定。
- 支持 MCP 工具配置透传。
- 支持通过 hooks 收集审计事件。
- 支持将 Moyuan 的 task context 转换为 Claude Code prompt。
- 支持 issue worktree 内执行，禁止直接操作未授权路径。
- 支持会话 id 持久化，任务失败后可 resume。
- 支持输出契约解析：summary、changed files、tests、risks、follow-up。
- 支持运行前后 diff 快照，用于判断真实改动。

建议封装命令：

```text
claude -p "<compiled task prompt>"
claude -c -p "<follow-up prompt>"
claude -r "<session-id>" "<resume prompt>"
claude mcp ...
```

注意：

- 不把 Claude Code 的项目配置和 Moyuan 项目配置混在一起。
- Moyuan 只生成必要的上下文和策略，不直接覆盖用户已有 `.claude/` 配置。
- 对文件写入和命令执行做 Moyuan 自身审计。
- Claude Code 生成的代码必须回到 Moyuan 质量门禁；不能因为 Claude Code 已执行自检就直接合入。

### Claude CLI + MiniMax-M2.7 Profile

前端复杂 UI 首版、视觉探索和高交互页面可以使用本地 `claude` CLI 承接代码生成，同时通过 MiniMax 的 Anthropic-compatible endpoint 提供模型能力。样式基线稳定后的前端代码修改、测试、修复和重构可以改由 Codex CLI 承接。Moyuan 的职责是选择 provider、注入运行环境、捕获 diff、执行质量门禁和 review；Native Runtime 只负责在授权 worktree 内生成候选代码。

推荐登记方式：

```bash
export MINIMAX_API_KEY="<local-only>"

./bin/moyuan model provider add \
  --id minimax-m27-claude \
  --name "MiniMax M2.7 via Claude CLI" \
  --vendor minimax \
  --api-type anthropic-compatible \
  --base-url https://api.minimaxi.com/anthropic \
  --auth-ref env:MINIMAX_API_KEY \
  --runtime claude_cli \
  --model MiniMax-M2.7 \
  --use-case frontend \
  --allow-sensitive-code \
  --allow-project-memory
```

运行时效果：

- `model route --role frontend --repo-edit` 会根据 issue intent 选择 Runtime：复杂 UI 首版优先启用且声明 `frontend` use-case 的 `claude_cli` provider；工程修改、测试修复和重构可选择 `codex_cli`。
- `runtime invoke claude_cli --provider minimax-m27-claude` 会向本次子进程注入 `ANTHROPIC_BASE_URL`、`ANTHROPIC_AUTH_TOKEN`、`ANTHROPIC_MODEL`、`ANTHROPIC_DEFAULT_*_MODEL`、`API_TIMEOUT_MS` 和 `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`。
- `orchestrator run <issue-id> --role frontend --runtime claude_cli` 在未显式传 `--provider` 时，会根据 Provider Route 自动选择匹配 provider。
- `.moyuan/runtime/*-native.json` 只保存 provider id、model id、command、stdout/stderr 和 `env_keys`；不会保存 `ANTHROPIC_AUTH_TOKEN` 的值。

约束：

- `auth_ref` 必须是 `env:MINIMAX_API_KEY` 或在 `policies/secrets.yaml` 登记过用途的 `secret:minimax_runtime_token`；不能写明文 key。
- Native Runtime 通过 Secret Resolver 解析 `env:` 和 `secret:`；当前可执行的 `secret:` backend 是 `secret:id -> policies/secrets.yaml -> env:KEY`。
- 允许处理代码上下文的 provider 必须显式开启 `allow_sensitive_code` 和 `allow_project_memory`，否则只能作为低风险规划、摘要或 memory 抽取候选。

## 5. Codex Adapter

Codex 适合承担代码生成、代码审查、测试补齐、云端或本地 Agentic coding 任务。

Codex CLI 在 Moyuan 中也作为 Native Agent Runtime 接入。它优先用于代码实现、测试补齐、审查修复和自动化返工；Moyuan 负责编排上下文、隔离分支、捕获 diff、执行质量门禁和决定是否合入。

规划能力：

- 支持本地 Codex CLI。
- 支持 Codex SDK 或 API 方式创建 coding task。
- 支持 Codex cloud task 的后续扩展。
- 支持 MCP server 配置，尤其是开发文档 MCP、memory MCP、GitHub MCP。
- 支持区分 ask/review/code 三类任务。
- 支持将 Moyuan 的 role、task、workspace policy 编译为 Codex prompt。
- 支持 ask/review/code 三种模式对应不同写权限。
- 支持 issue worktree 内执行，运行前后捕获 git diff。
- 支持把 Codex 输出转成 run report 和 review input。
- 支持 Codex CLI 失败后降级到 Claude CLI 或模型 API。

建议任务模式：

```yaml
task_modes:
  ask:
    write_allowed: false
    use_cases:
      - 架构理解
      - 重构建议
      - 方案比较
  review:
    write_allowed: false
    use_cases:
      - diff 审查
      - 风险识别
      - 测试缺口分析
  code:
    write_allowed: true
    use_cases:
      - 功能实现
      - bug 修复
      - 测试补齐
```

注意：

- 云端任务需要明确环境、依赖安装和网络策略。
- 本地 CLI 写入文件时要纳入 Moyuan 的 run diff 审计。
- 模型名称不应硬编码在业务流程中，统一放入 routing policy。
- Codex CLI 的写权限由 Moyuan policy 控制，不能绕过 protected paths 和命令 allowlist。

## 6. 国产大模型 API Adapter

首批建议适配两类：

1. OpenAI-compatible API
   - DeepSeek。
   - 智谱 GLM OpenAI-compatible 形态。
   - Moonshot/Kimi OpenAI-compatible 形态。
   - 第三方 OpenAI-compatible 网关。
   - 其他兼容 `/chat/completions` 或 Responses-like 协议的服务。

2. 厂商原生 API
   - 阿里云通义千问 DashScope。
   - 百度智能云千帆。
   - 火山方舟。
   - 腾讯混元。
   - MiniMax。
   - 智谱 GLM 原生 API。

### 统一接口

```ts
interface ModelAdapter {
  name: string;
  capabilities(): ModelCapabilities;
  invoke(request: ModelRequest): Promise<ModelResponse>;
  stream(request: ModelRequest): AsyncIterable<ModelEvent>;
  estimateCost?(request: ModelRequest): Promise<CostEstimate>;
  validateConfig(): Promise<ValidationResult>;
}
```

### 能力差异处理

不同模型在工具调用、结构化输出、上下文长度、函数调用、JSON 稳定性、代码能力和中文能力上差异明显。

统一处理策略：

- Adapter 声明能力，上层按能力路由。
- 对不支持工具调用的模型，退化为纯文本规划或总结。
- 对 JSON 稳定性较弱的模型，增加 schema repair。
- 对上下文较短的模型，优先提供摘要和引用。
- 对代码能力较弱的模型，限制为需求分析、总结、分类、文档草稿。

## 7. Image Adapter

Image Adapter 负责图像生成和图像编辑能力。首个目标是接入 OpenAI `gpt-image-2`，用于辅助生成被管理项目的架构流程设计图和配套讲解。

适用场景：

- 项目总体架构图。
- 代码生命周期流程图。
- 多 Agent 协作流程图。
- Issue Graph 可视化。
- 部署拓扑图。
- 发布投产流程图。

统一流程：

- 从项目理解、模块地图、Issue Graph、服务器资源和发布配置中抽取结构化信息。
- 先生成 `diagram_spec`，再生成图像 prompt。
- 真实生成时调用 `gpt-image-2` 生成横版 2K 图片，默认尺寸为 `3072x2048`。
- 保存图片、prompt、spec 和 Markdown 讲解。
- 检查图片可读性、节点一致性和敏感信息泄露。
- 图像脚本执行必须进入受控 execution：默认 dry-run，只在显式 approval 和运行开关同时满足时允许 script mode。

Phase 2 当前落地：

- CLI：`moyuan visuals diagram plan`、`moyuan visuals assets`、`moyuan visuals asset show <asset-id>`、`moyuan visuals asset render <asset-id>`、`moyuan visuals renders`。
- API：`POST /v1/projects/:project_id/visuals/diagrams/plan`、`GET /v1/projects/:project_id/visuals/assets`、`GET /v1/projects/:project_id/visuals/assets/:asset_id`、`POST /v1/projects/:project_id/visuals/assets/:asset_id/render`、`GET /v1/projects/:project_id/visuals/render-executions`。
- 运行期产物：`.moyuan/visuals/specs/`、`.moyuan/visuals/prompts/`、`.moyuan/visuals/assets/`、`.moyuan/visuals/executions/`。
- Diagram plan 会生成脱敏后的 `diagram_spec`、prompt 和 asset record。
- Provider Route 使用 `image-diagram` 策略检查 `gpt_image_2`；provider 不可用时保留 `route_blocked` asset record。
- Render execution 默认 `dry_run`，记录脚本预览和 `no_image_api_called`。
- `script` mode 必须同时满足 `--approved`、`MOYUAN_ALLOW_IMAGE_SCRIPT=1`、provider `auth_ref` 可解析、脚本文件存在；执行只记录 `auth_ref` 和注入的 env key 名，不记录 token 值。
- `script` mode 完成后会生成 `quality` 检查结果，并把可预览产物写入 `.moyuan/visuals/previews/index.jsonl`。

注意：

- `gpt-image-2` 不参与代码生成、代码审查和合入决策。
- 不把密钥、私网 IP、token、环境变量值和账号信息送入图像 prompt。
- 4K 只作为显式实验参数或本地后处理目标，不作为默认生成尺寸。
- 图像产物只能作为辅助说明，真实架构依据仍是项目理解、配置和代码分析结果。

## 8. Tool Adapter

### Shell Adapter

职责：

- 执行命令。
- 捕获 stdout/stderr/exit code。
- 支持超时。
- 支持工作目录限制。
- 支持 allowlist/denylist。
- 写入审计日志。

### Git Adapter

职责：

- 接入本地 Git 仓库。
- clone 远程 Git 仓库。
- 识别 GitHub、Gitee、GitLab 和通用 Git URL。
- 读取 remote、默认分支、当前分支和 head commit。
- 读取状态和 diff。
- 检查 dirty worktree。
- 创建分支。
- 按任务创建 `moyuan/<task-id>-<slug>` 工作分支。
- 同步 base branch。
- 检测 merge/rebase 冲突。
- 提交 commit。
- 推送远程任务分支。
- 通过 provider adapter 创建 PR/MR。
- 生成提交信息建议。
- 创建 patch。
- 防止误覆盖用户改动。

Provider 子适配：

- GitHub Adapter：远程仓库 metadata、PR、issue link、branch protection 读取。
- Gitee Adapter：远程仓库 metadata、PR、issue link、分支保护读取。
- GitLab Adapter：远程仓库 metadata、MR、issue link、branch protection 读取。
- Generic Git Adapter：clone、fetch、checkout、branch、commit、push。

### Test Adapter

职责：

- 识别测试命令。
- 运行测试。
- 解析测试结果。
- 记录失败用例。
- 给 Agent 提供修复上下文。

### MCP Adapter

职责：

- 管理 MCP server 配置。
- 将 MCP tools 暴露给支持 MCP 的 Agent 后端。
- 为不支持 MCP 的模型提供工具代理。

建议首批 MCP：

- filesystem：受控文件访问。
- git：仓库状态和 diff。
- memory：长期记忆。
- docs：官方文档检索。
- issue tracker：GitHub/GitLab/Jira 后续接入。

## 9. 路由策略

模型路由输入：

- task type。
- role。
- 项目策略。
- 预算。
- 数据敏感等级。
- 是否需要代码写入。
- 是否需要工具调用。
- 是否需要中文理解。
- 是否需要长上下文。

示例：

```yaml
routes:
  - when:
      role: backend_tuning
      needs_code_edit: true
      complexity: high
    use: coding_deep_reasoning

  - when:
      role: planner
      language: zh-CN
      complexity: medium
    use: low_cost_text

  - when:
      role: reviewer
      needs_diff_review: true
    use: review_reasoning
```

## 10. 错误分类

Adapter 错误需要统一分类：

- `AUTH_ERROR`：鉴权失败。
- `RATE_LIMIT`：限流。
- `QUOTA_EXCEEDED`：额度不足。
- `MODEL_UNAVAILABLE`：模型不可用。
- `CONTEXT_TOO_LARGE`：上下文过大。
- `TOOL_DENIED`：工具权限不足。
- `COMMAND_FAILED`：命令执行失败。
- `STRUCTURED_OUTPUT_INVALID`：结构化输出解析失败。
- `PROVIDER_INTERNAL_ERROR`：服务端错误。
- `NETWORK_ERROR`：网络错误。

## 11. 外部能力基线

当前规划参考的公开能力基线：

- OpenAI Codex 可作为代码 Agent 读取、修改和运行代码，并支持云端任务、IDE、CLI、SDK 等接入形态。
- OpenAI 文档 MCP 可为 Codex/IDE 提供只读开发文档检索。
- Claude Code 支持 CLI、headless print 模式、项目级 settings、MCP 和 hooks。

这些能力会随厂商版本变化，实际实现时必须以官方文档和本地 CLI `--help` 输出为准。

参考资料：

- OpenAI Codex CLI Getting Started：https://help.openai.com/en/articles/11096431-openai-codex-ci-getting-started
- OpenAI Codex overview：https://help.openai.com/en/articles/11369540-using-codex-with-chatgpt
- Claude Code CLI reference：https://docs.anthropic.com/en/docs/claude-code/cli-reference
- Claude Code settings：https://docs.anthropic.com/en/docs/claude-code/settings
- Claude Code hooks：https://docs.anthropic.com/en/docs/claude-code/hooks
