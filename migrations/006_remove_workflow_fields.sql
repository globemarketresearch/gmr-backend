-- Migration: Remove workflow management fields from reports table
-- Created: 2026-01-05
-- Description: Removes workflow_status, scheduled_publish_at, approved_by, and approved_at
--              columns and their associated indexes from the reports table

-- Drop indexes first
DROP INDEX IF EXISTS idx_reports_workflow_status;
DROP INDEX IF EXISTS idx_reports_scheduled_publish_at;

-- Drop workflow management columns
ALTER TABLE reports DROP COLUMN IF EXISTS workflow_status;
ALTER TABLE reports DROP COLUMN IF EXISTS scheduled_publish_at;
ALTER TABLE reports DROP COLUMN IF EXISTS approved_by;
ALTER TABLE reports DROP COLUMN IF EXISTS approved_at;
