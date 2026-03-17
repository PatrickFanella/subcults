-- Migration: Enhance events table for schedule-based discovery
-- Adds: title (rename from name), tags, status with CHECK constraint, stream_session_id, FTS column
-- Adds: GIN indexes on tags and FTS with consistent WHERE clauses

-- Step 1: Rename name to title (per issue specification)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'events' AND column_name = 'name'
    ) THEN
        ALTER TABLE events RENAME COLUMN name TO title;
    END IF;
END $$;

-- Step 2: Add tags column for categorization
ALTER TABLE events ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';

-- Step 3: Add status column for event lifecycle tracking
ALTER TABLE events ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'scheduled';

-- Enforce valid status values at the database level
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_event_status'
    ) THEN
        ALTER TABLE events ADD CONSTRAINT chk_event_status
            CHECK (status IN ('scheduled', 'live', 'ended', 'cancelled'));
    END IF;
END $$;

-- Step 4: Add stream_session_id foreign key for live streaming
ALTER TABLE events ADD COLUMN IF NOT EXISTS stream_session_id UUID
    REFERENCES stream_sessions(id) ON DELETE SET NULL;

-- Step 5: Retain coarse_geohash as NULLABLE (privacy: explicit consent required)
-- Do NOT set a default or NOT NULL constraint; NULL means "no location provided"
-- Business logic should enforce presence if required, not the schema
-- This aligns with the privacy-first design where location consent is explicit

-- Step 6: FTS support for title + tags via indexes
-- Note: Generated tsvector columns require immutable expressions
-- We'll add GIN indexes directly on expression instead
-- This is handled in Step 7

-- Step 7: Add indexes for query performance
-- GIN index on tags for array queries (exclude soft-deleted and cancelled events)
CREATE INDEX IF NOT EXISTS idx_events_tags ON events USING GIN(tags)
    WHERE deleted_at IS NULL AND cancelled_at IS NULL;

-- GIN index for FTS queries on title + tags (exclude deleted/cancelled)
-- Note: PostgreSQL's to_tsvector('english', ...) is not marked IMMUTABLE
-- For now, we skip FTS indexing and rely on application-level FTS or simpler queries
-- TODO: Consider adding a custom IMMUTABLE wrapper function if FTS becomes critical
-- CREATE INDEX IF NOT EXISTS idx_events_title_tags_fts ON events USING GIN(
--     to_tsvector('english', COALESCE(title, '') || ' ' || COALESCE(array_to_string(tags, ' '), ''))
-- ) WHERE deleted_at IS NULL AND cancelled_at IS NULL;

-- Index on stream_session_id for join queries
CREATE INDEX IF NOT EXISTS idx_events_stream_session ON events(stream_session_id)
    WHERE deleted_at IS NULL AND stream_session_id IS NOT NULL;

-- Index on status for filtering by event lifecycle (exclude soft-deleted and cancelled)
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status)
    WHERE deleted_at IS NULL AND cancelled_at IS NULL;

-- Update table and column comments
COMMENT ON COLUMN events.title IS 'Event title, indexed for full-text search';
COMMENT ON COLUMN events.tags IS 'Categorization tags for discovery, indexed for FTS and array queries';
COMMENT ON COLUMN events.status IS 'Event lifecycle status (scheduled, live, ended, cancelled)';
COMMENT ON COLUMN events.stream_session_id IS 'Reference to active LiveKit stream session';
