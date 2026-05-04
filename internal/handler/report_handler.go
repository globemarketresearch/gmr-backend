package handler

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/healthcare-market-research/backend/internal/domain/user"
	"github.com/healthcare-market-research/backend/internal/middleware"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type ReportHandler struct {
	service      service.ReportService
	authorRepo   repository.AuthorRepository
	auditService service.AuditService
}

func NewReportHandler(service service.ReportService, authorRepo repository.AuthorRepository, auditService service.AuditService) *ReportHandler {
	return &ReportHandler{
		service:      service,
		authorRepo:   authorRepo,
		auditService: auditService,
	}
}

// Helper functions

// isAdminOrEditor checks if the current user is an admin or editor
func isAdminOrEditor(c *fiber.Ctx) bool {
	userCtx := c.Locals("user")
	if userCtx == nil {
		return false
	}
	currentUser, ok := userCtx.(*user.User)
	if !ok {
		return false
	}
	return currentUser.Role == "admin" || currentUser.Role == "editor"
}

// stripAdminFields removes sensitive admin fields from reports for public API responses
func stripAdminFields(reports []report.Report) []report.Report {
	for i := range reports {
		reports[i].CreatedBy = nil
		reports[i].UpdatedBy = nil
		reports[i].InternalNotes = ""
	}
	return reports
}

// getUserID safely extracts user ID from fiber context
func getUserID(c *fiber.Ctx) (uint, error) {
	userCtx := c.Locals("user")
	if userCtx == nil {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "User not authenticated")
	}
	currentUser, ok := userCtx.(*user.User)
	if !ok {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "Invalid user context")
	}
	return currentUser.ID, nil
}

// GetAll godoc
// @Summary Get all reports with optional filters
// @Description Get a paginated list of healthcare market research reports with optional filters for status, category, geography, and search. Admin users can use additional filters for user tracking and date ranges. Admin-only fields are automatically included in responses for authenticated admin/editor users.
// @Tags Reports
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param status query string false "Filter by status (draft or published)"
// @Param category query string false "Filter by category slug"
// @Param geography query string false "Filter by geography (comma-separated, e.g., 'North America,Europe')"
// @Param search query string false "Search in title, summary, and description"
// @Param deleted query string false "Admin only: Show deleted reports (true/false, default: false)"
// @Param created_by query int false "Admin only: Filter by creator user ID"
// @Param updated_by query int false "Admin only: Filter by last updater user ID"
// @Param created_after query string false "Admin only: Filter by created date (ISO 8601, e.g., 2024-01-01T00:00:00Z)"
// @Param created_before query string false "Admin only: Filter by created date (ISO 8601)"
// @Param updated_after query string false "Admin only: Filter by updated date (ISO 8601)"
// @Param updated_before query string false "Admin only: Filter by updated date (ISO 8601)"
// @Param published_after query string false "Admin only: Filter by published date (ISO 8601)"
// @Param published_before query string false "Admin only: Filter by published date (ISO 8601)"
// @Success 200 {object} response.Response{data=[]report.Report,meta=response.Meta} "List of reports with pagination metadata. Admin fields (created_by, updated_by, internal_notes) are included only for authenticated admin/editor users."
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports [get]
func (h *ReportHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Check if any filters are provided
	status := c.Query("status")
	category := c.Query("category")
	geographyParam := c.Query("geography")
	search := c.Query("search")
	deletedParam := c.Query("deleted")

	// Admin filters
	var createdBy *uint
	if createdByStr := c.Query("created_by"); createdByStr != "" {
		id, err := strconv.ParseUint(createdByStr, 10, 32)
		if err == nil {
			val := uint(id)
			createdBy = &val
		}
	}

	var updatedBy *uint
	if updatedByStr := c.Query("updated_by"); updatedByStr != "" {
		id, err := strconv.ParseUint(updatedByStr, 10, 32)
		if err == nil {
			val := uint(id)
			updatedBy = &val
		}
	}

	// Date range parsing
	var createdAfter, createdBefore, updatedAfter, updatedBefore, publishedAfter, publishedBefore *time.Time
	if after := c.Query("created_after"); after != "" {
		if t, err := time.Parse(time.RFC3339, after); err == nil {
			createdAfter = &t
		}
	}
	if before := c.Query("created_before"); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			createdBefore = &t
		}
	}
	if after := c.Query("updated_after"); after != "" {
		if t, err := time.Parse(time.RFC3339, after); err == nil {
			updatedAfter = &t
		}
	}
	if before := c.Query("updated_before"); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			updatedBefore = &t
		}
	}
	if after := c.Query("published_after"); after != "" {
		if t, err := time.Parse(time.RFC3339, after); err == nil {
			publishedAfter = &t
		}
	}
	if before := c.Query("published_before"); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			publishedBefore = &t
		}
	}

	// Validate sort_by parameter
	var sortBy string
	if s := c.Query("sort_by", ""); s != "" {
		allowed := map[string]string{
			"publish_date_desc": "r.publish_date DESC NULLS LAST",
			"created_at_desc":   "r.created_at DESC",
			"updated_at_desc":   "r.updated_at DESC",
		}
		if mapped, ok := allowed[s]; ok {
			sortBy = mapped
		}
	}

	var reports []report.Report
	var total int64
	var err error

	// Parse deleted parameter
	showDeleted := deletedParam == "true"

	// If any filters are provided, use the filtered endpoint
	hasFilters := status != "" || category != "" || geographyParam != "" || search != "" ||
		createdBy != nil || updatedBy != nil ||
		createdAfter != nil || createdBefore != nil || updatedAfter != nil || updatedBefore != nil ||
		publishedAfter != nil || publishedBefore != nil || showDeleted || sortBy != ""

	if hasFilters {
		// Parse geography into array
		var geography []string
		if geographyParam != "" {
			geography = strings.Split(geographyParam, ",")
			// Trim whitespace from each geography
			for i := range geography {
				geography[i] = strings.TrimSpace(geography[i])
			}
		}

		filters := repository.ReportFilters{
			Status:          status,
			Category:        category,
			Geography:       geography,
			Search:          search,
			CreatedBy:       createdBy,
			UpdatedBy:       updatedBy,
			CreatedAfter:    createdAfter,
			CreatedBefore:   createdBefore,
			UpdatedAfter:    updatedAfter,
			UpdatedBefore:   updatedBefore,
			PublishedAfter:  publishedAfter,
			PublishedBefore: publishedBefore,
			ShowDeleted:     showDeleted,
			SortBy:          sortBy,
			Page:            page,
			Limit:           limit,
		}

		reports, total, err = h.service.GetAllWithFilters(filters)
	} else {
		// No filters, use the default GetAll
		reports, total, err = h.service.GetAll(page, limit)
	}

	if err != nil {
		return response.InternalError(c, "Failed to fetch reports")
	}

	// Strip admin fields if user is not admin/editor
	responseData := reports
	if !isAdminOrEditor(c) {
		responseData = stripAdminFields(reports)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, responseData, meta)
}

// GetBySlug godoc
// @Summary Get report by slug or ID
// @Description Get a single healthcare market research report by its unique slug or numeric ID. If the parameter is numeric, it will be treated as an ID; otherwise, it will be treated as a slug.
// @Tags Reports
// @Accept json
// @Produce json
// @Param slug path string true "Report slug or ID"
// @Success 200 {object} response.Response{data=report.ReportWithRelations} "Report details with relations"
// @Failure 400 {object} response.Response{error=string} "Bad request - slug or ID is required"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Router /api/v1/reports/{slug} [get]
func (h *ReportHandler) GetBySlug(c *fiber.Ctx) error {
	param := c.Params("slug")
	if param == "" {
		return response.BadRequest(c, "Slug or ID is required")
	}

	// Try to parse as uint (ID)
	if id, err := strconv.ParseUint(param, 10, 32); err == nil {
		// It's a numeric ID
		report, err := h.service.GetByID(uint(id))
		if err != nil {
			return response.NotFound(c, "Report not found")
		}
		return response.Success(c, report)
	}

	// It's a slug
	report, err := h.service.GetBySlug(param)
	if err != nil {
		return response.NotFound(c, "Report not found")
	}

	return response.Success(c, report)
}

// GetByCategorySlug godoc
// @Summary Get reports by category slug
// @Description Get a paginated list of reports belonging to a specific category
// @Tags Reports
// @Accept json
// @Produce json
// @Param slug path string true "Category slug"
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} response.Response{data=[]report.ReportWithRelations,meta=response.Meta} "List of reports in category with pagination metadata"
// @Failure 400 {object} response.Response{error=string} "Bad request - category slug is required"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/categories/{slug}/reports [get]
func (h *ReportHandler) GetByCategorySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.BadRequest(c, "Category slug is required")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	reports, total, err := h.service.GetByCategorySlug(slug, page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to fetch reports")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, reports, meta)
}

// GetByAuthorID godoc
// @Summary Get reports by author ID
// @Description Get a paginated list of reports that include the specified author
// @Tags Reports
// @Param id path int true "Author ID"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} response.Response{data=[]report.Report,meta=response.Meta}
// @Failure 400 {object} response.Response "Invalid author ID"
// @Failure 404 {object} response.Response "Author not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /api/v1/reports/author/{id} [get]
func (h *ReportHandler) GetByAuthorID(c *fiber.Ctx) error {
	// Parse and validate author ID
	authorID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid author ID")
	}

	// Validate author exists
	_, err = h.authorRepo.GetByID(uint(authorID))
	if err != nil {
		return response.NotFound(c, "Author not found")
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	// Validate and normalize pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Fetch reports
	reports, total, err := h.service.GetByAuthorID(uint(authorID), page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to fetch reports")
	}

	// Strip admin fields if user is not admin/editor
	responseData := reports
	if !isAdminOrEditor(c) {
		responseData = stripAdminFields(reports)
	}

	// Calculate pagination metadata
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, responseData, meta)
}

// Search godoc
// @Summary Search reports
// @Description Search for healthcare market research reports by query text (searches in title, description, and summary)
// @Tags Reports
// @Accept json
// @Produce json
// @Param q query string true "Search query text"
// @Param page query int false "Page number (default: 1, min: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} response.Response{data=[]report.ReportWithRelations,meta=response.Meta} "Search results with pagination metadata"
// @Failure 400 {object} response.Response{error=string} "Bad request - search query is required"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/search [get]
func (h *ReportHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return response.BadRequest(c, "Search query is required")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	reports, total, err := h.service.Search(query, page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to search reports")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, reports, meta)
}

// Create godoc
// @Summary Create a new report
// @Description Create a new healthcare market research report. Automatically sets created_by and updated_by to the authenticated user. Requires admin or editor role.
// @Tags Reports
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param report body report.Report true "Report data"
// @Success 201 {object} response.Response{data=report.Report} "Created report with auto-populated admin tracking fields"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports [post]
func (h *ReportHandler) Create(c *fiber.Ctx) error {
	// Get authenticated user ID
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	var req report.Report
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	// Validate required fields based on frontend expectations
	if req.Title == "" {
		return response.BadRequest(c, "Title is required (minimum 10 characters)")
	}
	if req.Slug == "" {
		return response.BadRequest(c, "Slug is required")
	}
	if req.CategoryID == 0 {
		return response.BadRequest(c, "Category ID is required")
	}
	if req.Summary == "" {
		return response.BadRequest(c, "Summary is required (minimum 50 characters)")
	}
	if len(req.Geography) == 0 {
		return response.BadRequest(c, "At least one geography is required")
	}

	// Set default values if not provided
	if req.Status == "" {
		req.Status = "draft"
	}

	if err := h.service.Create(&req, userID); err != nil {
		return response.InternalError(c, "Failed to create report")
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	entry := middleware.NewAuditEntry(auditCtx, audit.ActionReportCreate)
	entry.EntityType = audit.EntityReport
	entry.EntityID = &req.ID
	h.auditService.LogAsync(entry)

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data:    req,
	})
}

// Update godoc
// @Summary Update a report
// @Description Update an existing healthcare market research report by ID. Automatically updates updated_by field. Creates version history when status changes to 'published'. Requires admin or editor role.
// @Tags Reports
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Param report body report.Report true "Updated report data"
// @Success 200 {object} response.Response{data=report.Report} "Updated report with auto-updated admin tracking fields"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid input"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports/{id} [put]
func (h *ReportHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	// Get user ID from context (set by auth middleware)
	userCtx := c.Locals("user")
	if userCtx == nil {
		return response.Unauthorized(c, "User not authenticated")
	}
	currentUser, ok := userCtx.(*user.User)
	if !ok {
		return response.Unauthorized(c, "Invalid user context")
	}

	var req report.Report
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	// Validate required fields
	if req.Title == "" {
		return response.BadRequest(c, "Title is required (minimum 10 characters)")
	}
	if req.Slug == "" {
		return response.BadRequest(c, "Slug is required")
	}
	if req.CategoryID == 0 {
		return response.BadRequest(c, "Category ID is required")
	}
	if req.Summary == "" {
		return response.BadRequest(c, "Summary is required (minimum 50 characters)")
	}
	if len(req.Geography) == 0 {
		return response.BadRequest(c, "At least one geography is required")
	}

	// Pass user ID to service for version history
	if err := h.service.Update(uint(id), &req, currentUser.ID); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Report not found")
		}
		return response.InternalError(c, "Failed to update report")
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionReportUpdate)
	auditEntry.EntityType = audit.EntityReport
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, req)
}

// Delete godoc
// @Summary Hard delete a report (permanent)
// @Description Permanently delete a healthcare market research report by ID. This action cannot be undone. The report and all associated data (images, versions, charts) will be permanently removed. Requires admin role.
// @Tags Reports
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Success 200 {object} response.Response{data=map[string]string} "Report permanently deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin role"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports/{id} [delete]
func (h *ReportHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Report not found")
		}
		return response.InternalError(c, "Failed to delete report")
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionReportDelete)
	auditEntry.EntityType = audit.EntityReport
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, fiber.Map{
		"message": "Report permanently deleted successfully",
	})
}

// SoftDelete godoc
// @Summary Soft delete a report
// @Description Soft delete a report by marking it as deleted. The report can be restored later. Requires admin or editor role.
// @Tags Reports
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Success 200 {object} response.Response{data=map[string]string} "Report soft deleted successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin or editor role"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports/{id}/soft-delete [patch]
func (h *ReportHandler) SoftDelete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	if err := h.service.SoftDelete(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Report not found")
		}
		return response.InternalError(c, "Failed to soft delete report")
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionReportDelete)
	auditEntry.EntityType = audit.EntityReport
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, fiber.Map{
		"message": "Report soft deleted successfully",
	})
}

// Restore godoc
// @Summary Restore a soft-deleted report
// @Description Restore a previously soft-deleted report. Requires admin role.
// @Tags Reports
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Success 200 {object} response.Response{data=map[string]string} "Report restored successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request - invalid ID"
// @Failure 401 {object} response.Response{error=string} "Unauthorized"
// @Failure 403 {object} response.Response{error=string} "Forbidden - requires admin role"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/reports/{id}/restore [patch]
func (h *ReportHandler) Restore(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID")
	}

	if err := h.service.Restore(uint(id)); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Report not found")
		}
		return response.InternalError(c, "Failed to restore report")
	}

	// Audit log
	auditCtx := middleware.GetAuditContext(c)
	auditEntry := middleware.NewAuditEntry(auditCtx, audit.ActionReportUpdate)
	auditEntry.EntityType = audit.EntityReport
	entityID := uint(id)
	auditEntry.EntityID = &entityID
	h.auditService.LogAsync(auditEntry)

	return response.Success(c, fiber.Map{
		"message": "Report restored successfully",
	})
}

// SchedulePublish godoc
// @Summary Schedule report publish
// @Description Schedule a report to be published at a future date/time
// @Tags Reports
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Param request body object true "Publish date"
// @Success 200 {object} object "Report scheduled successfully"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 404 {object} response.Response{error=string} "Report not found"
// @Router /api/v1/reports/{id}/schedule [patch]
func (h *ReportHandler) SchedulePublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID format")
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

	r, err := h.service.SchedulePublish(uint(id), publishDate)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(fiber.Map{"report": r})
}

// CancelScheduledPublish godoc
// @Summary Cancel scheduled publish
// @Description Cancel scheduled publishing for a report
// @Tags Reports
// @Accept json
// @Produce json
// @Param id path int true "Report ID"
// @Success 200 {object} object "Schedule cancelled"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Router /api/v1/reports/{id}/cancel-schedule [patch]
func (h *ReportHandler) CancelScheduledPublish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid report ID format")
	}

	r, err := h.service.CancelScheduledPublish(uint(id))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.JSON(fiber.Map{"report": r})
}

// GetLinkSuggestions returns a lightweight list of published reports for internal linking
func (h *ReportHandler) GetLinkSuggestions(c *fiber.Ctx) error {
	items, err := h.service.GetLinkSuggestions()
	if err != nil {
		return response.InternalError(c, "Failed to fetch link suggestions")
	}
	return response.Success(c, items)
}

// GetSitemap godoc
// @Summary Get published report slugs for sitemap generation
// @Description Returns paginated slugs and dates for published reports (up to 1000 per page). Intended for sitemap generation only.
// @Tags Reports
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 1000, max: 1000)"
// @Success 200 {object} response.Response "Sitemap data"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/sitemap/reports [get]
func (h *ReportHandler) GetSitemap(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "1000"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 1000 {
		limit = 1000
	}

	reports, total, err := h.service.GetAllWithFilters(repository.ReportFilters{
		Status: "published",
		Page:   page,
		Limit:  limit,
		SortBy: "r.publish_date DESC NULLS LAST",
	})
	if err != nil {
		return response.InternalError(c, "Failed to fetch reports for sitemap")
	}

	type sitemapItem struct {
		Slug        string     `json:"slug"`
		UpdatedAt   time.Time  `json:"updated_at"`
		PublishDate *time.Time `json:"publish_date,omitempty"`
	}

	items := make([]sitemapItem, len(reports))
	for i, r := range reports {
		items[i] = sitemapItem{
			Slug:        r.Slug,
			UpdatedAt:   r.UpdatedAt,
			PublishDate: r.PublishDate,
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
