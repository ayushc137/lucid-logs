package analytics

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the analytics business logic interface.
type Service interface {
	// GenerateChart generates chart data based on the request configuration.
	GenerateChart(ctx context.Context, userID string, req *ChartRequest) (*ChartResponse, error)

	// GetTaskMetrics retrieves task productivity metrics.
	GetTaskMetrics(ctx context.Context, userID string, period string, start, end *time.Time) (*TaskMetrics, error)

	// GetEmotionMetrics retrieves emotion/mood metrics.
	GetEmotionMetrics(ctx context.Context, userID string, period string, start, end *time.Time) (*EmotionMetrics, error)

	// GetGoalMetrics retrieves goal progress and streak metrics.
	GetGoalMetrics(ctx context.Context, userID string) (*GoalMetrics, error)

	// GetCategoryMetrics retrieves category time distribution.
	GetCategoryMetrics(ctx context.Context, userID string, period string, start, end *time.Time) (*CategoryMetrics, error)

	// GetDashboard retrieves combined dashboard metrics.
	GetDashboard(ctx context.Context, userID string, period string) (*DashboardResponse, error)
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a new analytics Service.
func NewService(repo Repository) Service {
	return &service{
		repo:   repo,
		logger: log.With().Str("service", "analytics").Logger(),
	}
}

// =============================================================================
// CHART GENERATION
// =============================================================================

func (s *service) GenerateChart(ctx context.Context, userID string, req *ChartRequest) (*ChartResponse, error) {
	start, end := s.resolvePeriod(req.Period, req.StartDate, req.EndDate)

	var data any
	var err error

	switch req.ChartType {
	case ChartTypeTimeSeries:
		data, err = s.repo.GetTimeSeriesData(ctx, userID, req.Metric, req.GroupBy, start, end)
	case ChartTypePie:
		if req.Metric == MetricQuadrant || req.Metric == MetricMood {
			data, err = s.repo.GetQuadrantDistribution(ctx, userID, start, end)
		} else {
			// Category distribution as pie
			catMetrics, catErr := s.repo.GetCategoryMetrics(ctx, userID, start, end)
			if catErr != nil {
				err = catErr
			} else {
				segments := make([]PieSegment, len(catMetrics.Distribution))
				for i, c := range catMetrics.Distribution {
					segments[i] = PieSegment{
						Label:      c.CategoryName,
						Value:      c.Hours,
						Percentage: c.Percentage,
					}
				}
				data = &PieChartData{Segments: segments, Total: catMetrics.TotalHours}
			}
		}
	case ChartTypeBar:
		catMetrics, catErr := s.repo.GetCategoryMetrics(ctx, userID, start, end)
		if catErr != nil {
			err = catErr
		} else {
			categories := make([]string, len(catMetrics.Distribution))
			values := make([]float64, len(catMetrics.Distribution))
			for i, c := range catMetrics.Distribution {
				categories[i] = c.CategoryName
				values[i] = c.Hours
			}
			data = &BarChartData{
				Categories: categories,
				Series:     []BarChartSeries{{Name: "Hours", Values: values}},
			}
		}
	case ChartTypeHeatmap:
		data, err = s.repo.GetProductivityHeatmap(ctx, userID, start, end)
	case ChartTypeGauge:
		taskMetrics, tErr := s.repo.GetTaskMetrics(ctx, userID, start, end)
		if tErr != nil {
			err = tErr
		} else {
			data = &GaugeData{
				Value:  taskMetrics.CompletionRate,
				Min:    0,
				Max:    100,
				Target: 80, // Default target
				Label:  "Completion Rate",
				Unit:   "%",
			}
		}
	default:
		// Fall back to time series
		data, err = s.repo.GetTimeSeriesData(ctx, userID, req.Metric, req.GroupBy, start, end)
	}

	if err != nil {
		s.logger.Error().Err(err).Str("chart_type", req.ChartType).Msg("chart generation failed")
		return nil, err
	}

	return &ChartResponse{
		ChartType: req.ChartType,
		Metric:    req.Metric,
		Data:      data,
		Meta: ChartMeta{
			Period:     req.Period,
			StartDate:  start.Format(time.RFC3339),
			EndDate:    end.Format(time.RFC3339),
			ComputedAt: time.Now().Format(time.RFC3339),
		},
	}, nil
}

// =============================================================================
// INDIVIDUAL METRICS
// =============================================================================

func (s *service) GetTaskMetrics(ctx context.Context, userID string, period string, start, end *time.Time) (*TaskMetrics, error) {
	startTime, endTime := s.resolvePeriod(period, start, end)
	return s.repo.GetTaskMetrics(ctx, userID, startTime, endTime)
}

func (s *service) GetEmotionMetrics(ctx context.Context, userID string, period string, start, end *time.Time) (*EmotionMetrics, error) {
	startTime, endTime := s.resolvePeriod(period, start, end)
	return s.repo.GetEmotionMetrics(ctx, userID, startTime, endTime)
}

func (s *service) GetGoalMetrics(ctx context.Context, userID string) (*GoalMetrics, error) {
	return s.repo.GetGoalMetrics(ctx, userID)
}

func (s *service) GetCategoryMetrics(ctx context.Context, userID string, period string, start, end *time.Time) (*CategoryMetrics, error) {
	startTime, endTime := s.resolvePeriod(period, start, end)
	return s.repo.GetCategoryMetrics(ctx, userID, startTime, endTime)
}

// =============================================================================
// DASHBOARD
// =============================================================================

func (s *service) GetDashboard(ctx context.Context, userID string, period string) (*DashboardResponse, error) {
	start, end := s.resolvePeriod(period, nil, nil)

	tasks, err := s.repo.GetTaskMetrics(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}

	emotions, err := s.repo.GetEmotionMetrics(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}

	goals, err := s.repo.GetGoalMetrics(ctx, userID)
	if err != nil {
		return nil, err
	}

	categories, err := s.repo.GetCategoryMetrics(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}

	return &DashboardResponse{
		Period:     period,
		Tasks:      *tasks,
		Emotions:   *emotions,
		Goals:      *goals,
		Categories: *categories,
	}, nil
}

// =============================================================================
// HELPERS
// =============================================================================

// resolvePeriod converts a period string to start/end times.
func (s *service) resolvePeriod(period string, start, end *time.Time) (time.Time, time.Time) {
	now := time.Now().UTC()

	if period == PeriodCustom && start != nil && end != nil {
		return *start, *end
	}

	var startTime, endTime time.Time

	switch period {
	case PeriodDay:
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endTime = startTime.Add(24 * time.Hour).Add(-time.Second)
	case PeriodWeek:
		// Go back to start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday -> 7
		}
		startTime = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
		endTime = startTime.Add(7 * 24 * time.Hour).Add(-time.Second)
	case PeriodMonth:
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		endTime = startTime.AddDate(0, 1, 0).Add(-time.Second)
	case PeriodQuarter:
		quarter := (int(now.Month()) - 1) / 3
		startMonth := time.Month(quarter*3 + 1)
		startTime = time.Date(now.Year(), startMonth, 1, 0, 0, 0, 0, time.UTC)
		endTime = startTime.AddDate(0, 3, 0).Add(-time.Second)
	case PeriodYear:
		startTime = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		endTime = time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)
	default:
		// Default to week
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startTime = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
		endTime = startTime.Add(7 * 24 * time.Hour).Add(-time.Second)
	}

	return startTime, endTime
}
