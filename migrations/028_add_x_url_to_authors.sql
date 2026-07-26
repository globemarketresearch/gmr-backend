-- Add x_url column to authors table
ALTER TABLE authors ADD COLUMN IF NOT EXISTS x_url VARCHAR(500);

-- Add index for performance if querying by X presence
CREATE INDEX IF NOT EXISTS idx_authors_x_url ON authors(x_url) WHERE x_url IS NOT NULL;
