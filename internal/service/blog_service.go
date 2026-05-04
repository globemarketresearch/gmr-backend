package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/blog"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/gosimple/slug"
)

type BlogService interface {
	Create(req *blog.CreateBlogRequest) (*blog.Blog, error)
	GetAll(query blog.GetBlogsQuery) ([]blog.Blog, int64, error)
	GetByCategorySlug(categorySlug string, page, limit int) ([]blog.Blog, int64, error)
	GetByID(id uint) (*blog.Blog, error)
	GetBySlug(slug string) (*blog.Blog, error)
	Update(id uint, req *blog.UpdateBlogRequest) (*blog.Blog, error)
	Delete(id uint) error
	SoftDelete(id uint) error
	Restore(id uint) error
	SubmitForReview(id uint) (*blog.Blog, error)
	Publish(id uint) (*blog.Blog, error)
	Unpublish(id uint) (*blog.Blog, error)
	SchedulePublish(id uint, publishDate time.Time) (*blog.Blog, error)
	CancelScheduledPublish(id uint) (*blog.Blog, error)
}

type blogService struct {
	repo repository.BlogRepository
}

func NewBlogService(repo repository.BlogRepository) BlogService {
	return &blogService{repo: repo}
}

func (s *blogService) Create(req *blog.CreateBlogRequest) (*blog.Blog, error) {
	// Validate status
	if req.Status != blog.StatusDraft && req.Status != blog.StatusReview && req.Status != blog.StatusPublished {
		return nil, fmt.Errorf("invalid status: must be 'draft', 'review', or 'published'")
	}

	// Generate slug from title
	blogSlug := slug.Make(req.Title)

	// Parse publish date
	publishDate, err := time.Parse(time.RFC3339, req.PublishDate)
	if err != nil {
		return nil, fmt.Errorf("invalid publishDate format: must be ISO 8601 (RFC3339)")
	}

	// Create blog
	b := &blog.Blog{
		Title:       req.Title,
		Slug:        blogSlug,
		Excerpt:     req.Excerpt,
		Content:     req.Content,
		CategoryID:  req.CategoryID,
		Tags:        req.Tags,
		AuthorID:    req.AuthorID,
		Status:      req.Status,
		PublishDate: &publishDate,
		Location:    req.Location,
	}

	// Set metadata if provided
	if req.Metadata != nil {
		b.Metadata = *req.Metadata
	}

	if err := s.repo.Create(b); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("blogs:*")

	return b, nil
}

func (s *blogService) GetAll(query blog.GetBlogsQuery) ([]blog.Blog, int64, error) {
	// Only cache non-filtered, non-search queries
	shouldCache := query.Status == "" && query.CategoryID == "" && query.Tags == "" &&
		query.AuthorID == "" && query.Location == "" && query.Search == ""

	if shouldCache {
		cacheKey := fmt.Sprintf("blogs:list:%d:%d", query.Page, query.Limit)

		type result struct {
			Blogs []blog.Blog `json:"blogs"`
			Total int64       `json:"total"`
		}

		var res result

		err := cache.GetOrSet(cacheKey, &res, 5*time.Minute, func() (interface{}, error) {
			blogs, total, err := s.repo.GetAll(query)
			if err != nil {
				return nil, err
			}
			return result{Blogs: blogs, Total: total}, nil
		})

		if err != nil {
			return nil, 0, err
		}

		return res.Blogs, res.Total, nil
	}

	// Don't cache filtered/search results
	return s.repo.GetAll(query)
}

func (s *blogService) GetByCategorySlug(categorySlug string, page, limit int) ([]blog.Blog, int64, error) {
	return s.repo.GetByCategorySlug(categorySlug, page, limit)
}

func (s *blogService) GetByID(id uint) (*blog.Blog, error) {
	cacheKey := fmt.Sprintf("blog:id:%d", id)

	var b blog.Blog

	err := cache.GetOrSet(cacheKey, &b, 10*time.Minute, func() (interface{}, error) {
		return s.repo.GetByID(id)
	})

	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (s *blogService) GetBySlug(slug string) (*blog.Blog, error) {
	cacheKey := fmt.Sprintf("blog:slug:%s", slug)

	var b blog.Blog

	err := cache.GetOrSet(cacheKey, &b, 10*time.Minute, func() (interface{}, error) {
		return s.repo.GetBySlug(slug)
	})

	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (s *blogService) Update(id uint, req *blog.UpdateBlogRequest) (*blog.Blog, error) {
	// Check if blog exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Build updates map
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
		// Validate status
		if *req.Status != blog.StatusDraft && *req.Status != blog.StatusReview && *req.Status != blog.StatusPublished {
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

	// Update blog
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("blogs:*")
	cache.Delete(fmt.Sprintf("blog:id:%d", id))

	// Fetch and return updated blog
	return s.repo.GetByID(id)
}

func (s *blogService) Delete(id uint) error {
	// Check if blog exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(id); err != nil {
		return err
	}

	// Invalidate caches
	cache.DeletePattern("blogs:*")
	cache.Delete(fmt.Sprintf("blog:id:%d", id))

	return nil
}

func (s *blogService) SubmitForReview(id uint) (*blog.Blog, error) {
	// Check if blog exists
	existingBlog, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check if already in review or published
	if existingBlog.Status == blog.StatusReview {
		return nil, fmt.Errorf("blog is already in review")
	}
	if existingBlog.Status == blog.StatusPublished {
		return nil, fmt.Errorf("cannot submit published blog for review")
	}

	if err := s.repo.SubmitForReview(id); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("blogs:*")
	cache.Delete(fmt.Sprintf("blog:id:%d", id))

	// Fetch and return updated blog
	return s.repo.GetByID(id)
}

func (s *blogService) Publish(id uint) (*blog.Blog, error) {
	// Check if blog exists
	existingBlog, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check if already published
	if existingBlog.Status == blog.StatusPublished {
		return nil, fmt.Errorf("blog is already published")
	}

	if err := s.repo.Publish(id); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("blogs:*")
	cache.Delete(fmt.Sprintf("blog:id:%d", id))

	// Fetch and return updated blog
	return s.repo.GetByID(id)
}

func (s *blogService) Unpublish(id uint) (*blog.Blog, error) {
	// Check if blog exists
	existingBlog, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check if currently published
	if existingBlog.Status != blog.StatusPublished {
		return nil, fmt.Errorf("blog is not published")
	}

	if err := s.repo.Unpublish(id); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("blogs:*")
	cache.Delete(fmt.Sprintf("blog:id:%d", id))

	// Fetch and return updated blog
	return s.repo.GetByID(id)
}

func (s *blogService) SoftDelete(id uint) error {
	// Check if blog exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.SoftDelete(id); err != nil {
		return err
	}

	// Invalidate caches
	cache.DeletePattern("blogs:*")
	cache.Delete(fmt.Sprintf("blog:id:%d", id))

	return nil
}

func (s *blogService) Restore(id uint) error {
	if err := s.repo.Restore(id); err != nil {
		return err
	}

	// Invalidate caches
	cache.DeletePattern("blogs:*")

	return nil
}

func (s *blogService) SchedulePublish(id uint, publishDate time.Time) (*blog.Blog, error) {
	// Validate publishDate is in future
	if publishDate.Before(time.Now()) {
		return nil, errors.New("publish date must be in the future")
	}

	// Get existing blog
	b, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate status (can't schedule already published)
	if b.Status == blog.StatusPublished {
		return nil, errors.New("cannot schedule already published blog")
	}

	// Schedule publish
	if err := s.repo.SchedulePublish(id, publishDate); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("blogs:*")
	cache.Delete(fmt.Sprintf("blog:id:%d", id))

	return s.repo.GetByID(id)
}

func (s *blogService) CancelScheduledPublish(id uint) (*blog.Blog, error) {
	// Get existing blog
	_, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Cancel schedule
	if err := s.repo.CancelScheduledPublish(id); err != nil {
		return nil, err
	}

	// Invalidate caches
	cache.DeletePattern("blogs:*")
	cache.Delete(fmt.Sprintf("blog:id:%d", id))

	return s.repo.GetByID(id)
}
