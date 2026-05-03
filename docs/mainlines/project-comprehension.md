# 项目接入与阅读理解主线

## 1. 目标

项目接入与阅读理解主线负责把一个本地路径或远程 Git 仓库接入 Moyuan，并建立后续 Agent 开发所需的项目画像、模块地图、命令清单、风险清单、skills 推荐和 memory candidates。

这条主线的关键要求：

- 用户提供本地路径或远程仓库地址后即可接入。
- 首次接入必须执行 full comprehension。
- 每次 fetch、pull、rebase 或 merge 后必须执行 incremental comprehension。
- 每次任务完成后执行 diff comprehension，更新受影响模块和 memory candidates。
- 阅读理解结果必须进入 `.moyuan/comprehension/`，不能只停留在模型上下文中。

## 2. 输入与输出

输入：

- 本地仓库路径或远程 Git URL。
- Git provider、认证方式和默认分支。
- 项目源码、README、docs、依赖文件、构建脚本和测试脚本。
- 上一次 comprehension 快照和当前 Git diff。

输出：

- project profile。
- module map。
- dependency map。
- commands。
- risks。
- skills recommendation。
- memory candidates。
- comprehension event log。

## 3. 端到端流程

```text
project add
  -> validate source
  -> clone or bind local repository
  -> detect git provider and branch state
  -> initialize .moyuan workspace
  -> run full comprehension
  -> write project profile and module map
  -> recommend skills and roles
  -> generate memory candidates
  -> write comprehension event
```

远程同步后：

```text
git fetch/pull/rebase
  -> detect changed commits
  -> collect changed files and diff
  -> run incremental comprehension
  -> update project profile patch
  -> update module map patch
  -> mark stale memory candidates
  -> write comprehension event
```

任务完成后：

```text
issue completed
  -> collect issue diff
  -> run diff comprehension
  -> update affected module facts
  -> generate lessons and quality memory candidates
  -> write run report
```

## 4. 决策点

调用策略：

- [项目阅读理解策略](../policies/project-comprehension-policy.md)
- [Memory 决策策略](../policies/memory-decision-policy.md)
- [Git 分支策略](../policies/git-branch-policy.md)

核心决策：

- 什么时候执行 full comprehension。
- 什么时候执行 incremental comprehension。
- 什么时候只执行 diff comprehension。
- 哪些内容进入长期 memory，哪些只保留为项目理解快照。
- 远程拉取失败、认证失败或 dirty worktree 时是否阻断。

## 5. 配置入口

- `.moyuan/repository.yaml`
- `.moyuan/policies/comprehension.yaml`
- `.moyuan/policies/memory.yaml`
- `.moyuan/skills/enabled.yaml`
- `.moyuan/skills/registry.yaml`
- `.moyuan/skills/bindings.yaml`

字段规则见 [配置 Schema 规则](../configuration-schema-spec.md)。

## 6. Workspace 产物

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

## 7. 日志与审计

必须记录：

- repository source。
- clone、fetch、pull、checkout、rebase、merge。
- comprehension mode。
- base commit 和 head commit。
- changed files。
- 生成或更新的产物路径。
- memory candidates 数量。
- 失败原因和恢复动作。

日志流：

- `git`
- `run`
- `memory`
- `audit`
- `error`

## 8. 验收标准

- 本地路径和远程仓库都能接入。
- 首次接入一定生成项目画像和模块地图。
- 远程分支拉取后一定执行增量理解。
- Agent 开发前能读取最新项目理解结果。
- 项目理解不会写入 secret 明文。
- 与最新代码冲突的旧 memory 能被标记为 stale candidate。
