# ADR-0006：Web Console 使用 Next.js 16

状态：accepted
日期：2026-05-05

## 背景

Moyuan 后续需要 Web Console 展示项目接入、阅读理解、Issue Graph、多 Agent 执行、质量门禁、Memory、Git、服务器资源、部署、线上冒烟、监控和审计。控制台页面会同时包含静态项目画像、频繁变化的运行状态、大型图谱、日志流、diff viewer、表格和审批交互。

此前前端方向只确定为 React 控制台，未冻结具体框架和端口。

## 决策

Web Console 采用：

- `Next.js 16`
- `React`
- `TypeScript`
- App Router
- Cache Components
- 前端开发端口 `3000`
- Go/Gin API 后端端口 `8080`

前端工程建议放在 `apps/console/`。Next.js 只承载 Web Console、BFF/proxy 和视图层聚合；Go/Gin 后端仍是权威控制面。

## 理由

- App Router 与 Server Components 适合把项目画像、配置索引、Issue Graph 初始数据等放在服务端渲染。
- Cache Components 和 `use cache` 能让稳定数据缓存显式化，同时用 Suspense 承接运行中数据。
- Turbopack 能降低大型控制台开发反馈成本。
- `proxy.ts` 和 rewrites 能明确前端到 Go API 的网络边界。
- Next.js 16 的 DevTools MCP 有利于 Claude CLI、Codex 等 agent 辅助调试前端页面、路由和日志。
- 端口 `3000` 是 Next.js 常用默认端口，便于本地开发和团队认知。

## 后果

正向影响：

- 前端可以同时支持服务端数据获取、交互式图谱、日志流和复杂控制台布局。
- 后续生产部署可以选择 Node.js server 或容器化。
- 前端 agent 开发任务可以围绕明确的 App Router、Client Island、Server Component 边界拆 issue。

约束：

- 不能把业务状态机迁移到 Next.js。
- 不能在 Client Component 中直接持有敏感凭证。
- 高风险动作必须走 Go 后端鉴权、审批和审计。
- 所有 API contract 仍以后端和 `docs/contracts/` 为准。

## 替代方案

Vite SPA：

- 优点是轻量、启动快。
- 缺点是缺少服务端渲染、缓存边界、BFF/proxy 和长期控制台架构能力。

纯 Go server-rendered UI：

- 优点是部署简单。
- 缺点是不适合复杂图谱、diff viewer、日志流和高交互控制台。

