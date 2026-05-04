-- Add internal_links JSONB column to reports, blogs, and press_releases tables
ALTER TABLE reports ADD COLUMN IF NOT EXISTS internal_links JSONB DEFAULT '[]'::jsonb;
ALTER TABLE blogs ADD COLUMN IF NOT EXISTS internal_links JSONB DEFAULT '[]'::jsonb;
ALTER TABLE press_releases ADD COLUMN IF NOT EXISTS internal_links JSONB DEFAULT '[]'::jsonb;
