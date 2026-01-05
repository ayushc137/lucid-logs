// Package goalentries provides goal entry management endpoints.
package goalentries

import (
	stderrors "errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles goal entry HTTP endpoints.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new goal entry Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the goal entry routes under a goal group.
// Expected to be called with a route group like /goals/:goalId/entries
//
// Routes registered:
//   - POST   /        : Create/log an entry
//   - GET    /        : List entries for date range
func RegisterRoutes(r *gin.RouterGroup, service Service, validator *validator.Validator) {
	h := NewHandler(service, validator)

	r.POST("", h.Create)
	r.GET("", h.List)
}

// =============================================================================
// LIST
// =============================================================================

// List handles GET /goals/{goalId}/entries - list entries for a date range.
//
// @Summary      List goal entries
// @Description  Get entries for a goal within a date range
// @Tags         goal-entries
// @Produce      json
// @Param        goalId     path   string true  "Goal ID"
// @Param        start_date query  string true  "Start date (YYYY-MM-DD)"
// @Param        end_date   query  string true  "End date (YYYY-MM-DD)"
// @Success      200 {object} GoalEntryListResponse
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/goals/{goalId}/entries [get]
func (h *Handler) List(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	goalID := c.Param("id")

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		response.BadRequest(c, "start_date and end_date are required")
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		response.BadRequest(c, "Invalid start_date format, use YYYY-MM-DD")
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		response.BadRequest(c, "Invalid end_date format, use YYYY-MM-DD")
		return
	}

	log.Debug().
		Str("goal_id", goalID).
		Str("start_date", startDateStr).
		Str("end_date", endDateStr).
		Msg("listing goal entries")

	resp, err := h.service.List(c.Request.Context(), goalID, user.UserID, startDate, endDate)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, resp)
}

// =============================================================================
// CREATE
// =============================================================================

// Create handles POST /goals/{goalId}/entries - log a goal entry.
//
// @Summary      Log goal entry
// @Description  Create or update a daily entry for a goal
// @Tags         goal-entries
// @Accept       json
// @Produce      json
// @Param        goalId  path   string        true "Goal ID"
// @Param        request body   CreateRequest true "Entry data"
// @Success      201 {object} GoalEntry
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/goals/{goalId}/entries [post]
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

	entry, err := h.service.Create(c.Request.Context(), goalID, user.UserID, &req)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		if errors.Is(err, errors.ErrBadRequest) {
			var appErr *errors.AppError
			if stderrors.As(err, &appErr) {
				response.Error(c, appErr)
				return
			}
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.Created(c, entry)
}
