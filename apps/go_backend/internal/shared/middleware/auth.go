// Package middleware provides HTTP middleware for the application.
//
// Middleware included:
//   - Auth: JWT authentication and user context injection
//   - Logger: Request/response logging
//   - Recovery: Panic recovery with logging
package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/response"
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
	// TraceContextKey stores the request trace/correlation ID.
	TraceContextKey ContextKey = "trace_id"
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
//	r.Group("/api/v1", func(r *gin.RouterGroup) {
//	    r.Use(middleware.Auth(authConfig))
//	    r.GET("/tasks", handler.ListTasks)
//	})
func Auth(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "Authorization header required")
			c.Abort()
			return
		}

		// Check Bearer prefix
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			response.Unauthorized(c, "Authorization header must start with 'Bearer '")
			c.Abort()
			return
		}

		// Extract token
		tokenString := strings.TrimSpace(authHeader[len(bearerPrefix):])
		if tokenString == "" {
			response.Unauthorized(c, "Token is empty")
			c.Abort()
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
			response.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		if !token.Valid {
			response.Unauthorized(c, "Token is not valid")
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Unauthorized(c, "Invalid token claims")
			c.Abort()
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
			response.Unauthorized(c, "Invalid token: missing user ID")
			c.Abort()
			return
		}

		// Ensure user ID has the correct prefix
		if !strings.HasPrefix(userID, "users:") {
			userID = "users:" + strings.TrimPrefix(userID, "user:") // Handle potential legacy "user:" prefix too
		}

		// Create authenticated user context
		authUser := &AuthenticatedUser{
			UserID:    userID,
			Namespace: cfg.Namespace,
			Database:  cfg.Database,
		}

		// Inject into request context (both Gin context and standard context)
		c.Set(string(UserContextKey), authUser)

		// Also update the request context so it propagates to services that use context.Context
		ctx := context.WithValue(c.Request.Context(), UserContextKey, authUser)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
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
//	user, ok := middleware.GetAuthenticatedUser(c.Request.Context())
//	if !ok {
//	    response.Error(c, errors.ErrUnauthorized)
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
//	user, err := middleware.MustGetAuthenticatedUser(c.Request.Context())
//	if err != nil {
//	    response.Error(c, err)
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
