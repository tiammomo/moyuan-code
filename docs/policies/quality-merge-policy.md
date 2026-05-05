# 质量与合入策略

## 1. 目标

决定 AI 生成的代码是否可接受、是否需要返工、是否允许合入 integration branch，以及什么时候阻断下游 issue。

覆盖率阈值、豁免和回退后 fix 验收由 [工程流程规范](../engineering-process-standards.md) 维护。

## 2. 输入事实

- issue acceptance criteria。
- subagent results。
- changed files。
- command results。
- test/lint/build/typecheck results。
- coverage report。
- runtime signals。
- repair attempt，如适用。
- quality report。
- review findings。
- style constraints。
- architecture boundaries。
- risk level。
- rework count。

## 3. 决策结果

- `QUALITY_PASSED`
- `QUALITY_FAILED`
- `COVERAGE_FAILED`
- `REVIEW_ACCEPTED`
- `REVIEW_REJECTED`
- `NEEDS_REWORK`
- `MERGE_ALLOWED`
- `MERGE_BLOCKED`
- `ESCALATE_TO_HUMAN`

## 4. 质量门禁决策树

```text
if runnable gate failed:
  QUALITY_FAILED(blocker)
else if build/typecheck/lint required and failed:
  QUALITY_FAILED(blocker)
else if tests required and failed:
  QUALITY_FAILED(blocker)
else if coverage gate failed:
  COVERAGE_FAILED(blocker)
else if test gap unacceptable:
  QUALITY_FAILED(blocker)
else if new duplicate ratio exceeds threshold:
  QUALITY_FAILED(high)
else if complexity exceeds threshold:
  QUALITY_FAILED(high)
else if architecture boundary violated:
  QUALITY_FAILED(blocker)
else if dependency/security risk high:
  QUALITY_FAILED(blocker)
else:
  QUALITY_PASSED
```

失败结果如果形成稳定 runtime signal，应交给 [Bug 判断与自我修复策略](./bug-detection-self-repair-policy.md) 判断是否为 confirmed bug。质量策略本身不直接决定是否自动修复。

## 5. Review 决策树

```text
if review has blocker finding:
  REVIEW_REJECTED
else if review has unresolved high finding:
  REVIEW_REJECTED
else if reviewer requested tests and tests absent:
  REVIEW_REJECTED
else:
  REVIEW_ACCEPTED
```

## 6. 合入决策树

```text
if quality not passed:
  MERGE_BLOCKED
else if subagent output contract invalid:
  MERGE_BLOCKED
else if review not accepted:
  MERGE_BLOCKED
else if acceptance criteria incomplete:
  MERGE_BLOCKED
else if integration checks fail:
  MERGE_BLOCKED
else if unresolved write conflict exists:
  MERGE_BLOCKED
else:
  MERGE_ALLOWED
```

## 7. 返工策略

```text
if rework_count < max_rework_rounds:
  NEEDS_REWORK
else:
  ESCALATE_TO_HUMAN
```

返工必须保留：

- 失败门禁。
- review findings。
- 返工目标。
- 禁止重复尝试的错误路径。

## 8. 阻断条件

- blocker quality finding。
- high review finding 未解决。
- build/test/lint/typecheck 失败。
- 覆盖率低于阈值且无审批豁免。
- 违反项目架构边界。
- 新增重复代码超过阈值。
- 合并冲突未解决。

## 9. 产物和日志

产物：

- `.moyuan/lifecycle/quality/`
- `.moyuan/lifecycle/quality/coverage/`
- `.moyuan/lifecycle/signals/`，如果验证失败形成运行信号。
- `.moyuan/lifecycle/reviews/`
- `.moyuan/lifecycle/merge-reports/`

当前实现：

- 单 issue merge decision 写入 `.moyuan/lifecycle/reviews/merge-decisions/` 和 `.moyuan/lifecycle/reviews/merge-decisions.jsonl`。
- batch merge queue 写入 `.moyuan/lifecycle/merge-reports/queues/` 和 `.moyuan/lifecycle/merge-reports/merge-queues.jsonl`。
- merge queue 只生成 `ready_to_merge`、`needs_rework`、`blocked` 决策事实源，不执行真实 Git merge。

日志：

- `quality`
- `run`
- `agent`
- `git`
- `audit`
- `error`

## 10. 关联配置

- `.moyuan/policies/code-quality.yaml`
- [Bug 判断与自我修复策略](./bug-detection-self-repair-policy.md)
- [工程流程规范](../engineering-process-standards.md)
- `.moyuan/policies/orchestration.yaml`
- `.moyuan/policies/permissions.yaml`

## 11. 验收用例

- 测试失败不能合入。
- reviewer rejected 不能合入。
- quality_guard accepted 但 reviewer rejected 仍不能合入。
- 覆盖率低于阈值且无豁免时不能合入。
- 超过返工次数后升级人工确认。
- 合入后必须重跑 integration checks。
