package service

import (
	"fmt"
	"log"
	"mime/multipart"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/mediamention"
	"github.com/healthcare-market-research/backend/internal/repository"
)

type MediaMentionService interface {
	GetAll(page, limit int, search string) ([]mediamention.MediaMention, int64, error)
	GetByID(id uint) (*mediamention.MediaMention, error)
	Create(mention *mediamention.MediaMention) error
	Update(id uint, mention *mediamention.MediaMention) error
	Delete(id uint) error
	UploadImage(id uint, file *multipart.FileHeader) (*mediamention.MediaMention, error)
	DeleteImage(id uint) error
}

type mediaMentionService struct {
	repo              repository.MediaMentionRepository
	cloudflareService CloudflareImagesService
}

func NewMediaMentionService(repo repository.MediaMentionRepository, cloudflareService CloudflareImagesService) MediaMentionService {
	return &mediaMentionService{
		repo:              repo,
		cloudflareService: cloudflareService,
	}
}

func (s *mediaMentionService) GetAll(page, limit int, search string) ([]mediamention.MediaMention, int64, error) {
	if search == "" {
		cacheKey := fmt.Sprintf("media_mentions:list:%d:%d", page, limit)

		type result struct {
			Mentions []mediamention.MediaMention `json:"mentions"`
			Total    int64                       `json:"total"`
		}

		var res result

		err := cache.GetOrSet(cacheKey, &res, 10*time.Minute, func() (interface{}, error) {
			mentions, total, err := s.repo.GetAll(page, limit, search)
			if err != nil {
				return nil, err
			}
			return result{Mentions: mentions, Total: total}, nil
		})

		if err != nil {
			return nil, 0, err
		}

		return res.Mentions, res.Total, nil
	}

	return s.repo.GetAll(page, limit, search)
}

func (s *mediaMentionService) GetByID(id uint) (*mediamention.MediaMention, error) {
	cacheKey := fmt.Sprintf("media_mention:id:%d", id)

	var mention mediamention.MediaMention

	err := cache.GetOrSet(cacheKey, &mention, 30*time.Minute, func() (interface{}, error) {
		return s.repo.GetByID(id)
	})

	if err != nil {
		return nil, err
	}

	return &mention, nil
}

func (s *mediaMentionService) Create(mention *mediamention.MediaMention) error {
	err := s.repo.Create(mention)
	if err != nil {
		return err
	}

	cache.DeletePattern("media_mentions:list:*")

	return nil
}

func (s *mediaMentionService) Update(id uint, mention *mediamention.MediaMention) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	mention.ID = id
	err = s.repo.Update(mention)
	if err != nil {
		return err
	}

	cache.DeletePattern("media_mentions:list:*")
	cache.Delete(fmt.Sprintf("media_mention:id:%d", id))

	return nil
}

func (s *mediaMentionService) Delete(id uint) error {
	mention, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if mention.ImageURL != "" {
		if err := s.cloudflareService.Delete(mention.ImageURL); err != nil {
			log.Printf("Warning: Failed to delete image from Cloudflare for media mention %d: %v", id, err)
		}
	}

	err = s.repo.Delete(id)
	if err != nil {
		return err
	}

	cache.DeletePattern("media_mentions:list:*")
	cache.Delete(fmt.Sprintf("media_mention:id:%d", id))

	return nil
}

func (s *mediaMentionService) UploadImage(id uint, file *multipart.FileHeader) (*mediamention.MediaMention, error) {
	mention, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if mention.ImageURL != "" {
		if err := s.cloudflareService.Delete(mention.ImageURL); err != nil {
			log.Printf("Warning: Failed to delete old image from Cloudflare for media mention %d: %v", id, err)
		}
	}

	metadata := map[string]string{
		"media_mention_id": fmt.Sprintf("%d", id),
		"type":             "media_mention_logo",
	}
	imageURL, err := s.cloudflareService.Upload(file, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	mention.ImageURL = imageURL
	if err := s.repo.Update(mention); err != nil {
		if deleteErr := s.cloudflareService.Delete(imageURL); deleteErr != nil {
			log.Printf("Warning: Failed to rollback image upload for media mention %d: %v", id, deleteErr)
		}
		return nil, fmt.Errorf("failed to update media mention: %w", err)
	}

	cache.DeletePattern("media_mentions:list:*")
	cache.Delete(fmt.Sprintf("media_mention:id:%d", id))

	return mention, nil
}

func (s *mediaMentionService) DeleteImage(id uint) error {
	mention, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if mention.ImageURL == "" {
		return fmt.Errorf("media mention has no image to delete")
	}

	if err := s.cloudflareService.Delete(mention.ImageURL); err != nil {
		return fmt.Errorf("failed to delete image from Cloudflare: %w", err)
	}

	mention.ImageURL = ""
	if err := s.repo.Update(mention); err != nil {
		return fmt.Errorf("failed to update media mention: %w", err)
	}

	cache.DeletePattern("media_mentions:list:*")
	cache.Delete(fmt.Sprintf("media_mention:id:%d", id))

	return nil
}
