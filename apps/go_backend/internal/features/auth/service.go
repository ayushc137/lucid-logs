// Package auth provides local libSQL authentication and JWT issuance.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/argon2"

	"github.com/lucid-logs/go-backend/internal/config"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
)

type Service interface {
	Login(context.Context, *LoginRequest) (*AuthResponse, error)
	Register(context.Context, *RegisterRequest) (*AuthResponse, error)
	ValidateToken(string) (*Claims, error)
}

type service struct {
	db     *database.DB
	cfg    *config.Config
	logger zerolog.Logger
}

type tokenClaims struct {
	jwt.RegisteredClaims
	ID string `json:"ID"`
}

type userDB struct {
	ID           string
	Email        string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewService(db *database.DB, cfg *config.Config) Service {
	return &service{db: db, cfg: cfg, logger: log.With().Str("feature", "auth").Logger()}
}

func (s *service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Username))
	var user userDB
	var isAdmin int
	var createdAt, updatedAt string
	err := s.db.SQL().QueryRowContext(ctx, `SELECT id,email,pass,is_admin,created_at,updated_at FROM users WHERE email = ?`, email).
		Scan(&user.ID, &user.Email, &user.PasswordHash, &isAdmin, &createdAt, &updatedAt)
	if err == sql.ErrNoRows || (err == nil && !VerifyPassword(req.Password, user.PasswordHash)) {
		return nil, errors.ErrInvalidCredentials
	}
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	user.IsAdmin = isAdmin != 0
	user.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	user.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return s.authResponse(&user)
}

func (s *service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Username))
	var exists int
	if err := s.db.SQL().QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)`, email).Scan(&exists); err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	if exists != 0 {
		return nil, errors.ErrUserExists
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, errors.ErrInternal.Wrap(err)
	}
	now := time.Now().UTC()
	user := &userDB{ID: "users:" + uuid.NewString(), Email: email, PasswordHash: hash, CreatedAt: now, UpdatedAt: now}
	_, err = s.db.SQL().ExecContext(ctx, `INSERT INTO users(id,email,pass,is_admin,preferences,created_at,updated_at) VALUES(?,?,?,0,'{}',?,?)`,
		user.ID, user.Email, user.PasswordHash, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errors.ErrUserExists
		}
		return nil, errors.ErrDatabase.Wrap(err)
	}
	return s.authResponse(user)
}

func (s *service) authResponse(user *userDB) (*AuthResponse, error) {
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, errors.ErrInternal.Wrap(err)
	}
	return &AuthResponse{Token: token, User: user.ID, IsAdmin: user.IsAdmin}, nil
}

// HashPassword creates an Argon2id encoded password hash.
func HashPassword(password string) (string, error) {
	const memory, iterations, parallelism, saltLength, keyLength = 64 * 1024, 3, 2, 16, 32
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, iterations, parallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

// VerifyPassword compares a password with an Argon2id encoded hash.
func VerifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func (s *service) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.ErrUnauthorized.WithMessage("Token is empty")
	}
	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.ErrUnauthorized.WithMessage("Invalid token")
	}
	userID := claims.ID
	if userID == "" {
		userID = claims.Subject
	}
	if userID == "" {
		return nil, errors.ErrUnauthorized.WithMessage("Missing user ID in token")
	}
	result := &Claims{ID: userID}
	if claims.IssuedAt != nil {
		result.IssuedAt = claims.IssuedAt.Time.Unix()
	}
	if claims.NotBefore != nil {
		result.NotBefore = claims.NotBefore.Time.Unix()
	}
	if claims.ExpiresAt != nil {
		result.ExpiresAt = claims.ExpiresAt.Time.Unix()
	}
	return result, nil
}

func (s *service) generateToken(userID string) (string, error) {
	now := time.Now().UTC()
	hours := s.cfg.JWT.ExpirationHours
	if hours <= 0 {
		hours = 24
	}
	claims := tokenClaims{ID: userID, RegisteredClaims: jwt.RegisteredClaims{
		Subject: userID, Issuer: s.cfg.JWT.Issuer, IssuedAt: jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(hours) * time.Hour)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(s.cfg.JWT.Secret))
}
