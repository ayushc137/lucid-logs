// Package goalentries provides goal entry data access using SurrealDB SDK.
package goalentries

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the goal entry data access interface.
type Repository interface {
	// FindByID retrieves an entry by ID.
	FindByID(ctx context.Context, id string) (*GoalEntry, error)

	// FindByGoalAndDate retrieves an entry for a specific goal and date.
	FindByGoalAndDate(ctx context.Context, goalID string, date time.Time) (*GoalEntry, error)

	// FindByGoalInRange retrieves entries for a goal within a date range.
	FindByGoalInRange(ctx context.Context, goalID string, startDate, endDate time.Time) ([]*GoalEntry, error)

	// Create creates a new entry.
	Create(ctx context.Context, goalID string, req *CreateRequest) (*GoalEntry, error)

	// Update updates an existing entry.
	Update(ctx context.Context, id string, req *UpdateRequest) (*GoalEntry, error)

	// AddTaskToEntry adds a task ID to an entry's task_ids array.
	AddTaskToEntry(ctx context.Context, id, taskID string) error

	// Delete removes an entry.
	Delete(ctx context.Context, id string) error
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new goal entry Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "goalentries").Logger(),
	}
}

// =============================================================================
// DATABASE MODEL
// =============================================================================

type entryDB struct {
	ID        models.RecordID      `json:"id,omitempty"`
	GoalID    models.RecordID      `json:"goal_id"`
	Date      database.SurrealTime `json:"date"`
	Value     *float64             `json:"value,omitempty"`
	Met       bool                 `json:"met"`
	TaskIDs   []string             `json:"task_ids"`
	Notes     string               `json:"notes,omitempty"`
	CreatedAt database.SurrealTime `json:"created_at"`
	UpdatedAt database.SurrealTime `json:"updated_at"`
}

func (e *entryDB) toEntry() *GoalEntry {
	return &GoalEntry{
		ID:        database.ToStringID(e.ID),
		GoalID:    database.ToStringID(e.GoalID),
		Date:      e.Date.Time,
		Value:     e.Value,
		Met:       e.Met,
		TaskIDs:   e.TaskIDs,
		Notes:     e.Notes,
		CreatedAt: e.CreatedAt.Time,
		UpdatedAt: e.UpdatedAt.Time,
	}
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindByID(ctx context.Context, id string) (*GoalEntry, error) {
	entryID := database.MustRecordID(Table, id)

	entry, err := database.QueryFirst[entryDB](ctx, r.db, `
		SELECT * FROM type::thing($id)
	`, map[string]any{
		"id": entryID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("entry_id", id).Msg("query failed for entry fetch")
		return nil, err
	}

	if entry == nil {
		return nil, errors.ErrNotFound
	}

	return entry.toEntry(), nil
}

func (r *repository) FindByGoalAndDate(ctx context.Context, goalID string, date time.Time) (*GoalEntry, error) {
	// Normalize date to start of day
	normalizedDate := date.Truncate(24 * time.Hour)

	entry, err := database.QueryFirst[entryDB](ctx, r.db, `
		SELECT * FROM goal_entries 
		WHERE goal_id = type::thing($goal_id)
		  AND time::floor(date, 1d) = time::floor($date, 1d)
	`, map[string]any{
		"goal_id": database.MustRecordID("goals", goalID),
		"date":    normalizedDate,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", goalID).Msg("query failed for entry by date")
		return nil, err
	}

	if entry == nil {
		return nil, errors.ErrNotFound
	}

	return entry.toEntry(), nil
}

func (r *repository) FindByGoalInRange(ctx context.Context, goalID string, startDate, endDate time.Time) ([]*GoalEntry, error) {
	entries, err := database.QueryAll[entryDB](ctx, r.db, `
		SELECT * FROM goal_entries 
		WHERE goal_id = type::thing($goal_id)
		  AND date >= $start_date
		  AND date <= $end_date
		ORDER BY date ASC
	`, map[string]any{
		"goal_id":    database.MustRecordID("goals", goalID),
		"start_date": startDate.Truncate(24 * time.Hour),
		"end_date":   endDate.Truncate(24 * time.Hour),
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", goalID).Msg("query failed for entries in range")
		return nil, err
	}

	result := make([]*GoalEntry, len(entries))
	for i := range entries {
		result[i] = entries[i].toEntry()
	}

	return result, nil
}

// =============================================================================
// CREATE OPERATION
// =============================================================================

func (r *repository) Create(ctx context.Context, goalID string, req *CreateRequest) (*GoalEntry, error) {
	entryID := generateRecordID()
	now := time.Now().UTC()

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, errors.ErrBadRequest.WithMessage("Invalid date format, use YYYY-MM-DD")
	}

	taskIDs := req.TaskIDs
	if taskIDs == nil {
		taskIDs = []string{}
	}

	createData := map[string]any{
		"goal_id":    database.MustRecordID("goals", goalID),
		"date":       date,
		"value":      req.Value,
		"met":        req.Met,
		"task_ids":   taskIDs,
		"notes":      req.Notes,
		"created_at": now,
		"updated_at": now,
	}

	_, err = database.QueryAll[entryDB](ctx, r.db, `
		CREATE type::thing($id) CONTENT $data
	`, map[string]any{
		"id":   entryID,
		"data": createData,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", goalID).Msg("create entry failed")
		return nil, err
	}

	r.logger.Info().Str("entry_id", database.ToStringID(entryID)).Str("goal_id", goalID).Msg("entry created")

	return r.FindByID(ctx, database.ToStringID(entryID))
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest) (*GoalEntry, error) {
	entryID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	updateData := map[string]any{
		"updated_at": now,
	}

	if req.Value != nil {
		updateData["value"] = *req.Value
	}
	if req.Met != nil {
		updateData["met"] = *req.Met
	}
	if req.Notes != nil {
		updateData["notes"] = *req.Notes
	}
	if req.TaskIDs != nil {
		updateData["task_ids"] = req.TaskIDs
	}

	_, err := database.QueryAll[entryDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE $data
	`, map[string]any{
		"id":   entryID,
		"data": updateData,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("entry_id", id).Msg("update entry failed")
		return nil, err
	}

	return r.FindByID(ctx, id)
}

func (r *repository) AddTaskToEntry(ctx context.Context, id, taskID string) error {
	entryID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	_, err := database.QueryAll[any](ctx, r.db, `
		UPDATE type::thing($id) SET 
			task_ids += $task_id,
			updated_at = $now
	`, map[string]any{
		"id":      entryID,
		"task_id": taskID,
		"now":     now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("entry_id", id).Msg("add task to entry failed")
		return err
	}

	return nil
}

// =============================================================================
// DELETE OPERATION
// =============================================================================

func (r *repository) Delete(ctx context.Context, id string) error {
	entryID := database.MustRecordID(Table, id)

	_, err := database.QueryAll[any](ctx, r.db, `
		DELETE type::thing($id)
	`, map[string]any{
		"id": entryID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("entry_id", id).Msg("delete entry failed")
		return err
	}

	r.logger.Info().Str("entry_id", id).Msg("entry deleted")
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

func generateRecordID() models.RecordID {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes) //nolint:errcheck // crypto/rand.Read never fails in practice
	return database.NewRecordID(Table, hex.EncodeToString(bytes))
}
