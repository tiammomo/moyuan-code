# 设计就绪门禁

本文定义 Moyuan Code 从文档规划进入代码实现前必须满足的设计就绪标准。未通过本文门禁前，不进入正式开发。

## 目标

- 降低后续实现中的返工和架构漂移。
- 防止核心概念、数据对象、权限和失败恢复在编码阶段临时补。
- 保证每个能力都有权威文档、配置入口、数据对象和验收标准。
- 保证后续 issue 拆分可以直接依据文档执行。

## 门禁结论

设计评审只能输出三类结论：

| 结论 | 含义 |
| --- | --- |
| `READY` | 可以进入实现拆分 |
| `READY_WITH_RISKS` | 可以进入有限实现，但必须记录风险和补齐计划 |
| `NOT_READY` | 不允许进入实现 |

默认策略：核心链路任一项为 `NOT_READY`，整体即为 `NOT_READY`。

## 必须通过的文档清单

| 文档 | 必须回答的问题 | 通过标准 |
| --- | --- | --- |
| [README](./README.md) | 读者如何进入文档体系 | 文档索引完整，核心原则明确 |
| [基础规范](./foundations/README.md) | 术语、对象、用户鉴权、权限、失败恢复是否统一 | 基础规范存在且互相不冲突 |
| [总体规划与生命周期路线图](./lifecycle-roadmap.md) | MVP、Phase、CLI 是否清楚 | CLI 只在此维护，Phase 验收可执行 |
| [参考架构](./reference-architecture.md) | 系统模块和状态机是否清楚 | 模块职责、状态、上下文链路明确 |
| [主线文档](./mainlines/README.md) | 端到端流程是否按真实生命周期组织 | 7 条主线覆盖平台用户与访问控制、接入、需求规划、开发、代码管理、服务器资源和发布投产 |
| [策略决策树](./policies/README.md) | 关键判断是否可转成实现策略 | 决策树、阻断条件、人工确认和日志要求明确 |
| [契约文档](./contracts/README.md) | 实现接口、错误、日志和迁移契约是否明确 | auth、schema、runtime、logging、workspace migration 契约存在 |
| [项目工作空间规范](./project-workspace-spec.md) | `.moyuan/` 目录和 schema 索引是否清楚 | 每个目录都有职责和权威文档 |
| [完整配置方案](./configuration-guide.md) | 项目运行需要哪些配置 | 核心 YAML 都有示例和校验清单 |
| [配置 Schema 规则](./configuration-schema-spec.md) | 配置字段哪些必填、可选、可为空、必须为空 | 核心 YAML 字段规则明确 |
| [Issues 编排与并发调度](./issue-orchestration.md) | 任务如何拆分、依赖和并发 | Issue Graph、ready queue、合入门禁明确 |
| [仓库接入、Git 与项目理解](./repository-onboarding-git-management.md) | 项目如何接入和理解 | 本地/远程接入、full/incremental comprehension 明确 |
| [代码生命周期质量门禁](./code-lifecycle-quality-gates.md) | AI 代码如何避免垃圾代码 | 测试、重复、复杂度、架构、安全、review 明确 |
| [Agent、Skills 与编排](./agent-skills-memory.md) | 多 Agent 如何分工 | role、team、skill、输出契约明确 |
| [Agent Memory 系统方案](./agent-memory-system.md) | Memory 如何记录、检索、整理 | record、retrieve、compact、审计明确 |
| [模型与工具适配规划](./model-tool-adapters.md) | 外部能力如何接入 | Provider、Runtime、Adapter、Image、错误分类明确 |

## 核心链路门禁

### 0. 主线和策略链路

必须明确：

- 平台用户与访问控制主线。
- 项目接入与阅读理解主线。
- 需求规划与 Issue 编排主线。
- 代码开发主线。
- 代码管理主线。
- 服务器资源管理主线。
- DevOps 发布投产主线。
- 每条主线引用的策略决策树。
- 策略的输入事实、决策结果、阻断条件和人工确认条件。
- 契约文档覆盖 auth、schema、runtime、logging 和 workspace migration。

通过标准：

- 可以从主线文档直接拆出端到端实现 issue。
- 可以从策略文档直接实现规则引擎、状态机或 runtime validator。
- 可以从契约文档直接定义 TypeScript interface、JSON Schema、日志事件和测试用例。

### 0.1 用户与鉴权链路

必须明确：

- local_single_user 和 team_server 两种运行模式。
- User、Organization、Membership、Service Account、API Token、Auth Session 和 Approval 对象。
- 任意命令如何生成 `auth_context`。
- 会话、Token、成员关系和项目角色如何参与鉴权。
- 高风险操作什么时候 `REQUIRE_APPROVAL`。
- 审批、拒绝、过期和取消如何写入审计。
- Token 和 Secret 明文不进入配置、日志、Memory 或图像 prompt。

通过标准：

- 任意主线执行前都能先判断 actor 是否有效。
- 用户禁用、会话过期、Token 撤销后不能继续执行写入、Git、服务器、发布或部署操作。
- 高风险操作不能绕过审批直接执行。

### 1. 项目接入链路

必须明确：

- 本地路径接入。
- 远程 Git 接入。
- GitHub/Gitee/GitLab/generic git provider 边界。
- GitHub 连接配置的必填、可选、可为空字段。
- GitHub token 或 SSH key 的权限和 secret 引用策略。
- 初始化 `.moyuan/`。
- 首次 full comprehension。
- 拉取远程分支后的 incremental comprehension。

通过标准：

- 能从文档直接拆出 CLI、schema、Git Adapter 和 comprehension 的实现 issue。

### 2. 需求到 Issue Graph 链路

必须明确：

- 需求丰富。
- 意图澄清。
- Issue 拆分。
- 依赖类型。
- Issue Graph。
- ready queue。
- 并发度计算。
- 用户可见计划。

通过标准：

- 能处理有前置依赖、可并发 issue 和阻塞 issue 的复杂任务。

### 3. 多 Agent 执行链路

必须明确：

- Agent role。
- Agent team。
- Claude CLI / Codex CLI Native Runtime。
- 普通模型 API Provider。
- Runtime 会话、输出、diff、失败降级。
- 权限继承。
- `auth_context`。

通过标准：

- 一个 issue 能被分配到明确 Agent 和 Runtime，并受鉴权、权限和质量门禁控制。

### 4. 代码质量链路

必须明确：

- 可运行性检查。
- 测试缺口。
- 重复代码。
- 复杂度。
- 架构边界。
- 依赖和安全。
- Reviewer 独立审核。
- 失败返工。

通过标准：

- AI 生成代码不能绕过质量门禁进入 accepted 或 merged。

### 5. Git 和合入链路

必须明确：

- task branch。
- issue worktree。
- epic integration branch。
- merge gate。
- dirty worktree 保护。
- 用户改动保护。
- push / PR / MR 审批。

通过标准：

- 系统不会覆盖用户改动，不会未经审查合入主分支。

### 6. Release / Deploy 链路

必须明确：

- release batch 建议。
- release branch。
- regression。
- release note。
- tag。
- GitHub/Gitee 发布。
- server resource group。
- test_dev / production 区分。
- smoke / monitor / rollback。

通过标准：

- 发布和生产部署不能绕过审批、资源策略和回滚策略。

### 7. Memory 链路

必须明确：

- Record Gate。
- Extraction。
- Staging dedup。
- Async commit。
- Layered storage。
- Retrieve。
- Compact。
- 项目理解联动。

通过标准：

- Memory 不会无限膨胀，不会把敏感信息写入长期记忆。

### 8. 日志和审计链路

必须明确：

- run、agent、model、git、quality、release、memory、audit、error 日志。
- trace_id / run_id / issue_id 关联。
- 脱敏规则。
- 审计不可丢失事件。

通过标准：

- 高风险操作、密钥访问、生产操作和失败恢复都可追踪。

### 9. 服务器资源管理链路

必须明确：

- 测试开发机和生产机区分。
- 云厂商、实例、规格、网络、系统、服务。
- 到期时间和续费提醒。
- 巡检、维护、退役。

通过标准：

- 生产部署只能引用已登记资源组。

### 10. 架构可视化链路

必须明确：

- gpt-image-2 provider。
- diagram spec。
- prompt 生成。
- 敏感信息过滤。
- 图片、讲解和索引产物。

通过标准：

- 架构图可以辅助讲解，但不作为事实来源。

## 基础规范门禁

### 术语一致性

必须满足：

- 新术语进入 [术语表](./foundations/glossary.md)。
- 文档中不使用未定义的核心概念。
- Issue、Task、Run、Runtime、Provider 等易混词边界清楚。

### 数据对象完整性

必须满足：

- 核心对象有职责、关键字段、生命周期、落盘位置和关联对象。
- 配置示例不替代对象定义。
- 对象之间通过 id 或路径引用。

### 权限完整性

必须满足：

- 身份认证、会话、API Token 和项目成员关系明确。
- 权限主体、资源、动作、决策明确。
- ALLOW / DENY / REQUIRE_APPROVAL 语义明确。
- 第三方 API、Native Runtime、生产服务器有单独边界。

### 失败恢复完整性

必须满足：

- 核心失败场景有触发条件、系统动作、禁止事项和恢复出口。
- 自动重试有上限。
- 生产失败有人工介入路径。

### 文档治理完整性

必须满足：

- 权威文档归属明确。
- CLI 只在路线图维护。
- 配置完整示例只在配置方案维护。
- 新能力有写入规则。
- 状态机总表已纳入基础规范。
- 契约文档已纳入 README 和设计门禁。

## 进入实现前的设计评审问题

评审时必须逐项回答：

1. 当前 MVP 范围是否明确？
2. 哪些能力进入 MVP，哪些能力延后？
3. 每个 MVP 能力是否有权威文档？
4. 每个 MVP 能力是否有核心数据对象？
5. 每个 MVP 能力是否有配置入口？
6. 每个 MVP 能力是否有失败恢复路径？
7. 每个高风险操作是否有权限和审计规则？
8. 每个主线是否能先建立 `auth_context` 再执行？
9. 用户禁用、会话过期、Token 撤销后如何阻断任务？
10. Claude CLI / Codex CLI 的执行边界是否清楚？
11. 第三方 API 的数据边界是否清楚？
12. 生产部署是否可以被完整追踪和回滚？
13. Memory 是否有防膨胀和防污染机制？
14. AI 生成代码失败后如何返工？
15. 文档是否存在重复权威来源？

## 设计债务记录

如果允许 `READY_WITH_RISKS`，必须记录：

```yaml
design_debt:
  id: debt-001
  title: schema 迁移策略需在 Phase 1 前确认
  affected_docs:
    - configuration-guide.md
  risk: 中
  mitigation: Phase 1 只支持 schema_version 1
  owner: core
  due_phase: Phase 1
```

设计债务不得用于绕过：

- 权限模型。
- 身份会话契约。
- 质量门禁。
- 密钥和敏感信息保护。
- 生产部署审批。
- 失败恢复。

## 不允许进入开发的情况

任一情况出现，结论为 `NOT_READY`：

- 核心术语未定义。
- 核心数据对象缺失。
- 没有权限模型。
- 没有身份、会话、API Token 或审批模型。
- 没有失败恢复路径。
- Claude CLI / Codex CLI 可以绕过质量门禁。
- 第三方 API 数据边界不清楚。
- 高风险操作可以绕过鉴权或审批。
- 生产部署没有回滚和审批策略。
- `.moyuan/` 目录职责不清楚。
- 配置字段缺少必填、可为空或必须为空规则。
- 关键实现契约缺失，导致 runtime、schema 或日志无法落地。
- 同一能力存在多个互相冲突的权威文档。

## 后续文档补强项

当前文档已经覆盖主要能力。进入实现前仍建议补强以下文档规格：

- 将 [配置 Schema 规则](./configuration-schema-spec.md) 转成可机器校验的 schema。
- 为 Git Adapter、Model Provider 补充更细的 adapter contract。
- 为 GitHub 之外的 Gitee、GitLab、generic git 补充独立接入字段表。
- 为配置项补充统一的 required/optional/nullable 机器可读标记。
- 为用户、组织、成员、Token 和审批补充配置/数据库迁移细则。
- 对配置示例做一次冗余和敏感信息审计。
