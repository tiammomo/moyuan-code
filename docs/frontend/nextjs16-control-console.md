# Next.js 16 控制台方案

状态：phase2-observability-added
责任角色：frontend_architect + product_designer + frontend
最后更新：2026-05-05

本文是 Moyuan Web Console 的前端唯一详细技术方案。它不重复后端 API、Provider、Memory、DevOps 和权限模型的完整定义，只规定前端如何承接这些能力。

## 1. 目标

Web Console 要让用户看清楚并控制一个项目从接入、理解、需求拆分、Issue Graph、并发执行、质量复核、合入、发版、部署、冒烟、监控到自我修复的全过程。

它不是单纯的 CRUD 后台，而是一个 AI Engineering Ops Console：

- 能解释 AI 为什么这么拆 issue。
- 能展示哪些任务并发、哪些任务等待前置条件。
- 能看到 Claude CLI、Codex CLI、MiniMax、GPT、GLM 等 provider 的路由结果。
- 能追踪每个 issue 的 diff、quality report、review finding 和返工记录。
- 能控制 GitHub/Gitee、服务器资源、部署计划、线上冒烟和生产监控。
- 能查看 Memory record、compact、命中证据和维护事件。

## 2. 技术基线

| 维度 | 选择 |
| --- | --- |
| Framework | `Next.js 16` |
| UI Runtime | `React` + App Router |
| Language | `TypeScript` |
| Dev Port | `3000` |
| Backend API | Go/Gin, `127.0.0.1:8080` |
| Styling | Tailwind CSS + CSS variables |
| Primitive UI | Radix UI / shadcn-style primitives |
| Server State | TanStack Query for client islands；Server Components 直接 fetch 低交互数据 |
| Graph | React Flow |
| Tables | TanStack Table |
| Editor/Diff | Monaco Editor 或轻量 diff viewer |
| Charts | Recharts 或 Tremor-style primitives |
| Tests | Vitest + Testing Library + Playwright + MSW |

Next.js 16 相关能力采用：

- App Router：所有页面和布局默认走 `app/`。
- Server Components first：页面级数据读取优先在 Server Components 完成。
- Client Components as islands：只把图谱、拖拽、日志流、表格交互、编辑器和命令面板做成 client island。
- Cache Components：启用 `cacheComponents: true`，用 `use cache`、`cacheLife`、`cacheTag` 管理稳定数据。
- Suspense boundaries：每个慢数据面板都要有独立边界，避免整页阻塞。
- Turbopack：默认开发和构建工具链，避免额外 bundler 复杂度。
- `proxy.ts`：显式表达网络边界，替代旧式 middleware 思维。
- DevTools MCP：预留给 Claude/Codex 调试前端页面、路由和日志。

参考：

- Next.js 16 Release：https://nextjs.org/blog/next-16
- App Router：https://nextjs.org/docs/app
- Cache Components：https://nextjs.org/docs/app/getting-started/cache-components
- Rewrites：https://nextjs.org/docs/app/api-reference/config/next-config-js/rewrites

## 3. 运行边界

开发环境：

```text
Next.js: http://127.0.0.1:3000
Go API:  http://127.0.0.1:8080
```

推荐代理：

```ts
// next.config.ts
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  cacheComponents: true,
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: "http://127.0.0.1:8080/v1/:path*",
      },
    ];
  },
};

export default nextConfig;
```

规则：

- 浏览器只访问 Next.js 的 `/api/*` 或 Server Component 页面，不直接散落调用 `8080`。
- Go API 仍是权威状态入口。
- Next.js Route Handler 只能做 BFF/proxy、cookie/session 辅助、流式转发和视图聚合，不能成为第二套业务状态机。
- 高风险动作不能只靠前端确认，必须调用后端 approval/authz。

## 4. 推荐目录结构

```text
apps/console/
  app/
    (console)/
      layout.tsx
      page.tsx
      projects/
      requirements/
      issues/
      runs/
      quality/
      memory/
      providers/
      resources/
      deployments/
      logs/
      settings/
    api/
      health/route.ts
  components/
    command/
    graph/
    layout/
    timeline/
    quality/
    memory/
    deployment/
    forms/
  lib/
    api/
    auth/
    query/
    schemas/
    routes/
    telemetry/
  styles/
  tests/
```

说明：

- `app/(console)` 是控制台主区域，不做营销首页。
- `components/graph` 承载 Issue Graph、Dependency Graph、Deployment Graph。
- `components/timeline` 承载 Run Timeline、Review Timeline、Deployment Events。
- `lib/schemas` 使用 Zod 镜像后端 contract，前端不手写散乱类型。
- `lib/api` 封装 Go API 调用、错误归一、trace id 和重试策略。

## 5. 页面模型

第一批页面按工作流而不是模块菜单组织。左侧只保留 7 个主入口，原细页面放到页内 tabs：

| 左侧入口 | 页内 tabs | 目标 |
| --- | --- | --- |
| 项目工作台 | 项目接入 | 查看已接入项目、切换当前项目、当前项目画像、Git 状态、需求录入、澄清补充和下一步建议 |
| 需求与 Issue | Issue Graph、批量执行 | 展示依赖、并发度、waiting reason、batch plan、worktree、merge queue 和 release batch readiness |
| 执行与恢复 | 运行时间线、操作证据 | 查看 Runtime 调用、Subagent 输出、runtime recovery、operation detail、evidence chain 和 repair candidate |
| 质量与验证 | 代码质量、测试验证 | 查看 quality report、review finding、测试缺口、dry-run、资源健康、post-deployment verification、monitor summary、rehearsal 和 release admission |
| 发布与部署 | 发布部署、执行安全 | 查看 Release、PR/MR、Deployment、Server Resource、write proof/admission、remote rehearsal、write adapter execution/recovery 和 control queue |
| AI 能力 | Provider、技能、Memory | 查看 Provider/runtime 状态、route preview、Visual assets、Skill registry/binding/effectiveness、Memory record/candidate/search |
| 权限与审计 | 权限审计 | 查看审批队列、决策账本、身份对象、审计事件、trace 和 schema 校验结果 |

## 6. 渲染模式

默认策略：

- 项目列表、项目画像、静态配置索引：Server Component + `use cache`。
- Issue Graph 页面壳：Server Component。
- Issue Graph 画布：Client Component。
- Run Timeline：Server Component 提供初始数据，Client Component 负责筛选、展开和轮询。
- 日志流：Client Component，后续升级 SSE/WebSocket。
- 表单：Server Action 或 Client mutation 二选一；涉及审批和高风险操作时必须调用后端 confirmation API。
- 大型 diff/editor：Client Component 懒加载。

缓存策略：

- 项目元数据、模块地图、provider registry：可缓存，按 project/version/tag revalidate。
- Run、quality、deployment、logs：默认请求时数据，不做长期缓存。
- Memory search：按 query 和 memory scope 短缓存，写入或 compact 后失效。
- Release suggestion：按 release id 缓存；新 accepted issue 或新 merge decision 后失效。

## 7. 前沿交互模式

控制台要有这些高优先级交互模式：

- Command Palette：跨项目跳转、执行安全命令、打开 issue/run/provider。
- Inspectable Graph：点击节点即可看到前置依赖、blocked reason、runtime、provider、quality gate。
- Split Pane：左边图谱或列表，右边详情，不频繁跳页。
- Timeline Native：Run、Review、Deploy、Memory compact 都以时间线展示。
- Diff-first Review：质量页面以 diff、finding、test gap 和 action 为核心。
- Human-in-the-loop Approval：高风险动作内联展示审批原因和影响范围。
- Progressive Disclosure：默认展示结论，点击后展开证据、日志、原始 JSON。
- Live Workbench：运行中 issue 的状态、日志和下一步持续刷新。
- Phase 2 Observability：把 runtime recoveries、subagent backlog、visual assets 和 visual render executions 放在同一屏，便于判断“失败如何恢复、任务为什么等待、架构图是否已规划、图片生成是否已进入受控执行”。
- Runtime Evidence Preview：runtime recovery 可展开 stdout、stderr 和 diff summary 的受控预览；后端只读取 recovery 记录指向且位于 `.moyuan/` 下的归档文件。
- Operation Repair Candidate Surface：Runtime Recoveries 面板展示 repair candidate、failure class、repair plan、evidence 数量和 review 结果；Operation Detail 可从选中的后端 operation 生成 repair candidate；approve/reject 必须调用后端 review API，approve 默认只创建 `review_ready` repair attempt，不自动运行修复。
- Controlled Actions：低风险动作可从 Console 触发后端 dry-run、preview 或 bounded run，例如 visual diagram plan、visual render dry-run、release provider preview、provider route preview、issue merge decision、git provider plan、memory search、control loop run；审批决定、approval create、API token、session revoke、service account、PR/MR create、release provider publish、deployment plan/execute、batch plan、资源新增/续期/退役/禁用、Provider 管理、Skill 管理、repair candidate review 等写操作必须调用后端受控 API，高风险动作仍进入 approval/authz。
- Modal-first Input：录入、补充参数、审批理由、执行模式、目标资源、Secret Ref、Approval ID 和 reviewer/reason 默认进入弹窗；页面常驻区域只保留状态、证据摘要和动作入口。已落地项目接入、需求澄清补充、服务器资源、deployment plan/execute、batch plan、风险复核、审批创建/决策、会话和 Token 撤销、Provider 管理、Provider Ops snapshot、Skill 管理、Memory 搜索、Quality report 详情、control queue 和写入 adapter create 链路。必填项必须显示 `*`，并在前端预检和后端 schema 错误之间保持一致。
- Project Context First：当前项目是 Console 的全局上下文，URL 使用 `?project=<project_id>` 表示；顶部常驻项目切换器，切换后服务端重新拉取该项目下的 Issue、运行、质量、发布、部署、AI 能力和审计数据，所有写操作都使用当前 `project_id`。
- Project Onboarding Fields：项目接入顶部不是说明卡片，只保留接入动作；下方项目列表展示绑定事实：`项目名称`、`Git 地址`、`本机路径`。当前项目默认置顶，Git 地址以后端 `remote_url` 或 source remote 字段为准，Git 远程缺失时显示“未绑定 Git 远程”。
- Workflow Navigation：左侧导航只保留 `项目工作台`、`需求与 Issue`、`执行与恢复`、`质量与验证`、`发布与部署`、`AI 能力`、`权限与审计` 7 个主入口；细能力在页面内 tabs 切换，避免把后端模块目录暴露给中文业务用户。
- Testing & Validation Surface：测试验证作为 `质量与验证` 内 tab，集中展示 quality signal、dry-run、resource health scan、monitor summary、post-deployment verification、deployment rehearsal、rehearsal scheduler 和 release admission。它只回答“是否可继续”，不承载 adapter 低层 guard。
- Write Adapter Surface：执行适配器作为 `发布与部署` 内 `执行安全` tab，集中展示 write proof/admission、provider proof requirement、remote execution rehearsal、write review packet、write execution plan、write adapter execution/recovery 和 control queue。它只展示后端执行事实和受控入口，不替代业务发布流水线。
- Skill Surface：Skill 作为 `AI 能力` 内 tab，展示 registry、binding、effectiveness 和最新 recommendation；新增、推荐、绑定、效果记录和禁用全部走弹窗。
- Operation Detail：从 Operation History 选中 release provider 或 deployment execution 后，Console 优先读取 operation detail 聚合 API，展开 Evidence Chain，显示 evidence decision、reasons 和 artifact path；API 不可用时回退到 snapshot，刷新按钮触发 `router.refresh()` 重新拉取当前状态。
- Schema-aware Forms：表单从 contract/schema 生成约束，错误能定位到字段；当前已先落地必填字段预检，后续再接入完整 schema metadata。
- Provider Telemetry Surface：Provider 面板展示 health/quota/cost/quality 摘要和近期 telemetry 记录；Provider route preview 展示 selected/skipped/blocked candidates、score、runtime/model 和 provider 侧原因。
- Resource Lifecycle Surface：Server Resources 面板展示 lifecycle alerts、expiration state、maintenance records 和资源续期/退役动作状态。
- Deployment Monitor Surface：Deployment Executions 面板展示 post-deployment history 和 post-deployment verification，把 smoke/monitor 状态、失败分类、rollback runbook、risk handoff recommendation 和 evidence chain 作为后端事实源展示，不在前端自行推断高风险结论。
- Operations Dashboard Surface：Console 优先读取后端 operations timeline，展示 post-deployment verification、resource deployment refs、resource lifecycle 和 risk review；前端只做筛选、展开和刷新，不重算 deployment readiness、release admission 或 maintenance policy。
- Control Loop Surface：Console 展示 control loop run、step decision、summary、duration、evidence 数量；手动触发只运行 bounded control loop，不启动常驻 scheduler。
- Batch Execution Surface：Console 展示后端 `batch_plans`、`batch_runs`、`worktrees` 和 `merge_queues`，用户可触发 batch dry-run 和 merge queue build；dependency、write scope、quality/review 和 merge readiness 只以后端事实源为准。
- AI Assist Surface：保留“让 agent 解释当前状态 / 生成修复建议 / 生成发布说明”的入口，但不能绕过后端门禁。

## 8. 设计语言

视觉方向：

- 专业、密集、清晰，接近工程控制台和云平台控制面。
- 使用中性色作为底，状态色明确表达 success/warning/error/blocked/running。
- 避免大面积渐变、营销 hero、装饰卡片和单色主题。
- 卡片只用于 repeated item、modal 和工具面板；页面区域用分栏、表格、图谱和时间线。
- 图标用于工具按钮和状态，不用大段文字解释功能。

核心组件：

- App Shell：项目切换、全局搜索、运行状态、用户菜单。
- Project Switcher。
- Issue Graph Canvas。
- Runtime Run Panel。
- Quality Finding List。
- Diff Viewer。
- Approval Drawer。
- Memory Record Inspector。
- Provider Health Matrix。
- Deployment Pipeline View。
- Audit Event Table。

## 9. 数据与错误处理

前端必须统一这些对象：

- `ApiResult<T>`
- `ApiError`
- `AuthContext`
- `ProjectSummary`
- `IssueGraph`
- `SchedulePlan`
- `RunState`
- `QualityReport`
- `ProviderRoute`
- `MemoryRecord`
- `ServerResource`
- `DeploymentPlan`
- `AuditEvent`

错误展示规则：

- `401/403`：展示权限缺口、当前身份和申请入口。
- `404`：展示资源 id、项目 id 和可能的刷新/重新接入动作。
- `409`：展示冲突对象，例如 worktree dirty、branch conflict、runtime slot。
- `422`：定位表单字段和 schema 规则。
- `500`：展示 trace id、日志入口和重试建议。

## 10. 质量门禁

前端实现必须具备：

- TypeScript strict。
- ESLint。
- Prettier 或等价 formatter。
- Vitest 覆盖核心纯函数、schema、API adapter。
- Testing Library 覆盖关键交互组件。
- Playwright 覆盖端到端主流程：项目列表、需求规划、Issue Graph、Run、Quality、Deployment。
- MSW 或测试替身覆盖 API 错误、loading、empty、blocked、needs_rework。
- 可访问性检查：键盘导航、焦点管理、颜色对比和 aria。

不允许：

- 在组件里散落裸 `fetch`。
- 在 Client Component 里硬编码后端 URL。
- 把 token、API key、SSH key、`.env` 明文传到浏览器。
- 页面靠轮询刷全量大对象；大对象必须分页、增量或局部刷新。
- UI 只展示成功态，缺少 loading、empty、error、blocked、needs_rework。

## 11. 实施顺序

第一阶段：前端骨架

- 创建 `apps/console`。已完成首版。
- Next.js 16、TypeScript、基础 App Shell。已完成首版。
- 固定端口 `3000`。已完成。
- 配置 `/api/* -> 127.0.0.1:8080/v1/*` rewrite。已完成。
- 建立 API adapter、错误模型和 demo fallback。已完成首版；demo fallback 仅用于后端不可用或没有项目列表的演示模式，live 项目上下文不得用 demo 数据补空。
- 项目上下文隔离：`?project=<project_id>` 是 Issue、运行、质量、发布、AI 能力和审计数据的全局上下文；前端需要兼容 Issue Graph 的 `nodes` 字段，并在当前项目没有真实图谱时展示空状态。
- 需求闭环：`需求与 Issue / 需求登记` 承接需求录入和拆 issue，`需求记录` 只展示通过 Moyuan requirement planner 生成的需求；commit/diff 信息只来自 Moyuan issue run 返回的受控运行结果，不扫描外部 git 历史。

第二阶段：可视化核心

- Projects。
- Project Overview。
- Requirement Planning。
- Issue Graph。
- Runs。
- Quality Review。

第三阶段：生产闭环

- AI 能力：Providers & Runtimes、Skills、Memory、Visual Assets。
- 发布与部署：Git & Release、Server Resources、Deployments、执行安全。
- 质量与验证：Testing & Validation、release admission、post-deployment verification。
- Write Adapters。
- Logs & Audit。

第四阶段：体验增强

- Command Palette。
- Live status。
- Diff-first Review。
- Approval Drawer。
- Modal coverage for all create/approve/execute/repair actions。
- AI Assist Surface。
- Playwright 主流程覆盖。
