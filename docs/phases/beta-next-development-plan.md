# Beta 实施记录

状态：in_progress
责任角色：orchestrator_owner + backend_owner + qa_owner
最后更新：2026-05-05

本文记录 Beta 阶段从规划到执行的实际顺序。稳定设计结论需要回写到对应主线、策略、契约或配置文档；本文件只记录阶段执行事实。

## 1. 当前基线

Phase 1 本地 CLI MVP 已完成，验收入口见 [Phase 1 Release Readiness](./phase1-release-readiness.md)。

当前可复用能力：

- `.moyuan/` 项目工作空间、项目接入、阅读理解和 Git 绑定。
- Issue graph、schedule、orchestrator issue/run 状态机。
- Runtime adapter、Claude CLI/Codex CLI 调用契约和 local shell fallback。
- Quality review gate、Memory record gate、repair controlled loop。
- Gin + GORM 基线，项目注册会同步 `.moyuan/state.db`。
- 部署计划后已具备受控 execution 记录、dry-run 预演和受限 local shell 执行框架。

## 2. Beta 第一批任务

| 优先级 | ID | 任务 | 状态 | 目标 | 退出条件 |
| --- | --- | --- | --- | --- | --- |
| P0 | `beta-001` | `state-query-api` | completed | 控制面 API 可查询项目核心状态 | API + 测试 + 文档同步 |
| P0 | `beta-002` | `issue-graph-api` | completed | API 可展示 issue graph、schedule 和队列 | issue graph 可被前端可视化读取 |
| P0 | `beta-003` | `requirement-to-issues` | completed | 需求丰富、澄清判断和 issue graph 生成 | 用户需求可转为 issues DAG |
| P1 | `beta-004` | `parallel-orchestration-engine` | completed | 自动并发、等待和 replan | 并发度由系统决策且可审计 |
| P1 | `beta-005` | `review-merge-pipeline` | completed | 复核通过后合入任务分支 | review gate 阻断未达标代码 |
| P1 | `beta-006` | `provider-registry-runtime-routing` | completed | Provider 和 Runtime 路由基线 | Provider 可配置、校验、路由和审计 |
| P1 | `beta-007` | `git-provider-pr-mr` | completed | GitHub/Gitee 分支、push、PR/MR 编排 | 任务分支可推送并形成 PR/MR 计划 |
| P1 | `beta-008` | `release-branch-pipeline` | completed | 版本分支、tag 和 GitHub/Gitee 发布记录 | 可根据积累量生成 release plan |
| P1 | `beta-009` | `server-resource-registry` | completed | 测试机/生产机资源纳管 | 可登记、查询、审计服务器资源 |
| P1 | `beta-010` | `devops-deploy-smoke-monitor` | completed | 部署、线上冒烟和生产监控计划 | 可生成受控部署计划 |
| P2 | `beta-011` | `controlled-deploy-executor` | completed | 受控部署执行器基线 | dry-run/local_shell 执行可审计，生产真实执行被阻断 |
| P1 | `beta-012` | `console-api-integration` | completed | Web Console 接入更多真实 API | 控制台可展示 live requirement、deploy execution、resource health |
| P1 | `beta-013` | `subagent-run-visibility` | completed | Subagent/run 过程可视化 | 运行队列、等待原因、review/quality 结果可追踪 |
| P2 | `beta-014` | `server-health-check-executor` | completed | 服务器健康检查执行器 | test_dev/staging 资源可执行受控 health check 并回写资源状态 |
| P1 | `beta-015` | `subagent-model` | completed | 显式 Subagent Instance 模型 | 每个 run 都有 role/runtime/scope/skills/memory 的可审计 subagent |
| P1 | `beta-016` | `quality-policy-api` | completed | 质量门禁策略和 findings 可解释 API | 控制台可查看 accepted/blocked/needs_rework 的证据 |
| P1 | `beta-017` | `console-quality-subagent-view` | completed | 控制台展示 Subagent 和质量解释 | Issue Inspector 可看到 subagent、quality explanation 和 rework reason |

## 3. 已完成任务：`beta-001 state-query-api`

范围：

- `GET /v1/projects`
- `GET /v1/projects/:project_id`
- `GET /v1/projects/:project_id/issues/:issue_id`
- `GET /v1/projects/:project_id/runs/:run_id`
- `GET /v1/projects/:project_id/quality/:report_id`
- `GET /v1/projects/:project_id/memory/search?q=&limit=`
- `GET /v1/projects/:project_id/memory/candidates?limit=`
- `GET /v1/projects/:project_id/repair/attempts/:attempt_id`

非目标：

- 不做写操作 API。
- 不做 Web Console。
- 不做自动 push、merge、deploy。
- 不改变 `.moyuan/` 文件状态作为当前事实来源的原则。

验收：

- 缺失项目和缺失状态返回 404。
- 查询接口使用 Gin router 测试覆盖。
- GORM Store 支持按 project id 查询。
- `go test ./...` 通过。

完成记录：

- 已新增 project、issue state、run state、quality report、memory search、memory candidates 和 repair attempt 只读 API。
- 已新增 Store `FindProject` 查询能力。
- 已覆盖 GORM Store、controlplane fallback、状态读取和 404 行为。
- 验证命令：`PATH=/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH go test ./...`。

## 4. 已完成任务：`beta-002 issue-graph-api`

范围：

- `GET /v1/projects/:project_id/epics/:epic_id/issue-graph`
- `GET /v1/projects/:project_id/epics/:epic_id/schedule`
- 统一返回 ready queue、blocked queue、running/review 占位队列和 blocked reason。

非目标：

- 不生成新 issue graph。
- 不执行调度。
- 不修改 issue 状态。

验收：

- 已有 Phase 1 issue graph 可通过 API 读取。
- 缺失项目返回 404。
- 缺失 epic 返回 404。
- `go test ./...` 通过。

完成记录：

- 已新增 issue graph 和 schedule 只读 API。
- schedule view 包含 ready queue、blocked queue、running queue、review queue、blocked reason 和当前并发度。
- 读取 API 不调用会写入状态的 scheduler build，避免 GET 请求改变项目状态。
- 验证命令：`PATH=/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH go test ./...`。

## 5. 已完成任务：`beta-003 requirement-to-issues`

范围：

- 新增 requirement planner 最小模块。
- 支持把用户任务描述整理为 requirement、clarification decision、acceptance criteria、test plan 和 issue graph draft。
- 提供 CLI/API 入口，先支持启发式拆分，不调用外部模型。

非目标：

- 不执行 issue。
- 不自动并发调度。
- 不创建远程 GitHub/Gitee issue。

验收：

- 用户输入一段任务描述后，可生成稳定 epic 和 issues。
- 任务描述过短或缺少目标时，返回 clarification required。
- 生成的 issue graph 可被 `beta-002` 的 API 读取。
- `go test ./...` 通过。

完成记录：

- 已新增 `internal/requirement` deterministic planner。
- 已支持 CLI：`moyuan requirement plan --text <text>`。
- 已支持 API：`POST /v1/projects/:project_id/requirements/plan` 和 `GET /v1/projects/:project_id/requirements/:requirement_id`。
- planner 会落盘 requirement plan、issue graph 和 schedule。
- 验证命令：`PATH=/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH go test ./...`。

## 6. 已完成任务：`beta-004 parallel-orchestration-engine`

范围：

- 扩展 schedule 计算，基于 ready queue、write scope、role/runtime 和风险控制并发度。
- 给每个 ready issue 输出 dispatch decision。
- 支持 blocked reason 更细分：dependency、write_scope_conflict、runtime_slot、approval_required。

非目标：

- 不真正并发执行 issue。
- 不创建多 worktree。
- 不自动合入。

验收：

- 同一写入范围的 ready issue 不会同时进入 dispatch。
- 不同写入范围的 ready issue 可被排入同一批。
- 输出可审计的 parallelism 和 waiting reason。
- `go test ./...` 通过。

完成记录：

- Scheduler plan 已新增 `dispatch_queue`、`waiting_queue`、`max_parallel`、`runtime_slots`。
- 同一写入范围冲突会进入 waiting，并给出 `write_scope_conflict`。
- 并发预算不足会进入 waiting，并给出 `runtime_slot`。
- API schedule 读取已返回 scheduler plan，包含 dispatch 决策。
- 验证命令：`PATH=/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH go test ./...`。

## 7. 已完成任务：`beta-005 review-merge-pipeline`

范围：

- 复用现有 quality report 和 review_status，定义 issue 完成后的 merge gate 结果。
- 生成 merge decision：ready_to_merge、needs_rework、blocked。
- 为后续 GitHub/Gitee PR/MR 提供只读决策依据。

非目标：

- 不执行 git merge。
- 不 push、不创建 PR/MR。
- 不修改生产分支。

验收：

- accepted issue + accepted quality report 可得到 ready_to_merge。
- rejected quality report 必须得到 needs_rework。
- 缺失质量报告或 issue 未 accepted 时必须 blocked。
- `go test ./...` 通过。

完成记录：

- 已新增 `internal/review` merge decision 模块。
- 已支持 CLI：`moyuan review merge-decision <issue-id>`。
- 已支持 API：`POST /v1/projects/:project_id/issues/:issue_id/merge-decision`。
- merge decision 会写入 `.moyuan/lifecycle/reviews/merge-decisions/` 和 JSONL 记录。
- 当前仍不执行 git merge、push 或 PR/MR 创建。
- 验证命令：`PATH=/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH go test ./...`。

## 8. 已完成任务：`beta-006 provider-registry-runtime-routing`

范围：

- 新增 Provider Registry 最小读写模型。
- 支持配置 GPT、Claude、GLM、MiniMax、第三方 API 和 CLI runtime 的 metadata。
- 支持不泄露 secret 的 provider list/show/route。
- 支持绑定 Native Runtime 的 provider env profile，例如 `minimax-m27-claude` -> `claude_cli` -> `MiniMax-M2.7`。

非目标：

- 不真实调用外部模型 API。
- 不保存明文 API key。
- 不让 provider 绕过 Native Runtime 的 diff、quality gate、review 和 protected path 控制。

验收：

- Provider 可添加、列出、禁用。
- Provider 配置中的 secret 只能以 env/secret ref 出现。
- 默认角色可路由到 Claude CLI、Codex CLI 或 Provider。
- `go test ./...` 通过。

完成记录：

- 已新增 `internal/providers` Provider Registry 和 Route Decision。
- 已支持 CLI：`moyuan model provider add/list/show/disable`、`moyuan model route`。
- 已支持 API：`GET/POST /v1/projects/:project_id/providers`、`GET /providers/:provider_id`、`POST /providers/:provider_id/disable`、`POST /provider-route`。
- Registry 当前写入 `.moyuan/models/providers.json`；只保存 `env:` 或 `secret:` auth ref，不保存明文 key。
- Runtime 调用可通过 `--provider` 显式选择 provider，并只把 `auth_ref` 解析成子进程环境变量；native metadata 只记录 `env_keys`。
- Orchestrator 在 `--role frontend --runtime claude_cli` 且未显式传 provider 时，会基于 Provider Route 选择匹配 provider。
- Scheduler 的默认角色 runtime 已收敛到 Provider 路由默认规则。
- 验证命令：`PATH=/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH go test ./...`。

## 9. 已完成任务：`beta-007 git-provider-pr-mr`

范围：

- 新增 Git Provider 最小能力声明：`github`、`gitee`、`generic_git`。
- 基于 review merge decision 生成 push/PR/MR plan。
- 支持任务分支推送前检查：clean worktree、remote 存在、review allowed、禁止未审核代码。
- 支持只创建本地可审计计划，真实 push/PR/MR 作为受控动作。

非目标：

- 不自动合入 main。
- 不自动发布 release。
- 不绕过 review/quality gate。

验收：

- 缺失 remote 时返回 blocked reason；缺少 PR/MR API auth 时降级为 manual create mode。
- review 未通过时不允许 push/PR/MR。
- GitHub/Gitee/generic git 能生成差异化 provider plan。
- `go test ./...` 通过。

完成记录：

- 已新增 `internal/gitprovider` Git Provider plan 模块。
- 已支持 CLI：`moyuan git provider plan <issue-id>`、`moyuan git provider show <plan-id>`。
- 已支持 API：`POST /v1/projects/:project_id/issues/:issue_id/git-provider-plan`、`GET /v1/projects/:project_id/git-provider-plans/:plan_id`。
- Plan 写入 `.moyuan/lifecycle/pull-requests/`，并记录 `git_provider.plan.created` 日志。
- 当前只生成计划，不执行 push、PR/MR 创建或 merge。
- 验证命令：`PATH=/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH go test ./...`。

## 10. 已完成任务：`beta-008 release-branch-pipeline`

范围：

- 基于已通过 review/merge gate 的 issue 生成 release suggestion。
- 管理 release branch、tag suggestion、release notes 和 Git provider publish plan。
- 只推到 GitHub/Gitee/GitLab/generic git 的远程记录层，不做服务器部署。

非目标：

- 不直接部署到服务器。
- 不绕过 beta-007 的 push/PR/MR plan。
- 不自动 tag 或发布正式 release，先输出可审计计划。

验收：

- 可根据 accepted issue 数量和风险给出是否发版建议。
- 可生成 release branch plan 和 release notes draft。
- 缺失 remote 或存在未合入 issue 时阻断。
- `go test ./...` 通过。

完成记录：

- 已新增 `internal/release` release suggestion 模块。
- 已支持 CLI：`moyuan release suggest [--version <version>] [--min-issues <n>]`、`moyuan release show <release-id>`。
- 已支持 API：`POST /v1/projects/:project_id/releases/suggest`、`GET /v1/projects/:project_id/releases/:release_id`。
- Release plan 和 release notes 写入 `.moyuan/lifecycle/releases/`，并记录 `release.plan.created` 日志。
- 当前只生成 release branch/tag/push 建议，不执行真实发布动作。
- 验证命令：`PATH=/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH go test ./...`。

## 11. 已完成任务：`beta-009 server-resource-registry`

范围：

- 新增服务器资源 registry，区分 `test_dev`、`staging`、`production`。
- 管理 host、provider、region、规格、到期时间、owner、用途、健康检查和维护记录。
- 提供 CLI/API 查询和登记入口，为后续部署、冒烟和监控做基础。

非目标：

- 不执行 SSH 连接。
- 不部署应用。
- 不修改云服务商资源。

验收：

- 可添加、列出、查看和禁用服务器资源。
- 到期时间、环境、资源规格和 owner 可查询。
- 生产资源必须显式标记 environment，不能默认为生产。
- `go test ./...` 通过。

完成记录：

- 已新增 `internal/serverresources` registry 模块。
- 已支持 CLI：`moyuan resources add/list/show/disable`、`moyuan resources expiration scan`。
- 已支持 API：`GET/POST /v1/projects/:project_id/resources`、`GET /resources/:resource_id`、`POST /resources/:resource_id/disable`、`GET /resources/expiration-scan`。
- Inventory 写入 `.moyuan/resources/inventory.json`，事件写入 `.moyuan/resources/events.jsonl`，并记录 audit log。
- 当前不执行 SSH、云 API、部署或监控调用。
- 验证命令：`PATH=/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH go test ./...`。

## 12. 已完成任务：`beta-010 devops-deploy-smoke-monitor`

范围：

- 基于 release plan 和 server resource registry 生成 deploy plan。
- 设计部署、线上冒烟、监控窗口和回滚状态，但先只生成计划。
- 区分 test_dev、staging、production，production 必须审批。

非目标：

- 不真实 SSH。
- 不真实部署。
- 不接入外部监控 API。

验收：

- 缺失 release plan 或 server resource 时阻断。
- production 缺少审批时阻断。
- test_dev 可生成 smoke/monitor plan。
- `go test ./...` 通过。

完成记录：

- 已新增 `internal/deployment` deploy/smoke/monitor plan 模块。
- 已支持 CLI：`moyuan deploy plan <release-id> --environment <env> [--resource <host-id>]`、`moyuan deploy show <deployment-id>`。
- 已支持 API：`POST /v1/projects/:project_id/deployments/plan`、`GET /v1/projects/:project_id/deployments/:deployment_id`。
- Deployment plan 写入 `.moyuan/lifecycle/deployments/`，并记录 `deployment.plan.created` 日志。
- 当前不执行 SSH、云 API、真实部署、真实冒烟或监控 API。
- 验证命令：`PATH=/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH go test ./...`。

## 13. 已完成任务：`beta-011 controlled-deploy-executor`

范围：

- 在 deployment plan 基础上接入受控执行器。
- 建立 deployment execution 记录。
- 支持默认 dry-run，预演部署、smoke、monitor 和 rollback 步骤。
- 支持受限 `local_shell`，仅允许极小安全命令集合，且必须显式 `approved`。
- production 非 dry-run 执行当前阶段强制阻断。

非目标：

- 不允许无审批生产部署。
- 不允许仓库配置直接覆盖执行 allowlist。
- 不允许把 secret value 写入日志、Memory 或模型上下文。
- 不接入真实 SSH、云厂商 API 或外部监控 API。

验收：

- deployment plan 未 ready 时不能执行。
- dry-run 不运行任何远程或本地命令。
- local shell 未审批时阻断，危险命令被 allowlist 阻断。
- execution 记录可通过 CLI/API 查询，并写入审计日志。
- `go test ./...` 通过。

完成记录：

- 已新增 `deployment.Execute` 和 `deployment.LoadExecution`。
- 已支持 CLI：`moyuan deploy execute <deployment-id>`、`moyuan deploy execution <execution-id>`。
- 已支持 API：`POST /v1/projects/:project_id/deployments/:deployment_id/execute`、`GET /v1/projects/:project_id/deployment-executions/:execution_id`。
- Execution 写入 `.moyuan/lifecycle/deployments/executions/`，汇总写入 `executions.jsonl`，并记录 `deployment.execution.created` 日志。
- 当前只开放 dry-run 和受限 local shell；production 真实执行仍阻断。

## 14. 已完成任务：`beta-012 console-api-integration`

范围：

- Web Console 从 demo fallback 逐步迁移到 live API。
- 增加 requirement plan 表单、deployment execution 面板和 server resource health 面板。
- 明确 loading、empty、error、blocked 和 stale 状态。

非目标：

- 不在前端直接读 `.moyuan` 文件。
- 不从前端执行高风险 Git、deploy 或 production 操作。
- 不把 demo 数据完全删除，空项目或 API 缺失时仍用于首屏可观察性。

验收：

- 控制台可用真实 API 展示项目、providers、resources、issue graph、schedule、deployments 和 deployment executions。
- Requirement Intake 可通过 Web Console 调用 `requirements/plan` 低风险入口。
- deployment execution 和 server resource health 有独立状态视图。
- `go test ./...`、`npm run typecheck`、`npm run build`、`npm audit --omit=dev` 通过。

完成记录：

- 已新增 API：`GET /v1/projects/:project_id/deployments`、`GET /v1/projects/:project_id/deployment-executions`。
- Web Console 已接入 deployments、deployment executions 和 live timeline。
- Web Console 已新增 Requirement Intake 表单，提交后展示 clarification 或 issue graph 生成结果。
- Server Resources 视图展示 health 信息，Deployment Executions 视图展示 dry-run/local_shell 执行状态。

## 15. 已完成任务：`beta-013 subagent-run-visibility`

范围：

- 为 subagent/run 暴露列表 API 和控制台视图。
- 展示 role、runtime、provider、blocked reason、quality/review 状态和 artifact 路径。
- 支持按 project/epic/issue 过滤。

非目标：

- 不新增真实 Subagent 执行器。
- 不改变 orchestrator run state 的权威来源。
- 不在前端直接读取 `.moyuan/orchestrator`。

验收：

- 用户能在控制台看清每个 issue 的执行队列、等待原因和复核结果。
- `go test ./...`、`npm run typecheck`、`npm run build`、`npm audit --omit=dev` 通过。

完成记录：

- 已新增 `orchestrator.ListRunStates`。
- 已支持 CLI：`moyuan orchestrator run list [--limit 20]`。
- 已支持 API：`GET /v1/projects/:project_id/runs?limit=...`。
- Web Console Run Timeline 已优先展示 live run states，并回退到 deployment executions 和 demo timeline。

## 16. 已完成任务：`beta-014 server-health-check-executor`

范围：

- 基于 server resource registry 的 healthcheck 配置执行受控健康检查。
- test_dev/staging 可自动执行，production 必须审批。
- 健康检查结果回写 resource inventory 和 events。

非目标：

- 不扫描未登记资源。
- 不允许前端或仓库配置绕过 production approval。
- 不对外部公网地址执行 HTTP scan；当前阶段 HTTP 目标限制为 `127.0.0.1` / `localhost`。

验收：

- 服务器资源支持 `health scan`，控制台可展示 last_status 和 scan history。
- `go test ./...` 通过。

完成记录：

- 已新增 `serverresources.HealthScan`。
- 已支持 CLI：`moyuan resources health scan [--environment test_dev] [--resource <resource-id>] [--approved]`。
- 已支持 API：`POST /v1/projects/:project_id/resources/health-scan`。
- Health scan report 写入 `.moyuan/resources/checks/`，汇总写入 `.moyuan/resources/checks.jsonl`，并记录 `server_resource.health_scan` 审计日志。
- HTTP health check 当前只允许 localhost/127.0.0.1，production 未审批时强制阻断。

## 17. 下一步任务建议

### `beta-015 subagent-model`

状态：completed

范围：

- 显式建模 Subagent Instance，连接 issue、role、runtime、provider、skills、memory scope、read/write scope 和 output contract。
- Run State 引用 subagent id，控制台可从 issue 追踪到 subagent 和 runtime session。

非目标：

- 不新增无限层级子任务。
- 不让 Subagent 绕过 Orchestrator、Runtime Adapter、Quality Gate 或 Reviewer。
- 不实现完整 Skill 推荐评分，只先记录 skills 绑定。

验收：

- 每个 orchestrator run 都有可审计 subagent instance。
- Subagent 不能自行越权扩大 scope 或绕过 quality/review。
- `go test ./...` 通过。

完成记录：

- 已新增 `internal/subagent`。
- Orchestrator run 会创建 Subagent Instance，并把 `subagent_id` 写入 run state 和 result。
- 已支持 CLI：`moyuan orchestrator subagent list`、`moyuan orchestrator subagent show <subagent-id>`。
- 已支持 API：`GET /v1/projects/:project_id/subagents`、`GET /v1/projects/:project_id/subagents/:subagent_id`。
- Subagent 写入 `.moyuan/agents/subagents/`，并记录 `subagent.created` / `subagent.finished` 日志。

### `beta-016 quality-policy-api`

状态：completed

范围：

- 暴露质量门禁策略、最近 findings、coverage/test evidence 和 review verdict。
- Web Console 可查看每个 issue 为什么 accepted、needs_rework 或 blocked。

非目标：

- 不修改质量门禁判定逻辑。
- 不降低 blocker finding 门槛。
- 不在控制台直接读取 quality report 文件。

验收：

- Quality Gate 不再只是报告文件，可以通过 API 和 CLI 形成可解释视图。
- `go test ./...` 通过。

完成记录：

- 已新增 `quality.CurrentPolicy`、`quality.ListReports`、`quality.Explain`。
- 已支持 CLI：`moyuan quality policy`、`moyuan quality reports`、`moyuan quality explain <report-id>`。
- 已支持 API：`GET /v1/projects/:project_id/quality-policy`、`GET /quality-reports`、`GET /quality/:report_id/explain`。
- Explanation 输出包含 decision、reasons、checks、findings 和 policy。

### `beta-017 console-quality-subagent-view`

范围草案：

- Web Console Issue Inspector 接入 subagent、run state、quality explanation。
- 让用户能从 issue graph 点击后看到为什么 blocked、needs_rework 或 accepted。

退出条件：

- Inspector 能展示 subagent id、runtime、provider、quality report、review status、blocking findings。

完成记录：

- Web Console snapshot 已接入 `subagents`、`quality-reports` 和 `quality explain` API。
- Issue Inspector 已展示 run id、subagent id、runtime/provider、runtime status、quality report、review status、quality decision、reasons、skills、output contract 和 blocking findings。
- Quality Gates 面板优先展示 live quality explanation，不再只依赖 demo 静态信号。
