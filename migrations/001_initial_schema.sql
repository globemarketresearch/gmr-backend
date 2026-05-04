-- Create categories table
CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    image_url VARCHAR(500),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create sub_categories table
CREATE TABLE IF NOT EXISTS sub_categories (
    id SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create market_segments table
CREATE TABLE IF NOT EXISTS market_segments (
    id SERIAL PRIMARY KEY,
    sub_category_id INTEGER NOT NULL REFERENCES sub_categories(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create reports table
CREATE TABLE IF NOT EXISTS reports (
    id SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES categories(id),
    sub_category_id INTEGER REFERENCES sub_categories(id),
    market_segment_id INTEGER REFERENCES market_segments(id),
    title VARCHAR(500) NOT NULL,
    slug VARCHAR(500) UNIQUE NOT NULL,
    description TEXT,
    summary TEXT,
    thumbnail_url VARCHAR(500),
    price DECIMAL(10, 2),
    currency VARCHAR(3) DEFAULT 'USD',
    page_count INTEGER DEFAULT 0,
    published_at TIMESTAMP,
    is_published BOOLEAN DEFAULT FALSE,
    is_featured BOOLEAN DEFAULT FALSE,
    view_count INTEGER DEFAULT 0,
    download_count INTEGER DEFAULT 0,
    meta_title VARCHAR(255),
    meta_description VARCHAR(500),
    meta_keywords VARCHAR(500),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create chart_metadata table
CREATE TABLE IF NOT EXISTS chart_metadata (
    id SERIAL PRIMARY KEY,
    report_id INTEGER NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    chart_type VARCHAR(50),
    description TEXT,
    data_points INTEGER DEFAULT 0,
    "order" INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);
CREATE INDEX IF NOT EXISTS idx_categories_is_active ON categories(is_active);

CREATE INDEX IF NOT EXISTS idx_sub_categories_slug ON sub_categories(slug);
CREATE INDEX IF NOT EXISTS idx_sub_categories_category_id ON sub_categories(category_id);
CREATE INDEX IF NOT EXISTS idx_sub_categories_is_active ON sub_categories(is_active);

CREATE INDEX IF NOT EXISTS idx_market_segments_slug ON market_segments(slug);
CREATE INDEX IF NOT EXISTS idx_market_segments_sub_category_id ON market_segments(sub_category_id);
CREATE INDEX IF NOT EXISTS idx_market_segments_is_active ON market_segments(is_active);

CREATE INDEX IF NOT EXISTS idx_reports_slug ON reports(slug);
CREATE INDEX IF NOT EXISTS idx_reports_category_id ON reports(category_id);
CREATE INDEX IF NOT EXISTS idx_reports_sub_category_id ON reports(sub_category_id);
CREATE INDEX IF NOT EXISTS idx_reports_market_segment_id ON reports(market_segment_id);
CREATE INDEX IF NOT EXISTS idx_reports_published_at ON reports(published_at);
CREATE INDEX IF NOT EXISTS idx_reports_is_published ON reports(is_published);

CREATE INDEX IF NOT EXISTS idx_chart_metadata_report_id ON chart_metadata(report_id);
CREATE INDEX IF NOT EXISTS idx_chart_metadata_is_active ON chart_metadata(is_active);
