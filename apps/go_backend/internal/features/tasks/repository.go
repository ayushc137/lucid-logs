// Package tasks provides task management functionality using libSQL/SQLite.
//
// This package implements:
//   - CRUD operations for tasks using typed SDK methods
//   - Category linking via category_id foreign key
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
//   - taskDB uses models.RecordID for ID and category_id reference fields
//   - Conversion to string happens in toTask() at the repository boundary
//   - This preserves the table:value API contract used across the codebase
//
// Emotion Tracking:
//   - InferredEmotion is calculated on CREATE/UPDATE, not on GET
//   - Uses weighted centroid of all emotion tags in positives/negatives
//   - Stored in database for efficient querying
package tasks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	models "github.com/lucid-logs/go-backend/internal/shared/recordid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/features/categories"
	"github.com/lucid-logs/go-backend/internal/features/emotions"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/lucid-logs/go-backend/internal/shared/timeutil"
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

	// FindFiltered retrieves tasks with filters, search, and pagination.
	// Supports full-text search on title, journal, and note fields.
	FindFiltered(ctx context.Context, userID string, filters TaskFilterParams, params pagination.Params) ([]*Task, int64, error)

	// Create creates a new task.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Task, error)

	// Update updates an existing task.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Task, error)

	// Delete soft-deletes a task.
	Delete(ctx context.Context, id, userID string) error

	// GetLastTaskEndTime retrieves the end time of the most recently finished task.
	GetLastTaskEndTime(ctx context.Context, userID string) (*time.Time, error)

	// FindGoalsForTask retrieves all goals linked to a task.
	FindGoalsForTask(ctx context.Context, taskID, userID string) ([]TaskGoalLink, error)
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
// This struct uses models.RecordID for the ID field so the JSON layer can
// round-trip the table:value API identifier. The Category field is hydrated
// via a LEFT JOIN against the categories table.
type taskDB struct {
	ID        models.RecordID `json:"id,omitempty"`
	Title     string          `json:"title"`
	Journal   string          `json:"journal"`
	StartDate time.Time       `json:"start_date"`
	EndDate   time.Time       `json:"end_date"`
	Completed bool            `json:"completed"`
	Priority  string          `json:"priority"`
	Source    string          `json:"source"`
	Note      string          `json:"note"`
	Positives []TaskItem      `json:"positives"`
	Negatives []TaskItem      `json:"negatives"`
	Category  *categoryDB     `json:"category,omitempty"` // Hydrated via LEFT JOIN

	// Emotion tracking
	EmotionID       *string                   `json:"emotion_id,omitempty"`
	InferredEmotion *emotions.InferredEmotion `json:"inferred_emotion,omitempty"`

	// Quantity (denormalized from quantity_value/unit_id columns)
	Quantity *Quantity `json:"quantity,omitempty"`

	// Linked goals (populated via subquery)
	LinkedGoals []linkedGoalDB `json:"linked_goals,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedBy string     `json:"created_by"`
}

// linkedGoalDB is the database representation of a linked goal from subquery.
type linkedGoalDB struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Icon          string   `json:"icon"`
	Color         string   `json:"color"`
	ImpactType    string   `json:"impact_type"`
	QuantityValue *float64 `json:"quantity_value"`
	UnitSymbol    string   `json:"unit_symbol"`
}

// categoryDB is the database representation of a category when fetched via JOIN.
//
// This struct uses models.RecordID for the ID field for type-safe
// interactions. Convert to categories.Category via toCategory().
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

// toCategory converts categoryDB to the domain model.
func (c *categoryDB) toCategory() *categories.Category {
	if c == nil {
		return nil
	}
	return &categories.Category{
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

// toTask converts the database model to the domain model.
//
// This is the boundary conversion point where models.RecordID is
// converted to string for API responses.
func (t *taskDB) toTask() *Task {
	var cat *categories.Category
	if t.Category != nil {
		cat = t.Category.toCategory()
	}

	// Convert linked goals
	var linkedGoals []LinkedGoalSummary
	if len(t.LinkedGoals) > 0 {
		linkedGoals = make([]LinkedGoalSummary, len(t.LinkedGoals))
		for i, lg := range t.LinkedGoals {
			linkedGoals[i] = LinkedGoalSummary{
				ID:            lg.ID,
				Title:         lg.Title,
				Icon:          lg.Icon,
				Color:         lg.Color,
				ImpactType:    lg.ImpactType,
				QuantityValue: lg.QuantityValue,
				UnitSymbol:    lg.UnitSymbol,
			}
		}
	}

	return &Task{
		ID:              database.ToStringID(t.ID),
		Title:           t.Title,
		Journal:         t.Journal,
		StartDate:       t.StartDate,
		EndDate:         t.EndDate,
		Completed:       t.Completed,
		Source:          t.Source,
		Note:            t.Note,
		Positives:       t.Positives,
		Negatives:       t.Negatives,
		Category:        cat,
		EmotionID:       t.EmotionID,
		InferredEmotion: t.InferredEmotion,
		Quantity:        t.Quantity,
		LinkedGoals:     linkedGoals,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
		DeletedAt:       t.DeletedAt,
		CreatedBy:       t.CreatedBy,
	}
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

// FindByID retrieves a task by ID for a specific user using SDK methods.
//
// The query LEFT JOINs the categories table so the hydrated Category is
// available on the returned task.
func (r *repository) FindByID(ctx context.Context, id, userID string) (*Task, error) {
	taskID := database.MustRecordID(Table, id)

	// Fetch the task with category hydrated via LEFT JOIN.
	task, err := database.QueryFirst[taskDB](ctx, r.db, `
		SELECT
			t.id AS id,
			t.title AS title,
			t.journal AS journal,
			t.start_date AS start_date,
			t.end_date AS end_date,
			t.completed AS completed,
			COALESCE(t.priority, '') AS priority,
			COALESCE(t.source, '') AS source,
			COALESCE(t.note, '') AS note,
			t.positives AS positives,
			t.negatives AS negatives,
			t.emotion_id AS emotion_id,
			t.inferred_emotion AS inferred_emotion,
			t.quantity_value AS quantity_value,
			t.unit_id AS unit_id,
			t.created_at AS created_at,
			t.updated_at AS updated_at,
			t.deleted_at AS deleted_at,
			t.created_by AS created_by,
			(
				SELECT json_object(
					'id', c.id,
					'name', c.name,
					'color', c.color,
					'created_at', c.created_at,
					'updated_at', c.updated_at,
					'deleted_at', c.deleted_at,
					'created_by', c.created_by,
					'updated_by', ''
				)
				FROM categories c
				WHERE c.id = t.category_id
				LIMIT 1
			) AS category
		FROM tasks t
		WHERE t.id = $id
	`, map[string]any{
		"id": database.ToStringID(taskID),
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

	// Hydrate linked goals for the task
	if err := r.hydrateLinkedGoals(ctx, result); err != nil {
		r.logger.Warn().Err(err).Str("task_id", id).Msg("failed to hydrate linked goals")
	}

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
	// Get total count using SQLite COUNT(*)
	total, err := database.QueryScalar[int64](ctx, r.db, `
		SELECT COUNT(*) FROM tasks
		WHERE created_by = $user AND deleted_at IS NULL
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("SDK QueryScalar failed for task count")
		return nil, 0, err
	}

	// Get paginated tasks using SDK's typed QueryAll
	tasksDB, err := database.QueryAll[taskDB](ctx, r.db, `
		SELECT
			t.id AS id,
			t.title AS title,
			t.journal AS journal,
			t.start_date AS start_date,
			t.end_date AS end_date,
			t.completed AS completed,
			COALESCE(t.priority, '') AS priority,
			COALESCE(t.source, '') AS source,
			COALESCE(t.note, '') AS note,
			t.positives AS positives,
			t.negatives AS negatives,
			t.emotion_id AS emotion_id,
			t.inferred_emotion AS inferred_emotion,
			t.quantity_value AS quantity_value,
			t.unit_id AS unit_id,
			t.created_at AS created_at,
			t.updated_at AS updated_at,
			t.deleted_at AS deleted_at,
			t.created_by AS created_by,
			(
				SELECT json_object(
					'id', c.id,
					'name', c.name,
					'color', c.color,
					'created_at', c.created_at,
					'updated_at', c.updated_at,
					'deleted_at', c.deleted_at,
					'created_by', c.created_by,
					'updated_by', ''
				)
				FROM categories c
				WHERE c.id = t.category_id
				LIMIT 1
			) AS category
		FROM tasks t
		WHERE t.created_by = $user AND t.deleted_at IS NULL
		ORDER BY t.start_date DESC
		LIMIT $limit OFFSET $offset
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

	return tasks, total, nil
}

// =============================================================================
// FIND FILTERED OPERATION (FTS + FILTERS)
// =============================================================================

// FindFiltered retrieves tasks with filters, search, and pagination.
//
// This method supports:
//   - Full-text search on title, journal, and note fields (case-insensitive LIKE)
//   - Category filtering
//   - Status filtering (completed/pending)
//   - Date range filtering
//   - Custom sorting
func (r *repository) FindFiltered(ctx context.Context, userID string, filters TaskFilterParams, params pagination.Params) ([]*Task, int64, error) {
	// Build dynamic WHERE conditions
	conditions := []string{"t.created_by = $user", "t.deleted_at IS NULL"}
	queryVars := map[string]any{
		"user":   userID,
		"limit":  params.Limit,
		"offset": params.Offset,
	}

	// Full-text search across title, journal, note using case-insensitive LIKE.
	hasSearch := filters.Search != ""
	if hasSearch {
		conditions = append(conditions, "(LOWER(t.title) LIKE LOWER('%' || $search || '%') OR LOWER(t.journal) LIKE LOWER('%' || $search || '%') OR LOWER(t.note) LIKE LOWER('%' || $search || '%'))")
		queryVars["search"] = filters.Search
	}

	// Category filter - NoCategoryFilter takes precedence
	if filters.NoCategoryFilter {
		// Filter for tasks without any category
		conditions = append(conditions, "t.category_id IS NULL")
	} else if filters.CategoryID != "" {
		catID := database.MustRecordID(categories.Table, filters.CategoryID)
		conditions = append(conditions, "t.category_id = $category")
		queryVars["category"] = database.ToStringID(catID)
	}

	// Status filter
	switch filters.Status {
	case StatusCompleted:
		conditions = append(conditions, "t.completed = 1")
	case StatusPending:
		conditions = append(conditions, "t.completed = 0")
		// StatusAll or empty: no filter
	}

	// Goal ID filter (via task_goals join table)
	if filters.GoalID != "" {
		goalID := database.MustRecordID("goals", filters.GoalID)
		conditions = append(conditions, "t.id IN (SELECT task_id FROM task_goals WHERE goal_id = $goal_id)")
		queryVars["goal_id"] = database.ToStringID(goalID)
	}

	// Activity ID filter (via created_from_activity join table)
	if filters.ActivityID != "" {
		activityID := database.MustRecordID("activities", filters.ActivityID)
		conditions = append(conditions, "t.id IN (SELECT task_id FROM created_from_activity WHERE activity_id = $activity_id)")
		queryVars["activity_id"] = database.ToStringID(activityID)
	}

	// Has quantity filter
	if filters.HasQuantity != nil {
		if *filters.HasQuantity {
			conditions = append(conditions, "t.quantity_value IS NOT NULL")
		} else {
			conditions = append(conditions, "t.quantity_value IS NULL")
		}
	}

	// Date range filter - parse strings to time.Time for proper datetime comparison
	if filters.StartDateFrom != "" {
		if parsedTime, err := time.Parse(time.RFC3339, filters.StartDateFrom); err == nil {
			conditions = append(conditions, "t.start_date >= $start_from")
			queryVars["start_from"] = parsedTime.UTC().Format(time.RFC3339Nano)
		}
	}
	if filters.StartDateTo != "" {
		if parsedTime, err := time.Parse(time.RFC3339, filters.StartDateTo); err == nil {
			conditions = append(conditions, "t.start_date <= $start_to")
			queryVars["start_to"] = parsedTime.UTC().Format(time.RFC3339Nano)
		}
	}

	// Build WHERE clause
	whereClause := strings.Join(conditions, " AND ")

	// Build ORDER BY clause
	orderClause := buildOrderClause(filters.SortField, filters.SortOrder, hasSearch)

	// Count query
	countQuery := `SELECT COUNT(*) FROM tasks t WHERE ` + whereClause

	total, err := database.QueryScalar[int64](ctx, r.db, countQuery, queryVars)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("FindFiltered count query failed")
		return nil, 0, err
	}

	// Main query with category fetch and linked goals subquery.
	// The linked_goals subquery fetches goal summary data for highlighting:
	// - Joins task_goals table with goals table
	// - Gets goal's category color via LEFT JOIN to categories
	// - Includes quantity_value for displaying contribution in popover
	selectQuery := `
		SELECT
			t.id AS id,
			t.title AS title,
			t.journal AS journal,
			t.start_date AS start_date,
			t.end_date AS end_date,
			t.completed AS completed,
			COALESCE(t.priority, '') AS priority,
			COALESCE(t.source, '') AS source,
			COALESCE(t.note, '') AS note,
			t.positives AS positives,
			t.negatives AS negatives,
			t.emotion_id AS emotion_id,
			t.inferred_emotion AS inferred_emotion,
			t.quantity_value AS quantity_value,
			t.unit_id AS unit_id,
			t.created_at AS created_at,
			t.updated_at AS updated_at,
			t.deleted_at AS deleted_at,
			t.created_by AS created_by,
			(
				SELECT json_object(
					'id', c.id,
					'name', c.name,
					'color', c.color,
					'created_at', c.created_at,
					'updated_at', c.updated_at,
					'deleted_at', c.deleted_at,
					'created_by', c.created_by,
					'updated_by', ''
				)
				FROM categories c
				WHERE c.id = t.category_id
				LIMIT 1
			) AS category,
			(
				SELECT json_group_array(json_object(
					'id', g.id,
					'title', g.title,
					'icon', COALESCE(g.icon, ''),
					'color', COALESCE(gc.color, ''),
					'impact_type', tg.impact_type,
					'quantity_value', tg.quantity_value,
					'unit_symbol', COALESCE(u.symbol, '')
				))
				FROM task_goals tg
				JOIN goals g ON g.id = tg.goal_id
				LEFT JOIN categories gc ON gc.id = g.category_id
				LEFT JOIN units u ON u.id = tg.unit_id
				WHERE tg.task_id = t.id
			) AS linked_goals
		FROM tasks t
		WHERE ` + whereClause + `
		` + orderClause + `
		LIMIT $limit OFFSET $offset
	`

	r.logger.Debug().
		Str("user_id", userID).
		Str("search", filters.Search).
		Str("category", filters.CategoryID).
		Str("status", filters.Status).
		Str("query", selectQuery).
		Msg("executing FindFiltered query")

	tasksDB, err := database.QueryAll[taskDB](ctx, r.db, selectQuery, queryVars)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("FindFiltered query failed")
		return nil, 0, err
	}

	// Convert to domain models
	tasks := make([]*Task, len(tasksDB))
	for i := range tasksDB {
		task := tasksDB[i].toTask()
		sanitizeCategory(task)
		tasks[i] = task
	}

	return tasks, total, nil
}

// buildOrderClause constructs the ORDER BY clause based on sort parameters.
func buildOrderClause(sortField, sortOrder string, hasSearch bool) string {
	// Default sort direction based on field type
	if sortOrder == "" {
		switch sortField {
		case SortByTitle:
			sortOrder = SortAsc
		default:
			sortOrder = SortDesc
		}
	}

	direction := "DESC"
	if sortOrder == SortAsc {
		direction = "ASC"
	}

	// Determine sort field
	field := "t.start_date" // default
	switch sortField {
	case SortByTitle:
		field = "t.title"
	case SortByCreatedAt:
		field = "t.created_at"
	case SortByStartDate:
		field = "t.start_date"
	}

	return "ORDER BY " + field + " " + direction
}

// =============================================================================
// CREATE OPERATION
// =============================================================================

// Create creates a new task.
//
// Inserts into tasks table and syncs emotion/goal edges via join tables.
// InferredEmotion is calculated at write time from positives/negatives emotions.
func (r *repository) Create(ctx context.Context, req *CreateRequest, userID string) (*Task, error) {
	// Validate category ownership if provided
	var categoryID *string
	if req.CategoryID != "" {
		catID := database.MustRecordID(categories.Table, req.CategoryID)
		exists, err := r.validateCategoryOwnership(ctx, catID, userID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.ErrCategoryNotFound
		}
		idStr := database.ToStringID(catID)
		categoryID = &idStr
	}

	// Parse dates (validated in service)
	startDate, _ := timeutil.ParseDateTime(req.StartDate) //nolint:errcheck // validated in service
	endDate, _ := timeutil.ParseDateTime(req.EndDate)     //nolint:errcheck // validated in service

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
	}

	// Generate task ID using models.RecordID
	taskID := generateTaskRecordID()
	taskIDStr := database.ToStringID(taskID)
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)

	// Marshal JSON-encoded columns
	positivesJSON, _ := json.Marshal(positives)
	negativesJSON, _ := json.Marshal(negatives)

	// Build the data map for the INSERT (only non-optional / always-present fields)
	data := map[string]any{
		"id":         taskIDStr,
		"created_by": userID,
		"title":      req.Title,
		"note":       req.Note,
		"journal":    req.Journal,
		"start_date": startDate.UTC().Format(time.RFC3339Nano),
		"end_date":   endDate.UTC().Format(time.RFC3339Nano),
		"completed":  false,
		"source":     source,
		"positives":  string(positivesJSON),
		"negatives":  string(negativesJSON),
		"metadata":   "{}",
		"created_at": nowStr,
		"updated_at": nowStr,
	}

	if categoryID != nil {
		data["category_id"] = *categoryID
	}

	// Emotion ID
	if req.EmotionID != nil && strings.TrimSpace(*req.EmotionID) != "" {
		data["emotion_id"] = strings.TrimSpace(*req.EmotionID)
	}

	// Inferred emotion stored as JSON text
	if inferredEmotion != nil {
		ieJSON, err := json.Marshal(inferredEmotion)
		if err != nil {
			r.logger.Warn().Err(err).Msg("failed to marshal inferred emotion")
		} else {
			data["inferred_emotion"] = string(ieJSON)
		}
	}

	// Activity link
	if req.ActivityID != "" {
		activityID := database.MustRecordID("activities", req.ActivityID)
		data["activity_id"] = database.ToStringID(activityID)
	}

	// Quantity
	if req.Quantity != nil {
		data["quantity_value"] = req.Quantity.Value
		if req.Quantity.UnitID != "" {
			data["unit_id"] = req.Quantity.UnitID
		}
	}

	// Use the SDK's typed Create to INSERT and return the new row
	type createResult struct {
		ID models.RecordID `json:"id"`
	}
	_, err := database.Create[createResult](ctx, r.db, Table, data)
	if err != nil {
		r.logger.Error().Err(err).
			Str("task_id", taskIDStr).
			Str("user_id", userID).
			Msg("SDK Create query failed for task")
		return nil, err
	}

	r.logger.Info().Str("task_id", taskIDStr).Msg("task created via SDK")

	// Sync emotion edges for analytics (async-safe, errors logged not returned)
	r.syncEmotionEdges(ctx, taskIDStr, req.EmotionID, positives, negatives)

	// Sync goal edges (async-safe) - pass startDate for goal_daily_stats updates
	if len(req.GoalLinks) > 0 {
		r.syncGoalEdges(ctx, taskIDStr, req.GoalLinks, startDate)
	}

	// Fetch and return the created task (with category hydrated)
	return r.FindByID(ctx, taskIDStr, userID)
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

// Update updates an existing task.
//
// Recalculates InferredEmotion when positives/negatives or emotion_id changes.
func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Task, error) {
	// Verify task exists and user has ownership
	existing, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	taskID := database.MustRecordID(Table, id)
	taskIDStr := database.ToStringID(taskID)
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)

	// Build update data dynamically with only provided fields
	updateData := map[string]any{
		"updated_at": nowStr,
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
		t, parseErr := timeutil.ParseDateTime(startStr)
		if parseErr != nil {
			return nil, errors.ErrBadRequest.WithMessage("Invalid start_date format")
		}
		newStart = &t
		updateData["start_date"] = t.UTC().Format(time.RFC3339Nano)
	}
	if req.EndDate != nil {
		endStr := strings.TrimSpace(*req.EndDate)
		if endStr == "" {
			return nil, errors.ErrBadRequest.WithMessage("end_date cannot be empty")
		}
		t, parseErr := timeutil.ParseDateTime(endStr)
		if parseErr != nil {
			return nil, errors.ErrBadRequest.WithMessage("Invalid end_date format")
		}
		newEnd = &t
		updateData["end_date"] = t.UTC().Format(time.RFC3339Nano)
	}
	if req.Completed != nil {
		updateData["completed"] = *req.Completed
	}
	if req.Note != nil {
		updateData["note"] = *req.Note
	}
	if req.Positives != nil {
		positivesJSON, _ := json.Marshal(req.Positives)
		updateData["positives"] = string(positivesJSON)
	}
	if req.Negatives != nil {
		negativesJSON, _ := json.Marshal(req.Negatives)
		updateData["negatives"] = string(negativesJSON)
	}
	if req.Quantity != nil {
		updateData["quantity_value"] = req.Quantity.Value
		if req.Quantity.UnitID != "" {
			updateData["unit_id"] = req.Quantity.UnitID
		}
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
		emotionItems := toEmotionItems(finalPositives, finalNegatives)
		if len(emotionItems.positives) > 0 || len(emotionItems.negatives) > 0 {
			ie := emotions.InferFromItems(emotionItems.positives, emotionItems.negatives)
			ieJSON, err := json.Marshal(ie)
			if err != nil {
				r.logger.Warn().Err(err).Msg("failed to marshal inferred emotion")
			} else {
				updateData["inferred_emotion"] = string(ieJSON)
			}
		} else {
			// Explicitly remove inferred emotion if no items
			updateData["inferred_emotion"] = nil
		}
	}

	// Use SDK's Merge helper for the UPDATE
	_, err = database.Merge[taskDB](ctx, r.db, taskIDStr, updateData)
	if err != nil {
		r.logger.Error().Err(err).Str("task_id", id).Msg("UPDATE query failed for task")
		return nil, err
	}

	r.logger.Info().Str("task_id", id).Msg("task updated via UPDATE query")

	// Sync emotion edges for analytics (async-safe, errors logged not returned)
	if req.Positives != nil || req.Negatives != nil || emotionChanged {
		r.syncEmotionEdges(ctx, id, finalEmotionID, finalPositives, finalNegatives)
	}

	// Sync goal edges if provided
	// Note: Client must send ALL current links as this replaces them
	// If GoalLinks is nil, we don't touch existing links (partial update)
	// To clear links, send empty array
	if req.GoalLinks != nil {
		// Use existing start date for stats - for updates, need to get from existing task
		r.syncGoalEdges(ctx, id, *req.GoalLinks, existing.StartDate)
	}

	// Fetch and return updated task with category hydrated
	return r.FindByID(ctx, id, userID)
}

// =============================================================================
// DELETE OPERATION
// =============================================================================

// Delete soft-deletes a task using the Merge helper.
func (r *repository) Delete(ctx context.Context, id, userID string) error {
	// Verify ownership first
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return err
	}

	taskID := database.MustRecordID(Table, id)
	taskIDStr := database.ToStringID(taskID)
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)

	// Use Merge for reliable soft delete
	_, err = database.Merge[taskDB](ctx, r.db, taskIDStr, map[string]any{
		"deleted_at": nowStr,
		"updated_at": nowStr,
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
func (r *repository) validateCategoryOwnership(ctx context.Context, categoryID models.RecordID, userID string) (bool, error) {
	// Use SDK's typed query for validation
	cats, err := database.QueryAll[categoryDB](ctx, r.db, `
		SELECT * FROM categories
		WHERE id = $id AND created_by = $user AND deleted_at IS NULL
	`, map[string]any{
		"id":   database.ToStringID(categoryID),
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

// GetLastTaskEndTime retrieves the end time of the most recently finished task.
func (r *repository) GetLastTaskEndTime(ctx context.Context, userID string) (*time.Time, error) {
	// Query to find the most recent task that has already finished
	query := `
		SELECT end_date FROM tasks
		WHERE created_by = $user
		  AND deleted_at IS NULL
		  AND end_date <= $now
		ORDER BY end_date DESC
		LIMIT 1
	`

	type result struct {
		EndDate time.Time `json:"end_date"`
	}

	// Use QueryFirst to get a single result
	record, err := database.QueryFirst[result](ctx, r.db, query, map[string]any{
		"user": userID,
		"now":  time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get last task end time")
		return nil, err
	}

	if record == nil {
		return nil, nil
	}

	return &record.EndDate, nil
}

func (r *repository) FindGoalsForTask(ctx context.Context, taskID, userID string) ([]TaskGoalLink, error) {
	tID := database.MustRecordID(Table, taskID)
	taskIDStr := database.ToStringID(tID)

	type goalCategoryDB struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}

	// Parse target JSON column for goal_target_value and goal_target_unit.
	// The goals.target column is TEXT holding JSON like:
	//   {"value": 10, "unit_id": "units:km"}
	type goalLinkDB struct {
		GoalID          string          `json:"goal_id"`
		GoalTitle       string          `json:"goal_title"`
		GoalIcon        string          `json:"goal_icon,omitempty"`
		ImpactType      string          `json:"impact_type"`
		QuantityValue   *float64        `json:"quantity_value,omitempty"`
		UnitID          *string         `json:"unit_id,omitempty"`
		IsMilestone     bool            `json:"is_milestone"`
		MilestoneLabel  string          `json:"milestone_label,omitempty"`
		GoalDescription string          `json:"goal_description,omitempty"`
		GoalStatus      string          `json:"goal_status,omitempty"`
		GoalPriority    int             `json:"goal_priority,omitempty"`
		GoalTargetValue *float64        `json:"goal_target_value,omitempty"`
		GoalTargetUnit  string          `json:"goal_target_unit,omitempty"`
		GoalCategory    *goalCategoryDB `json:"goal_category,omitempty"`
		LinkedAt        *time.Time      `json:"linked_at,omitempty"`
		Notes           string          `json:"notes,omitempty"`
	}

	goalsDB, err := database.QueryAll[goalLinkDB](ctx, r.db, `
		SELECT
			tg.goal_id AS goal_id,
			g.title AS goal_title,
			COALESCE(g.icon, '') AS goal_icon,
			tg.impact_type AS impact_type,
			tg.quantity_value AS quantity_value,
			tg.unit_id AS unit_id,
			CASE WHEN tg.is_milestone = 1 THEN 1 ELSE 0 END AS is_milestone,
			COALESCE(tg.milestone_label, '') AS milestone_label,
			COALESCE(g.description, '') AS goal_description,
			COALESCE(g.status, '') AS goal_status,
			COALESCE(g.priority, 0) AS goal_priority,
			json_extract(g.target, '$.value') AS goal_target_value,
			COALESCE(json_extract(g.target, '$.unit_id'), '') AS goal_target_unit,
			(
				SELECT json_object(
					'id', c.id,
					'name', c.name,
					'color', c.color
				)
				FROM categories c
				WHERE c.id = g.category_id
				LIMIT 1
			) AS goal_category,
			tg.created_at AS linked_at,
			COALESCE(tg.notes, '') AS notes
		FROM task_goals tg
		JOIN goals g ON g.id = tg.goal_id
		WHERE tg.task_id = $task_id AND g.deleted_at IS NULL
	`, map[string]any{
		"task_id": taskIDStr,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("task_id", taskID).Msg("find goals for task failed")
		return nil, err
	}

	goals := make([]TaskGoalLink, len(goalsDB))
	for i, g := range goalsDB {
		link := TaskGoalLink{
			GoalID:          g.GoalID,
			GoalTitle:       g.GoalTitle,
			GoalIcon:        g.GoalIcon,
			ImpactType:      g.ImpactType,
			QuantityValue:   g.QuantityValue,
			UnitID:          g.UnitID,
			IsMilestone:     g.IsMilestone,
			MilestoneLabel:  g.MilestoneLabel,
			GoalDescription: g.GoalDescription,
			GoalStatus:      g.GoalStatus,
			GoalPriority:    g.GoalPriority,
			GoalTargetValue: g.GoalTargetValue,
			GoalTargetUnit:  g.GoalTargetUnit,
			Notes:           g.Notes,
		}

		if g.LinkedAt != nil && !g.LinkedAt.IsZero() {
			linkedAt := *g.LinkedAt
			link.LinkedAt = &linkedAt
		}
		if g.GoalCategory != nil && g.GoalCategory.ID != "" {
			link.GoalCategory = &struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
			}{
				ID:    g.GoalCategory.ID,
				Name:  g.GoalCategory.Name,
				Color: g.GoalCategory.Color,
			}
		}

		goals[i] = link
	}

	return goals, nil
}

// hydrateLinkedGoals populates the LinkedGoals field on a Task by querying the
// task_goals join table. Used by FindByID where the SELECT does not include
// the linked_goals subquery.
func (r *repository) hydrateLinkedGoals(ctx context.Context, task *Task) error {
	if task == nil {
		return nil
	}
	linked, err := database.QueryAll[linkedGoalDB](ctx, r.db, `
		SELECT
			g.id AS id,
			g.title AS title,
			COALESCE(g.icon, '') AS icon,
			COALESCE(gc.color, '') AS color,
			tg.impact_type AS impact_type,
			tg.quantity_value AS quantity_value,
			COALESCE(u.symbol, '') AS unit_symbol
		FROM task_goals tg
		JOIN goals g ON g.id = tg.goal_id
		LEFT JOIN categories gc ON gc.id = g.category_id
		LEFT JOIN units u ON u.id = tg.unit_id
		WHERE tg.task_id = $task_id
	`, map[string]any{
		"task_id": task.ID,
	})
	if err != nil {
		return err
	}

	task.LinkedGoals = make([]LinkedGoalSummary, len(linked))
	for i, lg := range linked {
		task.LinkedGoals[i] = LinkedGoalSummary{
			ID:            lg.ID,
			Title:         lg.Title,
			Icon:          lg.Icon,
			Color:         lg.Color,
			ImpactType:    lg.ImpactType,
			QuantityValue: lg.QuantityValue,
			UnitSymbol:    lg.UnitSymbol,
		}
	}
	return nil
}

// generateTaskRecordID generates a unique task ID as models.RecordID.
func generateTaskRecordID() models.RecordID {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
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

// syncEmotionEdges inserts rows into the task_emotions join table linking
// a task to emotions for analytics.
// This enables efficient queries like "all tasks where user felt E16" and
// emotion frequency aggregations.
//
// Edge types:
//   - "primary": The main emotion selected for the task
//   - "positive": Emotions from positive items
//   - "negative": Emotions from negative items
//
// Optimized to delete existing rows then re-insert in a batch.
func (r *repository) syncEmotionEdges(ctx context.Context, taskID string, emotionID *string, positives, negatives []TaskItem) {
	taskRID := database.MustRecordID(Table, taskID)
	taskIDStr := database.ToStringID(taskRID)

	// First, delete existing edges
	_, err := database.QueryAll[any](ctx, r.db, `
		DELETE FROM task_emotions WHERE task_id = $task_id
	`, map[string]any{
		"task_id": taskIDStr,
	})
	if err != nil {
		r.logger.Warn().Err(err).Str("task_id", taskID).Msg("failed to delete old emotion edges")
	}

	// Prepare data for batch insert
	type edgeData struct {
		EmotionID string  `json:"emotion_id"`
		Type      string  `json:"type"`
		Text      *string `json:"text,omitempty"`
	}
	edges := make([]edgeData, 0)

	// Add primary emotion
	if emotionID != nil && *emotionID != "" {
		edges = append(edges, edgeData{EmotionID: *emotionID, Type: "primary"})
	}

	// Add positive emotions
	for _, item := range positives {
		if item.EmotionID != nil && *item.EmotionID != "" {
			t := item.Text
			edges = append(edges, edgeData{EmotionID: *item.EmotionID, Type: "positive", Text: &t})
		}
	}

	// Add negative emotions
	for _, item := range negatives {
		if item.EmotionID != nil && *item.EmotionID != "" {
			t := item.Text
			edges = append(edges, edgeData{EmotionID: *item.EmotionID, Type: "negative", Text: &t})
		}
	}

	if len(edges) == 0 {
		return
	}

	// Execute per-edge INSERT statements (SQLite has no FOR loop in SQL).
	for _, edge := range edges {
		edgeID := generateEdgeRecordID("task_emotions")
		data := map[string]any{
			"id":         edgeID,
			"task_id":    taskIDStr,
			"emotion_id": edge.EmotionID,
			"type":       edge.Type,
			"created_at": time.Now().UTC().Format(time.RFC3339Nano),
		}
		if edge.Text != nil {
			data["text"] = *edge.Text
		}
		if _, err := database.Create[any](ctx, r.db, "task_emotions", data); err != nil {
			r.logger.Warn().Err(err).
				Str("task_id", taskID).
				Str("emotion_id", edge.EmotionID).
				Str("type", edge.Type).
				Msg("failed to insert emotion edge")
		}
	}
}

// convertQuantityInput converts a QuantityInput to a Quantity.
func convertQuantityInput(input *QuantityInput) *Quantity {
	if input == nil {
		return nil
	}
	return &Quantity{
		Value:  input.Value,
		UnitID: input.UnitID,
	}
}

// =============================================================================
// GOAL EDGE SYNC
// =============================================================================

// syncGoalEdges creates rows in the task_goals join table linking task to goals.
// It also updates goal_daily_stats for progress tracking.
func (r *repository) syncGoalEdges(ctx context.Context, taskID string, links []GoalLinkInput, taskStartDate time.Time) {
	taskRID := database.MustRecordID(Table, taskID)
	taskIDStr := database.ToStringID(taskRID)

	// Delete existing edges for this task
	_, err := database.QueryAll[any](ctx, r.db, `
		DELETE FROM task_goals WHERE task_id = $task_id
	`, map[string]any{
		"task_id": taskIDStr,
	})
	if err != nil {
		r.logger.Warn().Err(err).Str("task_id", taskID).Msg("failed to delete old goal edges")
	}

	if len(links) == 0 {
		return
	}

	for _, link := range links {
		// Defaults
		impactType := link.ImpactType
		if impactType == "" {
			impactType = "neutral"
		}

		// Normalize goal ID to goals:value form if needed
		goalIDStr := link.GoalID
		if !strings.Contains(goalIDStr, ":") {
			goalIDStr = "goals:" + goalIDStr
		}

		edgeID := generateEdgeRecordID("task_goals")
		data := map[string]any{
			"id":              edgeID,
			"task_id":         taskIDStr,
			"goal_id":         goalIDStr,
			"impact_type":     impactType,
			"is_milestone":    link.IsMilestone,
			"milestone_label": link.MilestoneLabel,
			"milestone_order": link.MilestoneOrder,
			"notes":           link.Notes,
			"source":          "manual",
			"created_at":      time.Now().UTC().Format(time.RFC3339Nano),
		}
		if link.QuantityValue > 0 {
			data["quantity_value"] = link.QuantityValue
		}

		if _, err := database.Create[any](ctx, r.db, "task_goals", data); err != nil {
			r.logger.Warn().Err(err).
				Str("task_id", taskID).
				Str("goal_id", goalIDStr).
				Msg("failed to insert goal edge")
			continue
		}
	}

	// Update goal_daily_stats for each linked goal
	r.updateGoalDailyStats(ctx, links, taskStartDate)
}

// updateGoalDailyStats updates the goal_daily_stats table for progress tracking.
// This is called after task_goals rows are created to maintain denormalized stats.
func (r *repository) updateGoalDailyStats(ctx context.Context, links []GoalLinkInput, taskStartDate time.Time) {
	// Truncate to start of day for consistent date grouping
	statDate := time.Date(taskStartDate.Year(), taskStartDate.Month(), taskStartDate.Day(), 0, 0, 0, 0, time.UTC)
	statDateStr := statDate.Format(time.RFC3339Nano)
	nowStr := time.Now().UTC().Format(time.RFC3339Nano)

	for _, link := range links {
		qty := link.QuantityValue
		if qty <= 0 {
			qty = 0 // Count contribution even without quantity
		}

		// Normalize goal ID
		goalIDStr := link.GoalID
		if !strings.Contains(goalIDStr, ":") {
			goalIDStr = "goals:" + goalIDStr
		}

		// Fetch the goal's created_by and target value for denormalized columns.
		type goalRow struct {
			CreatedBy  string  `json:"created_by"`
			TargetJSON *string `json:"target"`
		}
		goal, err := database.QueryFirst[goalRow](ctx, r.db, `
			SELECT created_by, target AS target FROM goals WHERE id = $goal_id
		`, map[string]any{
			"goal_id": goalIDStr,
		})
		if err != nil {
			r.logger.Warn().Err(err).
				Str("goal_id", link.GoalID).
				Msg("failed to fetch goal for daily stats update")
			continue
		}
		if goal == nil {
			continue
		}

		var targetValue *float64
		if goal.TargetJSON != nil && *goal.TargetJSON != "" {
			var target struct {
				Value  *float64 `json:"value"`
				UnitID string   `json:"unit_id"`
			}
			if json.Unmarshal([]byte(*goal.TargetJSON), &target) == nil {
				targetValue = target.Value
			}
		}

		// Upsert goal_daily_stats using INSERT ... ON CONFLICT.
		// The primary key (goal_id, date) determines uniqueness.
		data := map[string]any{
			"goal_id":            goalIDStr,
			"date":               statDate.Format("2006-01-02"),
			"created_by":         goal.CreatedBy,
			"daily_value":        qty,
			"cumulative_value":   0,
			"contribution_count": 1,
			"status":             "pending",
			"target_value":       targetValue,
			"created_at":         nowStr,
			"updated_at":         nowStr,
		}
		// Use a raw upsert via QueryAll since the SDK's Create helper would
		// conflict on the composite primary key. The upsert increments
		// daily_value and contribution_count for existing rows.
		if targetValue != nil {
			data["target_value"] = *targetValue
		}

		if _, err := database.QueryAll[any](ctx, r.db, `
			INSERT INTO goal_daily_stats
				(goal_id, date, created_by, daily_value, cumulative_value,
				 contribution_count, status, target_value, created_at, updated_at)
			VALUES
				($goal_id, $date, $created_by, $daily_value, $cumulative_value,
				 $contribution_count, $status, $target_value, $created_at, $updated_at)
			ON CONFLICT(goal_id, date) DO UPDATE SET
				daily_value = daily_value + excluded.daily_value,
				contribution_count = contribution_count + excluded.contribution_count,
				updated_at = excluded.updated_at
		`, map[string]any{
			"goal_id":            data["goal_id"],
			"date":               data["date"],
			"created_by":         data["created_by"],
			"daily_value":        data["daily_value"],
			"cumulative_value":   data["cumulative_value"],
			"contribution_count": data["contribution_count"],
			"status":             data["status"],
			"target_value":       targetValue,
			"created_at":         nowStr,
			"updated_at":         nowStr,
		}); err != nil {
			r.logger.Warn().Err(err).
				Str("goal_id", link.GoalID).
				Str("date", statDateStr).
				Float64("qty", qty).
				Msg("failed to update goal_daily_stats")
		}
	}
}

// generateEdgeRecordID generates a unique record ID for a join-table row.
// Uses random hex like the task ID generator.
func generateEdgeRecordID(table string) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return table + ":" + hex.EncodeToString(bytes)
}
