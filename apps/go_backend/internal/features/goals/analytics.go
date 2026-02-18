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
	GoalID            string               `json:"goal_id"`
	Date              database.SurrealTime `json:"date"`
	DailyValue        float64              `json:"daily_value"`
	CumulativeValue   float64              `json:"cumulative_value"`
	ContributionCount int                  `json:"contribution_count"`
	TargetValue       *float64             `json:"target_value,omitempty"`
	StreakAtDate      int                  `json:"streak_at_date"`
	Status            string               `json:"status"`
	CreatedAt         database.SurrealTime `json:"created_at"`
	UpdatedAt         database.SurrealTime `json:"updated_at"`
}

func (d *dailyStatsDB) toDailyStats() *DailyStats {
	return &DailyStats{
		GoalID:            d.GoalID,
		Date:              d.Date.Time,
		DailyValue:        d.DailyValue,
		CumulativeValue:   d.CumulativeValue,
		ContributionCount: d.ContributionCount,
		TargetValue:       d.TargetValue,
		StreakAtDate:      d.StreakAtDate,
		Status:            d.Status,
		CreatedAt:         d.CreatedAt.Time,
		UpdatedAt:         d.UpdatedAt.Time,
	}
}

// =============================================================================
// ANALYTICS REPOSITORY METHODS
// =============================================================================

// GetDailyProgress retrieves daily progress stats for a goal within a date range.
// This reads from the pre-computed goal_daily_stats table for O(1) lookups.
func (r *repository) GetDailyProgress(ctx context.Context, goalID, userID string, startDate, endDate time.Time) ([]DailyStats, error) {
	gID := database.MustRecordID(Table, goalID)

	statsDB, err := database.QueryAll[dailyStatsDB](ctx, r.db, `
		SELECT 
			type::string(goal_id) as goal_id,
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
		"start_date": startDate,
		"end_date":   endDate,
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
	_, err := database.QueryAll[any](ctx, r.db, `
		-- Aggregate existing task_goals data by goal and date
		LET $aggregated = (
			SELECT
				out AS goal_id,
				out.created_by AS created_by,
				time::floor(
					IF in.completed_at IS NOT NONE 
					THEN in.completed_at 
					ELSE in.start_date END,
					1d
				) AS date,
				math::sum(quantity_value) AS daily_value,
				count() AS contribution_count,
				out.target.value AS target_value
			FROM task_goals
			WHERE out.created_by = <record>$user_id
			  AND out.deleted_at IS NONE
			GROUP BY goal_id, date
		);
		
		-- Insert aggregated data into goal_daily_stats
		FOR $row IN $aggregated {
			UPSERT goal_daily_stats SET
				goal_id = $row.goal_id,
				date = $row.date,
				created_by = $row.created_by,
				daily_value = $row.daily_value,
				contribution_count = $row.contribution_count,
				target_value = $row.target_value,
				status = IF $row.target_value IS NOT NONE AND $row.daily_value >= $row.target_value 
					THEN "met" 
					ELSE "pending" 
				END,
				updated_at = time::now()
			WHERE goal_id = $row.goal_id AND date = $row.date;
		};
	`, map[string]any{
		"user_id": userID,
	})

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
	gID := database.MustRecordID(Table, goalID)
	truncatedDate := date.Truncate(24 * time.Hour)

	_, err := database.QueryAll[any](ctx, r.db, `
		LET $goal = (SELECT * FROM $goal_id)[0];
		LET $stats = (
			SELECT
				math::sum(quantity_value) AS total,
				count() AS count
			FROM <-task_goals
			WHERE in.deleted_at IS NONE
			  AND ($goal.target.track_completed_only IS NOT TRUE OR in.completed = true)
			  AND time::floor(
					IF in.completed_at IS NOT NONE 
					THEN in.completed_at 
					ELSE in.start_date END,
					1d
				) = $date
			GROUP ALL
		)[0];
		
		UPSERT goal_daily_stats SET
			goal_id = $goal_id,
			date = $date,
			created_by = $goal.created_by,
			daily_value = $stats.total ?? 0,
			contribution_count = $stats.count ?? 0,
			target_value = $goal.target.value,
			status = IF $goal.target.value IS NOT NONE AND ($stats.total ?? 0) >= $goal.target.value 
				THEN "met" 
				ELSE "pending" 
			END,
			updated_at = time::now()
		WHERE goal_id = $goal_id AND date = $date;
	`, map[string]any{
		"goal_id": gID,
		"date":    truncatedDate,
	})

	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", goalID).Time("date", date).Msg("failed to recalculate daily stats")
		return err
	}

	return nil
}
