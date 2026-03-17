-- Rollback: Revert events table to pre-enhancement schema
-- Removes: tags, status, stream_session_id, FTS column
-- Restores: name column (from title)
--
-- WARNING: This migration contains DESTRUCTIVE operations:
-- 1. Tags data will be lost
-- 2. Status data will be lost
-- 3. Stream session associations will be lost
-- 4. FTS index will be removed

-- Step 1: Drop new indexes
DROP INDEX IF EXISTS idx_events_tags;
DROP INDEX IF EXISTS idx_events_title_tags_fts;
DROP INDEX IF EXISTS idx_events_stream_session;
DROP INDEX IF EXISTS idx_events_status;

-- Step 2: Drop FTS column
ALTER TABLE events DROP COLUMN IF EXISTS title_tags_fts;

-- Step 3: Drop stream_session_id column
ALTER TABLE events DROP COLUMN IF EXISTS stream_session_id;

-- Step 4: Drop status column
ALTER TABLE events DROP CONSTRAINT IF EXISTS chk_event_status;
ALTER TABLE events DROP COLUMN IF EXISTS status;

-- Step 5: Drop tags column
ALTER TABLE events DROP COLUMN IF EXISTS tags;

-- Step 6: coarse_geohash remains NULLABLE (no change needed)
-- The up migration no longer modifies this column

-- Step 7: Rename title back to name
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'events' AND column_name = 'title'
    ) THEN
        ALTER TABLE events RENAME COLUMN title TO name;
    END IF;
END $$;
