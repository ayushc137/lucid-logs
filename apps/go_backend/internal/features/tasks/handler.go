// Package tasks provides task management endpoints.
package tasks

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

// Handler handles task HTTP endpoints.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new task Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the task routes.
//
// Routes registered:
//   - GET    /        : List tasks with pagination
//   - POST   /        : Create a new task
//   - GET    /{id}    : Get task by ID
//   - PUT    /{id}    : Update task
//   - DELETE /{id}    : Soft delete task
func RegisterRoutes(r *gin.RouterGroup, service Service, validator *validator.Validator) {
	h := NewHandler(service, validator)

	r.GET("/", h.List)
	r.POST("/", h.Create)
	r.GET("/:id", h.Get)
	r.PUT("/:id", h.Update)
	r.DELETE("/:id", h.Delete)
}

// =============================================================================
// LIST
// =============================================================================

// List handles GET /tasks - list tasks with pagination.
//
// @Summary      List tasks
// @Description  Get paginated list of tasks for the authenticated user
// @Tags         tasks
// @Produce      json
// @Param        limit  query int false "Items per page (default 20, max 100)"
// @Param        offset query int false "Items to skip (default 0)"
// @Success      200 {object} tasks.TaskPageResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/tasks [get]
func (h *Handler) List(c *gin.Context) {
	// Get authenticated user
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	// Parse pagination
	params := pagination.FromRequest(c)

	log.Debug().
		Str("user_id", user.UserID).
		Int("limit", params.Limit).
		Int("offset", params.Offset).
		Msg("listing tasks")

	// Get tasks
	resp, err := h.service.List(c.Request.Context(), user.UserID, params)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, resp)
}

// =============================================================================
// GET
// =============================================================================

// Get handles GET /tasks/{id} - get task by ID.
//
// @Summary      Get task by ID
// @Description  Get a single task by its ID
// @Tags         tasks
// @Produce      json
// @Param        id path string true "Task ID"
// @Success      200 {object} Task
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/tasks/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	taskID := c.Param("id")

	task, err := h.service.Get(c.Request.Context(), taskID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, task)
}

// =============================================================================
// CREATE
// =============================================================================

// Create handles POST /tasks - create a new task.
//
// @Summary      Create task
// @Description  Create a new task
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        request body CreateRequest true "Task data"
// @Success      201 {object} Task
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/tasks [post]
func (h *Handler) Create(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	// Parse request
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	// Validate
	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	// Create task
	task, err := h.service.Create(c.Request.Context(), &req, user.UserID)
	if err != nil {
		handleTaskError(c, err)
		return
	}

	response.Created(c, task)
}

// =============================================================================
// UPDATE
// =============================================================================

// Update handles PUT /tasks/{id} - update a task.
//
// @Summary      Update task
// @Description  Update an existing task
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        id      path string        true "Task ID"
// @Param        request body UpdateRequest true "Update data"
// @Success      200 {object} Task
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/tasks/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	taskID := c.Param("id")

	// Parse request
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	// Validate
	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	// Update task
	task, err := h.service.Update(c.Request.Context(), taskID, &req, user.UserID)
	if err != nil {
		handleTaskError(c, err)
		return
	}

	response.OK(c, task)
}

// =============================================================================
// DELETE
// =============================================================================

// Delete handles DELETE /tasks/{id} - soft delete a task.
//
// @Summary      Delete task
// @Description  Soft delete a task
// @Tags         tasks
// @Produce      json
// @Param        id path string true "Task ID"
// @Success      200 {object} response.OperationMessage
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/tasks/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	taskID := c.Param("id")

	err := h.service.Delete(c.Request.Context(), taskID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.Message(c, http.StatusOK, "Task deleted")
}

// =============================================================================
// ERROR HANDLING
// =============================================================================

// handleTaskError handles common task errors.
func handleTaskError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errors.ErrNotFound):
		response.NotFound(c)
	case errors.Is(err, errors.ErrInvalidDateRange):
		response.Error(c, errors.ErrInvalidDateRange)
	case errors.Is(err, errors.ErrCategoryNotFound):
		response.Error(c, errors.ErrCategoryNotFound)
	default:
		response.ErrorFromErr(c, err)
	}
}
