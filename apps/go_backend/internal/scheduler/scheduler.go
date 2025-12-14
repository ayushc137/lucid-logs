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
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/lucid-logs/go-backend/internal/features/retrospectives"
	"github.com/lucid-logs/go-backend/internal/features/users"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	// Check if retro already exists for today
	exists, err := s.retroSvc.ExistsForToday(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("check retro exists failed")
		return
	}

	if exists {
		s.logger.Debug().Str("user_id", userID).Msg("retro already exists for today")
		return
	}

	// Generate the retro
	_, err = s.retroSvc.GenerateDaily(ctx, userID, userNow)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("generate daily retro failed")
		return
	}

	s.logger.Info().
		Str("user_id", userID).
		Str("timezone", timezone).
		Msg("daily retro generated")
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
