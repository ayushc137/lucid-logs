// Package bootstrap provides initialization utilities for the application.
//
// RecordID Convention:
//   - userRecordDB uses models.RecordID for ID field
//   - This enables type-safe queries without SELECT type::string(id) casts
//
// See: https://surrealdb.com/docs/sdk/golang
package bootstrap

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go/pkg/models"

	"github.com/lucid-logs/go-backend/internal/config"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
)

// userRecordDB is the internal database representation for admin bootstrap.
//
// This struct uses models.RecordID for the ID field, allowing SurrealDB SDK
// to populate it directly without type::string casts in queries.
type userRecordDB struct {
	ID    models.RecordID `json:"id,omitempty"`
	Email string          `json:"email"`
}

// EnsureDevAdmin makes sure the default admin user exists in development.
//
// No type::string(id) cast needed since userRecordDB.ID is models.RecordID.
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

	// Ensure unique index on email to prevent duplicates
	_, err := database.QueryAll[any](ctx, db, `
		DEFINE INDEX IF NOT EXISTS idx_users_email ON TABLE users COLUMNS email UNIQUE;
	`, nil)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to define unique index on users.email")
		// Continue anyway, as it might exist or be a permission issue
	}

	// Check if admin user exists using SDK's typed query
	// models.RecordID handles ID deserialization automatically
	existing, err := database.QueryAll[userRecordDB](ctx, db, `
		SELECT id, email FROM users WHERE email = $email LIMIT 1
	`, map[string]any{
		"email": username,
	})
	if err != nil {
		logger.Warn().Err(err).Msg("failed to check for existing admin user")
		return errors.ErrDatabase.Wrap(err)
	}

	if len(existing) > 0 {
		return nil
	}

	// Create admin user - no RETURN block needed, SDK deserializes RecordID directly
	_, err = database.QueryAll[userRecordDB](ctx, db, `
		CREATE users CONTENT {
			email: $email,
			pass: crypto::argon2::generate($password),
			is_admin: true
		}
	`, map[string]any{
		"email":    username,
		"password": password,
	})
	if err != nil {
		logger.Error().Err(err).Str("admin_user", username).Msg("failed to create admin user")
		return errors.ErrDatabase.Wrap(err)
	}

	logger.Info().
		Str("admin_user", username).
		Msg("seeded development admin account")

	return nil
}
