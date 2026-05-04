# 代码生命周期质量门禁

## 1. 目标

AI 生成代码不能只以“生成完成”为结束条件。Moyuan 的代码生命周期必须保证每次代码生成后都经过自动验证和独立审核，尽量避免：

- 不可运行代码。
- 无测试或测试缺口明显的代码。
- 重复代码。
- 过度复杂代码。
- 过度抽象代码。
- 破坏已有架构边界的代码。
- 没有必要的新依赖。
- 只满足表面需求但无法长期维护的代码。

核心规则：任何代码生成任务必须先通过 Quality Gates，才能进入 `COMPLETED`。如果代码由 Subagent 生成，门禁对象是 Subagent 输出收敛后的 diff，而不是单个模型回复。

运行过程中发现的异常不直接等同于代码质量门禁失败。运行异常先进入 [运行反馈与自我修复主线](./mainlines/runtime-feedback-self-repair.md)，被判断为 confirmed bug 后，才创建修复任务或 Repair Attempt，并重新回到本文定义的质量门禁。

Commit、Issue、回退后 fix、发版要求和覆盖率阈值的完整规则见 [工程流程规范](./engineering-process-standards.md)。

## 2. 生命周期位置

质量门禁插入在 `IMPLEMENTATION` 之后、`REVIEW` 之前，并且与 `VERIFICATION` 形成闭环。

```text
IMPLEMENTATION
  -> QUALITY_CHECK
  -> VERIFICATION
  -> REVIEW
  -> ACCEPTED

任一阶段失败：
  -> NEEDS_REWORK
  -> IMPLEMENTATION
```

状态含义：

- `QUALITY_CHECK`：静态质量检查、复杂度、重复度、架构边界和测试缺口分析。
- `VERIFICATION`：运行测试、lint、build、类型检查和回归脚本。
- `REVIEW`：Review Agent 对 diff 做独立判断。
- `NEEDS_REWORK`：质量门禁失败，退回实现 Agent 修改。
- `ACCEPTED`：质量门禁和审核均通过，可以进入完成、提交或 PR/MR。

## 3. 必须执行的门禁

### 可运行性门禁

目标：确保生成代码至少能构建、能测试、能被项目工具链接受。

检查项：

- build 命令通过。
- lint 命令通过。
- typecheck 命令通过。
- 单元测试通过。
- 相关集成测试通过。
- 生成代码没有语法错误。

默认策略：

```yaml
quality_gates:
  runnable:
    required: true
    block_on_failure: true
    commands:
      build: auto_detect
      lint: auto_detect
      typecheck: auto_detect
      test: auto_detect
```

### 测试缺口门禁

目标：防止 AI 只改业务代码但不补测试。

检查项：

- 是否修改了核心逻辑但没有新增或更新测试。
- 是否覆盖成功路径、失败路径和边界条件。
- 是否缺少回归测试。
- 是否只添加了无断言测试。

默认策略：

```yaml
quality_gates:
  test_gap:
    required: true
    block_on_high_risk_gap: true
    require_tests_for:
      - business_logic
      - bugfix
      - public_api
      - data_migration
      - auth
      - payment
      - concurrency
```

### 覆盖率门禁

目标：防止新代码和修复代码缺少可量化测试保护。

默认阈值、changed files coverage、new code coverage、高风险模块阈值和豁免规则由 [工程流程规范](./engineering-process-standards.md) 维护。

默认策略：

```yaml
quality_gates:
  coverage:
    required: true
    block_on_regression: true
    thresholds_ref: .moyuan/policies/engineering.yaml#engineering.coverage.default_thresholds
```

### 重复代码门禁

目标：避免 AI 为了完成局部任务复制已有逻辑。

检查项：

- 新增代码是否与已有函数、类、模块高度相似。
- 是否重复实现已有工具函数。
- 是否复制了测试夹具、mock、schema、常量。
- 是否可以复用现有抽象而未复用。

默认策略：

```yaml
quality_gates:
  duplication:
    required: true
    block_on_new_duplicate: true
    thresholds_ref: .moyuan/policies/code-quality.yaml#gates.duplication
    ignore:
      - generated/**
      - snapshots/**
```

### 复杂度门禁

目标：避免 AI 生成难维护的大函数、大类和复杂控制流。

检查项：

- 函数圈复杂度。
- 单函数行数。
- 单文件新增行数。
- 嵌套层级。
- 分支数量。
- 参数数量。
- 是否引入不必要的抽象层。

默认策略：

```yaml
quality_gates:
  complexity:
    required: true
    block_on_regression: true
    thresholds_ref: .moyuan/policies/code-quality.yaml#gates.complexity
```

### 架构边界门禁

目标：防止 AI 为了快速实现而破坏项目边界。

检查项：

- 是否跨越未授权模块写文件。
- 是否违反现有分层架构。
- 是否绕过已有服务、仓储、权限或校验层。
- 是否引入全局状态或隐式副作用。
- 是否修改 protected paths。

默认策略：

```yaml
quality_gates:
  architecture:
    required: true
    block_on_boundary_violation: true
    require_design_review_for:
      - new_dependency
      - public_api_change
      - database_schema_change
      - cross_module_change
```

### 依赖和安全门禁

目标：避免不必要依赖、敏感信息泄露和明显安全风险。

检查项：

- 是否新增依赖。
- 新依赖是否有替代的项目内能力。
- 是否读写密钥文件。
- 是否引入命令注入、SQL 注入、XSS、权限绕过等风险。
- 是否把敏感信息写入代码或测试。

默认策略：

```yaml
quality_gates:
  dependency_security:
    required: true
    block_on_high_severity: true
    require_approval_for_new_dependency: true
    secret_scan: true
```

## 4. 审核 Agent

每次代码生成后至少需要两个独立角色参与：

- Implementer Agent：负责实现，例如 `backend`、`frontend`、`backend_tuning`。
- Reviewer Agent：负责审查，不允许由同一个 Agent 自审后直接通过。

实现和审核应由不同 Subagent 实例承担。即使它们使用同一个 Runtime 后端，也必须有不同 role、不同输出契约和独立 run 记录。

推荐角色：

| Role | 职责 |
| --- | --- |
| tester | 生成测试计划、运行测试、判断测试缺口 |
| reviewer | 审查 diff、识别 bug、回归风险和维护性问题 |
| quality_guard | 检查重复代码、复杂度、过度抽象和代码异味 |
| security | 检查安全风险、依赖风险和敏感信息 |
| architect | 对跨模块、公共 API、数据库迁移做设计复核 |

任务复杂度较低时，可以只启用 `tester + reviewer`。涉及核心逻辑、跨模块、性能、安全或数据库变更时，必须追加 `quality_guard`、`security` 或 `architect`。

## 5. 质量门禁配置

`.moyuan/policies/code-quality.yaml`：

```yaml
schema_version: 1

quality:
  enabled: true
  required_for_all_code_tasks: true
  fail_task_on_blocker: true
  max_rework_rounds: 3
  require_independent_review: true
  allow_self_review: false

gates:
  runnable:
    enabled: true
    severity: blocker
  test_gap:
    enabled: true
    severity: blocker
  coverage:
    enabled: true
    severity: blocker
    thresholds_ref: .moyuan/policies/engineering.yaml#engineering.coverage.default_thresholds
  duplication:
    enabled: true
    severity: blocker
    thresholds_ref: .moyuan/policies/code-quality.yaml#gates.duplication
  complexity:
    enabled: true
    severity: blocker
    thresholds_ref: .moyuan/policies/code-quality.yaml#gates.complexity
  architecture:
    enabled: true
    severity: blocker
  dependency_security:
    enabled: true
    severity: blocker

review:
  required_reviewers:
    - reviewer
  required_for_high_risk:
    - quality_guard
    - security
    - architect
  high_risk_triggers:
    - public_api_change
    - database_schema_change
    - auth_change
    - payment_change
    - concurrency_change
    - new_dependency
    - cross_module_change

reports:
  write_quality_report: true
  report_path: .moyuan/lifecycle/quality

self_repair:
  enabled: true
  mode: auto_repair_low_risk
  max_attempts_per_bug: 2
  require_regression_test: true
  auto_create_bug_candidate: true
  require_approval_for:
    - production
    - auth
    - security
    - payment
    - migration
    - public_api
```

## 6. 执行流程

```text
Agent writes code
  -> collect diff
  -> classify changed files
  -> run quality gates
  -> run tests/build/lint/typecheck
  -> capture runtime signals if verification fails
  -> reviewer reads diff + reports
  -> accepted or needs_rework
```

如果失败：

1. Orchestrator 将 task 状态设为 `NEEDS_REWORK`。
2. 失败原因写入 quality report。
3. Implementer Agent 只能针对失败项做最小修改。
4. 重新运行门禁。
5. 超过 `max_rework_rounds` 后停止自动修复，交给用户判断。

## 7. Review 输出契约

Reviewer Agent 必须输出结构化结论：

```yaml
status: accepted | needs_rework | rejected
summary: string
findings:
  - severity: blocker | high | medium | low
    category: correctness | duplication | complexity | architecture | testing | security | maintainability
    file: string
    line: number
    description: string
    recommendation: string
quality_score:
  correctness: 0-100
  maintainability: 0-100
  test_coverage: 0-100
  simplicity: 0-100
  architecture_fit: 0-100
decision:
  can_complete: boolean
  requires_user_approval: boolean
```

阻断规则：

- 存在 `blocker` finding：不能完成。
- 存在高风险测试缺口：不能完成。
- build/lint/test/typecheck 失败：不能完成。
- 复杂度或重复度超过阈值：不能完成。
- 修改 protected paths 未获批准：不能完成。

## 8. Run 记录

每次执行必须写入质量结果：

```json
{
  "quality_gates": {
    "status": "failed",
    "rework_round": 1,
    "checks": [
      {
        "name": "duplication",
        "status": "failed",
        "severity": "blocker",
        "summary": "新增代码重复实现了已有 auth helper"
      },
      {
        "name": "complexity",
        "status": "passed",
        "severity": "blocker"
      }
    ],
    "review_decision": "needs_rework",
    "report": ".moyuan/lifecycle/quality/run-20260503-001.md"
  }
}
```

## 9. 落地范围

CLI 和 Phase 以 [总体规划与生命周期路线图](./lifecycle-roadmap.md) 为唯一权威来源。本模块至少需要实现：

- 每次代码任务后自动采集 diff。
- 自动运行项目配置的 build/lint/test/typecheck。
- Reviewer Agent 审查 diff。
- 检查是否新增测试或更新测试。
- 检查简单重复代码和大函数。
- 将质量报告写入 `.moyuan/lifecycle/quality/`。
- 不通过时任务进入 `NEEDS_REWORK`。
- 后续如支持 suppress，必须记录原因、负责人和过期时间，避免质量门禁被永久绕过。
