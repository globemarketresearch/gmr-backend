package service

import (
	"fmt"

	"github.com/healthcare-market-research/backend/internal/domain/order"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/pkg/email"
	"github.com/healthcare-market-research/backend/pkg/logger"
	"github.com/healthcare-market-research/backend/pkg/paypal"
	"github.com/healthcare-market-research/backend/pkg/stripe"
)

type OrderService interface {
	CreateOrder(req *order.CreateOrderRequest) (*order.CreateOrderResponse, error)
	CaptureOrder(orderID uint) (*order.CaptureOrderResponse, error)
	CaptureStripeOrder(orderID uint) (*order.ConfirmStripeOrderResponse, error)
	HandleStripeWebhook(payload []byte, sigHeader string) error
	GetAll(query order.GetOrdersQuery) ([]order.Order, int64, error)
	GetByID(id uint) (*order.Order, error)
	GetStats() (*order.OrderStats, error)
	UpdateStatus(id uint, req *order.UpdateStatusRequest, updatedBy *uint) error
}

type orderService struct {
	repo         repository.OrderRepository
	reportRepo   repository.ReportRepository
	paypalClient *paypal.Client
	stripeClient *stripe.Client
	emailSvc     email.EmailService
}

func NewOrderService(
	repo repository.OrderRepository,
	reportRepo repository.ReportRepository,
	paypalClient *paypal.Client,
	stripeClient *stripe.Client,
	emailSvc email.EmailService,
) OrderService {
	return &orderService{
		repo:         repo,
		reportRepo:   reportRepo,
		paypalClient: paypalClient,
		stripeClient: stripeClient,
		emailSvc:     emailSvc,
	}
}

func (s *orderService) CreateOrder(req *order.CreateOrderRequest) (*order.CreateOrderResponse, error) {
	// Validate required fields
	if req.CustomerName == "" {
		return nil, fmt.Errorf("customer_name is required")
	}
	if req.CustomerEmail == "" {
		return nil, fmt.Errorf("customer_email is required")
	}
	if req.ReportSlug == "" {
		return nil, fmt.Errorf("report_slug is required")
	}

	// Default to PayPal if not specified
	paymentMethod := req.PaymentMethod
	if paymentMethod != "stripe" {
		paymentMethod = "paypal"
	}

	// Look up the report from DB (price comes from DB only, never from browser)
	report, err := s.reportRepo.GetBySlug(req.ReportSlug)
	if err != nil {
		return nil, fmt.Errorf("report not found: %s", req.ReportSlug)
	}

	// Use discounted price if available, else regular price
	price := report.Price
	if report.DiscountedPrice > 0 {
		price = report.DiscountedPrice
	}

	if price <= 0 {
		return nil, fmt.Errorf("report does not have a valid price")
	}

	description := fmt.Sprintf("Healthcare Market Research Report: %s", report.Title)

	o := &order.Order{
		CustomerName:    req.CustomerName,
		CustomerEmail:   req.CustomerEmail,
		CustomerCompany: req.CustomerCompany,
		CustomerPhone:   req.CustomerPhone,
		CustomerCountry: req.CustomerCountry,
		ReportID:        uint(report.ID),
		ReportTitle:     report.Title,
		ReportSlug:      report.Slug,
		Amount:          price,
		Currency:        "USD",
		PaymentMethod:   paymentMethod,
		Status:          order.StatusPendingPayment,
	}

	resp := &order.CreateOrderResponse{PaymentMethod: paymentMethod}

	if paymentMethod == "stripe" {
		// Amount in cents (Stripe uses smallest currency unit)
		amountCents := int64(price * 100)
		metadata := map[string]string{
			"report_id":   fmt.Sprintf("%d", report.ID),
			"report_slug": report.Slug,
		}
		clientSecret, piID, err := s.stripeClient.CreatePaymentIntent(amountCents, "USD", description, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to create Stripe PaymentIntent: %w", err)
		}
		o.StripePaymentIntentID = piID
		resp.StripeClientSecret = clientSecret
	} else {
		amountStr := fmt.Sprintf("%.2f", price)
		paypalOrder, err := s.paypalClient.CreateOrder(amountStr, "USD", description, fmt.Sprintf("report-%d", report.ID))
		if err != nil {
			return nil, fmt.Errorf("failed to create PayPal order: %w", err)
		}
		paypalOrderID := paypalOrder.ID
		o.PaypalOrderID = &paypalOrderID
		resp.PaypalOrderID = paypalOrder.ID
	}

	if err := s.repo.Create(o); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	resp.OrderID = o.ID
	return resp, nil
}

func (s *orderService) CaptureOrder(orderID uint) (*order.CaptureOrderResponse, error) {
	o, err := s.repo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found")
	}

	if o.Status != order.StatusPendingPayment {
		return nil, fmt.Errorf("order is not in pending_payment status (current: %s)", o.Status)
	}

	if o.PaypalOrderID == nil || *o.PaypalOrderID == "" {
		return nil, fmt.Errorf("order has no PayPal order ID")
	}

	// Capture the PayPal payment
	captureID, captureStatus, err := s.paypalClient.CaptureOrder(*o.PaypalOrderID)
	if err != nil {
		return nil, fmt.Errorf("PayPal capture failed: %w", err)
	}

	// Update DB status
	if err := s.repo.UpdateStatus(o.ID, order.StatusPaymentReceived, captureID, "", nil); err != nil {
		logger.Error("Failed to update order status after capture", "order_id", o.ID, "error", err)
		return nil, fmt.Errorf("failed to update order status")
	}

	// Reload order for email
	o.PaypalCaptureID = captureID
	o.Status = order.StatusPaymentReceived

	// Fire-and-forget email notifications
	go func() {
		if err := s.emailSvc.SendOrderConfirmation(o); err != nil {
			logger.Error("Failed to send order confirmation email", "order_id", o.ID, "error", err)
		}
	}()
	go func() {
		if err := s.emailSvc.SendOrderAdminNotification(o); err != nil {
			logger.Error("Failed to send order admin notification email", "order_id", o.ID, "error", err)
		}
	}()

	logger.Info("Order captured", "order_id", o.ID, "capture_status", captureStatus)

	return &order.CaptureOrderResponse{
		OrderID:         o.ID,
		Status:          order.StatusPaymentReceived,
		PaypalCaptureID: captureID,
	}, nil
}

func (s *orderService) CaptureStripeOrder(orderID uint) (*order.ConfirmStripeOrderResponse, error) {
	o, err := s.repo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found")
	}

	if o.Status != order.StatusPendingPayment {
		return nil, fmt.Errorf("order is not in pending_payment status (current: %s)", o.Status)
	}

	if o.PaymentMethod != "stripe" {
		return nil, fmt.Errorf("order was not created with Stripe payment method")
	}

	if o.StripePaymentIntentID == "" {
		return nil, fmt.Errorf("order has no Stripe PaymentIntent ID")
	}

	// Verify with Stripe that the PaymentIntent actually succeeded
	piStatus, err := s.stripeClient.RetrievePaymentIntent(o.StripePaymentIntentID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify Stripe payment: %w", err)
	}

	if piStatus != "succeeded" {
		return nil, fmt.Errorf("Stripe payment not completed (status: %s)", piStatus)
	}

	// Update DB
	if err := s.repo.UpdateStripeCapture(o.ID, o.StripePaymentIntentID); err != nil {
		logger.Error("Failed to update order status after Stripe capture", "order_id", o.ID, "error", err)
		return nil, fmt.Errorf("failed to update order status")
	}

	o.Status = order.StatusPaymentReceived

	// Fire-and-forget email notifications
	go func() {
		if err := s.emailSvc.SendOrderConfirmation(o); err != nil {
			logger.Error("Failed to send order confirmation email", "order_id", o.ID, "error", err)
		}
	}()
	go func() {
		if err := s.emailSvc.SendOrderAdminNotification(o); err != nil {
			logger.Error("Failed to send order admin notification email", "order_id", o.ID, "error", err)
		}
	}()

	logger.Info("Stripe order captured", "order_id", o.ID, "pi_id", o.StripePaymentIntentID)

	return &order.ConfirmStripeOrderResponse{
		OrderID:               o.ID,
		Status:                order.StatusPaymentReceived,
		StripePaymentIntentID: o.StripePaymentIntentID,
	}, nil
}

func (s *orderService) HandleStripeWebhook(payload []byte, sigHeader string) error {
	// Verify the webhook came from Stripe
	if err := s.stripeClient.VerifyWebhookSignature(payload, sigHeader); err != nil {
		return fmt.Errorf("webhook signature invalid: %w", err)
	}

	event, err := stripe.ParseWebhookEvent(payload)
	if err != nil {
		return err
	}

	switch event.Type {
	case "payment_intent.succeeded":
		pi, err := stripe.PaymentIntentFromEvent(event)
		if err != nil {
			return err
		}

		o, err := s.repo.GetByStripePaymentIntentID(pi.ID)
		if err != nil {
			// Order may not exist (payment from another system) — not an error
			logger.Warn("Stripe webhook: no order found for PaymentIntent", "pi_id", pi.ID)
			return nil
		}

		// Idempotency: skip if already captured
		if o.Status != order.StatusPendingPayment {
			logger.Info("Stripe webhook: order already processed", "order_id", o.ID, "status", o.Status)
			return nil
		}

		if err := s.repo.UpdateStripeCapture(o.ID, pi.ID); err != nil {
			return fmt.Errorf("webhook: failed to update order status: %w", err)
		}

		o.Status = order.StatusPaymentReceived
		go func() {
			if err := s.emailSvc.SendOrderConfirmation(o); err != nil {
				logger.Error("Failed to send order confirmation email", "order_id", o.ID, "error", err)
			}
		}()
		go func() {
			if err := s.emailSvc.SendOrderAdminNotification(o); err != nil {
				logger.Error("Failed to send order admin notification email", "order_id", o.ID, "error", err)
			}
		}()

		logger.Info("Stripe webhook: order captured", "order_id", o.ID, "pi_id", pi.ID)

	default:
		// Acknowledge unhandled event types without error
		logger.Info("Stripe webhook: unhandled event type", "type", event.Type)
	}

	return nil
}

func (s *orderService) GetAll(query order.GetOrdersQuery) ([]order.Order, int64, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 || query.Limit > 100 {
		query.Limit = 20
	}
	return s.repo.GetAll(query)
}

func (s *orderService) GetByID(id uint) (*order.Order, error) {
	return s.repo.GetByID(id)
}

func (s *orderService) GetStats() (*order.OrderStats, error) {
	return s.repo.GetStats()
}

func (s *orderService) UpdateStatus(id uint, req *order.UpdateStatusRequest, updatedBy *uint) error {
	// Validate status transition
	validStatuses := map[order.OrderStatus]bool{
		order.StatusPendingPayment:  true,
		order.StatusPaymentReceived: true,
		order.StatusProcessing:      true,
		order.StatusDelivered:       true,
		order.StatusCancelled:       true,
		order.StatusRefunded:        true,
	}

	if !validStatuses[req.Status] {
		return fmt.Errorf("invalid status: %s", req.Status)
	}

	return s.repo.UpdateStatus(id, req.Status, "", req.AdminNotes, updatedBy)
}
