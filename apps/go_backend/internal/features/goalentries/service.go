package goalentries

import (
	"context"
	"time"

	"github.com/lucid-logs/go-backend/internal/features/goals"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the goal entry business logic interface.
type Service interface {
	// Get retrieves an entry by ID.
	Get(ctx context.Context, id string) (*GoalEntry, error)

	// GetForGoalAndDate retrieves an entry for a specific goal and date.
	GetForGoalAndDate(ctx context.Context, goalID, userID string, date time.Time) (*GoalEntry, error)

	// List retrieves entries for a goal within a date range.
	List(ctx context.Context, goalID, userID string, startDate, endDate time.Time) (*GoalEntryListResponse, error)

	// Create creates or updates an entry for a goal on a specific date.
	Create(ctx context.Context, goalID, userID string, req *CreateRequest) (*GoalEntry, error)

	// Update updates an existing entry.
	Update(ctx context.Context, id, goalID, userID string, req *UpdateRequest) (*GoalEntry, error)

	// LogTaskContribution records a task as contributing to a goal entry.
	LogTaskContribution(ctx context.Context, goalID, taskID, userID string, value *float64) error
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo     Repository
	goalRepo goals.Repository
	goalSvc  goals.Service
	logger   zerolog.Logger
}

// NewService creates a new goal entry Service.
func NewService(repo Repository, goalRepo goals.Repository, goalSvc goals.Service) Service {
	return &service{
		repo:     repo,
		goalRepo: goalRepo,
		goalSvc:  goalSvc,
		logger:   log.With().Str("service", "goalentries").Logger(),
	}
}

// =============================================================================
// GET
// =============================================================================

func (s *service) Get(ctx context.Context, id string) (*GoalEntry, error) {
	entry, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("entry_id", id).Msg("failed to get entry")
		return nil, err
	}

	return entry, nil
}

func (s *service) GetForGoalAndDate(ctx context.Context, goalID, userID string, date time.Time) (*GoalEntry, error) {
	// Verify user owns the goal
	_, err := s.goalRepo.FindByID(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	entry, err := s.repo.FindByGoalAndDate(ctx, goalID, date)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		return nil, err
	}

	return entry, nil
}

// =============================================================================
// LIST
// =============================================================================

func (s *service) List(ctx context.Context, goalID, userID string, startDate, endDate time.Time) (*GoalEntryListResponse, error) {
	// Verify user owns the goal
	_, err := s.goalRepo.FindByID(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	entries, err := s.repo.FindByGoalInRange(ctx, goalID, startDate, endDate)
	if err != nil {
		s.logger.Error().Err(err).Str("goal_id", goalID).Msg("failed to list entries")
		return nil, err
	}

	// Calculate summary
	summary := &EntrySummary{
		TotalDays: len(entries),
	}
	for _, e := range entries {
		if e.Met {
			summary.MetDays++
		} else {
			summary.MissedDays++
		}
		if e.Value != nil {
			summary.TotalValue += *e.Value
		}
	}
	if summary.TotalDays > 0 {
		summary.SuccessRate = float64(summary.MetDays) / float64(summary.TotalDays) * 100
	}

	return &GoalEntryListResponse{
		GoalID:  goalID,
		Entries: entries,
		Summary: summary,
	}, nil
}

// =============================================================================
// CREATE
// =============================================================================

func (s *service) Create(ctx context.Context, goalID, userID string, req *CreateRequest) (*GoalEntry, error) {
	// Verify user owns the goal
	goal, err := s.goalRepo.FindByID(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, errors.ErrBadRequest.WithMessage("Invalid date format, use YYYY-MM-DD")
	}

	// Check if entry already exists for this date
	existing, _ := s.repo.FindByGoalAndDate(ctx, goalID, date)
	if existing != nil {
		// Update existing entry instead of creating duplicate
		updateReq := &UpdateRequest{
			Value: req.Value,
			Met:   &req.Met,
		}
		if req.Notes != "" {
			updateReq.Notes = &req.Notes
		}
		if req.TaskIDs != nil {
			updateReq.TaskIDs = req.TaskIDs
		}
		return s.repo.Update(ctx, existing.ID, updateReq)
	}

	// Create new entry
	entry, err := s.repo.Create(ctx, goalID, req)
	if err != nil {
		s.logger.Error().Err(err).Str("goal_id", goalID).Msg("failed to create entry")
		return nil, err
	}

	// Update streak if entry is met
	if req.Met {
		if err := s.goalSvc.UpdateStreak(ctx, goalID, userID, true); err != nil {
			s.logger.Warn().Err(err).Str("goal_id", goalID).Msg("failed to update streak")
		}
	}

	s.logger.Info().
		Str("entry_id", entry.ID).
		Str("goal_id", goalID).
		Bool("met", req.Met).
		Msg("entry created")

	// Log goal progress if measurable
	if goal.Target != nil && req.Value != nil {
		s.logger.Debug().
			Str("goal_id", goalID).
			Float64("value", *req.Value).
			Msg("progress logged for measurable goal")
	}

	return entry, nil
}

// =============================================================================
// UPDATE
// =============================================================================

func (s *service) Update(ctx context.Context, id, goalID, userID string, req *UpdateRequest) (*GoalEntry, error) {
	// Verify user owns the goal
	_, err := s.goalRepo.FindByID(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	entry, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("entry_id", id).Msg("failed to update entry")
		return nil, err
	}

	s.logger.Info().Str("entry_id", id).Msg("entry updated")

	return entry, nil
}

// =============================================================================
// TASK CONTRIBUTION
// =============================================================================

// LogTaskContribution is called when a task with activity_key is created.
// It creates or updates the goal entry for today with the task contribution.
func (s *service) LogTaskContribution(ctx context.Context, goalID, taskID, userID string, value *float64) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	dateStr := today.Format("2006-01-02")

	// Check if entry exists for today
	existing, _ := s.repo.FindByGoalAndDate(ctx, goalID, today)

	if existing != nil {
		// Add task to existing entry
		if err := s.repo.AddTaskToEntry(ctx, existing.ID, taskID); err != nil {
			return err
		}

		// Update value if provided
		if value != nil {
			newValue := *value
			if existing.Value != nil {
				newValue = *existing.Value + *value
			}
			_, err := s.repo.Update(ctx, existing.ID, &UpdateRequest{
				Value: &newValue,
			})
			if err != nil {
				return err
			}
		}
	} else {
		// Create new entry for today
		_, err := s.repo.Create(ctx, goalID, &CreateRequest{
			Date:    dateStr,
			Value:   value,
			Met:     false, // Will be updated when target is met
			TaskIDs: []string{taskID},
		})
		if err != nil {
			return err
		}
	}

	s.logger.Debug().
		Str("goal_id", goalID).
		Str("task_id", taskID).
		Msg("task contribution logged")

	return nil
}
