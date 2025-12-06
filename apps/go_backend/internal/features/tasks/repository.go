// Package tasks provides task management functionality using SurrealDB SDK.
//
// This package implements:
//   - CRUD operations for tasks using typed SDK methods
//   - Category linking (record links)
//   - Emotion tracking with inferred emotion calculation
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
// RecordID Convention:
//   - taskDB uses models.RecordID for ID and category link fields
//   - Conversion to string happens in toTask() at the repository boundary
//   - This enables type-safe queries without SELECT type::string(id) casts
//
// Emotion Tracking:
//   - InferredEmotion is calculated on CREATE/UPDATE, not on GET
//   - Uses weighted centroid of all emotion tags in positives/negatives
//   - Stored in database for efficient querying
//
// See: https://surrealdb.com/docs/sdk/golang
package tasks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/lucid-logs/go-backend/internal/features/categories"
	"github.com/lucid-logs/go-backend/internal/features/emotions"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/lucid-logs/go-backend/internal/shared/timeutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go/pkg/models"
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
//
// This struct uses models.RecordID for the ID field, allowing SurrealDB SDK
// to populate it directly without type::string casts in queries.
//
// The Category field uses categoryDB when fetched via FETCH clause,
// which also uses models.RecordID for its ID.
type taskDB struct {
	ID        models.RecordID      `json:"id,omitempty"`
	Title     string               `json:"title"`
	Journal   string               `json:"journal"`
	StartDate database.SurrealTime `json:"start_date"`
	EndDate   database.SurrealTime `json:"end_date"`
	Completed bool                 `json:"completed"`
	Priority  int                  `json:"priority"`
	Source    string               `json:"source"`
	Note      string               `json:"note"`
	Positives []TaskItem           `json:"positives"`
	Negatives []TaskItem           `json:"negatives"`
	Category  *categoryDB          `json:"category,omitempty"` // Hydrated via FETCH

	// Emotion tracking
	EmotionID       *string                   `json:"emotion_id,omitempty"`
	InferredEmotion *emotions.InferredEmotion `json:"inferred_emotion,omitempty"`

	CreatedAt database.SurrealTime  `json:"created_at"`
	UpdatedAt database.SurrealTime  `json:"updated_at"`
	DeletedAt *database.SurrealTime `json:"deleted_at,omitempty"`
	CreatedBy string                `json:"created_by"`
	UpdatedBy string                `json:"updated_by"`
}

// categoryDB is the database representation of a category when fetched.
//
// This struct uses models.RecordID for the ID field for type-safe
// SurrealDB interactions. Convert to categories.Category via toCategory().
type categoryDB struct {
	ID        models.RecordID       `json:"id,omitempty"`
	Name      string                `json:"name"`
	Color     string                `json:"color"`
	CreatedAt database.SurrealTime  `json:"created_at"`
	UpdatedAt database.SurrealTime  `json:"updated_at"`
	DeletedAt *database.SurrealTime `json:"deleted_at,omitempty"`
	CreatedBy string                `json:"created_by"`
	UpdatedBy string                `json:"updated_by"`
}

// toCategory converts categoryDB to the domain model.
func (c *categoryDB) toCategory() *categories.Category {
	if c == nil {
		return nil
	}
	var deletedAt *time.Time
	if c.DeletedAt != nil && !c.DeletedAt.IsZero() {
		dt := c.DeletedAt.Time
		deletedAt = &dt
	}
	return &categories.Category{
		ID:        database.ToStringID(c.ID),
		Name:      c.Name,
		Color:     c.Color,
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
		DeletedAt: deletedAt,
		CreatedBy: c.CreatedBy,
		UpdatedBy: c.UpdatedBy,
	}
}

// toTask converts the database model to the domain model.
//
// This is the boundary conversion point where models.RecordID is
// converted to string for API responses.
func (t *taskDB) toTask() *Task {
	var cat *categories.Category
	if t.Category != nil {
		cat = t.Category.toCategory()
	}

	var deletedAt *time.Time
	if t.DeletedAt != nil && !t.DeletedAt.IsZero() {
		dt := t.DeletedAt.Time
		deletedAt = &dt
	}

	return &Task{
		ID:              database.ToStringID(t.ID),
		Title:           t.Title,
		Journal:         t.Journal,
		StartDate:       t.StartDate.Time,
		EndDate:         t.EndDate.Time,
		Completed:       t.Completed,
		Priority:        t.Priority,
		Source:          t.Source,
		Note:            t.Note,
		Positives:       t.Positives,
		Negatives:       t.Negatives,
		Category:        cat,
		EmotionID:       t.EmotionID,
		InferredEmotion: t.InferredEmotion,
		CreatedAt:       t.CreatedAt.Time,
		UpdatedAt:       t.UpdatedAt.Time,
		DeletedAt:       deletedAt,
		CreatedBy:       t.CreatedBy,
		UpdatedBy:       t.UpdatedBy,
	}
}

// =============================================================================
// CREATE DATA STRUCTURES
// =============================================================================

// taskCreateData is the data structure for creating a task.
//
// This matches SurrealDB's expected format for CREATE operations.
// Category uses *models.RecordID for type-safe record linking.
type taskCreateData struct {
	Title     string           `json:"title"`
	Journal   string           `json:"journal"`
	StartDate time.Time        `json:"start_date"`
	EndDate   time.Time        `json:"end_date"`
	Completed bool             `json:"completed"`
	Priority  int              `json:"priority"`
	Source    string           `json:"source"`
	Note      string           `json:"note"`
	Positives []TaskItem       `json:"positives"`
	Negatives []TaskItem       `json:"negatives"`
	Category  *models.RecordID `json:"category,omitempty"` // Record link or nil

	// Emotion tracking
	EmotionID       *string                   `json:"emotion_id,omitempty"`
	InferredEmotion *emotions.InferredEmotion `json:"inferred_emotion,omitempty"`

	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

// FindByID retrieves a task by ID for a specific user using SDK methods.
//
// This uses the database.QueryFirst[T]() SDK wrapper for type-safe queries.
// The query fetches the task with its category hydrated via FETCH.
// No type::string(id) cast needed since taskDB.ID is models.RecordID.
func (r *repository) FindByID(ctx context.Context, id, userID string) (*Task, error) {
	taskID := database.MustRecordID(Table, id)

	// Use SDK's typed query to fetch task with category
	// models.RecordID handles ID serialization automatically
	task, err := database.QueryFirst[taskDB](ctx, r.db, `
		SELECT * FROM type::thing($id) FETCH category
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
//
// No type::string(id) cast needed since taskDB.ID is models.RecordID.
func (r *repository) FindPaginated(ctx context.Context, userID string, params pagination.Params) ([]*Task, int64, error) {
	// Get total count using server-side function with SDK's QueryScalar
	total, err := database.QueryScalar[float64](ctx, r.db, `
		RETURN (SELECT count() FROM tasks
			WHERE created_by = $user AND deleted_at = NONE
			GROUP ALL)[0].count OR 0
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("SDK QueryScalar failed for task count")
		return nil, 0, err
	}

	// Get paginated tasks using SDK's typed QueryAll
	// models.RecordID handles ID deserialization automatically
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
// Category links use models.RecordID for type-safe record references.
// InferredEmotion is calculated at write time from positives/negatives emotions.
// See: https://surrealdb.com/docs/sdk/golang/methods/create
func (r *repository) Create(ctx context.Context, req *CreateRequest, userID string) (*Task, error) {
	// Validate category ownership if provided
	var categoryLink *models.RecordID
	if req.CategoryID != "" {
		catID := database.MustRecordID(categories.Table, req.CategoryID)
		exists, err := r.validateCategoryOwnership(ctx, catID, userID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.ErrCategoryNotFound
		}
		// Use models.RecordID for type-safe record link
		categoryLink = &catID
	}

	// Parse dates
	startDate, _ := timeutil.ParseDateTime(req.StartDate)
	endDate, _ := timeutil.ParseDateTime(req.EndDate)

	// Prepare default values
	positives := req.Positives
	if positives == nil {
		positives = []TaskItem{}
	}
	negatives := req.Negatives
	if negatives == nil {
		negatives = []TaskItem{}
	}
	source := req.Source
	if source == "" {
		source = SourceManual
	}

	// Calculate inferred emotion from positives/negatives
	var inferredEmotion *emotions.InferredEmotion
	emotionItems := toEmotionItems(positives, negatives)
	if len(emotionItems.positives) > 0 || len(emotionItems.negatives) > 0 {
		inferredEmotion = emotions.InferFromItems(emotionItems.positives, emotionItems.negatives)
	} else if req.EmotionID != nil && *req.EmotionID != "" {
		// Fall back to single emotion if no items have emotions
		inferredEmotion = emotions.InferFromSingle(*req.EmotionID)
	}

	// Generate task ID using models.RecordID
	taskID := generateTaskRecordID()
	now := time.Now().UTC()

	// Create task data for SDK Create
	createData := taskCreateData{
		Title:           req.Title,
		Journal:         req.Journal,
		StartDate:       startDate,
		EndDate:         endDate,
		Completed:       false,
		Priority:        req.Priority,
		Source:          source,
		Note:            req.Note,
		Positives:       positives,
		Negatives:       negatives,
		Category:        categoryLink,
		EmotionID:       req.EmotionID,
		InferredEmotion: inferredEmotion,
		CreatedBy:       userID,
		UpdatedBy:       userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Use CREATE query but only decode the ID to avoid time parsing issues
	type createResult struct {
		ID models.RecordID `json:"id"`
	}

	results, err := database.QueryAll[createResult](ctx, r.db, `
		CREATE type::thing($id) CONTENT $data
	`, map[string]any{
		"id":   taskID,
		"data": createData,
	})
	if err != nil {
		r.logger.Error().Err(err).
			Str("task_id", database.ToStringID(taskID)).
			Str("user_id", userID).
			Msg("SDK Create query failed for task")
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.ErrInternal.WithMessage("Failed to create task")
	}

	taskIDStr := database.ToStringID(taskID)
	r.logger.Info().Str("task_id", taskIDStr).Msg("task created via SDK")

	// Sync emotion edges for analytics (async-safe, errors logged not returned)
	r.syncEmotionEdges(ctx, taskIDStr, req.EmotionID, positives, negatives)

	// Fetch and return the created task (with category hydrated)
	return r.FindByID(ctx, taskIDStr, userID)
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

// Update updates an existing task using query-based UPDATE.
//
// Uses UPDATE query for reliable single-record updates with models.RecordID.
// Recalculates InferredEmotion when positives/negatives or emotion_id changes.
func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Task, error) {
	// Verify task exists and user has ownership
	existing, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	taskID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	// Build update data dynamically with only provided fields
	updateData := map[string]any{
		"updated_by": userID,
		"updated_at": now,
	}

	if req.Title != nil {
		updateData["title"] = *req.Title
	}
	if req.Journal != nil {
		updateData["journal"] = *req.Journal
	}

	var (
		newStart *time.Time
		newEnd   *time.Time
	)

	if req.StartDate != nil {
		startStr := strings.TrimSpace(*req.StartDate)
		if startStr == "" {
			return nil, errors.ErrBadRequest.WithMessage("start_date cannot be empty")
		}
		t, err := timeutil.ParseDateTime(startStr)
		if err != nil {
			return nil, errors.ErrBadRequest.WithMessage("Invalid start_date format")
		}
		newStart = &t
		updateData["start_date"] = t
	}
	if req.EndDate != nil {
		endStr := strings.TrimSpace(*req.EndDate)
		if endStr == "" {
			return nil, errors.ErrBadRequest.WithMessage("end_date cannot be empty")
		}
		t, err := timeutil.ParseDateTime(endStr)
		if err != nil {
			return nil, errors.ErrBadRequest.WithMessage("Invalid end_date format")
		}
		newEnd = &t
		updateData["end_date"] = t
	}
	if req.Completed != nil {
		updateData["completed"] = *req.Completed
	}
	if req.Priority != nil {
		updateData["priority"] = *req.Priority
	}
	if req.Note != nil {
		updateData["note"] = *req.Note
	}
	if req.Positives != nil {
		updateData["positives"] = req.Positives
	}
	if req.Negatives != nil {
		updateData["negatives"] = req.Negatives
	}

	// Handle emotion fields
	emotionChanged := false
	if req.EmotionID != nil {
		emotionID := strings.TrimSpace(*req.EmotionID)
		if emotionID == "" {
			updateData["emotion_id"] = nil
		} else {
			updateData["emotion_id"] = emotionID
		}
		emotionChanged = true
	}

	// Handle category update
	if req.CategoryID != nil {
		catID := strings.TrimSpace(*req.CategoryID)
		if catID == "" {
			// Remove category link - set to NONE in SurrealDB
			updateData["category"] = nil
		} else {
			// Validate and set new category
			categoryRID := database.MustRecordID(categories.Table, catID)
			exists, err := r.validateCategoryOwnership(ctx, categoryRID, userID)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, errors.ErrCategoryNotFound
			}
			updateData["category"] = categoryRID
		}
	}

	finalStart := existing.StartDate
	if newStart != nil {
		finalStart = *newStart
	}
	finalEnd := existing.EndDate
	if newEnd != nil {
		finalEnd = *newEnd
	}
	if finalEnd.Before(finalStart) {
		return nil, errors.ErrInvalidDateRange
	}

	// Determine final values for positives, negatives, and emotion
	finalPositives := existing.Positives
	if req.Positives != nil {
		finalPositives = req.Positives
	}
	finalNegatives := existing.Negatives
	if req.Negatives != nil {
		finalNegatives = req.Negatives
	}
	finalEmotionID := existing.EmotionID
	if req.EmotionID != nil {
		finalEmotionID = req.EmotionID
	}

	// Recalculate inferred emotion if positives, negatives, or emotion changed
	if req.Positives != nil || req.Negatives != nil || emotionChanged {
		// Calculate inferred emotion
		emotionItems := toEmotionItems(finalPositives, finalNegatives)
		if len(emotionItems.positives) > 0 || len(emotionItems.negatives) > 0 {
			updateData["inferred_emotion"] = emotions.InferFromItems(emotionItems.positives, emotionItems.negatives)
		} else {
			// Fall back to single emotion
			if finalEmotionID != nil && *finalEmotionID != "" {
				updateData["inferred_emotion"] = emotions.InferFromSingle(*finalEmotionID)
			} else {
				updateData["inferred_emotion"] = nil
			}
		}
	}

	// Use UPDATE query for reliable single-record update
	_, err = database.QueryAll[taskDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE $data
	`, map[string]any{
		"id":   taskID,
		"data": updateData,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("task_id", id).Msg("UPDATE query failed for task")
		return nil, err
	}

	r.logger.Info().Str("task_id", id).Msg("task updated via UPDATE query")

	// Sync emotion edges for analytics (async-safe, errors logged not returned)
	if req.Positives != nil || req.Negatives != nil || emotionChanged {
		r.syncEmotionEdges(ctx, id, finalEmotionID, finalPositives, finalNegatives)
	}

	// Fetch and return updated task with category hydrated
	return r.FindByID(ctx, id, userID)
}

// =============================================================================
// DELETE OPERATION
// =============================================================================

// Delete soft-deletes a task using query-based UPDATE.
//
// Uses UPDATE query for reliable single-record soft delete with models.RecordID.
func (r *repository) Delete(ctx context.Context, id, userID string) error {
	// Verify ownership first
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return err
	}

	taskID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	// Use UPDATE query for reliable soft delete
	_, err = database.QueryAll[taskDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE {
			deleted_at: $now,
			updated_by: $user,
			updated_at: $now
		}
	`, map[string]any{
		"id":   taskID,
		"now":  now,
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("task_id", id).Msg("UPDATE query failed for soft delete")
		return err
	}

	r.logger.Info().Str("task_id", id).Msg("task soft-deleted via UPDATE query")
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// validateCategoryOwnership checks if a category exists and belongs to the user.
//
// Uses models.RecordID for type-safe category reference.
func (r *repository) validateCategoryOwnership(ctx context.Context, categoryID models.RecordID, userID string) (bool, error) {
	// Use SDK's typed query for validation
	cats, err := database.QueryAll[categoryDB](ctx, r.db, `
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
			Str("category_id", database.ToStringID(categoryID)).
			Str("user_id", userID).
			Msg("category validation failed")
		return false, nil
	}

	return true, nil
}

// formatTaskID ensures the ID has the table prefix (string version).
func formatTaskID(id string) string {
	return database.RecordID(Table, id)
}

// generateTaskRecordID generates a unique task ID as models.RecordID.
func generateTaskRecordID() models.RecordID {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return database.NewRecordID(Table, hex.EncodeToString(bytes))
}

// sanitizeCategory removes deleted categories from task.
func sanitizeCategory(task *Task) {
	if task.Category != nil && task.Category.DeletedAt != nil {
		task.Category = nil
	}
}

// emotionItemsResult holds converted emotion items for inference calculation.
type emotionItemsResult struct {
	positives []emotions.TaskItem
	negatives []emotions.TaskItem
}

// toEmotionItems converts TaskItem slices to emotions.TaskItem slices for inference.
func toEmotionItems(positives, negatives []TaskItem) emotionItemsResult {
	result := emotionItemsResult{
		positives: make([]emotions.TaskItem, len(positives)),
		negatives: make([]emotions.TaskItem, len(negatives)),
	}

	for i, item := range positives {
		result.positives[i] = emotions.TaskItem{
			Text:      item.Text,
			EmotionID: item.EmotionID,
		}
	}

	for i, item := range negatives {
		result.negatives[i] = emotions.TaskItem{
			Text:      item.Text,
			EmotionID: item.EmotionID,
		}
	}

	return result
}

// =============================================================================
// EMOTION EDGE SYNC
// =============================================================================

// syncEmotionEdges creates graph edges linking task to emotions for analytics.
// This enables efficient queries like "all tasks where user felt E16" and
// emotion frequency aggregations.
//
// Edge types:
//   - "primary": The main emotion selected for the task
//   - "positive": Emotions from positive items
//   - "negative": Emotions from negative items
func (r *repository) syncEmotionEdges(ctx context.Context, taskID string, emotionID *string, positives, negatives []TaskItem) {
	// Delete existing edges for this task
	_, err := database.QueryAll[any](ctx, r.db, `
		DELETE task_emotions WHERE in = type::thing($task_id)
	`, map[string]any{
		"task_id": database.MustRecordID(Table, taskID),
	})
	if err != nil {
		r.logger.Warn().Err(err).Str("task_id", taskID).Msg("failed to delete old emotion edges")
	}

	// Create edge for primary emotion
	if emotionID != nil && *emotionID != "" {
		r.createEmotionEdge(ctx, taskID, *emotionID, "primary", nil)
	}

	// Create edges for positive items
	for _, item := range positives {
		if item.EmotionID != nil && *item.EmotionID != "" {
			r.createEmotionEdge(ctx, taskID, *item.EmotionID, "positive", &item.Text)
		}
	}

	// Create edges for negative items
	for _, item := range negatives {
		if item.EmotionID != nil && *item.EmotionID != "" {
			r.createEmotionEdge(ctx, taskID, *item.EmotionID, "negative", &item.Text)
		}
	}
}

// createEmotionEdge creates a single task -> emotion edge.
func (r *repository) createEmotionEdge(ctx context.Context, taskID, emotionID, edgeType string, text *string) {
	data := map[string]any{
		"type": edgeType,
	}
	if text != nil {
		data["text"] = *text
	}

	// RELATE requires proper record IDs: tasks:xxx -> task_emotions -> emotions:E16
	_, err := database.QueryAll[any](ctx, r.db, `
		RELATE $task_id->task_emotions->$emotion_id CONTENT $data
	`, map[string]any{
		"task_id":    database.MustRecordID(Table, taskID),
		"emotion_id": database.NewRecordID("emotions", emotionID), // emotions:E16
		"data":       data,
	})
	if err != nil {
		r.logger.Warn().
			Err(err).
			Str("task_id", taskID).
			Str("emotion_id", emotionID).
			Str("type", edgeType).
			Msg("failed to create emotion edge")
	}
}
