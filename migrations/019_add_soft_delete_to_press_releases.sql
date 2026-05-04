-- Add soft delete column to press_releases table
ALTER TABLE press_releases ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;

-- Create index for soft delete queries
CREATE INDEX idx_press_releases_deleted_at ON press_releases(deleted_at);
