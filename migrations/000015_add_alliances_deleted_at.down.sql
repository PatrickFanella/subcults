-- Rollback: Remove deleted_at column from alliances table

DROP INDEX IF EXISTS idx_alliances_active;
DROP INDEX IF EXISTS idx_alliances_deleted_at;

ALTER TABLE alliances DROP COLUMN IF EXISTS deleted_at;
