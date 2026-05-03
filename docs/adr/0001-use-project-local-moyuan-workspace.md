# ADR-0001 使用项目本地 `.moyuan/` 工作空间

状态：accepted
日期：2026-05-03
决策者：architect

## 背景

Moyuan 需要持续管理多个被管理项目。每个项目都有独立仓库、配置、Agent、Memory、Issue Graph、日志、服务器资源和发布状态。

## 决策

每个被管理项目默认使用项目本地 `.moyuan/` 工作空间保存项目级配置、状态、产物和审计索引。控制面对象可以放在本地身份文件或控制面数据库，但项目生命周期状态必须能随项目独立迁移和审计。

## 影响

- 项目接入后必须初始化 `.moyuan/`。
- 所有配置和生命周期产物必须有 schema 或权威文档。
- `.moyuan/` 需要原子写、锁、事务和迁移能力。
- secret 明文不能进入 `.moyuan/`。

## 替代方案

- 全局数据库统一保存所有项目状态：便于集中查询，但弱化项目隔离和本地可审计性。
- 只使用 GitHub/Gitee issue 和 PR 状态：不适合本地仓库、多 Provider、多 Runtime 和 Memory 管理。

## 相关文档

- [项目工作空间规范](../project-workspace-spec.md)
- [持久化与并发一致性](../persistence-concurrency-consistency.md)
- [Workspace 迁移契约](../contracts/workspace-migration-contract.md)
