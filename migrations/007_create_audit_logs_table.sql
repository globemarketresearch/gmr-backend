-- Migration: Create audit_logs table for tracking user actions
-- Description: Comprehensive audit logging with change tracking, IP addresses, and request context

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,

    -- Actor information
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    user_email VARCHAR(255),  -- Denormalized for historical records
    user_role VARCHAR(20),     -- Denormalized snapshot at time of action

    -- Action metadata
    action VARCHAR(100) NOT NULL,  -- e.g., "auth.login", "user.create", "report.update"
    entity_type VARCHAR(50),       -- e.g., "user", "report", "category", "author"
    entity_id INTEGER,             -- ID of affected entity

    -- Request context
    ip_address VARCHAR(45),        -- IPv4 or IPv6
    user_agent TEXT,               -- Browser/client user agent
    request_id VARCHAR(100),       -- Trace requests via middleware

    -- Change tracking (for updates)
    changes JSONB,                 -- {"field": {"old": "value", "new": "value"}}

    -- Result
    status VARCHAR(20) NOT NULL,   -- "success" or "failure"
    error_message TEXT,            -- If status is "failure", store error details

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_type ON audit_logs(entity_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_id ON audit_logs(entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_status ON audit_logs(status);
CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id ON audit_logs(request_id);

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_action ON audit_logs(user_id, action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(entity_type, entity_id);

-- Comments for documentation
COMMENT ON TABLE audit_logs IS 'Audit trail of all significant user actions in the system';
COMMENT ON COLUMN audit_logs.user_id IS 'Reference to users table, nullable to preserve logs if user deleted';
COMMENT ON COLUMN audit_logs.user_email IS 'Denormalized email for historical record keeping';
COMMENT ON COLUMN audit_logs.user_role IS 'User role at the time of action (denormalized for audit trail)';
COMMENT ON COLUMN audit_logs.action IS 'Standardized action identifier (e.g., auth.login, user.create)';
COMMENT ON COLUMN audit_logs.changes IS 'JSONB field storing before/after values for update operations';
COMMENT ON COLUMN audit_logs.request_id IS 'Request ID from middleware for tracing related operations';
