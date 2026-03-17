-- Remove organizer control columns from stream_sessions table
DROP INDEX IF EXISTS idx_stream_sessions_featured_participant;
DROP INDEX IF EXISTS idx_stream_sessions_is_locked;

ALTER TABLE stream_sessions
DROP COLUMN IF EXISTS featured_participant,
DROP COLUMN IF EXISTS is_locked;
