package repository

import (
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/user"
	"gorm.io/gorm"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(u *user.User) error
	GetByID(id uint) (*user.User, error)
	GetByEmail(email string) (*user.User, error)
	GetAll(page, limit int) ([]user.User, int64, error)
	GetByRole(role string, page, limit int) ([]user.User, int64, error)
	Update(u *user.User) error
	Delete(id uint) error
	UpdateLastLogin(id uint) error
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository instance
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create creates a new user in the database
func (r *userRepository) Create(u *user.User) error {
	return r.db.Create(u).Error
}

// GetByID retrieves a user by their ID
func (r *userRepository) GetByID(id uint) (*user.User, error) {
	var u user.User
	err := r.db.Where("id = ? AND is_active = ?", id, true).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByEmail retrieves a user by their email address
func (r *userRepository) GetByEmail(email string) (*user.User, error) {
	var u user.User
	err := r.db.Where("email = ? AND is_active = ?", email, true).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetAll retrieves all users with pagination
func (r *userRepository) GetAll(page, limit int) ([]user.User, int64, error) {
	var users []user.User
	var total int64

	offset := (page - 1) * limit

	// Count total active users
	if err := r.db.Model(&user.User{}).Where("is_active = ?", true).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch users with pagination
	err := r.db.Where("is_active = ?", true).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error

	return users, total, err
}

// GetByRole retrieves users by role with pagination
func (r *userRepository) GetByRole(role string, page, limit int) ([]user.User, int64, error) {
	var users []user.User
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&user.User{}).Where("is_active = ? AND role = ?", true, role)

	// Count total active users with the specified role
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch users with pagination
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error

	return users, total, err
}

// Update updates a user's information
func (r *userRepository) Update(u *user.User) error {
	return r.db.Save(u).Error
}

// Delete soft-deletes a user by setting is_active to false
func (r *userRepository) Delete(id uint) error {
	return r.db.Model(&user.User{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// UpdateLastLogin updates the last_login_at timestamp for a user
func (r *userRepository) UpdateLastLogin(id uint) error {
	now := time.Now()
	return r.db.Model(&user.User{}).
		Where("id = ?", id).
		Update("last_login_at", &now).Error
}
