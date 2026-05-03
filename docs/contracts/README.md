# 契约文档

契约文档面向后续实现，定义模块之间必须遵守的接口、输入输出、错误类型、日志事件和验收规则。

契约文档不重复主线流程，不重复策略决策树，也不替代配置 schema。它负责回答：

- 实现代码需要暴露什么接口。
- 输入和输出结构是什么。
- 错误如何分类。
- 状态和日志如何记录。
- 哪些行为必须有测试覆盖。

## 契约列表

| 契约 | 文档 | 作用 |
| --- | --- | --- |
| Schema 校验契约 | [schema-validation-contract.md](./schema-validation-contract.md) | 将配置规则转成机器可校验 schema 和 runtime validator |
| Runtime Adapter 契约 | [runtime-adapter-contract.md](./runtime-adapter-contract.md) | 统一 Claude CLI、Codex CLI 和其他 Agent Runtime 的调用边界 |
| 日志与审计事件契约 | [logging-audit-event-contract.md](./logging-audit-event-contract.md) | 定义核心日志、审计事件、状态变更和 trace 关联 |
| Workspace 迁移契约 | [workspace-migration-contract.md](./workspace-migration-contract.md) | 管理 `.moyuan/` schema_version、迁移、回滚和兼容 |

## 契约优先级

后续实现顺序建议：

1. Schema 校验契约。
2. 日志与审计事件契约。
3. Runtime Adapter 契约。
4. Workspace 迁移契约。

原因：配置校验和日志是所有主线的基础，Runtime Adapter 是代码开发闭环的基础，Workspace 迁移可在 Phase 1 MVP 稳定后逐步完善。
