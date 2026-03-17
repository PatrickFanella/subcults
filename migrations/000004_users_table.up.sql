-- Migration: Create users table for core identity and ATProto linking
-- Foundation for ownership and membership relations

-- Users table - core identity with ATProto integration
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    did TEXT UNIQUE, -- ATProto Decentralized Identifier (optional, linked after OAuth)
    handle TEXT NOT NULL UNIQUE, -- Unique user handle (UNIQUE constraint creates implicit index)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index on did for ATProto linking queries (partial index for non-null values)
CREATE INDEX IF NOT EXISTS idx_users_did ON users(did) WHERE did IS NOT NULL;

COMMENT ON TABLE users IS 'Core user identity with optional ATProto DID linking';
COMMENT ON COLUMN users.did IS 'ATProto Decentralized Identifier, linked after OAuth flow';
COMMENT ON COLUMN users.handle IS 'Unique user handle for identification';
