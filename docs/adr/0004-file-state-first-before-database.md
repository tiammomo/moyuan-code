# ADR-0004 文件化状态优先于数据库

状态：accepted
日期：2026-05-03
决策者：architect

## 背景

Moyuan 当前处于规划设计阶段，需要先把 Project、Issue、Run、Subagent、Memory、Release、Deployment 等对象和状态流转稳定下来。过早引入数据库会把 schema、迁移和查询优化提前复杂化。

## 决策

规划和 MVP 阶段优先使用 `.moyuan/` 文件化状态、JSONL 日志、YAML 配置和 Markdown 报告。数据库可以作为后续 State Store 的实现替换，但不能改变上层对象语义、契约和状态机。

## 影响

- Workspace API 和 State Store 必须抽象读写，不允许业务模块散落文件操作。
- 需要原子写、锁、版本、事务 journal 和崩溃恢复。
- 后续迁移到 SQLite 或服务端数据库时，保持对象 id、状态机和日志语义不变。

## 替代方案

- 直接使用 SQLite：事务更强，但早期 schema 变化成本更高。
- 直接使用服务端数据库：适合团队版控制面，但不利于本地项目隔离和快速迭代。

## 相关文档

- [持久化与并发一致性](../persistence-concurrency-consistency.md)
- [项目工作空间规范](../project-workspace-spec.md)
- [实现模块拆分](../implementation-module-map.md)
