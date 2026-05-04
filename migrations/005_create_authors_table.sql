-- Create authors table
CREATE TABLE IF NOT EXISTS authors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(100),
    bio TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_authors_name ON authors(name);

-- Add constraint to ensure name has minimum length
ALTER TABLE authors ADD CONSTRAINT check_name_length CHECK (LENGTH(name) >= 2);

-- Add constraint to limit bio length
ALTER TABLE authors ADD CONSTRAINT check_bio_length CHECK (LENGTH(bio) <= 1000);
