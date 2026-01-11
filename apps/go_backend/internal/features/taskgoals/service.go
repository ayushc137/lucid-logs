package taskgoals

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/features/goals"
	"github.com/lucid-logs/go-backend/internal/features/tasks"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the task-goal linking business logic interface.
type Service interface {
	// Link creates a task-goal relationship.
	Link(ctx context.Context, taskID string, req *LinkRequest, userID string) (*TaskGoal, error)

	// LinkBatch links a task to multiple goals.
	LinkBatch(ctx context.Context, taskID string, req *BatchLinkRequest, userID string) ([]*TaskGoal, error)

	// Unlink removes a task-goal relationship.
	Unlink(ctx context.Context, taskID, goalID, userID string) error
}

// GoalLogger is an interface for logging goal events.
// This allows logging task link/unlink events without circular imports.
type GoalLogger interface {
	LogEvent(ctx context.Context, goalID, event, userID string, changes map[string]any, stats *goals.GoalStats) error
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo        Repository
	taskService tasks.Service
	goalService goals.Service
	goalLogger  GoalLogger
	logger      zerolog.Logger
}

// NewService creates a new task-goal linking Service.
// goalLogger is optional - pass nil to disable goal event logging.
func NewService(repo Repository, taskService tasks.Service, goalService goals.Service, goalLogger GoalLogger) Service {
	return &service{
		repo:        repo,
		taskService: taskService,
		goalService: goalService,
		goalLogger:  goalLogger,
		logger:      log.With().Str("service", "taskgoals").Logger(),
	}
}

// =============================================================================
// LINK
// =============================================================================

func (s *service) Link(ctx context.Context, taskID string, req *LinkRequest, userID string) (*TaskGoal, error) {
	// Verify task exists and user owns it
	task, err := s.taskService.Get(ctx, taskID, userID)
	if err != nil {
		s.logger.Debug().Err(err).Str("task_id", taskID).Msg("task not found or not owned by user")
		return nil, errors.ErrNotFound.WithMessage("Task not found")
	}

	// Verify goal exists and user owns it
	goal, err := s.goalService.Get(ctx, req.GoalID, userID)
	if err != nil {
		s.logger.Debug().Err(err).Str("goal_id", req.GoalID).Msg("goal not found or not owned by user")
		return nil, errors.ErrNotFound.WithMessage("Goal not found")
	}

	// Check if already linked
	existing, err := s.repo.FindByTaskAndGoal(ctx, taskID, req.GoalID)
	if err != nil && !errors.Is(err, errors.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		s.logger.Debug().
			Str("task_id", taskID).
			Str("goal_id", req.GoalID).
			Msg("task-goal link already exists")
		return nil, errors.ErrConflict.WithMessage("Task is already linked to this goal")
	}

	// Create the link
	link, err := s.repo.Create(ctx, taskID, req.GoalID, req, userID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("task_id", taskID).
			Str("goal_id", req.GoalID).
			Msg("failed to create task-goal link")
		return nil, err
	}

	s.logger.Info().
		Str("task_id", taskID).
		Str("goal_id", req.GoalID).
		Str("impact_type", req.ImpactType).
		Msg("task linked to goal")

	// Log the task_linked event to goal history
	if s.goalLogger != nil {
		changes := map[string]any{
			"task_id":    taskID,
			"task_title": task.Title,
		}
		if req.ImpactType != "" {
			changes["impact_type"] = req.ImpactType
		}
		if req.ImpactMagnitude > 0 {
			changes["impact_magnitude"] = req.ImpactMagnitude
		}
		if req.QuantityValue != nil && *req.QuantityValue > 0 {
			changes["quantity_value"] = *req.QuantityValue
		}
		if err := s.goalLogger.LogEvent(ctx, req.GoalID, "task_linked", userID, changes, nil); err != nil {
			s.logger.Warn().Err(err).Str("goal_id", req.GoalID).Msg("failed to log task_linked event")
		}
	}

	// Update goal stats (record completion may update streak)
	if task.Completed {
		if err := s.goalService.RecordCompletion(ctx, goal.ID, userID, task.EndDate); err != nil {
			s.logger.Warn().Err(err).Str("goal_id", goal.ID).Msg("failed to record completion")
		}
	}

	return link, nil
}

// =============================================================================
// LINK BATCH
// =============================================================================

func (s *service) LinkBatch(ctx context.Context, taskID string, req *BatchLinkRequest, userID string) ([]*TaskGoal, error) {
	// Verify task exists first
	task, err := s.taskService.Get(ctx, taskID, userID)
	if err != nil {
		return nil, errors.ErrNotFound.WithMessage("Task not found")
	}
	_ = task

	results := make([]*TaskGoal, 0, len(req.Links))

	for _, linkReq := range req.Links {
		link, err := s.Link(ctx, taskID, &linkReq, userID)
		if err != nil {
			// Log but continue with other links
			s.logger.Warn().Err(err).
				Str("task_id", taskID).
				Str("goal_id", linkReq.GoalID).
				Msg("failed to create one link in batch")
			continue
		}
		results = append(results, link)
	}

	if len(results) == 0 && len(req.Links) > 0 {
		return nil, errors.ErrBadRequest.WithMessage("Failed to create any links")
	}

	s.logger.Info().
		Str("task_id", taskID).
		Int("requested", len(req.Links)).
		Int("created", len(results)).
		Msg("batch link completed")

	return results, nil
}

// =============================================================================
// UNLINK
// =============================================================================

func (s *service) Unlink(ctx context.Context, taskID, goalID, userID string) error {
	// Verify task ownership
	task, err := s.taskService.Get(ctx, taskID, userID)
	if err != nil {
		return errors.ErrNotFound.WithMessage("Task not found")
	}

	// Check link exists
	existing, err := s.repo.FindByTaskAndGoal(ctx, taskID, goalID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return errors.ErrNotFound.WithMessage("Task-goal link not found")
		}
		return err
	}
	_ = existing

	// Delete the link
	if err := s.repo.Delete(ctx, taskID, goalID); err != nil {
		s.logger.Error().Err(err).
			Str("task_id", taskID).
			Str("goal_id", goalID).
			Msg("failed to delete task-goal link")
		return err
	}

	s.logger.Info().
		Str("task_id", taskID).
		Str("goal_id", goalID).
		Msg("task unlinked from goal")

	// Log the task_unlinked event to goal history
	if s.goalLogger != nil {
		changes := map[string]any{
			"task_id":    taskID,
			"task_title": task.Title,
		}
		if err := s.goalLogger.LogEvent(ctx, goalID, "task_unlinked", userID, changes, nil); err != nil {
			s.logger.Warn().Err(err).Str("goal_id", goalID).Msg("failed to log task_unlinked event")
		}
	}

	return nil
}
