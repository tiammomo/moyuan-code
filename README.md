# Moyuan Code

Moyuan Code 是面向代码开发全生命周期的多 Agent 编排框架。当前进入 `Phase 1` 本地 CLI MVP 实施阶段。

核心设计入口见 [docs/README.md](./docs/README.md)。
下一步开发任务规划见 [docs/phase1-next-development-plan.md](./docs/phase1-next-development-plan.md)。

## 当前实现状态

- 控制面后端：Go。
- 图像生成辅助脚本：Node.js，仅保留在 `scripts/`。
- Phase 1 已实现最小 CLI 骨架：workspace、auth、logging、git、project comprehension、issue graph、runtime adapter、orchestrator、scheduler、memory、repair、quality gate。
- Phase 1 e2e smoke 已覆盖本地项目和本地 bare remote 模拟远程项目的完整 CLI 链路。
- Runtime 已捕获 before/after git snapshot、changed files、diff summary，并能阻断脏工作区和保护路径变更。
- Claude CLI / Codex CLI 已具备 prompt file、cwd、env allowlist、stdout/stderr、result contract 和失败分类的最小调用契约。
- Orchestrator 已持久化 issue/run 状态机，并支持查询 accepted、needs_rework、runtime 和 quality 状态。
- Quality 已输出结构化 findings 和 review_status，能因敏感文件、保护路径、runtime 风险和大 diff 阻断 accepted。
- 下一批实现重点：memory record gate。

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
./bin/moyuan issue graph phase1-epic --root /path/to/repo
./bin/moyuan orchestrator plan phase1-epic --root /path/to/repo
./bin/moyuan runtime invoke local_shell --prompt "printf ok" --root /path/to/repo
./bin/moyuan memory add --summary "项目事实" --root /path/to/repo
./bin/moyuan repair signal --type test_failure --summary "测试失败" --root /path/to/repo
./bin/moyuan quality check phase1-001 --root /path/to/repo
```

所有被管理项目的配置、状态、日志、项目理解和质量报告都会写入项目内 `.moyuan/`。
