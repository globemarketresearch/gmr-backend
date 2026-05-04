-- Fix scheduled_publish_enabled column to be non-nullable with default false
-- This resolves issues where NULL values prevent the scheduler from working correctly

-- Update any NULL values to false first
UPDATE blogs SET scheduled_publish_enabled = FALSE WHERE scheduled_publish_enabled IS NULL;
UPDATE press_releases SET scheduled_publish_enabled = FALSE WHERE scheduled_publish_enabled IS NULL;
UPDATE reports SET scheduled_publish_enabled = FALSE WHERE scheduled_publish_enabled IS NULL;

-- Alter columns to be non-nullable with default false
ALTER TABLE blogs ALTER COLUMN scheduled_publish_enabled SET NOT NULL;
ALTER TABLE blogs ALTER COLUMN scheduled_publish_enabled SET DEFAULT FALSE;

ALTER TABLE press_releases ALTER COLUMN scheduled_publish_enabled SET NOT NULL;
ALTER TABLE press_releases ALTER COLUMN scheduled_publish_enabled SET DEFAULT FALSE;

ALTER TABLE reports ALTER COLUMN scheduled_publish_enabled SET NOT NULL;
ALTER TABLE reports ALTER COLUMN scheduled_publish_enabled SET DEFAULT FALSE;
