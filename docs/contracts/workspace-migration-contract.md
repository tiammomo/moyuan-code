# Workspace 迁移契约

## 1. 目标

定义 `.moyuan/` 工作空间在 schema_version 变化时如何迁移、回滚和兼容，避免后续版本升级破坏已有项目。

## 2. 版本规则

- 每个配置文件必须包含 `schema_version`。
- MVP 只支持 `schema_version: 1`。
- 新增字段必须有默认值或迁移步骤。
- 删除字段必须先 deprecated，再移除。
- 迁移必须可审计。

## 3. 迁移接口

```ts
interface Migration {
  id: string;
  from_version: number;
  to_version: number;
  description: string;
  affected_files: string[];
  precheck(root: string): Promise<MigrationCheck>;
  apply(root: string): Promise<MigrationResult>;
  rollback(root: string): Promise<MigrationResult>;
}

interface MigrationCheck {
  ok: boolean;
  blockers: string[];
  warnings: string[];
}

interface MigrationResult {
  ok: boolean;
  changed_files: string[];
  backup_path?: string;
  errors: string[];
}
```

## 4. 迁移流程

```text
detect workspace schema versions
  -> find migration path
  -> run precheck
  -> create backup
  -> apply migration
  -> validate workspace
  -> write migration event
```

## 5. 回滚流程

```text
migration failed
  -> stop all writes
  -> restore backup
  -> validate restored workspace
  -> write rollback event
  -> mark workspace migration_failed
```

## 6. 禁止事项

- 不允许无备份迁移。
- 不允许迁移时改源码目录。
- 不允许迁移时删除用户自定义配置。
- 不允许把密码、API Token、session secret 或云凭证明文写入迁移备份。
- 不允许静默修复无法识别字段。
- 不允许跨多个 major schema version 自动跳迁，除非有明确迁移链。

## 7. 迁移记录

```text
.moyuan/migrations/
  events.jsonl
  backups/
  reports/
```

每次迁移记录：

- migration id。
- from_version。
- to_version。
- affected files。
- backup path。
- validation result。
- rollback result，如适用。

## 8. 验收用例

- schema_version 1 工作空间无需迁移。
- 未知 schema_version 必须阻断。
- 迁移前必须创建 backup。
- 迁移后必须运行 schema validation。
- 迁移失败必须能回滚到原工作空间。
