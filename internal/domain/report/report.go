package report

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/author"
)

// MarketMetrics contains market size and forecast data
type MarketMetrics struct {
	CurrentRevenue   string `json:"currentRevenue,omitempty"`
	CurrentYear      int    `json:"currentYear,omitempty"`
	ForecastRevenue  string `json:"forecastRevenue,omitempty"`
	ForecastYear     int    `json:"forecastYear,omitempty"`
	CAGR             string `json:"cagr,omitempty"`
	CAGRStartYear    int    `json:"cagrStartYear,omitempty"`
	CAGREndYear      int    `json:"cagrEndYear,omitempty"`
}

// Value implements the driver.Valuer interface for GORM
func (m MarketMetrics) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface for GORM
func (m *MarketMetrics) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &m)
}

// KeyPlayer represents a market competitor
type KeyPlayer struct {
	Name        string `json:"name"`
	MarketShare string `json:"marketShare,omitempty"`
	Rank        int    `json:"rank,omitempty"`
	Description string `json:"description,omitempty"`
}

// KeyPlayers is a slice of KeyPlayer with JSON marshaling
type KeyPlayers []KeyPlayer

func (k KeyPlayers) Value() (driver.Value, error) {
	return json.Marshal(k)
}

func (k *KeyPlayers) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &k)
}

// ReportSections contains all report content sections
type ReportSections struct {
	KeyPlayers      string `json:"keyPlayers"`
	MarketDetails   string `json:"marketDetails"`
	TableOfContents string `json:"tableOfContents"`
}

func (s ReportSections) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *ReportSections) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &s)
}

// FAQ represents a frequently asked question
type FAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type FAQs []FAQ

func (f FAQs) Value() (driver.Value, error) {
	return json.Marshal(f)
}

func (f *FAQs) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &f)
}

// StringSlice is a custom type for string arrays
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &s)
}

// UintSlice is a custom type for uint arrays
type UintSlice []uint

func (u UintSlice) Value() (driver.Value, error) {
	return json.Marshal(u)
}

func (u *UintSlice) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &u)
}

// UserReference represents minimal user info for author field
type UserReference struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// LinkSuggestionItem is a lightweight DTO for internal link suggestions
type LinkSuggestionItem struct {
	ID           uint   `json:"id"`
	Title        string `json:"title"`
	Slug         string `json:"slug"`
	MetaKeywords string `json:"meta_keywords"`
}

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

type Report struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	CategoryID      uint            `json:"category_id" gorm:"index;not null"`
	CategoryName     string          `json:"category_name,omitempty" gorm:"->"`
	CategoryImageURL string          `json:"category_image_url,omitempty" gorm:"->"`
	Title           string          `json:"title" gorm:"type:varchar(500);not null"`
	Slug            string          `json:"slug" gorm:"type:varchar(500);uniqueIndex;not null"`
	Description     string          `json:"description" gorm:"type:text"`
	Summary         string          `json:"summary" gorm:"type:text;not null"`

	// Pricing
	Price            float64        `json:"price" gorm:"type:decimal(10,2);default:0"`
	DiscountedPrice  float64        `json:"discounted_price" gorm:"type:decimal(10,2);default:0"`
	Currency         string         `json:"currency" gorm:"type:varchar(3);default:'USD'"`

	// Report details
	PageCount       int             `json:"page_count" gorm:"default:0"`
	Formats         StringSlice     `json:"formats,omitempty" gorm:"type:jsonb"`
	Geography       StringSlice     `json:"geography" gorm:"type:jsonb;not null"`

	// Status and access
	Status                  string     `json:"status" gorm:"type:varchar(20);default:'draft';index"`
	IsFeatured              bool       `json:"is_featured" gorm:"default:false"`

	// Publishing
	PublishDate             *time.Time `json:"publish_date" gorm:"index"`
	ScheduledPublishEnabled bool       `json:"scheduled_publish_enabled" gorm:"default:false"`

	// Authors (JSON array of user IDs)
	AuthorIDs       UintSlice       `json:"author_ids,omitempty" gorm:"type:jsonb"`

	// Market data
	MarketMetrics   *MarketMetrics  `json:"market_metrics,omitempty" gorm:"type:jsonb"`
	KeyPlayers      KeyPlayers      `json:"key_players,omitempty" gorm:"type:jsonb"`

	// Content sections
	Sections        ReportSections  `json:"sections" gorm:"type:jsonb;not null"`

	// FAQs
	FAQs            FAQs            `json:"faqs,omitempty" gorm:"type:jsonb"`

	// SEO Metadata
	MetaTitle       string          `json:"meta_title,omitempty" gorm:"type:varchar(255)"`
	MetaDescription string          `json:"meta_description,omitempty" gorm:"type:varchar(500)"`
	MetaKeywords    string          `json:"meta_keywords,omitempty" gorm:"type:varchar(500)"`

	// Internal links configuration
	InternalLinks   InternalLinks   `json:"internal_links,omitempty" gorm:"type:jsonb"`

	// User tracking (admin metadata)
	CreatedBy         *uint          `json:"created_by,omitempty" gorm:"index"`
	UpdatedBy         *uint          `json:"updated_by,omitempty" gorm:"index"`
	InternalNotes     string         `json:"internal_notes,omitempty" gorm:"type:text"`

	// Timestamps
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at,omitempty" gorm:"index"`
}

type ChartMetadata struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ReportID    uint      `json:"report_id" gorm:"index;not null"`
	Title       string    `json:"title" gorm:"type:varchar(500);not null"`
	ChartType   string    `json:"chart_type" gorm:"type:varchar(50)"` // bar, line, pie, etc.
	Description string    `json:"description" gorm:"type:text"`
	DataPoints  int       `json:"data_points" gorm:"default:0"`
	Order       int       `json:"order" gorm:"default:0"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Report      *Report   `json:"report,omitempty" gorm:"foreignKey:ReportID"`
}

// ReportVersion stores historical versions of reports
type ReportVersion struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	ReportID        uint            `json:"report_id" gorm:"index;not null"`
	VersionNumber   int             `json:"version_number" gorm:"not null"`
	PublishedBy     uint            `json:"published_by" gorm:"not null"` // User ID who published
	PublishedAt     time.Time       `json:"published_at" gorm:"not null"`
	Sections        ReportSections  `json:"sections" gorm:"type:jsonb;not null"`

	// SEO Metadata (versioned)
	MetaTitle       string          `json:"meta_title,omitempty" gorm:"type:varchar(255)"`
	MetaDescription string          `json:"meta_description,omitempty" gorm:"type:varchar(500)"`
	MetaKeywords    string          `json:"meta_keywords,omitempty" gorm:"type:varchar(500)"`

	CreatedAt       time.Time       `json:"created_at"`
	Report          *Report         `json:"report,omitempty" gorm:"foreignKey:ReportID"`
}

type ReportWithRelations struct {
	Report
	CategoryName string           `json:"category_name,omitempty"`
	Authors      []author.Author  `json:"authors,omitempty" gorm:"-"`
	Charts       []ChartMetadata  `json:"charts,omitempty" gorm:"-"`
	Versions     []ReportVersion  `json:"versions,omitempty" gorm:"-"`
}

// UserInfo represents user information for admin responses
type UserInfo struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// AdminReportResponse includes all fields plus admin metadata
type AdminReportResponse struct {
	Report                  // Embed base report
	CategoryName string     `json:"category_name,omitempty"`

	// Creator/updater info
	CreatedByUser  *UserInfo `json:"created_by_user,omitempty"`
	UpdatedByUser  *UserInfo `json:"updated_by_user,omitempty"`
	ApprovedByUser *UserInfo `json:"approved_by_user,omitempty"`

	// Relations
	Authors  []UserInfo       `json:"authors,omitempty"`
	Charts   []ChartMetadata  `json:"charts,omitempty"`
	Versions []ReportVersion  `json:"versions,omitempty"`
}
