package handler

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/redirect"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type RedirectHandler struct {
	service service.RedirectService
}

func NewRedirectHandler(service service.RedirectService) *RedirectHandler {
	return &RedirectHandler{service: service}
}

// GetAll godoc
// @Summary Get all redirects
// @Description Get a paginated list of redirects with optional filtering
// @Tags Redirects
// @Accept json
// @Produce json
// @Param search query string false "Search in source URL, destination URL, notes"
// @Param enabled query string false "Filter by enabled status: true, false"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} redirect.RedirectListResponse "List of redirects with pagination"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/redirects [get]
func (h *RedirectHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	query := redirect.GetRedirectsQuery{
		Search:  c.Query("search", ""),
		Enabled: c.Query("enabled", ""),
		Page:    page,
		Limit:   limit,
	}

	redirects, total, err := h.service.GetAll(query)
	if err != nil {
		return response.InternalError(c, "Failed to fetch redirects")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(redirect.RedirectListResponse{
		Redirects:  redirects,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// GetByID godoc
// @Summary Get redirect by ID
// @Description Get a single redirect rule by ID
// @Tags Redirects
// @Accept json
// @Produce json
// @Param id path int true "Redirect ID"
// @Success 200 {object} redirect.RedirectResponse "Redirect details"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 404 {object} response.Response{error=string} "Not found"
// @Router /api/v1/redirects/{id} [get]
func (h *RedirectHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid redirect ID format")
	}

	rd, err := h.service.GetByID(id)
	if err != nil {
		return response.NotFound(c, "Redirect not found")
	}

	return c.JSON(redirect.RedirectResponse{Redirect: *rd})
}

// Create godoc
// @Summary Create redirect
// @Description Create a new URL redirect rule
// @Tags Redirects
// @Accept json
// @Produce json
// @Param redirect body redirect.CreateRedirectRequest true "Redirect data"
// @Success 201 {object} redirect.RedirectResponse "Redirect created successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/redirects [post]
func (h *RedirectHandler) Create(c *fiber.Ctx) error {
	var req redirect.CreateRedirectRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	rd, err := h.service.Create(&req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(redirect.RedirectResponse{Redirect: *rd})
}

// Update godoc
// @Summary Update redirect
// @Description Update an existing redirect rule
// @Tags Redirects
// @Accept json
// @Produce json
// @Param id path int true "Redirect ID"
// @Param redirect body redirect.UpdateRedirectRequest true "Redirect update data"
// @Success 200 {object} redirect.RedirectResponse "Redirect updated successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 404 {object} response.Response{error=string} "Not found"
// @Router /api/v1/redirects/{id} [put]
func (h *RedirectHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid redirect ID format")
	}

	var req redirect.UpdateRedirectRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	rd, err := h.service.Update(id, &req)
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Redirect not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(redirect.RedirectResponse{Redirect: *rd})
}

// Delete godoc
// @Summary Delete redirect
// @Description Soft delete a redirect rule by ID
// @Tags Redirects
// @Accept json
// @Produce json
// @Param id path int true "Redirect ID"
// @Success 204 "Redirect deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 404 {object} response.Response{error=string} "Not found"
// @Router /api/v1/redirects/{id} [delete]
func (h *RedirectHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid redirect ID format")
	}

	if err := h.service.Delete(id); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Redirect not found")
		}
		return response.InternalError(c, "Failed to delete redirect")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Toggle godoc
// @Summary Toggle redirect enabled status
// @Description Toggle the enabled/disabled state of a redirect
// @Tags Redirects
// @Accept json
// @Produce json
// @Param id path int true "Redirect ID"
// @Success 200 {object} redirect.RedirectResponse "Redirect toggled successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 404 {object} response.Response{error=string} "Not found"
// @Router /api/v1/redirects/{id}/toggle [patch]
func (h *RedirectHandler) Toggle(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid redirect ID format")
	}

	rd, err := h.service.Toggle(id)
	if err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Redirect not found")
		}
		return response.InternalError(c, "Failed to toggle redirect")
	}

	return c.JSON(redirect.RedirectResponse{Redirect: *rd})
}

// BulkDelete godoc
// @Summary Bulk delete redirects
// @Description Soft delete multiple redirect rules by IDs
// @Tags Redirects
// @Accept json
// @Produce json
// @Param request body redirect.BulkDeleteRequest true "Array of redirect IDs to delete"
// @Success 200 {object} response.Response "Redirects deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/redirects/bulk [delete]
func (h *RedirectHandler) BulkDelete(c *fiber.Ctx) error {
	var req redirect.BulkDeleteRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	if len(req.IDs) == 0 {
		return response.BadRequest(c, "No IDs provided")
	}

	if err := h.service.BulkDelete(req.IDs); err != nil {
		return response.InternalError(c, "Failed to delete redirects: "+err.Error())
	}

	return response.Success(c, strconv.Itoa(len(req.IDs))+" redirects deleted successfully")
}

// GetActive godoc
// @Summary Get all active redirects
// @Description Get all enabled redirect rules (public endpoint, used by middleware)
// @Tags Redirects
// @Produce json
// @Success 200 {object} response.Response "List of active redirects"
// @Router /api/v1/redirects/active [get]
func (h *RedirectHandler) GetActive(c *fiber.Ctx) error {
	redirects, err := h.service.GetAllActive()
	if err != nil {
		return response.InternalError(c, "Failed to fetch active redirects")
	}

	c.Set("Cache-Control", "public, s-maxage=60, stale-while-revalidate=300")
	return response.Success(c, redirects)
}

// IncrementHit godoc
// @Summary Increment redirect hit count
// @Description Atomically increment the hit count for a redirect (called by public middleware)
// @Tags Redirects
// @Produce json
// @Param id path int true "Redirect ID"
// @Success 200 {object} response.Response "Hit count incremented"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Router /api/v1/redirects/hit/{id} [post]
func (h *RedirectHandler) IncrementHit(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid redirect ID format")
	}

	if err := h.service.IncrementHitCount(id); err != nil {
		return response.InternalError(c, "Failed to increment hit count")
	}

	return response.Success(c, "ok")
}
