package goals

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

// Service defines the goal business logic interface.
//
// The service layer:
//   - Validates business rules
//   - Orchestrates repository calls
//   - Manages linked template creation
//   - Handles cross-cutting concerns (logging, etc.)
type Service interface {
	// List retrieves paginated goals for a user with optional filters.
	List(ctx context.Context, userID string, params pagination.Params, filters GoalFilters) (*pagination.Response[*Goal], error)

	// Get retrieves a single goal by ID.
	Get(ctx context.Context, id, userID string) (*Goal, error)

	// GetByActivityKey retrieves a goal by its activity key.
	GetByActivityKey(ctx context.Context, activityKey, userID string) (*Goal, error)

	// Create creates a new goal and optionally auto-creates a linked template.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Goal, error)

	// Update updates an existing goal.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Goal, error)

	// Delete soft-deletes a goal.
	Delete(ctx context.Context, id, userID string) error

	// GetTodayGoals retrieves recurring goals with today's completion status.
	GetTodayGoals(ctx context.Context, userID string) (*TodayGoalsResponse, error)

	// UpdateStreak updates streak information after a goal entry.
	UpdateStreak(ctx context.Context, goalID, userID string, met bool) error

	// SetLinkedTemplate updates the linked template ID.
	SetLinkedTemplate(ctx context.Context, goalID, templateID, userID string) error
}

// TemplateCreator is an interface for creating templates.
// This allows the goals service to create linked templates without circular imports.
type TemplateCreator interface {
	CreateForGoal(ctx context.Context, goal *Goal, userID string) (templateID string, err error)
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo            Repository
	templateCreator TemplateCreator
	logger          zerolog.Logger
}

// NewService creates a new goal Service.
// templateCreator can be nil if template auto-creation is not needed.
func NewService(repo Repository, templateCreator TemplateCreator) Service {
	return &service{
		repo:            repo,
		templateCreator: templateCreator,
		logger:          log.With().Str("service", "goals").Logger(),
	}
}

// =============================================================================
// LIST
// =============================================================================

func (s *service) List(ctx context.Context, userID string, params pagination.Params, filters GoalFilters) (*pagination.Response[*Goal], error) {
	goals, total, err := s.repo.FindPaginated(ctx, userID, params, filters)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to list goals")
		return nil, err
	}

	s.logger.Debug().
		Str("user_id", userID).
		Int("count", len(goals)).
		Int64("total", total).
		Msg("goals listed")

	resp := pagination.NewResponse(goals, total, params)
	return &resp, nil
}

// =============================================================================
// GET
// =============================================================================

func (s *service) Get(ctx context.Context, id, userID string) (*Goal, error) {
	goal, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("goal_id", id).Msg("failed to get goal")
		return nil, err
	}

	return goal, nil
}

func (s *service) GetByActivityKey(ctx context.Context, activityKey, userID string) (*Goal, error) {
	goal, err := s.repo.FindByActivityKey(ctx, activityKey, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("activity_key", activityKey).Msg("failed to get goal by activity key")
		return nil, err
	}

	return goal, nil
}

// =============================================================================
// CREATE
// =============================================================================

func (s *service) Create(ctx context.Context, req *CreateRequest, userID string) (*Goal, error) {
	// Parse and validate dates if provided
	if req.StartDate != nil {
		if _, err := timeutil.ParseDateTime(*req.StartDate); err != nil {
			return nil, errors.ErrBadRequest.WithMessage("Invalid start_date format")
		}
	}
	if req.Deadline != nil {
		if _, err := timeutil.ParseDateTime(*req.Deadline); err != nil {
			return nil, errors.ErrBadRequest.WithMessage("Invalid deadline format")
		}
	}

	// Validate goal type and required fields
	if req.GoalType == GoalTypeMeasurable && req.Target == nil {
		return nil, errors.ErrBadRequest.WithMessage("Target is required for measurable goals")
	}

	// Create the goal
	goal, err := s.repo.Create(ctx, req, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to create goal")
		return nil, err
	}

	s.logger.Info().
		Str("goal_id", goal.ID).
		Str("activity_key", goal.ActivityKey).
		Str("user_id", userID).
		Msg("goal created")

	// Auto-create linked template for recurring goals
	if goal.Recurrence != nil && s.templateCreator != nil {
		templateID, err := s.templateCreator.CreateForGoal(ctx, goal, userID)
		if err != nil {
			s.logger.Warn().Err(err).Str("goal_id", goal.ID).Msg("failed to auto-create template")
			// Don't fail goal creation for template creation failure
		} else {
			// Update goal with linked template
			if err := s.repo.UpdateLinkedTemplate(ctx, goal.ID, templateID, userID); err != nil {
				s.logger.Warn().Err(err).Str("goal_id", goal.ID).Msg("failed to link template to goal")
			} else {
				goal.LinkedTemplate = &templateID
			}
		}
	}

	return goal, nil
}

// =============================================================================
// UPDATE
// =============================================================================

func (s *service) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Goal, error) {
	// Parse and validate dates if provided
	if req.StartDate != nil {
		if _, err := timeutil.ParseDateTime(*req.StartDate); err != nil {
			return nil, errors.ErrBadRequest.WithMessage("Invalid start_date format")
		}
	}
	if req.Deadline != nil {
		if _, err := timeutil.ParseDateTime(*req.Deadline); err != nil {
			return nil, errors.ErrBadRequest.WithMessage("Invalid deadline format")
		}
	}

	goal, err := s.repo.Update(ctx, id, req, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("goal_id", id).Msg("failed to update goal")
		return nil, err
	}

	s.logger.Info().
		Str("goal_id", id).
		Str("user_id", userID).
		Msg("goal updated")

	return goal, nil
}

// =============================================================================
// DELETE
// =============================================================================

func (s *service) Delete(ctx context.Context, id, userID string) error {
	err := s.repo.Delete(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("goal_id", id).Msg("failed to delete goal")
		return err
	}

	s.logger.Info().
		Str("goal_id", id).
		Str("user_id", userID).
		Msg("goal deleted")

	return nil
}

// =============================================================================
// TODAY'S GOALS
// =============================================================================

func (s *service) GetTodayGoals(ctx context.Context, userID string) (*TodayGoalsResponse, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	goals, err := s.repo.FindRecurringForDate(ctx, userID, today)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get today's goals")
		return nil, err
	}

	todayGoals := make([]*TodayGoal, len(goals))
	for i, goal := range goals {
		// Check if goal should be active on this day of week
		if goal.Recurrence != nil && len(goal.Recurrence.ActiveDays) > 0 {
			dayName := strings.ToLower(today.Weekday().String()[:3])
			isActive := false
			for _, d := range goal.Recurrence.ActiveDays {
				if d == dayName {
					isActive = true
					break
				}
			}
			if !isActive {
				continue // Skip goals not active today
			}
		}

		todayGoals[i] = &TodayGoal{
			Goal:     goal,
			TodayMet: false, // Will be populated by goal entries lookup
			Streak:   goal.CurrentStreak,
		}
	}

	return &TodayGoalsResponse{
		Date:  today.Format("2006-01-02"),
		Goals: todayGoals,
	}, nil
}

// =============================================================================
// STREAK MANAGEMENT
// =============================================================================

func (s *service) UpdateStreak(ctx context.Context, goalID, userID string, met bool) error {
	goal, err := s.repo.FindByID(ctx, goalID, userID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	var newStreak, newLongest int
	var lastCompleted *time.Time

	if met {
		// Increment streak
		newStreak = goal.CurrentStreak + 1
		if newStreak > goal.LongestStreak {
			newLongest = newStreak
		} else {
			newLongest = goal.LongestStreak
		}
		lastCompleted = &now
	} else {
		// Check grace days
		if goal.Recurrence != nil && goal.GraceDaysUsed < goal.Recurrence.GraceDays {
			// Use a grace day, streak continues
			newStreak = goal.CurrentStreak
			newLongest = goal.LongestStreak
		} else {
			// Streak broken
			newStreak = 0
			newLongest = goal.LongestStreak
		}
		lastCompleted = goal.LastCompletedDate
	}

	return s.repo.UpdateStreak(ctx, goalID, newStreak, newLongest, lastCompleted, userID)
}

func (s *service) SetLinkedTemplate(ctx context.Context, goalID, templateID, userID string) error {
	return s.repo.UpdateLinkedTemplate(ctx, goalID, templateID, userID)
}
