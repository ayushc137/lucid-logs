// Package activitylogs provides activity log HTTP endpoints.
package activitylogs

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/lucid-logs/go-backend/internal/shared/response"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles activity logs HTTP endpoints.
type Handler struct {
	repo Repository
}

// NewHandler creates a new activity logs Handler.
func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the activity logs routes.
//
// Routes registered:
//   - GET /activity          : List all activity for user
//   - GET /activity/goals    : List goal activity
//   - GET /activity/tasks    : List task activity
func RegisterRoutes(r *gin.RouterGroup, repo Repository) {
	h := NewHandler(repo)

	r.GET("", h.List)
	r.GET("/goals", h.ListGoalActivity)
	r.GET("/tasks", h.ListTaskActivity)
}

// =============================================================================
// LIST
// =============================================================================

// List handles GET /activity - list all activity.
//
// @Summary      List activity logs
// @Description  Get paginated list of all activity for the authenticated user
// @Tags         activity
// @Produce      json
// @Param        limit  query int false "Items per page (default 20, max 100)"
// @Param        offset query int false "Items to skip (default 0)"
// @Param        days   query int false "Number of days to include (default 30)"
// @Success      200 {object} ActivityLogsResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activity [get]
func (h *Handler) List(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	params := pagination.FromRequest(c)
	entityType := c.Query("type") // Optional filter by entity type

	logs, total, err := h.repo.FindByUser(c.Request.Context(), user.UserID, params, entityType)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, ActivityLogsResponse{
		Logs:  logs,
		Total: total,
	})
}

// ListGoalActivity handles GET /activity/goals - list goal activity.
//
// @Summary      List goal activity
// @Description  Get activity logs for goals only
// @Tags         activity
// @Produce      json
// @Param        limit  query int false "Items per page (default 20, max 100)"
// @Param        offset query int false "Items to skip (default 0)"
// @Success      200 {object} ActivityLogsResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activity/goals [get]
func (h *Handler) ListGoalActivity(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	params := pagination.FromRequest(c)

	logs, total, err := h.repo.FindByUser(c.Request.Context(), user.UserID, params, EntityTypeGoal)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, ActivityLogsResponse{
		Logs:  logs,
		Total: total,
	})
}

// ListTaskActivity handles GET /activity/tasks - list task activity.
//
// @Summary      List task activity
// @Description  Get activity logs for tasks only
// @Tags         activity
// @Produce      json
// @Param        limit  query int false "Items per page (default 20, max 100)"
// @Param        offset query int false "Items to skip (default 0)"
// @Success      200 {object} ActivityLogsResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activity/tasks [get]
func (h *Handler) ListTaskActivity(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	params := pagination.FromRequest(c)

	logs, total, err := h.repo.FindByUser(c.Request.Context(), user.UserID, params, EntityTypeTask)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, ActivityLogsResponse{
		Logs:  logs,
		Total: total,
	})
}

// Unused import fix
var _ = strconv.Atoi
