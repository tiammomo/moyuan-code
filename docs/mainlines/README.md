# 主线文档

主线文档按真实项目生命周期和平台治理流程组织，帮助读者从平台用户与访问控制、被管理项目接入、需求规划、开发、代码管理、服务器资源到发布投产完整走通。

主线不是按模块名划分，而是按专家判定标准划分。满足以下条件越多，越应该成为主线：

- 有明确生命周期阶段，不只是工具或配置项。
- 有独立输入、输出和持久化产物。
- 有会阻断后续流程的关键决策点。
- 有独立责任角色或 owner。
- 会被多个横切能力引用。
- 出错后需要独立失败恢复路径。

主线文档不重复对象字段、完整配置和策略细节，只回答：

- 这条主线负责什么。
- 输入和输出是什么。
- 端到端流程如何走。
- 哪些策略决策树会被调用。
- 需要写入哪些产物和日志。
- 什么时候阻断、返工或升级人工确认。

## 主线列表

| 主线 | 文档 | 目标 |
| --- | --- | --- |
| 平台用户与访问控制 | [platform-user-access.md](./platform-user-access.md) | 管理用户、组织、会话、API Token、角色、审批和审计 |
| 项目接入与阅读理解 | [project-comprehension.md](./project-comprehension.md) | 接入本地/远程仓库，建立项目画像、模块地图和理解快照 |
| 需求规划与 Issue 编排 | [requirement-planning.md](./requirement-planning.md) | 将用户需求完善、澄清、拆分为 Issue Graph 和可执行 schedule |
| 代码开发 | [code-development.md](./code-development.md) | 消费 ready issue，完成多 Agent 实现、测试、质量复核和返工 |
| 代码管理 | [code-management.md](./code-management.md) | 管理任务分支、worktree、integration branch、PR/MR 和用户改动保护 |
| 服务器资源管理 | [server-resource-management.md](./server-resource-management.md) | 统一维护测试开发机、生产机、云资产、到期、巡检和资源组 |
| DevOps 发布投产 | [devops-release-deployment.md](./devops-release-deployment.md) | 管理 release branch、tag、部署、线上冒烟、监控、回滚和复盘 |

## 横切能力

以下能力不是主线，但会被每条主线引用：

- Agent Runtime 和模型 Provider：[模型与工具适配规划](../model-tool-adapters.md)
- Agent role、team 和 skills：[Agent、Skills 与编排](../agent-skills-memory.md)
- Memory：[Agent Memory 系统方案](../agent-memory-system.md)
- 鉴权：[鉴权与访问控制策略](../policies/auth-access-policy.md)
- 权限：[权限模型](../foundations/permission-model.md)
- 失败恢复：[失败恢复设计](../foundations/failure-recovery.md)
- 配置：[完整配置方案](../configuration-guide.md) 和 [配置 Schema 规则](../configuration-schema-spec.md)
- 核心对象：[核心数据对象](../foundations/core-data-objects.md)

## 主线与策略关系

主线描述流程，策略描述决策。

例如需求规划主线会调用：

- [鉴权与访问控制策略](../policies/auth-access-policy.md)
- [Issue 调度策略](../policies/issue-scheduling-policy.md)
- [项目阅读理解策略](../policies/project-comprehension-policy.md)
- [Provider 路由策略](../policies/provider-routing-policy.md)
- [Memory 决策策略](../policies/memory-decision-policy.md)

代码开发主线会调用：

- [鉴权与访问控制策略](../policies/auth-access-policy.md)
- [质量与合入策略](../policies/quality-merge-policy.md)
- [Provider 路由策略](../policies/provider-routing-policy.md)
- [Memory 决策策略](../policies/memory-decision-policy.md)

策略文档必须能转成实现里的规则引擎、状态机或校验器。
