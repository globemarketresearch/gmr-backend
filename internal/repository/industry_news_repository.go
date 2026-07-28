package repository

import (
	"strings"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
	"gorm.io/gorm"
)

type IndustryNewsRepository interface {
	Create(n *industry_news.IndustryNews) error
	GetAll(query industry_news.GetIndustryNewsQuery) ([]industry_news.IndustryNews, int64, error)
	GetByCategorySlug(categorySlug string, page, limit int) ([]industry_news.IndustryNews, int64, error)
	GetByID(id uint) (*industry_news.IndustryNews, error)
	GetBySlug(slug string) (*industry_news.IndustryNews, error)
	Update(id uint, updates map[string]interface{}) error
	Delete(id uint) error
	SoftDelete(id uint) error
	Restore(id uint) error
	SubmitForReview(id uint) error
	Publish(id uint) error
	Unpublish(id uint) error
	PublishScheduled(now time.Time) error
	SchedulePublish(id uint, publishDate time.Time) error
	CancelScheduledPublish(id uint) error
}

type industryNewsRepository struct {
	db *gorm.DB
}

func NewIndustryNewsRepository(db *gorm.DB) IndustryNewsRepository {
	return &industryNewsRepository{db: db}
}

func (r *industryNewsRepository) Create(n *industry_news.IndustryNews) error {
	return r.db.Create(n).Error
}

func (r *industryNewsRepository) GetAll(query industry_news.GetIndustryNewsQuery) ([]industry_news.IndustryNews, int64, error) {
	var items []industry_news.IndustryNews
	var total int64

	db := r.db.Model(&industry_news.IndustryNews{})

	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	if query.CategorySlug != "" {
		db = db.Where("category_id = (SELECT id FROM categories WHERE slug = ? AND is_active = true)", query.CategorySlug)
	} else if query.CategoryID != "" {
		db = db.Where("category_id = ?", query.CategoryID)
	}

	if query.Tags != "" {
		tags := strings.Split(query.Tags, ",")
		tagConditions := make([]string, 0, len(tags))
		tagValues := make([]interface{}, 0, len(tags))

		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagConditions = append(tagConditions, "tags LIKE ?")
				tagValues = append(tagValues, "%"+tag+"%")
			}
		}

		if len(tagConditions) > 0 {
			db = db.Where(strings.Join(tagConditions, " OR "), tagValues...)
		}
	}

	if query.AuthorID != "" {
		db = db.Where("author_id = ?", query.AuthorID)
	}

	if query.Location != "" {
		db = db.Where("location ILIKE ?", "%"+query.Location+"%")
	}

	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("title ILIKE ? OR excerpt ILIKE ? OR content ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if query.Deleted == "true" {
		db = db.Where("deleted_at IS NOT NULL")
	} else {
		db = db.Where("deleted_at IS NULL")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.Limit
	orderClause := "created_at DESC"
	if query.SortBy != "" {
		orderClause = query.SortBy
	}
	db = db.Order(orderClause).Offset(offset).Limit(query.Limit)

	if err := db.Preload("Author").Preload("Category").Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *industryNewsRepository) GetByCategorySlug(categorySlug string, page, limit int) ([]industry_news.IndustryNews, int64, error) {
	var items []industry_news.IndustryNews
	var total int64

	offset := (page - 1) * limit

	if err := r.db.Model(&industry_news.IndustryNews{}).
		Where("category_id = (SELECT id FROM categories WHERE slug = ? AND is_active = true) AND deleted_at IS NULL AND status = 'published'", categorySlug).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Where("category_id = (SELECT id FROM categories WHERE slug = ? AND is_active = true) AND deleted_at IS NULL AND status = 'published'", categorySlug).
		Order("created_at DESC").Offset(offset).Limit(limit).
		Preload("Author").Preload("Category").Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *industryNewsRepository) GetByID(id uint) (*industry_news.IndustryNews, error) {
	var n industry_news.IndustryNews
	if err := r.db.Preload("Author").Preload("Category").Where("deleted_at IS NULL").First(&n, id).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *industryNewsRepository) GetBySlug(slug string) (*industry_news.IndustryNews, error) {
	var n industry_news.IndustryNews
	if err := r.db.Preload("Author").Preload("Category").Where("slug = ? AND deleted_at IS NULL", slug).First(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *industryNewsRepository) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Updates(updates).Error
}

func (r *industryNewsRepository) Delete(id uint) error {
	return r.db.Delete(&industry_news.IndustryNews{}, id).Error
}

func (r *industryNewsRepository) SubmitForReview(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Update("status", industry_news.StatusReview).Error
}

func (r *industryNewsRepository) Publish(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Update("status", industry_news.StatusPublished).Error
}

func (r *industryNewsRepository) Unpublish(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Update("status", industry_news.StatusDraft).Error
}

func (r *industryNewsRepository) SoftDelete(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *industryNewsRepository) Restore(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

func (r *industryNewsRepository) PublishScheduled(now time.Time) error {
	return r.db.Model(&industry_news.IndustryNews{}).
		Where("scheduled_publish_enabled = ? AND status != ? AND publish_date <= ?",
			true, industry_news.StatusPublished, now).
		Updates(map[string]interface{}{
			"status":                    industry_news.StatusPublished,
			"scheduled_publish_enabled": false,
		}).Error
}

func (r *industryNewsRepository) SchedulePublish(id uint, publishDate time.Time) error {
	return r.db.Model(&industry_news.IndustryNews{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"publish_date":              publishDate,
			"scheduled_publish_enabled": true,
		}).Error
}

func (r *industryNewsRepository) CancelScheduledPublish(id uint) error {
	return r.db.Model(&industry_news.IndustryNews{}).
		Where("id = ?", id).
		Update("scheduled_publish_enabled", false).Error
}
