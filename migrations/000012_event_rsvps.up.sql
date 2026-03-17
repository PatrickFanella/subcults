-- Migration: Add event_rsvps table for attendance tracking
-- Adds: event_rsvps table with PK on (event_id, user_id) for RSVP management

-- Step 1: Create event_rsvps table
CREATE TABLE IF NOT EXISTS event_rsvps (
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, user_id)
);

-- Step 2: Add CHECK constraint for valid status values
ALTER TABLE event_rsvps ADD CONSTRAINT chk_rsvp_status
    CHECK (status IN ('going', 'maybe'));

-- Step 3: Add indexes for query performance
-- Note: Index on event_id alone is not needed because PostgreSQL can use the
-- left portion of the composite PRIMARY KEY (event_id, user_id) for queries
-- filtering on event_id. We only need an index on user_id for reverse lookups.

-- Index on user_id for user's RSVP history queries
CREATE INDEX IF NOT EXISTS idx_event_rsvps_user_id ON event_rsvps(user_id);

-- Step 4: Add table and column comments
COMMENT ON TABLE event_rsvps IS 'Tracks user RSVP status for events (going/maybe)';
COMMENT ON COLUMN event_rsvps.event_id IS 'Reference to the event';
COMMENT ON COLUMN event_rsvps.user_id IS 'Reference to the user (not FK to allow guest RSVPs)';
COMMENT ON COLUMN event_rsvps.status IS 'RSVP status (going, maybe)';
COMMENT ON COLUMN event_rsvps.created_at IS 'Timestamp when RSVP was first created';
COMMENT ON COLUMN event_rsvps.updated_at IS 'Timestamp when RSVP was last updated';
