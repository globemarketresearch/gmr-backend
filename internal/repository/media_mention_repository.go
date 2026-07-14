package repository

import (
	"github.com/healthcare-market-research/backend/internal/domain/mediamention"
	"gorm.io/gorm"
)

type MediaMentionRepository interface {
	GetAll(page, limit int, search string) ([]mediamention.MediaMention, int64, error)
	GetByID(id uint) (*mediamention.MediaMention, error)
	Create(mention *mediamention.MediaMention) error
	Update(mention *mediamention.MediaMention) error
	Delete(id uint) error
}

type mediaMentionRepository struct {
	db *gorm.DB
}

func NewMediaMentionRepository(db *gorm.DB) MediaMentionRepository {
	return &mediaMentionRepository{db: db}
}

func (r *mediaMentionRepository) GetAll(page, limit int, search string) ([]mediamention.MediaMention, int64, error) {
	var mentions []mediamention.MediaMention
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&mediamention.MediaMention{})

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("title ILIKE ?", searchPattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("display_order ASC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&mentions).Error

	return mentions, total, err
}

func (r *mediaMentionRepository) GetByID(id uint) (*mediamention.MediaMention, error) {
	var mention mediamention.MediaMention
	err := r.db.First(&mention, id).Error
	if err != nil {
		return nil, err
	}
	return &mention, nil
}

func (r *mediaMentionRepository) Create(mention *mediamention.MediaMention) error {
	return r.db.Create(mention).Error
}

func (r *mediaMentionRepository) Update(mention *mediamention.MediaMention) error {
	return r.db.Save(mention).Error
}

func (r *mediaMentionRepository) Delete(id uint) error {
	return r.db.Delete(&mediamention.MediaMention{}, id).Error
}
