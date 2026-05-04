-- Add soft delete column to reports table
ALTER TABLE reports ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;

-- Create index for soft delete queries
CREATE INDEX idx_reports_deleted_at ON reports(deleted_at);
