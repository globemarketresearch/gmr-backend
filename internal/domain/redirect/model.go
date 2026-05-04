package redirect

import (
	"time"
)

// Redirect represents a URL redirect rule in the database
type Redirect struct {
	ID             int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	SourceURL      string     `json:"sourceUrl" gorm:"type:varchar(500);not null"`
	DestinationURL *string    `json:"destinationUrl" gorm:"type:varchar(500)"`
	RedirectType   int        `json:"redirectType" gorm:"not null;default:301"`
	IsEnabled      bool       `json:"isEnabled" gorm:"not null;default:true"`
	HitCount       int64      `json:"hitCount" gorm:"not null;default:0"`
	Notes          string     `json:"notes" gorm:"type:varchar(500)"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty" gorm:"index"`
}

// TableName specifies the table name for GORM
func (Redirect) TableName() string {
	return "redirects"
}

// CreateRedirectRequest is the request body for creating a new redirect
type CreateRedirectRequest struct {
	SourceURL      string  `json:"sourceUrl"`
	DestinationURL *string `json:"destinationUrl"`
	RedirectType   int     `json:"redirectType"`
	Notes          string  `json:"notes"`
}

// UpdateRedirectRequest is the request body for updating a redirect
type UpdateRedirectRequest struct {
	SourceURL      *string `json:"sourceUrl"`
	DestinationURL *string `json:"destinationUrl"`
	RedirectType   *int    `json:"redirectType"`
	IsEnabled      *bool   `json:"isEnabled"`
	Notes          *string `json:"notes"`
}

// GetRedirectsQuery represents query parameters for filtering redirects
type GetRedirectsQuery struct {
	Search  string
	Enabled string // "true" | "false" | ""
	Page    int
	Limit   int
}

// RedirectListResponse represents a list of redirects with pagination
type RedirectListResponse struct {
	Redirects  []Redirect `json:"redirects"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"totalPages"`
}

// RedirectResponse represents a single redirect response
type RedirectResponse struct {
	Redirect Redirect `json:"redirect"`
}

// BulkDeleteRequest contains IDs for bulk deletion
type BulkDeleteRequest struct {
	IDs []int64 `json:"ids"`
}
