# Moyuan Code

Moyuan Code 是面向代码开发全生命周期的多 Agent 编排框架。当前 `Phase 1` 本地 CLI MVP 和 Beta 第一批控制面能力已完成，进入 Phase 2 多模型、Skills、Native Runtime 和 Subagent 调度深化。

核心设计入口见 [docs/README.md](./docs/README.md)。
Phase 规划与验收记录见 [docs/phases/](./docs/phases/README.md)。

## 当前实现状态

- 控制面后端：Go。
- 后端框架基线：Gin + GORM，Phase 1 本地 State Store 使用 SQLite。
- 图像生成辅助脚本：Node.js，仅保留在 `scripts/`。
- Phase 1 已实现最小 CLI 骨架：workspace、auth、logging、git、project comprehension、issue graph、runtime adapter、orchestrator、scheduler、memory、repair、quality gate。
- Beta 已推进控制面状态 API、issue graph API、需求拆分、并发调度、review/merge decision、Provider Registry/runtime routing、Git Provider push/PR/MR plan、release suggestion、服务器资源 registry、deploy/smoke/monitor plan、Subagent 可审计模型和质量解释视图。
- Phase 1 e2e smoke 已覆盖本地项目和本地 bare remote 模拟远程项目的完整 CLI 链路。
- Runtime 已捕获 before/after git snapshot、changed files、diff summary，并能阻断脏工作区和保护路径变更。
- Claude CLI / Codex CLI 已具备 prompt file、cwd、env allowlist、provider env profile、stdout/stderr、result contract 和失败分类的最小调用契约。
- Orchestrator 已持久化 issue/run 状态机，并支持查询 accepted、needs_rework、runtime 和 quality 状态。
- Quality 已输出结构化 findings 和 review_status，能因敏感文件、保护路径、runtime 风险和大 diff 阻断 accepted。
- API/State Store 已建立 Gin router 和 GORM SQLite 基线，项目注册会同步 `.moyuan/state.db`。
- Memory 已具备 record gate、staging、dedup、敏感信息阻断和 compact 自动摘要。
- Repair 已具备受控 attempt、最大尝试次数、runtime 执行、quality gate、状态查询和修复经验 Memory 沉淀。
- 下一批实现重点：Phase 2 `task-model-strategy-switch`，随后进入 Native Runtime 会话恢复和 Subagent 调度深化。

## 本地运行

```bash
go test ./...
go run ./cmd/moyuan --help
./bin/moyuan --help
cd apps/console && npm run dev
```

如果本机没有全局 Go，可先安装 Go 1.22+ 后再运行以上命令。

Web Console 本地端口为 `127.0.0.1:3000`，Go/Gin API 本地端口为 `127.0.0.1:8080`。

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
./bin/moyuan model provider add --id glm-main --vendor zhipu --api-type openai-compatible --auth-ref env:GLM_API_KEY --root /path/to/repo
./bin/moyuan model route --role backend --repo-edit --root /path/to/repo
./bin/moyuan model provider add --id minimax-m27-claude --vendor minimax --api-type anthropic-compatible --base-url https://api.minimaxi.com/anthropic --auth-ref env:MINIMAX_API_KEY --runtime claude_cli --model MiniMax-M2.7 --use-case frontend --allow-sensitive-code --allow-project-memory --root /path/to/repo
./bin/moyuan runtime invoke claude_cli --provider minimax-m27-claude --prompt "实现前端 issue" --root /path/to/repo
./bin/moyuan git provider plan phase1-001 --root /path/to/repo
./bin/moyuan release suggest --version v0.1.0 --root /path/to/repo
./bin/moyuan resources add --id dev-1 --environment test_dev --host 10.0.0.10 --provider local_vm --owner dev --auth-ref env:DEV_SERVER_SSH_KEY --root /path/to/repo
./bin/moyuan deploy plan <release-id> --environment test_dev --resource dev-1 --root /path/to/repo
```

所有被管理项目的配置、状态、日志、项目理解和质量报告都会写入项目内 `.moyuan/`。
