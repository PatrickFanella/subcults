-- Rollback: Remove cancellation_reason field from events table

ALTER TABLE events DROP COLUMN IF EXISTS cancellation_reason;
