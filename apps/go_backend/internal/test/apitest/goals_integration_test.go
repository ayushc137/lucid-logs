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
