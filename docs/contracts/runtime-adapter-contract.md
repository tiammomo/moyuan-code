# Runtime Adapter 契约

## 1. 目标

统一 Claude CLI、Codex CLI 和后续 Native Agent Runtime 的调用边界，确保它们能被 Orchestrator 安全调度，并且所有文件写入、命令执行、输出和错误都可审计。

## 2. Runtime 类型

首批支持：

- `claude_cli`
- `codex_cli`

后续可扩展：

- 本地自定义 agent。
- 远程 agent service。
- CI agent。

## 3. 输入契约

```ts
interface RuntimeInvocation {
  run_id: string;
  subagent_id?: string;
  project_id: string;
  issue_id: string;
  auth_context: {
    actor_id: string;
    actor_type: "user" | "service_account" | "system";
    auth_method: "local_identity" | "session" | "api_token" | "service_account";
    roles: string[];
    trace_id: string;
  };
  role: string;
  skill_binding_ids: string[];
  runtime_id: string;
  provider_id?: string;
  model_id?: string;
  mode: "ask" | "code" | "review" | "test" | "plan";
  workspace_root: string;
  worktree_path: string;
  branch: string;
  compiled_prompt_path: string;
  context_files: string[];
  allowed_paths: string[];
  protected_paths: string[];
  allowed_commands: string[];
  timeout_seconds: number;
  env_refs: string[];
  provider_env_profile?: {
    enabled: boolean;
    allowed_env_keys: string[];
  };
}
```

## 4. 输出契约

```ts
interface RuntimeResult {
  run_id: string;
  subagent_id?: string;
  runtime_id: string;
  provider_id?: string;
  model_id?: string;
  status:
    | "completed"
    | "failed"
    | "timeout"
    | "cancelled"
    | "needs_user_input";
  summary: string;
  changed_files: Array<{
    path: string;
    change_type: "added" | "modified" | "deleted" | "renamed";
    reason?: string;
  }>;
  commands: Array<{
    command: string;
    status: "passed" | "failed" | "skipped";
    exit_code?: number;
  }>;
  tests: Array<{
    name: string;
    status: "passed" | "failed" | "skipped";
  }>;
  risks: Array<{
    severity: "low" | "medium" | "high" | "blocker";
    message: string;
  }>;
  runtime_signals: Array<{
    signal_type: "test_failure" | "runtime_error" | "review_finding";
    summary: string;
    evidence_refs: string[];
  }>;
  memory_candidates: string[];
  native_session_id?: string;
  recovery_id?: string;
  env_keys?: string[];
}
```

Provider env profile 规则：

- Native Runtime 只能接收白名单环境变量，不能继承完整用户环境。
- `provider_id` 指向 Provider Registry；`auth_ref` 在执行前解析为子进程环境变量，结果文件和日志只记录 `env_keys`。
- `claude_cli` 可注入 `ANTHROPIC_BASE_URL`、`ANTHROPIC_AUTH_TOKEN` 和模型相关变量，用于 MiniMax-M2.7 等 Anthropic-compatible provider。
- Runtime Adapter 不负责决定代码是否可合入，所有 diff 必须回到质量门禁和 review。

Native Runtime recovery 输出：

```ts
interface RuntimeRecovery {
  id: string;
  run_id: string;
  subagent_id?: string;
  issue_id?: string;
  runtime_id: "claude_cli" | "codex_cli";
  provider_id?: string;
  model_id?: string;
  native_session_id: string;
  status: "archived" | "blocked";
  failure_category:
    | "runtime_failed"
    | "runtime_unavailable"
    | "pre_existing_dirty_worktree"
    | "protected_paths_changed"
    | "diff_unavailable";
  fallback_candidate?: "claude_cli" | "codex_cli";
  resume_hint: string;
  prompt_path?: string;
  metadata_path?: string;
  stdout_path?: string;
  stderr_path?: string;
  diff_summary_path?: string;
  changed_files: string[];
  risks: string[];
}
```

当前实现只归档恢复上下文和建议 fallback candidate，不自动执行真实 resume，也不自动切换 runtime。

## 5. 执行约束

Runtime 必须：

- 在 issue worktree 内执行。
- 继承 Orchestrator 下发的 `auth_context`，不能自行提升身份或角色。
- 执行前记录 git diff。
- 执行后记录 git diff。
- 不能写 protected paths。
- 不能绕过 command policy。
- 不能直接合入分支。
- 不能直接 push。
- 不能跳过质量门禁。

## 6. 健康检查

```ts
interface RuntimeHealth {
  runtime_id: string;
  command: string;
  ok: boolean;
  version?: string;
  last_checked_at: string;
  error?: string;
}
```

健康检查失败时：

- 不启动新 run。
- 已在 ready queue 的 issue 标记 `waiting_runtime_slot` 或 `runtime_unavailable`。
- 如果策略允许，尝试 fallback runtime。

## 7. 错误分类

| 错误 | 含义 | 默认处理 |
| --- | --- | --- |
| `runtime_unavailable` | CLI 不存在或健康检查失败 | fallback 或 blocked |
| `auth_failed` | 本地登录或 API key 缺失 | 需要用户处理 |
| `timeout` | 超时 | 可重试一次 |
| `permission_denied` | 文件或命令越权 | 阻断并审计 |
| `invalid_output` | 输出不符合契约 | 返工或 fallback |
| `dirty_worktree` | worktree 不可安全写入 | 阻断 |

## 8. 日志要求

必须记录：

- runtime started。
- runtime completed/failed。
- command started/completed/failed。
- diff before/after。
- native session id。
- recovery id。
- fallback decision。
- permission denied。

日志流：

- `agent`
- `run`
- `model`
- `git`
- `audit`
- `error`

## 9. 验收用例

- Claude CLI 不健康时，frontend issue 不启动或走 fallback。
- Codex CLI 写 protected path 时被阻断。
- Runtime 输出缺少 changed files 时返回 `invalid_output`。
- Runtime 完成后必须进入质量门禁。
- Runtime 不允许直接 push。
- Native Runtime 失败后能通过 CLI/API 查询 recovery 记录、stdout/stderr、diff summary 和 fallback candidate。
