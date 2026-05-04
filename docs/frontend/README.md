# 前端控制台文档

状态：planning-ready
责任角色：frontend_architect + product_designer + frontend
最后更新：2026-05-05

本目录定义 Moyuan Web Console 的前端技术方案。当前只做文档和实现约束，不代表前端工程已经创建。

## 1. 技术结论

- 前端框架：`Next.js 16 + React + TypeScript`。
- 前端端口：`3000`。
- 后端端口：`8080`。
- 前端定位：Moyuan Control Console，面向多 Agent 工程运维、代码生命周期、服务器资源和发布投产的工作台。
- 后端边界：Go/Gin API Server 仍是唯一核心控制面，Next.js 不接管主业务状态。
- 默认开发分工：前端实现继续优先交给 `frontend` role + `claude_cli`，后端继续优先交给 `backend` role + `codex_cli`。

端口约定：

```text
Frontend: http://127.0.0.1:3000
Backend:  http://127.0.0.1:8080
```

## 2. 文档入口

| 文档 | 作用 |
| --- | --- |
| [Next.js 16 控制台方案](./nextjs16-control-console.md) | 前端架构、渲染模式、数据访问、页面结构和质量策略 |

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

