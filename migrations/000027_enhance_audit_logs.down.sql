-- Revert audit_logs enhancements

-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_ip_anonymization;
DROP INDEX IF EXISTS idx_audit_logs_outcome;

-- Drop columns (constraint will be dropped automatically)
ALTER TABLE audit_logs DROP COLUMN IF EXISTS ip_anonymized_at;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS previous_hash;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS outcome;
