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

	"github.com/gin-gonic/gin"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/response"
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

// RegisterRoutes registers the health check routes.
//
// Routes registered:
//   - GET / : Basic health check
func RegisterRoutes(r *gin.RouterGroup, db *database.DB) {
	h := NewHandler(db)
	r.GET("/", h.Check)
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
func (h *Handler) Check(c *gin.Context) {
	response.OK(c, HealthResponse{Status: "ok"})
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
func (h *Handler) CheckWithDB(c *gin.Context) {
	// Check database connectivity using SDK's QueryScalar
	dbStatus := "connected"
	_, err := database.QueryScalar[int](c.Request.Context(), h.db, "RETURN 1", nil)
	if err != nil {
		dbStatus = "disconnected"
		c.JSON(http.StatusServiceUnavailable, response.APIResponse{
			Data: HealthResponse{
				Status:   "degraded",
				Database: dbStatus,
			},
		})
		return
	}

	response.OK(c, HealthResponse{
		Status:   "ok",
		Database: dbStatus,
	})
}
