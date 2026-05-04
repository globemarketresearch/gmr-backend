package repository

import (
	"github.com/healthcare-market-research/backend/internal/domain/category"
	"gorm.io/gorm"
)

type CategoryRepository interface {
	GetAll(page, limit int) ([]category.Category, int64, error)
	GetBySlug(slug string) (*category.Category, error)
	GetByID(id uint) (*category.Category, error)
	Update(cat *category.Category) error
}

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) GetAll(page, limit int) ([]category.Category, int64, error) {
	var categories []category.Category
	var total int64

	offset := (page - 1) * limit

	// Use raw SQL for better performance
	countSQL := "SELECT COUNT(*) FROM categories WHERE is_active = true"
	if err := r.db.Raw(countSQL).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// Raw SQL for fetching categories
	querySQL := `
		SELECT id, name, slug, description, image_url, is_active, created_at, updated_at
		FROM categories
		WHERE is_active = true
		ORDER BY name ASC
		LIMIT ? OFFSET ?
	`

	err := r.db.Raw(querySQL, limit, offset).Scan(&categories).Error

	return categories, total, err
}

func (r *categoryRepository) GetBySlug(slug string) (*category.Category, error) {
	var cat category.Category

	// Use raw SQL for better performance
	querySQL := `
		SELECT id, name, slug, description, image_url, is_active, created_at, updated_at
		FROM categories
		WHERE slug = ? AND is_active = true
		LIMIT 1
	`

	err := r.db.Raw(querySQL, slug).Scan(&cat).Error
	if err != nil {
		return nil, err
	}

	return &cat, nil
}

func (r *categoryRepository) GetByID(id uint) (*category.Category, error) {
	var cat category.Category

	// Use raw SQL for better performance
	querySQL := `
		SELECT id, name, slug, description, image_url, is_active, created_at, updated_at
		FROM categories
		WHERE id = ? AND is_active = true
		LIMIT 1
	`

	err := r.db.Raw(querySQL, id).Scan(&cat).Error
	if err != nil {
		return nil, err
	}

	return &cat, nil
}

func (r *categoryRepository) Update(cat *category.Category) error {
	return r.db.Save(cat).Error
}
