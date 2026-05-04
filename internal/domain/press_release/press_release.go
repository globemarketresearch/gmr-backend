package press_release

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/author"
	"github.com/healthcare-market-research/backend/internal/domain/category"
)

// PressReleaseStatus represents the status of a press release
type PressReleaseStatus string

const (
	StatusDraft     PressReleaseStatus = "draft"
	StatusReview    PressReleaseStatus = "review"
	StatusPublished PressReleaseStatus = "published"
)

// InternalLinkEntry represents a single internal link configuration
type InternalLinkEntry struct {
	Keyword     string `json:"keyword"`
	TargetID    int    `json:"targetId"`
	TargetTitle string `json:"targetTitle"`
	TargetType  string `json:"targetType"` // "report", "blog", "press-release"
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

// PressReleaseMetadata contains SEO metadata for a press release
type PressReleaseMetadata struct {
	MetaTitle       string   `json:"metaTitle,omitempty"`
	MetaDescription string   `json:"metaDescription,omitempty"`
	Keywords        []string `json:"keywords,omitempty"`
}

// Value implements the driver.Valuer interface for GORM
func (m PressReleaseMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface for GORM
func (m *PressReleaseMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &m)
}

// PressRelease represents a press release in the database
type PressRelease struct {
	ID          uint                   `json:"id" gorm:"primaryKey"`
	Title       string                 `json:"title" gorm:"type:varchar(200);not null"`
	Slug        string                 `json:"slug" gorm:"type:varchar(250);uniqueIndex;not null"`
	Excerpt     string                 `json:"excerpt" gorm:"type:varchar(500);not null"`
	Content     string                 `json:"content" gorm:"type:text;not null"`
	CategoryID  uint                   `json:"categoryId" gorm:"not null;index"`
	Tags        string                 `json:"tags" gorm:"type:varchar(500)"`
	AuthorID    uint                   `json:"authorId" gorm:"not null;index"`
	Author      *author.Author         `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
	Category    *category.Category     `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Status                  PressReleaseStatus `json:"status" gorm:"type:varchar(20);default:'draft';index"`
	PublishDate             *time.Time         `json:"publishDate,omitempty" gorm:"index"`
	ScheduledPublishEnabled bool               `json:"scheduledPublishEnabled" gorm:"default:false"`
	Location                string             `json:"location,omitempty" gorm:"type:varchar(255)"`
	ReportURL               string             `json:"reportUrl,omitempty" gorm:"type:varchar(500)"`
	Metadata      PressReleaseMetadata   `json:"metadata" gorm:"type:jsonb"`
	InternalLinks InternalLinks          `json:"internalLinks,omitempty" gorm:"type:jsonb"`
	ReviewedBy    *uint                  `json:"reviewedBy,omitempty" gorm:"index"`
	ReviewedAt  *time.Time             `json:"reviewedAt,omitempty"`
	DeletedAt   *time.Time             `json:"deletedAt,omitempty" gorm:"index"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// TableName specifies the table name for GORM
func (PressRelease) TableName() string {
	return "press_releases"
}

// CreatePressReleaseRequest is the request body for creating a new press release
type CreatePressReleaseRequest struct {
	Title       string                `json:"title" validate:"required,min=10,max=200"`
	Excerpt     string                `json:"excerpt" validate:"required,min=50,max=500"`
	Content     string                `json:"content" validate:"required,min=100"`
	CategoryID  uint                  `json:"categoryId" validate:"required"`
	Tags        string                `json:"tags"`
	AuthorID    uint                  `json:"authorId" validate:"required"`
	Status      PressReleaseStatus    `json:"status" validate:"required,oneof=draft review published"`
	PublishDate string                `json:"publishDate" validate:"required"`
	Location    string                `json:"location,omitempty"`
	ReportURL   string                `json:"reportUrl,omitempty"`
	Metadata    *PressReleaseMetadata `json:"metadata,omitempty"`
}

// UpdatePressReleaseRequest is the request body for updating a press release
type UpdatePressReleaseRequest struct {
	Title       *string                `json:"title,omitempty" validate:"omitempty,min=10,max=200"`
	Slug        *string                `json:"slug,omitempty" validate:"omitempty,min=1,max=250"`
	Excerpt     *string                `json:"excerpt,omitempty" validate:"omitempty,min=50,max=500"`
	Content     *string                `json:"content,omitempty" validate:"omitempty,min=100"`
	CategoryID  *uint                  `json:"categoryId,omitempty"`
	Tags        *string                `json:"tags,omitempty"`
	AuthorID    *uint                  `json:"authorId,omitempty"`
	Status      *PressReleaseStatus    `json:"status,omitempty" validate:"omitempty,oneof=draft review published"`
	PublishDate *string                `json:"publishDate,omitempty"`
	Location      *string                `json:"location,omitempty"`
	ReportURL     *string                `json:"reportUrl,omitempty"`
	Metadata      *PressReleaseMetadata  `json:"metadata,omitempty"`
	InternalLinks *InternalLinks         `json:"internalLinks,omitempty"`
}

// GetPressReleasesQuery represents query parameters for filtering press releases
type GetPressReleasesQuery struct {
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

// PressReleaseListResponse represents a list of press releases with pagination
type PressReleaseListResponse struct {
	PressReleases []PressRelease `json:"pressReleases"`
	Total         int64          `json:"total"`
	Page          int            `json:"page"`
	Limit         int            `json:"limit"`
	TotalPages    int            `json:"totalPages"`
}

// PressReleaseResponse represents a single press release response
type PressReleaseResponse struct {
	PressRelease PressRelease `json:"pressRelease"`
}
