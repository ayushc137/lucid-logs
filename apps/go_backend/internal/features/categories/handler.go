package categories

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/daily-journal/go-backend/internal/shared/errors"
	"github.com/daily-journal/go-backend/internal/shared/middleware"
	"github.com/daily-journal/go-backend/internal/shared/pagination"
	"github.com/daily-journal/go-backend/internal/shared/response"
	"github.com/daily-journal/go-backend/internal/shared/validator"
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

// Routes returns the category routes.
func Routes(service Service, validator *validator.Validator) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(service, validator)

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}

// =============================================================================
// HANDLERS
// =============================================================================

// List handles GET /categories
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	params := pagination.FromRequest(r)

	resp, err := h.service.List(r.Context(), user.UserID, params)
	if err != nil {
		response.ErrorFromErr(w, err)
		return
	}

	response.OK(w, resp)
}

// Get handles GET /categories/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	categoryID := chi.URLParam(r, "id")

	cat, err := h.service.Get(r.Context(), categoryID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.ErrorFromErr(w, err)
		return
	}

	response.OK(w, cat)
}

// Create handles POST /categories
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(w, errs)
		return
	}

	cat, err := h.service.Create(r.Context(), &req, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrCategoryNameExists) {
			response.Conflict(w, "Category with this name already exists")
			return
		}
		response.ErrorFromErr(w, err)
		return
	}

	response.Created(w, cat)
}

// Update handles PUT /categories/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	categoryID := chi.URLParam(r, "id")

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(w, errs)
		return
	}

	cat, err := h.service.Update(r.Context(), categoryID, &req, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		if errors.Is(err, errors.ErrCategoryNameExists) {
			response.Conflict(w, "Category with this name already exists")
			return
		}
		response.ErrorFromErr(w, err)
		return
	}

	response.OK(w, cat)
}

// Delete handles DELETE /categories/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	categoryID := chi.URLParam(r, "id")

	err := h.service.Delete(r.Context(), categoryID, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.ErrorFromErr(w, err)
		return
	}

	response.Message(w, http.StatusOK, "Category deleted")
}
