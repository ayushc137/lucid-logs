package categories

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

// Handler handles category HTTP endpoints.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new category Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the category routes.
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

// List handles GET /categories
//
// @Summary      List categories
// @Description  Get paginated categories for the authenticated user
// @Tags         categories
// @Produce      json
// @Param        limit  query int false "Items per page (default 20, max 100)"
// @Param        offset query int false "Items to skip (default 0)"
// @Param        search query string false "Search by category name"
// @Success      200 {object} categories.CategoryPageResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/categories [get]
func (h *Handler) List(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	params := pagination.FromRequest(c)
	search := c.Query("search")

	resp, err := h.service.List(c.Request.Context(), user.UserID, params, search)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, resp)
}

// Get handles GET /categories/{id}
//
// @Summary      Get category
// @Description  Get a single category by ID
// @Tags         categories
// @Produce      json
// @Param        id path string true "Category ID"
// @Success      200 {object} Category
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/categories/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	categoryID := c.Param("id")

	cat, err := h.service.Get(c.Request.Context(), categoryID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, cat)
}

// Create handles POST /categories
//
// @Summary      Create category
// @Description  Create a category for organizing tasks
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        request body CreateRequest true "Category data"
// @Success      201 {object} Category
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      409 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/categories [post]
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

	cat, err := h.service.Create(c.Request.Context(), &req, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrCategoryNameExists) {
			response.Conflict(c, "Category with this name already exists")
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.Created(c, cat)
}

// Update handles PUT /categories/{id}
//
// @Summary      Update category
// @Description  Update the name or color of a category
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        id      path string        true "Category ID"
// @Param        request body UpdateRequest true "Update payload"
// @Success      200 {object} Category
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      409 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/categories/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	categoryID := c.Param("id")

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	cat, err := h.service.Update(c.Request.Context(), categoryID, &req, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		if errors.Is(err, errors.ErrCategoryNameExists) {
			response.Conflict(c, "Category with this name already exists")
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, cat)
}

// Delete handles DELETE /categories/{id}
//
// @Summary      Delete category
// @Description  Soft delete a category
// @Tags         categories
// @Produce      json
// @Param        id path string true "Category ID"
// @Success      200 {object} response.OperationMessage
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/categories/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	categoryID := c.Param("id")

	err := h.service.Delete(c.Request.Context(), categoryID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.Message(c, http.StatusOK, "Category deleted")
}
