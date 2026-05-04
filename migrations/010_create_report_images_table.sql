-- Migration: Create report_images table for storing multiple images per report
-- Description: Stores image URLs from external services (charts, graphs, diagrams, infographics)
--              for internal/admin use only. Not exposed through public report APIs.

CREATE TABLE IF NOT EXISTS report_images (
    id BIGSERIAL PRIMARY KEY,

    -- Core fields
    report_id INTEGER NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    image_url VARCHAR(500) NOT NULL,

    -- Descriptive metadata
    title VARCHAR(255),

    -- Management fields
    is_active BOOLEAN DEFAULT TRUE,
    uploaded_by INTEGER REFERENCES users(id) ON DELETE SET NULL,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT chk_report_images_title_length CHECK (title IS NULL OR LENGTH(TRIM(title)) >= 2)
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_report_images_report_id ON report_images(report_id);
CREATE INDEX IF NOT EXISTS idx_report_images_is_active ON report_images(is_active);
CREATE INDEX IF NOT EXISTS idx_report_images_uploaded_by ON report_images(uploaded_by);

-- Composite index for common query pattern (active images for a report)
CREATE INDEX IF NOT EXISTS idx_report_images_report_active ON report_images(report_id, is_active);

-- Comments for documentation
COMMENT ON TABLE report_images IS 'Stores multiple images per report (charts, graphs, diagrams, infographics) for internal/admin use only';
COMMENT ON COLUMN report_images.id IS 'Primary key using BIGSERIAL for high volume (multiple images per report)';
COMMENT ON COLUMN report_images.report_id IS 'Foreign key to reports table, cascades on delete';
COMMENT ON COLUMN report_images.image_url IS 'URL to image from existing image service (max 500 chars)';
COMMENT ON COLUMN report_images.title IS 'Descriptive title for the image (optional, min 2 chars if provided)';
COMMENT ON COLUMN report_images.is_active IS 'Soft delete flag - false to hide without losing data';
COMMENT ON COLUMN report_images.uploaded_by IS 'Foreign key to users table, preserves image if user deleted (SET NULL)';
COMMENT ON COLUMN report_images.created_at IS 'Timestamp of image creation';
COMMENT ON COLUMN report_images.updated_at IS 'Timestamp of last update';
