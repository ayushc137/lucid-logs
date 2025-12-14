package goalactions

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

// Repository defines the goal action data access interface.
type Repository interface {
	// FindByGoalID retrieves all actions for a goal.
	FindByGoalID(ctx context.Context, goalID, userID string) ([]*GoalAction, error)

	// FindByID retrieves a single action by ID.
	FindByID(ctx context.Context, id, goalID, userID string) (*GoalAction, error)

	// Create creates a new goal action.
	Create(ctx context.Context, goalID string, req *CreateRequest, userID string) (*GoalAction, error)

	// Update updates an existing goal action.
	Update(ctx context.Context, id, goalID string, req *UpdateRequest, userID string) (*GoalAction, error)

	// Delete soft-deletes a goal action.
	Delete(ctx context.Context, id, goalID, userID string) error

	// MarkComplete marks an action as completed.
	MarkComplete(ctx context.Context, id, goalID, userID string) (*GoalAction, error)

	// MarkIncomplete marks an action as not completed.
	MarkIncomplete(ctx context.Context, id, goalID, userID string) (*GoalAction, error)

	// GetMaxOrder retrieves the maximum order value for a goal's actions.
	GetMaxOrder(ctx context.Context, goalID, userID string) (int, error)

	// CountCompleted counts completed actions for a goal.
	CountCompleted(ctx context.Context, goalID, userID string) (int, int, error) // completed, total

	// Reorder updates the order of actions.
	Reorder(ctx context.Context, goalID string, actionIDs []string, userID string) error
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new goal action Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "goalactions").Logger(),
	}
}

// =============================================================================
// DATABASE MODEL
// =============================================================================

type goalActionDB struct {
	ID        models.RecordID `json:"id,omitempty"`
	GoalID    string          `json:"goal_id"`
	CreatedBy string          `json:"created_by"`

	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Order       int    `json:"order"`

	QuantityValue *float64 `json:"quantity_value,omitempty"`
	QuantityUnit  *string  `json:"quantity_unit,omitempty"`

	Completed   bool                  `json:"completed"`
	CompletedAt *database.SurrealTime `json:"completed_at,omitempty"`

	CreatedAt database.SurrealTime  `json:"created_at"`
	UpdatedAt database.SurrealTime  `json:"updated_at"`
	DeletedAt *database.SurrealTime `json:"deleted_at,omitempty"`
}

func (a *goalActionDB) toGoalAction() *GoalAction {
	action := &GoalAction{
		ID:            database.ToStringID(a.ID),
		GoalID:        a.GoalID,
		CreatedBy:     a.CreatedBy,
		Title:         a.Title,
		Description:   a.Description,
		Order:         a.Order,
		QuantityValue: a.QuantityValue,
		QuantityUnit:  a.QuantityUnit,
		Completed:     a.Completed,
		CreatedAt:     a.CreatedAt.Time,
		UpdatedAt:     a.UpdatedAt.Time,
	}

	if a.CompletedAt != nil && !a.CompletedAt.IsZero() {
		t := a.CompletedAt.Time
		action.CompletedAt = &t
	}
	if a.DeletedAt != nil && !a.DeletedAt.IsZero() {
		t := a.DeletedAt.Time
		action.DeletedAt = &t
	}

	return action
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindByGoalID(ctx context.Context, goalID, userID string) ([]*GoalAction, error) {
	actionsDB, err := database.QueryAll[goalActionDB](ctx, r.db, `
		SELECT * FROM goal_actions 
		WHERE goal_id = $goal_id 
		  AND created_by = $user 
		  AND deleted_at IS NONE
		ORDER BY order ASC
	`, map[string]any{
		"goal_id": goalID,
		"user":    userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", goalID).Msg("find actions by goal failed")
		return nil, err
	}

	actions := make([]*GoalAction, len(actionsDB))
	for i := range actionsDB {
		actions[i] = actionsDB[i].toGoalAction()
	}

	return actions, nil
}

func (r *repository) FindByID(ctx context.Context, id, goalID, userID string) (*GoalAction, error) {
	actionID := database.MustRecordID(Table, id)

	action, err := database.QueryFirst[goalActionDB](ctx, r.db, `
		SELECT * FROM type::thing($id)
	`, map[string]any{
		"id": actionID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("action_id", id).Msg("query failed for action fetch")
		return nil, err
	}

	if action == nil {
		return nil, errors.ErrNotFound
	}

	if action.CreatedBy != userID || action.GoalID != goalID || action.DeletedAt != nil {
		return nil, errors.ErrNotFound
	}

	return action.toGoalAction(), nil
}

// =============================================================================
// CREATE OPERATION
// =============================================================================

func (r *repository) Create(ctx context.Context, goalID string, req *CreateRequest, userID string) (*GoalAction, error) {
	actionID := generateRecordID()
	now := time.Now().UTC()

	// Get max order if not provided
	order := 0
	if req.Order != nil {
		order = *req.Order
	} else {
		maxOrder, _ := r.GetMaxOrder(ctx, goalID, userID)
		order = maxOrder + 1
	}

	createData := map[string]any{
		"goal_id":     goalID,
		"created_by":  userID,
		"title":       req.Title,
		"description": req.Description,
		"order":       order,
		"completed":   false,
		"created_at":  now,
		"updated_at":  now,
	}

	if req.QuantityValue != nil {
		createData["quantity_value"] = *req.QuantityValue
	}
	if req.QuantityUnit != nil {
		createData["quantity_unit"] = *req.QuantityUnit
	}

	_, err := database.QueryAll[goalActionDB](ctx, r.db, `
		CREATE type::thing($id) CONTENT $data
	`, map[string]any{
		"id":   actionID,
		"data": createData,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", goalID).Msg("create action failed")
		return nil, err
	}

	r.logger.Info().Str("action_id", database.ToStringID(actionID)).Str("goal_id", goalID).Msg("action created")

	return r.FindByID(ctx, database.ToStringID(actionID), goalID, userID)
}

// =============================================================================
// UPDATE OPERATIONS
// =============================================================================

func (r *repository) Update(ctx context.Context, id, goalID string, req *UpdateRequest, userID string) (*GoalAction, error) {
	// Verify ownership
	_, err := r.FindByID(ctx, id, goalID, userID)
	if err != nil {
		return nil, err
	}

	actionID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	updateData := map[string]any{
		"updated_at": now,
	}

	if req.Title != nil {
		updateData["title"] = *req.Title
	}
	if req.Description != nil {
		updateData["description"] = *req.Description
	}
	if req.Order != nil {
		updateData["order"] = *req.Order
	}
	if req.QuantityValue != nil {
		updateData["quantity_value"] = *req.QuantityValue
	}
	if req.QuantityUnit != nil {
		updateData["quantity_unit"] = *req.QuantityUnit
	}
	if req.Completed != nil {
		updateData["completed"] = *req.Completed
		if *req.Completed {
			updateData["completed_at"] = now
		} else {
			updateData["completed_at"] = nil
		}
	}

	_, err = database.QueryAll[goalActionDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE $data
	`, map[string]any{
		"id":   actionID,
		"data": updateData,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("action_id", id).Msg("update action failed")
		return nil, err
	}

	r.logger.Info().Str("action_id", id).Msg("action updated")

	return r.FindByID(ctx, id, goalID, userID)
}

func (r *repository) MarkComplete(ctx context.Context, id, goalID, userID string) (*GoalAction, error) {
	_, err := r.FindByID(ctx, id, goalID, userID)
	if err != nil {
		return nil, err
	}

	actionID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	_, err = database.QueryAll[goalActionDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE {
			completed: true,
			completed_at: $now,
			updated_at: $now
		}
	`, map[string]any{
		"id":  actionID,
		"now": now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("action_id", id).Msg("mark complete failed")
		return nil, err
	}

	r.logger.Info().Str("action_id", id).Msg("action marked complete")

	return r.FindByID(ctx, id, goalID, userID)
}

func (r *repository) MarkIncomplete(ctx context.Context, id, goalID, userID string) (*GoalAction, error) {
	_, err := r.FindByID(ctx, id, goalID, userID)
	if err != nil {
		return nil, err
	}

	actionID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	_, err = database.QueryAll[goalActionDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE {
			completed: false,
			completed_at: NONE,
			updated_at: $now
		}
	`, map[string]any{
		"id":  actionID,
		"now": now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("action_id", id).Msg("mark incomplete failed")
		return nil, err
	}

	r.logger.Info().Str("action_id", id).Msg("action marked incomplete")

	return r.FindByID(ctx, id, goalID, userID)
}

// =============================================================================
// DELETE OPERATION
// =============================================================================

func (r *repository) Delete(ctx context.Context, id, goalID, userID string) error {
	_, err := r.FindByID(ctx, id, goalID, userID)
	if err != nil {
		return err
	}

	actionID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	_, err = database.QueryAll[goalActionDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE {
			deleted_at: $now,
			updated_at: $now
		}
	`, map[string]any{
		"id":  actionID,
		"now": now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("action_id", id).Msg("delete action failed")
		return err
	}

	r.logger.Info().Str("action_id", id).Msg("action deleted")
	return nil
}

// =============================================================================
// UTILITY OPERATIONS
// =============================================================================

func (r *repository) GetMaxOrder(ctx context.Context, goalID, userID string) (int, error) {
	result, err := database.QueryScalar[float64](ctx, r.db, `
		RETURN (SELECT math::max(order) as max_order FROM goal_actions 
		 WHERE goal_id = $goal_id 
		   AND created_by = $user 
		   AND deleted_at IS NONE
		 GROUP ALL)[0].max_order OR 0
	`, map[string]any{
		"goal_id": goalID,
		"user":    userID,
	})
	if err != nil {
		return 0, err
	}
	return int(result), nil
}

func (r *repository) CountCompleted(ctx context.Context, goalID, userID string) (int, int, error) {
	type countResult struct {
		Completed int `json:"completed"`
		Total     int `json:"total"`
	}

	result, err := database.QueryFirst[countResult](ctx, r.db, `
		SELECT 
			count(IF completed = true THEN 1 ELSE NONE END) as completed,
			count() as total
		FROM goal_actions 
		WHERE goal_id = $goal_id 
		  AND created_by = $user 
		  AND deleted_at IS NONE
		GROUP ALL
	`, map[string]any{
		"goal_id": goalID,
		"user":    userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", goalID).Msg("count completed failed")
		return 0, 0, err
	}

	if result == nil {
		return 0, 0, nil
	}

	return result.Completed, result.Total, nil
}

func (r *repository) Reorder(ctx context.Context, goalID string, actionIDs []string, userID string) error {
	now := time.Now().UTC()

	for i, id := range actionIDs {
		actionID := database.MustRecordID(Table, id)
		_, err := database.QueryAll[goalActionDB](ctx, r.db, `
			UPDATE type::thing($id) MERGE {
				order: $order,
				updated_at: $now
			} WHERE created_by = $user AND goal_id = $goal_id
		`, map[string]any{
			"id":      actionID,
			"order":   i + 1,
			"now":     now,
			"user":    userID,
			"goal_id": goalID,
		})
		if err != nil {
			r.logger.Error().Err(err).Str("action_id", id).Msg("reorder failed")
			return err
		}
	}

	r.logger.Info().Str("goal_id", goalID).Int("count", len(actionIDs)).Msg("actions reordered")
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

func generateRecordID() models.RecordID {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return database.NewRecordID(Table, hex.EncodeToString(bytes))
}
