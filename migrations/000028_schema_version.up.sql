-- Schema version tracking table for application-level compatibility checks.
-- This is separate from golang-migrate's schema_migrations table.
-- Services query this at startup to enforce minimum schema compatibility.
CREATE TABLE IF NOT EXISTS schema_version (
    id          SERIAL PRIMARY KEY,
    version     INTEGER NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert the current schema version (28 = this migration)
INSERT INTO schema_version (version, description)
VALUES (28, 'schema_version tracking table');

-- Backfill checkpoint table for tracking backfill progress
CREATE TABLE IF NOT EXISTS backfill_checkpoints (
    id          SERIAL PRIMARY KEY,
    source      TEXT NOT NULL,           -- 'jetstream' or 'car'
    cursor_ts   BIGINT NOT NULL DEFAULT 0,  -- last processed timestamp (microseconds)
    car_offset  BIGINT NOT NULL DEFAULT 0,  -- byte offset for CAR file resume
    status      TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
    records_processed BIGINT NOT NULL DEFAULT 0,
    records_skipped   BIGINT NOT NULL DEFAULT 0,
    errors_count      BIGINT NOT NULL DEFAULT 0,
    started_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    metadata    JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_backfill_checkpoints_source_status
    ON backfill_checkpoints (source, status);
