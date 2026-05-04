package form

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// FormCategory represents the type of form submission
type FormCategory string

const (
	CategoryContact              FormCategory = "contact"
	CategoryRequestSample        FormCategory = "request-sample"
	CategoryRequestCustomization FormCategory = "request-customization"
	CategoryScheduleDemo         FormCategory = "schedule-demo"
)

// FormStatus represents the processing status of a submission
type FormStatus string

const (
	StatusPending   FormStatus = "pending"
	StatusProcessed FormStatus = "processed"
	StatusArchived  FormStatus = "archived"
)

// ContactFormData contains fields specific to contact form submissions
type ContactFormData struct {
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Company  string `json:"company"`
	Phone    string `json:"phone,omitempty"`
	Country  string `json:"country,omitempty"`
	Subject  string `json:"subject"`
	Message  string `json:"message"`
}

// RequestSampleFormData contains fields specific to request sample form submissions
type RequestSampleFormData struct {
	FullName       string `json:"fullName"`
	Email          string `json:"email"`
	Company        string `json:"company"`
	JobTitle       string `json:"jobTitle"`
	Phone          string `json:"phone,omitempty"`
	Country        string `json:"country,omitempty"`
	ReportTitle    string `json:"reportTitle"`
	AdditionalInfo string `json:"additionalInfo,omitempty"`
}

// FormData is a flexible JSON field that can contain either ContactFormData or RequestSampleFormData
type FormData map[string]interface{}

// Value implements the driver.Valuer interface for GORM
func (f FormData) Value() (driver.Value, error) {
	return json.Marshal(f)
}

// Scan implements the sql.Scanner interface for GORM
func (f *FormData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &f)
}

// SubmissionMetadata contains tracking information about the submission
type SubmissionMetadata struct {
	SubmittedAt string `json:"submittedAt"`
	IPAddress   string `json:"ipAddress,omitempty"`
	UserAgent   string `json:"userAgent,omitempty"`
	Referrer    string `json:"referrer,omitempty"`
	PageURL     string `json:"pageURL,omitempty"`
}

// Value implements the driver.Valuer interface for GORM
func (m SubmissionMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface for GORM
func (m *SubmissionMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &m)
}

// FormSubmission represents a form submission in the database
type FormSubmission struct {
	ID       uint         `json:"id" gorm:"primaryKey"`
	Category FormCategory `json:"category" gorm:"type:varchar(30);not null;index"`
	Status   FormStatus   `json:"status" gorm:"type:varchar(20);default:'pending';index"`

	// Form-specific data stored as JSONB
	Data FormData `json:"data" gorm:"type:jsonb;not null"`

	// Metadata about the submission
	Metadata SubmissionMetadata `json:"metadata" gorm:"type:jsonb"`

	// Processing tracking
	ProcessedAt *time.Time `json:"processedAt,omitempty"`
	ProcessedBy *uint      `json:"processedBy,omitempty"` // Admin user ID
	Notes       string     `json:"notes,omitempty" gorm:"type:text"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName specifies the table name for GORM
func (FormSubmission) TableName() string {
	return "form_submissions"
}

// CreateSubmissionRequest is the request body for creating a new submission
type CreateSubmissionRequest struct {
	Category FormCategory       `json:"category"`
	Data     FormData           `json:"data"`
	Metadata SubmissionMetadata `json:"metadata,omitempty"`
}

// SubmissionResponse is the response after creating a submission
type SubmissionResponse struct {
	Success      bool         `json:"success"`
	SubmissionID uint         `json:"submissionId"`
	Category     FormCategory `json:"category"`
	Message      string       `json:"message"`
	CreatedAt    time.Time    `json:"createdAt"`
}

// GetSubmissionsQuery represents query parameters for filtering submissions
type GetSubmissionsQuery struct {
	Category  string
	Status    string
	DateFrom  string
	DateTo    string
	Search    string
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

// SubmissionStats represents statistics about form submissions
type SubmissionStats struct {
	Total      int64                  `json:"total"`
	ByCategory map[string]int64       `json:"byCategory"`
	ByStatus   map[string]int64       `json:"byStatus"`
	Recent     map[string]int64       `json:"recent"`
}

// DeleteResponse represents the response after deleting submissions
type DeleteResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	DeletedCount int    `json:"deletedCount,omitempty"`
	DeletedID    uint   `json:"deletedId,omitempty"`
	DeletedIDs   []uint `json:"deletedIds,omitempty"`
}

// BulkDeleteRequest represents the request body for bulk delete
type BulkDeleteRequest struct {
	IDs []uint `json:"ids"`
}
