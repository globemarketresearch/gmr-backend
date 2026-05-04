package service

import (
	"errors"
	"fmt"

	"github.com/healthcare-market-research/backend/internal/domain/redirect"
	"github.com/healthcare-market-research/backend/internal/repository"
)

var validRedirectTypes = map[int]bool{
	301: true,
	302: true,
	307: true,
	308: true,
	410: true,
	451: true,
}

type RedirectService interface {
	GetAll(query redirect.GetRedirectsQuery) ([]redirect.Redirect, int64, error)
	GetByID(id int64) (*redirect.Redirect, error)
	Create(req *redirect.CreateRedirectRequest) (*redirect.Redirect, error)
	Update(id int64, req *redirect.UpdateRedirectRequest) (*redirect.Redirect, error)
	Delete(id int64) error
	Toggle(id int64) (*redirect.Redirect, error)
	IncrementHitCount(id int64) error
	GetAllActive() ([]redirect.Redirect, error)
	BulkDelete(ids []int64) error
}

type redirectService struct {
	repo repository.RedirectRepository
}

func NewRedirectService(repo repository.RedirectRepository) RedirectService {
	return &redirectService{repo: repo}
}

func (s *redirectService) GetAll(query redirect.GetRedirectsQuery) ([]redirect.Redirect, int64, error) {
	return s.repo.GetAll(query)
}

func (s *redirectService) GetByID(id int64) (*redirect.Redirect, error) {
	return s.repo.GetByID(id)
}

func (s *redirectService) Create(req *redirect.CreateRedirectRequest) (*redirect.Redirect, error) {
	if req.SourceURL == "" {
		return nil, errors.New("sourceUrl is required")
	}

	if !validRedirectTypes[req.RedirectType] {
		return nil, fmt.Errorf("invalid redirectType: must be one of 301, 302, 307, 308, 410, 451")
	}

	// For types that require a destination, validate it's provided
	if req.RedirectType != 410 && req.RedirectType != 451 {
		if req.DestinationURL == nil || *req.DestinationURL == "" {
			return nil, errors.New("destinationUrl is required for redirect types 301, 302, 307, 308")
		}
	}

	// Check for duplicate source URL
	existing, err := s.repo.GetBySourceURL(req.SourceURL)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("a redirect for '%s' already exists", req.SourceURL)
	}

	rd := &redirect.Redirect{
		SourceURL:      req.SourceURL,
		DestinationURL: req.DestinationURL,
		RedirectType:   req.RedirectType,
		Notes:          req.Notes,
		IsEnabled:      true,
	}

	if err := s.repo.Create(rd); err != nil {
		return nil, err
	}

	return rd, nil
}

func (s *redirectService) Update(id int64, req *redirect.UpdateRedirectRequest) (*redirect.Redirect, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.SourceURL != nil {
		// Check for duplicate if source URL is changing
		if *req.SourceURL != existing.SourceURL {
			dup, err := s.repo.GetBySourceURL(*req.SourceURL)
			if err == nil && dup != nil {
				return nil, fmt.Errorf("a redirect for '%s' already exists", *req.SourceURL)
			}
		}
		updates["source_url"] = *req.SourceURL
	}

	if req.DestinationURL != nil {
		updates["destination_url"] = req.DestinationURL
	}

	if req.RedirectType != nil {
		if !validRedirectTypes[*req.RedirectType] {
			return nil, fmt.Errorf("invalid redirectType: must be one of 301, 302, 307, 308, 410, 451")
		}
		updates["redirect_type"] = *req.RedirectType
	}

	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}

	if req.Notes != nil {
		updates["notes"] = *req.Notes
	}

	if len(updates) == 0 {
		return existing, nil
	}

	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}

	return s.repo.GetByID(id)
}

func (s *redirectService) Delete(id int64) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *redirectService) Toggle(id int64) (*redirect.Redirect, error) {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.ToggleEnabled(id); err != nil {
		return nil, err
	}

	return s.repo.GetByID(id)
}

func (s *redirectService) IncrementHitCount(id int64) error {
	return s.repo.IncrementHitCount(id)
}

func (s *redirectService) GetAllActive() ([]redirect.Redirect, error) {
	return s.repo.GetAllActive()
}

func (s *redirectService) BulkDelete(ids []int64) error {
	if len(ids) == 0 {
		return errors.New("no IDs provided")
	}
	return s.repo.BulkDelete(ids)
}
