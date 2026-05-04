package repository

import (
	"strings"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/press_release"
	"gorm.io/gorm"
)

type PressReleaseRepository interface {
	Create(pr *press_release.PressRelease) error
	GetAll(query press_release.GetPressReleasesQuery) ([]press_release.PressRelease, int64, error)
	GetByCategorySlug(categorySlug string, page, limit int) ([]press_release.PressRelease, int64, error)
	GetByID(id uint) (*press_release.PressRelease, error)
	GetBySlug(slug string) (*press_release.PressRelease, error)
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

type pressReleaseRepository struct {
	db *gorm.DB
}

func NewPressReleaseRepository(db *gorm.DB) PressReleaseRepository {
	return &pressReleaseRepository{db: db}
}

func (r *pressReleaseRepository) Create(pr *press_release.PressRelease) error {
	return r.db.Create(pr).Error
}

func (r *pressReleaseRepository) GetAll(query press_release.GetPressReleasesQuery) ([]press_release.PressRelease, int64, error) {
	var pressReleases []press_release.PressRelease
	var total int64

	db := r.db.Model(&press_release.PressRelease{})

	// Apply filters
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	if query.CategorySlug != "" {
		db = db.Where("category_id = (SELECT id FROM categories WHERE slug = ? AND is_active = true)", query.CategorySlug)
	} else if query.CategoryID != "" {
		db = db.Where("category_id = ?", query.CategoryID)
	}

	if query.Tags != "" {
		// Split comma-separated tags and search for any of them
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

	// Apply soft delete filter
	if query.Deleted == "true" {
		db = db.Where("deleted_at IS NOT NULL")
	} else {
		db = db.Where("deleted_at IS NULL")
	}

	// Count total before pagination
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (query.Page - 1) * query.Limit
	orderClause := "created_at DESC"
	if query.SortBy != "" {
		orderClause = query.SortBy
	}
	db = db.Order(orderClause).Offset(offset).Limit(query.Limit)

	// Fetch press releases with author and category details
	if err := db.Preload("Author").Preload("Category").Find(&pressReleases).Error; err != nil {
		return nil, 0, err
	}

	return pressReleases, total, nil
}

func (r *pressReleaseRepository) GetByCategorySlug(categorySlug string, page, limit int) ([]press_release.PressRelease, int64, error) {
	var pressReleases []press_release.PressRelease
	var total int64

	offset := (page - 1) * limit

	if err := r.db.Model(&press_release.PressRelease{}).
		Where("category_id = (SELECT id FROM categories WHERE slug = ? AND is_active = true) AND deleted_at IS NULL AND status = 'published'", categorySlug).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Where("category_id = (SELECT id FROM categories WHERE slug = ? AND is_active = true) AND deleted_at IS NULL AND status = 'published'", categorySlug).
		Order("created_at DESC").Offset(offset).Limit(limit).
		Preload("Author").Preload("Category").Find(&pressReleases).Error; err != nil {
		return nil, 0, err
	}

	return pressReleases, total, nil
}

func (r *pressReleaseRepository) GetByID(id uint) (*press_release.PressRelease, error) {
	var pr press_release.PressRelease
	if err := r.db.Preload("Author").Preload("Category").Where("deleted_at IS NULL").First(&pr, id).Error; err != nil {
		return nil, err
	}
	return &pr, nil
}

func (r *pressReleaseRepository) GetBySlug(slug string) (*press_release.PressRelease, error) {
	var pr press_release.PressRelease
	if err := r.db.Preload("Author").Preload("Category").Where("slug = ? AND deleted_at IS NULL", slug).First(&pr).Error; err != nil {
		return nil, err
	}
	return &pr, nil
}

func (r *pressReleaseRepository) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&press_release.PressRelease{}).Where("id = ?", id).Updates(updates).Error
}

func (r *pressReleaseRepository) Delete(id uint) error {
	return r.db.Delete(&press_release.PressRelease{}, id).Error
}

func (r *pressReleaseRepository) SubmitForReview(id uint) error {
	return r.db.Model(&press_release.PressRelease{}).Where("id = ?", id).Update("status", press_release.StatusReview).Error
}

func (r *pressReleaseRepository) Publish(id uint) error {
	return r.db.Model(&press_release.PressRelease{}).Where("id = ?", id).Update("status", press_release.StatusPublished).Error
}

func (r *pressReleaseRepository) Unpublish(id uint) error {
	return r.db.Model(&press_release.PressRelease{}).Where("id = ?", id).Update("status", press_release.StatusDraft).Error
}

func (r *pressReleaseRepository) SoftDelete(id uint) error {
	return r.db.Model(&press_release.PressRelease{}).Where("id = ?", id).Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *pressReleaseRepository) Restore(id uint) error {
	return r.db.Model(&press_release.PressRelease{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

func (r *pressReleaseRepository) PublishScheduled(now time.Time) error {
	return r.db.Model(&press_release.PressRelease{}).
		Where("scheduled_publish_enabled = ? AND status != ? AND publish_date <= ?",
			true, press_release.StatusPublished, now).
		Updates(map[string]interface{}{
			"status":                    press_release.StatusPublished,
			"scheduled_publish_enabled": false,
		}).Error
}

func (r *pressReleaseRepository) SchedulePublish(id uint, publishDate time.Time) error {
	return r.db.Model(&press_release.PressRelease{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"publish_date":              publishDate,
			"scheduled_publish_enabled": true,
		}).Error
}

func (r *pressReleaseRepository) CancelScheduledPublish(id uint) error {
	return r.db.Model(&press_release.PressRelease{}).
		Where("id = ?", id).
		Update("scheduled_publish_enabled", false).Error
}
