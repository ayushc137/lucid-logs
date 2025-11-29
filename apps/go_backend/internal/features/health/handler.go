// Package health provides health check endpoints.
//
// Endpoints:
//   - GET /health: Basic health check (always returns 200)
//   - GET /api/v1/health: Health check with database status
//
// These endpoints are used by:
//   - Load balancers for routing decisions
//   - Kubernetes for liveness/readiness probes
//   - Monitoring systems for uptime checks
package health

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/daily-journal/go-backend/internal/shared/database"
	"github.com/daily-journal/go-backend/internal/shared/response"
)

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// HealthResponse represents the health check response.
//
// @Description Health check response
type HealthResponse struct {
	Status   string `json:"status" example:"ok"`
	Database string `json:"database,omitempty" example:"connected"`
}

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles health check endpoints.
type Handler struct {
	db *database.DB
}

// NewHandler creates a new health Handler.
func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// =============================================================================
// ROUTES
// =============================================================================

// Routes returns the health check routes.
//
// Routes registered:
//   - GET / : Basic health check
func Routes(db *database.DB) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(db)

	r.Get("/", h.Check)

	return r
}

// =============================================================================
// HANDLERS
// =============================================================================

// Check performs a basic health check.
//
// @Summary      Basic health check
// @Description  Returns 200 OK if the service is running
// @Tags         health
// @Produce      json
// @Success      200 {object} HealthResponse
// @Router       /health [get]
func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	response.OK(w, HealthResponse{Status: "ok"})
}

// CheckWithDB performs a health check including database connectivity.
//
// @Summary      Health check with database status
// @Description  Returns health status including database connectivity
// @Tags         health
// @Produce      json
// @Success      200 {object} HealthResponse "Healthy"
// @Failure      503 {object} response.APIResponse "Database unavailable"
// @Router       /api/v1/health [get]
func (h *Handler) CheckWithDB(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity using SDK's QueryScalar
	dbStatus := "connected"
	_, err := database.QueryScalar[int](r.Context(), h.db, "RETURN 1", nil)
	if err != nil {
		dbStatus = "disconnected"
		response.JSON(w, http.StatusServiceUnavailable, response.APIResponse{
			Data: HealthResponse{
				Status:   "degraded",
				Database: dbStatus,
			},
		})
		return
	}

	response.OK(w, HealthResponse{
		Status:   "ok",
		Database: dbStatus,
	})
}
