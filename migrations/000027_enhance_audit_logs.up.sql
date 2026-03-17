-- Add outcome and hash chain support to audit_logs table
-- This migration enhances audit logging for comprehensive compliance requirements

-- Add outcome column to track success/failure
ALTER TABLE audit_logs ADD COLUMN outcome VARCHAR(20) NOT NULL DEFAULT 'success';

-- Add previous_hash column for tamper detection via hash chain
ALTER TABLE audit_logs ADD COLUMN previous_hash VARCHAR(64);

-- Add IP anonymization tracking
ALTER TABLE audit_logs ADD COLUMN ip_anonymized_at TIMESTAMPTZ;

-- Add constraint to ensure outcome is either 'success' or 'failure'
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_outcome_check 
    CHECK (outcome IN ('success', 'failure'));

-- Add index for querying by outcome
CREATE INDEX idx_audit_logs_outcome ON audit_logs(outcome, created_at DESC);

-- Add index for finding logs needing IP anonymization (older than 90 days)
CREATE INDEX idx_audit_logs_ip_anonymization ON audit_logs(created_at, ip_anonymized_at) 
    WHERE ip_anonymized_at IS NULL AND ip_address IS NOT NULL;

COMMENT ON COLUMN audit_logs.outcome IS 'Operation outcome: success or failure';
COMMENT ON COLUMN audit_logs.previous_hash IS 'SHA-256 hash of previous log entry for tamper detection';
COMMENT ON COLUMN audit_logs.ip_anonymized_at IS 'Timestamp when IP address was anonymized (after 90 days)';
