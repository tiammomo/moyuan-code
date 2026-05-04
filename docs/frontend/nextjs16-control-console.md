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

第一批页面按工作流而不是模块菜单组织：

| 页面 | 目标 |
| --- | --- |
| Projects | 查看已接入项目、理解状态、Git 状态、最近风险 |
| Project Overview | 项目画像、模块地图、命令、Memory 摘要、下一步建议 |
| Requirement Planning | 输入需求、查看澄清判断、完善后的需求和 Issue Graph |
| Issue Graph | 图形化展示依赖、并发度、waiting reason 和执行计划 |
| Runs | 查看 Runtime 调用、Subagent 输出、diff、stdout/stderr 和风险 |
| Quality Review | 查看 quality report、review finding、测试缺口和返工建议 |
| Memory | 查看 record、candidate、compact、命中证据和维护事件 |
| Providers & Runtimes | 查看 Claude CLI、Codex CLI、MiniMax、GPT、GLM、gpt-image-2 状态 |
| Runtime Recoveries | 查看原生 runtime 失败归档、fallback candidate、resume hint、stdout/stderr 和 diff 摘要预览 |
| Subagent Backlog | 查看 retry/archive 后进入调度等待的 subagent、失败原因和重试预算 |
| Visual Assets | 查看架构图 plan、diagram spec、prompt、route decision、render execution、script path 和图片生成状态；支持受控 dry-run render |
| Git & Release | 查看分支、PR/MR plan、release suggestion 和 tag/push 计划 |
| Server Resources | 查看 test_dev、production 机器、到期、健康和维护窗口 |
| Deployments | 查看部署计划、审批、线上冒烟、监控和 rollback |
| Logs & Audit | 查看核心日志、审计事件和 trace |
| Settings | 项目配置索引、权限、策略和 schema 校验结果 |

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
- Controlled Actions：低风险动作可从 Console 触发后端 dry-run 或 preview，例如 visual render dry-run；高风险动作仍必须进入 approval/authz。
- Schema-aware Forms：表单从 contract/schema 生成约束，错误能定位到字段。
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
- 建立 API adapter、错误模型和 demo fallback。已完成首版。

第二阶段：可视化核心

- Projects。
- Project Overview。
- Requirement Planning。
- Issue Graph。
- Runs。
- Quality Review。

第三阶段：生产闭环

- Providers & Runtimes。
- Memory。
- Git & Release。
- Server Resources。
- Deployments。
- Logs & Audit。

第四阶段：体验增强

- Command Palette。
- Live status。
- Diff-first Review。
- Approval Drawer。
- AI Assist Surface。
- Playwright 主流程覆盖。
