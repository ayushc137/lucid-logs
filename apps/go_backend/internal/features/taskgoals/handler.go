// Package taskgoals provides task-goal linking endpoints.
package taskgoals

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles task-goal linking HTTP endpoints.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new task-goal linking Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the task-goal linking routes.
//
// Routes registered (under /tasks/:taskId/goals):
//   - POST   /        : Link task to goal(s)
//   - DELETE /:goalId : Unlink task from goal
func RegisterRoutes(r *gin.RouterGroup, service Service, validator *validator.Validator) {
	h := NewHandler(service, validator)

	r.POST("/", h.Link)
	r.DELETE("/:goalId", h.Unlink)
}

// =============================================================================
// LINK
// =============================================================================

// Link handles POST /tasks/:taskId/goals - link task to goal(s).
//
// @Summary      Link task to goal
// @Description  Create a link between a task and one or more goals with impact tracking
// @Tags         task-goals
// @Accept       json
// @Produce      json
// @Param        taskId  path   string      true "Task ID"
// @Param        request body   LinkRequest true "Link data (single goal)"
// @Success      201 {object} TaskGoal
// @Failure      400 {object} response.APIResponse "Invalid request or validation failed"
// @Failure      401 {object} response.APIResponse "Authentication required"
// @Failure      404 {object} response.APIResponse "Task or goal not found"
// @Failure      409 {object} response.APIResponse "Link already exists"
// @Failure      500 {object} response.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/tasks/{taskId}/goals [post]
func (h *Handler) Link(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		response.BadRequest(c, "Task ID is required")
		return
	}

	// Try single link first
	var req LinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	log.Debug().
		Str("user_id", user.UserID).
		Str("task_id", taskID).
		Str("goal_id", req.GoalID).
		Msg("linking task to goal")

	link, err := h.service.Link(c.Request.Context(), taskID, &req, user.UserID)
	if err != nil {
		handleLinkError(c, err)
		return
	}

	response.Created(c, link)
}

// =============================================================================
// UNLINK
// =============================================================================

// Unlink handles DELETE /tasks/:taskId/goals/:goalId - unlink task from goal.
//
// @Summary      Unlink task from goal
// @Description  Remove the link between a task and a goal
// @Tags         task-goals
// @Produce      json
// @Param        taskId path string true "Task ID"
// @Param        goalId path string true "Goal ID"
// @Success      200 {object} response.OperationMessage "Successfully unlinked"
// @Failure      401 {object} response.APIResponse "Authentication required"
// @Failure      404 {object} response.APIResponse "Task, goal, or link not found"
// @Failure      500 {object} response.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/tasks/{taskId}/goals/{goalId} [delete]
func (h *Handler) Unlink(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	taskID := c.Param("id")
	goalID := c.Param("goalId")

	if taskID == "" || goalID == "" {
		response.BadRequest(c, "Task ID and Goal ID are required")
		return
	}

	log.Debug().
		Str("user_id", user.UserID).
		Str("task_id", taskID).
		Str("goal_id", goalID).
		Msg("unlinking task from goal")

	err := h.service.Unlink(c.Request.Context(), taskID, goalID, user.UserID)
	if err != nil {
		handleLinkError(c, err)
		return
	}

	response.Message(c, http.StatusOK, "Task unlinked from goal")
}

// =============================================================================
// ERROR HANDLING
// =============================================================================

func handleLinkError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errors.ErrNotFound):
		response.NotFound(c)
	case errors.Is(err, errors.ErrConflict):
		response.Error(c, err.(*errors.AppError))
	case errors.Is(err, errors.ErrBadRequest):
		response.Error(c, err.(*errors.AppError))
	default:
		response.ErrorFromErr(c, err)
	}
}
