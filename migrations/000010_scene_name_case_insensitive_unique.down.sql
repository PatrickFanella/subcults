-- Migration rollback: Remove case-insensitive unique constraint on scene names

DROP INDEX IF EXISTS idx_scenes_owner_name_unique;
