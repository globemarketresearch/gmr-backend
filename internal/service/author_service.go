package service

import (
	"fmt"
	"log"
	"mime/multipart"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/author"
	"github.com/healthcare-market-research/backend/internal/repository"
)

type AuthorService interface {
	GetAll(page, limit int, search string) ([]author.Author, int64, error)
	GetByID(id uint) (*author.Author, error)
	Create(author *author.Author) error
	Update(id uint, author *author.Author) error
	Delete(id uint) error
	UploadImage(authorID uint, file *multipart.FileHeader) (*author.Author, error)
	DeleteImage(authorID uint) error
}

type authorService struct {
	repo           repository.AuthorRepository
	cloudflareService CloudflareImagesService
}

func NewAuthorService(repo repository.AuthorRepository, cloudflareService CloudflareImagesService) AuthorService {
	return &authorService{
		repo:           repo,
		cloudflareService: cloudflareService,
	}
}

func (s *authorService) GetAll(page, limit int, search string) ([]author.Author, int64, error) {
	// Cache only non-search queries
	if search == "" {
		cacheKey := fmt.Sprintf("authors:list:%d:%d", page, limit)

		type result struct {
			Authors []author.Author `json:"authors"`
			Total   int64           `json:"total"`
		}

		var res result

		// Use cache-aside pattern
		err := cache.GetOrSet(cacheKey, &res, 10*time.Minute, func() (interface{}, error) {
			authors, total, err := s.repo.GetAll(page, limit, search)
			if err != nil {
				return nil, err
			}
			return result{Authors: authors, Total: total}, nil
		})

		if err != nil {
			return nil, 0, err
		}

		return res.Authors, res.Total, nil
	}

	// Don't cache search results
	return s.repo.GetAll(page, limit, search)
}

func (s *authorService) GetByID(id uint) (*author.Author, error) {
	cacheKey := fmt.Sprintf("author:id:%d", id)

	var auth author.Author

	// Use cache-aside pattern
	err := cache.GetOrSet(cacheKey, &auth, 30*time.Minute, func() (interface{}, error) {
		return s.repo.GetByID(id)
	})

	if err != nil {
		return nil, err
	}

	return &auth, nil
}

func (s *authorService) Create(auth *author.Author) error {
	err := s.repo.Create(auth)
	if err != nil {
		return err
	}

	// Invalidate list caches
	cache.DeletePattern("authors:list:*")

	return nil
}

func (s *authorService) Update(id uint, auth *author.Author) error {
	// Check if author exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	auth.ID = id
	err = s.repo.Update(auth)
	if err != nil {
		return err
	}

	// Invalidate caches
	cache.DeletePattern("authors:list:*")
	cache.Delete(fmt.Sprintf("author:id:%d", id))

	return nil
}

func (s *authorService) Delete(id uint) error {
	// Check if author is referenced in any reports
	isReferenced, err := s.repo.IsReferencedInReports(id)
	if err != nil {
		return err
	}

	if isReferenced {
		return fmt.Errorf("cannot delete author: author is referenced in one or more reports")
	}

	// Get author to check if they have an image
	auth, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	// Delete image from Cloudflare if exists (before deleting author)
	if auth.ImageURL != "" {
		if err := s.cloudflareService.Delete(auth.ImageURL); err != nil {
			// Log error but don't fail deletion
			log.Printf("Warning: Failed to delete image from Cloudflare for author %d: %v", id, err)
		}
	}

	err = s.repo.Delete(id)
	if err != nil {
		return err
	}

	// Invalidate caches
	cache.DeletePattern("authors:list:*")
	cache.Delete(fmt.Sprintf("author:id:%d", id))

	return nil
}

// UploadImage uploads an image for an author
func (s *authorService) UploadImage(authorID uint, file *multipart.FileHeader) (*author.Author, error) {
	// Get author to verify existence and get current image URL
	auth, err := s.repo.GetByID(authorID)
	if err != nil {
		return nil, err
	}

	// Delete old image from Cloudflare if exists
	if auth.ImageURL != "" {
		if err := s.cloudflareService.Delete(auth.ImageURL); err != nil {
			// Log error but continue with upload
			log.Printf("Warning: Failed to delete old image from Cloudflare for author %d: %v", authorID, err)
		}
	}

	// Upload new image to Cloudflare
	metadata := map[string]string{
		"author_id": fmt.Sprintf("%d", authorID),
		"type":      "author_profile",
	}
	imageURL, err := s.cloudflareService.Upload(file, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	// Update author with new image URL
	auth.ImageURL = imageURL
	if err := s.repo.Update(auth); err != nil {
		// Rollback: delete the uploaded image
		if deleteErr := s.cloudflareService.Delete(imageURL); deleteErr != nil {
			log.Printf("Warning: Failed to rollback image upload for author %d: %v", authorID, deleteErr)
		}
		return nil, fmt.Errorf("failed to update author: %w", err)
	}

	// Invalidate caches
	cache.DeletePattern("authors:list:*")
	cache.Delete(fmt.Sprintf("author:id:%d", authorID))

	return auth, nil
}

// DeleteImage deletes an author's image
func (s *authorService) DeleteImage(authorID uint) error {
	// Get author to verify existence and get image URL
	auth, err := s.repo.GetByID(authorID)
	if err != nil {
		return err
	}

	if auth.ImageURL == "" {
		return fmt.Errorf("author has no image to delete")
	}

	// Delete image from Cloudflare
	if err := s.cloudflareService.Delete(auth.ImageURL); err != nil {
		return fmt.Errorf("failed to delete image from Cloudflare: %w", err)
	}

	// Clear image URL in database
	auth.ImageURL = ""
	if err := s.repo.Update(auth); err != nil {
		return fmt.Errorf("failed to update author: %w", err)
	}

	// Invalidate caches
	cache.DeletePattern("authors:list:*")
	cache.Delete(fmt.Sprintf("author:id:%d", authorID))

	return nil
}
