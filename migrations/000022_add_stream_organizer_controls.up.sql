-- Add organizer control columns to stream_sessions table
ALTER TABLE stream_sessions
ADD COLUMN IF NOT EXISTS is_locked BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS featured_participant VARCHAR(255);

-- Add indexes for common queries
CREATE INDEX IF NOT EXISTS idx_stream_sessions_is_locked ON stream_sessions(is_locked) WHERE is_locked = TRUE;
CREATE INDEX IF NOT EXISTS idx_stream_sessions_featured_participant ON stream_sessions(featured_participant) WHERE featured_participant IS NOT NULL;
