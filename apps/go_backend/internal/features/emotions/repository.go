package emotions

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/shared/database"
)

// =============================================================================
// REPOSITORY
// =============================================================================

// Repository provides emotion data access.
type Repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new emotion repository.
func NewRepository(db *database.DB) *Repository {
	return &Repository{
		db:     db,
		logger: log.With().Str("repository", "emotions").Logger(),
	}
}

// =============================================================================
// QUERIES
// =============================================================================

// GetAll retrieves all 100 emotions from the database.
func (r *Repository) GetAll(ctx context.Context) ([]*Emotion, error) {
	emotions, err := database.QueryAll[emotionDB](ctx, r.db, `
		SELECT * FROM emotions ORDER BY id
	`, nil)
	if err != nil {
		r.logger.Error().Err(err).Msg("failed to fetch emotions")
		return nil, err
	}
	// Convert to domain models
	result := make([]*Emotion, len(emotions))
	for i := range emotions {
		result[i] = emotions[i].toEmotion()
	}
	return result, nil
}

// GetByID retrieves a single emotion by ID (e.g., "E16" or "emotions:E16").
func (r *Repository) GetByID(ctx context.Context, id string) (*Emotion, error) {
	// Strip "emotions:" prefix if present
	cleanID := id
	if len(id) > 9 && id[:9] == "emotions:" {
		cleanID = id[9:]
	}

	emotion, err := database.QueryFirst[emotionDB](ctx, r.db, `
		SELECT * FROM $id
	`, map[string]any{
		"id": database.MustRecordID("emotions", cleanID),
	})
	if err != nil {
		return nil, err
	}
	return emotion.toEmotion(), nil
}

// GetByQuadrant retrieves all emotions for a specific quadrant.
func (r *Repository) GetByQuadrant(ctx context.Context, quadrant string) ([]*Emotion, error) {
	emotions, err := database.QueryAll[emotionDB](ctx, r.db, `
		SELECT * FROM emotions WHERE quadrant = $quadrant ORDER BY y DESC, x ASC
	`, map[string]any{
		"quadrant": quadrant,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("quadrant", quadrant).Msg("failed to fetch emotions by quadrant")
		return nil, err
	}
	// Convert to domain models
	result := make([]*Emotion, len(emotions))
	for i := range emotions {
		result[i] = emotions[i].toEmotion()
	}
	return result, nil
}

// BuildGridResponse fetches all emotions and organizes them by quadrant.
func (r *Repository) BuildGridResponse(ctx context.Context) (*GridResponse, error) {
	emotions, err := r.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	resp := &GridResponse{
		Yellow: make([]*Emotion, 0, 25),
		Green:  make([]*Emotion, 0, 25),
		Red:    make([]*Emotion, 0, 25),
		Blue:   make([]*Emotion, 0, 25),
		Total:  len(emotions),
	}

	for _, e := range emotions {
		switch e.Quadrant {
		case "yellow":
			resp.Yellow = append(resp.Yellow, e)
		case "green":
			resp.Green = append(resp.Green, e)
		case "red":
			resp.Red = append(resp.Red, e)
		case "blue":
			resp.Blue = append(resp.Blue, e)
		}
	}

	return resp, nil
}

// IsValidEmotionID checks if an emotion exists in the database.
func (r *Repository) IsValidEmotionID(ctx context.Context, id string) bool {
	emotion, err := r.GetByID(ctx, id)
	return err == nil && emotion != nil
}
