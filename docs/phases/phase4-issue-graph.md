# Phase 4 实现 Issue Graph

状态：in_progress
责任角色：orchestrator_owner + api_owner + frontend_owner + security_owner + devops_owner + qa_owner
最后更新：2026-05-05

Phase 4 的目标是把 Phase 3 已完成的单机受控执行能力推进到团队协作、审计可查、审批可追踪和生产维护可持续。Phase 4 不直接追求大规模分布式执行，优先补齐生产团队使用前最容易失控的权限、审计、审批、Git 协同和服务器维护边界。

## 1. Phase 4 目标

- 所有 run、agent、model、Git、质量、发布、部署、服务器和错误日志可以按 project、issue、run、channel 查询。
- 高风险操作形成 approval record，能追溯申请人、审批人、决策、理由和执行结果。
- 本地团队模式具备 session、API token、service account 和最小权限边界。
- GitHub/Gitee PR/MR 计划、链接和状态可以被系统记录和同步。
- 服务器资源能长期维护，包含到期时间、巡检状态、续费提醒、退役状态和环境引用。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase4-001` | `audit-log-query-api-console` | completed | 提供核心 JSONL/GORM 审计日志查询 API，并在 Console 增加 Audit 面板 | Phase 3 readiness | `api_owner` + `frontend_owner` | 能按 channel、issue、run、limit 查询脱敏后的核心日志 |
| `phase4-002` | `approval-record-store-api` | completed | 为 release、deploy、visual script、provider probe 等高风险动作建立 approval record | `phase4-001` | `security_owner` | 高风险动作有 request、decision、reason、actor 和 result |
| `phase4-003` | `team-auth-session-token-baseline` | planned | 增加 local team session、API token、service account 的最小实现 | `phase4-002` | `security_owner` + `api_owner` | API 能区分 user/session/token/service account，并写入审计 |
| `phase4-004` | `git-pr-mr-plan-sync` | planned | 建立 GitHub/Gitee PR/MR plan、remote link、status refresh 和失败降级 | `phase4-001` | `git_owner` | 系统能记录 PR/MR 计划和远程状态，不直接绕过 review |
| `phase4-005` | `server-resource-maintenance` | planned | 服务器到期、巡检、续费提醒、退役和环境引用维护 | `phase3-002e` | `devops_owner` | 资源生命周期状态可查询、可审计、可被部署流水线引用 |

## 3. 建议执行顺序

1. 先做 `phase4-001`，让后续审批、团队鉴权、Git 同步和服务器维护都有统一审计查询入口。
2. 再做 `phase4-002`，把高风险动作从“受控 API”升级为“可审批、可追责的操作记录”。
3. `phase4-003` 在审批记录后推进，避免先做登录系统但没有审计和授权落点。
4. `phase4-004` 和 `phase4-005` 可以并行，分别补齐代码协作和服务器维护主线。

## 4. 收口规则

- Phase 4 的任何新操作入口都必须写入 audit log。
- API 返回日志和错误内容必须脱敏，不允许泄露 token、API key、SSH key、云凭证或 `.env`。
- 审批、权限、Git remote 和服务器维护状态必须有测试覆盖。
- Console 只能展示和触发后端受控 API，不能直接伪造权威状态。
