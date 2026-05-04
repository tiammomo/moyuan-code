# Git Provider 接入配置

本文定义 Moyuan Code 接入 GitHub、Gitee、GitLab、自建 Git 和通用 Git URL 时的配置、认证、能力声明和失败降级规则。

权威边界：

| 内容 | 权威文档 |
| --- | --- |
| 仓库接入流程和 Git Adapter 触发点 | [仓库接入与 Git Provider Adapter](./repository-onboarding-git-management.md) |
| 项目阅读理解 full/incremental/diff 流程 | [项目接入与阅读理解主线](./mainlines/project-comprehension.md) |
| Git 分支、worktree、PR/MR 和用户改动保护 | [Git 分支策略](./policies/git-branch-policy.md) |
| 配置字段 required/nullable/must-be-null 规则 | [配置 Schema 规则](./configuration-schema-spec.md) |
| Secret、命令、网络和生产权限 | [权限模型](./foundations/permission-model.md) |

## 1. 接入目标

Git Provider 接入必须支持：

- clone、fetch、pull 和 remote metadata 读取。
- 默认分支、remote、当前 commit 和 branch 状态识别。
- task branch、integration branch、release branch 的创建和推送。
- PR/MR 创建、更新和链接。
- branch protection 或保护分支规则读取。
- release branch、tag、PR/MR 发布到远程平台。

Git Provider 文档只定义远程服务商差异，不重复项目阅读理解、Issue 编排和质量门禁。

当前已落地 Git Provider plan、preview、create guard 和状态同步：

- CLI：`moyuan git provider plan <issue-id>`、`list`、`show <plan-id>`、`sync <plan-id>`、`preview <plan-id>`、`create <plan-id> [--approved]`。
- API：`POST /v1/projects/:project_id/issues/:issue_id/git-provider-plan`、`GET /v1/projects/:project_id/git-provider-plans`、`GET /v1/projects/:project_id/git-provider-plans/:plan_id`、`POST /v1/projects/:project_id/git-provider-plans/:plan_id/sync`、`POST /v1/projects/:project_id/git-provider-plans/:plan_id/preview`、`POST /v1/projects/:project_id/git-provider-plans/:plan_id/create`。
- 输出位置：`.moyuan/lifecycle/pull-requests/`。
- 默认只生成可审计 push/PR/MR plan 和 preview，不真实执行 push、PR/MR 创建或合并。
- 已接入门禁：dirty worktree、remote 缺失、review merge decision 未通过时阻断。
- `github`、`gitee`、`gitlab` 会输出 PR/MR 计划和远程 compare/new 链接；缺少 API auth 时降级为 manual create mode。`generic_git` 只输出 push plan 和手动 PR/MR 指引。
- Phase 5 已支持 GitHub/Gitee create adapter：必须 `--approved`、authz `git:write`、`secret` resolver 通过，并设置 `MOYUAN_ALLOW_GIT_PROVIDER_WRITE=1` 才会调用远程 API；否则保持 preview-only。

## 2. Provider 类型

| Provider | 场景 | API 能力 | PR/MR 能力 | 降级方式 |
| --- | --- | --- | --- | --- |
| `github` | GitHub.com 或 GitHub Enterprise | REST/GraphQL | Pull Request | 退化为 git push + 手动 PR 指引 |
| `gitee` | Gitee 公网或企业版 | OpenAPI | Pull Request | 退化为 git push + 手动 PR 指引 |
| `gitlab` | GitLab.com 或自建 GitLab | REST API | Merge Request | 退化为 git push + 手动 MR 指引 |
| `generic_git` | 自建 Git、SSH URL、未知平台 | 无统一 API | 无承诺 | 只保证 clone/fetch/branch/push |

Provider Adapter 必须显式声明能力。Orchestrator 只能调用声明为可用的能力。

```yaml
provider: github
capabilities:
  clone: true
  fetch: true
  push: true
  default_branch_detect: true
  pull_request: true
  merge_request: false
  branch_protection_read: true
  issue_link: true
auth:
  methods:
    - https_token
    - ssh_key
    - credential_helper
    - app
```

## 3. 配置文件

| 文件 | 作用 |
| --- | --- |
| `.moyuan/repository.yaml` | 仓库来源、provider、remote、默认分支和 Provider 专项配置 |
| `.moyuan/policies/secrets.yaml` | token、SSH key、App private key 或 credential helper 引用 |
| `.moyuan/policies/permissions.yaml` | push、PR/MR、tag、保护分支读取等审批策略 |
| `.moyuan/policies/release.yaml` | release branch、tag、远程发布和 PR/MR 策略 |

日志和状态：

| 目录 | 作用 |
| --- | --- |
| `.moyuan/logs/git/` | clone、fetch、checkout、branch、push、PR/MR 日志 |
| `.moyuan/logs/audit/` | secret 访问、push、tag、PR/MR、保护分支读取审计 |
| `.moyuan/lifecycle/releases/` | release branch、tag、PR/MR 和发布记录 |

## 4. repository.yaml 样例

```yaml
schema_version: 1

repository:
  source:
    type: remote_git
    provider: github
    url: https://github.com/org/order-service.git
    clone_path: ~/.moyuan/workspaces/github.com/org/order-service

  provider_config:
    owner: org
    repo: order-service
    host: github.com
    api_base_url: https://api.github.com
    web_base_url: https://github.com
    auth:
      method: https_token
      token_ref: secret:git_provider_token
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
    body_template: .moyuan/templates/pull-request.md
    draft: true
    labels: [moyuan]
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

字段 required、nullable、conditional_required 和 must_be_null 规则由 [配置 Schema 规则](./configuration-schema-spec.md) 维护。

## 5. 认证方式

| 认证方式 | 适用 Provider | 用途 | 约束 |
| --- | --- | --- | --- |
| `https_token` | GitHub、Gitee、GitLab | API、clone、fetch、push、PR/MR | token 必须通过 secret ref 引用 |
| `ssh_key` | 所有 Git Provider | clone、fetch、push | 私钥路径只能通过 secret ref 引用 |
| `credential_helper` | 本地 Git 环境 | 复用本机 Git 凭证 | 不允许把 helper 输出写入日志或 Memory |
| `app` | GitHub App、GitLab App 等 | 组织级安装、短期 token | private key 必须是引用 |

`policies/secrets.yaml` 示例：

```yaml
schema_version: 1

secrets:
  git_provider_token:
    type: token
    ref: env:GIT_PROVIDER_TOKEN
    usage:
      - repository.clone
      - repository.fetch
      - repository.push
      - pull_request.create
      - branch_protection.read
      - release.tag_push

  git_provider_ssh_key:
    type: ssh_key
    ref: env:GIT_PROVIDER_SSH_KEY_PATH
    usage:
      - repository.clone
      - repository.fetch
      - repository.push
```

## 6. 能力权限矩阵

| 场景 | 必需能力 | 必需权限 | 可为空 |
| --- | --- | --- | --- |
| 公共仓库只读接入 | clone、fetch | 无 token 或只读 token | `default_branch`、PR/MR 配置 |
| 私有仓库只读接入 | clone、fetch | repo read 或 SSH read | PR/MR 配置 |
| 自动创建 task branch | push、branch | repo write 或 SSH write | PR/MR 配置 |
| 自动创建 PR/MR | push、pull_request 或 merge_request | repo write + PR/MR write | reviewers、assignees、labels |
| 读取保护分支 | branch_protection_read | 对应平台读权限 | 读取失败可降级人工确认 |
| 发布到远程 | push、tag push | repo write | PR/MR，如果只推 branch/tag |

如果能力缺失，系统必须降级为显式状态，而不是假装成功。

## 7. 接入校验流程

```text
parse repository url
  -> detect provider
  -> validate auth method
  -> verify secret refs
  -> git ls-remote
  -> detect default branch
  -> clone or bind local directory
  -> read remote metadata if API available
  -> record provider capabilities
  -> initialize .moyuan
  -> trigger full project comprehension
  -> write git and audit logs
```

## 8. 失败处理

| 失败 | 处理 |
| --- | --- |
| clone 失败 | 接入失败，提示检查 URL、token、SSH key 或网络 |
| fetch 失败 | 暂停增量理解，保留旧项目画像并标记 stale risk |
| push 失败 | 保留本地 branch 和 diff，标记需要人工处理 |
| PR/MR 创建失败 | 生成手动 PR/MR 指引，不重复 push |
| 保护分支读取失败 | merge gate 标记 `unknown`，要求人工确认 |
| token 权限不足 | 禁用对应能力，不自动扩大权限 |
| provider API 不可用 | 降级为 `generic_git` 能力集合 |

## 9. 敏感信息规则

- token、SSH key、App private key 不得写入 YAML 明文。
- 日志只记录 secret id，不记录 secret value。
- GitHub/Gitee PR/MR 创建 token 通过 [Secret Resolver 契约](./contracts/secret-resolver-contract.md) 解析，用途必须包含 `pull_request.create`。
- PR/MR body 不得包含 secret、`.env`、私网 IP、生产凭证。
- 第三方模型不得接收 token、credential helper 输出或完整远程凭证上下文。
- 图像 prompt 不得包含远程仓库密钥、私有 endpoint 或内部 IP。

## 10. 服务商说明

| Provider | 推荐认证 | 备注 |
| --- | --- | --- |
| GitHub | fine-grained token 或 GitHub App | 权限按仓库和 endpoint 最小化 |
| Gitee | personal access token 或企业应用 | OpenAPI 权限按组织策略配置 |
| GitLab | project/group access token 或 OAuth/App | 自建 GitLab 需要配置 host 和 api_base_url |
| generic Git | SSH key 或 credential helper | 不承诺 API、PR/MR、保护分支读取 |

具体服务商 API 权限在实现阶段以对应官方文档为准。本文只维护 Moyuan 的抽象配置和降级规则。
