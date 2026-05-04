package handler

import (
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"github.com/healthcare-market-research/backend/internal/domain/blog"
	"github.com/healthcare-market-research/backend/internal/middleware"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type BlogHandler struct {
	service      service.BlogService
	auditService service.AuditService
}

func NewBlogHandler(service service.BlogService, auditService service.AuditService) *BlogHandler {
	return &BlogHandler{service: service, auditService: auditService}
}

// GetByCategorySlug godoc
// @Summary Get blogs by category slug
// @Description Get a paginated list of published blogs for a specific category
// @Tags Blogs
// @Accept json
// @Produce json
// @Param slug path string true "Category slug"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} blog.BlogListResponse "List of blogs with pagination"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/categories/{slug}/blogs [get]
func (h *BlogHandler) GetByCategorySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	blogs, total, err := h.service.GetByCategorySlug(slug, page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to fetch blogs")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(blog.BlogListResponse{
		Blogs:      blogs,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// Create godoc
// @Summary Create blog
// @Description Create a new blog post
// @Tags Blogs
// @Accept json
// @Produce json
// @Param blog body blog.CreateBlogRequest true "Blog data"
// @Success 201 {object} blog.BlogResponse "Blog created successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/blogs [post]
func (h *BlogHandler) Create(c *fiber.Ctx) error {
	var req blog.CreateBlogRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	b, err := h.service.Create(&req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionBlogCreate)
	auditEntry.EntityType = audit.EntityBlog
	auditEntry.EntityID = &b.ID
	h.auditService.LogAsync(auditEntry)

	return c.Status(fiber.StatusCreated).JSON(blog.BlogResponse{Blog: *b})
}

// GetAll godoc
// @Summary Get all blogs
// @Description Get a paginated list of blogs with optional filtering
// @Tags Blogs
// @Accept json
// @Produce json
// @Param status query string false "Filter by status: draft, review, published"
// @Param categoryId query int false "Filter by category ID"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param authorId query int false "Filter by author ID"
// @Param location query string false "Filter by location"
// @Param search query string false "Search in title, excerpt, content"
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} blog.BlogListResponse "List of blogs with pagination"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/blogs [get]
func (h *BlogHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Validate sort_by parameter
	var sortBy string
	if s := c.Query("sort_by", ""); s != "" {
		allowed := map[string]string{
			"publish_date_desc": "publish_date DESC NULLS LAST",
			"created_at_desc":   "created_at DESC",
			"updated_at_desc":   "updated_at DESC",
		}
		if mapped, ok := allowed[s]; ok {
			sortBy = mapped
		}
	}

	query := blog.GetBlogsQuery{
		Status:       c.Query("status", ""),
		CategoryID:   c.Query("categoryId", ""),
		CategorySlug: c.Query("category", ""),
		Tags:         c.Query("tags", ""),
		AuthorID:     c.Query("authorId", ""),
		Location:     c.Query("location", ""),
		Search:       c.Query("search", ""),
		Deleted:      c.Query("deleted", ""),
		SortBy:       sortBy,
		Page:         page,
		Limit:        limit,
	}

	blogs, total, err := h.service.GetAll(query)
	if err != nil {
		return response.InternalError(c, "Failed to fetch blogs")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(blog.BlogListResponse{
		Blogs:      blogs,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// GetByID godoc
// @Summary Get blog by ID
// @Description Get a single blog post by ID
// @Tags Blogs
// @Accept json
// @Produce json
// @Param id path int true "Blog ID"
// @Success 200 {object} blog.BlogResponse "Blog details"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Blog not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/blogs/{id} [get]
func (h *BlogHandler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Blog ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid blog ID format")
	}

	b, err := h.service.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Blog not found")
	}

	return c.JSON(blog.BlogResponse{Blog: *b})
}

// GetBySlug godoc
// @Summary Get blog by slug
// @Description Get a single blog post by slug
// @Tags Blogs
// @Accept json
// @Produce json
// @Param slug path string true "Blog slug"
// @Success 200 {object} blog.BlogResponse "Blog details"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid slug"
// @Failure 404 {object} response.Response{error=string} "Blog not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/blogs/slug/{slug} [get]
func (h *BlogHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.BadRequest(c, "Blog slug is required")
	}

	b, err := h.service.GetBySlug(slug)
	if err != nil {
		return response.NotFound(c, "Blog not found")
	}

	return c.JSON(blog.BlogResponse{Blog: *b})
}

// Update godoc
// @Summary Update blog
// @Description Update an existing blog post
// @Tags Blogs
// @Accept json
// @Produce json
// @Param id path int true "Blog ID"
// @Param blog body blog.UpdateBlogRequest true "Blog update data"
// @Success 200 {object} blog.BlogResponse "Blog updated successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 404 {object} response.Response{error=string} "Blog not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/blogs/{id} [put]
func (h *BlogHandler) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Blog ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid blog ID format")
	}

	var req blog.UpdateBlogRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	b, err := h.service.Update(uint(id), &req)
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Blog not found")
		}
		return response.BadRequest(c, err.Error())
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionBlogUpdate)
	auditEntry.EntityType = audit.EntityBlog
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.JSON(blog.BlogResponse{Blog: *b})
}

// Delete godoc
// @Summary Delete blog
// @Description Delete a blog post by ID
// @Tags Blogs
// @Accept json
// @Produce json
// @Param id path int true "Blog ID"
// @Success 204 "Blog deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Blog not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/blogs/{id} [delete]
func (h *BlogHandler) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Blog ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid blog ID format")
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Blog not found")
		}
		return response.InternalError(c, "Failed to delete blog")
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionBlogDelete)
	auditEntry.EntityType = audit.EntityBlog
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.SendStatus(fiber.StatusNoContent)
}

// SubmitForReview godoc
// @Summary Submit blog for review
// @Description Change blog status from draft to review
// @Tags Blogs
// @Accept json
// @Produce json
// @Param id path int true "Blog ID"
// @Success 200 {object} blog.BlogResponse "Blog submitted for review successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or blog cannot be submitted"
// @Failure 404 {object} response.Response{error=string} "Blog not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/blogs/{id}/submit-review [patch]
func (h *BlogHandler) SubmitForReview(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Blog ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid blog ID format")
	}

	b, err := h.service.SubmitForReview(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Blog not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(blog.BlogResponse{Blog: *b})
}

// Publish godoc
// @Summary Publish blog
// @Description Change blog status to published
// @Tags Blogs
// @Accept json
// @Produce json
// @Param id path int true "Blog ID"
// @Success 200 {object} blog.BlogResponse "Blog published successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or blog cannot be published"
// @Failure 404 {object} response.Response{error=string} "Blog not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/blogs/{id}/publish [patch]
func (h *BlogHandler) Publish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Blog ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid blog ID format")
	}

	b, err := h.service.Publish(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Blog not found")
		}
		return response.BadRequest(c, err.Error())
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionBlogPublish)
	auditEntry.EntityType = audit.EntityBlog
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.JSON(blog.BlogResponse{Blog: *b})
}

// Unpublish godoc
// @Summary Unpublish blog
// @Description Change blog status from published back to draft
// @Tags Blogs
// @Accept json
// @Produce json
// @Param id path int true "Blog ID"
// @Success 200 {object} blog.BlogResponse "Blog unpublished successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or blog is not published"
// @Failure 404 {object} response.Response{error=string} "Blog not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/blogs/{id}/unpublish [patch]
func (h *BlogHandler) Unpublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Blog ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid blog ID format")
	}

	b, err := h.service.Unpublish(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Blog not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(blog.BlogResponse{Blog: *b})
}

// SoftDelete godoc
// @Summary Soft delete blog
// @Description Soft delete a blog post by ID (moves to trash)
// @Tags Blogs
// @Accept json
// @Produce json
// @Param id path int true "Blog ID"
// @Success 200 {object} response.Response "Blog moved to trash successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Blog not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/blogs/{id}/soft-delete [patch]
func (h *BlogHandler) SoftDelete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Blog ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid blog ID format")
	}

	if err := h.service.SoftDelete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Blog not found")
		}
		return response.InternalError(c, "Failed to soft delete blog")
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionBlogDelete)
	auditEntry.EntityType = audit.EntityBlog
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, "Blog moved to trash successfully")
}

// Restore godoc
// @Summary Restore blog
// @Description Restore a soft deleted blog post by ID
// @Tags Blogs
// @Accept json
// @Produce json
// @Param id path int true "Blog ID"
// @Success 200 {object} response.Response "Blog restored successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Blog not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/blogs/{id}/restore [patch]
func (h *BlogHandler) Restore(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Blog ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid blog ID format")
	}

	if err := h.service.Restore(uint(id)); err != nil {
		return response.InternalError(c, "Failed to restore blog")
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionBlogUpdate)
	auditEntry.EntityType = audit.EntityBlog
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, "Blog restored successfully")
}

// SchedulePublish godoc
// @Summary Schedule blog publish
// @Description Schedule a blog to be published at a future date/time
// @Tags Blogs
// @Accept json
// @Produce json
// @Param id path int true "Blog ID"
// @Param request body object true "Publish date"
// @Success 200 {object} blog.BlogResponse "Blog scheduled successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 404 {object} response.Response{error=string} "Blog not found"
// @Router /api/v1/blogs/{id}/schedule [patch]
func (h *BlogHandler) SchedulePublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid blog ID format")
	}

	var req struct {
		PublishDate string `json:"publishDate"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	publishDate, err := time.Parse(time.RFC3339, req.PublishDate)
	if err != nil {
		return response.BadRequest(c, "Invalid date format (use ISO 8601)")
	}

	b, err := h.service.SchedulePublish(uint(id), publishDate)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(blog.BlogResponse{Blog: *b})
}

// CancelScheduledPublish godoc
// @Summary Cancel scheduled publish
// @Description Cancel scheduled publishing for a blog
// @Tags Blogs
// @Accept json
// @Produce json
// @Param id path int true "Blog ID"
// @Success 200 {object} blog.BlogResponse "Schedule cancelled"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Router /api/v1/blogs/{id}/cancel-schedule [patch]
func (h *BlogHandler) CancelScheduledPublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid blog ID format")
	}

	b, err := h.service.CancelScheduledPublish(uint(id))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(blog.BlogResponse{Blog: *b})
}

// GetSitemap godoc
// @Summary Get published blog slugs for sitemap generation
// @Description Returns paginated slugs and dates for published blogs (up to 1000 per page). Intended for sitemap generation only.
// @Tags Blogs
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 1000, max: 1000)"
// @Success 200 {object} response.Response "Sitemap data"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/sitemap/blogs [get]
func (h *BlogHandler) GetSitemap(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "1000"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 1000 {
		limit = 1000
	}

	blogs, total, err := h.service.GetAll(blog.GetBlogsQuery{
		Status: "published",
		Page:   page,
		Limit:  limit,
		SortBy: "publish_date DESC NULLS LAST",
	})
	if err != nil {
		return response.InternalError(c, "Failed to fetch blogs for sitemap")
	}

	type sitemapItem struct {
		Slug        string     `json:"slug"`
		UpdatedAt   time.Time  `json:"updated_at"`
		PublishDate *time.Time `json:"publish_date,omitempty"`
	}

	items := make([]sitemapItem, len(blogs))
	for i, b := range blogs {
		items[i] = sitemapItem{
			Slug:        b.Slug,
			UpdatedAt:   b.UpdatedAt,
			PublishDate: b.PublishDate,
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return response.SuccessWithMeta(c, items, &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}
