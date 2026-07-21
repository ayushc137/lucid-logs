// Package taskgoals provides task-goal relationship management using SQLite/libSQL.
//
// This package implements:
//   - INSERT operations for creating task→goal edges in the task_goals join table
//   - Delete operations for unlinking
//   - Edge lookup by task+goal for duplicate prevention
//
// Database Architecture:
//
// The task_goals table is a many-to-many join table between tasks and goals:
//
//	INSERT INTO task_goals (id, task_id, goal_id, impact_type, ...) VALUES (...)
//
// The schema enforces UNIQUE(task_id, goal_id) for duplicate prevention and
// uses ON DELETE CASCADE to clean up edges when either endpoint is removed.
package taskgoals

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	models "github.com/lucid-logs/go-backend/internal/shared/recordid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the task-goal link data access interface.
type Repository interface {
	// Create creates a new task-goal link using RELATE.
	Create(ctx context.Context, taskID, goalID string, req *LinkRequest, userID string) (*TaskGoal, error)

	// CreateBatch creates multiple task-goal links in a single operation.
	// This is more efficient than calling Create multiple times.
	CreateBatch(ctx context.Context, taskID string, links []LinkRequest) ([]*TaskGoal, error)

	// FindByTaskAndGoal checks if a link exists between a task and goal.
	FindByTaskAndGoal(ctx context.Context, taskID, goalID string) (*TaskGoal, error)

	// Delete removes a task-goal link.
	Delete(ctx context.Context, taskID, goalID string) error

	// DeleteByEdgeID removes a link by its edge ID.
	DeleteByEdgeID(ctx context.Context, edgeID string) error

	// DeleteByTask removes all goal links for a task.
	DeleteByTask(ctx context.Context, taskID string) error
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new task-goal Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "taskgoals").Logger(),
	}
}

// =============================================================================
// DATABASE MODEL
// =============================================================================

// taskGoalDB is the internal database representation.
//
// JSON tags map directly to the task_goals table columns. The In/Out field
// names are preserved for compatibility with toTaskGoal(), but their JSON
// tags (task_id/goal_id) match the SQLite schema so the row decoder maps
// them correctly.
type taskGoalDB struct {
	ID             models.RecordID   `json:"id,omitempty"`
	In             models.RecordID   `json:"task_id"` // Task ID (schema column: task_id)
	Out            models.RecordID   `json:"goal_id"` // Goal ID (schema column: goal_id)
	ImpactType     string            `json:"impact_type"`
	QuantityValue  *float64          `json:"quantity_value,omitempty"`
	UnitID         *string           `json:"unit_id,omitempty"`
	Notes          string            `json:"notes,omitempty"`
	Source         string            `json:"source"`
	IsMilestone    bool              `json:"is_milestone"`
	MilestoneLabel string            `json:"milestone_label,omitempty"`
	MilestoneOrder int               `json:"milestone_order"`
	CreatedAt      database.FlexTime `json:"created_at"`
}

func (t *taskGoalDB) toTaskGoal() *TaskGoal {
	return &TaskGoal{
		ID:             database.ToStringID(t.ID),
		TaskID:         database.ToStringID(t.In),
		GoalID:         database.ToStringID(t.Out),
		ImpactType:     t.ImpactType,
		QuantityValue:  t.QuantityValue,
		UnitID:         t.UnitID,
		Notes:          t.Notes,
		Source:         t.Source,
		IsMilestone:    t.IsMilestone,
		MilestoneLabel: t.MilestoneLabel,
		MilestoneOrder: t.MilestoneOrder,
		CreatedAt:      t.CreatedAt.Time,
	}
}

// =============================================================================
// CREATE
// =============================================================================

func (r *repository) Create(ctx context.Context, taskID, goalID string, req *LinkRequest, userID string) (*TaskGoal, error) {
	// Normalize to table:value record IDs for storage.
	taskRef := database.RecordID("tasks", taskID)
	goalRef := database.RecordID("goals", goalID)
	edgeID := generateEdgeRecordID()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Set defaults
	source := SourceManual

	// Use database.Create to INSERT and SELECT the row back in one call.
	result, err := database.Create[taskGoalDB](ctx, r.db, Table, map[string]any{
		"id":              edgeID.String(),
		"task_id":         taskRef,
		"goal_id":         goalRef,
		"impact_type":     req.ImpactType,
		"quantity_value":  req.QuantityValue,
		"unit_id":         req.UnitID,
		"notes":           req.Notes,
		"source":          source,
		"is_milestone":    req.IsMilestone,
		"milestone_label": req.MilestoneLabel,
		"milestone_order": req.MilestoneOrder,
		"created_at":      now,
	})
	if err != nil {
		r.logger.Error().Err(err).
			Str("task_id", taskID).
			Str("goal_id", goalID).
			Msg("failed to create task-goal link")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	if result == nil {
		return nil, errors.ErrDatabase.WithMessage("failed to create task-goal link")
	}

	r.logger.Info().
		Str("task_id", taskID).
		Str("goal_id", goalID).
		Str("impact_type", req.ImpactType).
		Msg("task-goal link created")

	return result.toTaskGoal(), nil
}

// =============================================================================
// CREATE BATCH
// =============================================================================

// CreateBatch creates multiple task-goal links efficiently.
//
// SQLite has no FOR loop in SQL; we iterate in Go and call
// database.Create for each link, collecting the results. The UNIQUE(task_id,
// goal_id) constraint prevents duplicates within the batch.
func (r *repository) CreateBatch(ctx context.Context, taskID string, links []LinkRequest) ([]*TaskGoal, error) {
	if len(links) == 0 {
		return []*TaskGoal{}, nil
	}

	taskRef := database.RecordID("tasks", taskID)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	source := SourceManual

	taskGoals := make([]*TaskGoal, 0, len(links))

	for _, link := range links {
		goalRef := database.RecordID("goals", link.GoalID)
		edgeID := generateEdgeRecordID()

		result, err := database.Create[taskGoalDB](ctx, r.db, Table, map[string]any{
			"id":              edgeID.String(),
			"task_id":         taskRef,
			"goal_id":         goalRef,
			"impact_type":     link.ImpactType,
			"quantity_value":  link.QuantityValue,
			"unit_id":         link.UnitID,
			"notes":           link.Notes,
			"source":          source,
			"is_milestone":    link.IsMilestone,
			"milestone_label": link.MilestoneLabel,
			"milestone_order": link.MilestoneOrder,
			"created_at":      now,
		})
		if err != nil {
			r.logger.Error().Err(err).
				Str("task_id", taskID).
				Str("goal_id", link.GoalID).
				Int("link_count", len(links)).
				Msg("failed to create batch task-goal links")
			return nil, errors.ErrDatabase.Wrap(err)
		}

		if result == nil {
			return nil, errors.ErrDatabase.WithMessage("failed to create task-goal link")
		}

		taskGoals = append(taskGoals, result.toTaskGoal())
	}

	r.logger.Info().
		Str("task_id", taskID).
		Int("link_count", len(links)).
		Msg("batch task-goal links created")

	return taskGoals, nil
}

// =============================================================================
// FIND BY TASK AND GOAL
// =============================================================================

func (r *repository) FindByTaskAndGoal(ctx context.Context, taskID, goalID string) (*TaskGoal, error) {
	taskRef := database.RecordID("tasks", taskID)
	goalRef := database.RecordID("goals", goalID)

	result, err := database.QueryFirst[taskGoalDB](ctx, r.db, `
		SELECT * FROM task_goals
		WHERE task_id = $task AND goal_id = $goal
		LIMIT 1
	`, map[string]any{
		"task": taskRef,
		"goal": goalRef,
	})
	if err != nil {
		r.logger.Error().Err(err).
			Str("task_id", taskID).
			Str("goal_id", goalID).
			Msg("failed to find task-goal link")
		return nil, err
	}

	if result == nil {
		return nil, errors.ErrNotFound
	}

	return result.toTaskGoal(), nil
}

// =============================================================================
// DELETE
// =============================================================================

func (r *repository) Delete(ctx context.Context, taskID, goalID string) error {
	taskRef := database.RecordID("tasks", taskID)
	goalRef := database.RecordID("goals", goalID)

	_, err := database.QueryAll[any](ctx, r.db, `
		DELETE FROM task_goals WHERE task_id = $task AND goal_id = $goal
	`, map[string]any{
		"task": taskRef,
		"goal": goalRef,
	})
	if err != nil {
		r.logger.Error().Err(err).
			Str("task_id", taskID).
			Str("goal_id", goalID).
			Msg("failed to delete task-goal link")
		return errors.ErrDatabase.Wrap(err)
	}

	r.logger.Info().
		Str("task_id", taskID).
		Str("goal_id", goalID).
		Msg("task-goal link deleted")

	return nil
}

func (r *repository) DeleteByEdgeID(ctx context.Context, edgeID string) error {
	edgeRef := database.RecordID(Table, edgeID)

	_, err := database.QueryAll[any](ctx, r.db, `
		DELETE FROM task_goals WHERE id = $id
	`, map[string]any{
		"id": edgeRef,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("edge_id", edgeID).Msg("failed to delete task-goal link by ID")
		return errors.ErrDatabase.Wrap(err)
	}

	r.logger.Info().Str("edge_id", edgeID).Msg("task-goal link deleted by ID")
	return nil
}

// DeleteByTask removes all goal links for a specific task.
// This is useful when deleting a task or resetting all its goal associations.
// The task_goals table has ON DELETE CASCADE on task_id, so the database
// handles cleanup automatically when a task row is deleted; this method
// supports explicit unlinking without removing the task itself.
func (r *repository) DeleteByTask(ctx context.Context, taskID string) error {
	taskRef := database.RecordID("tasks", taskID)

	_, err := database.QueryAll[any](ctx, r.db, `
		DELETE FROM task_goals WHERE task_id = $task
	`, map[string]any{
		"task": taskRef,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to delete all task-goal links for task")
		return errors.ErrDatabase.Wrap(err)
	}

	r.logger.Info().Str("task_id", taskID).Msg("all task-goal links deleted for task")
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// generateEdgeRecordID produces a unique task_goals:<hex> record ID for new edges.
func generateEdgeRecordID() models.RecordID {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes) //nolint:gosec // crypto/rand.Read never fails in practice
	return database.NewRecordID(Table, hex.EncodeToString(bytes))
}
