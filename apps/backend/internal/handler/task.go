package handler

import (
	"net/http"
	"strconv"

	"github.com/daily-journal/backend/internal/model"
	"github.com/daily-journal/backend/internal/repository"
	"github.com/daily-journal/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type TaskHandler struct {
	repo *repository.TaskRepository
}

func NewTaskHandler(repo *repository.TaskRepository) *TaskHandler {
	return &TaskHandler{repo: repo}
}

// CreateTask godoc
// @Summary      Create a new task
// @Description  Create a detailed task with validation
// @Tags         tasks
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        task  body      model.CreateTaskRequest  true  "Task content" example({"journal":"Capture high-level goals","end_date":"2025-11-25","negatives":["string"],"note":"string","positives":["string"],"priority":0,"source":"manual","start_date":"2025-11-24","title":"Plan tomorrow"})
// @Success      201   {object}  model.Task
// @Router       /tasks [post]
func (h *TaskHandler) Create(c *gin.Context) {
	logger := log.Ctx(c.Request.Context())

	var req model.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err)
		return
	}

	startDate := model.NormalizeDate(req.StartDate.TimeValue())
	endDate := model.NormalizeDate(req.EndDate.TimeValue())

	if !endDate.IsZero() && endDate.Before(startDate) {
		response.BadRequest(c, "end_date must be on or after start_date")
		return
	}

	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}

	task := &model.Task{
		Title:     req.Title,
		Journal:   req.Journal,
		StartDate: startDate,
		EndDate:   endDate,
		Priority:  req.Priority,
		Source:    req.Source,
		Notes:     req.Note,
		Positives: req.Positives,
		Negatives: req.Negatives,
	}
	if task.CreatedBy == "" {
		task.CreatedBy = userID
	}
	if task.UpdatedBy == "" {
		task.UpdatedBy = userID
	}

	logger.Debug().
		Str("user_id", userID).
		Str("title", task.Title).
		Msg("creating task")

	created, err := h.repo.Create(task)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create task")
		response.InternalError(c, err)
		return
	}

	logger.Info().
		Str("task_id", created.ID).
		Msg("task created successfully")

	response.Success(c, http.StatusCreated, created)
}

// ListTasks godoc
// @Summary      List all tasks
// @Tags         tasks
// @Security     BearerAuth
// @Produce      json
// @Param        limit   query     int  false  "Number of tasks to return (max 100)" default(25)
// @Param        offset  query     int  false  "Number of tasks to skip"             default(0)
// @Success      200     {array}   model.Task
// @Router       /tasks [get]
func (h *TaskHandler) List(c *gin.Context) {
	logger := log.Ctx(c.Request.Context())

	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}

	limit := parsePositiveInt(c.DefaultQuery("limit", "25"), 25, 100)
	offset := parsePositiveInt(c.DefaultQuery("offset", "0"), 0, 100000)

	logger.Debug().
		Str("user_id", userID).
		Int("limit", limit).
		Int("offset", offset).
		Msg("listing tasks")

	tasks, err := h.repo.ListByUserPaginated(userID, limit, offset)
	if err != nil {
		logger.Error().Err(err).Msg("failed to list tasks")
		response.InternalError(c, err)
		return
	}

	logger.Info().
		Int("count", len(tasks)).
		Msg("tasks listed successfully")

	response.Success(c, http.StatusOK, tasks)
}

func parsePositiveInt(value string, defaultValue, max int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return defaultValue
	}
	if n > max {
		return max
	}
	return n
}

// GetTask godoc
// @Summary      Get a task by ID
// @Tags         tasks
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Task ID"
// @Success      200  {object}  model.Task
// @Router       /tasks/{id} [get]
func (h *TaskHandler) Get(c *gin.Context) {
	logger := log.Ctx(c.Request.Context())
	id := c.Param("id")

	logger.Debug().Str("task_id", id).Msg("fetching task")

	task, err := h.repo.Get(id)
	if err != nil {
		logger.Warn().Err(err).Str("task_id", id).Msg("task not found")
		response.NotFound(c, "Task")
		return
	}

	response.Success(c, http.StatusOK, task)
}

// UpdateTask godoc
// @Summary      Update a task
// @Tags         tasks
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string                   true  "Task ID"
// @Param        task  body      model.UpdateTaskRequest  true  "Task update"
// @Success      200   {object}  model.Task
// @Router       /tasks/{id} [put]
func (h *TaskHandler) Update(c *gin.Context) {
	logger := log.Ctx(c.Request.Context())
	id := c.Param("id")

	var req model.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err)
		return
	}

	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}

	logger.Debug().Str("task_id", id).Str("user_id", userID).Msg("updating task")

	existing, err := h.repo.Get(id)
	if err != nil {
		logger.Warn().Err(err).Str("task_id", id).Msg("task not found for update")
		response.NotFound(c, "Task")
		return
	}
	existing.StartDate = model.NormalizeDate(existing.StartDate)
	existing.EndDate = model.NormalizeDate(existing.EndDate)

	// Update fields
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Journal != nil {
		existing.Journal = *req.Journal
	}
	if req.StartDate != nil {
		existing.StartDate = model.NormalizeDate(req.StartDate.TimeValue())
	}
	if req.EndDate != nil {
		existing.EndDate = model.NormalizeDate(req.EndDate.TimeValue())
		if existing.EndDate.Before(existing.StartDate) {
			response.BadRequest(c, "end_date must be on or after start_date")
			return
		}
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}
	if req.Note != nil {
		existing.Notes = *req.Note
	}
	if req.IsCompleted != nil {
		existing.IsCompleted = *req.IsCompleted
	}

	existing.UpdatedBy = userID

	updated, err := h.repo.Update(id, existing)
	if err != nil {
		logger.Error().Err(err).Str("task_id", id).Msg("failed to update task")
		response.InternalError(c, err)
		return
	}

	logger.Info().Str("task_id", id).Msg("task updated successfully")
	response.Success(c, http.StatusOK, updated)
}

// DeleteTask godoc
// @Summary      Delete a task
// @Tags         tasks
// @Security     BearerAuth
// @Param        id   path      string  true  "Task ID"
// @Success      204  {object}  nil
// @Router       /tasks/{id} [delete]
func (h *TaskHandler) Delete(c *gin.Context) {
	logger := log.Ctx(c.Request.Context())
	id := c.Param("id")

	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}

	logger.Debug().Str("task_id", id).Str("user_id", userID).Msg("deleting task")

	if err := h.repo.Delete(id, userID); err != nil {
		logger.Error().Err(err).Str("task_id", id).Msg("failed to delete task")
		response.InternalError(c, err)
		return
	}

	logger.Info().Str("task_id", id).Msg("task deleted successfully")
	response.NoContent(c)
}

func userIDFromContext(c *gin.Context) (string, bool) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "Authentication required")
		return "", false
	}
	return userID, true
}
