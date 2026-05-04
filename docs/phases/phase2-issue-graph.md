# Phase 2 实现 Issue Graph

状态：in_progress
责任角色：orchestrator_owner + adapter_owner + skills_owner + qa_owner
最后更新：2026-05-05

本文记录 Phase 2 的执行图。Phase 2 不重复 Phase 1 和 Beta 已完成的控制面、质量解释、Subagent 可视化和基础 Provider 能力，而是在现有底座上正式推进多模型、Skills、Native Runtime 强化和 Subagent 调度深化。

## 1. Phase 2 目标

- 让模型服务商、CLI Runtime、第三方 API 网关具备统一 registry、健康、额度、用量和路由策略。
- 让 Skills 成为可发现、可绑定、可执行、可复盘、可降权或禁用的工程能力组件。
- 让 Claude CLI、Codex CLI、国产模型 API 和 gpt-image-2 进入同一任务编排、审计和质量门禁闭环。
- 让 Subagent 支持更清晰的调度、重试、归档、输出收敛和效果反馈。

## 2. 已有基线

- `beta-006 provider-registry-runtime-routing` 已完成 Provider Registry 和路由基线。
- `beta-015 subagent-model` 已完成 Subagent Instance 数据模型和审计文件。
- `beta-016 quality-policy-api` 已完成 Quality Explanation API。
- `beta-017 console-quality-subagent-view` 已完成 Issue Inspector 中的 subagent、quality explanation 和 blocking findings 展示。

## 3. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `phase2-001` | `skill-registry-store-api` | completed | Skill Definition、来源、版本、风险、适配 role 和启用状态落盘/API/CLI | Beta | `skills_owner` | 可登记、查询、禁用 skill，配置不保存明文密钥 |
| `phase2-002` | `find-skills-recommendation-adapter` | completed | 接入或封装 `find-skills` 推荐入口，生成推荐理由和候选绑定 | `phase2-001` | `adapter_owner` | 项目/issue/role 可得到 skills 推荐结果并落盘 |
| `phase2-003` | `role-skill-binding-policy` | completed | role、issue、subagent 的 skill binding 策略和冲突规则 | `phase2-001`,`phase2-002` | `orchestrator_owner` | Agent 执行前可解析启用 skills 和禁止项 |
| `phase2-004` | `skill-effectiveness-feedback` | completed | 记录 skill 对质量、返工、耗时和 review 的影响 | `phase2-003` | `qa_owner` | Skill 效果能影响后续推荐、降权和禁用 |
| `phase2-005` | `provider-health-quota-usage` | completed | Provider 健康、额度、用量、成本、第三方标识和数据策略扩展 | `beta-006` | `adapter_owner` | Provider 可展示可用性、额度风险和路由阻断原因 |
| `phase2-006` | `task-model-strategy-switch` | completed | 同一任务按策略切换模型、Runtime 和 fallback | `phase2-005` | `orchestrator_owner` | 同一 issue 可审计地切换模型策略且不绕过质量门禁 |
| `phase2-007` | `native-runtime-session-recovery` | planned | Claude CLI/Codex CLI session resume、失败降级、diff capture 增强 | `beta-015`,`phase2-006` | `runtime_owner` | CLI runtime 失败后可恢复、归档或安全降级 |
| `phase2-008` | `gpt-image-2-diagram-pipeline` | planned | 架构图/流程图/部署拓扑图的 diagram spec、讲解文档和资产索引 | `phase2-005` | `visualization_owner` | 可读取项目理解和 Issue Graph 生成可追踪图像资产 |
| `phase2-009` | `subagent-scheduler-retry-archive` | planned | Subagent 调度、重试、归档、输出收敛和审计增强 | `phase2-003`,`phase2-007` | `orchestrator_owner` | Subagent 可被调度、重试、归档、审计和聚合输出 |

## 4. 推荐执行顺序

1. 先做 `phase2-001`，让 Skills 有稳定数据模型、配置边界和 API。
2. 再做 `phase2-002`、`phase2-003`，把推荐和绑定接入 Agent/Role/Subagent。
3. `phase2-004` 在质量报告和 review 已稳定后接入效果反馈。
4. `phase2-005`、`phase2-006` 扩展 Provider 和模型策略切换。
5. `phase2-007`、`phase2-009` 强化 Native Runtime 和 Subagent 调度。
6. `phase2-008` 可以在 Provider 额度/策略稳定后并行推进。

## 5. 当前执行入口

下一步执行 `phase2-007 native-runtime-session-recovery`。

实现边界：

- 不直接安装第三方 skills 到被管理项目，先做 registry、推荐记录和绑定配置。
- 不让 skill 直接扩大 Subagent 写入范围。
- 不保存 API 明文密钥，只保存 `env:`、`secret:` 或本地引用。
- Skill 推荐和绑定必须进入审计日志，并能被禁用、降权和复盘。
