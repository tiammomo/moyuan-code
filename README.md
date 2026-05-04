# Moyuan Code

Moyuan Code 是面向代码开发全生命周期的多 Agent 编排框架。当前 `Phase 1` 本地 CLI MVP 已完成主要闭环，进入 Beta 阶段控制面 API、issue 编排和生产级能力扩展。

核心设计入口见 [docs/README.md](./docs/README.md)。
Phase 规划与验收记录见 [docs/phases/](./docs/phases/README.md)。

## 当前实现状态

- 控制面后端：Go。
- 后端框架基线：Gin + GORM，Phase 1 本地 State Store 使用 SQLite。
- 图像生成辅助脚本：Node.js，仅保留在 `scripts/`。
- Phase 1 已实现最小 CLI 骨架：workspace、auth、logging、git、project comprehension、issue graph、runtime adapter、orchestrator、scheduler、memory、repair、quality gate。
- Phase 1 e2e smoke 已覆盖本地项目和本地 bare remote 模拟远程项目的完整 CLI 链路。
- Runtime 已捕获 before/after git snapshot、changed files、diff summary，并能阻断脏工作区和保护路径变更。
- Claude CLI / Codex CLI 已具备 prompt file、cwd、env allowlist、stdout/stderr、result contract 和失败分类的最小调用契约。
- Orchestrator 已持久化 issue/run 状态机，并支持查询 accepted、needs_rework、runtime 和 quality 状态。
- Quality 已输出结构化 findings 和 review_status，能因敏感文件、保护路径、runtime 风险和大 diff 阻断 accepted。
- API/State Store 已建立 Gin router 和 GORM SQLite 基线，项目注册会同步 `.moyuan/state.db`。
- Memory 已具备 record gate、staging、dedup、敏感信息阻断和 compact 自动摘要。
- Repair 已具备受控 attempt、最大尝试次数、runtime 执行、quality gate、状态查询和修复经验 Memory 沉淀。
- 下一批实现重点：Beta 阶段控制面状态 API、issue graph API、自动并发编排、review/merge pipeline 和生产级能力扩展。

## 本地运行

```bash
go test ./...
go run ./cmd/moyuan --help
./bin/moyuan --help
```

如果本机没有全局 Go，可先安装 Go 1.22+ 后再运行以上命令。

## Phase 1 示例

```bash
./bin/moyuan project add --local /path/to/repo --root /path/to/repo
./bin/moyuan auth whoami --root /path/to/repo
./bin/moyuan requirement plan --text "add backend API to inspect issue graph with go test verification" --root /path/to/repo
./bin/moyuan issue graph phase1-epic --root /path/to/repo
./bin/moyuan orchestrator plan phase1-epic --root /path/to/repo
./bin/moyuan runtime invoke local_shell --prompt "printf ok" --root /path/to/repo
./bin/moyuan api serve --addr 127.0.0.1:8080 --root /path/to/repo
./bin/moyuan memory add --summary "项目事实" --root /path/to/repo
./bin/moyuan memory candidates --root /path/to/repo
./bin/moyuan repair signal --type test_failure --summary "测试失败" --root /path/to/repo
./bin/moyuan repair run <repair-plan-id> --prompt "修复命令" --root /path/to/repo
./bin/moyuan repair status <repair-attempt-id> --root /path/to/repo
./bin/moyuan quality check phase1-001 --root /path/to/repo
./bin/moyuan review merge-decision phase1-001 --root /path/to/repo
```

所有被管理项目的配置、状态、日志、项目理解和质量报告都会写入项目内 `.moyuan/`。
