# Bug 判断与自我修复策略

本文定义运行过程中如何判断一个异常是否为 bug、是否允许自动修复、何时升级人工，以及如何把修复经验转成长期能力。

功能回退后的 fix issue、hotfix 和回归测试要求见 [工程流程规范](../engineering-process-standards.md)。

## 1. 目标

- 避免把偶发环境问题误判为代码 bug。
- 避免把新需求伪装成自动修复。
- 自动处理低风险、可复现、写入范围明确的问题。
- 所有自动修复都可追踪、可回滚、可审查。
- 让修复结果反哺 Memory、测试策略、skills 和模型路由。

## 2. 输入事实

| 输入 | 说明 |
| --- | --- |
| `signal_type` | test_failure、runtime_error、smoke_failure、monitor_alert、user_feedback、review_finding |
| `source` | run、issue、commit、release、deployment、user、monitor |
| `reproducible` | 是否稳定复现 |
| `evidence_count` | 证据数量 |
| `affected_scope` | 文件、模块、接口、环境 |
| `recent_changes` | 相关 issue、commit、release |
| `expected_behavior` | 需求、测试、契约或用户确认 |
| `risk_level` | low、medium、high、critical |
| `environment` | local、test_dev、staging、production |
| `write_scope` | 修复允许写入范围 |
| `auth_context` | 当前 actor 和权限 |
| `history` | 历史 bug、quality report、Memory 命中 |

## 3. 决策结果

| 结果 | 含义 |
| --- | --- |
| `CONFIRMED_BUG` | 可确认为 bug |
| `NOT_BUG` | 不是 bug |
| `NEEDS_EVIDENCE` | 证据不足，需要复现或人工判断 |
| `ENHANCEMENT_CANDIDATE` | 更像增强需求，不自动修复 |
| `AUTO_REPAIR_ALLOWED` | 可以自动修复 |
| `REPAIR_ISSUE_ONLY` | 只创建修复 issue |
| `REQUIRE_APPROVAL` | 需要审批 |
| `BLOCKED` | 禁止自动修复 |

## 4. Bug 分类树

```text
if signal is stable test/build/typecheck/lint failure:
  CONFIRMED_BUG

else if signal is smoke failure and reproduces in staging or test_dev:
  CONFIRMED_BUG

else if user_feedback conflicts with accepted requirement or contract:
  CONFIRMED_BUG

else if reviewer finding has concrete file/line/root cause:
  CONFIRMED_BUG

else if signal is one-time external timeout:
  NOT_BUG or NEEDS_EVIDENCE

else if signal requires new behavior not in requirement:
  ENHANCEMENT_CANDIDATE

else if environment, credential or dependency service is missing:
  NOT_BUG with environment_issue

else:
  NEEDS_EVIDENCE
```

## 5. 自动修复决策树

```text
if auth_context invalid:
  BLOCKED

if classification != CONFIRMED_BUG:
  do not auto repair

if environment == production:
  REQUIRE_APPROVAL

if risk_level in high/critical:
  REQUIRE_APPROVAL

if affected_scope includes auth/security/payment/migration/public_api:
  REQUIRE_APPROVAL

if write_scope is empty or crosses protected paths:
  BLOCKED

if reproduction command is missing:
  REPAIR_ISSUE_ONLY

if similar repair failed repeatedly:
  REPAIR_ISSUE_ONLY or REQUIRE_APPROVAL

if bug is low risk and has regression test path:
  AUTO_REPAIR_ALLOWED

else:
  REPAIR_ISSUE_ONLY
```

## 6. 修复执行规则

自动修复必须：

- 创建 repair attempt。
- 创建隔离分支或 issue worktree。
- 只做最小修改。
- 优先补充失败测试或回归测试。
- 运行复现命令。
- 运行相关质量门禁。
- 经过 reviewer 或 quality_guard 独立判断。
- 写入 run、quality、error、memory 和 audit 事件。

自动修复禁止：

- 直接修改生产服务器代码。
- 删除失败测试来让结果变绿。
- 降低质量门禁阈值。
- 忽略 failing test。
- 扩大写入范围。
- 新增依赖来掩盖根因。
- 在未确认 bug 时直接修改代码。

## 7. Enhancement 判断树

```text
if feedback asks for new behavior:
  ENHANCEMENT_CANDIDATE

if current behavior matches accepted requirement but user wants better UX/performance:
  ENHANCEMENT_CANDIDATE

if monitor shows non-blocking performance trend:
  ENHANCEMENT_CANDIDATE or tuning_issue

if enhancement is low-risk configuration or documentation improvement:
  create improvement candidate
else:
  create normal epic or issue
```

增强候选不会直接进入自动修复。它必须进入需求规划或 tuning issue，由 Issue Graph 管理依赖和风险。

## 8. 学习与能力增强树

```text
if repair succeeded and review accepted:
  record bug_signature
  record root_cause
  record fix_pattern
  record regression_test

if same bug pattern appears >= threshold:
  suggest new quality rule or test template

if bug tied to module boundary:
  update module map candidate

if bug tied to missing skill:
  recommend skill binding

if runtime/model repeatedly fails same class:
  adjust routing recommendation

if memory confidence low:
  keep as candidate and require curator review
```

## 9. 阻断条件

必须阻断：

- 鉴权失败。
- 无法确定是否 bug 且存在写入风险。
- 需要访问 secret 明文才能修复。
- 修复可能影响生产数据。
- 修复需要破坏性 Git 操作。
- 自动修复超过轮次上限。
- Reviewer 给出 blocker。

## 10. 产物和日志

每次判断必须写入：

- bug candidate。
- classification result。
- repair decision。
- blocked reason，如适用。

自动修复必须写入：

- repair plan。
- repair attempt result。
- quality report。
- review result。
- memory candidates。

## 11. 验收用例

- 稳定单测失败被分类为 `CONFIRMED_BUG`。
- 用户提出新需求被分类为 `ENHANCEMENT_CANDIDATE`，不会自动改代码。
- 生产环境错误默认需要审批。
- 低风险 bug 自动修复后必须新增或更新回归测试。
- 自动修复失败超过上限后停止，生成人工处理 issue。
- 成功修复后类似 bug 再出现时能检索到历史 fix pattern。
