-- Rollback: Remove Stripe Connected Account onboarding status tracking

-- Drop constraints
ALTER TABLE scenes DROP CONSTRAINT IF EXISTS chk_account_onboarded_consistency;
ALTER TABLE scenes DROP CONSTRAINT IF EXISTS chk_connected_account_status;

-- Drop indexes
DROP INDEX IF EXISTS idx_scenes_onboarded_at;
DROP INDEX IF EXISTS idx_scenes_connected_account_status;

-- Drop columns
ALTER TABLE scenes DROP COLUMN IF EXISTS account_onboarded_at;
ALTER TABLE scenes DROP COLUMN IF EXISTS connected_account_status;
