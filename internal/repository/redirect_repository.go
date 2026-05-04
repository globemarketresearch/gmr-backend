package repository

import (
	"github.com/healthcare-market-research/backend/internal/domain/redirect"
	"gorm.io/gorm"
)

type RedirectRepository interface {
	Create(r *redirect.Redirect) error
	GetAll(query redirect.GetRedirectsQuery) ([]redirect.Redirect, int64, error)
	GetByID(id int64) (*redirect.Redirect, error)
	GetBySourceURL(sourceURL string) (*redirect.Redirect, error)
	Update(id int64, updates map[string]interface{}) error
	Delete(id int64) error
	ToggleEnabled(id int64) error
	IncrementHitCount(id int64) error
	GetAllActive() ([]redirect.Redirect, error)
	BulkDelete(ids []int64) error
}

type redirectRepository struct {
	db *gorm.DB
}

func NewRedirectRepository(db *gorm.DB) RedirectRepository {
	return &redirectRepository{db: db}
}

func (r *redirectRepository) Create(rd *redirect.Redirect) error {
	return r.db.Create(rd).Error
}

func (r *redirectRepository) GetAll(query redirect.GetRedirectsQuery) ([]redirect.Redirect, int64, error) {
	var redirects []redirect.Redirect
	var total int64

	db := r.db.Model(&redirect.Redirect{}).Where("deleted_at IS NULL")

	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("source_url ILIKE ? OR destination_url ILIKE ? OR notes ILIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	if query.Enabled == "true" {
		db = db.Where("is_enabled = ?", true)
	} else if query.Enabled == "false" {
		db = db.Where("is_enabled = ?", false)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 || query.Limit > 100 {
		query.Limit = 20
	}

	offset := (query.Page - 1) * query.Limit
	db = db.Order("created_at DESC").Offset(offset).Limit(query.Limit)

	if err := db.Find(&redirects).Error; err != nil {
		return nil, 0, err
	}

	return redirects, total, nil
}

func (r *redirectRepository) GetByID(id int64) (*redirect.Redirect, error) {
	var rd redirect.Redirect
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&rd).Error; err != nil {
		return nil, err
	}
	return &rd, nil
}

func (r *redirectRepository) GetBySourceURL(sourceURL string) (*redirect.Redirect, error) {
	var rd redirect.Redirect
	if err := r.db.Where("source_url = ? AND deleted_at IS NULL", sourceURL).First(&rd).Error; err != nil {
		return nil, err
	}
	return &rd, nil
}

func (r *redirectRepository) Update(id int64, updates map[string]interface{}) error {
	return r.db.Model(&redirect.Redirect{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
}

func (r *redirectRepository) Delete(id int64) error {
	return r.db.Model(&redirect.Redirect{}).Where("id = ?", id).Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *redirectRepository) ToggleEnabled(id int64) error {
	return r.db.Model(&redirect.Redirect{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("is_enabled", gorm.Expr("NOT is_enabled")).Error
}

func (r *redirectRepository) IncrementHitCount(id int64) error {
	return r.db.Model(&redirect.Redirect{}).
		Where("id = ?", id).
		Update("hit_count", gorm.Expr("hit_count + 1")).Error
}

func (r *redirectRepository) GetAllActive() ([]redirect.Redirect, error) {
	var redirects []redirect.Redirect
	if err := r.db.Where("is_enabled = ? AND deleted_at IS NULL", true).Find(&redirects).Error; err != nil {
		return nil, err
	}
	return redirects, nil
}

func (r *redirectRepository) BulkDelete(ids []int64) error {
	return r.db.Model(&redirect.Redirect{}).
		Where("id IN ?", ids).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}
