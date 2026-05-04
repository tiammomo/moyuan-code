# ADR-0004 文件化状态与 GORM State Store 分层

状态：accepted
日期：2026-05-03
决策者：architect

## 背景

Moyuan 当前已进入 Phase 1 本地 CLI MVP 实施阶段，需要保留 `.moyuan/` 文件化状态的可审计性，同时开始建立后续 API server 和查询型控制面的数据库基础。单纯文件化状态不利于 Gin API、项目列表、运行状态查询和团队版演进；但过早把所有权威状态迁入数据库，会让 schema、迁移和恢复成本提前复杂化。

## 决策

Phase 1 使用分层持久化：

- `.moyuan/` 文件化状态、JSONL 日志、YAML 配置和 Markdown 报告继续作为可审计权威产物。
- `GORM + SQLite` 作为 State Store 的实现基线，优先承载项目注册、列表查询和后续 API 查询型状态。
- HTTP/API 入口统一使用 `Gin`。
- 数据库实现不能改变上层对象语义、契约和状态机。

## 影响

- Workspace API 和 State Store 必须抽象读写，不允许业务模块散落文件操作。
- 需要原子写、锁、版本、事务 journal 和崩溃恢复。
- Phase 1 默认 SQLite 路径为 `.moyuan/state.db`。
- 后续迁移到 PostgreSQL 或团队版服务端数据库时，保持对象 id、状态机和日志语义不变。

## 替代方案

- 仅使用文件：可审计性强，但 API 查询、分页、过滤和团队版扩展成本较高。
- 直接使用服务端数据库：适合团队版控制面，但不利于本地项目隔离和快速迭代。
- 自研轻量 HTTP wrapper：初期依赖少，但中间件、绑定、测试和生态需要自行维护。

## 相关文档

- [持久化与并发一致性](../persistence-concurrency-consistency.md)
- [项目工作空间规范](../project-workspace-spec.md)
- [实现模块拆分](../implementation-module-map.md)
- [后端技术栈与本地环境](../backend-tech-stack.md)
