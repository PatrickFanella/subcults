-- Migration: Create ingestion_idempotency table for Jetstream indexer
-- Purpose: Track processed AT Protocol records to prevent duplicate ingestion
-- Related: subculture-collective/subcults#370 - Transaction consistency

-- ============================================
-- Create ingestion_idempotency table
-- ============================================

CREATE TABLE IF NOT EXISTS ingestion_idempotency (
    idempotency_key VARCHAR(64) PRIMARY KEY,
    did VARCHAR(255) NOT NULL,
    collection VARCHAR(255) NOT NULL,
    rkey VARCHAR(255) NOT NULL,
    rev VARCHAR(255) NOT NULL,
    record_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for cleanup queries (remove entries older than retention period)
CREATE INDEX IF NOT EXISTS idx_ingestion_idempotency_created_at 
    ON ingestion_idempotency(created_at);

-- Index for lookups by DID and collection (for debugging/auditing)
CREATE INDEX IF NOT EXISTS idx_ingestion_idempotency_did_collection 
    ON ingestion_idempotency(did, collection);

-- Add comments for documentation
COMMENT ON TABLE ingestion_idempotency IS 'Tracks processed AT Protocol records to prevent duplicate ingestion during Jetstream indexing';
COMMENT ON COLUMN ingestion_idempotency.idempotency_key IS 'SHA256 hash of (did + collection + rkey + rev) for deterministic deduplication';
COMMENT ON COLUMN ingestion_idempotency.did IS 'AT Protocol DID of the record creator';
COMMENT ON COLUMN ingestion_idempotency.collection IS 'AT Protocol collection name';
COMMENT ON COLUMN ingestion_idempotency.rkey IS 'AT Protocol record key (rkey)';
COMMENT ON COLUMN ingestion_idempotency.rev IS 'AT Protocol revision identifier';
COMMENT ON COLUMN ingestion_idempotency.record_id IS 'Internal UUID of the created/updated record';
COMMENT ON COLUMN ingestion_idempotency.created_at IS 'Timestamp when the record was first ingested';
