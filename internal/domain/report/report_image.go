package report

import "time"

// ReportImage represents an image associated with a report (charts, graphs, diagrams, infographics)
// These images are for internal/admin use only and are not exposed through public report APIs
// @Description Report image model for storing multiple images per report
type ReportImage struct {
	ID         uint      `json:"id" gorm:"primaryKey" example:"1"`                                                                                                                                                        // Unique identifier
	ReportID   uint      `json:"report_id" gorm:"index:idx_report_images_report_id;index:idx_report_images_report_active,priority:1;not null;constraint:OnDelete:CASCADE" example:"123"`                                // ID of the associated report
	ImageURL   string    `json:"image_url" gorm:"type:varchar(500);not null" example:"https://example.com/images/chart-1.png"`                                                                                           // URL to the image from external image service
	Title      string    `json:"title,omitempty" gorm:"type:varchar(255)" example:"Market Size Forecast 2024-2030"`                                                                                                      // Descriptive title for the image
	IsActive   bool      `json:"is_active" gorm:"default:true;index:idx_report_images_is_active;index:idx_report_images_report_active,priority:2" example:"true"`                                                       // Soft delete flag - false to hide without losing data
	UploadedBy *uint     `json:"uploaded_by,omitempty" gorm:"index:idx_report_images_uploaded_by;constraint:OnDelete:SET NULL" example:"5"`                                                                              // ID of user who uploaded the image
	CreatedAt  time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`                                                                                                                                               // Timestamp when image was created
	UpdatedAt  time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`                                                                                                                                               // Timestamp when image was last updated

	// Foreign key relationship
	Report *Report `json:"report,omitempty" gorm:"foreignKey:ReportID;constraint:OnDelete:CASCADE"` // Associated report (optional in responses)
}
