package users

import (
	"time"

	models "github.com/lucid-logs/go-backend/internal/shared/recordid"

	"github.com/lucid-logs/go-backend/internal/shared/database"
)

// User represents a user record with metadata (API-facing domain model).
// Use string IDs for API boundaries.
type User struct {
	ID          string          `json:"id"`
	Email       string          `json:"email"`
	IsAdmin     bool            `json:"is_admin"`
	Preferences UserPreferences `json:"preferences,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// UserPreferences contains user-configurable settings.
type UserPreferences struct {
	DailyRetro      *DailyRetroSettings `json:"daily_retro,omitempty"`
	WeeklyRetroDay  string              `json:"weekly_retro_day,omitempty"`  // "sunday", "monday", etc.
	MonthlyRetroDay int                 `json:"monthly_retro_day,omitempty"` // 1-31
	Timezone        string              `json:"timezone,omitempty"`          // Default timezone
	AI              *AISettings         `json:"ai,omitempty"`
}

// AISettings contains per-user LLM configuration for retro insights.
type AISettings struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`            // preset key or "custom"
	BaseURL  string `json:"base_url,omitempty"`  // custom only
	APIKey   string `json:"api_key,omitempty"`   // write-only — never serialized on GET paths
	Model    string `json:"model"`
	// HasKey is set when constructing GET responses; not stored in DB.
	HasKey bool `json:"has_key,omitempty"`
}

// SafeAISettings returns a copy of the settings suitable for API GET responses:
// the API key is stripped and HasKey is set.
func SafeAISettings(ai *AISettings) *AISettings {
	if ai == nil {
		return nil
	}
	clone := *ai
	clone.HasKey = ai.APIKey != ""
	clone.APIKey = ""
	return &clone
}

// DailyRetroSettings controls automatic daily retro generation.
type DailyRetroSettings struct {
	Enabled             bool   `json:"enabled"`
	Time                string `json:"time"`     // "21:00" (24h format)
	Timezone            string `json:"timezone"` // "Asia/Kolkata"
	NotificationEnabled bool   `json:"notification_enabled"`
	AutoGenerate        bool   `json:"auto_generate"` // Generate even if user doesn't open app
}

// userDB is the internal database representation of a user.
//
// This struct uses models.RecordID for the ID field, keeping the table:value
// to populate it directly without type::string casts in queries.
// Convert to domain model via toUser() at the repository boundary.
type userDB struct {
	ID          models.RecordID `json:"id,omitempty"`
	Email       string          `json:"email"`
	IsAdmin     bool            `json:"is_admin"`
	Preferences *preferencesDB  `json:"preferences,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type preferencesDB struct {
	DailyRetro      *DailyRetroSettings `json:"daily_retro,omitempty"`
	WeeklyRetroDay  string              `json:"weekly_retro_day,omitempty"`
	MonthlyRetroDay int                 `json:"monthly_retro_day,omitempty"`
	Timezone        string              `json:"timezone,omitempty"`
	AI              *AISettings         `json:"ai,omitempty"`
}

// toUser converts the database model to the domain model.
//
// This is the boundary conversion point where models.RecordID is
// converted to string for API responses.
func (u *userDB) toUser() *User {
	user := &User{
		ID:        database.ToStringID(u.ID),
		Email:     u.Email,
		IsAdmin:   u.IsAdmin,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}

	if u.Preferences != nil {
		user.Preferences = UserPreferences{
			DailyRetro:      u.Preferences.DailyRetro,
			WeeklyRetroDay:  u.Preferences.WeeklyRetroDay,
			MonthlyRetroDay: u.Preferences.MonthlyRetroDay,
			Timezone:        u.Preferences.Timezone,
			AI:              u.Preferences.AI,
		}
	}

	return user
}

// UpdateRequest represents allowed user updates.
type UpdateRequest struct {
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
	Password *string `json:"password,omitempty" validate:"omitempty,min=6"`
}

// UpdatePreferencesRequest is for updating user preferences.
//
// @Description Request to update user preferences
type UpdatePreferencesRequest struct {
	DailyRetro      *DailyRetroSettings `json:"daily_retro,omitempty"`
	WeeklyRetroDay  *string             `json:"weekly_retro_day,omitempty"`
	MonthlyRetroDay *int                `json:"monthly_retro_day,omitempty" validate:"omitempty,min=1,max=31"`
	Timezone        *string             `json:"timezone,omitempty"`
	AI              *AISettings         `json:"ai,omitempty"`
}
