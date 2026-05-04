-- Add report_url column to press_releases table
ALTER TABLE press_releases ADD COLUMN IF NOT EXISTS report_url VARCHAR(500);
