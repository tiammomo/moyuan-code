# 主线文档

主线文档按真实项目生命周期组织，帮助读者从一个被管理项目的接入、开发、代码管理、服务器资源到发布投产完整走通。

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
| 项目接入与阅读理解 | [project-comprehension.md](./project-comprehension.md) | 接入本地/远程仓库，建立项目画像、模块地图和理解快照 |
| 代码开发 | [code-development.md](./code-development.md) | 从用户需求到 Issue Graph，再到多 Agent 开发、质量复核和返工 |
| 代码管理 | [code-management.md](./code-management.md) | 管理任务分支、worktree、integration branch、PR/MR 和用户改动保护 |
| 服务器资源管理 | [server-resource-management.md](./server-resource-management.md) | 统一维护测试开发机、生产机、云资产、到期、巡检和资源组 |
| DevOps 发布投产 | [devops-release-deployment.md](./devops-release-deployment.md) | 管理 release branch、tag、部署、线上冒烟、监控、回滚和复盘 |

## 横切能力

以下能力不是主线，但会被每条主线引用：

- Agent Runtime 和模型 Provider：[模型与工具适配规划](../model-tool-adapters.md)
- Agent role、team 和 skills：[Agent、Skills 与编排](../agent-skills-memory.md)
- Memory：[Agent Memory 系统方案](../agent-memory-system.md)
- 权限：[权限模型](../foundations/permission-model.md)
- 失败恢复：[失败恢复设计](../foundations/failure-recovery.md)
- 配置：[完整配置方案](../configuration-guide.md) 和 [配置 Schema 规则](../configuration-schema-spec.md)
- 核心对象：[核心数据对象](../foundations/core-data-objects.md)

## 主线与策略关系

主线描述流程，策略描述决策。

例如代码开发主线会调用：

- [Issue 调度策略](../policies/issue-scheduling-policy.md)
- [质量与合入策略](../policies/quality-merge-policy.md)
- [Provider 路由策略](../policies/provider-routing-policy.md)
- [Memory 决策策略](../policies/memory-decision-policy.md)

策略文档必须能转成实现里的规则引擎、状态机或校验器。
