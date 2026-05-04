package audit

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Action constants - standardized action names
const (
	// Authentication actions
	ActionLogin        = "auth.login"
	ActionLoginFailed  = "auth.login_failed"
	ActionLogout       = "auth.logout"
	ActionTokenRefresh = "auth.token_refresh"

	// User management actions
	ActionUserCreate = "user.create"
	ActionUserUpdate = "user.update"
	ActionUserDelete = "user.delete"
	ActionRoleChange = "user.role_change"

	// Report actions
	ActionReportCreate  = "report.create"
	ActionReportUpdate  = "report.update"
	ActionReportDelete  = "report.delete"
	ActionReportPublish = "report.publish"

	// Category actions
	ActionCategoryCreate = "category.create"
	ActionCategoryUpdate = "category.update"
	ActionCategoryDelete = "category.delete"

	// Author actions
	ActionAuthorCreate = "author.create"
	ActionAuthorUpdate = "author.update"
	ActionAuthorDelete = "author.delete"

	// Blog actions
	ActionBlogCreate  = "blog.create"
	ActionBlogUpdate  = "blog.update"
	ActionBlogDelete  = "blog.delete"
	ActionBlogPublish = "blog.publish"

	// Press Release actions
	ActionPressReleaseCreate  = "press_release.create"
	ActionPressReleaseUpdate  = "press_release.update"
	ActionPressReleaseDelete  = "press_release.delete"
	ActionPressReleasePublish = "press_release.publish"
)

// EntityType constants
const (
	EntityUser         = "user"
	EntityReport       = "report"
	EntityCategory     = "category"
	EntityAuthor       = "author"
	EntityBlog         = "blog"
	EntityPressRelease = "press_release"
)

// Status constants
const (
	StatusSuccess = "success"
	StatusFailure = "failure"
)

// FieldChange represents before/after values for a field
type FieldChange struct {
	Old interface{} `json:"old"`
	New interface{} `json:"new"`
}

// Changes maps field names to their before/after values
type Changes map[string]FieldChange

// Value implements the driver.Valuer interface for database storage
func (c Changes) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface for database retrieval
func (c *Changes) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, c)
}

// AuditLog represents an audit trail entry
type AuditLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       *uint     `json:"user_id,omitempty" gorm:"index"`
	UserEmail    string    `json:"user_email" gorm:"type:varchar(255)"`
	UserRole     string    `json:"user_role" gorm:"type:varchar(20)"`
	Action       string    `json:"action" gorm:"type:varchar(100);not null;index"`
	EntityType   string    `json:"entity_type,omitempty" gorm:"type:varchar(50);index"`
	EntityID     *uint     `json:"entity_id,omitempty" gorm:"index"`
	IPAddress    string    `json:"ip_address" gorm:"type:varchar(45)"`
	UserAgent    string    `json:"user_agent" gorm:"type:text"`
	RequestID    string    `json:"request_id" gorm:"type:varchar(100);index"`
	Changes      Changes   `json:"changes,omitempty" gorm:"type:jsonb"`
	Status       string    `json:"status" gorm:"type:varchar(20);not null;index"`
	ErrorMessage string    `json:"error_message,omitempty" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
}

// TableName specifies the table name for GORM
func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditLogFilters for querying audit logs
type AuditLogFilters struct {
	UserID       *uint
	Action       string
	ActionPrefix string // Comma-separated prefixes, e.g. "auth." or "blog.,press_release."
	EntityType   string
	EntityID     *uint
	Status       string
	StartDate    *time.Time
	EndDate      *time.Time
	IPAddress    string
	Page         int
	Limit        int
}

// AuditLogResponse for API responses
type AuditLogResponse struct {
	ID           uint                   `json:"id"`
	UserID       *uint                  `json:"user_id,omitempty"`
	UserEmail    string                 `json:"user_email"`
	UserRole     string                 `json:"user_role"`
	Action       string                 `json:"action"`
	EntityType   string                 `json:"entity_type,omitempty"`
	EntityID     *uint                  `json:"entity_id,omitempty"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	RequestID    string                 `json:"request_id"`
	Changes      map[string]FieldChange `json:"changes,omitempty"`
	Status       string                 `json:"status"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ToResponse converts AuditLog to AuditLogResponse
func (a *AuditLog) ToResponse() *AuditLogResponse {
	return &AuditLogResponse{
		ID:           a.ID,
		UserID:       a.UserID,
		UserEmail:    a.UserEmail,
		UserRole:     a.UserRole,
		Action:       a.Action,
		EntityType:   a.EntityType,
		EntityID:     a.EntityID,
		IPAddress:    a.IPAddress,
		UserAgent:    a.UserAgent,
		RequestID:    a.RequestID,
		Changes:      a.Changes,
		Status:       a.Status,
		ErrorMessage: a.ErrorMessage,
		CreatedAt:    a.CreatedAt,
	}
}

// AuditEntry is a builder for creating audit logs
type AuditEntry struct {
	UserID       *uint
	UserEmail    string
	UserRole     string
	Action       string
	EntityType   string
	EntityID     *uint
	IPAddress    string
	UserAgent    string
	RequestID    string
	Changes      Changes
	Status       string
	ErrorMessage string
}

// ToAuditLog converts AuditEntry to AuditLog
func (e *AuditEntry) ToAuditLog() *AuditLog {
	return &AuditLog{
		UserID:       e.UserID,
		UserEmail:    e.UserEmail,
		UserRole:     e.UserRole,
		Action:       e.Action,
		EntityType:   e.EntityType,
		EntityID:     e.EntityID,
		IPAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		RequestID:    e.RequestID,
		Changes:      e.Changes,
		Status:       e.Status,
		ErrorMessage: e.ErrorMessage,
	}
}
