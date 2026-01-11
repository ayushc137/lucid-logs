package retrospectives

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

// Handler handles HTTP requests for retrospectives.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new retrospectives Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTE REGISTRATION
// =============================================================================

// RegisterRoutes registers the retrospective routes.
//
// Routes:
//   - GET    /retrospectives           List retrospectives
//   - GET    /retrospectives/:id       Get retrospective by ID
//   - POST   /retrospectives/generate  Generate new retrospective
//   - PUT    /retrospectives/:id       Update user content
//   - DELETE /retrospectives/:id       Delete retrospective
func RegisterRoutes(rg *gin.RouterGroup, service Service, validator *validator.Validator) {
	h := NewHandler(service, validator)

	rg.GET("", h.List)
	rg.GET("/:id", h.Get)
	rg.POST("/generate", h.Generate)
	rg.PUT("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
}

// =============================================================================
// HANDLERS
// =============================================================================

// List godoc
// @Summary List retrospectives
// @Description Retrieves paginated list of user's retrospectives
// @Tags Retrospectives
// @Produce json
// @Param limit query int false "Items per page (default 20, max 100)"
// @Param offset query int false "Items to skip (default 0)"
// @Success 200 {object} ListResponse
// @Failure 401 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/retrospectives [get]
func (h *Handler) List(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	params := pagination.FromRequest(c)

	resp, err := h.service.List(c.Request.Context(), user.UserID, params)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, resp)
}

// Get godoc
// @Summary Get retrospective by ID
// @Description Retrieves a single retrospective with auto-summary and user content
// @Tags Retrospectives
// @Produce json
// @Param id path string true "Retrospective ID"
// @Success 200 {object} Retrospective
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/retrospectives/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	id := c.Param("id")

	retro, err := h.service.Get(c.Request.Context(), id, user.UserID)
	if err != nil {
		handleRetroError(c, err)
		return
	}

	response.OK(c, retro)
}

// Generate godoc
// @Summary Generate retrospective
// @Description Generates a new retrospective with auto-computed analytics summary
// @Tags Retrospectives
// @Accept json
// @Produce json
// @Param request body GenerateRequest true "Generation parameters"
// @Success 201 {object} Retrospective
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/retrospectives/generate [post]
func (h *Handler) Generate(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	retro, err := h.service.Generate(c.Request.Context(), user.UserID, &req)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.Created(c, retro)
}

// Update godoc
// @Summary Update retrospective
// @Description Updates user reflection content on a retrospective
// @Tags Retrospectives
// @Accept json
// @Produce json
// @Param id path string true "Retrospective ID"
// @Param request body UpdateRequest true "Update data"
// @Success 200 {object} Retrospective
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/retrospectives/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	id := c.Param("id")

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	retro, err := h.service.Update(c.Request.Context(), id, &req, user.UserID)
	if err != nil {
		handleRetroError(c, err)
		return
	}

	response.OK(c, retro)
}

// Delete godoc
// @Summary Delete retrospective
// @Description Soft-deletes a retrospective
// @Tags Retrospectives
// @Produce json
// @Param id path string true "Retrospective ID"
// @Success 200 {object} response.OperationMessage
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Security BearerAuth
// @Router /api/v1/retrospectives/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	id := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), id, user.UserID); err != nil {
		handleRetroError(c, err)
		return
	}

	response.Message(c, http.StatusOK, "Retrospective deleted")
}

// =============================================================================
// ERROR HANDLING
// =============================================================================

func handleRetroError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errors.ErrNotFound):
		response.NotFound(c)
	default:
		response.ErrorFromErr(c, err)
	}
}
