package handler

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/form"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type FormHandler struct {
	service service.FormService
}

func NewFormHandler(service service.FormService) *FormHandler {
	return &FormHandler{service: service}
}

// Create godoc
// @Summary Create form submission
// @Description Submit a new form (contact or request-sample)
// @Tags Forms
// @Accept json
// @Produce json
// @Param submission body form.CreateSubmissionRequest true "Form submission data"
// @Success 201 {object} form.SubmissionResponse "Submission created successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input or validation error"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/forms/submissions [post]
func (h *FormHandler) Create(c *fiber.Ctx) error {
	var req form.CreateSubmissionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	// Get client IP and user agent from request
	if req.Metadata.IPAddress == "" {
		req.Metadata.IPAddress = c.IP()
	}
	if req.Metadata.UserAgent == "" {
		req.Metadata.UserAgent = c.Get("User-Agent")
	}
	if req.Metadata.Referrer == "" {
		req.Metadata.Referrer = c.Get("Referer")
	}
	if req.Metadata.PageURL == "" {
		if origin := c.Get("Origin"); origin != "" {
			req.Metadata.PageURL = origin
		}
	}

	submissionResp, err := h.service.Create(&req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(submissionResp)
}

// GetAll godoc
// @Summary Get all form submissions
// @Description Get a paginated list of form submissions with optional filtering
// @Tags Forms
// @Accept json
// @Produce json
// @Param category query string false "Filter by category: contact, request-sample"
// @Param status query string false "Filter by status: pending, processed, archived"
// @Param dateFrom query string false "Start date (ISO 8601 format)"
// @Param dateTo query string false "End date (ISO 8601 format)"
// @Param search query string false "Search in name, email, company"
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param sortBy query string false "Sort field: createdAt, company, name (default: createdAt)"
// @Param sortOrder query string false "Sort order: asc, desc (default: desc)"
// @Success 200 {object} response.Response{data=[]form.FormSubmission,meta=response.Meta} "List of submissions with pagination"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/forms/submissions [get]
func (h *FormHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	query := form.GetSubmissionsQuery{
		Category:  c.Query("category", ""),
		Status:    c.Query("status", ""),
		DateFrom:  c.Query("dateFrom", ""),
		DateTo:    c.Query("dateTo", ""),
		Search:    c.Query("search", ""),
		Page:      page,
		Limit:     limit,
		SortBy:    c.Query("sortBy", ""),
		SortOrder: c.Query("sortOrder", ""),
	}

	submissions, total, err := h.service.GetAll(query)
	if err != nil {
		return response.InternalError(c, "Failed to fetch submissions")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, submissions, meta)
}

// GetByID godoc
// @Summary Get single submission
// @Description Get a single form submission by ID
// @Tags Forms
// @Accept json
// @Produce json
// @Param id path int true "Submission ID"
// @Success 200 {object} response.Response{data=form.FormSubmission} "Submission details"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Submission not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/forms/submissions/{id} [get]
func (h *FormHandler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Submission ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid submission ID format")
	}

	submission, err := h.service.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Submission not found")
	}

	return response.Success(c, submission)
}

// GetByCategory godoc
// @Summary Get submissions by category
// @Description Get form submissions filtered by category (contact or request-sample)
// @Tags Forms
// @Accept json
// @Produce json
// @Param category path string true "Category: contact or request-sample"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} response.Response{data=[]form.FormSubmission,meta=response.Meta} "List of submissions"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid category"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/forms/submissions/category/{category} [get]
func (h *FormHandler) GetByCategory(c *fiber.Ctx) error {
	category := c.Params("category")
	if category != "contact" && category != "request-sample" && category != "request-customization" && category != "schedule-demo" {
		return response.BadRequest(c, "Invalid category: must be 'contact', 'request-sample', 'request-customization', or 'schedule-demo'")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	submissions, total, err := h.service.GetByCategory(category, page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to fetch submissions")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, submissions, meta)
}

// Delete godoc
// @Summary Delete submission
// @Description Delete a form submission by ID
// @Tags Forms
// @Accept json
// @Produce json
// @Param id path int true "Submission ID"
// @Success 200 {object} form.DeleteResponse "Submission deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 404 {object} response.Response{error=string} "Submission not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/forms/submissions/{id} [delete]
func (h *FormHandler) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Submission ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid submission ID format")
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Submission not found")
		}
		return response.InternalError(c, "Failed to delete submission")
	}

	return response.Success(c, form.DeleteResponse{
		Success:   true,
		Message:   "Submission deleted successfully",
		DeletedID: uint(id),
	})
}

// BulkDelete godoc
// @Summary Bulk delete submissions
// @Description Delete multiple form submissions by IDs
// @Tags Forms
// @Accept json
// @Produce json
// @Param request body form.BulkDeleteRequest true "Array of submission IDs to delete"
// @Success 200 {object} form.DeleteResponse "Submissions deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/forms/submissions [delete]
func (h *FormHandler) BulkDelete(c *fiber.Ctx) error {
	var req form.BulkDeleteRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	if len(req.IDs) == 0 {
		return response.BadRequest(c, "No IDs provided")
	}

	deletedCount, err := h.service.BulkDelete(req.IDs)
	if err != nil {
		return response.InternalError(c, "Failed to delete submissions: "+err.Error())
	}

	return response.Success(c, form.DeleteResponse{
		Success:      true,
		Message:      strconv.FormatInt(deletedCount, 10) + " submissions deleted successfully",
		DeletedCount: int(deletedCount),
		DeletedIDs:   req.IDs,
	})
}

// GetStats godoc
// @Summary Get submission statistics
// @Description Get statistics about form submissions (totals by category, status, recent counts)
// @Tags Forms
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=form.SubmissionStats} "Submission statistics"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/forms/stats [get]
func (h *FormHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats()
	if err != nil {
		return response.InternalError(c, "Failed to fetch statistics")
	}

	return response.Success(c, stats)
}

// UpdateStatus godoc
// @Summary Update submission status
// @Description Update the processing status of a form submission
// @Tags Forms
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Submission ID"
// @Param request body map[string]string true "Status update (status: pending/processed/archived)"
// @Success 200 {object} response.Response{data=map[string]string} "Status updated successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 404 {object} response.Response{error=string} "Submission not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/forms/submissions/{id}/status [patch]
func (h *FormHandler) UpdateStatus(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, "Submission ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid submission ID format")
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	// Get user ID from context (set by auth middleware)
	var processedBy *uint
	userID := c.Locals("userID")
	if userID != nil {
		if uid, ok := userID.(uint); ok {
			processedBy = &uid
		}
	}

	if err := h.service.UpdateStatus(uint(id), form.FormStatus(req.Status), processedBy); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Submission not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{
		"message": "Status updated successfully",
		"status":  req.Status,
	})
}
