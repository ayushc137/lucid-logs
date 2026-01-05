package units

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

// Handler handles unit HTTP endpoints.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new unit Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the unit routes.
func RegisterRoutes(r *gin.RouterGroup, service Service, validator *validator.Validator) {
	h := NewHandler(service, validator)

	r.GET("", h.List)
	r.POST("", h.Create)
	r.GET("/:id", h.Get)
	r.PUT("/:id", h.Update)
	r.DELETE("/:id", h.Delete)
}

// =============================================================================
// HANDLERS
// =============================================================================

// List handles GET /units
//
// @Summary      List units
// @Description  Get all available units (system + user's custom units)
// @Tags         units
// @Produce      json
// @Param        system_only query bool false "Only return system-provided units"
// @Success      200 {object} UnitListResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/units [get]
func (h *Handler) List(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	systemOnly := c.Query("system_only") == "true"

	resp, err := h.service.List(c.Request.Context(), user.UserID, systemOnly)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, resp)
}

// Get handles GET /units/{id}
//
// @Summary      Get unit
// @Description  Get a single unit by ID
// @Tags         units
// @Produce      json
// @Param        id path string true "Unit ID (e.g., km, mi, hr)"
// @Success      200 {object} Unit
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/units/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	_, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	unitID := c.Param("id")

	unit, err := h.service.Get(c.Request.Context(), unitID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, unit)
}

// Create handles POST /units
//
// @Summary      Create custom unit
// @Description  Create a custom unit for tracking quantities
// @Tags         units
// @Accept       json
// @Produce      json
// @Param        request body CreateRequest true "Unit data"
// @Success      201 {object} Unit
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/units [post]
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

	unit, err := h.service.Create(c.Request.Context(), &req, user.UserID)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.Created(c, unit)
}

// Update handles PUT /units/{id}
//
// @Summary      Update custom unit
// @Description  Update name or symbol of a custom unit (cannot modify system units)
// @Tags         units
// @Accept       json
// @Produce      json
// @Param        id      path string        true "Unit ID"
// @Param        request body UpdateRequest true "Update payload"
// @Success      200 {object} Unit
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      403 {object} response.APIResponse "Cannot modify system units"
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/units/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	unitID := c.Param("id")

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	unit, err := h.service.Update(c.Request.Context(), unitID, &req, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		if errors.Is(err, errors.ErrForbidden) {
			response.Forbidden(c, "Cannot modify system units")
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, unit)
}

// Delete handles DELETE /units/{id}
//
// @Summary      Delete custom unit
// @Description  Delete a custom unit (cannot delete system units)
// @Tags         units
// @Produce      json
// @Param        id path string true "Unit ID"
// @Success      200 {object} response.OperationMessage
// @Failure      401 {object} response.APIResponse
// @Failure      403 {object} response.APIResponse "Cannot delete system units"
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/units/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	unitID := c.Param("id")

	err := h.service.Delete(c.Request.Context(), unitID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		if errors.Is(err, errors.ErrForbidden) {
			response.Forbidden(c, "Cannot delete system units")
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.Message(c, http.StatusOK, "Unit deleted")
}
