package service

import (
	"fmt"
	"log"
	"mime/multipart"

	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/healthcare-market-research/backend/internal/repository"
)

type ReportImageService interface {
	UploadImage(reportID uint, file *multipart.FileHeader, title string, uploadedBy uint) (*report.ReportImage, error)
	UpdateImageMetadata(imageID uint, title *string, isActive *bool) (*report.ReportImage, error)
	DeleteImage(imageID uint) error
	GetImagesByReport(reportID uint, activeOnly bool) ([]report.ReportImage, error)
	GetImageByID(imageID uint) (*report.ReportImage, error)
}

type reportImageService struct {
	reportImageRepo   repository.ReportImageRepository
	reportRepo        repository.ReportRepository
	cloudflareService CloudflareImagesService
}

func NewReportImageService(
	reportImageRepo repository.ReportImageRepository,
	reportRepo repository.ReportRepository,
	cloudflareService CloudflareImagesService,
) ReportImageService {
	return &reportImageService{
		reportImageRepo:   reportImageRepo,
		reportRepo:        reportRepo,
		cloudflareService: cloudflareService,
	}
}

// UploadImage uploads an image for a report with rollback on failure
func (s *reportImageService) UploadImage(reportID uint, file *multipart.FileHeader, title string, uploadedBy uint) (*report.ReportImage, error) {
	// Validate that the report exists
	_, err := s.reportRepo.GetByID(reportID)
	if err != nil {
		return nil, fmt.Errorf("report not found: %w", err)
	}

	// Upload to Cloudflare with metadata
	metadata := map[string]string{
		"report_id":   fmt.Sprintf("%d", reportID),
		"type":        "report_image",
		"uploaded_by": fmt.Sprintf("%d", uploadedBy),
	}

	imageURL, err := s.cloudflareService.Upload(file, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	// Create database record
	image := &report.ReportImage{
		ReportID:   reportID,
		ImageURL:   imageURL,
		Title:      title,
		IsActive:   true,
		UploadedBy: &uploadedBy,
	}

	if err := s.reportImageRepo.Create(image); err != nil {
		// Rollback: delete the uploaded image from Cloudflare
		if deleteErr := s.cloudflareService.Delete(imageURL); deleteErr != nil {
			log.Printf("Warning: Failed to rollback image upload for report %d: %v", reportID, deleteErr)
		}
		return nil, fmt.Errorf("failed to create image record: %w", err)
	}

	return image, nil
}

// UpdateImageMetadata updates image title and/or is_active status (partial updates supported)
func (s *reportImageService) UpdateImageMetadata(imageID uint, title *string, isActive *bool) (*report.ReportImage, error) {
	// Get the existing image
	image, err := s.reportImageRepo.FindByID(imageID)
	if err != nil {
		return nil, fmt.Errorf("image not found: %w", err)
	}

	// Apply partial updates
	if title != nil {
		image.Title = *title
	}

	if isActive != nil {
		image.IsActive = *isActive
	}

	// Update in database
	if err := s.reportImageRepo.Update(image); err != nil {
		return nil, fmt.Errorf("failed to update image: %w", err)
	}

	return image, nil
}

// DeleteImage permanently deletes an image from both Cloudflare R2 and the database
func (s *reportImageService) DeleteImage(imageID uint) error {
	image, err := s.reportImageRepo.FindByID(imageID)
	if err != nil {
		return fmt.Errorf("image not found: %w", err)
	}

	// Delete from Cloudflare R2
	if err := s.cloudflareService.Delete(image.ImageURL); err != nil {
		log.Printf("Warning: Failed to delete image from Cloudflare for image %d: %v", imageID, err)
	}

	// Hard delete from database
	if err := s.reportImageRepo.Delete(imageID); err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}

// GetImagesByReport retrieves all images for a report with optional active filter
func (s *reportImageService) GetImagesByReport(reportID uint, activeOnly bool) ([]report.ReportImage, error) {
	if activeOnly {
		return s.reportImageRepo.FindActiveByReportID(reportID)
	}
	return s.reportImageRepo.FindByReportID(reportID)
}

// GetImageByID retrieves a single image by ID
func (s *reportImageService) GetImageByID(imageID uint) (*report.ReportImage, error) {
	image, err := s.reportImageRepo.FindByID(imageID)
	if err != nil {
		return nil, fmt.Errorf("image not found: %w", err)
	}
	return image, nil
}
