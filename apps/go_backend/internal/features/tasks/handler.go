package tasks

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/daily-journal/go-backend/internal/shared/errors"
	"github.com/daily-journal/go-backend/internal/shared/middleware"
	"github.com/daily-journal/go-backend/internal/shared/pagination"
	"github.com/daily-journal/go-backend/internal/shared/response"
	"github.com/daily-journal/go-backend/internal/shared/validator"
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

// Routes returns the task routes.
//
// Routes registered:
//   - GET    /        : List tasks with pagination
//   - POST   /        : Create a new task
//   - GET    /{id}    : Get task by ID
//   - PUT    /{id}    : Update task
//   - DELETE /{id}    : Soft delete task
func Routes(service Service, validator *validator.Validator) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(service, validator)

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
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
// @Success      200 {object} pagination.Response[Task]
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/tasks [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	// Parse pagination
	params := pagination.FromRequest(r)

	log.Debug().
		Str("user_id", user.UserID).
		Int("limit", params.Limit).
		Int("offset", params.Offset).
		Msg("listing tasks")

	// Get tasks
	resp, err := h.service.List(r.Context(), user.UserID, params)
	if err != nil {
		response.ErrorFromErr(w, err)
		return
	}

	response.OK(w, resp)
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
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	taskID := chi.URLParam(r, "id")

	task, err := h.service.Get(r.Context(), taskID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.ErrorFromErr(w, err)
		return
	}

	response.OK(w, task)
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
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	// Parse request
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate
	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(w, errs)
		return
	}

	// Create task
	task, err := h.service.Create(r.Context(), &req, user.UserID)
	if err != nil {
		handleTaskError(w, err)
		return
	}

	response.Created(w, task)
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
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	taskID := chi.URLParam(r, "id")

	// Parse request
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate
	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(w, errs)
		return
	}

	// Update task
	task, err := h.service.Update(r.Context(), taskID, &req, user.UserID)
	if err != nil {
		handleTaskError(w, err)
		return
	}

	response.OK(w, task)
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
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	taskID := chi.URLParam(r, "id")

	err := h.service.Delete(r.Context(), taskID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.ErrorFromErr(w, err)
		return
	}

	response.Message(w, http.StatusOK, "Task deleted")
}

// =============================================================================
// ERROR HANDLING
// =============================================================================

// handleTaskError handles common task errors.
func handleTaskError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errors.ErrNotFound):
		response.NotFound(w)
	case errors.Is(err, errors.ErrInvalidDateRange):
		response.Error(w, errors.ErrInvalidDateRange)
	case errors.Is(err, errors.ErrCategoryNotFound):
		response.Error(w, errors.ErrCategoryNotFound)
	default:
		response.ErrorFromErr(w, err)
	}
}
