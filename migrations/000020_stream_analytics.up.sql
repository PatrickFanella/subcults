-- Stream analytics and participant events tables
-- Tracks detailed engagement metrics for stream sessions

-- Participant events table - tracks individual join/leave events for analytics
CREATE TABLE IF NOT EXISTS stream_participant_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stream_session_id UUID NOT NULL REFERENCES stream_sessions(id) ON DELETE CASCADE,
    participant_did VARCHAR(255) NOT NULL,
    event_type VARCHAR(20) NOT NULL CHECK (event_type IN ('join', 'leave')),
    
    -- Geographic distribution (privacy-safe, no precise location)
    -- Uses coarse geohash derived from user's location at time of join
    geohash_prefix VARCHAR(4), -- First 4 chars for regional distribution (~20km precision)
    
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    
    -- Index for efficient analytics queries
);

CREATE INDEX idx_participant_events_stream ON stream_participant_events(stream_session_id, occurred_at);
CREATE INDEX idx_participant_events_type ON stream_participant_events(stream_session_id, event_type, occurred_at);

COMMENT ON TABLE stream_participant_events IS 'Individual participant join/leave events for detailed analytics';
COMMENT ON COLUMN stream_participant_events.geohash_prefix IS 'Coarse geohash (4 chars, ~20km) for privacy-safe geographic distribution';

-- Stream analytics table - computed metrics for ended streams
CREATE TABLE IF NOT EXISTS stream_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stream_session_id UUID NOT NULL UNIQUE REFERENCES stream_sessions(id) ON DELETE CASCADE,
    
    -- Core engagement metrics
    peak_concurrent_listeners INTEGER NOT NULL DEFAULT 0,
    total_unique_participants INTEGER NOT NULL DEFAULT 0,
    total_join_attempts INTEGER NOT NULL DEFAULT 0,
    
    -- Timing metrics
    stream_duration_seconds INTEGER NOT NULL DEFAULT 0,
    engagement_lag_seconds INTEGER, -- Time from stream start to first join (NULL if no joins)
    
    -- Retention metrics
    avg_listen_duration_seconds FLOAT, -- Average time participants stayed
    median_listen_duration_seconds FLOAT, -- Median time participants stayed
    
    -- Geographic distribution (privacy-safe aggregate)
    -- JSONB format: {"abc": 5, "def": 3} where keys are 4-char geohash prefixes and values are participant counts
    geographic_distribution JSONB DEFAULT '{}'::jsonb,
    
    -- Metadata
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_peak_listeners_positive CHECK (peak_concurrent_listeners >= 0),
    CONSTRAINT chk_unique_participants_positive CHECK (total_unique_participants >= 0),
    CONSTRAINT chk_join_attempts_positive CHECK (total_join_attempts >= 0),
    CONSTRAINT chk_duration_positive CHECK (stream_duration_seconds >= 0)
);

CREATE INDEX idx_stream_analytics_session ON stream_analytics(stream_session_id);
CREATE INDEX idx_stream_analytics_computed ON stream_analytics(computed_at);

COMMENT ON TABLE stream_analytics IS 'Computed analytics for ended stream sessions';
COMMENT ON COLUMN stream_analytics.engagement_lag_seconds IS 'Time from stream start to first join event (NULL if no joins occurred)';
COMMENT ON COLUMN stream_analytics.geographic_distribution IS 'Privacy-safe aggregate: map of 4-char geohash prefixes to participant counts';
