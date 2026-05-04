package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/user"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type DashboardHandler struct {
	service service.DashboardService
}

func NewDashboardHandler(service service.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

// GetStats godoc
// @Summary Get dashboard statistics
// @Description Retrieves comprehensive dashboard statistics including reports, blogs, press releases, users, and leads. User stats are only included for admin users.
// @Tags Dashboard
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=dashboard.DashboardStats} "Dashboard statistics"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/dashboard/stats [get]
func (h *DashboardHandler) GetStats(c *fiber.Ctx) error {
	// Get user from context (set by auth middleware)
	userInterface := c.Locals("user")
	if userInterface == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	// Extract user and role
	u, ok := userInterface.(*user.User)
	if !ok {
		return response.InternalError(c, "Failed to get user from context")
	}

	stats, err := h.service.GetStats(u.Role)
	if err != nil {
		return response.InternalError(c, "Failed to fetch dashboard statistics: "+err.Error())
	}

	return response.Success(c, stats)
}

// GetActivity godoc
// @Summary Get recent activity feed
// @Description Retrieves recent system activities from audit logs
// @Tags Dashboard
// @Accept json
// @Produce json
// @Security Bearer
// @Param limit query int false "Number of activities to return (default: 10, max: 50)"
// @Success 200 {object} response.Response{data=dashboard.ActivityResponse} "Recent activities"
// @Failure 401 {object} response.Response{error=string} "Unauthorized - authentication required"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/dashboard/activity [get]
func (h *DashboardHandler) GetActivity(c *fiber.Ctx) error {
	// Get user from context (set by auth middleware)
	user := c.Locals("user")
	if user == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	// Parse limit parameter
	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	activities, err := h.service.GetRecentActivity(limit)
	if err != nil {
		return response.InternalError(c, "Failed to fetch recent activities: "+err.Error())
	}

	return response.Success(c, activities)
}
