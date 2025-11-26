package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	SourceManual = "manual"
)

// BaseModel contains common fields for all database items
type BaseModel struct {
	ID        string     `json:"id,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"` // Soft delete
	CreatedBy string     `json:"created_by"`           // User ID
	UpdatedBy string     `json:"updated_by"`           // User ID
}

// Task represents a daily task or event
type DateOnly struct {
	time.Time
}

func (d *DateOnly) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "" || s == "null" {
		d.Time = time.Time{}
		return nil
	}

	layouts := []string{
		"2006-01-02",
		time.RFC3339,
	}

	var parsed time.Time
	var err error
	for _, layout := range layouts {
		parsed, err = time.Parse(layout, s)
		if err == nil {
			d.Time = NormalizeDate(parsed)
			return nil
		}
	}

	return fmt.Errorf("invalid date format %q, expected YYYY-MM-DD", s)
}

func (d DateOnly) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.Time.UTC().Format("2006-01-02"))
}

func (d *DateOnly) TimeValue() time.Time {
	if d == nil {
		return time.Time{}
	}
	return d.Time
}

func NewDateOnly(t time.Time) DateOnly {
	return DateOnly{Time: NormalizeDate(t)}
}

type Task struct {
	BaseModel

	Title   string `json:"title" validate:"required" example:"Plan tomorrow"`
	Journal string `json:"journal" example:"Capture high-level goals"`

	// Time tracking
	StartDate time.Time `json:"start_date" validate:"required" swaggertype:"string" format:"date-time" example:"2025-11-24T00:00:00Z"`
	EndDate   time.Time `json:"end_date" validate:"required,gtfield=StartDate" swaggertype:"string" format:"date-time" example:"2025-11-25T00:00:00Z"`

	// Status & Priority
	IsCompleted bool `json:"is_completed" example:"false"`
	Priority    int  `json:"priority" example:"1"`   // -ve = wishes, higher = higher priority
	Planned     bool `json:"planned" example:"true"` // Auto-calculated

	// Metadata
	Source string `json:"source" example:"manual" enums:"manual" default:"manual"` // Default: "manual"

	// Content
	Notes     string   `json:"note" example:"Focus on top priorities"`
	Positives []string `json:"positives" example:"[\"Felt great\",\"In flow\"]"`
	Negatives []string `json:"negatives" example:"[\"Got distracted\"]"`
}

// CalculatePlannedStatus determines if a task is "Planned" based on time rules
func (t *Task) CalculatePlannedStatus() {
	t.StartDate = NormalizeDate(t.StartDate)
	t.EndDate = NormalizeDate(t.EndDate)

	now := NormalizeDate(time.Now().UTC())
	// "default true for future items start >= now and != end"
	if (t.StartDate.After(now) || t.StartDate.Equal(now)) && !t.StartDate.Equal(t.EndDate) {
		t.Planned = true
	} else {
		t.Planned = false
	}
}

// SetDefaults sets default values for new tasks
func (t *Task) SetDefaults() {
	if t.Source == "" || t.Source != SourceManual {
		t.Source = SourceManual
	}
	// Priority defaults to 0 (neutral) if not set
}

func NormalizeDate(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// CreateTaskRequest Payload
type CreateTaskRequest struct {
	Title     string   `json:"title" binding:"required" example:"Plan tomorrow"`
	Journal   string   `json:"journal" example:"Capture high-level goals"`
	StartDate DateOnly `json:"start_date" binding:"required" swaggertype:"string" format:"date-time" example:"2025-11-24T00:00:00Z"`
	EndDate   DateOnly `json:"end_date" binding:"required" swaggertype:"string" format:"date-time" example:"2025-11-25T00:00:00Z"`
	Priority  int      `json:"priority"`
	Source    string   `json:"source" binding:"omitempty,oneof=manual" enums:"manual" default:"manual"`
	Note      string   `json:"note"`
	Positives []string `json:"positives"`
	Negatives []string `json:"negatives"`
}

// UpdateTaskRequest Payload
type UpdateTaskRequest struct {
	Title       *string   `json:"title"`
	Journal     *string   `json:"journal"`
	StartDate   *DateOnly `json:"start_date" swaggertype:"string" format:"date-time" example:"2025-11-26T00:00:00Z"`
	EndDate     *DateOnly `json:"end_date" swaggertype:"string" format:"date-time" example:"2025-11-27T00:00:00Z"`
	IsCompleted *bool     `json:"is_completed"`
	Priority    *int      `json:"priority"`
	Note        *string   `json:"note"`
	Positives   []string  `json:"positives"`
	Negatives   []string  `json:"negatives"`
}

// AuthRequest for Login/Register
type AuthRequest struct {
	Username string `json:"username" binding:"required" example:"admin@example.com"`
	Password string `json:"password" binding:"required" example:"adminadmin"`
}

// AuthResponse
type AuthResponse struct {
	Token string `json:"token"`
	User  string `json:"user"` // User ID
}
