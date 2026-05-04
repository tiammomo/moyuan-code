# Phase 规划与执行记录

本目录只放阶段性规划、issue graph、验收记录和执行收口文档。稳定设计、策略、契约和配置说明仍保留在 `docs/` 其他主线目录中。

## 当前阶段

- [Beta 实现 Issue Graph](./beta-issue-graph.md)：Beta 阶段 issue graph、依赖和执行顺序。
- [Beta 实施记录](./beta-next-development-plan.md)：Beta 已完成任务、当前任务、执行范围和验收记录。
- [Phase 1 Release Readiness](./phase1-release-readiness.md)：Phase 1 本地 CLI MVP 的验收入口。
- [Phase 1 实现 Issue Graph](./phase1-issue-graph.md)：Phase 1 已完成 issue graph 和依赖关系。
- [Phase 1 实施收口记录](./phase1-next-development-plan.md)：Phase 1 执行记录、完成范围和剩余边界。

## 维护规则

1. 每个 Phase 必须有独立 issue graph、执行记录和验收结论。
2. Phase 文档可以描述临时取舍，但稳定结论必须回写到对应主线、策略、契约或配置文档。
3. 已完成 Phase 不再作为待办清单维护，只保留事实、边界和可复现验证命令。
4. 新 Phase 开始前，先在本目录新增阶段入口，再进入代码实现。
