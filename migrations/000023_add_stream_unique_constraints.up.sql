-- Add unique partial indexes to prevent concurrent streams
-- These constraints ensure only one active stream per scene/event at the database level

-- Unique constraint: only one active stream per scene
CREATE UNIQUE INDEX idx_stream_scene_active_unique 
ON stream_sessions(scene_id) 
WHERE ended_at IS NULL AND scene_id IS NOT NULL;

-- Unique constraint: only one active stream per event  
CREATE UNIQUE INDEX idx_stream_event_active_unique 
ON stream_sessions(event_id) 
WHERE ended_at IS NULL AND event_id IS NOT NULL;
