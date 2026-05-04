-- Add soft delete column to blogs table
ALTER TABLE blogs ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;

-- Create index for soft delete queries
CREATE INDEX idx_blogs_deleted_at ON blogs(deleted_at);
