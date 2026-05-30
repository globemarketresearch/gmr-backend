package service

import (
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/form"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/pkg/email"
	"github.com/healthcare-market-research/backend/pkg/logger"
)

type FormService interface {
	Create(req *form.CreateSubmissionRequest) (*form.SubmissionResponse, error)
	GetAll(query form.GetSubmissionsQuery) ([]form.FormSubmission, int64, error)
	GetByID(id uint) (*form.FormSubmission, error)
	GetByCategory(category string, page, limit int) ([]form.FormSubmission, int64, error)
	Delete(id uint) error
	BulkDelete(ids []uint) (int64, error)
	GetStats() (*form.SubmissionStats, error)
	UpdateStatus(id uint, status form.FormStatus, processedBy *uint) error
}

type formService struct {
	repo     repository.FormRepository
	emailSvc email.EmailService
}

func NewFormService(repo repository.FormRepository, emailSvc email.EmailService) FormService {
	return &formService{repo: repo, emailSvc: emailSvc}
}

func (s *formService) Create(req *form.CreateSubmissionRequest) (*form.SubmissionResponse, error) {
	// Validate category
	validCategories := map[form.FormCategory]bool{
		form.CategoryContact:              true,
		form.CategoryRequestSample:        true,
		form.CategoryRequestCustomization: true,
		form.CategoryScheduleDemo:         true,
		form.CategoryNewsletter:           true,
		form.CategoryPublishNews:          true,
	}
	if !validCategories[req.Category] {
		return nil, fmt.Errorf("invalid category: must be one of contact, request-sample, request-customization, schedule-demo, newsletter, publish-news")
	}

	// Validate required fields based on category
	if err := s.validateFormData(req.Category, req.Data); err != nil {
		return nil, err
	}

	// Set default metadata timestamp if not provided
	if req.Metadata.SubmittedAt == "" {
		req.Metadata.SubmittedAt = time.Now().Format(time.RFC3339)
	}

	submission := &form.FormSubmission{
		Category: req.Category,
		Status:   form.StatusPending,
		Data:     req.Data,
		Metadata: req.Metadata,
	}

	if err := s.repo.Create(submission); err != nil {
		return nil, err
	}

	// Send email notification asynchronously (fire-and-forget)
	go func() {
		if err := s.emailSvc.SendFormNotification(submission); err != nil {
			logger.Error("Failed to send form notification email", "error", err)
		}
	}()

	// Invalidate caches
	cache.DeletePattern("forms:*")

	return &form.SubmissionResponse{
		Success:      true,
		SubmissionID: submission.ID,
		Category:     req.Category,
		Message:      "Form submitted successfully",
		CreatedAt:    submission.CreatedAt,
	}, nil
}

func (s *formService) validateFormData(category form.FormCategory, data form.FormData) error {
	// Newsletter and publish-news only require email
	if category == form.CategoryNewsletter || category == form.CategoryPublishNews {
		if data["email"] == nil || data["email"] == "" {
			return fmt.Errorf("email is required")
		}
		return nil
	}

	// Common required fields for all other categories
	if data["fullName"] == nil || data["fullName"] == "" {
		return fmt.Errorf("fullName is required")
	}
	if data["email"] == nil || data["email"] == "" {
		return fmt.Errorf("email is required")
	}
	if data["company"] == nil || data["company"] == "" {
		return fmt.Errorf("company is required")
	}

	// Category-specific validation
	if category == form.CategoryContact {
		if data["subject"] == nil || data["subject"] == "" {
			return fmt.Errorf("subject is required for contact form")
		}
		if data["message"] == nil || data["message"] == "" {
			return fmt.Errorf("message is required for contact form")
		}
	}

	if category == form.CategoryRequestSample || category == form.CategoryRequestCustomization {
		if data["jobTitle"] == nil || data["jobTitle"] == "" {
			return fmt.Errorf("jobTitle is required for request sample form")
		}
		if data["reportTitle"] == nil || data["reportTitle"] == "" {
			return fmt.Errorf("reportTitle is required for request sample form")
		}
	}

	return nil
}

func (s *formService) GetAll(query form.GetSubmissionsQuery) ([]form.FormSubmission, int64, error) {
	// Only cache non-filtered, non-search queries
	shouldCache := query.Category == "" && query.Status == "" && query.DateFrom == "" &&
		query.DateTo == "" && query.Search == "" && query.SortBy == "" && query.SortOrder == ""

	if shouldCache {
		cacheKey := fmt.Sprintf("forms:list:%d:%d", query.Page, query.Limit)

		type result struct {
			Submissions []form.FormSubmission `json:"submissions"`
			Total       int64                 `json:"total"`
		}

		var res result

		err := cache.GetOrSet(cacheKey, &res, 5*time.Minute, func() (interface{}, error) {
			submissions, total, err := s.repo.GetAll(query)
			if err != nil {
				return nil, err
			}
			return result{Submissions: submissions, Total: total}, nil
		})

		if err != nil {
			return nil, 0, err
		}

		return res.Submissions, res.Total, nil
	}

	// Don't cache filtered/search results
	return s.repo.GetAll(query)
}

func (s *formService) GetByID(id uint) (*form.FormSubmission, error) {
	cacheKey := fmt.Sprintf("form:id:%d", id)

	var submission form.FormSubmission

	err := cache.GetOrSet(cacheKey, &submission, 10*time.Minute, func() (interface{}, error) {
		return s.repo.GetByID(id)
	})

	if err != nil {
		return nil, err
	}

	return &submission, nil
}

func (s *formService) GetByCategory(category string, page, limit int) ([]form.FormSubmission, int64, error) {
	cacheKey := fmt.Sprintf("forms:category:%s:%d:%d", category, page, limit)

	type result struct {
		Submissions []form.FormSubmission `json:"submissions"`
		Total       int64                 `json:"total"`
	}

	var res result

	err := cache.GetOrSet(cacheKey, &res, 5*time.Minute, func() (interface{}, error) {
		submissions, total, err := s.repo.GetByCategory(category, page, limit)
		if err != nil {
			return nil, err
		}
		return result{Submissions: submissions, Total: total}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	return res.Submissions, res.Total, nil
}

func (s *formService) Delete(id uint) error {
	err := s.repo.Delete(id)
	if err != nil {
		return err
	}

	// Invalidate caches
	cache.DeletePattern("forms:*")
	cache.Delete(fmt.Sprintf("form:id:%d", id))

	return nil
}

func (s *formService) BulkDelete(ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("no IDs provided")
	}

	deletedCount, err := s.repo.BulkDelete(ids)
	if err != nil {
		return 0, err
	}

	// Invalidate caches
	cache.DeletePattern("forms:*")
	for _, id := range ids {
		cache.Delete(fmt.Sprintf("form:id:%d", id))
	}

	return deletedCount, nil
}

func (s *formService) GetStats() (*form.SubmissionStats, error) {
	cacheKey := "forms:stats"

	var stats form.SubmissionStats

	err := cache.GetOrSet(cacheKey, &stats, 2*time.Minute, func() (interface{}, error) {
		return s.repo.GetStats()
	})

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (s *formService) UpdateStatus(id uint, status form.FormStatus, processedBy *uint) error {
	// Validate status
	if status != form.StatusPending && status != form.StatusProcessed && status != form.StatusArchived {
		return fmt.Errorf("invalid status: must be 'pending', 'processed', or 'archived'")
	}

	err := s.repo.UpdateStatus(id, status, processedBy)
	if err != nil {
		return err
	}

	// Invalidate caches
	cache.DeletePattern("forms:*")
	cache.Delete(fmt.Sprintf("form:id:%d", id))

	return nil
}
