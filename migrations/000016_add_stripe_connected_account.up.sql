-- Migration: Add Stripe Connect account ID to scenes table
-- Adds: connected_account_id column for tracking Stripe Express account linkage

-- ============================================
-- Add connected_account_id column
-- ============================================

ALTER TABLE scenes ADD COLUMN IF NOT EXISTS connected_account_id VARCHAR(255);

-- Create index for Stripe account lookups (exclude soft-deleted scenes)
CREATE INDEX IF NOT EXISTS idx_scenes_connected_account ON scenes(connected_account_id)
    WHERE deleted_at IS NULL AND connected_account_id IS NOT NULL;

COMMENT ON COLUMN scenes.connected_account_id IS 'Stripe Connect Express account ID for direct payments';
