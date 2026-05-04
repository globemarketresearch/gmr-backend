-- Add GIN index for author_ids JSONB column in reports table
-- This index optimizes JSONB containment queries for author filtering

CREATE INDEX IF NOT EXISTS idx_reports_author_ids
ON reports USING GIN (author_ids jsonb_path_ops);
