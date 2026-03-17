-- Rollback: Remove AT Protocol record key tracking columns

DROP INDEX IF EXISTS idx_stream_sessions_record_key;
ALTER TABLE stream_sessions DROP COLUMN IF EXISTS record_rkey;
ALTER TABLE stream_sessions DROP COLUMN IF EXISTS record_did;

DROP INDEX IF EXISTS idx_alliances_record_key;
ALTER TABLE alliances DROP COLUMN IF EXISTS record_rkey;
ALTER TABLE alliances DROP COLUMN IF EXISTS record_did;

DROP INDEX IF EXISTS idx_memberships_record_key;
ALTER TABLE memberships DROP COLUMN IF EXISTS record_rkey;
ALTER TABLE memberships DROP COLUMN IF EXISTS record_did;

DROP INDEX IF EXISTS idx_posts_record_key;
ALTER TABLE posts DROP COLUMN IF EXISTS record_rkey;
ALTER TABLE posts DROP COLUMN IF EXISTS record_did;

DROP INDEX IF EXISTS idx_events_record_key;
ALTER TABLE events DROP COLUMN IF EXISTS record_rkey;
ALTER TABLE events DROP COLUMN IF EXISTS record_did;

DROP INDEX IF EXISTS idx_scenes_record_key;
ALTER TABLE scenes DROP COLUMN IF EXISTS record_rkey;
ALTER TABLE scenes DROP COLUMN IF EXISTS record_did;
