-- Rollback: Remove scene-level moderation status and audit fields

-- Drop constraints
ALTER TABLE scenes DROP CONSTRAINT IF EXISTS chk_moderation_consistency;
ALTER TABLE scenes DROP CONSTRAINT IF EXISTS chk_moderation_status;

-- Drop indexes
DROP INDEX IF EXISTS idx_scenes_visible;
DROP INDEX IF EXISTS idx_scenes_moderation_timestamp;
DROP INDEX IF EXISTS idx_scenes_moderation_status;

-- Drop columns
ALTER TABLE scenes DROP COLUMN IF EXISTS moderation_timestamp;
ALTER TABLE scenes DROP COLUMN IF EXISTS moderated_by;
ALTER TABLE scenes DROP COLUMN IF EXISTS moderation_reason;
ALTER TABLE scenes DROP COLUMN IF EXISTS moderation_status;
