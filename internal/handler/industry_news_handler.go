package handler

import (
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"github.com/healthcare-market-research/backend/internal/domain/industry_news"
	"github.com/healthcare-market-research/backend/internal/middleware"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type IndustryNewsHandler struct {
	service      service.IndustryNewsService
	auditService service.AuditService
}

func NewIndustryNewsHandler(service service.IndustryNewsService, auditService service.AuditService) *IndustryNewsHandler {
	return &IndustryNewsHandler{service: service, auditService: auditService}
}

// GetByCategorySlug godoc
// @Summary Get industry news by category slug
// @Description Get a paginated list of published industry news articles for a specific category
// @Tags IndustryNews
// @Accept json
// @Produce json
// @Param slug path string true "Category slug"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} industry_news.IndustryNewsListResponse "List of industry news articles with pagination"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/categories/{slug}/industry-news [get]
func (h *IndustryNewsHandler) GetByCategorySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	items, total, err := h.service.GetByCategorySlug(slug, page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to fetch industry news")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(industry_news.IndustryNewsListResponse{
		IndustryNews: items,
		Total:        total,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	})
}

// Create godoc
// @Summary Create industry news article
// @Description Create a new industry news article. Admin/editor only.
// @Tags IndustryNews
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param industryNews body industry_news.CreateIndustryNewsRequest true "Industry news data"
// @Success 201 {object} industry_news.IndustryNewsResponse "Industry news article created successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/industry-news [post]
func (h *IndustryNewsHandler) Create(c *fiber.Ctx) error {
	var req industry_news.CreateIndustryNewsRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	n, err := h.service.Create(&req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsCreate)
	auditEntry.EntityType = audit.EntityIndustryNews
	auditEntry.EntityID = &n.ID
	h.auditService.LogAsync(auditEntry)

	return c.Status(fiber.StatusCreated).JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

// GetAll godoc
// @Summary Get all industry news articles
// @Description Get a paginated list of industry news articles with optional filtering
// @Tags IndustryNews
// @Accept json
// @Produce json
// @Param status query string false "Filter by status: draft, review, published"
// @Param categoryId query int false "Filter by category ID"
// @Param category query string false "Filter by category slug"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param authorId query int false "Filter by author ID"
// @Param location query string false "Filter by location"
// @Param search query string false "Search in title, excerpt, content"
// @Param deleted query string false "Set to 'true' to list soft-deleted (trashed) articles"
// @Param sort_by query string false "Sort order: publish_date_desc, created_at_desc, updated_at_desc"
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} industry_news.IndustryNewsListResponse "List of industry news articles with pagination"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/industry-news [get]
func (h *IndustryNewsHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

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

	query := industry_news.GetIndustryNewsQuery{
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

	items, total, err := h.service.GetAll(query)
	if err != nil {
		return response.InternalError(c, "Failed to fetch industry news")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(industry_news.IndustryNewsListResponse{
		IndustryNews: items,
		Total:        total,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	})
}

// GetByID godoc
// @Summary Get industry news article by ID
// @Description Get a single industry news article by ID
// @Tags IndustryNews
// @Accept json
// @Produce json
// @Param id path int true "Industry news article ID"
// @Success 200 {object} industry_news.IndustryNewsResponse "Industry news article details"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Industry news article not found"
// @Router /api/v1/industry-news/{id} [get]
func (h *IndustryNewsHandler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Industry news ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	n, err := h.service.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Industry news article not found")
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

// GetBySlug godoc
// @Summary Get industry news article by slug
// @Description Get a single industry news article by slug
// @Tags IndustryNews
// @Accept json
// @Produce json
// @Param slug path string true "Industry news article slug"
// @Success 200 {object} industry_news.IndustryNewsResponse "Industry news article details"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid slug"
// @Failure 404 {object} response.Response{error=string} "Industry news article not found"
// @Router /api/v1/industry-news/slug/{slug} [get]
func (h *IndustryNewsHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.BadRequest(c, "Industry news slug is required")
	}

	n, err := h.service.GetBySlug(slug)
	if err != nil {
		return response.NotFound(c, "Industry news article not found")
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

// Update godoc
// @Summary Update industry news article
// @Description Update an existing industry news article. Admin/editor only.
// @Tags IndustryNews
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Industry news article ID"
// @Param industryNews body industry_news.UpdateIndustryNewsRequest true "Industry news update data"
// @Success 200 {object} industry_news.IndustryNewsResponse "Industry news article updated successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Industry news article not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/industry-news/{id} [put]
func (h *IndustryNewsHandler) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Industry news ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	var req industry_news.UpdateIndustryNewsRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	n, err := h.service.Update(uint(id), &req)
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.BadRequest(c, err.Error())
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsUpdate)
	auditEntry.EntityType = audit.EntityIndustryNews
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

// Delete godoc
// @Summary Delete industry news article
// @Description Hard delete an industry news article by ID. Admin only.
// @Tags IndustryNews
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Industry news article ID"
// @Success 204 "Industry news article deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin role"
// @Failure 404 {object} response.Response{error=string} "Industry news article not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/industry-news/{id} [delete]
func (h *IndustryNewsHandler) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Industry news ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.InternalError(c, "Failed to delete industry news article")
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsDelete)
	auditEntry.EntityType = audit.EntityIndustryNews
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.SendStatus(fiber.StatusNoContent)
}

// SubmitForReview godoc
// @Summary Submit industry news article for review
// @Description Change industry news article status from draft to review. Admin/editor only.
// @Tags IndustryNews
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Industry news article ID"
// @Success 200 {object} industry_news.IndustryNewsResponse "Industry news article submitted for review successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or article cannot be submitted"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Industry news article not found"
// @Router /api/v1/industry-news/{id}/submit-review [patch]
func (h *IndustryNewsHandler) SubmitForReview(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	n, err := h.service.SubmitForReview(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

// Publish godoc
// @Summary Publish industry news article
// @Description Change industry news article status to published. Admin/editor only.
// @Tags IndustryNews
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Industry news article ID"
// @Success 200 {object} industry_news.IndustryNewsResponse "Industry news article published successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or article cannot be published"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Industry news article not found"
// @Router /api/v1/industry-news/{id}/publish [patch]
func (h *IndustryNewsHandler) Publish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	n, err := h.service.Publish(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.BadRequest(c, err.Error())
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsPublish)
	auditEntry.EntityType = audit.EntityIndustryNews
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

// Unpublish godoc
// @Summary Unpublish industry news article
// @Description Change industry news article status from published back to draft. Admin/editor only.
// @Tags IndustryNews
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Industry news article ID"
// @Success 200 {object} industry_news.IndustryNewsResponse "Industry news article unpublished successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or article is not published"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Industry news article not found"
// @Router /api/v1/industry-news/{id}/unpublish [patch]
func (h *IndustryNewsHandler) Unpublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	n, err := h.service.Unpublish(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

// SoftDelete godoc
// @Summary Soft delete industry news article
// @Description Soft delete an industry news article by ID (moves to trash). Admin/editor only.
// @Tags IndustryNews
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Industry news article ID"
// @Success 200 {object} response.Response "Industry news article moved to trash successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Industry news article not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/industry-news/{id}/soft-delete [patch]
func (h *IndustryNewsHandler) SoftDelete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	if err := h.service.SoftDelete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Industry news article not found")
		}
		return response.InternalError(c, "Failed to soft delete industry news article")
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsDelete)
	auditEntry.EntityType = audit.EntityIndustryNews
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, "Industry news article moved to trash successfully")
}

// Restore godoc
// @Summary Restore industry news article
// @Description Restore a soft deleted industry news article by ID. Admin only.
// @Tags IndustryNews
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Industry news article ID"
// @Success 200 {object} response.Response "Industry news article restored successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin role"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/industry-news/{id}/restore [patch]
func (h *IndustryNewsHandler) Restore(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	if err := h.service.Restore(uint(id)); err != nil {
		return response.InternalError(c, "Failed to restore industry news article")
	}

	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionIndustryNewsUpdate)
	auditEntry.EntityType = audit.EntityIndustryNews
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, "Industry news article restored successfully")
}

// SchedulePublish godoc
// @Summary Schedule industry news publish
// @Description Schedule an industry news article to be published at a future date/time. Admin/editor only.
// @Tags IndustryNews
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Industry news article ID"
// @Param request body object true "Publish date"
// @Success 200 {object} industry_news.IndustryNewsResponse "Industry news article scheduled successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Router /api/v1/industry-news/{id}/schedule [patch]
func (h *IndustryNewsHandler) SchedulePublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
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

	n, err := h.service.SchedulePublish(uint(id), publishDate)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

// CancelScheduledPublish godoc
// @Summary Cancel scheduled publish
// @Description Cancel scheduled publishing for an industry news article. Admin/editor only.
// @Tags IndustryNews
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Industry news article ID"
// @Success 200 {object} industry_news.IndustryNewsResponse "Schedule cancelled"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Router /api/v1/industry-news/{id}/cancel-schedule [patch]
func (h *IndustryNewsHandler) CancelScheduledPublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid industry news ID format")
	}

	n, err := h.service.CancelScheduledPublish(uint(id))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(industry_news.IndustryNewsResponse{IndustryNews: *n})
}

// GetSitemap godoc
// @Summary Get published industry news slugs for sitemap generation
// @Description Returns paginated slugs and dates for published industry news articles (up to 1000 per page). Intended for sitemap generation only.
// @Tags IndustryNews
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 1000, max: 1000)"
// @Success 200 {object} response.Response "Sitemap data"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/sitemap/industry-news [get]
func (h *IndustryNewsHandler) GetSitemap(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "1000"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 1000 {
		limit = 1000
	}

	items, total, err := h.service.GetAll(industry_news.GetIndustryNewsQuery{
		Status: "published",
		Page:   page,
		Limit:  limit,
		SortBy: "publish_date DESC NULLS LAST",
	})
	if err != nil {
		return response.InternalError(c, "Failed to fetch industry news for sitemap")
	}

	type sitemapItem struct {
		Slug        string     `json:"slug"`
		UpdatedAt   time.Time  `json:"updated_at"`
		PublishDate *time.Time `json:"publish_date,omitempty"`
	}

	items2 := make([]sitemapItem, len(items))
	for i, n := range items {
		items2[i] = sitemapItem{
			Slug:        n.Slug,
			UpdatedAt:   n.UpdatedAt,
			PublishDate: n.PublishDate,
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return response.SuccessWithMeta(c, items2, &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}
