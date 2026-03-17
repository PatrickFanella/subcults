-- Migration: Add indexes for feed pagination performance
-- Optimizes queries for GET /scenes/{id}/feed and GET /events/{id}/feed endpoints
-- These indexes support ORDER BY created_at DESC with scene_id/event_id filter

-- Composite index on scene_id + created_at for scene feeds
-- Supports: SELECT * FROM posts WHERE scene_id = ? AND deleted_at IS NULL ORDER BY created_at DESC, id ASC
-- The existing idx_posts_scene only indexes scene_id; this adds created_at for sorting performance
DROP INDEX IF EXISTS idx_posts_scene_feed;
CREATE INDEX idx_posts_scene_feed 
    ON posts(scene_id, created_at DESC, id ASC) 
    WHERE deleted_at IS NULL AND scene_id IS NOT NULL;

-- Composite index on event_id + created_at for event feeds  
-- Supports: SELECT * FROM posts WHERE event_id = ? AND deleted_at IS NULL ORDER BY created_at DESC, id ASC
DROP INDEX IF EXISTS idx_posts_event_feed;
CREATE INDEX idx_posts_event_feed 
    ON posts(event_id, created_at DESC, id ASC) 
    WHERE deleted_at IS NULL AND event_id IS NOT NULL;

COMMENT ON INDEX idx_posts_scene_feed IS 'Optimizes scene feed pagination with cursor-based ordering';
COMMENT ON INDEX idx_posts_event_feed IS 'Optimizes event feed pagination with cursor-based ordering';
