# ADR-0002 Native Agent Runtime 边界

状态：accepted
日期：2026-05-03
决策者：architect

## 背景

Moyuan 需要调用 Claude CLI 和 Codex CLI 这类具备读写仓库、执行命令和长任务能力的原生 Agent。它们能力强，但也带来文件写入、命令执行和权限越界风险。

## 决策

Claude CLI 和 Codex CLI 作为 Native Agent Runtime 接入。Moyuan 负责编排上下文、限制 read/write scope、捕获 diff、执行质量门禁和决定合入。Native Runtime 不能直接 push、tag、deploy 或 accepted issue。

默认分工：

- 前端、UI、复杂交互和设计任务可优先 Claude CLI；样式基线稳定后的前端代码修改、测试、修复和重构允许 Codex CLI 参与或主导。
- 后端、测试、review、quality_guard、repair 和后端调优优先 Codex CLI。

## 影响

- Runtime Adapter 必须支持 session、输出契约、diff snapshot、错误分类和审计。
- Subagent Plan 必须声明 scope、role、runtime、skills 和输出契约。
- 所有 Native Runtime 输出必须回到 Quality Gate 和 Reviewer。

## 替代方案

- 只使用普通模型 API：安全边界更简单，但代码开发能力不足。
- 允许 CLI 自主管理分支和发布：实现快，但风险高，不符合质量和审计目标。

## 相关文档

- [模型与工具适配规划](../model-tool-adapters.md)
- [Runtime Adapter 契约](../contracts/runtime-adapter-contract.md)
- [Subagent 与 Skills 系统方案](../subagents-skills-system.md)
- [安全威胁模型](../threat-model.md)
