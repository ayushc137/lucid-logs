// Package categories provides category management functionality using SurrealDB SDK.
//
// Categories are used to organize tasks. Each user can have multiple
// categories, and each task can be linked to one category.
//
// SDK Methods Used:
//   - database.Select[T]() - Type-safe record selection
//   - database.Create[T]() - Type-safe record creation
//   - database.Merge[T]()  - Type-safe partial updates
//   - database.QueryAll[T]() - Type-safe query execution
//   - database.QueryFirst[T]() - Single record queries
//   - database.QueryScalar[T]() - Scalar value queries
//
// See: https://surrealdb.com/docs/sdk/golang
package categories

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/daily-journal/go-backend/internal/shared/database"
	"github.com/daily-journal/go-backend/internal/shared/errors"
	"github.com/daily-journal/go-backend/internal/shared/pagination"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the category data access interface.
type Repository interface {
	FindByID(ctx context.Context, id, userID string) (*Category, error)
	FindPaginated(ctx context.Context, userID string, params pagination.Params) ([]*Category, int64, error)
	Create(ctx context.Context, req *CreateRequest, userID string) (*Category, error)
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Category, error)
	Delete(ctx context.Context, id, userID string) error
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new category Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "categories").Logger(),
	}
}

// =============================================================================
// DATABASE MODEL
// =============================================================================

// categoryDB is the internal database representation of a category.
// This struct maps directly to SurrealDB fields with proper JSON tags.
type categoryDB struct {
	ID        string     `json:"id,omitempty"`
	Name      string     `json:"name"`
	Color     string     `json:"color"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedBy string     `json:"created_by"`
	UpdatedBy string     `json:"updated_by"`
}

// toCategory converts the database model to the domain model.
func (c *categoryDB) toCategory() *Category {
	return &Category{
		ID:        c.ID,
		Name:      c.Name,
		Color:     c.Color,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		DeletedAt: c.DeletedAt,
		CreatedBy: c.CreatedBy,
		UpdatedBy: c.UpdatedBy,
	}
}

// =============================================================================
// CREATE/UPDATE DATA STRUCTURES
// =============================================================================

// categoryCreateData is the data structure for creating a category.
// This matches SurrealDB's expected format for CREATE operations.
type categoryCreateData struct {
	Name      string `json:"name"`
	Color     string `json:"color"`
	CreatedBy string `json:"created_by"`
	UpdatedBy string `json:"updated_by"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// categoryMergeData is the data structure for merging/updating a category.
type categoryMergeData struct {
	Name      *string `json:"name,omitempty"`
	Color     *string `json:"color,omitempty"`
	UpdatedBy string  `json:"updated_by"`
	UpdatedAt string  `json:"updated_at"`
}

// softDeleteData is the data structure for soft-deleting a record.
type softDeleteData struct {
	DeletedAt string `json:"deleted_at"`
	UpdatedBy string `json:"updated_by"`
	UpdatedAt string `json:"updated_at"`
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

// FindByID retrieves a category by ID using SDK's typed query.
//
// This uses database.QueryFirst[T]() for type-safe single record queries.
func (r *repository) FindByID(ctx context.Context, id, userID string) (*Category, error) {
	categoryID := formatCategoryID(id)

	// Use SDK's typed query to fetch category
	cat, err := database.QueryFirst[categoryDB](ctx, r.db, `
		SELECT * FROM type::thing($id) WHERE deleted_at = NONE
	`, map[string]any{
		"id": categoryID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", id).Msg("SDK query failed for category fetch")
		return nil, err
	}

	if cat == nil {
		return nil, errors.ErrNotFound
	}

	// Verify ownership
	if cat.CreatedBy != userID || cat.DeletedAt != nil {
		return nil, errors.ErrNotFound
	}

	return cat.toCategory(), nil
}

// FindPaginated retrieves categories for a user with pagination using SDK methods.
//
// This uses:
//   - database.QueryScalar[T]() for counting records
//   - database.QueryAll[T]() for fetching paginated results
func (r *repository) FindPaginated(ctx context.Context, userID string, params pagination.Params) ([]*Category, int64, error) {
	// Get count using SDK's QueryScalar
	total, err := database.QueryScalar[float64](ctx, r.db, `
		RETURN fn::category::count_for_user($user)
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("SDK QueryScalar failed for category count")
		return nil, 0, err
	}

	// Get categories using SDK's typed QueryAll
	catsDB, err := database.QueryAll[categoryDB](ctx, r.db, `
		SELECT * FROM categories
		WHERE created_by = $user AND deleted_at = NONE
		ORDER BY name ASC
		LIMIT $limit START $offset
	`, map[string]any{
		"user":   userID,
		"limit":  params.Limit,
		"offset": params.Offset,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("SDK QueryAll failed for category list")
		return nil, 0, err
	}

	// Convert to domain models
	cats := make([]*Category, len(catsDB))
	for i := range catsDB {
		cats[i] = catsDB[i].toCategory()
	}

	return cats, int64(total), nil
}

// =============================================================================
// CREATE OPERATION
// =============================================================================

// Create creates a new category using SDK's Create method.
//
// This uses database.Create[T]() for type-safe record creation.
// See: https://surrealdb.com/docs/sdk/golang/methods/create
func (r *repository) Create(ctx context.Context, req *CreateRequest, userID string) (*Category, error) {
	// Check for duplicate name
	if err := r.checkDuplicateName(ctx, req.Name, "", userID); err != nil {
		return nil, err
	}

	categoryID := generateCategoryID()
	now := time.Now().UTC().Format(time.RFC3339)

	// Create category data for SDK Create
	createData := categoryCreateData{
		Name:      req.Name,
		Color:     req.Color,
		CreatedBy: userID,
		UpdatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Use SDK's Create method for type-safe creation
	_, err := database.Create[categoryDB](ctx, r.db, categoryID, createData)
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", categoryID).Msg("SDK Create failed for category")
		return nil, err
	}

	r.logger.Info().Str("category_id", categoryID).Msg("category created via SDK")

	return r.FindByID(ctx, categoryID, userID)
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

// Update updates an existing category using SDK's Merge method.
//
// This uses database.Merge[T]() for type-safe partial updates.
// See: https://surrealdb.com/docs/sdk/golang/methods/merge
func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Category, error) {
	existing, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Build merge data with only provided fields
	mergeData := categoryMergeData{
		UpdatedBy: userID,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Check for duplicate name if changing
	if req.Name != nil {
		if err := r.checkDuplicateName(ctx, *req.Name, existing.ID, userID); err != nil {
			return nil, err
		}
		mergeData.Name = req.Name
	}
	if req.Color != nil {
		mergeData.Color = req.Color
	}

	categoryID := formatCategoryID(id)

	// Use SDK's Merge method for partial update
	_, err = database.Merge[categoryDB](ctx, r.db, categoryID, mergeData)
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", id).Msg("SDK Merge failed for category update")
		return nil, err
	}

	r.logger.Info().Str("category_id", id).Msg("category updated via SDK")

	return r.FindByID(ctx, id, userID)
}

// =============================================================================
// DELETE OPERATION
// =============================================================================

// Delete soft-deletes a category using SDK's Merge method.
//
// This uses database.Merge[T]() to set the deleted_at timestamp.
// See: https://surrealdb.com/docs/sdk/golang/methods/merge
func (r *repository) Delete(ctx context.Context, id, userID string) error {
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return err
	}

	categoryID := formatCategoryID(id)
	now := time.Now().UTC().Format(time.RFC3339)

	// Use SDK's Merge method for soft delete
	softDelete := softDeleteData{
		DeletedAt: now,
		UpdatedBy: userID,
		UpdatedAt: now,
	}

	_, err = database.Merge[categoryDB](ctx, r.db, categoryID, softDelete)
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", id).Msg("SDK Merge failed for soft delete")
		return err
	}

	r.logger.Info().Str("category_id", id).Msg("category soft-deleted via SDK")
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// checkDuplicateName checks if a category name already exists for the user.
func (r *repository) checkDuplicateName(ctx context.Context, name, excludeID, userID string) error {
	if excludeID == "" {
		excludeID = Table + ":none"
	}

	// Use SDK's typed QueryAll for duplicate check
	cats, err := database.QueryAll[categoryDB](ctx, r.db, `
		SELECT id FROM categories
		WHERE created_by = $user AND name = $name AND deleted_at = NONE AND id != type::thing($exclude_id)
		LIMIT 1
	`, map[string]any{
		"user":       userID,
		"name":       name,
		"exclude_id": excludeID,
	})
	if err != nil {
		return errors.ErrDatabase.Wrap(err)
	}

	if len(cats) > 0 {
		return errors.ErrCategoryNameExists
	}

	return nil
}

func formatCategoryID(id string) string {
	if strings.HasPrefix(id, Table+":") {
		return id
	}
	return Table + ":" + id
}

func generateCategoryID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return Table + ":" + hex.EncodeToString(bytes)
}
