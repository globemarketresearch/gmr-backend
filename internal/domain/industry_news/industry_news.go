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
