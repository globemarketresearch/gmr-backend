-- Migration: Create press_releases table
-- Description: Stores press releases with metadata, status, and SEO information

CREATE TABLE IF NOT EXISTS press_releases (
    id BIGSERIAL PRIMARY KEY,

    -- Core fields
    title VARCHAR(200) NOT NULL,
    slug VARCHAR(250) UNIQUE NOT NULL,
    excerpt VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,

    -- Categorization
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    tags VARCHAR(500), -- comma-separated tags: "healthcare,ai,market-analysis"

    -- Author and workflow
    author_id INTEGER NOT NULL REFERENCES authors(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft, review, published
    publish_date TIMESTAMP,

    -- Location (optional)
    location VARCHAR(255),

    -- SEO Metadata stored as JSONB
    metadata JSONB DEFAULT '{}'::jsonb,

    -- Review tracking
    reviewed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT chk_press_releases_title_length CHECK (LENGTH(TRIM(title)) >= 10 AND LENGTH(TRIM(title)) <= 200),
    CONSTRAINT chk_press_releases_excerpt_length CHECK (LENGTH(TRIM(excerpt)) >= 50 AND LENGTH(TRIM(excerpt)) <= 500),
    CONSTRAINT chk_press_releases_content_length CHECK (LENGTH(TRIM(content)) >= 100),
    CONSTRAINT chk_press_releases_status CHECK (status IN ('draft', 'review', 'published'))
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_press_releases_category_id ON press_releases(category_id);
CREATE INDEX IF NOT EXISTS idx_press_releases_author_id ON press_releases(author_id);
CREATE INDEX IF NOT EXISTS idx_press_releases_status ON press_releases(status);
CREATE INDEX IF NOT EXISTS idx_press_releases_publish_date ON press_releases(publish_date);
CREATE INDEX IF NOT EXISTS idx_press_releases_location ON press_releases(location);
CREATE INDEX IF NOT EXISTS idx_press_releases_reviewed_by ON press_releases(reviewed_by);

-- Composite index for common query pattern (published press releases by category)
CREATE INDEX IF NOT EXISTS idx_press_releases_status_category ON press_releases(status, category_id);

-- Full-text search index on title, excerpt, and content
CREATE INDEX IF NOT EXISTS idx_press_releases_search ON press_releases USING gin(to_tsvector('english', title || ' ' || excerpt || ' ' || content));

-- Comments for documentation
COMMENT ON TABLE press_releases IS 'Stores press releases with metadata, status, and SEO information';
COMMENT ON COLUMN press_releases.id IS 'Primary key using BIGSERIAL for scalability';
COMMENT ON COLUMN press_releases.title IS 'Press release title (10-200 characters)';
COMMENT ON COLUMN press_releases.slug IS 'URL-friendly slug for the press release (unique)';
COMMENT ON COLUMN press_releases.excerpt IS 'Short summary of the press release (50-500 characters)';
COMMENT ON COLUMN press_releases.content IS 'HTML content of the press release (minimum 100 characters)';
COMMENT ON COLUMN press_releases.category_id IS 'Foreign key to categories table';
COMMENT ON COLUMN press_releases.tags IS 'Comma-separated tags for categorization';
COMMENT ON COLUMN press_releases.author_id IS 'Foreign key to authors table';
COMMENT ON COLUMN press_releases.status IS 'Press release status: draft, review, or published';
COMMENT ON COLUMN press_releases.publish_date IS 'Date and time when the press release should be/was published';
COMMENT ON COLUMN press_releases.location IS 'Optional location information';
COMMENT ON COLUMN press_releases.metadata IS 'JSONB field for SEO metadata (metaTitle, metaDescription, keywords)';
COMMENT ON COLUMN press_releases.reviewed_by IS 'User ID who reviewed the press release (if status is review or published)';
COMMENT ON COLUMN press_releases.reviewed_at IS 'Timestamp when the press release was reviewed';
COMMENT ON COLUMN press_releases.created_at IS 'Timestamp of press release creation';
COMMENT ON COLUMN press_releases.updated_at IS 'Timestamp of last update';
