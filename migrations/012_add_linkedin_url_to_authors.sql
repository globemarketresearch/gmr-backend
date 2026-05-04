-- Add linkedin_url column to authors table
ALTER TABLE authors ADD COLUMN IF NOT EXISTS linkedin_url VARCHAR(500);

-- Add index for performance if querying by LinkedIn presence
CREATE INDEX IF NOT EXISTS idx_authors_linkedin_url ON authors(linkedin_url) WHERE linkedin_url IS NOT NULL;
