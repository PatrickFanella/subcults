-- Migration: Add trust_weight, since columns to memberships and reason, status, since columns to alliances
-- Required for trust graph computation and filtering

-- ============================================
-- MEMBERSHIPS TABLE UPDATES
-- ============================================

-- Add trust_weight column with default 0.5 and constraint 0-1
ALTER TABLE memberships ADD COLUMN IF NOT EXISTS trust_weight FLOAT DEFAULT 0.5;

-- Add CHECK constraint for trust_weight between 0 and 1
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_membership_trust_weight'
    ) THEN
        ALTER TABLE memberships ADD CONSTRAINT chk_membership_trust_weight
            CHECK (trust_weight >= 0.0 AND trust_weight <= 1.0);
    END IF;
END $$;

-- Add since column (timestamp when membership started)
-- Default to created_at for existing rows
ALTER TABLE memberships ADD COLUMN IF NOT EXISTS since TIMESTAMPTZ;

-- Update existing rows to use created_at as since value
UPDATE memberships SET since = created_at WHERE since IS NULL;

-- Make since NOT NULL after backfilling
ALTER TABLE memberships ALTER COLUMN since SET NOT NULL;
ALTER TABLE memberships ALTER COLUMN since SET DEFAULT NOW();

-- Index on trust_weight for filtering high-trust members
CREATE INDEX IF NOT EXISTS idx_memberships_trust_weight ON memberships(trust_weight)
    WHERE status = 'active';

-- ============================================
-- ALLIANCES TABLE UPDATES
-- ============================================

-- Add reason column for documenting alliance purpose
ALTER TABLE alliances ADD COLUMN IF NOT EXISTS reason TEXT;

-- Add status column for alliance lifecycle (pending, active, rejected, dissolved)
ALTER TABLE alliances ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'active';

-- Backfill any NULL statuses to 'active' before setting NOT NULL
UPDATE alliances SET status = 'active' WHERE status IS NULL;
ALTER TABLE alliances ALTER COLUMN status SET NOT NULL;
-- Add CHECK constraint for valid status values
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_alliance_status'
    ) THEN
        ALTER TABLE alliances ADD CONSTRAINT chk_alliance_status
            CHECK (status IN ('pending', 'active', 'rejected', 'dissolved'));
    END IF;
END $$;

-- Add since column (timestamp when alliance started)
-- Default to created_at for existing rows
ALTER TABLE alliances ADD COLUMN IF NOT EXISTS since TIMESTAMPTZ;

-- Update existing rows to use created_at as since value
UPDATE alliances SET since = created_at WHERE since IS NULL;

-- Make since NOT NULL after backfilling
ALTER TABLE alliances ALTER COLUMN since SET NOT NULL;
ALTER TABLE alliances ALTER COLUMN since SET DEFAULT NOW();

-- Index on weight for filtering by alliance strength
CREATE INDEX IF NOT EXISTS idx_alliances_weight ON alliances(weight);

-- Index on status for filtering active/pending alliances
CREATE INDEX IF NOT EXISTS idx_alliances_status ON alliances(status);

-- Composite index on status and weight for combined filtering
CREATE INDEX IF NOT EXISTS idx_alliances_status_weight ON alliances(status, weight)
    WHERE status = 'active';

-- ============================================
-- COMMENTS
-- ============================================

COMMENT ON COLUMN memberships.trust_weight IS 'Base trust weight (0.0-1.0) for trust score computation. Used with role multiplier.';
COMMENT ON COLUMN memberships.since IS 'Timestamp when the membership started';

COMMENT ON COLUMN alliances.reason IS 'Optional description of why this alliance was formed';
COMMENT ON COLUMN alliances.status IS 'Alliance lifecycle status (pending, active, rejected, dissolved)';
COMMENT ON COLUMN alliances.since IS 'Timestamp when the alliance was established';
