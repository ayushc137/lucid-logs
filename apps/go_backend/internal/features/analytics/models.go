// Package analytics provides a generic, extensible analytics and charting API.
//
// This package implements:
//   - Flexible chart configuration types supporting any frontend visualization
//   - Metric calculators for tasks, emotions, goals, categories
//   - Time series and aggregation queries
//   - Emotion analysis (valence, arousal, quadrant distribution)
//
// The design allows frontend to request any chart type with custom parameters
// without requiring backend changes for new visualizations.
package analytics

import (
	"time"
)

// =============================================================================
// CHART REQUEST/RESPONSE TYPES
// =============================================================================

// ChartRequest is a flexible request structure for generating any chart.
//
// @Description Generic request for generating chart data
type ChartRequest struct {
	ChartType string     `json:"chart_type" validate:"required,oneof=time_series pie bar heatmap radar scatter gauge" example:"time_series"`
	Metric    string     `json:"metric" validate:"required" example:"mood"`
	Period    string     `json:"period" validate:"required,oneof=day week month quarter year custom" example:"week"`
	StartDate *time.Time `json:"start_date,omitempty" example:"2025-12-01T00:00:00Z"`
	EndDate   *time.Time `json:"end_date,omitempty" example:"2025-12-14T23:59:59Z"`
	GroupBy   string     `json:"group_by,omitempty" example:"day"` // "hour", "day", "week", "month", "category", "quadrant"
	Filters   Filters    `json:"filters,omitempty"`
}

// Filters provides extensible filtering options.
type Filters struct {
	CategoryIDs []string `json:"category_ids,omitempty"`
	GoalIDs     []string `json:"goal_ids,omitempty"`
	Quadrants   []string `json:"quadrants,omitempty"` // "yellow", "green", "red", "blue"
	TaskStatus  []string `json:"task_status,omitempty"`
	LifeDomain  []string `json:"life_domain,omitempty"`
}

// ChartResponse is a generic response supporting all chart types.
//
// @Description Generic chart data response
type ChartResponse struct {
	ChartType string    `json:"chart_type"`
	Metric    string    `json:"metric"`
	Data      any       `json:"data"` // Shape varies by chart_type
	Meta      ChartMeta `json:"meta"`
}

// ChartMeta contains metadata about the chart data.
type ChartMeta struct {
	Period     string `json:"period"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	DataPoints int    `json:"data_points"`
	ComputedAt string `json:"computed_at"`
}

// =============================================================================
// TIME SERIES DATA TYPES
// =============================================================================

// TimeSeriesData for line/area charts.
type TimeSeriesData struct {
	Labels []string          `json:"labels"` // Date/time labels
	Series []TimeSeriesSerie `json:"series"` // One or more data series
}

// TimeSeriesSerie represents a single series in a time series chart.
type TimeSeriesSerie struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color,omitempty"`
}

// =============================================================================
// PIE/DONUT CHART DATA
// =============================================================================

// PieChartData for pie/donut charts.
type PieChartData struct {
	Segments []PieSegment `json:"segments"`
	Total    float64      `json:"total"`
}

// PieSegment represents a slice in a pie chart.
type PieSegment struct {
	Label      string  `json:"label"`
	Value      float64 `json:"value"`
	Percentage float64 `json:"percentage"`
	Color      string  `json:"color,omitempty"`
}

// =============================================================================
// BAR CHART DATA
// =============================================================================

// BarChartData for bar/column charts.
type BarChartData struct {
	Categories []string         `json:"categories"`
	Series     []BarChartSeries `json:"series"`
}

// BarChartSeries represents a data series in a bar chart.
type BarChartSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color,omitempty"`
}

// =============================================================================
// HEATMAP DATA (for activity/emotion heatmaps)
// =============================================================================

// HeatmapData for calendar/grid heatmaps.
type HeatmapData struct {
	XLabels []string    `json:"x_labels"` // Columns (e.g., days of week)
	YLabels []string    `json:"y_labels"` // Rows (e.g., hours)
	Values  [][]float64 `json:"values"`   // 2D value matrix
	Min     float64     `json:"min"`
	Max     float64     `json:"max"`
}

// =============================================================================
// RADAR/SPIDER CHART DATA (for balance wheel)
// =============================================================================

// RadarChartData for radar/spider charts.
type RadarChartData struct {
	Axes   []string           `json:"axes"` // Category labels around the radar
	Series []RadarChartSeries `json:"series"`
}

// RadarChartSeries represents a data series in a radar chart.
type RadarChartSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"` // Value for each axis
	Color  string    `json:"color,omitempty"`
}

// =============================================================================
// GAUGE DATA (for single metrics)
// =============================================================================

// GaugeData for gauge/progress indicators.
type GaugeData struct {
	Value  float64 `json:"value"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Target float64 `json:"target,omitempty"`
	Label  string  `json:"label"`
	Unit   string  `json:"unit,omitempty"`
}

// =============================================================================
// METRIC-SPECIFIC RESPONSE TYPES
// =============================================================================

// TaskMetrics contains productivity metrics.
//
// @Description Task productivity metrics
type TaskMetrics struct {
	CompletionRate     float64             `json:"completion_rate"` // % of tasks completed
	TotalTasks         int                 `json:"total_tasks"`
	CompletedTasks     int                 `json:"completed_tasks"`
	PostponedTasks     int                 `json:"postponed_tasks"`
	AbandonedTasks     int                 `json:"abandoned_tasks"`
	TotalDurationHours float64             `json:"total_duration_hours"`
	AvgTaskDuration    float64             `json:"avg_task_duration_minutes"`
	Velocity           float64             `json:"velocity"`   // Tasks per day
	PeakHours          []int               `json:"peak_hours"` // Most productive hours
	ByCategory         []CategoryBreakdown `json:"by_category,omitempty"`
}

// CategoryBreakdown shows metrics per category.
type CategoryBreakdown struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	TaskCount    int     `json:"task_count"`
	Hours        float64 `json:"hours"`
	Percentage   float64 `json:"percentage"`
}

// EmotionMetrics contains mood/emotion analytics.
//
// @Description Emotion and mood metrics with 6D vector analysis
type EmotionMetrics struct {
	AverageValence       float64        `json:"average_valence"`   // -1 to +1
	AverageArousal       float64        `json:"average_arousal"`   // -1 to +1
	MoodStability        float64        `json:"mood_stability"`    // 0 to 1 (1 = stable)
	DominantQuadrant     string         `json:"dominant_quadrant"` // yellow/green/red/blue
	QuadrantDistribution QuadrantDist   `json:"quadrant_distribution"`
	EmotionCounts        map[string]int `json:"emotion_counts"` // Emotion name -> count
	TopEmotions          []EmotionCount `json:"top_emotions"`
	Trend                []DailyMood    `json:"trend,omitempty"` // Daily mood over time
}

// QuadrantDist shows percentage in each mood quadrant.
type QuadrantDist struct {
	Yellow float64 `json:"yellow"` // High energy, pleasant
	Green  float64 `json:"green"`  // Low energy, pleasant
	Red    float64 `json:"red"`    // High energy, unpleasant
	Blue   float64 `json:"blue"`   // Low energy, unpleasant
}

// EmotionCount for top emotions ranking.
type EmotionCount struct {
	EmotionID   string `json:"emotion_id"`
	EmotionName string `json:"emotion_name"`
	Quadrant    string `json:"quadrant"`
	Count       int    `json:"count"`
}

// DailyMood for mood trend over time.
type DailyMood struct {
	Date     string  `json:"date"`
	Valence  float64 `json:"valence"`
	Arousal  float64 `json:"arousal"`
	Quadrant string  `json:"quadrant"`
}

// GoalMetrics contains goal progress and streak analytics.
//
// @Description Goal and streak metrics
type GoalMetrics struct {
	ActiveGoals     int            `json:"active_goals"`
	CompletedGoals  int            `json:"completed_goals"`
	AvgProgress     float64        `json:"avg_progress"`      // Average % progress
	TotalStreakDays int            `json:"total_streak_days"` // Sum of all current streaks
	GoalProgress    []GoalProgress `json:"goal_progress"`
	StreakLeaders   []StreakInfo   `json:"streak_leaders"` // Top streaks
}

// GoalProgress shows progress for a single goal.
type GoalProgress struct {
	GoalID       string  `json:"goal_id"`
	GoalTitle    string  `json:"goal_title"`
	GoalType     string  `json:"goal_type"`
	Progress     float64 `json:"progress"` // 0-100%
	CurrentValue float64 `json:"current_value,omitempty"`
	TargetValue  float64 `json:"target_value,omitempty"`
	Status       string  `json:"status"`
}

// StreakInfo shows streak data for a goal.
type StreakInfo struct {
	GoalID        string `json:"goal_id"`
	GoalTitle     string `json:"goal_title"`
	CurrentStreak int    `json:"current_streak"`
	LongestStreak int    `json:"longest_streak"`
}

// CategoryMetrics contains time distribution analytics.
//
// @Description Category time distribution metrics
type CategoryMetrics struct {
	TotalHours     float64               `json:"total_hours"`
	Distribution   []CategoryBreakdown   `json:"distribution"`
	NeglectedAreas []NeglectedCategory   `json:"neglected_areas,omitempty"`
	LifeDomainDist []LifeDomainBreakdown `json:"life_domain_distribution,omitempty"`
}

// NeglectedCategory identifies categories with no recent activity.
type NeglectedCategory struct {
	CategoryID      string `json:"category_id"`
	CategoryName    string `json:"category_name"`
	DaysSinceActive int    `json:"days_since_active"`
}

// LifeDomainBreakdown shows time per life domain.
type LifeDomainBreakdown struct {
	Domain     string  `json:"domain"`
	Hours      float64 `json:"hours"`
	Percentage float64 `json:"percentage"`
}

// =============================================================================
// DASHBOARD RESPONSE
// =============================================================================

// DashboardResponse combines key metrics for overview.
//
// @Description Combined dashboard metrics
type DashboardResponse struct {
	Period     string          `json:"period"`
	Tasks      TaskMetrics     `json:"tasks"`
	Emotions   EmotionMetrics  `json:"emotions"`
	Goals      GoalMetrics     `json:"goals"`
	Categories CategoryMetrics `json:"categories"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Chart types
	ChartTypeTimeSeries = "time_series"
	ChartTypePie        = "pie"
	ChartTypeBar        = "bar"
	ChartTypeHeatmap    = "heatmap"
	ChartTypeRadar      = "radar"
	ChartTypeScatter    = "scatter"
	ChartTypeGauge      = "gauge"

	// Periods
	PeriodDay     = "day"
	PeriodWeek    = "week"
	PeriodMonth   = "month"
	PeriodQuarter = "quarter"
	PeriodYear    = "year"
	PeriodCustom  = "custom"

	// Metrics
	MetricTaskCompletion = "task_completion"
	MetricMood           = "mood"
	MetricValence        = "valence"
	MetricArousal        = "arousal"
	MetricQuadrant       = "quadrant"
	MetricCategoryTime   = "category_time"
	MetricGoalProgress   = "goal_progress"
	MetricStreak         = "streak"
	MetricProductivity   = "productivity"
	MetricLifeBalance    = "life_balance"

	// Table name
	Table = "agg_daily"
)
