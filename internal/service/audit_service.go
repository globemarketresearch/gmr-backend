package service

import (
	"fmt"

	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/pkg/logger"
)

// AuditService defines the interface for audit logging operations
type AuditService interface {
	Log(entry *audit.AuditEntry) error
	LogAsync(entry *audit.AuditEntry)
	GetByID(id uint) (*audit.AuditLogResponse, error)
	GetAll(filters audit.AuditLogFilters) ([]audit.AuditLogResponse, int64, error)
}

type auditService struct {
	repo    repository.AuditRepository
	logChan chan *audit.AuditEntry
}

// NewAuditService creates a new audit service instance with background worker
func NewAuditService(repo repository.AuditRepository) AuditService {
	s := &auditService{
		repo:    repo,
		logChan: make(chan *audit.AuditEntry, 100), // Buffered channel for 100 entries
	}

	// Start background worker goroutine
	go s.processAuditLogs()

	logger.Info("Audit service initialized with background worker")

	return s
}

// Log synchronously creates an audit log entry
// Use this when you need to ensure the log is persisted before continuing
func (s *auditService) Log(entry *audit.AuditEntry) error {
	log := entry.ToAuditLog()
	if err := s.repo.Create(log); err != nil {
		logger.Error("Failed to create audit log synchronously", "error", err, "action", entry.Action)
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// LogAsync asynchronously creates an audit log entry (non-blocking)
// This is the preferred method for most audit logging to avoid blocking main operations
func (s *auditService) LogAsync(entry *audit.AuditEntry) {
	select {
	case s.logChan <- entry:
		// Successfully queued for processing
	default:
		// Channel is full - log warning but don't block
		logger.Warn("Audit log channel full, dropping log entry", "action", entry.Action, "user_id", entry.UserID)
	}
}

// processAuditLogs is a background worker that processes queued audit logs
func (s *auditService) processAuditLogs() {
	for entry := range s.logChan {
		log := entry.ToAuditLog()
		if err := s.repo.Create(log); err != nil {
			// Log error but continue processing
			logger.Error("Failed to create audit log in background worker", "error", err, "action", entry.Action)
		}
	}
}

// GetByID retrieves a specific audit log by ID
func (s *auditService) GetByID(id uint) (*audit.AuditLogResponse, error) {
	log, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}
	return log.ToResponse(), nil
}

// GetAll retrieves audit logs with filtering and pagination
func (s *auditService) GetAll(filters audit.AuditLogFilters) ([]audit.AuditLogResponse, int64, error) {
	logs, total, err := s.repo.GetAll(filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs: %w", err)
	}

	responses := make([]audit.AuditLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = *log.ToResponse()
	}

	return responses, total, nil
}
