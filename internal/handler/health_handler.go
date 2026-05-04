package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check godoc
// @Summary Health check
// @Description Check if the API is running and healthy
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]string} "API is healthy"
// @Router /health [get]
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"status": "ok",
		"message": "Healthcare Market Research API is running",
	})
}
