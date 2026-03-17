-- Migration: Add IMMUTABLE wrapper for to_tsvector and enable FTS GIN indexes
-- This migration addresses the TODOs in previous migrations by creating an IMMUTABLE
-- wrapper function for to_tsvector, allowing us to create functional GIN indexes.
--
-- Background: PostgreSQL's to_tsvector('english', text) is marked VOLATILE because
-- the text search configuration name is resolved at runtime. By creating a wrapper
-- that hard-codes the configuration, we can mark it IMMUTABLE and use it in indexes.

-- ============================================
-- STEP 1: Create IMMUTABLE wrapper function
-- ============================================

-- This function wraps to_tsvector with a hard-coded 'english' configuration,
-- allowing it to be marked IMMUTABLE for use in functional indexes.
CREATE OR REPLACE FUNCTION to_tsvector_immutable(text)
RETURNS tsvector
LANGUAGE sql
IMMUTABLE PARALLEL SAFE
AS $$
    SELECT to_tsvector('english', $1);
$$;

COMMENT ON FUNCTION to_tsvector_immutable(text) IS
    'IMMUTABLE wrapper for to_tsvector with hard-coded english configuration. Required for functional GIN indexes on tsvector expressions.';

-- ============================================
-- STEP 2: Add FTS GIN index for scenes
-- ============================================

-- Full-text search on scenes: name + description
-- Excludes soft-deleted scenes
CREATE INDEX IF NOT EXISTS idx_scenes_name_desc_tags_fts ON scenes
USING GIN(
    to_tsvector_immutable(
        COALESCE(name, '') || ' ' ||
        COALESCE(description, '')
    )
)
WHERE deleted_at IS NULL;

COMMENT ON INDEX idx_scenes_name_desc_tags_fts IS
    'Full-text search index on scene name and description using IMMUTABLE wrapper';

-- ============================================
-- STEP 3: Add FTS GIN index for events
-- ============================================

-- Full-text search on events: title
-- Excludes soft-deleted and cancelled events
CREATE INDEX IF NOT EXISTS idx_events_title_tags_fts ON events
USING GIN(
    to_tsvector_immutable(
        COALESCE(title, '')
    )
)
WHERE deleted_at IS NULL AND cancelled_at IS NULL;

COMMENT ON INDEX idx_events_title_tags_fts IS
    'Full-text search index on event title using IMMUTABLE wrapper';

-- ============================================
-- STEP 4: Add FTS GIN index for posts
-- ============================================

-- Full-text search on posts: text content
-- Excludes soft-deleted posts
CREATE INDEX IF NOT EXISTS idx_posts_text_fts ON posts
USING GIN(
    to_tsvector_immutable(COALESCE(text, ''))
)
WHERE deleted_at IS NULL;

COMMENT ON INDEX idx_posts_text_fts IS
    'Full-text search index on post text content using IMMUTABLE wrapper';
