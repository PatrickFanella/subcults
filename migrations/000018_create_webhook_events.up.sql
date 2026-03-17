-- Migration: Create webhook_events table for idempotency tracking
-- Adds: webhook_events table to prevent duplicate event processing

-- ============================================
-- Create webhook_events table
-- ============================================

CREATE TABLE IF NOT EXISTS webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(255) NOT NULL UNIQUE,
    event_type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for processed_at lookups
CREATE INDEX IF NOT EXISTS idx_webhook_events_processed_at ON webhook_events(processed_at);

-- Add comments
COMMENT ON TABLE webhook_events IS 'Tracks processed Stripe webhook events for idempotency';
COMMENT ON COLUMN webhook_events.event_id IS 'Stripe event ID (e.g., evt_...)';
COMMENT ON COLUMN webhook_events.event_type IS 'Stripe event type (e.g., checkout.session.completed)';
COMMENT ON COLUMN webhook_events.processed_at IS 'Timestamp when event was processed';
