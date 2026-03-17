-- Migration: Add cancellation_reason field to events table
-- This field stores the optional reason for event cancellation
-- Note: cancelled_at field already exists from 000000_initial_schema.up.sql

ALTER TABLE events ADD COLUMN IF NOT EXISTS cancellation_reason TEXT;

COMMENT ON COLUMN events.cancellation_reason IS 'Optional reason provided when an event is cancelled';
