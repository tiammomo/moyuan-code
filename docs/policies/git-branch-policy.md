# Git 分支策略

## 1. 目标

决定什么时候创建分支、worktree、integration branch、release branch、commit、push 和 PR/MR，并保护用户已有改动不被系统覆盖。

Commit message、自动 commit 条件和禁止事项由 [工程流程规范](../engineering-process-standards.md) 维护。

## 2. 输入事实

- repository source。
- provider capabilities。
- current branch。
- default branch。
- dirty worktree。
- untracked files。
- issue id。
- epic id。
- quality and review status。
- branch policy。
- commit policy。
- user approval。

## 3. 决策结果

- `CREATE_ISSUE_BRANCH`
- `CREATE_ISSUE_WORKTREE`
- `CREATE_EPIC_INTEGRATION_BRANCH`
- `MERGE_TO_INTEGRATION`
- `CREATE_RELEASE_BRANCH`
- `CREATE_COMMIT`
- `COMMIT_BLOCKED`
- `PUSH_ALLOWED`
- `PR_MR_ALLOWED`
- `BLOCKED_USER_CHANGES`

## 4. 用户改动保护树

```text
if dirty worktree and task needs file write:
  BLOCKED_USER_CHANGES
else if untracked files overlap write scope:
  BLOCKED_USER_CHANGES
else if operation would overwrite user file:
  BLOCKED_USER_CHANGES
else:
  continue
```

系统默认不执行：

- `git reset --hard`
- `git clean`
- force push
- 自动 stash
- 删除未合并分支

## 5. 分支创建树

```text
if issue ready and clean worktree:
  sync base branch
  run incremental comprehension
  CREATE_ISSUE_BRANCH
  CREATE_ISSUE_WORKTREE

if epic has multiple issues:
  CREATE_EPIC_INTEGRATION_BRANCH

if integration branch accepted and release policy suggests release:
  CREATE_RELEASE_BRANCH
```

## 6. 合并树

```text
if issue quality passed and review accepted:
  merge issue branch to epic integration branch
else:
  block merge

if merge conflict:
  create conflict report
  replan or rework
```

## 7. 远程操作树

Commit：

```text
if allow_auto_commit == false:
  write commit suggestion
else if quality or review not accepted:
  COMMIT_BLOCKED
else if commit message invalid:
  COMMIT_BLOCKED
else if diff outside issue write_scope:
  COMMIT_BLOCKED
else:
  CREATE_COMMIT
```

```text
if allow_auto_push == false:
  require user approval
else if provider auth missing:
  block push
else if branch protection blocks push:
  create PR/MR if allowed
else:
  push branch
```

PR/MR：

```text
if allow_auto_pr == true and provider supports PR/MR:
  create PR/MR
else:
  write PR/MR suggestion
```

## 8. 阻断条件

- dirty worktree。
- provider auth missing。
- branch protection blocks direct push。
- commit message 不符合规范。
- merge conflict。
- quality/review 未通过。
- release branch 来源不是 accepted integration branch。

## 9. 产物和日志

产物：

- `.moyuan/lifecycle/branches/`
- `.moyuan/lifecycle/merge-reports/`
- `.moyuan/lifecycle/pull-requests/`

日志：

- `git`
- `run`
- `audit`
- `error`

## 10. 关联配置

- `.moyuan/repository.yaml`
- `.moyuan/policies/orchestration.yaml`
- `.moyuan/policies/release.yaml`
- `.moyuan/policies/permissions.yaml`
- [工程流程规范](../engineering-process-standards.md)

## 11. 验收用例

- dirty worktree 时不会创建写入任务。
- issue 未通过质量门禁不会合入 integration branch。
- branch protection 阻止 push 时会转为 PR/MR 建议。
- commit message 不符合规范时不会自动提交。
- release branch 不会从普通 issue branch 创建。
