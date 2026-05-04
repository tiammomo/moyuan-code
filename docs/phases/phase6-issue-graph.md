# Phase 6 实现 Issue Graph

状态：in_progress
责任角色：orchestrator_owner + security_owner + devops_owner + git_owner + provider_owner + frontend_owner + qa_owner
最后更新：2026-05-05

Phase 6 的目标是在 Phase 5 的强制门禁基础上，推进真实外部执行前的可靠性治理。Phase 6 不追求一次性打开全部生产执行，而是把 approval、部署、CI/CD、Provider 观测和 Console 操作拆成可验证的受控闭环。

## 1. Phase 6 目标

- Approval record 支持消费、过期和重放防护，真实外部写操作不能重复使用同一审批。
- Deployment adapter 从本地 dry-run/local shell 推进到 SSH/云厂商 preview 和受控执行准备。
- CI/CD provider 能生成 release workflow、远程 release 和回归状态记录。
- Provider registry 能记录 quota、cost、health 和模型路由反馈。
- Console 从单页工作台向多页面、schema-aware forms 和操作结果追踪演进。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase6-001` | `approval-consumption-replay-guard` | completed | approval record 消费、重放防护和 PR/MR create 接入 | Phase 5 readiness | `security_owner` + `git_owner` | 已消费 approval 不能再次触发真实 PR/MR create |
| `phase6-002` | `deployment-ssh-preview-adapter` | completed | SSH/云厂商 deployment adapter 的 preview/dry-run/execute 状态模型 | `phase6-001` | `devops_owner` | 生产执行仍需 approval，test_dev/staging 可形成可审计 dry-run |
| `phase6-003` | `ci-cd-release-provider-adapter` | planned | GitHub/Gitee release、tag、workflow run 和回归状态同步 | `phase6-001` | `git_owner` + `release_manager` | release 发布只在质量门禁和审批满足时生成远程动作 |
| `phase6-004` | `provider-cost-health-telemetry` | planned | Provider quota/cost/health 采集、预算状态和路由反馈 | Phase 5 readiness | `provider_owner` | Provider 路由能读取健康、额度和成本信号 |
| `phase6-005` | `console-routes-schema-forms` | planned | Console 多页面化、schema-aware forms、操作结果追踪 | `phase6-001` | `frontend_owner` | 高风险表单能展示后端 schema 错误和最新 execution 状态 |

## 3. 建议执行顺序

1. 先做 `phase6-001`，因为真实外部写操作必须先具备审批消费和重放防护。
2. `phase6-002` 和 `phase6-003` 都依赖审批消费，否则部署和 release 远程动作无法安全打开。
3. `phase6-004` 可与 DevOps/Git 工作并行，作为 Provider routing 的质量输入。
4. `phase6-005` 在后端状态对象稳定后推进，避免前端提前固化错误表单。

## 4. 收口规则

- 每个真实外部写操作必须有 preview/dry-run、approval proof、secret resolver、audit event 和失败降级。
- Approval record 一旦被真实写操作消费，不能被同一动作重复使用。
- 生产部署仍默认关闭，直到 SSH/云厂商 adapter 具备回滚、烟测、监控和审批消费闭环。
- Console 只展示和提交后端状态，不自行构造权威执行结果。
