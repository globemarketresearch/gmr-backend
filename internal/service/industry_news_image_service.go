package service

import (
	"fmt"
	"log"
	"mime/multipart"

	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
	"github.com/healthcare-market-research/backend/internal/repository"
)

type IndustryNewsImageService interface {
	UploadImage(industryNewsID uint, file *multipart.FileHeader, title string, uploadedBy uint) (*industry_news.IndustryNewsImage, error)
	UpdateImageMetadata(imageID uint, title *string, isActive *bool) (*industry_news.IndustryNewsImage, error)
	DeleteImage(imageID uint) error
	GetImagesByIndustryNews(industryNewsID uint, activeOnly bool) ([]industry_news.IndustryNewsImage, error)
	GetImageByID(imageID uint) (*industry_news.IndustryNewsImage, error)
}

type industryNewsImageService struct {
	industryNewsImageRepo repository.IndustryNewsImageRepository
	industryNewsRepo      repository.IndustryNewsRepository
	cloudflareService     CloudflareImagesService
}

func NewIndustryNewsImageService(
	industryNewsImageRepo repository.IndustryNewsImageRepository,
	industryNewsRepo repository.IndustryNewsRepository,
	cloudflareService CloudflareImagesService,
) IndustryNewsImageService {
	return &industryNewsImageService{
		industryNewsImageRepo: industryNewsImageRepo,
		industryNewsRepo:      industryNewsRepo,
		cloudflareService:     cloudflareService,
	}
}

func (s *industryNewsImageService) UploadImage(industryNewsID uint, file *multipart.FileHeader, title string, uploadedBy uint) (*industry_news.IndustryNewsImage, error) {
	_, err := s.industryNewsRepo.GetByID(industryNewsID)
	if err != nil {
		return nil, fmt.Errorf("industry news article not found: %w", err)
	}

	metadata := map[string]string{
		"industry_news_id": fmt.Sprintf("%d", industryNewsID),
		"type":             "industry_news_image",
		"uploaded_by":      fmt.Sprintf("%d", uploadedBy),
	}

	imageURL, err := s.cloudflareService.Upload(file, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	image := &industry_news.IndustryNewsImage{
		IndustryNewsID: industryNewsID,
		ImageURL:       imageURL,
		Title:          title,
		IsActive:       true,
		UploadedBy:     &uploadedBy,
	}

	if err := s.industryNewsImageRepo.Create(image); err != nil {
		if deleteErr := s.cloudflareService.Delete(imageURL); deleteErr != nil {
			log.Printf("Warning: Failed to rollback image upload for industry news %d: %v", industryNewsID, deleteErr)
		}
		return nil, fmt.Errorf("failed to create image record: %w", err)
	}

	return image, nil
}

func (s *industryNewsImageService) UpdateImageMetadata(imageID uint, title *string, isActive *bool) (*industry_news.IndustryNewsImage, error) {
	image, err := s.industryNewsImageRepo.FindByID(imageID)
	if err != nil {
		return nil, fmt.Errorf("image not found: %w", err)
	}

	if title != nil {
		image.Title = *title
	}

	if isActive != nil {
		image.IsActive = *isActive
	}

	if err := s.industryNewsImageRepo.Update(image); err != nil {
		return nil, fmt.Errorf("failed to update image: %w", err)
	}

	return image, nil
}

func (s *industryNewsImageService) DeleteImage(imageID uint) error {
	image, err := s.industryNewsImageRepo.FindByID(imageID)
	if err != nil {
		return fmt.Errorf("image not found: %w", err)
	}

	if err := s.cloudflareService.Delete(image.ImageURL); err != nil {
		log.Printf("Warning: Failed to delete image from Cloudflare for image %d: %v", imageID, err)
	}

	if err := s.industryNewsImageRepo.Delete(imageID); err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}

func (s *industryNewsImageService) GetImagesByIndustryNews(industryNewsID uint, activeOnly bool) ([]industry_news.IndustryNewsImage, error) {
	if activeOnly {
		return s.industryNewsImageRepo.FindActiveByIndustryNewsID(industryNewsID)
	}
	return s.industryNewsImageRepo.FindByIndustryNewsID(industryNewsID)
}

func (s *industryNewsImageService) GetImageByID(imageID uint) (*industry_news.IndustryNewsImage, error) {
	image, err := s.industryNewsImageRepo.FindByID(imageID)
	if err != nil {
		return nil, fmt.Errorf("image not found: %w", err)
	}
	return image, nil
}
