// Package goallogs provides goal logs HTTP endpoints.
package goallogs

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/lucid-logs/go-backend/internal/shared/response"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles goal logs HTTP endpoints.
type Handler struct {
	repo Repository
}

// NewHandler creates a new goal logs Handler.
func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the goal logs routes under /goals/:id/logs.
//
// Routes registered:
//   - GET /goals/:id/logs : List logs for a goal
//   - GET /goals/:id/logs/summary : Get aggregated summary
func RegisterRoutes(r *gin.RouterGroup, repo Repository) {
	h := NewHandler(repo)

	r.GET("/:id/logs", h.List)
	r.GET("/:id/logs/summary", h.GetSummary)
}

// =============================================================================
// LIST
// =============================================================================

// List handles GET /goals/:id/logs - list logs for a goal.
//
// @Summary      List goal logs
// @Description  Get paginated list of logs for a specific goal
// @Tags         goal-logs
// @Produce      json
// @Param        id path string true "Goal ID"
// @Param        limit   query int false "Items per page (default 20, max 100)"
// @Param        offset  query int false "Items to skip (default 0)"
// @Success      200 {object} GoalLogsResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/goals/{id}/logs [get]
func (h *Handler) List(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")
	params := pagination.FromRequest(c)

	logs, total, err := h.repo.FindByGoal(c.Request.Context(), goalID, user.UserID, params)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, GoalLogsResponse{
		GoalID: goalID,
		Logs:   logs,
		Total:  int(total),
	})
}

// =============================================================================
// GET SUMMARY
// =============================================================================

// GetSummary handles GET /goals/:id/logs/summary - get aggregated summary.
//
// @Summary      Get goal logs summary
// @Description  Get aggregated history summary for a goal
// @Tags         goal-logs
// @Produce      json
// @Param        id path string true "Goal ID"
// @Param        days    query int false "Number of days to include (default 30)"
// @Success      200 {object} GoalLogsSummary
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/goals/{id}/logs/summary [get]
func (h *Handler) GetSummary(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")

	// Default to 30 days
	days := 30
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	summary, err := h.repo.GetSummary(c.Request.Context(), goalID, user.UserID, days)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, summary)
}
