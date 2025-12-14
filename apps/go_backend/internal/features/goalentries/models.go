// Package goalentries provides daily goal tracking functionality.
//
// This package implements:
//   - Daily entry logging for recurring goals (habits)
//   - Automatic task linking
//   - Value tracking for measurable goals
//
// Goal entries are created:
//   - Manually via API when user logs goal completion
//   - Automatically when tasks with matching activity_key are created
package goalentries

import (
	"time"
)

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// GoalEntry represents a single day's tracking entry for a recurring goal.
//
// Entries track whether a goal was met on a specific date, along with
// the actual value achieved and contributing tasks.
type GoalEntry struct {
	ID     string `json:"id,omitempty"`
	GoalID string `json:"goal_id"`

	// The date this entry is for (YYYY-MM-DD, time component ignored)
	Date time.Time `json:"date"`

	// Actual value achieved (for measurable goals)
	Value *float64 `json:"value,omitempty"`

	// Was the target met for this period?
	Met bool `json:"met"`

	// Task IDs that contributed to this entry
	TaskIDs []string `json:"task_ids,omitempty"`

	// Optional notes
	Notes string `json:"notes,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for logging a goal entry.
//
// @Description Request payload for creating a goal entry
type CreateRequest struct {
	Date    string   `json:"date" validate:"required" example:"2025-12-13"`
	Value   *float64 `json:"value,omitempty" example:"1.5"`
	Met     bool     `json:"met" example:"true"`
	Notes   string   `json:"notes,omitempty" validate:"max=1000"`
	TaskIDs []string `json:"task_ids,omitempty"`
}

// UpdateRequest is the request payload for updating a goal entry.
//
// @Description Request payload for updating a goal entry
type UpdateRequest struct {
	Value   *float64 `json:"value,omitempty" example:"2.5"`
	Met     *bool    `json:"met,omitempty" example:"true"`
	Notes   *string  `json:"notes,omitempty" validate:"omitempty,max=1000" example:"Exceeded my target today!"`
	TaskIDs []string `json:"task_ids,omitempty"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// GoalEntryListResponse is the response for listing goal entries.
type GoalEntryListResponse struct {
	GoalID  string        `json:"goal_id"`
	Entries []*GoalEntry  `json:"entries"`
	Summary *EntrySummary `json:"summary,omitempty"`
}

// EntrySummary provides aggregated stats for a date range.
type EntrySummary struct {
	TotalDays   int     `json:"total_days"`
	MetDays     int     `json:"met_days"`
	MissedDays  int     `json:"missed_days"`
	SuccessRate float64 `json:"success_rate"` // met / total * 100
	TotalValue  float64 `json:"total_value,omitempty"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name for goal entries.
	Table = "goal_entries"
)
