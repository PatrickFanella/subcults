-- Rollback: Enhance scenes table migration
-- Removes: tags, visibility, palette, owner_user_id, FTS column
-- Reverts: coarse_geohash back to nullable

-- Drop FTS column and index
DROP INDEX IF EXISTS idx_scenes_name_desc_tags_fts;
ALTER TABLE scenes DROP COLUMN IF EXISTS name_desc_tags_fts;

-- Restore coarse_geohash to nullable
ALTER TABLE scenes ALTER COLUMN coarse_geohash DROP NOT NULL;

-- Drop owner_user_id FK and column
DROP INDEX IF EXISTS idx_scenes_owner_user_id;
ALTER TABLE scenes DROP CONSTRAINT IF EXISTS fk_scenes_owner_user;
ALTER TABLE scenes DROP COLUMN IF EXISTS owner_user_id;

-- Restore primary_color and secondary_color from palette
-- Extract colors back from JSONB before dropping palette
-- Limitation: Only extracts 'primary' and 'secondary' keys; any additional palette keys will be lost
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS primary_color VARCHAR(7);
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS secondary_color VARCHAR(7);

UPDATE scenes 
SET 
    primary_color = palette->>'primary',
    secondary_color = palette->>'secondary'
WHERE palette IS NOT NULL AND palette != '{}'::jsonb;

-- Drop palette column
ALTER TABLE scenes DROP COLUMN IF EXISTS palette;

-- Drop visibility column and related objects
DROP INDEX IF EXISTS idx_scenes_visibility;
ALTER TABLE scenes DROP CONSTRAINT IF EXISTS chk_scene_visibility;
ALTER TABLE scenes DROP COLUMN IF EXISTS visibility;

-- Drop tags column and index
DROP INDEX IF EXISTS idx_scenes_tags;
ALTER TABLE scenes DROP COLUMN IF EXISTS tags;

-- Restore original idx_scenes_owner with WHERE clause from 000000_initial_schema.up.sql
DROP INDEX IF EXISTS idx_scenes_owner;
CREATE INDEX idx_scenes_owner ON scenes(owner_did) WHERE deleted_at IS NULL;
