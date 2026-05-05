# Phase 14 Release Readiness

状态：ready
责任角色：release_owner + git_owner + devops_owner + backend_owner + frontend_owner + security_owner + qa_owner
最后更新：2026-05-05

Phase 14 已完成“受控远程发布与部署执行”的第一批能力。系统已经可以从 release candidate 触发 release provider publish gate、生成稳定 PR/MR plan、衔接 deployment execution、聚合 smoke/monitor/rollback feedback，并在 Console 展示整条执行流水线。

## 1. 完成范围

- `phase14-001 release-candidate-provider-publish-bridge`：release candidate 可调用受控 provider publish，默认 approval required 或 preview-only，真实写入仍复用 `MOYUAN_ALLOW_RELEASE_PROVIDER_WRITE`、approval、secret resolver 和 replay guard。
- `phase14-002 release-candidate-pr-mr-create-bridge`：release candidate 可生成稳定 Git Provider PR/MR plan，真实 PR/MR create 继续走既有高风险 create endpoint、approval、`MOYUAN_ALLOW_GIT_PROVIDER_WRITE` 和 replay guard。
- `phase14-003 candidate-deployment-execution-bridge`：release candidate 可衔接 deployment execution，复用 dry-run、ssh preview、local shell、ssh execute 和生产阻断。
- `phase14-004 post-deploy-smoke-monitor-feedback`：candidate 可聚合 deployment post-deployment history，输出 smoke、monitor、rollback、failure class、severity 和 evidence 摘要。
- `phase14-005 console-release-execution-surface`：Console 已展示 provider publish、PR/MR plan、deployment execution 和 candidate feedback，并只调用后端受控 API。

## 2. 验证结论

最新收口门禁：

```bash
git diff --check
PATH="/tmp/moyuan-go-apt/usr/lib/go-1.22/bin:$PATH" go test ./...
cd apps/console && npm run typecheck
cd apps/console && npm run build
```

结论：通过。

最近提交：

```text
2d9ade3 feat: surface release execution flow
0b8b8cd feat: add candidate deployment feedback
034e295 feat: bridge candidate deployment execution
cfc90d2 feat: add candidate pr mr plan bridge
3408954 feat: bridge candidate provider publish
bef67f7 docs: open phase 14 release execution
```

## 3. 保留边界

- Provider publish 不默认执行远程写入；没有 approval、secret 和写开关时只给出阻断或 preview-only。
- PR/MR plan endpoint 不直接创建远程 PR/MR；真实创建仍必须走 `git-provider-plans/:plan_id/create`。
- Deployment execution 继续复用既有部署执行器；production real execution 仍默认阻断。
- Candidate feedback 只聚合已有 post-deployment history，不重复执行 smoke、monitor 或 rollback。
- Console 不执行 Git、Provider 或部署命令，不消费 approval，不自行计算发布、PR/MR 或部署 readiness。

## 4. 进入 Phase 15 的理由

Phase 14 已经把 release candidate 的发布和部署执行链路接通到受控 API 与 Console。下一阶段应聚焦执行门禁加固和生产运行质量：

- deployment execution 的 approval proof、approval consumption 和 replay guard 需要补齐到与 release provider/git provider 同级。
- rollback execution 仍停留在建议和 runbook，需要受控执行链路。
- production observability 可以从 feedback 聚合推进到持续 monitor、告警和自动修复建议。
- Console 需要进一步展示 approval queue、rollback runbook、monitor 趋势和生产风险摘要。

Phase 15 应聚焦“部署审批加固、回退执行与生产可观测性增强”，继续保持生产写入默认关闭。
