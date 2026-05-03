# 总体规划与生命周期路线图

## 1. 产品定位

`moyuan-code` 是面向代码开发全生命周期的多 Agent 编排框架。它不是单一聊天机器人，也不是单纯 CLI 包装器，而是项目级智能研发工作台。

核心目标：

- 调用 Claude Code、Codex 和多种国产大模型 API。
- 以项目为单位隔离配置、任务、memory、skills、模型策略和审计记录。
- 支持本地路径和远程 Git 仓库接入。
- 每次项目接入和远程分支同步后自动执行项目阅读理解。
- 通过多 Agent 分工完成需求、设计、开发、质量门禁、测试、review、发布和复盘。
- 让可靠结论进入可治理的 Agent Memory 系统，支撑长期迭代。

## 2. 核心能力

| 能力 | 说明 | 权威文档 |
| --- | --- | --- |
| 多 Agent 编排 | role、team、handoff、输出契约 | [Agent、Skills 与编排](./agent-skills-memory.md) |
| Issues 编排 | 自动拆分 issues、依赖图、并发调度、ready queue | [Issues 编排与并发调度](./issue-orchestration.md) |
| 仓库接入与理解 | 本地/远程仓库、Git 分支、项目阅读理解 | [仓库接入、Git 与项目理解](./repository-onboarding-git-management.md) |
| 项目工作空间 | `.moyuan/` schema 索引 | [项目工作空间规范](./project-workspace-spec.md) |
| 质量门禁 | 测试、重复度、复杂度、review、返工 | [代码生命周期质量门禁](./code-lifecycle-quality-gates.md) |
| Agent Memory | record gate、抽取、暂存、异步写入、检索、维护 | [Agent Memory 系统方案](./agent-memory-system.md) |
| 模型和工具适配 | Claude Code、Codex、国产模型、Shell/Git/Test/MCP | [模型与工具适配规划](./model-tool-adapters.md) |
| 架构 | Orchestrator、Agent Runtime、Memory Engine、Adapter Layer | [参考架构](./reference-architecture.md) |

## 3. 关键抽象

- Project：被管理的软件项目，对应一个仓库或多仓集合。
- Workspace：项目内 `.moyuan/` 控制区，保存配置、状态、记忆和产物。
- Agent：角色、工具权限、memory 范围、skills、模型策略和输出契约的组合。
- Epic：用户提出的原始开发目标，会被拆成多个 issues。
- Issue：最小可执行开发单元，具备依赖、写入范围、验收标准和测试计划。
- Issue Graph：issues 之间的依赖 DAG。
- Task：一次可追踪工作单元，可以对应一个 issue 或 issue 内的一次执行。
- Run：Task 的一次执行实例，包含 Agent、模型、工具、Git、质量、测试和 memory 记录。
- Adapter：外部能力封装层，例如 Codex、Claude Code、国产模型、Git、Shell、MCP。
- Skill：可复用任务能力包。
- Memory：跨任务保留的结构化上下文。

## 4. 端到端流程

```text
添加项目
  -> 绑定/克隆仓库
  -> 初始化 .moyuan 工作空间
  -> Full Project Comprehension
  -> 生成项目画像 / 模块地图 / memory candidates
  -> 推荐 Agent roles / skills / 模型策略
  -> 创建 Epic
  -> 丰富任务需求 / 判断是否需要意图澄清
  -> 自动拆分 Issues
  -> 构建 Issue Graph
  -> 用户可见 Issue Graph / Schedule
  -> 计算 ready queue 和并发度
  -> 同步 base branch
  -> Incremental Project Comprehension
  -> 创建任务分支
  -> 多 Agent 协作开发
  -> 质量门禁
  -> 测试验证
  -> 独立 Review
  -> 提交 / Push / PR / 发布
  -> Memory 沉淀和复盘
```

## 5. 项目生命周期

```text
DISCOVERY
  -> PLANNING
  -> DESIGN
  -> IMPLEMENTATION
  -> QUALITY_CHECK
  -> VERIFICATION
  -> REVIEW
  -> RELEASE
  -> OPERATION
  -> RETROSPECTIVE
  -> NEXT_ITERATION
```

### DISCOVERY

目标：理解项目。

动作：扫描目录结构、执行 full project comprehension、识别技术栈、识别构建/测试/lint 命令、生成项目画像、模块地图和 memory candidates。

产物：`.moyuan/project.yaml`、`.moyuan/comprehension/project-profile.md`、`.moyuan/comprehension/module-map.md`、skills 推荐结果、初始 Agent team 推荐。

### PLANNING

目标：把需求变成可执行任务。

动作：丰富任务需求、判断是否需要意图澄清、生成验收标准、拆解 issues、评估依赖、构建用户可见 issue graph、选择 Agent team、计算 ready queue。

产物：requirement、epic、issues、issue graph、task plan、approval request。

### DESIGN

目标：形成可审查技术方案。

动作：分析现有代码、设计接口和数据结构、定义模块边界、判断迁移需求、定义测试策略。

产物：design 文档、ADR 候选、写入范围。

### IMPLEMENTATION

目标：执行代码修改。

动作：分配 Agent、控制写入范围、执行代码变更、记录 diff、完成 Agent handoff。

产物：代码变更、run 日志、memory candidates。

### QUALITY_CHECK

目标：阻止不可用、重复、复杂、过度抽象和破坏架构边界的 AI 代码进入完成状态。

动作：检查 diff、测试缺口、重复代码、复杂度、架构边界、依赖和安全风险。

产物：quality report、quality gate 结构化结果、返工建议。

### VERIFICATION

目标：证明变更有效。

动作：运行 test、lint、build、typecheck、benchmark 或回归脚本。

产物：测试报告、验证结论、修复任务。

### REVIEW

目标：降低回归和维护风险。

动作：独立审查 diff 和 quality report，检查测试缺口、安全风险、项目决策一致性，并给出 `accepted`、`needs_rework` 或 `rejected`。

产物：review findings、风险清单、合并建议。

### RELEASE

目标：准备发布。

动作：基于 integration branch 累计 issues 和风险给出 release/deploy 建议，创建 release branch，运行完整回归，生成 release notes，检查迁移和兼容性，完成审批，推送到 GitHub/Gitee，按策略创建 tag 或 PR/MR，结合服务器配置自动部署投产，执行线上冒烟、监控窗口和必要回滚。

产物：release suggestion、release branch、release note、tag、PR/MR、deployment record、smoke test result、monitor report、rollback plan、approval record、release memory。

### OPERATION

目标：跟踪线上或使用反馈。

动作：接收日志、指标和用户反馈，关联到任务和版本，生成 bugfix 或 tuning task。

产物：issue/task、operation memory。

### RETROSPECTIVE

目标：沉淀经验。

动作：总结迭代成果、识别重复问题、更新 project memory、更新 skills 推荐权重。

产物：retrospective、lessons memory、下一轮建议。

## 6. CLI 路线

所有 CLI/MVP 命令以本节为唯一权威来源。专题文档只描述模块能力，不重复路线图。

### MVP 命令

```text
moyuan project add --local <path>
moyuan project add --remote <git-url>
moyuan project list
moyuan init <project>
moyuan inspect
moyuan comprehend
moyuan comprehend --full
moyuan comprehend --since <commit>
moyuan comprehend status
moyuan status
moyuan epic create
moyuan epic plan <epic-id>
moyuan issue graph <epic-id>
moyuan issue schedule <epic-id>
moyuan issue list
moyuan task create
moyuan task list
moyuan run <task-id>
moyuan git status
moyuan git branch list
moyuan quality check <task-id>
moyuan quality report <run-id>
moyuan logs tail
moyuan logs query --run <run-id>
moyuan memory add
moyuan memory search
moyuan skills recommend
moyuan report <run-id>
```

### Beta 命令

```text
moyuan agent list
moyuan agent enable <role>
moyuan model provider add
moyuan model provider list
moyuan model provider disable <provider>
moyuan model list
moyuan model test <provider>
moyuan model health check
moyuan model usage report
moyuan visuals architecture generate
moyuan visuals architecture explain <diagram-id>
moyuan visuals architecture edit <diagram-id>
moyuan runtime list
moyuan runtime health check
moyuan runtime session list
moyuan runtime session resume <session-id>
moyuan lifecycle next
moyuan issue run-ready <epic-id>
moyuan issue replan <epic-id>
moyuan review <task-id>
moyuan review <task-id> --quality-gate
moyuan release prepare
moyuan release suggest
moyuan release publish <release-id>
moyuan deploy run <release-id> --env <env>
moyuan deploy status <release-id>
moyuan deploy rollback <release-id>
moyuan resources add
moyuan resources list
moyuan resources show <host-id>
moyuan resources check --group <group-id>
moyuan resources expiration scan
moyuan resources renewal record <host-id>
moyuan resources retire <host-id>
moyuan git branch create <task-id>
moyuan git sync
moyuan git sync --comprehend
moyuan git commit <task-id>
moyuan git push <task-id>
moyuan memory record
moyuan memory retrieve
moyuan memory candidates
moyuan memory approve <candidate-id>
moyuan memory reject <candidate-id>
moyuan workspace doctor
```

### Production 命令

```text
moyuan server start
moyuan api-token create
moyuan team sync
moyuan policy audit
moyuan logs export
moyuan logs audit
moyuan memory curate
moyuan memory audit
moyuan skills evaluate
moyuan ci run
moyuan repo pr create <task-id>
```

## 7. 落地阶段

### Phase 0：规划与规格

目标：

- 明确核心抽象。
- 明确 `.moyuan/` schema。
- 明确仓库接入、项目理解、质量门禁和 memory 策略。
- 明确首批 adapter。

验收：

- 文档可直接拆分实现 issue。
- 每个能力只有一个权威文档展开细节。
- [设计就绪门禁](./design-readiness-checklist.md) 结论必须为 `READY` 或明确记录 `READY_WITH_RISKS` 的设计债务。
- 权限、失败恢复、核心数据对象和文档维护规则必须完成并接入 README。

### Phase 1：本地 CLI MVP

目标：

- 支持本地路径接入。
- 支持远程 Git URL clone。
- 初始化项目工作空间。
- 添加项目后自动执行 full project comprehension。
- 每次拉取远程分支后自动执行 incremental comprehension。
- 识别 remote、default branch、当前 branch 和 dirty worktree。
- 为任务自动创建独立 Git 分支。
- 支持从用户目标自动生成 epic、issues 和 issue graph。
- 支持需求丰富和意图澄清判断。
- 支持用户可见 issue graph 和 schedule。
- 支持基于 issue 依赖和写入范围计算 ready queue。
- 创建和执行任务。
- 每次代码生成后自动执行质量门禁。
- 支持 Reviewer Agent 独立审核 diff 和 quality report。
- 保存 run 记录。
- 支持基础 memory。
- 支持 Claude Code/Codex 命令适配的最小闭环。

验收：

- 一个真实本地项目和一个远程 Git 项目可跑通“接入 -> 理解 -> 计划 -> 修改 -> 质量 -> 测试 -> review -> 报告”。
- 一个复杂开发目标可自动拆分为多个 issues，并能按依赖关系串行/并发执行。
- 每个 issue 复核通过后才可合入 epic integration branch。
- 能基于累计 accepted issues、风险和变更范围给出 release/deploy 建议。
- 发布流程包含 release branch、回归、release notes、审批、tag、push 到 GitHub/Gitee、PR/MR、服务器部署、线上冒烟、生产监控和回滚。
- 所有配置和产物都保存在 `.moyuan/`。
- AI 生成代码未通过测试、审查、复杂度、重复度或测试缺口检查时不能进入完成状态。

### Phase 2：多模型与 Skills

目标：

- 实现 provider registry。
- 接入至少 2 个国产模型 API。
- 支持 GLM、MiniMax、GPT、Claude 和第三方 API 网关登记、检测、禁用和用量记录。
- 支持 gpt-image-2 生成项目架构图、流程图、部署拓扑图和配套讲解。
- 支持 Claude CLI 和 Codex CLI 作为 Native Agent Runtime 调用、会话恢复、diff 捕获和失败降级。
- 实现模型路由策略。
- 实现 `find-skills` 推荐入口。
- 实现 role-skill 动态绑定。

验收：

- 同一任务可切换模型策略。
- 模型服务商能统一登记账号、模型、额度、健康状态、第三方标识和数据策略。
- 架构可视化能读取项目理解和 Issue Graph，生成图片、diagram spec 和讲解文档。
- Claude CLI 和 Codex CLI 能在 issue worktree 内执行，并把改动交回质量门禁。
- skills 推荐结果可落盘。
- Agent 执行时能引用启用 skills。

### Phase 3：Memory 强化

目标：

- 实现 Record Gate。
- 实现轻量抽取和 memory candidates。
- 实现暂存去重和异步写入。
- 实现关键词 + 向量混合检索。
- 实现 memory approval 和 curator。

验收：

- 新任务可以检索历史决策、项目事实和 lessons。
- 项目理解结果可以转为 memory candidates。
- 过时和冲突 memory 能被标记。
- 敏感信息不会进入长期 memory。

### Phase 4：团队协作与审计

目标：

- 增加 API server。
- 支持多人共享配置。
- 支持审计日志。
- 支持统一核心日志查询、导出和脱敏。
- 支持权限策略。
- 支持 CI/CD 集成。
- 支持 GitHub/Gitee/GitLab PR/MR 创建与状态同步。
- 支持服务器资源清单、资源组、基础配置、云资源到期时间、连通性检查、续费维护和环境引用。

验收：

- 团队能复用 roles、models、skills 和 memory。
- 高风险操作有审计和确认。
- run、agent、model、Git、质量、发布部署和错误日志可按 trace/run/issue 查询。
- 测试开发机和生产机能被统一登记、查看、巡检、续费提醒、退役，并被 deploy pipeline 引用。

### Phase 5：Web Console 与企业化

目标：

- 提供 Web Console。
- 支持多项目看板。
- 支持组织级 policies。
- 支持模型成本统计。
- 支持私有化运行。

验收：

- 多项目生命周期状态可视化。
- 可查看 task/run/review/release 全链路。
- 支持组织统一模型和权限治理。

## 8. 技术选型建议

语言优先建议 TypeScript。原因是 CLI、API server、配置生态、Node 工具链、Claude Code SDK、OpenAI SDK 和 Web Console 集成都更直接。后续如强调单二进制分发和本地性能，可用 Rust 重写 CLI 或执行沙箱组件。

存储路线：

- MVP：YAML + JSONL + SQLite。
- Beta：SQLite + 本地向量索引。
- Production：PostgreSQL + pgvector + 对象存储。

Adapter 路线：

- MVP：Shell、Git、GitHub/Gitee remote onboarding、Claude Code CLI、Codex CLI/API、OpenAI-compatible Model。
- Beta：DashScope、DeepSeek、Zhipu、MCP。
- Production：GitHub/Gitee/GitLab PR/MR、CI/CD、Observability、Enterprise SSO/Policy。

## 9. 近期任务拆分

1. 定义 `.moyuan/` schema。
2. 实现 `moyuan project add --local`。
3. 实现 `moyuan project add --remote` 和远程 clone。
4. 实现 `moyuan init`。
5. 实现项目 inspect。
6. 实现 `moyuan comprehend --full`。
7. 实现 `moyuan comprehend --since <commit>`。
8. 实现 Git 状态识别和任务分支创建。
9. 实现拉取远程分支后的增量阅读理解。
10. 实现 task/run 数据模型。
11. 实现 Claude Code Adapter 最小命令调用。
12. 实现 Codex Adapter 最小命令调用。
13. 实现 OpenAI-compatible Adapter。
14. 实现 Agent role 配置加载。
15. 实现 memory add/search。
16. 实现 skills recommend 的接口占位和结果落盘。
17. 实现 run report。
18. 用一个真实本地项目和一个远程 Git 项目跑通端到端流程。
