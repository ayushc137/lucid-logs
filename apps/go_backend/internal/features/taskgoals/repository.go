// Package taskgoals provides task-goal relationship management using SurrealDB.
//
// This package implements:
//   - RELATE operations for creating task→goal edges
//   - Delete operations for unlinking
//   - Edge lookup by task+goal for duplicate prevention
//
// Database Architecture:
//
// Uses SurrealDB RELATE for creating edges:
//
//	RELATE tasks:abc -> goals:xyz SET impact_type = "positive", ...
//
// The task_goals table is a relation table (TYPE RELATION IN tasks OUT goals).
package taskgoals

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go/pkg/models"

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
type taskGoalDB struct {
	ID             models.RecordID      `json:"id,omitempty"`
	In             models.RecordID      `json:"in"`  // Task ID
	Out            models.RecordID      `json:"out"` // Goal ID
	ImpactType     string               `json:"impact_type"`
	QuantityValue  *float64             `json:"quantity_value,omitempty"`
	UnitID         *string              `json:"unit_id,omitempty"`
	Notes          string               `json:"notes,omitempty"`
	Source         string               `json:"source"`
	IsMilestone    bool                 `json:"is_milestone"`
	MilestoneLabel string               `json:"milestone_label,omitempty"`
	MilestoneOrder int                  `json:"milestone_order"`
	CreatedAt      database.SurrealTime `json:"created_at"`
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
	taskRecordID := database.MustRecordID("tasks", taskID)
	goalRecordID := database.MustRecordID("goals", goalID)
	now := time.Now().UTC()

	// Set defaults
	source := SourceManual

	// Use RELATE to create the edge
	result, err := database.QueryFirst[taskGoalDB](ctx, r.db, `
		RELATE $task -> task_goals -> $goal SET
			impact_type = $impact_type,
			quantity_value = $quantity_value,
			unit_id = $unit_id,
			notes = $notes,
			source = $source,
			is_milestone = $is_milestone,
			milestone_label = $milestone_label,
			milestone_order = $milestone_order,
			created_at = $created_at
	`, map[string]any{
		"task":            taskRecordID,
		"goal":            goalRecordID,
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

// CreateBatch creates multiple task-goal links efficiently in a single query.
// This uses SurrealDB's FOR loop to minimize database round trips.
func (r *repository) CreateBatch(ctx context.Context, taskID string, links []LinkRequest) ([]*TaskGoal, error) {
	if len(links) == 0 {
		return []*TaskGoal{}, nil
	}

	taskRecordID := database.MustRecordID("tasks", taskID)
	now := time.Now().UTC()

	// Build array of link data for batch insertion
	linkData := make([]map[string]any, len(links))
	for i, link := range links {
		linkData[i] = map[string]any{
			"goal_id":         link.GoalID,
			"impact_type":     link.ImpactType,
			"quantity_value":  link.QuantityValue,
			"unit_id":         link.UnitID,
			"notes":           link.Notes,
			"is_milestone":    link.IsMilestone,
			"milestone_label": link.MilestoneLabel,
			"milestone_order": link.MilestoneOrder,
		}
	}

	// Use FOR loop for batch creation in single query
	results, err := database.QueryAll[taskGoalDB](ctx, r.db, `
		FOR $link IN $links {
			LET $goal = type::thing("goals", string::split($link.goal_id, ":")[1]);
			RELATE $task -> task_goals -> $goal SET
				impact_type = $link.impact_type,
				quantity_value = $link.quantity_value,
				unit_id = $link.unit_id,
				notes = $link.notes,
				source = $source,
				is_milestone = $link.is_milestone,
				milestone_label = $link.milestone_label,
				milestone_order = $link.milestone_order,
				created_at = $now;
		}
	`, map[string]any{
		"task":   taskRecordID,
		"links":  linkData,
		"source": SourceManual,
		"now":    now,
	})
	if err != nil {
		r.logger.Error().Err(err).
			Str("task_id", taskID).
			Int("link_count", len(links)).
			Msg("failed to create batch task-goal links")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	taskGoals := make([]*TaskGoal, len(results))
	for i, tg := range results {
		taskGoals[i] = tg.toTaskGoal()
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
	taskRecordID := database.MustRecordID("tasks", taskID)
	goalRecordID := database.MustRecordID("goals", goalID)

	result, err := database.QueryFirst[taskGoalDB](ctx, r.db, `
		SELECT * FROM task_goals 
		WHERE in = $task AND out = $goal
	`, map[string]any{
		"task": taskRecordID,
		"goal": goalRecordID,
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
	taskRecordID := database.MustRecordID("tasks", taskID)
	goalRecordID := database.MustRecordID("goals", goalID)

	_, err := database.QueryAll[taskGoalDB](ctx, r.db, `
		DELETE task_goals WHERE in = $task AND out = $goal
	`, map[string]any{
		"task": taskRecordID,
		"goal": goalRecordID,
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
	edgeRecordID := database.MustRecordID(Table, edgeID)

	_, err := database.QueryAll[taskGoalDB](ctx, r.db, `
		DELETE type::thing($id)
	`, map[string]any{
		"id": edgeRecordID,
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
func (r *repository) DeleteByTask(ctx context.Context, taskID string) error {
	taskRecordID := database.MustRecordID("tasks", taskID)

	_, err := database.QueryAll[taskGoalDB](ctx, r.db, `
		DELETE task_goals WHERE in = $task
	`, map[string]any{
		"task": taskRecordID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("task_id", taskID).Msg("failed to delete all task-goal links for task")
		return errors.ErrDatabase.Wrap(err)
	}

	r.logger.Info().Str("task_id", taskID).Msg("all task-goal links deleted for task")
	return nil
}
