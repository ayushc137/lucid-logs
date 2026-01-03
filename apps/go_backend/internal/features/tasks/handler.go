// Package tasks provides task management endpoints.
package tasks

import (
	"net/http"
	"strconv"
	"time"

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

	r.GET("", h.List)
	r.POST("", h.Create)
	r.GET("/last-end-time", h.GetLastTaskEndTime)
	r.GET("/:id", h.Get)
	r.PUT("/:id", h.Update)
	r.DELETE("/:id", h.Delete)
}

// =============================================================================
// GET LAST TASK END TIME
// =============================================================================

// GetLastTaskEndTime handles GET /tasks/last-end-time - get end time of last finished task.
//
// @Summary      Get last task end time
// @Description  Get the end time of the most recently finished task
// @Tags         tasks
// @Produce      json
// @Success      200 {object} map[string]string
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/tasks/last-end-time [get]
func (h *Handler) GetLastTaskEndTime(c *gin.Context) {
	// Get authenticated user
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	endTime, err := h.service.GetLastTaskEndTime(c.Request.Context(), user.UserID)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	if endTime == nil {
		response.OK(c, map[string]any{"end_time": nil})
		return
	}

	response.OK(c, map[string]any{"end_time": endTime.Format(time.RFC3339)})
}

// =============================================================================
// LIST
// =============================================================================

// List handles GET /tasks - list tasks with pagination, filters, and search.
//
// @Summary      List tasks
// @Description  Get paginated list of tasks with optional filters and full-text search
// @Tags         tasks
// @Produce      json
// @Param        limit          query int    false "Items per page (default 20, max 100)"
// @Param        offset         query int    false "Items to skip (default 0)"
// @Param        search         query string false "Full-text search across title, journal, note"
// @Param        category_id    query string false "Filter by category ID"
// @Param        status         query string false "Filter by status: all, completed, pending"
// @Param        priority_min   query int    false "Filter by minimum priority (1-10)"
// @Param        priority_max   query int    false "Filter by maximum priority (1-10)"
// @Param        start_date_from query string false "Filter tasks starting on or after (RFC3339)"
// @Param        start_date_to   query string false "Filter tasks starting on or before (RFC3339)"
// @Param        sort_field     query string false "Sort by: start_date, priority, title, created_at"
// @Param        sort_order     query string false "Sort direction: asc or desc"
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

	// Parse filter parameters
	filters := parseFilterParams(c)

	// Check if any filters are active
	hasFilters := filters.Search != "" ||
		filters.CategoryID != "" ||
		filters.NoCategoryFilter ||
		filters.Status != "" ||
		filters.PriorityMin != nil ||
		filters.PriorityMax != nil ||
		filters.StartDateFrom != "" ||
		filters.StartDateTo != "" ||
		filters.SortField != ""

	log.Debug().
		Str("user_id", user.UserID).
		Int("limit", params.Limit).
		Int("offset", params.Offset).
		Str("search", filters.Search).
		Str("category", filters.CategoryID).
		Str("status", filters.Status).
		Bool("has_filters", hasFilters).
		Msg("listing tasks")

	var resp *pagination.Response[*Task]
	var err error

	if hasFilters {
		// Use filtered query with FTS support
		resp, err = h.service.ListFiltered(c.Request.Context(), user.UserID, filters, params)
	} else {
		// Use simple paginated query for better performance
		resp, err = h.service.List(c.Request.Context(), user.UserID, params)
	}

	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, resp)
}

// parseFilterParams extracts filter parameters from the request query string.
func parseFilterParams(c *gin.Context) TaskFilterParams {
	filters := TaskFilterParams{
		Search:           c.Query("search"),
		CategoryID:       c.Query("category_id"),
		NoCategoryFilter: c.Query("no_category") == "true",
		Status:           c.Query("status"),
		StartDateFrom:    c.Query("start_date_from"),
		StartDateTo:      c.Query("start_date_to"),
		SortField:        c.Query("sort_field"),
		SortOrder:        c.Query("sort_order"),
	}

	// Parse priority_min
	if minStr := c.Query("priority_min"); minStr != "" {
		if min, err := strconv.Atoi(minStr); err == nil && min >= 1 && min <= 10 {
			filters.PriorityMin = &min
		}
	}

	// Parse priority_max
	if maxStr := c.Query("priority_max"); maxStr != "" {
		if max, err := strconv.Atoi(maxStr); err == nil && max >= 1 && max <= 10 {
			filters.PriorityMax = &max
		}
	}

	return filters
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
