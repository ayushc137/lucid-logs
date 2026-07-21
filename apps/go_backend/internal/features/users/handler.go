package users

import (
	"github.com/gin-gonic/gin"

	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	validatorpkg "github.com/lucid-logs/go-backend/internal/shared/validator"
)

// Handler provides HTTP handlers for users.
type Handler struct {
	service   Service
	validator *validatorpkg.Validator
}

// NewHandler creates a new Handler.
func NewHandler(service Service, validator *validatorpkg.Validator) *Handler {
	return &Handler{service: service, validator: validator}
}

// RegisterRoutes registers user routes.
func RegisterRoutes(r *gin.RouterGroup, service Service, validator *validatorpkg.Validator) {
	h := NewHandler(service, validator)

	r.GET("/me", h.Me)
	r.PUT("/me/preferences", h.UpdatePreferences)
	r.GET("/:id", h.Get)
	r.PATCH("/:id", h.Update)
	r.DELETE("/:id", h.Delete)
}

// Me returns the authenticated user's profile.
func (h *Handler) Me(c *gin.Context) {
	authUser, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	user, err := h.service.Get(c.Request.Context(), authUser.UserID, authUser.UserID)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	// Strip API key from AI settings
	user.Preferences.AI = SafeAISettings(user.Preferences.AI)

	response.OK(c, user)
}

// Get handles GET /users/{id}
func (h *Handler) Get(c *gin.Context) {
	authUser, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	user, err := h.service.Get(c.Request.Context(), authUser.UserID, id)
	// service enforces permissions
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	// Strip API key from AI settings
	user.Preferences.AI = SafeAISettings(user.Preferences.AI)

	response.OK(c, user)
}

// UpdatePreferences handles PUT /users/me/preferences
func (h *Handler) UpdatePreferences(c *gin.Context) {
	authUser, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	user, err := h.service.UpdatePreferences(c.Request.Context(), authUser.UserID, authUser.UserID, &req)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, user)
}

// Update handles PATCH /users/{id}
func (h *Handler) Update(c *gin.Context) {
	authUser, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	user, err := h.service.Update(c.Request.Context(), authUser.UserID, id, &req)
	if err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, user)
}

// Delete handles DELETE /users/{id}
func (h *Handler) Delete(c *gin.Context) {
	authUser, appErr := middleware.MustGetAuthenticatedUser(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	if err := h.service.Delete(c.Request.Context(), authUser.UserID, id); err != nil {
		response.ErrorFromErr(c, err)
		return
	}

	response.NoContent(c)
}
