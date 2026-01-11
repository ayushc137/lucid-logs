// Package users provides user management functionality using SurrealDB SDK.
//
// RecordID Convention:
//   - userDB uses models.RecordID for ID field
//   - Conversion to string happens in toUser() at the repository boundary
//   - This enables type-safe queries without SELECT type::string(id) casts
//
// See: https://surrealdb.com/docs/sdk/golang
package users

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
)

// Repository defines user data access operations.
type Repository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	CountAdmins(ctx context.Context) (int64, error)
	UpdateEmail(ctx context.Context, id, email string) (*User, error)
	UpdatePassword(ctx context.Context, id, password string) error
	UpdatePreferences(ctx context.Context, id string, req *UpdatePreferencesRequest) (*User, error)
	GetUsersForRetroGeneration(ctx context.Context, now time.Time) ([]*User, error)
	Delete(ctx context.Context, id string) error
}

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a user repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "users").Logger(),
	}
}

const userTable = "users"

// FindByID retrieves a user by ID using SDK's typed Select.
//
// No type::string(id) cast needed since userDB.ID is models.RecordID.
func (r *repository) FindByID(ctx context.Context, id string) (*User, error) {
	userID := database.RecordID(userTable, id) //nolint:staticcheck // deprecated but safe
	result, err := database.Select[userDB](ctx, r.db, userID)
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	if result == nil {
		return nil, errors.ErrNotFound.WithMessage("User not found")
	}
	return result.toUser(), nil
}

// FindByEmail retrieves a user by email using SDK's typed query.
//
// No type::string(id) cast needed since userDB.ID is models.RecordID.
func (r *repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := database.QueryFirst[userDB](ctx, r.db, `
		SELECT * FROM users
		WHERE email = $email
		LIMIT 1
	`, map[string]any{
		"email": email,
	})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	if user == nil {
		return nil, nil
	}
	return user.toUser(), nil
}

func (r *repository) CountAdmins(ctx context.Context) (int64, error) {
	count, err := database.QueryScalar[float64](ctx, r.db, `
		RETURN (SELECT count() FROM users WHERE is_admin = true)[0].count OR 0
	`, nil)
	if err != nil {
		return 0, errors.ErrDatabase.Wrap(err)
	}
	return int64(count), nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	userID := database.RecordID(userTable, id) //nolint:staticcheck // deprecated but safe
	if _, err := database.Delete[userDB](ctx, r.db, userID); err != nil {
		return errors.ErrDatabase.Wrap(err)
	}
	return nil
}

func (r *repository) UpdateEmail(ctx context.Context, id, email string) (*User, error) {
	userID := database.RecordID(userTable, id) //nolint:staticcheck // deprecated but safe
	updated, err := database.Merge[userDB](ctx, r.db, userID, map[string]any{
		"email": strings.ToLower(strings.TrimSpace(email)),
	})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	return updated.toUser(), nil
}

func (r *repository) UpdatePassword(ctx context.Context, id, password string) error {
	userID := database.MustRecordID(userTable, id)
	_, err := database.QueryAll[userDB](ctx, r.db, `
		UPDATE type::thing($id) SET pass = crypto::argon2::generate($password)
	`, map[string]any{
		"id":       userID,
		"password": password,
	})
	if err != nil {
		return errors.ErrDatabase.Wrap(err)
	}
	return nil
}

func (r *repository) UpdatePreferences(ctx context.Context, id string, req *UpdatePreferencesRequest) (*User, error) {
	userID := database.MustRecordID(userTable, id)
	now := time.Now().UTC()

	updateData := map[string]any{
		"updated_at": now,
	}

	prefs := map[string]any{}
	if req.DailyRetro != nil {
		prefs["daily_retro"] = req.DailyRetro
	}
	if req.WeeklyRetroDay != nil {
		prefs["weekly_retro_day"] = *req.WeeklyRetroDay
	}
	if req.MonthlyRetroDay != nil {
		prefs["monthly_retro_day"] = *req.MonthlyRetroDay
	}
	if req.Timezone != nil {
		prefs["timezone"] = *req.Timezone
	}

	if len(prefs) > 0 {
		updateData["preferences"] = prefs
	}

	_, err := database.QueryAll[userDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE {
			updated_at: $updated_at,
			preferences: $preferences
		}
	`, map[string]any{
		"id":          userID,
		"updated_at":  now,
		"preferences": prefs,
	})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}

	return r.FindByID(ctx, id)
}

// GetUsersForRetroGeneration returns users whose daily retro time matches now.
// It checks each user's configured time and timezone to see if it's time.
func (r *repository) GetUsersForRetroGeneration(ctx context.Context, now time.Time) ([]*User, error) {
	// Get all users with retro enabled
	results, err := database.QueryAll[userDB](ctx, r.db, `
		SELECT * FROM users
		WHERE preferences.daily_retro.enabled = true
		  AND preferences.daily_retro.auto_generate = true
	`, nil)
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}

	var eligible []*User
	for _, u := range results {
		user := u.toUser()
		if user.Preferences.DailyRetro == nil {
			continue
		}

		// Check if user's configured time matches now
		if isRetroTime(user.Preferences.DailyRetro, now) {
			eligible = append(eligible, user)
		}
	}

	return eligible, nil
}

// isRetroTime checks if the current time matches the user's configured retro time.
func isRetroTime(settings *DailyRetroSettings, serverNow time.Time) bool {
	if settings == nil || settings.Time == "" {
		return false
	}

	// Load user's timezone
	loc, err := time.LoadLocation(settings.Timezone)
	if err != nil {
		loc = time.UTC
	}

	// Convert server time to user's timezone
	userNow := serverNow.In(loc)
	userTimeStr := userNow.Format("15:04")

	// Check if times match (within the same minute)
	return userTimeStr == settings.Time
}
