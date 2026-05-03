# 仓库接入、Git 与项目理解

## 1. 目标

Moyuan 管理项目时，用户只需要提供代码仓库位置即可接入。仓库位置支持两种形式：

- 本地仓库路径，例如 `/home/user/projects/order-service`。
- 远程 Git 仓库，例如 GitHub、Gitee、GitLab、自建 Git 服务或通用 Git URL。

接入后系统必须初始化 `.moyuan/` 工作空间、识别 Git 状态，并立即执行项目阅读理解。每次拉取远程分支后，也必须执行增量阅读理解，确保 Agent 在最新项目画像和模块地图上工作。

## 2. 接入流程

### 本地路径接入

处理流程：

1. 校验路径存在。
2. 判断是否为 Git 仓库。
3. 如果不是 Git 仓库，提示用户选择是否初始化 Git。
4. 读取 remote、default branch、当前 branch 和 worktree 状态。
5. 创建或更新 `.moyuan/`。
6. 自动执行 full project comprehension。
7. 生成项目画像、模块地图、skills 推荐和初始 memory candidates。

### 远程仓库接入

处理流程：

1. 解析仓库 URL。
2. 识别 provider：`github`、`gitee`、`gitlab`、`generic_git`。
3. 校验认证方式：SSH key、HTTPS token、本地 Git credential helper。
4. clone 到框架管理的 workspace。
5. 读取默认分支。
6. 初始化 `.moyuan/`。
7. 记录 remote metadata。
8. 自动执行 full project comprehension。
9. 生成项目画像、模块地图、skills 推荐和初始 memory candidates。

## 3. 远程 Provider

首批支持：

- GitHub。
- Gitee。
- GitLab。
- 通用 Git URL。

GitHub 的必填字段、认证方式、token 权限和可空字段由 [GitHub 接入配置](./github-integration.md) 维护。本文只描述通用仓库接入流程。

Provider Adapter 需要声明能力：

```yaml
provider: github
capabilities:
  clone: true
  fetch: true
  push: true
  pull_request: true
  issue_link: true
  branch_protection_read: true
  default_branch_detect: true
auth:
  methods:
    - ssh
    - https_token
    - credential_helper
```

对于 `generic_git`，MVP 只保证 clone、fetch、checkout、branch、commit、push 等标准 Git 能力，不承诺 PR/MR 能力。

## 4. 项目阅读理解

项目阅读理解分两类。

### Full Comprehension

首次接入项目后执行。

输入：

- 仓库目录结构。
- 语言、框架和依赖文件。
- 构建、测试、lint、typecheck 配置。
- 入口文件。
- 核心模块。
- API、路由、数据模型。
- CI/CD 配置。
- README、docs、ADR。
- Git remote 和默认分支。

输出：

- 项目画像。
- 模块地图。
- 架构摘要。
- 构建和测试命令。
- 风险文件。
- 推荐 Agent roles。
- 推荐 skills。
- 初始 memory candidates。

### Incremental Comprehension

fetch、pull、rebase、merge 或任务完成后执行。

输入：

- 上一次理解时的 commit。
- 当前 head commit。
- `git diff <previous>..<current>`。
- 变更文件列表。
- 受影响模块。
- 更新后的构建和测试配置。

输出：

- 项目画像 patch。
- 新增或变化的模块说明。
- 可能失效的 memory。
- 需要补充的 facts、lessons、decisions 候选。
- 对当前任务的影响分析。

## 5. 阅读理解触发点

```text
project add
  -> clone or bind local path
  -> detect git status
  -> full comprehension
  -> project profile
  -> module map
  -> skills recommendation
  -> memory candidates
```

```text
git fetch/pull/rebase
  -> detect changed commits
  -> incremental comprehension
  -> update project profile
  -> update module map
  -> mark stale memory
  -> write comprehension event
```

```text
task completed
  -> diff comprehension
  -> update project profile if needed
  -> propose memory candidates
  -> write run report
```

## 6. 阅读理解产物

产物目录：

```text
.moyuan/comprehension/
  project-profile.md
  module-map.md
  dependency-map.md
  commands.md
  risks.md
  events.jsonl
  snapshots/
```

`project-profile.md` 记录项目用途、技术栈、主要模块、入口、启动方式、构建/测试/lint 命令、关键依赖、代码组织方式和风险。

`module-map.md` 记录模块名称、路径、职责、依赖关系、主要接口和禁止跨越的边界。

`events.jsonl` 记录每次理解事件。

## 7. Git 分支策略

每个开发任务有独立工作分支。默认分支命名：

```text
moyuan/<task-id>-<slug>
```

推荐策略：

```yaml
git:
  default_branch: main
  branch_policy:
    mode: task_branch
    naming: moyuan/{task_id}-{slug}
    base: default_branch
    sync_before_run: true
    require_clean_worktree: true
    allow_auto_commit: false
    allow_auto_push: false
    allow_auto_pr: false
    delete_branch_after_merge: false
```

任务开始前：

```text
Task Created
  -> detect base branch
  -> check worktree
  -> fetch remote
  -> checkout base branch
  -> pull/rebase base branch
  -> incremental comprehension
  -> create task branch
  -> run agents
```

## 8. 用户改动保护

必须遵守：

- 不自动 stash 用户改动，除非用户显式授权。
- 不自动 reset、clean 或 checkout 覆盖用户文件。
- 不在 dirty worktree 中启动自动代码修改。
- 不自动 force push。
- 不自动删除未合并分支。

遇到 dirty worktree 时，系统停止自动代码修改，并建议用户提交、暂存、备份 patch 或指定新 workspace。

## 9. PR/MR 策略

PR/MR 创建需要 provider adapter 支持。默认不自动创建 PR/MR，除非项目策略启用。

PR/MR 内容必须包含：

- 任务背景。
- 实现摘要。
- 修改文件。
- 测试结果。
- 风险和回滚建议。
- run report 链接或路径。

## 10. Memory 联动

阅读理解结果不会全部进入长期 memory，而是先生成 memory candidates。

自动候选：

- 构建命令。
- 测试命令。
- lint/typecheck 命令。
- 稳定模块职责。
- 稳定入口文件。
- 已确认技术栈。

需要确认：

- 架构决策。
- 模块边界变更。
- 发布流程变更。
- 安全策略变更。

需要标记过时：

- 指向已删除文件的 facts。
- 与最新代码冲突的模块说明。
- 已替换的构建、测试或启动命令。
- 不再成立的 API 约定。

## 11. 审计记录

需要记录的操作：

- clone。
- fetch。
- pull/rebase 后的 incremental comprehension。
- checkout。
- branch create/delete。
- merge/rebase。
- commit。
- push。
- PR/MR create/update。
- diff collection。
- conflict detection。
- comprehension event。
