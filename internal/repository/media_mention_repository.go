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

const mediaMentionReportJoinSQL = `
	SELECT m.*, r.slug as report_slug, r.title as report_title
	FROM media_mentions m
	LEFT JOIN reports r ON m.report_id = r.id AND r.status = 'published' AND r.deleted_at IS NULL
`

func (r *mediaMentionRepository) GetAll(page, limit int, search string) ([]mediamention.MediaMention, int64, error) {
	var mentions []mediamention.MediaMention
	var total int64

	offset := (page - 1) * limit

	countQuery := r.db.Model(&mediamention.MediaMention{})
	if search != "" {
		countQuery = countQuery.Where("title ILIKE ?", "%"+search+"%")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	querySQL := mediaMentionReportJoinSQL
	args := []interface{}{}
	if search != "" {
		querySQL += " WHERE m.title ILIKE ?"
		args = append(args, "%"+search+"%")
	}
	querySQL += " ORDER BY m.display_order ASC, m.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := r.db.Raw(querySQL, args...).Scan(&mentions).Error
	return mentions, total, err
}

func (r *mediaMentionRepository) GetByID(id uint) (*mediamention.MediaMention, error) {
	var mention mediamention.MediaMention
	querySQL := mediaMentionReportJoinSQL + " WHERE m.id = ?"

	err := r.db.Raw(querySQL, id).Scan(&mention).Error
	if err != nil {
		return nil, err
	}
	if mention.ID == 0 {
		return nil, gorm.ErrRecordNotFound
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
