// Package goallogs provides goal logs repository for database operations.
package goallogs

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	models "github.com/lucid-logs/go-backend/internal/shared/recordid"

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
	CreatedBy         models.RecordID      `json:"created_by"`
	CreatedAt         database.SurrealTime `json:"created_at"`

	// Value tracking
	ValueContributed *float64 `json:"value_contributed,omitempty"`
	ValueUnit        string   `json:"value_unit,omitempty"`
	ProgressBefore   *float64 `json:"progress_before,omitempty"`
	ProgressAfter    *float64 `json:"progress_after,omitempty"`

	// Task details (populated on read via subquery)
	TriggeringTask *triggeringTaskDB `json:"triggering_task,omitempty"`
}

type triggeringTaskDB struct {
	ID        string                `json:"id"`
	Title     string                `json:"title"`
	StartDate *database.SurrealTime `json:"start_date,omitempty"`
	EndDate   *database.SurrealTime `json:"end_date,omitempty"`
	Completed bool                  `json:"completed"`
	EmotionID *string               `json:"emotion_id,omitempty"`
	Category  *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"category,omitempty"`
}

func (l *goalLogDB) toGoalLog() *GoalLog {
	log := &GoalLog{
		ID:                database.ToStringID(l.ID),
		GoalID:            database.ToStringID(l.GoalID),
		Event:             l.Event,
		Changes:           l.Changes,
		TriggeredByTaskID: l.TriggeredByTaskID,
		SnapshotID:        database.ToStringID(l.SnapshotID),
		CreatedAt:         l.CreatedAt.Time,
		ValueContributed:  l.ValueContributed,
		ValueUnit:         l.ValueUnit,
		ProgressBefore:    l.ProgressBefore,
		ProgressAfter:     l.ProgressAfter,
	}

	if l.TriggeringTask != nil {
		taskInfo := &TriggeringTaskInfo{
			ID:        l.TriggeringTask.ID,
			Title:     l.TriggeringTask.Title,
			Completed: l.TriggeringTask.Completed,
			EmotionID: l.TriggeringTask.EmotionID,
		}
		if l.TriggeringTask.StartDate != nil {
			t := l.TriggeringTask.StartDate.Time
			taskInfo.StartDate = &t
		}
		if l.TriggeringTask.EndDate != nil {
			t := l.TriggeringTask.EndDate.Time
			taskInfo.EndDate = &t
		}
		if l.TriggeringTask.Category != nil {
			taskInfo.Category = l.TriggeringTask.Category
		}
		log.TriggeringTask = taskInfo
	}

	return log
}

type goalSnapshotDB struct {
	ID        models.RecordID      `json:"id,omitempty"`
	GoalID    models.RecordID      `json:"goal_id"`
	Status    string               `json:"status"`
	Stats     *goals.GoalStats     `json:"stats,omitempty"`
	Target    *goals.Target        `json:"target,omitempty"`
	CreatedBy models.RecordID      `json:"created_by"`
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
	now := time.Now()
	goalID := database.MustRecordID(goals.Table, req.GoalID)
	userRecordID := database.MustRecordID("users", userID)

	// First create a snapshot if stats are provided
	snapshotID := ""
	if req.Stats != nil {
		snapshot, err := database.QueryFirst[goalSnapshotDB](ctx, r.db, `
			CREATE goal_snapshots CONTENT {
				goal_id: $goal_id,
				status: $status,
				stats: $stats,
				created_by: $user,
				created_at: $now
			}
		`, map[string]any{
			"goal_id": goalID,
			"status":  goals.StatusActive, // Will be updated from actual goal
			"stats":   req.Stats,
			"user":    userRecordID,
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

	// Prepare changes - use NONE if nil
	changes := req.Changes
	if changes == nil {
		changes = map[string]any{}
	}

	// Build content data with optional fields
	contentData := map[string]any{
		"event_type":           req.Event,
		"changes":              changes,
		"triggered_by_task_id": req.TriggeredByTaskID,
		"created_by":           userRecordID,
		"created_at":           now,
		"value_contributed":    req.ValueContributed,
		"value_unit":           req.ValueUnit,
		"progress_before":      req.ProgressBefore,
		"progress_after":       req.ProgressAfter,
	}

	if snapshotID != "" {
		snapshotRecordID := database.MustRecordID(SnapshotsTable, snapshotID)

		logResult, err = database.QueryFirst[goalLogDB](ctx, r.db, `
			RELATE $goal_id->goal_logs->$snapshot_id CONTENT $content
		`, map[string]any{
			"goal_id":     goalID,
			"snapshot_id": snapshotRecordID,
			"content":     contentData,
		})
	} else {
		// For events without snapshots, we still need a RELATE but with a null out
		// We'll create a dummy snapshot or skip the out field

		logResult, err = database.QueryFirst[goalLogDB](ctx, r.db, `
			CREATE goal_logs SET
				`+"`in`"+` = $goal_id,
				`+"`out`"+` = NONE,
				event_type = $event,
				changes = $changes,
				triggered_by_task_id = $triggered_by,
				created_by = $user,
				created_at = $now,
				value_contributed = $value_contributed,
				value_unit = $value_unit,
				progress_before = $progress_before,
				progress_after = $progress_after
		`, map[string]any{
			"goal_id":           goalID,
			"event":             req.Event,
			"changes":           changes,
			"triggered_by":      req.TriggeredByTaskID,
			"user":              userRecordID,
			"now":               now,
			"value_contributed": req.ValueContributed,
			"value_unit":        req.ValueUnit,
			"progress_before":   req.ProgressBefore,
			"progress_after":    req.ProgressAfter,
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
	userRecordID := database.MustRecordID("users", userID)

	// Count total
	countResult, err := database.QueryFirst[struct {
		Count int64 `json:"count"`
	}](ctx, r.db, `
		SELECT count() as count FROM goal_logs
		WHERE `+"`in`"+` = $goal_id AND created_by = $user
	`, map[string]any{
		"goal_id": gID,
		"user":    userRecordID,
	})
	if err != nil {
		return nil, 0, errors.ErrDatabase.Wrap(err)
	}

	total := int64(0)
	if countResult != nil {
		total = countResult.Count
	}

	// Fetch logs with task details when triggered_by_task_id is present
	logsDB, err := database.QueryAll[goalLogDB](ctx, r.db, `
		SELECT *,
			IF triggered_by_task_id != NONE AND triggered_by_task_id != "" THEN
				(SELECT
					type::string(id) as id,
					title,
					start_date,
					end_date,
					completed,
					emotion_id,
					category
				FROM <record>triggered_by_task_id)[0]
			ELSE
				NONE
			END as triggering_task
		FROM goal_logs
		WHERE `+"`in`"+` = $goal_id AND created_by = $user
		ORDER BY created_at DESC
		LIMIT $limit START $offset
	`, map[string]any{
		"goal_id": gID,
		"user":    userRecordID,
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
		SELECT * FROM $id
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

// LogTaskEvent logs an event triggered by a task with additional task-related metadata.
func (a *GoalLoggerAdapter) LogTaskEvent(ctx context.Context, goalID, event, userID, triggeredByTaskID string, changes map[string]any, valueContributed *float64, valueUnit string) error {
	_, err := a.repo.LogEvent(ctx, &LogEventRequest{
		GoalID:            goalID,
		Event:             event,
		Changes:           changes,
		TriggeredByTaskID: triggeredByTaskID,
		ValueContributed:  valueContributed,
		ValueUnit:         valueUnit,
	}, userID)
	return err
}
