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
	return ensureAdmin(ctx, db, cfg.Admin.Username, cfg.Admin.Password, "development")
}

// EnsureInitialAdmin seeds the first admin account in any environment when
// ADMIN_SEED=true and both ADMIN_USERNAME and ADMIN_PASSWORD are set.
//
// This is the first-run mechanism for self-hosted production deployments: the
// container starts with no users, so an explicit opt-in env flag creates the
// initial login. If the user already exists it is a no-op, and if the flag is
// unset it does nothing (safe to leave configured; only runs when requested).
func EnsureInitialAdmin(ctx context.Context, db *database.DB, cfg *config.Config) error {
	if cfg == nil || !cfg.Admin.Seed {
		return nil
	}
	return ensureAdmin(ctx, db, cfg.Admin.Username, cfg.Admin.Password, "initial")
}

// ensureAdmin creates an admin user if one with the given email does not exist.
func ensureAdmin(ctx context.Context, db *database.DB, rawUsername, rawPassword, label string) error {
	username := strings.ToLower(strings.TrimSpace(rawUsername))
	password := strings.TrimSpace(rawPassword)
	if username == "" || password == "" {
		log.Warn().Str("component", "bootstrap").
			Msgf("%s admin seed requested but ADMIN_USERNAME/ADMIN_PASSWORD are empty; skipping", label)
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
		logger.Info().Str("admin_user", username).Msgf("%s admin account already exists; skipping seed", label)
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
		logger.Error().Err(err).Str("admin_user", username).Msgf("failed to create %s admin user", label)
		return errors.ErrDatabase.Wrap(err)
	}

	logger.Info().
		Str("admin_user", username).
		Msgf("seeded %s admin account", label)

	return nil
}
