package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/gosimple/slug"
)

type IndustryNewsService interface {
	Create(req *industry_news.CreateIndustryNewsRequest) (*industry_news.IndustryNews, error)
	GetAll(query industry_news.GetIndustryNewsQuery) ([]industry_news.IndustryNews, int64, error)
	GetByCategorySlug(categorySlug string, page, limit int) ([]industry_news.IndustryNews, int64, error)
	GetByID(id uint) (*industry_news.IndustryNews, error)
	GetBySlug(slug string) (*industry_news.IndustryNews, error)
	Update(id uint, req *industry_news.UpdateIndustryNewsRequest) (*industry_news.IndustryNews, error)
	Delete(id uint) error
	SoftDelete(id uint) error
	Restore(id uint) error
	SubmitForReview(id uint) (*industry_news.IndustryNews, error)
	Publish(id uint) (*industry_news.IndustryNews, error)
	Unpublish(id uint) (*industry_news.IndustryNews, error)
	SchedulePublish(id uint, publishDate time.Time) (*industry_news.IndustryNews, error)
	CancelScheduledPublish(id uint) (*industry_news.IndustryNews, error)
}

type industryNewsService struct {
	repo repository.IndustryNewsRepository
}

func NewIndustryNewsService(repo repository.IndustryNewsRepository) IndustryNewsService {
	return &industryNewsService{repo: repo}
}

func (s *industryNewsService) Create(req *industry_news.CreateIndustryNewsRequest) (*industry_news.IndustryNews, error) {
	if req.Status != industry_news.StatusDraft && req.Status != industry_news.StatusReview && req.Status != industry_news.StatusPublished {
		return nil, fmt.Errorf("invalid status: must be 'draft', 'review', or 'published'")
	}

	newsSlug := slug.Make(req.Title)

	publishDate, err := time.Parse(time.RFC3339, req.PublishDate)
	if err != nil {
		return nil, fmt.Errorf("invalid publishDate format: must be ISO 8601 (RFC3339)")
	}

	n := &industry_news.IndustryNews{
		Title:       req.Title,
		Slug:        newsSlug,
		Excerpt:     req.Excerpt,
		Content:     req.Content,
		CategoryID:  req.CategoryID,
		Tags:        req.Tags,
		AuthorID:    req.AuthorID,
		Status:      req.Status,
		PublishDate: &publishDate,
		Location:    req.Location,
	}

	if req.Metadata != nil {
		n.Metadata = *req.Metadata
	}

	if err := s.repo.Create(n); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")

	return n, nil
}

func (s *industryNewsService) GetAll(query industry_news.GetIndustryNewsQuery) ([]industry_news.IndustryNews, int64, error) {
	shouldCache := query.Status == "" && query.CategoryID == "" && query.Tags == "" &&
		query.AuthorID == "" && query.Location == "" && query.Search == ""

	if shouldCache {
		cacheKey := fmt.Sprintf("industry_news:list:%d:%d", query.Page, query.Limit)

		type result struct {
			IndustryNews []industry_news.IndustryNews `json:"industryNews"`
			Total        int64                        `json:"total"`
		}

		var res result

		err := cache.GetOrSet(cacheKey, &res, 5*time.Minute, func() (interface{}, error) {
			items, total, err := s.repo.GetAll(query)
			if err != nil {
				return nil, err
			}
			return result{IndustryNews: items, Total: total}, nil
		})

		if err != nil {
			return nil, 0, err
		}

		return res.IndustryNews, res.Total, nil
	}

	return s.repo.GetAll(query)
}

func (s *industryNewsService) GetByCategorySlug(categorySlug string, page, limit int) ([]industry_news.IndustryNews, int64, error) {
	return s.repo.GetByCategorySlug(categorySlug, page, limit)
}

func (s *industryNewsService) GetByID(id uint) (*industry_news.IndustryNews, error) {
	cacheKey := fmt.Sprintf("industry_news:id:%d", id)

	var n industry_news.IndustryNews

	err := cache.GetOrSet(cacheKey, &n, 10*time.Minute, func() (interface{}, error) {
		return s.repo.GetByID(id)
	})

	if err != nil {
		return nil, err
	}

	return &n, nil
}

func (s *industryNewsService) GetBySlug(slugStr string) (*industry_news.IndustryNews, error) {
	cacheKey := fmt.Sprintf("industry_news:slug:%s", slugStr)

	var n industry_news.IndustryNews

	err := cache.GetOrSet(cacheKey, &n, 10*time.Minute, func() (interface{}, error) {
		return s.repo.GetBySlug(slugStr)
	})

	if err != nil {
		return nil, err
	}

	return &n, nil
}

func (s *industryNewsService) Update(id uint, req *industry_news.UpdateIndustryNewsRequest) (*industry_news.IndustryNews, error) {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = *req.Title
	}

	if req.Slug != nil {
		updates["slug"] = slug.Make(*req.Slug)
	} else if req.Title != nil {
		updates["slug"] = slug.Make(*req.Title)
	}

	if req.Excerpt != nil {
		updates["excerpt"] = *req.Excerpt
	}

	if req.Content != nil {
		updates["content"] = *req.Content
	}

	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}

	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}

	if req.AuthorID != nil {
		updates["author_id"] = *req.AuthorID
	}

	if req.Status != nil {
		if *req.Status != industry_news.StatusDraft && *req.Status != industry_news.StatusReview && *req.Status != industry_news.StatusPublished {
			return nil, fmt.Errorf("invalid status: must be 'draft', 'review', or 'published'")
		}
		updates["status"] = *req.Status
	}

	if req.PublishDate != nil {
		publishDate, err := time.Parse(time.RFC3339, *req.PublishDate)
		if err != nil {
			return nil, fmt.Errorf("invalid publishDate format: must be ISO 8601 (RFC3339)")
		}
		updates["publish_date"] = publishDate
	}

	if req.Location != nil {
		updates["location"] = *req.Location
	}

	if req.Metadata != nil {
		updates["metadata"] = *req.Metadata
	}

	if req.InternalLinks != nil {
		updates["internal_links"] = *req.InternalLinks
	}

	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}

func (s *industryNewsService) Delete(id uint) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(id); err != nil {
		return err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return nil
}

func (s *industryNewsService) SubmitForReview(id uint) (*industry_news.IndustryNews, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if existing.Status == industry_news.StatusReview {
		return nil, fmt.Errorf("industry news article is already in review")
	}
	if existing.Status == industry_news.StatusPublished {
		return nil, fmt.Errorf("cannot submit published industry news article for review")
	}

	if err := s.repo.SubmitForReview(id); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}

func (s *industryNewsService) Publish(id uint) (*industry_news.IndustryNews, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if existing.Status == industry_news.StatusPublished {
		return nil, fmt.Errorf("industry news article is already published")
	}

	if err := s.repo.Publish(id); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}

func (s *industryNewsService) Unpublish(id uint) (*industry_news.IndustryNews, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if existing.Status != industry_news.StatusPublished {
		return nil, fmt.Errorf("industry news article is not published")
	}

	if err := s.repo.Unpublish(id); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}

func (s *industryNewsService) SoftDelete(id uint) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.SoftDelete(id); err != nil {
		return err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return nil
}

func (s *industryNewsService) Restore(id uint) error {
	if err := s.repo.Restore(id); err != nil {
		return err
	}

	cache.DeletePattern("industry_news:*")

	return nil
}

func (s *industryNewsService) SchedulePublish(id uint, publishDate time.Time) (*industry_news.IndustryNews, error) {
	if publishDate.Before(time.Now()) {
		return nil, errors.New("publish date must be in the future")
	}

	n, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if n.Status == industry_news.StatusPublished {
		return nil, errors.New("cannot schedule already published industry news article")
	}

	if err := s.repo.SchedulePublish(id, publishDate); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}

func (s *industryNewsService) CancelScheduledPublish(id uint) (*industry_news.IndustryNews, error) {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CancelScheduledPublish(id); err != nil {
		return nil, err
	}

	cache.DeletePattern("industry_news:*")
	cache.Delete(fmt.Sprintf("industry_news:id:%d", id))

	return s.repo.GetByID(id)
}
