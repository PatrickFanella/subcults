-- Rollback: Remove FTS GIN indexes and IMMUTABLE wrapper function

-- Drop FTS indexes
DROP INDEX IF EXISTS idx_posts_text_fts;
DROP INDEX IF EXISTS idx_events_title_tags_fts;
DROP INDEX IF EXISTS idx_scenes_name_desc_tags_fts;

-- Drop IMMUTABLE wrapper function
DROP FUNCTION IF EXISTS to_tsvector_immutable(text);
