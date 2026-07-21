package apitest_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/lucid-logs/go-backend/internal/test/apitest"
)

// TestTasksCRUDCoversFullLifecycle exercises create → get → update → delete → verify gone.
func TestTasksCRUDCoversFullLifecycle(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("tasks@example.com", "password123")

	// CREATE
	createReq := map[string]any{
		"title":      "Morning standup",
		"start_date": time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339),
		"end_date":   time.Now().UTC().Format(time.RFC3339),
		"journal":    "Discussed blockers",
		"source":     "manual",
		"note":       "Follow up on API latency",
	}
	req := apitest.JSONRequest(t, "POST", "/api/v1/tasks", createReq, apitest.WithToken(token))
	rr := apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var createResp struct {
		Data struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			Journal   string `json:"journal"`
			Source    string `json:"source"`
			Note      string `json:"note"`
			Completed bool   `json:"completed"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &createResp)

	if createResp.Data.ID == "" {
		t.Fatal("create response missing ID")
	}
	if createResp.Data.Title != "Morning standup" {
		t.Errorf("title = %q, want %q", createResp.Data.Title, "Morning standup")
	}
	if createResp.Data.Completed {
		t.Error("new task should not be completed")
	}
	if createResp.Data.Source != "manual" {
		t.Errorf("source = %q, want %q", createResp.Data.Source, "manual")
	}

	taskID := createResp.Data.ID

	// GET
	req = apitest.JSONRequest(t, "GET", fmt.Sprintf("/api/v1/tasks/%s", taskID), nil, apitest.WithToken(token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var getResp struct {
		Data struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Journal string `json:"journal"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &getResp)
	if getResp.Data.ID != taskID {
		t.Errorf("GET id = %q, want %q", getResp.Data.ID, taskID)
	}

	// UPDATE
	updateReq := map[string]any{
		"title":     "Updated standup",
		"completed": true,
	}
	req = apitest.JSONRequest(t, "PUT", fmt.Sprintf("/api/v1/tasks/%s", taskID), updateReq, apitest.WithToken(token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var updateResp struct {
		Data struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			Completed bool   `json:"completed"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &updateResp)
	if updateResp.Data.Title != "Updated standup" {
		t.Errorf("updated title = %q, want %q", updateResp.Data.Title, "Updated standup")
	}
	if !updateResp.Data.Completed {
		t.Error("task should be completed after update")
	}

	// DELETE (soft delete)
	req = apitest.JSONRequest(t, "DELETE", fmt.Sprintf("/api/v1/tasks/%s", taskID), nil, apitest.WithToken(token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	// GET after delete should 404
	req = apitest.JSONRequest(t, "GET", fmt.Sprintf("/api/v1/tasks/%s", taskID), nil, apitest.WithToken(token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertError(t, rr, http.StatusNotFound, "NOT_FOUND")
}

// TestTasksDateFiltering verifies tasks are filtered by start_date correctly.
func TestTasksDateFiltering(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("tasks-date@example.com", "password123")

	// Create tasks on different days
	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	for _, tc := range []struct {
		date  time.Time
		title string
	}{
		{yesterday, "Yesterday task"},
		{today, "Today task"},
		{tomorrow, "Tomorrow task"},
	} {
		req := apitest.JSONRequest(t, "POST", "/api/v1/tasks", map[string]any{
			"title":      tc.title,
			"start_date": tc.date.Format(time.RFC3339),
			"end_date":   tc.date.Add(1 * time.Hour).Format(time.RFC3339),
		}, apitest.WithToken(token))
		rr := apitest.Do(app.Router, req)
		apitest.AssertStatus(t, rr, http.StatusOK)
	}

	// Filter by start_date = today
	req := apitest.JSONRequest(t, "GET", "/api/v1/tasks?start_date="+today.Format("2006-01-02"), nil, apitest.WithToken(token))
	rr := apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var listResp struct {
		Data []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"data"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	apitest.Decode(t, rr, &listResp)

	if listResp.Pagination.Total != 1 {
		t.Errorf("total = %d, want 1; tasks: %+v", listResp.Pagination.Total, listResp.Data)
	}
	if len(listResp.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(listResp.Data))
	}
	if listResp.Data[0].Title != "Today task" {
		t.Errorf("title = %q, want %q", listResp.Data[0].Title, "Today task")
	}
}

// TestTasksCategoryLinking verifies tasks can be linked to categories.
func TestTasksCategoryLinking(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("tasks-cat@example.com", "password123")

	// Create a category first
	req := apitest.JSONRequest(t, "POST", "/api/v1/categories", map[string]any{
		"name":  "Work",
		"color": "#FF5733",
	}, apitest.WithToken(token))
	rr := apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var catResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &catResp)
	categoryID := catResp.Data.ID

	// Create task with category
	req = apitest.JSONRequest(t, "POST", "/api/v1/tasks", map[string]any{
		"title":      "Categorized task",
		"start_date": time.Now().UTC().Format(time.RFC3339),
		"end_date":   time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
		"category_id": categoryID,
	}, apitest.WithToken(token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var taskResp struct {
		Data struct {
			ID       string `json:"id"`
			Category *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"category"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &taskResp)

	if taskResp.Data.Category == nil {
		t.Fatal("task should have category hydrated")
	}
	if taskResp.Data.Category.ID != categoryID {
		t.Errorf("category ID = %q, want %q", taskResp.Data.Category.ID, categoryID)
	}
	if taskResp.Data.Category.Name != "Work" {
		t.Errorf("category name = %q, want %q", taskResp.Data.Category.Name, "Work")
	}
}

// TestTasksGoalLinking verifies tasks can be linked to goals via taskgoals.
func TestTasksGoalLinking(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("tasks-goal@example.com", "password123")

	// Create a goal
	req := apitest.JSONRequest(t, "POST", "/api/v1/goals", map[string]any{
		"title": "Learn Go",
	}, apitest.WithToken(token))
	rr := apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var goalResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &goalResp)
	goalID := goalResp.Data.ID

	// Create a task
	req = apitest.JSONRequest(t, "POST", "/api/v1/tasks", map[string]any{
		"title":      "Read Go docs",
		"start_date": time.Now().UTC().Format(time.RFC3339),
		"end_date":   time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
	}, apitest.WithToken(token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var taskResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &taskResp)
	taskID := taskResp.Data.ID

	// Link task to goal
	req = apitest.JSONRequest(t, "POST", fmt.Sprintf("/api/v1/tasks/%s/goals", taskID), map[string]any{
		"goal_id":     goalID,
		"impact_type": "positive",
	}, apitest.WithToken(token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	// Verify link via GET /tasks/:id/goals
	req = apitest.JSONRequest(t, "GET", fmt.Sprintf("/api/v1/tasks/%s/goals", taskID), nil, apitest.WithToken(token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var linksResp struct {
		Data []struct {
			GoalID     string `json:"goal_id"`
			ImpactType string `json:"impact_type"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &linksResp)

	if len(linksResp.Data) != 1 {
		t.Fatalf("len(links) = %d, want 1", len(linksResp.Data))
	}
	if linksResp.Data[0].GoalID != goalID {
		t.Errorf("goal_id = %q, want %q", linksResp.Data[0].GoalID, goalID)
	}
	if linksResp.Data[0].ImpactType != "positive" {
		t.Errorf("impact_type = %q, want %q", linksResp.Data[0].ImpactType, "positive")
	}
}

// TestTasksCompletion verifies the completed flag flows correctly.
func TestTasksCompletion(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("tasks-complete@example.com", "password123")

	// Create incomplete task
	req := apitest.JSONRequest(t, "POST", "/api/v1/tasks", map[string]any{
		"title":      "Incomplete task",
		"start_date": time.Now().UTC().Format(time.RFC3339),
		"end_date":   time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
	}, apitest.WithToken(token))
	rr := apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var createResp struct {
		Data struct {
			ID        string `json:"id"`
			Completed bool   `json:"completed"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &createResp)
	if createResp.Data.Completed {
		t.Error("new task should not be completed")
	}

	// Mark completed
	req = apitest.JSONRequest(t, "PUT", fmt.Sprintf("/api/v1/tasks/%s", createResp.Data.ID), map[string]any{
		"completed": true,
	}, apitest.WithToken(token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var updateResp struct {
		Data struct {
			Completed bool `json:"completed"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &updateResp)
	if !updateResp.Data.Completed {
		t.Error("task should be completed after update")
	}

	// Verify in list with status filter
	req = apitest.JSONRequest(t, "GET", "/api/v1/tasks?status=completed", nil, apitest.WithToken(token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var listResp struct {
		Data []struct {
			ID        string `json:"id"`
			Completed bool   `json:"completed"`
		} `json:"data"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	apitest.Decode(t, rr, &listResp)
	if listResp.Pagination.Total != 1 {
		t.Errorf("completed tasks total = %d, want 1", listResp.Pagination.Total)
	}
}

// TestTasksValidation verifies malformed requests are rejected.
func TestTasksValidation(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("tasks-val@example.com", "password123")

	tests := []struct {
		name    string
		payload map[string]any
	}{
		{"missing title", map[string]any{"start_date": time.Now().UTC().Format(time.RFC3339), "end_date": time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)}},
		{"missing start_date", map[string]any{"title": "No start", "end_date": time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)}},
		{"missing end_date", map[string]any{"title": "No end", "start_date": time.Now().UTC().Format(time.RFC3339)}},
		{"invalid date format", map[string]any{"title": "Bad date", "start_date": "not-a-date", "end_date": "also-not-a-date"}},
		{"empty payload", map[string]any{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := apitest.JSONRequest(t, "POST", "/api/v1/tasks", tc.payload, apitest.WithToken(token))
			rr := apitest.Do(app.Router, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400; body: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

// TestTasksPagination verifies limit/offset work correctly.
func TestTasksPagination(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("tasks-page@example.com", "password123")

	// Create 5 tasks
	for i := 0; i < 5; i++ {
		req := apitest.JSONRequest(t, "POST", "/api/v1/tasks", map[string]any{
			"title":      fmt.Sprintf("Task %d", i),
			"start_date": time.Now().UTC().Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
			"end_date":   time.Now().UTC().Add(time.Duration(i+1) * time.Hour).Format(time.RFC3339),
		}, apitest.WithToken(token))
		rr := apitest.Do(app.Router, req)
		apitest.AssertStatus(t, rr, http.StatusOK)
	}

	// Page 1: limit=2, offset=0
	req := apitest.JSONRequest(t, "GET", "/api/v1/tasks?limit=2&offset=0", nil, apitest.WithToken(token))
	rr := apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var page1 struct {
		Data []struct {
			Title string `json:"title"`
		} `json:"data"`
		Pagination struct {
			Total  int `json:"total"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		} `json:"pagination"`
	}
	apitest.Decode(t, rr, &page1)

	if page1.Pagination.Total != 5 {
		t.Errorf("total = %d, want 5", page1.Pagination.Total)
	}
	if len(page1.Data) != 2 {
		t.Errorf("page1 len = %d, want 2", len(page1.Data))
	}

	// Page 2: limit=2, offset=2
	req = apitest.JSONRequest(t, "GET", "/api/v1/tasks?limit=2&offset=2", nil, apitest.WithToken(token))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var page2 struct {
		Data []struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &page2)

	if len(page2.Data) != 2 {
		t.Errorf("page2 len = %d, want 2", len(page2.Data))
	}

	// Verify no overlap between pages
	if page1.Data[0].Title == page2.Data[0].Title {
		t.Error("page1 and page2 should have different tasks")
	}
}

// TestTasksMultiTenancy verifies users can only see their own tasks.
func TestTasksMultiTenancy(t *testing.T) {
	app := apitest.NewTestApp(t)
	defer app.Close()

	tokenA := app.RegisterAndLogin("user-a@example.com", "password123")
	tokenB := app.RegisterAndLogin("user-b@example.com", "password123")

	// User A creates a task
	req := apitest.JSONRequest(t, "POST", "/api/v1/tasks", map[string]any{
		"title":      "User A task",
		"start_date": time.Now().UTC().Format(time.RFC3339),
		"end_date":   time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
	}, apitest.WithToken(tokenA))
	rr := apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var createResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	apitest.Decode(t, rr, &createResp)
	taskID := createResp.Data.ID

	// User B cannot see user A's task
	req = apitest.JSONRequest(t, "GET", fmt.Sprintf("/api/v1/tasks/%s", taskID), nil, apitest.WithToken(tokenB))
	rr = apitest.Do(app.Router, req)
	apitest.AssertError(t, rr, http.StatusNotFound, "NOT_FOUND")

	// User B's list is empty
	req = apitest.JSONRequest(t, "GET", "/api/v1/tasks", nil, apitest.WithToken(tokenB))
	rr = apitest.Do(app.Router, req)
	apitest.AssertStatus(t, rr, http.StatusOK)

	var listResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	apitest.Decode(t, rr, &listResp)
	if listResp.Pagination.Total != 0 {
		t.Errorf("user B should see 0 tasks, got %d", listResp.Pagination.Total)
	}
}
