# 状态机总表

本文统一 Moyuan Code 主要对象的状态来源、状态含义和跨对象流转关系。详细对象字段仍以 [核心数据对象](./core-data-objects.md) 为准，失败恢复以 [失败恢复设计](./failure-recovery.md) 为准。

## 1. 设计原则

- 每个状态必须有唯一 owner。
- 状态变化必须写入日志或审计。
- 失败状态不能静默跳过。
- 主线文档只描述状态如何被使用，不重复完整状态定义。
- 策略文档只定义状态变化的决策条件。

## 2. 状态机目录

| 对象 | 状态 owner | 主要文档 | 日志流 |
| --- | --- | --- | --- |
| User | Identity Manager | [平台用户与访问控制主线](../mainlines/platform-user-access.md) | `audit` |
| Membership | Identity Manager | [平台用户与访问控制主线](../mainlines/platform-user-access.md) | `audit` |
| API Token | Identity Manager | [身份会话契约](../contracts/auth-session-contract.md) | `audit` |
| Auth Session | Identity Manager | [身份会话契约](../contracts/auth-session-contract.md) | `audit`、`error` |
| Approval | Orchestrator | [鉴权与访问控制策略](../policies/auth-access-policy.md) | `audit` |
| Project | Project Workspace Manager | [项目工作空间规范](../project-workspace-spec.md) | `run`、`audit` |
| Project Comprehension | Project Workspace Manager | [项目接入与阅读理解主线](../mainlines/project-comprehension.md) | `run`、`git`、`memory` |
| Epic | Orchestrator | [需求规划与 Issue 编排主线](../mainlines/requirement-planning.md) | `run`、`agent` |
| Issue | Orchestrator | [Issues 编排与并发调度](../issue-orchestration.md) | `run`、`agent`、`quality` |
| Schedule | Scheduler | [Issue 调度策略](../policies/issue-scheduling-policy.md) | `run` |
| Run | Agent Runtime | [参考架构](../reference-architecture.md) | `run`、`agent`、`model` |
| Quality Report | Quality Guard | [质量与合入策略](../policies/quality-merge-policy.md) | `quality` |
| Subagent | Orchestrator | [Subagent 与 Skills 系统方案](../subagents-skills-system.md) | `agent`、`run`、`audit` |
| Skill Definition | Skill Registry | [Subagent 与 Skills 系统方案](../subagents-skills-system.md) | `agent`、`audit` |
| Skill Binding | Skill Registry | [Subagent 与 Skills 系统方案](../subagents-skills-system.md) | `agent`、`audit` |
| Skill Effectiveness | Skill Registry | [Subagent 与 Skills 系统方案](../subagents-skills-system.md) | `agent`、`memory` |
| Runtime Signal | Self Repair Engine | [运行反馈与自我修复主线](../mainlines/runtime-feedback-self-repair.md) | `error`、`quality`、`release` |
| Bug Candidate | Self Repair Engine | [Bug 判断与自我修复策略](../policies/bug-detection-self-repair-policy.md) | `error`、`quality`、`memory` |
| Repair Attempt | Orchestrator | [自我修复契约](../contracts/self-repair-contract.md) | `run`、`quality`、`audit` |
| Improvement Record | Memory Engine | [自我修复契约](../contracts/self-repair-contract.md) | `memory`、`audit` |
| Runtime Session | Agent Runtime | [模型与工具适配规划](../model-tool-adapters.md) | `agent`、`error` |
| Memory Record | Memory Engine | [Agent Memory 系统方案](../agent-memory-system.md) | `memory`、`audit` |
| Server Resource | Resource Manager | [服务器资源策略](../policies/server-resource-policy.md) | `run`、`audit` |
| Release | Release Manager | [发布投产策略](../policies/release-deployment-policy.md) | `release`、`git` |
| Deployment | Deployment Runner | [DevOps 发布投产主线](../mainlines/devops-release-deployment.md) | `release`、`audit`、`error` |

## 3. User 状态

```text
invited -> active -> suspended -> disabled -> archived
```

说明：

- `suspended` 表示临时停用，可恢复。
- `disabled` 表示不可继续执行新操作。

## 4. Membership 状态

```text
invited -> active -> suspended -> removed
```

## 5. API Token 状态

```text
created -> active -> rotated -> revoked
```

终止状态：

```text
expired
```

## 6. Auth Session 状态

```text
created -> active -> idle -> expired
```

终止状态：

```text
revoked
invalid
```

## 7. Approval 状态

```text
requested -> approved -> consumed -> archived
```

失败或终止状态：

```text
rejected
expired
cancelled
```

## 8. Project 状态

```text
created -> onboarding -> comprehending -> ready -> active -> archived
```

失败出口：

```text
onboarding_failed
comprehension_failed
```

进入 `ready` 的条件：

- repository 已绑定或 clone 完成。
- `.moyuan/` 已初始化。
- full comprehension 已完成。
- 基础配置通过 schema 校验。

## 9. Project Comprehension 状态

```text
requested -> scanning -> analyzing -> writing_outputs -> completed
```

失败出口：

```text
failed
partial
stale
```

说明：

- `partial` 表示部分目录或文件无法读取，但不影响核心项目画像。
- `stale` 表示远程分支更新后理解结果已过期。

## 10. Epic 状态

```text
created -> refining -> planning -> scheduled -> running -> completed -> released -> archived
```

失败出口：

```text
needs_user_input
replan_required
cancelled
failed
```

## 11. Issue 状态

```text
created
  -> planned
  -> blocked
  -> ready
  -> running
  -> quality_checking
  -> verifying
  -> reviewing
  -> accepted
  -> merged
  -> done
```

失败和返工出口：

```text
needs_rework
failed
cancelled
```

`ready` 条件：

- hard dependencies 已满足。
- 必要 contract 已 accepted。
- 写入范围不冲突。
- Runtime、worktree、预算和权限满足要求。
- 无待处理用户澄清或审批。

## 12. Run 状态

```text
created -> context_assembling -> dispatched -> running -> collecting_outputs -> completed
```

失败出口：

```text
failed
timeout
cancelled
needs_user_input
```

## 13. Quality Report 状态

```text
requested -> running_gates -> completed -> accepted
```

失败出口：

```text
failed
needs_rework
blocked
```

## 14. Subagent 状态

```text
planned -> context_assembled -> dispatched -> running -> output_collected -> validated -> completed -> archived
```

失败和返工出口：

```text
blocked
failed
timeout
needs_rework
needs_user_input
cancelled
superseded
```

## 15. Skill Definition 状态

```text
registered -> candidate -> enabled -> disabled -> deprecated -> archived
```

## 16. Skill Binding 状态

```text
candidate -> enabled -> disabled -> deprecated
```

## 17. Skill Effectiveness 状态

```text
recorded -> aggregated -> applied -> archived
```

失败出口：

```text
invalid
rejected
```

## 18. Runtime Signal 状态

```text
captured -> normalized -> correlated -> classified -> archived
```

失败出口：

```text
invalid
insufficient_context
```

## 19. Bug Candidate 状态

```text
detected -> classifying -> confirmed -> issue_created -> repairing -> repaired -> archived
```

分流和失败出口：

```text
not_bug
needs_evidence
enhancement_candidate
blocked
```

## 20. Repair Attempt 状态

```text
planned -> branch_created -> running -> quality_checking -> reviewing -> accepted -> merged
```

失败出口：

```text
blocked
failed
needs_rework
escalated
cancelled
```

## 21. Improvement Record 状态

```text
candidate -> approved -> applied -> archived
```

失败出口：

```text
rejected
superseded
```

## 22. Runtime Session 状态

```text
created -> active -> idle -> resumable -> closed
```

失败出口：

```text
unhealthy
lost
expired
```

## 23. Memory Record 状态

```text
candidate -> staged -> committed -> indexed -> active
```

维护状态：

```text
stale
merged
compacted
conflict_review_required
archived
```

## 24. Server Resource 状态

```text
registered -> checking -> active -> maintenance_required -> retired
```

失败或风险状态：

```text
unreachable
unhealthy
expired
blocked_for_production
```

## 25. Release 状态

```text
suggested -> planned -> branch_created -> regression_running -> ready_to_publish -> published -> completed
```

失败出口：

```text
blocked
failed
rollback_required
cancelled
```

## 26. Deployment 状态

```text
created -> precheck_running -> deploying -> smoke_testing -> monitoring -> healthy -> completed
```

失败和回滚出口：

```text
precheck_failed
deploy_failed
smoke_failed
monitor_failed
rollback_running
rolled_back
manual_intervention_required
```

## 27. 状态变更记录要求

每次状态变化必须记录：

- object_type。
- object_id。
- previous_status。
- next_status。
- reason。
- triggered_by。
- timestamp。
- trace_id。
- run_id，如适用。
- issue_id，如适用。
- approval_id，如适用。

状态变化记录属于核心日志契约，详见 [日志与审计事件契约](../contracts/logging-audit-event-contract.md)。
