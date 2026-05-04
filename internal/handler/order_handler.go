package handler

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/order"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type OrderHandler struct {
	service service.OrderService
}

func NewOrderHandler(service service.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

// Create godoc
// @Summary Create an order
// @Description Create a new order for a report and get a PayPal order ID
// @Tags Orders
// @Accept json
// @Produce json
// @Param request body order.CreateOrderRequest true "Order details"
// @Success 201 {object} order.CreateOrderResponse "Order created"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/orders [post]
func (h *OrderHandler) Create(c *fiber.Ctx) error {
	var req order.CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	result, err := h.service.CreateOrder(&req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// Capture godoc
// @Summary Capture payment for an order
// @Description Capture the PayPal payment for a pending order
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} order.CaptureOrderResponse "Payment captured"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 404 {object} response.Response{error=string} "Order not found"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/orders/{id}/capture [post]
func (h *OrderHandler) Capture(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid order ID")
	}

	result, err := h.service.CaptureOrder(uint(id))
	if err != nil {
		if err.Error() == "order not found" {
			return response.NotFound(c, "Order not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, result)
}

// StripeCapture godoc
// @Summary Confirm Stripe payment for an order
// @Description Verify the Stripe PaymentIntent succeeded and mark the order as paid
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} order.ConfirmStripeOrderResponse "Payment confirmed"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 404 {object} response.Response{error=string} "Order not found"
// @Router /api/v1/orders/{id}/stripe-capture [post]
func (h *OrderHandler) StripeCapture(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid order ID")
	}

	result, err := h.service.CaptureStripeOrder(uint(id))
	if err != nil {
		if err.Error() == "order not found" {
			return response.NotFound(c, "Order not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, result)
}

// GetAll godoc
// @Summary List all orders
// @Description Get paginated list of orders with optional filtering
// @Tags Orders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param status query string false "Filter by status"
// @Param search query string false "Search by customer name/email/company/report"
// @Param dateFrom query string false "Start date (RFC3339)"
// @Param dateTo query string false "End date (RFC3339)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param sortBy query string false "Sort field: createdAt, amount, status"
// @Param sortOrder query string false "Sort order: asc, desc (default: desc)"
// @Success 200 {object} response.Response{data=[]order.Order,meta=response.Meta} "Orders list"
// @Failure 401 {object} response.Response{error=string} "Unauthorized"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/orders [get]
func (h *OrderHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	query := order.GetOrdersQuery{
		Status:    c.Query("status", ""),
		Search:    c.Query("search", ""),
		DateFrom:  c.Query("dateFrom", ""),
		DateTo:    c.Query("dateTo", ""),
		Page:      page,
		Limit:     limit,
		SortBy:    c.Query("sortBy", ""),
		SortOrder: c.Query("sortOrder", ""),
	}

	orders, total, err := h.service.GetAll(query)
	if err != nil {
		return response.InternalError(c, "Failed to fetch orders")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return response.SuccessWithMeta(c, orders, meta)
}

// GetStats godoc
// @Summary Get order statistics
// @Description Get aggregate order statistics for the admin dashboard
// @Tags Orders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=order.OrderStats} "Order statistics"
// @Failure 401 {object} response.Response{error=string} "Unauthorized"
// @Failure 500 {object} response.Response{error=string} "Internal server error"
// @Router /api/v1/orders/stats [get]
func (h *OrderHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats()
	if err != nil {
		return response.InternalError(c, "Failed to fetch order statistics")
	}
	return response.Success(c, stats)
}

// GetByID godoc
// @Summary Get single order
// @Description Get a single order by ID
// @Tags Orders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} response.Response{data=order.Order} "Order details"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 401 {object} response.Response{error=string} "Unauthorized"
// @Failure 404 {object} response.Response{error=string} "Order not found"
// @Router /api/v1/orders/{id} [get]
func (h *OrderHandler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid order ID")
	}

	o, err := h.service.GetByID(uint(id))
	if err != nil {
		return response.NotFound(c, "Order not found")
	}

	return response.Success(c, o)
}

// UpdateStatus godoc
// @Summary Update order status
// @Description Admin endpoint to update order status and notes
// @Tags Orders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Param request body order.UpdateStatusRequest true "Status update"
// @Success 200 {object} response.Response{data=map[string]string} "Status updated"
// @Failure 400 {object} response.Response{error=string} "Bad request"
// @Failure 401 {object} response.Response{error=string} "Unauthorized"
// @Failure 404 {object} response.Response{error=string} "Order not found"
// @Router /api/v1/orders/{id}/status [patch]
func (h *OrderHandler) UpdateStatus(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid order ID")
	}

	var req order.UpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body: "+err.Error())
	}

	// Get user ID from context (set by auth middleware)
	var updatedBy *uint
	if userID := c.Locals("userID"); userID != nil {
		if uid, ok := userID.(uint); ok {
			updatedBy = &uid
		}
	}

	if err := h.service.UpdateStatus(uint(id), &req, updatedBy); err != nil {
		if err.Error() == "record not found" {
			return response.NotFound(c, "Order not found")
		}
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{
		"message": "Status updated successfully",
		"status":  req.Status,
	})
}
