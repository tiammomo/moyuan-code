# Phase 3 Release Readiness

状态：ready_for_next_phase
最后更新：2026-05-05

本文记录 Phase 3 第一批配置可执行化、Console 操作流、Provider 探测、Visual script 受控执行和 release/deploy 控制能力的收口验证。稳定设计结论已回写到配置、主线、策略、契约和 Console 文档。

## 1. 验证范围

已完成能力：

- `.moyuan/project.yaml`、`repository.yaml`、`policies/access.yaml` 可被读取、校验并阻断错误配置。
- `.moyuan/models/providers.yaml`、`routing.yaml`、`visuals/architecture-visuals.yaml`、`runtimes/agent-runtimes.yaml`、`policies/server-resources.yaml`、`environments.yaml`、`release.yaml`、`budget.yaml` 已纳入 workspace validator。
- Provider refresh 支持可选轻量 HTTP probe，默认不外呼，探测失败可解释且不落盘密钥。
- Visual script mode 通过 asset provider `auth_ref` 注入环境变量，记录 auth ref、注入 key、质量检查和预览索引，并脱敏 stdout/stderr。
- Console 已支持受控操作：Visual dry-run、Runtime artifact preview、Release suggest、Deployment dry-run 和 Resource health scan。
- Release/deploy/smoke/monitor 操作以 dry-run、审批和状态记录为默认边界。

不在本次收口内：

- 生产级分布式队列、多 worker 和跨机器并发执行。
- 真实外部 `find-skills` marketplace adapter。
- 真实云厂商账单、配额和成本 API。
- 真实 GitHub/Gitee/GitLab PR/MR 创建、合并和状态双向同步。
- 真实生产非 dry-run 部署、线上烟测和监控告警联动。
- `secret:` auth ref 的 secret manager 解析。
- 团队级多用户登录、RBAC 会话 UI 和组织级权限治理。

## 2. 验证命令

后端：

```bash
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
```

前端：

```bash
cd apps/console
npm run typecheck
npm run build
```

Git：

```bash
git diff --check
git status --short
```

## 3. 验证结论

当前验证结论：

- Phase 3 issue graph 中 `phase3-001` 到 `phase3-007` 第一批 issue 均为 `completed`。
- 配置 validator 已覆盖当前文档规划中的核心项目工作空间配置域。
- Console 操作流只调用后端受控 API，不在前端直接改写权威状态。
- Provider 和 Visual 的真实外呼入口均默认关闭或需要显式参数、审批、质量检查和审计记录。
- 当前 main 分支持续推送到 GitHub。

## 4. 新增运行入口

Workspace：

```bash
moyuan workspace validate
moyuan workspace doctor
```

Provider：

```bash
moyuan model provider refresh --probe
moyuan model provider refresh --provider <provider-id> --probe --approved --probe-timeout-ms 1500
```

Phase 4 起，`--probe` 未带 `--approved` 时不会外呼上游，会生成 approval record。

Visual：

```bash
moyuan visuals asset render <asset-id> --mode dry_run
moyuan visuals asset render <asset-id> --mode script --approved
moyuan visuals renders
```

Console 受控动作：

- Visual Assets：`Dry Run`。
- Runtime Recoveries：`Artifacts`。
- Deployment Executions：`Suggest Release`、`Dry Run`、`Health Scan`。

## 5. 产物位置

- `.moyuan/logs/`
- `.moyuan/models/providers.json`
- `.moyuan/visuals/executions/`
- `.moyuan/visuals/previews/index.jsonl`
- `.moyuan/runtimes/recoveries/`
- `.moyuan/releases/`
- `.moyuan/deployments/`
- `.moyuan/resources/`
- `.moyuan/state.db`

## 6. 剩余风险

- 当前 release/deploy 操作仍以计划、建议、dry-run 和状态记录为主，真实生产部署要在 Phase 4 后继续加审批、凭证、回滚和监控门禁。
- Provider probe 当前只支持 `env:` token 解析，`secret:` 引用需要 secret manager 后才能用于真实探测和脚本执行。
- Console 已具备操作入口，但审计查询、审批记录、用户会话和权限管理还没有形成完整团队工作台。
- GitHub/Gitee 目前只完成远程仓库接入和 main 推送工作流，PR/MR 创建、review 状态和发布分支同步需要后续 adapter。

## 7. 下一阶段入口

Phase 4 进入“团队协作与审计 / production hardening”：

1. 增加统一审计日志查询 API 和 Console Audit 面板。
2. 增加高风险操作 approval record store。
3. 增加本地团队模式的 session、API token 和 service account 基线。
4. 增加 GitHub/Gitee PR/MR plan 与状态同步 adapter。
5. 增加服务器资源续费、巡检、退役和维护计划。

执行入口见 [Phase 4 实现 Issue Graph](./phase4-issue-graph.md) 和 [Phase 4 实施记录](./phase4-next-development-plan.md)。
