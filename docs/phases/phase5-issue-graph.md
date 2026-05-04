# Phase 5 实现 Issue Graph

状态：planned
责任角色：orchestrator_owner + security_owner + api_owner + adapter_owner + frontend_owner + devops_owner + qa_owner
最后更新：2026-05-05

Phase 5 的目标是把 Phase 4 的可审计状态对象推进到“真实外部执行前可强制约束”的门禁层。Phase 5 仍保持真实外部写操作默认关闭，先完成鉴权、Secret 解析、adapter preview/dry-run 和 Console 操作表单。

## 1. Phase 5 目标

- API 请求统一解析 AuthContext，并按 role/scope/action/risk 做 allow/deny/approval decision。
- `secret:` 引用可被受控 resolver 解析给 adapter 使用，但不会进入日志、Memory、prompt 或 Console。
- GitHub/Gitee PR/MR adapter 可创建 preview，并在审批后执行真实创建。
- 部署烟测和监控 adapter 可记录线上检查结果，但生产执行必须审批。
- Console 能完成用户/Token/审批/PR/MR/维护的受控操作，而不是只展示状态。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase5-001` | `auth-context-rbac-middleware` | planned | API middleware 解析 session/API token/service account，输出 AuthContext，并执行最小 RBAC | Phase 4 readiness | `security_owner` + `api_owner` | 高风险 API 不能绕过 authz decision，拒绝/审批都有 audit event |
| `phase5-002` | `secret-ref-resolver` | planned | 支持 `secret:` 和 `env:` 引用解析、用途校验、脱敏和 adapter 注入 | `phase5-001` | `security_owner` + `adapter_owner` | secret 明文不落盘，adapter 只拿到允许的环境变量 |
| `phase5-003` | `github-gitee-pr-mr-adapter` | planned | GitHub/Gitee PR/MR preview、create、status refresh 和失败降级 | `phase5-001`,`phase5-002` | `git_owner` | PR/MR 默认 preview，真实 create 需 approval/authz |
| `phase5-004` | `deployment-smoke-monitor-adapters` | planned | 部署后 smoke/monitor 结果记录、失败阻断和 rollback 建议 | `phase5-001`,`phase5-002` | `devops_owner` | test_dev 可 dry-run/record，production 必须 approval |
| `phase5-005` | `console-controlled-forms` | planned | Console 增加审批、用户、Token、PR/MR、维护操作表单 | `phase5-001` | `frontend_owner` | 前端只调用受控 API，不直接伪造权威状态 |

## 3. 建议执行顺序

1. 先做 `phase5-001`，否则后续 Secret、PR/MR 和部署 adapter 没有统一鉴权入口。
2. 再做 `phase5-002`，让外部 adapter 具备安全拿凭证的前提。
3. `phase5-003` 和 `phase5-004` 可以并行，但真实外部写操作必须默认 dry-run/preview。
4. `phase5-005` 在 API 稳定后推进，避免前端表单先行造成状态伪造。

## 4. 收口规则

- Phase 5 的任何真实外部写操作必须同时满足 authz allow、approval required/approved、secret resolver allowed 和 audit event。
- Secret 明文不得出现在 JSON 响应、日志、Memory、prompt、Console 或测试 fixture。
- Adapter 必须先有 preview/dry-run，再允许真实 execute。
- Console 表单提交后以后端返回为权威状态，不在前端自行构造成功状态。
