// Package tasks provides task management functionality.
//
// This package implements:
//   - CRUD operations for tasks
//   - Category linking via in_category relation
//   - Emotion tracking (emotion_id on task, structured positives/negatives)
//   - Inferred emotion calculation (computed on write using emotion default intensities)
//   - Goal linking via task_goals relation
//   - Activity tracking via created_from_activity relation
//   - Soft delete support
//
// Database Architecture:
//   - in_category: RELATE table for category assignment
//   - task_goals: RELATE table for goal-task links with impact metadata
//   - created_from_activity: RELATE table tracking task origin from activity
//   - task_emotions: RELATE table for emotion analytics
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
// Relationships (via graph edges, not stored on task):
//   - Category: via in_category relation
//   - Activity: via created_from_activity relation
//   - Goals: via task_goals relation
//
// @Description Task/journal entry entity
type Task struct {
	ID        string     `json:"id,omitempty"`
	Title     string     `json:"title"`
	Journal   string     `json:"journal"`
	StartDate time.Time  `json:"start_date"`
	EndDate   time.Time  `json:"end_date"`
	Completed bool       `json:"completed"`
	Source    string     `json:"source"` // "manual", "template", "quick"
	Note      string     `json:"note"`
	Positives []TaskItem `json:"positives"`
	Negatives []TaskItem `json:"negatives"`

	// Emotion tracking
	EmotionID       *string                   `json:"emotion_id,omitempty"`       // e.g., "emotions:E16"
	InferredEmotion *emotions.InferredEmotion `json:"inferred_emotion,omitempty"` // Computed on write

	// Quantity (for measurable goals)
	Quantity *Quantity `json:"quantity,omitempty"`

	// Metadata
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedBy string     `json:"-"` // Hidden: ownership field

	// Populated via graph queries (not stored on task)
	Category    *categories.Category    `json:"category,omitempty"`     // From in_category edge
	Emotion     *emotions.EmotionDetail `json:"emotion,omitempty"`      // Full emotion details
	LinkedGoals []LinkedGoalSummary     `json:"linked_goals,omitempty"` // From task_goals edge
}

// Quantity represents a measured value with unit for task contribution to goals.
//
// @Description Quantity measurement for task
type Quantity struct {
	Value  float64 `json:"value"`   // e.g., 5.0, 30
	UnitID string  `json:"unit_id"` // e.g., "units:km", "units:min"
}

// TaskItem represents a structured positive/negative item with optional emotion.
// Intensity is taken from the emotion's default Intensity value.
//
// @Description Positive/negative reflection item
type TaskItem struct {
	Text      string  `json:"text" example:"Good team collaboration"`
	EmotionID *string `json:"emotion_id,omitempty" example:"emotions:E16"` // Optional: emotions:E01-E100
}

// LinkedGoalSummary is a lightweight goal summary for task list responses.
// Used for highlighting linked goals in the UI without full goal details.
//
// @Description Lightweight goal summary for task highlighting
type LinkedGoalSummary struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Icon          string   `json:"icon,omitempty"`
	Color         string   `json:"color,omitempty"` // From goal's category
	ImpactType    string   `json:"impact_type"`     // "positive", "negative", "neutral"
	QuantityValue *float64 `json:"quantity_value,omitempty"`
	UnitSymbol    string   `json:"unit_symbol,omitempty"` // e.g., "km", "min"
}

// TaskGoalLink represents a linked goal with impact metadata.
// This is populated from the task_goals relation.
//
// @Description Goal linked to task with impact data
type TaskGoalLink struct {
	GoalID         string   `json:"goal_id"`
	GoalTitle      string   `json:"goal_title"`
	GoalIcon       string   `json:"goal_icon,omitempty"`
	ImpactType     string   `json:"impact_type"` // "positive", "negative", "neutral"
	QuantityValue  *float64 `json:"quantity_value,omitempty"`
	UnitID         *string  `json:"unit_id,omitempty"`
	IsMilestone    bool     `json:"is_milestone,omitempty"`
	MilestoneLabel string   `json:"milestone_label,omitempty"`

	// Additional goal details for rich display
	GoalDescription string   `json:"goal_description,omitempty"`
	GoalStatus      string   `json:"goal_status,omitempty"`
	GoalPriority    int      `json:"goal_priority,omitempty"`
	GoalTargetValue *float64 `json:"goal_target_value,omitempty"`
	GoalTargetUnit  string   `json:"goal_target_unit,omitempty"`
	GoalProgress    float64  `json:"goal_progress,omitempty"` // Current progress percentage
	GoalCategory    *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"goal_category,omitempty"`

	// Link metadata
	LinkedAt *time.Time `json:"linked_at,omitempty"`
	Notes    string     `json:"notes,omitempty"`
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for creating a new task.
//
// Required fields: title, start_date, end_date
// Optional fields: journal, source, note, positives, negatives, category_id, emotion_id
//
// Emotion IDs: Use format "emotions:E01" to "emotions:E100" (e.g., "emotions:E16" for Happy)
// Positives/Negatives: Array of items with text and optional emotion_id
//
// @Description Request payload for creating a task
type CreateRequest struct {
	Title     string `json:"title" validate:"required,min=1,max=500" example:"Morning standup"`
	Journal   string `json:"journal" validate:"max=10000" example:"Daily team sync meeting"`
	StartDate string `json:"start_date" validate:"required,datetime_flexible" example:"2025-12-06T09:00:00Z"`
	EndDate   string `json:"end_date" validate:"required,datetime_flexible" example:"2025-12-06T09:30:00Z"`
	Source    string `json:"source,omitempty" example:"manual"`
	Note      string `json:"note,omitempty" validate:"max=5000" example:"Focus on blockers"`
	Completed bool   `json:"completed,omitempty" example:"false"`

	Positives []TaskItem `json:"positives,omitempty"` // [{"text": "...", "emotion_id": "emotions:E16"}]
	Negatives []TaskItem `json:"negatives,omitempty"` // [{"text": "...", "emotion_id": "emotions:E61"}]

	// Category (creates in_category edge)
	CategoryID string `json:"category_id,omitempty" example:"categories:work123"`

	// Emotion
	EmotionID *string `json:"emotion_id,omitempty" validate:"omitempty,emotion_id" example:"emotions:E16"`

	// Quantity (for measurable contributions)
	Quantity *QuantityInput `json:"quantity,omitempty"`

	// Activity source (creates created_from_activity edge)
	ActivityID string `json:"activity_id,omitempty" example:"activities:morning_run"`

	// Goal linking (creates task_goals edges)
	GoalLinks []GoalLinkInput `json:"goal_links,omitempty"`
}

// QuantityInput is the input format for quantity.
type QuantityInput struct {
	Value  float64 `json:"value" validate:"gte=0" example:"5.0"`
	UnitID string  `json:"unit_id" validate:"required" example:"units:km"`
}

// GoalLinkInput is the input for linking a task to a goal.
type GoalLinkInput struct {
	GoalID         string  `json:"goal_id" validate:"required" example:"goals:hydration123"`
	ImpactType     string  `json:"impact_type,omitempty" validate:"omitempty,oneof=positive negative neutral" example:"positive"`
	QuantityValue  float64 `json:"quantity_value,omitempty" example:"5.0"`
	IsMilestone    bool    `json:"is_milestone,omitempty" example:"false"`
	MilestoneLabel string  `json:"milestone_label,omitempty" example:"Module 3 Complete"`
	MilestoneOrder int     `json:"milestone_order,omitempty" example:"3"`
	Notes          string  `json:"notes,omitempty" example:"Morning session"`
}

// UpdateRequest is the request payload for updating a task.
//
// All fields are optional. Only provided fields will be updated.
// Use null or empty string for category_id to remove the category link.
// Use null for emotion_id to remove the emotion.
//
// @Description Request payload for updating a task
type UpdateRequest struct {
	Title     *string          `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Journal   *string          `json:"journal,omitempty" validate:"omitempty,max=10000"`
	StartDate *string          `json:"start_date,omitempty" validate:"omitempty,datetime_flexible"`
	EndDate   *string          `json:"end_date,omitempty" validate:"omitempty,datetime_flexible"`
	Completed *bool            `json:"completed,omitempty"`
	Note      *string          `json:"note,omitempty" validate:"omitempty,max=5000"`
	Positives []TaskItem       `json:"positives,omitempty"`
	Negatives []TaskItem       `json:"negatives,omitempty"`
	EmotionID *string          `json:"emotion_id,omitempty" validate:"omitempty,emotion_id"`
	Quantity  *QuantityInput   `json:"quantity,omitempty"`
	GoalLinks *[]GoalLinkInput `json:"goal_links,omitempty"`
}

// LinkGoalRequest is the request for linking a task to a goal.
//
// @Description Request for linking task to goal
type LinkGoalRequest struct {
	GoalID         string  `json:"goal_id" validate:"required" example:"goals:hydration123"`
	ImpactType     string  `json:"impact_type,omitempty" validate:"omitempty,oneof=positive negative neutral" example:"positive"`
	QuantityValue  float64 `json:"quantity_value,omitempty" example:"5.0"`
	IsMilestone    bool    `json:"is_milestone,omitempty" example:"false"`
	MilestoneLabel string  `json:"milestone_label,omitempty" example:"Module 3 Complete"`
	MilestoneOrder int     `json:"milestone_order,omitempty" example:"3"`
	Notes          string  `json:"notes,omitempty" example:"Morning session"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// TaskPageResponse documents the paginated response returned by the List endpoint.
//
// @Description Paginated list of tasks
type TaskPageResponse struct {
	Items   []*Task `json:"items"`
	Total   int64   `json:"total"`
	Limit   int     `json:"limit"`
	Offset  int     `json:"offset"`
	HasMore bool    `json:"has_more"`
}

// TaskGoalsResponse is the response for GET /tasks/{id}/goals.
//
// @Description Goals linked to a task
type TaskGoalsResponse struct {
	TaskID string         `json:"task_id"`
	Goals  []TaskGoalLink `json:"goals"`
}

// =============================================================================
// FILTER PARAMETERS
// =============================================================================

// TaskFilterParams contains filter criteria for listing tasks.
//
// Filters can be combined (AND logic). Empty values are ignored.
// Search matches on title, journal, and note fields.
//
// @Description Query parameters for filtering tasks
type TaskFilterParams struct {
	// Search performs full-text search across title, journal, and note fields
	Search string `json:"search,omitempty"`

	// CategoryID filters by specific category (via in_category edge)
	CategoryID string `json:"category_id,omitempty"`

	// NoCategoryFilter filters for tasks without any category assigned
	NoCategoryFilter bool `json:"no_category,omitempty"`

	// Status filters by completion status: "all", "completed", "pending"
	Status string `json:"status,omitempty"`

	// Date range filters
	StartDateFrom string `json:"start_date_from,omitempty"` // RFC3339
	StartDateTo   string `json:"start_date_to,omitempty"`   // RFC3339

	// Goal filter (via task_goals edge)
	GoalID string `json:"goal_id,omitempty"`

	// Activity filter (via created_from_activity edge)
	ActivityID string `json:"activity_id,omitempty"`

	// HasQuantity filters for tasks with quantity set
	HasQuantity *bool `json:"has_quantity,omitempty"`

	// Sorting
	SortField string `json:"sort_field,omitempty"` // start_date, title, created_at
	SortOrder string `json:"sort_order,omitempty"` // asc, desc
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the database table name for tasks.
	Table = "tasks"

	// Source types
	SourceManual   = "manual"
	SourceActivity = "activity"
	SourceQuick    = "quick"

	// Status filter constants
	StatusAll       = "all"
	StatusCompleted = "completed"
	StatusPending   = "pending"

	// Sort field constants
	SortByStartDate = "start_date"
	SortByTitle     = "title"
	SortByCreatedAt = "created_at"

	// Sort order constants
	SortAsc  = "asc"
	SortDesc = "desc"

	// Relation table names
	TaskGoalsTable           = "task_goals"
	CreatedFromActivityTable = "created_from_activity"
)

// Impact types for task-goal links
const (
	ImpactPositive = "positive"
	ImpactNegative = "negative"
	ImpactNeutral  = "neutral"
)
