# 前端控制台文档

状态：implementation-started
责任角色：frontend_architect + product_designer + frontend
最后更新：2026-05-06

本目录定义 Moyuan Web Console 的前端技术方案。当前已创建第一版可运行控制台工程，后续以前端体验、真实 API 接入和可视化编排能力为主线迭代。

## 1. 技术结论

- 前端框架：`Next.js 16 + React + TypeScript`。
- 前端端口：`3000`。
- 后端端口：`8080`。
- 前端定位：Moyuan Control Console，面向多 Agent 工程运维、代码生命周期、服务器资源和发布投产的工作台。
- 后端边界：Go/Gin API Server 仍是唯一核心控制面，Next.js 不接管主业务状态。
- 默认开发分工：复杂 UI 首版、视觉探索和高交互页面可优先使用 `frontend` role + `claude_cli`；样式稳定后的前端代码修改、测试、修复和重构可以由 `codex_cli` 参与或主导；后端继续优先交给 `backend` role + `codex_cli`。

端口约定：

```text
Frontend: http://127.0.0.1:3000
Backend:  http://127.0.0.1:8080
```

## 2. 文档入口

| 文档 | 作用 |
| --- | --- |
| [Next.js 16 控制台方案](./nextjs16-control-console.md) | 前端架构、渲染模式、数据访问、页面结构和质量策略 |
| [Console 弹窗化与功能呈现缺口](./console-gap-and-modal-roadmap.md) | 记录还需弹窗化的操作、后端已有但前端未完整呈现的能力、测试验证和执行适配器的导航归位 |

## 2.1 当前实现入口

首个可运行控制台位于：

```text
apps/console/
```

本地运行：

```bash
cd apps/console
npm install
npm run dev
```

验证：

```bash
npm run typecheck
npm run build
npm audit --omit=dev
```

当前 live API 接入：

- 项目、Issue Graph、Schedule、Runs、Providers、Skills、Resources、Memory candidates。
- Deployment plans 和 Deployment executions。
- Requirement Intake 表单归位到 `需求与 Issue / 需求登记`，通过 `/api/projects/:project_id/requirements/plan` 调用后端低风险规划入口，并在 `需求记录` 中展示拆出的 issue、完成数和受控 commit 数。
- Provider telemetry、审批队列、身份对象、PR/MR plan、release provider execution、evidence 和 operation history/detail。
- Console 已支持多视图切换、受控表单必填字段预检、弹窗化录入、Batch Execution 操作面、Integration & Release 链路、provider route preview、Provider 管理、Skill 管理、Memory 搜索、Quality report 详情、control loop run、repair review 和 operation detail drill-down；所有成功/失败状态仍以后端 API 返回为准。
- Batches 视图已接入 batch plan create/run、worker slot、worktree、merge queue、integration preview、integration apply 和 release batch readiness；前端只调用受控 API，不自行计算合入或发版结论。
- Phase 28 已用真实 requirement plan 走通过浏览器验收流：`需求登记 -> Issue Graph -> 检查器 -> 批量计划 -> dry-run`。批量执行卡片以 Phase/需求语义作为主标题，后端 decision/reason 码在前端翻译为中文可读状态，并等待本次 dry-run API 完成后再刷新展示。
- 左侧导航已从 13 个模块入口收敛为 7 个工作流入口：`项目工作台`、`需求与 Issue`、`执行与恢复`、`质量与验证`、`发布与部署`、`AI 能力`、`权限与审计`；原 `测试验证`、`执行适配器`、`技能`、`Memory`、`Provider` 等细功能改为页面内 tabs。`需求与 Issue` 内部包含 `需求登记`、`Issue Graph` 和 `批量执行`。
- 项目是全局上下文：顶部项目切换器写入 `?project=<project_id>`，切换后重新拉取该项目下的 Issue、执行、质量、部署、AI 能力和审计数据。
- 项目接入顶部只保留接入动作，项目列表卡片统一使用 `项目名称`、`Git 地址`、`本机路径`、`技术栈` 字段；当前项目默认置顶，Git 地址以后端返回的 `remote_url` 为准，技术栈以后端按当前项目目录探测出的 languages/frameworks/package managers 为准。
- live 模式下项目相关视图不得回退展示 demo 数据；当前项目没有待执行 Issue Graph、Provider、质量信号或时间线时展示空状态，避免把其他项目或样例项目的技术栈、Issue、执行记录混入当前项目上下文。
- Issue Graph 主画布只展示当前优先级最高的未完成需求中尚未执行完成的 planned/ready/running/rework 节点；存在 `Phase N` 编号时按编号小的优先，否则按创建时间早的优先。已完成需求和已接受 issue 保留在 `需求记录`、Runs、Quality 和审计视图，不再回灌到当前待执行图。
- Issue Graph 不再从 `phase1-epic` 或旧 `local-cli-mvp` 默认 Phase1 图兜底取数；没有未完成需求规划时显示“当前没有待执行的 Issue Graph”。
- Issue Graph 前端呈现为轻量 SVG DAG：直接使用需求 graph 的 `nodes + depends_on` 计算依赖层级、绘制连线、节点点击高亮上游依赖和下游影响范围，不引入重型图编辑库。
- Requirement/Issue ID 展示层会清理历史数据中因旧版 UTF-8 字节截断产生的 `�` 替换字符；后端新生成 requirement slug 已改为按字符截断，避免中文 ID 再出现浏览器乱码。
- 受控提交只来自 Moyuan issue run 的 runtime/orchestrator 结果，前端展示 `commit_before`、`commit_after`、diff summary 和 changed files；用户通过其他方式修改或提交的代码不进入需求记录。
- 部署、资源、Provider、Skill、审计、Memory、质量详情和执行适配器的新增/补充参数入口均采用弹窗；必填字段显示 `*`，提交状态以后端返回为准。

## 3. 设计原则

- 工作台优先，不做营销型首页。
- 图谱优先，Issue Graph、Run Timeline、Deployment Pipeline 和 Memory Flow 要成为一等视图。
- 状态可解释，任何 blocked、needs_rework、approval_required 都必须能看到原因、证据和下一步。
- 操作可回滚，高风险动作必须走确认、审批、审计和 rollback 视图。
- 密度适中，页面要适合长期盯盘和反复操作，不做大面积装饰。
- 前沿但克制，优先使用 Next.js 16 的 App Router、Cache Components、Suspense、Server Components 和 `proxy.ts` 网络边界，而不是堆叠复杂前端状态库。

## 4. 与现有文档关系

- API 和状态来源：[参考架构](../reference-architecture.md)、[实现模块拆分](../implementation-module-map.md)。
- Issue Graph 和调度：[Issues 编排与并发调度](../issue-orchestration.md)。
- Runtime 和 Provider：[模型与工具适配规划](../model-tool-adapters.md)。
- 鉴权和权限：[鉴权与访问控制策略](../policies/auth-access-policy.md)、[权限模型](../foundations/permission-model.md)。
- 发布投产：[DevOps 发布投产主线](../mainlines/devops-release-deployment.md)。
