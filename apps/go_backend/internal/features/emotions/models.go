package emotions

import (
	"github.com/surrealdb/surrealdb.go/pkg/models"

	"github.com/lucid-logs/go-backend/internal/shared/database"
)

// =============================================================================
// DATABASE MODELS
// =============================================================================

// emotionDB is the internal database representation of an emotion.
// Uses models.RecordID for SurrealDB SDK compatibility.
type emotionDB struct {
	ID          models.RecordID `json:"id,omitempty"`
	Name        string          `json:"name"`
	Emoji       string          `json:"emoji"`
	Quadrant    string          `json:"quadrant"`
	X           int             `json:"x"`
	Y           int             `json:"y"`
	Valence     float64         `json:"valence"`
	Arousal     float64         `json:"arousal"`
	Dominance   float64         `json:"dominance"`
	Intensity   float64         `json:"intensity"`
	Certainty   float64         `json:"certainty"`
	Social      float64         `json:"social"`
	Description string          `json:"description"`
}

// toEmotion converts the database model to the domain model.
func (e *emotionDB) toEmotion() *Emotion {
	return &Emotion{
		ID:          database.ToStringID(e.ID),
		Name:        e.Name,
		Emoji:       e.Emoji,
		Quadrant:    e.Quadrant,
		X:           e.X,
		Y:           e.Y,
		Valence:     e.Valence,
		Arousal:     e.Arousal,
		Dominance:   e.Dominance,
		Intensity:   e.Intensity,
		Certainty:   e.Certainty,
		Social:      e.Social,
		Description: e.Description,
	}
}

// Emotion represents an emotion record (API-facing domain model).
type Emotion struct {
	ID          string  `json:"id"` // e.g., "emotions:E16"
	Name        string  `json:"name"`
	Emoji       string  `json:"emoji"`
	Quadrant    string  `json:"quadrant"`
	X           int     `json:"x"`
	Y           int     `json:"y"`
	Valence     float64 `json:"valence"`   // -1 to +1: pleasant/unpleasant
	Arousal     float64 `json:"arousal"`   // -1 to +1: high/low energy
	Dominance   float64 `json:"dominance"` // -1 to +1: in control/controlled
	Intensity   float64 `json:"intensity"` // 0.1 to 1.0: default weight
	Certainty   float64 `json:"certainty"` // -1 to +1: sure/unsure
	Social      float64 `json:"social"`    // -1 to +1: social/individual
	Description string  `json:"description"`
}

// =============================================================================
// GRID RESPONSE TYPES
// =============================================================================

// GridResponse is the response for GET /emotions/grid.
// Returns full Emotion data for each quadrant to support all UI features.
type GridResponse struct {
	Yellow []*Emotion `json:"yellow"`
	Green  []*Emotion `json:"green"`
	Red    []*Emotion `json:"red"`
	Blue   []*Emotion `json:"blue"`
	Total  int        `json:"total"`
}

// =============================================================================
// DETAIL RESPONSE
// =============================================================================

// EmotionDetail is the full emotion data for GET /emotions/:id.
type EmotionDetail struct {
	ID          string  `json:"id"` // e.g., "emotions:E16"
	Name        string  `json:"name"`
	Emoji       string  `json:"emoji"`
	Quadrant    string  `json:"quadrant"`
	X           int     `json:"x"`
	Y           int     `json:"y"`
	Valence     float64 `json:"valence"`
	Arousal     float64 `json:"arousal"`
	Dominance   float64 `json:"dominance"`
	Intensity   float64 `json:"intensity"`
	Certainty   float64 `json:"certainty"`
	Social      float64 `json:"social"`
	Description string  `json:"description"`
}

// =============================================================================
// INFERRED EMOTION
// =============================================================================

// InferredEmotion represents the server-calculated emotional state from
// all emotion inputs on a task. Calculated and stored on write, not read.
type InferredEmotion struct {
	// Weighted centroid coordinates
	Valence   float64 `json:"valence"`   // -1 to +1
	Arousal   float64 `json:"arousal"`   // -1 to +1
	Dominance float64 `json:"dominance"` // -1 to +1

	// Classification
	Quadrant         string `json:"quadrant"`           // Dominant quadrant
	ClosestEmotionID string `json:"closest_emotion_id"` // Nearest emotion

	// Metadata
	PositiveCount int     `json:"positive_count"` // Number of positive items with emotions
	NegativeCount int     `json:"negative_count"` // Number of negative items with emotions
	Dissonance    float64 `json:"dissonance"`     // 0-1: internal conflict score
}

// =============================================================================
// TASK ITEM TYPES
// =============================================================================

// TaskItem represents a structured positive/negative item with optional emotion.
// Intensity is taken from the emotion's default value.
type TaskItem struct {
	Text      string  `json:"text"`
	EmotionID *string `json:"emotion_id,omitempty"` // Optional: e.g., "emotions:E16"
}

// =============================================================================
// INFER REQUEST/RESPONSE
// =============================================================================

// InferRequest is the request body for POST /emotions/infer.
type InferRequest struct {
	Positives []TaskItem `json:"positives"`
	Negatives []TaskItem `json:"negatives"`
}

// InferResponse is the response for POST /emotions/infer.
type InferResponse struct {
	InferredEmotion *InferredEmotion `json:"inferred_emotion"`
	ClosestEmotion  *EmotionDetail   `json:"closest_emotion,omitempty"`
}

// =============================================================================
// CONVERSION HELPERS
// =============================================================================

// ToDetail converts an Emotion to the full EmotionDetail format.
func (e *Emotion) ToDetail() *EmotionDetail {
	return &EmotionDetail{
		ID:          e.ID,
		Name:        e.Name,
		Emoji:       e.Emoji,
		Quadrant:    e.Quadrant,
		X:           e.X,
		Y:           e.Y,
		Valence:     e.Valence,
		Arousal:     e.Arousal,
		Dominance:   e.Dominance,
		Intensity:   e.Intensity,
		Certainty:   e.Certainty,
		Social:      e.Social,
		Description: e.Description,
	}
}
