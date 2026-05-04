package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/pkg/response"
)

// RateLimit returns a middleware that implements rate limiting using Redis
func RateLimit(maxAttempts int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get client IP
		clientIP := c.IP()

		// Create rate limit key based on IP and endpoint
		endpoint := c.Path()
		rateLimitKey := fmt.Sprintf("rate_limit:%s:%s", clientIP, endpoint)

		// Get current count
		var currentCount int
		err := cache.Get(rateLimitKey, &currentCount)

		if err != nil {
			// Key doesn't exist, this is the first attempt
			currentCount = 0
		}

		// Check if limit exceeded
		if currentCount >= maxAttempts {
			retryAfter := int(window.Seconds())
			return response.TooManyRequests(c, "Too many requests. Please try again later.", retryAfter)
		}

		// Increment counter
		currentCount++

		// Store updated count with TTL
		if err := cache.Set(rateLimitKey, currentCount, window); err != nil {
			// Log error but don't fail the request if Redis is unavailable
			fmt.Printf("Warning: Failed to update rate limit in Redis: %v\n", err)
		}

		// Add rate limit headers
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", maxAttempts))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", maxAttempts-currentCount))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(window).Unix()))

		return c.Next()
	}
}
