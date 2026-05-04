# Phase 5 Release Readiness

状态：completed
最后更新：2026-05-05

本文记录 Phase 5 鉴权强制门禁、Secret Resolver、真实外部 adapter preview/dry-run、部署检查和 Console 受控表单的收口结论。Phase 5 的执行图见 [Phase 5 实现 Issue Graph](./phase5-issue-graph.md)，实施记录见 [Phase 5 实施记录](./phase5-next-development-plan.md)。

## 1. 完成范围

- API authz middleware 已覆盖 provider refresh、approval decide、auth session/token/service account 写操作、deployment execute、visual render、resource renew/retire、git provider sync 和 PR/MR create。
- Secret Resolver 已支持 `env:` / `secret:` 引用、用途校验、脱敏注入和审计。
- GitHub/Gitee PR/MR adapter 已支持 preview、create guard、approval proof、secret resolver 和写开关。
- Deployment execution 已记录 smoke、monitor 和 rollback suggestion；生产真实执行仍默认阻断。
- Console 已支持审批、session、API token、service account、PR/MR、服务器维护的受控操作表单。

## 2. 验证命令

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
git diff --check
```

本轮 Phase 5 收口中，上述命令均已通过。

本地服务可访问性：

```bash
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:3000/
```

当前本机已有服务监听：

- Web Console：`127.0.0.1:3000`
- Go/Gin API：`127.0.0.1:8080`

## 3. 关键安全结论

- 裸 `approved: true` 不再足以触发 GitHub/Gitee PR/MR 真实创建；必须携带已批准且 target/action 匹配的 `approval_id`。
- API token 创建只在响应中返回一次 token value；Console 只展示短 preview，不把明文 token 写入列表、日志或文档。
- Secret 明文只在 adapter 执行时临时解析，不进入日志、Memory、prompt、runtime metadata 或 Console 列表。
- team_server 模式下，高风险写操作缺少有效 Bearer token 或 session 会被拒绝。

## 4. 剩余风险

- Deployment 仍未接入真实 SSH/云厂商发布；当前只允许受限 local shell 和本地 HTTP healthcheck。
- Provider probe 仍是轻量可达性检查，未接入完整服务商账单、额度和模型质量监控。
- Approval record 目前只校验，不做一次性消费、锁定或过期策略。
- Console 仍是单页工作台，后续需要按多项目、权限、发布和资源管理拆分路由。

## 5. 下一阶段入口

建议下一阶段聚焦真实生产适配前的执行可靠性和企业化治理：

- approval consumption、过期和重放防护。
- deployment SSH/云厂商 adapter 的 preview/dry-run/execute 三段式。
- CI/CD provider adapter 和 GitHub/Gitee release 发布。
- Console 多页面化、表单 schema 化和操作结果追踪。
- provider 成本、quota、健康和模型路由反馈闭环。
