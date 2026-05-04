# 运行反馈与自我修复主线

本文定义 Moyuan Code 在项目运行、测试、部署、用户反馈和持续使用过程中，如何自动发现疑似 bug、判断是否真是 bug、决定是否允许自动修复，并把修复经验沉淀为长期能力。

本主线不替代代码质量门禁。质量门禁负责阻止坏代码合入；本主线负责在运行和使用过程中持续发现问题、闭环修复和提升系统能力。

## 1. 目标

- 自动收集运行失败、测试失败、日志异常、线上冒烟失败、用户反馈和重复质量问题。
- 判断信号是 confirmed bug、非 bug、环境问题、偶发失败还是 enhancement candidate。
- 对低风险、可复现、写入范围明确的问题自动创建修复任务并执行最小修复。
- 自动补充回归测试，防止同类 bug 反复出现。
- 将 bug signature、root cause、fix pattern、回归测试和成功修复经验写入 Memory。
- 让项目越使用，项目理解、测试策略、质量规则、Agent 分工和 skills 推荐越准确。

## 2. 边界

本主线负责：

- 运行信号采集和归一化。
- bug candidate 生成。
- bug/非 bug/enhancement 分类。
- 自动修复策略。
- 修复验证和质量复核。
- 修复经验沉淀。
- 能力增强建议。

本主线不负责：

- 未经确认的产品需求扩展。
- 生产环境直接热改代码。
- 绕过 Issue Graph 的大范围重构。
- 替代人工事故处理。
- 替代被管理项目自身的业务监控系统。

## 3. 输入信号

| 信号 | 来源 | 典型例子 |
| --- | --- | --- |
| test_failure | test、lint、build、typecheck、benchmark | 单测失败、构建失败、类型错误 |
| runtime_error | CLI、server log、agent run、deployment log | exception、crash、exit code 非 0 |
| smoke_failure | 发布投产主线 | 健康检查失败、关键接口返回异常 |
| monitor_alert | 生产监控或观测系统 | 错误率升高、延迟异常、资源耗尽 |
| user_feedback | 用户反馈 | “刚才生成的接口跑不通” |
| review_finding | reviewer、quality_guard、安全审计 | 发现 bug、回归风险、测试缺口 |
| repeated_pattern | Memory 和历史质量报告 | 同一模块多次出现同类错误 |

## 4. 端到端流程

```text
collect runtime signal
  -> normalize signal
  -> correlate run / issue / commit / release / deployment
  -> classify bug candidate
  -> reproduce or gather evidence
  -> confirmed_bug / not_bug / needs_evidence / enhancement_candidate
  -> decide auto repair / create issue / require approval / ignore
  -> create repair branch or issue worktree
  -> apply minimal fix + regression test
  -> run quality gates + verification + review
  -> merge if accepted
  -> record bug signature and fix pattern
  -> update memory / project comprehension / skills recommendation
```

## 5. Bug 判断标准

可判定为 confirmed bug：

- 测试、构建、类型检查或 lint 稳定失败。
- 运行错误可以关联到当前代码路径、commit、issue 或 release。
- 线上冒烟失败能稳定复现。
- 用户反馈与已确认需求、接口契约或验收标准冲突。
- Reviewer 发现确定性逻辑错误，且能给出触发条件。
- 同一错误 signature 在短时间窗口内重复出现。

不能直接判定为 bug：

- 只有一次无法复现的外部服务超时。
- 用户提出的是新需求或体验增强。
- 环境变量、凭证、依赖服务不可用导致的运行失败。
- 测试本身不稳定且无法确认业务代码错误。
- 监控告警缺少关联 commit、release 或可操作证据。

需要更多证据：

- 错误只出现在生产但无法复现。
- 日志被脱敏后缺少关键上下文。
- 涉及并发、性能、数据迁移或第三方服务。
- 修复可能改变公共 API、权限、安全、支付或数据一致性。

## 6. 自动修复模式

| 模式 | 说明 | 适用条件 |
| --- | --- | --- |
| observe_only | 只记录信号，不自动创建修复 | 新接入项目、规则未稳定 |
| candidate_only | 只创建 bug candidate，等待确认 | 证据不足或风险较高 |
| issue_only | 自动创建修复 issue，不自动改代码 | 需要人工排期或跨模块 |
| auto_repair_low_risk | 自动修复低风险 bug | 可复现、写入范围明确、有测试命令 |
| require_approval | 修复前必须审批 | 生产、权限、安全、迁移、支付、公共 API |

自动修复必须满足：

- 有 `auth_context`。
- 有可追踪 bug candidate。
- 有最小写入范围。
- 有复现步骤或验证命令。
- 有修复上限。
- 修复后必须经过质量门禁和独立 review。

## 7. 自我增强机制

每次 confirmed bug 或成功修复后，系统要评估是否生成能力增强建议：

- 更新项目理解中的风险模块。
- 给模块地图增加容易出错的边界说明。
- 增加或调整测试策略。
- 增加质量门禁规则或阈值建议。
- 推荐更合适的 skill。
- 调整 Agent role 组合，例如为某类任务追加 tester、security 或 architect。
- 调整模型路由，例如同类 bug 修复优先使用更适合的 Runtime。
- 写入 Memory 的 bug signature、root cause、fix pattern 和 regression test。

能力增强不能自动扩大权限，也不能自动放宽质量门禁。降低门禁、忽略测试、扩大写入范围必须走审批。

## 8. 产物

| 产物 | 位置 | 说明 |
| --- | --- | --- |
| runtime signal | `.moyuan/lifecycle/signals/` | 归一化后的运行信号 |
| bug candidate | `.moyuan/lifecycle/bug-candidates/` | 疑似 bug 及证据 |
| repair attempt | `.moyuan/lifecycle/repair-attempts/` | 自动修复计划、执行和结果 |
| improvement record | `.moyuan/lifecycle/improvements/` | 能力增强建议和应用结果 |
| quality report | `.moyuan/lifecycle/quality/` | 修复后的质量门禁结果 |
| memory candidate | `.moyuan/memory/candidates/` | bug 模式、修复经验和回归测试经验 |

## 9. 关联策略

必须调用：

- [Bug 判断与自我修复策略](../policies/bug-detection-self-repair-policy.md)
- [质量与合入策略](../policies/quality-merge-policy.md)
- [Issue 调度策略](../policies/issue-scheduling-policy.md)
- [Memory 决策策略](../policies/memory-decision-policy.md)
- [鉴权与访问控制策略](../policies/auth-access-policy.md)

实现接口见 [自我修复契约](../contracts/self-repair-contract.md)。

## 10. 阻断条件

必须阻断自动修复：

- 无法建立 `auth_context`。
- 无法判断是不是 bug。
- 缺少可复现步骤且风险不低。
- 写入范围跨越多个核心模块。
- 涉及生产数据、权限、安全、支付、数据库迁移或公共 API。
- 修复需要删除大量代码或新增依赖。
- 修复会降低测试、质量门禁或安全规则。
- 达到自动修复轮次上限。

## 11. 验收标准

- 运行失败能生成结构化 runtime signal。
- 稳定测试失败能自动生成 confirmed bug candidate。
- 非 bug 或 enhancement 能被区分，不直接改代码。
- 低风险 confirmed bug 能自动创建 repair attempt，并补充回归测试。
- 自动修复后的代码必须通过质量门禁和独立 review。
- 修复经验能进入 Memory，并在后续类似任务中被检索。
- runtime execution 和 quality gate 结果能反哺 provider telemetry，供后续路由参考。
- 同类 bug 重复出现时，系统能提升风险等级或建议新增质量规则。
