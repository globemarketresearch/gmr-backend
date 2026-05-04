package handler

import (
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

// AuditHandler handles HTTP requests for audit log operations
type AuditHandler struct {
	auditService service.AuditService
}

// NewAuditHandler creates a new audit handler instance
func NewAuditHandler(auditService service.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// GetAll godoc
// @Summary Get all audit logs
// @Description Get all audit logs with filtering and pagination (admin only)
// @Tags Audit
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param user_id query int false "Filter by user ID"
// @Param action query string false "Filter by action (e.g., auth.login, user.create)"
// @Param entity_type query string false "Filter by entity type (e.g., user, report)"
// @Param entity_id query int false "Filter by entity ID"
// @Param status query string false "Filter by status (success or failure)"
// @Param start_date query string false "Filter by start date (RFC3339 format)"
// @Param end_date query string false "Filter by end date (RFC3339 format)"
// @Param ip_address query string false "Filter by IP address"
// @Success 200 {object} response.Response{data=[]audit.AuditLogResponse}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Router /api/v1/audit-logs [get]
func (h *AuditHandler) GetAll(c *fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filters := audit.AuditLogFilters{
		Page:  page,
		Limit: limit,
	}

	// Parse optional filters
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			val := uint(id)
			filters.UserID = &val
		}
	}

	if action := c.Query("action"); action != "" {
		filters.Action = action
	}

	if actionPrefix := c.Query("action_prefix"); actionPrefix != "" {
		filters.ActionPrefix = actionPrefix
	}

	if entityType := c.Query("entity_type"); entityType != "" {
		filters.EntityType = entityType
	}

	if entityIDStr := c.Query("entity_id"); entityIDStr != "" {
		if id, err := strconv.ParseUint(entityIDStr, 10, 32); err == nil {
			val := uint(id)
			filters.EntityID = &val
		}
	}

	if status := c.Query("status"); status != "" {
		filters.Status = status
	}

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			filters.StartDate = &t
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			filters.EndDate = &t
		}
	}

	if ipAddress := c.Query("ip_address"); ipAddress != "" {
		filters.IPAddress = ipAddress
	}

	// Retrieve audit logs
	logs, total, err := h.auditService.GetAll(filters)
	if err != nil {
		return response.InternalError(c, "Failed to retrieve audit logs")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return response.SuccessWithMeta(c, logs, &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetByID godoc
// @Summary Get audit log by ID
// @Description Get a specific audit log by ID (admin only)
// @Tags Audit
// @Produce json
// @Security BearerAuth
// @Param id path int true "Audit Log ID"
// @Success 200 {object} response.Response{data=audit.AuditLogResponse}
// @Failure 400 {object} response.Response{error=string}
// @Failure 404 {object} response.Response{error=string}
// @Router /api/v1/audit-logs/{id} [get]
func (h *AuditHandler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid audit log ID")
	}

	log, err := h.auditService.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Audit log not found")
	}

	return response.Success(c, log)
}
