# 仓库接入与 Git Adapter

本文只定义仓库接入、Git Provider Adapter 和远程同步后的阅读理解触发边界。

权威边界：

| 内容 | 权威文档 |
| --- | --- |
| 项目阅读理解流程、full/incremental/diff 模式 | [项目接入与阅读理解主线](./mainlines/project-comprehension.md) |
| Git 分支、worktree、合入和用户改动保护策略 | [Git 分支策略](./policies/git-branch-policy.md) |
| GitHub token、SSH、必填和可空字段 | [GitHub 接入配置](./github-integration.md) |
| Issue 分支、integration branch 和并发工作区 | [Issues 编排与并发调度](./issue-orchestration.md) |
| 配置字段必填、可空和必须为空 | [配置 Schema 规则](./configuration-schema-spec.md) |

## 1. 接入目标

用户提供仓库位置后，Moyuan 必须能把仓库纳入项目生命周期管理。

支持两种入口：

- 本地路径：已有代码目录或已有 Git 仓库。
- 远程仓库：GitHub、Gitee、GitLab、自建 Git 服务或通用 Git URL。

接入完成的判定：

- `.moyuan/` 工作空间初始化完成。
- Git remote、默认分支、当前分支和 worktree 状态已记录。
- 项目阅读理解已完成，生成项目画像、模块地图、命令清单和 memory candidates。
- 后续 fetch、pull、rebase 或 merge 后会触发增量阅读理解。

## 2. 接入流程

| 场景 | 流程 | 产物 |
| --- | --- | --- |
| 本地 Git 仓库 | 校验路径 -> 读取 Git 状态 -> 初始化 `.moyuan/` -> full comprehension | repository metadata、project profile |
| 本地非 Git 目录 | 校验路径 -> 请求用户确认是否初始化 Git -> 初始化 `.moyuan/` -> full comprehension | repository metadata、init decision |
| 远程仓库 | 解析 URL -> 识别 provider -> 校验认证 -> clone -> 初始化 `.moyuan/` -> full comprehension | clone workspace、remote metadata |
| 远程同步 | fetch/pull/rebase/merge -> diff commits -> incremental comprehension -> stale memory marking | comprehension event、profile patch |

## 3. Provider 能力

Provider Adapter 必须声明能力，Orchestrator 只调用声明为可用的动作。

| Provider | 必需能力 | 可选能力 |
| --- | --- | --- |
| `github` | clone、fetch、push、branch、default branch detect | PR、Issue link、branch protection |
| `gitee` | clone、fetch、push、branch、default branch detect | PR/MR、Issue link |
| `gitlab` | clone、fetch、push、branch、default branch detect | MR、Issue link、pipeline trigger |
| `generic_git` | clone、fetch、checkout、branch、commit、push | 无承诺 PR/MR 能力 |

最小 Provider 描述：

```yaml
provider: github
capabilities:
  clone: true
  fetch: true
  push: true
  pull_request: true
  default_branch_detect: true
auth:
  methods: [ssh, https_token, credential_helper]
```

## 4. Git 触发点

```text
project add
  -> bind local path or clone remote
  -> detect git state
  -> full comprehension
```

```text
git fetch/pull/rebase/merge
  -> detect changed commits
  -> incremental comprehension
  -> update project profile and module map
  -> mark stale memory
```

```text
issue start
  -> require clean worktree
  -> sync base branch if policy allows
  -> incremental comprehension
  -> create issue branch/worktree
```

## 5. 工作空间产物

```text
.moyuan/
  repository.yaml
  comprehension/
    project-profile.md
    module-map.md
    dependency-map.md
    commands.md
    events.jsonl
  logs/
    git/
```

`events.jsonl` 至少记录：

- project add。
- clone。
- fetch/pull/rebase/merge。
- checkout。
- branch create/delete。
- push。
- PR/MR create/update。
- comprehension event。

## 6. 用户改动保护

仓库 Adapter 不允许破坏用户本地修改。

禁止行为：

- 自动 reset、clean 或 checkout 覆盖用户文件。
- 未授权自动 stash 用户改动。
- dirty worktree 中启动自动代码修改。
- force push。
- 删除未合并分支。

遇到 dirty worktree 时，系统只能进入 `BLOCKED(dirty_worktree)`，并提示用户提交、暂存、备份 patch 或指定新 workspace。

## 7. 验收标准

- 本地路径和远程 URL 都能被接入。
- 接入后自动触发 full comprehension。
- 每次远程同步后自动触发 incremental comprehension。
- Git Provider 能力不足时可以降级为 generic Git。
- Git 操作全部进入审计日志。
- 用户未提交改动不会被自动覆盖。
