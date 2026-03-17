-- Remove unique partial indexes for concurrent stream prevention
DROP INDEX IF EXISTS idx_stream_scene_active_unique;
DROP INDEX IF EXISTS idx_stream_event_active_unique;
