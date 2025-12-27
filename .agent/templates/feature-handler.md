---
description: Template for creating a new Go feature handler following the established pattern
---

# Feature Handler Template

Use this template when creating new HTTP handlers in `internal/features/{name}/`.

## handler.go

```go
// Package {name} provides {description}.
package {name}

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

// Handler handles {name} HTTP endpoints.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new {name} Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the {name} routes.
//
// Routes registered:
//   - GET    /        : List {name} with pagination
//   - POST   /        : Create a new {singular}
//   - GET    /{id}    : Get {singular} by ID
//   - PUT    /{id}    : Update {singular}
//   - DELETE /{id}    : Soft delete {singular}
func RegisterRoutes(r *gin.RouterGroup, service Service, validator *validator.Validator) {
	h := NewHandler(service, validator)

	r.GET("", h.List)
	r.POST("", h.Create)
	r.GET("/:id", h.Get)
	r.PUT("/:id", h.Update)
	r.DELETE("/:id", h.Delete)
}

// =============================================================================
// LIST
// =============================================================================

// List handles GET /{name} - list with pagination.
//
// @Summary      List {name}
// @Description  Get paginated list of {name}
// @Tags         {name}
// @Produce      json
// @Param        limit  query int false "Items per page (default 20, max 100)"
// @Param        offset query int false "Items to skip (default 0)"
// @Success      200 {object} {Name}PageResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/{name} [get]
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

// =============================================================================
// GET
// =============================================================================

// Get handles GET /{name}/{id} - get by ID.
//
// @Summary      Get {singular} by ID
// @Description  Get a single {singular} by its ID
// @Tags         {name}
// @Produce      json
// @Param        id path string true "{Singular} ID"
// @Success      200 {object} {Singular}
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/{name}/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	id := c.Param("id")

	item, err := h.service.Get(c.Request.Context(), id, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, item)
}

// =============================================================================
// CREATE
// =============================================================================

// Create handles POST /{name} - create new.
//
// @Summary      Create {singular}
// @Description  Create a new {singular}
// @Tags         {name}
// @Accept       json
// @Produce      json
// @Param        request body CreateRequest true "{Singular} data"
// @Success      201 {object} {Singular}
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/{name} [post]
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

	item, err := h.service.Create(c.Request.Context(), &req, user.UserID)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.Created(c, item)
}

// =============================================================================
// UPDATE
// =============================================================================

// Update handles PUT /{name}/{id} - update existing.
//
// @Summary      Update {singular}
// @Description  Update an existing {singular}
// @Tags         {name}
// @Accept       json
// @Produce      json
// @Param        id      path string        true "{Singular} ID"
// @Param        request body UpdateRequest true "Update data"
// @Success      200 {object} {Singular}
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/{name}/{id} [put]
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

	item, err := h.service.Update(c.Request.Context(), id, &req, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, item)
}

// =============================================================================
// DELETE
// =============================================================================

// Delete handles DELETE /{name}/{id} - soft delete.
//
// @Summary      Delete {singular}
// @Description  Soft delete a {singular}
// @Tags         {name}
// @Produce      json
// @Param        id path string true "{Singular} ID"
// @Success      200 {object} response.OperationMessage
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/{name}/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	user, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	id := c.Param("id")

	err := h.service.Delete(c.Request.Context(), id, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.ErrorFromErr(c, err)
		return
	}

	response.Message(c, http.StatusOK, "{Singular} deleted")
}
```

## Usage

Replace placeholders:
- `{name}` → feature name in lowercase plural (e.g., `widgets`)
- `{singular}` → singular lowercase (e.g., `widget`)
- `{Singular}` → singular PascalCase (e.g., `Widget`)
- `{Name}` → plural PascalCase (e.g., `Widgets`)
- `{description}` → brief description of the feature
