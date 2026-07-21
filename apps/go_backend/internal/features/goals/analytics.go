// Package goals provides goal management functionality.
// This file contains analytics-related types and repository methods for goal progress tracking.

package goals

import (
	"context"
	"time"

	"github.com/lucid-logs/go-backend/internal/shared/database"
)

// =============================================================================
// ANALYTICS MODELS
// =============================================================================

// DailyStats represents pre-computed daily progress for a goal.
// Stored in goal_daily_stats table for O(1) lookups.
//
// @Description Daily progress statistics for a goal
type DailyStats struct {
	GoalID            string    `json:"goal_id"`
	Date              time.Time `json:"date"`
	DailyValue        float64   `json:"daily_value"`
	CumulativeValue   float64   `json:"cumulative_value"`
	ContributionCount int       `json:"contribution_count"`
	TargetValue       *float64  `json:"target_value,omitempty"`
	StreakAtDate      int       `json:"streak_at_date"`
	Status            string    `json:"status"` // "pending", "met", "missed", "exceeded"
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// DailyProgressResponse is the response for GET /goals/:id/daily-progress
//
// @Description Daily progress data for a goal over a date range
type DailyProgressResponse struct {
	GoalID          string       `json:"goal_id"`
	TargetPerPeriod float64      `json:"target_per_period"`
	Data            []DailyStats `json:"data"`
}

// =============================================================================
// DATABASE MODELS
// =============================================================================

type dailyStatsDB struct {
	GoalID            string    `json:"goal_id"`
	Date              time.Time `json:"date"`
	DailyValue        float64   `json:"daily_value"`
	CumulativeValue   float64   `json:"cumulative_value"`
	ContributionCount int       `json:"contribution_count"`
	TargetValue       *float64  `json:"target_value,omitempty"`
	StreakAtDate      int       `json:"streak_at_date"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (d *dailyStatsDB) toDailyStats() *DailyStats {
	return &DailyStats{
		GoalID:            d.GoalID,
		Date:              d.Date,
		DailyValue:        d.DailyValue,
		CumulativeValue:   d.CumulativeValue,
		ContributionCount: d.ContributionCount,
		TargetValue:       d.TargetValue,
		StreakAtDate:      d.StreakAtDate,
		Status:            d.Status,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
}

// =============================================================================
// ANALYTICS REPOSITORY METHODS
// =============================================================================

// GetDailyProgress retrieves daily progress stats for a goal within a date range.
// This reads from the pre-computed goal_daily_stats table for O(1) lookups.
func (r *repository) GetDailyProgress(ctx context.Context, goalID, userID string, startDate, endDate time.Time) ([]DailyStats, error) {
	gID := resolveGoalID(goalID)

	statsDB, err := database.QueryAll[dailyStatsDB](ctx, r.db, `
		SELECT
			goal_id,
			date,
			daily_value,
			cumulative_value,
			contribution_count,
			target_value,
			streak_at_date,
			status,
			created_at,
			updated_at
		FROM goal_daily_stats
		WHERE goal_id = $goal_id
		  AND date >= $start_date
		  AND date <= $end_date
		ORDER BY date ASC
	`, map[string]any{
		"goal_id":    gID,
		"start_date": startDate.UTC().Format(time.RFC3339Nano),
		"end_date":   endDate.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", goalID).Msg("failed to fetch daily progress")
		return nil, err
	}

	stats := make([]DailyStats, len(statsDB))
	for i := range statsDB {
		stats[i] = *statsDB[i].toDailyStats()
	}

	return stats, nil
}

// BackfillDailyStats populates goal_daily_stats from existing task_goals data.
// This is a one-time migration helper to populate historical data.
func (r *repository) BackfillDailyStats(ctx context.Context, userID string) error {
	// Aggregate existing task_goals data by goal and date, then UPSERT into
	// goal_daily_stats. The aggregation matches each task to its goal using
	// the task_goals join table and buckets by the task's completion date
	// (falling back to start_date when completed_at is missing).
	_, err := r.db.SQL().ExecContext(ctx, `
		INSERT INTO goal_daily_stats
			(goal_id, date, created_by, daily_value, contribution_count,
			 target_value, cumulative_value, streak_at_date, status,
			 created_at, updated_at)
		SELECT
			tg.goal_id AS goal_id,
			date(COALESCE(t.completed_at, t.start_date)) AS date,
			g.created_by AS created_by,
			COALESCE(SUM(tg.quantity_value), 0) AS daily_value,
			COUNT(*) AS contribution_count,
			json_extract(g.target, '$.value') AS target_value,
			0 AS cumulative_value,
			g.current_streak AS streak_at_date,
			CASE
				WHEN json_extract(g.target, '$.value') IS NOT NULL
				  AND COALESCE(SUM(tg.quantity_value), 0) >= json_extract(g.target, '$.value')
				THEN 'met'
				ELSE 'pending'
			END AS status,
			strftime('%Y-%m-%dT%H:%M:%fZ', 'now') AS created_at,
			strftime('%Y-%m-%dT%H:%M:%fZ', 'now') AS updated_at
		FROM task_goals tg
		JOIN goals g ON g.id = tg.goal_id
		JOIN tasks t ON t.id = tg.task_id
		WHERE g.created_by = ?
		  AND g.deleted_at IS NULL
		GROUP BY tg.goal_id, date
		ON CONFLICT(goal_id, date) DO UPDATE SET
			daily_value = excluded.daily_value,
			contribution_count = excluded.contribution_count,
			target_value = excluded.target_value,
			status = excluded.status,
			updated_at = excluded.updated_at
	`, userID)

	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("failed to backfill daily stats")
		return err
	}

	r.logger.Info().Str("user_id", userID).Msg("daily stats backfill completed")
	return nil
}

// RecalculateDailyStats recalculates stats for a specific goal and date.
// Useful for manual correction or after bulk updates.
func (r *repository) RecalculateDailyStats(ctx context.Context, goalID string, date time.Time) error {
	gID := resolveGoalID(goalID)
	dateStr := date.UTC().Truncate(24 * time.Hour).Format("2006-01-02")
	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Aggregate contributions for the given goal/date from task_goals joined
	// to tasks (respects track_completed_only when set on the goal's target).
	_, err := r.db.SQL().ExecContext(ctx, `
		INSERT INTO goal_daily_stats
			(goal_id, date, created_by, daily_value, contribution_count,
			 target_value, cumulative_value, streak_at_date, status,
			 created_at, updated_at)
		SELECT
			g.id AS goal_id,
			? AS date,
			g.created_by AS created_by,
			COALESCE(SUM(tg.quantity_value), 0) AS daily_value,
			COUNT(*) AS contribution_count,
			json_extract(g.target, '$.value') AS target_value,
			0 AS cumulative_value,
			g.current_streak AS streak_at_date,
			CASE
				WHEN json_extract(g.target, '$.value') IS NOT NULL
				  AND COALESCE(SUM(tg.quantity_value), 0) >= json_extract(g.target, '$.value')
				THEN 'met'
				ELSE 'pending'
			END AS status,
			? AS created_at,
			? AS updated_at
		FROM goals g
		LEFT JOIN task_goals tg ON tg.goal_id = g.id
		LEFT JOIN tasks t ON t.id = tg.task_id AND t.deleted_at IS NULL
		WHERE g.id = ?
		  AND (
			json_extract(g.target, '$.track_completed_only') IS NULL
			OR json_extract(g.target, '$.track_completed_only') != 1
			OR t.completed = 1
		  )
		  AND date(COALESCE(t.completed_at, t.start_date)) = ?
		GROUP BY g.id
		ON CONFLICT(goal_id, date) DO UPDATE SET
			daily_value = excluded.daily_value,
			contribution_count = excluded.contribution_count,
			target_value = excluded.target_value,
			status = excluded.status,
			updated_at = excluded.updated_at
	`, dateStr, now, now, gID, dateStr)

	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", goalID).Time("date", date).Msg("failed to recalculate daily stats")
		return err
	}

	return nil
}
