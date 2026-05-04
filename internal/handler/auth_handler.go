package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"github.com/healthcare-market-research/backend/internal/domain/user"
	"github.com/healthcare-market-research/backend/internal/middleware"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type AuthHandler struct {
	authService  service.AuthService
	auditService service.AuditService
}

func NewAuthHandler(authService service.AuthService, auditService service.AuditService) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		auditService: auditService,
	}
}

// Login godoc
// @Summary User login
// @Description Authenticate user with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param credentials body user.LoginRequest true "Login credentials"
// @Success 200 {object} response.Response{data=user.LoginResponse}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Failure 429 {object} response.Response{error=string}
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req user.LoginRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		return response.BadRequest(c, "Email and password are required")
	}

	// Validate email format (basic validation)
	if !strings.Contains(req.Email, "@") {
		return response.BadRequest(c, "Invalid email format")
	}

	// Authenticate user
	loginResp, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		// Log failed login attempt
		auditCtx := middleware.GetAuditContext(c)
		entry := middleware.NewAuditEntry(auditCtx, audit.ActionLoginFailed)
		entry.Status = audit.StatusFailure
		entry.ErrorMessage = "Invalid credentials"
		entry.UserEmail = req.Email // Set email even though login failed
		h.auditService.LogAsync(entry)

		if err == service.ErrInvalidCredentials {
			return response.Unauthorized(c, "Invalid email or password")
		}
		return response.InternalError(c, "Failed to authenticate user")
	}

	// Log successful login
	auditCtx := middleware.GetAuditContext(c)
	entry := middleware.NewAuditEntry(auditCtx, audit.ActionLogin)
	entry.UserID = &loginResp.User.ID
	entry.UserEmail = loginResp.User.Email
	entry.UserRole = loginResp.User.Role
	entry.EntityType = audit.EntityUser
	entry.EntityID = &loginResp.User.ID
	h.auditService.LogAsync(entry)

	return response.Success(c, loginResp)
}

// Refresh godoc
// @Summary Refresh access token
// @Description Generate new access token using refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param refresh body user.RefreshRequest true "Refresh token"
// @Success 200 {object} response.Response{data=user.RefreshResponse}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req user.RefreshRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate refresh token
	if req.RefreshToken == "" {
		return response.BadRequest(c, "Refresh token is required")
	}

	// Refresh tokens
	refreshResp, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		if err == service.ErrInvalidToken || err == service.ErrTokenRevoked {
			return response.Unauthorized(c, "Invalid or expired refresh token")
		}
		return response.InternalError(c, "Failed to refresh token")
	}

	return response.Success(c, refreshResp)
}

// Logout godoc
// @Summary User logout
// @Description Invalidate refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param logout body user.LogoutRequest false "Refresh token to invalidate"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response{error=string}
// @Failure 401 {object} response.Response{error=string}
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// Get user from context (set by RequireAuth middleware)
	u, err := middleware.GetUserFromContext(c)
	if err != nil {
		return response.Unauthorized(c, "Authentication required")
	}

	var req user.LogoutRequest

	// Parse request body (refresh token is optional in body)
	if err := c.BodyParser(&req); err != nil {
		// If no body, that's okay - we might not have refresh token
		req.RefreshToken = ""
	}

	// If refresh token provided, invalidate it
	if req.RefreshToken != "" {
		if err := h.authService.Logout(u.ID, req.RefreshToken); err != nil {
			// Don't fail logout if token invalidation fails
			// Just log the error
		}
	}

	// Log successful logout
	auditCtx := middleware.GetAuditContext(c)
	entry := middleware.NewAuditEntry(auditCtx, audit.ActionLogout)
	entry.EntityType = audit.EntityUser
	entry.EntityID = &u.ID
	h.auditService.LogAsync(entry)

	return response.Success(c, map[string]string{
		"message": "Logout successful",
	})
}
