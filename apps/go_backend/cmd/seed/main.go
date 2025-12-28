// Package main provides a comprehensive data seeder for development and testing.
//
// This CLI tool populates the database with sample data across 7 days in the past
// and 3 days in the future for realistic stress testing.
//
// Features:
//   - Tasks with/without categories, emotions, positives/negatives
//   - Short and long duration tasks
//   - Goals: discrete, measurable, recurring (habits), epic with milestones
//   - Templates for quick logging
//   - Task-goal links
//   - Realistic data distribution for heavy user testing
//
// Usage:
//
//	go run ./cmd/seed
//	go run ./cmd/seed --reset  # Delete existing data first
//	# or
//	task seed
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/config"
	"github.com/lucid-logs/go-backend/internal/shared/database"
)

// API client configuration
var (
	apiBaseURL string
	authToken  string
	httpClient *http.Client
)

func main() {
	// Parse flags
	resetFlag := flag.Bool("reset", false, "Delete existing seeded data before creating new data")
	flag.Parse()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup pretty logging for CLI
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	})

	// Only run in dev mode
	if !cfg.IsDev() {
		log.Fatal().Msg("❌ Data seeding is only allowed in development mode")
	}

	log.Info().Msg("🌱 Starting comprehensive data seeder...")

	// Connect to database for direct operations (reset, lookup)
	ctx := context.Background()
	db, err := database.New(ctx, database.Config{
		URL:        cfg.Database.WebSocketURL(),
		Namespace:  cfg.Database.Namespace,
		Database:   cfg.Database.Database,
		Username:   cfg.Database.User,
		Password:   cfg.Database.Password,
		LogQueries: false,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close(ctx)

	// Setup API client
	apiBaseURL = fmt.Sprintf("http://127.0.0.1:%d/api/v1", cfg.Server.Port)
	httpClient = &http.Client{Timeout: 30 * time.Second}

	// Authenticate and get token
	authToken, err = authenticate(cfg.Admin.Username, cfg.Admin.Password)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to authenticate - make sure backend is running")
	}
	log.Info().Str("email", cfg.Admin.Username).Msg("✅ Authenticated as admin user")

	// Find admin user ID
	userID, err := findAdminUser(ctx, db, cfg.Admin.Username)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to find admin user")
	}

	// Reset existing data if flag is set
	if *resetFlag {
		log.Info().Msg("🗑️ Resetting existing seeded data...")
		if err := resetSeededData(ctx, db, userID); err != nil {
			log.Fatal().Err(err).Msg("Failed to reset data")
		}
		log.Info().Msg("✅ Existing data deleted")
	}

	// Create comprehensive seed data
	if err := seedAll(ctx, db, userID); err != nil {
		log.Fatal().Err(err).Msg("Failed to seed data")
	}

	log.Info().Msg("🎉 Comprehensive data seeding complete!")
}

// =============================================================================
// AUTHENTICATION
// =============================================================================

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponseData struct {
	Token   string `json:"token"`
	User    string `json:"user"`
	IsAdmin bool   `json:"is_admin"`
}

type loginResponse struct {
	Data loginResponseData `json:"data"`
}

func authenticate(email, password string) (string, error) {
	payload, _ := json.Marshal(loginRequest{Username: email, Password: password})
	resp, err := http.Post(apiBaseURL+"/auth/login", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode login response: %w", err)
	}

	if result.Data.Token == "" {
		return "", fmt.Errorf("empty token in response")
	}

	return result.Data.Token, nil
}

// =============================================================================
// API HELPER
// =============================================================================

func apiRequest(method, path string, body any) (map[string]any, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, apiBaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			// Try parsing as array
			var arr []any
			if err := json.Unmarshal(respBody, &arr); err != nil {
				return nil, fmt.Errorf("failed to parse response: %s", string(respBody))
			}
			result = map[string]any{"items": arr}
		}
	}

	// Unwrap the "data" field if present (standard API response format)
	if data, ok := result["data"].(map[string]any); ok {
		return data, nil
	}

	// Handle paginated responses where data contains items array
	if data, ok := result["data"]; ok {
		if dataMap, ok := data.(map[string]any); ok {
			return dataMap, nil
		}
	}

	return result, nil
}

// =============================================================================
// DATABASE HELPERS
// =============================================================================

type userResult struct {
	ID string `json:"id"`
}

func findAdminUser(ctx context.Context, db *database.DB, email string) (string, error) {
	users, err := database.QueryAll[userResult](ctx, db, `
		SELECT type::string(id) as id FROM users WHERE email = $email LIMIT 1
	`, map[string]any{
		"email": email,
	})
	if err != nil {
		return "", err
	}
	if len(users) == 0 {
		return "", fmt.Errorf("admin user not found (email: %s)", email)
	}
	return users[0].ID, nil
}

func resetSeededData(ctx context.Context, db *database.DB, userID string) error {
	// Delete task_goals relations
	_, _ = database.QueryAll[userResult](ctx, db, `
		DELETE task_goals WHERE in.created_by = $user OR out.created_by = $user
	`, map[string]any{"user": userID})

	// Delete goal entries
	_, _ = database.QueryAll[userResult](ctx, db, `
		DELETE goal_entries
	`, nil)

	// Delete tasks
	_, err := database.QueryAll[userResult](ctx, db, `
		DELETE tasks WHERE created_by = $user
	`, map[string]any{"user": userID})
	if err != nil {
		return err
	}

	// Delete goals
	_, _ = database.QueryAll[userResult](ctx, db, `
		DELETE goals WHERE created_by = $user
	`, map[string]any{"user": userID})

	// Delete templates
	_, _ = database.QueryAll[userResult](ctx, db, `
		DELETE templates WHERE created_by = $user
	`, map[string]any{"user": userID})

	// Delete categories
	_, err = database.QueryAll[userResult](ctx, db, `
		DELETE categories WHERE created_by = $user
	`, map[string]any{"user": userID})
	if err != nil {
		return err
	}

	// Cleanup orphaned data
	_, _ = database.QueryAll[userResult](ctx, db, `
		DELETE tasks WHERE created_by NOT IN (SELECT id FROM users)
	`, nil)
	_, _ = database.QueryAll[userResult](ctx, db, `
		DELETE categories WHERE created_by NOT IN (SELECT id FROM users)
	`, nil)
	_, _ = database.QueryAll[userResult](ctx, db, `
		DELETE goals WHERE created_by NOT IN (SELECT id FROM users)
	`, nil)

	return nil
}

// =============================================================================
// SEED ORCHESTRATOR
// =============================================================================

func seedAll(ctx context.Context, db *database.DB, userID string) error {
	now := time.Now()

	// 1. Create Categories
	categories, err := seedCategories()
	if err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}
	log.Info().Int("count", len(categories)).Msg("✅ Categories created")

	// 2. Create Goals
	goals, err := seedGoals(categories)
	if err != nil {
		return fmt.Errorf("failed to seed goals: %w", err)
	}
	log.Info().Int("count", len(goals)).Msg("✅ Goals created")

	// 3. Create Templates
	templates, err := seedTemplates(categories)
	if err != nil {
		return fmt.Errorf("failed to seed templates: %w", err)
	}
	log.Info().Int("count", len(templates)).Msg("✅ Templates created")

	// 4. Create Tasks for each day
	totalTasks := 0
	for dayOffset := -7; dayOffset <= 3; dayOffset++ {
		day := now.AddDate(0, 0, dayOffset)
		taskCount, err := seedTasksForDay(day, dayOffset, categories, goals)
		if err != nil {
			log.Warn().Err(err).Int("day_offset", dayOffset).Msg("Failed to seed some tasks")
		}
		totalTasks += taskCount
	}
	log.Info().Int("count", totalTasks).Msg("✅ Tasks created")

	return nil
}

// =============================================================================
// CATEGORY SEEDING
// =============================================================================

type categoryDef struct {
	name  string
	color string
}

var categoryDefs = []categoryDef{
	{"Work", "#3B82F6"},       // Blue
	{"Personal", "#10B981"},   // Green
	{"Health", "#EF4444"},     // Red
	{"Learning", "#8B5CF6"},   // Purple
	{"Errands", "#F59E0B"},    // Amber
	{"Social", "#EC4899"},     // Pink
	{"Finance", "#06B6D4"},    // Cyan
	{"Creative", "#F97316"},   // Orange
	{"Travel", "#14B8A6"},     // Teal
	{"Home", "#84CC16"},       // Lime
	{"Family", "#E879F9"},     // Fuchsia
	{"Meditation", "#6366F1"}, // Indigo
	{"Reading", "#A855F7"},    // Violet
	{"Projects", "#0EA5E9"},   // Sky
}

func seedCategories() (map[string]string, error) {
	result := make(map[string]string)

	for _, cat := range categoryDefs {
		resp, err := apiRequest("POST", "/categories", map[string]any{
			"name":  cat.name,
			"color": cat.color,
		})
		if err != nil {
			// Try to get existing category
			listResp, listErr := apiRequest("GET", "/categories?limit=100", nil)
			if listErr == nil {
				if items, ok := listResp["items"].([]any); ok {
					for _, item := range items {
						if m, ok := item.(map[string]any); ok {
							if m["name"] == cat.name {
								if id, ok := m["id"].(string); ok {
									result[cat.name] = id
								}
							}
						}
					}
				}
			}
			if result[cat.name] == "" {
				log.Warn().Err(err).Str("category", cat.name).Msg("Failed to create category")
			}
			continue
		}

		if id, ok := resp["id"].(string); ok {
			result[cat.name] = id
		}
	}

	return result, nil
}

// =============================================================================
// GOAL SEEDING
// =============================================================================

type goalDef struct {
	title       string
	description string
	why         string
	goalType    string
	icon        string
	color       string
	category    string
	priority    int
	valueScore  int
	lifeDomain  string
	recurrence  map[string]any
	target      map[string]any
	startDate   *time.Time
	deadline    *time.Time
}

func seedGoals(categories map[string]string) (map[string]string, error) {
	result := make(map[string]string)
	now := time.Now()
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	endOfYear := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, now.Location())
	nextMonth := now.AddDate(0, 1, 0)
	nextWeek := now.AddDate(0, 0, 7)

	goals := []goalDef{
		// Measurable recurring goals (habits)
		{
			title:       "Drink 3L Water Daily",
			description: "Stay hydrated throughout the day for better health and focus",
			why:         "Improve energy levels, skin health, and cognitive function",
			goalType:    "measurable",
			icon:        "💧",
			color:       "#3B82F6",
			category:    "Health",
			priority:    3,
			valueScore:  5,
			lifeDomain:  "health",
			recurrence:  map[string]any{"frequency": 1, "period": "day", "grace_days": 1},
			target:      map[string]any{"value": 3.0, "unit": "liters", "per_period": true},
		},
		{
			title:       "Exercise 5x per Week",
			description: "Regular physical activity for strength and cardio",
			why:         "Maintain physical fitness and mental clarity",
			goalType:    "measurable",
			icon:        "🏃",
			color:       "#EF4444",
			category:    "Health",
			priority:    3,
			valueScore:  5,
			lifeDomain:  "health",
			recurrence:  map[string]any{"frequency": 5, "period": "week", "grace_days": 1, "active_days": []string{"mon", "tue", "wed", "thu", "fri"}},
			target:      map[string]any{"value": 30, "unit": "minutes", "per_period": true},
		},
		{
			title:       "Meditate Daily",
			description: "Morning mindfulness practice",
			why:         "Reduce stress and improve focus",
			goalType:    "measurable",
			icon:        "🧘",
			color:       "#8B5CF6",
			category:    "Meditation",
			priority:    2,
			valueScore:  4,
			lifeDomain:  "wellbeing",
			recurrence:  map[string]any{"frequency": 1, "period": "day", "before_time": "09:00"},
			target:      map[string]any{"value": 15, "unit": "minutes", "per_period": true},
		},
		{
			title:       "Read 30 Minutes Daily",
			description: "Daily reading habit for growth and relaxation",
			why:         "Continuous learning and mental stimulation",
			goalType:    "measurable",
			icon:        "📚",
			color:       "#A855F7",
			category:    "Reading",
			priority:    2,
			valueScore:  4,
			lifeDomain:  "learning",
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
			target:      map[string]any{"value": 30, "unit": "minutes", "per_period": true},
		},
		// Measurable one-time goals
		{
			title:       "Run 100km This Month",
			description: "Monthly running challenge",
			why:         "Build endurance and prepare for marathon",
			goalType:    "measurable",
			icon:        "🏅",
			color:       "#10B981",
			category:    "Health",
			priority:    3,
			valueScore:  5,
			lifeDomain:  "health",
			target:      map[string]any{"value": 100, "unit": "km", "per_period": false},
			startDate:   &startOfYear,
			deadline:    &nextMonth,
		},
		{
			title:       "Save $5000",
			description: "Emergency fund savings goal",
			why:         "Financial security and peace of mind",
			goalType:    "measurable",
			icon:        "💰",
			color:       "#06B6D4",
			category:    "Finance",
			priority:    3,
			valueScore:  5,
			lifeDomain:  "finance",
			target:      map[string]any{"value": 5000, "unit": "dollars", "per_period": false},
			startDate:   &startOfYear,
			deadline:    &endOfYear,
		},
		// Discrete (one-time) goals
		{
			title:       "Complete Go Backend Course",
			description: "Finish the advanced Go programming course",
			why:         "Career advancement and skill development",
			goalType:    "discrete",
			icon:        "🎓",
			color:       "#0EA5E9",
			category:    "Learning",
			priority:    2,
			valueScore:  4,
			lifeDomain:  "career",
			startDate:   &startOfYear,
			deadline:    &nextMonth,
		},
		{
			title:       "Organize Home Office",
			description: "Declutter and set up ergonomic workspace",
			why:         "Better productivity and comfort while working from home",
			goalType:    "discrete",
			icon:        "🏠",
			color:       "#84CC16",
			category:    "Home",
			priority:    2,
			valueScore:  3,
			lifeDomain:  "environment",
			deadline:    &nextWeek,
		},
		{
			title:       "Plan Family Vacation",
			description: "Research and book summer vacation",
			why:         "Quality time with family and relaxation",
			goalType:    "discrete",
			icon:        "✈️",
			color:       "#14B8A6",
			category:    "Travel",
			priority:    2,
			valueScore:  4,
			lifeDomain:  "relationships",
			deadline:    &nextMonth,
		},
		// Avoidance goals
		{
			title:       "No Social Media Before 10am",
			description: "Avoid social media scrolling in the morning",
			why:         "Protect morning focus and productivity",
			goalType:    "avoidance",
			icon:        "📵",
			color:       "#F97316",
			category:    "Personal",
			priority:    2,
			valueScore:  4,
			lifeDomain:  "productivity",
			recurrence:  map[string]any{"frequency": 1, "period": "day", "before_time": "10:00"},
		},
		{
			title:       "No Junk Food",
			description: "Eliminate processed and fast food",
			why:         "Better health and energy levels",
			goalType:    "avoidance",
			icon:        "🍟",
			color:       "#EF4444",
			category:    "Health",
			priority:    2,
			valueScore:  4,
			lifeDomain:  "health",
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// Epic goal with milestones (we'll add children separately)
		{
			title:       "Launch SaaS Product",
			description: "Build and launch a profitable software product",
			why:         "Financial independence and creative fulfillment",
			goalType:    "epic",
			icon:        "🚀",
			color:       "#F59E0B",
			category:    "Projects",
			priority:    3,
			valueScore:  5,
			lifeDomain:  "career",
			startDate:   &startOfYear,
			deadline:    &endOfYear,
		},
		// Recurring habits with different frequencies
		{
			title:       "Weekly Review",
			description: "Review goals, tasks, and plan next week",
			why:         "Stay organized and aligned with priorities",
			goalType:    "discrete",
			icon:        "📋",
			color:       "#6366F1",
			category:    "Personal",
			priority:    3,
			valueScore:  5,
			lifeDomain:  "productivity",
			recurrence:  map[string]any{"frequency": 1, "period": "week", "active_days": []string{"sun"}},
		},
		{
			title:       "Monthly Budget Review",
			description: "Review and adjust monthly finances",
			why:         "Financial awareness and control",
			goalType:    "discrete",
			icon:        "📊",
			color:       "#06B6D4",
			category:    "Finance",
			priority:    2,
			valueScore:  4,
			lifeDomain:  "finance",
			recurrence:  map[string]any{"frequency": 1, "period": "month"},
		},
		{
			title:       "Call Parents Weekly",
			description: "Regular check-in with family",
			why:         "Maintain strong family bonds",
			goalType:    "discrete",
			icon:        "📞",
			color:       "#E879F9",
			category:    "Family",
			priority:    2,
			valueScore:  5,
			lifeDomain:  "relationships",
			recurrence:  map[string]any{"frequency": 1, "period": "week", "active_days": []string{"sat", "sun"}},
		},
	}

	for _, g := range goals {
		payload := map[string]any{
			"title":       g.title,
			"description": g.description,
			"why":         g.why,
			"goal_type":   g.goalType,
			"icon":        g.icon,
			"color":       g.color,
			"priority":    g.priority,
			"value_score": g.valueScore,
			"life_domain": g.lifeDomain,
		}

		if g.category != "" && categories[g.category] != "" {
			payload["category_id"] = categories[g.category]
		}
		if g.recurrence != nil {
			payload["recurrence"] = g.recurrence
		}
		if g.target != nil {
			payload["target"] = g.target
		}
		if g.startDate != nil {
			payload["start_date"] = g.startDate.Format(time.RFC3339)
		}
		if g.deadline != nil {
			payload["deadline"] = g.deadline.Format(time.RFC3339)
		}

		resp, err := apiRequest("POST", "/goals", payload)
		if err != nil {
			log.Warn().Err(err).Str("goal", g.title).Msg("Failed to create goal")
			continue
		}

		if id, ok := resp["id"].(string); ok {
			result[g.title] = id
		}
	}

	return result, nil
}

// =============================================================================
// TEMPLATE SEEDING
// =============================================================================

type templateDef struct {
	title            string
	description      string
	icon             string
	color            string
	defaultDuration  int // seconds
	defaultPriority  int
	category         string
	isQuickLog       bool
	quickLogOrder    int
	quantityEnabled  bool
	quantityDefault  float64
	quantityUnit     string
	quantityStep     float64
	expectedQuadrant string
	activityKey      string
}

func seedTemplates(categories map[string]string) (map[string]string, error) {
	result := make(map[string]string)

	templates := []templateDef{
		{
			title:            "Morning Run",
			description:      "Quick morning jog",
			icon:             "🏃",
			color:            "#EF4444",
			defaultDuration:  1800, // 30 min
			defaultPriority:  2,
			category:         "Health",
			isQuickLog:       true,
			quickLogOrder:    1,
			quantityEnabled:  true,
			quantityDefault:  5.0,
			quantityUnit:     "km",
			quantityStep:     0.5,
			expectedQuadrant: "yellow",
			activityKey:      "running",
		},
		{
			title:            "Gym Workout",
			description:      "Strength training session",
			icon:             "💪",
			color:            "#EF4444",
			defaultDuration:  3600, // 1 hour
			defaultPriority:  2,
			category:         "Health",
			isQuickLog:       true,
			quickLogOrder:    2,
			quantityEnabled:  true,
			quantityDefault:  60,
			quantityUnit:     "minutes",
			quantityStep:     15,
			expectedQuadrant: "yellow",
			activityKey:      "gym",
		},
		{
			title:            "Meditation Session",
			description:      "Mindfulness practice",
			icon:             "🧘",
			color:            "#8B5CF6",
			defaultDuration:  900, // 15 min
			defaultPriority:  2,
			category:         "Meditation",
			isQuickLog:       true,
			quickLogOrder:    3,
			quantityEnabled:  true,
			quantityDefault:  15,
			quantityUnit:     "minutes",
			quantityStep:     5,
			expectedQuadrant: "green",
			activityKey:      "meditation",
		},
		{
			title:            "Reading",
			description:      "Book reading session",
			icon:             "📚",
			color:            "#A855F7",
			defaultDuration:  1800, // 30 min
			defaultPriority:  1,
			category:         "Reading",
			isQuickLog:       true,
			quickLogOrder:    4,
			quantityEnabled:  true,
			quantityDefault:  30,
			quantityUnit:     "minutes",
			quantityStep:     10,
			expectedQuadrant: "green",
			activityKey:      "reading",
		},
		{
			title:           "Water Intake",
			description:     "Log water consumption",
			icon:            "💧",
			color:           "#3B82F6",
			defaultDuration: 60, // 1 min
			defaultPriority: 1,
			category:        "Health",
			isQuickLog:      true,
			quickLogOrder:   5,
			quantityEnabled: true,
			quantityDefault: 0.5,
			quantityUnit:    "liters",
			quantityStep:    0.25,
			activityKey:     "water",
		},
		{
			title:            "Deep Work Session",
			description:      "Focused work block",
			icon:             "🎯",
			color:            "#3B82F6",
			defaultDuration:  5400, // 90 min
			defaultPriority:  3,
			category:         "Work",
			isQuickLog:       true,
			quickLogOrder:    6,
			quantityEnabled:  true,
			quantityDefault:  90,
			quantityUnit:     "minutes",
			quantityStep:     30,
			expectedQuadrant: "yellow",
			activityKey:      "deep_work",
		},
		{
			title:            "Coffee Break",
			description:      "Short break with coffee",
			icon:             "☕",
			color:            "#F59E0B",
			defaultDuration:  900, // 15 min
			defaultPriority:  1,
			category:         "Personal",
			isQuickLog:       true,
			quickLogOrder:    7,
			expectedQuadrant: "green",
		},
		{
			title:           "Team Meeting",
			description:     "Scheduled team sync",
			icon:            "👥",
			color:           "#3B82F6",
			defaultDuration: 1800, // 30 min
			defaultPriority: 2,
			category:        "Work",
			isQuickLog:      false,
			activityKey:     "meeting",
		},
		{
			title:           "Client Call",
			description:     "External client meeting",
			icon:            "📞",
			color:           "#10B981",
			defaultDuration: 2700, // 45 min
			defaultPriority: 3,
			category:        "Work",
			isQuickLog:      false,
			activityKey:     "client_call",
		},
		{
			title:           "Code Review",
			description:     "Review pull requests",
			icon:            "🔍",
			color:           "#0EA5E9",
			defaultDuration: 3600, // 1 hour
			defaultPriority: 2,
			category:        "Work",
			isQuickLog:      false,
			activityKey:     "code_review",
		},
		{
			title:            "Learning Session",
			description:      "Study and skill building",
			icon:             "🎓",
			color:            "#8B5CF6",
			defaultDuration:  3600, // 1 hour
			defaultPriority:  2,
			category:         "Learning",
			isQuickLog:       true,
			quickLogOrder:    8,
			quantityEnabled:  true,
			quantityDefault:  60,
			quantityUnit:     "minutes",
			quantityStep:     30,
			expectedQuadrant: "yellow",
			activityKey:      "learning",
		},
		{
			title:            "Journaling",
			description:      "Reflection and writing",
			icon:             "📝",
			color:            "#10B981",
			defaultDuration:  900, // 15 min
			defaultPriority:  1,
			category:         "Personal",
			isQuickLog:       true,
			quickLogOrder:    9,
			expectedQuadrant: "green",
			activityKey:      "journaling",
		},
	}

	for i, t := range templates {
		payload := map[string]any{
			"title":            t.title,
			"description":      t.description,
			"icon":             t.icon,
			"color":            t.color,
			"default_duration": t.defaultDuration,
			"default_priority": t.defaultPriority,
			"is_quick_log":     t.isQuickLog,
			"quick_log_order":  t.quickLogOrder,
		}

		if t.category != "" && categories[t.category] != "" {
			payload["default_category_id"] = categories[t.category]
		}
		if t.quantityEnabled {
			payload["quantity_enabled"] = true
			payload["quantity_default"] = t.quantityDefault
			payload["quantity_unit"] = t.quantityUnit
			payload["quantity_step"] = t.quantityStep
		}
		if t.expectedQuadrant != "" {
			payload["expected_quadrant"] = t.expectedQuadrant
		}
		if t.activityKey != "" {
			payload["activity_key"] = t.activityKey
		}

		resp, err := apiRequest("POST", "/templates", payload)
		if err != nil {
			log.Warn().Err(err).Str("template", t.title).Msg("Failed to create template")
			continue
		}

		if id, ok := resp["id"].(string); ok {
			result[t.title] = id
			log.Debug().Str("template", t.title).Int("order", i).Msg("Created template")
		}
	}

	return result, nil
}

// =============================================================================
// TASK SEEDING
// =============================================================================

type taskDef struct {
	title     string
	journal   string
	hour      int
	minute    int
	duration  time.Duration // minutes
	priority  int
	completed bool
	category  string
	emotionID string // e.g., "emotions:E16"
	positives []taskItem
	negatives []taskItem
	note      string
}

type taskItem struct {
	text      string
	emotionID string
}

// Emotion IDs from the 100 emotions grid
var (
	// Yellow (High Energy + Pleasant)
	emotionHappy        = "emotions:E16"
	emotionExcited      = "emotions:E03"
	emotionProud        = "emotions:E11"
	emotionMotivated    = "emotions:E15"
	emotionInspired     = "emotions:E18"
	emotionCurious      = "emotions:E23"
	emotionConfident    = "emotions:E12"
	emotionJoyful       = "emotions:E06"
	emotionEnthusiastic = "emotions:E09"
	emotionAccomplished = "emotions:E19"

	// Green (Low Energy + Pleasant)
	emotionCalm      = "emotions:E44"
	emotionContent   = "emotions:E26"
	emotionGrateful  = "emotions:E31"
	emotionRelaxed   = "emotions:E45"
	emotionPeaceful  = "emotions:E48"
	emotionSatisfied = "emotions:E27"
	emotionRelieved  = "emotions:E29"

	// Red (High Energy + Unpleasant)
	emotionFrustrated  = "emotions:E61"
	emotionStressed    = "emotions:E59"
	emotionAnxious     = "emotions:E60"
	emotionWorried     = "emotions:E65"
	emotionOverwhelmed = "emotions:E55"
	emotionNervous     = "emotions:E69"
	emotionImpatient   = "emotions:E75"

	// Blue (Low Energy + Unpleasant)
	emotionTired        = "emotions:E91"
	emotionBored        = "emotions:E95"
	emotionDrained      = "emotions:E93"
	emotionSad          = "emotions:E84"
	emotionDisappointed = "emotions:E76"
)

func seedTasksForDay(day time.Time, dayOffset int, categories map[string]string, goals map[string]string) (int, error) {
	var tasks []taskDef

	isPast := dayOffset < 0
	isToday := dayOffset == 0
	isWeekend := day.Weekday() == time.Saturday || day.Weekday() == time.Sunday

	// Generate realistic tasks based on day type
	if isPast {
		tasks = generatePastDayTasks(day, dayOffset, isWeekend)
	} else if isToday {
		tasks = generateTodayTasks(day, isWeekend)
	} else {
		tasks = generateFutureTasks(day, dayOffset, isWeekend)
	}

	count := 0
	for _, t := range tasks {
		startTime := time.Date(day.Year(), day.Month(), day.Day(), t.hour, t.minute, 0, 0, day.Location())
		endTime := startTime.Add(t.duration)

		payload := map[string]any{
			"title":      t.title,
			"journal":    t.journal,
			"start_date": startTime.Format(time.RFC3339),
			"end_date":   endTime.Format(time.RFC3339),
			"priority":   t.priority,
		}

		if t.category != "" && categories[t.category] != "" {
			payload["category_id"] = categories[t.category]
		}
		if t.emotionID != "" {
			payload["emotion_id"] = t.emotionID
		}
		if len(t.positives) > 0 {
			positives := make([]map[string]any, len(t.positives))
			for i, p := range t.positives {
				item := map[string]any{"text": p.text}
				if p.emotionID != "" {
					item["emotion_id"] = p.emotionID
				}
				positives[i] = item
			}
			payload["positives"] = positives
		}
		if len(t.negatives) > 0 {
			negatives := make([]map[string]any, len(t.negatives))
			for i, n := range t.negatives {
				item := map[string]any{"text": n.text}
				if n.emotionID != "" {
					item["emotion_id"] = n.emotionID
				}
				negatives[i] = item
			}
			payload["negatives"] = negatives
		}
		if t.note != "" {
			payload["note"] = t.note
		}

		resp, err := apiRequest("POST", "/tasks", payload)
		if err != nil {
			log.Warn().Err(err).Str("task", t.title).Msg("Failed to create task")
			continue
		}

		taskID, _ := resp["id"].(string)

		// Mark completed if needed (past tasks or some today tasks)
		if t.completed && taskID != "" {
			// Extract just the ID part (strip "tasks:" prefix if present)
			idPart := taskID
			if parts := strings.SplitN(taskID, ":", 2); len(parts) == 2 {
				idPart = parts[1]
			}
			_, err := apiRequest("PUT", "/tasks/"+idPart, map[string]any{
				"completed": true,
			})
			if err != nil {
				log.Warn().Err(err).Str("task", t.title).Msg("Failed to mark task completed")
			}
		}

		count++
	}

	log.Debug().
		Int("count", count).
		Str("date", day.Format("2006-01-02")).
		Int("offset", dayOffset).
		Bool("weekend", isWeekend).
		Msg("Created tasks for day")

	return count, nil
}

// =============================================================================
// TASK GENERATORS
// =============================================================================

func generatePastDayTasks(day time.Time, dayOffset int, isWeekend bool) []taskDef {
	var tasks []taskDef

	// Vary completion rates - not all past tasks should be complete!
	// More recent days have more incomplete tasks
	var completionRate float64
	switch {
	case dayOffset == -1: // Yesterday - most incomplete
		completionRate = 0.55
	case dayOffset == -2: // 2 days ago
		completionRate = 0.65
	case dayOffset == -3: // 3 days ago
		completionRate = 0.70
	case dayOffset >= -5: // 4-5 days ago
		completionRate = 0.75
	default: // 6-7 days ago
		completionRate = 0.80
	}

	if isWeekend {
		tasks = generateWeekendTasks(true, completionRate)
	} else {
		tasks = generateWorkdayTasks(true, completionRate)
	}

	// Add extra tasks based on day for variety
	extraTasks := generateExtraTasks(dayOffset, true, completionRate)
	tasks = append(tasks, extraTasks...)

	return tasks
}

// generateExtraTasks adds variety of extra tasks based on day offset
func generateExtraTasks(dayOffset int, isPast bool, completionRate float64) []taskDef {
	var tasks []taskDef

	// Common extra tasks that can appear on any day
	commonTasks := []taskDef{
		// Very short tasks (1-5 min)
		{title: "Take medication", hour: 8, minute: 0, duration: 1 * time.Minute, priority: 10, category: "Health"},
		{title: "Check calendar", hour: 7, minute: 30, duration: 2 * time.Minute, priority: 5},
		{title: "Quick stretch", hour: 10, minute: 0, duration: 5 * time.Minute, priority: 4, category: "Health"},
		{title: "Water plants", hour: 8, minute: 15, duration: 3 * time.Minute, priority: 3, category: "Home"},
		{title: "Feed pets", hour: 7, minute: 0, duration: 5 * time.Minute, priority: 9, category: "Home"},
		{title: "Reply to urgent email", hour: 9, minute: 15, duration: 5 * time.Minute, priority: 8, category: "Work"},

		// Short tasks (10-20 min)
		{title: "Coffee break", hour: 10, minute: 30, duration: 15 * time.Minute, priority: 3, emotionID: emotionRelaxed},
		{title: "Quick walk around block", hour: 15, minute: 0, duration: 15 * time.Minute, priority: 4, category: "Health"},
		{title: "Organize desk", hour: 11, minute: 0, duration: 10 * time.Minute, priority: 2, category: "Work"},
		{title: "Review notes", hour: 16, minute: 0, duration: 15 * time.Minute, priority: 5, category: "Learning"},
		{title: "Snack break", hour: 16, minute: 30, duration: 10 * time.Minute, priority: 2},
		{title: "Check social media", hour: 13, minute: 0, duration: 10 * time.Minute, priority: 1, emotionID: emotionBored},

		// Medium tasks (30-60 min)
		{title: "Catch up on news", hour: 7, minute: 30, duration: 30 * time.Minute, priority: 3, category: "Personal"},
		{title: "Phone call with friend", hour: 18, minute: 0, duration: 30 * time.Minute, priority: 5, category: "Social", emotionID: emotionHappy},
		{title: "Light yoga", hour: 6, minute: 30, duration: 30 * time.Minute, priority: 6, category: "Health", emotionID: emotionCalm},
		{title: "Update task list", hour: 21, minute: 0, duration: 20 * time.Minute, priority: 5, category: "Personal"},
		{title: "Online shopping research", hour: 20, minute: 0, duration: 30 * time.Minute, priority: 3, category: "Errands"},

		// Not started / incomplete tasks (past but never completed)
		{title: "Clean out closet", hour: 14, minute: 0, duration: 90 * time.Minute, priority: 3, category: "Home", completed: false},
		{title: "Backup photos", hour: 15, minute: 0, duration: 45 * time.Minute, priority: 4, category: "Personal", completed: false},
		{title: "Update resume", hour: 19, minute: 0, duration: 60 * time.Minute, priority: 5, category: "Work", completed: false},
	}

	// Add some common tasks randomly
	numToAdd := 3 + rand.Intn(5) // 3-7 extra tasks
	for i := 0; i < numToAdd && i < len(commonTasks); i++ {
		task := commonTasks[rand.Intn(len(commonTasks))]
		// Randomize completion unless explicitly set
		if task.completed == false && isPast {
			task.completed = shouldComplete(isPast, completionRate)
		}
		tasks = append(tasks, task)
	}

	// Day-specific extra tasks
	switch dayOffset {
	case -1: // Yesterday - add incomplete tasks
		tasks = append(tasks,
			taskDef{
				title:   "Evening Movie Night",
				journal: "<p>Watched a great documentary about space exploration</p>",
				hour:    20, minute: 0,
				duration:  120 * time.Minute,
				priority:  3,
				completed: true,
				category:  "Personal",
				emotionID: emotionRelaxed,
				positives: []taskItem{{text: "Quality relaxation time", emotionID: emotionContent}},
			},
			taskDef{
				title:   "Finish project report",
				journal: "<p>Still working on the quarterly report</p>",
				hour:    14, minute: 0,
				duration:  3 * time.Hour,
				priority:  9,
				completed: false, // Incomplete!
				category:  "Work",
				emotionID: emotionStressed,
				negatives: []taskItem{{text: "Too much to do", emotionID: emotionOverwhelmed}},
			},
			taskDef{
				title: "Call dentist for appointment",
				hour:  10, minute: 0,
				duration:  5 * time.Minute,
				priority:  6,
				completed: false, // Forgot to do
				category:  "Health",
			},
			taskDef{
				title: "Pay credit card bill",
				hour:  12, minute: 0,
				duration:  5 * time.Minute,
				priority:  10,
				completed: false, // Important but missed
				category:  "Finance",
			},
		)

	case -2: // 2 days ago
		tasks = append(tasks,
			taskDef{
				title:   "Grocery shopping",
				journal: "<p>Weekly groceries from the supermarket</p>",
				hour:    17, minute: 0,
				duration:  1 * time.Hour,
				priority:  7,
				completed: true,
				category:  "Errands",
				emotionID: emotionSatisfied,
			},
			taskDef{
				title:   "Watch tutorial videos",
				journal: "<p>Learning React hooks</p>",
				hour:    20, minute: 0,
				duration:  90 * time.Minute,
				priority:  5,
				completed: false, // Didn't finish
				category:  "Learning",
			},
			taskDef{
				title: "Clean bathroom",
				hour:  11, minute: 0,
				duration:  45 * time.Minute,
				priority:  4,
				completed: false, // Skipped
				category:  "Home",
			},
		)

	case -3: // 3 days ago
		tasks = append(tasks,
			taskDef{
				title:   "Doctor's Appointment",
				journal: "<p>Annual checkup - everything looks good!</p>",
				hour:    10, minute: 30,
				duration:  45 * time.Minute,
				priority:  9,
				completed: true,
				category:  "Health",
				emotionID: emotionRelieved,
				positives: []taskItem{{text: "Good health news", emotionID: emotionGrateful}},
			},
			taskDef{
				title: "Submit expense report",
				hour:  15, minute: 0,
				duration:  30 * time.Minute,
				priority:  8,
				completed: false, // Forgot to submit
				category:  "Work",
			},
		)

	case -4: // 4 days ago
		tasks = append(tasks,
			taskDef{
				title:   "Long bike ride",
				journal: "<p>20km ride through the countryside</p>",
				hour:    7, minute: 0,
				duration:  2 * time.Hour,
				priority:  6,
				completed: true,
				category:  "Health",
				emotionID: emotionExcited,
				positives: []taskItem{
					{text: "Beautiful weather", emotionID: emotionJoyful},
					{text: "Great exercise", emotionID: emotionMotivated},
				},
			},
			taskDef{
				title: "Reply to parent's email",
				hour:  19, minute: 0,
				duration:  20 * time.Minute,
				priority:  5,
				completed: false,
				category:  "Family",
			},
		)

	case -5: // 5 days ago
		tasks = append(tasks,
			taskDef{
				title:   "Team Building Event",
				journal: "<p>Great day with colleagues, bowling and dinner</p>",
				hour:    14, minute: 0,
				duration:  4 * time.Hour,
				priority:  6,
				completed: true,
				category:  "Social",
				emotionID: emotionJoyful,
				positives: []taskItem{
					{text: "Fun time with colleagues", emotionID: emotionJoyful},
					{text: "Good conversations", emotionID: emotionContent},
				},
			},
			taskDef{
				title: "Research vacation destinations",
				hour:  20, minute: 0,
				duration:  1 * time.Hour,
				priority:  3,
				completed: shouldComplete(true, 0.5),
				category:  "Travel",
				emotionID: emotionCurious,
			},
		)

	case -6: // 6 days ago
		tasks = append(tasks,
			taskDef{
				title:   "Clean garage",
				journal: "<p>Finally organized the cluttered garage</p>",
				hour:    10, minute: 0,
				duration:  3 * time.Hour,
				priority:  4,
				completed: true,
				category:  "Home",
				emotionID: emotionAccomplished,
				positives: []taskItem{{text: "So much more space now!", emotionID: emotionSatisfied}},
				negatives: []taskItem{{text: "Found some broken items", emotionID: emotionDisappointed}},
			},
			taskDef{
				title: "Renew library books",
				hour:  11, minute: 0,
				duration:  5 * time.Minute,
				priority:  5,
				completed: false, // Forgot
				category:  "Errands",
			},
		)

	case -7: // Week ago
		tasks = append(tasks,
			taskDef{
				title:   "Weekly Planning Session",
				journal: "<p>Set goals for the week and prioritized tasks</p>",
				hour:    9, minute: 0,
				duration:  1 * time.Hour,
				priority:  8,
				completed: true,
				category:  "Personal",
				emotionID: emotionMotivated,
				positives: []taskItem{{text: "Clear plan for the week", emotionID: emotionConfident}},
			},
			taskDef{
				title: "Deep clean kitchen",
				hour:  14, minute: 0,
				duration:  2 * time.Hour,
				priority:  5,
				completed: shouldComplete(true, 0.6),
				category:  "Home",
			},
			taskDef{
				title: "Meal prep for the week",
				hour:  16, minute: 0,
				duration:  90 * time.Minute,
				priority:  6,
				completed: true,
				category:  "Home",
				emotionID: emotionSatisfied,
			},
		)
	}

	return tasks
}

func generateTodayTasks(day time.Time, isWeekend bool) []taskDef {
	var tasks []taskDef
	currentHour := time.Now().Hour()

	if isWeekend {
		tasks = generateWeekendTasks(false, 0.3)
	} else {
		tasks = generateWorkdayTasks(false, 0.4)
	}

	// Mark morning tasks as completed if it's afternoon
	for i := range tasks {
		if tasks[i].hour < currentHour-1 {
			tasks[i].completed = true
		}
	}

	// Add current time specific tasks
	if currentHour >= 12 {
		tasks = append(tasks, taskDef{
			title:   "Afternoon Coffee Break",
			journal: "<p>Quick break to recharge</p>",
			hour:    14, minute: 30,
			duration:  15 * time.Minute,
			priority:  3,
			completed: currentHour >= 15,
			category:  "Personal",
			emotionID: emotionRelaxed,
		})
	}

	if currentHour >= 18 {
		tasks = append(tasks, taskDef{
			title:   "Evening Review",
			journal: "<p>Review what was accomplished today and plan for tomorrow</p>",
			hour:    21, minute: 0,
			duration:  20 * time.Minute,
			priority:  5,
			completed: false,
			category:  "Personal",
		})
	}

	return tasks
}

func generateFutureTasks(day time.Time, dayOffset int, isWeekend bool) []taskDef {
	var tasks []taskDef

	if isWeekend {
		tasks = generateWeekendTasks(false, 0)
	} else {
		tasks = generateWorkdayTasks(false, 0)
	}

	// Add common future micro-tasks
	futureMicroTasks := []taskDef{
		{title: "Prepare meeting agenda", hour: 8, minute: 30, duration: 10 * time.Minute, priority: 6, category: "Work"},
		{title: "Set reminder for bills", hour: 9, minute: 0, duration: 2 * time.Minute, priority: 8, category: "Finance"},
		{title: "Pick up dry cleaning", hour: 17, minute: 30, duration: 15 * time.Minute, priority: 4, category: "Errands"},
		{title: "Call insurance company", hour: 10, minute: 0, duration: 20 * time.Minute, priority: 7, category: "Finance"},
		{title: "Buy birthday gift", hour: 13, minute: 0, duration: 45 * time.Minute, priority: 6, category: "Errands"},
		{title: "Schedule car service", hour: 11, minute: 0, duration: 10 * time.Minute, priority: 5},
	}

	// Add 2-3 random micro-tasks
	numMicro := 2 + rand.Intn(2)
	for i := 0; i < numMicro && i < len(futureMicroTasks); i++ {
		tasks = append(tasks, futureMicroTasks[rand.Intn(len(futureMicroTasks))])
	}

	// Add future-specific events
	switch dayOffset {
	case 1: // Tomorrow
		tasks = append(tasks,
			taskDef{
				title:   "Dentist Appointment",
				journal: "<p>Regular dental checkup</p>",
				hour:    11, minute: 0,
				duration: 1 * time.Hour,
				priority: 8,
				category: "Health",
			},
			taskDef{
				title: "Prepare presentation slides",
				hour:  14, minute: 0,
				duration: 2 * time.Hour,
				priority: 9,
				category: "Work",
			},
			taskDef{
				title:   "Team dinner",
				journal: "<p>Quarterly team celebration</p>",
				hour:    19, minute: 0,
				duration: 2 * time.Hour,
				priority: 5,
				category: "Social",
			},
			taskDef{
				title: "Quick grocery run",
				hour:  17, minute: 0,
				duration: 20 * time.Minute,
				priority: 6,
				category: "Errands",
			},
			taskDef{
				title: "Reply to HR email",
				hour:  9, minute: 30,
				duration: 10 * time.Minute,
				priority: 7,
				category: "Work",
			},
		)

	case 2: // 2 days from now
		tasks = append(tasks,
			taskDef{
				title:   "Project Deadline",
				journal: "<p>Submit final deliverables for the project</p>",
				hour:    17, minute: 0,
				duration: 2 * time.Hour,
				priority: 10,
				category: "Work",
			},
			taskDef{
				title: "Quarterly review meeting",
				hour:  10, minute: 0,
				duration: 90 * time.Minute,
				priority: 9,
				category: "Work",
			},
			taskDef{
				title: "Lunch with client",
				hour:  12, minute: 30,
				duration: 90 * time.Minute,
				priority: 8,
				category: "Work",
			},
			taskDef{
				title: "Book flight tickets",
				hour:  20, minute: 0,
				duration: 30 * time.Minute,
				priority: 6,
				category: "Travel",
			},
			taskDef{
				title: "Water all plants",
				hour:  7, minute: 30,
				duration: 10 * time.Minute,
				priority: 3,
				category: "Home",
			},
			taskDef{
				title: "Evening run",
				hour:  18, minute: 0,
				duration: 45 * time.Minute,
				priority: 5,
				category: "Health",
			},
		)

	case 3: // 3 days from now
		tasks = append(tasks,
			taskDef{
				title:   "Coffee with Mentor",
				journal: "<p>Quarterly catch-up and career discussion</p>",
				hour:    15, minute: 0,
				duration: 1 * time.Hour,
				priority: 7,
				category: "Social",
			},
			taskDef{
				title: "Code review for intern",
				hour:  10, minute: 0,
				duration: 1 * time.Hour,
				priority: 6,
				category: "Work",
			},
			taskDef{
				title: "Annual subscription renewal",
				hour:  9, minute: 0,
				duration: 5 * time.Minute,
				priority: 8,
				category: "Finance",
			},
			taskDef{
				title: "Clean home office",
				hour:  14, minute: 0,
				duration: 90 * time.Minute,
				priority: 4,
				category: "Home",
			},
			taskDef{
				title:   "Online course: Advanced TypeScript",
				journal: "<p>Chapter 5: Generics and Type Guards</p>",
				hour:    20, minute: 0,
				duration: 2 * time.Hour,
				priority: 5,
				category: "Learning",
			},
			taskDef{
				title: "Meal planning",
				hour:  11, minute: 0,
				duration: 20 * time.Minute,
				priority: 4,
				category: "Home",
			},
		)
	}

	return tasks
}

func generateWorkdayTasks(isPast bool, completionRate float64) []taskDef {
	tasks := []taskDef{
		// Morning routine
		{
			title:   "Morning Meditation",
			journal: "<p>Started the day with 15 minutes of mindfulness</p>",
			hour:    6, minute: 30,
			duration:  15 * time.Minute,
			priority:  6,
			completed: shouldComplete(isPast, completionRate),
			category:  "Meditation",
			emotionID: emotionPeaceful,
			positives: []taskItem{{text: "Centered and calm", emotionID: emotionCalm}},
		},
		{
			title:   "Morning Workout",
			journal: "<p>30 minute cardio session</p>",
			hour:    7, minute: 0,
			duration:  30 * time.Minute,
			priority:  7,
			completed: shouldComplete(isPast, completionRate),
			category:  "Health",
			emotionID: emotionMotivated,
		},
		{
			title:   "Morning Standup",
			journal: "<p>Daily team sync meeting</p>",
			hour:    9, minute: 0,
			duration:  15 * time.Minute,
			priority:  8,
			completed: shouldComplete(isPast, completionRate),
			category:  "Work",
		},
		{
			title:   "Check Emails & Messages",
			journal: "<p>Process inbox and respond to urgent items</p>",
			hour:    9, minute: 30,
			duration:  30 * time.Minute,
			priority:  7,
			completed: shouldComplete(isPast, completionRate),
			category:  "Work",
		},
		// Deep work block
		{
			title:   "Deep Work: Feature Development",
			journal: "<p>Focused coding session on the main feature</p>",
			hour:    10, minute: 0,
			duration:  2 * time.Hour,
			priority:  10,
			completed: shouldComplete(isPast, completionRate),
			category:  "Work",
			emotionID: emotionConfident,
			positives: []taskItem{
				{text: "Made good progress", emotionID: emotionAccomplished},
				{text: "Solved complex problem", emotionID: emotionProud},
			},
			note: "Completed the authentication module",
		},
		// Lunch
		{
			title:   "Lunch Break",
			journal: "<p>Healthy lunch and short walk</p>",
			hour:    12, minute: 30,
			duration:  1 * time.Hour,
			priority:  5,
			completed: shouldComplete(isPast, completionRate),
			category:  "Personal",
			emotionID: emotionContent,
		},
		// Afternoon work
		{
			title:   "Code Review Session",
			journal: "<p>Review pending pull requests from the team</p>",
			hour:    14, minute: 0,
			duration:  1 * time.Hour,
			priority:  7,
			completed: shouldComplete(isPast, completionRate),
			category:  "Work",
		},
		{
			title:   "Bug Fix: Search Functionality",
			journal: "<p>Debug and fix the search API issue</p>",
			hour:    15, minute: 0,
			duration:  90 * time.Minute,
			priority:  8,
			completed: shouldComplete(isPast, completionRate),
			category:  "Work",
			emotionID: emotionFrustrated,
			negatives: []taskItem{{text: "Spent too long debugging", emotionID: emotionFrustrated}},
			positives: []taskItem{{text: "Finally found the root cause", emotionID: emotionRelieved}},
		},
		{
			title:   "Team Planning Meeting",
			journal: "<p>Sprint planning for next week</p>",
			hour:    16, minute: 30,
			duration:  45 * time.Minute,
			priority:  7,
			completed: shouldComplete(isPast, completionRate),
			category:  "Work",
		},
		// Evening
		{
			title:   "Evening Walk",
			journal: "<p>30 minute walk around the neighborhood</p>",
			hour:    18, minute: 0,
			duration:  30 * time.Minute,
			priority:  5,
			completed: shouldComplete(isPast, completionRate),
			category:  "Health",
			emotionID: emotionRelaxed,
		},
		{
			title:   "Dinner & Family Time",
			journal: "<p>Cooking and eating together</p>",
			hour:    19, minute: 0,
			duration:  1 * time.Hour,
			priority:  6,
			completed: shouldComplete(isPast, completionRate),
			category:  "Family",
			emotionID: emotionGrateful,
		},
		{
			title:   "Reading",
			journal: "<p>Reading 'Atomic Habits' - Chapter on habit stacking</p>",
			hour:    21, minute: 0,
			duration:  30 * time.Minute,
			priority:  4,
			completed: shouldComplete(isPast, completionRate),
			category:  "Reading",
			emotionID: emotionInspired,
			positives: []taskItem{{text: "Learned new productivity techniques", emotionID: emotionCurious}},
		},
		{
			title:   "Evening Journaling",
			journal: "<p>Reflecting on the day's achievements and challenges</p>",
			hour:    21, minute: 30,
			duration:  15 * time.Minute,
			priority:  4,
			completed: shouldComplete(isPast, completionRate),
			category:  "Personal",
			emotionID: emotionContent,
		},
	}

	// Add some variety - tasks without categories or emotions
	tasks = append(tasks, taskDef{
		title:   "Quick Phone Call",
		journal: "<p>Brief call with a friend</p>",
		hour:    17, minute: 30,
		duration:  10 * time.Minute,
		priority:  3,
		completed: shouldComplete(isPast, completionRate),
		// No category - testing uncategorized
	})

	tasks = append(tasks, taskDef{
		title:   "Water Plants",
		journal: "", // No journal - testing empty
		hour:    8, minute: 0,
		duration:  10 * time.Minute,
		priority:  2,
		completed: shouldComplete(isPast, completionRate),
		category:  "Home",
		// No emotion - testing without emotion
	})

	// Very short task
	tasks = append(tasks, taskDef{
		title: "Take Vitamins",
		hour:  7, minute: 45,
		duration:  2 * time.Minute,
		priority:  4,
		completed: shouldComplete(isPast, completionRate),
		category:  "Health",
	})

	// Very long task
	tasks = append(tasks, taskDef{
		title:   "Online Course: Go Advanced Patterns",
		journal: "<p>Completed modules on concurrency patterns and generics</p>",
		hour:    20, minute: 0,
		duration:  3 * time.Hour,
		priority:  6,
		completed: shouldComplete(isPast, completionRate),
		category:  "Learning",
		emotionID: emotionEnthusiastic,
		positives: []taskItem{
			{text: "Understanding complex concepts", emotionID: emotionProud},
			{text: "Applied learning to current project", emotionID: emotionAccomplished},
		},
	})

	// ==========================================================================
	// SLEEP TASKS (very long - 6-8 hours)
	// ==========================================================================
	tasks = append(tasks, taskDef{
		title:   "Night Sleep",
		journal: "<p>Good night's rest</p>",
		hour:    23, minute: 0,
		duration:  7 * time.Hour, // Ends at 6am next day
		priority:  8,
		completed: shouldComplete(isPast, completionRate),
		category:  "Health",
		emotionID: emotionPeaceful,
	})

	// Nap
	if rand.Float64() < 0.3 { // 30% chance of nap
		tasks = append(tasks, taskDef{
			title: "Power Nap",
			hour:  13, minute: 30,
			duration:  20 * time.Minute,
			priority:  4,
			completed: shouldComplete(isPast, completionRate),
			category:  "Health",
			emotionID: emotionRelaxed,
		})
	}

	// ==========================================================================
	// MICRO TASKS (seconds to 2 minutes)
	// ==========================================================================
	microTasks := []taskDef{
		{title: "Drink water", hour: 8, minute: 30, duration: 10 * time.Second, priority: 6, category: "Health"},
		{title: "Drink water", hour: 10, minute: 0, duration: 10 * time.Second, priority: 6, category: "Health"},
		{title: "Drink water", hour: 12, minute: 0, duration: 10 * time.Second, priority: 6, category: "Health"},
		{title: "Drink water", hour: 14, minute: 30, duration: 10 * time.Second, priority: 6, category: "Health"},
		{title: "Drink water", hour: 16, minute: 0, duration: 10 * time.Second, priority: 6, category: "Health"},
		{title: "Drink water", hour: 18, minute: 30, duration: 10 * time.Second, priority: 6, category: "Health"},
		{title: "Quick stretch at desk", hour: 11, minute: 0, duration: 30 * time.Second, priority: 3},
		{title: "Eye rest - look away from screen", hour: 10, minute: 30, duration: 20 * time.Second, priority: 4, category: "Health"},
		{title: "Eye rest - look away from screen", hour: 14, minute: 0, duration: 20 * time.Second, priority: 4, category: "Health"},
		{title: "Take a deep breath", hour: 15, minute: 30, duration: 15 * time.Second, priority: 2, category: "Health", emotionID: emotionCalm},
		{title: "Check phone notifications", hour: 9, minute: 45, duration: 30 * time.Second, priority: 2},
		{title: "Refill coffee", hour: 10, minute: 15, duration: 1 * time.Minute, priority: 2},
		{title: "Refill coffee", hour: 15, minute: 0, duration: 1 * time.Minute, priority: 2},
		{title: "Quick bathroom break", hour: 11, minute: 30, duration: 2 * time.Minute, priority: 5},
		{title: "Quick bathroom break", hour: 15, minute: 45, duration: 2 * time.Minute, priority: 5},
		{title: "Reply to Slack message", hour: 9, minute: 20, duration: 30 * time.Second, priority: 5, category: "Work"},
		{title: "Lock computer", hour: 18, minute: 0, duration: 5 * time.Second, priority: 7, category: "Work"},
	}

	// Add 5-8 random micro tasks
	numMicro := 5 + rand.Intn(4)
	for i := 0; i < numMicro && i < len(microTasks); i++ {
		task := microTasks[rand.Intn(len(microTasks))]
		task.completed = shouldComplete(isPast, completionRate)
		tasks = append(tasks, task)
	}

	// ==========================================================================
	// OVERLAPPING TASKS (to test timeline overlap handling)
	// ==========================================================================
	// These tasks intentionally overlap to stress-test the UI
	overlappingTasks := []taskDef{
		// 10:00-12:00 block with overlaps
		{
			title:   "Background music while working",
			journal: "<p>Lo-fi beats for focus</p>",
			hour:    10, minute: 0,
			duration:  2 * time.Hour,
			priority:  1,
			completed: shouldComplete(isPast, completionRate),
		},
		// 14:00-15:30 block with overlaps
		{
			title: "Podcast while doing chores",
			hour:  14, minute: 0,
			duration:  90 * time.Minute,
			priority:  2,
			completed: shouldComplete(isPast, completionRate),
			category:  "Learning",
		},
		// Conference call while taking notes (same time)
		{
			title: "Take notes during meeting",
			hour:  9, minute: 0, // Same time as standup
			duration:  15 * time.Minute,
			priority:  6,
			completed: shouldComplete(isPast, completionRate),
			category:  "Work",
		},
	}

	// Add 1-2 overlapping tasks
	if rand.Float64() < 0.7 {
		tasks = append(tasks, overlappingTasks[rand.Intn(len(overlappingTasks))])
	}
	if rand.Float64() < 0.5 {
		tasks = append(tasks, overlappingTasks[rand.Intn(len(overlappingTasks))])
	}

	// ==========================================================================
	// CONCURRENT/PARALLEL TASKS (things done together)
	// ==========================================================================
	concurrentTasks := []taskDef{
		{
			title:   "Listen to audiobook during commute",
			journal: "<p>Continuing 'The Psychology of Money'</p>",
			hour:    8, minute: 30,
			duration:  30 * time.Minute,
			priority:  4,
			completed: shouldComplete(isPast, completionRate),
			category:  "Learning",
			emotionID: emotionCurious,
		},
		{
			title: "Morning commute",
			hour:  8, minute: 30, // Same time as audiobook
			duration:  30 * time.Minute,
			priority:  5,
			completed: shouldComplete(isPast, completionRate),
			category:  "Personal",
		},
		{
			title: "Watch tutorial while eating lunch",
			hour:  12, minute: 30, // Same time as lunch break
			duration:  30 * time.Minute,
			priority:  3,
			completed: shouldComplete(isPast, completionRate),
			category:  "Learning",
		},
		{
			title: "Listen to music while cleaning",
			hour:  19, minute: 30,
			duration:  20 * time.Minute,
			priority:  2,
			completed: shouldComplete(isPast, completionRate),
		},
		{
			title: "Tidy up desk",
			hour:  19, minute: 30, // Same time as music
			duration:  20 * time.Minute,
			priority:  3,
			completed: shouldComplete(isPast, completionRate),
			category:  "Home",
		},
		{
			title:   "Call with friend while walking",
			journal: "<p>Catching up while getting steps in</p>",
			hour:    18, minute: 0, // Same time as evening walk
			duration:  25 * time.Minute,
			priority:  5,
			completed: shouldComplete(isPast, completionRate),
			category:  "Social",
			emotionID: emotionHappy,
		},
	}

	// Add 2-4 concurrent task pairs
	numConcurrent := 2 + rand.Intn(3)
	for i := 0; i < numConcurrent && i < len(concurrentTasks); i++ {
		tasks = append(tasks, concurrentTasks[i])
	}

	return tasks
}

func generateWeekendTasks(isPast bool, completionRate float64) []taskDef {
	tasks := []taskDef{
		// Morning routine
		{
			title:   "Sleep In & Lazy Morning",
			journal: "<p>No alarm, woke up naturally</p>",
			hour:    8, minute: 30,
			duration:  30 * time.Minute,
			priority:  3,
			completed: shouldComplete(isPast, completionRate),
			emotionID: emotionRelaxed,
		},
		{
			title:   "Weekend Meditation",
			journal: "<p>Extended 30-minute morning meditation</p>",
			hour:    9, minute: 0,
			duration:  30 * time.Minute,
			priority:  5,
			completed: shouldComplete(isPast, completionRate),
			category:  "Meditation",
			emotionID: emotionPeaceful,
		},
		{
			title:   "Big Breakfast",
			journal: "<p>Pancakes and fresh fruit</p>",
			hour:    9, minute: 30,
			duration:  45 * time.Minute,
			priority:  4,
			completed: shouldComplete(isPast, completionRate),
			category:  "Personal",
			emotionID: emotionContent,
		},
		{
			title:   "Long Run",
			journal: "<p>10km run through the park</p>",
			hour:    10, minute: 30,
			duration:  1 * time.Hour,
			priority:  7,
			completed: shouldComplete(isPast, completionRate),
			category:  "Health",
			emotionID: emotionMotivated,
			positives: []taskItem{
				{text: "Beat personal best pace", emotionID: emotionProud},
				{text: "Beautiful weather", emotionID: emotionJoyful},
			},
		},
		// Midday
		{
			title:   "Grocery Shopping",
			journal: "<p>Weekly groceries at farmer's market</p>",
			hour:    12, minute: 0,
			duration:  1 * time.Hour,
			priority:  6,
			completed: shouldComplete(isPast, completionRate),
			category:  "Errands",
		},
		{
			title:   "Lunch with Friends",
			journal: "<p>Caught up with old friends at favorite restaurant</p>",
			hour:    13, minute: 0,
			duration:  2 * time.Hour,
			priority:  7,
			completed: shouldComplete(isPast, completionRate),
			category:  "Social",
			emotionID: emotionJoyful,
			positives: []taskItem{
				{text: "Great conversation", emotionID: emotionHappy},
				{text: "Delicious food", emotionID: emotionSatisfied},
			},
		},
		// Afternoon
		{
			title:   "House Cleaning",
			journal: "<p>Deep cleaning of living room and kitchen</p>",
			hour:    15, minute: 0,
			duration:  90 * time.Minute,
			priority:  6,
			completed: shouldComplete(isPast, completionRate),
			category:  "Home",
			positives: []taskItem{{text: "Clean and organized space", emotionID: emotionSatisfied}},
		},
		{
			title:   "Creative Project",
			journal: "<p>Working on personal side project - building a CLI tool</p>",
			hour:    17, minute: 0,
			duration:  2 * time.Hour,
			priority:  5,
			completed: shouldComplete(isPast, completionRate),
			category:  "Creative",
			emotionID: emotionInspired,
			positives: []taskItem{
				{text: "Making progress on passion project", emotionID: emotionExcited},
			},
		},
		// Evening
		{
			title:   "Video Call with Parents",
			journal: "<p>Weekly check-in with family</p>",
			hour:    19, minute: 0,
			duration:  45 * time.Minute,
			priority:  7,
			completed: shouldComplete(isPast, completionRate),
			category:  "Family",
			emotionID: emotionGrateful,
		},
		{
			title:   "Movie Night",
			journal: "<p>Relaxing with a good film</p>",
			hour:    20, minute: 0,
			duration:  2 * time.Hour,
			priority:  3,
			completed: shouldComplete(isPast, completionRate),
			category:  "Personal",
			emotionID: emotionRelaxed,
		},
		{
			title:   "Weekly Review & Planning",
			journal: "<p>Reviewing the week and planning ahead</p>",
			hour:    22, minute: 0,
			duration:  45 * time.Minute,
			priority:  8,
			completed: shouldComplete(isPast, completionRate),
			category:  "Personal",
			emotionID: emotionMotivated,
			positives: []taskItem{
				{text: "Clear vision for next week", emotionID: emotionConfident},
			},
		},
	}

	// Add some tasks without emotions or categories for variety
	tasks = append(tasks, taskDef{
		title: "Quick Errand - Post Office",
		hour:  11, minute: 30,
		duration:  20 * time.Minute,
		priority:  5,
		completed: shouldComplete(isPast, completionRate),
		category:  "Errands",
	})

	// Task with only negatives (to test edge case)
	tasks = append(tasks, taskDef{
		title:   "Fix Leaky Faucet",
		journal: "<p>Finally fixed the kitchen faucet that's been dripping</p>",
		hour:    14, minute: 0,
		duration:  1 * time.Hour,
		priority:  6,
		completed: shouldComplete(isPast, completionRate),
		category:  "Home",
		emotionID: emotionFrustrated, // Started frustrated
		negatives: []taskItem{
			{text: "Took longer than expected", emotionID: emotionFrustrated},
			{text: "Had to make extra trip to hardware store", emotionID: emotionImpatient},
		},
		positives: []taskItem{
			{text: "Learned a new skill", emotionID: emotionProud},
			{text: "Saved money on plumber", emotionID: emotionSatisfied},
		},
		note: "Needed a 3/8 inch washer, but had 1/2 inch. Second trip got the right one.",
	})

	// Task with many items (stress testing)
	tasks = append(tasks, taskDef{
		title:   "Day Reflection",
		journal: "<p>Comprehensive review of the day</p>",
		hour:    22, minute: 30,
		duration:  20 * time.Minute,
		priority:  4,
		completed: shouldComplete(isPast, completionRate),
		category:  "Personal",
		emotionID: emotionGrateful,
		positives: []taskItem{
			{text: "Spent quality time with family"},
			{text: "Got good exercise"},
			{text: "Made progress on projects", emotionID: emotionAccomplished},
			{text: "Enjoyed good food", emotionID: emotionContent},
			{text: "Felt connected and social", emotionID: emotionHappy},
		},
		negatives: []taskItem{
			{text: "Didn't get to all planned tasks"},
			{text: "Spent too much time on phone in morning", emotionID: emotionDisappointed},
		},
	})

	// ==========================================================================
	// SLEEP TASKS (very long - weekend sleep in!)
	// ==========================================================================
	tasks = append(tasks, taskDef{
		title:   "Weekend Sleep In",
		journal: "<p>Catching up on rest</p>",
		hour:    23, minute: 30,
		duration:  9 * time.Hour, // Weekend sleep - longer!
		priority:  7,
		completed: shouldComplete(isPast, completionRate),
		category:  "Health",
		emotionID: emotionPeaceful,
	})

	// Afternoon nap
	if rand.Float64() < 0.5 { // 50% chance of weekend nap
		tasks = append(tasks, taskDef{
			title:   "Afternoon Nap",
			journal: "<p>Weekend rest</p>",
			hour:    15, minute: 0,
			duration:  1 * time.Hour,
			priority:  3,
			completed: shouldComplete(isPast, completionRate),
			category:  "Health",
			emotionID: emotionRelaxed,
		})
	}

	// ==========================================================================
	// MICRO TASKS (seconds to 2 minutes)
	// ==========================================================================
	weekendMicroTasks := []taskDef{
		{title: "Drink water", hour: 9, minute: 0, duration: 10 * time.Second, priority: 6, category: "Health"},
		{title: "Drink water", hour: 11, minute: 0, duration: 10 * time.Second, priority: 6, category: "Health"},
		{title: "Drink water", hour: 14, minute: 0, duration: 10 * time.Second, priority: 6, category: "Health"},
		{title: "Drink water", hour: 17, minute: 0, duration: 10 * time.Second, priority: 6, category: "Health"},
		{title: "Drink water", hour: 20, minute: 0, duration: 10 * time.Second, priority: 6, category: "Health"},
		{title: "Quick stretch", hour: 10, minute: 30, duration: 1 * time.Minute, priority: 3, category: "Health"},
		{title: "Take vitamins", hour: 9, minute: 15, duration: 30 * time.Second, priority: 5, category: "Health"},
		{title: "Feed the cat", hour: 8, minute: 0, duration: 2 * time.Minute, priority: 9, category: "Home"},
		{title: "Feed the cat", hour: 18, minute: 0, duration: 2 * time.Minute, priority: 9, category: "Home"},
		{title: "Check mailbox", hour: 11, minute: 0, duration: 1 * time.Minute, priority: 3},
		{title: "Water garden", hour: 9, minute: 30, duration: 5 * time.Minute, priority: 4, category: "Home"},
		{title: "Take out trash", hour: 10, minute: 0, duration: 2 * time.Minute, priority: 6, category: "Home"},
	}

	// Add 4-7 random micro tasks
	numMicro := 4 + rand.Intn(4)
	for i := 0; i < numMicro && i < len(weekendMicroTasks); i++ {
		task := weekendMicroTasks[rand.Intn(len(weekendMicroTasks))]
		task.completed = shouldComplete(isPast, completionRate)
		tasks = append(tasks, task)
	}

	// ==========================================================================
	// OVERLAPPING/CONCURRENT WEEKEND TASKS
	// ==========================================================================
	concurrentWeekendTasks := []taskDef{
		{
			title: "Listen to podcast while cooking",
			hour:  12, minute: 0,
			duration:  45 * time.Minute,
			priority:  3,
			completed: shouldComplete(isPast, completionRate),
			category:  "Learning",
		},
		{
			title: "Make brunch",
			hour:  12, minute: 0, // Same time as podcast
			duration:  30 * time.Minute,
			priority:  5,
			completed: shouldComplete(isPast, completionRate),
			category:  "Home",
		},
		{
			title: "Music while cleaning",
			hour:  15, minute: 0, // Same time as house cleaning
			duration:  90 * time.Minute,
			priority:  1,
			completed: shouldComplete(isPast, completionRate),
		},
		{
			title: "TV in background while folding laundry",
			hour:  16, minute: 0,
			duration:  45 * time.Minute,
			priority:  2,
			completed: shouldComplete(isPast, completionRate),
		},
		{
			title: "Fold laundry",
			hour:  16, minute: 0, // Same time as TV
			duration:  30 * time.Minute,
			priority:  4,
			completed: shouldComplete(isPast, completionRate),
			category:  "Home",
		},
		{
			title: "Call sibling while walking dog",
			hour:  17, minute: 30,
			duration:  30 * time.Minute,
			priority:  5,
			completed: shouldComplete(isPast, completionRate),
			category:  "Family",
			emotionID: emotionHappy,
		},
		{
			title: "Walk the dog",
			hour:  17, minute: 30, // Same time as call
			duration:  30 * time.Minute,
			priority:  7,
			completed: shouldComplete(isPast, completionRate),
			category:  "Home",
		},
	}

	// Add 2-4 concurrent tasks
	numConcurrent := 2 + rand.Intn(3)
	for i := 0; i < numConcurrent && i < len(concurrentWeekendTasks); i++ {
		tasks = append(tasks, concurrentWeekendTasks[i])
	}

	return tasks
}

// shouldComplete determines if a task should be completed based on random chance.
// Uses the completion rate probability for past tasks.
func shouldComplete(isPast bool, rate float64) bool {
	if !isPast {
		return false
	}
	// Use random chance based on completion rate
	return rand.Float64() < rate
}
