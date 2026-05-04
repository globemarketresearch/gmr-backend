package auth

import "github.com/golang-jwt/jwt/v5"

// TokenType constants
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// JWTClaims represents the custom JWT claims structure
type JWTClaims struct {
	ID        uint   `json:"id"`         // User ID (matches frontend expectation)
	Email     string `json:"email"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}
