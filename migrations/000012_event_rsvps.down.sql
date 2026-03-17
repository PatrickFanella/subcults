-- Migration rollback: Remove event_rsvps table

-- Drop index
DROP INDEX IF EXISTS idx_event_rsvps_user_id;

-- Drop the table
DROP TABLE IF EXISTS event_rsvps;
