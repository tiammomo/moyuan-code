# 项目阅读理解策略

## 1. 目标

决定什么时候执行 full、incremental 或 diff comprehension，并决定哪些理解结果进入项目快照、哪些进入 memory candidates、哪些旧记忆需要标记为 stale。

## 2. 输入事实

- repository source type。
- 当前 Git branch。
- 当前 commit。
- 上一次 comprehension commit。
- changed files。
- diff size。
- dependency/config file changes。
- README/docs/ADR changes。
- build/test/lint command changes。
- deleted or moved files。
- current task type。

## 3. 决策结果

- `RUN_FULL_COMPREHENSION`
- `RUN_INCREMENTAL_COMPREHENSION`
- `RUN_DIFF_COMPREHENSION`
- `SKIP_WITH_REASON`
- `BLOCKED_NEEDS_USER_ACTION`

## 4. 决策树

```text
if project has no comprehension snapshot:
  RUN_FULL_COMPREHENSION
else if repository source changed:
  RUN_FULL_COMPREHENSION
else if default branch changed:
  RUN_INCREMENTAL_COMPREHENSION
else if dependency/config/build/test/lint files changed:
  RUN_INCREMENTAL_COMPREHENSION
else if README/docs/ADR changed:
  RUN_INCREMENTAL_COMPREHENSION
else if task completed and issue diff exists:
  RUN_DIFF_COMPREHENSION
else if changed files only affect generated artifacts:
  SKIP_WITH_REASON(generated_only)
else:
  RUN_INCREMENTAL_COMPREHENSION
```

远程同步：

```text
if git fetch/pull/rebase/merge completed:
  if changed commit range is empty:
    SKIP_WITH_REASON(no_change)
  else:
    RUN_INCREMENTAL_COMPREHENSION
```

## 5. 阻断条件

- 仓库路径不存在。
- Git 状态不可读。
- 远程认证失败。
- diff 无法计算。
- 关键配置文件损坏。
- 当前工作区 dirty 且任务需要代码写入。

## 6. 人工确认条件

- 仓库不是 Git 仓库，是否初始化 Git。
- 远程仓库认证失败，需要用户补充 token 或 SSH key。
- 项目过大，需要选择排除目录。
- 项目含敏感目录，需要确认扫描边界。

## 7. Memory 处理

进入 memory candidates：

- 稳定构建命令。
- 稳定测试命令。
- 稳定模块职责。
- 已确认技术栈。
- 用户明确的项目偏好。

标记 stale candidate：

- 指向已删除或移动文件的 memory。
- 与最新代码结构冲突的模块说明。
- 被替换的构建、测试、部署命令。

## 8. 产物和日志

产物：

- `.moyuan/comprehension/project-profile.md`
- `.moyuan/comprehension/module-map.md`
- `.moyuan/comprehension/dependency-map.md`
- `.moyuan/comprehension/commands.md`
- `.moyuan/comprehension/events.jsonl`

日志：

- `git`
- `run`
- `memory`
- `audit`
- `error`

## 9. 关联配置

- `.moyuan/policies/comprehension.yaml`
- `.moyuan/repository.yaml`
- `.moyuan/policies/memory.yaml`

## 10. 验收用例

- 新增项目时必须执行 full comprehension。
- pull 后有代码变化时必须执行 incremental comprehension。
- 任务完成后必须执行 diff comprehension。
- 只有生成图片变化时可以跳过理解并记录原因。
- 删除文件后，相关 memory 被标记为 stale candidate。
