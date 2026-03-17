-- Rollback: Remove feed pagination indexes
-- Restores the database to its previous state before feed optimization

DROP INDEX IF EXISTS idx_posts_event_feed;
DROP INDEX IF EXISTS idx_posts_scene_feed;
