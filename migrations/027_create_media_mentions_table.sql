-- Create media_mentions table
CREATE TABLE IF NOT EXISTS media_mentions (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    link VARCHAR(500),
    image_url VARCHAR(500),
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index for ordering
CREATE INDEX IF NOT EXISTS idx_media_mentions_display_order ON media_mentions(display_order);

-- Add constraint to ensure title has minimum length
ALTER TABLE media_mentions ADD CONSTRAINT check_media_mention_title_length CHECK (LENGTH(title) >= 2);
