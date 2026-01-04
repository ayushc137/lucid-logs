// Package units provides unit management for quantity tracking.
//
// This package implements:
//   - System-defined units (km, miles, hours, liters, etc.)
//   - User-defined custom units
//   - Unit lookup for goal/task quantity validation
//
// Units are used by:
//   - Goal targets (e.g., "Run 100 km")
//   - Task quantities (e.g., "Ran 5 km")
//   - Task-goal links (e.g., quantity contribution)
//
// Database Architecture:
//
//	units table with system and user-created units
//	Seeded with common units on first run
package units

import "time"

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// Unit represents a unit of measurement for quantities.
//
// Units can be system-provided (is_system=true) or user-created.
// System units cannot be modified or deleted.
//
// @Description Unit of measurement for tracking quantities
type Unit struct {
	ID        string `json:"id,omitempty"` // units:km, units:hr, etc.
	CreatedBy string `json:"-"`            // User ID (empty for system units)

	// Core fields
	Name   string `json:"name"`   // "kilometers", "hours", "liters"
	Symbol string `json:"symbol"` // "km", "hr", "L"
	Type   string `json:"type"`   // "distance", "time", "volume", "count", "custom"

	// System flag
	IsSystem bool `json:"is_system"` // true = system-provided, cannot delete

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for creating a custom unit.
//
// @Description Request payload for creating a custom unit
type CreateRequest struct {
	Name   string `json:"name" validate:"required,min=1,max=50" example:"pomodoros"`
	Symbol string `json:"symbol" validate:"required,min=1,max=10" example:"🍅"`
	Type   string `json:"type,omitempty" validate:"omitempty,oneof=distance time volume count custom" example:"custom"`
}

// UpdateRequest is the request payload for updating a custom unit.
//
// @Description Request payload for updating a custom unit
type UpdateRequest struct {
	Name   *string `json:"name,omitempty" validate:"omitempty,min=1,max=50" example:"focus sessions"`
	Symbol *string `json:"symbol,omitempty" validate:"omitempty,min=1,max=10" example:"⏱️"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// UnitListResponse is the response for GET /units.
//
// @Description List of available units
type UnitListResponse struct {
	Items      []*Unit `json:"items"`
	SystemOnly bool    `json:"system_only,omitempty"` // If filtered to system units only
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name for units.
	Table = "units"

	// Unit types
	TypeDistance = "distance"
	TypeTime     = "time"
	TypeVolume   = "volume"
	TypeCount    = "count"
	TypeCustom   = "custom"
)

// SystemUnits defines all built-in units that are seeded on startup.
// These cannot be deleted or modified by users.
var SystemUnits = []Unit{
	// Distance
	{ID: "units:km", Name: "kilometers", Symbol: "km", Type: TypeDistance, IsSystem: true},
	{ID: "units:mi", Name: "miles", Symbol: "mi", Type: TypeDistance, IsSystem: true},
	{ID: "units:m", Name: "meters", Symbol: "m", Type: TypeDistance, IsSystem: true},
	{ID: "units:steps", Name: "steps", Symbol: "steps", Type: TypeDistance, IsSystem: true},

	// Time
	{ID: "units:min", Name: "minutes", Symbol: "min", Type: TypeTime, IsSystem: true},
	{ID: "units:hr", Name: "hours", Symbol: "hr", Type: TypeTime, IsSystem: true},
	{ID: "units:sec", Name: "seconds", Symbol: "sec", Type: TypeTime, IsSystem: true},

	// Volume
	{ID: "units:l", Name: "liters", Symbol: "L", Type: TypeVolume, IsSystem: true},
	{ID: "units:ml", Name: "milliliters", Symbol: "ml", Type: TypeVolume, IsSystem: true},
	{ID: "units:cups", Name: "cups", Symbol: "cups", Type: TypeVolume, IsSystem: true},

	// Count
	{ID: "units:count", Name: "count", Symbol: "×", Type: TypeCount, IsSystem: true},
	{ID: "units:pages", Name: "pages", Symbol: "pg", Type: TypeCount, IsSystem: true},
	{ID: "units:reps", Name: "repetitions", Symbol: "reps", Type: TypeCount, IsSystem: true},
	{ID: "units:sets", Name: "sets", Symbol: "sets", Type: TypeCount, IsSystem: true},
	{ID: "units:items", Name: "items", Symbol: "items", Type: TypeCount, IsSystem: true},

	// Other common units
	{ID: "units:cal", Name: "calories", Symbol: "cal", Type: TypeCount, IsSystem: true},
	{ID: "units:dollars", Name: "dollars", Symbol: "$", Type: TypeCount, IsSystem: true},
}
