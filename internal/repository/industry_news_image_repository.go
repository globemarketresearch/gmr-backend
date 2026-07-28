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
	// IndustryNewsImage.IsActive carries a `gorm:"default:true"` tag. GORM
	// unconditionally omits zero-valued fields with a static default from the
	// INSERT statement (regardless of Select), so an explicit IsActive:false
	// would otherwise be silently persisted as true. Follow up with an
	// explicit column update whenever the caller wants it inactive.
	wantInactive := !image.IsActive
	if err := r.db.Create(image).Error; err != nil {
		return err
	}
	if wantInactive {
		return r.db.Model(image).Update("is_active", false).Error
	}
	return nil
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
