-- Stream quality metrics table for audio quality monitoring
-- Tracks bitrate, jitter, packet loss, and audio levels per participant

CREATE TABLE IF NOT EXISTS stream_quality_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stream_session_id UUID NOT NULL REFERENCES stream_sessions(id) ON DELETE CASCADE,
    participant_id VARCHAR(255) NOT NULL, -- LiveKit participant identity
    
    -- Audio quality metrics
    bitrate_kbps FLOAT, -- Audio bitrate in kilobits per second
    jitter_ms FLOAT, -- Jitter in milliseconds
    packet_loss_percent FLOAT, -- Packet loss percentage (0-100)
    audio_level FLOAT, -- Audio level (0.0-1.0, where 1.0 is loudest)
    
    -- Network quality indicators
    rtt_ms FLOAT, -- Round-trip time in milliseconds
    
    -- Metadata
    measured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_bitrate_positive CHECK (bitrate_kbps IS NULL OR bitrate_kbps >= 0),
    CONSTRAINT chk_jitter_positive CHECK (jitter_ms IS NULL OR jitter_ms >= 0),
    CONSTRAINT chk_packet_loss_range CHECK (packet_loss_percent IS NULL OR (packet_loss_percent >= 0 AND packet_loss_percent <= 100)),
    CONSTRAINT chk_audio_level_range CHECK (audio_level IS NULL OR (audio_level >= 0 AND audio_level <= 1)),
    CONSTRAINT chk_rtt_positive CHECK (rtt_ms IS NULL OR rtt_ms >= 0)
);

-- Indexes for efficient queries
CREATE INDEX idx_quality_metrics_session ON stream_quality_metrics(stream_session_id, measured_at DESC);
CREATE INDEX idx_quality_metrics_participant ON stream_quality_metrics(participant_id, measured_at DESC);
-- Efficient lookup of latest metrics for a participant in a session
CREATE INDEX idx_quality_metrics_session_participant_latest ON stream_quality_metrics(stream_session_id, participant_id, measured_at DESC);
-- High packet loss scanning within recent time windows
CREATE INDEX idx_quality_metrics_packet_loss ON stream_quality_metrics(stream_session_id, measured_at DESC, packet_loss_percent DESC) WHERE packet_loss_percent > 5.0;

COMMENT ON TABLE stream_quality_metrics IS 'Real-time audio quality metrics for stream participants';
COMMENT ON COLUMN stream_quality_metrics.bitrate_kbps IS 'Audio bitrate in kilobits per second';
COMMENT ON COLUMN stream_quality_metrics.jitter_ms IS 'Jitter (packet delay variation) in milliseconds';
COMMENT ON COLUMN stream_quality_metrics.packet_loss_percent IS 'Packet loss percentage (0-100)';
COMMENT ON COLUMN stream_quality_metrics.audio_level IS 'Audio level from 0.0 (silent) to 1.0 (loudest)';
COMMENT ON COLUMN stream_quality_metrics.rtt_ms IS 'Round-trip time in milliseconds';
