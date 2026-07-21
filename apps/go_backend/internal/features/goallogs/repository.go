// Package goallogs provides goal logs repository for database operations.
package goallogs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	models "github.com/lucid-logs/go-backend/internal/shared/recordid"
	"github.com/rs/zerolog"

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

// goalLogDB mirrors the goal_logs table columns. The value-tracking fields
// (value_contributed, value_unit, progress_before, progress_after) have no
// dedicated columns in the SQLite schema, so they are persisted inside the
// `changes` JSON payload and decoded back here on read.
type goalLogDB struct {
	ID                models.RecordID `json:"id,omitempty"`
	GoalID            string          `json:"goal_id"`
	SnapshotID        *string         `json:"snapshot_id,omitempty"`
	Event             string          `json:"event_type"`
	Changes           map[string]any  `json:"changes,omitempty"`
	TriggeredByTaskID string          `json:"triggered_by_task_id,omitempty"`
	CreatedBy         string          `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`

	// Value tracking (decoded from changes JSON on read)
	ValueContributed *float64 `json:"value_contributed,omitempty"`
	ValueUnit        string   `json:"value_unit,omitempty"`
	ProgressBefore   *float64 `json:"progress_before,omitempty"`
	ProgressAfter    *float64 `json:"progress_after,omitempty"`

	// Task details (populated via LEFT JOIN on read; not a stored column)
	TriggeringTask *triggeringTaskDB `json:"triggering_task,omitempty"`
}

type triggeringTaskDB struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Completed bool       `json:"completed"`
	EmotionID *string    `json:"emotion_id,omitempty"`
	Category  *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"category,omitempty"`
}

// extractValueTracking pulls the value-tracking fields the writer embedded in
// the changes payload back out into the dedicated struct fields.
func (l *goalLogDB) extractValueTracking() {
	if l.Changes == nil {
		return
	}
	if v, ok := l.Changes["__value_contributed"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			l.ValueContributed = &f
		}
	}
	if v, ok := l.Changes["__value_unit"]; ok && v != nil {
		if s, ok := v.(string); ok {
			l.ValueUnit = s
		}
	}
	if v, ok := l.Changes["__progress_before"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			l.ProgressBefore = &f
		}
	}
	if v, ok := l.Changes["__progress_after"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			l.ProgressAfter = &f
		}
	}
}

func (l *goalLogDB) toGoalLog() *GoalLog {
	l.extractValueTracking()

	// Strip the private tracking keys so they do not leak into API responses.
	if l.Changes != nil {
		filtered := make(map[string]any, len(l.Changes))
		for k, v := range l.Changes {
			if k == "__value_contributed" || k == "__value_unit" || k == "__progress_before" || k == "__progress_after" {
				continue
			}
			filtered[k] = v
		}
		l.Changes = filtered
	}

	log := &GoalLog{
		ID:                database.ToStringID(l.ID),
		GoalID:            database.RecordID(goals.Table, l.GoalID),
		Event:             l.Event,
		Changes:           l.Changes,
		TriggeredByTaskID: l.TriggeredByTaskID,
		CreatedAt:         l.CreatedAt,
		ValueContributed:  l.ValueContributed,
		ValueUnit:         l.ValueUnit,
		ProgressBefore:    l.ProgressBefore,
		ProgressAfter:     l.ProgressAfter,
	}

	if l.SnapshotID != nil && *l.SnapshotID != "" {
		log.SnapshotID = database.RecordID(SnapshotsTable, *l.SnapshotID)
	}

	if l.TriggeringTask != nil {
		taskInfo := &TriggeringTaskInfo{
			ID:        l.TriggeringTask.ID,
			Title:     l.TriggeringTask.Title,
			Completed: l.TriggeringTask.Completed,
			EmotionID: l.TriggeringTask.EmotionID,
			StartDate: l.TriggeringTask.StartDate,
			EndDate:   l.TriggeringTask.EndDate,
		}
		if l.TriggeringTask.Category != nil {
			taskInfo.Category = l.TriggeringTask.Category
		}
		log.TriggeringTask = taskInfo
	}

	return log
}

// goalSnapshotDB reflects the goal_snapshots table. The table stores the
// snapshot body (status + stats + target) in a single JSON `snapshot` column.
type goalSnapshotDB struct {
	ID        models.RecordID `json:"id,omitempty"`
	GoalID    string          `json:"goal_id"`
	CreatedBy string          `json:"created_by"`
	CreatedAt time.Time       `json:"created_at"`
	Snapshot  json.RawMessage `json:"snapshot,omitempty"`
}

// snapshotBody is the JSON payload persisted in the goal_snapshots.snapshot column.
type snapshotBody struct {
	Status string           `json:"status"`
	Stats  *goals.GoalStats `json:"stats,omitempty"`
	Target *goals.Target    `json:"target,omitempty"`
}

func (s *goalSnapshotDB) toGoalSnapshot() *GoalSnapshot {
	out := &GoalSnapshot{
		ID:        database.ToStringID(s.ID),
		GoalID:    database.RecordID(goals.Table, s.GoalID),
		CreatedAt: s.CreatedAt,
	}
	if len(s.Snapshot) > 0 {
		var body snapshotBody
		if err := json.Unmarshal(s.Snapshot, &body); err == nil {
			out.Status = body.Status
			out.Stats = body.Stats
			out.Target = body.Target
		}
	}
	if out.Status == "" {
		out.Status = goals.StatusActive
	}
	return out
}

// =============================================================================
// LOG EVENT
// =============================================================================

func (r *repository) LogEvent(ctx context.Context, req *LogEventRequest, userID string) (*GoalLog, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	goalID := database.RecordID(goals.Table, req.GoalID)
	userRecordID := database.RecordID("users", userID)

	// Prepare changes payload. Embed value-tracking fields with reserved keys so
	// they survive the single JSON column on the goal_logs table.
	changes := req.Changes
	if changes == nil {
		changes = map[string]any{}
	} else {
		// Copy so we don't mutate caller's map.
		dup := make(map[string]any, len(changes))
		for k, v := range changes {
			dup[k] = v
		}
		changes = dup
	}
	if req.ValueContributed != nil {
		changes["__value_contributed"] = *req.ValueContributed
	}
	if req.ValueUnit != "" {
		changes["__value_unit"] = req.ValueUnit
	}
	if req.ProgressBefore != nil {
		changes["__progress_before"] = *req.ProgressBefore
	}
	if req.ProgressAfter != nil {
		changes["__progress_after"] = *req.ProgressAfter
	}

	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return nil, errors.ErrInternal.Wrap(err)
	}

	// First create a snapshot if stats are provided.
	snapshotID := ""
	if req.Stats != nil {
		body := snapshotBody{
			Status: goals.StatusActive, // Will be updated from actual goal
			Stats:  req.Stats,
			Target: nil,
		}
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			r.logger.Warn().Err(err).Msg("Failed to marshal snapshot body")
		} else {
			newSnapshotID := database.NewRecordID(SnapshotsTable, generateSnapshotID()).String()
			_, err = database.QueryAll[any](ctx, r.db, `
				INSERT INTO goal_snapshots (id, goal_id, created_by, snapshot, created_at)
				VALUES ($id, $goal_id, $user, $snapshot, $now)
			`, map[string]any{
				"id":       newSnapshotID,
				"goal_id":  goalID,
				"user":     userRecordID,
				"snapshot": string(bodyJSON),
				"now":      now,
			})
			if err != nil {
				r.logger.Warn().Err(err).Msg("Failed to create snapshot")
			} else {
				snapshotID = newSnapshotID
			}
		}
	}

	logID := database.NewRecordID(LogsTable, generateLogID()).String()

	if snapshotID != "" {
		_, err = database.QueryAll[any](ctx, r.db, `
			INSERT INTO goal_logs (id, goal_id, snapshot_id, event_type, changes, triggered_by_task_id, created_by, created_at)
			VALUES ($id, $goal_id, $snapshot_id, $event, $changes, $triggered_by, $user, $now)
		`, map[string]any{
			"id":           logID,
			"goal_id":      goalID,
			"snapshot_id":  snapshotID,
			"event":        req.Event,
			"changes":      string(changesJSON),
			"triggered_by": nullableString(req.TriggeredByTaskID),
			"user":         userRecordID,
			"now":          now,
		})
	} else {
		_, err = database.QueryAll[any](ctx, r.db, `
			INSERT INTO goal_logs (id, goal_id, snapshot_id, event_type, changes, triggered_by_task_id, created_by, created_at)
			VALUES ($id, $goal_id, NULL, $event, $changes, $triggered_by, $user, $now)
		`, map[string]any{
			"id":           logID,
			"goal_id":      goalID,
			"event":        req.Event,
			"changes":      string(changesJSON),
			"triggered_by": nullableString(req.TriggeredByTaskID),
			"user":         userRecordID,
			"now":          now,
		})
	}
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}

	// Re-read the row so the returned GoalLog reflects committed state.
	logResult, err := database.QueryFirst[goalLogDB](ctx, r.db, `
		SELECT id, goal_id, snapshot_id, event_type, changes, triggered_by_task_id, created_by, created_at
		FROM goal_logs WHERE id = $id
	`, map[string]any{"id": logID})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	if logResult == nil {
		return nil, errors.ErrInternal.WithMessage("goal log disappeared after insert")
	}

	return logResult.toGoalLog(), nil
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindByGoal(ctx context.Context, goalID, userID string, params pagination.Params) ([]*GoalLog, int64, error) {
	gID := database.RecordID(goals.Table, goalID)
	userRecordID := database.RecordID("users", userID)

	// Count total.
	total, err := database.QueryScalar[int64](ctx, r.db, `
		SELECT COUNT(*) FROM goal_logs
		WHERE goal_id = $goal_id AND created_by = $user
	`, map[string]any{
		"goal_id": gID,
		"user":    userRecordID,
	})
	if err != nil {
		return nil, 0, errors.ErrDatabase.Wrap(err)
	}

	// Fetch logs with task details when triggered_by_task_id is present.
	// SQLite cannot return a struct via subquery, so we
	// build the triggering_task object via a json_object() aggregation from a
	// LEFT JOIN against the tasks table (plus its category).
	logsDB, err := database.QueryAll[goalLogDB](ctx, r.db, `
		SELECT
			gl.id            AS id,
			gl.goal_id       AS goal_id,
			gl.snapshot_id   AS snapshot_id,
			gl.event_type    AS event_type,
			gl.changes       AS changes,
			gl.triggered_by_task_id AS triggered_by_task_id,
			gl.created_by    AS created_by,
			gl.created_at    AS created_at,
			CASE
				WHEN gl.triggered_by_task_id IS NOT NULL AND gl.triggered_by_task_id != '' THEN
					json_object(
						'id',         t.id,
						'title',      t.title,
						'start_date', t.start_date,
						'end_date',   t.end_date,
						'completed',  t.completed,
						'emotion_id', t.emotion_id,
						'category',   CASE
							WHEN c.id IS NOT NULL THEN json_object('id', c.id, 'name', c.name, 'color', c.color)
							ELSE NULL
						END
					)
				ELSE NULL
			END AS triggering_task
		FROM goal_logs gl
		LEFT JOIN tasks t ON t.id = gl.triggered_by_task_id
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE gl.goal_id = $goal_id AND gl.created_by = $user
		ORDER BY gl.created_at DESC
		LIMIT $limit OFFSET $offset
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
	sID := database.RecordID(SnapshotsTable, snapshotID)

	snapshot, err := database.QueryFirst[goalSnapshotDB](ctx, r.db, `
		SELECT id, goal_id, created_by, snapshot, created_at
		FROM goal_snapshots WHERE id = $id
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
	gID := database.RecordID(goals.Table, goalID)
	userRecordID := database.RecordID("users", userID)
	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour).UTC().Format(time.RFC3339Nano)

	summary, err := database.QueryFirst[GoalLogsSummary](ctx, r.db, `
		SELECT
			COUNT(CASE WHEN event_type = 'target_met' THEN 1 END) AS days_met,
			COUNT(CASE WHEN event_type = 'target_exceeded' THEN 1 END) AS days_missed
		FROM goal_logs
		WHERE goal_id = $goal_id
		  AND created_by = $user
		  AND created_at >= $start_date
	`, map[string]any{
		"goal_id":    gID,
		"user":       userRecordID,
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

// =============================================================================
// HELPERS
// =============================================================================

// nullableString returns the string for non-empty input, or nil for empty input
// so SQLite stores NULL rather than an empty string.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// generateLogID returns a 32-character hex identifier for a goal_logs row.
func generateLogID() string {
	return generateHexID(16)
}

// generateSnapshotID returns a 32-character hex identifier for a goal_snapshots row.
func generateSnapshotID() string {
	return generateHexID(16)
}

// generateHexID returns n random bytes as a hex string (length 2*n).
func generateHexID(n int) string {
	bytes := make([]byte, n)
	_, _ = rand.Read(bytes) //nolint:errcheck // crypto/rand.Read never fails in practice
	return hex.EncodeToString(bytes)
}
