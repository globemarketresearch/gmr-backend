package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/report"
	"gorm.io/gorm"
)

// ReportFilters contains all possible filter options for reports
type ReportFilters struct {
	// Public filters
	Status      string   // 'draft' or 'published'
	Category    string   // Category slug
	Geography   []string // Array of geography strings
	Search      string   // Full-text search query
	Page        int
	Limit       int

	// Admin filters
	CreatedBy       *uint      // Filter by creator user ID
	UpdatedBy       *uint      // Filter by last updater user ID
	CreatedAfter    *time.Time // Date range start
	CreatedBefore   *time.Time // Date range end
	UpdatedAfter    *time.Time
	UpdatedBefore   *time.Time
	PublishedAfter  *time.Time
	PublishedBefore *time.Time
	IncludeDrafts   bool       // For admin, show drafts
	ShowDeleted     bool       // For admin, show only deleted reports
	SortBy          string     // Custom sort clause (e.g. "r.publish_date DESC NULLS LAST")
}

type ReportRepository interface {
	GetAll(page, limit int) ([]report.Report, int64, error)
	GetAllWithFilters(filters ReportFilters) ([]report.Report, int64, error)
	GetBySlug(slug string) (*report.ReportWithRelations, error)
	GetByID(id uint) (*report.Report, error)
	GetByIDWithRelations(id uint) (*report.ReportWithRelations, error)
	GetByCategorySlug(categorySlug string, page, limit int) ([]report.Report, int64, error)
	GetByAuthorID(authorID uint, page, limit int) ([]report.Report, int64, error)
	Search(query string, page, limit int) ([]report.Report, int64, error)
	GetChartsByReportID(reportID uint) ([]report.ChartMetadata, error)
	Create(report *report.Report) error
	Update(report *report.Report) error
	Delete(id uint) error
	SoftDelete(id uint) error
	Restore(id uint) error
	// Version history methods
	CreateVersion(version *report.ReportVersion) error
	GetVersionsByReportID(reportID uint) ([]report.ReportVersion, error)
	GetLatestVersionNumber(reportID uint) (int, error)
	// Scheduled publishing methods
	PublishScheduled(now time.Time) error
	SchedulePublish(id uint, publishDate time.Time) error
	CancelScheduledPublish(id uint) error
	// Internal linking
	GetLinkSuggestions() ([]report.LinkSuggestionItem, error)
}

type reportRepository struct {
	db         *gorm.DB
	authorRepo AuthorRepository
}

func NewReportRepository(db *gorm.DB, authorRepo AuthorRepository) ReportRepository {
	return &reportRepository{
		db:         db,
		authorRepo: authorRepo,
	}
}

func (r *reportRepository) GetAll(page, limit int) ([]report.Report, int64, error) {
	var reports []report.Report
	var total int64

	offset := (page - 1) * limit

	// Exclude soft-deleted reports
	countSQL := "SELECT COUNT(*) FROM reports WHERE deleted_at IS NULL"
	if err := r.db.Raw(countSQL).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	querySQL := `
		SELECT r.*, c.name as category_name, c.image_url as category_image_url
		FROM reports r
		LEFT JOIN categories c ON r.category_id = c.id
		WHERE r.deleted_at IS NULL
		ORDER BY COALESCE(r.id) DESC
		LIMIT ? OFFSET ?
	`

	err := r.db.Raw(querySQL, limit, offset).Scan(&reports).Error

	return reports, total, err
}

func (r *reportRepository) GetAllWithFilters(filters ReportFilters) ([]report.Report, int64, error) {
	var reports []report.Report
	var total int64

	offset := (filters.Page - 1) * filters.Limit

	// Build WHERE clause dynamically
	var conditions []string
	var args []interface{}
	var countArgs []interface{}

	// Handle deleted reports filter
	if filters.ShowDeleted {
		// Show only deleted reports
		conditions = append(conditions, "r.deleted_at IS NOT NULL")
	} else {
		// Exclude soft-deleted reports (default behavior)
		conditions = append(conditions, "r.deleted_at IS NULL")
	}

	// Status filter
	if filters.Status != "" {
		conditions = append(conditions, "r.status = ?")
		args = append(args, filters.Status)
		countArgs = append(countArgs, filters.Status)
	}

	// Category filter
	if filters.Category != "" {
		conditions = append(conditions, "c.slug = ?")
		args = append(args, filters.Category)
		countArgs = append(countArgs, filters.Category)
	}

	// Geography filter - check if any of the provided geographies are in the report's geography JSON array
	if len(filters.Geography) > 0 {
		geographyConditions := make([]string, len(filters.Geography))
		for i, geo := range filters.Geography {
			geographyConditions[i] = "r.geography::jsonb ? ?"
			args = append(args, geo)
			countArgs = append(countArgs, geo)
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(geographyConditions, " OR ")))
	}

	// Search filter
	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		conditions = append(conditions, "(r.title ILIKE ? OR r.summary ILIKE ? OR r.description ILIKE ?)")
		args = append(args, searchPattern, searchPattern, searchPattern)
		countArgs = append(countArgs, searchPattern, searchPattern, searchPattern)
	}

	// Admin filters
	if filters.CreatedBy != nil {
		conditions = append(conditions, "r.created_by = ?")
		args = append(args, *filters.CreatedBy)
		countArgs = append(countArgs, *filters.CreatedBy)
	}

	if filters.UpdatedBy != nil {
		conditions = append(conditions, "r.updated_by = ?")
		args = append(args, *filters.UpdatedBy)
		countArgs = append(countArgs, *filters.UpdatedBy)
	}

	// Date range filters
	if filters.CreatedAfter != nil {
		conditions = append(conditions, "r.created_at >= ?")
		args = append(args, *filters.CreatedAfter)
		countArgs = append(countArgs, *filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		conditions = append(conditions, "r.created_at <= ?")
		args = append(args, *filters.CreatedBefore)
		countArgs = append(countArgs, *filters.CreatedBefore)
	}

	if filters.UpdatedAfter != nil {
		conditions = append(conditions, "r.updated_at >= ?")
		args = append(args, *filters.UpdatedAfter)
		countArgs = append(countArgs, *filters.UpdatedAfter)
	}
	if filters.UpdatedBefore != nil {
		conditions = append(conditions, "r.updated_at <= ?")
		args = append(args, *filters.UpdatedBefore)
		countArgs = append(countArgs, *filters.UpdatedBefore)
	}

	if filters.PublishedAfter != nil {
		conditions = append(conditions, "r.publish_date >= ?")
		args = append(args, *filters.PublishedAfter)
		countArgs = append(countArgs, *filters.PublishedAfter)
	}
	if filters.PublishedBefore != nil {
		conditions = append(conditions, "r.publish_date <= ?")
		args = append(args, *filters.PublishedBefore)
		countArgs = append(countArgs, *filters.PublishedBefore)
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM reports r
		LEFT JOIN categories c ON r.category_id = c.id
		WHERE %s
	`, whereClause)

	if err := r.db.Raw(countSQL, countArgs...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// Determine sort order
	orderClause := "COALESCE(r.id) DESC"
	if filters.SortBy != "" {
		orderClause = filters.SortBy
	}

	// Fetch query
	querySQL := fmt.Sprintf(`
		SELECT r.*, c.name as category_name, c.image_url as category_image_url
		FROM reports r
		LEFT JOIN categories c ON r.category_id = c.id
		WHERE %s
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, whereClause, orderClause)

	args = append(args, filters.Limit, offset)
	err := r.db.Raw(querySQL, args...).Scan(&reports).Error

	return reports, total, err
}

func (r *reportRepository) GetBySlug(slug string) (*report.ReportWithRelations, error) {
	var result report.ReportWithRelations

	// Use raw SQL to properly handle JSONB fields and joins
	querySQL := `
		SELECT r.*, c.name as category_name, c.image_url as category_image_url
		FROM reports r
		INNER JOIN categories c ON r.category_id = c.id AND c.is_active = true
		WHERE r.slug = $1 AND r.deleted_at IS NULL
	`

	err := r.db.Raw(querySQL, slug).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	// Get authors
	if len(result.AuthorIDs) > 0 {
		authors, _ := r.authorRepo.GetByIDs(result.AuthorIDs)
		result.Authors = authors
	}

	// Get charts
	charts, _ := r.GetChartsByReportID(result.ID)
	result.Charts = charts

	// Get versions
	versions, _ := r.GetVersionsByReportID(result.ID)
	result.Versions = versions

	return &result, nil
}

func (r *reportRepository) GetByCategorySlug(categorySlug string, page, limit int) ([]report.Report, int64, error) {
	var reports []report.Report
	var total int64

	offset := (page - 1) * limit

	countSQL := `
		SELECT COUNT(*)
		FROM reports r
		INNER JOIN categories c ON r.category_id = c.id
		WHERE c.slug = ? AND c.is_active = true AND r.deleted_at IS NULL
	`
	if err := r.db.Raw(countSQL, categorySlug).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	querySQL := `
		SELECT r.*, c.name as category_name, c.image_url as category_image_url
		FROM reports r
		INNER JOIN categories c ON r.category_id = c.id
		WHERE c.slug = ? AND c.is_active = true AND r.deleted_at IS NULL
		ORDER BY COALESCE(r.id) DESC
		LIMIT ? OFFSET ?
	`

	err := r.db.Raw(querySQL, categorySlug, limit, offset).Scan(&reports).Error

	return reports, total, err
}

func (r *reportRepository) GetByAuthorID(authorID uint, page, limit int) ([]report.Report, int64, error) {
	var reports []report.Report
	var total int64
	offset := (page - 1) * limit

	// Use @> (contains) operator instead of ? operator to avoid GORM conflicts
	// Build the JSONB array string to check if author_ids contains the authorID
	authorIDStr := fmt.Sprintf("[%d]", authorID)

	// Count query - JSONB containment check, exclude soft-deleted
	countSQL := `
		SELECT COUNT(*)
		FROM reports r
		WHERE r.author_ids::jsonb @> ?::jsonb AND r.deleted_at IS NULL
	`
	if err := r.db.Raw(countSQL, authorIDStr).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// Data query with category join and pagination, exclude soft-deleted
	querySQL := `
		SELECT r.*, c.name as category_name, c.image_url as category_image_url
		FROM reports r
		LEFT JOIN categories c ON r.category_id = c.id
		WHERE r.author_ids::jsonb @> ?::jsonb AND r.deleted_at IS NULL
		ORDER BY COALESCE(r.id) DESC
		LIMIT ? OFFSET ?
	`
	err := r.db.Raw(querySQL, authorIDStr, limit, offset).Scan(&reports).Error

	return reports, total, err
}

func (r *reportRepository) Search(query string, page, limit int) ([]report.Report, int64, error) {
	var reports []report.Report
	var total int64

	offset := (page - 1) * limit
	searchPattern := "%" + query + "%"

	countSQL := `
		SELECT COUNT(*)
		FROM reports r
		LEFT JOIN categories c ON r.category_id = c.id
		WHERE (r.title ILIKE ? OR r.description ILIKE ? OR r.summary ILIKE ?) AND r.deleted_at IS NULL
	`
	if err := r.db.Raw(countSQL, searchPattern, searchPattern, searchPattern).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	querySQL := `
		SELECT r.*, c.name as category_name, c.image_url as category_image_url
		FROM reports r
		LEFT JOIN categories c ON r.category_id = c.id
		WHERE (r.title ILIKE ? OR r.description ILIKE ? OR r.summary ILIKE ?) AND r.deleted_at IS NULL
		ORDER BY COALESCE(r.id) DESC
		LIMIT ? OFFSET ?
	`

	err := r.db.Raw(querySQL, searchPattern, searchPattern, searchPattern, limit, offset).Scan(&reports).Error

	return reports, total, err
}

func (r *reportRepository) GetChartsByReportID(reportID uint) ([]report.ChartMetadata, error) {
	var charts []report.ChartMetadata

	// Use raw SQL for better performance
	querySQL := `
		SELECT id, report_id, title, chart_type, description,
		       data_points, "order", is_active, created_at, updated_at
		FROM chart_metadata
		WHERE report_id = ? AND is_active = true
		ORDER BY "order" ASC
	`

	err := r.db.Raw(querySQL, reportID).Scan(&charts).Error
	return charts, err
}

func (r *reportRepository) Create(rep *report.Report) error {
	return r.db.Create(rep).Error
}

func (r *reportRepository) Update(rep *report.Report) error {
	return r.db.Save(rep).Error
}

func (r *reportRepository) Delete(id uint) error {
	return r.db.Delete(&report.Report{}, id).Error
}

func (r *reportRepository) GetByID(id uint) (*report.Report, error) {
	var rep report.Report
	err := r.db.First(&rep, id).Error
	if err != nil {
		return nil, err
	}
	return &rep, nil
}

func (r *reportRepository) GetByIDWithRelations(id uint) (*report.ReportWithRelations, error) {
	var result report.ReportWithRelations

	// Use raw SQL to properly handle JSONB fields and joins
	querySQL := `
		SELECT r.*, c.name as category_name, c.image_url as category_image_url
		FROM reports r
		INNER JOIN categories c ON r.category_id = c.id AND c.is_active = true
		WHERE r.id = $1 AND r.deleted_at IS NULL
	`

	err := r.db.Raw(querySQL, id).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	// Get authors
	if len(result.AuthorIDs) > 0 {
		authors, _ := r.authorRepo.GetByIDs(result.AuthorIDs)
		result.Authors = authors
	}

	// Get charts
	charts, _ := r.GetChartsByReportID(result.ID)
	result.Charts = charts

	// Get versions
	versions, _ := r.GetVersionsByReportID(result.ID)
	result.Versions = versions

	return &result, nil
}

// Version history methods

func (r *reportRepository) CreateVersion(version *report.ReportVersion) error {
	return r.db.Create(version).Error
}

func (r *reportRepository) GetVersionsByReportID(reportID uint) ([]report.ReportVersion, error) {
	var versions []report.ReportVersion
	err := r.db.Where("report_id = ?", reportID).
		Order("version_number DESC").
		Find(&versions).Error
	return versions, err
}

func (r *reportRepository) GetLatestVersionNumber(reportID uint) (int, error) {
	var maxVersion int
	err := r.db.Model(&report.ReportVersion{}).
		Where("report_id = ?", reportID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(&maxVersion).Error
	return maxVersion, err
}

func (r *reportRepository) SoftDelete(id uint) error {
	now := time.Now()
	return r.db.Model(&report.Report{}).Where("id = ?", id).Update("deleted_at", now).Error
}

func (r *reportRepository) Restore(id uint) error {
	return r.db.Model(&report.Report{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

func (r *reportRepository) PublishScheduled(now time.Time) error {
	return r.db.Model(&report.Report{}).
		Where("scheduled_publish_enabled = ? AND status != ? AND publish_date <= ?",
			true, "published", now).
		Updates(map[string]interface{}{
			"status":                    "published",
			"scheduled_publish_enabled": false,
		}).Error
}

func (r *reportRepository) SchedulePublish(id uint, publishDate time.Time) error {
	return r.db.Model(&report.Report{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"publish_date":              publishDate,
			"scheduled_publish_enabled": true,
		}).Error
}

func (r *reportRepository) CancelScheduledPublish(id uint) error {
	return r.db.Model(&report.Report{}).
		Where("id = ?", id).
		Update("scheduled_publish_enabled", false).Error
}

func (r *reportRepository) GetLinkSuggestions() ([]report.LinkSuggestionItem, error) {
	var items []report.LinkSuggestionItem
	err := r.db.Raw(`
		SELECT id, title, slug, meta_keywords
		FROM reports
		WHERE status = 'published' AND deleted_at IS NULL
		ORDER BY title
	`).Scan(&items).Error
	return items, err
}
