// Package tasks provides task management functionality.
//
// This package implements:
//   - CRUD operations for tasks
//   - Category linking (record links)
//   - Emotion tracking (emotion_id on task, structured positives/negatives)
//   - Inferred emotion calculation (computed on write using emotion default intensities)
//   - Soft delete support
//   - Pagination
//
// Database Architecture:
//
// Tasks use SurrealDB's schemaless tables with record links:
//   - task.category = categories:abc123 (record link)
//   - FETCH hydrates the linked category automatically
//   - Permissions use $auth.id for ownership
package tasks

import (
	"time"

	"github.com/lucid-logs/go-backend/internal/features/categories"
	"github.com/lucid-logs/go-backend/internal/features/emotions"
)

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// Task represents a task/journal entry in the system.
//
// Fields marked with json:"-" are hidden from API responses.
// System-managed fields (created_at, updated_at, deleted_at) are read-only.
//
// Emotion fields:
//   - EmotionID: User's primary emotion from mood meter grid (e.g., "emotions:E16")
//   - Positives/Negatives: Structured items with optional emotion tags
//   - InferredEmotion: Server-calculated emotional state (computed on write)
//
// Goal integration fields:
//   - ActivityKey: For auto-linking to goals with matching activity_key
//   - TemplateID: Reference to the template used to create this task
//   - Quantity: Tracked value for measurable goals (e.g., "500ml", "30min")
//   - LinkedGoals: Goals linked to this task with impact metadata
type Task struct {
	ID        string               `json:"id,omitempty"`
	Title     string               `json:"title"`
	Journal   string               `json:"journal"`
	StartDate time.Time            `json:"start_date"`
	EndDate   time.Time            `json:"end_date"`
	Completed bool                 `json:"completed"`
	Priority  int                  `json:"priority"`
	Source    string               `json:"source"`
	Note      string               `json:"note"`
	Positives []TaskItem           `json:"positives"`
	Negatives []TaskItem           `json:"negatives"`
	Category  *categories.Category `json:"category,omitempty"`

	// Emotion tracking
	EmotionID       *string                   `json:"emotion_id,omitempty"`       // e.g., "emotions:E16"
	InferredEmotion *emotions.InferredEmotion `json:"inferred_emotion,omitempty"` // Computed on write

	// Goal/Template integration
	ActivityKey *string        `json:"activity_key,omitempty"` // For auto-linking to goals
	TemplateID  *string        `json:"template_id,omitempty"`  // Source template (templates:xyz)
	Quantity    *Quantity      `json:"quantity,omitempty"`     // For measurable goals
	LinkedGoals []TaskGoalLink `json:"linked_goals,omitempty"` // Goals linked to this task

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedBy string     `json:"-"` // Hidden: ownership field
	UpdatedBy string     `json:"-"` // Hidden: audit field
}

// Quantity represents a measured value with unit for task contribution to goals.
type Quantity struct {
	Value float64 `json:"value"` // e.g., 500, 30
	Unit  string  `json:"unit"`  // e.g., "ml", "minutes", "pages"
}

// TaskItem represents a structured positive/negative item with optional emotion.
// Intensity is taken from the emotion's default Intensity value.
type TaskItem struct {
	Text      string  `json:"text" example:"Good team collaboration"`
	EmotionID *string `json:"emotion_id,omitempty" example:"emotions:E16"` // Optional: emotions:E01-E100
}

// TaskGoalLink represents a linked goal with impact metadata.
// This is populated via SurrealDB FETCH from the task_goals relation.
type TaskGoalLink struct {
	GoalID          string   `json:"goal_id"`
	GoalTitle       string   `json:"goal_title"`
	ImpactType      string   `json:"impact_type"`      // "positive", "negative", "neutral"
	ImpactMagnitude int      `json:"impact_magnitude"` // 1-5
	QuantityValue   *float64 `json:"quantity_value,omitempty"`
	QuantityUnit    *string  `json:"quantity_unit,omitempty"`
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for creating a new task.
//
// Required fields: title, start_date, end_date
// Optional fields: journal, priority, source, note, positives, negatives, category_id, emotion_id
//
// Emotion IDs: Use format "emotions:E01" to "emotions:E100" (e.g., "emotions:E16" for Happy)
// Positives/Negatives: Array of items with text and optional emotion_id
//
// @Description Request payload for creating a task
type CreateRequest struct {
	Title      string     `json:"title" validate:"required,min=1,max=500" example:"Morning standup"`
	Journal    string     `json:"journal" validate:"max=10000" example:"Daily team sync meeting"`
	StartDate  string     `json:"start_date" validate:"required,datetime_flexible" example:"2025-12-06T09:00:00Z"`
	EndDate    string     `json:"end_date" validate:"required,datetime_flexible" example:"2025-12-06T09:30:00Z"`
	Priority   int        `json:"priority" validate:"min=-100,max=100" example:"1"`
	Source     string     `json:"source,omitempty" example:"manual"`
	Note       string     `json:"note,omitempty" validate:"max=5000" example:"Focus on blockers"`
	Positives  []TaskItem `json:"positives,omitempty"` // [{"text": "...", "emotion_id": "emotions:E16"}]
	Negatives  []TaskItem `json:"negatives,omitempty"` // [{"text": "...", "emotion_id": "emotions:E61"}]
	CategoryID string     `json:"category_id,omitempty" example:"categories:work123"`
	EmotionID  *string    `json:"emotion_id,omitempty" validate:"omitempty,emotion_id" example:"emotions:E16"` // Primary emotion
}

// UpdateRequest is the request payload for updating a task.
//
// All fields are optional. Only provided fields will be updated.
// Use null or empty string for category_id to remove the category link.
// Use null for emotion_id to remove the emotion.
//
// Emotion IDs: Use format "emotions:E01" to "emotions:E100" (e.g., "emotions:E16" for Happy)
//
// @Description Request payload for updating a task
type UpdateRequest struct {
	Title      *string    `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Journal    *string    `json:"journal,omitempty" validate:"omitempty,max=10000"`
	StartDate  *string    `json:"start_date,omitempty" validate:"omitempty,datetime_flexible"`
	EndDate    *string    `json:"end_date,omitempty" validate:"omitempty,datetime_flexible"`
	Completed  *bool      `json:"completed,omitempty"`
	Priority   *int       `json:"priority,omitempty" validate:"omitempty,min=-100,max=100"`
	Note       *string    `json:"note,omitempty" validate:"omitempty,max=5000"`
	Positives  []TaskItem `json:"positives,omitempty"` // [{"text": "...", "emotion_id": "emotions:E16"}]
	Negatives  []TaskItem `json:"negatives,omitempty"` // [{"text": "...", "emotion_id": "emotions:E61"}]
	CategoryID *string    `json:"category_id,omitempty"`
	EmotionID  *string    `json:"emotion_id,omitempty" validate:"omitempty,emotion_id"` // Primary emotion
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name for tasks.
	Table = "tasks"

	// SourceManual is the default source for manually created tasks.
	SourceManual = "manual"
)

// TaskPageResponse documents the paginated response returned by the List endpoint.
type TaskPageResponse struct {
	Items   []*Task `json:"items"`
	Total   int64   `json:"total"`
	Limit   int     `json:"limit"`
	Offset  int     `json:"offset"`
	HasMore bool    `json:"has_more"`
}

// =============================================================================
// FILTER PARAMETERS
// =============================================================================

// TaskFilterParams contains filter criteria for listing tasks.
//
// Filters can be combined (AND logic). Empty values are ignored.
// Search uses SurrealDB full-text search on title, journal, and note fields.
//
// Example URL: /api/v1/tasks?search=meeting&category_id=cat123&status=pending&priority_min=5
type TaskFilterParams struct {
	// Search performs full-text search across title, journal, and note fields
	// Uses SurrealDB FTS with BM25 ranking and English stemming
	Search string `json:"search,omitempty"`

	// CategoryID filters by specific category
	CategoryID string `json:"category_id,omitempty"`

	// Status filters by completion status: "all", "completed", "pending"
	Status string `json:"status,omitempty"`

	// PriorityMin filters tasks with priority >= this value (1-10)
	PriorityMin *int `json:"priority_min,omitempty"`

	// PriorityMax filters tasks with priority <= this value (1-10)
	PriorityMax *int `json:"priority_max,omitempty"`

	// StartDateFrom filters tasks starting on or after this date (RFC3339)
	StartDateFrom string `json:"start_date_from,omitempty"`

	// StartDateTo filters tasks starting on or before this date (RFC3339)
	StartDateTo string `json:"start_date_to,omitempty"`

	// SortField specifies the field to sort by: "start_date", "priority", "title", "created_at"
	SortField string `json:"sort_field,omitempty"`

	// SortOrder specifies sort direction: "asc" or "desc" (default: desc for dates, asc for title)
	SortOrder string `json:"sort_order,omitempty"`
}

// Status filter constants
const (
	StatusAll       = "all"
	StatusCompleted = "completed"
	StatusPending   = "pending"
)

// Sort field constants
const (
	SortByStartDate = "start_date"
	SortByPriority  = "priority"
	SortByTitle     = "title"
	SortByCreatedAt = "created_at"
)

// Sort order constants
const (
	SortAsc  = "asc"
	SortDesc = "desc"
)
