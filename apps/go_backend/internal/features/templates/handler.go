// Package templates provides template management endpoints.
package templates

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles template HTTP endpoints.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new template Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the template routes.
//
// Routes registered:
//   - GET    /           : List templates with pagination
//   - POST   /           : Create a new template
//   - GET    /quick-log  : Get quick-log templates
//   - GET    /{id}       : Get template by ID
//   - PUT    /{id}       : Update template
//   - DELETE /{id}       : Soft delete template
func RegisterRoutes(r *gin.RouterGroup, service Service, validator *validator.Validator) {
	h := NewHandler(service, validator)

	r.GET("", h.List)
	r.POST("", h.Create)
	r.GET("/quick-log", h.GetQuickLog)
	r.GET("/:id", h.Get)
	r.PUT("/:id", h.Update)
	r.DELETE("/:id", h.Delete)
}

// =============================================================================
// LIST
// =============================================================================

// List handles GET /templates - list templates with pagination.
//
// @Summary      List templates
// @Description  Get paginated list of templates for the authenticated user
// @Tags         templates
// @Produce      json
// @Param        limit  query int false "Items per page (default 20, max 100)"
// @Param        offset query int false "Items to skip (default 0)"
// @Success      200 {object} TemplatePageResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/templates [get]
func (h *Handler) List(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	params := pagination.FromRequest(c)

	log.Debug().
		Str("user_id", user.UserID).
		Int("limit", params.Limit).
		Int("offset", params.Offset).
		Msg("listing templates")

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

// Get handles GET /templates/{id} - get template by ID.
//
// @Summary      Get template by ID
// @Description  Get a single template by its ID
// @Tags         templates
// @Produce      json
// @Param        id path string true "Template ID"
// @Success      200 {object} TaskTemplate
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/templates/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	templateID := c.Param("id")

	template, err := h.service.Get(c.Request.Context(), templateID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, template)
}

// =============================================================================
// GET QUICK LOG
// =============================================================================

// GetQuickLog handles GET /templates/quick-log - get quick-log templates.
//
// @Summary      Get quick-log templates
// @Description  Get templates configured for quick logging
// @Tags         templates
// @Produce      json
// @Success      200 {array} TaskTemplate
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/templates/quick-log [get]
func (h *Handler) GetQuickLog(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	templates, err := h.service.GetQuickLog(c.Request.Context(), user.UserID)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, templates)
}

// =============================================================================
// CREATE
// =============================================================================

// Create handles POST /templates - create a new template.
//
// @Summary      Create template
// @Description  Create a new task template
// @Tags         templates
// @Accept       json
// @Produce      json
// @Param        request body CreateRequest true "Template data"
// @Success      201 {object} TaskTemplate
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/templates [post]
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

	template, err := h.service.Create(c.Request.Context(), &req, user.UserID)
	if err != nil {
		handleTemplateError(c, err)
		return
	}

	response.Created(c, template)
}

// =============================================================================
// UPDATE
// =============================================================================

// Update handles PUT /templates/{id} - update a template.
//
// @Summary      Update template
// @Description  Update an existing template
// @Tags         templates
// @Accept       json
// @Produce      json
// @Param        id      path   string        true "Template ID"
// @Param        request body   UpdateRequest true "Update data"
// @Success      200 {object} TaskTemplate
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/templates/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	templateID := c.Param("id")

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	template, err := h.service.Update(c.Request.Context(), templateID, &req, user.UserID)
	if err != nil {
		handleTemplateError(c, err)
		return
	}

	response.OK(c, template)
}

// =============================================================================
// DELETE
// =============================================================================

// Delete handles DELETE /templates/{id} - soft delete a template.
//
// @Summary      Delete template
// @Description  Soft delete a template
// @Tags         templates
// @Produce      json
// @Param        id path string true "Template ID"
// @Success      200 {object} response.OperationMessage
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/templates/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	templateID := c.Param("id")

	err := h.service.Delete(c.Request.Context(), templateID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		if appErr := errors.AsAppError(err); appErr != nil {
			response.Error(c, appErr)
		} else {
			response.ErrorFromErr(c, err)
		}
		return
	}

	response.Message(c, http.StatusOK, "Template deleted")
}

// =============================================================================
// ERROR HANDLING
// =============================================================================

func handleTemplateError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errors.ErrNotFound):
		response.NotFound(c)
	case errors.Is(err, errors.ErrBadRequest):
		if appErr := errors.AsAppError(err); appErr != nil {
			response.Error(c, appErr)
		} else {
			response.ErrorFromErr(c, err)
		}
	default:
		response.ErrorFromErr(c, err)
	}
}
