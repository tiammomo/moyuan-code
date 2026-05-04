# Phase 1 Release Readiness

状态：completed
责任角色：orchestrator_owner + core_engineer + qa_owner
最后更新：2026-05-04

本文是 Phase 1 本地 CLI MVP 的验收与运行说明。后续实现进入下一阶段时，以本文确认当前能力边界，不再把 Phase 1 视为规划阶段。

## 1. 结论

Phase 1 本地 CLI MVP 已具备可复现闭环：

- 本地路径和远程 Git 仓库接入。
- 项目阅读理解和增量理解。
- Issue Graph、schedule、orchestrator plan/run/status。
- Claude CLI、Codex CLI 和 local shell runtime adapter 基线。
- Runtime diff capture、脏工作区保护和保护路径阻断。
- Quality report、review_status 和结构化 findings。
- Memory record gate、staging、dedup、敏感信息阻断和 compact。
- Self-repair signal、candidate、plan、attempt、quality gate、status 和经验沉淀。
- Gin API router 和 GORM SQLite State Store 基线。

## 2. 本地验证命令

推荐验证：

```bash
go test ./...
go run ./cmd/moyuan --help
./bin/moyuan --help
```

如果本机没有全局 Go，可使用项目当前验证过的 Go 1.22+ 环境后再执行。

## 3. 最短使用路径

```bash
./bin/moyuan project add --local /path/to/repo --root /path/to/repo
./bin/moyuan auth whoami --root /path/to/repo
./bin/moyuan comprehend --full --root /path/to/repo
./bin/moyuan issue graph phase1-epic --root /path/to/repo
./bin/moyuan orchestrator plan phase1-epic --root /path/to/repo
./bin/moyuan runtime invoke local_shell --prompt "printf ok" --root /path/to/repo
./bin/moyuan quality check phase1-001 --root /path/to/repo
./bin/moyuan memory add --summary "项目事实需要在未来任务中复用" --root /path/to/repo
./bin/moyuan memory candidates --root /path/to/repo
./bin/moyuan repair signal --type test_failure --summary "测试失败" --root /path/to/repo
```

`repair signal` 会返回 `repair_plan.id`。低风险 confirmed bug 可以继续执行：

```bash
./bin/moyuan repair run <repair-plan-id> --runtime local_shell --prompt "<repair command>" --root /path/to/repo
./bin/moyuan repair status <repair-attempt-id> --root /path/to/repo
```

## 4. 主要产物

所有项目本地状态都写入被管理项目的 `.moyuan/`：

| 目录/文件 | 内容 |
| --- | --- |
| `.moyuan/project.yaml` | 项目配置 |
| `.moyuan/repository.yaml` | 仓库配置 |
| `.moyuan/state.db` | GORM SQLite 查询型状态 |
| `.moyuan/comprehension/` | 项目画像、模块地图、命令识别和理解事件 |
| `.moyuan/lifecycle/issue-graphs/` | Issue Graph |
| `.moyuan/lifecycle/schedules/` | schedule 和 ready/blocked queue |
| `.moyuan/orchestrator/` | issue/run 状态和结果 |
| `.moyuan/runtime/` | runtime result、diff summary 和 native metadata |
| `.moyuan/lifecycle/quality/reports/` | JSON/Markdown quality report |
| `.moyuan/memory/` | candidates、staging、records 和 compact |
| `.moyuan/repair/` | signals、bug candidates、repair plans 和 attempts |
| `.moyuan/logs/` | run、audit、quality、memory 等 JSONL 日志 |

## 5. Phase 1 边界

以下能力不属于 Phase 1 本地 CLI MVP：

- Web Console。
- team_server。
- 多用户组织级协作。
- GitHub/Gitee/GitLab PR/MR 自动创建。
- 生产服务器部署、线上冒烟和监控。
- 完整向量检索 Memory。
- 完整 Provider Registry、模型成本统计和预算治理。
- 自动 release/deployment 流水线。

这些能力保留在 Beta/Production 阶段继续拆分。

## 6. 剩余风险

- Claude CLI / Codex CLI 当前已具备调用契约和 fake CLI 回归测试，真实复杂任务的 session resume、结构化输出解析和失败恢复仍需后续增强。
- Quality gate 当前覆盖 build/lint/test、protected path、敏感文件、runtime risk 和大 diff，复杂度、重复代码和覆盖率阈值仍需后续细化。
- Repair run 不自动提交、不自动合入；这是 Phase 1 的安全边界，后续需要接入分支策略和 review 合入流程。
- GORM State Store 当前只承载项目注册等基线索引，issue/run/memory 全量索引仍保留在后续阶段。

## 7. 提交状态

Phase 1 关键收口提交：

- `4f570f4 feat: adopt gin gorm backend baseline`
- `ba2e766 feat: add memory record gate`
- `1ae4c57 feat: add controlled repair loop`
