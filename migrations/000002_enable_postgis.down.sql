-- Rollback: PostGIS extension
-- WARNING: Dropping PostGIS will remove all geography/geometry columns and data!
-- This is intentionally left as a no-op to prevent data loss.
-- The extension will be removed by 000000_initial_schema.down.sql when rolling back fully.

-- No-op: Do not drop PostGIS here to preserve data in dependent tables
SELECT 1;
