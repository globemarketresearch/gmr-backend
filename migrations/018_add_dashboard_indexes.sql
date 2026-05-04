-- Add indexes for dashboard performance optimization
-- These indexes improve query performance for dashboard statistics

-- Reports table indexes
CREATE INDEX IF NOT EXISTS idx_reports_status_created
ON reports(status, created_at DESC)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_reports_view_count
ON reports(view_count DESC)
WHERE status = 'published' AND deleted_at IS NULL;

-- Form submissions table indexes
CREATE INDEX IF NOT EXISTS idx_form_submissions_created
ON form_submissions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_form_submissions_status
ON form_submissions(status);

-- Audit logs table indexes
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_action
ON audit_logs(created_at DESC, action);

CREATE INDEX IF NOT EXISTS idx_audit_logs_entity
ON audit_logs(entity_type, entity_id);

-- Blog posts table indexes
CREATE INDEX IF NOT EXISTS idx_blogs_status_created
ON blogs(status, created_at DESC)
WHERE deleted_at IS NULL;

-- Press releases table indexes
CREATE INDEX IF NOT EXISTS idx_press_releases_status_created
ON press_releases(status, created_at DESC)
WHERE deleted_at IS NULL;

-- Users table indexes for dashboard stats
CREATE INDEX IF NOT EXISTS idx_users_role
ON users(role);

CREATE INDEX IF NOT EXISTS idx_users_last_login
ON users(last_login_at DESC);
