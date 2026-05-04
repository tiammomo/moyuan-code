# 持久化与并发一致性

状态：ready
责任角色：core_engineer + orchestrator_owner
最后更新：2026-05-03

本文定义 Moyuan Code 在 `.moyuan/` 工作空间中的状态持久化、GORM State Store、并发写入、锁、事务、崩溃恢复和幂等规则。它是 Workspace Manager 和 State Store 的实现依据。

## 1. 目标

- 保证多个 issue、Subagent、Runtime 和质量检查并发时不会写坏状态。
- 保证进程崩溃、命令中断、机器重启后可以恢复或回滚。
- 保证日志、审计、生命周期状态和 Memory 写入可追踪。
- 避免 AI agent 并发写同一状态文件造成丢失更新。

## 2. 边界

本文只定义状态一致性原则和最小机制。

不在本文展开：

- `.moyuan/` 完整目录职责，见 [项目工作空间规范](./project-workspace-spec.md)。
- schema_version 迁移接口，见 [Workspace 迁移契约](./contracts/workspace-migration-contract.md)。
- 日志事件字段，见 [日志与审计事件契约](./contracts/logging-audit-event-contract.md)。
- Issue 调度规则，见 [Issues 编排与并发调度](./issue-orchestration.md)。

## 3. 状态分类

| 类型 | 示例 | 写入要求 |
| --- | --- | --- |
| 配置状态 | `project.yaml`、`repository.yaml`、`policies/*.yaml` | 低频写，必须 schema 校验和备份 |
| 生命周期状态 | issues、runs、schedules、quality、reviews | 高频写，必须原子写和版本检查 |
| 日志事件 | run、agent、model、git、audit、error | append-only，允许分片 |
| Runtime 输出 | stdout、stderr、diff、session output | 大文件独立存储，状态中只保存引用 |
| Memory 状态 | candidates、staging、facts、compact | 先暂存，后异步合并 |
| 临时状态 | lock、transaction、tmp、cache | 可清理，但必须可恢复 |
| 查询型状态 | project registry、后续 issue/run index | 由 GORM + SQLite 承载，不能替代审计日志 |

## 4. 写入原则

所有状态写入必须满足：

- 通过 Workspace API，不直接散落文件写入。
- 写前读取当前 `version` 或 `etag`。
- 写入临时文件后原子 rename。
- 写入完成后记录 log event。
- 高风险状态写入记录 audit event。
- schema 校验失败时不覆盖旧文件。

推荐写入流程：

```text
load current state
  -> validate input
  -> acquire scoped lock
  -> check version
  -> write tmp file
  -> fsync if supported
  -> atomic rename
  -> append log event
  -> release lock
```

## 5. 锁策略

锁粒度必须尽量小。

| 锁 | 保护范围 | 示例 |
| --- | --- | --- |
| project lock | 初始化、迁移、全局配置变更 | `.moyuan/.locks/project.lock` |
| issue lock | 单个 issue 状态 | `.moyuan/.locks/issues/issue-001.lock` |
| graph lock | Issue Graph 和 schedule 更新 | `.moyuan/.locks/graphs/epic-001.lock` |
| run lock | 单个 run 状态和输出索引 | `.moyuan/.locks/runs/run-001.lock` |
| memory lock | staging 合并和 compact | `.moyuan/.locks/memory.lock` |
| release lock | release branch、tag、部署计划 | `.moyuan/.locks/releases/release-001.lock` |

锁规则：

- 锁必须有 owner、pid、created_at、expires_at。
- 锁超时不能直接删除，必须先判断 owner 是否存活。
- 需要多个锁时按固定顺序获取：project -> graph -> issue -> run -> memory -> release。
- 获取失败必须返回结构化 blocked reason。

## 6. 乐观并发控制

每个可变状态对象必须带版本。

```json
{
  "id": "issue-001",
  "version": 7,
  "updated_at": "2026-05-03T00:00:00Z",
  "status": "READY"
}
```

更新时必须声明 expected version：

```text
update issue-001 expected_version=7
```

如果当前版本不是 7：

- 不覆盖。
- 重新读取状态。
- 判断是否可合并。
- 不可合并时进入 conflict recovery。

## 7. 事务与恢复

跨多个文件的状态变更必须写 transaction journal。

```text
.moyuan/tmp/transactions/tx-<id>.json
```

事务记录至少包含：

- `tx_id`
- `status`: pending | committed | rolled_back | interrupted
- `intent`
- `affected_paths`
- `before_refs`
- `after_refs`
- `created_at`
- `owner`

崩溃恢复流程：

```text
scan pending transactions
  -> verify affected paths
  -> if after files complete: mark committed
  -> if partial writes: restore before refs or mark interrupted
  -> emit recovery log
  -> require manual review if uncertain
```

不能自动恢复的情况：

- Git merge 已部分完成且存在冲突。
- release tag 已推送远程。
- production deployment 已开始执行。
- Memory compact 发现来源冲突。

这些情况必须进入人工确认。

## 8. Append-only 日志

日志采用 append-only JSONL 或分片文件。

要求：

- 每条日志有 `trace_id`、`run_id` 或相关对象 id。
- audit log 不允许静默修改。
- 日志写入失败不能阻断低风险只读操作，但必须阻断高风险写操作。
- 日志可以轮转，但索引必须保留。

## 8.1 GORM State Store

Phase 1 起引入 `GORM + SQLite` 作为本地 State Store 基线。

规则：

- 默认数据库路径为 `.moyuan/state.db`。
- GORM model 只能承载查询、索引和 API 读取需要的状态，不替代 JSONL 审计日志。
- 文件化状态和数据库状态同时存在时，以可审计文件和日志作为恢复依据。
- 写入流程应先完成领域状态变更，再同步写入 GORM index；同步失败必须返回错误或记录恢复事件，不能静默制造不一致。
- 数据库迁移必须由 `internal/store` 统一管理，业务模块不能自行打开数据库并迁移表。
- 团队版迁移到 PostgreSQL 时，GORM model 字段、对象 id 和状态机语义保持兼容。

## 9. Memory 一致性

Memory 不直接同步写长期库。

推荐流程：

```text
memory candidate
  -> staging append
  -> dedup window
  -> async write
  -> compact
  -> index update
```

规则：

- Record Gate 只决定是否进入候选，不直接写长期记忆。
- Staging 去重失败时保留原始候选和冲突记录。
- Compact 必须保留来源引用。
- 检索索引更新失败时，长期记录仍保留，但标记 `index_status=stale`。

## 10. Runtime 输出一致性

Native Runtime 可能产生文件变更、命令输出和会话状态。

规则：

- Runtime 开始前记录 base commit 和 diff snapshot。
- Runtime 结束后记录 changed files 和 diff snapshot。
- 输出契约校验通过前，不更新 issue 为 accepted。
- Runtime 失败时保留输出引用，便于 review 和 retry。
- 同一 issue 的 retry 必须创建新 run，不覆盖旧 run。

## 11. Git 状态一致性

Git 操作需要双重状态：

- Git 仓库真实状态。
- `.moyuan/` 生命周期状态。

规则：

- 分支创建、merge、tag、push 前先记录 intent。
- 成功后写 committed event。
- 失败后记录 recovery hint。
- `.moyuan/` 状态不能声称已合入，除非 Git merge 已确认成功。
- 远程 push 成功后不能自动回滚本地状态而不记录 audit。

## 12. 幂等性

以下操作必须设计为幂等或可检测重复：

- 项目初始化。
- full/incremental comprehension 触发。
- issue graph 写入。
- schedule 计算。
- Subagent dispatch。
- quality gate 执行。
- memory candidate 写入。
- release note 生成。

幂等 key 建议：

```text
project_id + operation + input_hash + base_commit + actor_id
```

## 13. 权限和安全

- 锁文件、事务日志和状态文件不得保存 secret 明文。
- 临时目录清理不能删除 audit log。
- 生产部署、Git push、tag、密钥访问等高风险状态必须有 audit event。
- 并发冲突不能通过强制覆盖解决，必须明确 owner 和恢复路径。

## 14. 验收标准

进入实现前，本文必须能支撑：

- 两个 issue 并发运行时不会互相覆盖状态。
- 进程在写 issue 状态中途崩溃后可以恢复。
- Runtime 输出失败不会污染 accepted 状态。
- Memory compact 失败不会丢失原始候选。
- Git merge 成功与 `.moyuan/` 合入状态保持一致。
- 所有高风险写入都有日志和审计。

## 15. 相关文档

- [项目工作空间规范](./project-workspace-spec.md)
- [Workspace 迁移契约](./contracts/workspace-migration-contract.md)
- [日志与审计事件契约](./contracts/logging-audit-event-contract.md)
- [Issues 编排与并发调度](./issue-orchestration.md)
- [框架自身测试策略](./framework-testing-strategy.md)
