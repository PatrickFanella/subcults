-- Migration: Add Stripe Connected Account onboarding status tracking
-- Tracks the state of Stripe's account setup process for scene owners
-- Related to issue #444 (Stripe Connect onboarding)

-- Add onboarding status column to scenes table
-- Status values: pending (not started), active (fully onboarded), restricted (limited access)
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS connected_account_status VARCHAR(50) NOT NULL DEFAULT 'pending';

-- Add timestamp for when account was fully onboarded
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS account_onboarded_at TIMESTAMPTZ;

-- Index on status for querying onboarded scenes
CREATE INDEX IF NOT EXISTS idx_scenes_connected_account_status ON scenes(connected_account_status) 
WHERE deleted_at IS NULL;

-- Index on connected_account_status and onboarded_at for analytics queries
CREATE INDEX IF NOT EXISTS idx_scenes_onboarded_at ON scenes(account_onboarded_at) 
WHERE deleted_at IS NULL AND connected_account_status = 'active';

-- Add constraint to ensure status is one of the valid values
ALTER TABLE scenes ADD CONSTRAINT IF NOT EXISTS chk_connected_account_status 
CHECK (connected_account_status IN ('pending', 'active', 'restricted'));

-- Add constraint to ensure account_onboarded_at is only set when status is active
ALTER TABLE scenes ADD CONSTRAINT IF NOT EXISTS chk_account_onboarded_consistency 
CHECK ((connected_account_status = 'active' AND account_onboarded_at IS NOT NULL) 
    OR (connected_account_status != 'active' AND account_onboarded_at IS NULL));

COMMENT ON COLUMN scenes.connected_account_status IS 'Stripe Connect onboarding status: pending (not started), active (fully onboarded), restricted (limited access)';
COMMENT ON COLUMN scenes.account_onboarded_at IS 'Timestamp when Stripe account was fully onboarded (non-null only when status is "active")';
