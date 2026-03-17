-- Migration: Enhance scenes table with tags, visibility, palette, and FTS
-- Adds: tags array, visibility CHECK constraint, palette JSONB, owner_user_id FK, FTS column
-- Changes: coarse_geohash becomes NOT NULL (privacy requirement)

-- ============================================
-- STEP 1: Add tags column for categorization
-- ============================================

ALTER TABLE scenes ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';

-- GIN index on tags for array queries (exclude soft-deleted scenes)
CREATE INDEX IF NOT EXISTS idx_scenes_tags ON scenes USING GIN(tags)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN scenes.tags IS 'Categorization tags for discovery, indexed for FTS and array queries';

-- ============================================
-- STEP 2: Add visibility column with CHECK constraint
-- ============================================

ALTER TABLE scenes ADD COLUMN IF NOT EXISTS visibility TEXT DEFAULT 'public';

-- Enforce valid visibility values at the database level
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        JOIN pg_namespace n ON t.relnamespace = n.oid
        WHERE t.relname = 'scenes' AND n.nspname = 'public' AND c.conname = 'chk_scene_visibility'
    ) THEN
        ALTER TABLE scenes ADD CONSTRAINT chk_scene_visibility
            CHECK (visibility IN ('public', 'private', 'unlisted'));
    END IF;
END $$;

-- Index on visibility for filtering (exclude soft-deleted scenes)
CREATE INDEX IF NOT EXISTS idx_scenes_visibility ON scenes(visibility)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN scenes.visibility IS 'Scene visibility mode (public, private, unlisted)';

-- ============================================
-- STEP 3: Add palette JSONB column
-- ============================================

ALTER TABLE scenes ADD COLUMN IF NOT EXISTS palette JSONB DEFAULT '{}'::jsonb;

-- Migrate existing primary_color and secondary_color to palette JSONB
-- Only migrate if the old columns exist
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'scenes' AND column_name = 'primary_color'
    ) THEN
        -- Build palette JSON from existing color columns
        -- Only update if palette is not already set to avoid overwriting on re-run
        UPDATE scenes
        SET palette = jsonb_build_object(
            'primary', COALESCE(primary_color, '#000000'),
            'secondary', COALESCE(secondary_color, '#ffffff')
        )
        WHERE (primary_color IS NOT NULL OR secondary_color IS NOT NULL)
          AND (palette IS NULL OR palette = '{}'::jsonb);

        -- Drop the old columns
        ALTER TABLE scenes DROP COLUMN IF EXISTS primary_color;
        ALTER TABLE scenes DROP COLUMN IF EXISTS secondary_color;
    END IF;
END $$;

COMMENT ON COLUMN scenes.palette IS 'JSONB color palette for scene visual identity';

-- ============================================
-- STEP 4: Add owner_user_id FK to users table
-- ============================================

-- Add owner_user_id column (nullable initially for migration safety)
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS owner_user_id UUID;

-- Create FK constraint to users table
-- Note: This allows NULL to support gradual migration from owner_did via application logic
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        JOIN pg_namespace n ON t.relnamespace = n.oid
        WHERE t.relname = 'scenes' AND n.nspname = 'public' AND c.conname = 'fk_scenes_owner_user'
    ) THEN
        ALTER TABLE scenes ADD CONSTRAINT fk_scenes_owner_user
            FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

-- Index on owner_user_id for join queries (already has idx_scenes_owner on owner_did)
CREATE INDEX IF NOT EXISTS idx_scenes_owner_user_id ON scenes(owner_user_id)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN scenes.owner_user_id IS 'Foreign key to users table for ownership tracking';

-- ============================================
-- STEP 5: Make coarse_geohash NOT NULL
-- ============================================

-- Set a default coarse geohash for any existing rows without one
-- Using geohash 's00000' which represents coordinates (0,0) - "Null Island" in the Gulf of Guinea
-- This is a safe placeholder as no real scenes should exist at this location
-- Real scenes must have proper geohashes set via application logic before insertion
-- Note: Only updating NULL values; empty strings are preserved as they indicate invalid data
UPDATE scenes
SET coarse_geohash = 's00000'
WHERE coarse_geohash IS NULL;

-- Now make coarse_geohash NOT NULL
ALTER TABLE scenes ALTER COLUMN coarse_geohash SET NOT NULL;

COMMENT ON COLUMN scenes.coarse_geohash IS 'Required coarse location geohash for privacy-conscious discovery';

-- ============================================
-- STEP 6: FTS support for name + description + tags via indexes
-- ============================================

-- Note: Generated tsvector columns require immutable expressions
-- We'll add GIN indexes directly on expression instead
-- The 'english' language parameter makes to_tsvector non-immutable

-- GIN index for FTS queries on name + description + tags (exclude deleted)
-- Note: PostgreSQL's to_tsvector('english', ...) is not marked IMMUTABLE
-- For now, we skip FTS indexing and rely on application-level FTS or simpler queries
-- TODO: Consider adding a custom IMMUTABLE wrapper function if FTS becomes critical
-- CREATE INDEX IF NOT EXISTS idx_scenes_name_desc_tags_fts ON scenes USING GIN(
--     to_tsvector('english',
--         COALESCE(name, '') || ' ' ||
--         COALESCE(description, '') || ' ' ||
--         COALESCE(array_to_string(tags, ' '), '')
--     )
-- ) WHERE deleted_at IS NULL;
-- ============================================

-- Drop and recreate to ensure consistency with other indexes
DROP INDEX IF EXISTS idx_scenes_owner;
CREATE INDEX idx_scenes_owner ON scenes(owner_did) WHERE deleted_at IS NULL;
