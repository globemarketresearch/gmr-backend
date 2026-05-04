package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"github.com/healthcare-market-research/backend/internal/domain/user"
)

// AuditContext contains audit-related information extracted from request context
type AuditContext struct {
	UserID    *uint
	UserEmail string
	UserRole  string
	IPAddress string
	UserAgent string
	RequestID string
}

// GetAuditContext extracts audit information from fiber context
func GetAuditContext(c *fiber.Ctx) *AuditContext {
	ctx := &AuditContext{
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}

	// Extract request ID from context (set by RequestID middleware)
	if requestID := c.Locals("requestid"); requestID != nil {
		if id, ok := requestID.(string); ok {
			ctx.RequestID = id
		}
	}

	// Get user info if authenticated
	if userInterface := c.Locals("user"); userInterface != nil {
		if u, ok := userInterface.(*user.User); ok {
			ctx.UserID = &u.ID
			ctx.UserEmail = u.Email
			ctx.UserRole = u.Role
		}
	}

	return ctx
}

// NewAuditEntry creates a new audit entry from context with the specified action
func NewAuditEntry(ctx *AuditContext, action string) *audit.AuditEntry {
	return &audit.AuditEntry{
		UserID:    ctx.UserID,
		UserEmail: ctx.UserEmail,
		UserRole:  ctx.UserRole,
		Action:    action,
		IPAddress: ctx.IPAddress,
		UserAgent: ctx.UserAgent,
		RequestID: ctx.RequestID,
		Status:    audit.StatusSuccess,
	}
}
