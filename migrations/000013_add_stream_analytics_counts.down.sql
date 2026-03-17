-- Remove join_count and leave_count columns from stream_sessions table

ALTER TABLE stream_sessions
    DROP COLUMN IF EXISTS join_count,
    DROP COLUMN IF EXISTS leave_count;
