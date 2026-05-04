# 代码管理主线

## 1. 目标

代码管理主线负责 Git 分支、worktree、integration branch、PR/MR、用户改动保护和远程发布前的代码状态治理。

Commit message、自动 commit、回退后 fix 和发版前 Git 要求见 [工程流程规范](../engineering-process-standards.md)。

这条主线保证：

- 每个 issue 在独立分支或 worktree 中执行。
- 多 issue 并发不会互相覆盖文件。
- 用户已有改动不会被自动覆盖。
- accepted issue 只能通过合入门禁进入 epic integration branch。
- release branch 只从通过门禁的 integration branch 创建。

## 2. 输入与输出

输入：

- repository metadata。
- 当前 Git 状态。
- issue graph 和 schedule。
- branch policy。
- quality report 和 review report。
- remote provider 能力。

输出：

- issue branch。
- issue worktree。
- epic integration branch。
- merge report。
- PR/MR draft。
- release branch candidate。
- Git audit events。

## 3. 端到端流程

```text
issue ready
  -> check clean worktree
  -> sync base branch
  -> run incremental comprehension
  -> create issue branch
  -> create issue worktree
  -> run agent
  -> collect diff
  -> quality and review accepted
  -> merge into epic integration branch
  -> run integration checks
  -> unlock downstream issues
```

PR/MR 流程：

```text
epic integration branch accepted
  -> create or update PR/MR if configured
  -> attach summary, tests, risks and rollback notes
  -> wait for configured approval
```

## 4. 决策点

调用策略：

- [Git 分支策略](../policies/git-branch-policy.md)
- [质量与合入策略](../policies/quality-merge-policy.md)
- [Issue 调度策略](../policies/issue-scheduling-policy.md)

核心决策：

- dirty worktree 时是否阻断。
- 是否创建新 worktree。
- 是否允许自动 commit。
- commit message 是否符合规范。
- 是否允许自动 push。
- 是否允许自动 PR/MR。
- 合并冲突是否自动修复、返工还是升级人工。
- 下游 issue 是否需要基于最新 integration branch 重跑。

## 5. 用户改动保护

默认禁止：

- 自动 reset。
- 自动 clean。
- 自动 checkout 覆盖用户文件。
- 自动 stash 用户改动。
- force push。
- 删除未合并分支。

遇到 dirty worktree 时，系统应停止自动写入，并给出可选处理：

- 用户自行 commit。
- 用户自行 stash。
- 用户授权创建独立 workspace。
- 用户授权保存 patch。

## 6. 配置入口

- `.moyuan/repository.yaml`
- `.moyuan/policies/orchestration.yaml`
- `.moyuan/policies/permissions.yaml`
- `.moyuan/policies/release.yaml`

远程 Git Provider 细节见 [Git Provider 接入配置](../git-provider-integration.md)。

## 7. Workspace 产物

```text
.moyuan/worktrees/
.moyuan/lifecycle/branches/
.moyuan/lifecycle/merge-reports/
.moyuan/lifecycle/pull-requests/
```

## 8. 日志与审计

必须记录：

- branch created。
- worktree created。
- fetch、pull、rebase、merge。
- merge conflict。
- commit created。
- push attempted/completed。
- PR/MR created/updated。
- protected user changes detected。

日志流：

- `git`
- `run`
- `quality`
- `audit`
- `error`

当前实现基线：

- Git Provider plan 已支持创建、查询、列表和同步状态记录。
- PR/MR plan 会记录 provider、base branch、target branch、remote link、remote status、preview decision、create decision 和 sync decision。
- 默认只生成受控计划和 preview；真实 GitHub/Gitee PR/MR create 必须通过 approval、authz、secret resolver 和写开关，不自动合并。

## 9. 验收标准

- 每个 running issue 有独立 branch/worktree。
- 用户未提交改动不会被覆盖。
- 合入前一定有质量报告和 review 报告。
- integration branch 合流失败会阻断下游。
- release branch 不会从未通过门禁的代码创建。
- 自动 commit 必须满足工程流程规范。
