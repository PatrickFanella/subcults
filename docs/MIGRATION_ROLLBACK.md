# Migration Rollback Documentation & Guard Rails

## Overview

This document provides rollback strategies for every type of database migration used in Subcults. All migrations are managed via [golang-migrate](https://github.com/golang-migrate/migrate) with paired up/down files in the `migrations/` directory.

## Migration Types & Rollback Strategies

### 1. Add Column

**Up**: `ALTER TABLE x ADD COLUMN y ...`
**Down**: `ALTER TABLE x DROP COLUMN y`

**Rollback Strategy**:

- Safe to rollback if no application code depends on the new column yet.
- Check that no running API/indexer instances reference the column before rolling back.
- If the column has a `NOT NULL` constraint with a default, the rollback is clean.

**Guard Rails**:

- Always add columns as `NULL` or with a `DEFAULT` first, then backfill, then add `NOT NULL` in a separate migration.
- Never combine column addition with data migration in one step.

```bash
# Rollback one migration
./scripts/migrate.sh down 1
```

### 2. Add Index

**Up**: `CREATE INDEX ...`
**Down**: `DROP INDEX ...`

**Rollback Strategy**:

- Always safe to rollback — indexes are not referenced by application code directly.
- Large index creation may lock the table; use `CONCURRENTLY` for production.

**Guard Rails**:

- Use `CREATE INDEX CONCURRENTLY` for tables with active traffic.
- Note: `CONCURRENTLY` cannot run inside a transaction. golang-migrate wraps migrations in transactions by default; for concurrent indexes, set `x-no-transaction` in the migration file or use a raw SQL execution.

### 3. Add Table

**Up**: `CREATE TABLE ...`
**Down**: `DROP TABLE ...`

**Rollback Strategy**:

- Safe if no data has been written to the table yet.
- If data exists, create an archive snapshot before rolling back.

**Guard Rails**:

- Never drop a table that contains user data without confirming an archive exists.
- Use `DROP TABLE IF EXISTS` in down migrations to handle partial states.

### 4. Modify Column Type

**Up**: `ALTER TABLE x ALTER COLUMN y TYPE ...`
**Down**: `ALTER TABLE x ALTER COLUMN y TYPE ... (original)`

**Rollback Strategy**:

- Risky if the type change is lossy (e.g., `TEXT` → `INTEGER` loses non-numeric values).
- Always test the down migration with production-like data in staging first.

**Guard Rails**:

- For lossy changes, use a multi-step approach:
  1. Add new column with new type
  2. Backfill new column from old
  3. Swap application references
  4. Drop old column (separate migration)

### 5. Destructive Changes (Drop Column / Drop Table)

**Up**: `ALTER TABLE x DROP COLUMN y` or `DROP TABLE x`
**Down**: _(cannot fully restore data)_

**Rollback Strategy**:

- **This is irreversible for data.** The down migration can recreate the structure but not the data.
- Before applying: take a full table snapshot.

**Guard Rails**:

- **Mandatory**: Archive affected data to cold storage before applying.
- **Mandatory**: Run in staging first with production data copy.
- **Mandatory**: Require explicit approval from a second team member.
- **Never** drop columns or tables without a 2-week deprecation window where application code no longer reads/writes the column.

### 6. Add/Modify Constraints

**Up**: `ALTER TABLE x ADD CONSTRAINT ...`
**Down**: `ALTER TABLE x DROP CONSTRAINT ...`

**Rollback Strategy**:

- Safe to rollback unless application code depends on the constraint for correctness.
- `NOT NULL` constraints may fail to add if existing data has NULLs — fix data first.

**Guard Rails**:

- Validate constraints with `NOT VALID` first, then `VALIDATE CONSTRAINT` in a separate step to avoid locking.
- Check for violations before adding: `SELECT count(*) FROM x WHERE column IS NULL`.

## Pre-Rollback Checklist

Before executing any rollback, verify the following:

- [ ] **No active backfill** — check `SELECT * FROM backfill_checkpoints WHERE status = 'running'`
- [ ] **Replication lag** — if using read replicas, ensure lag < 1s: `SELECT pg_last_wal_replay_lsn()`
- [ ] **Audit log consistent** — check latest audit entries for the affected table
- [ ] **Application instances** — confirm which version of API/indexer is running and whether it depends on the migration being rolled back
- [ ] **Archive snapshot** — for destructive changes, confirm archive exists and is verified
- [ ] **Staging tested** — rollback has been executed successfully in staging

## Rollback Decision Record Template

When a rollback is performed, create a record in `docs/rollbacks/` using this template:

```markdown
# Rollback: Migration NNNNNN

**Date**: YYYY-MM-DD HH:MM UTC
**Performed by**: @username
**Migration**: NNNNNN_description.up.sql
**Reason**: [Brief explanation of why rollback was needed]

## Impact Assessment

- **Tables affected**: [list]
- **Data loss**: [none / describe what was lost]
- **Downtime**: [duration]
- **Users affected**: [estimate]

## Steps Executed

1. [Step-by-step record of what was done]

## Verification

- [ ] Schema matches expected state
- [ ] Application starts without errors
- [ ] Health checks pass
- [ ] Sample queries return expected results

## Follow-up Actions

- [ ] [Any remediation needed]
- [ ] [Root cause analysis]
```

## Guard Rail Summary

| Rule                                                | Enforcement                                       |
| --------------------------------------------------- | ------------------------------------------------- |
| No column drops without archive                     | Pre-deploy check in `scripts/pre-deploy-check.sh` |
| Staging dry-run required for destructive migrations | CI pipeline gate                                  |
| `CONCURRENTLY` for indexes on active tables         | Code review checklist                             |
| Multi-step for type changes                         | Code review checklist                             |
| `NOT VALID` + `VALIDATE` for constraints            | Code review checklist                             |
| Down migration required for every up                | `scripts/migrate.sh` validates paired files       |
| No data migration in DDL migration                  | Code review checklist                             |

## Integration with Deployment

The deployment script (`scripts/deploy-production.sh`) should:

1. **Before migration**: Record current version via `./scripts/migrate.sh version`
2. **Apply migration**: `./scripts/migrate.sh up`
3. **Verify**: Run health checks and smoke tests
4. **On failure**: Execute `./scripts/migrate.sh down 1` to rollback the last applied migration
5. **Log**: Create rollback decision record if rollback was triggered

## Common Rollback Commands

```bash
# Check current migration version
./scripts/migrate.sh version

# Rollback the last migration
./scripts/migrate.sh down 1

# Rollback the last N migrations
./scripts/migrate.sh down N

# Force-set version (use when migration is in dirty state)
./scripts/migrate.sh force VERSION_NUMBER

# Verify schema state after rollback
psql "$DATABASE_URL" -c "\dt"
psql "$DATABASE_URL" -c "\d table_name"
```

## Related Documents

- [Migrations README](../migrations/README.md) — migration file format and naming
- [Deployment Checklist](OPERATIONS.md) — deployment procedure
- [Data Retention Policy](legal/DATA_RETENTION_POLICY.md) — archive requirements
- [Backfill Plan](BACKFILL_PLAN.md) — handling rollbacks during backfill operations
