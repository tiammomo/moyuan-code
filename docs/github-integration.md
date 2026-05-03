# GitHub 接入配置

本文定义 Moyuan Code 连接 GitHub 仓库需要的配置、必填字段、可选字段、可为空字段和权限要求。Gitee、GitLab 和 generic git 的规则由后续独立文档补充。

## 1. 目标

GitHub 接入需要支持：

- clone / fetch / pull。
- 识别默认分支、remote、当前 commit。
- 创建 task branch、epic branch、release branch。
- push task branch 或 release branch。
- 创建 PR。
- 读取 branch protection。
- 关联 GitHub issue 或 PR。
- 将发布分支和 tag 推送到 GitHub。

GitHub 认证推荐优先使用 fine-grained personal access token 或 GitHub App。GitHub 官方文档说明，fine-grained personal access token 可以限制到指定仓库和细粒度权限，token 权限不会超过 token 所属用户本身拥有的权限。

## 2. 配置文件位置

GitHub 接入涉及三类配置：

| 文件 | 作用 |
| --- | --- |
| `.moyuan/repository.yaml` | 仓库来源、provider、remote、分支策略 |
| `.moyuan/policies/secrets.yaml` | GitHub token 或 SSH key 的引用 |
| `.moyuan/policies/permissions.yaml` | push、PR、tag、branch protection 等高风险操作审批策略 |

运行状态和审计记录：

| 目录 | 作用 |
| --- | --- |
| `.moyuan/logs/git/` | Git 操作日志 |
| `.moyuan/logs/audit/` | push、PR、tag、secret 访问等审计 |
| `.moyuan/lifecycle/releases/` | release branch、tag、PR/MR 记录 |

## 3. repository.yaml 字段

```yaml
schema_version: 1

repository:
  source:
    type: remote_git
    provider: github
    url: https://github.com/org/order-service.git
    clone_path: ~/.moyuan/workspaces/github.com/org/order-service

  github:
    owner: org
    repo: order-service
    host: github.com
    api_base_url: https://api.github.com
    web_base_url: https://github.com
    auth:
      method: https_token
      token_ref: secret:github_token
      ssh_key_ref: null
      credential_helper: null

  default_remote: origin
  default_branch: main

git:
  branch_policy:
    mode: task_branch
    naming: moyuan/{issue_id}-{slug}
    base: default_branch
    sync_before_run: true
    require_clean_worktree: true
    allow_auto_commit: false
    allow_auto_push: false
    allow_auto_pr: false

  pull_request:
    enabled: true
    base: default_branch
    title_template: "[Moyuan] {issue_id}: {title}"
    body_template: .moyuan/templates/github-pr.md
    draft: true
    labels:
      - moyuan
    reviewers: []
    assignees: []

  branch_protection:
    read: true
    enforce_before_merge: true

  release_branch:
    enabled: true
    naming: release/{version}
    allow_tag_push: false
```

## 4. 字段必填规则

### repository.source

| 字段 | 是否必填 | 可为空 | 说明 |
| --- | --- | --- | --- |
| `type` | 是 | 否 | GitHub 远程仓库固定为 `remote_git` |
| `provider` | 是 | 否 | 固定为 `github` |
| `url` | 是 | 否 | GitHub clone URL，HTTPS 或 SSH |
| `clone_path` | 否 | 是 | 为空时由系统按 provider/owner/repo 自动生成 |

### repository.github

| 字段 | 是否必填 | 可为空 | 说明 |
| --- | --- | --- | --- |
| `owner` | 是 | 否 | GitHub org 或 user |
| `repo` | 是 | 否 | 仓库名 |
| `host` | 否 | 否 | 默认 `github.com`；GitHub Enterprise 必填 |
| `api_base_url` | 否 | 否 | 默认 `https://api.github.com`；GitHub Enterprise 必填 |
| `web_base_url` | 否 | 否 | 默认 `https://github.com` |
| `auth.method` | 是 | 否 | `https_token`、`ssh_key`、`credential_helper`、`github_app` |
| `auth.token_ref` | 视情况 | 是 | `https_token` 或 `github_app` 需要 |
| `auth.ssh_key_ref` | 视情况 | 是 | `ssh_key` 需要 |
| `auth.credential_helper` | 否 | 是 | 使用本地 Git credential helper 时可填 |

### repository default

| 字段 | 是否必填 | 可为空 | 说明 |
| --- | --- | --- | --- |
| `default_remote` | 否 | 否 | 默认 `origin` |
| `default_branch` | 否 | 是 | 为空时从 GitHub metadata 或 remote HEAD 读取 |

### git.branch_policy

| 字段 | 是否必填 | 可为空 | 说明 |
| --- | --- | --- | --- |
| `mode` | 是 | 否 | MVP 使用 `task_branch` |
| `naming` | 是 | 否 | 任务分支命名模板 |
| `base` | 否 | 否 | 默认 `default_branch` |
| `sync_before_run` | 否 | 否 | 默认 `true` |
| `require_clean_worktree` | 否 | 否 | 默认 `true` |
| `allow_auto_commit` | 否 | 否 | 默认 `false` |
| `allow_auto_push` | 否 | 否 | 默认 `false` |
| `allow_auto_pr` | 否 | 否 | 默认 `false` |

### git.pull_request

| 字段 | 是否必填 | 可为空 | 说明 |
| --- | --- | --- | --- |
| `enabled` | 否 | 否 | 默认 `false` |
| `base` | 否 | 是 | 为空时使用 default branch |
| `title_template` | 否 | 否 | 默认由 issue title 生成 |
| `body_template` | 否 | 是 | 为空时使用系统默认模板 |
| `draft` | 否 | 否 | 默认 `true` |
| `labels` | 否 | 是 | 可为空数组 |
| `reviewers` | 否 | 是 | 可为空数组 |
| `assignees` | 否 | 是 | 可为空数组 |

## 5. secrets.yaml 字段

```yaml
schema_version: 1

secrets:
  github_token:
    type: token
    ref: env:GITHUB_TOKEN
    usage:
      - repository.clone
      - repository.fetch
      - repository.push
      - pull_request.create
      - branch_protection.read
      - release.tag_push

  github_ssh_key:
    type: ssh_key
    ref: env:GITHUB_SSH_KEY_PATH
    usage:
      - repository.clone
      - repository.fetch
      - repository.push

  github_app_private_key:
    type: private_key
    ref: env:GITHUB_APP_PRIVATE_KEY_PATH
    usage:
      - github_app.installation_token
```

| 字段 | 是否必填 | 可为空 | 说明 |
| --- | --- | --- | --- |
| `github_token` | 视认证方式 | 是 | HTTPS token、API、PR 创建需要 |
| `github_ssh_key` | 视认证方式 | 是 | SSH clone/push 需要 |
| `github_app_private_key` | 视认证方式 | 是 | GitHub App 模式需要 |
| `ref` | 是 | 否 | 必须是 env 或 secret manager 引用，不能是明文 |
| `usage` | 是 | 否 | 必须声明用途 |

## 6. GitHub token 权限建议

### 只读接入

用于 clone、fetch、读取 metadata 和项目理解。

建议权限：

| 权限 | 级别 |
| --- | --- |
| Metadata | read |
| Contents | read |

### 分支开发

用于创建 task branch、push 分支。

建议权限：

| 权限 | 级别 |
| --- | --- |
| Metadata | read |
| Contents | read/write |

### 创建 PR

用于创建 pull request。

建议权限：

| 权限 | 级别 |
| --- | --- |
| Metadata | read |
| Contents | read |
| Pull requests | read/write |

如果 Moyuan 同时需要 push head branch，则还需要 `Contents: read/write`。

### 读取 branch protection

用于 merge gate 前读取保护规则。

建议权限：

| 权限 | 级别 |
| --- | --- |
| Metadata | read |
| Administration | read |

如果组织策略不允许 Administration read，则 branch protection 检查可以降级为 `unknown`，但合入必须要求人工确认。

### Release tag push

用于推送 tag 和 release branch。

建议权限：

| 权限 | 级别 |
| --- | --- |
| Metadata | read |
| Contents | read/write |

## 7. 场景配置矩阵

| 场景 | 必填 | 可为空 |
| --- | --- | --- |
| 公共仓库只读接入 | `source.url`、`provider` | `github_token`、`default_branch`、`pull_request` |
| 私有仓库只读接入 | `source.url`、`provider`、`auth.method`、对应 secret ref | `pull_request` |
| 自动创建 task branch | `branch_policy.naming`、push 权限 | `pull_request` |
| 自动创建 PR | `pull_request.enabled`、`github_token`、PR 权限 | `reviewers`、`assignees`、`labels` |
| 读取 branch protection | `branch_protection.read`、对应权限 | 读取失败时可降级为人工确认 |
| 发布到 GitHub | `release_branch`、push 权限 | `pull_request`，如果只 push branch/tag |
| GitHub Enterprise | `host`、`api_base_url`、`web_base_url` | 默认 GitHub 公网 URL |

## 8. 接入校验流程

添加 GitHub 项目时必须执行：

1. 解析 owner 和 repo。
2. 校验 URL 格式。
3. 校验认证方式。
4. 检查 secret ref 是否存在。
5. 执行 `git ls-remote`。
6. 读取默认分支。
7. clone 或绑定本地目录。
8. 读取 remote metadata。
9. 检查 token 权限是否满足启用能力。
10. 初始化 `.moyuan/`。
11. 运行 full project comprehension。
12. 写入 git/audit 日志。

## 9. 权限失败处理

| 失败 | 处理 |
| --- | --- |
| clone 失败 | 标记接入失败，提示检查 URL、token 或 SSH key |
| fetch 失败 | 暂停项目理解，保留旧项目画像 |
| push 失败 | 不创建 PR，标记 issue 需要人工处理 |
| PR 创建失败 | 保留 branch 和 diff，生成手动 PR 指引 |
| branch protection 读取失败 | merge gate 标记为 unknown，要求人工确认 |
| token 权限不足 | 禁用对应能力，不自动扩大权限 |

## 10. 敏感信息规则

- token、SSH key、GitHub App private key 不得写入 YAML 明文。
- 日志只记录 secret id，不记录 secret value。
- PR body 不得包含 secret、`.env`、私网 IP、生产凭证。
- 第三方模型不得接收 GitHub token 或 credential helper 输出。

## 11. 参考来源

- GitHub Docs：Fine-grained personal access tokens 可以限制到指定仓库和细粒度权限。
- GitHub Docs：创建 pull request 的 REST API 对 fine-grained token 需要 Pull requests 权限，且通常需要 Contents 读取权限。
- GitHub Docs：GitHub REST API 会按 endpoint 声明 fine-grained token 所需权限。
