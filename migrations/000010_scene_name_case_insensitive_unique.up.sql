-- Migration: Add case-insensitive unique constraint on scene names per owner
-- Ensures that scene names are unique per owner regardless of case to prevent confusion
-- and abuse (e.g., "MyScene" vs "myscene" vs "MYSCENE" all considered duplicates)

-- Create a unique index on (owner_did, LOWER(name)) for non-deleted scenes
-- This enforces case-insensitive uniqueness at the database level
CREATE UNIQUE INDEX IF NOT EXISTS idx_scenes_owner_name_unique 
    ON scenes (owner_did, LOWER(name)) 
    WHERE deleted_at IS NULL;

COMMENT ON INDEX idx_scenes_owner_name_unique IS 'Ensures case-insensitive scene name uniqueness per owner for non-deleted scenes';
