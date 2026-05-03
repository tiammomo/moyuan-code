# 核心数据对象

本文定义 Moyuan Code 的核心对象索引、对象职责、owner、落盘范围和跨对象关系。

权威边界：

| 内容 | 权威文档 |
| --- | --- |
| 状态定义和状态流转 | [状态机总表](./state-machine-catalog.md) |
| 配置字段必填、可空、必须为空 | [配置 Schema 规则](../configuration-schema-spec.md) |
| `.moyuan/` 落盘目录 | [项目工作空间规范](../project-workspace-spec.md) |
| Subagent、Skill 字段和接口 | [Subagent 与 Skill 契约](../contracts/subagent-skill-contract.md) |
| Auth、Session、Token 字段和接口 | [身份会话契约](../contracts/auth-session-contract.md) |
| Self-repair 字段和接口 | [自我修复契约](../contracts/self-repair-contract.md) |
| Runtime、日志、迁移契约 | [契约文档](../contracts/README.md) |

## 1. 设计原则

- 每个对象必须有稳定 `id`。
- 每个对象必须能追踪来源、版本、创建时间和更新时间。
- 跨对象引用使用 id 或 ref，不复制完整对象。
- 运行产物、配置对象和控制面对象分离。
- 敏感信息只保存引用，不保存明文。
- 本文不维护完整字段 schema，只维护对象语义和关系。

## 2. 对象分组

| 分组 | 对象 |
| --- | --- |
| 身份与访问 | User、Organization、Membership、Service Account、API Token、Auth Session、Approval |
| 项目与仓库 | Project、Repository、Workspace、Project Comprehension |
| 需求与执行 | Epic、Issue、Issue Graph、Schedule、Run |
| Agent 能力 | Agent Role、Agent Team、Subagent、Skill Definition、Skill Binding、Skill Effectiveness、Runtime Session |
| 模型与策略 | Model Provider、Model Policy |
| Memory 与质量 | Memory Record、Quality Report |
| 运行反馈与自我修复 | Runtime Signal、Bug Candidate、Repair Attempt、Improvement Record |
| 资源与发布 | Server Resource、Resource Group、Release、Deployment、Environment |
| 可视化与审计 | Visual Diagram、Diagram Spec、Audit Log、Log Event |

## 3. 对象索引

| 对象 | 职责 | Owner | 主要落盘/存储 | 关联对象 |
| --- | --- | --- | --- | --- |
| User | 人类用户身份 | Auth Service | 控制面数据库或本地身份文件 | Organization、Membership、Auth Session、Approval |
| Organization | 团队、租户或组织边界 | Auth Service | 控制面数据库 | User、Project、Service Account |
| Membership | 用户或服务账号的角色绑定 | Auth Service | 控制面数据库 / `.moyuan/policies/access.yaml` 引用 | User、Service Account、Project |
| Service Account | CI、发布、部署等非人类 actor | Auth Service | 控制面数据库 | API Token、Run、Release、Deployment |
| API Token | Moyuan API 或自动化凭证引用 | Auth Service | 密钥系统引用 / token hash | User、Service Account、Audit Log |
| Auth Session | 登录会话和请求上下文 | Auth Service | session store | User、Organization、Project |
| Approval | 高风险操作审批记录 | Approval Service | 控制面数据库 / audit log | User、Run、Release、Deployment |
| Project | 被 Moyuan 管理的软件项目 | Project Service | `.moyuan/project.yaml` | Repository、Workspace、Epic、Memory Record |
| Repository | Git 仓库来源和 remote 元数据 | Git Adapter | `.moyuan/repository.yaml` | Project、Issue、Run |
| Workspace | 项目独立 `.moyuan/` 工作空间 | Workspace Manager | `.moyuan/` | Project、Config、Run、Logs |
| Project Comprehension | 项目画像、模块地图和命令清单 | Comprehension Engine | `.moyuan/comprehension/` | Repository、Memory Record、Issue |
| Epic | 用户提出的开发目标 | Orchestrator | `.moyuan/lifecycle/epics/` | Issue Graph、Release |
| Issue | 最小可执行开发单元 | Orchestrator | `.moyuan/lifecycle/issues/` | Epic、Run、Subagent、Quality Report |
| Issue Graph | issues 依赖 DAG | Orchestrator | `.moyuan/lifecycle/issue-graphs/` | Epic、Issue、Schedule |
| Schedule | ready/blocked/running 队列和并发计划 | Scheduler | `.moyuan/lifecycle/schedules/` | Issue Graph、Run、Runtime Session |
| Run | 一次任务执行审计单元 | Orchestrator | `.moyuan/lifecycle/runs/`、logs | Issue、Subagent、Quality Report |
| Agent Role | 职责模板 | Agent Manager | `agents/roles.yaml` | Agent Team、Subagent、Skill Binding |
| Agent Team | 多角色协作配置 | Agent Manager | `agents/teams.yaml` | Agent Role、Issue |
| Subagent | 被调度的一次执行实例 | Subagent Manager | `.moyuan/agents/subagents/` | Issue、Run、Runtime Session、Skill |
| Skill Definition | 可复用能力定义 | Skill Registry | `skills/registry.yaml` | Skill Binding、Subagent |
| Skill Binding | Skill 到项目/角色/issue/subagent 的绑定 | Skill Registry | `skills/bindings.yaml` | Skill Definition、Agent Role、Subagent |
| Skill Effectiveness | Skill 使用效果反馈 | Skill Registry | `skills/effectiveness/` | Skill Definition、Quality Report、Memory Record |
| Runtime Session | Claude CLI、Codex CLI 或模型调用会话 | Runtime Adapter | `.moyuan/runtimes/sessions/` | Subagent、Run、Log Event |
| Model Provider | GPT、Claude、GLM、MiniMax、第三方 API 等账号引用 | Model Gateway | `models/providers.yaml` | Model Policy、Runtime Session |
| Model Policy | 任务到 provider/model 的路由规则 | Model Gateway | `models/routing.yaml` | Agent Role、Runtime Session |
| Memory Record | 可检索、可维护、可审计的长期记忆 | Memory Engine | `.moyuan/memory/` | Project、Issue、Run、Skill Effectiveness |
| Quality Report | 质量门禁和 review 结论 | Quality Engine | `.moyuan/lifecycle/quality/` | Issue、Run、Release |
| Runtime Signal | 运行、测试、冒烟、监控或用户反馈信号 | Runtime Monitor | logs / `.moyuan/runtime/signals/` | Bug Candidate、Run、Deployment |
| Bug Candidate | 疑似 bug 判断结果 | Self Repair Engine | `.moyuan/runtime/bug-candidates/` | Runtime Signal、Repair Attempt |
| Repair Attempt | 自动或半自动修复尝试 | Self Repair Engine | `.moyuan/runtime/repair-attempts/` | Bug Candidate、Run、Quality Report |
| Improvement Record | 能力增强候选 | Improvement Engine | `.moyuan/runtime/improvements/` | Repair Attempt、Skill、Memory Record |
| Server Resource | 被登记和维护的服务器资产 | Resource Manager | `policies/server-resources.yaml` | Resource Group、Deployment |
| Resource Group | 一组服务器资源 | Resource Manager | `policies/server-resources.yaml` | Server Resource、Environment |
| Environment | test/staging/production 环境配置 | Release Manager | `policies/environments.yaml` | Resource Group、Deployment |
| Release | 版本分支、tag、PR/MR 和发版记录 | Release Manager | `.moyuan/lifecycle/releases/` | Epic、Issue、Quality Report、Deployment |
| Deployment | 投产、冒烟、监控和回滚记录 | Release Manager | `.moyuan/lifecycle/deployments/` | Release、Environment、Server Resource |
| Visual Diagram | gpt-image-2 辅助生成的架构图 | Visual Service | `.moyuan/visuals/`、`docs/assets/` | Project Comprehension、Diagram Spec |
| Diagram Spec | 生成架构图前的结构化图定义 | Visual Service | `.moyuan/visuals/specs/` | Visual Diagram |
| Audit Log | 审批、密钥访问、高风险命令和权限拒绝 | Audit Service | `.moyuan/logs/audit/` | User、Run、Release、Deployment |
| Log Event | 核心运行日志事件 | Logging Service | `.moyuan/logs/` | Run、Subagent、Model Provider、Git、Memory |

## 4. 关键关系

项目接入：

```text
Project
  -> Repository
  -> Workspace
  -> Project Comprehension
  -> Memory Record candidates
```

需求开发：

```text
Epic
  -> Issue Graph
  -> Issue
  -> Schedule
  -> Subagent
  -> Runtime Session
  -> Run
  -> Quality Report
```

自我修复：

```text
Runtime Signal
  -> Bug Candidate
  -> Repair Attempt
  -> Quality Report
  -> Improvement Record
  -> Memory Record candidates
```

发布投产：

```text
Release
  -> Deployment
  -> Environment
  -> Resource Group
  -> Server Resource
```

身份与审计：

```text
User / Service Account
  -> Auth Session
  -> Approval
  -> Run / Release / Deployment
  -> Audit Log
```

## 5. 存储边界

| 类型 | 存储位置 | 说明 |
| --- | --- | --- |
| 项目级配置 | `.moyuan/*.yaml`、`.moyuan/policies/*.yaml` | 不保存凭证明文 |
| 生命周期产物 | `.moyuan/lifecycle/` | epic、issue、run、quality、release、deployment |
| Agent 产物 | `.moyuan/agents/`、`.moyuan/runtimes/` | Subagent、Runtime session 和输出 |
| Memory | `.moyuan/memory/` | 候选、暂存、长期记忆、compact 结果 |
| 项目理解 | `.moyuan/comprehension/` | project profile、module map、events |
| 日志审计 | `.moyuan/logs/` | run、agent、model、git、quality、release、memory、audit、error |
| 控制面对象 | 控制面数据库或本地身份文件 | User、Organization、Membership、API Token、Approval |
| 密钥 | secret manager、环境变量或系统 credential store | 只在配置里保存引用 |

## 6. 引用规则

跨对象引用规则：

- `project_id` 引用 Project。
- `repository_id` 引用 Repository。
- `epic_id` 引用 Epic。
- `issue_id` 引用 Issue。
- `run_id` 引用 Run。
- `subagent_id` 引用 Subagent。
- `memory_id` 引用 Memory Record。
- `release_id` 引用 Release。
- `deployment_id` 引用 Deployment。
- `server_resource_id` 引用 Server Resource。
- `user_id` 和 `service_account_id` 引用 actor。

禁止：

- 在 Issue 中复制完整 Project Comprehension。
- 在 Release 中复制完整 Server Resource。
- 在 Memory 中保存 secret 明文。
- 在日志中保存完整 prompt、response、`.env` 或 token。

## 7. 变更规则

新增核心对象时必须同步更新：

- 本文对象索引。
- [术语表](./glossary.md)。
- [状态机总表](./state-machine-catalog.md)。
- [配置 Schema 规则](../configuration-schema-spec.md)，如果对象有配置字段。
- 对应契约文档，如果对象会被代码接口使用。

不允许在主线、策略或配置文档中重新定义对象完整字段。需要字段细节时，应新增或更新契约/schema 文档。
