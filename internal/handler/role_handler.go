package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/role"
	"github.com/healthcare-market-research/backend/pkg/response"
)

// RoleHandler handles HTTP requests for role operations
type RoleHandler struct{}

// NewRoleHandler creates a new role handler instance
func NewRoleHandler() *RoleHandler {
	return &RoleHandler{}
}

// GetAll godoc
// @Summary Get all roles
// @Description Get all available roles with their permissions and descriptions
// @Tags Roles
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]role.RoleInfo}
// @Failure 401 {object} response.Response{error=string}
// @Router /api/v1/roles [get]
func (h *RoleHandler) GetAll(c *fiber.Ctx) error {
	roles := role.GetAllRoles()
	return response.Success(c, roles)
}

// GetByName godoc
// @Summary Get role by name
// @Description Get detailed information about a specific role
// @Tags Roles
// @Produce json
// @Security BearerAuth
// @Param name path string true "Role name (admin, editor, or viewer)"
// @Success 200 {object} response.Response{data=role.RoleInfo}
// @Failure 404 {object} response.Response{error=string}
// @Router /api/v1/roles/{name} [get]
func (h *RoleHandler) GetByName(c *fiber.Ctx) error {
	roleName := c.Params("name")

	roleInfo, exists := role.GetRoleInfo(roleName)
	if !exists {
		return response.NotFound(c, "Role not found")
	}

	return response.Success(c, roleInfo)
}
