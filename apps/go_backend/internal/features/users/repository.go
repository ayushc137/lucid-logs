// Package users provides user management persistence on libSQL.
package users

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/features/auth"
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

func NewRepository(db *database.DB) Repository {
	return &repository{db: db, logger: log.With().Str("repository", "users").Logger()}
}

const userColumns = `id,email,is_admin,preferences,created_at,updated_at`

func (r *repository) FindByID(ctx context.Context, id string) (*User, error) {
	user, err := database.QueryFirst[userDB](ctx, r.db,
		`SELECT `+userColumns+` FROM users WHERE id=$id`,
		map[string]any{"id": database.RecordID("users", id)})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	if user == nil {
		return nil, errors.ErrNotFound.WithMessage("User not found")
	}
	return user.toUser(), nil
}

func (r *repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	user, err := database.QueryFirst[userDB](ctx, r.db,
		`SELECT `+userColumns+` FROM users WHERE email=$email LIMIT 1`,
		map[string]any{"email": strings.ToLower(strings.TrimSpace(email))})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	if user == nil {
		return nil, nil
	}
	return user.toUser(), nil
}

func (r *repository) CountAdmins(ctx context.Context) (int64, error) {
	count, err := database.QueryScalar[int64](ctx, r.db, `SELECT COUNT(*) FROM users WHERE is_admin=1`, nil)
	if err != nil {
		return 0, errors.ErrDatabase.Wrap(err)
	}
	return count, nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	result, err := r.db.SQL().ExecContext(ctx, `DELETE FROM users WHERE id=?`, database.RecordID("users", id))
	if err != nil {
		return errors.ErrDatabase.Wrap(err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

func (r *repository) UpdateEmail(ctx context.Context, id, email string) (*User, error) {
	result, err := r.db.SQL().ExecContext(ctx, `UPDATE users SET email=?,updated_at=? WHERE id=?`,
		strings.ToLower(strings.TrimSpace(email)), time.Now().UTC().Format(time.RFC3339Nano), database.RecordID("users", id))
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, errors.ErrNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *repository) UpdatePassword(ctx context.Context, id, password string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return errors.ErrInternal.Wrap(err)
	}
	result, err := r.db.SQL().ExecContext(ctx, `UPDATE users SET pass=?,updated_at=? WHERE id=?`,
		hash, time.Now().UTC().Format(time.RFC3339Nano), database.RecordID("users", id))
	if err != nil {
		return errors.ErrDatabase.Wrap(err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

func (r *repository) UpdatePreferences(ctx context.Context, id string, req *UpdatePreferencesRequest) (*User, error) {
	current, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	prefs := current.Preferences
	if req.DailyRetro != nil {
		prefs.DailyRetro = req.DailyRetro
	}
	if req.WeeklyRetroDay != nil {
		prefs.WeeklyRetroDay = *req.WeeklyRetroDay
	}
	if req.MonthlyRetroDay != nil {
		prefs.MonthlyRetroDay = *req.MonthlyRetroDay
	}
	if req.Timezone != nil {
		prefs.Timezone = *req.Timezone
	}
	if req.AI != nil {
		// Preserve existing API key if new key is empty (user didn't change it)
		if req.AI.APIKey == "" && prefs.AI != nil && prefs.AI.APIKey != "" {
			req.AI.APIKey = prefs.AI.APIKey
		}
		prefs.AI = req.AI
	}
	encoded, err := json.Marshal(prefs)
	if err != nil {
		return nil, errors.ErrInternal.Wrap(err)
	}
	_, err = r.db.SQL().ExecContext(ctx, `UPDATE users SET preferences=?,updated_at=? WHERE id=?`,
		string(encoded), time.Now().UTC().Format(time.RFC3339Nano), database.RecordID("users", id))
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	return r.FindByID(ctx, id)
}

func (r *repository) GetUsersForRetroGeneration(ctx context.Context, now time.Time) ([]*User, error) {
	rows, err := database.QueryAll[userDB](ctx, r.db,
		`SELECT `+userColumns+` FROM users WHERE json_extract(preferences,'$.daily_retro.enabled')=1 AND json_extract(preferences,'$.daily_retro.auto_generate')=1`, nil)
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	eligible := make([]*User, 0, len(rows))
	for i := range rows {
		user := rows[i].toUser()
		if user.Preferences.DailyRetro != nil && isRetroTime(user.Preferences.DailyRetro, now) {
			eligible = append(eligible, user)
		}
	}
	return eligible, nil
}

func isRetroTime(settings *DailyRetroSettings, serverNow time.Time) bool {
	if settings == nil || settings.Time == "" {
		return false
	}
	loc, err := time.LoadLocation(settings.Timezone)
	if err != nil {
		loc = time.UTC
	}
	return serverNow.In(loc).Format("15:04") == settings.Time
}
