package goalactions

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles HTTP requests for goal actions.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new goal actions Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTE REGISTRATION
// =============================================================================

// RegisterRoutes registers the goal action routes.
//
// Consolidated endpoints:
//   - GET    /goals/:id/actions            List all actions for a goal
//   - POST   /goals/:id/actions            Create a new action
//   - GET    /goals/:id/actions/:actionId  Get a single action
//   - PATCH  /goals/:id/actions/:actionId  Update an action (including completion)
//   - DELETE /goals/:id/actions/:actionId  Delete an action
//   - PATCH  /goals/:id/actions            Reorder actions (bulk operation)
func RegisterRoutes(rg *gin.RouterGroup, service Service, validator *validator.Validator) {
	h := NewHandler(service, validator)

	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.PATCH("", h.Reorder) // Bulk reorder
	rg.GET("/:actionId", h.Get)
	rg.PATCH("/:actionId", h.Update) // Update including completion status
	rg.DELETE("/:actionId", h.Delete)
}

// =============================================================================
// HANDLERS
// =============================================================================

// List godoc
// @Summary List goal actions
// @Description Retrieves all actions for a goal
// @Tags Goal Actions
// @Accept json
// @Produce json
// @Param id path string true "Goal ID"
// @Success 200 {object} ActionListResponse
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/goals/{id}/actions [get]
func (h *Handler) List(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")

	resp, err := h.service.List(c.Request.Context(), goalID, user.UserID)
	if err != nil {
		handleActionError(c, err)
		return
	}

	response.OK(c, resp)
}

// Get godoc
// @Summary Get a goal action
// @Description Retrieves a single goal action by ID
// @Tags Goal Actions
// @Accept json
// @Produce json
// @Param id path string true "Goal ID"
// @Param actionId path string true "Action ID"
// @Success 200 {object} GoalAction
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/goals/{id}/actions/{actionId} [get]
func (h *Handler) Get(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")
	actionID := c.Param("actionId")

	action, err := h.service.Get(c.Request.Context(), actionID, goalID, user.UserID)
	if err != nil {
		handleActionError(c, err)
		return
	}

	response.OK(c, action)
}

// Create godoc
// @Summary Create a goal action
// @Description Creates a new action for a goal
// @Tags Goal Actions
// @Accept json
// @Produce json
// @Param id path string true "Goal ID"
// @Param request body CreateRequest true "Action data"
// @Success 201 {object} GoalAction
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/goals/{id}/actions [post]
func (h *Handler) Create(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	action, err := h.service.Create(c.Request.Context(), goalID, &req, user.UserID)
	if err != nil {
		handleActionError(c, err)
		return
	}

	response.Created(c, action)
}

// Update godoc
// @Summary Update a goal action
// @Description Updates an existing goal action. Set "completed" field to mark complete/incomplete.
// @Tags Goal Actions
// @Accept json
// @Produce json
// @Param id path string true "Goal ID"
// @Param actionId path string true "Action ID"
// @Param request body UpdateRequest true "Action data (all fields optional)"
// @Success 200 {object} GoalAction
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/goals/{id}/actions/{actionId} [patch]
func (h *Handler) Update(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")
	actionID := c.Param("actionId")

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	action, err := h.service.Update(c.Request.Context(), actionID, goalID, &req, user.UserID)
	if err != nil {
		handleActionError(c, err)
		return
	}

	response.OK(c, action)
}

// Delete godoc
// @Summary Delete a goal action
// @Description Soft-deletes a goal action
// @Tags Goal Actions
// @Accept json
// @Produce json
// @Param id path string true "Goal ID"
// @Param actionId path string true "Action ID"
// @Success 200 {object} response.OperationMessage
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/goals/{id}/actions/{actionId} [delete]
func (h *Handler) Delete(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")
	actionID := c.Param("actionId")

	if err := h.service.Delete(c.Request.Context(), actionID, goalID, user.UserID); err != nil {
		handleActionError(c, err)
		return
	}

	response.Message(c, http.StatusOK, "Action deleted")
}

// Reorder godoc
// @Summary Reorder actions
// @Description Reorders actions within a goal
// @Tags Goal Actions
// @Accept json
// @Produce json
// @Param id path string true "Goal ID"
// @Param request body ReorderRequest true "Ordered list of action IDs"
// @Success 200 {object} response.OperationMessage
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/goals/{id}/actions [patch]
func (h *Handler) Reorder(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")

	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	if err := h.service.Reorder(c.Request.Context(), goalID, &req, user.UserID); err != nil {
		handleActionError(c, err)
		return
	}

	response.Message(c, http.StatusOK, "Actions reordered")
}

// =============================================================================
// ERROR HANDLING
// =============================================================================

func handleActionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errors.ErrNotFound):
		response.NotFound(c)
	default:
		response.ErrorFromErr(c, err)
	}
}
