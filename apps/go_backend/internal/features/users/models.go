package users

import (
	"time"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// User represents a user record with metadata (API-facing domain model).
// Use string IDs for API boundaries.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// userDB is the internal database representation of a user.
//
// This struct uses models.RecordID for the ID field, allowing SurrealDB SDK
// to populate it directly without type::string casts in queries.
// Convert to domain model via toUser() at the repository boundary.
type userDB struct {
	ID        models.RecordID `json:"id,omitempty"`
	Email     string          `json:"email"`
	IsAdmin   bool            `json:"is_admin"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// toUser converts the database model to the domain model.
//
// This is the boundary conversion point where models.RecordID is
// converted to string for API responses.
func (u *userDB) toUser() *User {
	return &User{
		ID:        database.ToStringID(u.ID),
		Email:     u.Email,
		IsAdmin:   u.IsAdmin,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// UpdateRequest represents allowed user updates.
type UpdateRequest struct {
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
	Password *string `json:"password,omitempty" validate:"omitempty,min=6"`
}
