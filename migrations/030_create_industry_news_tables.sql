-- Migration: Create industry_news and industry_news_images tables
-- Description: Industry News content module (mirrors press_releases/blogs schema),
--              plus a per-article image gallery (mirrors report_images), and a
--              seeded default author for Industry News authorship.

CREATE TABLE IF NOT EXISTS industry_news (
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
    scheduled_publish_enabled BOOLEAN DEFAULT FALSE,

    -- Location (optional)
    location VARCHAR(255),

    -- SEO Metadata and internal links stored as JSONB
    metadata JSONB DEFAULT '{}'::jsonb,
    internal_links JSONB DEFAULT '[]'::jsonb,

    -- Review tracking
    reviewed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP,

    -- Soft delete
    deleted_at TIMESTAMP NULL DEFAULT NULL,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT chk_industry_news_title_length CHECK (LENGTH(TRIM(title)) >= 10 AND LENGTH(TRIM(title)) <= 200),
    CONSTRAINT chk_industry_news_excerpt_length CHECK (LENGTH(TRIM(excerpt)) >= 50 AND LENGTH(TRIM(excerpt)) <= 500),
    CONSTRAINT chk_industry_news_content_length CHECK (LENGTH(TRIM(content)) >= 100),
    CONSTRAINT chk_industry_news_status CHECK (status IN ('draft', 'review', 'published'))
);

CREATE INDEX IF NOT EXISTS idx_industry_news_category_id ON industry_news(category_id);
CREATE INDEX IF NOT EXISTS idx_industry_news_author_id ON industry_news(author_id);
CREATE INDEX IF NOT EXISTS idx_industry_news_status ON industry_news(status);
CREATE INDEX IF NOT EXISTS idx_industry_news_publish_date ON industry_news(publish_date);
CREATE INDEX IF NOT EXISTS idx_industry_news_location ON industry_news(location);
CREATE INDEX IF NOT EXISTS idx_industry_news_reviewed_by ON industry_news(reviewed_by);
CREATE INDEX IF NOT EXISTS idx_industry_news_deleted_at ON industry_news(deleted_at);
CREATE INDEX IF NOT EXISTS idx_industry_news_status_category ON industry_news(status, category_id);
CREATE INDEX IF NOT EXISTS idx_industry_news_search ON industry_news USING gin(to_tsvector('english', title || ' ' || excerpt || ' ' || content));

-- Partial index for the scheduled-publish sweep (mirrors 019_add_scheduled_publishing.sql)
CREATE INDEX IF NOT EXISTS idx_industry_news_scheduled_publish
  ON industry_news(status, scheduled_publish_enabled, publish_date)
  WHERE scheduled_publish_enabled = TRUE AND status != 'published';

COMMENT ON TABLE industry_news IS 'Stores industry news articles with metadata, status, and SEO information';
COMMENT ON COLUMN industry_news.title IS 'Industry news title (10-200 characters)';
COMMENT ON COLUMN industry_news.slug IS 'URL-friendly slug for the industry news article (unique)';
COMMENT ON COLUMN industry_news.excerpt IS 'Short summary of the article (50-500 characters)';
COMMENT ON COLUMN industry_news.content IS 'HTML content of the article (minimum 100 characters)';
COMMENT ON COLUMN industry_news.category_id IS 'Foreign key to categories table';
COMMENT ON COLUMN industry_news.tags IS 'Comma-separated tags for categorization';
COMMENT ON COLUMN industry_news.author_id IS 'Foreign key to authors table';
COMMENT ON COLUMN industry_news.status IS 'Article status: draft, review, or published';
COMMENT ON COLUMN industry_news.publish_date IS 'Date and time when the article should be/was published';
COMMENT ON COLUMN industry_news.location IS 'Optional location information';
COMMENT ON COLUMN industry_news.metadata IS 'JSONB field for SEO metadata (metaTitle, metaDescription, keywords)';
COMMENT ON COLUMN industry_news.internal_links IS 'JSONB field for internal link tracking';
COMMENT ON COLUMN industry_news.reviewed_by IS 'User ID who reviewed the article (if status is review or published)';
COMMENT ON COLUMN industry_news.reviewed_at IS 'Timestamp when the article was reviewed';

CREATE TABLE IF NOT EXISTS industry_news_images (
    id BIGSERIAL PRIMARY KEY,

    industry_news_id INTEGER NOT NULL REFERENCES industry_news(id) ON DELETE CASCADE,
    image_url VARCHAR(500) NOT NULL,

    title VARCHAR(255),

    is_active BOOLEAN DEFAULT TRUE,
    uploaded_by INTEGER REFERENCES users(id) ON DELETE SET NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_industry_news_images_title_length CHECK (title IS NULL OR LENGTH(TRIM(title)) >= 2)
);

CREATE INDEX IF NOT EXISTS idx_industry_news_images_industry_news_id ON industry_news_images(industry_news_id);
CREATE INDEX IF NOT EXISTS idx_industry_news_images_is_active ON industry_news_images(is_active);
CREATE INDEX IF NOT EXISTS idx_industry_news_images_uploaded_by ON industry_news_images(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_industry_news_images_news_active ON industry_news_images(industry_news_id, is_active);

COMMENT ON TABLE industry_news_images IS 'Stores multiple images per industry news article for admin use, pasted into the TipTap content editor';
COMMENT ON COLUMN industry_news_images.industry_news_id IS 'Foreign key to industry_news table, cascades on delete';
COMMENT ON COLUMN industry_news_images.image_url IS 'URL to image in Cloudflare R2 (max 500 chars)';
COMMENT ON COLUMN industry_news_images.title IS 'Descriptive title for the image (optional, min 2 chars if provided)';
COMMENT ON COLUMN industry_news_images.is_active IS 'Soft delete flag - false to hide without losing data';
COMMENT ON COLUMN industry_news_images.uploaded_by IS 'Foreign key to users table, preserves image if user deleted (SET NULL)';

-- Seed the default "Globe Market Research" author used for new Industry News articles.
-- Idempotent: only inserts if no author with this exact name already exists.
INSERT INTO authors (name, role, bio)
SELECT 'Globe Market Research', 'Editorial Team', 'Official news desk of Globe Market Research.'
WHERE NOT EXISTS (
    SELECT 1 FROM authors WHERE name = 'Globe Market Research'
);
