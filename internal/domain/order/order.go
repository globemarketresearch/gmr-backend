package order

import (
	"time"

	"gorm.io/gorm"
)

// OrderStatus represents valid order statuses.
type OrderStatus string

const (
	StatusPendingPayment  OrderStatus = "pending_payment"
	StatusPaymentReceived OrderStatus = "payment_received"
	StatusProcessing      OrderStatus = "processing"
	StatusDelivered       OrderStatus = "delivered"
	StatusCancelled       OrderStatus = "cancelled"
	StatusRefunded        OrderStatus = "refunded"
)

// Order is the GORM model for the orders table.
type Order struct {
	ID                   uint           `gorm:"primaryKey;autoIncrement"                  json:"id"`
	CustomerName         string         `gorm:"size:255;not null"                         json:"customer_name"`
	CustomerEmail        string         `gorm:"size:255;not null"                         json:"customer_email"`
	CustomerCompany      string         `gorm:"size:255"                                  json:"customer_company,omitempty"`
	CustomerPhone        string         `gorm:"size:50"                                   json:"customer_phone,omitempty"`
	CustomerCountry      string         `gorm:"size:100"                                  json:"customer_country,omitempty"`
	ReportID             uint           `gorm:"not null"                                  json:"report_id"`
	ReportTitle          string         `gorm:"size:500;not null"                         json:"report_title"`
	ReportSlug           string         `gorm:"size:500;not null"                         json:"report_slug"`
	Amount               float64        `gorm:"type:decimal(10,2);not null"               json:"amount"`
	Currency             string         `gorm:"size:3;not null;default:USD"               json:"currency"`
	PaymentMethod        string         `gorm:"size:20;not null;default:paypal"           json:"payment_method"`
	PaypalOrderID        *string        `gorm:"size:100;uniqueIndex"                      json:"paypal_order_id,omitempty"`
	PaypalCaptureID      string         `gorm:"size:100"                                  json:"paypal_capture_id,omitempty"`
	StripePaymentIntentID string        `gorm:"size:255"                                  json:"stripe_payment_intent_id,omitempty"`
	Status               OrderStatus    `gorm:"size:30;not null;default:pending_payment"  json:"status"`
	FulfilledAt          *time.Time     `                                                 json:"fulfilled_at,omitempty"`
	FulfilledBy          *uint          `                                                 json:"fulfilled_by,omitempty"`
	AdminNotes           string         `gorm:"type:text"                                 json:"admin_notes,omitempty"`
	CreatedAt            time.Time      `                                                 json:"created_at"`
	UpdatedAt            time.Time      `                                                 json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index"                                     json:"-"`
}

// ============ DTOs ============

// CreateOrderRequest is the body sent by the client to POST /api/v1/orders.
type CreateOrderRequest struct {
	CustomerName    string `json:"customer_name"    validate:"required"`
	CustomerEmail   string `json:"customer_email"   validate:"required,email"`
	CustomerCompany string `json:"customer_company"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerCountry string `json:"customer_country"`
	ReportSlug      string `json:"report_slug"      validate:"required"`
	// PaymentMethod selects the payment gateway: "paypal" (default) or "stripe".
	PaymentMethod string `json:"payment_method"`
}

// CreateOrderResponse is returned after successful order creation.
// Exactly one of PaypalOrderID or StripeClientSecret will be set,
// depending on the requested payment_method.
type CreateOrderResponse struct {
	OrderID            uint   `json:"order_id"`
	PaymentMethod      string `json:"payment_method"`
	PaypalOrderID      string `json:"paypal_order_id,omitempty"`
	StripeClientSecret string `json:"stripe_client_secret,omitempty"`
}

// CaptureOrderResponse is returned after successful PayPal payment capture.
type CaptureOrderResponse struct {
	OrderID         uint        `json:"order_id"`
	Status          OrderStatus `json:"status"`
	PaypalCaptureID string      `json:"paypal_capture_id"`
}

// ConfirmStripeOrderResponse is returned after the Stripe payment is verified server-side.
type ConfirmStripeOrderResponse struct {
	OrderID               uint        `json:"order_id"`
	Status                OrderStatus `json:"status"`
	StripePaymentIntentID string      `json:"stripe_payment_intent_id"`
}

// UpdateStatusRequest is used by admin to PATCH order status.
type UpdateStatusRequest struct {
	Status     OrderStatus `json:"status"      validate:"required"`
	AdminNotes string      `json:"admin_notes"`
}

// GetOrdersQuery holds filter/pagination params for listing orders.
type GetOrdersQuery struct {
	Status    string `query:"status"`
	Search    string `query:"search"`
	DateFrom  string `query:"dateFrom"`
	DateTo    string `query:"dateTo"`
	Page      int    `query:"page"`
	Limit     int    `query:"limit"`
	SortBy    string `query:"sortBy"`
	SortOrder string `query:"sortOrder"`
}

// OrderStats holds aggregate counts for the admin dashboard.
type OrderStats struct {
	Total            int64            `json:"total"`
	TotalRevenue     float64          `json:"total_revenue"`
	ByStatus         map[string]int64 `json:"by_status"`
	RecentCount      int64            `json:"recent_count"`       // last 30 days
	RecentRevenue    float64          `json:"recent_revenue"`     // last 30 days
}
