# ADR-0005 Go 控制面后端，Python 作为 worker

状态：accepted
日期：2026-05-04
决策者：architect

## 背景

Moyuan 需要同时承担 CLI、API、编排、Git、workspace、质量门禁、发布、审计和模型邻接能力。单纯把所有逻辑塞进一个动态语言运行时，会让控制面、执行侧和模型侧职责混杂，长期维护成本偏高。

## 决策

- 控制面后端采用 `Go`。
- 模型邻接、文本处理和轻量分析采用 `Python`。
- `Go` 负责唯一的权威状态、调度、审批、Git、workspace 和日志。
- `Python` 只作为 worker/helper，不直接拥有写入权。
- 两者之间先用版本化 JSON 协议协作，后续可升级为 `gRPC`。

## 影响

- 项目可以获得更稳定的控制面二进制、清晰的并发模型和更强的系统集成能力。
- Python 侧可以更灵活地接入文本处理、规则化分析和模型邻接工具。
- 本地开发需要同时维护 Go 和 Python 两套环境。
- 需要显式定义 Go/Python 的接口协议、错误类型和版本兼容策略。

## 替代方案

- `TypeScript` 一体化实现：适合快速搭建，但长期控制面和执行侧边界更容易混淆。
- `纯 Python` 实现：模型邻接方便，但控制面和系统集成的稳定性不如 Go。
- `纯 Go` 实现：控制面稳定，但模型邻接和文本处理生态不如 Python 灵活。

## 相关文档

- [后端技术栈与本地环境](../backend-tech-stack.md)
- [实现模块拆分](../implementation-module-map.md)
- [模型与工具适配规划](../model-tool-adapters.md)
- [配置方案](../configuration-guide.md)
