// Package bootstrap provides initialization utilities for the application.
package bootstrap

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/config"
	"github.com/lucid-logs/go-backend/internal/features/auth"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
)

// EnsureDevAdmin makes sure the default admin user exists in development.
func EnsureDevAdmin(ctx context.Context, db *database.DB, cfg *config.Config) error {
	if cfg == nil || !cfg.IsDev() {
		return nil
	}

	username := strings.ToLower(strings.TrimSpace(cfg.Admin.Username))
	password := strings.TrimSpace(cfg.Admin.Password)
	if username == "" || password == "" {
		return nil
	}

	logger := log.With().Str("component", "bootstrap").Logger()

	// Check if admin user exists
	var exists int
	if err := db.SQL().QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)`, username).Scan(&exists); err != nil {
		logger.Warn().Err(err).Msg("failed to check for existing admin user")
		return errors.ErrDatabase.Wrap(err)
	}
	if exists != 0 {
		return nil
	}

	// Create admin user with Argon2id password hash
	hash, err := auth.HashPassword(password)
	if err != nil {
		return errors.ErrInternal.Wrap(err)
	}
	userID := "users:" + uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = db.SQL().ExecContext(ctx,
		`INSERT INTO users(id,email,pass,is_admin,preferences,created_at,updated_at) VALUES(?,?,?,1,'{}',?,?)`,
		userID, username, hash, now, now)
	if err != nil {
		logger.Error().Err(err).Str("admin_user", username).Msg("failed to create admin user")
		return errors.ErrDatabase.Wrap(err)
	}

	logger.Info().
		Str("admin_user", username).
		Msg("seeded development admin account")

	return nil
}
