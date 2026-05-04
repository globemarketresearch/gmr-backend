package handler

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"github.com/healthcare-market-research/backend/internal/domain/user"
	"github.com/healthcare-market-research/backend/internal/middleware"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type UserHandler struct {
	userService  service.UserService
	auditService service.AuditService
}

func NewUserHandler(userService service.UserService, auditService service.AuditService) *UserHandler {
	return &UserHandler{
		userService:  userService,
		auditService: auditService,
	}
}

// GetMe godoc
// @Summary Get current user
// @Description Get the currently authenticated user's information
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=user.UserResponse}
// @Failure 401 {object} response.Response{error=string}
// @Router /api/v1/users/me [get]
func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	// Get user from context (set by RequireAuth middleware)
	u, err := middleware.GetUserFromContext(c)
	if err != nil {
		return response.Unauthorized(c, "Authentication required")
	}

	return response.Success(c, u.ToUserResponse())
}

// GetAll godoc
// @Summary Get all users
// @Description Get all users with pagination (admin only)
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response{data=[]user.UserResponse}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Router /api/v1/users [get]
func (h *UserHandler) GetAll(c *fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	// Validate and set defaults
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Get users
	users, total, err := h.userService.GetAll(page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to retrieve users")
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return response.SuccessWithMeta(c, users, &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// Create godoc
// @Summary Create a new user
// @Description Create a new user (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user body user.CreateUserRequest true "User data"
// @Success 201 {object} response.Response{data=user.UserResponse}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Router /api/v1/users [post]
func (h *UserHandler) Create(c *fiber.Ctx) error {
	var req user.CreateUserRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" || req.Name == "" || req.Role == "" {
		return response.BadRequest(c, "Email, password, name, and role are required")
	}

	// Create user
	newUser, err := h.userService.Create(&req)
	if err != nil {
		if err == service.ErrEmailTaken {
			return response.BadRequest(c, "Email already in use")
		}
		if err == service.ErrInvalidUserData {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "Failed to create user")
	}

	// Log user creation
	auditCtx := middleware.GetAuditContext(c)
	entry := middleware.NewAuditEntry(auditCtx, audit.ActionUserCreate)
	entry.EntityType = audit.EntityUser
	entry.EntityID = &newUser.ID
	h.auditService.LogAsync(entry)

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data:    newUser.ToUserResponse(),
	})
}

// Update godoc
// @Summary Update a user
// @Description Update a user's information (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param user body user.UpdateUserRequest true "User data to update"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Failure 404 {object} response.Response{error=string}
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) Update(c *fiber.Ctx) error {
	// Get user ID from params
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	var req user.UpdateUserRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Update user
	if err := h.userService.Update(uint(id), &req); err != nil {
		if err == service.ErrUserNotFound {
			return response.NotFound(c, "User not found")
		}
		if err == service.ErrEmailTaken {
			return response.BadRequest(c, "Email already in use")
		}
		if err == service.ErrInvalidUserData {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "Failed to update user")
	}

	// Log user update (simplified - not tracking specific field changes)
	auditCtx := middleware.GetAuditContext(c)
	entry := middleware.NewAuditEntry(auditCtx, audit.ActionUserUpdate)
	entry.EntityType = audit.EntityUser
	entityID := uint(id)
	entry.EntityID = &entityID
	// For role changes, log specially
	if req.Role != nil {
		roleEntry := middleware.NewAuditEntry(auditCtx, audit.ActionRoleChange)
		roleEntry.EntityType = audit.EntityUser
		roleEntry.EntityID = &entityID
		h.auditService.LogAsync(roleEntry)
	}
	h.auditService.LogAsync(entry)

	return response.Success(c, map[string]string{
		"message": "User updated successfully",
	})
}

// Delete godoc
// @Summary Delete a user
// @Description Soft-delete a user (admin only)
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Failure 404 {object} response.Response{error=string}
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c *fiber.Ctx) error {
	// Get user ID from params
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	// Delete user
	if err := h.userService.Delete(uint(id)); err != nil {
		if err == service.ErrUserNotFound {
			return response.NotFound(c, "User not found")
		}
		return response.InternalError(c, "Failed to delete user")
	}

	// Log user deletion
	auditCtx := middleware.GetAuditContext(c)
	entry := middleware.NewAuditEntry(auditCtx, audit.ActionUserDelete)
	entry.EntityType = audit.EntityUser
	entityID := uint(id)
	entry.EntityID = &entityID
	h.auditService.LogAsync(entry)

	return response.Success(c, map[string]string{
		"message": "User deleted successfully",
	})
}

// GetByID godoc
// @Summary Get user by ID
// @Description Get a specific user by their ID (admin only)
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} response.Response{data=user.UserResponse}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Failure 404 {object} response.Response{error=string}
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetByID(c *fiber.Ctx) error {
	// Get user ID from params
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	// Get user
	user, err := h.userService.GetByID(uint(id))
	if err != nil {
		if err == service.ErrUserNotFound {
			return response.NotFound(c, "User not found")
		}
		return response.InternalError(c, "Failed to retrieve user")
	}

	return response.Success(c, user)
}

// GetByRole godoc
// @Summary Get users by role
// @Description Get all users with a specific role (admin only)
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param role path string true "Role name (admin, editor, or viewer)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response{data=[]user.UserResponse}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Failure 403 {object} response.Response{error=string}
// @Router /api/v1/users/by-role/{role} [get]
func (h *UserHandler) GetByRole(c *fiber.Ctx) error {
	roleName := c.Params("role")

	// Validate role
	if !user.IsValidRole(roleName) {
		return response.BadRequest(c, "Invalid role name")
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	// Validate and set defaults
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Get users by role
	users, total, err := h.userService.GetByRole(roleName, page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to retrieve users")
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return response.SuccessWithMeta(c, users, &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}
