package repository

import (
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/order"
	"gorm.io/gorm"
)

type OrderRepository interface {
	Create(o *order.Order) error
	GetByID(id uint) (*order.Order, error)
	GetByPaypalOrderID(paypalOrderID string) (*order.Order, error)
	GetByStripePaymentIntentID(piID string) (*order.Order, error)
	GetAll(query order.GetOrdersQuery) ([]order.Order, int64, error)
	UpdateStatus(id uint, status order.OrderStatus, captureID, adminNotes string, fulfilledBy *uint) error
	UpdateStripeCapture(id uint, piID string) error
	GetStats() (*order.OrderStats, error)
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(o *order.Order) error {
	return r.db.Create(o).Error
}

func (r *orderRepository) GetByID(id uint) (*order.Order, error) {
	var o order.Order
	err := r.db.First(&o, id).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *orderRepository) GetByPaypalOrderID(paypalOrderID string) (*order.Order, error) {
	var o order.Order
	err := r.db.Where("paypal_order_id = ?", paypalOrderID).First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *orderRepository) GetByStripePaymentIntentID(piID string) (*order.Order, error) {
	var o order.Order
	err := r.db.Where("stripe_payment_intent_id = ?", piID).First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *orderRepository) GetAll(query order.GetOrdersQuery) ([]order.Order, int64, error) {
	var orders []order.Order
	var total int64

	offset := (query.Page - 1) * query.Limit

	dbQuery := r.db.Model(&order.Order{})

	if query.Status != "" {
		dbQuery = dbQuery.Where("status = ?", query.Status)
	}

	if query.DateFrom != "" {
		if t, err := time.Parse(time.RFC3339, query.DateFrom); err == nil {
			dbQuery = dbQuery.Where("created_at >= ?", t)
		}
	}

	if query.DateTo != "" {
		if t, err := time.Parse(time.RFC3339, query.DateTo); err == nil {
			dbQuery = dbQuery.Where("created_at <= ?", t)
		}
	}

	if query.Search != "" {
		like := "%" + query.Search + "%"
		dbQuery = dbQuery.Where(
			"customer_name ILIKE ? OR customer_email ILIKE ? OR customer_company ILIKE ? OR report_title ILIKE ?",
			like, like, like, like,
		)
	}

	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sorting
	sortBy := "created_at"
	if query.SortBy == "amount" {
		sortBy = "amount"
	} else if query.SortBy == "status" {
		sortBy = "status"
	}

	sortOrder := "DESC"
	if query.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	err := dbQuery.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Limit(query.Limit).
		Offset(offset).
		Find(&orders).Error

	return orders, total, err
}

func (r *orderRepository) UpdateStatus(id uint, status order.OrderStatus, captureID, adminNotes string, fulfilledBy *uint) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if captureID != "" {
		updates["paypal_capture_id"] = captureID
	}
	if adminNotes != "" {
		updates["admin_notes"] = adminNotes
	}
	if status == order.StatusDelivered {
		now := time.Now()
		updates["fulfilled_at"] = &now
		if fulfilledBy != nil {
			updates["fulfilled_by"] = fulfilledBy
		}
	}

	result := r.db.Model(&order.Order{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *orderRepository) UpdateStripeCapture(id uint, piID string) error {
	result := r.db.Model(&order.Order{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":                  order.StatusPaymentReceived,
		"stripe_payment_intent_id": piID,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *orderRepository) GetStats() (*order.OrderStats, error) {
	stats := &order.OrderStats{
		ByStatus: make(map[string]int64),
	}

	// Total count
	if err := r.db.Model(&order.Order{}).Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// Total revenue (payment_received, processing, delivered)
	type revenueResult struct {
		TotalRevenue float64
	}
	var rev revenueResult
	if err := r.db.Model(&order.Order{}).
		Where("status IN ?", []string{"payment_received", "processing", "delivered"}).
		Select("COALESCE(SUM(amount), 0) as total_revenue").
		Scan(&rev).Error; err != nil {
		return nil, err
	}
	stats.TotalRevenue = rev.TotalRevenue

	// By status
	type statusCount struct {
		Status string
		Count  int64
	}
	var counts []statusCount
	if err := r.db.Model(&order.Order{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&counts).Error; err != nil {
		return nil, err
	}
	for _, c := range counts {
		stats.ByStatus[c.Status] = c.Count
	}

	// Recent 30 days
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err := r.db.Model(&order.Order{}).
		Where("created_at >= ?", thirtyDaysAgo).
		Count(&stats.RecentCount).Error; err != nil {
		return nil, err
	}

	var recentRev revenueResult
	if err := r.db.Model(&order.Order{}).
		Where("created_at >= ? AND status IN ?", thirtyDaysAgo, []string{"payment_received", "processing", "delivered"}).
		Select("COALESCE(SUM(amount), 0) as total_revenue").
		Scan(&recentRev).Error; err != nil {
		return nil, err
	}
	stats.RecentRevenue = recentRev.TotalRevenue

	return stats, nil
}
