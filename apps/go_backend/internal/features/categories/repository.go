// Package categories provides category management functionality on libSQL.
//
// Categories are used to organize tasks. Each user can have multiple
// categories, and each task can be linked to one category.
//
// Database Helpers Used:
//   - database.QueryFirst[T]()  - Single record queries
//   - database.QueryAll[T]()    - Multi-record queries
//   - database.QueryScalar[T]() - Scalar value queries (COUNT)
//   - database.Create[T]()      - Record creation
//   - database.Merge[T]()       - Partial updates
//
// RecordID Convention:
//   - categoryDB uses models.RecordID for ID field, keeping the public
//     "categories:<uuid>" identifier contract. Conversion to a plain string
//     happens in toCategory() at the repository boundary.
package categories

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	models "github.com/lucid-logs/go-backend/internal/shared/recordid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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
// The ID field uses models.RecordID to preserve the public "categories:<uuid>"
// identifier contract. The SQLite column stores the full "categories:<uuid>"
// string; toCategory() unwraps it for API responses.
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
// FIND OPERATIONS
// =============================================================================

// FindByID retrieves a category by ID for the given user.
//
// Uses database.QueryFirst[T]() to fetch a single record. Soft-deleted
// categories are filtered out by the deleted_at IS NULL clause.
func (r *repository) FindByID(ctx context.Context, id, userID string) (*Category, error) {
	categoryID := database.RecordID(Table, id)

	cat, err := database.QueryFirst[categoryDB](ctx, r.db, `
		SELECT id, name, color, created_at, updated_at, deleted_at, created_by, updated_by
		FROM categories
		WHERE id = $id AND created_by = $user AND deleted_at IS NULL
	`, map[string]any{
		"id":   categoryID,
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", id).Msg("query failed for category fetch")
		return nil, err
	}

	if cat == nil {
		return nil, errors.ErrNotFound
	}

	return cat.toCategory(), nil
}

// FindPaginated retrieves categories for a user with pagination and optional search.
//
// This uses:
//   - database.QueryScalar[T]() for counting records
//   - database.QueryAll[T]() for fetching paginated results
func (r *repository) FindPaginated(ctx context.Context, userID string, params pagination.Params, search string) ([]*Category, int64, error) {
	// Build dynamic query for count
	countQuery := `
		SELECT COUNT(*) FROM categories
		WHERE created_by = $user AND deleted_at IS NULL
	`

	queryParams := map[string]any{
		"user":   userID,
		"limit":  params.Limit,
		"offset": params.Offset,
	}

	// Add search filter if provided (case-insensitive substring match)
	if search != "" {
		countQuery += ` AND LOWER(name) LIKE '%' || LOWER($search) || '%'`
		queryParams["search"] = search
	}

	// Get count using QueryScalar
	total, err := database.QueryScalar[int64](ctx, r.db, countQuery, queryParams)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("QueryScalar failed for category count")
		return nil, 0, err
	}

	// Build dynamic query for results
	resultQuery := `
		SELECT id, name, color, created_at, updated_at, deleted_at, created_by, updated_by
		FROM categories
		WHERE created_by = $user AND deleted_at IS NULL
	`

	if search != "" {
		resultQuery += ` AND LOWER(name) LIKE '%' || LOWER($search) || '%'`
	}

	resultQuery += ` ORDER BY name ASC LIMIT $limit OFFSET $offset`

	// Get categories using typed QueryAll
	catsDB, err := database.QueryAll[categoryDB](ctx, r.db, resultQuery, queryParams)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("QueryAll failed for category list")
		return nil, 0, err
	}

	// Convert to domain models
	cats := make([]*Category, len(catsDB))
	for i := range catsDB {
		cats[i] = catsDB[i].toCategory()
	}

	return cats, total, nil
}

// =============================================================================
// CREATE OPERATION
// =============================================================================

// Create creates a new category using the database.Create helper.
//
// The category ID is generated as "categories:<hex>" and the row is inserted
// with all fields populated. The freshly created row is returned.
func (r *repository) Create(ctx context.Context, req *CreateRequest, userID string) (*Category, error) {
	// Check for duplicate name
	if err := r.checkDuplicateName(ctx, req.Name, "", userID); err != nil {
		return nil, err
	}

	categoryID := generateCategoryRecordID()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	data := map[string]any{
		"id":         categoryID,
		"name":       req.Name,
		"color":      req.Color,
		"created_by": userID,
		"updated_by": userID,
		"created_at": now,
		"updated_at": now,
	}

	cat, err := database.Create[categoryDB](ctx, r.db, Table, data)
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", categoryID.String()).Msg("Create failed for category")
		return nil, err
	}

	if cat == nil {
		return nil, errors.ErrInternal.WithMessage("Failed to create category")
	}

	r.logger.Info().Str("category_id", categoryID.String()).Msg("category created")

	return cat.toCategory(), nil
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

// Update updates an existing category using database.Merge.
//
// Only the fields supplied on UpdateRequest are written, plus updated_at and
// updated_by. The freshly merged row is returned.
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

	categoryID := database.RecordID(Table, id)
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

	cat, err := database.Merge[categoryDB](ctx, r.db, categoryID, updateData)
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", id).Msg("Merge failed for category update")
		return nil, err
	}

	if cat == nil {
		return nil, errors.ErrNotFound
	}

	r.logger.Info().Str("category_id", id).Msg("category updated")

	return cat.toCategory(), nil
}

// =============================================================================
// DELETE OPERATION
// =============================================================================

// Delete soft-deletes a category by setting deleted_at via database.Merge.
func (r *repository) Delete(ctx context.Context, id, userID string) error {
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return err
	}

	categoryID := database.RecordID(Table, id)
	now := time.Now().UTC()

	_, err = database.Merge[categoryDB](ctx, r.db, categoryID, map[string]any{
		"deleted_at": now,
		"updated_by": userID,
		"updated_at": now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("category_id", id).Msg("Merge failed for soft delete")
		return err
	}

	r.logger.Info().Str("category_id", id).Msg("category soft-deleted")
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// checkDuplicateName checks if a category name already exists for the user.
//
// excludeID is the category ID to exclude from the duplicate check (used when
// updating a category in place). An empty excludeID checks against all the
// user's categories.
func (r *repository) checkDuplicateName(ctx context.Context, name, excludeID, userID string) error {
	excludeRecordID := database.RecordID(Table, "none")
	if excludeID != "" {
		excludeRecordID = database.RecordID(Table, excludeID)
	}

	cats, err := database.QueryAll[categoryDB](ctx, r.db, `
		SELECT id FROM categories
		WHERE created_by = $user AND name = $name AND deleted_at IS NULL AND id != $exclude_id
		LIMIT 1
	`, map[string]any{
		"user":       userID,
		"name":       name,
		"exclude_id": excludeRecordID,
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
