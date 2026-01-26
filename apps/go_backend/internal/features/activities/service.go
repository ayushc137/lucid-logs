package activities

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/features/goals"
	"github.com/lucid-logs/go-backend/internal/features/tasks"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the activity business logic interface.
type Service interface {
	// List retrieves paginated activities for a user.
	List(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Activity], error)

	// Get retrieves a single activity by ID.
	Get(ctx context.Context, id, userID string) (*Activity, error)

	// GetPinned retrieves pinned activities for quick bar.
	GetPinned(ctx context.Context, userID string) ([]*Activity, error)

	// Create creates a new activity.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Activity, error)

	// Update updates an existing activity.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Activity, error)

	// Delete soft-deletes an activity.
	Delete(ctx context.Context, id, userID string) error

	// InstantLog creates a completed task from the activity.
	InstantLog(ctx context.Context, id string, req *InstantLogRequest, userID string) (*InstantLogResponse, error)

	// Schedule returns pre-filled task data for the task form.
	Schedule(ctx context.Context, id string, req *ScheduleRequest, userID string) (*ScheduleResponse, error)

	// GetLinkedGoals retrieves goals linked to an activity.
	GetLinkedGoals(ctx context.Context, id, userID string) ([]ActivityGoalLinkDetail, error)

	// LinkGoal links an activity to a goal.
	LinkGoal(ctx context.Context, activityID string, link *GoalLinkInput, userID string) error

	// UnlinkGoal removes a goal link.
	UnlinkGoal(ctx context.Context, activityID, goalID, userID string) error
}

// =============================================================================
// DEPENDENCIES
// =============================================================================

// TaskCreator creates tasks (to avoid circular imports).
type TaskCreator interface {
	Create(ctx context.Context, req *tasks.CreateRequest, userID string) (*tasks.Task, error)
}

// GoalProgressUpdater updates goal progress.
type GoalProgressUpdater interface {
	GetByID(ctx context.Context, id, userID string) (*goals.Goal, error)
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo        Repository
	taskCreator TaskCreator
	goalGetter  GoalProgressUpdater
	logger      zerolog.Logger
}

// NewService creates a new activity Service.
func NewService(repo Repository, taskCreator TaskCreator, goalGetter GoalProgressUpdater) Service {
	return &service{
		repo:        repo,
		taskCreator: taskCreator,
		goalGetter:  goalGetter,
		logger:      log.With().Str("service", "activities").Logger(),
	}
}

// List retrieves paginated activities for a user.
func (s *service) List(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Activity], error) {
	result, err := s.repo.FindPaginated(ctx, userID, params)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to list activities")
		return nil, err
	}
	return result, nil
}

// Get retrieves a single activity by ID.
func (s *service) Get(ctx context.Context, id, userID string) (*Activity, error) {
	activity, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("id", id).Msg("failed to get activity")
		return nil, err
	}
	return activity, nil
}

// GetPinned retrieves pinned activities for quick bar.
func (s *service) GetPinned(ctx context.Context, userID string) ([]*Activity, error) {
	activities, err := s.repo.FindPinned(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get pinned activities")
		return nil, err
	}
	return activities, nil
}

// Create creates a new activity.
func (s *service) Create(ctx context.Context, req *CreateRequest, userID string) (*Activity, error) {
	activity, err := s.repo.Create(ctx, req, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to create activity")
		return nil, err
	}

	s.logger.Info().
		Str("activity_id", activity.ID).
		Str("user_id", userID).
		Msg("activity created")

	return activity, nil
}

// Update updates an existing activity.
func (s *service) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Activity, error) {
	// Verify activity exists and belongs to user
	_, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	activity, err := s.repo.Update(ctx, id, req, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("id", id).Msg("failed to update activity")
		return nil, err
	}

	s.logger.Info().
		Str("activity_id", id).
		Str("user_id", userID).
		Msg("activity updated")

	return activity, nil
}

// Delete soft-deletes an activity.
func (s *service) Delete(ctx context.Context, id, userID string) error {
	// Verify activity exists and belongs to user
	_, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		return err
	}

	err = s.repo.Delete(ctx, id, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("id", id).Msg("failed to delete activity")
		return err
	}

	s.logger.Info().
		Str("activity_id", id).
		Str("user_id", userID).
		Msg("activity deleted")

	return nil
}

// InstantLog creates a completed task from the activity.
func (s *service) InstantLog(ctx context.Context, id string, req *InstantLogRequest, userID string) (*InstantLogResponse, error) {
	// Get activity with linked goals
	activity, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Determine timestamp
	now := time.Now()
	if req.Timestamp != nil {
		parsed, err := time.Parse(time.RFC3339, *req.Timestamp)
		if err == nil {
			now = parsed
		}
	}

	// Calculate end time based on default duration
	duration := activity.DefaultDuration
	if duration == 0 {
		duration = 60 // Default 1 minute for instant logs
	}
	endTime := now.Add(time.Duration(duration) * time.Second)

	// Determine quantity
	quantity := activity.QuantityDefault
	if req.Quantity != nil {
		quantity = *req.Quantity
	}

	// Build journal/notes
	journal := activity.Description
	if req.Notes != "" {
		if journal != "" {
			journal = journal + "\n\n" + req.Notes
		} else {
			journal = req.Notes
		}
	}

	// Build goal links from activity
	var goalLinks []tasks.GoalLinkInput
	for _, gl := range activity.Goals {
		if gl.AutoLinkTasks {
			// Calculate quantity for this goal
			// Priority: request quantity > goal-link default_quantity > activity default * multiplier
			var goalQuantity float64
			if req.Quantity != nil {
				// User specified quantity in request - apply multiplier
				goalQuantity = *req.Quantity * gl.QuantityMultiplier
			} else if gl.DefaultQuantity != nil {
				// Use per-goal default quantity
				goalQuantity = *gl.DefaultQuantity
			} else {
				// Fall back to activity default * multiplier
				goalQuantity = activity.QuantityDefault * gl.QuantityMultiplier
			}

			impact := gl.DefaultImpact
			if impact == "" {
				impact = activity.DefaultImpact
			}
			if impact == "" {
				impact = ImpactPositive
			}

			goalLinks = append(goalLinks, tasks.GoalLinkInput{
				GoalID:        gl.GoalID,
				ImpactType:    impact,
				QuantityValue: goalQuantity,
			})
		}
	}

	// Create the task
	taskReq := &tasks.CreateRequest{
		Title:     activity.Title,
		Journal:   journal,
		StartDate: now.Format(time.RFC3339),
		EndDate:   endTime.Format(time.RFC3339),
		Source:    "activity",
		Completed: activity.DefaultCompleted,
		GoalLinks: goalLinks,
	}

	// Set category if available
	if activity.Category != nil {
		taskReq.CategoryID = activity.Category.ID
	}

	// Set emotion if available
	if activity.DefaultEmotionID != "" {
		taskReq.EmotionID = &activity.DefaultEmotionID
	}

	// Set quantity if enabled
	if activity.QuantityEnabled {
		taskReq.Quantity = &tasks.QuantityInput{
			Value:  quantity,
			UnitID: activity.QuantityUnitID,
		}
	}

	task, err := s.taskCreator.Create(ctx, taskReq, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("activity_id", id).Msg("failed to create task from activity")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	// Update activity usage stats
	if err := s.repo.IncrementUseCount(ctx, id); err != nil {
		s.logger.Warn().Err(err).Str("activity_id", id).Msg("failed to increment use count")
	}

	// Build goal update summaries
	var goalsUpdated []GoalUpdateSummary
	for _, gl := range activity.Goals {
		if gl.AutoLinkTasks && gl.Goal != nil {
			// Calculate quantity the same way as for goal links
			var goalQuantity float64
			if req.Quantity != nil {
				goalQuantity = *req.Quantity * gl.QuantityMultiplier
			} else if gl.DefaultQuantity != nil {
				goalQuantity = *gl.DefaultQuantity
			} else {
				goalQuantity = activity.QuantityDefault * gl.QuantityMultiplier
			}
			summary := GoalUpdateSummary{
				GoalID:     gl.GoalID,
				GoalTitle:  gl.Goal.Title,
				GoalIcon:   gl.Goal.Icon,
				ValueAdded: goalQuantity,
			}
			if gl.Goal.Target != nil {
				summary.TargetValue = gl.Goal.Target.Value
			}
			goalsUpdated = append(goalsUpdated, summary)
		}
	}

	s.logger.Info().
		Str("activity_id", id).
		Str("task_id", task.ID).
		Str("user_id", userID).
		Int("goals_linked", len(goalLinks)).
		Msg("instant log created")

	return &InstantLogResponse{
		TaskID:       task.ID,
		TaskTitle:    task.Title,
		GoalsUpdated: goalsUpdated,
	}, nil
}

// Schedule returns pre-filled task data for the task form.
func (s *service) Schedule(ctx context.Context, id string, req *ScheduleRequest, userID string) (*ScheduleResponse, error) {
	// Get activity with linked goals
	activity, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Build task defaults
	defaults := TaskDefaults{
		Title:    activity.Title,
		Journal:  activity.Description,
		Duration: activity.DefaultDuration,
		Priority: activity.DefaultPriority,
	}

	if activity.Category != nil {
		defaults.CategoryID = activity.Category.ID
	}
	if activity.DefaultEmotionID != "" {
		defaults.EmotionID = activity.DefaultEmotionID
	}
	if activity.QuantityEnabled {
		defaults.Quantity = activity.QuantityDefault
		defaults.QuantityUnit = activity.QuantityUnitID
	}

	// Build goal link defaults
	var goalLinks []GoalLinkDefault
	for _, gl := range activity.Goals {
		if gl.AutoLinkTasks && gl.Goal != nil {
			link := GoalLinkDefault{
				GoalID:       gl.GoalID,
				GoalTitle:    gl.Goal.Title,
				GoalIcon:     gl.Goal.Icon,
				ImpactType:   gl.DefaultImpact,
				QuantityStep: activity.QuantityStep,
			}
			// Use per-goal default quantity if set, otherwise fall back to activity default * multiplier
			if gl.DefaultQuantity != nil {
				link.DefaultQuantity = gl.DefaultQuantity
			} else if activity.QuantityEnabled {
				qty := activity.QuantityDefault * gl.QuantityMultiplier
				link.DefaultQuantity = &qty
			}
			// Unit always comes from goal's target
			if gl.Goal.Target != nil {
				link.QuantityUnit = gl.Goal.Target.UnitID
			}
			goalLinks = append(goalLinks, link)
		}
	}

	return &ScheduleResponse{
		Activity:     activity,
		TaskDefaults: defaults,
		GoalLinks:    goalLinks,
	}, nil
}

// GetLinkedGoals retrieves goals linked to an activity.
func (s *service) GetLinkedGoals(ctx context.Context, id, userID string) ([]ActivityGoalLinkDetail, error) {
	// Verify activity exists and belongs to user
	_, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	goals, err := s.repo.FindLinkedGoals(ctx, id, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("id", id).Msg("failed to get linked goals")
		return nil, err
	}

	return goals, nil
}

// LinkGoal links an activity to a goal.
func (s *service) LinkGoal(ctx context.Context, activityID string, link *GoalLinkInput, userID string) error {
	// Verify activity exists and belongs to user
	_, err := s.repo.FindByID(ctx, activityID, userID)
	if err != nil {
		return err
	}

	// Verify goal exists and belongs to user
	if s.goalGetter != nil {
		_, err = s.goalGetter.GetByID(ctx, link.GoalID, userID)
		if err != nil {
			return errors.ErrNotFound.WithMessage("Goal not found")
		}
	}

	err = s.repo.LinkGoal(ctx, activityID, link)
	if err != nil {
		s.logger.Error().Err(err).
			Str("activity_id", activityID).
			Str("goal_id", link.GoalID).
			Msg("failed to link goal")
		return err
	}

	s.logger.Info().
		Str("activity_id", activityID).
		Str("goal_id", link.GoalID).
		Msg("goal linked to activity")

	return nil
}

// UnlinkGoal removes a goal link.
func (s *service) UnlinkGoal(ctx context.Context, activityID, goalID, userID string) error {
	// Verify activity exists and belongs to user
	_, err := s.repo.FindByID(ctx, activityID, userID)
	if err != nil {
		return err
	}

	err = s.repo.UnlinkGoal(ctx, activityID, goalID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("activity_id", activityID).
			Str("goal_id", goalID).
			Msg("failed to unlink goal")
		return err
	}

	s.logger.Info().
		Str("activity_id", activityID).
		Str("goal_id", goalID).
		Msg("goal unlinked from activity")

	return nil
}

// =============================================================================
// GOAL ACTIVITY CREATOR
// =============================================================================

// GoalActivityCreator implements goals.ActivityCreator interface.
// This allows goals service to auto-create activities without circular imports.
type GoalActivityCreator struct {
	repo Repository
}

// NewGoalActivityCreator creates a new GoalActivityCreator.
func NewGoalActivityCreator(repo Repository) *GoalActivityCreator {
	return &GoalActivityCreator{repo: repo}
}

// CreateForGoal creates an activity linked to a goal.
func (c *GoalActivityCreator) CreateForGoal(ctx context.Context, goal *goals.Goal, pinned bool, defaultDuration int, userID string) (string, error) {
	// Infer icon from goal or use default
	icon := goal.Icon
	if icon == "" {
		icon = "⚡"
	}

	// Build activity from goal
	req := &CreateRequest{
		Title:            goal.Title,
		Icon:             icon,
		Description:      goal.Description,
		DefaultDuration:  defaultDuration,
		DefaultPriority:  3,
		DefaultCompleted: true, // Instant log by default
		DefaultImpact:    ImpactPositive,
		Pinned:           pinned,
	}

	// Set quantity from goal target if available
	var defaultQuantity *float64
	if goal.Target != nil {
		req.QuantityEnabled = true
		req.QuantityStep = 1
		// Set per-goal default quantity (not activity-level)
		qty := 1.0
		defaultQuantity = &qty
	}

	// Always link to the goal with per-goal default quantity
	req.GoalLinks = []GoalLinkInput{{
		GoalID:             goal.ID,
		AutoLinkTasks:      true,
		QuantityMultiplier: 1.0,
		DefaultQuantity:    defaultQuantity,
		DefaultImpact:      ImpactPositive,
	}}

	// Set category from goal if available
	if goal.Category != nil {
		req.CategoryID = goal.Category.ID
	}

	activity, err := c.repo.Create(ctx, req, userID)
	if err != nil {
		return "", err
	}

	log.Info().
		Str("activity_id", activity.ID).
		Str("goal_id", goal.ID).
		Msg("activity auto-created for goal")

	return activity.ID, nil
}
