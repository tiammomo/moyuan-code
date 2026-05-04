# Phase 2 Release Readiness

状态：ready_for_next_phase
最后更新：2026-05-05

本文记录 Phase 2 第一批多模型、Skills、Native Runtime、Subagent 调度和 Visual Diagram 能力的收口验证。稳定设计结论已回写到对应主线、策略、契约和配置文档。

## 1. 验证范围

已完成能力：

- Skill Registry、recommendation、binding、effectiveness。
- Provider health、quota、usage、cost snapshot。
- Task model strategy routing。
- Native Runtime recovery archive。
- Subagent retry/archive state 和 scheduler backlog。
- Visual diagram plan、diagram spec、prompt 和 asset index。
- Console Phase 2 observability：runtime recoveries、subagent backlog、visual assets、visual render executions。

不在本次收口内：

- 真实外部 `find-skills` 网络调用。
- 真实云厂商账单/健康检查 API。
- 真实 Claude/Codex session resume 命令。
- 自动切换 fallback runtime 执行代码。
- 真实 `gpt-image-2` 图像 API 执行。
- 真实运行中的 Console 日志流、diff 展开和 visual asset 图片预览。

## 2. 验证命令

后端：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
```

前端：

```bash
cd apps/console
npm run typecheck
npm run build
```

Git：

```bash
git status --short
git log --oneline -8
```

## 3. 验证结论

当前验证结果：

- `go test ./...` 通过。
- `npm run typecheck` 通过。
- `npm run build` 通过，Next.js 16.2.4 production build 成功。
- Phase 2 issue graph 中 `phase2-001` 到 `phase2-009` 均为 `completed`。
- 当前 main 已推送到 GitHub。

## 4. 新增运行入口

Provider：

```bash
moyuan model provider ops <provider-id> --health ok --quota-status ok
moyuan model provider refresh --provider <provider-id>
moyuan model route --strategy low-cost-memory
moyuan model route --strategy image-diagram
```

Workspace：

```bash
moyuan workspace validate
moyuan workspace doctor
```

Runtime recovery：

```bash
moyuan runtime recovery list
moyuan runtime recovery show <recovery-id>
```

Skills：

```bash
moyuan skills recommend --role backend --task-type quality
moyuan skills bind --skill tdd --target-type role --target backend
moyuan skills effectiveness add --skill tdd --issue phase1-001 --outcome helped
```

Visual diagrams：

```bash
moyuan visuals diagram plan --type multi-agent
moyuan visuals assets
moyuan visuals asset show <asset-id>
moyuan visuals asset render <asset-id> --mode dry_run
moyuan visuals renders
```

## 5. 产物位置

- `.moyuan/skills/`
- `.moyuan/models/providers.json`
- `.moyuan/runtimes/recoveries/`
- `.moyuan/runtimes/sessions/`
- `.moyuan/agents/subagents/`
- `.moyuan/scheduler/`
- `.moyuan/visuals/specs/`
- `.moyuan/visuals/prompts/`
- `.moyuan/visuals/assets/`
- `.moyuan/visuals/executions/`
- `.moyuan/state.db`
- `.moyuan/logs/`

## 6. 剩余风险

- Provider refresh 已支持本地可验证信号，不外呼云厂商账单或模型 API。
- Skill recommendation 当前是本地规则 fallback，不是真实外部 marketplace adapter。
- Runtime recovery 只归档上下文和建议 fallback candidate，不自动 resume。
- Subagent retry/archive 已进入调度输入，但还没有生产级队列和 worker。
- Visual render execution 默认不执行真实图像 API；script mode 需要 approval、运行开关和环境密钥。
- Console 已有 Phase 2 可视化入口，并能展示 visual render execution；但还未展开到完整日志、diff 和图片预览。

## 7. 下一批建议

优先级建议：

1. Skills 外部 `find-skills` adapter 接入。
2. Console 深化日志流、diff 展开、visual asset 图片预览和人工审批动作。
3. Visual script mode 接入 auth ref、密钥注入审计和图片结果质量检查。
4. Provider refresh 后续接真实服务商轻量探测 adapter。
5. Workspace validator 后续接 YAML 解析和字段级 schema contract。
