-- Migration: Deprecated - allow_precise is now in 000000_initial_schema
-- This migration is kept for version continuity but performs no operations
-- The allow_precise column is created in the base schema with proper constraints

-- No-op migration (allow_precise already exists in base schema)
SELECT 1; -- Placeholder to make migration valid
