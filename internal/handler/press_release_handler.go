package handler

import (
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"github.com/healthcare-market-research/backend/internal/domain/press_release"
	"github.com/healthcare-market-research/backend/internal/middleware"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type PressReleaseHandler struct {
	service      service.PressReleaseService
	auditService service.AuditService
}

func NewPressReleaseHandler(service service.PressReleaseService, auditService service.AuditService) *PressReleaseHandler {
	return &PressReleaseHandler{service: service, auditService: auditService}
}

// GetByCategorySlug godoc
// @Summary Get press releases by category slug
// @Description Get a paginated list of published press releases for a specific category
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param slug path string true "Category slug"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} press_release.PressReleaseListResponse "List of press releases with pagination"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/categories/{slug}/press-releases [get]
func (h *PressReleaseHandler) GetByCategorySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	pressReleases, total, err := h.service.GetByCategorySlug(slug, page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to fetch press releases")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(press_release.PressReleaseListResponse{
		PressReleases: pressReleases,
		Total:         total,
		Page:          page,
		Limit:         limit,
		TotalPages:    totalPages,
	})
}

// Create godoc
// @Summary Create press release
// @Description Create a new press release
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param pressRelease body press_release.CreatePressReleaseRequest true "Press release data"
// @Success 201 {object} press_release.PressReleaseResponse "Press release created successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases [post]
func (h *PressReleaseHandler) Create(c *fiber.Ctx) error {
	var req press_release.CreatePressReleaseRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	pr, err := h.service.Create(&req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionPressReleaseCreate)
	auditEntry.EntityType = audit.EntityPressRelease
	auditEntry.EntityID = &pr.ID
	h.auditService.LogAsync(auditEntry)

	return c.Status(fiber.StatusCreated).JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// GetAll godoc
// @Summary Get all press releases
// @Description Get a paginated list of press releases with optional filtering
// @Tags PressReleases
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
// @Success 200 {object} press_release.PressReleaseListResponse "List of press releases with pagination"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases [get]
func (h *PressReleaseHandler) GetAll(c *fiber.Ctx) error {
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

	query := press_release.GetPressReleasesQuery{
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

	pressReleases, total, err := h.service.GetAll(query)
	if err != nil {
		return response.InternalError(c, "Failed to fetch press releases")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(press_release.PressReleaseListResponse{
		PressReleases: pressReleases,
		Total:         total,
		Page:          page,
		Limit:         limit,
		TotalPages:    totalPages,
	})
}

// GetByID godoc
// @Summary Get press release by ID
// @Description Get a single press release by ID
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 200 {object} press_release.PressReleaseResponse "Press release details"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id} [get]
func (h *PressReleaseHandler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	pr, err := h.service.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Press release not found")
	}

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// GetBySlug godoc
// @Summary Get press release by slug
// @Description Get a single press release by slug
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param slug path string true "Press release slug"
// @Success 200 {object} press_release.PressReleaseResponse "Press release details"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid slug"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/slug/{slug} [get]
func (h *PressReleaseHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.BadRequest(c, "Press release slug is required")
	}

	pr, err := h.service.GetBySlug(slug)
	if err != nil {
		return response.NotFound(c, "Press release not found")
	}

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// Update godoc
// @Summary Update press release
// @Description Update an existing press release
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Param pressRelease body press_release.UpdatePressReleaseRequest true "Press release update data"
// @Success 200 {object} press_release.PressReleaseResponse "Press release updated successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id} [put]
func (h *PressReleaseHandler) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	var req press_release.UpdatePressReleaseRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	pr, err := h.service.Update(uint(id), &req)
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Press release not found")
		}
		return response.BadRequest(c, err.Error())
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionPressReleaseUpdate)
	auditEntry.EntityType = audit.EntityPressRelease
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// Delete godoc
// @Summary Delete press release
// @Description Delete a press release by ID
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 204 "Press release deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id} [delete]
func (h *PressReleaseHandler) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Press release not found")
		}
		return response.InternalError(c, "Failed to delete press release")
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionPressReleaseDelete)
	auditEntry.EntityType = audit.EntityPressRelease
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.SendStatus(fiber.StatusNoContent)
}

// SubmitForReview godoc
// @Summary Submit press release for review
// @Description Change press release status from draft to review
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 200 {object} press_release.PressReleaseResponse "Press release submitted for review successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or press release cannot be submitted"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id}/submit-review [patch]
func (h *PressReleaseHandler) SubmitForReview(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	pr, err := h.service.SubmitForReview(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Press release not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// Publish godoc
// @Summary Publish press release
// @Description Change press release status to published
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 200 {object} press_release.PressReleaseResponse "Press release published successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or press release cannot be published"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id}/publish [patch]
func (h *PressReleaseHandler) Publish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	pr, err := h.service.Publish(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Press release not found")
		}
		return response.BadRequest(c, err.Error())
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionPressReleasePublish)
	auditEntry.EntityType = audit.EntityPressRelease
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// Unpublish godoc
// @Summary Unpublish press release
// @Description Change press release status from published back to draft
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 200 {object} press_release.PressReleaseResponse "Press release unpublished successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID or press release is not published"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id}/unpublish [patch]
func (h *PressReleaseHandler) Unpublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	pr, err := h.service.Unpublish(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Press release not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// SoftDelete godoc
// @Summary Soft delete press release
// @Description Soft delete a press release by ID (moves to trash)
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 200 {object} response.Response "Press release moved to trash successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id}/soft-delete [patch]
func (h *PressReleaseHandler) SoftDelete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	if err := h.service.SoftDelete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Press release not found")
		}
		return response.InternalError(c, "Failed to soft delete press release")
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionPressReleaseDelete)
	auditEntry.EntityType = audit.EntityPressRelease
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, "Press release moved to trash successfully")
}

// Restore godoc
// @Summary Restore press release
// @Description Restore a soft deleted press release by ID
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param id path int true "Press release ID"
// @Success 200 {object} response.Response "Press release restored successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/press-releases/{id}/restore [patch]
func (h *PressReleaseHandler) Restore(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Press release ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	if err := h.service.Restore(uint(id)); err != nil {
		return response.InternalError(c, "Failed to restore press release")
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionPressReleaseUpdate)
	auditEntry.EntityType = audit.EntityPressRelease
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, "Press release restored successfully")
}

// SchedulePublish godoc
// @Summary Schedule press release publish
// @Description Schedule a press release to be published at a future date/time
// @Tags Press Releases
// @Accept json
// @Produce json
// @Param id path int true "Press Release ID"
// @Param request body object true "Publish date"
// @Success 200 {object} press_release.PressReleaseResponse "Press release scheduled successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 404 {object} response.Response{error=string} "Press release not found"
// @Router /api/v1/press-releases/{id}/schedule [patch]
func (h *PressReleaseHandler) SchedulePublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
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

	pr, err := h.service.SchedulePublish(uint(id), publishDate)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// CancelScheduledPublish godoc
// @Summary Cancel scheduled publish
// @Description Cancel scheduled publishing for a press release
// @Tags Press Releases
// @Accept json
// @Produce json
// @Param id path int true "Press Release ID"
// @Success 200 {object} press_release.PressReleaseResponse "Schedule cancelled"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Router /api/v1/press-releases/{id}/cancel-schedule [patch]
func (h *PressReleaseHandler) CancelScheduledPublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid press release ID format")
	}

	pr, err := h.service.CancelScheduledPublish(uint(id))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(press_release.PressReleaseResponse{PressRelease: *pr})
}

// GetSitemap godoc
// @Summary Get published press release slugs for sitemap generation
// @Description Returns paginated slugs and dates for published press releases (up to 1000 per page). Intended for sitemap generation only.
// @Tags PressReleases
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 1000, max: 1000)"
// @Success 200 {object} response.Response "Sitemap data"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/sitemap/press-releases [get]
func (h *PressReleaseHandler) GetSitemap(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "1000"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 1000 {
		limit = 1000
	}

	pressReleases, total, err := h.service.GetAll(press_release.GetPressReleasesQuery{
		Status: "published",
		Page:   page,
		Limit:  limit,
		SortBy: "publish_date DESC NULLS LAST",
	})
	if err != nil {
		return response.InternalError(c, "Failed to fetch press releases for sitemap")
	}

	type sitemapItem struct {
		Slug        string     `json:"slug"`
		UpdatedAt   time.Time  `json:"updated_at"`
		PublishDate *time.Time `json:"publish_date,omitempty"`
	}

	items := make([]sitemapItem, len(pressReleases))
	for i, pr := range pressReleases {
		items[i] = sitemapItem{
			Slug:        pr.Slug,
			UpdatedAt:   pr.UpdatedAt,
			PublishDate: pr.PublishDate,
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
