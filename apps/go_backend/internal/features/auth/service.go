// Package auth provides authentication functionality using SurrealDB SDK.
//
// This package handles:
//   - User login with SurrealDB's built-in auth
//   - User registration
//   - JWT token validation
//
// SDK Methods Used:
//   - surrealdb.Query[T]() - Type-safe user queries
//   - database.QueryAll[T]() - Type-safe query wrapper
//
// See: https://surrealdb.com/docs/sdk/golang
package auth

import (
	"context"
	"strings"

	"github.com/daily-journal/go-backend/internal/config"
	"github.com/daily-journal/go-backend/internal/shared/database"
	"github.com/daily-journal/go-backend/internal/shared/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the authentication service interface.
//
// This interface enables:
//   - Dependency injection in handlers
//   - Easy mocking for tests
//   - Decoupling from implementation details
type Service interface {
	// Login authenticates a user and returns a JWT token.
	Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error)

	// Register creates a new user and returns a JWT token.
	Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error)

	// ValidateToken validates a JWT token and returns the claims.
	ValidateToken(token string) (*SurrealClaims, error)
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

// service is the production implementation of Service.
type service struct {
	db     *database.DB
	cfg    *config.Config
	logger zerolog.Logger
}

// NewService creates a new auth Service.
func NewService(db *database.DB, cfg *config.Config) Service {
	return &service{
		db:     db,
		cfg:    cfg,
		logger: log.With().Str("feature", "auth").Logger(),
	}
}

// =============================================================================
// LOGIN
// =============================================================================

// Login authenticates a user using SurrealDB SDK and returns a JWT token.
//
// This method uses the SDK's Query function with crypto::argon2::compare
// for password verification, then generates a JWT token on success.
//
// See: https://surrealdb.com/docs/sdk/golang/methods/query
func (s *service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	s.logger.Debug().Str("username", req.Username).Msg("login attempt")

	email := strings.ToLower(strings.TrimSpace(req.Username))

	// Use SDK's typed Query function to find and verify user
	// The query uses SurrealDB's crypto::argon2::compare for password verification
	users, err := database.QueryAll[User](ctx, s.db, `
		SELECT id, email FROM user 
		WHERE email = $email AND crypto::argon2::compare(pass, $password)
	`, map[string]any{
		"email":    email,
		"password": req.Password,
	})
	if err != nil {
		s.logger.Error().Err(err).Str("username", req.Username).Msg("login query failed")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	if len(users) == 0 {
		s.logger.Warn().Str("username", req.Username).Msg("invalid credentials")
		return nil, errors.ErrInvalidCredentials
	}

	user := users[0]

	// Generate JWT token
	token, err := s.generateToken(user.ID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to generate token")
		return nil, errors.ErrInternal.Wrap(err)
	}

	s.logger.Info().Str("user_id", user.ID).Msg("login successful")

	return &AuthResponse{
		Token: token,
		User:  user.ID,
	}, nil
}

// =============================================================================
// REGISTER
// =============================================================================

// Register creates a new user using SurrealDB SDK and returns a JWT token.
//
// This method uses the SDK's typed Query functions for:
//   - Checking if user already exists
//   - Creating user with argon2-hashed password
//
// See: https://surrealdb.com/docs/sdk/golang/methods/query
func (s *service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	s.logger.Debug().Str("username", req.Username).Msg("registration attempt")

	email := strings.ToLower(strings.TrimSpace(req.Username))

	// Check if user already exists using SDK's typed query
	existingUsers, err := database.QueryAll[User](ctx, s.db, `
		SELECT id FROM user WHERE email = $email LIMIT 1
	`, map[string]any{
		"email": email,
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to check existing user")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	if len(existingUsers) > 0 {
		return nil, errors.ErrUserExists
	}

	// Create user with argon2 hashed password using SDK's typed query
	// SurrealDB's crypto::argon2::generate automatically hashes the password
	users, err := database.QueryAll[User](ctx, s.db, `
		CREATE user CONTENT {
			email: $email,
			pass: crypto::argon2::generate($password),
			created_at: time::now(),
			updated_at: time::now()
		}
	`, map[string]any{
		"email":    email,
		"password": req.Password,
	})
	if err != nil {
		s.logger.Error().Err(err).Str("username", req.Username).Msg("registration failed")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	if len(users) == 0 {
		s.logger.Error().Msg("failed to create user: no result returned")
		return nil, errors.ErrInternal
	}

	user := users[0]

	// Generate JWT token
	token, err := s.generateToken(user.ID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to generate token")
		return nil, errors.ErrInternal.Wrap(err)
	}

	s.logger.Info().Str("user_id", user.ID).Msg("user registered successfully")

	return &AuthResponse{
		Token: token,
		User:  user.ID,
	}, nil
}

// =============================================================================
// TOKEN VALIDATION
// =============================================================================

// ValidateToken validates a JWT token and returns the claims.
func (s *service) ValidateToken(tokenString string) (*SurrealClaims, error) {
	if tokenString == "" {
		return nil, errors.ErrUnauthorized.WithMessage("Token is empty")
	}

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// Validate algorithm (SurrealDB uses HS512)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.cfg.JWT.Secret), nil
	})

	if err != nil {
		return nil, errors.ErrUnauthorized.WithMessage("Invalid token")
	}

	if !token.Valid {
		return nil, errors.ErrUnauthorized.WithMessage("Token is not valid")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.ErrUnauthorized.WithMessage("Invalid token claims")
	}

	// Build SurrealClaims
	surrealClaims := &SurrealClaims{}

	if id, ok := claims["ID"].(string); ok {
		surrealClaims.ID = id
	} else if sub, ok := claims["sub"].(string); ok {
		surrealClaims.ID = sub
	}

	if surrealClaims.ID == "" {
		return nil, errors.ErrUnauthorized.WithMessage("Missing user ID in token")
	}

	if ns, ok := claims["NS"].(string); ok {
		surrealClaims.Namespace = ns
	}
	if db, ok := claims["DB"].(string); ok {
		surrealClaims.Database = db
	}
	if ac, ok := claims["AC"].(string); ok {
		surrealClaims.Access = ac
	}
	if exp, ok := claims["exp"].(float64); ok {
		surrealClaims.ExpiresAt = int64(exp)
	}

	return surrealClaims, nil
}

// =============================================================================
// HELPERS
// =============================================================================

// generateToken generates a JWT token for a user.
func (s *service) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"ID":  userID,
		"sub": userID,
		"NS":  s.cfg.Database.Namespace,
		"DB":  s.cfg.Database.Database,
		"AC":  "account",
		"iss": s.cfg.JWT.Issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}
