package mediamention

import (
	"time"
)

// MediaMention represents a citation of GMR research by an external publication
type MediaMention struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Title          string    `json:"title" gorm:"type:varchar(255);not null"`
	Link           string    `json:"link,omitempty" gorm:"type:varchar(500)"`
	ImageURL       string    `json:"imageUrl,omitempty" gorm:"type:varchar(500)"`
	DisplayOrder   int       `json:"displayOrder" gorm:"default:0"`
	ReportID       *uint     `json:"reportId,omitempty" gorm:"index"`
	ReportLinkText string    `json:"reportLinkText,omitempty" gorm:"type:varchar(255)"`
	ReportSlug     string    `json:"reportSlug,omitempty" gorm:"->"`
	ReportTitle    string    `json:"reportTitle,omitempty" gorm:"->"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// TableName specifies the table name for GORM
func (MediaMention) TableName() string {
	return "media_mentions"
}
