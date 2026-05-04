-- Remove unused fields from reports table
-- These fields were not being actively used in the application:
-- - thumbnail_url: Never written to or displayed
-- - view_count: Referenced in dashboard but never incremented
-- - download_count: Referenced in dashboard but never incremented

ALTER TABLE reports DROP COLUMN IF EXISTS thumbnail_url;
ALTER TABLE reports DROP COLUMN IF EXISTS view_count;
ALTER TABLE reports DROP COLUMN IF EXISTS download_count;
