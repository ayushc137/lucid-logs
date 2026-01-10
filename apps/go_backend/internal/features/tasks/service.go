package tasks

import (
	"context"
	"strings"
	"time"

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

	// ListFiltered retrieves paginated tasks with filters and search.
	// Supports full-text search on title, journal, and note fields.
	ListFiltered(ctx context.Context, userID string, filters TaskFilterParams, params pagination.Params) (*pagination.Response[*Task], error)

	// Get retrieves a single task by ID.
	Get(ctx context.Context, id, userID string) (*Task, error)

	// Create creates a new task.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Task, error)

	// Update updates an existing task.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Task, error)

	// Delete soft-deletes a task.
	Delete(ctx context.Context, id, userID string) error

	// GetLastTaskEndTime retrieves the end time of the most recently finished task.
	GetLastTaskEndTime(ctx context.Context, userID string) (*time.Time, error)
}

// ActivityLogger is an interface for logging activity events.
// This allows the tasks service to log activities without circular imports.
type ActivityLogger interface {
	Log(ctx context.Context, entityType, entityID, entityTitle, entityIcon, event, userID string, changes map[string]any) error
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

// service is the production implementation of Service.
type service struct {
	repo           Repository
	activityLogger ActivityLogger
	logger         zerolog.Logger
}

// NewService creates a new task Service.
// activityLogger can be nil if activity logging is not needed.
func NewService(repo Repository, activityLogger ActivityLogger) Service {
	return &service{
		repo:           repo,
		activityLogger: activityLogger,
		logger:         log.With().Str("service", "tasks").Logger(),
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
// LIST FILTERED
// =============================================================================

// ListFiltered retrieves paginated tasks with filters and search.
func (s *service) ListFiltered(ctx context.Context, userID string, filters TaskFilterParams, params pagination.Params) (*pagination.Response[*Task], error) {
	tasks, total, err := s.repo.FindFiltered(ctx, userID, filters, params)
	if err != nil {
		s.logger.Error().Err(err).
			Str("user_id", userID).
			Str("search", filters.Search).
			Str("category", filters.CategoryID).
			Str("status", filters.Status).
			Msg("failed to list filtered tasks")
		return nil, err
	}

	s.logger.Debug().
		Str("user_id", userID).
		Str("search", filters.Search).
		Int("count", len(tasks)).
		Int64("total", total).
		Msg("filtered tasks listed")

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

	// Log the activity
	if s.activityLogger != nil {
		changes := map[string]any{
			"title":      task.Title,
			"start_date": task.StartDate,
			"end_date":   task.EndDate,
			"completed":  task.Completed,
		}
		if task.Category != nil {
			changes["category"] = task.Category.Name
		}
		if err := s.activityLogger.Log(ctx, "task", task.ID, task.Title, "", "created", userID, changes); err != nil {
			s.logger.Warn().Err(err).Str("task_id", task.ID).Msg("failed to log task created activity")
		}
	}

	return task, nil
}

// =============================================================================
// UPDATE
// =============================================================================

func (s *service) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Task, error) {
	// Get old task for change detection
	oldTask, _ := s.repo.FindByID(ctx, id, userID)

	startDate, err := validateOptionalDate("start_date", req.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := validateOptionalDate("end_date", req.EndDate)
	if err != nil {
		return nil, err
	}

	// Validate date ordering when both provided in payload
	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		return nil, errors.ErrInvalidDateRange
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

	// Log the activity
	if s.activityLogger != nil {
		changes := make(map[string]any)
		if req.Title != nil {
			changes["title"] = *req.Title
		}
		if req.StartDate != nil {
			changes["start_date"] = *req.StartDate
		}
		if req.EndDate != nil {
			changes["end_date"] = *req.EndDate
		}
		if req.Completed != nil {
			changes["completed"] = *req.Completed
		}
		if task.Category != nil {
			changes["category"] = task.Category.Name
		}
		if req.Journal != nil {
			changes["journal_updated"] = true
		}
		if req.EmotionID != nil {
			changes["emotion_updated"] = true
		}

		// Detect completion changes
		if oldTask != nil && task.Completed != oldTask.Completed {
			changes["previous_completed"] = oldTask.Completed
			changes["new_completed"] = task.Completed
		}

		title := task.Title
		if err := s.activityLogger.Log(ctx, "task", id, title, "", "updated", userID, changes); err != nil {
			s.logger.Warn().Err(err).Str("task_id", id).Msg("failed to log task updated activity")
		}
	}

	return task, nil
}

// =============================================================================
// DELETE
// =============================================================================

// Delete soft-deletes a task.
func (s *service) Delete(ctx context.Context, id, userID string) error {
	// Get task info before deletion for logging
	task, _ := s.repo.FindByID(ctx, id, userID)

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

	// Log the activity
	if s.activityLogger != nil && task != nil {
		changes := map[string]any{
			"title": task.Title,
		}
		if task.Category != nil {
			changes["category"] = task.Category.Name
		}
		if err := s.activityLogger.Log(ctx, "task", id, task.Title, "", "deleted", userID, changes); err != nil {
			s.logger.Warn().Err(err).Str("task_id", id).Msg("failed to log task deleted activity")
		}
	}

	return nil
}

// =============================================================================
// GET LAST TASK END TIME
// =============================================================================

// GetLastTaskEndTime retrieves the end time of the most recently finished task.
func (s *service) GetLastTaskEndTime(ctx context.Context, userID string) (*time.Time, error) {
	endTime, err := s.repo.GetLastTaskEndTime(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get last task end time")
		return nil, err
	}
	return endTime, nil
}

func validateOptionalDate(field string, value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, errors.ErrBadRequest.WithMessage(field + " cannot be empty")
	}
	parsed, err := timeutil.ParseDateTime(trimmed)
	if err != nil {
		return nil, errors.ErrBadRequest.WithMessage("Invalid " + field + " format")
	}
	*value = trimmed
	return &parsed, nil
}
