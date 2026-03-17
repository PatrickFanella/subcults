-- Add join_count and leave_count columns to stream_sessions table
-- These columns track aggregate join/leave events for historical queries and analytics

ALTER TABLE stream_sessions
    ADD COLUMN join_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN leave_count INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN stream_sessions.join_count IS 'Total number of join events for this stream session';
COMMENT ON COLUMN stream_sessions.leave_count IS 'Total number of leave events for this stream session';
