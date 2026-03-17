-- Rollback: Revert posts table to original schema
-- Removes: attachments, labels, FTS column, scene_id/event_id constraint
-- Restores: scene_id NOT NULL, content column name, attachment_url column
--
-- WARNING: This migration contains DESTRUCTIVE operations:
-- 1. Posts with multiple attachments will lose all but the first attachment
-- 2. Posts associated only with events (scene_id IS NULL) will be PERMANENTLY DELETED
-- 3. Labels data will be lost
--
-- Consider backing up the posts table before running this rollback:
--   CREATE TABLE posts_backup AS SELECT * FROM posts;

-- Step 1: Drop new indexes
DROP INDEX IF EXISTS idx_posts_labels;
DROP INDEX IF EXISTS idx_posts_text_fts;
DROP INDEX IF EXISTS idx_posts_author;

-- Step 2: Drop the association constraint
ALTER TABLE posts DROP CONSTRAINT IF EXISTS chk_post_association;

-- Step 3: Drop FTS column
ALTER TABLE posts DROP COLUMN IF EXISTS text_fts;

-- Step 4: Restore attachment_url column from attachments JSONB
ALTER TABLE posts ADD COLUMN IF NOT EXISTS attachment_url VARCHAR(500);

-- Migrate first attachment URL back to the single column
UPDATE posts 
SET attachment_url = attachments->0->>'url'
WHERE attachments IS NOT NULL 
  AND jsonb_array_length(attachments) > 0 
  AND attachments->0->>'url' IS NOT NULL;

-- Drop attachments column
ALTER TABLE posts DROP COLUMN IF EXISTS attachments;

-- Step 5: Drop labels column
ALTER TABLE posts DROP COLUMN IF EXISTS labels;

-- Step 6: Rename text back to content
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'posts' AND column_name = 'text'
    ) THEN
        ALTER TABLE posts RENAME COLUMN text TO content;
    END IF;
END $$;

-- Step 7: Delete posts with NULL scene_id (they can't exist in original schema)
-- This is destructive but necessary to restore NOT NULL constraint
DELETE FROM posts WHERE scene_id IS NULL;

-- Step 8: Make scene_id NOT NULL again
ALTER TABLE posts ALTER COLUMN scene_id SET NOT NULL;

-- Step 9: Recreate original indexes
DROP INDEX IF EXISTS idx_posts_scene;
CREATE INDEX idx_posts_scene ON posts(scene_id) WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_posts_event;
CREATE INDEX idx_posts_event ON posts(event_id) WHERE deleted_at IS NULL;

-- Restore idx_posts_created index from original schema
DROP INDEX IF EXISTS idx_posts_created;
CREATE INDEX idx_posts_created ON posts(created_at DESC) WHERE deleted_at IS NULL;

-- Note: The original schema didn't have comments on posts table, so no need to restore any
