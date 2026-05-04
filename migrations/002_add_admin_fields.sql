-- Migration: Add admin tracking and workflow fields to reports table
-- Created: 2026-01-03
-- Description: Adds user tracking (created_by, updated_by), workflow management,
--              and internal notes to support admin panel features

-- Add user tracking columns
ALTER TABLE reports ADD COLUMN IF NOT EXISTS created_by INTEGER REFERENCES users(id);
ALTER TABLE reports ADD COLUMN IF NOT EXISTS updated_by INTEGER REFERENCES users(id);
ALTER TABLE reports ADD COLUMN IF NOT EXISTS internal_notes TEXT;

-- Add workflow management columns
ALTER TABLE reports ADD COLUMN IF NOT EXISTS workflow_status VARCHAR(50) DEFAULT 'draft';
ALTER TABLE reports ADD COLUMN IF NOT EXISTS scheduled_publish_at TIMESTAMP;
ALTER TABLE reports ADD COLUMN IF NOT EXISTS approved_by INTEGER REFERENCES users(id);
ALTER TABLE reports ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP;

-- Add indexes for efficient filtering
CREATE INDEX IF NOT EXISTS idx_reports_created_by ON reports(created_by);
CREATE INDEX IF NOT EXISTS idx_reports_updated_by ON reports(updated_by);
CREATE INDEX IF NOT EXISTS idx_reports_workflow_status ON reports(workflow_status);
CREATE INDEX IF NOT EXISTS idx_reports_scheduled_publish_at ON reports(scheduled_publish_at);
CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at);

-- Add comment for workflow_status column
COMMENT ON COLUMN reports.workflow_status IS 'Workflow state: draft, pending_review, approved, rejected, scheduled, published, archived';
