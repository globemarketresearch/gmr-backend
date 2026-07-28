# Industry News Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a third content module, "Industry News", to `gmr-backend`, `gmr-admin`, and `gmr`, matching the Press Release module's feature set (categories, tags, draft/review/publish workflow, scheduled publishing, SEO metadata, soft delete/trash), plus a seeded default author ("Globe Market Research") and a Report-style image gallery manager in the admin form.

**Architecture:** Backend is Go + Fiber + GORM over Postgres, layered `domain → repository → service → handler`, wired in `cmd/api/main.go`. GORM `AutoMigrate` (driven by the struct list in `internal/db/db.go`) creates tables/columns/basic FKs at server startup; the numbered `migrations/*.sql` files are a human-run supplement for CHECK constraints, custom indexes, and full-text search that AutoMigrate does not create — they are not executed by any Go code, so both must be kept in sync manually. Admin and frontend are separate Next.js App Router apps, each importing typed API clients per feature folder.

**Tech Stack:** Go 1.x, Fiber v2, GORM (Postgres driver), Next.js 15 App Router, React 19, react-hook-form + zod, TipTap v3, Tailwind CSS v4, shadcn/ui.

## Global Constraints

- Default author name for new Industry News articles: `"Globe Market Research"` — seeded into the `authors` table, pre-selected (not hardcoded) in the admin form's `AuthorSelector`.
- Image upload must reuse the existing Cloudflare R2 upload service (`internal/service/cloudflare_images_service.go`) unchanged — no new storage backend code.
- Public frontend routes: `/industry-news` (list) and `/industry-news/[slug]` (detail) — both plural, no PR-style singular/plural split.
- No changes to existing PR, Statistics/Blog, or Report modules.
- Backend has no test framework for CRUD-content domains (press_release/blog have zero `*_test.go` files) — those layers get a manual `curl`/`go build` verification step, not invented unit tests. Image-gallery domains DO have precedent tests (`report_image_repository_test.go` using `gorm.io/driver/sqlite` in-memory + testify; `report_image_service_test.go` using hand-rolled mocks + stdlib `testing`) — the Industry News image layers must follow those same two patterns.
- `gmr-admin` and `gmr` have no test framework configured (no Jest/Vitest/Playwright config found) — admin/frontend tasks have no automated test step; verification is `npm run type-check` / `npm run build` plus a manual walkthrough note.
- Prettier (gmr-admin): 100-char width, single quotes, 2-space indent, trailing commas (ES5), semicolons. Imports use the `@/` path alias.

---

## Task 1: Migration SQL — `industry_news` + `industry_news_images` tables + default author seed

**Files:**
- Create: `gmr-backend/migrations/030_create_industry_news_tables.sql`

**Interfaces:**
- Produces: tables `industry_news`, `industry_news_images`, and a seeded `authors` row named `Globe Market Research`, matching the columns Task 2's GORM structs will declare.

- [ ] **Step 1: Write the migration file**

```sql
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
```

- [ ] **Step 2: Manually verify against the local dev database**

This project does not auto-run `migrations/*.sql` from Go code — `internal/db/db.go`'s `Migrate()` uses GORM `AutoMigrate` against the struct list (wired in Task 2), and the numbered SQL files are applied by hand as a supplement for constraints/indexes AutoMigrate can't express. Run, against your local dev Postgres:

```bash
psql "$DATABASE_URL" -f migrations/030_create_industry_news_tables.sql
```

Expected: no errors; `\d industry_news` and `\d industry_news_images` in `psql` show the columns/constraints above, and `SELECT name FROM authors WHERE name = 'Globe Market Research';` returns one row. If Task 2 has already run once (AutoMigrate created bare tables first), this file is still safe to run — every statement is `IF NOT EXISTS`/idempotent except the `CHECK` constraints, which will error harmlessly with "already exists" if genuinely duplicated; that's expected and fine.

- [ ] **Step 3: Commit**

```bash
git add migrations/030_create_industry_news_tables.sql
git commit -m "Add industry_news and industry_news_images migration"
```

---

## Task 2: Domain structs, audit constants, AutoMigrate registration

**Files:**
- Create: `gmr-backend/internal/domain/industry_news/industry_news.go`
- Create: `gmr-backend/internal/domain/industry_news/industry_news_image.go`
- Modify: `gmr-backend/internal/domain/audit/audit_log.go`
- Modify: `gmr-backend/internal/db/db.go`

**Interfaces:**
- Produces: `industry_news.IndustryNews`, `industry_news.IndustryNewsStatus` (`StatusDraft`/`StatusReview`/`StatusPublished`), `industry_news.InternalLinkEntry`/`InternalLinks`, `industry_news.IndustryNewsMetadata`, `industry_news.CreateIndustryNewsRequest`, `industry_news.UpdateIndustryNewsRequest`, `industry_news.GetIndustryNewsQuery`, `industry_news.IndustryNewsListResponse`, `industry_news.IndustryNewsResponse`, `industry_news.IndustryNewsImage`; `audit.ActionIndustryNewsCreate/Update/Delete/Publish`, `audit.EntityIndustryNews`.
- Consumes: `internal/domain/author.Author`, `internal/domain/category.Category` (existing).

- [ ] **Step 1: Write `industry_news.go`**

```go
package industry_news

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/author"
	"github.com/healthcare-market-research/backend/internal/domain/category"
)

// IndustryNewsStatus represents the status of an industry news article
type IndustryNewsStatus string

const (
	StatusDraft     IndustryNewsStatus = "draft"
	StatusReview    IndustryNewsStatus = "review"
	StatusPublished IndustryNewsStatus = "published"
)

// InternalLinkEntry represents a single internal link configuration
type InternalLinkEntry struct {
	Keyword     string `json:"keyword"`
	TargetID    int    `json:"targetId"`
	TargetTitle string `json:"targetTitle"`
	TargetType  string `json:"targetType"` // "report", "blog", "press-release", "industry-news"
	TargetURL   string `json:"targetUrl"`
	LinkedCount int    `json:"linkedCount"`
}

// InternalLinks is a slice of InternalLinkEntry with JSONB support
type InternalLinks []InternalLinkEntry

func (il InternalLinks) Value() (driver.Value, error) {
	return json.Marshal(il)
}

func (il *InternalLinks) Scan(value interface{}) error {
	if value == nil {
		*il = InternalLinks{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &il)
}

// IndustryNewsMetadata contains SEO metadata for an industry news article
type IndustryNewsMetadata struct {
	MetaTitle       string   `json:"metaTitle,omitempty"`
	MetaDescription string   `json:"metaDescription,omitempty"`
	Keywords        []string `json:"keywords,omitempty"`
}

func (m IndustryNewsMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *IndustryNewsMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &m)
}

// IndustryNews represents an industry news article in the database
type IndustryNews struct {
	ID          uint                `json:"id" gorm:"primaryKey"`
	Title       string              `json:"title" gorm:"type:varchar(200);not null"`
	Slug        string              `json:"slug" gorm:"type:varchar(250);uniqueIndex;not null"`
	Excerpt     string              `json:"excerpt" gorm:"type:varchar(500);not null"`
	Content     string              `json:"content" gorm:"type:text;not null"`
	CategoryID  uint                `json:"categoryId" gorm:"not null;index"`
	Tags        string              `json:"tags" gorm:"type:varchar(500)"`
	AuthorID    uint                `json:"authorId" gorm:"not null;index"`
	Author      *author.Author      `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
	Category    *category.Category  `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Status                  IndustryNewsStatus   `json:"status" gorm:"type:varchar(20);default:'draft';index"`
	PublishDate             *time.Time           `json:"publishDate,omitempty" gorm:"index"`
	ScheduledPublishEnabled bool                 `json:"scheduledPublishEnabled" gorm:"default:false"`
	Location                string               `json:"location,omitempty" gorm:"type:varchar(255)"`
	Metadata      IndustryNewsMetadata   `json:"metadata" gorm:"type:jsonb"`
	InternalLinks InternalLinks          `json:"internalLinks,omitempty" gorm:"type:jsonb"`
	ReviewedBy    *uint                  `json:"reviewedBy,omitempty" gorm:"index"`
	ReviewedAt  *time.Time             `json:"reviewedAt,omitempty"`
	DeletedAt   *time.Time             `json:"deletedAt,omitempty" gorm:"index"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

func (IndustryNews) TableName() string {
	return "industry_news"
}

// CreateIndustryNewsRequest is the request body for creating a new industry news article
type CreateIndustryNewsRequest struct {
	Title       string                 `json:"title" validate:"required,min=10,max=200"`
	Excerpt     string                 `json:"excerpt" validate:"required,min=50,max=500"`
	Content     string                 `json:"content" validate:"required,min=100"`
	CategoryID  uint                   `json:"categoryId" validate:"required"`
	Tags        string                 `json:"tags"`
	AuthorID    uint                   `json:"authorId" validate:"required"`
	Status      IndustryNewsStatus     `json:"status" validate:"required,oneof=draft review published"`
	PublishDate string                 `json:"publishDate" validate:"required"`
	Location    string                 `json:"location,omitempty"`
	Metadata    *IndustryNewsMetadata  `json:"metadata,omitempty"`
}

// UpdateIndustryNewsRequest is the request body for updating an industry news article
type UpdateIndustryNewsRequest struct {
	Title       *string                `json:"title,omitempty" validate:"omitempty,min=10,max=200"`
	Slug        *string                `json:"slug,omitempty" validate:"omitempty,min=1,max=250"`
	Excerpt     *string                `json:"excerpt,omitempty" validate:"omitempty,min=50,max=500"`
	Content     *string                `json:"content,omitempty" validate:"omitempty,min=100"`
	CategoryID  *uint                  `json:"categoryId,omitempty"`
	Tags        *string                `json:"tags,omitempty"`
	AuthorID    *uint                  `json:"authorId,omitempty"`
	Status      *IndustryNewsStatus    `json:"status,omitempty" validate:"omitempty,oneof=draft review published"`
	PublishDate *string                `json:"publishDate,omitempty"`
	Location      *string                `json:"location,omitempty"`
	Metadata      *IndustryNewsMetadata  `json:"metadata,omitempty"`
	InternalLinks *InternalLinks         `json:"internalLinks,omitempty"`
}

// GetIndustryNewsQuery represents query parameters for filtering industry news
type GetIndustryNewsQuery struct {
	Status       string
	CategoryID   string
	CategorySlug string
	Tags         string
	AuthorID     string
	Location     string
	Search       string
	Deleted      string
	Page         int
	Limit        int
	SortBy       string
}

// IndustryNewsListResponse represents a list of industry news articles with pagination
type IndustryNewsListResponse struct {
	IndustryNews []IndustryNews `json:"industryNews"`
	Total        int64          `json:"total"`
	Page         int            `json:"page"`
	Limit        int            `json:"limit"`
	TotalPages   int            `json:"totalPages"`
}

// IndustryNewsResponse represents a single industry news article response
type IndustryNewsResponse struct {
	IndustryNews IndustryNews `json:"industryNews"`
}
```

- [ ] **Step 2: Write `industry_news_image.go`**

```go
package industry_news

import "time"

// IndustryNewsImage represents an image associated with an industry news article.
// These images are for admin use only (copy URL into the TipTap content editor).
type IndustryNewsImage struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	IndustryNewsID uint      `json:"industry_news_id" gorm:"index:idx_industry_news_images_news_id;index:idx_industry_news_images_news_active,priority:1;not null;constraint:OnDelete:CASCADE"`
	ImageURL       string    `json:"image_url" gorm:"type:varchar(500);not null"`
	Title          string    `json:"title,omitempty" gorm:"type:varchar(255)"`
	IsActive       bool      `json:"is_active" gorm:"default:true;index:idx_industry_news_images_is_active;index:idx_industry_news_images_news_active,priority:2"`
	UploadedBy     *uint     `json:"uploaded_by,omitempty" gorm:"index:idx_industry_news_images_uploaded_by;constraint:OnDelete:SET NULL"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	IndustryNews *IndustryNews `json:"industry_news,omitempty" gorm:"foreignKey:IndustryNewsID;constraint:OnDelete:CASCADE"`
}

func (IndustryNewsImage) TableName() string {
	return "industry_news_images"
}
```

- [ ] **Step 3: Add audit constants**

In `gmr-backend/internal/domain/audit/audit_log.go`, add a new block after the existing `// Press Release actions` block (after line 50, before the closing `)` of the first `const` block):

```go
	// Industry News actions
	ActionIndustryNewsCreate  = "industry_news.create"
	ActionIndustryNewsUpdate  = "industry_news.update"
	ActionIndustryNewsDelete  = "industry_news.delete"
	ActionIndustryNewsPublish = "industry_news.publish"
```

And in the `// EntityType constants` block, add after `EntityPressRelease = "press_release"`:

```go
	EntityIndustryNews = "industry_news"
```

- [ ] **Step 4: Register both structs in GORM AutoMigrate**

In `gmr-backend/internal/db/db.go`, add the import:

```go
	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
```

And add both structs to the `DB.AutoMigrate(...)` call, after `&press_release.PressRelease{},`:

```go
		&industry_news.IndustryNews{},
		&industry_news.IndustryNewsImage{},
```

Then, immediately after the existing `fk_report_images_user` constraint-check block (after the `if !constraintExists { ... }` closes, before the `media_mentions` block), add the equivalent for the new image table:

```go
	// Add foreign key constraint for industry_news_images.uploaded_by (GORM doesn't auto-create this)
	var industryNewsImagesConstraintExists bool
	DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints
			WHERE constraint_name = 'fk_industry_news_images_user'
			AND table_name = 'industry_news_images'
			AND table_schema = CURRENT_SCHEMA()
		)
	`).Scan(&industryNewsImagesConstraintExists)

	if !industryNewsImagesConstraintExists {
		DB.Exec(`
			ALTER TABLE industry_news_images
			ADD CONSTRAINT fk_industry_news_images_user
			FOREIGN KEY (uploaded_by)
			REFERENCES users(id)
			ON DELETE SET NULL
		`)
	}
```

- [ ] **Step 5: Verify it builds**

Run: `cd gmr-backend && go build ./...`
Expected: no compile errors.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/industry_news internal/domain/audit/audit_log.go internal/db/db.go
git commit -m "Add industry_news domain structs and register with AutoMigrate"
```

---

## Task 3: `IndustryNewsRepository`

**Files:**
- Create: `gmr-backend/internal/repository/industry_news_repository.go`

**Interfaces:**
- Consumes: `industry_news.IndustryNews`, `industry_news.GetIndustryNewsQuery` (Task 2).
- Produces: `IndustryNewsRepository` interface + `NewIndustryNewsRepository(db *gorm.DB) IndustryNewsRepository`, methods: `Create`, `GetAll`, `GetByCategorySlug`, `GetByID`, `GetBySlug`, `Update`, `Delete`, `SoftDelete`, `Restore`, `SubmitForReview`, `Publish`, `Unpublish`, `PublishScheduled`, `SchedulePublish`, `CancelScheduledPublish` — used by Task 5 (service) and Task 6 (image service, for existence checks) and Task 7 (scheduler wiring).

- [ ] **Step 1: Write the repository**

```go
package repository

import (
	"strings"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
	"gorm.io/gorm"
)

type IndustryNewsRepository interface {
	Create(n *industry_news.IndustryNews) error
	GetAll(query industry_news.GetIndustryNewsQuery) ([]industry_news.IndustryNews, int64, error)
	GetByCategorySlug(categorySlug string, page, limit int) ([]industry_news.IndustryNews, int64, error)
	GetByID(id uint) (*industry_news.IndustryNews, error)
	GetBySlug(slug string) (*industry_news.IndustryNews, error)
	Update(id uint, updates map[string]interface{}) error
	Delete(id uint) error
	SoftDelete(id uint) error
	Restore(id uint) error
	SubmitForReview(id uint) error
	Publish(id uint) error
	Unpublish(id uint) error
	PublishScheduled(now time.Time) error
	SchedulePublish(id uint, publishDate time.Time) error
	CancelScheduledPublish(id uint) error
}

type industryNewsRepository struct {
	db *gorm.DB
}

func NewIndustryNewsRepository(db *gorm.DB) IndustryNewsRepository {
	return &industryNewsRepository{db: db}
}

func (r *industryNewsRepository) Create(n *industry_news.IndustryNews) error {
	return r.db.Create(n).Error
}

func (r *industryNewsRepository) GetAll(query industry_news.GetIndustryNewsQuery) ([]industry_news.IndustryNews, int64, error) {
	var items []industry_news.IndustryNews
	var total int64

	db := r.db.Model(&industry_news.IndustryNews{})

	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	if query.CategorySlug != "" {
		db = db.Where("category_id = (SELECT id FROM categories WHERE slug = ? AND is_active = true)", query.CategorySlug)
	} else if query.CategoryID != "" {
		db = db.Where("category_id = ?", query.CategoryID)
	}

	if query.Tags != "" {
		tags := strings.Split(query.Tags, ",")
		tagConditions := make([]string, 0, len(tags))
		tagValues := make([]interface{}, 0, len(tags))

		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagConditions = append(tagConditions, "tags LIKE ?")
				tagValues = append(tagValues, "%"+tag+"%")
			}
		}

		if len(tagConditions) > 0 {
			db = db.Where(strings.Join(tagConditions, " OR "), tagValues...)
		}
	}

	if query.AuthorID != "" {
		db = db.Where("author_id = ?", query.AuthorID)
	}

	if query.Location != "" {
		db = db.Where("location ILIKE ?", "%"+query.Location+"%")
	}

	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("title ILIKE ? OR excerpt ILIKE ? OR content ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if query.Deleted == "true" {
		db = db.Where("deleted_at IS NOT NULL")
	} else {
		db = db.Where("deleted_at IS NULL")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.Limit
	orderClause := "created_at DESC"
	if query.SortBy != "" {
		orderClause = query.SortBy
	}
	db = db.Order(orderClause).Offset(offset).Limit(query.Limit)

	if err := db.Preload("Author").Preload("Category").Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *industryNewsRepository) GetByCategorySlug(categorySlug string, page, limit int) ([]industry_news.IndustryNews, int64, error) {
	var items []industry_news.IndustryNews
	var total int64

	offset := (page - 1) * limit

	if err := r.db.Model(&industry_news.IndustryNews{}).
		Where("category_id = (SELECT id FROM categories WHERE slug = ? AND is_active = true) AND deleted_at IS NULL AND status = 'published'", categorySlug).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Where("category_id = (SELECT id FROM categories WHERE slug = ? AND is_active = true) AND deleted_at IS NULL AND status = 'published'", categorySlug).
		Order("created_at DESC").Offset(offset).Limit(limit).
		Preload("Author").Preload("Category").Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *industryNewsRepository) GetByID(id uint) (*industry_news.IndustryNews, error) {
	var n industry_news.IndustryNews
	if err := r.db.Preload("Author").Preload("Category").Where("deleted_at IS NULL").First(&n, id).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *industryNewsRepository) GetBySlug(slug string) (*industry_news.IndustryNews, error) {
	var n industry_news.IndustryNews
	if err := r.db.Preload("Author").Preload("Category").Where("slug = ? AND deleted_at IS NULL", slug).First(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *industryNewsRepository) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Updates(updates).Error
}

func (r *industryNewsRepository) Delete(id uint) error {
	return r.db.Delete(&industry_news.IndustryNews{}, id).Error
}

func (r *industryNewsRepository) SubmitForReview(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Update("status", industry_news.StatusReview).Error
}

func (r *industryNewsRepository) Publish(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Update("status", industry_news.StatusPublished).Error
}

func (r *industryNewsRepository) Unpublish(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Update("status", industry_news.StatusDraft).Error
}

func (r *industryNewsRepository) SoftDelete(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *industryNewsRepository) Restore(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

func (r *industryNewsRepository) PublishScheduled(now time.Time) error {
	return r.db.Model(&industry_news.IndustryNews{}).
		Where("scheduled_publish_enabled = ? AND status != ? AND publish_date <= ?",
			true, industry_news.StatusPublished, now).
		Updates(map[string]interface{}{
			"status":                    industry_news.StatusPublished,
			"scheduled_publish_enabled": false,
		}).Error
}

func (r *industryNewsRepository) SchedulePublish(id uint, publishDate time.Time) error {
	return r.db.Model(&industry_news.IndustryNews{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"publish_date":              publishDate,
			"scheduled_publish_enabled": true,
		}).Error
}

func (r *industryNewsRepository) CancelScheduledPublish(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).
		Where("id = ?", id).
		Update("scheduled_publish_enabled", false).Error
}
```

- [ ] **Step 2: Verify it builds**

Run: `cd gmr-backend && go build ./...`
Expected: no compile errors. No test file is added here — `press_release_repository.go` (the direct template) has no `*_test.go` precedent in this codebase.

- [ ] **Step 3: Commit**

```bash
git add internal/repository/industry_news_repository.go
git commit -m "Add IndustryNewsRepository"
```

---

## Task 4: `IndustryNewsImageRepository` + repository test

**Files:**
- Create: `gmr-backend/internal/repository/industry_news_image_repository.go`
- Test: `gmr-backend/internal/repository/industry_news_image_repository_test.go`

**Interfaces:**
- Consumes: `industry_news.IndustryNewsImage`, `industry_news.IndustryNews` (Task 2).
- Produces: `IndustryNewsImageRepository` interface + `NewIndustryNewsImageRepository(db *gorm.DB) IndustryNewsImageRepository`, methods: `Create`, `FindByID`, `FindByIndustryNewsID`, `FindActiveByIndustryNewsID`, `Update`, `Delete`, `CountByIndustryNewsID`, `CountActiveByIndustryNewsID` — used by Task 6 (image service).

- [ ] **Step 1: Write the repository**

```go
package repository

import (
	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
	"gorm.io/gorm"
)

type IndustryNewsImageRepository interface {
	Create(image *industry_news.IndustryNewsImage) error
	FindByID(id uint) (*industry_news.IndustryNewsImage, error)
	FindByIndustryNewsID(industryNewsID uint) ([]industry_news.IndustryNewsImage, error)
	FindActiveByIndustryNewsID(industryNewsID uint) ([]industry_news.IndustryNewsImage, error)
	Update(image *industry_news.IndustryNewsImage) error
	Delete(id uint) error
	CountByIndustryNewsID(industryNewsID uint) (int64, error)
	CountActiveByIndustryNewsID(industryNewsID uint) (int64, error)
}

type industryNewsImageRepository struct {
	db *gorm.DB
}

func NewIndustryNewsImageRepository(db *gorm.DB) IndustryNewsImageRepository {
	return &industryNewsImageRepository{db: db}
}

func (r *industryNewsImageRepository) Create(image *industry_news.IndustryNewsImage) error {
	return r.db.Create(image).Error
}

func (r *industryNewsImageRepository) FindByID(id uint) (*industry_news.IndustryNewsImage, error) {
	var image industry_news.IndustryNewsImage
	err := r.db.First(&image, id).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

func (r *industryNewsImageRepository) FindByIndustryNewsID(industryNewsID uint) ([]industry_news.IndustryNewsImage, error) {
	var images []industry_news.IndustryNewsImage
	err := r.db.Where("industry_news_id = ?", industryNewsID).
		Order("created_at DESC").
		Find(&images).Error
	return images, err
}

func (r *industryNewsImageRepository) FindActiveByIndustryNewsID(industryNewsID uint) ([]industry_news.IndustryNewsImage, error) {
	var images []industry_news.IndustryNewsImage
	err := r.db.Where("industry_news_id = ? AND is_active = ?", industryNewsID, true).
		Order("created_at DESC").
		Find(&images).Error
	return images, err
}

func (r *industryNewsImageRepository) Update(image *industry_news.IndustryNewsImage) error {
	return r.db.Save(image).Error
}

func (r *industryNewsImageRepository) Delete(id uint) error {
	return r.db.Delete(&industry_news.IndustryNewsImage{}, id).Error
}

func (r *industryNewsImageRepository) CountByIndustryNewsID(industryNewsID uint) (int64, error) {
	var count int64
	err := r.db.Model(&industry_news.IndustryNewsImage{}).
		Where("industry_news_id = ?", industryNewsID).
		Count(&count).Error
	return count, err
}

func (r *industryNewsImageRepository) CountActiveByIndustryNewsID(industryNewsID uint) (int64, error) {
	var count int64
	err := r.db.Model(&industry_news.IndustryNewsImage{}).
		Where("industry_news_id = ? AND is_active = ?", industryNewsID, true).
		Count(&count).Error
	return count, err
}
```

- [ ] **Step 2: Write the failing test**

```go
package repository

import (
	"testing"

	"github.com/healthcare-market-research/backend/internal/domain/category"
	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupIndustryNewsImageTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&category.Category{}, &industry_news.IndustryNews{}, &industry_news.IndustryNewsImage{})
	require.NoError(t, err)

	return db
}

func createTestIndustryNews(t *testing.T, db *gorm.DB) uint {
	cat := &category.Category{Name: "Test Category", Slug: "test-category"}
	require.NoError(t, db.Create(cat).Error)

	item := &industry_news.IndustryNews{
		Title:      "A Sufficiently Long Test Title",
		Slug:       "a-sufficiently-long-test-title",
		Excerpt:    "An excerpt that is long enough to satisfy the fifty character minimum requirement.",
		Content:    "Content that is long enough to satisfy the one hundred character minimum requirement for industry news articles in this test suite.",
		CategoryID: cat.ID,
		AuthorID:   1,
		Status:     industry_news.StatusDraft,
	}
	require.NoError(t, db.Create(item).Error)
	return item.ID
}

func TestIndustryNewsImageRepository_Create(t *testing.T) {
	db := setupIndustryNewsImageTestDB(t)
	repo := NewIndustryNewsImageRepository(db)
	newsID := createTestIndustryNews(t, db)

	userID := uint(5)
	image := &industry_news.IndustryNewsImage{
		IndustryNewsID: newsID,
		ImageURL:       "https://example.com/image1.png",
		Title:          "Test Image",
		IsActive:       true,
		UploadedBy:     &userID,
	}

	err := repo.Create(image)
	require.NoError(t, err)
	assert.NotZero(t, image.ID)
	assert.NotZero(t, image.CreatedAt)
	assert.NotZero(t, image.UpdatedAt)
}
```

- [ ] **Step 3: Run it to make sure it fails**

Run: `cd gmr-backend && go test ./internal/repository/... -run TestIndustryNewsImageRepository_Create -v`
Expected: FAIL — `NewIndustryNewsImageRepository` / `industry_news.IndustryNewsImage` undefined (this step should already pass once Steps 1–2 above are both in place; if run strictly test-first, it fails with "undefined: NewIndustryNewsImageRepository").

- [ ] **Step 4: Run the full test file to confirm it passes**

Add the remaining coverage mirroring `report_image_repository_test.go` (`TestReportImageRepository_FindByID`, `_FindByReportID`, `_FindActiveByReportID`, `_Update`, `_Delete`, `_CountByReportID`, `_CountActiveByReportID` — read that file for the exact bodies and translate `ReportID`→`IndustryNewsID`, `reportRepo`/`createTestReport`→`createTestIndustryNews`), then run:

Run: `cd gmr-backend && go test ./internal/repository/... -run TestIndustryNewsImageRepository -v`
Expected: PASS, all subtests green.

- [ ] **Step 5: Commit**

```bash
git add internal/repository/industry_news_image_repository.go internal/repository/industry_news_image_repository_test.go
git commit -m "Add IndustryNewsImageRepository with sqlite-backed tests"
```

---

## Task 5: `IndustryNewsService`

**Files:**
- Create: `gmr-backend/internal/service/industry_news_service.go`

**Interfaces:**
- Consumes: `repository.IndustryNewsRepository` (Task 3), `industry_news.*` types (Task 2), `internal/cache` package (`cache.GetOrSet`, `cache.DeletePattern`, `cache.Delete` — existing, used identically by `press_release_service.go`).
- Produces: `IndustryNewsService` interface + `NewIndustryNewsService(repo repository.IndustryNewsRepository) IndustryNewsService`, methods: `Create`, `GetAll`, `GetByCategorySlug`, `GetByID`, `GetBySlug`, `Update`, `Delete`, `SoftDelete`, `Restore`, `SubmitForReview`, `Publish`, `Unpublish`, `SchedulePublish`, `CancelScheduledPublish` — used by Task 7 (handler).

- [ ] **Step 1: Write the service**

```go
package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/gosimple/slug"
)

type IndustryNewsService interface {
	Create(req *industry_news.CreateIndustryNewsRequest) (*industry_news.IndustryNews, error)
	GetAll(query industry_news.GetIndustryNewsQuery) ([]industry_news.IndustryNews, int64, error)
	GetByCategorySlug(categorySlug string, page, limit int) ([]industry_news.IndustryNews, int64, error)
	GetByID(id uint) (*industry_news.IndustryNews, error)
	GetBySlug(slug string) (*industry_news.IndustryNews, error)
	Update(id uint, req *industry_news.UpdateIndustryNewsRequest) (*industry_news.IndustryNews, error)
	Delete(id uint) error
	SoftDelete(id uint) error
	Restore(id uint) error
	SubmitForReview(id uint) (*industry_news.IndustryNews, error)
	Publish(id uint) (*industry_news.IndustryNews, error)
	Unpublish(id uint) (*industry_news.IndustryNews, error)
	SchedulePublish(id uint, publishDate time.Time) (*industry_news.IndustryNews, error)
	CancelScheduledPublish(id uint) (*industry_news.IndustryNews, error)
}

type industryNewsService struct {
	repo repository.IndustryNewsRepository
}

func NewIndustryNewsService(repo repository.IndustryNewsRepository) IndustryNewsService {
	return &industryNewsService{repo: repo}
}

func (s *industryNewsService) Create(req *industry_news.CreateIndustryNewsRequest) (*industry_news.IndustryNews, error) {
	if req.Status != industry_news.StatusDraft && req.Status != industry_news.StatusReview && req.Status != industry_news.StatusPublished {
		return nil, fmt.Errorf("invalid status: must be 'draft', 'review', or 'published'")
	}

	newsSlug := slug.Make(req.Title)

	publishDate, err := time.Parse(time.RFC3339, req.PublishDate)
	if err != nil {
		return nil, fmt.Errorf("invalid publishDate format: must be ISO 8601 (RFC3339)")
	}

	n := &industry_news.IndustryNews{
		Title:       req.Title,
		Slug:        newsSlug,
		Excerpt:     req.Excerpt,
		Content:     req.Content,
		CategoryID:  req.CategoryID,
		Tags:        req.Tags,
		AuthorID:    req.AuthorID,
		Status:      req.Status,
		PublishDate: &publishDate,
		Location:    req.Location,
	}

	if req.Metadata != nil {
		n.Metadata = *req.Metadata
	}

	if err := s.repo.Create(n); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")

	return n, nil
}

func (s *industryNewsService) GetAll(query industry_news.GetIndustryNewsQuery) ([]industry_news.IndustryNews, int64, error) {
	shouldCache := query.Status == "" && query.CategoryID == "" && query.Tags == "" &&
		query.AuthorID == "" && query.Location == "" && query.Search == ""

	if shouldCache {
		cacheKey := fmt.Sprintf("industry_news:list:%d:%d", query.Page, query.Limit)

		type result struct {
			IndustryNews []industry_news.IndustryNews `json:"industryNews"`
			Total        int64                        `json:"total"`
		}

		var res result

		err := cache.GetOrSet(cacheKey, &res, 5*time.Minute, func() (interface{}, error) {
			items, total, err := s.repo.GetAll(query)
			if err != nil {
				return nil, err
			}
			return result{IndustryNews: items, Total: total}, nil
		})

		if err != nil {
			return nil, 0, err
		}

		return res.IndustryNews, res.Total, nil
	}

	return s.repo.GetAll(query)
}

func (s *industryNewsService) GetByCategorySlug(categorySlug string, page, limit int) ([]industry_news.IndustryNews, int64, error) {
	return s.repo.GetByCategorySlug(categorySlug, page, limit)
}

func (s *industryNewsService) GetByID(id uint) (*industry_news.IndustryNews, error) {
	cacheKey := fmt.Sprintf("industry_news:id:%d", id)

	var n industry_news.IndustryNews

	err := cache.GetOrSet(cacheKey, &n, 10*time.Minute, func() (interface{}, error) {
		return s.repo.GetByID(id)
	})

	if err != nil {
		return nil, err
	}

	return &n, nil
}

func (s *industryNewsService) GetBySlug(slugStr string) (*industry_news.IndustryNews, error) {
	cacheKey := fmt.Sprintf("industry_news:slug:%s", slugStr)

	var n industry_news.IndustryNews

	err := cache.GetOrSet(cacheKey, &n, 10*time.Minute, func() (interface{}, error) {
		return s.repo.GetBySlug(slugStr)
	})

	if err != nil {
		return nil, err
	}

	return &n, nil
}

func (s *industryNewsService) Update(id uint, req *industry_news.UpdateIndustryNewsRequest) (*industry_news.IndustryNews, error) {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = *req.Title
	}

	if req.Slug != nil {
		updates["slug"] = slug.Make(*req.Slug)
	} else if req.Title != nil {
		updates["slug"] = slug.Make(*req.Title)
	}

	if req.Excerpt != nil {
		updates["excerpt"] = *req.Excerpt
	}

	if req.Content != nil {
		updates["content"] = *req.Content
	}

	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}

	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}

	if req.AuthorID != nil {
		updates["author_id"] = *req.AuthorID
	}

	if req.Status != nil {
		if *req.Status != industry_news.StatusDraft && *req.Status != industry_news.StatusReview && *req.Status != industry_news.StatusPublished {
			return nil, fmt.Errorf("invalid status: must be 'draft', 'review', or 'published'")
		}
		updates["status"] = *req.Status
	}

	if req.PublishDate != nil {
		publishDate, err := time.Parse(time.RFC3339, *req.PublishDate)
		if err != nil {
			return nil, fmt.Errorf("invalid publishDate format: must be ISO 8601 (RFC3339)")
		}
		updates["publish_date"] = publishDate
	}

	if req.Location != nil {
		updates["location"] = *req.Location
	}

	if req.Metadata != nil {
		updates["metadata"] = *req.Metadata
	}

	if req.InternalLinks != nil {
		updates["internal_links"] = *req.InternalLinks
	}

	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}

func (s *industryNewsService) Delete(id uint) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(id); err != nil {
		return err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return nil
}

func (s *industryNewsService) SubmitForReview(id uint) (*industry_news.IndustryNews, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if existing.Status == industry_news.StatusReview {
		return nil, fmt.Errorf("industry news article is already in review")
	}
	if existing.Status == industry_news.StatusPublished {
		return nil, fmt.Errorf("cannot submit published industry news article for review")
	}

	if err := s.repo.SubmitForReview(id); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}

func (s *industryNewsService) Publish(id uint) (*industry_news.IndustryNews, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if existing.Status == industry_news.StatusPublished {
		return nil, fmt.Errorf("industry news article is already published")
	}

	if err := s.repo.Publish(id); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}

func (s *industryNewsService) Unpublish(id uint) (*industry_news.IndustryNews, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if existing.Status != industry_news.StatusPublished {
		return nil, fmt.Errorf("industry news article is not published")
	}

	if err := s.repo.Unpublish(id); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}

func (s *industryNewsService) SoftDelete(id uint) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.SoftDelete(id); err != nil {
		return err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return nil
}

func (s *industryNewsService) Restore(id uint) error {
	if err := s.repo.Restore(id); err != nil {
		return err
	}

	cache.DeletePattern("industry_news:*")

	return nil
}

func (s *industryNewsService) SchedulePublish(id uint, publishDate time.Time) (*industry_news.IndustryNews, error) {
	if publishDate.Before(time.Now()) {
		return nil, errors.New("publish date must be in the future")
	}

	n, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if n.Status == industry_news.StatusPublished {
		return nil, errors.New("cannot schedule already published industry news article")
	}

	if err := s.repo.SchedulePublish(id, publishDate); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}

func (s *industryNewsService) CancelScheduledPublish(id uint) (*industry_news.IndustryNews, error) {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CancelScheduledPublish(id); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}
```

- [ ] **Step 2: Verify it builds**

Run: `cd gmr-backend && go build ./...`
Expected: no compile errors. No test file — `press_release_service.go` (the direct template) has no `*_test.go` precedent in this codebase.

- [ ] **Step 3: Commit**

```bash
git add internal/service/industry_news_service.go
git commit -m "Add IndustryNewsService"
```

---

## Task 6: `IndustryNewsImageService` + service test

**Files:**
- Create: `gmr-backend/internal/service/industry_news_image_service.go`
- Test: `gmr-backend/internal/service/industry_news_image_service_test.go`

**Interfaces:**
- Consumes: `repository.IndustryNewsImageRepository` (Task 4), `repository.IndustryNewsRepository` (Task 3, only `GetByID` is used), `service.CloudflareImagesService` (existing — `Upload(file *multipart.FileHeader, metadata map[string]string) (string, error)`, `Delete(imageURL string) error`).
- Produces: `IndustryNewsImageService` interface + `NewIndustryNewsImageService(imageRepo repository.IndustryNewsImageRepository, newsRepo repository.IndustryNewsRepository, cloudflareService CloudflareImagesService) IndustryNewsImageService`, methods: `UploadImage`, `UpdateImageMetadata`, `DeleteImage`, `GetImagesByIndustryNews`, `GetImageByID` — used by Task 7 (handler).

- [ ] **Step 1: Write the service**

```go
package service

import (
	"fmt"
	"log"
	"mime/multipart"

	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
	"github.com/healthcare-market-research/backend/internal/repository"
)

type IndustryNewsImageService interface {
	UploadImage(industryNewsID uint, file *multipart.FileHeader, title string, uploadedBy uint) (*industry_news.IndustryNewsImage, error)
	UpdateImageMetadata(imageID uint, title *string, isActive *bool) (*industry_news.IndustryNewsImage, error)
	DeleteImage(imageID uint) error
	GetImagesByIndustryNews(industryNewsID uint, activeOnly bool) ([]industry_news.IndustryNewsImage, error)
	GetImageByID(imageID uint) (*industry_news.IndustryNewsImage, error)
}

type industryNewsImageService struct {
	industryNewsImageRepo repository.IndustryNewsImageRepository
	industryNewsRepo      repository.IndustryNewsRepository
	cloudflareService     CloudflareImagesService
}

func NewIndustryNewsImageService(
	industryNewsImageRepo repository.IndustryNewsImageRepository,
	industryNewsRepo repository.IndustryNewsRepository,
	cloudflareService CloudflareImagesService,
) IndustryNewsImageService {
	return &industryNewsImageService{
		industryNewsImageRepo: industryNewsImageRepo,
		industryNewsRepo:      industryNewsRepo,
		cloudflareService:     cloudflareService,
	}
}

func (s *industryNewsImageService) UploadImage(industryNewsID uint, file *multipart.FileHeader, title string, uploadedBy uint) (*industry_news.IndustryNewsImage, error) {
	_, err := s.industryNewsRepo.GetByID(industryNewsID)
	if err != nil {
		return nil, fmt.Errorf("industry news article not found: %w", err)
	}

	metadata := map[string]string{
		"industry_news_id": fmt.Sprintf("%d", industryNewsID),
		"type":             "industry_news_image",
		"uploaded_by":      fmt.Sprintf("%d", uploadedBy),
	}

	imageURL, err := s.cloudflareService.Upload(file, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	image := &industry_news.IndustryNewsImage{
		IndustryNewsID: industryNewsID,
		ImageURL:       imageURL,
		Title:          title,
		IsActive:       true,
		UploadedBy:     &uploadedBy,
	}

	if err := s.industryNewsImageRepo.Create(image); err != nil {
		if deleteErr := s.cloudflareService.Delete(imageURL); deleteErr != nil {
			log.Printf("Warning: Failed to rollback image upload for industry news %d: %v", industryNewsID, deleteErr)
		}
		return nil, fmt.Errorf("failed to create image record: %w", err)
	}

	return image, nil
}

func (s *industryNewsImageService) UpdateImageMetadata(imageID uint, title *string, isActive *bool) (*industry_news.IndustryNewsImage, error) {
	image, err := s.industryNewsImageRepo.FindByID(imageID)
	if err != nil {
		return nil, fmt.Errorf("image not found: %w", err)
	}

	if title != nil {
		image.Title = *title
	}

	if isActive != nil {
		image.IsActive = *isActive
	}

	if err := s.industryNewsImageRepo.Update(image); err != nil {
		return nil, fmt.Errorf("failed to update image: %w", err)
	}

	return image, nil
}

func (s *industryNewsImageService) DeleteImage(imageID uint) error {
	image, err := s.industryNewsImageRepo.FindByID(imageID)
	if err != nil {
		return fmt.Errorf("image not found: %w", err)
	}

	if err := s.cloudflareService.Delete(image.ImageURL); err != nil {
		log.Printf("Warning: Failed to delete image from Cloudflare for image %d: %v", imageID, err)
	}

	if err := s.industryNewsImageRepo.Delete(imageID); err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}

func (s *industryNewsImageService) GetImagesByIndustryNews(industryNewsID uint, activeOnly bool) ([]industry_news.IndustryNewsImage, error) {
	if activeOnly {
		return s.industryNewsImageRepo.FindActiveByIndustryNewsID(industryNewsID)
	}
	return s.industryNewsImageRepo.FindByIndustryNewsID(industryNewsID)
}

func (s *industryNewsImageService) GetImageByID(imageID uint) (*industry_news.IndustryNewsImage, error) {
	image, err := s.industryNewsImageRepo.FindByID(imageID)
	if err != nil {
		return nil, fmt.Errorf("image not found: %w", err)
	}
	return image, nil
}
```

- [ ] **Step 2: Write the failing test**

Mirror `report_image_service_test.go` exactly, with `report`→`industry_news`, `ReportID`→`IndustryNewsID`, `reportRepo`→`industryNewsRepo`, `mockReportRepository`→`mockIndustryNewsRepository` (implementing only the subset of `repository.IndustryNewsRepository` the service actually calls plus stub methods for the rest, same as `mockReportRepository` stubs unused `ReportRepository` methods), and `"report_image"` metadata type string → `"industry_news_image"`. Start with the upload-success case:

```go
package service

import (
	"testing"

	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
)

type mockIndustryNewsImageRepository struct {
	createFunc                       func(image *industry_news.IndustryNewsImage) error
	findByIDFunc                     func(id uint) (*industry_news.IndustryNewsImage, error)
	findByIndustryNewsIDFunc         func(industryNewsID uint) ([]industry_news.IndustryNewsImage, error)
	findActiveByIndustryNewsIDFunc   func(industryNewsID uint) ([]industry_news.IndustryNewsImage, error)
	updateFunc                       func(image *industry_news.IndustryNewsImage) error
	deleteFunc                       func(id uint) error
	countByIndustryNewsIDFunc        func(industryNewsID uint) (int64, error)
	countActiveByIndustryNewsIDFunc  func(industryNewsID uint) (int64, error)
}

func (m *mockIndustryNewsImageRepository) Create(image *industry_news.IndustryNewsImage) error {
	if m.createFunc != nil {
		return m.createFunc(image)
	}
	image.ID = 1
	return nil
}

func (m *mockIndustryNewsImageRepository) FindByID(id uint) (*industry_news.IndustryNewsImage, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	userID := uint(5)
	return &industry_news.IndustryNewsImage{
		ID:             id,
		IndustryNewsID: 1,
		ImageURL:       "https://imagedelivery.net/test/image-id/public",
		Title:          "Test Image",
		IsActive:       true,
		UploadedBy:     &userID,
	}, nil
}

func (m *mockIndustryNewsImageRepository) FindByIndustryNewsID(industryNewsID uint) ([]industry_news.IndustryNewsImage, error) {
	if m.findByIndustryNewsIDFunc != nil {
		return m.findByIndustryNewsIDFunc(industryNewsID)
	}
	return []industry_news.IndustryNewsImage{}, nil
}

func (m *mockIndustryNewsImageRepository) FindActiveByIndustryNewsID(industryNewsID uint) ([]industry_news.IndustryNewsImage, error) {
	if m.findActiveByIndustryNewsIDFunc != nil {
		return m.findActiveByIndustryNewsIDFunc(industryNewsID)
	}
	return []industry_news.IndustryNewsImage{}, nil
}

func (m *mockIndustryNewsImageRepository) Update(image *industry_news.IndustryNewsImage) error {
	if m.updateFunc != nil {
		return m.updateFunc(image)
	}
	return nil
}

func (m *mockIndustryNewsImageRepository) Delete(id uint) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *mockIndustryNewsImageRepository) CountByIndustryNewsID(industryNewsID uint) (int64, error) {
	if m.countByIndustryNewsIDFunc != nil {
		return m.countByIndustryNewsIDFunc(industryNewsID)
	}
	return 0, nil
}

func (m *mockIndustryNewsImageRepository) CountActiveByIndustryNewsID(industryNewsID uint) (int64, error) {
	if m.countActiveByIndustryNewsIDFunc != nil {
		return m.countActiveByIndustryNewsIDFunc(industryNewsID)
	}
	return 0, nil
}

type mockIndustryNewsRepositoryForImages struct {
	getByIDFunc func(id uint) (*industry_news.IndustryNews, error)
}

func (m *mockIndustryNewsRepositoryForImages) Create(n *industry_news.IndustryNews) error { return nil }
func (m *mockIndustryNewsRepositoryForImages) GetAll(query industry_news.GetIndustryNewsQuery) ([]industry_news.IndustryNews, int64, error) {
	return nil, 0, nil
}
func (m *mockIndustryNewsRepositoryForImages) GetByCategorySlug(categorySlug string, page, limit int) ([]industry_news.IndustryNews, int64, error) {
	return nil, 0, nil
}
func (m *mockIndustryNewsRepositoryForImages) GetByID(id uint) (*industry_news.IndustryNews, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &industry_news.IndustryNews{ID: id, Title: "Test Industry News"}, nil
}
func (m *mockIndustryNewsRepositoryForImages) GetBySlug(slug string) (*industry_news.IndustryNews, error) {
	return nil, nil
}
func (m *mockIndustryNewsRepositoryForImages) Update(id uint, updates map[string]interface{}) error {
	return nil
}
func (m *mockIndustryNewsRepositoryForImages) Delete(id uint) error       { return nil }
func (m *mockIndustryNewsRepositoryForImages) SoftDelete(id uint) error   { return nil }
func (m *mockIndustryNewsRepositoryForImages) Restore(id uint) error      { return nil }
func (m *mockIndustryNewsRepositoryForImages) SubmitForReview(id uint) error { return nil }
func (m *mockIndustryNewsRepositoryForImages) Publish(id uint) error      { return nil }
func (m *mockIndustryNewsRepositoryForImages) Unpublish(id uint) error    { return nil }
func (m *mockIndustryNewsRepositoryForImages) PublishScheduled(now time.Time) error { return nil }
func (m *mockIndustryNewsRepositoryForImages) SchedulePublish(id uint, publishDate time.Time) error {
	return nil
}
func (m *mockIndustryNewsRepositoryForImages) CancelScheduledPublish(id uint) error { return nil }

func TestIndustryNewsImageService_UploadImage_Success(t *testing.T) {
	uploadedImageURL := "https://imagedelivery.net/test/new-image-id/public"
	userID := uint(5)

	mockImageRepo := &mockIndustryNewsImageRepository{
		createFunc: func(image *industry_news.IndustryNewsImage) error {
			if image.IndustryNewsID != 1 {
				t.Errorf("Expected industryNewsID 1, got %d", image.IndustryNewsID)
			}
			if image.ImageURL != uploadedImageURL {
				t.Errorf("Expected imageURL %s, got %s", uploadedImageURL, image.ImageURL)
			}
			if *image.UploadedBy != userID {
				t.Errorf("Expected uploaded_by %d, got %d", userID, *image.UploadedBy)
			}
			image.ID = 1
			return nil
		},
	}

	mockNewsRepo := &mockIndustryNewsRepositoryForImages{}

	mockCloudflare := &mockCloudflareService{
		uploadFunc: func(file interface{}, metadata map[string]string) (string, error) {
			if metadata["industry_news_id"] != "1" {
				t.Errorf("Expected industry_news_id metadata '1', got '%s'", metadata["industry_news_id"])
			}
			if metadata["type"] != "industry_news_image" {
				t.Errorf("Expected type metadata 'industry_news_image', got '%s'", metadata["type"])
			}
			return uploadedImageURL, nil
		},
	}

	service := NewIndustryNewsImageService(mockImageRepo, mockNewsRepo, mockCloudflare)
	fileHeader := createTestFileHeader("chart.png", "fake image content")

	image, err := service.UploadImage(1, fileHeader, "Test Chart", userID)
	if err != nil {
		t.Errorf("UploadImage() error = %v, want nil", err)
	}

	if image.ID != 1 {
		t.Errorf("Expected image ID 1, got %d", image.ID)
	}
}
```

Note: `createTestFileHeader` and `mockCloudflareService` already exist in this package (defined for the report-image tests) — reuse them directly, do not redefine. Add the missing `"time"` import to the file.

- [ ] **Step 3: Run it to make sure it fails**

Run: `cd gmr-backend && go test ./internal/service/... -run TestIndustryNewsImageService_UploadImage_Success -v`
Expected: FAIL (compile error) until Step 1 is also in place — `NewIndustryNewsImageService` undefined.

- [ ] **Step 4: Add the remaining test coverage and confirm it passes**

Mirror the rest of `report_image_service_test.go` (`_UploadImage_WithoutTitle`, `_UploadImage_ReportNotFound`→`_IndustryNewsNotFound`, `_UploadImage_CloudflareUploadFails`, `_UploadImage_DatabaseCreateFails_Rollback`, `_UpdateImageMetadata_Success`, `_UpdateImageMetadata_PartialUpdate_TitleOnly`, `_UpdateImageMetadata_ImageNotFound`, `_DeleteImage_Success`, `_DeleteImage_ImageNotFound`, `_GetImagesByReport_AllImages`→`_GetImagesByIndustryNews_AllImages`, `_GetImagesByReport_ActiveOnly`→`_GetImagesByIndustryNews_ActiveOnly`, `_GetImageByID_Success`, `_GetImageByID_NotFound`) with the same renames, then run:

Run: `cd gmr-backend && go test ./internal/service/... -run TestIndustryNewsImageService -v`
Expected: PASS, all subtests green.

- [ ] **Step 5: Commit**

```bash
git add internal/service/industry_news_image_service.go internal/service/industry_news_image_service_test.go
git commit -m "Add IndustryNewsImageService with mock-backed tests"
```

---

## Task 7: `IndustryNewsHandler` + `IndustryNewsImageHandler`

**Files:**
- Create: `gmr-backend/internal/handler/industry_news_handler.go`
- Create: `gmr-backend/internal/handler/industry_news_image_handler.go`

**Interfaces:**
- Consumes: `service.IndustryNewsService` (Task 5), `service.IndustryNewsImageService` (Task 6), `service.AuditService` (existing), `audit.ActionIndustryNews*`/`audit.EntityIndustryNews` (Task 2), `pkg/response`, `pkg/validation.ValidateImageFile` (existing).
- Produces: `NewIndustryNewsHandler(service service.IndustryNewsService, auditService service.AuditService) *IndustryNewsHandler` with methods `GetByCategorySlug`, `Create`, `GetAll`, `GetByID`, `GetBySlug`, `Update`, `Delete`, `SubmitForReview`, `Publish`, `Unpublish`, `SoftDelete`, `Restore`, `SchedulePublish`, `CancelScheduledPublish`, `GetSitemap`; `NewIndustryNewsImageHandler(service service.IndustryNewsImageService) *IndustryNewsImageHandler` with methods `UploadImage`, `ListImages`, `GetByID`, `UpdateMetadata`, `DeleteImage` — used by Task 8 (route registration).

- [ ] **Step 1: Write `industry_news_handler.go`**

```go
package handler

import (
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
	"github.com/healthcare-market-research/backend/internal/middleware"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type IndustryNewsHandler struct {
	service      service.IndustryNewsService
	auditService service.AuditService
}

func NewIndustryNewsHandler(service service.IndustryNewsService, auditService service.AuditService) *IndustryNewsHandler {
	return &IndustryNewsHandler{service: service, auditService: auditService}
}

func (h *IndustryNewsHandler) GetByCategorySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	items, total, err := h.service.GetByCategorySlug(slug, page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to fetch industry news")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(industry_news.IndustryNewsListResponse{
		IndustryNews: items,
		Total:        total,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	})
}

func (h *IndustryNewsHandler) Create(c *fiber.Ctx) error {
	var req industry_news.CreateIndustryNewsRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	n, err := h.service.Create(&req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsCreate)
	auditEntry.EntityType = audit.EntityIndustryNews
	auditEntry.EntityID = &n.ID
	h.auditService.LogAsync(auditEntry)

	return c.Status(fiber.StatusCreated).JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

func (h *IndustryNewsHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var sortBy string
	if s := c.Query("sort_by", ""); s != "" {
		allowed := map[string]string{
			"publish_date_desc": "publish_date DESC NULLS LAST",
			"created_at_desc":   "created_at DESC",
			"updated_at_desc":   "updated_at DESC",
		}
		if mapped, ok := allowed[s]; ok {
			sortBy = mapped
		}
	}

	query := industry_news.GetIndustryNewsQuery{
		Status:       c.Query("status", ""),
		CategoryID:   c.Query("categoryId", ""),
		CategorySlug: c.Query("category", ""),
		Tags:         c.Query("tags", ""),
		AuthorID:     c.Query("authorId", ""),
		Location:     c.Query("location", ""),
		Search:       c.Query("search", ""),
		Deleted:      c.Query("deleted", ""),
		SortBy:       sortBy,
		Page:         page,
		Limit:        limit,
	}

	items, total, err := h.service.GetAll(query)
	if err != nil {
		return response.InternalError(c, "Failed to fetch industry news")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(industry_news.IndustryNewsListResponse{
		IndustryNews: items,
		Total:        total,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	})
}

func (h *IndustryNewsHandler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Industry news ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	n, err := h.service.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Industry news article not found")
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

func (h *IndustryNewsHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.BadRequest(c, "Industry news slug is required")
	}

	n, err := h.service.GetBySlug(slug)
	if err != nil {
		return response.NotFound(c, "Industry news article not found")
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

func (h *IndustryNewsHandler) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Industry news ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	var req industry_news.UpdateIndustryNewsRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	n, err := h.service.Update(uint(id), &req)
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.BadRequest(c, err.Error())
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsUpdate)
	auditEntry.EntityType = audit.EntityIndustryNews
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

func (h *IndustryNewsHandler) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Industry news ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.InternalError(c, "Failed to delete industry news article")
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsDelete)
	auditEntry.EntityType = audit.EntityIndustryNews
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *IndustryNewsHandler) SubmitForReview(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	n, err := h.service.SubmitForReview(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

func (h *IndustryNewsHandler) Publish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	n, err := h.service.Publish(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.BadRequest(c, err.Error())
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsPublish)
	auditEntry.EntityType = audit.EntityIndustryNews
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

func (h *IndustryNewsHandler) Unpublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	n, err := h.service.Unpublish(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

func (h *IndustryNewsHandler) SoftDelete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	if err := h.service.SoftDelete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.InternalError(c, "Failed to soft delete industry news article")
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsDelete)
	auditEntry.EntityType = audit.EntityIndustryNews
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, "Industry news article moved to trash successfully")
}

func (h *IndustryNewsHandler) Restore(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	if err := h.service.Restore(uint(id)); err != nil {
		return response.InternalError(c, "Failed to restore industry news article")
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsUpdate)
	auditEntry.EntityType = audit.EntityIndustryNews
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, "Industry news article restored successfully")
}

func (h *IndustryNewsHandler) SchedulePublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	var req struct {
		PublishDate string `json:"publishDate"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	publishDate, err := time.Parse(time.RFC3339, req.PublishDate)
	if err != nil {
		return response.BadRequest(c, "Invalid date format (use ISO 8601)")
	}

	n, err := h.service.SchedulePublish(uint(id), publishDate)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

func (h *IndustryNewsHandler) CancelScheduledPublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	n, err := h.service.CancelScheduledPublish(uint(id))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

func (h *IndustryNewsHandler) GetSitemap(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "1000"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 1000 {
		limit = 1000
	}

	items, total, err := h.service.GetAll(industry_news.GetIndustryNewsQuery{
		Status: "published",
		Page:   page,
		Limit:  limit,
		SortBy: "publish_date DESC NULLS LAST",
	})
	if err != nil {
		return response.InternalError(c, "Failed to fetch industry news for sitemap")
	}

	type sitemapItem struct {
		Slug        string     `json:"slug"`
		UpdatedAt   time.Time  `json:"updated_at"`
		PublishDate *time.Time `json:"publish_date,omitempty"`
	}

	items2 := make([]sitemapItem, len(items))
	for i, n := range items {
		items2[i] = sitemapItem{
			Slug:        n.Slug,
			UpdatedAt:   n.UpdatedAt,
			PublishDate: n.PublishDate,
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return response.SuccessWithMeta(c, items2, &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}
```

- [ ] **Step 2: Write `industry_news_image_handler.go`**

```go
package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
	"github.com/healthcare-market-research/backend/pkg/validation"
)

type IndustryNewsImageHandler struct {
	service service.IndustryNewsImageService
}

func NewIndustryNewsImageHandler(service service.IndustryNewsImageService) *IndustryNewsImageHandler {
	return &IndustryNewsImageHandler{service: service}
}

func (h *IndustryNewsImageHandler) UploadImage(c *fiber.Ctx) error {
	industryNewsID, err := strconv.ParseUint(c.Params("newsId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID")
	}

	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return response.Unauthorized(c, "User not authenticated")
	}

	file, err := c.FormFile("image")
	if err != nil {
		return response.BadRequest(c, "No image file provided")
	}

	if err := validation.ValidateImageFile(file); err != nil {
		return response.BadRequest(c, err.Error())
	}

	title := strings.TrimSpace(c.FormValue("title"))

	if title != "" && (len(title) < 2 || len(title) > 255) {
		return response.BadRequest(c, "Title must be between 2 and 255 characters")
	}

	image, err := h.service.UploadImage(uint(industryNewsID), file, title, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.InternalError(c, "Failed to upload image: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data:    image,
	})
}

func (h *IndustryNewsImageHandler) ListImages(c *fiber.Ctx) error {
	industryNewsID, err := strconv.ParseUint(c.Params("newsId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID")
	}

	activeOnly := c.Query("active") == "true"

	images, err := h.service.GetImagesByIndustryNews(uint(industryNewsID), activeOnly)
	if err != nil {
		return response.InternalError(c, "Failed to fetch images")
	}

	return response.Success(c, images)
}

func (h *IndustryNewsImageHandler) GetByID(c *fiber.Ctx) error {
	imageID, err := strconv.ParseUint(c.Params("imageId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	image, err := h.service.GetImageByID(uint(imageID))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Image not found")
		}
		return response.InternalError(c, "Failed to fetch image")
	}

	return response.Success(c, image)
}

func (h *IndustryNewsImageHandler) UpdateMetadata(c *fiber.Ctx) error {
	imageID, err := strconv.ParseUint(c.Params("imageId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	var req UpdateImageMetadataRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title != "" && (len(title) < 2 || len(title) > 255) {
			return response.BadRequest(c, "Title must be between 2 and 255 characters")
		}
		req.Title = &title
	}

	image, err := h.service.UpdateImageMetadata(uint(imageID), req.Title, req.IsActive)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Image not found")
		}
		return response.InternalError(c, "Failed to update image metadata: "+err.Error())
	}

	return response.Success(c, image)
}

func (h *IndustryNewsImageHandler) DeleteImage(c *fiber.Ctx) error {
	imageID, err := strconv.ParseUint(c.Params("imageId"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid image ID")
	}

	if err := h.service.DeleteImage(uint(imageID)); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return response.NotFound(c, "Image not found")
		}
		return response.InternalError(c, "Failed to delete image: "+err.Error())
	}

	return response.Success(c, fiber.Map{
		"message": "Image deleted successfully",
	})
}
```

Note: `UpdateImageMetadataRequest` is already defined in `report_image_handler.go` in this same `handler` package — reused here, not redeclared (Go allows this since both files are in package `handler`).

- [ ] **Step 3: Verify it builds**

Run: `cd gmr-backend && go build ./...`
Expected: no compile errors.

- [ ] **Step 4: Commit**

```bash
git add internal/handler/industry_news_handler.go internal/handler/industry_news_image_handler.go
git commit -m "Add IndustryNewsHandler and IndustryNewsImageHandler"
```

---

## Task 8: Wire into `main.go` and the scheduler

**Files:**
- Modify: `gmr-backend/cmd/api/main.go`
- Modify: `gmr-backend/internal/service/scheduler_service.go`

**Interfaces:**
- Consumes: everything from Tasks 2–7.
- Produces: live HTTP routes for the Industry News module; scheduled-publish sweep now includes `industry_news`.

- [ ] **Step 1: Update `scheduler_service.go` to include Industry News**

In `gmr-backend/internal/service/scheduler_service.go`, add `industryNewsRepo` to the struct, constructor, and sweep:

```go
type schedulerService struct {
	reportRepo       repository.ReportRepository
	blogRepo         repository.BlogRepository
	pressReleaseRepo repository.PressReleaseRepository
	industryNewsRepo repository.IndustryNewsRepository
	ticker           *time.Ticker
	stopCh           chan struct{}
}

func NewSchedulerService(
	reportRepo repository.ReportRepository,
	blogRepo repository.BlogRepository,
	pressReleaseRepo repository.PressReleaseRepository,
	industryNewsRepo repository.IndustryNewsRepository,
) SchedulerService {
	return &schedulerService{
		reportRepo:       reportRepo,
		blogRepo:         blogRepo,
		pressReleaseRepo: pressReleaseRepo,
		industryNewsRepo: industryNewsRepo,
		stopCh:           make(chan struct{}),
	}
}
```

And in `processScheduledPublishes`, after the `pressReleaseRepo.PublishScheduled(now)` block:

```go
	if err := s.industryNewsRepo.PublishScheduled(now); err != nil {
		logger.Error("Failed to publish scheduled industry news", "error", err)
	}
```

- [ ] **Step 2: Wire repositories, services, and handlers in `main.go`**

After the line `pressReleaseRepo := repository.NewPressReleaseRepository(db.DB)` (repository init block), add:

```go
	industryNewsRepo := repository.NewIndustryNewsRepository(db.DB)
	industryNewsImageRepo := repository.NewIndustryNewsImageRepository(db.DB)
```

After the line `pressReleaseService := service.NewPressReleaseService(pressReleaseRepo)` (service init block), add:

```go
	industryNewsService := service.NewIndustryNewsService(industryNewsRepo)
	industryNewsImageService := service.NewIndustryNewsImageService(industryNewsImageRepo, industryNewsRepo, cloudflareService)
```

Change the `schedulerService` construction to pass the new repo:

```go
	schedulerService := service.NewSchedulerService(reportRepo, blogRepo, pressReleaseRepo, industryNewsRepo)
```

After the line `pressReleaseHandler := handler.NewPressReleaseHandler(pressReleaseService, auditService)` (handler init block), add:

```go
	industryNewsHandler := handler.NewIndustryNewsHandler(industryNewsService, auditService)
	industryNewsImageHandler := handler.NewIndustryNewsImageHandler(industryNewsImageService)
```

- [ ] **Step 3: Register routes**

After the `// Press Release routes` block (after `v1.Patch("/press-releases/:id/cancel-schedule", ...)`), add:

```go
	// Industry News routes
	v1.Get("/industry-news", industryNewsHandler.GetAll)
	v1.Get("/industry-news/slug/:slug", industryNewsHandler.GetBySlug)
	v1.Get("/industry-news/images/:imageId", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsImageHandler.GetByID)
	v1.Patch("/industry-news/images/:imageId", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsImageHandler.UpdateMetadata)
	v1.Delete("/industry-news/images/:imageId", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsImageHandler.DeleteImage)
	v1.Get("/industry-news/:id", industryNewsHandler.GetByID)
	v1.Post("/industry-news", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsHandler.Create)
	v1.Put("/industry-news/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsHandler.Update)
	v1.Delete("/industry-news/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin"), industryNewsHandler.Delete)
	v1.Patch("/industry-news/:id/submit-review", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsHandler.SubmitForReview)
	v1.Patch("/industry-news/:id/publish", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsHandler.Publish)
	v1.Patch("/industry-news/:id/unpublish", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsHandler.Unpublish)
	v1.Patch("/industry-news/:id/soft-delete", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsHandler.SoftDelete)
	v1.Patch("/industry-news/:id/restore", middleware.RequireAuth(authService), middleware.RequireRole("admin"), industryNewsHandler.Restore)
	v1.Patch("/industry-news/:id/schedule", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsHandler.SchedulePublish)
	v1.Patch("/industry-news/:id/cancel-schedule", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsHandler.CancelScheduledPublish)
	v1.Post("/industry-news/:newsId/images", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsImageHandler.UploadImage)
	v1.Get("/industry-news/:newsId/images", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), industryNewsImageHandler.ListImages)
```

(The two `images/:imageId` routes are registered before `GET /industry-news/:id` deliberately, mirroring how `press-releases/slug/:slug` is ordered relative to `press-releases/:id` elsewhere in this file — Fiber's underlying radix-tree router prioritizes static path segments over `:id` params regardless of registration order, so this ordering is for readability, not correctness, but keep it for consistency with the rest of the file.)

In the `// Category routes` block, after `v1.Get("/categories/:slug/press-releases", pressReleaseHandler.GetByCategorySlug)`, add:

```go
	v1.Get("/categories/:slug/industry-news", industryNewsHandler.GetByCategorySlug)
```

In the `// Sitemap routes` block, after `v1.Get("/sitemap/press-releases", pressReleaseHandler.GetSitemap)`, add:

```go
	v1.Get("/sitemap/industry-news", industryNewsHandler.GetSitemap)
```

- [ ] **Step 4: Verify it builds and starts**

Run: `cd gmr-backend && go build ./...`
Expected: no compile errors.

If a local dev Postgres is running (see Task 1 Step 2 for the migration), start the server and smoke-test manually:

```bash
go run cmd/api/main.go
```

In another terminal:

```bash
curl -s http://localhost:8080/api/v1/industry-news | head -c 300
```

Expected: `200 OK` with an empty `industryNews: []` list (or existing seeded data), not a 404/500.

- [ ] **Step 5: Commit**

```bash
git add cmd/api/main.go internal/service/scheduler_service.go
git commit -m "Wire Industry News routes, services, and scheduler into main.go"
```

---

## Task 9: Admin — types, config, validation schema

**Files:**
- Create: `gmr-admin/lib/types/industry-news.ts`
- Create: `gmr-admin/lib/config/industry-news.ts`
- Create: `gmr-admin/lib/validation/industry-news-schema.ts`

**Interfaces:**
- Consumes: `UserReference`, `ReportAuthor`, `InternalLinkEntry` from `gmr-admin/lib/types/reports.ts` (existing, reused by `press-releases.ts` the same way).
- Produces: `IndustryNewsStatus`, `IndustryNewsMetadata`, `ApiIndustryNews`, `IndustryNews`, `IndustryNewsFilters`, `IndustryNewsResponse`, `IndustryNewsesResponse` (list), `CreateIndustryNewsData`, `UpdateIndustryNewsData`, `IndustryNewsFormData` types; `INDUSTRY_NEWS_STATUS_CONFIG`, `INDUSTRY_NEWS_WORKFLOW_TRANSITIONS`, length-limit constants; `industryNewsFormSchema` — used by Tasks 10–14.

- [ ] **Step 1: Write `lib/types/industry-news.ts`**

```typescript
// Re-use UserReference and ReportAuthor from reports
import type { UserReference, ReportAuthor, InternalLinkEntry } from './reports';
export type { InternalLinkEntry };

// Industry News status enum with workflow states
export type IndustryNewsStatus = 'draft' | 'review' | 'published';

// SEO metadata for industry news (matches API spec)
export interface IndustryNewsMetadata {
  metaTitle?: string;
  metaDescription?: string;
  keywords?: string[];
}

// Version history item
export interface IndustryNewsVersion {
  id: string;
  versionNumber: number;
  summary: string;
  createdAt: string;
  author: UserReference;
  content: string;
  title: string;
  excerpt: string;
}

// API Industry News interface (matches backend response)
export interface ApiIndustryNews {
  id: number;
  title: string;
  slug: string;
  excerpt: string;
  content: string; // HTML from rich text editor
  authorId: number;
  author?: ReportAuthor; // Author details populated from API
  categoryId: number; // Single category ID from API
  category?: {
    id: number;
    name: string;
    slug: string;
    description?: string;
    image_url?: string;
    is_active: boolean;
    created_at: string;
    updated_at: string;
  }; // Category details from API
  tags: string; // Comma-separated tags from API
  status: IndustryNewsStatus;
  publishDate: string;
  scheduledPublishEnabled?: boolean;
  location?: string;
  metadata: IndustryNewsMetadata;
  createdAt: string;
  updatedAt: string;
  reviewedBy?: number;
  reviewedAt?: string;
  deletedAt?: string;
}

// Main Industry News interface (matches API response)
export interface IndustryNews {
  id: number;
  title: string;
  slug: string;
  excerpt: string;
  content: string; // HTML from rich text editor
  authorId: number;
  author?: ReportAuthor; // Author details populated from API
  categoryId: number; // Single category ID from API
  category?: {
    id: number;
    name: string;
    slug: string;
    description?: string;
    image_url?: string;
    is_active: boolean;
    created_at: string;
    updated_at: string;
  }; // Category details from API
  tags: string; // Comma-separated tags from API
  status: IndustryNewsStatus;
  publishDate: string;
  scheduledPublishEnabled?: boolean;
  location?: string;
  metadata: IndustryNewsMetadata;
  internalLinks?: InternalLinkEntry[];
  createdAt: string;
  updatedAt: string;
  reviewedBy?: number;
  reviewedAt?: string;
  deletedAt?: string;
}

// List filters (matches API query parameters)
export interface IndustryNewsFilters {
  status?: IndustryNewsStatus;
  categoryId?: number;
  tags?: string;
  authorId?: number;
  location?: string;
  search?: string;
  page?: number;
  limit?: number;
}

// API response types
export interface IndustryNewsesResponse {
  industryNews: IndustryNews[];
  total: number;
  page: number;
  limit: number;
  totalPages: number;
}

export interface IndustryNewsResponse {
  industryNews: IndustryNews;
}

// Form data for create (matches API CreateIndustryNewsRequest)
export interface CreateIndustryNewsData {
  title: string;
  slug: string;
  excerpt: string;
  content: string;
  categoryId: number;
  tags: string;
  authorId: number;
  status: IndustryNewsStatus;
  publishDate: string;
  scheduledPublishEnabled?: boolean;
  location?: string;
  metadata?: IndustryNewsMetadata;
}

// Form data for update (matches API UpdateIndustryNewsRequest)
export interface UpdateIndustryNewsData {
  title?: string;
  slug?: string;
  excerpt?: string;
  content?: string;
  categoryId?: number;
  tags?: string;
  authorId?: number;
  status?: IndustryNewsStatus;
  publishDate?: string;
  scheduledPublishEnabled?: boolean;
  location?: string;
  metadata?: IndustryNewsMetadata;
  internalLinks?: InternalLinkEntry[];
}

// UI Form data (what the form uses before sending to API)
export interface IndustryNewsFormData {
  title: string;
  slug: string;
  excerpt: string;
  content: string;
  categoryId: number;
  tags?: string;
  authorId: number;
  status: IndustryNewsStatus;
  publishDate: string;
  scheduledPublishEnabled?: boolean;
  location?: string;
  metadata?: IndustryNewsMetadata;
  internalLinks?: InternalLinkEntry[];
}
```

- [ ] **Step 2: Write `lib/config/industry-news.ts`**

```typescript
// Import statistic categories for reuse (same categories table as PR/Statistics)
import { STATISTIC_CATEGORIES } from './statistics';

export { STATISTIC_CATEGORIES as INDUSTRY_NEWS_CATEGORIES };

// Popular tags for quick selection
export const POPULAR_INDUSTRY_NEWS_TAGS = [
  'healthcare',
  'market research',
  'pharmaceuticals',
  'medical devices',
  'biotechnology',
  'digital health',
  'AI in healthcare',
  'telemedicine',
  'clinical trials',
  'FDA',
  'innovation',
  'startups',
] as const;

// Industry News status labels with descriptions
export const INDUSTRY_NEWS_STATUS_CONFIG = {
  draft: {
    label: 'Draft',
    description: 'Not visible to public. Only you can see this.',
    color: 'secondary',
  },
  review: {
    label: 'In Review',
    description: 'Submitted for review. Awaiting approval.',
    color: 'warning',
  },
  published: {
    label: 'Published',
    description: 'Visible to public on the website.',
    color: 'success',
  },
} as const;

// Workflow transitions
export const INDUSTRY_NEWS_WORKFLOW_TRANSITIONS = {
  draft: ['review', 'published'],
  review: ['draft', 'published'],
  published: ['draft'],
} as const;

// Reading time calculation
export const INDUSTRY_NEWS_WORDS_PER_MINUTE = 200;

// Pagination defaults
export const INDUSTRY_NEWS_PER_PAGE = 10;
export const MAX_INDUSTRY_NEWS_PER_PAGE = 50;

// Excerpt length limits (from API spec)
export const INDUSTRY_NEWS_EXCERPT_MIN_LENGTH = 50;
export const INDUSTRY_NEWS_EXCERPT_MAX_LENGTH = 500;

// Title length limits (from API spec)
export const INDUSTRY_NEWS_TITLE_MIN_LENGTH = 10;
export const INDUSTRY_NEWS_TITLE_MAX_LENGTH = 200;

// Content length limits (from API spec)
export const INDUSTRY_NEWS_CONTENT_MIN_LENGTH = 100;

// Default author name pre-selected for new Industry News articles
export const INDUSTRY_NEWS_DEFAULT_AUTHOR_NAME = 'Globe Market Research';
```

- [ ] **Step 3: Write `lib/validation/industry-news-schema.ts`**

```typescript
import { z } from 'zod';
import {
  INDUSTRY_NEWS_TITLE_MIN_LENGTH,
  INDUSTRY_NEWS_TITLE_MAX_LENGTH,
  INDUSTRY_NEWS_EXCERPT_MIN_LENGTH,
  INDUSTRY_NEWS_EXCERPT_MAX_LENGTH,
  INDUSTRY_NEWS_CONTENT_MIN_LENGTH,
} from '@/lib/config/industry-news';

export const industryNewsFormSchema = z.object({
  title: z
    .string()
    .min(
      INDUSTRY_NEWS_TITLE_MIN_LENGTH,
      `Title must be at least ${INDUSTRY_NEWS_TITLE_MIN_LENGTH} characters`
    )
    .max(
      INDUSTRY_NEWS_TITLE_MAX_LENGTH,
      `Title must be at most ${INDUSTRY_NEWS_TITLE_MAX_LENGTH} characters`
    ),
  slug: z
    .string()
    .min(1, 'Slug is required')
    .regex(
      /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
      'Slug must be lowercase letters, numbers, and hyphens only'
    ),
  excerpt: z
    .string()
    .min(
      INDUSTRY_NEWS_EXCERPT_MIN_LENGTH,
      `Excerpt must be at least ${INDUSTRY_NEWS_EXCERPT_MIN_LENGTH} characters`
    )
    .max(
      INDUSTRY_NEWS_EXCERPT_MAX_LENGTH,
      `Excerpt must be at most ${INDUSTRY_NEWS_EXCERPT_MAX_LENGTH} characters`
    ),
  content: z
    .string()
    .min(
      INDUSTRY_NEWS_CONTENT_MIN_LENGTH,
      `Content must be at least ${INDUSTRY_NEWS_CONTENT_MIN_LENGTH} characters`
    ),
  categoryId: z.number().positive('Category is required'),
  tags: z.string().default(''),
  authorId: z.number().positive('Author is required'),
  status: z.enum(['draft', 'review', 'published']),
  publishDate: z.string(),
  location: z.string().optional(),
  metadata: z
    .object({
      metaTitle: z.string().optional(),
      metaDescription: z.string().optional(),
      keywords: z.array(z.string()).optional().default([]),
    })
    .default(() => ({ metaTitle: '', metaDescription: '', keywords: [] })),
});
```

- [ ] **Step 4: Verify types compile**

Run: `cd gmr-admin && npm run type-check`
Expected: no new errors (these three files are not yet imported anywhere, so this mainly confirms they're self-consistent TypeScript).

- [ ] **Step 5: Commit**

```bash
git add lib/types/industry-news.ts lib/config/industry-news.ts lib/validation/industry-news-schema.ts
git commit -m "Add Industry News types, config, and validation schema"
```

---

## Task 10: Admin — API client

**Files:**
- Create: `gmr-admin/lib/api/industry-news.ts`
- Create: `gmr-admin/lib/api/industry-news-images.ts`

**Interfaces:**
- Consumes: `apiClient` from `./client` (existing), types from Task 9.
- Produces: `fetchIndustryNewsList`, `fetchIndustryNewsById`, `createIndustryNews`, `updateIndustryNews`, `deleteIndustryNews`, `submitForReview`, `publishIndustryNews`, `unpublishIndustryNews`, `softDeleteIndustryNews`, `restoreIndustryNews`, `schedulePublish`, `cancelScheduledPublish`, `fetchTrashedIndustryNews`; `fetchIndustryNewsImages`, `uploadIndustryNewsImage`, `updateIndustryNewsImageMetadata`, `deleteIndustryNewsImage` — used by Task 11 (hooks) and Task 12 (form/images manager).

- [ ] **Step 1: Write `lib/api/industry-news.ts`**

```typescript
import { apiClient } from './client';
import type {
  IndustryNewsesResponse,
  IndustryNewsFilters,
  IndustryNewsResponse,
  IndustryNewsFormData,
  CreateIndustryNewsData,
  UpdateIndustryNewsData,
  ApiIndustryNews,
  IndustryNews,
} from '@/lib/types/industry-news';

function transformFormDataToApi(
  data: IndustryNewsFormData
): CreateIndustryNewsData | UpdateIndustryNewsData {
  return {
    ...data,
  };
}

async function transformApiIndustryNewsToIndustryNews(
  apiIndustryNews: ApiIndustryNews
): Promise<IndustryNews> {
  return {
    ...apiIndustryNews,
  };
}

export async function fetchIndustryNewsList(
  filters?: IndustryNewsFilters
): Promise<IndustryNewsesResponse> {
  return apiClient.get<IndustryNewsesResponse>('/v1/industry-news', {
    params: filters as Record<string, unknown>,
  });
}

export async function fetchIndustryNewsById(id: number): Promise<IndustryNewsResponse> {
  return apiClient.get<IndustryNewsResponse>(`/v1/industry-news/${id}`);
}

export async function createIndustryNews(
  data: IndustryNewsFormData
): Promise<IndustryNewsResponse> {
  const apiData = transformFormDataToApi(data) as CreateIndustryNewsData;
  return apiClient.post<IndustryNewsResponse>('/v1/industry-news', apiData);
}

export async function updateIndustryNews(
  id: number,
  data: Partial<IndustryNewsFormData>
): Promise<IndustryNewsResponse> {
  const apiData: UpdateIndustryNewsData = {
    ...data,
  };
  return apiClient.put<IndustryNewsResponse>(`/v1/industry-news/${id}`, apiData);
}

export async function deleteIndustryNews(id: number): Promise<void> {
  return apiClient.delete(`/v1/industry-news/${id}`);
}

export async function submitForReview(id: number): Promise<IndustryNewsResponse> {
  return apiClient.patch<IndustryNewsResponse>(`/v1/industry-news/${id}/submit-review`);
}

export async function publishIndustryNews(id: number): Promise<IndustryNewsResponse> {
  return apiClient.patch<IndustryNewsResponse>(`/v1/industry-news/${id}/publish`);
}

export async function unpublishIndustryNews(id: number): Promise<IndustryNewsResponse> {
  return apiClient.patch<IndustryNewsResponse>(`/v1/industry-news/${id}/unpublish`);
}

export async function softDeleteIndustryNews(id: number): Promise<void> {
  return apiClient.patch(`/v1/industry-news/${id}/soft-delete`);
}

export async function restoreIndustryNews(id: number): Promise<void> {
  return apiClient.patch(`/v1/industry-news/${id}/restore`);
}

export async function schedulePublish(
  id: string | number,
  publishDate: Date
): Promise<IndustryNewsResponse> {
  const response = await apiClient.patch<{ industryNews: ApiIndustryNews }>(
    `/v1/industry-news/${id}/schedule`,
    { publishDate: publishDate.toISOString() }
  );
  const industryNews = await transformApiIndustryNewsToIndustryNews(response.industryNews);
  return { industryNews };
}

export async function cancelScheduledPublish(id: string | number): Promise<IndustryNewsResponse> {
  const response = await apiClient.patch<{ industryNews: ApiIndustryNews }>(
    `/v1/industry-news/${id}/cancel-schedule`
  );
  const industryNews = await transformApiIndustryNewsToIndustryNews(response.industryNews);
  return { industryNews };
}

export async function fetchTrashedIndustryNews(
  filters?: IndustryNewsFilters
): Promise<IndustryNewsesResponse> {
  return apiClient.get<IndustryNewsesResponse>('/v1/industry-news', {
    params: { ...filters, deleted: 'true' } as Record<string, unknown>,
  });
}
```

- [ ] **Step 2: Write `lib/api/industry-news-images.ts`**

```typescript
/**
 * Industry News Images API
 * Handles image uploads, retrieval, and management for industry news articles
 */

import { apiClient } from './client';
import type { ApiResponse } from '@/lib/types/reports';
import type { ReportImage, ReportImageMetadataUpdate } from '@/lib/types/reports';

interface ApiIndustryNewsImage {
  id: number;
  industry_news_id: number;
  image_url: string;
  title?: string;
  is_active: boolean;
  uploaded_by?: number;
  created_at: string;
  updated_at: string;
}

interface IndustryNewsImageApiResponse extends ApiResponse<ApiIndustryNewsImage> {
  data: ApiIndustryNewsImage;
}

interface IndustryNewsImagesApiResponse extends ApiResponse<ApiIndustryNewsImage[]> {
  data: ApiIndustryNewsImage[];
}

function transformIndustryNewsImage(apiImage: ApiIndustryNewsImage): ReportImage {
  return {
    id: apiImage.id,
    reportId: apiImage.industry_news_id,
    imageUrl: apiImage.image_url,
    title: apiImage.title,
    isActive: apiImage.is_active,
    createdAt: apiImage.created_at,
    updatedAt: apiImage.updated_at,
  };
}

export async function fetchIndustryNewsImages(
  industryNewsId: number | string,
  active?: boolean
): Promise<ReportImage[]> {
  const params: Record<string, unknown> = {};
  if (active !== undefined) {
    params.active = active;
  }

  const response = await apiClient.get<IndustryNewsImagesApiResponse>(
    `/v1/industry-news/${industryNewsId}/images`,
    { params }
  );

  if (!response.success) {
    throw new Error(response.error || 'Failed to fetch industry news images');
  }

  return (response.data || []).map(transformIndustryNewsImage);
}

export async function uploadIndustryNewsImage(
  industryNewsId: number | string,
  file: File,
  title?: string
): Promise<ReportImage> {
  const formData = new FormData();
  formData.append('image', file);
  if (title) {
    formData.append('title', title);
  }

  const response = await apiClient.upload<IndustryNewsImageApiResponse>(
    `/v1/industry-news/${industryNewsId}/images`,
    formData
  );

  if (!response.success || !response.data) {
    throw new Error(response.error || 'Failed to upload image');
  }

  return transformIndustryNewsImage(response.data);
}

export async function updateIndustryNewsImageMetadata(
  imageId: number | string,
  metadata: ReportImageMetadataUpdate
): Promise<ReportImage> {
  const apiMetadata: Record<string, unknown> = {};
  if (metadata.title !== undefined) {
    apiMetadata.title = metadata.title;
  }
  if (metadata.isActive !== undefined) {
    apiMetadata.is_active = metadata.isActive;
  }

  const response = await apiClient.patch<IndustryNewsImageApiResponse>(
    `/v1/industry-news/images/${imageId}`,
    apiMetadata
  );

  if (!response.success || !response.data) {
    throw new Error(response.error || 'Failed to update image metadata');
  }

  return transformIndustryNewsImage(response.data);
}

export async function deleteIndustryNewsImage(imageId: number | string): Promise<void> {
  const response = await apiClient.delete<ApiResponse<{ message: string }>>(
    `/v1/industry-news/images/${imageId}`
  );

  if (!response.success) {
    throw new Error(response.error || 'Failed to delete image');
  }
}
```

Note: this reuses the `ReportImage` / `ReportImageMetadataUpdate` types from `lib/types/reports.ts` rather than declaring a parallel `IndustryNewsImage` type — the shape is identical (`id`, `reportId`→repurposed as the parent content ID, `imageUrl`, `title`, `isActive`, `createdAt`, `updatedAt`) and this keeps `IndustryNewsImagesManager` (Task 12) a near-verbatim copy of `ReportImagesManager`.

- [ ] **Step 3: Verify types compile**

Run: `cd gmr-admin && npm run type-check`
Expected: no new errors.

- [ ] **Step 4: Commit**

```bash
git add lib/api/industry-news.ts lib/api/industry-news-images.ts
git commit -m "Add Industry News admin API client"
```

---

## Task 11: Admin — hooks

**Files:**
- Create: `gmr-admin/hooks/use-industry-news.ts`
- Create: `gmr-admin/hooks/use-industry-news-list.ts`

**Interfaces:**
- Consumes: `lib/api/industry-news.ts` (Task 10), `lib/types/industry-news.ts` (Task 9).
- Produces: `useIndustryNews()` (single-item CRUD + workflow), `useIndustryNewsList(initialFilters?)` (list/filter/soft-delete/restore) — used by Task 14 (pages).

- [ ] **Step 1: Write `hooks/use-industry-news.ts`**

```typescript
'use client';

import { useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import type { IndustryNews, IndustryNewsFormData } from '@/lib/types/industry-news';
import {
  fetchIndustryNewsById,
  createIndustryNews,
  updateIndustryNews,
  deleteIndustryNews,
  submitForReview,
  publishIndustryNews,
  unpublishIndustryNews,
  schedulePublish,
  cancelScheduledPublish,
} from '@/lib/api/industry-news';

interface UseIndustryNewsReturn {
  industryNews: IndustryNews | null;
  isLoading: boolean;
  isSaving: boolean;
  error: string | null;
  fetchIndustryNewsItem: (id: number) => Promise<void>;
  saveIndustryNews: (id: number | null, data: IndustryNewsFormData) => Promise<IndustryNews | null>;
  removeIndustryNews: (id: number) => Promise<void>;
  submitIndustryNewsForReview: (id: number) => Promise<IndustryNews | null>;
  publishIndustryNewsPost: (id: number) => Promise<IndustryNews | null>;
  unpublishIndustryNewsPost: (id: number) => Promise<IndustryNews | null>;
  scheduleIndustryNewsPublish: (id: string, publishDate: Date) => Promise<IndustryNews | null>;
  cancelIndustryNewsSchedule: (id: string) => Promise<IndustryNews | null>;
}

export function useIndustryNews(): UseIndustryNewsReturn {
  const [industryNews, setIndustryNews] = useState<IndustryNews | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const router = useRouter();

  const fetchIndustryNewsItem = useCallback(async (id: number) => {
    try {
      setIsLoading(true);
      setError(null);
      const { industryNews } = await fetchIndustryNewsById(id);
      setIndustryNews(industryNews);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load industry news';
      setError(errorMessage);
      toast.error(errorMessage);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const saveIndustryNews = useCallback(
    async (id: number | null, data: IndustryNewsFormData): Promise<IndustryNews | null> => {
      try {
        setIsSaving(true);
        setError(null);

        const response = id ? await updateIndustryNews(id, data) : await createIndustryNews(data);

        setIndustryNews(response.industryNews);
        toast.success(id ? 'Industry news updated successfully' : 'Industry news created successfully');

        if (!id) {
          router.push(`/industry-news/${response.industryNews.id}`);
        }

        return response.industryNews;
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to save industry news';
        setError(errorMessage);
        toast.error(errorMessage);
        return null;
      } finally {
        setIsSaving(false);
      }
    },
    [router]
  );

  const removeIndustryNews = useCallback(
    async (id: number) => {
      try {
        setIsSaving(true);
        await deleteIndustryNews(id);
        toast.success('Industry news deleted successfully');
        router.push('/industry-news');
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to delete industry news';
        toast.error(errorMessage);
      } finally {
        setIsSaving(false);
      }
    },
    [router]
  );

  const submitIndustryNewsForReview = useCallback(
    async (id: number): Promise<IndustryNews | null> => {
      try {
        setIsSaving(true);
        setError(null);
        const response = await submitForReview(id);
        setIndustryNews(response.industryNews);
        toast.success('Industry news submitted for review');
        return response.industryNews;
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to submit for review';
        setError(errorMessage);
        toast.error(errorMessage);
        return null;
      } finally {
        setIsSaving(false);
      }
    },
    []
  );

  const publishIndustryNewsPost = useCallback(async (id: number): Promise<IndustryNews | null> => {
    try {
      setIsSaving(true);
      setError(null);
      const response = await publishIndustryNews(id);
      setIndustryNews(response.industryNews);
      toast.success('Industry news published successfully');
      return response.industryNews;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to publish industry news';
      setError(errorMessage);
      toast.error(errorMessage);
      return null;
    } finally {
      setIsSaving(false);
    }
  }, []);

  const unpublishIndustryNewsPost = useCallback(
    async (id: number): Promise<IndustryNews | null> => {
      try {
        setIsSaving(true);
        setError(null);
        const response = await unpublishIndustryNews(id);
        setIndustryNews(response.industryNews);
        toast.success('Industry news unpublished');
        return response.industryNews;
      } catch (err) {
        const errorMessage =
          err instanceof Error ? err.message : 'Failed to unpublish industry news';
        setError(errorMessage);
        toast.error(errorMessage);
        return null;
      } finally {
        setIsSaving(false);
      }
    },
    []
  );

  const scheduleIndustryNewsPublish = useCallback(
    async (id: string, publishDate: Date): Promise<IndustryNews | null> => {
      try {
        setIsSaving(true);
        setError(null);
        const response = await schedulePublish(id, publishDate);
        setIndustryNews(response.industryNews);
        toast.success('Industry news scheduled for publishing');
        return response.industryNews;
      } catch (err) {
        const errorMessage =
          err instanceof Error ? err.message : 'Failed to schedule industry news';
        setError(errorMessage);
        toast.error(errorMessage);
        return null;
      } finally {
        setIsSaving(false);
      }
    },
    []
  );

  const cancelIndustryNewsSchedule = useCallback(
    async (id: string): Promise<IndustryNews | null> => {
      try {
        setIsSaving(true);
        setError(null);
        const response = await cancelScheduledPublish(id);
        setIndustryNews(response.industryNews);
        toast.success('Scheduled publishing cancelled');
        return response.industryNews;
      } catch (err) {
        const errorMessage =
          err instanceof Error ? err.message : 'Failed to cancel scheduled publishing';
        setError(errorMessage);
        toast.error(errorMessage);
        return null;
      } finally {
        setIsSaving(false);
      }
    },
    []
  );

  return {
    industryNews,
    isLoading,
    isSaving,
    error,
    fetchIndustryNewsItem,
    saveIndustryNews,
    removeIndustryNews,
    submitIndustryNewsForReview,
    publishIndustryNewsPost,
    unpublishIndustryNewsPost,
    scheduleIndustryNewsPublish,
    cancelIndustryNewsSchedule,
  };
}
```

- [ ] **Step 2: Write `hooks/use-industry-news-list.ts`**

```typescript
'use client';

import { useState, useEffect, useCallback } from 'react';
import { toast } from 'sonner';
import type { IndustryNews, IndustryNewsFilters } from '@/lib/types/industry-news';
import {
  fetchIndustryNewsList,
  softDeleteIndustryNews,
  restoreIndustryNews,
} from '@/lib/api/industry-news';

interface UseIndustryNewsListReturn {
  industryNewsItems: IndustryNews[];
  total: number;
  totalPages: number;
  currentPage: number;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
  setFilters: (filters: IndustryNewsFilters) => void;
  softDelete: (id: number) => Promise<void>;
  restore: (id: number) => Promise<void>;
}

export function useIndustryNewsList(initialFilters?: IndustryNewsFilters): UseIndustryNewsListReturn {
  const [industryNewsItems, setIndustryNewsItems] = useState<IndustryNews[]>([]);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [currentPage, setCurrentPage] = useState(initialFilters?.page || 1);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFiltersState] = useState<IndustryNewsFilters>(initialFilters || {});

  const fetchData = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);

      const data = await fetchIndustryNewsList(filters);

      setIndustryNewsItems(data.industryNews);
      setTotal(data.total);
      setTotalPages(data.totalPages);
      setCurrentPage(data.page);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load industry news';
      setError(errorMessage);
      toast.error(errorMessage);
    } finally {
      setIsLoading(false);
    }
  }, [filters]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const setFilters = useCallback((newFilters: IndustryNewsFilters) => {
    setFiltersState(prev => ({ ...prev, ...newFilters }));
  }, []);

  const handleSoftDelete = useCallback(
    async (id: number) => {
      try {
        await softDeleteIndustryNews(id);
        await fetchData();
      } catch (err) {
        const errorMessage =
          err instanceof Error ? err.message : 'Failed to move industry news to trash';
        toast.error(errorMessage);
        throw err;
      }
    },
    [fetchData]
  );

  const handleRestore = useCallback(
    async (id: number) => {
      try {
        await restoreIndustryNews(id);
        await fetchData();
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to restore industry news';
        toast.error(errorMessage);
        throw err;
      }
    },
    [fetchData]
  );

  return {
    industryNewsItems,
    total,
    totalPages,
    currentPage,
    isLoading,
    error,
    refetch: fetchData,
    setFilters,
    softDelete: handleSoftDelete,
    restore: handleRestore,
  };
}
```

- [ ] **Step 3: Verify types compile**

Run: `cd gmr-admin && npm run type-check`
Expected: no new errors.

- [ ] **Step 4: Commit**

```bash
git add hooks/use-industry-news.ts hooks/use-industry-news-list.ts
git commit -m "Add Industry News admin hooks"
```

---

## Task 12: Admin — form component (with default-author logic) + images manager

**Files:**
- Create: `gmr-admin/components/industry-news/industry-news-form.tsx`
- Create: `gmr-admin/components/industry-news/industry-news-images-manager.tsx`

**Interfaces:**
- Consumes: `AuthorSelector` (`components/statistics/author-selector.tsx`, existing, unchanged), `fetchAuthors` (`lib/api/authors.ts`, existing), `TiptapEditor` (`components/reports/tiptap-editor.tsx`, existing), `InternalLinkPanel` (existing), types/config/validation from Task 9, API client from Task 10.
- Produces: `IndustryNewsForm` (props: `industryNews?`, `onSubmit`, `onPreview?`, `isSaving`, `formId?`), `IndustryNewsImagesManager` (props: `industryNewsId?`, `disabled?`) — used by Task 14 (pages).

- [ ] **Step 1: Write `industry-news-images-manager.tsx`**

This is `report-images-manager.tsx` (Task context: `gmr-admin/components/reports/report-images-manager.tsx`) trimmed to drop the report-only chart-builder integration (`ChartBuilderDialog`, `handleChartSave`, `reportData` prop) — Industry News has no chart generator requirement — and retargeted at the Task 10 API module:

```tsx
'use client';

import { useState, useRef, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent } from '@/components/ui/card';
import { Upload, X, Loader2, Pencil, Check, Image as ImageIcon, Eye, EyeOff, Copy } from 'lucide-react';
import { toast } from 'sonner';
import {
  fetchIndustryNewsImages,
  uploadIndustryNewsImage,
  updateIndustryNewsImageMetadata,
  deleteIndustryNewsImage,
} from '@/lib/api/industry-news-images';
import type { ReportImage } from '@/lib/types/reports';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';

interface IndustryNewsImagesManagerProps {
  industryNewsId?: number | string;
  disabled?: boolean;
}

export function IndustryNewsImagesManager({ industryNewsId, disabled }: IndustryNewsImagesManagerProps) {
  const [images, setImages] = useState<ReportImage[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const [previewImage, setPreviewImage] = useState<ReportImage | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (industryNewsId) {
      loadImages();
    }
  }, [industryNewsId]);

  const loadImages = async () => {
    if (!industryNewsId) return;

    try {
      setIsLoading(true);
      const fetchedImages = await fetchIndustryNewsImages(industryNewsId);
      setImages(fetchedImages);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to load images';
      toast.error(message);
    } finally {
      setIsLoading(false);
    }
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file || !industryNewsId) return;

    const validTypes = ['image/jpeg', 'image/png', 'image/webp', 'image/gif'];
    if (!validTypes.includes(file.type)) {
      toast.error('Invalid file type. Please upload a JPEG, PNG, WebP, or GIF image.');
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
      return;
    }

    const maxSize = 10 * 1024 * 1024; // 10MB
    if (file.size > maxSize) {
      toast.error('File size too large. Maximum size is 10MB.');
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
      return;
    }

    try {
      setIsUploading(true);
      const uploadedImage = await uploadIndustryNewsImage(industryNewsId, file);
      setImages(prev => [...prev, uploadedImage]);
      toast.success('Image uploaded successfully');
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to upload image';
      toast.error(message);
    } finally {
      setIsUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleDelete = async (imageId: number) => {
    if (!confirm('Are you sure you want to delete this image?')) return;

    try {
      await deleteIndustryNewsImage(imageId);
      setImages(prev => prev.filter(img => img.id !== imageId));
      toast.success('Image deleted successfully');
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to delete image';
      toast.error(message);
    }
  };

  const handleStartEdit = (image: ReportImage) => {
    setEditingId(image.id);
    setEditTitle(image.title || '');
  };

  const handleCancelEdit = () => {
    setEditingId(null);
    setEditTitle('');
  };

  const handleSaveEdit = async (imageId: number) => {
    try {
      const updatedImage = await updateIndustryNewsImageMetadata(imageId, {
        title: editTitle || undefined,
      });
      setImages(prev => prev.map(img => (img.id === imageId ? updatedImage : img)));
      setEditingId(null);
      setEditTitle('');
      toast.success('Title updated successfully');
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update title';
      toast.error(message);
    }
  };

  const handleToggleActive = async (image: ReportImage) => {
    try {
      const updatedImage = await updateIndustryNewsImageMetadata(image.id, {
        isActive: !image.isActive,
      });
      setImages(prev => prev.map(img => (img.id === image.id ? updatedImage : img)));
      toast.success(`Image ${updatedImage.isActive ? 'activated' : 'deactivated'}`);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update image status';
      toast.error(message);
    }
  };

  const handleCopyImage = async (image: ReportImage) => {
    try {
      if (!navigator.clipboard || !ClipboardItem) {
        await navigator.clipboard.writeText(image.imageUrl);
        toast.success('Image URL copied to clipboard');
        return;
      }

      const htmlContent = `<img src="${image.imageUrl}" alt="${image.title || ''}" />`;
      const plainText = image.imageUrl;

      const clipboardItem = new ClipboardItem({
        'text/html': new Blob([htmlContent], { type: 'text/html' }),
        'text/plain': new Blob([plainText], { type: 'text/plain' }),
      });

      await navigator.clipboard.write([clipboardItem]);
      toast.success('Image URL copied! You can now paste it into the editor');
    } catch (error) {
      try {
        await navigator.clipboard.writeText(image.imageUrl);
        toast.warning('Copied image URL as text. Paste it in the editor.');
      } catch (fallbackError) {
        console.error('Fallback copy failed:', fallbackError);
        const message = error instanceof Error ? error.message : 'Failed to copy image';
        toast.error(message);
      }
    }
  };

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  if (!industryNewsId) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center p-8">
          <div className="text-center text-muted-foreground">
            <ImageIcon className="h-12 w-12 mx-auto mb-2 opacity-50" />
            <p>Please save the article first to upload images.</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 flex-wrap">
        <Input
          ref={fileInputRef}
          type="file"
          accept="image/jpeg,image/png,image/webp,image/gif"
          onChange={handleFileChange}
          disabled={disabled || isUploading}
          className="hidden"
        />
        <Button
          type="button"
          variant="outline"
          onClick={handleUploadClick}
          disabled={disabled || isUploading}
        >
          {isUploading ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              Uploading...
            </>
          ) : (
            <>
              <Upload className="h-4 w-4 mr-2" />
              Upload Image
            </>
          )}
        </Button>
        <p className="text-xs text-muted-foreground">
          Upload images (JPG, PNG, WebP, GIF - Max 10MB)
        </p>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center p-8">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : images.length === 0 ? (
        <Card>
          <CardContent className="flex items-center justify-center p-8">
            <div className="text-center text-muted-foreground">
              <ImageIcon className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p>No images uploaded yet.</p>
              <p className="text-xs mt-1">Upload your first image to get started.</p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {images.map(image => (
            <Card key={image.id} className={!image.isActive ? 'opacity-50' : ''}>
              <CardContent className="p-4 space-y-3">
                <div
                  className="relative aspect-video bg-muted rounded-md overflow-hidden cursor-pointer hover:opacity-80 transition-opacity"
                  onClick={() => setPreviewImage(image)}
                >
                  <img
                    src={image.imageUrl}
                    alt={image.title || 'Industry news image'}
                    className="object-cover w-full h-full"
                  />
                  {!image.isActive && (
                    <div className="absolute inset-0 flex items-center justify-center bg-black/50">
                      <Badge variant="secondary">Inactive</Badge>
                    </div>
                  )}
                </div>

                <div className="space-y-2">
                  {editingId === image.id ? (
                    <div className="flex gap-2">
                      <Input
                        value={editTitle}
                        onChange={e => setEditTitle(e.target.value)}
                        placeholder="Image title (optional)"
                        className="flex-1"
                        autoFocus
                      />
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        onClick={() => handleSaveEdit(image.id)}
                      >
                        <Check className="h-4 w-4" />
                      </Button>
                      <Button type="button" size="sm" variant="ghost" onClick={handleCancelEdit}>
                        <X className="h-4 w-4" />
                      </Button>
                    </div>
                  ) : (
                    <div className="flex items-start justify-between gap-2">
                      <p className="text-sm text-muted-foreground flex-1 truncate">
                        {image.title || 'Untitled'}
                      </p>
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        onClick={() => handleStartEdit(image)}
                        disabled={disabled}
                      >
                        <Pencil className="h-3 w-3" />
                      </Button>
                    </div>
                  )}
                </div>

                <div className="flex flex-col gap-2">
                  <div className="flex gap-2">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => handleToggleActive(image)}
                      disabled={disabled}
                      className="flex-1"
                    >
                      {image.isActive ? (
                        <>
                          <EyeOff className="h-3 w-3 mr-1" />
                          Hide
                        </>
                      ) : (
                        <>
                          <Eye className="h-3 w-3 mr-1" />
                          Show
                        </>
                      )}
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => handleCopyImage(image)}
                      disabled={disabled}
                      className="flex-1"
                    >
                      <Copy className="h-3 w-3 mr-1" />
                      Copy
                    </Button>
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => handleDelete(image.id)}
                    disabled={disabled}
                    className="w-full"
                  >
                    <X className="h-3 w-3 mr-1" />
                    Delete
                  </Button>
                </div>

                <div className="text-xs text-muted-foreground">
                  <p>ID: {image.id}</p>
                  <p>Uploaded: {new Date(image.createdAt).toLocaleDateString()}</p>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={!!previewImage} onOpenChange={() => setPreviewImage(null)}>
        <DialogContent className="max-w-4xl">
          <DialogHeader>
            <DialogTitle>{previewImage?.title || 'Image Preview'}</DialogTitle>
            <DialogDescription>
              Uploaded on {previewImage && new Date(previewImage.createdAt).toLocaleString()}
            </DialogDescription>
          </DialogHeader>
          {previewImage && (
            <div className="w-full">
              <img
                src={previewImage.imageUrl}
                alt={previewImage.title || 'Industry news image'}
                className="w-full h-auto rounded-md"
              />
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
```

- [ ] **Step 2: Write `industry-news-form.tsx`**

This is `press-release-form.tsx` with the `reportUrl` field removed, the `IndustryNewsImagesManager` added as its own card, and default-author selection logic added:

```tsx
'use client';

import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription,
} from '@/components/ui/form';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { TiptapEditor } from '@/components/reports/tiptap-editor';
import { AuthorSelector } from '@/components/statistics/author-selector';
import { IndustryNewsImagesManager } from './industry-news-images-manager';
import { fetchCategories, type Category } from '@/lib/api/categories';
import { fetchAuthors } from '@/lib/api/authors';
import {
  INDUSTRY_NEWS_TITLE_MAX_LENGTH,
  INDUSTRY_NEWS_EXCERPT_MAX_LENGTH,
  INDUSTRY_NEWS_DEFAULT_AUTHOR_NAME,
} from '@/lib/config/industry-news';
import type {
  IndustryNewsFormData,
  IndustryNews,
  InternalLinkEntry,
} from '@/lib/types/industry-news';
import { industryNewsFormSchema } from '@/lib/validation/industry-news-schema';
import { Save, Eye, Wand2, Copy } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { CharacterCounter } from '@/components/seo/character-counter';
import { SEO_LIMITS } from '@/lib/config/seo';
import { measureTextWidth, SERP_FONTS } from '@/lib/utils/text-measurement';
import { toast } from 'sonner';
import { config } from '@/lib/config';
import { generateSlug } from '@/lib/utils/slug';
import { InternalLinkPanel } from '@/components/editor/internal-link-panel';
import type { TiptapEditorLike } from '@/hooks/use-internal-link-keywords';

interface IndustryNewsFormProps {
  industryNews?: IndustryNews;
  onSubmit: (data: IndustryNewsFormData) => Promise<void>;
  onPreview?: () => void;
  isSaving: boolean;
  formId?: string;
}

export function IndustryNewsForm({
  industryNews,
  onSubmit,
  onPreview,
  isSaving,
  formId,
}: IndustryNewsFormProps) {
  const [keywordInput, setKeywordInput] = useState('');
  const [categories, setCategories] = useState<Category[]>([]);
  const [isLoadingCategories, setIsLoadingCategories] = useState(true);
  const [contentEditor, setContentEditor] = useState<TiptapEditorLike | null>(null);
  const [internalLinks, setInternalLinks] = useState<InternalLinkEntry[]>(
    industryNews?.internalLinks ?? []
  );

  useEffect(() => {
    loadCategories();
  }, []);

  const loadCategories = async () => {
    try {
      setIsLoadingCategories(true);
      const response = await fetchCategories({ limit: 100 });
      setCategories(response.categories.filter(cat => cat.isActive));
    } catch (error) {
      console.error('Failed to load categories:', error);
    } finally {
      setIsLoadingCategories(false);
    }
  };

  const form = useForm<IndustryNewsFormData>({
    resolver: zodResolver(industryNewsFormSchema),
    defaultValues: industryNews
      ? {
          title: industryNews.title,
          slug: industryNews.slug,
          excerpt: industryNews.excerpt,
          content: industryNews.content,
          categoryId: industryNews.categoryId || 0,
          tags: industryNews.tags || '',
          authorId: industryNews.authorId,
          status: industryNews.status,
          publishDate: industryNews.publishDate,
          location: industryNews.location || '',
          metadata: {
            metaTitle: industryNews.metadata?.metaTitle || '',
            metaDescription: industryNews.metadata?.metaDescription || '',
            keywords: industryNews.metadata?.keywords || [],
          },
        }
      : {
          title: '',
          slug: '',
          excerpt: '',
          content: '',
          categoryId: 0,
          tags: '',
          authorId: 0,
          status: 'draft',
          publishDate: new Date().toISOString(),
          location: '',
          metadata: {
            metaTitle: '',
            metaDescription: '',
            keywords: [],
          },
        },
  });

  // Pre-select the "Globe Market Research" default author on create only.
  useEffect(() => {
    if (industryNews) return; // editing an existing article — don't override authorId
    if (form.getValues('authorId')) return; // already set (e.g. by fillSampleData)

    (async () => {
      try {
        const response = await fetchAuthors();
        const defaultAuthor = response.data.find(a => a.name === INDUSTRY_NEWS_DEFAULT_AUTHOR_NAME);
        if (defaultAuthor && !form.getValues('authorId')) {
          form.setValue('authorId', defaultAuthor.id);
        }
      } catch (error) {
        console.error('Failed to load default author:', error);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [industryNews]);

  const fillSampleData = () => {
    const sampleData: Partial<IndustryNewsFormData> = {
      title: 'Healthcare AI Investment Reaches Record High Across Global Markets',
      slug: 'healthcare-ai-investment-reaches-record-high',
      excerpt:
        'Industry analysts report unprecedented capital inflows into healthcare artificial intelligence startups, signaling a fundamental shift in how care delivery is financed and modernized worldwide.',
      content:
        '<p><strong>Global Markets</strong> - Investment in healthcare artificial intelligence has reached a record high this quarter, according to newly released industry data, as venture capital and strategic investors continue to pour funding into companies building clinical decision support, diagnostics, and administrative automation tools.</p><h2>Key Trends</h2><ul><li>Diagnostic AI startups captured the largest share of new funding</li><li>Strategic investment from incumbent health systems is accelerating</li><li>Regulatory clarity in major markets is de-risking later-stage deals</li></ul><h2>Market Outlook</h2><p>Analysts expect the trend to continue as reimbursement pathways mature and more AI-enabled tools receive regulatory clearance across major markets.</p>',
      categoryId: categories.length > 0 ? categories[0].id : 0,
      tags: 'healthcare AI, market trends, digital health, investment',
      location: 'Global',
      metadata: {
        metaTitle: 'Healthcare AI Investment Reaches Record High | Industry News',
        metaDescription:
          'Industry analysts report record capital inflows into healthcare AI, signaling a shift in how care delivery is financed and modernized.',
        keywords: ['healthcare AI', 'digital health investment', 'market trends', 'health tech'],
      },
    };

    Object.entries(sampleData).forEach(([key, value]) => {
      form.setValue(key as keyof IndustryNewsFormData, value);
    });
  };

  const handleFormSubmit = async (data: IndustryNewsFormData) => {
    await onSubmit({ ...data, internalLinks });
  };

  return (
    <Form {...form}>
      <form id={formId} onSubmit={form.handleSubmit(handleFormSubmit)} className="space-y-6">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Basic Information</CardTitle>
              <Button type="button" variant="outline" size="sm" onClick={fillSampleData}>
                <Wand2 className="h-4 w-4 mr-2" />
                Fill Sample Data
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField
              control={form.control}
              name="title"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Industry News Title</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="Enter a compelling title for this article..."
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {field.value.length}/{INDUSTRY_NEWS_TITLE_MAX_LENGTH} characters
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="slug"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Slug</FormLabel>
                  <div className="flex gap-2">
                    <FormControl>
                      <Input placeholder="url-friendly-slug-for-article" {...field} />
                    </FormControl>
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      onClick={() => {
                        const title = form.getValues('title');
                        if (title) {
                          const slug = generateSlug(title);
                          form.setValue('slug', slug);
                          toast.success('Slug generated from title');
                        }
                      }}
                      disabled={!form.watch('title')}
                      title="Generate from title"
                    >
                      <Wand2 className="h-4 w-4" />
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      onClick={() => {
                        if (field.value) {
                          navigator.clipboard.writeText(
                            `${config.preview.domain}/industry-news/${field.value}`
                          );
                          toast.success('URL copied to clipboard');
                        }
                      }}
                      disabled={!field.value}
                      title="Copy URL"
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                  </div>
                  <FormDescription>
                    URL-friendly identifier (lowercase, hyphens only)
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="excerpt"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Excerpt</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder="Write a brief summary that appears in industry news listings..."
                      rows={3}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {field.value.length}/{INDUSTRY_NEWS_EXCERPT_MAX_LENGTH} characters
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="categoryId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="flex items-center gap-1">
                    Category
                    <span className="text-destructive">*</span>
                  </FormLabel>
                  <Select
                    onValueChange={value => field.onChange(Number(value))}
                    value={field.value ? String(field.value) : undefined}
                    disabled={isLoadingCategories}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue
                          placeholder={
                            isLoadingCategories ? 'Loading categories...' : 'Select category'
                          }
                        />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {categories.map(cat => (
                        <SelectItem key={cat.id} value={String(cat.id)}>
                          {cat.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>Select the industry news category</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="tags"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Tags</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="Enter tags (comma-separated, e.g., AI in healthcare, digital health)"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Enter comma-separated tags to help readers discover your content
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="authorId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Author</FormLabel>
                  <FormControl>
                    <AuthorSelector
                      value={String(field.value)}
                      onChange={value => field.onChange(Number(value))}
                    />
                  </FormControl>
                  <FormDescription>
                    Defaults to &quot;{INDUSTRY_NEWS_DEFAULT_AUTHOR_NAME}&quot; — change to credit a
                    specific author if needed.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="location"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Location</FormLabel>
                  <FormControl>
                    <Input placeholder="Enter location (e.g., New York, USA)" {...field} />
                  </FormControl>
                  <FormDescription>Optional location field for the article</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Industry News Content</CardTitle>
          </CardHeader>
          <CardContent>
            <FormField
              control={form.control}
              name="content"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <TiptapEditor
                      content={field.value}
                      onChange={field.onChange}
                      placeholder="Start writing the article..."
                      onEditorReady={ed => setContentEditor(ed as TiptapEditorLike | null)}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Images</CardTitle>
          </CardHeader>
          <CardContent>
            <IndustryNewsImagesManager industryNewsId={industryNews?.id} />
          </CardContent>
        </Card>

        <InternalLinkPanel
          editor={contentEditor}
          onContentChange={html => form.setValue('content', html, { shouldDirty: true })}
          onLinksChange={setInternalLinks}
          initialLinks={industryNews?.internalLinks ?? []}
        />

        <Card>
          <CardHeader>
            <CardTitle>SEO Metadata</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField
              control={form.control}
              name="metadata.metaTitle"
              render={({ field }) => (
                <FormItem>
                  <div className="flex justify-between items-center">
                    <FormLabel>Meta Title</FormLabel>
                    <CharacterCounter
                      current={field.value?.length || 0}
                      max={SEO_LIMITS.metaTitle.max}
                      optimal={SEO_LIMITS.metaTitle.optimal}
                      pixelWidth={{
                        current: measureTextWidth(field.value || '', SERP_FONTS.title),
                        max: SEO_LIMITS.metaTitle.pixelWidth.max,
                      }}
                      variant="inline"
                    />
                  </div>
                  <FormControl>
                    <Input placeholder="SEO-friendly title (optional)" {...field} />
                  </FormControl>
                  <FormDescription>Leave empty to use the article title</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="metadata.metaDescription"
              render={({ field }) => (
                <FormItem>
                  <div className="flex justify-between items-center">
                    <FormLabel>Meta Description</FormLabel>
                    <CharacterCounter
                      current={field.value?.length || 0}
                      max={SEO_LIMITS.metaDescription.max}
                      optimal={SEO_LIMITS.metaDescription.optimal}
                      pixelWidth={{
                        current: measureTextWidth(field.value || '', SERP_FONTS.description),
                        max: SEO_LIMITS.metaDescription.pixelWidth.max,
                      }}
                      variant="inline"
                    />
                  </div>
                  <FormControl>
                    <Textarea placeholder="SEO description (120-160 characters)" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="metadata.keywords"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Keywords</FormLabel>
                  <div className="flex gap-2">
                    <Input
                      placeholder="Add keyword"
                      value={keywordInput}
                      onChange={e => setKeywordInput(e.target.value)}
                      onKeyDown={e => {
                        if (e.key === 'Enter') {
                          e.preventDefault();
                          if (keywordInput.trim()) {
                            field.onChange([...(field.value || []), keywordInput.trim()]);
                            setKeywordInput('');
                          }
                        }
                      }}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => {
                        if (keywordInput.trim()) {
                          field.onChange([...(field.value || []), keywordInput.trim()]);
                          setKeywordInput('');
                        }
                      }}
                    >
                      Add
                    </Button>
                  </div>
                  <div className="flex flex-wrap gap-2 mt-2">
                    {field.value?.map((keyword, i) => (
                      <Badge key={i} variant="secondary">
                        {keyword}
                        <button
                          type="button"
                          className="ml-2"
                          onClick={() => {
                            field.onChange(field.value?.filter((_, idx) => idx !== i));
                          }}
                        >
                          ×
                        </button>
                      </Badge>
                    ))}
                  </div>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        {(onPreview || !formId) && (
          <Card>
            <CardContent className="pt-6">
              <div className="flex justify-between">
                <div>
                  {onPreview && (
                    <Button type="button" variant="outline" onClick={onPreview}>
                      <Eye className="h-4 w-4 mr-2" />
                      Preview
                    </Button>
                  )}
                </div>
                {!formId && (
                  <Button type="submit" disabled={isSaving}>
                    <Save className="h-4 w-4 mr-2" />
                    {isSaving ? 'Saving...' : 'Save Industry News'}
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        )}
      </form>
    </Form>
  );
}
```

- [ ] **Step 3: Verify types compile**

Run: `cd gmr-admin && npm run type-check`
Expected: no new errors. Confirm `fetchAuthors()` returns `{ data: ReportAuthor[] }` as used above — check `lib/api/authors.ts` if the type check flags a mismatch (the AuthorSelector component already calls it the same way in Step 48 of `author-selector.tsx`).

- [ ] **Step 4: Commit**

```bash
git add components/industry-news/industry-news-form.tsx components/industry-news/industry-news-images-manager.tsx
git commit -m "Add Industry News admin form with default-author selection and image gallery"
```

---

## Task 13: Admin — list table + filters components

**Files:**
- Create: `gmr-admin/components/industry-news/industry-news-list.tsx`
- Create: `gmr-admin/components/industry-news/industry-news-filters.tsx`

**Interfaces:**
- Consumes: types/config from Task 9.
- Produces: `IndustryNewsList` (props: `industryNewsItems`, `isLoading`, `viewMode?`, `onDelete?`, `onSoftDelete?`, `onRestore?`, `onHardDelete?`), `IndustryNewsFiltersComponent` (props: `filters`, `onFiltersChange`, `authors?`) — used by Task 14 (pages).

- [ ] **Step 1: Write `industry-news-list.tsx`**

Direct copy of `press-release-list.tsx` renamed (`pressRelease`→`industryNewsItem` per row, `PressReleaseList`→`IndustryNewsList`, routes `/press-releases/...`→`/industry-news/...`, public preview path `/press-release/`→`/industry-news/`):

```tsx
'use client';

import Link from 'next/link';
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { TableSkeleton } from '@/components/ui/skeletons/table-skeleton';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import type { IndustryNews, IndustryNewsStatus } from '@/lib/types/industry-news';
import { formatDate } from '@/lib/utils/date';
import { Edit, Eye, Trash2, ExternalLink, RotateCcw } from 'lucide-react';
import { INDUSTRY_NEWS_STATUS_CONFIG } from '@/lib/config/industry-news';
import { config } from '@/lib/config';

interface IndustryNewsListProps {
  industryNewsItems: IndustryNews[];
  isLoading: boolean;
  viewMode?: 'active' | 'trash';
  onDelete?: (id: number) => void;
  onSoftDelete?: (id: number) => void;
  onRestore?: (id: number) => void;
  onHardDelete?: (id: number) => void;
}

function getStatusBadgeVariant(
  status: IndustryNewsStatus
): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'published':
      return 'default';
    case 'review':
      return 'outline';
    case 'draft':
    default:
      return 'secondary';
  }
}

function getAuthorInitials(name?: string): string {
  if (!name) return 'U';
  return name
    .split(' ')
    .map(n => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);
}

export function IndustryNewsList({
  industryNewsItems,
  isLoading,
  viewMode = 'active',
  onDelete,
  onSoftDelete,
  onRestore,
  onHardDelete,
}: IndustryNewsListProps) {
  if (isLoading) {
    return <TableSkeleton rows={5} columns={6} showHeader={true} showActions={true} />;
  }

  if (industryNewsItems.length === 0) {
    return (
      <div className="text-center py-12 border rounded-lg">
        <p className="text-muted-foreground">No industry news found</p>
      </div>
    );
  }

  return (
    <div className="fade-in">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Title</TableHead>
            <TableHead>Author</TableHead>
            <TableHead>Category</TableHead>
            <TableHead>Location</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Updated</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {industryNewsItems.map(item => {
            return (
              <TableRow key={item.id} className={viewMode === 'trash' ? 'opacity-70' : ''}>
                <TableCell className="font-medium max-w-xs">
                  <div className="truncate">{item.title}</div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <Avatar className="h-6 w-6">
                      <AvatarFallback className="text-xs">
                        {getAuthorInitials(item.author?.name)}
                      </AvatarFallback>
                    </Avatar>
                    <span className="text-sm truncate max-w-[100px]">
                      {item.author?.name || 'Unknown Author'}
                    </span>
                  </div>
                </TableCell>
                <TableCell>{item.category?.name || 'Uncategorized'}</TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {item.location || '-'}
                </TableCell>
                <TableCell>
                  {item.status === 'draft' &&
                  item.scheduledPublishEnabled &&
                  item.publishDate &&
                  new Date(item.publishDate) > new Date() ? (
                    <Badge variant="outline">Scheduled</Badge>
                  ) : (
                    <Badge variant={getStatusBadgeVariant(item.status)}>
                      {INDUSTRY_NEWS_STATUS_CONFIG[item.status].label}
                    </Badge>
                  )}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {formatDate(item.updatedAt)}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-2">
                    {viewMode === 'active' && (
                      <>
                        <Button variant="ghost" size="sm" asChild>
                          <Link href={`/industry-news/${item.id}/preview`}>
                            <Eye className="h-4 w-4" />
                          </Link>
                        </Button>
                        {config.preview.domain && (
                          <Button variant="ghost" size="sm" asChild title="Preview on public site">
                            <Link
                              href={`${config.preview.domain}/industry-news/${item.slug}`}
                              target="_blank"
                              rel="noopener noreferrer"
                            >
                              <ExternalLink className="h-4 w-4" />
                            </Link>
                          </Button>
                        )}
                        <Button variant="ghost" size="sm" asChild>
                          <Link href={`/industry-news/${item.id}`}>
                            <Edit className="h-4 w-4" />
                          </Link>
                        </Button>
                        {(onSoftDelete || onDelete) && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() =>
                              onSoftDelete ? onSoftDelete(item.id) : onDelete?.(item.id)
                            }
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        )}
                      </>
                    )}
                    {viewMode === 'trash' && (
                      <>
                        {onRestore && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => onRestore(item.id)}
                            title="Restore"
                          >
                            <RotateCcw className="h-4 w-4 text-green-600" />
                          </Button>
                        )}
                        {onHardDelete && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => onHardDelete(item.id)}
                            title="Permanent Delete"
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        )}
                      </>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
```

- [ ] **Step 2: Write `industry-news-filters.tsx`**

Direct copy of `press-release-filters.tsx` renamed:

```tsx
'use client';

import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Button } from '@/components/ui/button';
import type { IndustryNewsFilters } from '@/lib/types/industry-news';
import type { ReportAuthor } from '@/lib/types/reports';
import { fetchCategories, type Category } from '@/lib/api/categories';
import { Search, X } from 'lucide-react';
import { useState, useEffect } from 'react';

interface IndustryNewsFiltersProps {
  filters: IndustryNewsFilters;
  onFiltersChange: (filters: IndustryNewsFilters) => void;
  authors?: ReportAuthor[];
}

export function IndustryNewsFiltersComponent({
  filters,
  onFiltersChange,
  authors = [],
}: IndustryNewsFiltersProps) {
  const [searchInput, setSearchInput] = useState(filters.search || '');
  const [categories, setCategories] = useState<Category[]>([]);
  const [isLoadingCategories, setIsLoadingCategories] = useState(true);

  useEffect(() => {
    loadCategories();
  }, []);

  const loadCategories = async () => {
    try {
      setIsLoadingCategories(true);
      const response = await fetchCategories({ limit: 100 });
      setCategories(response.categories.filter(cat => cat.isActive));
    } catch (error) {
      console.error('Failed to load categories:', error);
    } finally {
      setIsLoadingCategories(false);
    }
  };

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onFiltersChange({ ...filters, search: searchInput, page: 1 });
  };

  const clearFilters = () => {
    setSearchInput('');
    onFiltersChange({ page: 1 });
  };

  const hasActiveFilters =
    filters.status || filters.categoryId || filters.authorId || filters.tags || filters.search;

  return (
    <div className="space-y-4">
      <form onSubmit={handleSearchSubmit} className="flex gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search industry news by title, excerpt, or tags..."
            value={searchInput}
            onChange={e => setSearchInput(e.target.value)}
            className="pl-10"
          />
        </div>
        <Button type="submit">Search</Button>
      </form>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Select
          value={filters.status || 'all'}
          onValueChange={value =>
            onFiltersChange({
              ...filters,
              status: value === 'all' ? undefined : (value as IndustryNewsFilters['status']),
              page: 1,
            })
          }
        >
          <SelectTrigger>
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="draft">Draft</SelectItem>
            <SelectItem value="review">In Review</SelectItem>
            <SelectItem value="published">Published</SelectItem>
          </SelectContent>
        </Select>

        <Select
          value={filters.categoryId ? String(filters.categoryId) : 'all'}
          onValueChange={value =>
            onFiltersChange({
              ...filters,
              categoryId: value === 'all' ? undefined : parseInt(value, 10),
              page: 1,
            })
          }
          disabled={isLoadingCategories}
        >
          <SelectTrigger>
            <SelectValue placeholder={isLoadingCategories ? 'Loading...' : 'Category'} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Categories</SelectItem>
            {categories.map(cat => (
              <SelectItem key={cat.id} value={String(cat.id)}>
                {cat.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Input
          type="text"
          placeholder="Filter by tag..."
          value={filters.tags || ''}
          onChange={e => {
            onFiltersChange({ ...filters, tags: e.target.value || undefined, page: 1 });
          }}
        />

        <Select
          value={filters.authorId ? String(filters.authorId) : 'all'}
          onValueChange={value =>
            onFiltersChange({
              ...filters,
              authorId: value === 'all' ? undefined : parseInt(value, 10),
              page: 1,
            })
          }
        >
          <SelectTrigger>
            <SelectValue placeholder="Author" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Authors</SelectItem>
            {authors.map(author => (
              <SelectItem key={author.id} value={String(author.id)}>
                {author.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {hasActiveFilters && (
        <Button variant="outline" size="sm" onClick={clearFilters}>
          <X className="h-4 w-4 mr-2" />
          Clear Filters
        </Button>
      )}
    </div>
  );
}
```

- [ ] **Step 3: Verify types compile**

Run: `cd gmr-admin && npm run type-check`
Expected: no new errors.

- [ ] **Step 4: Commit**

```bash
git add components/industry-news/industry-news-list.tsx components/industry-news/industry-news-filters.tsx
git commit -m "Add Industry News admin list and filters components"
```

---

## Task 14: Admin — pages (list/new/edit/preview/trash) + sidebar nav

**Files:**
- Create: `gmr-admin/app/(dashboard)/industry-news/page.tsx`
- Create: `gmr-admin/app/(dashboard)/industry-news/new/page.tsx`
- Create: `gmr-admin/app/(dashboard)/industry-news/[id]/page.tsx`
- Create: `gmr-admin/app/(dashboard)/industry-news/[id]/preview/page.tsx`
- Create: `gmr-admin/app/(dashboard)/industry-news/trash/page.tsx`
- Modify: `gmr-admin/lib/navigation.ts`

**Interfaces:**
- Consumes: hooks (Task 11), form (Task 12), list/filters (Task 13), types/config (Task 9).

- [ ] **Step 1: Write `app/(dashboard)/industry-news/new/page.tsx`**

```tsx
'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { IndustryNewsForm } from '@/components/industry-news/industry-news-form';
import { useIndustryNews } from '@/hooks/use-industry-news';
import { useAuth } from '@/contexts/auth-context';

export default function CreateIndustryNewsPage() {
  const { user } = useAuth();
  const router = useRouter();
  const { saveIndustryNews, isSaving } = useIndustryNews();

  useEffect(() => {
    if (user && user.role !== 'admin' && user.role !== 'editor') {
      router.push('/industry-news');
    }
  }, [user, router]);

  if (!user || (user.role !== 'admin' && user.role !== 'editor')) {
    return null;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Create Industry News</h1>
        <p className="text-muted-foreground mt-2">Write and publish a new industry news article</p>
      </div>

      <IndustryNewsForm
        onSubmit={async data => {
          await saveIndustryNews(null, data);
        }}
        isSaving={isSaving}
      />
    </div>
  );
}
```

- [ ] **Step 2: Write `app/(dashboard)/industry-news/[id]/page.tsx`**

```tsx
'use client';

import { useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { IndustryNewsForm } from '@/components/industry-news/industry-news-form';
import { WorkflowStatus } from '@/components/shared/workflow-status';
import { ScheduledPublishCard } from '@/components/shared/scheduled-publish-card';
import {
  INDUSTRY_NEWS_STATUS_CONFIG,
  INDUSTRY_NEWS_WORKFLOW_TRANSITIONS,
} from '@/lib/config/industry-news';
import { useIndustryNews } from '@/hooks/use-industry-news';
import { useAuth } from '@/contexts/auth-context';
import { FormSkeleton } from '@/components/ui/skeletons/form-skeleton';
import { Skeleton } from '@/components/ui/skeleton';
import { AlertCircle, Save } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';

export default function EditIndustryNewsPage() {
  const params = useParams();
  const router = useRouter();
  const { user } = useAuth();
  const {
    industryNews,
    isLoading,
    error,
    fetchIndustryNewsItem,
    saveIndustryNews,
    isSaving,
    submitIndustryNewsForReview,
    publishIndustryNewsPost,
    unpublishIndustryNewsPost,
    scheduleIndustryNewsPublish,
    cancelIndustryNewsSchedule,
  } = useIndustryNews();
  const industryNewsId = params.id as string;

  useEffect(() => {
    if (industryNewsId) {
      fetchIndustryNewsItem(parseInt(industryNewsId, 10));
    }
  }, [industryNewsId, fetchIndustryNewsItem]);

  useEffect(() => {
    if (user && user.role !== 'admin' && user.role !== 'editor') {
      router.push('/industry-news');
    }
  }, [user, router]);

  const handleStatusChange = async (newStatus: 'draft' | 'review' | 'published') => {
    const id = parseInt(industryNewsId, 10);
    if (newStatus === 'review') {
      await submitIndustryNewsForReview(id);
    } else if (newStatus === 'published') {
      await publishIndustryNewsPost(id);
    } else if (newStatus === 'draft') {
      await unpublishIndustryNewsPost(id);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="space-y-2">
          <Skeleton className="h-9 w-48" />
          <Skeleton className="h-5 w-96" />
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <FormSkeleton sections={1} fieldsPerSection={6} showTabs={false} />
          </div>
          <div className="space-y-6">
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-64 w-full" />
          </div>
        </div>
      </div>
    );
  }

  if (error || !industryNews) {
    return (
      <div className="flex flex-col items-center justify-center p-12 border rounded-lg">
        <AlertCircle className="h-12 w-12 text-destructive mb-4" />
        <p className="text-lg font-semibold mb-2">Failed to load industry news</p>
        <p className="text-sm text-muted-foreground mb-4">{error}</p>
        <Button onClick={() => fetchIndustryNewsItem(parseInt(industryNewsId, 10))}>Retry</Button>
      </div>
    );
  }

  const isAdmin = user?.role === 'admin';

  return (
    <div className="space-y-6 fade-in">
      <div>
        <h1 className="text-3xl font-bold">Edit Industry News</h1>
        <p className="text-muted-foreground mt-2">{industryNews.title}</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <IndustryNewsForm
            industryNews={industryNews}
            onSubmit={async data => {
              await saveIndustryNews(parseInt(industryNewsId, 10), data);
            }}
            onPreview={() => router.push(`/industry-news/${industryNewsId}/preview`)}
            isSaving={isSaving}
            formId="industry-news-edit-form"
          />
        </div>

        <div className="space-y-6">
          <Card>
            <CardContent className="pt-6">
              <Button
                type="submit"
                form="industry-news-edit-form"
                disabled={isSaving}
                className="w-full"
              >
                <Save className="h-4 w-4 mr-2" />
                {isSaving ? 'Saving...' : 'Save Industry News'}
              </Button>
            </CardContent>
          </Card>
          <WorkflowStatus
            currentStatus={industryNews.status}
            onStatusChange={handleStatusChange}
            isSaving={isSaving}
            isAdmin={isAdmin}
            statusConfig={INDUSTRY_NEWS_STATUS_CONFIG}
            workflowTransitions={INDUSTRY_NEWS_WORKFLOW_TRANSITIONS}
          />
          <ScheduledPublishCard
            currentScheduledDate={
              industryNews.scheduledPublishEnabled ? industryNews.publishDate : undefined
            }
            currentStatus={industryNews.status}
            onSchedule={async date => {
              await scheduleIndustryNewsPublish(industryNewsId, date);
            }}
            onCancelSchedule={async () => {
              await cancelIndustryNewsSchedule(industryNewsId);
            }}
            isSaving={isSaving}
          />
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Write `app/(dashboard)/industry-news/[id]/preview/page.tsx`**

Direct copy of `press-releases/[id]/preview/page.tsx` renamed (drops the `reportUrl`-derived sidebar CTA, which Industry News doesn't have — Industry News never had that field):

```tsx
'use client';

import { useEffect } from 'react';
import { useParams } from 'next/navigation';
import { useIndustryNews } from '@/hooks/use-industry-news';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { ArrowLeft, Edit, Calendar, MapPin } from 'lucide-react';
import Link from 'next/link';
import { formatRelativeTime } from '@/lib/utils/date';
import { INDUSTRY_NEWS_STATUS_CONFIG } from '@/lib/config/industry-news';

function getAuthorInitials(name?: string): string {
  if (!name) return 'U';
  return name
    .split(' ')
    .map(n => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);
}

export default function PreviewIndustryNewsPage() {
  const params = useParams();
  const { industryNews, isLoading, error, fetchIndustryNewsItem } = useIndustryNews();
  const industryNewsId = parseInt(params.id as string, 10);

  useEffect(() => {
    if (industryNewsId && !isNaN(industryNewsId)) {
      fetchIndustryNewsItem(industryNewsId);
    }
  }, [industryNewsId, fetchIndustryNewsItem]);

  if (isLoading) {
    return (
      <div className="space-y-6 max-w-4xl mx-auto">
        <Skeleton className="h-12 w-64" />
        <Skeleton className="h-96 w-full" />
      </div>
    );
  }

  if (error || !industryNews) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">Error loading industry news</p>
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <div className="flex items-center justify-between">
        <Button variant="ghost" asChild>
          <Link href="/industry-news">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Industry News
          </Link>
        </Button>
        <Button asChild>
          <Link href={`/industry-news/${industryNewsId}`}>
            <Edit className="mr-2 h-4 w-4" />
            Edit Industry News
          </Link>
        </Button>
      </div>

      <Card>
        <CardHeader className="space-y-4">
          <div className="flex items-center gap-2">
            <Badge
              variant={
                industryNews.status === 'published'
                  ? 'default'
                  : industryNews.status === 'review'
                    ? 'outline'
                    : 'secondary'
              }
            >
              {INDUSTRY_NEWS_STATUS_CONFIG[industryNews.status].label}
            </Badge>
          </div>

          <h1 className="text-3xl font-bold">{industryNews.title}</h1>

          <p className="text-lg text-muted-foreground">{industryNews.excerpt}</p>

          <div className="flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
            {industryNews.author && (
              <div className="flex items-center gap-2">
                <Avatar className="h-8 w-8">
                  <AvatarFallback>{getAuthorInitials(industryNews.author.name)}</AvatarFallback>
                </Avatar>
                <span>{industryNews.author.name}</span>
              </div>
            )}

            {industryNews.author && <span>•</span>}

            <div className="flex items-center gap-1">
              <Calendar className="h-4 w-4" />
              <span>{formatRelativeTime(industryNews.publishDate)}</span>
            </div>

            {industryNews.location && (
              <>
                <span>•</span>
                <div className="flex items-center gap-1">
                  <MapPin className="h-4 w-4" />
                  <span>{industryNews.location}</span>
                </div>
              </>
            )}
          </div>

          {industryNews.tags && (
            <div className="flex flex-wrap gap-2">
              {industryNews.tags.split(',').map((tag, idx) => (
                <Badge key={idx} variant="secondary">
                  {tag.trim()}
                </Badge>
              ))}
            </div>
          )}
        </CardHeader>

        <CardContent>
          <div
            className="prose prose-neutral dark:prose-invert max-w-none"
            dangerouslySetInnerHTML={{ __html: industryNews.content }}
          />

          {industryNews.author?.bio && (
            <div className="mt-8 pt-8 border-t">
              <div className="flex items-start gap-4">
                <Avatar className="h-12 w-12">
                  <AvatarFallback>{getAuthorInitials(industryNews.author.name)}</AvatarFallback>
                </Avatar>
                <div>
                  <p className="font-semibold">{industryNews.author.name}</p>
                  <p className="text-sm text-muted-foreground">{industryNews.author.bio}</p>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardContent className="space-y-4">
          {industryNews.metadata.keywords && industryNews.metadata.keywords.length > 0 && (
            <div>
              <p className="text-sm font-medium mb-2">Keywords</p>
              <div className="flex flex-wrap gap-2">
                {industryNews.metadata.keywords.map((keyword, i) => (
                  <Badge key={i} variant="outline">
                    {keyword}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
```

- [ ] **Step 4: Write `app/(dashboard)/industry-news/page.tsx`**

Direct copy of `press-releases/page.tsx` renamed (routes `/press-releases/...`→`/industry-news/...`, hook `usePressReleases`→`useIndustryNewsList`, list field `pressReleases`→`industryNewsItems`):

```tsx
'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { IndustryNewsFiltersComponent } from '@/components/industry-news/industry-news-filters';
import { IndustryNewsList } from '@/components/industry-news/industry-news-list';
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination';
import { useIndustryNewsList } from '@/hooks/use-industry-news-list';
import { useAuth } from '@/contexts/auth-context';
import { Plus, RefreshCw, Trash2 } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { toast } from 'sonner';
import { fetchAuthors } from '@/lib/api/authors';
import type { ReportAuthor } from '@/lib/types/reports';
import type { IndustryNewsFilters } from '@/lib/types/industry-news';

export default function IndustryNewsPage() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { user } = useAuth();
  const initialPage = Number(searchParams.get('page')) > 0 ? Number(searchParams.get('page')) : 1;
  const [filters, setFilters] = useState<IndustryNewsFilters>({ page: initialPage });
  const [authors, setAuthors] = useState<ReportAuthor[]>([]);
  const {
    industryNewsItems,
    total,
    totalPages,
    currentPage,
    isLoading,
    refetch,
    setFilters: updateFilters,
    softDelete,
  } = useIndustryNewsList(filters);
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    industryNewsId: number | null;
  }>({
    open: false,
    industryNewsId: null,
  });

  const syncPageParam = (page: number) => {
    const params = new URLSearchParams(searchParams.toString());
    params.set('page', String(page));
    router.replace(`${pathname}?${params.toString()}`, { scroll: false });
  };

  const applyFilters = (nextFilters: IndustryNewsFilters) => {
    const normalizedFilters = { ...nextFilters, page: nextFilters.page ?? 1 };
    setFilters(normalizedFilters);
    updateFilters(normalizedFilters);
    syncPageParam(normalizedFilters.page);
  };

  const loadAuthors = async () => {
    try {
      const { data: authors } = await fetchAuthors();
      setAuthors(authors);
    } catch {
      // Error is logged by the API client
    }
  };

  useEffect(() => {
    loadAuthors();
  }, []);

  useEffect(() => {
    const pageParam = searchParams.get('page');

    if (!pageParam) {
      syncPageParam(1);
    }
  }, [pathname, router, searchParams]);

  useEffect(() => {
    const pageParam = Number(searchParams.get('page'));

    if (currentPage > 0 && currentPage !== pageParam) {
      syncPageParam(currentPage);
    }
  }, [currentPage, pathname, router, searchParams]);

  const handleDelete = async () => {
    if (!deleteDialog.industryNewsId) return;

    try {
      await softDelete(deleteDialog.industryNewsId);
      toast.success('Industry news moved to trash successfully');
      setDeleteDialog({ open: false, industryNewsId: null });
    } catch {
      toast.error('Failed to move industry news to trash');
    }
  };

  const canCreateEdit = user?.role === 'admin' || user?.role === 'editor';

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Industry News</h1>
          <p className="text-muted-foreground mt-2">Manage industry news ({total} total)</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={refetch}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          {canCreateEdit && (
            <Button variant="outline" size="sm" asChild>
              <Link href="/industry-news/trash">
                <Trash2 className="mr-2 h-4 w-4" />
                View Trash
              </Link>
            </Button>
          )}
          {canCreateEdit && (
            <Button asChild>
              <Link href="/industry-news/new">
                <Plus className="mr-2 h-4 w-4" />
                Create Industry News
              </Link>
            </Button>
          )}
        </div>
      </div>

      <IndustryNewsFiltersComponent
        filters={filters}
        onFiltersChange={newFilters => {
          applyFilters(newFilters);
        }}
        authors={authors}
      />

      <IndustryNewsList
        industryNewsItems={industryNewsItems}
        isLoading={isLoading}
        onSoftDelete={
          canCreateEdit ? id => setDeleteDialog({ open: true, industryNewsId: id }) : undefined
        }
      />

      {!isLoading && totalPages > 1 && (
        <Pagination>
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious
                href="#"
                onClick={e => {
                  e.preventDefault();
                  if (currentPage > 1) applyFilters({ ...filters, page: currentPage - 1 });
                }}
                aria-disabled={currentPage <= 1}
              />
            </PaginationItem>

            {Array.from({ length: totalPages }, (_, i) => i + 1).map(page => (
              <PaginationItem key={page}>
                <PaginationLink
                  href="#"
                  isActive={page === currentPage}
                  onClick={e => {
                    e.preventDefault();
                    if (page !== currentPage) applyFilters({ ...filters, page });
                  }}
                >
                  {page}
                </PaginationLink>
              </PaginationItem>
            ))}

            <PaginationItem>
              <PaginationNext
                href="#"
                onClick={e => {
                  e.preventDefault();
                  if (currentPage < totalPages) applyFilters({ ...filters, page: currentPage + 1 });
                }}
                aria-disabled={currentPage >= totalPages}
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      )}

      <Dialog
        open={deleteDialog.open}
        onOpenChange={open => setDeleteDialog({ open, industryNewsId: null })}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Move Industry News to Trash</DialogTitle>
            <DialogDescription>
              Are you sure you want to move this article to trash? You can restore it later from
              the trash.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialog({ open: false, industryNewsId: null })}
            >
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              Move to Trash
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
```

- [ ] **Step 5: Write `app/(dashboard)/industry-news/trash/page.tsx`**

Direct copy of `press-releases/trash/page.tsx` renamed the same way as Step 4 (`fetchTrashedPressReleases`→`fetchTrashedIndustryNews`, `deletePressRelease`→`deleteIndustryNews`, `restorePressRelease`→`restoreIndustryNews`):

```tsx
'use client';

import { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { Trash2, RotateCcw, ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { IndustryNewsList } from '@/components/industry-news/industry-news-list';
import { IndustryNewsFiltersComponent } from '@/components/industry-news/industry-news-filters';
import { PaginationWrapper as Pagination } from '@/components/ui/pagination-wrapper';
import { useAuth } from '@/contexts/auth-context';
import {
  deleteIndustryNews,
  fetchTrashedIndustryNews,
  restoreIndustryNews,
} from '@/lib/api/industry-news';
import { fetchAuthors } from '@/lib/api/authors';
import type { IndustryNews, IndustryNewsFilters } from '@/lib/types/industry-news';
import type { ReportAuthor } from '@/lib/types/reports';

function useTrashedIndustryNews(initialFilters?: IndustryNewsFilters) {
  const [industryNewsItems, setIndustryNewsItems] = useState<IndustryNews[]>([]);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [currentPage, setCurrentPage] = useState(initialFilters?.page || 1);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFiltersState] = useState<IndustryNewsFilters>(initialFilters || {});

  const fetchData = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const response = await fetchTrashedIndustryNews(filters);
      setIndustryNewsItems(response.industryNews);
      setTotal(response.total);
      setTotalPages(response.totalPages);
      setCurrentPage(response.page);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch industry news';
      setError(errorMessage);
      toast.error('Failed to load trashed industry news');
    } finally {
      setIsLoading(false);
    }
  }, [filters]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const setFilters = useCallback((newFilters: IndustryNewsFilters) => {
    setFiltersState(prev => ({ ...prev, ...newFilters }));
  }, []);

  const restore = useCallback(
    async (id: number) => {
      try {
        await restoreIndustryNews(id);
        await fetchData();
      } catch (error) {
        throw error;
      }
    },
    [fetchData]
  );

  const hardDelete = useCallback(
    async (id: number) => {
      try {
        await deleteIndustryNews(id);
        await fetchData();
      } catch (error) {
        throw error;
      }
    },
    [fetchData]
  );

  return {
    industryNewsItems,
    total,
    totalPages,
    currentPage,
    isLoading,
    error,
    refetch: fetchData,
    setFilters,
    restore,
    hardDelete,
  };
}

export default function IndustryNewsTrashPage() {
  const router = useRouter();
  const { user } = useAuth();
  const [filters, setFilters] = useState<IndustryNewsFilters>({ page: 1, limit: 10 });
  const [authors, setAuthors] = useState<ReportAuthor[]>([]);
  const {
    industryNewsItems,
    total,
    totalPages,
    currentPage,
    isLoading,
    refetch,
    setFilters: updateFilters,
    restore,
    hardDelete,
  } = useTrashedIndustryNews(filters);

  const [restoreDialog, setRestoreDialog] = useState<{
    open: boolean;
    industryNewsId: number | null;
  }>({
    open: false,
    industryNewsId: null,
  });

  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    industryNewsId: number | null;
  }>({
    open: false,
    industryNewsId: null,
  });

  const isAdmin = user?.role === 'admin';

  const loadAuthors = async () => {
    try {
      const { data: authors } = await fetchAuthors();
      setAuthors(authors);
    } catch {
      // Error is logged by the API client
    }
  };

  useEffect(() => {
    loadAuthors();
  }, []);

  const handleRestore = async () => {
    if (!restoreDialog.industryNewsId) return;

    try {
      await restore(restoreDialog.industryNewsId);
      toast.success('Industry news restored successfully');
      setRestoreDialog({ open: false, industryNewsId: null });
    } catch (error) {
      console.error('Failed to restore industry news:', error);
      toast.error('Failed to restore industry news');
    }
  };

  const handlePermanentDelete = async () => {
    if (!deleteDialog.industryNewsId) return;

    try {
      await hardDelete(deleteDialog.industryNewsId);
      toast.success('Industry news permanently deleted');
      setDeleteDialog({ open: false, industryNewsId: null });
    } catch (error) {
      console.error('Failed to delete industry news:', error);
      toast.error('Failed to delete industry news permanently');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => router.push('/industry-news')}>
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <Trash2 className="h-8 w-8" />
              Trash
            </h1>
            <p className="text-muted-foreground mt-1">
              {total} {total === 1 ? 'article' : 'articles'} in trash
            </p>
          </div>
        </div>

        <Button onClick={refetch} variant="outline">
          Refresh
        </Button>
      </div>

      <IndustryNewsFiltersComponent
        filters={filters}
        onFiltersChange={newFilters => {
          setFilters({ ...newFilters, page: 1 });
          updateFilters({ ...newFilters, page: 1 });
        }}
        authors={authors}
      />

      {isLoading ? (
        <div>Loading...</div>
      ) : industryNewsItems.length === 0 ? (
        <div className="text-center py-12 border rounded-lg">
          <Trash2 className="h-12 w-12 mx-auto text-muted-foreground mb-2" />
          <p className="text-lg font-medium">Trash is empty</p>
          <p className="text-muted-foreground mt-1">Deleted industry news will appear here</p>
        </div>
      ) : (
        <>
          <IndustryNewsList
            industryNewsItems={industryNewsItems}
            isLoading={isLoading}
            viewMode="trash"
            onRestore={id => setRestoreDialog({ open: true, industryNewsId: id })}
            onHardDelete={
              isAdmin ? id => setDeleteDialog({ open: true, industryNewsId: id }) : undefined
            }
          />

          {!isLoading && totalPages > 1 && (
            <Pagination
              currentPage={currentPage}
              totalPages={totalPages}
              onPageChange={page => {
                const newFilters = { ...filters, page };
                setFilters(newFilters);
                updateFilters(newFilters);
              }}
            />
          )}
        </>
      )}

      <AlertDialog
        open={restoreDialog.open}
        onOpenChange={open => setRestoreDialog({ ...restoreDialog, open })}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Restore Industry News?</AlertDialogTitle>
            <AlertDialogDescription>
              This article will be restored and moved back to active industry news.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleRestore}>
              <RotateCcw className="h-4 w-4 mr-2" />
              Restore
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={deleteDialog.open}
        onOpenChange={open => setDeleteDialog({ ...deleteDialog, open })}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Permanently Delete Industry News?</AlertDialogTitle>
            <AlertDialogDescription className="text-destructive font-medium">
              This action cannot be undone! The article and all its data will be permanently
              deleted from the database.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handlePermanentDelete}
              className="bg-destructive hover:bg-destructive/90"
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Delete Permanently
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
```

- [ ] **Step 6: Add the sidebar nav entry**

In `gmr-admin/lib/navigation.ts`, add `Rss` to the `lucide-react` import list (it's not already used — `Newspaper` is taken by Media Mentions):

```typescript
  Rss,
```

Then, after the `Press Releases` block (after its closing `},` around line 110), add:

```typescript
  {
    title: 'Industry News',
    href: '/industry-news',
    icon: Rss,
    roles: ['admin', 'editor'],
    children: [
      {
        title: 'All Industry News',
        href: '/industry-news',
        icon: Rss,
        roles: ['admin', 'editor'],
      },
      {
        title: 'Create Industry News',
        href: '/industry-news/new',
        icon: Rss,
        roles: ['admin', 'editor'],
      },
      {
        title: 'Trash',
        href: '/industry-news/trash',
        icon: Trash2,
        roles: ['admin', 'editor'],
      },
    ],
  },
```

- [ ] **Step 7: Verify types compile and build**

Run: `cd gmr-admin && npm run type-check && npm run build`
Expected: no errors.

- [ ] **Step 8: Manual walkthrough (no test framework in this app)**

Run: `cd gmr-admin && npm run dev`, log in as an admin/editor user, then:
1. Navigate to `/industry-news` — empty list renders without error.
2. Click "Create Industry News" — form loads, Author field pre-fills to "Globe Market Research" once authors load.
3. Fill required fields (or click "Fill Sample Data"), save — redirects to the edit page for the new article.
4. On the edit page, upload an image in the Images card — it appears in the gallery; click "Copy" and confirm a toast fires.
5. Visit `/industry-news/trash` — empty, no console errors.

- [ ] **Step 9: Commit**

```bash
git add app/\(dashboard\)/industry-news lib/navigation.ts
git commit -m "Add Industry News admin pages and sidebar navigation"
```

---

## Task 15: Frontend — API types, client, mapper

**Files:**
- Create: `gmr/lib/api/industry-news.types.ts`
- Create: `gmr/lib/api/industry-news.ts`
- Modify: `gmr/lib/api/mappers.ts`
- Modify: `gmr/lib/api/index.ts`

**Interfaces:**
- Consumes: `apiFetch`, `buildQueryString`, `ApiResponse`, `PaginationMeta` from `./config` (existing); `ApiCategory` from `./categories.types`; `ApiAuthor` from `./common.types`.
- Produces: `ApiIndustryNews`, `IndustryNews`, `IndustryNewsFilters`, `IndustryNewsListData`, `IndustryNewsDetailData` types; `getIndustryNewsList`, `getIndustryNewsBySlug`, `searchIndustryNews` functions; `mapApiIndustryNewsToIndustryNews`, `mapApiIndustryNewsListToIndustryNewsList` mapper functions — used by Task 16 (components) and Task 17 (pages).

- [ ] **Step 1: Write `lib/api/industry-news.types.ts`**

Copy of `press-releases.types.ts` with the `reportUrl` field removed (Industry News has no report-linking feature) and `PressRelease*` renamed to `IndustryNews*`:

```typescript
// Industry News types for API integration
import type { ApiCategory } from './categories.types';
import type { ApiAuthor } from './common.types';

export type IndustryNewsStatus = 'draft' | 'review' | 'published';

export interface ApiIndustryNewsMetadata {
  metaTitle?: string;
  metaDescription?: string;
  keywords?: string[];
  [key: string]: string | string[] | undefined;
}

export interface ApiIndustryNews {
  id: number;
  slug: string;
  title: string;
  excerpt: string;
  content: string;
  categoryId?: number;
  authorId?: number;
  author?: ApiAuthor;
  category?: ApiCategory;
  tags?: string;
  status: IndustryNewsStatus;
  publishDate?: string | null;
  scheduledPublishEnabled?: boolean;
  location?: string;
  metadata?: ApiIndustryNewsMetadata;
  createdAt: string;
  updatedAt: string;
  reviewedAt?: string | null;
  reviewedBy?: number | null;
}

/**
 * UI Industry News interface (used by components)
 */
export interface IndustryNews {
  id: number;
  slug: string;
  title: string;
  excerpt: string;
  category: string;
  author: string;
  date: string;
  readTime: string;
  content: string;

  tags?: string[];
  location?: string;

  authorId?: number;
  categoryId?: number;
  authorDetails?: ApiAuthor;
  categoryDetails?: ApiCategory;

  metadata?: {
    metaTitle?: string;
    metaDescription?: string;
    keywords?: string[];
  };

  status?: IndustryNewsStatus;
  publishDate?: string | null;
  scheduledPublishEnabled?: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface IndustryNewsFilters {
  page?: number;
  limit?: number;
  status?: IndustryNewsStatus;
  category?: string;
  categoryId?: number;
  authorId?: number;
  search?: string;
  sort_by?: string;
}

export interface IndustryNewsListData {
  industryNews: ApiIndustryNews[];
  page: number;
  limit: number;
  total: number;
  totalPages: number;
}

export interface IndustryNewsDetailData {
  industryNews: ApiIndustryNews;
}
```

- [ ] **Step 2: Write `lib/api/industry-news.ts`**

Copy of `press-releases.ts` with `press-releases`→`industry-news` routes/types and the `reportUrl` field dropped:

```typescript
// Industry News API client functions

import { apiFetch, buildQueryString, type ApiResponse, type PaginationMeta } from './config';
import type {
  IndustryNewsFilters,
  IndustryNewsListData,
  IndustryNewsDetailData,
  IndustryNews,
  ApiIndustryNews,
} from './industry-news.types';
import { mapApiIndustryNewsListToIndustryNewsList, mapApiIndustryNewsToIndustryNews } from './mappers';

export async function getIndustryNewsList(
  filters?: IndustryNewsFilters
): Promise<ApiResponse<IndustryNews[]>> {
  const params: Record<string, string | number | boolean | undefined> = {
    page: filters?.page || 1,
    limit: filters?.limit || 100,
    status: filters?.status || 'published',
    ...(filters?.category && { category: filters.category }),
    ...(filters?.categoryId && { categoryId: filters.categoryId }),
    ...(filters?.authorId && { authorId: filters.authorId }),
    ...(filters?.search && { search: filters.search }),
    ...(filters?.sort_by && { sort_by: filters.sort_by }),
  };

  const queryString = buildQueryString(params);
  const response = await apiFetch<IndustryNewsListData>(`/api/v1/industry-news${queryString}`);

  if (!response.success) {
    return response;
  }

  let apiIndustryNews: ApiIndustryNews[];

  if (Array.isArray(response.data)) {
    apiIndustryNews = response.data;
  } else if (response.data && typeof response.data === 'object' && 'industryNews' in response.data) {
    apiIndustryNews = (response.data as { industryNews: ApiIndustryNews[] }).industryNews;
  } else {
    console.error('Unexpected response structure:', response.data);
    return {
      success: false,
      error: 'invalid_response',
      message: 'API returned unexpected response structure',
    };
  }

  const mappedIndustryNews = mapApiIndustryNewsListToIndustryNewsList(apiIndustryNews);

  const rawData = response.data as { total?: number; page?: number; limit?: number; totalPages?: number };
  const mappedMeta: PaginationMeta | undefined = rawData.total !== undefined
    ? {
        currentPage: rawData.page ?? 1,
        totalPages: rawData.totalPages ?? 1,
        totalItems: Number(rawData.total ?? 0),
        itemsPerPage: rawData.limit ?? 10,
        hasNextPage: (rawData.page ?? 1) < (rawData.totalPages ?? 1),
        hasPreviousPage: (rawData.page ?? 1) > 1,
      }
    : undefined;

  return {
    success: true,
    data: mappedIndustryNews,
    meta: mappedMeta,
  };
}

export async function getIndustryNewsBySlug(slug: string): Promise<ApiResponse<IndustryNews>> {
  const response = await apiFetch<IndustryNewsDetailData>(`/api/v1/industry-news/slug/${slug}`);

  if (!response.success) {
    return response;
  }

  let apiIndustryNewsItem: ApiIndustryNews;

  if (response.data && typeof response.data === 'object') {
    if ('industryNews' in response.data) {
      apiIndustryNewsItem = (response.data as { industryNews: ApiIndustryNews }).industryNews;
    } else {
      apiIndustryNewsItem = response.data;
    }
  } else {
    console.error('Unexpected response structure for getIndustryNewsBySlug:', response.data);
    return {
      success: false,
      error: 'invalid_response',
      message: 'API returned unexpected response structure',
    };
  }

  const mappedIndustryNews = mapApiIndustryNewsToIndustryNews(apiIndustryNewsItem);

  return {
    success: true,
    data: mappedIndustryNews,
  };
}

export async function searchIndustryNews(
  query: string,
  page: number = 1,
  limit: number = 50
): Promise<ApiResponse<IndustryNews[]>> {
  return getIndustryNewsList({
    search: query,
    page,
    limit,
    status: 'published',
  });
}
```

- [ ] **Step 3: Add mapper functions to `lib/api/mappers.ts`**

Add the import at the top of the file, alongside the existing `ApiPressRelease, PressRelease` import:

```typescript
import type { ApiIndustryNews, IndustryNews } from './industry-news.types';
```

Then append these two functions at the end of the file (after `mapApiPressReleasesToPressReleases`), mirroring the press-release mapper exactly:

```typescript
/**
 * Maps API industry news response to UI IndustryNews interface
 *
 * @param apiIndustryNews - Industry news data from API
 * @returns IndustryNews formatted for UI components
 */
export function mapApiIndustryNewsToIndustryNews(apiIndustryNews: ApiIndustryNews): IndustryNews {
  const category = apiIndustryNews.category?.name || 'Industry News';
  const author = apiIndustryNews.author?.name || 'Globe Market Research';
  const date = formatDate(apiIndustryNews.publishDate || apiIndustryNews.createdAt);
  const readTime = calculateReadTime(apiIndustryNews.content);
  const tags = parseTags(apiIndustryNews.tags);

  return {
    id: apiIndustryNews.id,
    slug: apiIndustryNews.slug,
    title: apiIndustryNews.title,
    excerpt: apiIndustryNews.excerpt,
    category,
    author,
    date,
    readTime,
    content: apiIndustryNews.content,
    tags,
    location: apiIndustryNews.location,

    authorId: apiIndustryNews.authorId,
    categoryId: apiIndustryNews.categoryId,
    authorDetails: apiIndustryNews.author,
    categoryDetails: apiIndustryNews.category,

    metadata: apiIndustryNews.metadata,

    status: apiIndustryNews.status,
    publishDate: apiIndustryNews.publishDate,
    scheduledPublishEnabled: apiIndustryNews.scheduledPublishEnabled,
    createdAt: apiIndustryNews.createdAt,
    updatedAt: apiIndustryNews.updatedAt,
  };
}

/**
 * Maps array of API industry news to UI IndustryNews array
 *
 * @param apiIndustryNewsList - Array of industry news from API
 * @returns Array of industry news formatted for UI
 */
export function mapApiIndustryNewsListToIndustryNewsList(apiIndustryNewsList: ApiIndustryNews[]): IndustryNews[] {
  if (!apiIndustryNewsList || !Array.isArray(apiIndustryNewsList)) {
    console.error('mapApiIndustryNewsListToIndustryNewsList received invalid input:', apiIndustryNewsList);
    return [];
  }

  return apiIndustryNewsList.map(mapApiIndustryNewsToIndustryNews);
}
```

Note the default author fallback is `'Globe Market Research'` (not `'Media Relations'` as in the press-release mapper) — this is the display-layer safety net described in the design spec; in practice `apiIndustryNews.author?.name` will always be populated because the backend always returns the joined `Author` record (seeded in Task 1), so this fallback should rarely if ever trigger.

- [ ] **Step 4: Export from `lib/api/index.ts`**

Add, alongside the existing `export * from './press-releases';` line:

```typescript
export * from './industry-news';
export * from './industry-news.types';
```

- [ ] **Step 5: Verify it builds**

Run: `cd gmr && npm run build`
Expected: no TypeScript errors (this app has no separate type-check script — `next build` performs the type check).

- [ ] **Step 6: Commit**

```bash
git add lib/api/industry-news.types.ts lib/api/industry-news.ts lib/api/mappers.ts lib/api/index.ts
git commit -m "Add Industry News frontend API client, types, and mappers"
```

---

## Task 16: Frontend — components (cards, listing client, related section)

**Files:**
- Create: `gmr/components/industry-news/IndustryNewsCard.tsx`
- Create: `gmr/components/industry-news/IndustryNewsListCard.tsx`
- Create: `gmr/components/industry-news/IndustryNewsListingClient.tsx`
- Create: `gmr/components/industry-news/RelatedIndustryNewsSection.tsx`

**Interfaces:**
- Consumes: `IndustryNews` type, `getIndustryNewsList` (Task 15); `Pagination` (`components/reports/Pagination`, existing), `FilterSidebar` (`components/reports/FilterSidebar`, existing).
- Produces: `IndustryNewsCard`, `IndustryNewsListCard` (default export), `IndustryNewsListingClient` (default export), `RelatedIndustryNewsSection` (default export) — used by Task 17 (pages).

- [ ] **Step 1: Write `IndustryNewsCard.tsx`**

```tsx
import Link from "next/link";
import { Card, CardHeader, CardTitle, CardDescription, CardFooter, Badge } from "@/components/ui";

interface IndustryNewsCardProps {
  slug: string;
  title: string;
  excerpt: string;
  category: string;
  author: string;
  date: string;
  readTime: string;
  location?: string;
}

export function IndustryNewsCard({
  slug,
  title,
  excerpt,
  category,
  author,
  date,
  readTime,
  location,
}: IndustryNewsCardProps) {
  return (
    <Link href={`/industry-news/${slug}`} className="group">
      <Card className="h-full hover:shadow-primary-lg hover:border-ocean-500 transition-all duration-300 hover:-translate-y-1">
        <CardHeader>
          <div className="mb-3">
            <Badge variant="default" className="bg-gradient-to-r from-slate-100 to-ocean-50 text-ocean-700 border border-ocean-200 shadow-sm">{category}</Badge>
          </div>
          <CardTitle className="line-clamp-2 group-hover:text-ocean-700 transition-colors">{title}</CardTitle>
          <CardDescription className="line-clamp-3 mt-2 text-slate-600">
            {excerpt}
          </CardDescription>
        </CardHeader>
        <CardFooter className="flex items-center justify-between text-sm text-slate-500 border-t border-slate-200 pt-4">
          <span className="font-medium text-slate-700">{author}</span>
          <div className="flex items-center gap-2">
            <span>📖 {readTime}</span>
            <span>•</span>
            <time>📅 {date}</time>
            {location && (
              <>
                <span>•</span>
                <span>📍 {location}</span>
              </>
            )}
          </div>
        </CardFooter>
      </Card>
    </Link>
  );
}
```

- [ ] **Step 2: Write `IndustryNewsListCard.tsx`**

```tsx
import Link from 'next/link';
import type { IndustryNews } from '@/lib/api/industry-news.types';

interface IndustryNewsListCardProps {
  industryNews: IndustryNews;
}

export default function IndustryNewsListCard({ industryNews }: IndustryNewsListCardProps) {
  let formattedDate = '';
  try {
    if (industryNews.date) {
      const d = new Date(industryNews.date);
      if (!isNaN(d.getTime())) {
        formattedDate = d.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
      }
    }
  } catch {
    formattedDate = industryNews.date || '';
  }

  const initials = industryNews.author
    ? industryNews.author.split(' ').map((n: string) => n[0]).join('').slice(0, 2).toUpperCase()
    : 'GM';

  return (
    <Link href={`/industry-news/${industryNews.slug}`} className="group block mb-3">
      <article
        className="relative rounded-xl px-5 py-5 transition-all duration-200 border"
        style={{
          background: 'var(--surface-raised)',
          borderColor: 'var(--border-color)',
          boxShadow: 'var(--shadow-card)',
        }}
        onMouseEnter={(e) => {
          const el = e.currentTarget;
          el.style.borderColor = 'var(--accent)';
          el.style.boxShadow = 'var(--shadow-card-hover)';
          el.style.transform = 'translateY(-1px)';
        }}
        onMouseLeave={(e) => {
          const el = e.currentTarget;
          el.style.borderColor = 'var(--border-color)';
          el.style.boxShadow = 'var(--shadow-card)';
          el.style.transform = 'translateY(0)';
        }}
      >
        <div className="flex items-center justify-between gap-3 mb-3">
          <span
            className="text-[10px] font-bold uppercase tracking-widest px-2.5 py-0.5 rounded-full"
            style={{ background: 'var(--accent-muted)', color: 'var(--accent)' }}
          >
            {industryNews.category || 'Industry News'}
          </span>
          {formattedDate && (
            <time className="text-xs shrink-0" style={{ color: 'var(--text-tertiary)' }}>{formattedDate}</time>
          )}
        </div>

        <h3
          className="text-[16px] font-bold leading-snug mb-2 transition-colors duration-200 group-hover:text-[var(--accent)]"
          style={{ color: 'var(--text-primary)' }}
        >
          {industryNews.title}
        </h3>

        <p className="text-sm leading-relaxed line-clamp-2 mb-4" style={{ color: 'var(--text-secondary)' }}>
          {industryNews.excerpt}
        </p>

        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2.5">
            <div
              className="w-6 h-6 rounded-full flex items-center justify-center text-[9px] font-bold text-white shrink-0"
              style={{ background: 'var(--accent)' }}
            >
              {initials}
            </div>
            <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>
              {industryNews.author}
            </span>
            {industryNews.readTime && (
              <>
                <span style={{ color: 'var(--border-color)' }} className="text-xs">·</span>
                <span className="text-xs" style={{ color: 'var(--text-tertiary)' }}>{industryNews.readTime}</span>
              </>
            )}
          </div>

          <span
            className="text-xs font-semibold flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-all duration-200 translate-x-1 group-hover:translate-x-0 shrink-0"
            style={{ color: 'var(--accent)' }}
          >
            Read More
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </span>
        </div>
      </article>
    </Link>
  );
}
```

- [ ] **Step 3: Write `IndustryNewsListingClient.tsx`**

Copy of `PressReleaseListingClient.tsx` renamed, using a distinct hero icon/emoji (📰 for Industry News vs 📢 for Press Releases) and image asset:

```tsx
'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import type { IndustryNews } from '@/lib/api/industry-news.types';
import IndustryNewsListCard from './IndustryNewsListCard';
import Pagination from '@/components/reports/Pagination';
import FilterSidebar from '@/components/reports/FilterSidebar';
import { getIndustryNewsList, isApiError } from '@/lib/api';

const ITEMS_PER_PAGE = 8;

interface IndustryNewsListingClientProps {
  industryNewsList: IndustryNews[];
  totalItems: number;
  totalPages: number;
}

export default function IndustryNewsListingClient({
  industryNewsList: initialIndustryNewsList,
  totalItems: initialTotalItems,
  totalPages: initialTotalPages,
}: IndustryNewsListingClientProps) {
  const storageKey = 'industry_news_page';
  const [industryNewsList, setIndustryNewsList] = useState<IndustryNews[]>(initialIndustryNewsList);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(initialTotalPages);
  const [, setTotalItems] = useState(initialTotalItems);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    const saved = sessionStorage.getItem(storageKey);
    const savedPage = saved ? Math.max(1, parseInt(saved, 10) || 1) : 1;
    if (savedPage !== 1) {
      setCurrentPage(savedPage);
      fetchPage(savedPage);
    }
  }, [storageKey]);

  async function fetchPage(page: number) {
    setIsLoading(true);
    const response = await getIndustryNewsList({
      status: 'published',
      page,
      limit: ITEMS_PER_PAGE,
      sort_by: 'publish_date_desc',
    });
    if (!isApiError(response)) {
      setIndustryNewsList(response.data);
      if (response.meta) {
        setTotalPages(response.meta.totalPages);
        setTotalItems(response.meta.totalItems);
      }
    }
    setIsLoading(false);
  }

  const handlePageChange = async (page: number) => {
    setCurrentPage(page);
    sessionStorage.setItem(storageKey, String(page));
    await fetchPage(page);
    document.getElementById('industry-news-list')?.scrollIntoView({ behavior: 'smooth' });
  };

  return (
    <>
      <div
        className="relative overflow-hidden border-b border-[var(--border-color)]"
        style={{ background: 'var(--featured-bg)' }}
      >
        <Image
          src="/assets/other/PressReleases.png"
          alt=""
          fill
          className="object-cover object-center"
          aria-hidden
          priority
        />
        <div aria-hidden="true" className="pointer-events-none absolute inset-0" style={{ background: 'linear-gradient(to bottom, rgba(3,26,61,0.35) 0%, rgba(3,26,61,0.55) 100%)' }} />
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0"
          style={{
            backgroundImage: `radial-gradient(circle, rgba(2,132,199,0.18) 1px, transparent 1px)`,
            backgroundSize: '28px 28px',
            maskImage: 'linear-gradient(to bottom, transparent, black 30%, black 70%, transparent)',
          }}
        />
        <div
          aria-hidden="true"
          className="pointer-events-none absolute -top-24 -right-24 w-96 h-96 rounded-full opacity-10"
          style={{ background: 'radial-gradient(circle, #0284c7, transparent 70%)' }}
        />
        <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16 lg:py-24">
          <nav aria-label="Breadcrumb" className="flex items-center gap-1.5 text-xs mb-6" style={{ color: 'rgba(255,255,255,0.45)' }}>
            <Link href="/" className="hover:text-white transition-colors">Home</Link>
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
            <span style={{ color: 'rgba(255,255,255,0.7)' }}>Industry News</span>
          </nav>

          <div className="flex items-start gap-5">
            <div
              aria-hidden="true"
              className="hidden sm:flex items-center justify-center w-16 h-16 rounded-2xl text-3xl shrink-0 mt-0.5"
              style={{
                background: 'rgba(2,132,199,0.15)',
                border: '1px solid rgba(2,132,199,0.3)',
              }}
            >
              📰
            </div>
            <div>
              <span
                className="inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-widest px-2.5 py-1 rounded-full mb-3"
                style={{ background: 'rgba(2,132,199,0.2)', color: '#7dd3fc', border: '1px solid rgba(2,132,199,0.3)' }}
              >
                <svg className="w-2.5 h-2.5" viewBox="0 0 10 10" fill="currentColor"><circle cx="5" cy="5" r="3"/></svg>
                Industry Updates
              </span>
              <h1 className="text-3xl sm:text-4xl font-bold leading-tight mb-3" style={{ color: '#fff', letterSpacing: '-0.03em' }}>
                Industry News
              </h1>
              <p className="text-sm sm:text-[15px] leading-relaxed max-w-2xl" style={{ color: 'rgba(255,255,255,0.6)' }}>
                Curated healthcare and market research industry news from Globe Market Research. Stay ahead of the trends shaping the market.
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid lg:grid-cols-[1fr_288px] gap-10">
          <main id="industry-news-list">
            {isLoading ? (
              <div className="space-y-4 mt-4">
                {Array.from({ length: ITEMS_PER_PAGE }).map((_, i) => (
                  <div key={i} className="h-32 bg-[var(--surface)] animate-pulse rounded-xl" />
                ))}
              </div>
            ) : industryNewsList.length > 0 ? (
              <>
                <div>
                  {industryNewsList.map((item) => (
                    <IndustryNewsListCard key={item.id} industryNews={item} />
                  ))}
                </div>
                <Pagination currentPage={currentPage} totalPages={totalPages} onPageChange={handlePageChange} />
              </>
            ) : (
              <div className="text-center py-20 border border-dashed border-[var(--border-color)] rounded-xl mt-4">
                <div className="text-5xl mb-4">📰</div>
                <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-2">No industry news found</h3>
                <p className="text-sm text-[var(--text-tertiary)]">Check back later for new updates</p>
              </div>
            )}
          </main>
          <aside className="hidden lg:block">
            <div className="sticky top-24">
              <FilterSidebar
                filters={{ industries: [], regions: [], reportTypes: [], priceRanges: [] }}
                onFilterChange={() => {}}
                totalCount={0}
              />
            </div>
          </aside>
        </div>
      </div>
    </>
  );
}
```

`/assets/other/PressReleases.png` is reused as-is for the hero background (no new image asset requested in the design spec); replace it later with a dedicated Industry News image if desired.

- [ ] **Step 4: Write `RelatedIndustryNewsSection.tsx`**

```tsx
import Link from 'next/link';
import { getIndustryNewsList, isApiError } from '@/lib/api';
import type { IndustryNews } from '@/lib/api/industry-news.types';

interface RelatedIndustryNewsSectionProps {
  categorySlug: string;
  categoryName: string;
  currentSlug: string;
}

function RelatedIndustryNewsCard({ industryNews }: { industryNews: IndustryNews }) {
  let formattedDate = '';
  try {
    if (industryNews.date) {
      const d = new Date(industryNews.date);
      if (!isNaN(d.getTime())) {
        formattedDate = d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
      }
    }
  } catch {
    formattedDate = '';
  }

  return (
    <Link href={`/industry-news/${industryNews.slug}`} className="group block h-full">
      <article
        className="relative h-full flex flex-col rounded-xl border transition-all duration-200"
        style={{
          background: 'var(--surface-raised)',
          borderColor: 'var(--border-color)',
          boxShadow: 'var(--shadow-card)',
        }}
      >
        <div
          className="h-[3px] w-full rounded-t-xl transition-all duration-300"
          style={{ background: 'var(--border-color)' }}
        />

        <div className="flex flex-col flex-1 px-5 py-4 gap-3">
          <div className="flex items-center justify-between gap-2">
            <span
              className="text-[10px] font-bold uppercase tracking-widest px-2.5 py-0.5 rounded-full leading-none"
              style={{ background: 'var(--accent-muted)', color: 'var(--accent)' }}
            >
              {industryNews.category}
            </span>
            {formattedDate && (
              <time className="text-[11px] shrink-0" style={{ color: 'var(--text-tertiary)' }}>
                {formattedDate}
              </time>
            )}
          </div>

          <h3
            className="text-[15px] font-bold leading-snug transition-colors duration-200 line-clamp-3"
            style={{ color: 'var(--text-primary)' }}
          >
            <span className="group-hover:text-[var(--accent)] transition-colors duration-200">
              {industryNews.title}
            </span>
          </h3>

          {industryNews.excerpt && (
            <p
              className="text-[13px] leading-relaxed line-clamp-2"
              style={{ color: 'var(--text-secondary)' }}
            >
              {industryNews.excerpt}
            </p>
          )}

          <div className="flex-1" />

          <div
            className="flex items-center justify-between gap-2 pt-3 mt-auto"
            style={{ borderTop: '1px solid var(--border-color)' }}
          >
            <div className="flex items-center gap-2 text-[11px]" style={{ color: 'var(--text-tertiary)' }}>
              {industryNews.author && (
                <span className="truncate max-w-[120px]">{industryNews.author}</span>
              )}
              {industryNews.readTime && (
                <>
                  <span style={{ color: 'var(--border-color)' }}>·</span>
                  <span>{industryNews.readTime}</span>
                </>
              )}
            </div>

            <span
              className="flex items-center gap-1 text-[11px] font-semibold translate-x-1 opacity-0 group-hover:opacity-100 group-hover:translate-x-0 transition-all duration-200"
              style={{ color: 'var(--accent)' }}
            >
              View
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </span>
          </div>
        </div>
      </article>
    </Link>
  );
}

export default async function RelatedIndustryNewsSection({
  categorySlug,
  categoryName,
  currentSlug,
}: RelatedIndustryNewsSectionProps) {
  const response = await getIndustryNewsList({
    category: categorySlug,
    status: 'published',
    limit: 9,
  });

  let related: IndustryNews[] = [];

  if (!isApiError(response)) {
    related = response.data.filter((item) => item.slug !== currentSlug).slice(0, 4);
  }

  if (related.length === 0) {
    const fallback = await getIndustryNewsList({ status: 'published', limit: 10 });
    if (!isApiError(fallback)) {
      related = fallback.data.filter((item) => item.slug !== currentSlug).slice(0, 4);
    }
  }

  if (related.length === 0) return null;

  return (
    <section className="mt-14 mb-8" aria-labelledby="related-industry-news-heading">
      <div className="flex items-center justify-between gap-4 mb-6">
        <div className="flex items-center gap-3">
          <div
            className="w-1 h-6 rounded-full shrink-0"
            style={{ background: 'var(--accent)' }}
          />
          <div>
            <h2
              id="related-industry-news-heading"
              className="text-xl font-bold leading-tight"
              style={{ color: 'var(--text-primary)', fontFamily: 'var(--font-heading)' }}
            >
              Related Industry News
            </h2>
            <p className="text-xs mt-0.5" style={{ color: 'var(--text-tertiary)' }}>
              More in {categoryName}
            </p>
          </div>
        </div>

        <Link
          href="/industry-news"
          className="flex items-center gap-1.5 text-xs font-semibold shrink-0 transition-colors duration-150"
          style={{ color: 'var(--accent)' }}
        >
          View all
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </Link>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {related.map((item) => (
          <RelatedIndustryNewsCard key={item.slug} industryNews={item} />
        ))}
      </div>
    </section>
  );
}
```

- [ ] **Step 5: Verify it builds**

Run: `cd gmr && npm run build`
Expected: no TypeScript errors (these components are not yet imported anywhere until Task 17, so this mainly confirms self-consistency).

- [ ] **Step 6: Commit**

```bash
git add components/industry-news
git commit -m "Add Industry News frontend components"
```

---

## Task 17: Frontend — pages (list + detail)

**Files:**
- Create: `gmr/app/industry-news/page.tsx`
- Create: `gmr/app/industry-news/loading.tsx`
- Create: `gmr/app/industry-news/[slug]/page.tsx`
- Create: `gmr/app/industry-news/[slug]/loading.tsx`

**Interfaces:**
- Consumes: `getIndustryNewsList`, `getIndustryNewsBySlug` (Task 15), `IndustryNewsListingClient`, `RelatedIndustryNewsSection` (Task 16); `getReportsByAuthorId`, `isApiError` (existing, `lib/api`); `AuthorHoverCard` (existing, `components/authors/AuthorHoverCard.tsx`); `StyledReportContent`, `ArticleContentWrapper`, `StructuredData`, `generateArticleSchema`, `generateBreadcrumbSchema`, `TrustedPartnersSidebar`, `ShareButtons`, `GooglePreferredSource` (all existing, unchanged).

- [ ] **Step 1: Write `app/industry-news/page.tsx`**

```tsx
import { Suspense } from "react";
import IndustryNewsListingClient from "@/components/industry-news/IndustryNewsListingClient";
import { getIndustryNewsList, isApiError } from "@/lib/api";
import IndustryNewsLoading from "./loading";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Industry News",
  description: "Stay updated with the latest industry news, market trends, and healthcare developments from Globe Market Research.",
  keywords: ["industry news", "healthcare news", "market trends", "market updates"],
  alternates: {
    canonical: '/industry-news',
  },
};

export const revalidate = 300;

async function IndustryNewsContent() {
  const response = await getIndustryNewsList({
    status: 'published',
    page: 1,
    limit: 8,
    sort_by: 'publish_date_desc',
  });

  if (isApiError(response)) {
    console.error('Failed to fetch industry news:', response.message);
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center space-y-4">
          <h2 className="text-2xl font-bold text-gray-900">Unable to Load Industry News</h2>
          <p className="text-gray-600">{response.message}</p>
        </div>
      </div>
    );
  }

  const totalItems = response.meta?.totalItems ?? response.data.length;
  const totalPages = response.meta?.totalPages ?? 1;

  return (
    <IndustryNewsListingClient
      industryNewsList={response.data}
      totalItems={totalItems}
      totalPages={totalPages}
    />
  );
}

export default function IndustryNewsPage() {
  return (
    <Suspense fallback={<IndustryNewsLoading />}>
      <IndustryNewsContent />
    </Suspense>
  );
}
```

- [ ] **Step 2: Write `app/industry-news/loading.tsx`**

Direct copy of `app/press-releases/loading.tsx` renamed:

```tsx
import { Skeleton } from "@/components/ui";

export default function IndustryNewsLoading() {
  return (
    <>
      <div className="bg-gradient-to-r from-slate-50 via-blue-50/40 to-slate-50 border-b border-slate-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 lg:py-10">
          <div className="flex items-center gap-2 mb-5">
            <Skeleton className="h-3 w-10" />
            <Skeleton className="h-3 w-2" />
            <Skeleton className="h-3 w-24" />
          </div>

          <div className="flex items-start gap-4">
            <Skeleton className="hidden sm:block h-14 w-14 rounded-2xl shrink-0" />
            <div className="flex-1">
              <Skeleton className="h-8 w-56 mb-3" />
              <Skeleton className="h-4 w-full max-w-lg mb-1.5" />
              <Skeleton className="h-4 w-2/3 max-w-md mb-4" />
              <Skeleton className="h-7 w-28 rounded-full" />
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <main>
          <div className="flex items-center pb-3 border-b border-slate-200 mb-1">
            <Skeleton className="h-3 w-36" />
          </div>

          {Array.from({ length: 7 }).map((_, i) => (
            <div key={i} className="py-6 pl-5 -ml-5 border-b border-slate-100">
              <div className="flex items-center gap-2.5 mb-2.5">
                <Skeleton className="h-3 w-28" />
                <Skeleton className="h-3 w-2 rounded-full" />
                <Skeleton className="h-3 w-16" />
              </div>
              <Skeleton className="h-5 w-full mb-1.5" />
              <Skeleton className={`h-5 mb-3 ${i % 3 === 0 ? 'w-3/4' : i % 3 === 1 ? 'w-5/6' : 'w-2/3'}`} />
              <Skeleton className="h-3.5 w-full mb-1.5" />
              <Skeleton className="h-3.5 w-4/5 mb-3" />
              <div className="flex items-center gap-3">
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-3 w-1" />
                <Skeleton className="h-3 w-16" />
              </div>
            </div>
          ))}
        </main>
      </div>
    </>
  );
}
```

- [ ] **Step 3: Write `app/industry-news/[slug]/page.tsx`**

Copy of `app/press-release/[slug]/page.tsx` with the `reportUrl`/related-report sidebar CTA removed (Industry News has no `reportUrl` field) and routes updated to `/industry-news`:

```tsx
import { notFound } from "next/navigation";
import Link from "next/link";
import { Section, Container, Card, CardContent } from "@/components/ui";
import { StyledReportContent } from "@/components/reports/StyledReportContent";
import { ArticleContentWrapper } from "@/components/shared/ArticleContentWrapper";
import { getIndustryNewsList, getIndustryNewsBySlug, getReportsByAuthorId, isApiError } from "@/lib/api";
import type { Metadata } from "next";
import { StructuredData, generateArticleSchema, generateBreadcrumbSchema } from "@/components/seo/StructuredData";
import { TrustedPartnersSidebar } from "@/components/contact";
import { ShareButtons } from "@/components/shared/ShareButtons";
import RelatedIndustryNewsSection from "@/components/industry-news/RelatedIndustryNewsSection";
import GooglePreferredSource from "@/components/reports/GooglePreferredSource";
import AuthorHoverCard from "@/components/authors/AuthorHoverCard";

export const revalidate = 300;

interface IndustryNewsPageProps {
  params: Promise<{
    slug: string;
  }>;
}

export async function generateStaticParams() {
  const response = await getIndustryNewsList({ status: 'published', limit: 100 });

  if (isApiError(response)) {
    console.error('Failed to fetch industry news for static params:', response.message);
    return [];
  }

  return response.data.map((item) => ({
    slug: item.slug,
  }));
}

export async function generateMetadata({ params }: IndustryNewsPageProps): Promise<Metadata> {
  try {
    const { slug } = await params;
    const response = await getIndustryNewsBySlug(slug);

    if (isApiError(response)) {
      return {
        title: "Industry News Not Found",
      };
    }

    const industryNews = response.data;

    const title = industryNews.metadata?.metaTitle || industryNews.title;
    const description = industryNews.metadata?.metaDescription || industryNews.excerpt;
    const keywords = industryNews.metadata?.keywords || ["healthcare industry news", "market trends", "industry announcements", "healthcare market updates"];

    return {
      title: { absolute: title },
      description,
      keywords,
      openGraph: {
        title: industryNews.metadata?.metaTitle || industryNews.title,
        description,
        type: "article",
        publishedTime: industryNews.publishDate || industryNews.createdAt,
        authors: industryNews.authorDetails ? [industryNews.authorDetails.name] : [industryNews.author],
      },
      twitter: {
        card: 'summary_large_image',
        title,
        description,
      },
      alternates: {
        canonical: `/industry-news/${slug}`,
      },
    };
  } catch {
    return { title: "Industry News Not Found" };
  }
}

export default async function IndustryNewsDetailPage({ params }: IndustryNewsPageProps) {
  const { slug } = await params;

  let response;
  try {
    response = await getIndustryNewsBySlug(slug);
  } catch {
    notFound();
  }

  if (isApiError(response)) {
    notFound();
  }

  const industryNews = response.data;

  const authorReportsResponse = industryNews.authorDetails
    ? await getReportsByAuthorId(industryNews.authorDetails.id, { status: 'published', limit: 1000 })
    : null;
  const authorReports =
    authorReportsResponse && !isApiError(authorReportsResponse) ? authorReportsResponse.data : [];
  const authorArticleCount = authorReports.length;
  const authorLatestPosts = authorReports
    .filter((r) => r.slug !== industryNews.slug)
    .slice(0, 4)
    .map((r) => ({ title: r.title, slug: r.slug, href: `/reports/${r.slug}` }));

  const articleSchema = generateArticleSchema({
    type: 'NewsArticle',
    title: industryNews.title,
    description: industryNews.excerpt,
    url: `https://www.globemarketresearch.com/industry-news/${industryNews.slug}`,
    datePublished: industryNews.publishDate || industryNews.createdAt || industryNews.date,
    dateModified: industryNews.updatedAt,
    author: industryNews.authorDetails?.name || industryNews.author,
    keywords: industryNews.metadata?.keywords,
  });

  const breadcrumbSchema = generateBreadcrumbSchema([
    { name: 'Home', url: 'https://www.globemarketresearch.com' },
    { name: 'Industry News', url: 'https://www.globemarketresearch.com/industry-news' },
    { name: industryNews.title, url: `https://www.globemarketresearch.com/industry-news/${industryNews.slug}` },
  ]);

  return (
    <>
      <StructuredData data={articleSchema} />
      <StructuredData data={breadcrumbSchema} />
      <div
        className="relative overflow-hidden border-b border-[var(--border-color)]"
        style={{ background: 'var(--featured-bg)' }}
      >
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0"
          style={{
            backgroundImage: `radial-gradient(circle, rgba(2,132,199,0.14) 1px, transparent 1px)`,
            backgroundSize: '28px 28px',
            maskImage: 'linear-gradient(to bottom, transparent, black 20%, black 80%, transparent)',
          }}
        />
        <div
          aria-hidden="true"
          className="pointer-events-none absolute -top-32 -right-32 w-[32rem] h-[32rem] rounded-full opacity-10"
          style={{ background: 'radial-gradient(circle, #0284c7, transparent 70%)' }}
        />
        <div className="relative max-w-[1400px] 2xl:max-w-[1760px] mx-auto px-4 sm:px-6 lg:px-8 py-12 lg:py-16">
          <nav aria-label="Breadcrumb" className="flex items-center gap-1.5 text-xs mb-7" style={{ color: 'rgba(255,255,255,0.4)' }}>
            <Link href="/" className="hover:text-white transition-colors">Home</Link>
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
            <Link href="/industry-news" className="hover:text-white transition-colors">Industry News</Link>
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
            <span className="truncate max-w-[180px]" style={{ color: 'rgba(255,255,255,0.65)' }}>{industryNews.title}</span>
          </nav>

          <div className="flex flex-wrap items-center gap-2 mb-5">
            <Link href={`/industry/${industryNews.category.toLowerCase().replace(/\s+/g, '-')}`}>
              <span
                className="inline-flex text-[10px] font-bold uppercase tracking-widest px-2.5 py-1 rounded-full cursor-pointer transition-all"
                style={{ background: 'rgba(2,132,199,0.2)', color: '#7dd3fc', border: '1px solid rgba(2,132,199,0.35)' }}
              >
                {industryNews.category}
              </span>
            </Link>
            <span
              className="inline-flex text-[10px] font-bold uppercase tracking-widest px-2.5 py-1 rounded-full"
              style={{ background: 'rgba(255,255,255,0.08)', color: 'rgba(255,255,255,0.5)', border: '1px solid rgba(255,255,255,0.12)' }}
            >
              Industry News
            </span>
          </div>

          <h1
            className="mb-5 font-bold leading-tight"
            style={{ color: '#fff', fontSize: 'clamp(1.25rem, 4vw, 2.25rem)', letterSpacing: '-0.03em' }}
          >
            {industryNews.title}
          </h1>

          <p className="mb-8 leading-relaxed" style={{ color: 'rgba(255,255,255,0.65)', fontSize: '1.0625rem' }}>
            {industryNews.excerpt}
          </p>

          <div
            className="flex flex-wrap items-center gap-4 pt-6 text-sm"
            style={{ borderTop: '1px solid rgba(255,255,255,0.1)', color: 'rgba(255,255,255,0.5)' }}
          >
            {industryNews.authorDetails ? (
              <AuthorHoverCard
                author={industryNews.authorDetails}
                articleCount={authorArticleCount}
                latestPosts={authorLatestPosts}
              >
                <div
                  className="w-9 h-9 rounded-full flex items-center justify-center text-xs font-bold text-white shrink-0"
                  style={{ background: 'var(--accent)' }}
                >
                  {industryNews.author.split(' ').map((n: string) => n[0]).join('').slice(0, 2).toUpperCase()}
                </div>
                <span className="font-medium" style={{ color: 'rgba(255,255,255,0.8)' }}>{industryNews.author}</span>
              </AuthorHoverCard>
            ) : (
              <div className="flex items-center gap-2.5">
                <div
                  className="w-9 h-9 rounded-full flex items-center justify-center text-xs font-bold text-white shrink-0"
                  style={{ background: 'var(--accent)' }}
                >
                  {industryNews.author.split(' ').map((n: string) => n[0]).join('').slice(0, 2).toUpperCase()}
                </div>
                <span className="font-medium" style={{ color: 'rgba(255,255,255,0.8)' }}>{industryNews.author}</span>
              </div>
            )}
            {industryNews.date && (
              <>
                <span style={{ color: 'rgba(255,255,255,0.2)' }}>·</span>
                <time style={{ color: 'rgba(255,255,255,0.5)' }}>{industryNews.date}</time>
              </>
            )}
            {industryNews.readTime && (
              <>
                <span style={{ color: 'rgba(255,255,255,0.2)' }}>·</span>
                <span>{industryNews.readTime}</span>
              </>
            )}
            {industryNews.location && (
              <>
                <span style={{ color: 'rgba(255,255,255,0.2)' }}>·</span>
                <span>{industryNews.location}</span>
              </>
            )}
            <div className="ml-auto">
              <ShareButtons
                title={industryNews.title}
                url={`https://www.globemarketresearch.com/industry-news/${industryNews.slug}`}
              />
            </div>
          </div>
        </div>
      </div>

      <Section className="pt-8" container={false}>
        <Container size="xl">
          <ArticleContentWrapper
            sidebar={
              <div className="space-y-6">
                <TrustedPartnersSidebar />
              </div>
            }
          >
            <article>
              <StyledReportContent htmlContent={industryNews.content} />
            </article>

            <GooglePreferredSource />

            <RelatedIndustryNewsSection
              categorySlug={industryNews.category.toLowerCase().replace(/\s+/g, '-')}
              categoryName={industryNews.category}
              currentSlug={industryNews.slug}
            />

            <div className="mt-12 pt-8 border-t border-[var(--border)]">
              <Link
                href="/industry-news"
                className="inline-flex items-center gap-2 text-[var(--primary)] hover:underline font-medium"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="m15 18-6-6 6-6"/>
                </svg>
                View all industry news
              </Link>
            </div>
          </ArticleContentWrapper>
        </Container>
      </Section>
    </>
  );
}
```

Note: `Card`/`CardContent` are imported from `@/components/ui` per the original file's pattern even though this trimmed version (no `reportUrl`/"Request Sample" sidebar block) no longer renders them directly — remove the unused imports if `next build`/ESLint flags them as unused.

- [ ] **Step 4: Write `app/industry-news/[slug]/loading.tsx`**

```tsx
export { default } from "@/app/industry-news/loading";
```

- [ ] **Step 5: Verify it builds**

Run: `cd gmr && npm run build`
Expected: no TypeScript or ESLint errors. `generateStaticParams` will attempt to call the live backend at build time — if the backend isn't running, this may log a fetch failure and fall back to an empty array (same behavior as the existing press-releases page), which is acceptable for a local build check.

- [ ] **Step 6: Manual walkthrough (no test framework in this app)**

Run: `cd gmr && npm run dev`, then in a browser:
1. Visit `http://localhost:3000/industry-news` — hero renders, empty-state or list renders without console errors.
2. If an article was published via the admin UI in Task 14 Step 8, visit its detail page directly and confirm the author card, content, and "Related Industry News" section render.

- [ ] **Step 7: Commit**

```bash
git add app/industry-news
git commit -m "Add Industry News frontend list and detail pages"
```

---

## Task 18: Frontend — navigation, footer, sitemap wiring

**Files:**
- Modify: `gmr/components/layout/Navigation.tsx`
- Modify: `gmr/components/layout/Footer.tsx`
- Modify: `gmr/next.config.ts`
- Create: `gmr/app/api/sitemap-industry-news/[page]/route.ts`
- Modify: `gmr/app/sitemap.ts`

**Interfaces:**
- Consumes: `apiFetch` from `lib/api/config` (existing) — used identically to the press-releases sitemap route.

- [ ] **Step 1: Add the nav link**

In `gmr/components/layout/Navigation.tsx`, add after the existing `<NavLink href="/press-releases" label="Press Releases" pathname={pathname} />` (around line 92):

```tsx
        <NavLink href="/industry-news" label="Industry News" pathname={pathname} />
```

And in the mobile menu items array (around line 163), add `{ name: "Industry News", href: "/industry-news" }` alongside the existing `{ name: "Press Releases", href: "/press-releases" }` entry:

```tsx
                {[...navItems, { name: "Methodology", href: "/research-methodology" }, { name: "Statistics", href: "/statistics" }, { name: "Press Releases", href: "/press-releases" }, { name: "Industry News", href: "/industry-news" }, { name: "Contact", href: "/contact" }].map((item) => {
```

- [ ] **Step 2: Add the footer link**

In `gmr/components/layout/Footer.tsx`, add after `{ href: "/press-releases", label: "Press Releases" }` (around line 18):

```tsx
    { href: "/industry-news", label: "Industry News" },
```

- [ ] **Step 3: Add the sitemap rewrite rule**

In `gmr/next.config.ts`, in the `beforeFiles` array (after the `/sitemap-press-releases-:page.xml` rule, around line 35):

```typescript
        {
          source: '/sitemap-industry-news-:page.xml',
          destination: '/api/sitemap-industry-news/:page',
        },
```

- [ ] **Step 4: Write the sitemap route**

`gmr/app/api/sitemap-industry-news/[page]/route.ts`, copy of `app/api/sitemap-press-releases/[page]/route.ts` renamed:

```typescript
import { NextResponse } from 'next/server';
import { apiFetch } from '@/lib/api/config';

const BASE_URL = 'https://www.globemarketresearch.com';
const ITEMS_PER_SITEMAP = 1000;

interface SitemapItem {
  slug: string;
  updated_at: string;
  publish_date?: string;
}

export async function GET(
  request: Request,
  { params }: { params: Promise<{ page?: string }> }
) {
  const { page: pageParam } = await params;
  const page = parseInt(pageParam ?? '', 10);

  if (isNaN(page) || page < 1) {
    return new NextResponse('Not Found', { status: 404 });
  }

  try {
    const res = await apiFetch<SitemapItem[]>(
      `/api/v1/sitemap/industry-news?page=${page}&limit=${ITEMS_PER_SITEMAP}`
    );

    if (!res.success || !res.data) {
      console.error('Error fetching industry news for sitemap');
      return new NextResponse('Error generating sitemap', { status: 500 });
    }

    if (res.data.length === 0) {
      return new NextResponse('Not Found', { status: 404 });
    }

    const sitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${res.data
  .map((item) => {
    const lastmod = new Date(item.publish_date || item.updated_at).toISOString();
    return `  <url>
    <loc>${BASE_URL}/industry-news/${item.slug}</loc>
    <lastmod>${lastmod}</lastmod>
    <changefreq>daily</changefreq>
    <priority>0.6</priority>
  </url>`;
  })
  .join('\n')}
</urlset>`;

    return new NextResponse(sitemap, {
      headers: {
        'Content-Type': 'application/xml',
        'Cache-Control': 'public, max-age=3600, s-maxage=3600',
      },
    });
  } catch (error) {
    console.error('Error generating industry news sitemap:', error);
    return new NextResponse('Error generating sitemap', { status: 500 });
  }
}
```

- [ ] **Step 5: Wire the sitemap index**

In `gmr/app/sitemap.ts`, update the `getSitemapTotalPages` type union:

```typescript
async function getSitemapTotalPages(type: 'reports' | 'statistics' | 'press-releases' | 'industry-news'): Promise<number> {
```

Add to the `Promise.all` destructure:

```typescript
  const [reportsTotalPages, statisticsTotalPages, prTotalPages, industryNewsTotalPages] = await Promise.all([
    getSitemapTotalPages('reports'),
    getSitemapTotalPages('statistics'),
    getSitemapTotalPages('press-releases'),
    getSitemapTotalPages('industry-news'),
  ]);
```

And after the "Press releases paginated sitemaps" loop, add:

```typescript
  // Industry News paginated sitemaps: sitemap-industry-news-1.xml, sitemap-industry-news-2.xml, ...
  for (let i = 1; i <= industryNewsTotalPages; i++) {
    entries.push({
      url: `${BASE_URL}/sitemap-industry-news-${i}.xml`,
      lastModified: now,
      changeFrequency: 'daily' as const,
      priority: 1.0,
    });
  }
```

Also update the file's top comment block to mention the new sub-sitemap, consistent with the existing style (optional but keeps the doc comment accurate).

- [ ] **Step 6: Verify it builds**

Run: `cd gmr && npm run build`
Expected: no TypeScript errors.

- [ ] **Step 7: Manual walkthrough**

Run: `cd gmr && npm run dev`, then:
1. Confirm "Industry News" appears in the header nav and footer, and the link navigates to `/industry-news`.
2. Visit `http://localhost:3000/sitemap.xml` and confirm it lists `sitemap-industry-news-1.xml` (if the backend returns at least one published article; otherwise `industryNewsTotalPages` is `1` per `getSitemapTotalPages`'s fallback, so the entry still appears — verify it 404s gracefully rather than 500ing when there's no data, matching the press-releases route's behavior).

- [ ] **Step 8: Commit**

```bash
git add components/layout/Navigation.tsx components/layout/Footer.tsx next.config.ts app/api/sitemap-industry-news app/sitemap.ts
git commit -m "Wire Industry News into frontend navigation, footer, and sitemap"
```

---
