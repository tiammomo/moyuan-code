# ADR 架构决策记录

状态：ready
责任角色：architect
最后更新：2026-05-03

本目录记录 Moyuan Code 的 Architecture Decision Records。ADR 用来说明已经确认或需要评审的关键技术决策，避免实现阶段重复争论或隐式改变架构方向。

## 维护边界

ADR 只记录“为什么这样选”。它不替代：

- [参考架构](../reference-architecture.md)
- [实现模块拆分](../implementation-module-map.md)
- [配置 Schema 规则](../configuration-schema-spec.md)
- [模型与工具适配规划](../model-tool-adapters.md)

## 状态

| 状态 | 含义 |
| --- | --- |
| `proposed` | 已提出，待评审 |
| `accepted` | 已接受，后续实现应遵守 |
| `superseded` | 已被新 ADR 替代 |
| `rejected` | 已拒绝，保留原因 |

## 模板

```md
# ADR-000X 标题

状态：accepted
日期：YYYY-MM-DD
决策者：architect

## 背景

## 决策

## 影响

## 替代方案

## 相关文档
```

## 当前 ADR

| ADR | 状态 | 决策 |
| --- | --- | --- |
| [ADR-0001](./0001-use-project-local-moyuan-workspace.md) | accepted | 每个被管理项目使用项目本地 `.moyuan/` 工作空间 |
| [ADR-0002](./0002-native-agent-runtime-boundary.md) | accepted | Claude CLI 和 Codex CLI 作为 Native Agent Runtime 接入，输出必须回到 Moyuan 门禁 |
| [ADR-0003](./0003-default-2k-image-generation.md) | accepted | 架构图默认生成横版 2K，4K 仅作为显式实验或后处理 |
| [ADR-0004](./0004-file-state-first-before-database.md) | accepted | 规划阶段优先文件化状态和 schema，数据库作为后续可替换实现 |
