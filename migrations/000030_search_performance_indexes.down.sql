-- Rollback: Remove search performance indexes added in migration 000030

DROP INDEX IF EXISTS idx_events_scene_upcoming;
DROP INDEX IF EXISTS idx_events_upcoming_geohash;
DROP INDEX IF EXISTS idx_scenes_public_created;
DROP INDEX IF EXISTS idx_scenes_search_visible;

DELETE FROM schema_version WHERE version = 30;
