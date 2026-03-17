-- Migration: Add scene-level moderation status and audit fields
-- Enables admin visibility controls and moderation tracking
-- Related to issue #452 (scene muting/hiding)

-- Add moderation status column
-- Status values: visible (default), hidden (admin muted), flagged (under review), suspended (serious violations)
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS moderation_status VARCHAR(50) DEFAULT 'visible';

-- Add textual reason for moderation action
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS moderation_reason TEXT;

-- Add audit fields for tracking who moderated and when
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS moderated_by VARCHAR(255); -- User DID of moderator
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS moderation_timestamp TIMESTAMPTZ;

-- Index on moderation_status for efficient filtering in search queries
CREATE INDEX IF NOT EXISTS idx_scenes_moderation_status ON scenes(moderation_status) 
WHERE deleted_at IS NULL;

-- Index for querying recently moderated scenes (audit trail)
CREATE INDEX IF NOT EXISTS idx_scenes_moderation_timestamp ON scenes(moderation_timestamp DESC) 
WHERE deleted_at IS NULL AND moderation_timestamp IS NOT NULL;

-- Composite index for search queries that exclude hidden/suspended scenes
CREATE INDEX IF NOT EXISTS idx_scenes_visible ON scenes(moderation_status, created_at DESC) 
WHERE deleted_at IS NULL AND moderation_status IN ('visible', 'flagged');

-- Add constraint to enforce valid status values
ALTER TABLE scenes ADD CONSTRAINT IF NOT EXISTS chk_moderation_status 
CHECK (moderation_status IN ('visible', 'hidden', 'flagged', 'suspended'));

-- Add constraint to ensure moderation fields are consistent:
-- Either unmoderated (visible) with all audit fields NULL,
-- or moderated (non-visible) with required audit fields populated.
ALTER TABLE scenes ADD CONSTRAINT IF NOT EXISTS chk_moderation_consistency 
CHECK (
    (
        moderation_status = 'visible'
        AND moderated_by IS NULL
        AND moderation_timestamp IS NULL
        AND moderation_reason IS NULL
    )
    OR (
        moderation_status != 'visible'
        AND moderated_by IS NOT NULL
        AND moderation_timestamp IS NOT NULL
    )
);

COMMENT ON COLUMN scenes.moderation_status IS 'Scene moderation status: visible (normal), hidden (admin muted), flagged (under review), suspended (serious violations)';
COMMENT ON COLUMN scenes.moderation_reason IS 'Human-readable reason for moderation action (optional)';
COMMENT ON COLUMN scenes.moderated_by IS 'User DID of the admin/moderator who took the action';
COMMENT ON COLUMN scenes.moderation_timestamp IS 'Timestamp when moderation action was taken';
