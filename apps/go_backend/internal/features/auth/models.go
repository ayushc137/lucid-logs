// Package auth provides authentication functionality.
//
// This package handles:
//   - User login
//   - User registration
//   - JWT token validation
//
// Authentication Flow:
//  1. User sends credentials to /auth/login or /auth/register
//  2. Server validates credentials against the users table (argon2id)
//  3. Server returns a signed JWT token with user claims
//  4. Client includes token in Authorization header for protected routes
//  5. Auth middleware validates token and injects user into context
package auth

import "time"

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// LoginRequest is the login request payload.
//
// @Description Login request payload
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=1" example:"admin@example.com"`
	Password string `json:"password" validate:"required,min=6" example:"adminadmin"`
}

// RegisterRequest is the registration request payload.
//
// @Description Registration request payload
type RegisterRequest struct {
	Username string `json:"username" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required,min=6" example:"password123"`
}

// AuthResponse is the authentication response with JWT token.
//
// @Description Authentication response with JWT token
type AuthResponse struct {
	Token   string `json:"token" example:"eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9..."`
	User    string `json:"user" example:"user:abc123"`
	IsAdmin bool   `json:"is_admin" example:"false"`
}

// =============================================================================
// DOMAIN MODELS
// =============================================================================

// User represents a user in the system.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// =============================================================================
// JWT CLAIMS
// =============================================================================

// Claims represents the JWT claims issued by this service.
type Claims struct {
	ID        string `json:"ID"`  // User ID (record ID)
	IssuedAt  int64  `json:"iat"` // Issued at
	NotBefore int64  `json:"nbf"` // Not before
	ExpiresAt int64  `json:"exp"` // Expiration
}
