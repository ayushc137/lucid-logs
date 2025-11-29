// Package middleware provides HTTP middleware for the application.
//
// Middleware included:
//   - Auth: JWT authentication and user context injection
//   - Logger: Request/response logging
//   - Recovery: Panic recovery with logging
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/daily-journal/go-backend/internal/shared/errors"
	"github.com/daily-journal/go-backend/internal/shared/response"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// CONTEXT KEYS
// =============================================================================

// ContextKey is a type for context keys to avoid collisions.
type ContextKey string

const (
	// UserContextKey is the context key for the authenticated user.
	UserContextKey ContextKey = "authenticated_user"
)

// =============================================================================
// AUTHENTICATED USER
// =============================================================================

// AuthenticatedUser represents the authenticated user extracted from JWT.
type AuthenticatedUser struct {
	UserID    string // SurrealDB record ID (e.g., "user:abc123")
	Namespace string // SurrealDB namespace
	Database  string // SurrealDB database
}

// =============================================================================
// AUTH MIDDLEWARE
// =============================================================================

// AuthConfig holds configuration for the auth middleware.
type AuthConfig struct {
	JWTSecret string
	Namespace string
	Database  string
}

// Auth creates JWT authentication middleware.
//
// This middleware:
//   - Extracts JWT from Authorization header (Bearer token)
//   - Validates the token signature and expiration
//   - Extracts user ID from claims
//   - Injects AuthenticatedUser into request context
//
// Example usage:
//
//	r.Group(func(r chi.Router) {
//	    r.Use(middleware.Auth(authConfig))
//	    r.Get("/tasks", handler.ListTasks)
//	})
func Auth(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "Authorization header required")
				return
			}

			// Check Bearer prefix
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				response.Unauthorized(w, "Authorization header must start with 'Bearer '")
				return
			}

			// Extract token
			tokenString := strings.TrimSpace(authHeader[len(bearerPrefix):])
			if tokenString == "" {
				response.Unauthorized(w, "Token is empty")
				return
			}

			// Parse and validate token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
				// Validate algorithm (SurrealDB uses HS512)
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.JWTSecret), nil
			})

			if err != nil {
				log.Warn().Err(err).Msg("token validation failed")
				response.Unauthorized(w, "Invalid or expired token")
				return
			}

			if !token.Valid {
				response.Unauthorized(w, "Token is not valid")
				return
			}

			// Extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				response.Unauthorized(w, "Invalid token claims")
				return
			}

			// Extract user ID from claims
			// SurrealDB uses "ID" claim, but we also support "sub" for flexibility
			userID := ""
			if id, ok := claims["ID"].(string); ok && id != "" {
				userID = id
			} else if sub, ok := claims["sub"].(string); ok && sub != "" {
				userID = sub
			}

			if userID == "" {
				response.Unauthorized(w, "Invalid token: missing user ID")
				return
			}

			// Create authenticated user context
			authUser := &AuthenticatedUser{
				UserID:    userID,
				Namespace: cfg.Namespace,
				Database:  cfg.Database,
			}

			// Inject into request context
			ctx := context.WithValue(r.Context(), UserContextKey, authUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// =============================================================================
// CONTEXT HELPERS
// =============================================================================

// GetAuthenticatedUser extracts the authenticated user from context.
//
// Returns the user and a boolean indicating if found.
//
// Example:
//
//	user, ok := middleware.GetAuthenticatedUser(r.Context())
//	if !ok {
//	    response.Error(w, errors.ErrUnauthorized)
//	    return
//	}
func GetAuthenticatedUser(ctx context.Context) (*AuthenticatedUser, bool) {
	user, ok := ctx.Value(UserContextKey).(*AuthenticatedUser)
	return user, ok
}

// MustGetAuthenticatedUser extracts the authenticated user or returns an error.
//
// This is a convenience function for handlers where auth middleware is guaranteed.
//
// Example:
//
//	user, err := middleware.MustGetAuthenticatedUser(r.Context())
//	if err != nil {
//	    response.Error(w, err)
//	    return
//	}
func MustGetAuthenticatedUser(ctx context.Context) (*AuthenticatedUser, *errors.AppError) {
	user, ok := GetAuthenticatedUser(ctx)
	if !ok {
		return nil, errors.ErrUnauthorized.WithMessage("Authentication required")
	}
	return user, nil
}

// UserID is a convenience function to get just the user ID from context.
//
// Returns empty string if not authenticated.
func UserID(ctx context.Context) string {
	user, ok := GetAuthenticatedUser(ctx)
	if !ok {
		return ""
	}
	return user.UserID
}
