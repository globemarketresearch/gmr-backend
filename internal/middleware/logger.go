package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/pkg/logger"
)

func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start)
		logger.LogRequest(c, duration)

		return err
	}
}
