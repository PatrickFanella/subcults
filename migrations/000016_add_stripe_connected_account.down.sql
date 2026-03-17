-- Rollback: Remove Stripe Connect account ID from scenes table

DROP INDEX IF EXISTS idx_scenes_connected_account;
ALTER TABLE scenes DROP COLUMN IF EXISTS connected_account_id;
