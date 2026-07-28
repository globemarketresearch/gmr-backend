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
