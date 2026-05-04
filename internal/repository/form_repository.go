package repository

import (
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/form"
	"gorm.io/gorm"
)

type FormRepository interface {
	Create(submission *form.FormSubmission) error
	GetAll(query form.GetSubmissionsQuery) ([]form.FormSubmission, int64, error)
	GetByID(id uint) (*form.FormSubmission, error)
	GetByCategory(category string, page, limit int) ([]form.FormSubmission, int64, error)
	Delete(id uint) error
	BulkDelete(ids []uint) (int64, error)
	GetStats() (*form.SubmissionStats, error)
	UpdateStatus(id uint, status form.FormStatus, processedBy *uint) error
}

type formRepository struct {
	db *gorm.DB
}

func NewFormRepository(db *gorm.DB) FormRepository {
	return &formRepository{db: db}
}

func (r *formRepository) Create(submission *form.FormSubmission) error {
	return r.db.Create(submission).Error
}

func (r *formRepository) GetAll(query form.GetSubmissionsQuery) ([]form.FormSubmission, int64, error) {
	var submissions []form.FormSubmission
	var total int64

	offset := (query.Page - 1) * query.Limit

	dbQuery := r.db.Model(&form.FormSubmission{})

	// Apply filters
	if query.Category != "" {
		dbQuery = dbQuery.Where("category = ?", query.Category)
	}

	if query.Status != "" {
		dbQuery = dbQuery.Where("status = ?", query.Status)
	}

	// Date range filtering
	if query.DateFrom != "" {
		dateFrom, err := time.Parse(time.RFC3339, query.DateFrom)
		if err == nil {
			dbQuery = dbQuery.Where("created_at >= ?", dateFrom)
		}
	}

	if query.DateTo != "" {
		dateTo, err := time.Parse(time.RFC3339, query.DateTo)
		if err == nil {
			dbQuery = dbQuery.Where("created_at <= ?", dateTo)
		}
	}

	// Search in name, email, company (searching in JSONB data field)
	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		dbQuery = dbQuery.Where(
			"data->>'fullName' ILIKE ? OR data->>'email' ILIKE ? OR data->>'company' ILIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}

	// Get total count
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortBy := "created_at"
	if query.SortBy != "" {
		switch query.SortBy {
		case "createdAt":
			sortBy = "created_at"
		case "company":
			sortBy = "data->>'company'"
		case "name":
			sortBy = "data->>'fullName'"
		default:
			sortBy = "created_at"
		}
	}

	sortOrder := "DESC"
	if query.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)

	// Get paginated results
	err := dbQuery.Order(orderClause).
		Limit(query.Limit).
		Offset(offset).
		Find(&submissions).Error

	return submissions, total, err
}

func (r *formRepository) GetByID(id uint) (*form.FormSubmission, error) {
	var submission form.FormSubmission
	err := r.db.First(&submission, id).Error
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

func (r *formRepository) GetByCategory(category string, page, limit int) ([]form.FormSubmission, int64, error) {
	var submissions []form.FormSubmission
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&form.FormSubmission{}).Where("category = ?", category)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&submissions).Error

	return submissions, total, err
}

func (r *formRepository) Delete(id uint) error {
	result := r.db.Delete(&form.FormSubmission{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *formRepository) BulkDelete(ids []uint) (int64, error) {
	result := r.db.Where("id IN ?", ids).Delete(&form.FormSubmission{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *formRepository) GetStats() (*form.SubmissionStats, error) {
	stats := &form.SubmissionStats{
		ByCategory: make(map[string]int64),
		ByStatus:   make(map[string]int64),
		Recent:     make(map[string]int64),
	}

	// Get total count
	if err := r.db.Model(&form.FormSubmission{}).Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// Get count by category
	type CategoryCount struct {
		Category string
		Count    int64
	}
	var categoryCounts []CategoryCount
	if err := r.db.Model(&form.FormSubmission{}).
		Select("category, COUNT(*) as count").
		Group("category").
		Scan(&categoryCounts).Error; err != nil {
		return nil, err
	}
	for _, cc := range categoryCounts {
		stats.ByCategory[cc.Category] = cc.Count
	}

	// Get count by status
	type StatusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []StatusCount
	if err := r.db.Model(&form.FormSubmission{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return nil, err
	}
	for _, sc := range statusCounts {
		stats.ByStatus[sc.Status] = sc.Count
	}

	// Get recent submissions
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Today
	var todayCount int64
	if err := r.db.Model(&form.FormSubmission{}).
		Where("created_at >= ?", today).
		Count(&todayCount).Error; err != nil {
		return nil, err
	}
	stats.Recent["today"] = todayCount

	// This week
	var weekCount int64
	if err := r.db.Model(&form.FormSubmission{}).
		Where("created_at >= ?", weekStart).
		Count(&weekCount).Error; err != nil {
		return nil, err
	}
	stats.Recent["thisWeek"] = weekCount

	// This month
	var monthCount int64
	if err := r.db.Model(&form.FormSubmission{}).
		Where("created_at >= ?", monthStart).
		Count(&monthCount).Error; err != nil {
		return nil, err
	}
	stats.Recent["thisMonth"] = monthCount

	return stats, nil
}

func (r *formRepository) UpdateStatus(id uint, status form.FormStatus, processedBy *uint) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == form.StatusProcessed {
		now := time.Now()
		updates["processed_at"] = &now
		if processedBy != nil {
			updates["processed_by"] = processedBy
		}
	}

	return r.db.Model(&form.FormSubmission{}).
		Where("id = ?", id).
		Updates(updates).Error
}
