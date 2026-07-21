package analytics

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/shared/database"
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

	// GetStreaks retrieves the dashboard streak summary.
	GetStreaks(ctx context.Context, userID string) (*StreaksResponse, error)

	// GetActivityHeatmap retrieves the GitHub-style logged-days heatmap.
	GetActivityHeatmap(ctx context.Context, userID string, start, end time.Time) (*ActivityHeatmapResponse, error)
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
			COUNT(*) as total,
			SUM(CASE WHEN completed = 1 THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status = 'postponed' THEN 1 ELSE 0 END) as postponed,
			SUM(CASE WHEN status = 'abandoned' THEN 1 ELSE 0 END) as abandoned,
			COALESCE(SUM(CAST(strftime('%s', end_date) AS REAL) - CAST(strftime('%s', start_date) AS REAL)), 0) as total_secs
		FROM tasks
		WHERE created_by = $user
		  AND start_date >= $start
		  AND start_date <= $end
		  AND deleted_at IS NULL
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

	// Get focus score: % of time on high-priority (4+) tasks
	focusScore := 0.0
	if result.TotalSecs > 0 {
		focusResult, focusErr := database.QueryFirst[struct {
			HighSecs float64 `json:"high_secs"`
		}](ctx, r.db, `
			SELECT COALESCE(SUM(CAST(strftime('%s', end_date) AS REAL) - CAST(strftime('%s', start_date) AS REAL)), 0) as high_secs
			FROM tasks
			WHERE created_by = $user
			  AND start_date >= $start
			  AND start_date <= $end
			  AND deleted_at IS NULL
			  AND CAST(priority AS INTEGER) >= 4
		`, map[string]any{
			"user":  userID,
			"start": start,
			"end":   end,
		})
		if focusErr != nil {
			r.logger.Warn().Err(focusErr).Msg("get focus score failed")
		} else if focusResult != nil {
			focusScore = (focusResult.HighSecs / result.TotalSecs) * 100
		}
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
		FocusScore:         focusScore,
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
			c.id as category_id,
			c.name as category_name,
			COUNT(*) as task_count,
			COALESCE(SUM(CAST(strftime('%s', t.end_date) AS REAL) - CAST(strftime('%s', t.start_date) AS REAL)), 0) as total_secs
		FROM tasks t
		JOIN categories c ON c.id = t.category_id
		WHERE t.created_by = $user
		  AND t.start_date >= $start
		  AND t.start_date <= $end
		  AND t.deleted_at IS NULL
		  AND t.category_id IS NOT NULL
		GROUP BY c.id, c.name
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
	for _, res := range results {
		totalSecs += res.TotalSecs
	}

	categories := make([]CategoryBreakdown, len(results))
	for i, res := range results {
		pct := 0.0
		if totalSecs > 0 {
			pct = (res.TotalSecs / totalSecs) * 100
		}
		categories[i] = CategoryBreakdown{
			CategoryID:   database.ToStringID(database.MustRecordID("categories", res.CategoryID)),
			CategoryName: res.CategoryName,
			TaskCount:    res.TaskCount,
			Hours:        res.TotalSecs / 3600,
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
			CAST(strftime('%H', start_date) AS INTEGER) as hour,
			COUNT(*) as count
		FROM tasks
		WHERE created_by = $user
		  AND start_date >= $start
		  AND start_date <= $end
		  AND completed = 1
		  AND deleted_at IS NULL
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
	for i, res := range results {
		hours[i] = res.Hour
	}
	return hours, nil
}

// =============================================================================
// EMOTION METRICS
// =============================================================================

type emotionMetricsDB struct {
	AvgValence   float64 `json:"avg_valence"`
	AvgArousal   float64 `json:"avg_arousal"`
	SumValence   float64 `json:"sum_valence"`
	SumSqValence float64 `json:"sum_sq_valence"`
	TotalCount   int     `json:"total_count"`
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
	// Get average valence/arousal and aggregates for computing std dev
	avgResult, err := database.QueryFirst[emotionMetricsDB](ctx, r.db, `
		SELECT
			AVG(e.valence) as avg_valence,
			AVG(e.arousal) as avg_arousal,
			COALESCE(SUM(e.valence), 0) as sum_valence,
			COALESCE(SUM(e.valence * e.valence), 0) as sum_sq_valence,
			COUNT(*) as total_count
		FROM tasks t
		JOIN task_emotions te ON te.task_id = t.id
		JOIN emotions e ON e.id = te.emotion_id
		WHERE t.created_by = $user
		  AND t.start_date >= $start
		  AND t.start_date <= $end
		  AND t.deleted_at IS NULL
	`, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if err != nil {
		r.logger.Error().Err(err).Msg("get emotion averages failed")
		return nil, err
	}

	if avgResult == nil || avgResult.TotalCount == 0 {
		return &EmotionMetrics{}, nil
	}

	// Compute standard deviation of valence using the computational formula:
	// variance = E[x^2] - (E[x])^2
	stdValence := 0.0
	if avgResult.TotalCount > 0 {
		meanSq := avgResult.SumSqValence / float64(avgResult.TotalCount)
		mean := avgResult.SumValence / float64(avgResult.TotalCount)
		variance := meanSq - (mean * mean)
		if variance > 0 {
			stdValence = math.Sqrt(variance)
		}
	}

	// Get quadrant distribution
	quadrants, err := database.QueryAll[quadrantCountDB](ctx, r.db, `
		SELECT
			e.quadrant as quadrant,
			COUNT(*) as count
		FROM tasks t
		JOIN task_emotions te ON te.task_id = t.id
		JOIN emotions e ON e.id = te.emotion_id
		WHERE t.created_by = $user
		  AND t.start_date >= $start
		  AND t.start_date <= $end
		  AND t.deleted_at IS NULL
		GROUP BY e.quadrant
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
			e.id as emotion_id,
			e.name as emotion_name,
			e.quadrant as quadrant,
			COUNT(*) as count
		FROM tasks t
		JOIN task_emotions te ON te.task_id = t.id
		JOIN emotions e ON e.id = te.emotion_id
		WHERE t.created_by = $user
		  AND t.start_date >= $start
		  AND t.start_date <= $end
		  AND t.deleted_at IS NULL
		GROUP BY e.id, e.name, e.quadrant
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
	if stdValence > 0 {
		// Normalize: max expected std dev is ~1 for valence range -1 to 1
		moodStability = 1.0 - (stdValence / 1.0)
		if moodStability < 0 {
			moodStability = 0
		}
	}

	// Get daily mood trend (avg valence/arousal per day)
	dailyTrend, trendErr := database.QueryAll[struct {
		Date    string  `json:"date"`
		Valence float64 `json:"valence"`
		Arousal float64 `json:"arousal"`
	}](ctx, r.db, `
		SELECT
			date(t.start_date) as date,
			AVG(e.valence) as valence,
			AVG(e.arousal) as arousal
		FROM tasks t
		JOIN task_emotions te ON te.task_id = t.id
		JOIN emotions e ON e.id = te.emotion_id
		WHERE t.created_by = $user
		  AND t.start_date >= $start
		  AND t.start_date <= $end
		  AND t.deleted_at IS NULL
		GROUP BY date(t.start_date)
		ORDER BY date
	`, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if trendErr != nil {
		r.logger.Warn().Err(trendErr).Msg("get mood trend failed")
	}

	trend := make([]DailyMood, len(dailyTrend))
	for i, d := range dailyTrend {
		quadrant := ""
		switch {
		case d.Valence >= 0 && d.Arousal >= 0:
			quadrant = "yellow"
		case d.Valence >= 0 && d.Arousal < 0:
			quadrant = "green"
		case d.Valence < 0 && d.Arousal >= 0:
			quadrant = "red"
		default:
			quadrant = "blue"
		}
		trend[i] = DailyMood{
			Date:     d.Date,
			Valence:  d.Valence,
			Arousal:  d.Arousal,
			Quadrant: quadrant,
		}
	}

	return &EmotionMetrics{
		AverageValence:       avgResult.AvgValence,
		AverageArousal:       avgResult.AvgArousal,
		MoodStability:        moodStability,
		DominantQuadrant:     dominantQuadrant,
		QuadrantDistribution: quadrantDist,
		TopEmotions:          topEmotionsList,
		Trend:                trend,
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
			CASE
				WHEN recurrence IS NOT NULL THEN 'habit'
				WHEN target IS NOT NULL AND json_extract(target, '$.operator') IN ('lte', 'eq') THEN 'avoidance'
				WHEN target IS NOT NULL THEN 'measurable'
				ELSE 'simple'
			END as goal_type,
			status,
			COALESCE(current_value, 0) as current_value,
			COALESCE(json_extract(target, '$.value'), 0) as target_value,
			current_streak,
			longest_streak
		FROM goals
		WHERE created_by = $user
		  AND deleted_at IS NULL
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
				date(start_date) as date,
				SUM(CASE WHEN completed = 1 THEN 1 ELSE 0 END) as value
			FROM tasks
			WHERE created_by = $user
			  AND start_date >= $start
			  AND start_date <= $end
			  AND deleted_at IS NULL
			GROUP BY date
			ORDER BY date
		`
	case MetricValence, MetricMood:
		query = `
			SELECT
				date(t.start_date) as date,
				AVG(e.valence) as value
			FROM tasks t
			JOIN task_emotions te ON te.task_id = t.id
			JOIN emotions e ON e.id = te.emotion_id
			WHERE t.created_by = $user
			  AND t.start_date >= $start
			  AND t.start_date <= $end
			  AND t.deleted_at IS NULL
			GROUP BY date
			ORDER BY date
		`
	case MetricArousal:
		query = `
			SELECT
				date(t.start_date) as date,
				AVG(e.arousal) as value
			FROM tasks t
			JOIN task_emotions te ON te.task_id = t.id
			JOIN emotions e ON e.id = te.emotion_id
			WHERE t.created_by = $user
			  AND t.start_date >= $start
			  AND t.start_date <= $end
			  AND t.deleted_at IS NULL
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
	for i, res := range results {
		labels[i] = res.Date
		values[i] = res.Value
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
			e.quadrant as quadrant,
			COUNT(*) as count
		FROM tasks t
		JOIN task_emotions te ON te.task_id = t.id
		JOIN emotions e ON e.id = te.emotion_id
		WHERE t.created_by = $user
		  AND t.start_date >= $start
		  AND t.start_date <= $end
		  AND t.deleted_at IS NULL
		GROUP BY e.quadrant
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
			CAST(strftime('%w', start_date) AS INTEGER) as day_of_week,
			CAST(strftime('%H', start_date) AS INTEGER) as hour,
			COUNT(*) as count
		FROM tasks
		WHERE created_by = $user
		  AND start_date >= $start
		  AND start_date <= $end
		  AND completed = 1
		  AND deleted_at IS NULL
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
	for _, res := range results {
		if res.DayOfWeek >= 0 && res.DayOfWeek < 7 && res.Hour >= 0 && res.Hour < 24 {
			values[res.DayOfWeek][res.Hour] = float64(res.Count)
			if float64(res.Count) > maxVal {
				maxVal = float64(res.Count)
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

// =============================================================================
// STREAKS (DASHBOARD WIDGET)
// =============================================================================

type streakRow struct {
	GoalID        string `json:"goal_id"`
	GoalTitle     string `json:"goal_title"`
	CurrentStreak int    `json:"current_streak"`
	LongestStreak int    `json:"longest_streak"`
}

// GetStreaks computes a dashboard streak summary from the goals table.
// TotalCurrentStreakDays is the sum of every active goal's current_streak.
// LongestStreakEver is the max longest_streak across all goals.
// ActiveStreaks lists goals with a non-zero current streak.
func (r *repository) GetStreaks(ctx context.Context, userID string) (*StreaksResponse, error) {
	rows, err := database.QueryAll[streakRow](ctx, r.db, `
		SELECT
			id as goal_id,
			title as goal_title,
			COALESCE(current_streak, 0) as current_streak,
			COALESCE(longest_streak, 0) as longest_streak
		FROM goals
		WHERE created_by = $user
		  AND deleted_at IS NULL
		  AND status = 'active'
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Msg("get streaks failed")
		return nil, err
	}

	resp := &StreaksResponse{
		ActiveStreaks: make([]StreakInfo, 0, len(rows)),
	}
	for _, row := range rows {
		resp.TotalCurrentStreakDays += row.CurrentStreak
		if row.LongestStreak > resp.LongestStreakEver {
			resp.LongestStreakEver = row.LongestStreak
		}
		if row.CurrentStreak > 0 {
			resp.ActiveStreaks = append(resp.ActiveStreaks, StreakInfo{
				GoalID:        row.GoalID,
				GoalTitle:     row.GoalTitle,
				CurrentStreak: row.CurrentStreak,
				LongestStreak: row.LongestStreak,
			})
		}
	}
	return resp, nil
}

// =============================================================================
// ACTIVITY HEATMAP (GitHub-style logged-days)
// =============================================================================

type activityDayRow struct {
	Date    string  `json:"date"`
	Count   int     `json:"count"`
	Seconds float64 `json:"seconds"`
}

// GetActivityHeatmap returns a per-day activity map for the requested range,
// including consecutive-day streaks computed across the user's full history
// (not just the window) so "current streak" is accurate.
func (r *repository) GetActivityHeatmap(ctx context.Context, userID string, start, end time.Time) (*ActivityHeatmapResponse, error) {
	// Per-day counts in the requested window.
	rows, err := database.QueryAll[activityDayRow](ctx, r.db, `
		SELECT
			date(start_date) as date,
			COUNT(*) as count,
			COALESCE(SUM(CAST(strftime('%s', end_date) AS REAL) - CAST(strftime('%s', start_date) AS REAL)), 0) as seconds
		FROM tasks
		WHERE created_by = $user
		  AND start_date >= $start
		  AND start_date <= $end
		  AND deleted_at IS NULL
		GROUP BY date(start_date)
		ORDER BY date
	`, map[string]any{
		"user":  userID,
		"start": start,
		"end":   end,
	})
	if err != nil {
		r.logger.Error().Err(err).Msg("get activity heatmap failed")
		return nil, err
	}

	byDate := make(map[string]activityDayRow, len(rows))
	maxCount := 0
	for _, row := range rows {
		byDate[row.Date] = row
		if row.Count > maxCount {
			maxCount = row.Count
		}
	}

	// Build every day in the range.
	days := []ActivityHeatmapDay{}
	daysLogged := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		row, ok := byDate[key]
		day := ActivityHeatmapDay{Date: key}
		if ok {
			day.Count = row.Count
			day.Minutes = row.Seconds / 60
			day.HasEntry = row.Count > 0
			if day.HasEntry {
				daysLogged++
			}
			// Intensity 0-4 scale based on count relative to max.
			switch {
			case row.Count == 0:
				day.Intensity = 0
			case maxCount <= 1:
				day.Intensity = 4
			default:
				ratio := float64(row.Count) / float64(maxCount)
				switch {
				case ratio >= 0.75:
					day.Intensity = 4
				case ratio >= 0.5:
					day.Intensity = 3
				case ratio >= 0.25:
					day.Intensity = 2
				default:
					day.Intensity = 1
				}
			}
		}
		days = append(days, day)
	}

	// Compute streaks across the user's entire logged history, not just the window,
	// so current/longest streaks reflect reality.
	allDates, err := database.QueryAll[activityDayRow](ctx, r.db, `
		SELECT DISTINCT date(start_date) as date, 1 as count, 0 as seconds
		FROM tasks
		WHERE created_by = $user
		  AND deleted_at IS NULL
		ORDER BY date
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		r.logger.Warn().Err(err).Msg("get all activity dates for streak computation failed")
	}

	currentStreak, longestStreak := computeConsecutiveDayStreaks(allDates)

	totalDays := int(end.Sub(start).Hours()/24) + 1
	return &ActivityHeatmapResponse{
		Days:          days,
		TotalDays:     totalDays,
		DaysLogged:    daysLogged,
		CurrentStreak: currentStreak,
		LongestStreak: longestStreak,
	}, nil
}

// computeConsecutiveDayStreaks returns (currentStreak, longestStreak) given a
// chronological list of distinct days on which the user logged anything.
// Current streak counts back from today (UTC); if today has no entry but
// yesterday does, the streak is still alive (the day isn't over).
func computeConsecutiveDayStreaks(rows []activityDayRow) (current, longest int) {
	if len(rows) == 0 {
		return 0, 0
	}

	dateSet := make(map[string]bool, len(rows))
	for _, r := range rows {
		dateSet[r.Date] = true
	}

	// Longest streak: walk the sorted list and count runs.
	longest = 0
	run := 0
	var prev time.Time
	for i, r := range rows {
		d, err := time.Parse("2006-01-02", r.Date)
		if err != nil {
			continue
		}
		if i == 0 || d.Sub(prev) > 24*time.Hour {
			run = 1
		} else {
			run++
		}
		if run > longest {
			longest = run
		}
		prev = d
	}

	// Current streak: count back from today (or yesterday if today is empty).
	today := time.Now().UTC().Truncate(24 * time.Hour)
	cursor := today
	if !dateSet[cursor.Format("2006-01-02")] {
		cursor = cursor.AddDate(0, 0, -1)
	}
	for dateSet[cursor.Format("2006-01-02")] {
		current++
		cursor = cursor.AddDate(0, 0, -1)
	}
	return current, longest
}
