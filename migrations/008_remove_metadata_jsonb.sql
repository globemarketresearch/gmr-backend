-- Migration to remove metadata JSONB column and use individual fields
-- This migrates data from metadata JSONB to individual columns

-- Step 1: Migrate data from metadata JSONB to individual columns where they're empty
UPDATE reports
SET
    meta_title = COALESCE(
        NULLIF(meta_title, ''),
        metadata->>'metaTitle',
        ''
    ),
    meta_description = COALESCE(
        NULLIF(meta_description, ''),
        metadata->>'metaDescription',
        ''
    ),
    meta_keywords = COALESCE(
        NULLIF(meta_keywords, ''),
        array_to_string(
            ARRAY(SELECT jsonb_array_elements_text(metadata->'keywords')),
            ','
        ),
        ''
    )
WHERE metadata IS NOT NULL AND metadata != '{}'::jsonb;

-- Step 2: Drop the metadata JSONB column from reports
ALTER TABLE reports DROP COLUMN IF EXISTS metadata;

-- Step 3: Update report_versions table - add individual metadata columns
ALTER TABLE report_versions ADD COLUMN IF NOT EXISTS meta_title VARCHAR(255);
ALTER TABLE report_versions ADD COLUMN IF NOT EXISTS meta_description VARCHAR(500);
ALTER TABLE report_versions ADD COLUMN IF NOT EXISTS meta_keywords VARCHAR(500);

-- Step 4: Migrate existing version metadata
UPDATE report_versions
SET
    meta_title = metadata->>'metaTitle',
    meta_description = metadata->>'metaDescription',
    meta_keywords = array_to_string(
        ARRAY(SELECT jsonb_array_elements_text(metadata->'keywords')),
        ','
    )
WHERE metadata IS NOT NULL AND metadata != '{}'::jsonb;

-- Step 5: Drop metadata from versions table
ALTER TABLE report_versions DROP COLUMN IF EXISTS metadata;
