// Package auth provides authentication endpoints.
package auth

import (
	"github.com/gin-gonic/gin"

	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/response"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles authentication HTTP endpoints.
type Handler struct {
	service             Service
	validator           *validator.Validator
	registrationEnabled bool
}

func NewHandler(service Service, validator *validator.Validator, registrationEnabled bool) *Handler {
	return &Handler{
		service:             service,
		validator:           validator,
		registrationEnabled: registrationEnabled,
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the auth routes.
//
// Routes registered:
//   - POST /login    : User login
//   - POST /register : User registration
func RegisterRoutes(r *gin.RouterGroup, service Service, validator *validator.Validator, registrationEnabled bool) {
	h := NewHandler(service, validator, registrationEnabled)

	r.POST("/login", h.Login)
	r.POST("/register", h.Register)
}

// =============================================================================
// HANDLERS
// =============================================================================

// Login handles user login.
//
// @Summary      User login
// @Description  Authenticate user and return JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200 {object} AuthResponse
// @Failure      400 {object} response.APIResponse "Invalid request"
// @Failure      401 {object} response.APIResponse "Invalid credentials"
// @Router       /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	// Parse request body
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	// Validate request
	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	// Attempt login
	resp, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, errors.ErrInvalidCredentials) {
			response.Error(c, errors.ErrInvalidCredentials)
			return
		}
		log.Error().Err(err).Msg("login failed")
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, resp)
}

// Register handles user registration.
//
// @Summary      User registration
// @Description  Create a new user and return JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Registration details"
// @Success      200 {object} AuthResponse
// @Failure      400 {object} response.APIResponse "Invalid request"
// @Failure      409 {object} response.APIResponse "User already exists"
// @Router       /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	// Registration is disabled (e.g. production single-user deployments).
	if !h.registrationEnabled {
		response.Forbidden(c, "Registration is disabled")
		return
	}

	// Parse request body
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON body")
		return
	}

	// Validate request
	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(c, errs)
		return
	}

	// Attempt registration
	resp, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, errors.ErrUserExists) {
			response.Error(c, errors.ErrUserExists)
			return
		}
		log.Error().Err(err).Msg("registration failed")
		response.ErrorFromErr(c, err)
		return
	}

	response.OK(c, resp)
}
