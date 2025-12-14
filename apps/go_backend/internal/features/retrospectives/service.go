package retrospectives

import (
	"context"
	"time"

	"github.com/lucid-logs/go-backend/internal/features/analytics"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the retrospective business logic interface.
type Service interface {
	// List retrieves retrospectives with pagination.
	List(ctx context.Context, userID string, params pagination.Params) (*ListResponse, error)

	// Get retrieves a single retrospective by ID.
	Get(ctx context.Context, id, userID string) (*Retrospective, error)

	// Generate creates a new retrospective with auto-computed summary.
	Generate(ctx context.Context, userID string, req *GenerateRequest) (*Retrospective, error)

	// GenerateDaily generates a daily retrospective (called by scheduler).
	GenerateDaily(ctx context.Context, userID string, date time.Time) (*Retrospective, error)

	// Update updates user content on a retrospective.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Retrospective, error)

	// Delete removes a retrospective.
	Delete(ctx context.Context, id, userID string) error

	// ExistsForToday checks if a retro exists for today.
	ExistsForToday(ctx context.Context, userID string) (bool, error)
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo          Repository
	analyticsRepo analytics.Repository
	logger        zerolog.Logger
}

// NewService creates a new retrospectives Service.
func NewService(repo Repository, analyticsRepo analytics.Repository) Service {
	return &service{
		repo:          repo,
		analyticsRepo: analyticsRepo,
		logger:        log.With().Str("service", "retrospectives").Logger(),
	}
}

// =============================================================================
// LIST
// =============================================================================

func (s *service) List(ctx context.Context, userID string, params pagination.Params) (*ListResponse, error) {
	retros, total, err := s.repo.FindPaginated(ctx, userID, params.Limit, params.Offset)
	if err != nil {
		s.logger.Error().Err(err).Msg("list retrospectives failed")
		return nil, err
	}

	return &ListResponse{
		Retrospectives: retros,
		Total:          total,
		Limit:          params.Limit,
		Offset:         params.Offset,
	}, nil
}

// =============================================================================
// GET
// =============================================================================

func (s *service) Get(ctx context.Context, id, userID string) (*Retrospective, error) {
	retro, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("retro_id", id).Msg("get retrospective failed")
		return nil, err
	}
	return retro, nil
}

// =============================================================================
// GENERATE
// =============================================================================

func (s *service) Generate(ctx context.Context, userID string, req *GenerateRequest) (*Retrospective, error) {
	start, end := s.resolveDateRange(req)

	// Check if already exists
	existing, _ := s.repo.FindByDateRange(ctx, userID, req.RetroType, start, end)
	if existing != nil {
		s.logger.Info().Str("retro_id", existing.ID).Msg("retrospective already exists")
		return existing, nil
	}

	// Generate auto-summary
	autoSummary, err := s.computeAutoSummary(ctx, userID, start, end)
	if err != nil {
		s.logger.Error().Err(err).Msg("compute auto summary failed")
		// Continue with empty summary rather than failing
		autoSummary = RetroAutoSummary{}
	}

	retro := &Retrospective{
		CreatedBy:   userID,
		RetroType:   req.RetroType,
		StartDate:   start,
		EndDate:     end,
		AutoSummary: autoSummary,
		UserContent: UserReflection{},
		Status:      StatusDraft,
		GeneratedAt: time.Now().UTC(),
	}

	created, err := s.repo.Create(ctx, retro)
	if err != nil {
		s.logger.Error().Err(err).Msg("create retrospective failed")
		return nil, err
	}

	s.logger.Info().
		Str("retro_id", created.ID).
		Str("type", req.RetroType).
		Time("start", start).
		Time("end", end).
		Msg("retrospective generated")

	return created, nil
}

// GenerateDaily is called by the scheduler for automatic daily retros.
func (s *service) GenerateDaily(ctx context.Context, userID string, date time.Time) (*Retrospective, error) {
	return s.Generate(ctx, userID, &GenerateRequest{
		RetroType: RetroTypeDaily,
		Date:      &date,
	})
}

// =============================================================================
// UPDATE
// =============================================================================

func (s *service) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Retrospective, error) {
	retro, err := s.repo.Update(ctx, id, req, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("retro_id", id).Msg("update retrospective failed")
		return nil, err
	}

	s.logger.Info().Str("retro_id", id).Msg("retrospective updated")
	return retro, nil
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
		s.logger.Error().Err(err).Str("retro_id", id).Msg("delete retrospective failed")
		return err
	}

	s.logger.Info().Str("retro_id", id).Msg("retrospective deleted")
	return nil
}

// =============================================================================
// EXISTS CHECK (for scheduler)
// =============================================================================

func (s *service) ExistsForToday(ctx context.Context, userID string) (bool, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour).Add(-time.Second)

	existing, err := s.repo.FindByDateRange(ctx, userID, RetroTypeDaily, start, end)
	if err != nil {
		return false, err
	}

	return existing != nil, nil
}

// =============================================================================
// AUTO-SUMMARY COMPUTATION
// =============================================================================

func (s *service) computeAutoSummary(ctx context.Context, userID string, start, end time.Time) (RetroAutoSummary, error) {
	summary := RetroAutoSummary{}

	// Get task metrics
	taskMetrics, err := s.analyticsRepo.GetTaskMetrics(ctx, userID, start, end)
	if err != nil {
		s.logger.Warn().Err(err).Msg("get task metrics for retro failed")
	} else {
		summary.Tasks = TasksSummary{
			Completed:          taskMetrics.CompletedTasks,
			Postponed:          taskMetrics.PostponedTasks,
			Cancelled:          taskMetrics.AbandonedTasks,
			TotalDurationHours: taskMetrics.TotalDurationHours,
		}
		// Add category breakdown
		for _, cat := range taskMetrics.ByCategory {
			summary.Tasks.ByCategory = append(summary.Tasks.ByCategory, CategoryCount{
				Category:      cat.CategoryName,
				Count:         cat.TaskCount,
				DurationHours: cat.Hours,
			})
		}
	}

	// Get emotion metrics
	emotionMetrics, err := s.analyticsRepo.GetEmotionMetrics(ctx, userID, start, end)
	if err != nil {
		s.logger.Warn().Err(err).Msg("get emotion metrics for retro failed")
	} else {
		summary.Mood = MoodSummary{
			AverageValence:   emotionMetrics.AverageValence,
			AverageArousal:   emotionMetrics.AverageArousal,
			DominantQuadrant: emotionMetrics.DominantQuadrant,
			QuadrantDistribution: QuadrantDist{
				Yellow: emotionMetrics.QuadrantDistribution.Yellow,
				Green:  emotionMetrics.QuadrantDistribution.Green,
				Red:    emotionMetrics.QuadrantDistribution.Red,
				Blue:   emotionMetrics.QuadrantDistribution.Blue,
			},
		}
	}

	// Get goal metrics
	goalMetrics, err := s.analyticsRepo.GetGoalMetrics(ctx, userID)
	if err != nil {
		s.logger.Warn().Err(err).Msg("get goal metrics for retro failed")
	} else {
		// Convert to GoalImpact format
		for _, gp := range goalMetrics.GoalProgress {
			summary.Goals.NetImpact = append(summary.Goals.NetImpact, GoalImpact{
				GoalID: gp.GoalID,
				Name:   gp.GoalTitle,
			})
		}
	}

	// Get category metrics
	categoryMetrics, err := s.analyticsRepo.GetCategoryMetrics(ctx, userID, start, end)
	if err != nil {
		s.logger.Warn().Err(err).Msg("get category metrics for retro failed")
	} else {
		for _, cat := range categoryMetrics.Distribution {
			summary.Categories.TimeDistribution = append(summary.Categories.TimeDistribution, CategoryTime{
				Category:   cat.CategoryName,
				Hours:      cat.Hours,
				Percentage: cat.Percentage,
			})
		}
	}

	return summary, nil
}

// =============================================================================
// HELPERS
// =============================================================================

func (s *service) resolveDateRange(req *GenerateRequest) (time.Time, time.Time) {
	now := time.Now().UTC()

	switch req.RetroType {
	case RetroTypeDaily:
		date := now
		if req.Date != nil {
			date = *req.Date
		}
		start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		end := start.Add(24 * time.Hour).Add(-time.Second)
		return start, end

	case RetroTypeWeekly:
		// Start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
		end := start.Add(7 * 24 * time.Hour).Add(-time.Second)
		return start, end

	case RetroTypeMonthly:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0).Add(-time.Second)
		return start, end

	case RetroTypeQuarterly:
		quarter := (int(now.Month()) - 1) / 3
		startMonth := time.Month(quarter*3 + 1)
		start := time.Date(now.Year(), startMonth, 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 3, 0).Add(-time.Second)
		return start, end

	case RetroTypeYearly:
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)
		return start, end

	case RetroTypeCustom:
		if req.StartDate != nil && req.EndDate != nil {
			return *req.StartDate, *req.EndDate
		}
		fallthrough

	default:
		// Default to today
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		end := start.Add(24 * time.Hour).Add(-time.Second)
		return start, end
	}
}
