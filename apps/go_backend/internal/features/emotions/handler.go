// Package emotions provides emotion grid endpoints.
//
// These endpoints serve the 100-emotion mood meter grid data from the database.
package emotions

import (
	"github.com/gin-gonic/gin"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/response"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles emotion HTTP endpoints.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new emotion Handler.
func NewHandler(db *database.DB) *Handler {
	return &Handler{
		repo: NewRepository(db),
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// RegisterRoutes registers the emotion routes.
//
// Routes registered (protected, requires authentication):
//   - GET /grid    : Get all 100 emotions organized by quadrant
//   - GET /:id     : Get single emotion details
//   - POST /infer  : Calculate inferred emotion from task items
func RegisterRoutes(r *gin.RouterGroup, db *database.DB) {
	h := NewHandler(db)

	r.GET("/grid", h.Grid)
	r.GET("/:id", h.Get)
	r.POST("/infer", h.Infer)
}

// =============================================================================
// GRID
// =============================================================================

// Grid handles GET /emotions/grid - get all emotions for mood meter UI.
//
// @Summary      Get emotion grid
// @Description  Get all 100 emotions organized by quadrant for mood meter display
// @Tags         emotions
// @Produce      json
// @Success      200 {object} GridResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/emotions/grid [get]
func (h *Handler) Grid(c *gin.Context) {
	resp, err := h.repo.BuildGridResponse(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, resp)
}

// =============================================================================
// GET
// =============================================================================

// Get handles GET /emotions/:id - get single emotion details.
//
// @Summary      Get emotion by ID
// @Description  Get detailed information about a single emotion
// @Tags         emotions
// @Produce      json
// @Param        id path string true "Emotion ID (e.g., emotions:E16, emotions:E61)"
// @Success      200 {object} EmotionDetail
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/emotions/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	emotionID := c.Param("id")

	emotion, err := h.repo.GetByID(c.Request.Context(), emotionID)
	if err != nil || emotion == nil {
		response.NotFound(c)
		return
	}

	response.OK(c, emotion.ToDetail())
}

// =============================================================================
// INFER
// =============================================================================

// Infer handles POST /emotions/infer - calculate inferred emotion from task items.
//
// @Summary      Infer emotion from task items
// @Description  Calculate inferred emotion from positive and negative task items with emotions
// @Tags         emotions
// @Accept       json
// @Produce      json
// @Param        request body InferRequest true "Positives and negatives arrays"
// @Success      200 {object} InferResponse
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/emotions/infer [post]
func (h *Handler) Infer(c *gin.Context) {
	var req InferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	// Calculate inferred emotion using existing logic
	inferredEmotion := InferFromItems(req.Positives, req.Negatives)

	var closestEmotionDetail *EmotionDetail
	if inferredEmotion != nil && inferredEmotion.ClosestEmotionID != "" {
		closest, err := h.repo.GetByID(c.Request.Context(), inferredEmotion.ClosestEmotionID)
		if err == nil && closest != nil {
			closestEmotionDetail = closest.ToDetail()
		}
	}

	response.OK(c, InferResponse{
		InferredEmotion: inferredEmotion,
		ClosestEmotion:  closestEmotionDetail,
	})
}
