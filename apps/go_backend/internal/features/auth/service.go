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
// RecordID Convention:
//   - userDB uses models.RecordID for ID field
//   - Conversion to string happens in toUser() at the service boundary
//   - This enables type-safe queries without SELECT type::string(id) casts
//
// See: https://surrealdb.com/docs/sdk/golang
package auth

import (
	"context"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go/pkg/models"

	"github.com/lucid-logs/go-backend/internal/config"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
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

type tokenClaims struct {
	jwt.RegisteredClaims
	ID string `json:"ID"`
	NS string `json:"NS"`
	DB string `json:"DB"`
	AC string `json:"AC"`
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
// DATABASE MODEL
// =============================================================================

// userDB is the internal database representation of a user.
//
// This struct uses models.RecordID for the ID field, allowing SurrealDB SDK
// to populate it directly without type::string casts in queries.
type userDB struct {
	ID        models.RecordID `json:"id,omitempty"`
	Email     string          `json:"email"`
	IsAdmin   bool            `json:"is_admin"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// toUser converts the database model to the domain model.
func (u *userDB) toUser() *User {
	return &User{
		ID:        database.ToStringID(u.ID),
		Email:     u.Email,
		IsAdmin:   u.IsAdmin,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// =============================================================================
// LOGIN
// =============================================================================

// Login authenticates a user using SurrealDB SDK and returns a JWT token.
//
// This method uses the SDK's Query function with crypto::argon2::compare
// for password verification, then generates a JWT token on success.
// No type::string(id) cast needed since userDB.ID is models.RecordID.
//
// See: https://surrealdb.com/docs/sdk/golang/methods/query
func (s *service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	s.logger.Debug().Str("username", req.Username).Msg("login attempt")

	email := strings.ToLower(strings.TrimSpace(req.Username))

	// Use SDK's typed Query function to find and verify user
	// The query uses SurrealDB's crypto::argon2::compare for password verification
	// models.RecordID handles ID deserialization automatically
	usersDB, err := database.QueryAll[userDB](ctx, s.db, `
		SELECT * FROM users 
		WHERE email = $email AND crypto::argon2::compare(pass, $password)
	`, map[string]any{
		"email":    email,
		"password": req.Password,
	})
	if err != nil {
		s.logger.Error().Err(err).Str("username", req.Username).Msg("login query failed")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	if len(usersDB) == 0 {
		s.logger.Warn().Str("username", req.Username).Msg("invalid credentials")
		return nil, errors.ErrInvalidCredentials
	}

	user := usersDB[0].toUser()

	// Generate JWT token
	token, err := s.generateToken(user.ID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to generate token")
		return nil, errors.ErrInternal.Wrap(err)
	}

	s.logger.Info().Str("user_id", user.ID).Msg("login successful")

	return &AuthResponse{
		Token:   token,
		User:    user.ID,
		IsAdmin: user.IsAdmin,
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
// No type::string(id) cast needed since userDB.ID is models.RecordID.
//
// See: https://surrealdb.com/docs/sdk/golang/methods/query
func (s *service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	s.logger.Debug().Str("username", req.Username).Msg("registration attempt")

	email := strings.ToLower(strings.TrimSpace(req.Username))

	// Check if user already exists using SDK's typed query
	// models.RecordID handles ID deserialization automatically
	existingUsers, err := database.QueryAll[userDB](ctx, s.db, `
		SELECT id FROM users WHERE email = $email LIMIT 1
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
	// No RETURN block needed - SDK deserializes RecordID directly
	usersDB, err := database.QueryAll[userDB](ctx, s.db, `
		CREATE users CONTENT {
			email: $email,
			pass: crypto::argon2::generate($password)
		}
	`, map[string]any{
		"email":    email,
		"password": req.Password,
	})
	if err != nil {
		s.logger.Error().Err(err).Str("username", req.Username).Msg("registration failed")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	if len(usersDB) == 0 {
		s.logger.Error().Msg("failed to create user: no result returned")
		return nil, errors.ErrInternal
	}

	user := usersDB[0].toUser()

	// Generate JWT token
	token, err := s.generateToken(user.ID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to generate token")
		return nil, errors.ErrInternal.Wrap(err)
	}

	s.logger.Info().Str("user_id", user.ID).Msg("user registered successfully")

	return &AuthResponse{
		Token:   token,
		User:    user.ID,
		IsAdmin: user.IsAdmin,
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

	claims := &tokenClaims{}
	// Parse and validate token (expiration, nbf, etc. handled by jwt library)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
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

	userID := claims.ID
	if userID == "" {
		userID = claims.Subject
	}
	if userID == "" {
		return nil, errors.ErrUnauthorized.WithMessage("Missing user ID in token")
	}

	surrealClaims := &SurrealClaims{
		ID:        userID,
		Namespace: claims.NS,
		Database:  claims.DB,
		Access:    claims.AC,
	}

	if claims.IssuedAt != nil {
		surrealClaims.IssuedAt = claims.IssuedAt.Time.Unix()
	}
	if claims.NotBefore != nil {
		surrealClaims.NotBefore = claims.NotBefore.Time.Unix()
	}
	if claims.ExpiresAt != nil {
		surrealClaims.ExpiresAt = claims.ExpiresAt.Time.Unix()
	}

	return surrealClaims, nil
}

// =============================================================================
// HELPERS
// =============================================================================

// generateToken generates a JWT token for a user.
func (s *service) generateToken(userID string) (string, error) {
	now := time.Now().UTC()
	expHours := s.cfg.JWT.ExpirationHours
	if expHours <= 0 {
		expHours = 24
	}
	expiration := now.Add(time.Duration(expHours) * time.Hour)

	claims := tokenClaims{
		ID: userID,
		NS: s.cfg.Database.Namespace,
		DB: s.cfg.Database.Database,
		AC: "account",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    s.cfg.JWT.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiration),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}
