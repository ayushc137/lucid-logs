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
// RecordID Convention:
//   - categoryDB uses models.RecordID for ID field
//   - Conversion to string happens in toCategory() at the repository boundary
//   - This enables type-safe queries without SELECT type::string(id) casts
//
// See: https://surrealdb.com/docs/sdk/golang
package categories

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	models "github.com/lucid-logs/go-backend/internal/shared/recordid"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the category data access interface.
type Repository interface {
	FindByID(ctx context.Context, id, userID string) (*Category, error)
	FindPaginated(ctx context.Context, userID string, params pagination.Params, search string) ([]*Category, int64, error)
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
//
// This struct uses models.RecordID for the ID field, allowing SurrealDB SDK
// to populate it directly without type::string casts in queries.
// Convert to domain model via toCategory() at the repository boundary.
type categoryDB struct {
	ID        models.RecordID `json:"id,omitempty"`
	Name      string          `json:"name"`
	Color     string          `json:"color"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *time.Time      `json:"deleted_at,omitempty"`
	CreatedBy string          `json:"created_by"`
	UpdatedBy string          `json:"updated_by"`
}

// toCategory converts the database model to the domain model.
//
// This is the boundary conversion point where models.RecordID is
// converted to string for API responses.
func (c *categoryDB) toCategory() *Category {
	return &Category{
		ID:        database.ToStringID(c.ID),
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
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

// FindByID retrieves a category by ID using SDK's typed query.
//
// This uses database.QueryFirst[T]() for type-safe single record queries.
// No type::string(id) cast needed since categoryDB.ID is models.RecordID.
func (r *repository) FindByID(ctx context.Context, id, userID string) (*Category, error) {
	categoryID := database.MustRecordID(Table, id)

	// Use SDK's typed query to fetch category
	// models.RecordID handles ID deserialization automatically
	cat, err := database.QueryFirst[categoryDB](ctx, r.db, `
		SELECT * FROM $id
		WHERE deleted_at = NONE AND created_by = $user
	`, map[string]any{
		"id":   categoryID,
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", id).Msg("SDK query failed for category fetch")
		return nil, err
	}

	if cat == nil {
		return nil, errors.ErrNotFound
	}

	// Verify ownership
	if cat.DeletedAt != nil {
		return nil, errors.ErrNotFound
	}

	return cat.toCategory(), nil
}

// FindPaginated retrieves categories for a user with pagination and optional search.
//
// This uses:
//   - database.QueryScalar[T]() for counting records
//   - database.QueryAll[T]() for fetching paginated results
//
// No type::string(id) cast needed since categoryDB.ID is models.RecordID.
func (r *repository) FindPaginated(ctx context.Context, userID string, params pagination.Params, search string) ([]*Category, int64, error) {
	// Build dynamic query for count
	countQuery := `
		RETURN (SELECT count() FROM categories
			WHERE created_by = $user AND deleted_at = NONE`

	queryParams := map[string]any{
		"user":   userID,
		"limit":  params.Limit,
		"offset": params.Offset,
	}

	// Add search filter if provided
	if search != "" {
		countQuery += ` AND string::lowercase(name) CONTAINS string::lowercase($search)`
		queryParams["search"] = search
	}

	countQuery += ` GROUP ALL)[0].count OR 0`

	// Get count using SDK's QueryScalar
	total, err := database.QueryScalar[float64](ctx, r.db, countQuery, queryParams)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("SDK QueryScalar failed for category count")
		return nil, 0, err
	}

	// Build dynamic query for results
	resultQuery := `
		SELECT * FROM categories
		WHERE created_by = $user AND deleted_at = NONE`

	if search != "" {
		resultQuery += ` AND string::lowercase(name) CONTAINS string::lowercase($search)`
	}

	resultQuery += ` ORDER BY name ASC LIMIT $limit START $offset`

	// Get categories using SDK's typed QueryAll
	// models.RecordID handles ID deserialization automatically
	catsDB, err := database.QueryAll[categoryDB](ctx, r.db, resultQuery, queryParams)
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
// Category ID uses models.RecordID for type-safe record references.
// See: https://surrealdb.com/docs/sdk/golang/methods/create
func (r *repository) Create(ctx context.Context, req *CreateRequest, userID string) (*Category, error) {
	// Check for duplicate name
	if err := r.checkDuplicateName(ctx, req.Name, "", userID); err != nil {
		return nil, err
	}

	categoryID := generateCategoryRecordID()
	now := time.Now().UTC()

	// Create category data for SDK Create
	createData := categoryCreateData{
		Name:      req.Name,
		Color:     req.Color,
		CreatedBy: userID,
		UpdatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Use SDK's typed QueryAll for creation to ensure correct table/ID handling
	// We use CREATE $id CONTENT $data to force the ID
	cats, err := database.QueryAll[categoryDB](ctx, r.db, `
		CREATE $id CONTENT $data
	`, map[string]any{
		"id":   categoryID,
		"data": createData,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", database.ToStringID(categoryID)).Msg("SDK Create query failed for category")
		return nil, err
	}

	if len(cats) == 0 {
		return nil, errors.ErrInternal.WithMessage("Failed to create category")
	}

	r.logger.Info().Str("category_id", database.ToStringID(categoryID)).Msg("category created via SDK")

	return cats[0].toCategory(), nil
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

// Update updates an existing category using query-based UPDATE.
//
// Uses UPDATE query for reliable single-record updates with models.RecordID.
func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Category, error) {
	existing, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Check for duplicate name if changing
	if req.Name != nil {
		if dupErr := r.checkDuplicateName(ctx, *req.Name, existing.ID, userID); dupErr != nil {
			return nil, dupErr
		}
	}

	categoryID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	// Build update data dynamically
	updateData := map[string]any{
		"updated_by": userID,
		"updated_at": now,
	}
	if req.Name != nil {
		updateData["name"] = *req.Name
	}
	if req.Color != nil {
		updateData["color"] = *req.Color
	}

	// Use UPDATE query for reliable single-record update
	cats, err := database.QueryAll[categoryDB](ctx, r.db, `
		UPDATE $id MERGE $data
	`, map[string]any{
		"id":   categoryID,
		"data": updateData,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", id).Msg("UPDATE query failed for category")
		return nil, err
	}

	if len(cats) == 0 {
		return nil, errors.ErrNotFound
	}

	r.logger.Info().Str("category_id", id).Msg("category updated via UPDATE query")

	return cats[0].toCategory(), nil
}

// =============================================================================
// DELETE OPERATION
// =============================================================================

// Delete soft-deletes a category using query-based UPDATE.
//
// Uses UPDATE query for reliable single-record soft delete with models.RecordID.
func (r *repository) Delete(ctx context.Context, id, userID string) error {
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return err
	}

	categoryID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	// Use UPDATE query for reliable soft delete
	_, err = database.QueryAll[categoryDB](ctx, r.db, `
		UPDATE $id MERGE {
			deleted_at: $now,
			updated_by: $user,
			updated_at: $now
		}
	`, map[string]any{
		"id":   categoryID,
		"now":  now,
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", id).Msg("UPDATE query failed for soft delete")
		return err
	}

	r.logger.Info().Str("category_id", id).Msg("category soft-deleted via UPDATE query")
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// checkDuplicateName checks if a category name already exists for the user.
func (r *repository) checkDuplicateName(ctx context.Context, name, excludeID, userID string) error {
	excludeRID := database.MustRecordID(Table, "none")
	if excludeID != "" {
		excludeRID = database.MustRecordID(Table, excludeID)
	}

	// Use SDK's typed QueryAll for duplicate check
	cats, err := database.QueryAll[categoryDB](ctx, r.db, `
		SELECT id FROM categories
		WHERE created_by = $user AND name = $name AND deleted_at = NONE AND id != $exclude_id
		LIMIT 1
	`, map[string]any{
		"user":       userID,
		"name":       name,
		"exclude_id": excludeRID,
	})
	if err != nil {
		return errors.ErrDatabase.Wrap(err)
	}

	if len(cats) > 0 {
		return errors.ErrCategoryNameExists
	}

	return nil
}

// generateCategoryRecordID generates a unique category ID as models.RecordID.
func generateCategoryRecordID() models.RecordID {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes) //nolint:errcheck // crypto/rand.Read never fails in practice
	return database.NewRecordID(Table, hex.EncodeToString(bytes))
}
