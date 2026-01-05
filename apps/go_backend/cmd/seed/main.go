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

	// 4. Seed 30 days of tasks with realistic patterns
	totalTasks, totalLinks := seedTasksMultiDay(categories, goals, templates)
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

func seedTasksMultiDay(categories, goals, templates map[string]string) (totalTasks, totalLinks int) {
	now := time.Now()

	// Seed past 30 days + today + 3 future days
	for dayOffset := -30; dayOffset <= 3; dayOffset++ {
		day := now.AddDate(0, 0, dayOffset)
		tasks, links := seedTasksForDay(day, dayOffset, categories, goals, templates)
		totalTasks += tasks
		totalLinks += links
	}

	return totalTasks, totalLinks
}

func seedTasksForDay(day time.Time, dayOffset int, categories, goals, templates map[string]string) (tasks, links int) {
	isPast := dayOffset < 0
	isWeekend := day.Weekday() == time.Saturday || day.Weekday() == time.Sunday

	// Random whether this day meets hydration goal
	meetsHydration := rand.Float32() < 0.8 // 80% chance

	// Water intake: 2-6 logs per day
	waterLogs := rand.Intn(5) + 2
	if meetsHydration {
		waterLogs = rand.Intn(3) + 4 // 4-6 to meet 3L
	}

	for i := 0; i < waterLogs; i++ {
		hour := 7 + i*3 + rand.Intn(2)               // Spread throughout day
		quantity := 0.5 + float64(rand.Intn(3))*0.25 // 0.5, 0.75, 1.0, 1.25

		if createTask(day, hour, 0, 5, "Log Water", "💧", categories["Health"],
			&quantity, "units:l", "Drink 3L Water Daily", goals, isPast) {
			tasks++
			links++
		}
	}

	// Running: 3-5 times per week on random days
	if rand.Float32() < 0.6 || !isWeekend {
		hour := 6 + rand.Intn(2)
		km := 3.0 + float64(rand.Intn(8)) // 3-10km

		if createTask(day, hour, 0, 30+int(km)*3, "Morning Run", "🏃", categories["Health"],
			&km, "units:km", "Run 100km This Month", goals, isPast) {
			tasks++
			links++
		}
	}

	// Coffee: 1-4 per day
	coffeeLogs := rand.Intn(4) + 1
	for i := 0; i < coffeeLogs; i++ {
		hour := 7 + i*4 + rand.Intn(2)
		qty := 1.0

		if createTask(day, hour, 30, 15, "Coffee", "☕", categories["Health"],
			&qty, "units:count", "Max 3 Coffees Per Day", goals, isPast) {
			tasks++
			links++
		}
	}

	// Reading: most days
	if rand.Float32() < 0.7 {
		hour := 20 + rand.Intn(2)
		minutes := float64(20 + rand.Intn(30)) // 20-50 minutes

		if createTask(day, hour, 0, int(minutes), "Reading Session", "📚", categories["Learning"],
			&minutes, "units:min", "Read 30 Minutes Daily", goals, isPast) {
			tasks++
			links++
		}
	}

	// Work tasks on weekdays
	if !isWeekend {
		// Standup
		if createTask(day, 9, 30, 15, "Daily Standup", "👥", categories["Work"], nil, "", "", nil, isPast) {
			tasks++
		}

		// Deep work
		if rand.Float32() < 0.8 {
			minutes := float64(60 + rand.Intn(60))
			if createTask(day, 10+rand.Intn(3), 0, int(minutes), "Deep Work Session", "🎯", categories["Work"],
				&minutes, "units:min", "", nil, isPast) {
				tasks++
			}
		}
	}

	return tasks, links
}

func createTask(day time.Time, hour, minute, durationMin int, title, icon, categoryID string,
	quantity *float64, unitID, goalTitle string, goals map[string]string, completed bool) bool {

	startTime := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
	endTime := startTime.Add(time.Duration(durationMin) * time.Minute)

	payload := map[string]any{
		"title":      title,
		"start_date": startTime.Format(time.RFC3339),
		"end_date":   endTime.Format(time.RFC3339),
		"source":     "manual",
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
			//nolint:errcheck // acceptable in seeder - map assignment always succeeds
			payload["goal_links"].([]map[string]any)[0]["quantity_value"] = *quantity
		}
	}

	// Add some emotions randomly
	if rand.Float32() < 0.3 {
		emotions := []string{"emotions:E16", "emotions:E26", "emotions:E44", "emotions:E31"}
		payload["emotion_id"] = emotions[rand.Intn(len(emotions))]
	}

	resp, err := apiRequest("POST", "/tasks", payload)
	if err != nil {
		return false
	}

	// Mark completed if past
	if completed {
		if taskID, ok := resp["id"].(string); ok {
			idPart := strings.TrimPrefix(taskID, "tasks:")
			//nolint:errcheck // ignore PUT errors for marking complete
			_, _ = apiRequest("PUT", "/tasks/"+idPart, map[string]any{"completed": true})
		}
	}

	return true
}
