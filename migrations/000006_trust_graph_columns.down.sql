-- Rollback: Remove trust_weight, since columns from memberships and reason, status, since from alliances

-- ============================================
-- ALLIANCES TABLE ROLLBACK
-- ============================================

-- Drop indexes first
DROP INDEX IF EXISTS idx_alliances_status_weight;
DROP INDEX IF EXISTS idx_alliances_status;
DROP INDEX IF EXISTS idx_alliances_weight;

-- Drop constraint
ALTER TABLE alliances DROP CONSTRAINT IF EXISTS chk_alliance_status;

-- Drop columns
ALTER TABLE alliances DROP COLUMN IF EXISTS since;
ALTER TABLE alliances DROP COLUMN IF EXISTS status;
ALTER TABLE alliances DROP COLUMN IF EXISTS reason;

-- ============================================
-- MEMBERSHIPS TABLE ROLLBACK
-- ============================================

-- Drop indexes first
DROP INDEX IF EXISTS idx_memberships_trust_weight;

-- Drop constraint
ALTER TABLE memberships DROP CONSTRAINT IF EXISTS chk_membership_trust_weight;

-- Drop columns
ALTER TABLE memberships DROP COLUMN IF EXISTS since;
ALTER TABLE memberships DROP COLUMN IF EXISTS trust_weight;
