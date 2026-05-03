# 文档维护规则

本文定义 Moyuan Code 文档的权威来源、维护边界、命名规则和变更流程，防止 docs 膨胀、重复和互相矛盾。

## 目标

- 每类信息只有一个权威展开位置。
- 新能力知道应该写到哪里。
- 配置示例不在多个文档重复维护。
- 生成资产、prompt 和图片有清晰归属，且不被误当成权威设计正文。
- 文档变更可审查、可追踪、可精简。
- 文档状态、评审、发布和归档有明确规则。

## 文档分层

Moyuan Code 文档分为七层：

| 层级 | 目录/文档 | 作用 |
| --- | --- | --- |
| 入口层 | `README.md` | 给读者导航，不承载完整方案 |
| 基础规范层 | `foundations/`、`design-readiness-checklist.md` | 定义术语、对象、权限、失败恢复和文档门禁 |
| 总体规划层 | `lifecycle-roadmap.md`、`reference-architecture.md` | 定义产品方向、生命周期、架构和阶段 |
| 主线层 | `mainlines/` | 按真实生命周期串联端到端流程 |
| 策略层 | `policies/` | 定义可实现的决策树、阻断条件和人工确认规则 |
| 专题设计层 | `issue-orchestration.md`、`agent-memory-system.md` 等 | 定义单一能力的完整设计 |
| 资产层 | `assets/`、`.moyuan/visuals/` | 保存架构图、讲解和可追溯生成产物 |

维护规则：

- 入口层只放索引和原则。
- 基础规范层优先被引用，不能依赖专题文档定义术语。
- 总体规划层描述全局流程，不展开模块内部细节。
- 主线层描述端到端执行流程，不重复完整配置和对象字段。
- 策略层描述决策树，不重复流程背景和专题细节。
- 专题设计层只能有一个权威文档负责一个能力。
- 资产层必须能追溯到生成 prompt 或说明文档；prompt 默认归为生成产物，不参与文档权威性判断。

## 权威来源原则

| 信息类型 | 权威文档 | 其他文档规则 |
| --- | --- | --- |
| 文档索引和阅读入口 | `README.md` | 只做简短说明和链接 |
| 产品定位、生命周期、CLI、Phase | `lifecycle-roadmap.md` | 专题文档不重复 CLI 列表 |
| 总体架构和状态机 | `reference-architecture.md` | 专题文档只引用状态机 |
| 端到端执行主线 | `mainlines/` | 专题文档不再重复整条生命周期流程 |
| 策略决策树 | `policies/` | 主线文档只引用策略，不展开所有判断分支 |
| Workspace 目录和 schema 索引 | `project-workspace-spec.md` | 不展开完整配置 |
| 配置组合示例 | `configuration-guide.md` | 其他文档只给局部片段或引用 |
| 配置字段规则 | `configuration-schema-spec.md` | 其他文档不重复 required/nullable 表 |
| 仓库、Git、项目理解 | `repository-onboarding-git-management.md` | 不在路线图重复流程细节 |
| Issue 编排和并发调度 | `issue-orchestration.md` | Agent 文档不重复 issue graph |
| 质量门禁 | `code-lifecycle-quality-gates.md` | Issue 文档只引用 gate 结果 |
| Agent、Team、Skills | `agent-skills-memory.md` | Memory 细节只引用 Memory 文档 |
| Agent Memory | `agent-memory-system.md` | 其他文档只说明如何调用 |
| 模型、Runtime、Adapter | `model-tool-adapters.md` | 配置细节回到 configuration guide |
| 术语 | `foundations/glossary.md` | 其他文档使用术语，不重新定义 |
| 核心数据对象 | `foundations/core-data-objects.md` | 配置文档不重复对象生命周期 |
| 权限模型 | `foundations/permission-model.md` | 配置文档只保留 YAML 示例 |
| 失败恢复 | `foundations/failure-recovery.md` | 状态机文档只保留状态定义 |
| 文档维护规则 | `foundations/documentation-governance.md` | 本文为准 |

## 文档负责人

每类文档必须有逻辑负责人。当前阶段可以不指定具体人名，但必须指定责任角色。

| 文档类型 | 责任角色 | 责任 |
| --- | --- | --- |
| README | doc_maintainer | 保持入口准确 |
| 基础规范 | architect | 保证术语、对象、权限一致 |
| 路线图 | product_owner + architect | 控制 MVP、Phase、CLI |
| 架构 | architect | 控制模块边界和状态机 |
| 主线 | product_owner + architect + domain_owner | 控制端到端流程和产物边界 |
| 策略 | architect + domain_owner | 控制决策树、阻断条件和人工确认规则 |
| 配置方案 | core_engineer | 控制 schema 和默认值 |
| Issue 编排 | orchestrator_owner | 控制任务拆分和调度规则 |
| Memory | memory_owner | 控制记录、检索、compact |
| 模型与工具 | adapter_owner | 控制 provider、runtime、adapter |
| 质量门禁 | quality_owner | 控制测试、review、阻断规则 |
| 仓库和 Git | git_owner | 控制接入、分支、用户改动保护 |
| 资产 | doc_maintainer | 控制图、prompt、讲解和敏感信息 |

变更文档时，必须确认受影响责任角色。

## 新能力写入规则

新增能力时按以下顺序判断：

1. 是否是新术语：先更新 `foundations/glossary.md`。
2. 是否新增核心对象：更新 `foundations/core-data-objects.md`。
3. 是否新增配置：更新 `configuration-guide.md`，在 `configuration-schema-spec.md` 补字段规则，并在 `project-workspace-spec.md` 加 schema 索引。
4. 是否新增 CLI：只更新 `lifecycle-roadmap.md`。
5. 是否新增状态或异常：更新 `reference-architecture.md` 或 `foundations/failure-recovery.md`。
6. 是否新增权限边界：更新 `foundations/permission-model.md`。
7. 是否影响端到端流程：更新 `mainlines/` 对应主线。
8. 是否新增判断规则：更新 `policies/` 对应策略。
9. 是否新增模块专题能力：更新对应专题文档。
10. 是否需要入口：更新 `README.md`。

## 文档生命周期

每个文档应处于以下状态之一：

| 状态 | 含义 | 允许进入实现 |
| --- | --- | --- |
| `draft` | 草案，可大幅调整 | 否 |
| `planned` | 已确认方向，待实现 | 仅可做原型 |
| `ready` | 已通过设计就绪门禁 | 是 |
| `active` | 当前实现依据 | 是 |
| `deprecated` | 已废弃，保留跳转 | 否 |
| `archived` | 历史记录，不再维护 | 否 |

状态升级规则：

- `draft -> planned`：能力方向已确认。
- `planned -> ready`：通过 [设计就绪门禁](../design-readiness-checklist.md)。
- `ready -> active`：代码实现开始依赖该文档。
- `active -> deprecated`：有新权威文档替代。
- `deprecated -> archived`：引用清理完成。

文档状态建议写在文档标题下方：

```md
状态：planned
责任角色：architect
最后更新：2026-05-03
```

## 文档命名规则

- 顶层专题文档使用 kebab-case。
- 基础规范放在 `docs/foundations/`。
- 主线文档放在 `docs/mainlines/`，按业务主线命名。
- 策略文档放在 `docs/policies/`，使用 `<domain>-policy.md` 命名。
- 新主线必须满足“独立生命周期、独立产物、关键阻断决策、独立 owner、横切引用、独立失败恢复”中的至少三项。
- 正式引用的图片和讲解可以放在 `docs/assets/`。
- 生成 prompt、diagram spec 和中间产物优先放在 `.moyuan/visuals/`，避免污染 docs 正文扫描。
- 图片、prompt、explanation 共享同一个前缀。
- 文件名应表达领域，不表达临时状态。
- 废弃文档不改名，顶部标记 `deprecated` 并指向新文档。

示例：

```text
docs/foundations/glossary.md
docs/foundations/core-data-objects.md
docs/mainlines/code-development.md
docs/policies/issue-scheduling-policy.md
docs/assets/moyuan-code-architecture-<timestamp>.png
docs/assets/moyuan-code-architecture-<timestamp>.explanation.md
.moyuan/visuals/prompts/moyuan-code-architecture-<timestamp>.prompt.md
```

## 文档结构模板

新增专题文档默认使用以下结构：

```text
标题：文档标题
状态：planned
责任角色：xxx
最后更新：YYYY-MM-DD

章节：
1. 目标
2. 边界
3. 核心概念
4. 流程
5. 配置或数据对象引用
6. 权限和安全
7. 失败恢复
8. 验收标准
9. 相关文档
```

如果文档只做索引，可以省略流程和验收，但必须说明维护边界。

## 配置示例规则

- 配置组合示例只在 `configuration-guide.md`。
- `project-workspace-spec.md` 只维护目录和 schema 索引。
- 专题文档可以放局部片段，但不能成为完整配置来源。
- 示例字段变化时，优先更新 `configuration-guide.md`。
- 如果示例和对象定义冲突，以 `foundations/core-data-objects.md` 的对象语义为准。

## 重复控制规则

允许重复：

- README 中的一句话摘要。
- 专题文档中的短引用。
- 配置文档中的局部 YAML 示例。

不允许重复：

- 同一对象生命周期在多个文档完整展开。
- CLI 命令列表散落在专题文档。
- Memory 机制在 Agent 文档重复说明。
- 权限策略在多个文档给出不同规则。
- 服务器资源字段在环境配置和资源配置中重复维护。
- 主线文档重复专题文档的大段细节。
- 策略文档重复主线文档的完整流程背景。

## 资产管理规则

纳入版本管理的资产：

- 关键架构图。
- 架构图讲解。
- 对外设计文档需要引用的图片。

不建议纳入版本管理的资产：

- 临时重试图片。
- 调试日志。
- 包含敏感上下文的 prompt。
- 大量中间草图。
- 仅用于复现图片生成、没有设计阅读价值的 prompt。

prompt 管理规则：

- 进入 `docs/assets/` 的 prompt 必须短小、脱敏、可读，并且确实需要被设计评审引用。
- 默认生成的 prompt 写入 `.moyuan/visuals/prompts/`，不计入 docs 正文巡检。
- 文档巡检、标题扫描和重复扫描默认排除 `*.prompt.md`。

敏感规则：

- prompt 不得包含 API key、token、密码、私网 IP、`.env` 明文。
- 图片不得展示真实密钥、账号密码或敏感生产细节。
- 需要外发分享的图必须经过人工检查。

## 文档变更 Checklist

每次文档变更前检查：

- 是否已有权威文档？
- 是否会造成重复？
- 是否需要更新 README 索引？
- 是否需要更新 mainlines 索引？
- 是否需要更新 policies 索引？
- 是否需要更新 schema 索引？
- 是否需要更新术语表？
- 是否新增核心对象？
- 是否涉及权限或敏感数据？
- 是否涉及失败恢复？
- 是否需要更新生成的架构图？
- 是否需要更新设计就绪门禁？
- 是否影响 Phase 或 MVP 范围？
- 是否需要新增或变更责任角色？

每次文档变更后检查：

- 链接是否有效。
- 标题编号是否连续。
- CLI 是否只出现在路线图。
- 配置是否只在配置文档完整展开。
- 主线是否只描述流程，不重复字段级配置。
- 策略是否以决策树形式表达。
- 新增资产是否命名清晰。
- 是否有明文密钥或敏感信息。
- 是否存在新的重复权威来源。
- 是否需要更新文档状态。

## 文档评审流程

文档变更分为三类：

| 类型 | 示例 | 评审要求 |
| --- | --- | --- |
| 小修 | 错字、链接、措辞 | 自检即可 |
| 普通变更 | 新增字段、补充流程、调整示例 | 对照 checklist 检查 |
| 关键变更 | 改生命周期、权限、对象、状态机、部署策略 | 必须重新运行设计就绪门禁相关部分 |

关键变更必须检查：

- 是否影响核心数据对象。
- 是否影响权限模型。
- 是否影响失败恢复。
- 是否影响 `.moyuan/` schema。
- 是否影响 MVP 或 Phase。
- 是否需要更新架构图。

## 文档巡检节奏

建议在以下节点做文档巡检：

- 每次进入新 Phase 前。
- 每次新增核心能力后。
- 每次准备开始实现前。
- 每次 release 前。
- 每次发生重大设计返工后。

巡检项目：

- README 索引是否完整。
- 权威来源表是否准确。
- CLI 是否只在路线图出现。
- 配置是否只在配置方案完整展开。
- 主线是否覆盖项目接入、需求规划、代码开发、代码管理、服务器资源和发布投产。
- 策略是否覆盖关键决策点和人工确认条件。
- 核心对象是否和配置一致。
- 权限和失败恢复是否覆盖新增能力。
- 图片、讲解和需要保留的 prompt 是否仍然准确。
- 是否存在过长文档需要拆分。

## 文档版本和变更记录

当前阶段不要求每个文档都维护 changelog，但关键文档进入 `active` 后应记录重要变更。

建议格式：

```md
## 变更记录

| 日期 | 变更 | 原因 |
| --- | --- | --- |
| 2026-05-03 | 新增 Runtime 权限边界 | 接入 Claude CLI / Codex CLI |
```

需要记录的变更：

- 核心对象字段变化。
- 权限策略变化。
- 失败恢复策略变化。
- `.moyuan/` 目录变化。
- CLI 命令变化。
- 发布部署流程变化。
- 第三方 API 数据策略变化。

## 合并和拆分规则

应该合并：

- 两个文档描述同一能力的完整流程。
- 一个文档只剩索引，没有独立价值。
- 同一配置在两个文档都被完整展开。

应该拆分：

- 单个文档超过 1500 行且包含多个独立领域。
- 一个文档既写配置，又写对象定义，又写实现计划。
- 新增能力有独立生命周期和独立权限边界。

拆分后必须：

- 保留旧入口跳转或更新 README。
- 更新 `project-workspace-spec.md` 的权威文档指向。
- 删除重复段落。

## 废弃规则

废弃文档必须：

- 顶部标记 `状态：deprecated`。
- 说明替代文档。
- 保留最小跳转说明。
- 从 README 索引中移除或标记废弃。
- 清理其他文档中的旧链接。

禁止：

- 直接删除仍被引用的文档。
- 同时保留两个内容冲突的 active 文档。
- 在废弃文档继续新增设计内容。

## 生成图维护规则

架构图需要更新的触发条件：

- 新增核心模块。
- 新增核心数据对象。
- 新增生命周期阶段。
- Runtime、Provider、Server Resource 或 Memory 设计发生结构变化。
- 发布和部署流程发生变化。

每张图必须配套：

- 图片文件。
- explanation 文件。
- prompt 文件或 prompt 引用。
- diagram spec，如果进入 `.moyuan/visuals/`。

图像资产命名：

```text
<project-or-topic>-<diagram-type>-<timestamp>.png
<project-or-topic>-<diagram-type>-<timestamp>.explanation.md
.moyuan/visuals/prompts/<project-or-topic>-<diagram-type>-<timestamp>.prompt.md
```

图像资产进入 README 或正式设计文档前，必须检查：

- 是否泄露敏感信息。
- 是否与当前文档一致。
- 是否有对应 prompt 引用和 explanation。
- 是否需要更新旧图或标记旧图过期。

## 文档状态标记

文档可以使用以下状态：

| 状态 | 含义 |
| --- | --- |
| `draft` | 草案，可大幅调整 |
| `planned` | 已确认方向，待实现 |
| `active` | 当前权威文档 |
| `deprecated` | 已废弃，保留跳转 |
| `archived` | 历史记录，不再维护 |

当前 docs 默认处于 `planned`，进入实现阶段后核心规范文档应升级为 `active`。

## 进入开发前的文档冻结

进入正式实现前应进行一次文档冻结：

1. 运行设计就绪门禁。
2. 确认 `READY` 或记录 `READY_WITH_RISKS` 设计债务。
3. 将作为实现依据的文档状态标记为 `ready`。
4. 后续关键变更必须说明是否影响已拆分 issue。

冻结不代表禁止修改文档，而是要求关键变更可追踪、可评审。

## 最小维护规则

如果时间有限，至少遵守：

- 新术语先改术语表。
- 新对象先改核心数据对象。
- 新 CLI 只改路线图。
- 新配置完整示例只改配置方案。
- 新权限边界改权限模型。
- 新失败场景改失败恢复。
- 新文档必须接入 README 或 foundations README。
- 开发前必须过设计就绪门禁。
