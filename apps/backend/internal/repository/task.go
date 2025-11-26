package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/daily-journal/backend/internal/model"
	"github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
)

const taskProjection = "id, title, journal, start_date, end_date, is_completed, priority, planned, source, note, positives, negatives, created_at, updated_at, deleted_at, created_by, updated_by"

type TaskRepository struct {
	db *DB
}

func NewTaskRepository(db *DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(task *model.Task) (*model.Task, error) {
	ctx := r.db.Context()
	logger := log.Ctx(ctx)

	task.CalculatePlannedStatus()
	task.SetDefaults()

	sql := fmt.Sprintf("CREATE tasks CONTENT $task RETURN %s", taskProjection)
	vars := map[string]any{"task": task}

	logger.Debug().Str("title", task.Title).Msg("creating task in database")

	created, err := r.querySingle(ctx, sql, vars)
	if err != nil {
		logger.Error().Err(err).Msg("database create failed")
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return created, nil
}

func (r *TaskRepository) List(userID string) ([]model.Task, error) {
	return r.ListByUser(userID)
}

// Fixed List method with correct generic type
func (r *TaskRepository) ListByUser(userID string) ([]model.Task, error) {
	ctx := r.db.Context()
	sql := fmt.Sprintf("SELECT %s FROM tasks WHERE created_by = $user AND deleted_at = NONE ORDER BY start_date DESC", taskProjection)
	vars := map[string]any{"user": userID}

	return r.queryList(ctx, sql, vars)
}

func (r *TaskRepository) ListByUserPaginated(userID string, limit, offset int) ([]model.Task, error) {
	ctx := r.db.Context()
	sql := fmt.Sprintf("SELECT %s FROM tasks WHERE created_by = $user AND deleted_at = NONE ORDER BY start_date DESC LIMIT $limit START $offset", taskProjection)
	vars := map[string]any{
		"user":   userID,
		"limit":  limit,
		"offset": offset,
	}

	return r.queryList(ctx, sql, vars)
}

func (r *TaskRepository) Get(id string) (*model.Task, error) {
	ctx := r.db.Context()
	recordID := ensureRecordID(id)
	sql := fmt.Sprintf("SELECT %s FROM tasks WHERE id = $record AND deleted_at = NONE", taskProjection)
	vars := map[string]any{"record": recordID}

	task, err := r.querySingle(ctx, sql, vars)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (r *TaskRepository) Update(id string, task *model.Task) (*model.Task, error) {
	ctx := r.db.Context()
	task.CalculatePlannedStatus()

	recordID := ensureRecordID(id)
	sql := fmt.Sprintf("UPDATE $record CONTENT $task RETURN %s", taskProjection)
	vars := map[string]any{
		"record": recordID,
		"task":   task,
	}

	updated, err := r.querySingle(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *TaskRepository) Delete(id string, userID string) error {
	ctx := r.db.Context()
	recordID := ensureRecordID(id)
	payload := map[string]any{
		"deleted_at": time.Now().UTC(),
	}
	if userID != "" {
		payload["updated_by"] = userID
	}

	_, err := surrealdb.Merge[model.Task](ctx, r.db.Client, recordID, payload)
	return err
}

func ensureRecordID(id string) string {
	if strings.Contains(id, ":") {
		return id
	}
	return fmt.Sprintf("tasks:%s", id)
}

func (r *TaskRepository) queryList(ctx context.Context, sql string, vars map[string]any) ([]model.Task, error) {
	results, err := surrealdb.Query[[]model.Task](ctx, r.db.Client, sql, vars)
	if err != nil {
		return nil, err
	}

	if len(*results) == 0 {
		return []model.Task{}, nil
	}

	first := (*results)[0]
	if first.Status != "OK" {
		if first.Error != nil {
			return nil, fmt.Errorf(first.Error.Message)
		}
		return nil, fmt.Errorf("query failed with status %s", first.Status)
	}

	return first.Result, nil
}

func (r *TaskRepository) querySingle(ctx context.Context, sql string, vars map[string]any) (*model.Task, error) {
	tasks, err := r.queryList(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("task not found")
	}
	return &tasks[0], nil
}
