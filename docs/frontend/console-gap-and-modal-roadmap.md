# Console 弹窗化与功能呈现缺口

状态：active
责任角色：frontend_owner + backend_owner + product_owner
最后更新：2026-05-06

本文记录 Console 当前前后端对照、仍需弹窗化的操作、后端已具备但前端尚未完整呈现的功能，以及测试验证和后端执行模块的导航归位规则。

## 1. 当前结论

- 录入/补充内容默认使用弹窗，不再把大表单长期压在页面卡片顶部。
- 常驻页面只展示状态、列表、证据摘要和动作入口；涉及字段填写、审批理由、执行模式、目标资源、Secret Ref、Approval ID 的操作进入弹窗。
- 前端左侧导航收敛为 7 个主入口，详细能力放进页面内 tabs：
  - `项目工作台`：项目接入、项目列表、当前项目画像和需求录入。
  - `需求与 Issue`：需求登记、需求记录、Issue Graph、批量执行、worktree、merge queue 和 release batch readiness。
  - `执行与恢复`：运行时间线、runtime recovery、subagent backlog、operation history/detail 和 repair candidate。
  - `质量与验证`：质量报告、test/lint/build/typecheck、部署 dry-run、资源健康扫描、monitor summary、post-deployment verification、rehearsal 和 release admission。
  - `发布与部署`：Release、PR/MR、Deployment、Server Resource，以及作为高级安全 tab 的执行适配器。
  - `AI 能力`：Provider、Visual assets、Skills、Memory 和 route preview。
  - `权限与审计`：审计导出、决策账本、审批队列、审计事件、会话、Token 和服务账号。
- `测试验证`、`执行适配器`、`技能`、`Memory`、`Provider` 不再作为左侧一级菜单；它们仍保留为对应主入口内的 tabs。
- 当前项目是全局工作上下文：URL 使用 `?project=<project_id>` 记录当前项目，顶部常驻项目切换器，需求登记、Issue、执行、质量、发布、审计等所有操作都基于当前项目 ID 调用后端 API。
- 需求入口归位到 `需求与 Issue / 需求登记`；登记后生成 requirement plan、issue graph 和 schedule，`需求记录` 展示进行中/已完成需求、拆分 issue、完成数和受控 commit 数。
- `Issue Graph` 主画布只呈现当前项目优先级最高的未完成需求里的待执行节点；存在 `Phase N` 编号时按编号小的优先，否则按创建时间早的优先。已完成的 Phase1/历史需求和已接受 issue 不再展示在主图中，只保留在需求记录、Runs、质量和审计视图。
- Phase 28 已作为真实需求计划完成一次浏览器闭环验收：从 `需求与 Issue / 需求登记` 进入 `Issue Graph`，在检查器查看依赖，再到 `批量执行` 创建计划并触发 dry-run。验收要求等待当前 dry-run API 完成后再判断成功，避免被历史运行记录误判。
- 批量执行页面必须优先展示用户能理解的 Phase/需求语义：卡片主标题使用 `Phase N 批量执行`，副标题展示 batch id、需求标题和时间；后端 `BATCH_PLAN_READY`、`BATCH_RUN_DRY_RUN`、`dispatch_ready`、`no_runtime_executed` 等 decision/reason 码翻译为中文状态。
- Requirement/Issue ID 展示不得露出 `�` 乱码。历史记录中已经存在的旧 ID 只在前端显示层清洗，不改写事实源；后端新 requirement slug 必须按字符截断，不能按字节截断中文。
- Commit 信息只展示 Moyuan issue run 管理过的结果：`runs` 返回 `commit_before`、`commit_after`、`diff_summary_path` 和 `changed_files`。外部手工修改、手工 commit 或其他工具更新不被补记到需求闭环。
- `项目接入` 顶部只保留接入操作入口；项目列表卡片按字段展示 `项目名称`、`Git 地址`、`本机路径`、`技术栈`，没有 Git 远程时显示“未绑定 Git 远程”。
- 项目列表默认把当前选中项目置顶；Git 地址以后端 `remote_url` 或项目 source 中的 remote 字段为准，本地项目已绑定 `origin` 时也必须展示。
- 当前项目数据必须隔离：live 模式下 Issue Graph、schedule、Provider、timeline、quality、Memory 等项目相关视图不能用 demo 数据补空；没有待执行图时显示中文空状态。
- 后端项目接入不再给所有项目套用 `local-cli-mvp` 默认图；前端也不再使用 `phase1-epic` 或旧默认 Phase1 图作为 Issue Graph 兜底，后端应按项目目录生成项目接入基线。

## 2. 已弹窗化

| 区域 | 操作 | 状态 |
| --- | --- | --- |
| 项目 | 接入本地项目、接入 GitHub/Gitee/Git 项目 | 已弹窗化 |
| 部署 | 新增服务器 | 已弹窗化 |
| 审计 | 创建会话、创建 API Token、保存服务账号 | 已弹窗化 |
| 审计 | 发起审批、会话撤销、API Token 撤销 | 已弹窗化 |
| 审计 | 审批批准/拒绝，补决策人和原因 | 已弹窗化 |
| 运行 | 修复候选批准/拒绝，补复核人和原因 | 已弹窗化 |
| 运行/操作 | 从操作详情创建 operation repair candidate | 已弹窗化 |
| Issue Graph | Issue merge decision 和 Git Provider plan 创建 | 已弹窗化 |
| 批量执行 | Batch plan 创建 | 已弹窗化 |
| 部署 | 服务器续期/退役，补执行人、到期日、原因 | 已弹窗化 |
| 部署 | Release Provider preview/publish，补 Release ID、Approval ID | 已弹窗化 |
| 部署 | PR/MR create，补 Approval ID | 已弹窗化 |
| 部署 | Deployment plan 创建，补 Release ID、环境、资源和批准状态 | 已弹窗化 |
| 部署 | Deployment execute，补执行模式、Approval ID 和 commands | 已弹窗化 |
| 部署 | 维护扫描、生命周期扫描、健康扫描、资源禁用 | 已弹窗化 |
| 测试验证 | 部署风险交接批准/拒绝，补 reviewer、reason、next step | 已弹窗化 |
| 执行适配器 | remote execution rehearsal、write review packet、write execution plan、write adapter execution | 已弹窗化 |
| 执行适配器 | control loop queue 入队和 queue run 消费 | 已弹窗化 |
| Provider | Provider 新增、Ops refresh、手工 Ops snapshot、禁用 | 已弹窗化 |
| Provider | Visual diagram plan 创建 | 已弹窗化 |
| 技能 | Skill 注册、推荐、绑定、效果记录、禁用 | 已弹窗化 |
| Memory | Memory 搜索 | 已弹窗化 |
| 质量 | Quality report detail/explain 详情 | 已弹窗详情 |

## 3. 仍建议弹窗化

P0：

- `resources/:id/disable` 当前已弹窗确认，但后端 route 暂未接收执行人和原因；需要后端 contract 增强后再补必填字段。

P1：

- Memory candidate 接受/忽略、compact 策略。当前后端只暴露 search 和 candidates 列表。
- Visual render 参数选择。当前 plan 创建和 asset dry-run 已有，render 仍只提供 dry_run 快捷入口。
- Quality policy 可视化策略面。当前 report detail/explain 已可展开，policy 仍以摘要为主。

## 4. 后端已具备但前端尚未完整呈现

| 后端能力 | 当前前端状态 | 建议归位 |
| --- | --- | --- |
| 多项目列表、项目详情、项目切换 | 已展示项目列表和当前项目；顶部项目切换器通过 `?project=<project_id>` 重新拉取当前项目上下文；项目卡展示 Git、本机路径和技术栈 | `项目工作台 / 项目接入` + 全局顶部 |
| Requirement list/detail | 已补 `/requirements` 列表，前端在 `需求登记` 展示需求记录、完成状态和拆分 issue | `需求与 Issue / 需求登记` |
| Issue 详情、Epic schedule 详情 | Issue Graph 已升级为 SVG DAG，按 `depends_on` 自动分层并绘制依赖边；主画布只展示未完成需求的待执行节点，点击节点高亮上游/下游，检查器展示受控 commit/diff 信息 | `需求与 Issue / Issue Graph` |
| Batch plan 创建 | 已补创建弹窗，batch dry-run 和 merge queue 保持后端事实源 | `需求与 Issue / 批量执行` |
| Deployment plan 创建 | 已补独立弹窗入口 | `发布与部署 / 发布部署` |
| Deployment execute 多模式 | 已补执行弹窗，支持 mode、Approval ID、commands | `质量与验证 / 测试验证` + `发布与部署 / 发布部署` |
| Resource expiration/maintenance/lifecycle scans | 已补维护/生命周期/健康扫描入口 | `质量与验证 / 测试验证` + `发布与部署 / 发布部署` |
| Resource disable | 已补禁用确认弹窗 | `发布与部署 / 发布部署` |
| Post-deployment history/detail、rollback execution detail | 列表摘要有，详情 drill-down 不完整 | `质量与验证 / 测试验证` |
| Deployment risk review | 已补批准/拒绝复核弹窗 | `质量与验证 / 测试验证` |
| Operation repair candidate create、repair attempt detail | 已可从操作详情创建 repair candidate；attempt 详情仍可继续增强 | `执行与恢复 / 运行时间线` + `执行与恢复 / 操作证据` |
| Control loop queue create/run | 已补入队和消费弹窗 | `发布与部署 / 执行安全` |
| Remote execution rehearsal create | 已补创建弹窗 | `发布与部署 / 执行安全` |
| Write review packet/create、write execution plan/create、write adapter execution/create | 已补 create 弹窗 | `发布与部署 / 执行安全` |
| Write adapter recovery approval/retry/repair runner | recovery 展示有，approval consumption 和 retry/repair dry-run runner 未形成前端闭环 | `发布与部署 / 执行安全` |
| Provider create/ops refresh/manual ops/disable | 已补管理弹窗 | `AI 能力 / Provider` |
| Skills registry/binding/effectiveness | 已补 `技能` tab 和弹窗 | `AI 能力 / 技能` |
| Memory search、requirement detail | 已补 Memory 搜索和需求澄清补充弹窗；requirement 历史详情仍可继续增强 | `AI 能力 / Memory` + `项目工作台 / 项目接入` |
| Visual diagram plan | 已补 plan 创建；asset/render dry-run 继续在 Provider 面展示 | `AI 能力 / Provider` |
| Approval create、session revoke | 已补手工审批、会话撤销和 Token 撤销 | `权限与审计 / 权限审计` |
| Quality policy、quality report detail/explain | 已补 report detail/explain 抽屉；policy 策略面仍可增强 | `质量与验证 / 代码质量` |

## 5. 放置规则

- 创建类、审批类、执行模式类、补充参数类操作必须使用弹窗。
- 查看类、证据链、历史记录、状态矩阵保留在页面内。
- `质量与验证 / 测试验证` 只展示“是否可继续”的信号和 dry-run/verification/rehearsal 操作，不放 Git/SSH adapter 低层 guard。
- `发布与部署 / 执行安全` 展示后端写入链路和 adapter 执行事实，不替代业务发布流水线。
- `发布与部署 / 发布部署` 展示 Release、PR/MR、Deployment、Server Resource 的业务闭环。
- `权限与审计 / 权限审计` 展示谁在何时做了什么、审批如何决策、身份对象如何变化。

## 6. 后续验收

- 新增任何 `POST`/`PATCH`/`DELETE` 前端入口时，同步补弹窗、必填 `*`、schema 错误展示和文档记录。
- 前端不自行构造成功态，所有状态以后端返回为准。
- 高风险动作必须显示 approval、receipt、rollback/smoke 或 write proof 中至少一个后端事实源。
- 每轮前端变更至少运行 `npm --prefix apps/console run typecheck` 和 `git diff --check`。
