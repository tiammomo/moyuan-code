# Phase 规划与执行记录

本目录只放阶段性规划、issue graph、验收记录和执行收口文档。稳定设计、策略、契约和配置说明仍保留在 `docs/` 其他主线目录中。

## 当前阶段

- [Phase 4 实现 Issue Graph](./phase4-issue-graph.md)：Phase 4 团队协作、审计查询、审批记录、Git 协同和服务器维护的依赖图。
- [Phase 4 实施记录](./phase4-next-development-plan.md)：Phase 4 当前任务、执行范围和验收记录。
- [Phase 3 Release Readiness](./phase3-release-readiness.md)：Phase 3 第一批能力的收口验证、运行入口和剩余风险。
- [Phase 3 实现 Issue Graph](./phase3-issue-graph.md)：Phase 3 配置可执行化、Console 操作流、Provider 探测和发布部署控制的依赖图。
- [Phase 3 实施记录](./phase3-next-development-plan.md)：Phase 3 已完成任务、执行范围和验收记录。
- [Phase 2 实现 Issue Graph](./phase2-issue-graph.md)：Phase 2 多模型、Skills、Native Runtime 和 Subagent 调度深化的依赖图。
- [Phase 2 实施记录](./phase2-next-development-plan.md)：Phase 2 已完成任务、执行范围和验收记录。
- [Phase 2 Release Readiness](./phase2-release-readiness.md)：Phase 2 第一批能力的收口验证、运行入口和剩余风险。
- [Beta 实现 Issue Graph](./beta-issue-graph.md)：Beta 阶段已完成 issue graph、依赖和执行顺序。
- [Beta 实施记录](./beta-next-development-plan.md)：Beta 已完成任务、执行范围和验收记录。
- [Phase 1 Release Readiness](./phase1-release-readiness.md)：Phase 1 本地 CLI MVP 的验收入口。
- [Phase 1 实现 Issue Graph](./phase1-issue-graph.md)：Phase 1 已完成 issue graph 和依赖关系。
- [Phase 1 实施收口记录](./phase1-next-development-plan.md)：Phase 1 执行记录、完成范围和剩余边界。

## 维护规则

1. 每个 Phase 必须有独立 issue graph、执行记录和验收结论。
2. Phase 文档可以描述临时取舍，但稳定结论必须回写到对应主线、策略、契约或配置文档。
3. 已完成 Phase 不再作为待办清单维护，只保留事实、边界和可复现验证命令。
4. 新 Phase 开始前，先在本目录新增阶段入口，再进入代码实现。
