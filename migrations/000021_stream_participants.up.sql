-- Add stream_participants table for tracking individual participants in real-time
-- Also add denormalized participant_count to stream_sessions for efficient queries

-- Create stream_participants table
CREATE TABLE IF NOT EXISTS stream_participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stream_session_id UUID NOT NULL REFERENCES stream_sessions(id) ON DELETE CASCADE,
    participant_id VARCHAR(255) NOT NULL, -- LiveKit participant identity (e.g., "user-abc123")
    user_did VARCHAR(255) NOT NULL, -- Decentralized Identifier
    
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ, -- NULL while participant is active
    
    -- Track reconnections: same user_did rejoining after leaving
    -- This allows us to distinguish between initial join and reconnection
    reconnection_count INT NOT NULL DEFAULT 0,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ensure we don't have duplicate active participants per session while they are active (left_at IS NULL)
-- Uses a partial unique index instead of a constraint because NULL values in UNIQUE constraints are treated as distinct
CREATE UNIQUE INDEX unique_active_participant
    ON stream_participants(stream_session_id, participant_id)
    WHERE left_at IS NULL;

-- Indexes for efficient queries
CREATE INDEX idx_stream_participants_session ON stream_participants(stream_session_id) WHERE left_at IS NULL;
CREATE INDEX idx_stream_participants_user ON stream_participants(user_did);
CREATE INDEX idx_stream_participants_joined ON stream_participants(joined_at);

-- Add denormalized participant_count column to stream_sessions
-- This avoids expensive COUNT(*) queries on every request
ALTER TABLE stream_sessions
ADD COLUMN IF NOT EXISTS active_participant_count INT NOT NULL DEFAULT 0;

-- Create index for active streams with participant counts
CREATE INDEX idx_stream_active_participant_count ON stream_sessions(active_participant_count) WHERE ended_at IS NULL;

COMMENT ON TABLE stream_participants IS 'Individual participant tracking for stream sessions with real-time state';
COMMENT ON COLUMN stream_participants.left_at IS 'NULL while participant is active in the stream';
COMMENT ON COLUMN stream_participants.reconnection_count IS 'Number of times this participant has rejoined after leaving';
COMMENT ON COLUMN stream_sessions.active_participant_count IS 'Denormalized count of active participants (left_at IS NULL) for efficient queries';
