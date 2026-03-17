-- Migration: Add deleted_at column to alliances table for soft delete support
-- Required for alliance CRUD endpoints

ALTER TABLE alliances ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Index on deleted_at for filtering active alliances
CREATE INDEX IF NOT EXISTS idx_alliances_deleted_at ON alliances(deleted_at);

-- Index for active alliances (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_alliances_active ON alliances(from_scene_id, to_scene_id)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN alliances.deleted_at IS 'Soft delete timestamp. When set, alliance is excluded from trust computation.';
