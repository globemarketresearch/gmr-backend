-- Add scheduled_publish_enabled flag to all three tables
ALTER TABLE reports ADD COLUMN IF NOT EXISTS scheduled_publish_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE blogs ADD COLUMN IF NOT EXISTS scheduled_publish_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE press_releases ADD COLUMN IF NOT EXISTS scheduled_publish_enabled BOOLEAN DEFAULT FALSE;

-- Create partial indexes for efficient scheduler queries
CREATE INDEX IF NOT EXISTS idx_reports_scheduled_publish
  ON reports(status, scheduled_publish_enabled, publish_date)
  WHERE scheduled_publish_enabled = TRUE AND status != 'published';

CREATE INDEX IF NOT EXISTS idx_blogs_scheduled_publish
  ON blogs(status, scheduled_publish_enabled, publish_date)
  WHERE scheduled_publish_enabled = TRUE AND status != 'published';

CREATE INDEX IF NOT EXISTS idx_press_releases_scheduled_publish
  ON press_releases(status, scheduled_publish_enabled, publish_date)
  WHERE scheduled_publish_enabled = TRUE AND status != 'published';
