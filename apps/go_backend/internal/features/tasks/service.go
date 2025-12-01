package tasks

import (
	"context"

	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/lucid-logs/go-backend/internal/shared/timeutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the task business logic interface.
//
// The service layer:
//   - Validates business rules
//   - Orchestrates repository calls
//   - Handles cross-cutting concerns (logging, etc.)
type Service interface {
	// List retrieves paginated tasks for a user.
	List(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Task], error)

	// Get retrieves a single task by ID.
	Get(ctx context.Context, id, userID string) (*Task, error)

	// Create creates a new task.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Task, error)

	// Update updates an existing task.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Task, error)

	// Delete soft-deletes a task.
	Delete(ctx context.Context, id, userID string) error
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

// service is the production implementation of Service.
type service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a new task Service.
func NewService(repo Repository) Service {
	return &service{
		repo:   repo,
		logger: log.With().Str("service", "tasks").Logger(),
	}
}

// =============================================================================
// LIST
// =============================================================================

// List retrieves paginated tasks for a user.
func (s *service) List(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Task], error) {
	tasks, total, err := s.repo.FindPaginated(ctx, userID, params)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to list tasks")
		return nil, err
	}

	s.logger.Debug().
		Str("user_id", userID).
		Int("count", len(tasks)).
		Int64("total", total).
		Msg("tasks listed")

	resp := pagination.NewResponse(tasks, total, params)
	return &resp, nil
}

// =============================================================================
// GET
// =============================================================================

// Get retrieves a single task by ID.
func (s *service) Get(ctx context.Context, id, userID string) (*Task, error) {
	task, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to get task")
		return nil, err
	}

	return task, nil
}

// =============================================================================
// CREATE
// =============================================================================

// Create creates a new task with validation.
func (s *service) Create(ctx context.Context, req *CreateRequest, userID string) (*Task, error) {
	// Parse and validate dates
	startDate, err := timeutil.ParseDateTime(req.StartDate)
	if err != nil {
		return nil, errors.ErrBadRequest.WithMessage("Invalid start_date format")
	}

	endDate, err := timeutil.ParseDateTime(req.EndDate)
	if err != nil {
		return nil, errors.ErrBadRequest.WithMessage("Invalid end_date format")
	}

	// Validate date range
	if endDate.Before(startDate) {
		return nil, errors.ErrInvalidDateRange
	}

	// Create task
	task, err := s.repo.Create(ctx, req, userID)
	if err != nil {
		if errors.Is(err, errors.ErrCategoryNotFound) {
			return nil, errors.ErrCategoryNotFound
		}
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to create task")
		return nil, err
	}

	s.logger.Info().
		Str("task_id", task.ID).
		Str("user_id", userID).
		Msg("task created")

	return task, nil
}

// =============================================================================
// UPDATE
// =============================================================================

// Update updates an existing task with validation.
func (s *service) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Task, error) {
	// Validate dates if both provided
	if req.StartDate != nil && req.EndDate != nil {
		startDate, err := timeutil.ParseDateTime(*req.StartDate)
		if err != nil {
			return nil, errors.ErrBadRequest.WithMessage("Invalid start_date format")
		}

		endDate, err := timeutil.ParseDateTime(*req.EndDate)
		if err != nil {
			return nil, errors.ErrBadRequest.WithMessage("Invalid end_date format")
		}

		if endDate.Before(startDate) {
			return nil, errors.ErrInvalidDateRange
		}
	}

	// Update task
	task, err := s.repo.Update(ctx, id, req, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		if errors.Is(err, errors.ErrCategoryNotFound) {
			return nil, errors.ErrCategoryNotFound
		}
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to update task")
		return nil, err
	}

	s.logger.Info().
		Str("task_id", id).
		Str("user_id", userID).
		Msg("task updated")

	return task, nil
}

// =============================================================================
// DELETE
// =============================================================================

// Delete soft-deletes a task.
func (s *service) Delete(ctx context.Context, id, userID string) error {
	err := s.repo.Delete(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("task_id", id).Msg("failed to delete task")
		return err
	}

	s.logger.Info().
		Str("task_id", id).
		Str("user_id", userID).
		Msg("task deleted")

	return nil
}
