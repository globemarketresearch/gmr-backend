package repository

import (
	"strings"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/blog"
	"gorm.io/gorm"
)

type BlogRepository interface {
	Create(b *blog.Blog) error
	GetAll(query blog.GetBlogsQuery) ([]blog.Blog, int64, error)
	GetByCategorySlug(categorySlug string, page, limit int) ([]blog.Blog, int64, error)
	GetByID(id uint) (*blog.Blog, error)
	GetBySlug(slug string) (*blog.Blog, error)
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

type blogRepository struct {
	db *gorm.DB
}

func NewBlogRepository(db *gorm.DB) BlogRepository {
	return &blogRepository{db: db}
}

func (r *blogRepository) Create(b *blog.Blog) error {
	return r.db.Create(b).Error
}

func (r *blogRepository) GetAll(query blog.GetBlogsQuery) ([]blog.Blog, int64, error) {
	var blogs []blog.Blog
	var total int64

	db := r.db.Model(&blog.Blog{})

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

	// Fetch blogs with author and category details
	if err := db.Preload("Author").Preload("Category").Find(&blogs).Error; err != nil {
		return nil, 0, err
	}

	return blogs, total, nil
}

func (r *blogRepository) GetByCategorySlug(categorySlug string, page, limit int) ([]blog.Blog, int64, error) {
	var blogs []blog.Blog
	var total int64

	offset := (page - 1) * limit

	if err := r.db.Model(&blog.Blog{}).
		Where("category_id = (SELECT id FROM categories WHERE slug = ? AND is_active = true) AND deleted_at IS NULL AND status = 'published'", categorySlug).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Where("category_id = (SELECT id FROM categories WHERE slug = ? AND is_active = true) AND deleted_at IS NULL AND status = 'published'", categorySlug).
		Order("created_at DESC").Offset(offset).Limit(limit).
		Preload("Author").Preload("Category").Find(&blogs).Error; err != nil {
		return nil, 0, err
	}

	return blogs, total, nil
}

func (r *blogRepository) GetByID(id uint) (*blog.Blog, error) {
	var b blog.Blog
	if err := r.db.Preload("Author").Preload("Category").Where("deleted_at IS NULL").First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *blogRepository) GetBySlug(slug string) (*blog.Blog, error) {
	var b blog.Blog
	if err := r.db.Preload("Author").Preload("Category").Where("slug = ? AND deleted_at IS NULL", slug).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *blogRepository) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&blog.Blog{}).Where("id = ?", id).Updates(updates).Error
}

func (r *blogRepository) Delete(id uint) error {
	return r.db.Delete(&blog.Blog{}, id).Error
}

func (r *blogRepository) SubmitForReview(id uint) error {
	return r.db.Model(&blog.Blog{}).Where("id = ?", id).Update("status", blog.StatusReview).Error
}

func (r *blogRepository) Publish(id uint) error {
	return r.db.Model(&blog.Blog{}).Where("id = ?", id).Update("status", blog.StatusPublished).Error
}

func (r *blogRepository) Unpublish(id uint) error {
	return r.db.Model(&blog.Blog{}).Where("id = ?", id).Update("status", blog.StatusDraft).Error
}

func (r *blogRepository) SoftDelete(id uint) error {
	return r.db.Model(&blog.Blog{}).Where("id = ?", id).Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *blogRepository) Restore(id uint) error {
	return r.db.Model(&blog.Blog{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

func (r *blogRepository) PublishScheduled(now time.Time) error {
	return r.db.Model(&blog.Blog{}).
		Where("scheduled_publish_enabled = ? AND status != ? AND publish_date <= ?",
			true, blog.StatusPublished, now).
		Updates(map[string]interface{}{
			"status":                    blog.StatusPublished,
			"scheduled_publish_enabled": false,
		}).Error
}

func (r *blogRepository) SchedulePublish(id uint, publishDate time.Time) error {
	return r.db.Model(&blog.Blog{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"publish_date":              publishDate,
			"scheduled_publish_enabled": true,
		}).Error
}

func (r *blogRepository) CancelScheduledPublish(id uint) error {
	return r.db.Model(&blog.Blog{}).
		Where("id = ?", id).
		Update("scheduled_publish_enabled", false).Error
}
