-- Add image_url column to authors table
ALTER TABLE authors ADD COLUMN IF NOT EXISTS image_url VARCHAR(500);

-- Add index for performance if querying by image presence
CREATE INDEX IF NOT EXISTS idx_authors_image_url ON authors(image_url) WHERE image_url IS NOT NULL;
