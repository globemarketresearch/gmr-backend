package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/user"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/internal/utils/auth"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrEmailTaken      = errors.New("email already in use")
	ErrInvalidUserData = errors.New("invalid user data")
)

// UserService defines the interface for user business logic
type UserService interface {
	Create(req *user.CreateUserRequest) (*user.User, error)
	GetByID(id uint) (*user.UserResponse, error)
	GetByEmail(email string) (*user.User, error)
	GetAll(page, limit int) ([]user.UserResponse, int64, error)
	GetByRole(role string, page, limit int) ([]user.UserResponse, int64, error)
	Update(id uint, req *user.UpdateUserRequest) error
	Delete(id uint) error
}

type userService struct {
	repo repository.UserRepository
}

// NewUserService creates a new user service instance
func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

// Create creates a new user
func (s *userService) Create(req *user.CreateUserRequest) (*user.User, error) {
	// Validate role
	if !user.IsValidRole(req.Role) {
		return nil, fmt.Errorf("%w: invalid role '%s'", ErrInvalidUserData, req.Role)
	}

	// Check if email is already taken
	existingUser, err := s.repo.GetByEmail(req.Email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailTaken
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check email availability: %w", err)
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	newUser := &user.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Name:         req.Name,
		Role:         req.Role,
		IsActive:     true,
	}

	if err := s.repo.Create(newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Invalidate user list cache
	cache.DeletePattern("users:list:*")
	cache.DeletePattern("users:total")

	return newUser, nil
}

// GetByID retrieves a user by ID with caching
func (s *userService) GetByID(id uint) (*user.UserResponse, error) {
	cacheKey := fmt.Sprintf("user:id:%d", id)

	var u user.User

	// Use cache-aside pattern
	err := cache.GetOrSet(cacheKey, &u, 30*time.Minute, func() (interface{}, error) {
		return s.repo.GetByID(id)
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return u.ToUserResponse(), nil
}

// GetByEmail retrieves a user by email (no caching for auth operations)
func (s *userService) GetByEmail(email string) (*user.User, error) {
	u, err := s.repo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return u, nil
}

// GetAll retrieves all users with pagination and caching
func (s *userService) GetAll(page, limit int) ([]user.UserResponse, int64, error) {
	cacheKey := fmt.Sprintf("users:list:%d:%d", page, limit)

	type result struct {
		Users []user.User `json:"users"`
		Total int64       `json:"total"`
	}

	var res result

	// Use cache-aside pattern
	err := cache.GetOrSet(cacheKey, &res, 10*time.Minute, func() (interface{}, error) {
		users, total, err := s.repo.GetAll(page, limit)
		if err != nil {
			return nil, err
		}
		return result{Users: users, Total: total}, nil
	})

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	// Convert to UserResponse
	userResponses := make([]user.UserResponse, len(res.Users))
	for i, u := range res.Users {
		userResponses[i] = *u.ToUserResponse()
	}

	return userResponses, res.Total, nil
}

// GetByRole retrieves users by role with pagination and caching
func (s *userService) GetByRole(role string, page, limit int) ([]user.UserResponse, int64, error) {
	cacheKey := fmt.Sprintf("users:role:%s:%d:%d", role, page, limit)

	type result struct {
		Users []user.User `json:"users"`
		Total int64       `json:"total"`
	}

	var res result

	// Use cache-aside pattern
	err := cache.GetOrSet(cacheKey, &res, 10*time.Minute, func() (interface{}, error) {
		users, total, err := s.repo.GetByRole(role, page, limit)
		if err != nil {
			return nil, err
		}
		return result{Users: users, Total: total}, nil
	})

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users by role: %w", err)
	}

	// Convert to UserResponse
	userResponses := make([]user.UserResponse, len(res.Users))
	for i, u := range res.Users {
		userResponses[i] = *u.ToUserResponse()
	}

	return userResponses, res.Total, nil
}

// Update updates a user's information
func (s *userService) Update(id uint, req *user.UpdateUserRequest) error {
	// Get existing user
	u, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Update fields if provided
	if req.Email != nil && *req.Email != u.Email {
		// Check if new email is already taken
		existingUser, err := s.repo.GetByEmail(*req.Email)
		if err == nil && existingUser != nil && existingUser.ID != id {
			return ErrEmailTaken
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to check email availability: %w", err)
		}
		u.Email = *req.Email
	}

	if req.Name != nil {
		u.Name = *req.Name
	}

	if req.Role != nil {
		if !user.IsValidRole(*req.Role) {
			return fmt.Errorf("%w: invalid role '%s'", ErrInvalidUserData, *req.Role)
		}
		u.Role = *req.Role
	}

	if req.IsActive != nil {
		u.IsActive = *req.IsActive
	}

	if req.Password != nil {
		hashedPassword, err := auth.HashPassword(*req.Password)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		u.PasswordHash = hashedPassword
	}

	// Update in database
	if err := s.repo.Update(u); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Invalidate caches
	cache.Delete(fmt.Sprintf("user:id:%d", id))
	cache.DeletePattern("users:list:*")
	cache.DeletePattern("users:total")

	return nil
}

// Delete soft-deletes a user
func (s *userService) Delete(id uint) error {
	// Check if user exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Soft delete
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Invalidate caches
	cache.Delete(fmt.Sprintf("user:id:%d", id))
	cache.DeletePattern("users:list:*")
	cache.DeletePattern("users:total")

	return nil
}
