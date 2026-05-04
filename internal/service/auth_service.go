package service

import (
	"errors"
	"fmt"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/healthcare-market-research/backend/internal/domain/user"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/internal/utils/auth"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenRevoked       = errors.New("token has been revoked")
)

// AuthService defines the interface for authentication business logic
type AuthService interface {
	Login(email, password string) (*user.LoginResponse, error)
	RefreshToken(refreshToken string) (*user.RefreshResponse, error)
	Logout(userID uint, refreshToken string) error
	ValidateAccessToken(token string) (*user.User, error)
}

type authService struct {
	userRepo repository.UserRepository
	cfg      *config.AuthConfig
}

// NewAuthService creates a new auth service instance
func NewAuthService(userRepo repository.UserRepository, cfg *config.AuthConfig) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// Login authenticates a user and returns tokens
func (s *authService) Login(email, password string) (*user.LoginResponse, error) {
	// Get user by email
	u, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check password
	if err := auth.CheckPassword(u.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, err := auth.GenerateAccessToken(u, s.cfg.JWTSecret, s.cfg.AccessTokenExpiry, s.cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := auth.GenerateRefreshToken(u, s.cfg.JWTSecret, s.cfg.RefreshTokenExpiry, s.cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Validate and extract refresh token claims to get JTI
	refreshClaims, err := auth.ValidateToken(refreshToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	// Store refresh token in Redis
	refreshTokenKey := fmt.Sprintf("refresh_token:%d:%s", u.ID, refreshClaims.RegisteredClaims.ID)
	if err := cache.Set(refreshTokenKey, refreshToken, s.cfg.RefreshTokenExpiry); err != nil {
		// Log error but don't fail (Redis might be unavailable)
		fmt.Printf("Warning: Failed to store refresh token in Redis: %v\n", err)
	}

	// Update last login timestamp
	if err := s.userRepo.UpdateLastLogin(u.ID); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: Failed to update last login: %v\n", err)
	}

	return &user.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    auth.GetTokenExpiry(s.cfg.AccessTokenExpiry),
		User:         u.ToUserResponse(),
	}, nil
}

// RefreshToken generates new tokens using a valid refresh token
func (s *authService) RefreshToken(refreshTokenStr string) (*user.RefreshResponse, error) {
	// Validate refresh token
	claims, err := auth.ValidateToken(refreshTokenStr, s.cfg.JWTSecret)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Verify it's a refresh token
	if claims.TokenType != auth.TokenTypeRefresh {
		return nil, ErrInvalidToken
	}

	// Check if token exists in Redis (not revoked)
	refreshTokenKey := fmt.Sprintf("refresh_token:%d:%s", claims.ID, claims.RegisteredClaims.ID)
	var storedToken string
	if err := cache.Get(refreshTokenKey, &storedToken); err != nil {
		// Token not found in Redis, consider it revoked
		return nil, ErrTokenRevoked
	}

	// Get user
	u, err := s.userRepo.GetByID(claims.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate new tokens
	newAccessToken, err := auth.GenerateAccessToken(u, s.cfg.JWTSecret, s.cfg.AccessTokenExpiry, s.cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := auth.GenerateRefreshToken(u, s.cfg.JWTSecret, s.cfg.RefreshTokenExpiry, s.cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Validate new refresh token to get JTI
	newRefreshClaims, err := auth.ValidateToken(newRefreshToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to validate new refresh token: %w", err)
	}

	// Delete old refresh token from Redis
	cache.Delete(refreshTokenKey)

	// Store new refresh token in Redis
	newRefreshTokenKey := fmt.Sprintf("refresh_token:%d:%s", u.ID, newRefreshClaims.RegisteredClaims.ID)
	if err := cache.Set(newRefreshTokenKey, newRefreshToken, s.cfg.RefreshTokenExpiry); err != nil {
		fmt.Printf("Warning: Failed to store new refresh token in Redis: %v\n", err)
	}

	return &user.RefreshResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    auth.GetTokenExpiry(s.cfg.AccessTokenExpiry),
		User:         u.ToUserResponse(),
	}, nil
}

// Logout revokes a refresh token
func (s *authService) Logout(userID uint, refreshTokenStr string) error {
	// Validate refresh token
	claims, err := auth.ValidateToken(refreshTokenStr, s.cfg.JWTSecret)
	if err != nil {
		return ErrInvalidToken
	}

	// Verify token belongs to the user
	if claims.ID != userID {
		return ErrInvalidToken
	}

	// Delete refresh token from Redis
	refreshTokenKey := fmt.Sprintf("refresh_token:%d:%s", userID, claims.RegisteredClaims.ID)
	if err := cache.Delete(refreshTokenKey); err != nil {
		fmt.Printf("Warning: Failed to delete refresh token from Redis: %v\n", err)
	}

	return nil
}

// ValidateAccessToken validates an access token and returns the user
func (s *authService) ValidateAccessToken(tokenStr string) (*user.User, error) {
	// Validate token
	claims, err := auth.ValidateToken(tokenStr, s.cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	// Verify it's an access token
	if claims.TokenType != auth.TokenTypeAccess {
		return nil, auth.ErrInvalidToken
	}

	// Get user
	u, err := s.userRepo.GetByID(claims.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return u, nil
}
