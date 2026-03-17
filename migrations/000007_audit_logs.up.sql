-- Audit logs table for tracking access to sensitive endpoints
-- Used for compliance, incident response, and privacy accountability
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_did VARCHAR(255) NOT NULL,
    entity_type VARCHAR(50) NOT NULL, -- 'scene', 'event', 'user', 'admin_panel', etc.
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL, -- 'access_precise_location', 'view_admin_panel', etc.
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Metadata
    request_id VARCHAR(255), -- Correlation with request logs
    ip_address VARCHAR(45), -- IPv4 or IPv6 address without port
    user_agent TEXT
);

-- Index for querying by entity (most common query pattern)
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id, created_at DESC);

-- Index for querying by user
CREATE INDEX idx_audit_logs_user ON audit_logs(user_did, created_at DESC);

-- Index for querying by action
CREATE INDEX idx_audit_logs_action ON audit_logs(action, created_at DESC);

COMMENT ON TABLE audit_logs IS 'Audit trail for access to precise location and sensitive endpoints';
COMMENT ON COLUMN audit_logs.entity_type IS 'Type of entity accessed (scene, event, user, admin_panel, etc.)';
COMMENT ON COLUMN audit_logs.action IS 'Action performed (access_precise_location, view_admin_panel, etc.)';
