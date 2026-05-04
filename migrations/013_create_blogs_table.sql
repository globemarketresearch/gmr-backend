-- Migration: Create blogs table
-- Description: Stores blog posts with metadata, status, and SEO information

CREATE TABLE IF NOT EXISTS blogs (
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
    CONSTRAINT chk_blogs_title_length CHECK (LENGTH(TRIM(title)) >= 10 AND LENGTH(TRIM(title)) <= 200),
    CONSTRAINT chk_blogs_excerpt_length CHECK (LENGTH(TRIM(excerpt)) >= 50 AND LENGTH(TRIM(excerpt)) <= 500),
    CONSTRAINT chk_blogs_content_length CHECK (LENGTH(TRIM(content)) >= 100),
    CONSTRAINT chk_blogs_status CHECK (status IN ('draft', 'review', 'published'))
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_blogs_category_id ON blogs(category_id);
CREATE INDEX IF NOT EXISTS idx_blogs_author_id ON blogs(author_id);
CREATE INDEX IF NOT EXISTS idx_blogs_status ON blogs(status);
CREATE INDEX IF NOT EXISTS idx_blogs_publish_date ON blogs(publish_date);
CREATE INDEX IF NOT EXISTS idx_blogs_location ON blogs(location);
CREATE INDEX IF NOT EXISTS idx_blogs_reviewed_by ON blogs(reviewed_by);

-- Composite index for common query pattern (published blogs by category)
CREATE INDEX IF NOT EXISTS idx_blogs_status_category ON blogs(status, category_id);

-- Full-text search index on title, excerpt, and content
CREATE INDEX IF NOT EXISTS idx_blogs_search ON blogs USING gin(to_tsvector('english', title || ' ' || excerpt || ' ' || content));

-- Comments for documentation
COMMENT ON TABLE blogs IS 'Stores blog posts with metadata, status, and SEO information';
COMMENT ON COLUMN blogs.id IS 'Primary key using BIGSERIAL for scalability';
COMMENT ON COLUMN blogs.title IS 'Blog title (10-200 characters)';
COMMENT ON COLUMN blogs.slug IS 'URL-friendly slug for the blog (unique)';
COMMENT ON COLUMN blogs.excerpt IS 'Short summary of the blog (50-500 characters)';
COMMENT ON COLUMN blogs.content IS 'HTML content of the blog (minimum 100 characters)';
COMMENT ON COLUMN blogs.category_id IS 'Foreign key to categories table';
COMMENT ON COLUMN blogs.tags IS 'Comma-separated tags for categorization';
COMMENT ON COLUMN blogs.author_id IS 'Foreign key to authors table';
COMMENT ON COLUMN blogs.status IS 'Blog status: draft, review, or published';
COMMENT ON COLUMN blogs.publish_date IS 'Date and time when the blog should be/was published';
COMMENT ON COLUMN blogs.location IS 'Optional location information';
COMMENT ON COLUMN blogs.metadata IS 'JSONB field for SEO metadata (metaTitle, metaDescription, keywords)';
COMMENT ON COLUMN blogs.reviewed_by IS 'User ID who reviewed the blog (if status is review or published)';
COMMENT ON COLUMN blogs.reviewed_at IS 'Timestamp when the blog was reviewed';
COMMENT ON COLUMN blogs.created_at IS 'Timestamp of blog creation';
COMMENT ON COLUMN blogs.updated_at IS 'Timestamp of last update';
