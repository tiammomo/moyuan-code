# Beta 实现 Issue Graph

状态：in_progress
责任角色：orchestrator_owner + backend_owner + qa_owner
最后更新：2026-05-04

本文记录 Phase 1 后的 Beta 阶段执行图。Beta 不重新规划 Phase 1 已完成能力，只在稳定本地 CLI MVP 上扩展控制面 API、任务编排、Provider、Git Provider、服务器资源和发布投产能力。

## 1. Beta 目标

- 将 `.moyuan/` 中已经沉淀的项目状态，通过 Gin API 形成稳定查询入口。
- 将用户需求到 issue graph、schedule、run、review、merge 的编排过程推进到可持续执行。
- 将 Provider、GitHub/Gitee、服务器资源、DevOps 发布和线上反馈逐步纳入统一控制面。
- 保持“先理解项目、再拆 issue、再执行、再复核、再合入”的质量边界。

## 2. Issue Graph

| ID | Issue | 状态 | 主要范围 | 依赖 | 建议角色 | 退出条件 |
| --- | --- | --- | --- | --- | --- | --- |
| `beta-001` | `state-query-api` | completed | Gin API 查询 project、issue、run、quality、memory、repair 的当前状态 | Phase 1 | `backend_owner` | API 可按 project id 读取核心状态，缺失资源返回 404，测试覆盖 |
| `beta-002` | `issue-graph-api` | completed | 暴露 epic/issue graph/schedule 查询接口，为 Web Console 和编排可视化做准备 | `beta-001` | `orchestrator_owner` | 可读取 issue graph、ready/blocked/review 队列和 blocked reason |
| `beta-003` | `requirement-to-issues` | completed | 将用户需求丰富、澄清判断、验收标准和 issue graph 生成接入 CLI/API | `beta-002` | `orchestrator_owner` | 需求可生成用户可见 issue graph，并标注依赖 |
| `beta-004` | `parallel-orchestration-engine` | completed | 根据依赖、写入范围、runtime slot 和风险自动决定并发度 | `beta-003` | `scheduler_owner` | ready queue 可并发调度，冲突 issue 自动等待 |
| `beta-005` | `review-merge-pipeline` | completed | issue 完成后执行复核、风格检查、门禁、合入或返工 | `beta-004` | `quality_owner` | review 通过后才允许合入任务分支 |
| `beta-006` | `provider-registry-runtime-routing` | completed | 管理 GPT、Claude、GLM、MiniMax、第三方 API 和 CLI agent runtime | `beta-001` | `adapter_owner` | Provider 可配置、校验、路由和审计 |
| `beta-007` | `git-provider-pr-mr` | completed | GitHub/Gitee 认证、分支、push、PR/MR 创建和状态回读 | `beta-005` | `git_owner` | 可推送任务分支并创建 PR/MR |
| `beta-008` | `release-branch-pipeline` | completed | 版本分支、release 建议、tag、GitHub/Gitee 发布记录 | `beta-007` | `release_owner` | 可按积累量建议发版并发布到 Git provider |
| `beta-009` | `server-resource-registry` | completed | 测试机/生产机、到期时间、配置、权限、健康和维护记录 | `beta-001` | `infra_owner` | 服务器资源可登记、查询、审计 |
| `beta-010` | `devops-deploy-smoke-monitor` | in_progress | 部署、线上冒烟、生产监控和后续更新维护 | `beta-008`,`beta-009` | `devops_owner` | 可对配置服务器执行受控发布和回滚 |

## 3. 推荐执行顺序

1. 先做 `beta-001`，让控制面有统一读取入口。
2. `beta-002`、`beta-003` 串行推进，避免 issue graph 生成和展示口径分裂。
3. `beta-004`、`beta-005` 在 issue graph API 稳定后推进，解决并发、等待、review 和合入。
4. `beta-006` 可以与 `beta-002` 并行，但不得影响本地 CLI fallback。
5. `beta-007`、`beta-008` 依赖 review/merge pipeline，不提前自动 push 或发版。
6. `beta-009`、`beta-010` 在 release pipeline 可审计后推进。

## 4. 当前执行入口

当前执行 `beta-010 devops-deploy-smoke-monitor`。

实现边界：

- 先基于 release plan 和 server resource registry 生成 deploy/smoke/monitor plan。
- production 必须审批，test_dev 可生成无审批演练计划。
- 本阶段只做计划和状态记录，不真实连接服务器、不部署。
- 状态事实仍以 `.moyuan/` 文件为准，云厂商和监控系统状态后续作为索引同步。
