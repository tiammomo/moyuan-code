# 工程流程规范

本文定义 Moyuan Code 管理项目时必须遵守的 commit、issue、fix、release 和测试覆盖率规范。它是工程流程类规则的唯一详细入口。

## 1. 目标

- 让 AI 和 Subagent 生成的变更可以被追踪、复核、回退和发版。
- 防止 issue 粒度失控、commit 信息不可读、回退后修复无证据。
- 明确发版前必须满足的质量、覆盖率、回滚和审批要求。
- 让 GitHub/Gitee、PR/MR、release note、Memory 和审计日志能共享同一套结构化信息。

## 2. Commit 规范

### 2.1 提交格式

Commit message 使用 Conventional Commits 兼容格式：

```text
<type>(<scope>): <summary>

<body>

Refs: <issue-id>
Run: <run-id>
Quality: <quality-report-id>
```

允许的 `type`：

| type | 说明 |
| --- | --- |
| feat | 新功能 |
| fix | 缺陷修复 |
| perf | 性能优化 |
| refactor | 不改变行为的重构 |
| test | 测试新增或修改 |
| docs | 文档 |
| build | 构建、依赖、包管理 |
| ci | CI/CD |
| chore | 维护性变更 |
| revert | 回退提交 |
| hotfix | 紧急修复 |

`scope` 应使用模块、包、服务或主线名，例如 `auth`、`api`、`frontend`、`memory`、`release`。

### 2.2 必填信息

由 Moyuan 自动生成或建议的 commit 必须关联：

- issue id。
- run id。
- quality report id。
- changed files summary。
- 测试命令和结果。
- reviewer 结论。

### 2.3 禁止事项

禁止：

- 使用 `update`、`misc`、`fix stuff` 这类不可追踪 summary。
- 一个 commit 混合多个无关 issue。
- 提交未通过质量门禁的代码。
- 提交 secret、`.env` 明文、生产凭证或未脱敏日志。
- 通过删除测试、降低阈值或扩大权限来让 commit 通过。
- 在未经审批时自动 push、tag 或创建 PR/MR。

### 2.4 自动 commit 条件

只有同时满足以下条件时，系统才允许自动创建 commit：

- `allow_auto_commit = true`。
- Issue 已通过质量门禁和 review。
- worktree 干净，且 diff 完全属于 issue write scope。
- commit message 校验通过。
- 没有待处理用户澄清或审批。
- 没有 high/blocker review finding。

默认策略仍是建议 commit，不自动提交。

## 3. Issue 规范

### 3.1 Issue 最小字段

每个 Issue 必须包含：

- `id`
- `title`
- `type`
- `description`
- `clarified_requirement`
- `depends_on`
- `read_scopes`
- `write_scopes`
- `acceptance_criteria`
- `test_plan`
- `risk_level`
- `assigned_team`
- `subagent_plan`
- `quality_gate`
- `style_constraints`
- `rollback_or_fix_plan`

### 3.2 Issue 命名

标题格式：

```text
<动词><对象><目的或边界>
```

示例：

- `实现登录接口的 token 签发逻辑`
- `补齐订单取消流程的回归测试`
- `修复用户列表分页参数校验`
- `定义支付回调 API 契约`

禁止：

- `优化代码`
- `修一下 bug`
- `做后台`
- `前端页面`

### 3.3 Issue 粒度

一个 Issue 应该满足：

- 能在一个独立分支或 worktree 内完成。
- 有明确验收标准。
- 写入范围可限定。
- 能单独运行相关测试。
- 失败后能独立返工。

应该拆分的情况：

- 同时涉及前端、后端、数据库迁移和部署。
- 涉及公共 API 或数据模型变更。
- 需要先设计契约再实现。
- 写入范围跨多个核心模块。
- 测试基础和业务实现互相依赖。

不应该拆分的情况：

- 拆分后每个 issue 都无法独立验收。
- 强行把同一个函数的连续修改拆成多个并发 issue。
- 为了增加并发而制造无意义任务。

### 3.4 Issue 状态要求

`ready` 条件：

- hard dependencies 已满足。
- 必要契约已 accepted。
- 写入范围不冲突。
- Subagent plan 可创建。
- Runtime、skills、预算、worktree 和权限满足要求。
- 没有待处理澄清或审批。

`accepted` 条件：

- 验收标准满足。
- 质量门禁通过。
- 覆盖率门禁通过或豁免被批准。
- independent review accepted。
- changed files 与 write scope 一致。
- 必要 memory candidates 已产生。

## 4. 功能回退后的 Fix 规范

本文区分三类动作：

| 动作 | 含义 | 使用场景 |
| --- | --- | --- |
| revert | 回退已合入变更 | 变更导致严重问题，需要恢复上一稳定行为 |
| fix-forward | 在当前分支继续修复 | 问题范围明确，修复风险低 |
| hotfix | 紧急修复并独立发版 | 生产事故、安全问题、阻断发布 |

### 4.1 回退触发条件

必须考虑回退：

- 发布后冒烟失败。
- 监控窗口出现 critical alert。
- 核心路径不可用。
- 数据一致性或权限安全风险。
- 修复需要较长时间且已有稳定上一版本。

不能盲目回退：

- 回退会破坏数据库迁移后的兼容性。
- 回退会丢失用户数据。
- 回退依赖不可恢复的外部状态。
- 回退范围不清楚。

### 4.2 回退后修复流程

```text
rollback or revert completed
  -> create regression issue
  -> preserve incident evidence
  -> classify root cause
  -> create fix issue or hotfix issue
  -> reproduce failure
  -> add regression test
  -> implement minimal fix
  -> run quality gates and coverage gates
  -> independent review
  -> release as hotfix or next batch
  -> record fix pattern and lesson memory
```

### 4.3 Fix Issue 必填字段

回退后的 fix issue 必须包含：

- 原 release id、deployment id 或 commit id。
- 回退方式：`revert`、`rollback`、`manual mitigation`。
- 事故或失败证据。
- root cause 假设和确认状态。
- 复现步骤。
- 回归测试计划。
- 修复策略。
- 风险和回滚计划。

### 4.4 修复验收

回退后的 fix 只有满足以下条件才允许合入：

- 失败可以复现，或有充分证据解释为何无法复现。
- 已新增或更新回归测试。
- 覆盖率没有低于阈值。
- reviewer 明确确认不是重复引入同类问题。
- 发布策略确认是 hotfix、单独发版还是进入下一批次。

## 5. 发版要求

### 5.1 发版前置条件

Release branch 只能从 accepted integration branch 创建。

发版前必须满足：

- included issues 全部 accepted。
- excluded issues 明确记录原因。
- full quality gates passed。
- regression tests passed。
- coverage gates passed 或豁免已审批。
- release note 已生成。
- migration checklist 已完成或确认不涉及迁移。
- rollback plan 存在。
- GitHub/Gitee push、tag、PR/MR 策略明确。
- 生产发布有审批。

### 5.2 发版批次

默认建议：

- 低风险功能：累计 3-7 个 accepted issues。
- 中风险功能：累计 2-4 个 accepted issues。
- 高风险功能：单独 release branch。
- breaking API、数据库迁移、鉴权、安全、支付：单独发版。
- hotfix/security：不等待批次，立即进入 hotfix/release 流水线。

### 5.3 Release Note 必填

Release note 必须包含：

- version。
- release id。
- included issues。
- excluded issues。
- 变更摘要。
- 风险说明。
- migration/config 说明。
- 测试和覆盖率摘要。
- rollback plan。
- PR/MR 或 tag 链接。

### 5.4 禁止发版

禁止发版：

- integration branch 未 accepted。
- 存在 blocker/high quality finding。
- 回归测试失败。
- 覆盖率低于阈值且无审批豁免。
- release note 缺失。
- rollback plan 缺失。
- 生产资源不健康。
- 生产监控或冒烟缺失。
- 需要审批但审批缺失或过期。

## 6. 测试覆盖率要求

### 6.1 默认阈值

默认覆盖率门禁：

| 指标 | 默认阈值 | 阻断 |
| --- | --- | --- |
| line coverage | 80% | 是 |
| branch coverage | 70% | 是 |
| function coverage | 80% | 是 |
| statement coverage | 80% | 是 |
| changed files coverage | 85% | 是 |
| new code coverage | 85% | 是 |

高风险模块建议阈值：

| 模块类型 | line | branch | changed files |
| --- | --- | --- | --- |
| auth/security/payment | 90% | 85% | 90% |
| database migration | 85% | 80% | 90% |
| public API | 85% | 80% | 90% |
| self-repair/release/deploy | 85% | 80% | 90% |

### 6.2 覆盖率策略

必须阻断：

- 新增核心逻辑没有测试。
- bugfix 没有回归测试。
- changed files coverage 低于阈值。
- 覆盖率比 baseline 下降超过允许值。
- 测试失败但被跳过或删除。

允许豁免：

- 纯文档变更。
- 纯配置且有 schema 校验。
- 生成代码或快照文件。
- 无法稳定测试的外部系统集成，但必须有手工验证记录和 reviewer 批准。

### 6.3 覆盖率报告

Coverage Report 必须记录：

- baseline coverage。
- current coverage。
- diff coverage。
- changed files coverage。
- uncovered critical paths。
- 豁免项和审批。
- 关联 issue、run、quality report。

覆盖率报告落盘：

- `.moyuan/lifecycle/quality/coverage/`

## 7. 配置入口

工程流程规范由以下配置控制：

| 配置 | 作用 |
| --- | --- |
| `.moyuan/policies/git.yaml` 或 `repository.yaml.git` | commit、branch、push、PR/MR 策略 |
| `.moyuan/policies/orchestration.yaml` | issue 规范、Subagent 和并发策略 |
| `.moyuan/policies/code-quality.yaml` | 质量门禁、覆盖率和 review |
| `.moyuan/policies/release.yaml` | 发版批次、release branch、tag、PR/MR、部署和回滚 |

## 8. 验收标准

- commit message 能被机器校验。
- issue 能被机器判断 ready、blocked、accepted。
- 回退后自动生成 fix issue 或 hotfix issue。
- release branch 创建前能校验发版门禁。
- 覆盖率报告能阻断低质量变更。
- 所有豁免必须有审批记录和审计事件。
