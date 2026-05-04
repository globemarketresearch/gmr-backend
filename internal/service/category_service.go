package service

import (
	"fmt"
	"log"
	"mime/multipart"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/category"
	"github.com/healthcare-market-research/backend/internal/repository"
)

type CategoryService interface {
	GetAll(page, limit int) ([]category.Category, int64, error)
	GetBySlug(slug string) (*category.Category, error)
	UploadImage(categoryID uint, file *multipart.FileHeader) (*category.Category, error)
}

type categoryService struct {
	repo              repository.CategoryRepository
	cloudflareService CloudflareImagesService
}

func NewCategoryService(repo repository.CategoryRepository, cloudflareService CloudflareImagesService) CategoryService {
	return &categoryService{repo: repo, cloudflareService: cloudflareService}
}

func (s *categoryService) GetAll(page, limit int) ([]category.Category, int64, error) {
	cacheKey := fmt.Sprintf("categories:list:%d:%d", page, limit)
	totalKey := "categories:total"

	type result struct {
		Categories []category.Category `json:"categories"`
		Total      int64               `json:"total"`
	}

	var res result

	// Use singleflight-protected cache-aside pattern
	err := cache.GetOrSet(cacheKey, &res, 10*time.Minute, func() (interface{}, error) {
		categories, total, err := s.repo.GetAll(page, limit)
		if err != nil {
			return nil, err
		}

		// Also cache total separately
		cache.Set(totalKey, total, 10*time.Minute)

		return result{Categories: categories, Total: total}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	return res.Categories, res.Total, nil
}

func (s *categoryService) GetBySlug(slug string) (*category.Category, error) {
	cacheKey := fmt.Sprintf("category:slug:%s", slug)

	var cat category.Category

	// Use singleflight-protected cache-aside pattern
	err := cache.GetOrSet(cacheKey, &cat, 30*time.Minute, func() (interface{}, error) {
		return s.repo.GetBySlug(slug)
	})

	if err != nil {
		return nil, err
	}

	return &cat, nil
}

// UploadImage uploads or replaces the feature image for a category
func (s *categoryService) UploadImage(categoryID uint, file *multipart.FileHeader) (*category.Category, error) {
	// Get category to verify existence and retrieve current image URL
	cat, err := s.repo.GetByID(categoryID)
	if err != nil {
		return nil, err
	}

	// Delete old image from Cloudflare if one exists
	if cat.ImageURL != "" {
		if err := s.cloudflareService.Delete(cat.ImageURL); err != nil {
			log.Printf("Warning: Failed to delete old image from Cloudflare for category %d: %v", categoryID, err)
		}
	}

	// Upload new image to Cloudflare
	metadata := map[string]string{
		"category_id": fmt.Sprintf("%d", categoryID),
		"type":        "category_image",
	}
	imageURL, err := s.cloudflareService.Upload(file, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	// Persist the new image URL
	cat.ImageURL = imageURL
	if err := s.repo.Update(cat); err != nil {
		// Rollback: remove the just-uploaded image
		if deleteErr := s.cloudflareService.Delete(imageURL); deleteErr != nil {
			log.Printf("Warning: Failed to rollback image upload for category %d: %v", categoryID, deleteErr)
		}
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	// Invalidate relevant caches
	cache.DeletePattern("categories:list:*")
	cache.Delete(fmt.Sprintf("category:slug:%s", cat.Slug))

	return cat, nil
}
