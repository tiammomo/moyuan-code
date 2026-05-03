# Memory 决策策略

## 1. 目标

决定什么时候检索 memory、什么时候记录 memory、如何处理候选、何时自动 compact、如何标记 stale 或冲突。

详细机制以 [Agent Memory 系统方案](../agent-memory-system.md) 为准。本文只把关键决策整理成可实现策略。

## 2. 输入事实

- user message。
- agent output。
- task type。
- project comprehension diff。
- quality/review result。
- release/deployment result。
- candidate memory。
- memory scope。
- current context token usage。
- stale evidence。
- conflict evidence。

## 3. 决策结果

- `RETRIEVE_MEMORY`
- `SKIP_RETRIEVE`
- `RECORD_CANDIDATE`
- `DROP_CANDIDATE`
- `COMPACT_REQUIRED`
- `MARK_STALE`
- `MERGE_DUPLICATE`
- `CONFLICT_REVIEW_REQUIRED`

## 4. Retrieve 决策树

```text
if task references previous decision/history/preference:
  RETRIEVE_MEMORY
else if task requires project background:
  RETRIEVE_MEMORY
else if user asks to continue previous plan:
  RETRIEVE_MEMORY
else if current task is simple and self-contained:
  SKIP_RETRIEVE
else:
  RETRIEVE_MEMORY(role_scoped)
```

## 5. Record 决策树

```text
if content is one-time operational instruction:
  DROP_CANDIDATE
else if content contains secret or sensitive raw value:
  DROP_CANDIDATE
else if content has long-term project value:
  RECORD_CANDIDATE
else if content corrects prior misunderstanding:
  RECORD_CANDIDATE
else if content is accepted architecture/quality/release decision:
  RECORD_CANDIDATE
else:
  DROP_CANDIDATE
```

## 6. Compact 决策树

```text
if run context tokens exceed threshold:
  COMPACT_REQUIRED
else if staging candidates exceed threshold:
  COMPACT_REQUIRED
else if after remote pull and many project facts changed:
  COMPACT_REQUIRED
else if after task complete and many run facts exist:
  COMPACT_REQUIRED
else if scheduled maintenance:
  COMPACT_REQUIRED
```

## 7. Stale 与冲突树

```text
if memory references deleted file:
  MARK_STALE
else if memory conflicts with latest project comprehension:
  MARK_STALE
else if two memories describe same entity with different values:
  CONFLICT_REVIEW_REQUIRED
else if candidates are semantically duplicated:
  MERGE_DUPLICATE
```

## 8. 安全规则

- 不保存完整 secret。
- 不保存 `.env` 明文。
- 不保存完整 prompt/response。
- 不保存大段源码。
- Memory 写入必须保留来源、时间、scope 和置信度。

## 9. 产物和日志

产物：

- `.moyuan/memory/candidates/`
- `.moyuan/memory/staging/`
- `.moyuan/memory/records/`
- `.moyuan/memory/compactions/`
- `.moyuan/memory/index/`

日志：

- `memory`
- `run`
- `agent`
- `audit`
- `error`

## 10. 关联配置

- `.moyuan/policies/memory.yaml`
- `.moyuan/models/routing.yaml`
- `.moyuan/agents/roles.yaml`

## 11. 验收用例

- 用户明确说“记住”时生成 candidate，但仍要过安全规则。
- secret 明文永不进入 memory。
- 远程 pull 后有模块变化时触发 stale 检查。
- context 超过阈值时自动 compact。
- 冲突 memory 需要 review，不直接覆盖。
