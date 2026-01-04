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

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the task-goal link data access interface.
type Repository interface {
	// Create creates a new task-goal link using RELATE.
	Create(ctx context.Context, taskID, goalID string, req *LinkRequest, userID string) (*TaskGoal, error)

	// FindByTaskAndGoal checks if a link exists between a task and goal.
	FindByTaskAndGoal(ctx context.Context, taskID, goalID string) (*TaskGoal, error)

	// Delete removes a task-goal link.
	Delete(ctx context.Context, taskID, goalID string) error

	// DeleteByEdgeID removes a link by its edge ID.
	DeleteByEdgeID(ctx context.Context, edgeID string) error
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
	ID              models.RecordID      `json:"id,omitempty"`
	In              models.RecordID      `json:"in"`  // Task ID
	Out             models.RecordID      `json:"out"` // Goal ID
	ImpactType      string               `json:"impact_type"`
	ImpactMagnitude int                  `json:"impact_magnitude"`
	QuantityValue   *float64             `json:"quantity_value,omitempty"`
	UnitID          *string              `json:"unit_id,omitempty"`
	Notes           string               `json:"notes,omitempty"`
	Source          string               `json:"source"`
	IsMilestone     bool                 `json:"is_milestone"`
	MilestoneLabel  string               `json:"milestone_label,omitempty"`
	MilestoneOrder  int                  `json:"milestone_order"`
	CreatedAt       database.SurrealTime `json:"created_at"`
}

func (t *taskGoalDB) toTaskGoal() *TaskGoal {
	return &TaskGoal{
		ID:              database.ToStringID(t.ID),
		TaskID:          database.ToStringID(t.In),
		GoalID:          database.ToStringID(t.Out),
		ImpactType:      t.ImpactType,
		ImpactMagnitude: t.ImpactMagnitude,
		QuantityValue:   t.QuantityValue,
		UnitID:          t.UnitID,
		Notes:           t.Notes,
		Source:          t.Source,
		IsMilestone:     t.IsMilestone,
		MilestoneLabel:  t.MilestoneLabel,
		MilestoneOrder:  t.MilestoneOrder,
		CreatedAt:       t.CreatedAt.Time,
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
	impactMagnitude := req.ImpactMagnitude
	if impactMagnitude == 0 {
		impactMagnitude = 1
	}

	// Use RELATE to create the edge
	result, err := database.QueryFirst[taskGoalDB](ctx, r.db, `
		RELATE $task -> task_goals -> $goal SET
			impact_type = $impact_type,
			impact_magnitude = $impact_magnitude,
			quantity_value = $quantity_value,
			unit_id = $unit_id,
			notes = $notes,
			source = $source,
			is_milestone = $is_milestone,
			milestone_label = $milestone_label,
			milestone_order = $milestone_order,
			created_at = $created_at
	`, map[string]any{
		"task":             taskRecordID,
		"goal":             goalRecordID,
		"impact_type":      req.ImpactType,
		"impact_magnitude": impactMagnitude,
		"quantity_value":   req.QuantityValue,
		"unit_id":          req.UnitID,
		"notes":            req.Notes,
		"source":           source,
		"is_milestone":     req.IsMilestone,
		"milestone_label":  req.MilestoneLabel,
		"milestone_order":  req.MilestoneOrder,
		"created_at":       now,
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
