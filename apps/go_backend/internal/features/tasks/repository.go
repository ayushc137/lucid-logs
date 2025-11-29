// Package tasks provides task management functionality using SurrealDB SDK.
//
// This package implements:
//   - CRUD operations for tasks using typed SDK methods
//   - Category linking (record links)
//   - Soft delete support
//   - Pagination
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
package tasks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/daily-journal/go-backend/internal/features/categories"
	"github.com/daily-journal/go-backend/internal/shared/database"
	"github.com/daily-journal/go-backend/internal/shared/errors"
	"github.com/daily-journal/go-backend/internal/shared/pagination"
	"github.com/daily-journal/go-backend/internal/shared/validator"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the task data access interface.
//
// This interface enables:
//   - Dependency injection
//   - Easy mocking for tests
//   - Swapping implementations (e.g., caching layer)
type Repository interface {
	// FindByID retrieves a task by ID for a specific user.
	FindByID(ctx context.Context, id, userID string) (*Task, error)

	// FindPaginated retrieves tasks for a user with pagination.
	FindPaginated(ctx context.Context, userID string, params pagination.Params) ([]*Task, int64, error)

	// Create creates a new task.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Task, error)

	// Update updates an existing task.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Task, error)

	// Delete soft-deletes a task.
	Delete(ctx context.Context, id, userID string) error
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

// repository is the production implementation of Repository.
type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new task Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "tasks").Logger(),
	}
}

// =============================================================================
// DATABASE MODEL
// =============================================================================

// taskDB is the internal database representation of a task.
// This struct maps directly to SurrealDB fields with proper JSON tags.
type taskDB struct {
	ID        string               `json:"id,omitempty"`
	Title     string               `json:"title"`
	Journal   string               `json:"journal"`
	StartDate time.Time            `json:"start_date"`
	EndDate   time.Time            `json:"end_date"`
	Completed bool                 `json:"completed"`
	Priority  int                  `json:"priority"`
	Source    string               `json:"source"`
	Note      string               `json:"note"`
	Positives []string             `json:"positives"`
	Negatives []string             `json:"negatives"`
	Category  *categories.Category `json:"category,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
	DeletedAt *time.Time           `json:"deleted_at,omitempty"`
	CreatedBy string               `json:"created_by"`
	UpdatedBy string               `json:"updated_by"`
}

// toTask converts the database model to the domain model.
func (t *taskDB) toTask() *Task {
	return &Task{
		ID:        t.ID,
		Title:     t.Title,
		Journal:   t.Journal,
		StartDate: t.StartDate,
		EndDate:   t.EndDate,
		Completed: t.Completed,
		Priority:  t.Priority,
		Source:    t.Source,
		Note:      t.Note,
		Positives: t.Positives,
		Negatives: t.Negatives,
		Category:  t.Category,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
		DeletedAt: t.DeletedAt,
		CreatedBy: t.CreatedBy,
		UpdatedBy: t.UpdatedBy,
	}
}

// =============================================================================
// CREATE DATA STRUCTURES
// =============================================================================

// taskCreateData is the data structure for creating a task.
// This matches SurrealDB's expected format for CREATE operations.
type taskCreateData struct {
	Title     string   `json:"title"`
	Journal   string   `json:"journal"`
	StartDate string   `json:"start_date"` // ISO8601 string for SurrealDB
	EndDate   string   `json:"end_date"`   // ISO8601 string for SurrealDB
	Completed bool     `json:"completed"`
	Priority  int      `json:"priority"`
	Source    string   `json:"source"`
	Note      string   `json:"note"`
	Positives []string `json:"positives"`
	Negatives []string `json:"negatives"`
	Category  any      `json:"category,omitempty"` // Record link or nil
	CreatedBy string   `json:"created_by"`
	UpdatedBy string   `json:"updated_by"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// taskMergeData is the data structure for merging/updating a task.
type taskMergeData struct {
	Title     *string  `json:"title,omitempty"`
	Journal   *string  `json:"journal,omitempty"`
	StartDate *string  `json:"start_date,omitempty"`
	EndDate   *string  `json:"end_date,omitempty"`
	Completed *bool    `json:"completed,omitempty"`
	Priority  *int     `json:"priority,omitempty"`
	Note      *string  `json:"note,omitempty"`
	Positives []string `json:"positives,omitempty"`
	Negatives []string `json:"negatives,omitempty"`
	Category  any      `json:"category,omitempty"` // Record link, nil, or omitted
	UpdatedBy string   `json:"updated_by"`
	UpdatedAt string   `json:"updated_at"`
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

// FindByID retrieves a task by ID for a specific user using SDK methods.
//
// This uses the database.QueryFirst[T]() SDK wrapper for type-safe queries.
// The query fetches the task with its category hydrated via FETCH.
func (r *repository) FindByID(ctx context.Context, id, userID string) (*Task, error) {
	taskID := formatTaskID(id)

	// Use SDK's typed query to fetch task with category
	// fn::task::with_category is a server-side function that handles FETCH
	task, err := database.QueryFirst[taskDB](ctx, r.db, `
		RETURN fn::task::with_category(type::thing($id))
	`, map[string]any{
		"id": taskID,
	})
	if err != nil {
		r.logger.Error().Err(err).
			Str("task_id", id).
			Str("user_id", userID).
			Msg("SDK query failed for task fetch")
		return nil, err
	}

	if task == nil {
		return nil, errors.ErrNotFound
	}

	// Verify ownership (defense in depth - DB should also check this)
	if task.CreatedBy != userID || task.DeletedAt != nil {
		return nil, errors.ErrNotFound
	}

	result := task.toTask()

	// Sanitize: remove deleted categories from response
	sanitizeCategory(result)

	return result, nil
}

// FindPaginated retrieves tasks for a user with pagination using SDK methods.
//
// This uses:
//   - database.QueryScalar[T]() for counting records
//   - database.QueryAll[T]() for fetching paginated results
func (r *repository) FindPaginated(ctx context.Context, userID string, params pagination.Params) ([]*Task, int64, error) {
	// Get total count using server-side function with SDK's QueryScalar
	total, err := database.QueryScalar[float64](ctx, r.db, `
		RETURN fn::task::count_for_user($user)
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("SDK QueryScalar failed for task count")
		return nil, 0, err
	}

	// Get paginated tasks using SDK's typed QueryAll
	tasksDB, err := database.QueryAll[taskDB](ctx, r.db, `
		SELECT * FROM tasks
		WHERE created_by = $user AND deleted_at = NONE
		ORDER BY start_date DESC
		LIMIT $limit START $offset
		FETCH category
	`, map[string]any{
		"user":   userID,
		"limit":  params.Limit,
		"offset": params.Offset,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("SDK QueryAll failed for task list")
		return nil, 0, err
	}

	// Convert to domain models and sanitize
	tasks := make([]*Task, len(tasksDB))
	for i := range tasksDB {
		task := tasksDB[i].toTask()
		sanitizeCategory(task)
		tasks[i] = task
	}

	return tasks, int64(total), nil
}

// =============================================================================
// CREATE OPERATION
// =============================================================================

// Create creates a new task using SDK's Create method.
//
// This uses database.Create[T]() for type-safe record creation.
// See: https://surrealdb.com/docs/sdk/golang/methods/create
func (r *repository) Create(ctx context.Context, req *CreateRequest, userID string) (*Task, error) {
	// Validate category ownership if provided
	var categoryLink any
	if req.CategoryID != "" {
		catID := formatCategoryID(req.CategoryID)
		exists, err := r.validateCategoryOwnership(ctx, catID, userID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.ErrCategoryNotFound
		}
		// Use raw record reference for SurrealDB
		categoryLink = catID
	}

	// Parse dates
	startDate, _ := validator.ParseDateTime(req.StartDate)
	endDate, _ := validator.ParseDateTime(req.EndDate)

	// Prepare default values
	positives := req.Positives
	if positives == nil {
		positives = []string{}
	}
	negatives := req.Negatives
	if negatives == nil {
		negatives = []string{}
	}
	source := req.Source
	if source == "" {
		source = SourceManual
	}

	// Generate task ID
	taskID := generateTaskID()
	now := time.Now().UTC().Format(time.RFC3339)

	// Create task data for SDK Create
	createData := taskCreateData{
		Title:     req.Title,
		Journal:   req.Journal,
		StartDate: startDate.Format(time.RFC3339),
		EndDate:   endDate.Format(time.RFC3339),
		Completed: false,
		Priority:  req.Priority,
		Source:    source,
		Note:      req.Note,
		Positives: positives,
		Negatives: negatives,
		Category:  categoryLink,
		CreatedBy: userID,
		UpdatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Use SDK's Create method for type-safe creation
	_, err := database.Create[taskDB](ctx, r.db, taskID, createData)
	if err != nil {
		r.logger.Error().Err(err).
			Str("task_id", taskID).
			Str("user_id", userID).
			Msg("SDK Create failed for task")
		return nil, err
	}

	r.logger.Info().Str("task_id", taskID).Msg("task created via SDK")

	// Fetch and return the created task (with category hydrated)
	return r.FindByID(ctx, taskID, userID)
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

// Update updates an existing task using SDK's Merge method.
//
// This uses database.Merge[T]() for type-safe partial updates.
// See: https://surrealdb.com/docs/sdk/golang/methods/merge
func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Task, error) {
	// Verify task exists and user has ownership
	if _, err := r.FindByID(ctx, id, userID); err != nil {
		return nil, err
	}

	// Build merge data with only provided fields
	mergeData := taskMergeData{
		UpdatedBy: userID,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if req.Title != nil {
		mergeData.Title = req.Title
	}
	if req.Journal != nil {
		mergeData.Journal = req.Journal
	}
	if req.StartDate != nil {
		if t, err := validator.ParseDateTime(*req.StartDate); err == nil {
			dateStr := t.Format(time.RFC3339)
			mergeData.StartDate = &dateStr
		}
	}
	if req.EndDate != nil {
		if t, err := validator.ParseDateTime(*req.EndDate); err == nil {
			dateStr := t.Format(time.RFC3339)
			mergeData.EndDate = &dateStr
		}
	}
	if req.Completed != nil {
		mergeData.Completed = req.Completed
	}
	if req.Priority != nil {
		mergeData.Priority = req.Priority
	}
	if req.Note != nil {
		mergeData.Note = req.Note
	}
	if req.Positives != nil {
		mergeData.Positives = req.Positives
	}
	if req.Negatives != nil {
		mergeData.Negatives = req.Negatives
	}

	// Handle category update
	if req.CategoryID != nil {
		catID := strings.TrimSpace(*req.CategoryID)
		if catID == "" {
			// Remove category link
			mergeData.Category = nil
		} else {
			// Validate and set new category
			formattedCatID := formatCategoryID(catID)
			exists, err := r.validateCategoryOwnership(ctx, formattedCatID, userID)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, errors.ErrCategoryNotFound
			}
			mergeData.Category = formattedCatID
		}
	}

	taskID := formatTaskID(id)

	// Use SDK's Merge method for partial update
	_, err := database.Merge[taskDB](ctx, r.db, taskID, mergeData)
	if err != nil {
		r.logger.Error().Err(err).Str("task_id", id).Msg("SDK Merge failed for task update")
		return nil, err
	}

	r.logger.Info().Str("task_id", id).Msg("task updated via SDK")

	// Fetch and return updated task
	return r.FindByID(ctx, id, userID)
}

// =============================================================================
// DELETE OPERATION
// =============================================================================

// Delete soft-deletes a task using SDK's Merge method.
//
// This uses database.Merge[T]() to set the deleted_at timestamp.
// See: https://surrealdb.com/docs/sdk/golang/methods/merge
func (r *repository) Delete(ctx context.Context, id, userID string) error {
	// Verify ownership first
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return err
	}

	taskID := formatTaskID(id)
	now := time.Now().UTC().Format(time.RFC3339)

	// Use SDK's Merge method for soft delete
	softDelete := softDeleteData{
		DeletedAt: now,
		UpdatedBy: userID,
		UpdatedAt: now,
	}

	_, err = database.Merge[taskDB](ctx, r.db, taskID, softDelete)
	if err != nil {
		r.logger.Error().Err(err).Str("task_id", id).Msg("SDK Merge failed for soft delete")
		return err
	}

	r.logger.Info().Str("task_id", id).Msg("task soft-deleted via SDK")
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// validateCategoryOwnership checks if a category exists and belongs to the user.
func (r *repository) validateCategoryOwnership(ctx context.Context, categoryID, userID string) (bool, error) {
	// Use SDK's typed query for validation
	cats, err := database.QueryAll[categories.Category](ctx, r.db, `
		SELECT id FROM type::thing($id) 
		WHERE created_by = $user AND deleted_at = NONE
	`, map[string]any{
		"id":   categoryID,
		"user": userID,
	})
	if err != nil {
		return false, errors.ErrDatabase.Wrap(err)
	}

	if len(cats) == 0 {
		r.logger.Warn().
			Str("category_id", categoryID).
			Str("user_id", userID).
			Msg("category validation failed")
		return false, nil
	}

	return true, nil
}

// formatTaskID ensures the ID has the table prefix.
func formatTaskID(id string) string {
	if strings.HasPrefix(id, Table+":") {
		return id
	}
	return Table + ":" + id
}

// formatCategoryID ensures the ID has the table prefix.
func formatCategoryID(id string) string {
	if strings.HasPrefix(id, categories.Table+":") {
		return id
	}
	return categories.Table + ":" + id
}

// generateTaskID generates a unique task ID.
func generateTaskID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return Table + ":" + hex.EncodeToString(bytes)
}

// sanitizeCategory removes deleted categories from task.
func sanitizeCategory(task *Task) {
	if task.Category != nil && task.Category.DeletedAt != nil {
		task.Category = nil
	}
}
