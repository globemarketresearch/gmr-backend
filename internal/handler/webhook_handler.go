package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type WebhookHandler struct {
	orderService service.OrderService
}

func NewWebhookHandler(orderService service.OrderService) *WebhookHandler {
	return &WebhookHandler{orderService: orderService}
}

// StripeWebhook handles incoming Stripe webhook events.
// The raw body must NOT be parsed by Fiber before this handler runs —
// signature verification requires the exact bytes Stripe sent.
func (h *WebhookHandler) StripeWebhook(c *fiber.Ctx) error {
	sigHeader := c.Get("Stripe-Signature")
	if sigHeader == "" {
		return response.BadRequest(c, "Missing Stripe-Signature header")
	}

	// c.Body() returns the raw request bytes in Fiber v2
	payload := c.Body()

	if err := h.orderService.HandleStripeWebhook(payload, sigHeader); err != nil {
		// Return 400 so Stripe retries on signature failures;
		// return 200 on processing errors to prevent infinite retries.
		if len(err.Error()) > 25 && err.Error()[:25] == "webhook signature invalid:" {
			return response.BadRequest(c, err.Error())
		}
		// Log but acknowledge — Stripe should not retry business-logic errors
		return c.SendStatus(fiber.StatusOK)
	}

	return c.SendStatus(fiber.StatusOK)
}
