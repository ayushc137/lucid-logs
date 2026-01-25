package activities

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles HTTP requests for activities.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new activities Handler.
func NewHandler(service Service) *Handler {
	return &Handler{
		service:   service,
		validator: validator.New(),
	}
}

// RegisterRoutes registers the activity routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	activities := rg.Group("/activities")
	{
		activities.GET("", h.List)
		activities.GET("/pinned", h.GetPinned)
		activities.POST("", h.Create)
		activities.GET("/:id", h.Get)
		activities.PUT("/:id", h.Update)
		activities.DELETE("/:id", h.Delete)

		// Actions
		activities.POST("/:id/instant", h.InstantLog)
		activities.POST("/:id/schedule", h.Schedule)

		// Goal linking
		activities.GET("/:id/goals", h.GetLinkedGoals)
		activities.POST("/:id/goals", h.LinkGoal)
		activities.DELETE("/:id/goals/:goalId", h.UnlinkGoal)
	}
}

// =============================================================================
// LIST
// =============================================================================

// List handles GET /activities - list all activities.
//
// @Summary      List activities
// @Description  Get paginated list of user's activities
// @Tags         activities
// @Produce      json
// @Param        limit  query   int  false "Items per page" default(20)
// @Param        offset query   int  false "Offset" default(0)
// @Success      200 {object} ActivityPageResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activities [get]
func (h *Handler) List(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	params := pagination.FromRequest(c)

	result, err := h.service.List(c.Request.Context(), user.UserID, params)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, result)
}

// =============================================================================
// GET PINNED
// =============================================================================

// GetPinned handles GET /activities/pinned - get pinned activities for quick bar.
//
// @Summary      Get pinned activities
// @Description  Get activities pinned to quick access bar
// @Tags         activities
// @Produce      json
// @Success      200 {array} Activity
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activities/pinned [get]
func (h *Handler) GetPinned(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	activities, err := h.service.GetPinned(c.Request.Context(), user.UserID)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, activities)
}

// =============================================================================
// GET
// =============================================================================

// Get handles GET /activities/:id - get a single activity.
//
// @Summary      Get activity
// @Description  Get a single activity by ID
// @Tags         activities
// @Produce      json
// @Param        id path string true "Activity ID"
// @Success      200 {object} Activity
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activities/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	activityID := c.Param("id")

	activity, err := h.service.Get(c.Request.Context(), activityID, user.UserID)
	if err != nil {
		handleActivityError(c, err)
		return
	}

	response.OK(c, activity)
}

// =============================================================================
// CREATE
// =============================================================================

// Create handles POST /activities - create a new activity.
//
// @Summary      Create activity
// @Description  Create a new activity
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        request body CreateRequest true "Activity data"
// @Success      201 {object} Activity
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activities [post]
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

	activity, err := h.service.Create(c.Request.Context(), &req, user.UserID)
	if err != nil {
		handleActivityError(c, err)
		return
	}

	response.Created(c, activity)
}

// =============================================================================
// UPDATE
// =============================================================================

// Update handles PUT /activities/:id - update an activity.
//
// @Summary      Update activity
// @Description  Update an existing activity
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        id      path   string        true "Activity ID"
// @Param        request body   UpdateRequest true "Update data"
// @Success      200 {object} Activity
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activities/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	activityID := c.Param("id")

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	activity, err := h.service.Update(c.Request.Context(), activityID, &req, user.UserID)
	if err != nil {
		handleActivityError(c, err)
		return
	}

	response.OK(c, activity)
}

// =============================================================================
// DELETE
// =============================================================================

// Delete handles DELETE /activities/:id - soft delete an activity.
//
// @Summary      Delete activity
// @Description  Soft delete an activity
// @Tags         activities
// @Produce      json
// @Param        id path string true "Activity ID"
// @Success      200 {object} response.OperationMessage
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activities/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	activityID := c.Param("id")

	err := h.service.Delete(c.Request.Context(), activityID, user.UserID)
	if err != nil {
		handleActivityError(c, err)
		return
	}

	response.Message(c, http.StatusOK, "Activity deleted")
}

// =============================================================================
// INSTANT LOG
// =============================================================================

// InstantLog handles POST /activities/:id/instant - create a task immediately.
//
// @Summary      Instant log
// @Description  Create a completed task from an activity immediately
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        id      path   string           true "Activity ID"
// @Param        request body   InstantLogRequest false "Log options"
// @Success      200 {object} InstantLogResponse
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activities/{id}/instant [post]
func (h *Handler) InstantLog(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	activityID := c.Param("id")

	var req InstantLogRequest
	// Allow empty body
	_ = c.ShouldBindJSON(&req)

	result, err := h.service.InstantLog(c.Request.Context(), activityID, &req, user.UserID)
	if err != nil {
		handleActivityError(c, err)
		return
	}

	response.OK(c, result)
}

// =============================================================================
// SCHEDULE
// =============================================================================

// Schedule handles POST /activities/:id/schedule - get pre-filled task data.
//
// @Summary      Schedule task
// @Description  Get pre-filled task data for the task form
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        id      path   string          true "Activity ID"
// @Param        request body   ScheduleRequest false "Schedule options"
// @Success      200 {object} ScheduleResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activities/{id}/schedule [post]
func (h *Handler) Schedule(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	activityID := c.Param("id")

	var req ScheduleRequest
	// Allow empty body
	_ = c.ShouldBindJSON(&req)

	result, err := h.service.Schedule(c.Request.Context(), activityID, &req, user.UserID)
	if err != nil {
		handleActivityError(c, err)
		return
	}

	response.OK(c, result)
}

// =============================================================================
// GOAL LINKING
// =============================================================================

// GetLinkedGoals handles GET /activities/:id/goals - get linked goals.
//
// @Summary      Get linked goals
// @Description  Get all goals linked to an activity
// @Tags         activities
// @Produce      json
// @Param        id path string true "Activity ID"
// @Success      200 {array} ActivityGoalLinkDetail
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activities/{id}/goals [get]
func (h *Handler) GetLinkedGoals(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	activityID := c.Param("id")

	goals, err := h.service.GetLinkedGoals(c.Request.Context(), activityID, user.UserID)
	if err != nil {
		handleActivityError(c, err)
		return
	}

	response.OK(c, goals)
}

// LinkGoal handles POST /activities/:id/goals - link a goal.
//
// @Summary      Link goal
// @Description  Link a goal to an activity
// @Tags         activities
// @Accept       json
// @Produce      json
// @Param        id      path   string        true "Activity ID"
// @Param        request body   GoalLinkInput true "Goal link config"
// @Success      200 {object} response.OperationMessage
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activities/{id}/goals [post]
func (h *Handler) LinkGoal(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	activityID := c.Param("id")

	var req GoalLinkInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	err := h.service.LinkGoal(c.Request.Context(), activityID, &req, user.UserID)
	if err != nil {
		handleActivityError(c, err)
		return
	}

	response.Message(c, http.StatusOK, "Goal linked")
}

// UnlinkGoal handles DELETE /activities/:id/goals/:goalId - unlink a goal.
//
// @Summary      Unlink goal
// @Description  Remove a goal link from an activity
// @Tags         activities
// @Produce      json
// @Param        id     path string true "Activity ID"
// @Param        goalId path string true "Goal ID"
// @Success      200 {object} response.OperationMessage
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/activities/{id}/goals/{goalId} [delete]
func (h *Handler) UnlinkGoal(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	activityID := c.Param("id")
	goalID := c.Param("goalId")

	err := h.service.UnlinkGoal(c.Request.Context(), activityID, goalID, user.UserID)
	if err != nil {
		handleActivityError(c, err)
		return
	}

	response.Message(c, http.StatusOK, "Goal unlinked")
}

// =============================================================================
// ERROR HANDLING
// =============================================================================

func handleActivityError(c *gin.Context, err error) {
	if errors.Is(err, errors.ErrNotFound) {
		response.NotFound(c)
		return
	}
	if appErr := errors.AsAppError(err); appErr != nil {
		response.Error(c, appErr)
	} else {
		response.ErrorFromErr(c, err)
	}
}
