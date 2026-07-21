// Package scheduler provides in-process scheduling for automated tasks.
//
// This package implements:
//   - User-configurable daily retrospective generation
//   - Per-minute checks for users whose retro time has arrived
//   - Integration with retrospectives service
//
// Uses go-co-op/gocron v2 for robust, timezone-aware scheduling.
package scheduler

import (
	"context"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/features/retrospectives"
	"github.com/lucid-logs/go-backend/internal/features/users"
)

// =============================================================================
// SCHEDULER
// =============================================================================

// Scheduler manages scheduled tasks for the application.
type Scheduler struct {
	scheduler  gocron.Scheduler
	retroSvc   retrospectives.Service
	userRepo   users.Repository
	logger     zerolog.Logger
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// Config holds scheduler configuration.
type Config struct {
	RetroService retrospectives.Service
	UserRepo     users.Repository
}

// New creates a new Scheduler.
func New(cfg Config) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		scheduler:  s,
		retroSvc:   cfg.RetroService,
		userRepo:   cfg.UserRepo,
		logger:     log.With().Str("component", "scheduler").Logger(),
		ctx:        ctx,
		cancelFunc: cancel,
	}, nil
}

// =============================================================================
// LIFECYCLE
// =============================================================================

// Start begins the scheduler and registers all jobs.
func (s *Scheduler) Start() error {
	s.logger.Info().Msg("starting scheduler...")

	// Job 1: Check for users whose retro time has arrived (every minute)
	_, err := s.scheduler.NewJob(
		gocron.DurationJob(1*time.Minute),
		gocron.NewTask(s.checkAndGenerateRetros),
		gocron.WithName("check_retro_times"),
	)
	if err != nil {
		return err
	}

	// Job 2: Daily cleanup/maintenance at 3 AM UTC
	_, err = s.scheduler.NewJob(
		gocron.CronJob("0 3 * * *", false), // 3:00 AM daily
		gocron.NewTask(s.runDailyMaintenance),
		gocron.WithName("daily_maintenance"),
	)
	if err != nil {
		return err
	}

	s.scheduler.Start()
	s.logger.Info().Msg("scheduler started")

	return nil
}

// Stop gracefully shuts down the scheduler.
func (s *Scheduler) Stop() error {
	s.logger.Info().Msg("stopping scheduler...")
	s.cancelFunc()
	return s.scheduler.Shutdown()
}

// =============================================================================
// RETRO GENERATION JOB
// =============================================================================

// checkAndGenerateRetros runs every minute to check for users whose
// configured retro time has arrived, and generates their daily retro.
func (s *Scheduler) checkAndGenerateRetros() {
	ctx, cancel := context.WithTimeout(s.ctx, 55*time.Second)
	defer cancel()

	now := time.Now()

	s.logger.Debug().Time("now", now).Msg("checking for retro generation")

	// Get all users with retro settings enabled
	usersForRetro, err := s.userRepo.GetUsersForRetroGeneration(ctx, now)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get users for retro generation")
		return
	}

	if len(usersForRetro) == 0 {
		return
	}

	s.logger.Info().Int("count", len(usersForRetro)).Msg("users due for retro generation")

	for _, user := range usersForRetro {
		s.generateRetroForUser(ctx, user.ID, user.Preferences.Timezone, now)
	}
}

func (s *Scheduler) generateRetroForUser(ctx context.Context, userID, timezone string, serverNow time.Time) {
	// Load user's timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		s.logger.Warn().Err(err).Str("timezone", timezone).Msg("invalid timezone, using UTC")
		loc = time.UTC
	}

	// Convert to user's local time for the retro date
	userNow := serverNow.In(loc)

	// --- Daily retro ---
	if err := s.maybeGenerateDaily(ctx, userID, userNow); err != nil {
		// errors are already logged; continue to weekly/monthly
	}

	// --- Weekly retro ---
	if err := s.maybeGenerateWeekly(ctx, userID, userNow); err != nil {
		// logged inside
	}

	// --- Monthly retro ---
	if err := s.maybeGenerateMonthly(ctx, userID, userNow); err != nil {
		// logged inside
	}
}

// maybeGenerateDaily generates the daily retro if it doesn't already exist.
func (s *Scheduler) maybeGenerateDaily(ctx context.Context, userID string, userNow time.Time) error {
	exists, err := s.retroSvc.ExistsForPeriod(ctx, userID, retrospectives.RetroTypeDaily)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("check daily retro exists failed")
		return err
	}
	if exists {
		s.logger.Debug().Str("user_id", userID).Msg("daily retro already exists")
		return nil
	}

	if _, err := s.retroSvc.GenerateDaily(ctx, userID, userNow); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("generate daily retro failed")
		return err
	}

	s.logger.Info().Str("user_id", userID).Str("type", "daily").Msg("retro generated")
	return nil
}

// maybeGenerateWeekly generates a weekly retro if today matches the user's
// configured WeeklyRetroDay (in their timezone) and one doesn't already exist
// for this week.
func (s *Scheduler) maybeGenerateWeekly(ctx context.Context, userID string, userNow time.Time) error {
	prefs, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("fetch user for weekly retro failed")
		return err
	}

	weeklyDay := prefs.Preferences.WeeklyRetroDay
	if weeklyDay == "" {
		return nil // not configured
	}

	if !strings.EqualFold(userNow.Weekday().String(), weeklyDay) {
		return nil // not today
	}

	exists, err := s.retroSvc.ExistsForPeriod(ctx, userID, retrospectives.RetroTypeWeekly)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("check weekly retro exists failed")
		return err
	}
	if exists {
		s.logger.Debug().Str("user_id", userID).Msg("weekly retro already exists")
		return nil
	}

	if _, err := s.retroSvc.GenerateWeekly(ctx, userID, userNow); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("generate weekly retro failed")
		return err
	}

	s.logger.Info().Str("user_id", userID).Str("type", "weekly").Msg("retro generated")
	return nil
}

// maybeGenerateMonthly generates a monthly retro if today's day-of-month
// matches the user's configured MonthlyRetroDay and one doesn't already exist
// for this month.
func (s *Scheduler) maybeGenerateMonthly(ctx context.Context, userID string, userNow time.Time) error {
	prefs, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("fetch user for monthly retro failed")
		return err
	}

	monthlyDay := prefs.Preferences.MonthlyRetroDay
	if monthlyDay <= 0 || monthlyDay > 31 {
		return nil // not configured
	}

	if userNow.Day() != monthlyDay {
		return nil // not today
	}

	exists, err := s.retroSvc.ExistsForPeriod(ctx, userID, retrospectives.RetroTypeMonthly)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("check monthly retro exists failed")
		return err
	}
	if exists {
		s.logger.Debug().Str("user_id", userID).Msg("monthly retro already exists")
		return nil
	}

	if _, err := s.retroSvc.GenerateMonthly(ctx, userID, userNow); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("generate monthly retro failed")
		return err
	}

	s.logger.Info().Str("user_id", userID).Str("type", "monthly").Msg("retro generated")
	return nil
}

// =============================================================================
// MAINTENANCE JOB
// =============================================================================

// runDailyMaintenance performs daily cleanup and maintenance tasks.
func (s *Scheduler) runDailyMaintenance() {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	s.logger.Info().Msg("running daily maintenance...")

	// Future: Add maintenance tasks
	// - Aggregate daily stats to agg_daily table
	// - Clean up old temporary data
	// - Update streak calculations

	_ = ctx // Placeholder for actual work

	s.logger.Info().Msg("daily maintenance completed")
}
