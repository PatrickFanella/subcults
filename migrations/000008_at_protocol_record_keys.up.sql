-- Migration: Add AT Protocol record key tracking for idempotent ingestion
-- Adds: did, rkey columns with unique constraints for scenes, events, posts, memberships, alliances, stream_sessions
-- Purpose: Map (did + rkey) to internal UUID for upsert operations during Jetstream ingestion

-- ============================================
-- SCENES TABLE
-- ============================================

-- Add AT Protocol record key columns
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS record_did VARCHAR(255);
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS record_rkey VARCHAR(255);

-- Create unique constraint on (record_did, record_rkey) for idempotent upserts
-- Only apply when both fields are non-null (allows existing records without keys)
CREATE UNIQUE INDEX IF NOT EXISTS idx_scenes_record_key 
    ON scenes(record_did, record_rkey) 
    WHERE record_did IS NOT NULL AND record_rkey IS NOT NULL;

COMMENT ON COLUMN scenes.record_did IS 'AT Protocol DID of the record creator';
COMMENT ON COLUMN scenes.record_rkey IS 'AT Protocol record key (rkey) for unique identification';

-- ============================================
-- EVENTS TABLE
-- ============================================

ALTER TABLE events ADD COLUMN IF NOT EXISTS record_did VARCHAR(255);
ALTER TABLE events ADD COLUMN IF NOT EXISTS record_rkey VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_events_record_key 
    ON events(record_did, record_rkey) 
    WHERE record_did IS NOT NULL AND record_rkey IS NOT NULL;

COMMENT ON COLUMN events.record_did IS 'AT Protocol DID of the record creator';
COMMENT ON COLUMN events.record_rkey IS 'AT Protocol record key (rkey) for unique identification';

-- ============================================
-- POSTS TABLE
-- ============================================

ALTER TABLE posts ADD COLUMN IF NOT EXISTS record_did VARCHAR(255);
ALTER TABLE posts ADD COLUMN IF NOT EXISTS record_rkey VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_posts_record_key 
    ON posts(record_did, record_rkey) 
    WHERE record_did IS NOT NULL AND record_rkey IS NOT NULL;

COMMENT ON COLUMN posts.record_did IS 'AT Protocol DID of the record creator';
COMMENT ON COLUMN posts.record_rkey IS 'AT Protocol record key (rkey) for unique identification';

-- ============================================
-- MEMBERSHIPS TABLE
-- ============================================

ALTER TABLE memberships ADD COLUMN IF NOT EXISTS record_did VARCHAR(255);
ALTER TABLE memberships ADD COLUMN IF NOT EXISTS record_rkey VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_memberships_record_key 
    ON memberships(record_did, record_rkey) 
    WHERE record_did IS NOT NULL AND record_rkey IS NOT NULL;

COMMENT ON COLUMN memberships.record_did IS 'AT Protocol DID of the record creator';
COMMENT ON COLUMN memberships.record_rkey IS 'AT Protocol record key (rkey) for unique identification';

-- ============================================
-- ALLIANCES TABLE
-- ============================================

ALTER TABLE alliances ADD COLUMN IF NOT EXISTS record_did VARCHAR(255);
ALTER TABLE alliances ADD COLUMN IF NOT EXISTS record_rkey VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_alliances_record_key 
    ON alliances(record_did, record_rkey) 
    WHERE record_did IS NOT NULL AND record_rkey IS NOT NULL;

COMMENT ON COLUMN alliances.record_did IS 'AT Protocol DID of the record creator';
COMMENT ON COLUMN alliances.record_rkey IS 'AT Protocol record key (rkey) for unique identification';

-- ============================================
-- STREAM_SESSIONS TABLE
-- ============================================

ALTER TABLE stream_sessions ADD COLUMN IF NOT EXISTS record_did VARCHAR(255);
ALTER TABLE stream_sessions ADD COLUMN IF NOT EXISTS record_rkey VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_stream_sessions_record_key 
    ON stream_sessions(record_did, record_rkey) 
    WHERE record_did IS NOT NULL AND record_rkey IS NOT NULL;

COMMENT ON COLUMN stream_sessions.record_did IS 'AT Protocol DID of the record creator';
COMMENT ON COLUMN stream_sessions.record_rkey IS 'AT Protocol record key (rkey) for unique identification';
