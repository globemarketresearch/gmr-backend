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
