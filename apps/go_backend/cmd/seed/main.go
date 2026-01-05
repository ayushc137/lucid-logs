// Package main provides a comprehensive data seeder for development and testing.
//
// This CLI tool populates the database with sample data across 30 days
// demonstrating all features of the simplified data model:
//
// Features:
//   - Units: System units seeding
//   - Goals: Measurable, habits, grouped, avoidance (via operators)
//   - Templates: Quick-log templates linked to goals via template_goals
//   - Tasks: With quantities, goal links, emotions, reflections
//   - Relations: in_category, task_goals, template_goals, goal_children
//   - Real-world scenarios: Hydration streak, running progress, project milestones
//
// Usage:
//
//	go run ./cmd/seed
//	go run ./cmd/seed --reset  # Delete existing data first
//
//nolint:gosec // G404: math/rand is acceptable for seeding test data
//nolint:gocritic // exitAfterDefer: we explicitly close resources before log.Fatal
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand" //nolint:gosec // G404: weak random is fine for test data
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

	log.Info().Msg("🌱 Starting data seeder (new data model)...")

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
	authToken, err = authenticate(ctx, cfg.Admin.Username, cfg.Admin.Password)
	if err != nil {
		db.Close(ctx) //nolint:errcheck // cleanup before exit
		//nolint:gocritic // exitAfterDefer: we explicitly close db above
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

	// Seed system units first
	if err := seedUnits(ctx, db); err != nil {
		log.Warn().Err(err).Msg("Failed to seed units")
	}

	// Create comprehensive seed data
	if err := seedAll(ctx, db, userID); err != nil {
		log.Fatal().Err(err).Msg("Failed to seed data")
	}

	log.Info().Msg("🎉 Data seeding complete!")
}

// =============================================================================
// AUTHENTICATION
// =============================================================================

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Data struct {
		Token   string `json:"token"`
		User    string `json:"user"`
		IsAdmin bool   `json:"is_admin"`
	} `json:"data"`
}

func authenticate(ctx context.Context, email, password string) (string, error) {
	payload, err := json.Marshal(loginRequest{Username: email, Password: password})
	if err != nil {
		return "", fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+"/auth/login", bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("login failed with status %d and could not read body", resp.StatusCode)
		}
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

	req, err := http.NewRequestWithContext(context.Background(), method, apiBaseURL+path, reqBody)
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
			return nil, fmt.Errorf("failed to parse response: %s", string(respBody))
		}
	}

	// Unwrap the "data" field if present
	if data, ok := result["data"].(map[string]any); ok {
		return data, nil
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
	// Delete all relations and entities in correct order
	tables := []string{
		"task_goals", "task_emotions", "template_goals", "created_from",
		"in_category", "goal_children", "goal_logs", "goal_snapshots",
		"tasks", "templates", "goals", "categories",
	}

	for _, table := range tables {
		//nolint:errcheck // ignore delete errors in reset
		_, _ = database.QueryAll[any](ctx, db, fmt.Sprintf(`
			DELETE %s WHERE created_by = $user OR true
		`, table), map[string]any{"user": userID})
	}

	return nil
}

func seedUnits(ctx context.Context, db *database.DB) error {
	now := time.Now().UTC()
	units := []struct {
		id, name, symbol, unitType string
	}{
		{"units:km", "kilometers", "km", "distance"},
		{"units:mi", "miles", "mi", "distance"},
		{"units:m", "meters", "m", "distance"},
		{"units:steps", "steps", "steps", "distance"},
		{"units:min", "minutes", "min", "time"},
		{"units:hr", "hours", "hr", "time"},
		{"units:sec", "seconds", "sec", "time"},
		{"units:l", "liters", "L", "volume"},
		{"units:ml", "milliliters", "ml", "volume"},
		{"units:cups", "cups", "cups", "volume"},
		{"units:count", "count", "×", "count"},
		{"units:pages", "pages", "pg", "count"},
		{"units:reps", "repetitions", "reps", "count"},
		{"units:sets", "sets", "sets", "count"},
		{"units:cal", "calories", "cal", "count"},
		{"units:dollars", "dollars", "$", "count"},
	}

	for _, u := range units {
		//nolint:errcheck // ignore upsert errors for idempotent seeding
		_, _ = database.QueryAll[any](ctx, db, `
			UPSERT type::thing($id) CONTENT {
				name: $name,
				symbol: $symbol,
				type: $type,
				is_system: true,
				created_by: "",
				created_at: $now,
				updated_at: $now
			}
		`, map[string]any{
			"id":     u.id,
			"name":   u.name,
			"symbol": u.symbol,
			"type":   u.unitType,
			"now":    now,
		})
	}

	log.Info().Int("count", len(units)).Msg("✅ System units seeded")
	return nil
}

// =============================================================================
// SEED ORCHESTRATOR
// =============================================================================

func seedAll(ctx context.Context, db *database.DB, userID string) error {
	// 1. Create Categories
	categories, err := seedCategories()
	if err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}
	log.Info().Int("count", len(categories)).Msg("✅ Categories created")

	// 2. Create Goals (using new model)
	goals, err := seedGoals(categories)
	if err != nil {
		return fmt.Errorf("failed to seed goals: %w", err)
	}
	log.Info().Int("count", len(goals)).Msg("✅ Goals created")

	// 3. Create Templates (linked to goals via template_goals)
	templates, err := seedTemplates(categories, goals)
	if err != nil {
		return fmt.Errorf("failed to seed templates: %w", err)
	}
	log.Info().Int("count", len(templates)).Msg("✅ Templates created")

	// 4. Seed 60 days of tasks with realistic patterns
	totalTasks, totalLinks := seedTasksMultiDay(ctx, db, categories, goals, templates)
	log.Info().Int("tasks", totalTasks).Int("links", totalLinks).Msg("✅ Tasks and goal links created")

	return nil
}

// =============================================================================
// CATEGORY SEEDING
// =============================================================================

var categoryDefs = []struct {
	name  string
	color string
}{
	{"Work", "#3B82F6"},
	{"Health", "#10B981"},
	{"Learning", "#8B5CF6"},
	{"Personal", "#F59E0B"},
	{"Finance", "#06B6D4"},
	{"Projects", "#0EA5E9"},
}

func seedCategories() (map[string]string, error) {
	result := make(map[string]string)

	for _, cat := range categoryDefs {
		resp, err := apiRequest("POST", "/categories", map[string]any{
			"name":  cat.name,
			"color": cat.color,
		})
		if err != nil {
			log.Warn().Err(err).Str("category", cat.name).Msg("Failed to create category")
			continue
		}
		if id, ok := resp["id"].(string); ok {
			result[cat.name] = id
		}
	}

	return result, nil
}

// =============================================================================
// GOAL SEEDING (New Model)
// =============================================================================

type goalDef struct {
	title       string
	description string
	icon        string
	category    string
	priority    int
	target      map[string]any // {value, operator, unit_id, per_period}
	recurrence  map[string]any // {frequency, period, ...}
	deadline    *time.Time
}

func seedGoals(categories map[string]string) (map[string]string, error) {
	result := make(map[string]string)
	now := time.Now()
	endOfMonth := time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 0, now.Location())
	endOfYear := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, now.Location())

	goals := []goalDef{
		// Measurable habit: Hydration (GTE operator)
		{
			title:       "Drink 3L Water Daily",
			description: "Stay hydrated throughout the day",
			icon:        "💧",
			category:    "Health",
			priority:    2,
			target:      map[string]any{"value": 3.0, "operator": "gte", "unit_id": "units:l", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// Measurable habit: Exercise (GTE operator)
		{
			title:       "Exercise 30 min Daily",
			description: "Daily physical activity",
			icon:        "🏃",
			category:    "Health",
			priority:    3,
			target:      map[string]any{"value": 30, "operator": "gte", "unit_id": "units:min", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// Measurable habit: Reading (GTE operator)
		{
			title:       "Read 30 Minutes Daily",
			description: "Daily reading habit",
			icon:        "📚",
			category:    "Learning",
			priority:    2,
			target:      map[string]any{"value": 30, "operator": "gte", "unit_id": "units:min", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// Measurable one-time: Running challenge
		{
			title:       "Run 100km This Month",
			description: "Monthly running challenge",
			icon:        "🏅",
			category:    "Health",
			priority:    3,
			target:      map[string]any{"value": 100, "operator": "gte", "unit_id": "units:km", "per_period": false},
			deadline:    &endOfMonth,
		},
		// Avoidance: Limit coffee (LTE operator)
		{
			title:       "Max 3 Coffees Per Day",
			description: "Limit caffeine intake",
			icon:        "☕",
			category:    "Health",
			priority:    2,
			target:      map[string]any{"value": 3, "operator": "lte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// Strict avoidance: Zero cigarettes (EQ operator)
		{
			title:       "Stay Smoke-Free",
			description: "Zero cigarettes",
			icon:        "🚭",
			category:    "Health",
			priority:    3,
			target:      map[string]any{"value": 0, "operator": "eq", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// Grouped goal: SaaS Launch (no target, will have children)
		{
			title:       "Launch Side Project",
			description: "Build and launch a web app",
			icon:        "🚀",
			category:    "Projects",
			priority:    3,
			deadline:    &endOfYear,
		},
		// Child goals for grouped goal (created separately, linked later)
		{
			title:       "Design UI/UX",
			description: "Complete the design phase",
			icon:        "🎨",
			category:    "Projects",
			priority:    2,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": false},
		},
		{
			title:       "Build MVP",
			description: "Develop minimum viable product",
			icon:        "⚙️",
			category:    "Projects",
			priority:    3,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": false},
		},
		{
			title:       "Beta Test",
			description: "Get 10 beta testers",
			icon:        "🧪",
			category:    "Projects",
			priority:    2,
			target:      map[string]any{"value": 10, "operator": "gte", "unit_id": "units:count", "per_period": false},
		},
		// Savings goal
		{
			title:       "Save $5000",
			description: "Emergency fund savings",
			icon:        "💰",
			category:    "Finance",
			priority:    3,
			target:      map[string]any{"value": 5000, "operator": "gte", "unit_id": "units:dollars", "per_period": false},
			deadline:    &endOfYear,
		},
	}

	for _, g := range goals {
		payload := map[string]any{
			"title":       g.title,
			"description": g.description,
			"icon":        g.icon,
			"priority":    g.priority,
		}

		if g.category != "" && categories[g.category] != "" {
			payload["category_id"] = categories[g.category]
		}
		if g.target != nil {
			payload["target"] = g.target
		}
		if g.recurrence != nil {
			payload["recurrence"] = g.recurrence
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

	// Link child goals to parent (Launch Side Project)
	parentID := result["Launch Side Project"]
	if parentID != "" {
		childGoals := []string{"Design UI/UX", "Build MVP", "Beta Test"}
		for i, childTitle := range childGoals {
			if childID, ok := result[childTitle]; ok && childID != "" {
				idPart := strings.TrimPrefix(parentID, "goals:")
				_, err := apiRequest("POST", "/goals/"+idPart+"/children", map[string]any{
					"child_goal_id": childID,
					"order":         i + 1,
					"required":      true,
				})
				if err != nil {
					log.Warn().Err(err).Str("parent", "Launch Side Project").Str("child", childTitle).Msg("Failed to link child goal")
				}
			}
		}
	}

	return result, nil
}

// =============================================================================
// TEMPLATE SEEDING (New Model)
// =============================================================================

type templateDef struct {
	title           string
	description     string
	icon            string
	category        string
	defaultDuration int // seconds
	isQuickLog      bool
	quickLogOrder   int
	quantityEnabled bool
	quantityDefault float64
	quantityStep    float64
	goalTitle       string // Goal to link via template_goals
}

func seedTemplates(categories, goals map[string]string) (map[string]string, error) {
	result := make(map[string]string)

	templates := []templateDef{
		{
			title:           "Log Water",
			description:     "Quick log water intake",
			icon:            "💧",
			category:        "Health",
			defaultDuration: 60,
			isQuickLog:      true,
			quickLogOrder:   1,
			quantityEnabled: true,
			quantityDefault: 0.5,
			quantityStep:    0.25,
			goalTitle:       "Drink 3L Water Daily",
		},
		{
			title:           "Morning Run",
			description:     "Quick log running",
			icon:            "🏃",
			category:        "Health",
			defaultDuration: 1800,
			isQuickLog:      true,
			quickLogOrder:   2,
			quantityEnabled: true,
			quantityDefault: 5.0,
			quantityStep:    0.5,
			goalTitle:       "Run 100km This Month",
		},
		{
			title:           "Gym Workout",
			description:     "Log gym session",
			icon:            "💪",
			category:        "Health",
			defaultDuration: 3600,
			isQuickLog:      true,
			quickLogOrder:   3,
			quantityEnabled: true,
			quantityDefault: 60,
			quantityStep:    15,
			goalTitle:       "Exercise 30 min Daily",
		},
		{
			title:           "Reading Session",
			description:     "Book reading time",
			icon:            "📚",
			category:        "Learning",
			defaultDuration: 1800,
			isQuickLog:      true,
			quickLogOrder:   4,
			quantityEnabled: true,
			quantityDefault: 30,
			quantityStep:    10,
			goalTitle:       "Read 30 Minutes Daily",
		},
		{
			title:           "Coffee",
			description:     "Log coffee consumption",
			icon:            "☕",
			category:        "Health",
			defaultDuration: 300,
			isQuickLog:      true,
			quickLogOrder:   5,
			quantityEnabled: true,
			quantityDefault: 1,
			quantityStep:    1,
			goalTitle:       "Max 3 Coffees Per Day",
		},
		{
			title:           "Deep Work",
			description:     "Focused work session",
			icon:            "🎯",
			category:        "Work",
			defaultDuration: 5400,
			isQuickLog:      true,
			quickLogOrder:   6,
			quantityEnabled: true,
			quantityDefault: 90,
			quantityStep:    30,
		},
	}

	for _, t := range templates {
		payload := map[string]any{
			"title":            t.title,
			"description":      t.description,
			"icon":             t.icon,
			"default_duration": t.defaultDuration,
			"is_quick_log":     t.isQuickLog,
			"quick_log_order":  t.quickLogOrder,
		}

		if t.category != "" && categories[t.category] != "" {
			payload["category_id"] = categories[t.category]
		}
		if t.quantityEnabled {
			payload["quantity_enabled"] = true
			payload["quantity_default"] = t.quantityDefault
			payload["quantity_step"] = t.quantityStep
		}

		// Link to goal via goal_links
		if t.goalTitle != "" && goals[t.goalTitle] != "" {
			payload["goal_links"] = []map[string]any{{
				"goal_id":         goals[t.goalTitle],
				"auto_link_tasks": true,
			}}
		}

		resp, err := apiRequest("POST", "/templates", payload)
		if err != nil {
			log.Warn().Err(err).Str("template", t.title).Msg("Failed to create template")
			continue
		}

		if id, ok := resp["id"].(string); ok {
			result[t.title] = id
		}
	}

	return result, nil
}

// =============================================================================
// TASK SEEDING (30 Days of Realistic Data)
// =============================================================================

func seedTasksMultiDay(ctx context.Context, db *database.DB, categories, goals, templates map[string]string) (totalTasks, totalLinks int) {
	now := time.Now()

	// Track streaks state in memory to simulate realistic progression
	streaks := make(map[string]int)

	// Seed past 12 days + today + 3 future days (approx 15 days total)
	for dayOffset := -12; dayOffset <= 3; dayOffset++ {
		day := now.AddDate(0, 0, dayOffset)
		tasks, links := seedTasksForDay(ctx, db, day, dayOffset, categories, goals, templates, streaks)
		totalTasks += tasks
		totalLinks += links
	}

	return totalTasks, totalLinks
}

func seedTasksForDay(ctx context.Context, db *database.DB, day time.Time, dayOffset int, categories, goals, templates map[string]string, streaks map[string]int) (tasks, links int) {
	isPast := dayOffset < 0
	isWeekend := day.Weekday() == time.Saturday || day.Weekday() == time.Sunday
	dayOfMonth := day.Day()

	// Track daily totals for this day to trigger goal logs
	dailyTotals := make(map[string]float64)

	// =========================================================================
	// SCENARIO 1: ONE GOAL, MULTIPLE TASKS (Hydration)
	// =========================================================================
	hydrationGoalTitle := "Drink 3L Water Daily"
	meetsHydration := rand.Float32() < 0.8 // 80% chance of meeting goal
	waterLogs := rand.Intn(5) + 2
	if meetsHydration {
		waterLogs = rand.Intn(3) + 4 // 4-6 to meet 3L
	}

	for i := 0; i < waterLogs; i++ {
		hour := 7 + i*3 + rand.Intn(2)
		quantity := 0.5 + float64(rand.Intn(3))*0.25

		if createTaskWithDetails(day, hour, 0, 5, "Log Water", "💧", categories["Health"],
			&quantity, "units:l", hydrationGoalTitle, goals, isPast, nil, nil, "quick") {
			tasks++
			links++
			dailyTotals[hydrationGoalTitle] += quantity
		}
	}

	// Check if hydration goal met
	if isPast && dailyTotals[hydrationGoalTitle] >= 3.0 {
		streaks[hydrationGoalTitle]++
		_ = createGoalLog(ctx, db, goals[hydrationGoalTitle], "target_met", map[string]any{
			"current_value": dailyTotals[hydrationGoalTitle],
			"streak":        streaks[hydrationGoalTitle],
		}, day)
		_ = updateGoalStreak(ctx, db, goals[hydrationGoalTitle], streaks[hydrationGoalTitle], day)
	} else if isPast {
		streaks[hydrationGoalTitle] = 0
		if streaks[hydrationGoalTitle] > 0 {
			_ = createGoalLog(ctx, db, goals[hydrationGoalTitle], "streak_broken", nil, day)
		}
	}

	// =========================================================================
	// SCENARIO 2: ABANDONED GOAL (Reading)
	// Starts strong, then stops after day -7
	// =========================================================================
	readingGoalTitle := "Read 30 Minutes Daily"
	if dayOffset < -7 { // Only do this in the first few days
		hour := 20 + rand.Intn(2)
		minutes := 30.0 + float64(rand.Intn(30))

		if createTaskWithDetails(day, hour, 0, int(minutes*60), "Evening Reading", "📚", categories["Learning"],
			&minutes, "units:min", readingGoalTitle, goals, isPast, nil, nil, "template") {
			tasks++
			links++
			dailyTotals[readingGoalTitle] += minutes
		}

		if isPast && dailyTotals[readingGoalTitle] >= 30 {
			streaks[readingGoalTitle]++
			_ = createGoalLog(ctx, db, goals[readingGoalTitle], "target_met", map[string]any{
				"current_value": dailyTotals[readingGoalTitle],
				"streak":        streaks[readingGoalTitle],
			}, day)
			_ = updateGoalStreak(ctx, db, goals[readingGoalTitle], streaks[readingGoalTitle], day)
		}
	} else if isPast && streaks[readingGoalTitle] > 0 {
		// Streak broken and never recovered
		streaks[readingGoalTitle] = 0
		_ = createGoalLog(ctx, db, goals[readingGoalTitle], "streak_broken", nil, day)
	}

	// =========================================================================
	// SCENARIO 3 & 4: ONE TASK -> MULTIPLE GOALS & MILESTONES (Running)
	// =========================================================================
	runGoalTitle := "Run 100km This Month"
	exerciseGoalTitle := "Exercise 30 min Daily"

	// Special Milestone Run on Day -5
	isMilestoneDay := dayOffset == -5

	if isMilestoneDay || (rand.Float32() < 0.6 || !isWeekend) {
		hour := 6 + rand.Intn(2)
		km := 3.0 + float64(rand.Intn(8)) // 3-10km

		// Force a big run on milestone day
		if isMilestoneDay {
			km = 20.0
		}

		durationMin := 30 + int(km)*5 // Slower pace for long run

		// Add emotions and reflections
		var positives []map[string]any
		if rand.Float32() < 0.5 {
			positives = []map[string]any{
				{"text": randomElement([]string{"Felt energetic", "Good weather", "Enjoyed the scenery"})},
			}
		}

		// Create the task linked to "Run 100km"
		// For milestone day, we inject specific milestone data
		goalLinks := []map[string]any{{
			"goal_id":        goals[runGoalTitle],
			"impact_type":    "positive",
			"quantity_value": km,
		}}

		if isMilestoneDay {
			goalLinks[0]["is_milestone"] = true
			goalLinks[0]["milestone_label"] = "20k Half-Marathon Prep"
			goalLinks[0]["milestone_order"] = 1
			goalLinks[0]["notes"] = "Pushing limits!"
		}

		// Also link to Exercise goal (One task -> Multiple goals)
		goalLinks = append(goalLinks, map[string]any{
			"goal_id":        goals[exerciseGoalTitle],
			"impact_type":    "positive",
			"quantity_value": float64(durationMin),
		})

		payload := map[string]any{
			"title":       "Morning Run",
			"start_date":  day.Add(time.Duration(hour) * time.Hour).Format(time.RFC3339),
			"end_date":    day.Add(time.Duration(hour)*time.Hour + time.Duration(durationMin)*time.Minute).Format(time.RFC3339),
			"category_id": categories["Health"],
			"source":      "template",
			"goal_links":  goalLinks,
		}

		if positives != nil {
			payload["positives"] = positives
		}

		_, err := apiRequest("POST", "/tasks", payload)
		if err == nil {
			tasks++
			links += 2
			dailyTotals[runGoalTitle] += km
			dailyTotals[exerciseGoalTitle] += float64(durationMin)
		}
	}

	// Update exercise streak
	if isPast && dailyTotals[exerciseGoalTitle] >= 30 {
		streaks[exerciseGoalTitle]++
		_ = createGoalLog(ctx, db, goals[exerciseGoalTitle], "target_met", map[string]any{
			"current_value": dailyTotals[exerciseGoalTitle],
			"streak":        streaks[exerciseGoalTitle],
		}, day)
		_ = updateGoalStreak(ctx, db, goals[exerciseGoalTitle], streaks[exerciseGoalTitle], day)
	} else if isPast {
		streaks[exerciseGoalTitle] = 0
	}

	// =========================================================================
	// SCENARIO 5: EPICS & MANUAL LINKING (Side Project)
	// Linking tasks to child goals or parent goal manually
	// =========================================================================

	// "Design UI/UX" child goal activity
	if dayOffset > -10 && dayOffset < -5 {
		if rand.Float32() < 0.7 {
			hour := 19 + rand.Intn(3)
			// Manual task linked to a child goal
			goalLinks := []map[string]any{{
				"goal_id":        goals["Design UI/UX"], // Child goal
				"impact_type":    "positive",
				"quantity_value": 1.0, // 1 session
			}}

			payload := map[string]any{
				"title":       "Figma Design Session",
				"start_date":  day.Add(time.Duration(hour) * time.Hour).Format(time.RFC3339),
				"end_date":    day.Add(time.Duration(hour)*time.Hour + 2*time.Hour).Format(time.RFC3339),
				"category_id": categories["Projects"],
				"source":      "manual", // Manual entry
				"goal_links":  goalLinks,
			}
			_, err := apiRequest("POST", "/tasks", payload)
			if err == nil {
				tasks++
				links++
			}
		}
	}

	// "Build MVP" child goal activity
	if dayOffset >= -5 {
		if rand.Float32() < 0.6 {
			hour := 20
			// Manual task linked to child AND parent (demonstrating impact on Epic)
			goalLinks := []map[string]any{
				{
					"goal_id":        goals["Build MVP"],
					"impact_type":    "positive",
					"quantity_value": 1.0,
				},
				{
					"goal_id":     goals["Launch Side Project"], // Parent goal interaction
					"impact_type": "neutral",                    // Just a log
					"notes":       "Contributing to overall epic",
				},
			}

			payload := map[string]any{
				"title":       "Coding MVP API",
				"start_date":  day.Add(time.Duration(hour) * time.Hour).Format(time.RFC3339),
				"end_date":    day.Add(time.Duration(hour)*time.Hour + 3*time.Hour).Format(time.RFC3339),
				"category_id": categories["Projects"],
				"source":      "manual",
				"goal_links":  goalLinks,
			}
			_, err := apiRequest("POST", "/tasks", payload)
			if err == nil {
				tasks++
				links += 2
			}
		}
	}

	// ---------------------------
	// GYM WORKOUT (2-3 times per week, weekdays)
	// ---------------------------
	if !isWeekend && rand.Float32() < 0.4 {
		hour := 17 + rand.Intn(2) // After work
		minutes := float64(45 + rand.Intn(30))

		positives := []map[string]any{}
		if rand.Float32() < 0.5 {
			positives = []map[string]any{
				{"text": randomElement([]string{"New PR on squats!", "Great pump", "Completed full routine", "Increased weight on bench"})},
			}
		}

		if createTaskWithDetails(day, hour, 0, int(minutes), "Gym Workout", "💪", categories["Health"],
			&minutes, "units:min", "Exercise 30 min Daily", goals, isPast, positives, nil, "template") {
			tasks++
			links++
		}
	}

	// ---------------------------
	// COFFEE (Limit goal - 1-4 per day, sometimes exceeds limit)
	// ---------------------------
	// Some days (10%) exceed the 3-cup limit to demonstrate avoidance tracking
	exceeCoffeeLimit := rand.Float32() < 0.10
	coffeeLogs := rand.Intn(3) + 1 // 1-3 normally
	if exceeCoffeeLimit {
		coffeeLogs = 4 + rand.Intn(2) // 4-5 on bad days
	}

	for i := 0; i < coffeeLogs; i++ {
		hour := 7 + i*3 + rand.Intn(2)
		qty := 1.0

		if createTaskWithDetails(day, hour, 30, 15, "Coffee", "☕", categories["Health"],
			&qty, "units:count", "Max 3 Coffees Per Day", goals, isPast, nil, nil, "quick") {
			tasks++
			links++
		}
	}

	// ---------------------------
	// READING HABIT (70% consistency)
	// ---------------------------
	if rand.Float32() < 0.7 {
		hour := 20 + rand.Intn(2)
		minutes := float64(20 + rand.Intn(30))

		positives := []map[string]any{}
		if rand.Float32() < 0.3 {
			positives = []map[string]any{
				{"text": randomElement([]string{"Finished a chapter", "Great insights", "Learning so much", "Couldn't put it down"})},
			}
		}

		if createTaskWithDetails(day, hour, 0, int(minutes), "Reading Session", "📚", categories["Learning"],
			&minutes, "units:min", "Read 30 Minutes Daily", goals, isPast, positives, nil, "template") {
			tasks++
			links++
		}
	}

	// ---------------------------
	// WORK TASKS (Weekdays only)
	// ---------------------------
	if !isWeekend {
		// Daily Standup
		var standupPositives, standupNegatives []map[string]any
		if rand.Float32() < 0.2 {
			standupPositives = []map[string]any{{"text": "Clear priorities for the day"}}
		}

		if createTaskWithDetails(day, 9, 30, 15, "Daily Standup", "👥", categories["Work"],
			nil, "", "", nil, isPast, standupPositives, standupNegatives, "manual") {
			tasks++
		}

		// Deep Work Session
		if rand.Float32() < 0.8 {
			minutes := float64(60 + rand.Intn(60))
			journal := ""
			if rand.Float32() < 0.3 {
				journal = randomElement([]string{
					"Focused on feature implementation. Good progress.",
					"Debugging session. Found and fixed the issue.",
					"Code review and documentation updates.",
					"Architecture planning and design work.",
				})
			}

			positives := []map[string]any{}
			negatives := []map[string]any{}
			if rand.Float32() < 0.4 {
				positives = []map[string]any{{"text": randomElement([]string{
					"Flow state achieved",
					"Completed ahead of schedule",
					"Clean code written",
				})}}
			}
			if rand.Float32() < 0.15 {
				negatives = []map[string]any{{"text": randomElement([]string{
					"Too many interruptions",
					"Unexpected complexity",
					"Need to revisit approach",
				})}}
			}

			if createTaskWithDetailsAndJournal(day, 10+rand.Intn(3), 0, int(minutes), "Deep Work Session", "🎯",
				categories["Work"], &minutes, "units:min", "", nil, isPast, positives, negatives, journal, "template") {
				tasks++
			}
		}

		// Meetings (1-3 per day)
		numMeetings := rand.Intn(3) + 1
		for i := 0; i < numMeetings; i++ {
			hour := 11 + i*2 + rand.Intn(2)
			dur := 30 + rand.Intn(30)
			title := randomElement([]string{"Team Sync", "1:1 Meeting", "Project Review", "Planning Session", "Client Call"})

			if createTaskWithDetails(day, hour, 0, dur, title, "📅", categories["Work"],
				nil, "", "", nil, isPast, nil, nil, "manual") {
				tasks++
			}
		}
	}

	// ---------------------------
	// SAVINGS CONTRIBUTIONS (Weekly on Fridays)
	// ---------------------------
	if day.Weekday() == time.Friday && isPast {
		amount := float64(100 + rand.Intn(200)) // $100-300 contribution
		if createTaskWithDetails(day, 12, 0, 5, "Weekly Savings Transfer", "💰", categories["Finance"],
			&amount, "units:dollars", "Save $5000", goals, isPast, nil, nil, "manual") {
			tasks++
			links++
		}
	}

	// ---------------------------
	// PROJECT MILESTONES (For grouped goal "Launch Side Project")
	// ---------------------------
	// Design milestone - early in the period
	if dayOffset == -25 && isPast {
		qty := 1.0
		positives := []map[string]any{{"text": "Design mockups completed and approved", "emotion_id": "emotions:E35"}}
		if createMilestoneTask(day, 15, 0, 120, "Complete UI Design Mockups", "🎨",
			categories["Projects"], &qty, "units:count", "Design UI/UX", goals, positives,
			"Design Phase Complete", 1) {
			tasks++
			links++
		}
	}

	// MVP milestone - mid period
	if dayOffset == -10 && isPast {
		qty := 1.0
		positives := []map[string]any{{"text": "Core features working, ready for testing", "emotion_id": "emotions:E26"}}
		if createMilestoneTask(day, 16, 0, 180, "MVP Feature Complete", "⚙️",
			categories["Projects"], &qty, "units:count", "Build MVP", goals, positives,
			"MVP Ready", 2) {
			tasks++
			links++
		}
	}

	// ---------------------------
	// MEDITATION / MINDFULNESS (Random days)
	// ---------------------------
	if rand.Float32() < 0.3 {
		minutes := float64(10 + rand.Intn(15)) // 10-25 minutes
		hour := 7 + rand.Intn(2)               // Morning

		positives := []map[string]any{}
		if rand.Float32() < 0.6 {
			positives = []map[string]any{
				{"text": randomElement([]string{"Felt calm and centered", "Good focus today", "Mind felt clear afterwards"}), "emotion_id": "emotions:E15"},
			}
		}

		if createTaskWithDetails(day, hour, 0, int(minutes), "Morning Meditation", "🧘", categories["Personal"],
			nil, "", "", nil, isPast, positives, nil, "template") {
			tasks++
		}
	}

	// ---------------------------
	// LEARNING ACTIVITIES (Weekends - online courses)
	// ---------------------------
	if isWeekend && rand.Float32() < 0.6 {
		minutes := float64(45 + rand.Intn(45))
		topics := []string{"Go Advanced Patterns", "System Design", "Cloud Architecture", "Database Optimization"}
		topic := randomElement(topics)

		positives := []map[string]any{{"text": "Learned something new", "emotion_id": "emotions:E44"}}
		journal := fmt.Sprintf("Studied: %s. Great content!", topic)

		if createTaskWithDetailsAndJournal(day, 10+rand.Intn(4), 0, int(minutes),
			fmt.Sprintf("Online Course: %s", topic), "🎓", categories["Learning"],
			nil, "", "", nil, isPast, positives, nil, journal, "manual") {
			tasks++
		}
	}

	// ---------------------------
	// PERSONAL ERRANDS (Weekends)
	// ---------------------------
	if isWeekend && rand.Float32() < 0.5 {
		errands := []string{"Grocery Shopping", "House Cleaning", "Cooking Meal Prep", "Laundry", "Car Maintenance"}
		title := randomElement(errands)
		dur := 30 + rand.Intn(60)

		if createTaskWithDetails(day, 10+rand.Intn(6), 0, dur, title, "🏠", categories["Personal"],
			nil, "", "", nil, isPast, nil, nil, "manual") {
			tasks++
		}
	}

	// ---------------------------
	// SOCIAL ACTIVITIES (Occasional)
	// ---------------------------
	if rand.Float32() < 0.15 {
		activities := []struct {
			title, icon string
		}{
			{"Dinner with Friends", "🍽️"},
			{"Movie Night", "🎬"},
			{"Phone Call with Family", "📱"},
			{"Game Night", "🎮"},
		}
		activity := activities[rand.Intn(len(activities))]
		hour := 18 + rand.Intn(3)
		dur := 60 + rand.Intn(120)

		positives := []map[string]any{{"text": "Quality time with loved ones", "emotion_id": "emotions:E16"}}
		emotionID := randomElement([]string{"emotions:E25", "emotions:E26", "emotions:E35", "emotions:E36"})

		if createTaskWithEmotionAndReflections(day, hour, 0, dur, activity.title, activity.icon,
			categories["Personal"], nil, "", "", nil, isPast, positives, nil, emotionID, "manual") {
			tasks++
		}
	}

	// ---------------------------
	// SPECIAL: Smoke-free tracking (avoidance goal)
	// ---------------------------
	// Very rarely (2%) a slip-up occurs to demonstrate avoidance tracking
	if rand.Float32() < 0.02 && isPast && dayOfMonth%7 == 0 {
		qty := 1.0
		negatives := []map[string]any{
			{"text": "Stressful day led to slip-up", "emotion_id": "emotions:E64"},
			{"text": "Need to work on coping mechanisms", "emotion_id": "emotions:E71"},
		}
		// This would be a negative impact on the smoke-free goal
		if createTaskWithNegativeImpact(day, 20, 0, 5, "Smoking Slip", "🚬", categories["Health"],
			&qty, "units:count", "Stay Smoke-Free", goals, negatives) {
			tasks++
			links++
		}
	}

	return tasks, links
}

// =============================================================================
// ENHANCED TASK CREATION HELPERS
// =============================================================================

// createTaskWithDetails creates a task with positives/negatives reflections
func createTaskWithDetails(day time.Time, hour, minute, durationMin int, title, icon, categoryID string,
	quantity *float64, unitID, goalTitle string, goals map[string]string, completed bool,
	positives, negatives []map[string]any, source string) bool {

	startTime := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
	endTime := startTime.Add(time.Duration(durationMin) * time.Minute)

	payload := map[string]any{
		"title":      title,
		"start_date": startTime.Format(time.RFC3339),
		"end_date":   endTime.Format(time.RFC3339),
		"source":     source,
	}

	if categoryID != "" {
		payload["category_id"] = categoryID
	}

	if quantity != nil && unitID != "" {
		payload["quantity"] = map[string]any{
			"value":   *quantity,
			"unit_id": unitID,
		}
	}

	// Add goal link
	if goalTitle != "" && goals != nil && goals[goalTitle] != "" {
		payload["goal_links"] = []map[string]any{{
			"goal_id":     goals[goalTitle],
			"impact_type": "positive",
		}}
		if quantity != nil {
			//nolint:errcheck // acceptable in seeder
			payload["goal_links"].([]map[string]any)[0]["quantity_value"] = *quantity
		}
	}

	// Add reflections
	if len(positives) > 0 {
		payload["positives"] = positives
	}
	if len(negatives) > 0 {
		payload["negatives"] = negatives
	}

	// Add random emotion
	if rand.Float32() < 0.3 {
		emotions := []string{"emotions:E16", "emotions:E25", "emotions:E26", "emotions:E35", "emotions:E44"}
		payload["emotion_id"] = emotions[rand.Intn(len(emotions))]
	}

	resp, err := apiRequest("POST", "/tasks", payload)
	if err != nil {
		return false
	}

	if completed {
		if taskID, ok := resp["id"].(string); ok {
			idPart := strings.TrimPrefix(taskID, "tasks:")
			//nolint:errcheck // ignore PUT errors
			_, _ = apiRequest("PUT", "/tasks/"+idPart, map[string]any{"completed": true})
		}
	}

	return true
}

// createTaskWithDetailsAndJournal creates a task with journal entry and reflections
func createTaskWithDetailsAndJournal(day time.Time, hour, minute, durationMin int, title, icon, categoryID string,
	quantity *float64, unitID, goalTitle string, goals map[string]string, completed bool,
	positives, negatives []map[string]any, journal, source string) bool {

	startTime := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
	endTime := startTime.Add(time.Duration(durationMin) * time.Minute)

	payload := map[string]any{
		"title":      title,
		"start_date": startTime.Format(time.RFC3339),
		"end_date":   endTime.Format(time.RFC3339),
		"source":     source,
	}

	if categoryID != "" {
		payload["category_id"] = categoryID
	}
	if journal != "" {
		payload["journal"] = journal
	}

	if quantity != nil && unitID != "" {
		payload["quantity"] = map[string]any{
			"value":   *quantity,
			"unit_id": unitID,
		}
	}

	if goalTitle != "" && goals != nil && goals[goalTitle] != "" {
		payload["goal_links"] = []map[string]any{{
			"goal_id":     goals[goalTitle],
			"impact_type": "positive",
		}}
		if quantity != nil {
			//nolint:errcheck // acceptable in seeder
			payload["goal_links"].([]map[string]any)[0]["quantity_value"] = *quantity
		}
	}

	if len(positives) > 0 {
		payload["positives"] = positives
	}
	if len(negatives) > 0 {
		payload["negatives"] = negatives
	}

	// Add random emotion with higher probability when there's a journal
	if rand.Float32() < 0.5 {
		emotions := []string{"emotions:E16", "emotions:E25", "emotions:E26", "emotions:E35"}
		payload["emotion_id"] = emotions[rand.Intn(len(emotions))]
	}

	resp, err := apiRequest("POST", "/tasks", payload)
	if err != nil {
		return false
	}

	if completed {
		if taskID, ok := resp["id"].(string); ok {
			idPart := strings.TrimPrefix(taskID, "tasks:")
			//nolint:errcheck // ignore PUT errors
			_, _ = apiRequest("PUT", "/tasks/"+idPart, map[string]any{"completed": true})
		}
	}

	return true
}

// createTaskWithEmotionAndReflections creates a task with specific emotion and reflections
func createTaskWithEmotionAndReflections(day time.Time, hour, minute, durationMin int, title, icon, categoryID string,
	quantity *float64, unitID, goalTitle string, goals map[string]string, completed bool,
	positives, negatives []map[string]any, emotionID, source string) bool {

	startTime := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
	endTime := startTime.Add(time.Duration(durationMin) * time.Minute)

	payload := map[string]any{
		"title":      title,
		"start_date": startTime.Format(time.RFC3339),
		"end_date":   endTime.Format(time.RFC3339),
		"source":     source,
	}

	if categoryID != "" {
		payload["category_id"] = categoryID
	}
	if emotionID != "" {
		payload["emotion_id"] = emotionID
	}

	if quantity != nil && unitID != "" {
		payload["quantity"] = map[string]any{
			"value":   *quantity,
			"unit_id": unitID,
		}
	}

	if goalTitle != "" && goals != nil && goals[goalTitle] != "" {
		payload["goal_links"] = []map[string]any{{
			"goal_id":     goals[goalTitle],
			"impact_type": "positive",
		}}
		if quantity != nil {
			//nolint:errcheck // acceptable in seeder
			payload["goal_links"].([]map[string]any)[0]["quantity_value"] = *quantity
		}
	}

	if len(positives) > 0 {
		payload["positives"] = positives
	}
	if len(negatives) > 0 {
		payload["negatives"] = negatives
	}

	resp, err := apiRequest("POST", "/tasks", payload)
	if err != nil {
		return false
	}

	if completed {
		if taskID, ok := resp["id"].(string); ok {
			idPart := strings.TrimPrefix(taskID, "tasks:")
			//nolint:errcheck // ignore PUT errors
			_, _ = apiRequest("PUT", "/tasks/"+idPart, map[string]any{"completed": true})
		}
	}

	return true
}

// createMilestoneTask creates a task that represents a milestone for a goal
func createMilestoneTask(day time.Time, hour, minute, durationMin int, title, icon, categoryID string,
	quantity *float64, unitID, goalTitle string, goals map[string]string,
	positives []map[string]any, milestoneLabel string, milestoneOrder int) bool {

	startTime := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
	endTime := startTime.Add(time.Duration(durationMin) * time.Minute)

	payload := map[string]any{
		"title":      title,
		"start_date": startTime.Format(time.RFC3339),
		"end_date":   endTime.Format(time.RFC3339),
		"source":     "manual",
		"completed":  true,
	}

	if categoryID != "" {
		payload["category_id"] = categoryID
	}

	if quantity != nil && unitID != "" {
		payload["quantity"] = map[string]any{
			"value":   *quantity,
			"unit_id": unitID,
		}
	}

	// Add milestone goal link
	if goalTitle != "" && goals != nil && goals[goalTitle] != "" {
		payload["goal_links"] = []map[string]any{{
			"goal_id":         goals[goalTitle],
			"impact_type":     "positive",
			"is_milestone":    true,
			"milestone_label": milestoneLabel,
			"milestone_order": milestoneOrder,
		}}
		if quantity != nil {
			//nolint:errcheck // acceptable in seeder
			payload["goal_links"].([]map[string]any)[0]["quantity_value"] = *quantity
		}
	}

	if len(positives) > 0 {
		payload["positives"] = positives
	}

	// Milestones get positive emotions
	emotions := []string{"emotions:E25", "emotions:E26", "emotions:E35"} // Joy, Excited, Proud
	payload["emotion_id"] = emotions[rand.Intn(len(emotions))]

	_, err := apiRequest("POST", "/tasks", payload)
	return err == nil
}

// createTaskWithNegativeImpact creates a task with negative impact on a goal (for avoidance goals)
func createTaskWithNegativeImpact(day time.Time, hour, minute, durationMin int, title, icon, categoryID string,
	quantity *float64, unitID, goalTitle string, goals map[string]string, negatives []map[string]any) bool {

	startTime := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
	endTime := startTime.Add(time.Duration(durationMin) * time.Minute)

	payload := map[string]any{
		"title":      title,
		"start_date": startTime.Format(time.RFC3339),
		"end_date":   endTime.Format(time.RFC3339),
		"source":     "manual",
		"completed":  true,
	}

	if categoryID != "" {
		payload["category_id"] = categoryID
	}

	if quantity != nil && unitID != "" {
		payload["quantity"] = map[string]any{
			"value":   *quantity,
			"unit_id": unitID,
		}
	}

	// Add negative impact goal link
	if goalTitle != "" && goals != nil && goals[goalTitle] != "" {
		payload["goal_links"] = []map[string]any{{
			"goal_id":     goals[goalTitle],
			"impact_type": "negative", // Important: negative impact for avoidance
		}}
		if quantity != nil {
			//nolint:errcheck // acceptable in seeder
			payload["goal_links"].([]map[string]any)[0]["quantity_value"] = *quantity
		}
	}

	if len(negatives) > 0 {
		payload["negatives"] = negatives
	}

	// Slip-ups get low energy/negative emotions
	emotions := []string{"emotions:E61", "emotions:E71", "emotions:E81"} // Disappointed, Guilty, Ashamed
	payload["emotion_id"] = emotions[rand.Intn(len(emotions))]

	_, err := apiRequest("POST", "/tasks", payload)
	return err == nil
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// randomElement returns a random element from a string slice
func randomElement(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[rand.Intn(len(items))]
}

// createGoalLog simulates the goal logic creating a log entry
func createGoalLog(ctx context.Context, db *database.DB, goalID, event string, changes map[string]any, createdAt time.Time) error {
	params := map[string]any{
		"goal":  database.MustRecordID("goals", goalID),
		"event": event,
		"now":   createdAt,
	}
	if changes != nil {
		params["changes"] = changes
	}

	_, err := database.QueryAll[any](ctx, db, `
		LET $snapshot = (CREATE goal_snapshots CONTENT {
			goal_id: $goal,
			status: "active",
			created_at: $now
		});
		
		RELATE $goal->goal_logs->($snapshot[0].id) CONTENT {
			event_type: $event,
			changes: $changes,
			created_at: $now,
			created_by: "seed"
		};
	`, params)
	return err
}

// updateGoalStreak updates the goal stats in DB
func updateGoalStreak(ctx context.Context, db *database.DB, goalID string, streak int, lastCompleted time.Time) error {
	_, err := database.QueryAll[any](ctx, db, `
		UPDATE type::thing($id) MERGE {
			current_streak: $streak,
			last_completed_date: $date,
			updated_at: $date
		}
	`, map[string]any{
		"id":     database.MustRecordID("goals", goalID),
		"streak": streak,
		"date":   lastCompleted,
	})
	return err
}
