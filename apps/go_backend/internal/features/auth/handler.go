package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/daily-journal/go-backend/internal/shared/errors"
	"github.com/daily-journal/go-backend/internal/shared/response"
	"github.com/daily-journal/go-backend/internal/shared/validator"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles authentication HTTP endpoints.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new auth Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// Routes returns the auth routes.
//
// Routes registered:
//   - POST /login    : User login
//   - POST /register : User registration
func Routes(service Service, validator *validator.Validator) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(service, validator)

	r.Post("/login", h.Login)
	r.Post("/register", h.Register)

	return r
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
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate request
	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(w, errs)
		return
	}

	// Attempt login
	resp, err := h.service.Login(r.Context(), &req)
	if err != nil {
		if errors.Is(err, errors.ErrInvalidCredentials) {
			response.Error(w, errors.ErrInvalidCredentials)
			return
		}
		log.Error().Err(err).Msg("login failed")
		response.ErrorFromErr(w, err)
		return
	}

	response.OK(w, resp)
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
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate request
	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(w, errs)
		return
	}

	// Attempt registration
	resp, err := h.service.Register(r.Context(), &req)
	if err != nil {
		if errors.Is(err, errors.ErrUserExists) {
			response.Error(w, errors.ErrUserExists)
			return
		}
		log.Error().Err(err).Msg("registration failed")
		response.ErrorFromErr(w, err)
		return
	}

	response.OK(w, resp)
}
