-- Remove sub_category_id and market_segment_id columns from reports table
ALTER TABLE reports DROP COLUMN IF EXISTS sub_category_id;
ALTER TABLE reports DROP COLUMN IF EXISTS market_segment_id;

-- Drop indexes for sub_categories
DROP INDEX IF EXISTS idx_sub_categories_slug;
DROP INDEX IF EXISTS idx_sub_categories_category_id;
DROP INDEX IF EXISTS idx_sub_categories_is_active;

-- Drop indexes for market_segments
DROP INDEX IF EXISTS idx_market_segments_slug;
DROP INDEX IF EXISTS idx_market_segments_sub_category_id;
DROP INDEX IF EXISTS idx_market_segments_is_active;

-- Drop indexes from reports table
DROP INDEX IF EXISTS idx_reports_sub_category_id;
DROP INDEX IF EXISTS idx_reports_market_segment_id;

-- Drop market_segments table
DROP TABLE IF EXISTS market_segments CASCADE;

-- Drop sub_categories table
DROP TABLE IF EXISTS sub_categories CASCADE;
