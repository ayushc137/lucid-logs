// Package goals provides goal management endpoints.
package goals

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles goal HTTP endpoints.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new goal Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the goal routes.
//
// Routes registered:
//   - GET    /        : List goals with pagination
//   - POST   /        : Create a new goal
//   - GET    /today   : Get today's recurring goals with status
//   - GET    /{id}    : Get goal by ID
//   - PUT    /{id}    : Update goal
//   - DELETE /{id}    : Soft delete goal
func RegisterRoutes(r *gin.RouterGroup, service Service, validator *validator.Validator) {
	h := NewHandler(service, validator)

	r.GET("", h.List)
	r.POST("", h.Create)
	r.GET("/today", h.GetToday)
	r.GET("/:id", h.Get)
	r.PUT("/:id", h.Update)
	r.DELETE("/:id", h.Delete)
}

// =============================================================================
// LIST
// =============================================================================

// List handles GET /goals - list goals with pagination and filters.
//
// @Summary      List goals
// @Description  Get paginated list of goals for the authenticated user
// @Tags         goals
// @Produce      json
// @Param        limit      query int    false "Items per page (default 20, max 100)"
// @Param        offset     query int    false "Items to skip (default 0)"
// @Param        status     query string false "Filter by status (active, completed, paused, abandoned)"
// @Param        goal_type  query string false "Filter by type (discrete, measurable, epic, avoidance)"
// @Param        recurring  query bool   false "Filter recurring (true) or one-time (false)"
// @Param        search     query string false "Search in title and description"
// @Param        sort_by    query string false "Sort by field (created_at, title, streak, priority) with optional -desc suffix"
// @Success      200 {object} GoalPageResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/goals [get]
func (h *Handler) List(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	params := pagination.FromRequest(c)

	// Parse filters from query params
	filters := GoalFilters{
		Status: c.Query("status"),
		Search: c.Query("search"),
		SortBy: c.Query("sort_by"),
	}
	if recurring := c.Query("recurring"); recurring != "" {
		isRecurring := recurring == "true"
		filters.IsRecurring = &isRecurring
	}
	if hasTarget := c.Query("measurable"); hasTarget != "" {
		hasTgt := hasTarget == "true"
		filters.HasTarget = &hasTgt
	}
	if hasChildren := c.Query("grouped"); hasChildren != "" {
		hasChild := hasChildren == "true"
		filters.HasChildren = &hasChild
	}

	log.Debug().
		Str("user_id", user.UserID).
		Int("limit", params.Limit).
		Int("offset", params.Offset).
		Str("status", filters.Status).
		Msg("listing goals")

	resp, err := h.service.List(c.Request.Context(), user.UserID, params, filters)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, resp)
}

// =============================================================================
// GET
// =============================================================================

// Get handles GET /goals/{id} - get goal by ID.
//
// @Summary      Get goal by ID
// @Description  Get a single goal by its ID
// @Tags         goals
// @Produce      json
// @Param        id path string true "Goal ID"
// @Success      200 {object} Goal
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/goals/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")

	goal, err := h.service.Get(c.Request.Context(), goalID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, goal)
}

// =============================================================================
// GET TODAY
// =============================================================================

// GetToday handles GET /goals/today - get today's recurring goals with status.
//
// @Summary      Get today's goals
// @Description  Get recurring goals with today's completion status
// @Tags         goals
// @Produce      json
// @Success      200 {object} TodayGoalsResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/goals/today [get]
func (h *Handler) GetToday(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	resp, err := h.service.GetTodayGoals(c.Request.Context(), user.UserID)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, resp)
}

// =============================================================================
// CREATE
// =============================================================================

// Create handles POST /goals - create a new goal.
//
// @Summary      Create goal
// @Description  Create a new goal (auto-creates linked template for recurring goals)
// @Tags         goals
// @Accept       json
// @Produce      json
// @Param        request body CreateRequest true "Goal data"
// @Success      201 {object} Goal
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/goals [post]
func (h *Handler) Create(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	goal, err := h.service.Create(c.Request.Context(), &req, user.UserID)
	if err != nil {
		handleGoalError(c, err)
		return
	}

	response.Created(c, goal)
}

// =============================================================================
// UPDATE
// =============================================================================

// Update handles PUT /goals/{id} - update a goal.
//
// @Summary      Update goal
// @Description  Update an existing goal
// @Tags         goals
// @Accept       json
// @Produce      json
// @Param        id      path   string        true "Goal ID"
// @Param        request body   UpdateRequest true "Update data"
// @Success      200 {object} Goal
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/goals/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	goal, err := h.service.Update(c.Request.Context(), goalID, &req, user.UserID)
	if err != nil {
		handleGoalError(c, err)
		return
	}

	response.OK(c, goal)
}

// =============================================================================
// DELETE
// =============================================================================

// Delete handles DELETE /goals/{id} - soft delete a goal.
//
// @Summary      Delete goal
// @Description  Soft delete a goal
// @Tags         goals
// @Produce      json
// @Param        id path string true "Goal ID"
// @Success      200 {object} response.OperationMessage
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/goals/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")

	err := h.service.Delete(c.Request.Context(), goalID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.Message(c, http.StatusOK, "Goal deleted")
}

// =============================================================================
// ERROR HANDLING
// =============================================================================

func handleGoalError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errors.ErrNotFound):
		response.NotFound(c)
	case errors.Is(err, errors.ErrBadRequest):
		response.Error(c, err.(*errors.AppError))
	default:
		response.ErrorFromErr(c, err)
	}
}
