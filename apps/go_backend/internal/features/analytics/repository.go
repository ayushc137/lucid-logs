package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the analytics data access interface.
type Repository interface {
	// GetTaskMetrics retrieves task productivity metrics for a period.
	GetTaskMetrics(ctx context.Context, userID string, start, end time.Time) (*TaskMetrics, error)

	// GetEmotionMetrics retrieves emotion/mood metrics for a period.
	GetEmotionMetrics(ctx context.Context, userID string, start, end time.Time) (*EmotionMetrics, error)

	// GetGoalMetrics retrieves goal progress and streak metrics.
	GetGoalMetrics(ctx context.Context, userID string) (*GoalMetrics, error)

	// GetCategoryMetrics retrieves category time distribution.
	GetCategoryMetrics(ctx context.Context, userID string, start, end time.Time) (*CategoryMetrics, error)

	// GetTimeSeriesData retrieves time series data for a metric.
	GetTimeSeriesData(ctx context.Context, userID string, metric, groupBy string, start, end time.Time) (*TimeSeriesData, error)

	// GetQuadrantDistribution retrieves emotion quadrant distribution.
	GetQuadrantDistribution(ctx context.Context, userID string, start, end time.Time) (*PieChartData, error)

	// GetProductivityHeatmap retrieves activity heatmap data.
	GetProductivityHeatmap(ctx context.Context, userID string, start, end time.Time) (*HeatmapData, error)
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new analytics Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "analytics").Logger(),
	}
}

// =============================================================================
// TASK METRICS
// =============================================================================

type taskMetricsDB struct {
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Postponed int     `json:"postponed"`
	Abandoned int     `json:"abandoned"`
	TotalSecs float64 `json:"total_secs"`
}

func (r *repository) GetTaskMetrics(ctx context.Context, userID string, start, end time.Time) (*TaskMetrics, error) {
	// Query task counts and duration
	result, err := database.QueryFirst[taskMetricsDB](ctx, r.db, `
		SELECT 
			count() as total,
			count(IF completed = true THEN 1 ELSE NONE END) as completed,
			count(IF status = "postponed" THEN 1 ELSE NONE END) as postponed,
			count(IF status = "abandoned" THEN 1 ELSE NONE END) as abandoned,
			math::sum(duration::secs(end_date - start_date)) as total_secs
		FROM tasks
		WHERE created_by = $user 
		  AND start_date >= $start 
		  AND start_date <= $end
		  AND deleted_at IS NONE
		GROUP ALL
	`, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if err != nil {
		r.logger.Error().Err(err).Msg("get task metrics failed")
		return nil, err
	}

	if result == nil {
		return &TaskMetrics{}, nil
	}

	days := end.Sub(start).Hours() / 24
	if days < 1 {
		days = 1
	}

	completionRate := 0.0
	denominator := result.Completed + result.Postponed + result.Abandoned
	if denominator > 0 {
		completionRate = float64(result.Completed) / float64(denominator) * 100
	}

	avgDuration := 0.0
	if result.Completed > 0 {
		avgDuration = (result.TotalSecs / float64(result.Completed)) / 60 // to minutes
	}

	// Get category breakdown
	categories, err := r.getCategoryBreakdown(ctx, userID, start, end)
	if err != nil {
		r.logger.Warn().Err(err).Msg("get category breakdown failed")
	}

	// Get peak hours
	peakHours, err := r.getPeakHours(ctx, userID, start, end)
	if err != nil {
		r.logger.Warn().Err(err).Msg("get peak hours failed")
	}

	return &TaskMetrics{
		CompletionRate:     completionRate,
		TotalTasks:         result.Total,
		CompletedTasks:     result.Completed,
		PostponedTasks:     result.Postponed,
		AbandonedTasks:     result.Abandoned,
		TotalDurationHours: result.TotalSecs / 3600,
		AvgTaskDuration:    avgDuration,
		Velocity:           float64(result.Completed) / days,
		PeakHours:          peakHours,
		ByCategory:         categories,
	}, nil
}

type categoryBreakdownDB struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	TaskCount    int     `json:"task_count"`
	TotalSecs    float64 `json:"total_secs"`
}

func (r *repository) getCategoryBreakdown(ctx context.Context, userID string, start, end time.Time) ([]CategoryBreakdown, error) {
	results, err := database.QueryAll[categoryBreakdownDB](ctx, r.db, `
		SELECT 
			category.id as category_id,
			category.name as category_name,
			count() as task_count,
			math::sum(duration::secs(end_date - start_date)) as total_secs
		FROM tasks
		WHERE created_by = $user 
		  AND start_date >= $start 
		  AND start_date <= $end
		  AND deleted_at IS NONE
		  AND category IS NOT NONE
		GROUP BY category
	`, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if err != nil {
		return nil, err
	}

	// Calculate total for percentages
	var totalSecs float64
	for _, r := range results {
		totalSecs += r.TotalSecs
	}

	categories := make([]CategoryBreakdown, len(results))
	for i, r := range results {
		pct := 0.0
		if totalSecs > 0 {
			pct = (r.TotalSecs / totalSecs) * 100
		}
		categories[i] = CategoryBreakdown{
			CategoryID:   database.ToStringID(database.MustRecordID("categories", r.CategoryID)),
			CategoryName: r.CategoryName,
			TaskCount:    r.TaskCount,
			Hours:        r.TotalSecs / 3600,
			Percentage:   pct,
		}
	}

	return categories, nil
}

type peakHourDB struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

func (r *repository) getPeakHours(ctx context.Context, userID string, start, end time.Time) ([]int, error) {
	results, err := database.QueryAll[peakHourDB](ctx, r.db, `
		SELECT 
			time::hour(start_date) as hour,
			count() as count
		FROM tasks
		WHERE created_by = $user 
		  AND start_date >= $start 
		  AND start_date <= $end
		  AND completed = true
		  AND deleted_at IS NONE
		GROUP BY hour
		ORDER BY count DESC
		LIMIT 3
	`, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if err != nil {
		return nil, err
	}

	hours := make([]int, len(results))
	for i, r := range results {
		hours[i] = r.Hour
	}
	return hours, nil
}

// =============================================================================
// EMOTION METRICS
// =============================================================================

type emotionMetricsDB struct {
	AvgValence float64 `json:"avg_valence"`
	AvgArousal float64 `json:"avg_arousal"`
	StdValence float64 `json:"std_valence"`
	TotalCount int     `json:"total_count"`
}

type quadrantCountDB struct {
	Quadrant string `json:"quadrant"`
	Count    int    `json:"count"`
}

type emotionCountDB struct {
	EmotionID   string `json:"emotion_id"`
	EmotionName string `json:"emotion_name"`
	Quadrant    string `json:"quadrant"`
	Count       int    `json:"count"`
}

func (r *repository) GetEmotionMetrics(ctx context.Context, userID string, start, end time.Time) (*EmotionMetrics, error) {
	// Get average valence/arousal
	avgResult, err := database.QueryFirst[emotionMetricsDB](ctx, r.db, `
		SELECT 
			math::mean(->task_emotions->emotions.valence) as avg_valence,
			math::mean(->task_emotions->emotions.arousal) as avg_arousal,
			math::stddev(->task_emotions->emotions.valence) as std_valence,
			count(->task_emotions) as total_count
		FROM tasks
		WHERE created_by = $user 
		  AND start_date >= $start 
		  AND start_date <= $end
		  AND deleted_at IS NONE
		GROUP ALL
	`, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if err != nil {
		r.logger.Error().Err(err).Msg("get emotion averages failed")
		return nil, err
	}

	if avgResult == nil {
		return &EmotionMetrics{}, nil
	}

	// Get quadrant distribution
	quadrants, err := database.QueryAll[quadrantCountDB](ctx, r.db, `
		SELECT 
			->task_emotions->emotions.quadrant as quadrant,
			count() as count
		FROM tasks
		WHERE created_by = $user 
		  AND start_date >= $start 
		  AND start_date <= $end
		  AND deleted_at IS NONE
		GROUP BY quadrant
	`, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if err != nil {
		r.logger.Warn().Err(err).Msg("get quadrant distribution failed")
	}

	quadrantDist := QuadrantDist{}
	dominantQuadrant := ""
	maxCount := 0
	totalQuadrant := 0

	for _, q := range quadrants {
		totalQuadrant += q.Count
		if q.Count > maxCount {
			maxCount = q.Count
			dominantQuadrant = q.Quadrant
		}
	}

	for _, q := range quadrants {
		pct := 0.0
		if totalQuadrant > 0 {
			pct = float64(q.Count) / float64(totalQuadrant) * 100
		}
		switch q.Quadrant {
		case "yellow":
			quadrantDist.Yellow = pct
		case "green":
			quadrantDist.Green = pct
		case "red":
			quadrantDist.Red = pct
		case "blue":
			quadrantDist.Blue = pct
		}
	}

	// Get top emotions
	topEmotions, err := database.QueryAll[emotionCountDB](ctx, r.db, `
		SELECT 
			->task_emotions->emotions.id as emotion_id,
			->task_emotions->emotions.name as emotion_name,
			->task_emotions->emotions.quadrant as quadrant,
			count() as count
		FROM tasks
		WHERE created_by = $user 
		  AND start_date >= $start 
		  AND start_date <= $end
		  AND deleted_at IS NONE
		GROUP BY emotion_id, emotion_name, quadrant
		ORDER BY count DESC
		LIMIT 5
	`, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if err != nil {
		r.logger.Warn().Err(err).Msg("get top emotions failed")
	}

	topEmotionsList := make([]EmotionCount, len(topEmotions))
	for i, e := range topEmotions {
		topEmotionsList[i] = EmotionCount(e)
	}

	// Calculate mood stability (1 - normalized std dev)
	moodStability := 1.0
	if avgResult.StdValence > 0 {
		// Normalize: max expected std dev is ~1 for valence range -1 to 1
		moodStability = 1.0 - (avgResult.StdValence / 1.0)
		if moodStability < 0 {
			moodStability = 0
		}
	}

	return &EmotionMetrics{
		AverageValence:       avgResult.AvgValence,
		AverageArousal:       avgResult.AvgArousal,
		MoodStability:        moodStability,
		DominantQuadrant:     dominantQuadrant,
		QuadrantDistribution: quadrantDist,
		TopEmotions:          topEmotionsList,
	}, nil
}

// =============================================================================
// GOAL METRICS
// =============================================================================

type goalProgressDB struct {
	GoalID        string  `json:"goal_id"`
	GoalTitle     string  `json:"goal_title"`
	GoalType      string  `json:"goal_type"`
	Status        string  `json:"status"`
	CurrentValue  float64 `json:"current_value"`
	TargetValue   float64 `json:"target_value"`
	CurrentStreak int     `json:"current_streak"`
	LongestStreak int     `json:"longest_streak"`
}

func (r *repository) GetGoalMetrics(ctx context.Context, userID string) (*GoalMetrics, error) {
	goals, err := database.QueryAll[goalProgressDB](ctx, r.db, `
		SELECT 
			id as goal_id,
			title as goal_title,
			goal_type,
			status,
			target.current_value as current_value,
			target.value as target_value,
			current_streak,
			longest_streak
		FROM goals
		WHERE created_by = $user 
		  AND deleted_at IS NONE
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Msg("get goal metrics failed")
		return nil, err
	}

	var activeCount, completedCount, totalStreakDays int
	var totalProgress float64
	progressList := make([]GoalProgress, 0, len(goals))
	streakLeaders := make([]StreakInfo, 0, len(goals))

	for _, g := range goals {
		if g.Status == "active" {
			activeCount++
		} else if g.Status == "completed" {
			completedCount++
		}

		totalStreakDays += g.CurrentStreak

		// Calculate progress percentage
		progress := 0.0
		if g.TargetValue > 0 {
			progress = (g.CurrentValue / g.TargetValue) * 100
			if progress > 100 {
				progress = 100
			}
		}
		totalProgress += progress

		progressList = append(progressList, GoalProgress{
			GoalID:       g.GoalID,
			GoalTitle:    g.GoalTitle,
			GoalType:     g.GoalType,
			Progress:     progress,
			CurrentValue: g.CurrentValue,
			TargetValue:  g.TargetValue,
			Status:       g.Status,
		})

		if g.CurrentStreak > 0 {
			streakLeaders = append(streakLeaders, StreakInfo{
				GoalID:        g.GoalID,
				GoalTitle:     g.GoalTitle,
				CurrentStreak: g.CurrentStreak,
				LongestStreak: g.LongestStreak,
			})
		}
	}

	avgProgress := 0.0
	if len(goals) > 0 {
		avgProgress = totalProgress / float64(len(goals))
	}

	return &GoalMetrics{
		ActiveGoals:     activeCount,
		CompletedGoals:  completedCount,
		AvgProgress:     avgProgress,
		TotalStreakDays: totalStreakDays,
		GoalProgress:    progressList,
		StreakLeaders:   streakLeaders,
	}, nil
}

// =============================================================================
// CATEGORY METRICS
// =============================================================================

func (r *repository) GetCategoryMetrics(ctx context.Context, userID string, start, end time.Time) (*CategoryMetrics, error) {
	categories, err := r.getCategoryBreakdown(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}

	var totalHours float64
	for _, c := range categories {
		totalHours += c.Hours
	}

	return &CategoryMetrics{
		TotalHours:   totalHours,
		Distribution: categories,
	}, nil
}

// =============================================================================
// TIME SERIES DATA
// =============================================================================

type dailyValueDB struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

func (r *repository) GetTimeSeriesData(ctx context.Context, userID, metric, groupBy string, start, end time.Time) (*TimeSeriesData, error) {
	var query string

	switch metric {
	case MetricTaskCompletion:
		query = `
			SELECT 
				time::format(time::floor(start_date, 1d), "%Y-%m-%d") as date,
				count(IF completed = true THEN 1 ELSE NONE END) as value
			FROM tasks
			WHERE created_by = $user 
			  AND start_date >= $start 
			  AND start_date <= $end
			  AND deleted_at IS NONE
			GROUP BY date
			ORDER BY date
		`
	case MetricValence, MetricMood:
		query = `
			SELECT 
				time::format(time::floor(start_date, 1d), "%Y-%m-%d") as date,
				math::mean(->task_emotions->emotions.valence) as value
			FROM tasks
			WHERE created_by = $user 
			  AND start_date >= $start 
			  AND start_date <= $end
			  AND deleted_at IS NONE
			GROUP BY date
			ORDER BY date
		`
	case MetricArousal:
		query = `
			SELECT 
				time::format(time::floor(start_date, 1d), "%Y-%m-%d") as date,
				math::mean(->task_emotions->emotions.arousal) as value
			FROM tasks
			WHERE created_by = $user 
			  AND start_date >= $start 
			  AND start_date <= $end
			  AND deleted_at IS NONE
			GROUP BY date
			ORDER BY date
		`
	default:
		return nil, fmt.Errorf("unsupported metric: %s", metric)
	}

	results, err := database.QueryAll[dailyValueDB](ctx, r.db, query, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("metric", metric).Msg("get time series failed")
		return nil, err
	}

	labels := make([]string, len(results))
	values := make([]float64, len(results))
	for i, r := range results {
		labels[i] = r.Date
		values[i] = r.Value
	}

	return &TimeSeriesData{
		Labels: labels,
		Series: []TimeSeriesSerie{
			{Name: metric, Values: values},
		},
	}, nil
}

// =============================================================================
// QUADRANT DISTRIBUTION (PIE CHART)
// =============================================================================

func (r *repository) GetQuadrantDistribution(ctx context.Context, userID string, start, end time.Time) (*PieChartData, error) {
	quadrants, err := database.QueryAll[quadrantCountDB](ctx, r.db, `
		SELECT 
			->task_emotions->emotions.quadrant as quadrant,
			count() as count
		FROM tasks
		WHERE created_by = $user 
		  AND start_date >= $start 
		  AND start_date <= $end
		  AND deleted_at IS NONE
		GROUP BY quadrant
	`, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if err != nil {
		return nil, err
	}

	var total float64
	for _, q := range quadrants {
		total += float64(q.Count)
	}

	segments := make([]PieSegment, len(quadrants))
	colors := map[string]string{
		"yellow": "#FBBF24",
		"green":  "#10B981",
		"red":    "#EF4444",
		"blue":   "#3B82F6",
	}

	for i, q := range quadrants {
		pct := 0.0
		if total > 0 {
			pct = float64(q.Count) / total * 100
		}
		segments[i] = PieSegment{
			Label:      q.Quadrant,
			Value:      float64(q.Count),
			Percentage: pct,
			Color:      colors[q.Quadrant],
		}
	}

	return &PieChartData{
		Segments: segments,
		Total:    total,
	}, nil
}

// =============================================================================
// PRODUCTIVITY HEATMAP
// =============================================================================

func (r *repository) GetProductivityHeatmap(ctx context.Context, userID string, start, end time.Time) (*HeatmapData, error) {
	type heatmapCell struct {
		DayOfWeek int `json:"day_of_week"`
		Hour      int `json:"hour"`
		Count     int `json:"count"`
	}

	results, err := database.QueryAll[heatmapCell](ctx, r.db, `
		SELECT 
			time::wday(start_date) as day_of_week,
			time::hour(start_date) as hour,
			count() as count
		FROM tasks
		WHERE created_by = $user 
		  AND start_date >= $start 
		  AND start_date <= $end
		  AND completed = true
		  AND deleted_at IS NONE
		GROUP BY day_of_week, hour
	`, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if err != nil {
		return nil, err
	}

	// Build 7x24 matrix (days x hours)
	values := make([][]float64, 7)
	for i := range values {
		values[i] = make([]float64, 24)
	}

	var maxVal float64
	for _, r := range results {
		if r.DayOfWeek >= 0 && r.DayOfWeek < 7 && r.Hour >= 0 && r.Hour < 24 {
			values[r.DayOfWeek][r.Hour] = float64(r.Count)
			if float64(r.Count) > maxVal {
				maxVal = float64(r.Count)
			}
		}
	}

	dayLabels := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	hourLabels := make([]string, 24)
	for i := 0; i < 24; i++ {
		hourLabels[i] = fmt.Sprintf("%02d:00", i)
	}

	return &HeatmapData{
		XLabels: hourLabels,
		YLabels: dayLabels,
		Values:  values,
		Min:     0,
		Max:     maxVal,
	}, nil
}
