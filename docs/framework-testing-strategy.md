# 框架自身测试策略

状态：planned
责任角色：quality_owner + core_engineer
最后更新：2026-05-03

本文定义 Moyuan Code 框架本体的测试策略。它不同于“被管理项目代码”的质量门禁，重点验证 Moyuan 自己的编排、状态、权限、Runtime Adapter、Memory、Git 和失败恢复是否可靠。

## 1. 目标

- 防止编排系统生成错误 Issue Graph、错误并发或错误合入。
- 防止 Native Runtime、模型 API、Git、Workspace 和 Memory 的边界失控。
- 让每个策略决策树和契约都有可执行测试覆盖。
- 支持后续重构和扩展服务商时保持行为稳定。

## 2. 边界

本文只维护 Moyuan 本体的测试方法。

不在本文展开：

- 被管理项目的 build、lint、test、coverage 阈值，见 [代码生命周期质量门禁](./code-lifecycle-quality-gates.md)。
- commit、issue、fix、release 规范，见 [工程流程规范](./engineering-process-standards.md)。
- Runtime Adapter 字段契约，见 [Runtime Adapter 契约](./contracts/runtime-adapter-contract.md)。

## 3. 测试分层

| 层级 | 目标 | 示例 |
| --- | --- | --- |
| Unit | 验证纯函数、策略和 schema | issue dependency validation、permission decision、memory record score |
| Contract | 验证模块接口和错误类型 | Runtime Adapter、Subagent output、logging event、workspace migration |
| Integration | 验证多个模块协同 | Issue Graph -> Scheduler -> Subagent Plan -> Quality Gate |
| E2E | 验证完整开发闭环 | fake repo 中提交需求，生成 issue，执行 fake runtime，合入 integration branch |
| Recovery | 验证崩溃和失败恢复 | lock conflict、partial write、runtime timeout、merge conflict |
| Regression | 防止历史 bug 回归 | known bad issue graph、bad memory record、bad review acceptance |

## 4. 测试替身

必须提供以下测试替身，避免测试依赖真实模型、真实仓库和真实服务器。

| 测试替身 | 用途 |
| --- | --- |
| fake repo | 模拟不同语言、框架、dirty worktree、merge conflict |
| fake Claude CLI | 返回可控 diff、失败、超时、无效输出契约 |
| fake Codex CLI | 返回可控代码修改、review finding、测试报告 |
| fake model provider | 模拟 GPT、Claude、GLM、MiniMax、第三方 API 响应和限流 |
| fake git remote | 模拟 GitHub/Gitee push、PR/MR、认证失败、权限不足 |
| fake server resource | 模拟 test_dev、production、SSH 失败、健康检查失败 |
| fake clock | 验证 token 过期、服务器到期、retry backoff、memory compact 周期 |
| fake command runner | 验证 build/lint/test 成功、失败、超时和输出截断 |

## 5. Golden Fixtures

需要维护一组 golden fixtures，作为行为稳定基线。

```text
fixtures/
  repos/
    node-api/
    react-app/
    monorepo/
    dirty-worktree/
  issue-graphs/
    serial-contract-first.yaml
    frontend-backend-parallel.yaml
    write-conflict-blocked.yaml
    review-rework-loop.yaml
  runtime-outputs/
    valid-subagent-output.json
    invalid-output-contract.json
    timeout.json
  memory/
    record-yes.json
    record-no.json
    compact-collision.json
  quality/
    duplicate-code-fail.json
    coverage-fail.json
    review-rejected.json
```

Golden fixture 变更规则：

- 必须说明行为变化原因。
- 必须关联设计文档或 ADR。
- 不能为了让测试通过而静默改 expected。

## 6. 必测场景

### 项目接入

- 本地路径接入后初始化 `.moyuan/`。
- 远程仓库 clone 后生成 `repository.yaml`。
- 首次接入触发 full comprehension。
- 拉取远程分支后触发 incremental/diff comprehension。
- dirty worktree 阻断危险操作。

### 需求到 Issue Graph

- 需求信息充足时不追问。
- 需求不可验证时进入 clarification。
- 后端契约 issue 阻塞后端实现和前端集成。
- 写入范围冲突导致串行。
- 用户可见 issue graph 包含 blocked reason。

### Multi-Agent 执行

- frontend issue 默认路由 Claude CLI。
- backend、test、review 默认路由 Codex CLI。
- Runtime 输出不符合契约时不能 accepted。
- Subagent 不能自行创建无限子任务。
- 权限不足时不创建 Subagent。

### 质量和合入

- build/lint/test 任一失败时不能 merge。
- reviewer rejected 时进入 needs_rework。
- duplicate code 和 complexity 超阈值时阻断。
- accepted issue 才能合入 epic integration branch。
- 下游 issue 只有依赖 accepted 后解锁。

### Memory

- 一次性操作指令不进入长期记忆。
- 经用户明确偏好或稳定项目事实可以进入 memory candidates。
- staging dedup 合并重复记录。
- compact 后保留来源、时间和审计。
- 低置信或敏感内容不写入长期 Memory。

### Git 和发布

- merge conflict 进入 blocked 或 needs_rework。
- push 到 GitHub/Gitee 需要正确权限。
- release branch、tag、PR/MR 有审计事件。
- 生产部署前必须有审批和资源组。

### 自我修复

- 测试失败先成为 Runtime Signal。
- 证据不足时不能自动修复。
- confirmed bug 且低风险才允许 repair attempt。
- repair 后必须回归测试和 review。
- 修复经验进入 memory candidate 或 improvement record。

## 7. 测试数据安全

- fixture 不能包含真实 token、API key、SSH key、私网 IP 或生产域名。
- prompt 和模型响应 fixture 必须脱敏。
- 测试日志默认保存在临时目录。
- golden fixture 不记录完整真实项目源码，只使用最小可复现样例。

## 8. CI 门禁

Moyuan 本体 CI 至少包含：

```text
format
lint
typecheck
unit
contract
integration
fixture-regression
secret-scan
docs-link-check
```

进入发布候选前增加：

```text
e2e-fake-runtime
workspace-recovery
git-branch-flow
memory-compact-regression
self-repair-regression
release-dry-run
```

## 9. 覆盖率要求

最低要求：

- 策略纯函数和 schema validator：接近全覆盖。
- Orchestrator、Scheduler、Workspace、Runtime Adapter：重点路径必须覆盖。
- 错误路径、权限拒绝、失败恢复、日志审计必须有测试。
- CLI 薄封装可以低覆盖，但必须有端到端 smoke。

禁止把覆盖率作为唯一质量判断。对 Orchestrator、Scheduler 和 Workspace，失败路径覆盖比行覆盖率更重要。

## 10. 验收标准

进入实现前，测试策略必须能回答：

- 如何不调用真实模型也能测试多 Agent 编排。
- 如何不连接真实 GitHub/Gitee 也能测试分支和 PR/MR 流程。
- 如何验证质量门禁不会被绕过。
- 如何验证崩溃后不会留下损坏 `.moyuan/` 状态。
- 如何把历史失败沉淀为 regression fixture。

## 11. 相关文档

- [实现模块拆分](./implementation-module-map.md)
- [持久化与并发一致性](./persistence-concurrency-consistency.md)
- [代码生命周期质量门禁](./code-lifecycle-quality-gates.md)
- [日志与审计事件契约](./contracts/logging-audit-event-contract.md)
- [设计就绪门禁](./design-readiness-checklist.md)
