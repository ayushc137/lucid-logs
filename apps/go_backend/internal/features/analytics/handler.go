package analytics

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles HTTP requests for analytics.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new analytics Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTE REGISTRATION
// =============================================================================

// RegisterRoutes registers the analytics routes.
//
// Routes:
//   - POST /analytics/charts      Generate chart data
//   - GET  /analytics/dashboard   Combined dashboard metrics
//   - GET  /analytics/metrics/tasks     Task productivity metrics
//   - GET  /analytics/metrics/emotions  Emotion/mood metrics
//   - GET  /analytics/metrics/goals     Goal progress metrics
//   - GET  /analytics/metrics/categories Category distribution
func RegisterRoutes(rg *gin.RouterGroup, service Service, validator *validator.Validator) {
	h := NewHandler(service, validator)

	rg.POST("/charts", h.GenerateChart)
	rg.GET("/dashboard", h.GetDashboard)
	rg.GET("/metrics/tasks", h.GetTaskMetrics)
	rg.GET("/metrics/emotions", h.GetEmotionMetrics)
	rg.GET("/metrics/goals", h.GetGoalMetrics)
	rg.GET("/metrics/categories", h.GetCategoryMetrics)
}

// =============================================================================
// HANDLERS
// =============================================================================

// GenerateChart godoc
// @Summary Generate chart data
// @Description Generates chart data based on flexible configuration
// @Tags Analytics
// @Accept json
// @Produce json
// @Param request body ChartRequest true "Chart configuration"
// @Success 200 {object} ChartResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/charts [post]
func (h *Handler) GenerateChart(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	var req ChartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	chart, err := h.service.GenerateChart(c.Request.Context(), user.UserID, &req)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, chart)
}

// GetDashboard godoc
// @Summary Get dashboard metrics
// @Description Retrieves combined dashboard metrics for tasks, emotions, goals, categories
// @Tags Analytics
// @Produce json
// @Param period query string false "Time period (day, week, month, quarter, year)" default(week)
// @Success 200 {object} DashboardResponse
// @Failure 401 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	period := c.DefaultQuery("period", PeriodWeek)

	dashboard, err := h.service.GetDashboard(c.Request.Context(), user.UserID, period)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, dashboard)
}

// GetTaskMetrics godoc
// @Summary Get task productivity metrics
// @Description Retrieves task completion rate, velocity, peak hours, category breakdown
// @Tags Analytics
// @Produce json
// @Param period query string false "Time period (day, week, month, quarter, year)" default(week)
// @Param start_date query string false "Start date for custom period (RFC3339)"
// @Param end_date query string false "End date for custom period (RFC3339)"
// @Success 200 {object} TaskMetrics
// @Failure 401 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/metrics/tasks [get]
func (h *Handler) GetTaskMetrics(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	period := c.DefaultQuery("period", PeriodWeek)
	start, end := h.parseDateRange(c)

	metrics, err := h.service.GetTaskMetrics(c.Request.Context(), user.UserID, period, start, end)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, metrics)
}

// GetEmotionMetrics godoc
// @Summary Get emotion/mood metrics
// @Description Retrieves valence, arousal, mood stability, quadrant distribution, top emotions
// @Tags Analytics
// @Produce json
// @Param period query string false "Time period (day, week, month, quarter, year)" default(week)
// @Param start_date query string false "Start date for custom period (RFC3339)"
// @Param end_date query string false "End date for custom period (RFC3339)"
// @Success 200 {object} EmotionMetrics
// @Failure 401 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/metrics/emotions [get]
func (h *Handler) GetEmotionMetrics(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	period := c.DefaultQuery("period", PeriodWeek)
	start, end := h.parseDateRange(c)

	metrics, err := h.service.GetEmotionMetrics(c.Request.Context(), user.UserID, period, start, end)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, metrics)
}

// GetGoalMetrics godoc
// @Summary Get goal progress metrics
// @Description Retrieves goal progress, streaks, and completion stats
// @Tags Analytics
// @Produce json
// @Success 200 {object} GoalMetrics
// @Failure 401 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/metrics/goals [get]
func (h *Handler) GetGoalMetrics(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	metrics, err := h.service.GetGoalMetrics(c.Request.Context(), user.UserID)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, metrics)
}

// GetCategoryMetrics godoc
// @Summary Get category time distribution
// @Description Retrieves time spent per category with percentages
// @Tags Analytics
// @Produce json
// @Param period query string false "Time period (day, week, month, quarter, year)" default(week)
// @Param start_date query string false "Start date for custom period (RFC3339)"
// @Param end_date query string false "End date for custom period (RFC3339)"
// @Success 200 {object} CategoryMetrics
// @Failure 401 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/metrics/categories [get]
func (h *Handler) GetCategoryMetrics(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	period := c.DefaultQuery("period", PeriodWeek)
	start, end := h.parseDateRange(c)

	metrics, err := h.service.GetCategoryMetrics(c.Request.Context(), user.UserID, period, start, end)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, metrics)
}

// =============================================================================
// HELPERS
// =============================================================================

func (h *Handler) parseDateRange(c *gin.Context) (*time.Time, *time.Time) {
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	var start, end *time.Time

	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = &t
		}
	}

	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = &t
		}
	}

	return start, end
}
