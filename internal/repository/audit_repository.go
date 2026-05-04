package repository

import (
	"strings"

	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"gorm.io/gorm"
)

// AuditRepository defines the interface for audit log data access
type AuditRepository interface {
	Create(log *audit.AuditLog) error
	GetByID(id uint) (*audit.AuditLog, error)
	GetAll(filters audit.AuditLogFilters) ([]audit.AuditLog, int64, error)
}

type auditRepository struct {
	db *gorm.DB
}

// NewAuditRepository creates a new audit repository instance
func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

// Create inserts a new audit log entry
func (r *auditRepository) Create(log *audit.AuditLog) error {
	return r.db.Create(log).Error
}

// GetByID retrieves a specific audit log by ID
func (r *auditRepository) GetByID(id uint) (*audit.AuditLog, error) {
	var log audit.AuditLog
	err := r.db.First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetAll retrieves audit logs with filtering and pagination
func (r *auditRepository) GetAll(filters audit.AuditLogFilters) ([]audit.AuditLog, int64, error) {
	var logs []audit.AuditLog
	var total int64

	query := r.db.Model(&audit.AuditLog{})

	// Apply filters
	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.ActionPrefix != "" {
		prefixes := strings.Split(filters.ActionPrefix, ",")
		if len(prefixes) == 1 {
			query = query.Where("action LIKE ?", strings.TrimSpace(prefixes[0])+"%")
		} else {
			orCond := r.db.Where("action LIKE ?", strings.TrimSpace(prefixes[0])+"%")
			for _, p := range prefixes[1:] {
				orCond = orCond.Or("action LIKE ?", strings.TrimSpace(p)+"%")
			}
			query = query.Where(orCond)
		}
	}
	if filters.EntityType != "" {
		query = query.Where("entity_type = ?", filters.EntityType)
	}
	if filters.EntityID != nil {
		query = query.Where("entity_id = ?", *filters.EntityID)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.StartDate != nil {
		query = query.Where("created_at >= ?", filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("created_at <= ?", filters.EndDate)
	}
	if filters.IPAddress != "" {
		query = query.Where("ip_address = ?", filters.IPAddress)
	}

	// Get total count before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (filters.Page - 1) * filters.Limit
	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(filters.Limit).
		Find(&logs).Error

	return logs, total, err
}
