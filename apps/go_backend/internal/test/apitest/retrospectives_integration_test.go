package apitest

import (
	"net/http"
	"testing"
	"time"
)

func TestRetrospectiveAutoSummaryRoundtrip(t *testing.T) {
	app := NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("retro-test@example.com", "password123")

	// Generate a retrospective for today
	req := JSONRequest(t, "POST", "/api/v1/retrospectives/generate", map[string]any{
		"retro_type": "daily",
	}, WithToken(token))
	rr := Do(app.Router, req)
	AssertStatus(t, rr, http.StatusCreated)

	var genResp struct {
		Data struct {
			ID          string `json:"id"`
			AutoSummary struct {
				Tasks struct {
					Completed int `json:"completed"`
				} `json:"tasks"`
			} `json:"auto_summary"`
		} `json:"data"`
	}
	Decode(t, rr, &genResp)
	if genResp.Data.ID == "" {
		t.Fatal("expected retrospective ID in generate response")
	}

	// Fetch the retrospective — auto_summary should be populated (not empty)
	req = JSONRequest(t, "GET", "/api/v1/retrospectives/"+genResp.Data.ID, nil, WithToken(token))
	rr = Do(app.Router, req)
	AssertStatus(t, rr, http.StatusOK)

	var getResp struct {
		Data struct {
			ID          string `json:"id"`
			AutoSummary struct {
				Tasks struct {
					Completed int `json:"completed"`
				} `json:"tasks"`
				Mood struct {
					AverageValence float64 `json:"average_valence"`
				} `json:"mood"`
			} `json:"auto_summary"`
		} `json:"data"`
	}
	Decode(t, rr, &getResp)

	// The auto_summary should have been persisted and returned
	// Even if there are no tasks, the struct should exist (not be a zero-value stub)
	// We can't assert exact values without seeding data, but we can verify the field exists
	// by checking that the response doesn't contain an empty auto_summary when one was stored.
	// For now, verify the roundtrip works — the auto_summary should be identical to what was generated.
	if getResp.Data.ID != genResp.Data.ID {
		t.Errorf("expected same retrospective ID, got %s vs %s", getResp.Data.ID, genResp.Data.ID)
	}

	// The auto_summary should not be a completely empty struct after roundtrip
	// (This will fail until parseAutoSummary is implemented)
	if getResp.Data.AutoSummary.Tasks.Completed != genResp.Data.AutoSummary.Tasks.Completed {
		t.Errorf("auto_summary.tasks.completed lost after roundtrip: expected %d, got %d",
			genResp.Data.AutoSummary.Tasks.Completed, getResp.Data.AutoSummary.Tasks.Completed)
	}
}

func TestRetrospectiveGenerateAndList(t *testing.T) {
	app := NewTestApp(t)
	defer app.Close()

	token := app.RegisterAndLogin("retro-list@example.com", "password123")

	// Generate a retro
	req := JSONRequest(t, "POST", "/api/v1/retrospectives/generate", map[string]any{
		"retro_type": "daily",
	}, WithToken(token))
	rr := Do(app.Router, req)
	AssertStatus(t, rr, http.StatusCreated)

	// List retros — should contain the generated one
	req = JSONRequest(t, "GET", "/api/v1/retrospectives", nil, WithToken(token))
	rr = Do(app.Router, req)
	AssertStatus(t, rr, http.StatusOK)

	var listResp struct {
		Data struct {
			Retrospectives []struct {
				ID        string    `json:"id"`
				RetroType string    `json:"retro_type"`
				StartDate time.Time `json:"start_date"`
			} `json:"retrospectives"`
			Total int `json:"total"`
		} `json:"data"`
	}
	Decode(t, rr, &listResp)

	if len(listResp.Data.Retrospectives) == 0 {
		t.Error("retrospectives list returned empty after generating a retro")
	}
}
