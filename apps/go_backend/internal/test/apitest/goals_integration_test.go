package apitest

import (
	"net/http"
	"testing"
)

func TestGoalsListReturnsCreatedGoals(t *testing.T) {
	app := NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("goals-list@example.com", "password123")

	// Create a goal
	createReq := map[string]any{
		"title":    "Test Goal",
		"priority": 1,
	}
	req := JSONRequest(t, "POST", "/api/v1/goals", createReq, WithToken(token))
	rr := Do(app.Router, req)
	AssertStatus(t, rr, http.StatusCreated)

	var createResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	Decode(t, rr, &createResp)
	if createResp.Data.ID == "" {
		t.Fatal("expected goal ID in create response")
	}

	// List goals — should contain the created goal
	req = JSONRequest(t, "GET", "/api/v1/goals", nil, WithToken(token))
	rr = Do(app.Router, req)
	AssertStatus(t, rr, http.StatusOK)

	var listResp struct {
		Data struct {
			Items []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"items"`
			Total int `json:"total"`
		} `json:"data"`
	}
	Decode(t, rr, &listResp)

	if listResp.Data.Total == 0 {
		t.Error("goals list returned empty after creating a goal")
	}
	if len(listResp.Data.Items) == 0 {
		t.Error("goals list returned 0 items after creating a goal")
	}

	// Verify the created goal is in the list
	found := false
	for _, g := range listResp.Data.Items {
		if g.ID == createResp.Data.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("created goal %s not found in list", createResp.Data.ID)
	}
}

// TestGoalsListStatusAllRegression verifies that status=all does not filter out
// all goals. This was a bug where the backend treated "all" as a literal status
// value instead of "no filter", causing GET /goals?status=all to return 0 items.
func TestGoalsListStatusAllRegression(t *testing.T) {
	app := NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("goals-status-all@example.com", "password123")

	// Create two goals
	for _, title := range []string{"Goal A", "Goal B"} {
		createReq := map[string]any{
			"title":    title,
			"priority": 1,
		}
		req := JSONRequest(t, "POST", "/api/v1/goals", createReq, WithToken(token))
		rr := Do(app.Router, req)
		AssertStatus(t, rr, http.StatusCreated)
	}

	var listResp struct {
		Data struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
			Total int `json:"total"`
		} `json:"data"`
	}

	// status=all should return both goals, not zero
	req := JSONRequest(t, "GET", "/api/v1/goals?status=all", nil, WithToken(token))
	rr := Do(app.Router, req)
	AssertStatus(t, rr, http.StatusOK)
	Decode(t, rr, &listResp)
	if listResp.Data.Total != 2 {
		t.Errorf("status=all: expected 2 goals, got %d", listResp.Data.Total)
	}

	// no status should also return both
	req = JSONRequest(t, "GET", "/api/v1/goals", nil, WithToken(token))
	rr = Do(app.Router, req)
	AssertStatus(t, rr, http.StatusOK)
	Decode(t, rr, &listResp)
	if listResp.Data.Total != 2 {
		t.Errorf("no status: expected 2 goals, got %d", listResp.Data.Total)
	}
}
