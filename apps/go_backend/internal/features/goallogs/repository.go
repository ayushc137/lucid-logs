// Package goallogs provides goal logs repository for database operations.
package goallogs

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/surrealdb/surrealdb.go/pkg/models"

	"github.com/lucid-logs/go-backend/internal/features/goals"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the goal logs data access interface.
type Repository interface {
	// LogEvent records a goal event in the history.
	LogEvent(ctx context.Context, req *LogEventRequest, userID string) (*GoalLog, error)

	// FindByGoal retrieves logs for a specific goal.
	FindByGoal(ctx context.Context, goalID, userID string, params pagination.Params) ([]*GoalLog, int64, error)

	// GetSnapshot retrieves a specific snapshot.
	GetSnapshot(ctx context.Context, snapshotID string) (*GoalSnapshot, error)

	// GetSummary retrieves aggregated history summary for a goal.
	GetSummary(ctx context.Context, goalID, userID string, days int) (*GoalLogsSummary, error)
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new goal logs Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: zerolog.New(zerolog.NewConsoleWriter()).With().Str("pkg", "goallogs").Logger(),
	}
}

// =============================================================================
// DATABASE MODELS
// =============================================================================

type goalLogDB struct {
	ID                models.RecordID      `json:"id,omitempty"`
	GoalID            models.RecordID      `json:"in"`         // RELATE in
	SnapshotID        models.RecordID      `json:"out"`        // RELATE out
	Event             string               `json:"event_type"` // Changed to match schema
	Changes           map[string]any       `json:"changes,omitempty"`
	TriggeredByTaskID string               `json:"triggered_by_task_id,omitempty"` // Changed to match schema
	CreatedAt         database.SurrealTime `json:"created_at"`
}

func (l *goalLogDB) toGoalLog() *GoalLog {
	return &GoalLog{
		ID:                database.ToStringID(l.ID),
		GoalID:            database.ToStringID(l.GoalID),
		Event:             l.Event,
		Changes:           l.Changes,
		TriggeredByTaskID: l.TriggeredByTaskID,
		SnapshotID:        database.ToStringID(l.SnapshotID),
		CreatedAt:         l.CreatedAt.Time,
	}
}

type goalSnapshotDB struct {
	ID        models.RecordID      `json:"id,omitempty"`
	GoalID    models.RecordID      `json:"goal_id"`
	Status    string               `json:"status"`
	Stats     *goals.GoalStats     `json:"stats,omitempty"`
	Target    *goals.Target        `json:"target,omitempty"`
	CreatedAt database.SurrealTime `json:"created_at"`
}

func (s *goalSnapshotDB) toGoalSnapshot() *GoalSnapshot {
	return &GoalSnapshot{
		ID:        database.ToStringID(s.ID),
		GoalID:    database.ToStringID(s.GoalID),
		Status:    s.Status,
		Stats:     s.Stats,
		Target:    s.Target,
		CreatedAt: s.CreatedAt.Time,
	}
}

// =============================================================================
// LOG EVENT
// =============================================================================

func (r *repository) LogEvent(ctx context.Context, req *LogEventRequest, userID string) (*GoalLog, error) {
	now := time.Now().UTC()
	goalID := database.MustRecordID(goals.Table, req.GoalID)

	// First create a snapshot if stats are provided
	snapshotID := ""
	if req.Stats != nil {
		snapshot, err := database.QueryFirst[goalSnapshotDB](ctx, r.db, `
			CREATE goal_snapshots CONTENT {
				goal_id: $goal_id,
				status: $status,
				stats: $stats,
				created_at: $now
			}
		`, map[string]any{
			"goal_id": goalID,
			"status":  goals.StatusActive, // Will be updated from actual goal
			"stats":   req.Stats,
			"now":     now,
		})
		if err != nil {
			r.logger.Warn().Err(err).Msg("Failed to create snapshot")
		} else if snapshot != nil {
			snapshotID = database.ToStringID(snapshot.ID)
		}
	}

	// Create the log entry using RELATE
	var logResult *goalLogDB
	var err error

	if snapshotID != "" {
		snapshotRecordID := database.MustRecordID(SnapshotsTable, snapshotID)

		// Prepare changes - use NONE if nil
		changes := req.Changes
		if changes == nil {
			changes = map[string]any{}
		}

		logResult, err = database.QueryFirst[goalLogDB](ctx, r.db, `
			RELATE $goal_id->goal_logs->$snapshot_id CONTENT {
				event_type: $event,
				changes: $changes,
				triggered_by_task_id: $triggered_by,
				created_at: $now
			}
		`, map[string]any{
			"goal_id":      goalID,
			"snapshot_id":  snapshotRecordID,
			"event":        req.Event,
			"changes":      changes,
			"triggered_by": req.TriggeredByTaskID,
			"now":          now,
		})
	} else {
		// For events without snapshots, we still need a RELATE but with a null out
		// We'll create a dummy snapshot or skip the out field

		// Prepare changes - use NONE if nil
		changes := req.Changes
		if changes == nil {
			changes = map[string]any{}
		}

		logResult, err = database.QueryFirst[goalLogDB](ctx, r.db, `
			CREATE goal_logs SET
				`+"`in`"+` = $goal_id,
				`+"`out`"+` = NONE,
				event_type = $event,
				changes = $changes,
				triggered_by_task_id = $triggered_by,
				created_at = $now
		`, map[string]any{
			"goal_id":      goalID,
			"event":        req.Event,
			"changes":      changes,
			"triggered_by": req.TriggeredByTaskID,
			"now":          now,
		})
	}
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}

	return logResult.toGoalLog(), nil
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindByGoal(ctx context.Context, goalID, userID string, params pagination.Params) ([]*GoalLog, int64, error) {
	gID := database.MustRecordID(goals.Table, goalID)

	// Count total
	countResult, err := database.QueryFirst[struct {
		Count int64 `json:"count"`
	}](ctx, r.db, `
		SELECT count() as count FROM goal_logs 
		WHERE `+"`in`"+` = $goal_id AND created_by = $user
	`, map[string]any{
		"goal_id": gID,
		"user":    userID,
	})
	if err != nil {
		return nil, 0, errors.ErrDatabase.Wrap(err)
	}

	total := int64(0)
	if countResult != nil {
		total = countResult.Count
	}

	// Fetch logs
	logsDB, err := database.QueryAll[goalLogDB](ctx, r.db, `
		SELECT * FROM goal_logs 
		WHERE `+"`in`"+` = $goal_id AND created_by = $user
		ORDER BY created_at DESC
		LIMIT $limit OFFSET $offset
	`, map[string]any{
		"goal_id": gID,
		"user":    userID,
		"limit":   params.Limit,
		"offset":  params.Offset,
	})
	if err != nil {
		return nil, 0, errors.ErrDatabase.Wrap(err)
	}

	logs := make([]*GoalLog, len(logsDB))
	for i, l := range logsDB {
		logs[i] = l.toGoalLog()
	}

	return logs, total, nil
}

func (r *repository) GetSnapshot(ctx context.Context, snapshotID string) (*GoalSnapshot, error) {
	sID := database.MustRecordID(SnapshotsTable, snapshotID)

	snapshot, err := database.QueryFirst[goalSnapshotDB](ctx, r.db, `
		SELECT * FROM type::thing($id)
	`, map[string]any{
		"id": sID,
	})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	if snapshot == nil {
		return nil, errors.ErrNotFound
	}

	return snapshot.toGoalSnapshot(), nil
}

func (r *repository) GetSummary(ctx context.Context, goalID, userID string, days int) (*GoalLogsSummary, error) {
	gID := database.MustRecordID(goals.Table, goalID)
	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	summary, err := database.QueryFirst[GoalLogsSummary](ctx, r.db, `
		LET $logs = SELECT * FROM goal_logs 
			WHERE `+"`in`"+` = $goal_id  
			AND created_by = $user
			AND created_at >= $start_date;
		
		LET $met = (SELECT count() FROM $logs WHERE event = "target_met").count ?? 0;
		LET $exceeded = (SELECT count() FROM $logs WHERE event = "target_exceeded").count ?? 0;
		
		RETURN {
			days_met: $met,
			days_missed: $exceeded
		};
	`, map[string]any{
		"goal_id":    gID,
		"user":       userID,
		"start_date": startDate,
	})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	if summary == nil {
		return &GoalLogsSummary{}, nil
	}

	return summary, nil
}

// =============================================================================
// GOAL LOGGER ADAPTER
// =============================================================================

// GoalLoggerAdapter implements goals.GoalLogger interface.
// This adapter allows the goals service to log events without circular imports.
type GoalLoggerAdapter struct {
	repo Repository
}

// NewGoalLoggerAdapter creates a new adapter that implements goals.GoalLogger.
func NewGoalLoggerAdapter(repo Repository) *GoalLoggerAdapter {
	return &GoalLoggerAdapter{repo: repo}
}

// LogEvent logs a goal event with optional stats snapshot.
func (a *GoalLoggerAdapter) LogEvent(ctx context.Context, goalID, event, userID string, changes map[string]any, stats *goals.GoalStats) error {
	_, err := a.repo.LogEvent(ctx, &LogEventRequest{
		GoalID:  goalID,
		Event:   event,
		Changes: changes,
		Stats:   stats,
	}, userID)
	return err
}
