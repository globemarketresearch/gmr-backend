package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/healthcare-market-research/backend/internal/repository"
)

type ReportService interface {
	GetAll(page, limit int) ([]report.Report, int64, error)
	GetAllWithFilters(filters repository.ReportFilters) ([]report.Report, int64, error)
	GetByID(id uint) (*report.ReportWithRelations, error)
	GetBySlug(slug string) (*report.ReportWithRelations, error)
	GetByCategorySlug(categorySlug string, page, limit int) ([]report.Report, int64, error)
	GetByAuthorID(authorID uint, page, limit int) ([]report.Report, int64, error)
	Search(query string, page, limit int) ([]report.Report, int64, error)
	Create(rep *report.Report, userID uint) error
	Update(id uint, rep *report.Report, userID uint) error
	Delete(id uint) error
	SoftDelete(id uint) error
	Restore(id uint) error
	SchedulePublish(id uint, publishDate time.Time) (*report.Report, error)
	CancelScheduledPublish(id uint) (*report.Report, error)
	GetLinkSuggestions() ([]report.LinkSuggestionItem, error)
}

type reportService struct {
	repo              repository.ReportRepository
	reportImageRepo   repository.ReportImageRepository
	cloudflareService CloudflareImagesService
}

func NewReportService(repo repository.ReportRepository, reportImageRepo repository.ReportImageRepository, cloudflareService CloudflareImagesService) ReportService {
	return &reportService{
		repo:              repo,
		reportImageRepo:   reportImageRepo,
		cloudflareService: cloudflareService,
	}
}

func (s *reportService) GetAll(page, limit int) ([]report.Report, int64, error) {
	// Always fetch fresh data from database (no caching)
	return s.repo.GetAll(page, limit)
}

func (s *reportService) GetByID(id uint) (*report.ReportWithRelations, error) {
	// Always fetch fresh data from database (no caching for ID-based access)
	return s.repo.GetByIDWithRelations(id)
}

func (s *reportService) GetBySlug(slug string) (*report.ReportWithRelations, error) {
	// Always fetch fresh data from database (no caching)
	return s.repo.GetBySlug(slug)
}

func (s *reportService) GetByCategorySlug(categorySlug string, page, limit int) ([]report.Report, int64, error) {
	// Always fetch fresh data from database (no caching)
	return s.repo.GetByCategorySlug(categorySlug, page, limit)
}

func (s *reportService) GetByAuthorID(authorID uint, page, limit int) ([]report.Report, int64, error) {
	// No caching for filtered queries (matches blog/press release pattern)
	return s.repo.GetByAuthorID(authorID, page, limit)
}

func (s *reportService) GetAllWithFilters(filters repository.ReportFilters) ([]report.Report, int64, error) {
	// Don't cache filtered results due to high variability
	return s.repo.GetAllWithFilters(filters)
}

func (s *reportService) Search(query string, page, limit int) ([]report.Report, int64, error) {
	// Search queries are not cached due to high variability
	return s.repo.Search(query, page, limit)
}

func (s *reportService) Create(rep *report.Report, userID uint) error {
	// Set user tracking fields
	rep.CreatedBy = &userID
	rep.UpdatedBy = &userID

	err := s.repo.Create(rep)
	if err != nil {
		return err
	}

	// Invalidate all report list caches
	cache.DeletePattern("reports:list:*")
	cache.DeletePattern("reports:total")

	// Invalidate category-specific caches if category is set
	if rep.CategoryID > 0 {
		cache.DeletePattern(fmt.Sprintf("reports:category:*"))
	}

	return nil
}

func (s *reportService) Update(id uint, rep *report.Report, userID uint) error {
	// Get existing report to check slug and status
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	rep.ID = id

	// Set updated_by field
	rep.UpdatedBy = &userID

	// Check if status is changing from draft to published
	statusChanged := existing.Status == "draft" && rep.Status == "published"

	// Update the report
	err = s.repo.Update(rep)
	if err != nil {
		return err
	}

	// If status changed to published, create a version history entry
	if statusChanged {
		// Get the latest version number
		latestVersion, err := s.repo.GetLatestVersionNumber(id)
		if err != nil {
			// Log error but don't fail the update
			fmt.Printf("Warning: could not get latest version number: %v\n", err)
		} else {
			// Create new version
			version := &report.ReportVersion{
				ReportID:        id,
				VersionNumber:   latestVersion + 1,
				PublishedBy:     userID,
				PublishedAt:     time.Now(),
				Sections:        rep.Sections,
				MetaTitle:       rep.MetaTitle,
				MetaDescription: rep.MetaDescription,
				MetaKeywords:    rep.MetaKeywords,
			}

			if err := s.repo.CreateVersion(version); err != nil {
				// Log error but don't fail the update
				fmt.Printf("Warning: could not create version history: %v\n", err)
			}
		}

		// Update publish_date if not set
		if rep.PublishDate == nil {
			now := time.Now()
			rep.PublishDate = &now
			s.repo.Update(rep) // Update again with publish_date
		}
	}

	// Invalidate caches
	cache.DeletePattern("reports:list:*")
	cache.DeletePattern("reports:total")
	cache.DeletePattern(fmt.Sprintf("reports:category:*"))

	// Invalidate the specific report cache (old slug)
	cache.Delete(fmt.Sprintf("report:slug:%s", existing.Slug))

	// Invalidate new slug if it changed
	if rep.Slug != existing.Slug {
		cache.Delete(fmt.Sprintf("report:slug:%s", rep.Slug))
	}

	return nil
}

func (s *reportService) Delete(id uint) error {
	// Get the report to invalidate slug-based cache
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	// Get all images for this report and delete from Cloudflare (best effort)
	images, _ := s.reportImageRepo.FindByReportID(id)
	for _, img := range images {
		if err := s.cloudflareService.Delete(img.ImageURL); err != nil {
			// Log error but don't fail the deletion
			log.Printf("Failed to delete image from Cloudflare: %s, error: %v", img.ImageURL, err)
		}
	}

	// Delete report (CASCADE will delete DB image records automatically)
	err = s.repo.Delete(id)
	if err != nil {
		return err
	}

	// Invalidate caches
	cache.DeletePattern("reports:list:*")
	cache.DeletePattern("reports:total")
	cache.DeletePattern(fmt.Sprintf("reports:category:*"))
	cache.Delete(fmt.Sprintf("report:slug:%s", existing.Slug))

	return nil
}

func (s *reportService) SoftDelete(id uint) error {
	// Get the report to invalidate slug-based cache
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	// Soft delete the report
	err = s.repo.SoftDelete(id)
	if err != nil {
		return err
	}

	// Invalidate caches (same pattern as hard delete)
	cache.DeletePattern("reports:list:*")
	cache.DeletePattern("reports:total")
	cache.DeletePattern(fmt.Sprintf("reports:category:*"))
	cache.Delete(fmt.Sprintf("report:slug:%s", existing.Slug))

	return nil
}

func (s *reportService) Restore(id uint) error {
	// Restore the report
	err := s.repo.Restore(id)
	if err != nil {
		return err
	}

	// Invalidate caches to show restored report
	cache.DeletePattern("reports:list:*")
	cache.DeletePattern("reports:total")
	cache.DeletePattern(fmt.Sprintf("reports:category:*"))

	return nil
}

func (s *reportService) SchedulePublish(id uint, publishDate time.Time) (*report.Report, error) {
	// Validate publishDate is in future
	if publishDate.Before(time.Now()) {
		return nil, errors.New("publish date must be in the future")
	}

	// Get existing report
	r, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate status (can't schedule already published)
	if r.Status == "published" {
		return nil, errors.New("cannot schedule already published report")
	}

	// Schedule publish
	if err := s.repo.SchedulePublish(id, publishDate); err != nil {
		return nil, err
	}

	return s.repo.GetByID(id)
}

func (s *reportService) CancelScheduledPublish(id uint) (*report.Report, error) {
	// Get existing report
	_, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Cancel schedule
	if err := s.repo.CancelScheduledPublish(id); err != nil {
		return nil, err
	}

	return s.repo.GetByID(id)
}

func (s *reportService) GetLinkSuggestions() ([]report.LinkSuggestionItem, error) {
	return s.repo.GetLinkSuggestions()
}
