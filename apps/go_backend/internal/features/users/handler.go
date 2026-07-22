package users

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lucid-logs/go-backend/internal/config"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/llm"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	validatorpkg "github.com/lucid-logs/go-backend/internal/shared/validator"
)

// Handler provides HTTP handlers for users.
type Handler struct {
	service      Service
	validator    *validatorpkg.Validator
	llmDefaults  config.LLMConfig
}

// NewHandler creates a new Handler.
func NewHandler(service Service, validator *validatorpkg.Validator, llmDefaults config.LLMConfig) *Handler {
	return &Handler{service: service, validator: validator, llmDefaults: llmDefaults}
}

// RegisterRoutes registers user routes.
func RegisterRoutes(r *gin.RouterGroup, service Service, validator *validatorpkg.Validator, llmDefaults config.LLMConfig) {
	h := NewHandler(service, validator, llmDefaults)

	r.GET("/me", h.Me)
	r.PUT("/me/preferences", h.UpdatePreferences)
	r.GET("/me/ai/models", h.ListAIModels)
	r.GET("/me/ai/defaults", h.AIDefaults)
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

// ListAIModels returns available models from the user's configured AI provider.
func (h *Handler) ListAIModels(c *gin.Context) {
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

	ai := user.Preferences.AI
	if ai == nil {
		response.BadRequest(c, "AI not configured")
		return
	}

	baseURL := llm.ResolveBaseURL(ai.Provider, ai.BaseURL)

	// Fall back to env default API key if the user hasn't set one
	apiKey := ai.APIKey
	if apiKey == "" {
		apiKey = h.llmDefaults.APIKey
	}

	client := llm.NewClient(baseURL, apiKey, "")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	models, err := client.ListModels(ctx)
	if err != nil {
		response.Error(c, errors.ErrInternal.WithMessage("Failed to fetch models: "+err.Error()))
		return
	}

	response.OK(c, gin.H{"models": models})
}

// AIDefaults returns env-level LLM defaults so the frontend can pre-fill
// the AI settings form for users who haven't configured AI yet.
func (h *Handler) AIDefaults(c *gin.Context) {
	response.OK(c, gin.H{
		"provider": h.llmDefaults.Provider,
		"base_url": h.llmDefaults.BaseURL,
		"model":    h.llmDefaults.Model,
		"has_key":  h.llmDefaults.APIKey != "",
	})
}
