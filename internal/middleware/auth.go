package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/user"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/internal/utils/auth"
	"github.com/healthcare-market-research/backend/pkg/response"
)

// RequireAuth returns a middleware that validates JWT tokens
func RequireAuth(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract token from Authorization header
		tokenString, err := auth.ExtractTokenFromHeader(c)
		if err != nil {
			return response.Unauthorized(c, "Missing or invalid authorization header")
		}

		// Validate token and get user
		u, err := authService.ValidateAccessToken(tokenString)
		if err != nil {
			if err == auth.ErrExpiredToken {
				return response.Unauthorized(c, "Token has expired")
			}
			return response.Unauthorized(c, "Invalid token")
		}

		// Store user in context for use in handlers
		c.Locals("user", u)
		c.Locals("userID", u.ID)

		return c.Next()
	}
}

// RequireRole returns a middleware that checks if the user has one of the required roles
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user from context (set by RequireAuth middleware)
		userInterface := c.Locals("user")
		if userInterface == nil {
			return response.Unauthorized(c, "Authentication required")
		}

		u, ok := userInterface.(*user.User)
		if !ok {
			return response.InternalError(c, "Failed to get user from context")
		}

		// Check if user's role matches any of the required roles
		hasRole := false
		for _, role := range roles {
			if u.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			return response.Forbidden(c, "Insufficient permissions")
		}

		return c.Next()
	}
}

// GetUserFromContext is a helper function to extract user from fiber context
func GetUserFromContext(c *fiber.Ctx) (*user.User, error) {
	userInterface := c.Locals("user")
	if userInterface == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Authentication required")
	}

	u, ok := userInterface.(*user.User)
	if !ok {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get user from context")
	}

	return u, nil
}
