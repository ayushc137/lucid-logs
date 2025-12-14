package goalactions

import (
	"context"

	"github.com/lucid-logs/go-backend/internal/features/goals"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the goal action business logic interface.
type Service interface {
	// List retrieves all actions for a goal.
	List(ctx context.Context, goalID, userID string) (*ActionListResponse, error)

	// Get retrieves a single action by ID.
	Get(ctx context.Context, id, goalID, userID string) (*GoalAction, error)

	// Create creates a new goal action.
	Create(ctx context.Context, goalID string, req *CreateRequest, userID string) (*GoalAction, error)

	// Update updates an existing goal action.
	Update(ctx context.Context, id, goalID string, req *UpdateRequest, userID string) (*GoalAction, error)

	// Delete removes a goal action.
	Delete(ctx context.Context, id, goalID, userID string) error

	// MarkComplete marks an action as completed. Triggers auto-completion check.
	MarkComplete(ctx context.Context, id, goalID, userID string) (*GoalAction, error)

	// MarkIncomplete marks an action as not completed.
	MarkIncomplete(ctx context.Context, id, goalID, userID string) (*GoalAction, error)

	// Reorder sets the order of actions.
	Reorder(ctx context.Context, goalID string, req *ReorderRequest, userID string) error
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo        Repository
	goalService goals.Service
	goalRepo    goals.Repository
	logger      zerolog.Logger
}

// NewService creates a new goal action Service.
func NewService(repo Repository, goalService goals.Service, goalRepo goals.Repository) Service {
	return &service{
		repo:        repo,
		goalService: goalService,
		goalRepo:    goalRepo,
		logger:      log.With().Str("service", "goalactions").Logger(),
	}
}

// =============================================================================
// LIST
// =============================================================================

func (s *service) List(ctx context.Context, goalID, userID string) (*ActionListResponse, error) {
	// Verify goal exists and user owns it
	_, err := s.goalService.Get(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	actions, err := s.repo.FindByGoalID(ctx, goalID, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("goal_id", goalID).Msg("failed to list actions")
		return nil, err
	}

	return &ActionListResponse{
		GoalID:  goalID,
		Actions: actions,
		Count:   len(actions),
	}, nil
}

// =============================================================================
// GET
// =============================================================================

func (s *service) Get(ctx context.Context, id, goalID, userID string) (*GoalAction, error) {
	// Verify goal exists and user owns it
	_, err := s.goalService.Get(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	action, err := s.repo.FindByID(ctx, id, goalID, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("action_id", id).Msg("failed to get action")
		return nil, err
	}

	return action, nil
}

// =============================================================================
// CREATE
// =============================================================================

func (s *service) Create(ctx context.Context, goalID string, req *CreateRequest, userID string) (*GoalAction, error) {
	// Verify goal exists and user owns it
	_, err := s.goalService.Get(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	action, err := s.repo.Create(ctx, goalID, req, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("goal_id", goalID).Msg("failed to create action")
		return nil, err
	}

	s.logger.Info().
		Str("action_id", action.ID).
		Str("goal_id", goalID).
		Str("user_id", userID).
		Msg("action created")

	return action, nil
}

// =============================================================================
// UPDATE
// =============================================================================

func (s *service) Update(ctx context.Context, id, goalID string, req *UpdateRequest, userID string) (*GoalAction, error) {
	// Verify goal exists and get its type for auto-completion
	goal, err := s.goalService.Get(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	// Get current action state to detect completion change
	var wasCompleted bool
	if req.Completed != nil {
		existingAction, err := s.repo.FindByID(ctx, id, goalID, userID)
		if err != nil {
			if errors.Is(err, errors.ErrNotFound) {
				return nil, errors.ErrNotFound
			}
			return nil, err
		}
		wasCompleted = existingAction.Completed
	}

	action, err := s.repo.Update(ctx, id, goalID, req, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("action_id", id).Msg("failed to update action")
		return nil, err
	}

	s.logger.Info().
		Str("action_id", id).
		Str("user_id", userID).
		Msg("action updated")

	// If completion status changed to true, check for goal auto-completion
	if req.Completed != nil && *req.Completed && !wasCompleted {
		if goal.GoalType == goals.GoalTypeDiscrete {
			if err := s.checkAndAutoCompleteGoal(ctx, goalID, goal, userID); err != nil {
				s.logger.Warn().Err(err).Str("goal_id", goalID).Msg("auto-completion check failed")
			}
		}
	}

	return action, nil
}

// =============================================================================
// DELETE
// =============================================================================

func (s *service) Delete(ctx context.Context, id, goalID, userID string) error {
	// Verify goal exists
	_, err := s.goalService.Get(ctx, goalID, userID)
	if err != nil {
		return err
	}

	err = s.repo.Delete(ctx, id, goalID, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("action_id", id).Msg("failed to delete action")
		return err
	}

	s.logger.Info().
		Str("action_id", id).
		Str("user_id", userID).
		Msg("action deleted")

	return nil
}

// =============================================================================
// MARK COMPLETE (with auto-completion check)
// =============================================================================

func (s *service) MarkComplete(ctx context.Context, id, goalID, userID string) (*GoalAction, error) {
	// Verify goal exists and get its type
	goal, err := s.goalService.Get(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	action, err := s.repo.MarkComplete(ctx, id, goalID, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("action_id", id).Msg("failed to mark complete")
		return nil, err
	}

	s.logger.Info().
		Str("action_id", id).
		Str("goal_id", goalID).
		Msg("action marked complete")

	// Check for auto-completion of the goal (only for discrete goals)
	if goal.GoalType == goals.GoalTypeDiscrete {
		if err := s.checkAndAutoCompleteGoal(ctx, goalID, goal, userID); err != nil {
			s.logger.Warn().Err(err).Str("goal_id", goalID).Msg("auto-completion check failed")
			// Don't fail the action completion for this
		}
	}

	return action, nil
}

func (s *service) MarkIncomplete(ctx context.Context, id, goalID, userID string) (*GoalAction, error) {
	// Verify goal exists
	_, err := s.goalService.Get(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	action, err := s.repo.MarkIncomplete(ctx, id, goalID, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("action_id", id).Msg("failed to mark incomplete")
		return nil, err
	}

	s.logger.Info().
		Str("action_id", id).
		Str("goal_id", goalID).
		Msg("action marked incomplete")

	return action, nil
}

// =============================================================================
// REORDER
// =============================================================================

func (s *service) Reorder(ctx context.Context, goalID string, req *ReorderRequest, userID string) error {
	// Verify goal exists
	_, err := s.goalService.Get(ctx, goalID, userID)
	if err != nil {
		return err
	}

	err = s.repo.Reorder(ctx, goalID, req.ActionIDs, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("goal_id", goalID).Msg("failed to reorder actions")
		return err
	}

	s.logger.Info().
		Str("goal_id", goalID).
		Int("count", len(req.ActionIDs)).
		Msg("actions reordered")

	return nil
}

// =============================================================================
// AUTO-COMPLETION LOGIC
// =============================================================================

// checkAndAutoCompleteGoal checks if all actions are complete and auto-completes the goal.
func (s *service) checkAndAutoCompleteGoal(ctx context.Context, goalID string, goal *goals.Goal, userID string) error {
	completed, total, err := s.repo.CountCompleted(ctx, goalID, userID)
	if err != nil {
		return err
	}

	// If no actions, don't auto-complete
	if total == 0 {
		return nil
	}

	// All actions completed?
	if completed == total {
		s.logger.Info().
			Str("goal_id", goalID).
			Int("completed", completed).
			Int("total", total).
			Msg("all actions complete, auto-completing goal")

		// Update goal status to completed
		if err := s.goalRepo.UpdateStatus(ctx, goalID, goals.StatusCompleted, userID); err != nil {
			return err
		}

		s.logger.Info().
			Str("goal_id", goalID).
			Msg("goal auto-completed")
	}

	return nil
}
