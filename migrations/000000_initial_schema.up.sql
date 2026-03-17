-- Initial schema for Subcults
-- Creates core tables: scenes, events, posts, memberships, alliances, stream_sessions, indexer_state

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Scenes table - core entity for underground music scenes
CREATE TABLE IF NOT EXISTS scenes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    owner_did VARCHAR(255) NOT NULL, -- Decentralized Identifier
    
    -- Location with privacy controls
    allow_precise BOOLEAN NOT NULL DEFAULT FALSE,
    precise_point GEOGRAPHY(Point, 4326), -- WGS84 coordinates, only if allow_precise=true
    coarse_geohash VARCHAR(20), -- Public coarse location
    
    -- Visual identity
    primary_color VARCHAR(7),
    secondary_color VARCHAR(7),
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ, -- Soft delete
    
    CONSTRAINT chk_precise_consent CHECK (
        allow_precise = TRUE OR precise_point IS NULL
    )
);

CREATE INDEX idx_scenes_owner ON scenes(owner_did) WHERE deleted_at IS NULL;
CREATE INDEX idx_scenes_geohash ON scenes(coarse_geohash) WHERE deleted_at IS NULL;
CREATE INDEX idx_scenes_location ON scenes USING GIST(precise_point) WHERE deleted_at IS NULL AND allow_precise = TRUE;

COMMENT ON TABLE scenes IS 'Underground music scenes with privacy-controlled location data';
COMMENT ON COLUMN scenes.allow_precise IS 'When false, precise_point must be NULL. Consent required for precise location storage.';

-- Events table - temporal happenings within scenes
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scene_id UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Location (can override scene location)
    allow_precise BOOLEAN NOT NULL DEFAULT FALSE,
    precise_point GEOGRAPHY(Point, 4326),
    coarse_geohash VARCHAR(20),
    
    -- Timing
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    
    CONSTRAINT chk_event_time CHECK (ends_at IS NULL OR starts_at < ends_at),
    CONSTRAINT chk_event_precise_consent CHECK (
        allow_precise = TRUE OR precise_point IS NULL
    )
);

CREATE INDEX idx_events_scene ON events(scene_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_events_time ON events(starts_at) WHERE deleted_at IS NULL AND cancelled_at IS NULL;
CREATE INDEX idx_events_geohash ON events(coarse_geohash) WHERE deleted_at IS NULL;
CREATE INDEX idx_events_location ON events USING GIST(precise_point) WHERE deleted_at IS NULL AND cancelled_at IS NULL AND allow_precise = TRUE;

COMMENT ON TABLE events IS 'Events within scenes with optional precise location data';
COMMENT ON COLUMN events.allow_precise IS 'When false, precise_point must be NULL. Consent required for precise location storage.';

-- Posts table - content within scenes/events
CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scene_id UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    event_id UUID REFERENCES events(id) ON DELETE SET NULL,
    author_did VARCHAR(255) NOT NULL,
    
    content TEXT NOT NULL,
    attachment_url VARCHAR(500), -- R2/external media URL
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_posts_scene ON posts(scene_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_event ON posts(event_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_created ON posts(created_at DESC) WHERE deleted_at IS NULL;

-- Memberships table - scene participation
CREATE TABLE IF NOT EXISTS memberships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scene_id UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    user_did VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member', -- member, curator, admin
    
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, active, rejected
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(scene_id, user_did)
);

CREATE INDEX idx_memberships_scene ON memberships(scene_id, status);
CREATE INDEX idx_memberships_user ON memberships(user_did, status);

-- Alliances table - trust relationships between scenes
CREATE TABLE IF NOT EXISTS alliances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_scene_id UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    to_scene_id UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    
    weight FLOAT NOT NULL DEFAULT 1.0, -- Trust weight 0.0-1.0
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_alliance_weight CHECK (weight >= 0.0 AND weight <= 1.0),
    CONSTRAINT chk_no_self_alliance CHECK (from_scene_id != to_scene_id),
    UNIQUE(from_scene_id, to_scene_id)
);

CREATE INDEX idx_alliances_from ON alliances(from_scene_id);
CREATE INDEX idx_alliances_to ON alliances(to_scene_id);

-- Stream sessions table - LiveKit audio rooms
CREATE TABLE IF NOT EXISTS stream_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scene_id UUID REFERENCES scenes(id) ON DELETE SET NULL,
    event_id UUID REFERENCES events(id) ON DELETE SET NULL,
    
    room_name VARCHAR(255) NOT NULL UNIQUE,
    host_did VARCHAR(255) NOT NULL,
    
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    
    participant_count INT DEFAULT 0
);

CREATE INDEX idx_stream_scene ON stream_sessions(scene_id) WHERE ended_at IS NULL;
CREATE INDEX idx_stream_event ON stream_sessions(event_id) WHERE ended_at IS NULL;
CREATE INDEX idx_stream_active ON stream_sessions(started_at) WHERE ended_at IS NULL;

-- Indexer state table - cursor tracking for Jetstream
CREATE TABLE IF NOT EXISTS indexer_state (
    id SERIAL PRIMARY KEY,
    cursor BIGINT NOT NULL DEFAULT 0,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert initial cursor
INSERT INTO indexer_state (cursor) VALUES (0);
