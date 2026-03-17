-- Telemetry events table for frontend analytics.
-- Stores batched telemetry events from the frontend telemetry service.
-- Retention policy: 30 days (archived or deleted by retention job).
CREATE TABLE IF NOT EXISTS telemetry_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID NOT NULL,
    user_did    TEXT,
    event_name  TEXT NOT NULL,
    event_payload JSONB,
    timestamp   BIGINT NOT NULL,          -- milliseconds since epoch (matches frontend)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telemetry_events_session_id
    ON telemetry_events (session_id);

CREATE INDEX IF NOT EXISTS idx_telemetry_events_user_did
    ON telemetry_events (user_did) WHERE user_did IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_telemetry_events_timestamp
    ON telemetry_events (timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_telemetry_events_event_name
    ON telemetry_events (event_name);

-- Client error logs table for frontend error collection.
-- Stores client-side errors with PII-redacted messages and deduplication.
-- Retention policy: 90 days.
CREATE TABLE IF NOT EXISTS client_error_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL,
    user_did        TEXT,
    error_type      TEXT NOT NULL,
    error_message   TEXT NOT NULL,
    error_stack     TEXT,
    component_stack TEXT,
    url             TEXT,
    user_agent      TEXT,
    error_hash      TEXT NOT NULL,            -- SHA256(session_id || message) for dedup
    occurred_at     BIGINT NOT NULL,          -- milliseconds since epoch
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Deduplicate errors within the same session
CREATE UNIQUE INDEX IF NOT EXISTS idx_client_error_logs_dedup
    ON client_error_logs (error_hash, session_id);

CREATE INDEX IF NOT EXISTS idx_client_error_logs_session_id
    ON client_error_logs (session_id);

CREATE INDEX IF NOT EXISTS idx_client_error_logs_error_type
    ON client_error_logs (error_type);

CREATE INDEX IF NOT EXISTS idx_client_error_logs_occurred_at
    ON client_error_logs (occurred_at DESC);

-- Error replay events table for session replay snapshots.
-- Stores DOM mutation/click/scroll events captured around the time of an error.
-- Cascading delete — when the parent error log is purged, replays go too.
-- Retention policy: 90 days (tied to error log retention).
CREATE TABLE IF NOT EXISTS error_replay_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    error_log_id    UUID NOT NULL REFERENCES client_error_logs(id) ON DELETE CASCADE,
    event_type      TEXT NOT NULL,            -- click, input, navigation, mutation, scroll, etc.
    event_data      JSONB,
    event_timestamp BIGINT NOT NULL,          -- milliseconds since epoch
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_error_replay_events_error_log_id
    ON error_replay_events (error_log_id);

CREATE INDEX IF NOT EXISTS idx_error_replay_events_event_type
    ON error_replay_events (event_type);

-- Update schema version
INSERT INTO schema_version (version, description)
VALUES (29, 'telemetry events and client error logs');
