// Package goals provides goal business logic.
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
//   - Computes goal stats on read
//   - Handles cross-cutting concerns (logging, etc.)
type Service interface {
	// List retrieves paginated goals for a user with optional filters.
	List(ctx context.Context, userID string, params pagination.Params, filters GoalFilters) (*pagination.Response[*Goal], error)

	// Get retrieves a single goal by ID with computed stats.
	Get(ctx context.Context, id, userID string) (*Goal, error)

	// Create creates a new goal.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Goal, error)

	// Update updates an existing goal.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Goal, error)

	// Delete soft-deletes a goal.
	Delete(ctx context.Context, id, userID string) error

	// GetTodayGoals retrieves recurring goals with today's completion status.
	GetTodayGoals(ctx context.Context, userID string) (*TodayGoalsResponse, error)

	// GetChildren retrieves child goals for a grouped goal.
	GetChildren(ctx context.Context, goalID, userID string) ([]*Goal, error)

	// AddChild adds a child goal to a parent (grouped goal).
	AddChild(ctx context.Context, parentID string, req *AddChildRequest, userID string) error

	// RemoveChild removes a child goal from a parent.
	RemoveChild(ctx context.Context, parentID, childID, userID string) error

	// UpdateCategory updates the category for a goal.
	UpdateCategory(ctx context.Context, goalID, categoryID, userID string) error

	// RecordCompletion updates the denormalized streak fields when a goal/habit is completed.
	// This should be called when a goal entry is marked as met or when a task contributes to a goal.
	RecordCompletion(ctx context.Context, goalID, userID string, completedDate time.Time) error
}

// TemplateCreator is an interface for creating templates.
// This allows the goals service to create linked templates without circular imports.
type TemplateCreator interface {
	CreateForGoal(ctx context.Context, goal *Goal, userID string) (templateID string, err error)
}

// GoalLogger is an interface for logging goal events.
// This allows the goals service to log events without circular imports with goallogs.
type GoalLogger interface {
	LogEvent(ctx context.Context, goalID, event, userID string, changes map[string]any, stats *GoalStats) error
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo            Repository
	templateCreator TemplateCreator
	goalLogger      GoalLogger
	logger          zerolog.Logger
}

// NewService creates a new goal Service.
// templateCreator and goalLogger can be nil if not needed.
func NewService(repo Repository, templateCreator TemplateCreator, goalLogger GoalLogger) Service {
	return &service{
		repo:            repo,
		templateCreator: templateCreator,
		goalLogger:      goalLogger,
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

	// Compute stats for each goal

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

	// Compute stats

	// Get children if this is a grouped goal
	children, err := s.repo.FindChildren(ctx, id, userID)
	if err == nil && len(children) > 0 {
		goal.Children = children
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

	// Validate target operator if provided
	if req.Target != nil && req.Target.Operator != "" {
		validOperator := false
		for _, op := range ValidOperators {
			if req.Target.Operator == op {
				validOperator = true
				break
			}
		}
		if !validOperator {
			return nil, errors.ErrBadRequest.WithMessage("Invalid target operator. Must be gte, lte, or eq")
		}
	}

	// Create the goal
	goal, err := s.repo.Create(ctx, req, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to create goal")
		return nil, err
	}

	s.logger.Info().
		Str("goal_id", goal.ID).
		Str("user_id", userID).
		Bool("is_habit", goal.Recurrence != nil).
		Bool("is_measurable", goal.Target != nil).
		Msg("goal created")

	// Auto-create linked template for recurring goals
	if goal.Recurrence != nil && s.templateCreator != nil {
		_, err := s.templateCreator.CreateForGoal(ctx, goal, userID)
		if err != nil {
			s.logger.Warn().Err(err).Str("goal_id", goal.ID).Msg("failed to auto-create template")
			// Don't fail goal creation for template creation failure
		}
	}

	// Log the created event
	if s.goalLogger != nil {
		changes := map[string]any{
			"title":       goal.Title,
			"description": goal.Description,
			"icon":        goal.Icon,
			"status":      goal.Status,
			"priority":    goal.Priority,
		}
		if goal.Recurrence != nil {
			changes["recurrence"] = map[string]any{
				"frequency": goal.Recurrence.Frequency,
				"period":    goal.Recurrence.Period,
			}
		}
		if goal.Target != nil {
			changes["target"] = map[string]any{
				"value":    goal.Target.Value,
				"operator": goal.Target.Operator,
				"unit_id":  goal.Target.UnitID,
			}
		}
		if err := s.goalLogger.LogEvent(ctx, goal.ID, "created", userID, changes, nil); err != nil {
			s.logger.Warn().Err(err).Str("goal_id", goal.ID).Msg("failed to log goal created event")
		}
	}

	return goal, nil
}

// =============================================================================
// UPDATE
// =============================================================================

func (s *service) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Goal, error) {
	// Get current state for change detection
	oldGoal, _ := s.repo.FindByID(ctx, id, userID)

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

	// Validate status if provided
	if req.Status != nil {
		validStatus := false
		for _, s := range ValidStatuses {
			if *req.Status == s {
				validStatus = true
				break
			}
		}
		if !validStatus {
			return nil, errors.ErrBadRequest.WithMessage("Invalid status. Must be active, completed, or archived")
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

	// Log the updated event
	if s.goalLogger != nil {
		// Determine event type based on status change
		eventType := "updated"
		changes := make(map[string]any)

		// Detect status changes for special event types
		if oldGoal != nil && req.Status != nil && *req.Status != oldGoal.Status {
			switch *req.Status {
			case StatusCompleted:
				eventType = "completed"
			case StatusArchived:
				eventType = "archived"
			case StatusActive:
				if oldGoal.Status == StatusCompleted || oldGoal.Status == StatusArchived {
					eventType = "reactivated"
				}
			}
			changes["previous_status"] = oldGoal.Status
			changes["new_status"] = *req.Status
		}

		// Record what was changed
		if req.Title != nil {
			changes["title"] = *req.Title
		}
		if req.Description != nil {
			changes["description"] = *req.Description
		}
		if req.Icon != nil {
			changes["icon"] = *req.Icon
		}
		if req.Priority != nil {
			changes["priority"] = *req.Priority
		}
		if req.Status != nil {
			changes["status"] = *req.Status
		}
		if req.Target != nil {
			changes["target"] = map[string]any{
				"value":    req.Target.Value,
				"operator": req.Target.Operator,
			}
		}
		if req.Recurrence != nil {
			changes["recurrence"] = map[string]any{
				"frequency": req.Recurrence.Frequency,
				"period":    req.Recurrence.Period,
			}
		}

		if err := s.goalLogger.LogEvent(ctx, id, eventType, userID, changes, nil); err != nil {
			s.logger.Warn().Err(err).Str("goal_id", id).Msg("failed to log goal updated event")
		}
	}

	return goal, nil
}

// =============================================================================
// DELETE
// =============================================================================

func (s *service) Delete(ctx context.Context, id, userID string) error {
	// Get goal info before deletion for logging
	goal, _ := s.repo.FindByID(ctx, id, userID)

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

	// Log the deleted event
	if s.goalLogger != nil && goal != nil {
		changes := map[string]any{
			"title":  goal.Title,
			"status": goal.Status,
		}
		if err := s.goalLogger.LogEvent(ctx, id, "deleted", userID, changes, nil); err != nil {
			s.logger.Warn().Err(err).Str("goal_id", id).Msg("failed to log goal deleted event")
		}
	}

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

	todayGoals := make([]*TodayGoal, 0, len(goals))
	for _, goal := range goals {
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

		// Compute stats to get today's status
		stats, err := s.repo.ComputeStats(ctx, goal.ID, userID)

		todayGoal := &TodayGoal{
			Goal:     goal,
			TodayMet: false,
			Streak:   0,
		}

		if err == nil && stats != nil {
			todayGoal.TodayMet = stats.TodayStatus == TodayStatusMet
			todayGoal.TodayValue = &stats.CurrentValue
			todayGoal.Streak = stats.CurrentStreak
		}

		todayGoals = append(todayGoals, todayGoal)
	}

	return &TodayGoalsResponse{
		Date:  today.Format("2006-01-02"),
		Goals: todayGoals,
	}, nil
}

// =============================================================================
// CHILD GOAL MANAGEMENT
// =============================================================================

func (s *service) GetChildren(ctx context.Context, goalID, userID string) ([]*Goal, error) {
	children, err := s.repo.FindChildren(ctx, goalID, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("goal_id", goalID).Msg("failed to get child goals")
		return nil, err
	}

	// Compute stats for each child
	for _, child := range children {
		stats, err := s.repo.ComputeStats(ctx, child.ID, userID)
		if err == nil {
			child.Stats = stats
		}
	}

	return children, nil
}

func (s *service) AddChild(ctx context.Context, parentID string, req *AddChildRequest, userID string) error {
	// Verify parent exists
	_, err := s.repo.FindByID(ctx, parentID, userID)
	if err != nil {
		return err
	}

	// Verify child exists
	_, err = s.repo.FindByID(ctx, req.ChildGoalID, userID)
	if err != nil {
		return err
	}

	required := true
	if req.Required != nil {
		required = *req.Required
	}

	if err := s.repo.AddChild(ctx, parentID, req.ChildGoalID, userID, req.Order, required); err != nil {
		s.logger.Error().Err(err).Str("parent", parentID).Str("child", req.ChildGoalID).Msg("failed to add child")
		return err
	}

	s.logger.Info().
		Str("parent_id", parentID).
		Str("child_id", req.ChildGoalID).
		Msg("child goal added")

	return nil
}

func (s *service) RemoveChild(ctx context.Context, parentID, childID, userID string) error {
	if err := s.repo.RemoveChild(ctx, parentID, childID, userID); err != nil {
		s.logger.Error().Err(err).Str("parent", parentID).Str("child", childID).Msg("failed to remove child")
		return err
	}

	s.logger.Info().
		Str("parent_id", parentID).
		Str("child_id", childID).
		Msg("child goal removed")

	return nil
}

// =============================================================================
// CATEGORY MANAGEMENT
// =============================================================================

func (s *service) UpdateCategory(ctx context.Context, goalID, categoryID, userID string) error {
	if err := s.repo.UpdateCategory(ctx, goalID, categoryID, userID); err != nil {
		s.logger.Error().Err(err).Str("goal", goalID).Str("category", categoryID).Msg("failed to update category")
		return err
	}

	s.logger.Info().
		Str("goal_id", goalID).
		Str("category_id", categoryID).
		Msg("goal category updated")

	return nil
}

// =============================================================================
// STREAK MANAGEMENT (Denormalized/Materialized Fields)
// =============================================================================

// RecordCompletion updates the denormalized streak fields when a goal/habit is completed.
// This is the "materialized view" pattern - we compute streaks on write rather than read.
// It also automatically logs the completion event.
func (s *service) RecordCompletion(ctx context.Context, goalID, userID string, completedDate time.Time) error {
	goal, err := s.repo.FindByID(ctx, goalID, userID)
	if err != nil {
		return err
	}

	// Normalize to date only (no time component)
	completedDate = completedDate.Truncate(24 * time.Hour)

	// Calculate new streak values
	var oldStreak, newStreak, longestStreak int
	var lastCompletedDate *time.Time

	if goal.Stats != nil {
		oldStreak = goal.Stats.CurrentStreak
		newStreak = goal.Stats.CurrentStreak
		longestStreak = goal.Stats.LongestStreak
		lastCompletedDate = goal.Stats.LastCompletedDate
	}

	// Determine the expected previous completion date based on recurrence
	var expectedPrevDate time.Time
	if goal.Recurrence != nil {
		switch goal.Recurrence.Period {
		case "day":
			expectedPrevDate = completedDate.AddDate(0, 0, -goal.Recurrence.Frequency)
		case "week":
			expectedPrevDate = completedDate.AddDate(0, 0, -7*goal.Recurrence.Frequency)
		case "month":
			expectedPrevDate = completedDate.AddDate(0, -goal.Recurrence.Frequency, 0)
		default:
			expectedPrevDate = completedDate.AddDate(0, 0, -1) // Default to daily
		}

		// Add grace days if configured
		if goal.Recurrence.GraceDays > 0 {
			expectedPrevDate = expectedPrevDate.AddDate(0, 0, -goal.Recurrence.GraceDays)
		}
	} else {
		// Non-recurring goal - just count completions
		expectedPrevDate = completedDate.AddDate(0, 0, -1)
	}

	// Check if this continues the streak or starts a new one
	streakBroken := false
	if lastCompletedDate != nil {
		lastDate := lastCompletedDate.Truncate(24 * time.Hour)

		// Same day - don't increment
		if lastDate.Equal(completedDate) {
			// Already completed today, no streak change
			return nil
		}

		// Check if within expected window (continues streak)
		if lastDate.After(expectedPrevDate) || lastDate.Equal(expectedPrevDate) {
			newStreak++
		} else {
			// Gap too large - streak broken, start new
			newStreak = 1
			streakBroken = oldStreak > 0
		}
	} else {
		// First completion ever
		newStreak = 1
	}

	// Update longest streak if current beats it
	if newStreak > longestStreak {
		longestStreak = newStreak
	}

	// Persist the denormalized streak fields
	if err := s.repo.UpdateStreaks(ctx, goalID, newStreak, longestStreak, &completedDate); err != nil {
		s.logger.Error().Err(err).Str("goal_id", goalID).Msg("failed to update streaks")
		return err
	}

	s.logger.Info().
		Str("goal_id", goalID).
		Int("current_streak", newStreak).
		Int("longest_streak", longestStreak).
		Time("completed_date", completedDate).
		Msg("goal completion recorded, streaks updated")

	// Auto-log the event if goal logger is configured
	if s.goalLogger != nil {
		// Determine event type
		eventType := "streak_updated"
		changes := map[string]any{
			"previous_streak": oldStreak,
			"current_streak":  newStreak,
			"completed_date":  completedDate.Format("2006-01-02"),
		}

		// Log streak_broken as a separate event first
		if streakBroken {
			brokenChanges := map[string]any{
				"previous_streak": oldStreak,
				"days_missed":     0, // Can be computed if needed
			}
			if err := s.goalLogger.LogEvent(ctx, goalID, "streak_broken", userID, brokenChanges, nil); err != nil {
				s.logger.Warn().Err(err).Str("goal_id", goalID).Msg("failed to log streak_broken event")
			}
		}

		// Check if target was met
		if goal.Target != nil && goal.Stats != nil {
			targetMet := false
			switch goal.Target.Operator {
			case OperatorGTE:
				targetMet = goal.Stats.CurrentValue >= goal.Target.Value
			case OperatorLTE:
				targetMet = goal.Stats.CurrentValue <= goal.Target.Value
			case OperatorEQ:
				targetMet = goal.Stats.CurrentValue == goal.Target.Value
			}
			if targetMet {
				eventType = "target_met"
				changes["target_value"] = goal.Target.Value
				changes["current_value"] = goal.Stats.CurrentValue
			}
		}

		// Create goal stats for snapshot
		stats := &GoalStats{
			CurrentStreak:     newStreak,
			LongestStreak:     longestStreak,
			LastCompletedDate: &completedDate,
		}
		if goal.Stats != nil {
			stats.CurrentValue = goal.Stats.CurrentValue
			stats.ProgressPercent = goal.Stats.ProgressPercent
			stats.TotalContributions = goal.Stats.TotalContributions
		}

		if err := s.goalLogger.LogEvent(ctx, goalID, eventType, userID, changes, stats); err != nil {
			// Log error but don't fail the operation
			s.logger.Warn().Err(err).Str("goal_id", goalID).Str("event", eventType).Msg("failed to log goal event")
		}
	}

	return nil
}
