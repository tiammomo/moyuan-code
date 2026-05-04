# 日志与审计事件契约

## 1. 目标

定义 Moyuan 核心日志和审计事件结构，保证 run、agent、model、git、quality、release、memory、audit、error 能通过 trace_id 串联。

## 2. 通用事件结构

```ts
interface LogEvent {
  event_id: string;
  event_type: string;
  stream:
    | "run"
    | "agent"
    | "model"
    | "git"
    | "quality"
    | "release"
    | "memory"
    | "audit"
    | "error";
  timestamp: string;
  project_id?: string;
  organization_id?: string;
  trace_id: string;
  actor_id?: string;
  actor_type?: "user" | "service_account" | "system";
  auth_method?: "local_identity" | "session" | "api_token" | "service_account";
  run_id?: string;
  epic_id?: string;
  issue_id?: string;
  subagent_id?: string;
  agent_role?: string;
  runtime_id?: string;
  skill_id?: string;
  branch?: string;
  commit?: string;
  severity?: "debug" | "info" | "warning" | "error" | "critical";
  payload: Record<string, unknown>;
}
```

## 3. 状态变更事件

```ts
interface StateChangedEvent extends LogEvent {
  event_type: "state_changed";
  payload: {
    object_type: string;
    object_id: string;
    previous_status: string;
    next_status: string;
    reason: string;
    triggered_by: string;
  };
}
```

状态来源见 [状态机总表](../foundations/state-machine-catalog.md)。

## 4. 审计事件

必须进入 `audit`：

- login/logout/session revoked。
- api token created/revoked/rotated。
- membership changed。
- role changed。
- auth decision allow/deny/require approval。
- approval requested/granted/rejected。
- `approval.requested`、`approval.decided`。
- protected path access denied。
- secret access requested/granted/denied。
- high risk command blocked。
- production deploy approval。
- production remote command。
- permission policy override。
- provider sensitive context blocked。

审计事件要求：

- append-only。
- 默认不允许删除。
- payload 脱敏。
- 必须包含 actor 或 triggered_by。

## 5. 脱敏规则

禁止写入日志：

- API key。
- token。
- password。
- SSH private key。
- `.env` 明文。
- 完整 prompt。
- 完整 model response。

允许写入：

- secret ref。
- provider id。
- model alias。
- token usage。
- cost。
- command exit code。
- diff summary。

## 6. 必填事件矩阵

| 主线 | 必填事件 |
| --- | --- |
| 平台用户与访问控制 | auth.login、auth.logout、auth.session.created、auth.session.revoked、auth.token.created、auth.token.revoked、auth.service_account.upserted、auth.decision.deny、auth.approval.created |
| 项目接入与阅读理解 | project_added、repository_cloned、comprehension_started、comprehension_completed |
| 需求规划与 Issue 编排 | requirement_refined、clarification_decided、issue_graph_created、schedule_created |
| 代码开发 | subagent.created、subagent.dispatched、runtime_started、runtime_completed、quality_started、review_completed |
| Subagent 与 Skills | subagent.planned、subagent.completed、subagent.failed、skill.recommended、skill.bound、skill.effectiveness.recorded |
| 运行反馈与自我修复 | self_repair.signal.captured、self_repair.bug.classified、self_repair.repair.planned、self_repair.repair.completed |
| 代码管理 | branch_created、worktree_created、commit.created、commit.blocked、merge_attempted、merge_completed、git_provider.plan.created、git_provider.status.synced |
| 服务器资源管理 | host_added、resource_check_completed、expiration_alert_created、server_resource.maintenance_scan、server_resource.renewed、server_resource.retired |
| DevOps 发布投产 | release_suggested、release_branch_created、release.coverage_checked、release_note.generated、deploy_started、smoke_completed、rollback_started |

## 7. 错误事件

```ts
interface ErrorEvent extends LogEvent {
  stream: "error";
  payload: {
    error_code: string;
    message: string;
    recoverable: boolean;
    recovery_action?: string;
    stacktrace_ref?: string;
  };
}
```

错误事件不得包含明文 secret 或完整 prompt。

## 8. 验收用例

- 任意 run 可以通过 `trace_id` 找到 agent、model、git、quality 和 error 事件。
- 任意写入、Git、服务器、发布和部署操作可以通过 `actor_id` 找到触发身份。
- 生产部署必须有 approval audit event。
- secret 访问必须有 audit event。
- 日志中出现明文 API key 时校验失败。
- 状态变化必须生成 `state_changed` 事件。

## 9. 当前查询接口

Phase 4 第一批已提供核心日志查询视图：

```http
GET /v1/projects/:project_id/audit-events?channel=all&limit=20
GET /v1/projects/:project_id/audit-events?channel=audit&issue_id=<issue-id>
GET /v1/projects/:project_id/audit-events?channel=run&run_id=<run-id>
```

查询参数：

- `channel` 或 `stream`：日志流名称，支持 `all` 聚合 `.moyuan/logs/*.jsonl`。
- `issue_id`：按 issue 过滤。
- `run_id`：按 run 过滤。
- `event`：按事件名过滤。
- `limit`：返回数量，最大 100。

返回结构：

```ts
interface AuditEventView {
  id: string;
  stream: string;
  channel: string;
  event: string;
  ts: string;
  issue_id?: string;
  run_id?: string;
  subagent_id?: string;
  trace_id?: string;
  status?: string;
  decision?: string;
  reason?: string;
  data: Record<string, unknown>;
}
```

实现规则：

- API 返回前必须脱敏 token、API key、password、secret、credential 和 private key。
- `channel`/`stream` 只能是字母、数字、下划线和短横线，禁止路径穿越。
- Console 只消费只读查询结果，不直接改写日志。
