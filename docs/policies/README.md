# 策略决策树

策略文档定义系统在关键节点如何做决策。主线文档描述流程，策略文档描述判断规则。

策略文档必须尽量接近可实现形态，后续应能转为：

- rules engine。
- state machine。
- runtime validator。
- policy evaluator。
- checklist gate。

## 策略文档格式

每份策略默认包含：

- 目标。
- 输入事实。
- 决策结果。
- 决策树。
- 阻断条件。
- 人工确认条件。
- 产物和日志。
- 关联配置。
- 验收用例。

## 策略列表

| 策略 | 文档 | 负责决策 |
| --- | --- | --- |
| 项目阅读理解策略 | [project-comprehension-policy.md](./project-comprehension-policy.md) | full/incremental/diff comprehension、stale memory |
| Issue 调度策略 | [issue-scheduling-policy.md](./issue-scheduling-policy.md) | 澄清、拆分、依赖、并发、等待队列、Runtime 分派 |
| 质量与合入策略 | [quality-merge-policy.md](./quality-merge-policy.md) | 质量门禁、返工、review、合入 integration branch |
| Git 分支策略 | [git-branch-policy.md](./git-branch-policy.md) | branch/worktree/PR/MR/用户改动保护 |
| 服务器资源策略 | [server-resource-policy.md](./server-resource-policy.md) | 服务器登记、资源组、到期、巡检、生产权限 |
| 发布投产策略 | [release-deployment-policy.md](./release-deployment-policy.md) | release batch、release branch、tag、deploy、smoke、rollback |
| Provider 路由策略 | [provider-routing-policy.md](./provider-routing-policy.md) | Claude/Codex/GPT/国产模型/第三方 API 路由和降级 |
| Memory 决策策略 | [memory-decision-policy.md](./memory-decision-policy.md) | record、retrieve、compact、stale、conflict |

## 策略优先级

当策略冲突时按以下顺序执行：

1. 权限和安全策略。
2. 用户改动保护。
3. 生产环境保护。
4. 质量和合入门禁。
5. 资源和预算限制。
6. Agent Runtime 和 Provider 路由。
7. 成本优化。

任何成本优化、并发优化或自动化操作都不能绕过权限、安全、质量和生产保护。
