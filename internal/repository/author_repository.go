package repository

import (
	"github.com/healthcare-market-research/backend/internal/domain/author"
	"gorm.io/gorm"
)

type AuthorRepository interface {
	GetAll(page, limit int, search string) ([]author.Author, int64, error)
	GetByID(id uint) (*author.Author, error)
	GetByIDs(ids []uint) ([]author.Author, error)
	Create(author *author.Author) error
	Update(author *author.Author) error
	Delete(id uint) error
	IsReferencedInReports(id uint) (bool, error)
}

type authorRepository struct {
	db *gorm.DB
}

func NewAuthorRepository(db *gorm.DB) AuthorRepository {
	return &authorRepository{db: db}
}

func (r *authorRepository) GetAll(page, limit int, search string) ([]author.Author, int64, error) {
	var authors []author.Author
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&author.Author{})

	// Apply search filter if provided
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("name ILIKE ? OR role ILIKE ? OR bio ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&authors).Error

	return authors, total, err
}

func (r *authorRepository) GetByID(id uint) (*author.Author, error) {
	var auth author.Author
	err := r.db.First(&auth, id).Error
	if err != nil {
		return nil, err
	}
	return &auth, nil
}

func (r *authorRepository) GetByIDs(ids []uint) ([]author.Author, error) {
	if len(ids) == 0 {
		return []author.Author{}, nil
	}

	var authors []author.Author
	err := r.db.Where("id IN ?", ids).Order("name ASC").Find(&authors).Error
	if err != nil {
		return nil, err
	}

	return authors, nil
}

func (r *authorRepository) Create(auth *author.Author) error {
	return r.db.Create(auth).Error
}

func (r *authorRepository) Update(auth *author.Author) error {
	return r.db.Save(auth).Error
}

func (r *authorRepository) Delete(id uint) error {
	return r.db.Delete(&author.Author{}, id).Error
}

func (r *authorRepository) IsReferencedInReports(id uint) (bool, error) {
	var count int64

	// Check if author ID exists in any report's author_ids JSONB array
	err := r.db.Raw(`
		SELECT COUNT(*)
		FROM reports
		WHERE author_ids::jsonb ? ?
	`, id).Scan(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
