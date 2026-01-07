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

	// 5. Update goal streaks and create goal history
	streakUpdates := seedGoalStreaksAndHistory(ctx, db, goals)
	log.Info().Int("updated", streakUpdates).Msg("✅ Goal streaks and history updated")

	return nil
}

// =============================================================================
// CATEGORY SEEDING
// =============================================================================

var categoryDefs = []struct {
	name  string
	color string
}{
	{"Work", "#3B82F6"},          // Blue - professional
	{"Health", "#10B981"},        // Green - vitality
	{"Learning", "#8B5CF6"},      // Purple - knowledge
	{"Personal", "#F59E0B"},      // Amber - warm personal
	{"Finance", "#06B6D4"},       // Cyan - money/growth
	{"Projects", "#0EA5E9"},      // Sky blue - building
	{"Family", "#EC4899"},        // Pink - love/family
	{"Relationships", "#F97316"}, // Orange - social warmth
	{"Hobbies", "#A855F7"},       // Violet - creative
	{"Home", "#84CC16"},          // Lime - domestic/nature
	{"Fitness", "#14B8A6"},       // Teal - active/energy
	{"Mindfulness", "#6366F1"},   // Indigo - calm/spiritual
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
		// Sleep goal (MAJOR MISSING ITEM)
		{
			title:       "Sleep 7-8 Hours",
			description: "Maintain healthy sleep schedule",
			icon:        "😴",
			category:    "Health",
			priority:    3,
			target:      map[string]any{"value": 7, "operator": "gte", "unit_id": "units:hr", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// Meditation habit
		{
			title:       "Meditate 10 Min Daily",
			description: "Daily mindfulness practice",
			icon:        "🧘",
			category:    "Personal",
			priority:    2,
			target:      map[string]any{"value": 10, "operator": "gte", "unit_id": "units:min", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// Steps goal
		{
			title:       "Walk 10,000 Steps Daily",
			description: "Stay active throughout the day",
			icon:        "🚶",
			category:    "Health",
			priority:    2,
			target:      map[string]any{"value": 10000, "operator": "gte", "unit_id": "units:steps", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// Meal tracking (healthy eating)
		{
			title:       "Cook Healthy Meals",
			description: "Prepare nutritious home-cooked meals",
			icon:        "🥗",
			category:    "Health",
			priority:    2,
			target:      map[string]any{"value": 2, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// Screen time limit (avoidance)
		{
			title:       "Limit Social Media",
			description: "Max 1 hour per day",
			icon:        "📱",
			category:    "Personal",
			priority:    2,
			target:      map[string]any{"value": 60, "operator": "lte", "unit_id": "units:min", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// Weekly deep work goal
		{
			title:       "20 Hours Deep Work Weekly",
			description: "Focused work sessions each week",
			icon:        "🎯",
			category:    "Work",
			priority:    3,
			target:      map[string]any{"value": 20, "operator": "gte", "unit_id": "units:hr", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "week"},
		},
		// Social connections
		{
			title:       "Connect with Friends Weekly",
			description: "Maintain social relationships",
			icon:        "👥",
			category:    "Personal",
			priority:    2,
			target:      map[string]any{"value": 2, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "week"},
		},
		// Learning goal
		{
			title:       "Learn 5 Hours Weekly",
			description: "Continuous learning and skill development",
			icon:        "📖",
			category:    "Learning",
			priority:    2,
			target:      map[string]any{"value": 5, "operator": "gte", "unit_id": "units:hr", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "week"},
		},
		// === FAMILY & RELATIONSHIPS GOALS ===
		{
			title:       "Weekly Family Dinner",
			description: "Have dinner with family once per week",
			icon:        "👨‍👩‍👧‍👦",
			category:    "Family",
			priority:    3,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "week", "active_days": []string{"sun"}},
		},
		{
			title:       "Call Parents Weekly",
			description: "Stay in touch with parents",
			icon:        "📞",
			category:    "Family",
			priority:    2,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "week"},
		},
		{
			title:       "Date Night Bi-weekly",
			description: "Quality time with partner",
			icon:        "💑",
			category:    "Relationships",
			priority:    3,
			target:      map[string]any{"value": 2, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "month"},
		},
		// === HOBBIES & CREATIVE GOALS ===
		{
			title:       "Practice Guitar 3x Weekly",
			description: "Regular music practice",
			icon:        "🎸",
			category:    "Hobbies",
			priority:    2,
			target:      map[string]any{"value": 3, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "week"},
		},
		{
			title:       "Create Art Weekly",
			description: "Paint, draw, or craft something",
			icon:        "🎨",
			category:    "Hobbies",
			priority:    2,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "week"},
		},
		// === HOME & MAINTENANCE GOALS ===
		{
			title:       "Deep Clean Home Weekly",
			description: "Thorough house cleaning",
			icon:        "🧹",
			category:    "Home",
			priority:    2,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "week", "active_days": []string{"sat"}},
		},
		{
			title:       "Organize One Area Monthly",
			description: "Declutter and organize different spaces",
			icon:        "📦",
			category:    "Home",
			priority:    1,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "month"},
		},
		// === FITNESS SPECIFIC GOALS (separate from general Health) ===
		{
			title:       "Strength Training 3x Week",
			description: "Build muscle and strength",
			icon:        "🏋️",
			category:    "Fitness",
			priority:    3,
			target:      map[string]any{"value": 3, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "week"},
		},
		{
			title:       "Stretch Daily",
			description: "Flexibility and recovery",
			icon:        "🤸",
			category:    "Fitness",
			priority:    2,
			target:      map[string]any{"value": 10, "operator": "gte", "unit_id": "units:min", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		// === MINDFULNESS & MENTAL HEALTH ===
		{
			title:       "Gratitude Journal Daily",
			description: "Write 3 things grateful for",
			icon:        "🙏",
			category:    "Mindfulness",
			priority:    2,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		{
			title:       "Weekly Reflection",
			description: "Review and plan the week",
			icon:        "📝",
			category:    "Mindfulness",
			priority:    2,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "week", "active_days": []string{"sun"}},
		},
		// === ONE-TIME SPECIFIC GOALS (with deadlines) ===
		{
			title:       "Complete Online Course",
			description: "Finish React advanced course",
			icon:        "🎓",
			category:    "Learning",
			priority:    2,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": false},
			deadline:    &endOfMonth,
		},
		{
			title:       "Read 5 Books This Year",
			description: "Annual reading goal",
			icon:        "📚",
			category:    "Learning",
			priority:    2,
			target:      map[string]any{"value": 5, "operator": "gte", "unit_id": "units:count", "per_period": false},
			deadline:    &endOfYear,
		},
		{
			title:       "Organize Photo Library",
			description: "Sort and backup all photos",
			icon:        "📸",
			category:    "Home",
			priority:    1,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": false},
			deadline:    &endOfMonth,
		},
		// === FINANCIAL GOALS (additional) ===
		{
			title:       "Track Expenses Daily",
			description: "Log all spending",
			icon:        "💵",
			category:    "Finance",
			priority:    2,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "day"},
		},
		{
			title:       "Budget Review Monthly",
			description: "Review and adjust budget",
			icon:        "📊",
			category:    "Finance",
			priority:    2,
			target:      map[string]any{"value": 1, "operator": "gte", "unit_id": "units:count", "per_period": true},
			recurrence:  map[string]any{"frequency": 1, "period": "month"},
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
			goalTitle:       "20 Hours Deep Work Weekly",
		},
		{
			title:           "Sleep",
			description:     "Log sleep duration",
			icon:            "😴",
			category:        "Health",
			defaultDuration: 28800, // 8 hours
			isQuickLog:      true,
			quickLogOrder:   7,
			quantityEnabled: true,
			quantityDefault: 8,
			quantityStep:    0.5,
			goalTitle:       "Sleep 7-8 Hours",
		},
		{
			title:           "Meditation",
			description:     "Mindfulness practice",
			icon:            "🧘",
			category:        "Personal",
			defaultDuration: 600, // 10 min
			isQuickLog:      true,
			quickLogOrder:   8,
			quantityEnabled: true,
			quantityDefault: 10,
			quantityStep:    5,
			goalTitle:       "Meditate 10 Min Daily",
		},
		{
			title:           "Walk/Steps",
			description:     "Log daily steps",
			icon:            "🚶",
			category:        "Health",
			defaultDuration: 3600,
			isQuickLog:      true,
			quickLogOrder:   9,
			quantityEnabled: true,
			quantityDefault: 5000,
			quantityStep:    1000,
			goalTitle:       "Walk 10,000 Steps Daily",
		},
		{
			title:           "Healthy Meal",
			description:     "Log home-cooked meal",
			icon:            "🥗",
			category:        "Health",
			defaultDuration: 3600,
			isQuickLog:      true,
			quickLogOrder:   10,
			quantityEnabled: true,
			quantityDefault: 1,
			quantityStep:    1,
			goalTitle:       "Cook Healthy Meals",
		},
		{
			title:           "Social Media",
			description:     "Track screen time",
			icon:            "📱",
			category:        "Personal",
			defaultDuration: 1800,
			isQuickLog:      true,
			quickLogOrder:   11,
			quantityEnabled: true,
			quantityDefault: 30,
			quantityStep:    15,
			goalTitle:       "Limit Social Media",
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

	// Seed past 20 days + today + 3 future days (focus on density per day, not length)
	for dayOffset := -20; dayOffset <= 3; dayOffset++ {
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
	// SLEEP TRACKING (Every night - MAJOR MISSING ITEM)
	// =========================================================================
	sleepGoalTitle := "Sleep 7-8 Hours"
	if isPast || dayOffset == 0 {
		// Sleep from previous night (22:00 to 6:00)
		sleepHours := 6.5 + rand.Float64()*2.0 // 6.5-8.5 hours
		sleepStart := day.Add(-6 * time.Hour)  // 22:00 previous day
		sleepEnd := day.Add(time.Duration(sleepHours-2) * time.Hour)

		payload := map[string]any{
			"title":       "Sleep",
			"start_date":  sleepStart.Format(time.RFC3339),
			"end_date":    sleepEnd.Format(time.RFC3339),
			"category_id": categories["Health"],
			"source":      "template",
			"goal_links": []map[string]any{{
				"goal_id":        goals[sleepGoalTitle],
				"impact_type":    "positive",
				"quantity_value": sleepHours,
			}},
		}

		// Add emotion based on sleep quality
		if sleepHours >= 7.5 {
			payload["emotion_id"] = "emotions:E25" // Well-rested, happy
			if rand.Float32() < 0.3 {
				payload["positives"] = []map[string]any{{"text": "Woke up refreshed"}}
			}
		} else if sleepHours < 6.5 {
			payload["emotion_id"] = "emotions:E72" // Tired
			if rand.Float32() < 0.4 {
				payload["negatives"] = []map[string]any{{"text": "Didn't sleep enough"}}
			}
		}

		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links++
			dailyTotals[sleepGoalTitle] += sleepHours
		}
	}

	// =========================================================================
	// MORNING ROUTINE (6:00-9:00)
	// =========================================================================
	if isPast || dayOffset == 0 {
		// Morning meditation (70% consistency)
		meditationGoalTitle := "Meditate 10 Min Daily"
		if rand.Float32() < 0.7 {
			minutes := float64(10 + rand.Intn(10))
			if createTaskWithDetails(day, 6, 30, int(minutes*60), "Morning Meditation", "🧘",
				categories["Personal"], &minutes, "units:min", meditationGoalTitle, goals, isPast, nil, nil, "template") {
				tasks++
				links++
				dailyTotals[meditationGoalTitle] += minutes
			}
		}

		// Breakfast (healthy meal)
		mealGoalTitle := "Cook Healthy Meals"
		if rand.Float32() < 0.8 {
			mealCount := 1.0
			if createTaskWithDetails(day, 7, 0, 30, "Breakfast", "🍳",
				categories["Health"], &mealCount, "units:count", mealGoalTitle, goals, isPast, nil, nil, "quick") {
				tasks++
				links++
				dailyTotals[mealGoalTitle] += mealCount
			}
		}

		// Morning commute (weekdays only)
		if !isWeekend {
			commuteMin := 25 + rand.Intn(20) // 25-45 min
			if createTaskWithDetails(day, 8, 0, commuteMin, "Commute to Work", "🚗",
				categories["Work"], nil, "", "", nil, isPast, nil, nil, "manual") {
				tasks++
			}
		}
	}

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
	// MULTIPLE SNACK/DRINK LOGS (Throughout day - realistic overlaps)
	// =========================================================================
	if rand.Float32() < 0.6 {
		// Mid-morning snack
		if createTaskWithDetails(day, 10, 30, 10, "Morning Snack", "🍎",
			categories["Personal"], nil, "", "", nil, isPast, nil, nil, "quick") {
			tasks++
		}
	}
	if rand.Float32() < 0.5 {
		// Afternoon tea/snack
		if createTaskWithDetails(day, 15, 30, 15, "Afternoon Tea", "🍵",
			categories["Personal"], nil, "", "", nil, isPast, nil, nil, "quick") {
			tasks++
		}
	}

	// =========================================================================
	// MICRO-BREAKS & BATHROOM BREAKS (Realistic daily interruptions)
	// =========================================================================
	if !isWeekend && rand.Float32() < 0.5 {
		// Quick bathroom/stretch break
		if createTaskWithDetails(day, 10+rand.Intn(6), 0, 5, "Quick Break", "🚶",
			categories["Personal"], nil, "", "", nil, isPast, nil, nil, "quick") {
			tasks++
		}
	}
	if !isWeekend && rand.Float32() < 0.4 {
		// Another micro-break
		if createTaskWithDetails(day, 14+rand.Intn(3), 0, 5, "Stretch Break", "🧘",
			categories["Personal"], nil, "", "", nil, isPast, nil, nil, "quick") {
			tasks++
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
	// WORK TASKS (Weekdays only - MUCH MORE DETAILED)
	// ---------------------------
	if !isWeekend {
		// Morning email check (realistic overlap with morning)
		if rand.Float32() < 0.7 {
			emailDur := 15 + rand.Intn(20)
			negatives := []map[string]any{}
			if rand.Float32() < 0.2 {
				negatives = []map[string]any{{"text": "Too many emails to process"}}
			}
			if createTaskWithDetails(day, 9, 0, emailDur, "Check Emails", "📧",
				categories["Work"], nil, "", "", nil, isPast, nil, negatives, "quick") {
				tasks++
			}
		}

		// Daily Standup
		var standupPositives, standupNegatives []map[string]any
		if rand.Float32() < 0.2 {
			standupPositives = []map[string]any{{"text": "Clear priorities for the day"}}
		}
		if rand.Float32() < 0.1 {
			standupNegatives = []map[string]any{{"text": "Meeting ran long"}}
		}

		if createTaskWithDetails(day, 9, 30, 15, "Daily Standup", "👥", categories["Work"],
			nil, "", "", nil, isPast, standupPositives, standupNegatives, "manual") {
			tasks++
		}

		// Slack/Teams messages throughout day (multiple quick logs)
		slackCount := 2 + rand.Intn(4) // 2-5 Slack sessions
		for i := 0; i < slackCount; i++ {
			hour := 10 + i*2 + rand.Intn(2)
			dur := 5 + rand.Intn(15)
			if createTaskWithDetails(day, hour, 0, dur, "Team Chat", "💬",
				categories["Work"], nil, "", "", nil, isPast, nil, nil, "quick") {
				tasks++
			}
		}

		// Deep Work Sessions (multiple throughout day)
		deepWorkCount := 1 + rand.Intn(3) // 1-3 deep work sessions
		for i := 0; i < deepWorkCount; i++ {
			startHour := 10 + i*3
			minutes := float64(45 + rand.Intn(75)) // 45-120 min
			journal := ""
			if rand.Float32() < 0.4 {
				journal = randomElement([]string{
					"Focused on feature implementation. Good progress.",
					"Debugging session. Found and fixed the issue.",
					"Code review and documentation updates.",
					"Architecture planning and design work.",
					"Refactoring legacy code. Making good improvements.",
					"Writing tests for new features.",
					"Performance optimization work.",
					"API design and implementation.",
				})
			}

			positives := []map[string]any{}
			negatives := []map[string]any{}
			if rand.Float32() < 0.5 {
				positives = []map[string]any{{"text": randomElement([]string{
					"Flow state achieved",
					"Completed ahead of schedule",
					"Clean code written",
					"Tests passing",
					"Good progress made",
					"Solved complex problem",
				}), "emotion_id": randomElement([]string{"emotions:E25", "emotions:E35", "emotions:E44"})}}
			}
			if rand.Float32() < 0.2 {
				negatives = []map[string]any{{"text": randomElement([]string{
					"Too many interruptions",
					"Unexpected complexity",
					"Need to revisit approach",
					"Blocked by dependencies",
					"Struggling with unclear requirements",
				}), "emotion_id": randomElement([]string{"emotions:E52", "emotions:E61", "emotions:E71"})}}
			}

			if createTaskWithDetailsAndJournal(day, startHour, 0, int(minutes), "Deep Work Session", "🎯",
				categories["Work"], &minutes, "units:min", "20 Hours Deep Work Weekly", goals, isPast, positives, negatives, journal, "template") {
				tasks++
				links++
			}
		}

		// Code Reviews (realistic for developers)
		if rand.Float32() < 0.6 {
			dur := 20 + rand.Intn(40)
			positives := []map[string]any{}
			if rand.Float32() < 0.3 {
				positives = []map[string]any{{"text": "Caught important issues"}}
			}
			if createTaskWithDetails(day, 14+rand.Intn(3), 0, dur, "Code Review", "🔍",
				categories["Work"], nil, "", "", nil, isPast, positives, nil, "manual") {
				tasks++
			}
		}

		// Meetings (2-4 per day with variety)
		numMeetings := 2 + rand.Intn(3)
		meetingTypes := []struct {
			title, icon string
			emotions    []string
		}{
			{"Team Sync", "👥", []string{"emotions:E15", "emotions:E25"}},
			{"1:1 Meeting", "🤝", []string{"emotions:E25", "emotions:E35"}},
			{"Project Review", "📊", []string{"emotions:E44", "emotions:E52"}},
			{"Planning Session", "📋", []string{"emotions:E15", "emotions:E44"}},
			{"Client Call", "📞", []string{"emotions:E52", "emotions:E25"}},
			{"Sprint Planning", "🎯", []string{"emotions:E15", "emotions:E44"}},
			{"Retrospective", "🔄", []string{"emotions:E15", "emotions:E25"}},
			{"Tech Discussion", "💡", []string{"emotions:E44", "emotions:E35"}},
		}

		for i := 0; i < numMeetings; i++ {
			hour := 11 + i*2 + rand.Intn(2)
			dur := 30 + rand.Intn(30)
			meeting := meetingTypes[rand.Intn(len(meetingTypes))]

			positives := []map[string]any{}
			negatives := []map[string]any{}
			var emotionID string

			if rand.Float32() < 0.3 {
				positives = []map[string]any{{"text": randomElement([]string{
					"Productive discussion",
					"Clear action items",
					"Good collaboration",
					"Made important decisions",
				})}}
				emotionID = meeting.emotions[0]
			}
			if rand.Float32() < 0.15 {
				negatives = []map[string]any{{"text": randomElement([]string{
					"Meeting could have been an email",
					"Ran over time",
					"Off-topic discussions",
					"No clear outcomes",
				})}}
				emotionID = "emotions:E61"
			}

			if createTaskWithEmotionAndReflections(day, hour, 0, dur, meeting.title, meeting.icon,
				categories["Work"], nil, "", "", nil, isPast, positives, negatives, emotionID, "manual") {
				tasks++
			}
		}

		// Afternoon email processing
		if rand.Float32() < 0.6 {
			emailDur := 10 + rand.Intn(15)
			if createTaskWithDetails(day, 16+rand.Intn(2), 0, emailDur, "Process Emails", "📧",
				categories["Work"], nil, "", "", nil, isPast, nil, nil, "quick") {
				tasks++
			}
		}

		// End of day wrap-up tasks
		if rand.Float32() < 0.4 {
			dur := 10 + rand.Intn(15)
			if createTaskWithDetails(day, 17, 0, dur, "Update Task Board", "📝",
				categories["Work"], nil, "", "", nil, isPast, nil, nil, "manual") {
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
	// LUNCH BREAK (Weekdays)
	// ---------------------------
	if !isWeekend && (isPast || dayOffset == 0) {
		lunchHour := 12 + rand.Intn(2) // 12:00-14:00
		lunchDur := 30 + rand.Intn(30) // 30-60 min

		// Sometimes cook, sometimes eat out
		if rand.Float32() < 0.6 {
			mealCount := 1.0
			if createTaskWithDetails(day, lunchHour, 0, lunchDur, "Lunch", "🍜",
				categories["Health"], &mealCount, "units:count", "Cook Healthy Meals", goals, isPast, nil, nil, "quick") {
				tasks++
				links++
			}
		} else {
			if createTaskWithDetails(day, lunchHour, 0, lunchDur, "Lunch Out", "🍽️",
				categories["Personal"], nil, "", "", nil, isPast, nil, nil, "manual") {
				tasks++
			}
		}
	}

	// ---------------------------
	// AFTERNOON BREAKS & SNACKS
	// ---------------------------
	if !isWeekend && rand.Float32() < 0.7 {
		// Quick walk or break
		if createTaskWithDetails(day, 15, 0, 15, "Afternoon Walk", "🚶",
			categories["Health"], nil, "", "", nil, isPast, nil, nil, "manual") {
			tasks++
		}
	}

	// ---------------------------
	// EVENING COMMUTE (Weekdays)
	// ---------------------------
	if !isWeekend && (isPast || dayOffset == 0) {
		commuteMin := 30 + rand.Intn(25) // Evening traffic is worse
		if createTaskWithDetails(day, 17+rand.Intn(2), 0, commuteMin, "Commute Home", "🚗",
			categories["Work"], nil, "", "", nil, isPast, nil, nil, "manual") {
			tasks++
		}
	}

	// ---------------------------
	// DINNER (Every day)
	// ---------------------------
	if isPast || dayOffset == 0 {
		dinnerHour := 18 + rand.Intn(2)
		dinnerDur := 45 + rand.Intn(30)

		// Most dinners are home-cooked
		if rand.Float32() < 0.7 {
			mealCount := 1.0
			positives := []map[string]any{}
			if rand.Float32() < 0.3 {
				positives = []map[string]any{{"text": randomElement([]string{
					"Tried new recipe", "Family dinner", "Delicious meal", "Healthy and satisfying",
				})}}
			}

			if createTaskWithDetails(day, dinnerHour, 0, dinnerDur, "Dinner", "🍝",
				categories["Health"], &mealCount, "units:count", "Cook Healthy Meals", goals, isPast, positives, nil, "quick") {
				tasks++
				links++
			}
		}
	}

	// ---------------------------
	// DAILY STEPS TRACKING
	// ---------------------------
	stepsGoalTitle := "Walk 10,000 Steps Daily"
	if isPast || dayOffset == 0 {
		// Combine all daily movement into step count
		baseSteps := 5000.0 + float64(rand.Intn(5000)) // 5k-10k base
		// Add bonus steps for active days
		if dailyTotals["Exercise 30 min Daily"] > 30 || dailyTotals["Run 100km This Month"] > 0 {
			baseSteps += float64(rand.Intn(5000)) // 0-5k bonus
		}

		steps := baseSteps
		payload := map[string]any{
			"title":       "Daily Steps",
			"start_date":  day.Add(6 * time.Hour).Format(time.RFC3339),
			"end_date":    day.Add(22 * time.Hour).Format(time.RFC3339),
			"category_id": categories["Health"],
			"source":      "template",
			"goal_links": []map[string]any{{
				"goal_id":        goals[stepsGoalTitle],
				"impact_type":    "positive",
				"quantity_value": steps,
			}},
		}

		if steps >= 10000 {
			payload["positives"] = []map[string]any{{"text": "Hit my step goal!"}}
		}

		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links++
		}
	}

	// ---------------------------
	// SOCIAL MEDIA TRACKING (Avoidance goal)
	// ---------------------------
	socialMediaGoalTitle := "Limit Social Media"
	if rand.Float32() < 0.9 { // Most days
		minutes := float64(20 + rand.Intn(80)) // 20-100 min (sometimes exceeds limit)
		hour := 19 + rand.Intn(3)

		negatives := []map[string]any{}
		if minutes > 60 {
			negatives = []map[string]any{{"text": "Spent too much time scrolling"}}
		}

		if createTaskWithDetails(day, hour, 0, int(minutes), "Social Media", "📱",
			categories["Personal"], &minutes, "units:min", socialMediaGoalTitle, goals, isPast, nil, negatives, "template") {
			tasks++
			links++
		}
	}

	// ---------------------------
	// EVENING ACTIVITIES (Various)
	// ---------------------------
	if rand.Float32() < 0.4 {
		activities := []struct {
			title, icon string
			duration    int
		}{
			{"Watch TV Show", "📺", 45},
			{"Video Games", "🎮", 60},
			{"YouTube", "🎬", 30},
			{"Podcast", "🎧", 40},
			{"Journal Writing", "📝", 20},
		}
		activity := activities[rand.Intn(len(activities))]

		if createTaskWithDetails(day, 20+rand.Intn(2), 0, activity.duration, activity.title, activity.icon,
			categories["Personal"], nil, "", "", nil, isPast, nil, nil, "manual") {
			tasks++
		}
	}

	// ---------------------------
	// LEARNING ACTIVITIES (Weekends - online courses)
	// ---------------------------
	learningGoalTitle := "Learn 5 Hours Weekly"
	if isWeekend && rand.Float32() < 0.7 {
		hours := 1.0 + rand.Float64()*2.0 // 1-3 hours
		minutes := hours * 60
		topics := []string{"Go Advanced Patterns", "System Design", "Cloud Architecture", "Database Optimization",
			"React Deep Dive", "GraphQL APIs", "Kubernetes", "Security Best Practices"}
		topic := randomElement(topics)

		positives := []map[string]any{{"text": "Learned something new", "emotion_id": "emotions:E44"}}
		journal := fmt.Sprintf("Studied: %s. Great content!", topic)

		payload := map[string]any{
			"title":       fmt.Sprintf("Online Course: %s", topic),
			"start_date":  day.Add(time.Duration(10+rand.Intn(4)) * time.Hour).Format(time.RFC3339),
			"end_date":    day.Add(time.Duration(10+rand.Intn(4))*time.Hour + time.Duration(minutes)*time.Minute).Format(time.RFC3339),
			"category_id": categories["Learning"],
			"source":      "manual",
			"journal":     journal,
			"positives":   positives,
			"goal_links": []map[string]any{{
				"goal_id":        goals[learningGoalTitle],
				"impact_type":    "positive",
				"quantity_value": hours,
			}},
		}

		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links++
		}
	}

	// ---------------------------
	// PERSONAL ERRANDS (Weekends - more variety)
	// ---------------------------
	if isWeekend {
		// Multiple errands per weekend day
		numErrands := 1 + rand.Intn(3) // 1-3 errands
		errands := []struct {
			title, icon string
			duration    int
		}{
			{"Grocery Shopping", "🛒", 60},
			{"House Cleaning", "🧹", 90},
			{"Cooking Meal Prep", "🍳", 120},
			{"Laundry", "👔", 45},
			{"Car Maintenance", "🚗", 30},
			{"Garden Work", "🌱", 60},
			{"Home Repairs", "🔧", 90},
			{"Organize Closet", "📦", 60},
		}

		for i := 0; i < numErrands; i++ {
			errand := errands[rand.Intn(len(errands))]
			startHour := 9 + i*3 + rand.Intn(2)

			if createTaskWithDetails(day, startHour, 0, errand.duration, errand.title, errand.icon,
				categories["Personal"], nil, "", "", nil, isPast, nil, nil, "manual") {
				tasks++
			}
		}
	}

	// ---------------------------
	// HOBBY ACTIVITIES (Weekends - Multiple with different emotions)
	// ---------------------------
	if isWeekend {
		// Morning hobby/activity
		if rand.Float32() < 0.6 {
			morningHobbies := []struct {
				title, icon, emotionID string
				duration               int
				positives              []map[string]any
			}{
				{"Photography Walk", "📷", "emotions:E35", 90, []map[string]any{{"text": "Captured beautiful shots", "emotion_id": "emotions:E35"}}},
				{"Morning Yoga", "🧘", "emotions:E15", 45, []map[string]any{{"text": "Feeling centered and calm", "emotion_id": "emotions:E15"}}},
				{"Bike Ride", "🚴", "emotions:E44", 60, []map[string]any{{"text": "Great exercise and fresh air", "emotion_id": "emotions:E44"}}},
				{"Farmers Market", "🥕", "emotions:E25", 75, []map[string]any{{"text": "Found fresh produce", "emotion_id": "emotions:E25"}}},
			}
			hobby := morningHobbies[rand.Intn(len(morningHobbies))]

			if createTaskWithEmotionAndReflections(day, 8+rand.Intn(3), 0, hobby.duration, hobby.title, hobby.icon,
				categories["Personal"], nil, "", "", nil, isPast, hobby.positives, nil, hobby.emotionID, "manual") {
				tasks++
			}
		}

		// Afternoon hobby (overlapping with other activities)
		if rand.Float32() < 0.7 {
			afternoonHobbies := []struct {
				title, icon, emotionID string
				duration               int
				positives, negatives   []map[string]any
			}{
				{"Playing Guitar", "🎸", "emotions:E25", 60, []map[string]any{{"text": "Learned new song", "emotion_id": "emotions:E25"}}, nil},
				{"Painting", "🎨", "emotions:E35", 120, []map[string]any{{"text": "Creative flow achieved", "emotion_id": "emotions:E35"}}, nil},
				{"Gardening", "🌻", "emotions:E15", 75, []map[string]any{{"text": "Relaxing and productive", "emotion_id": "emotions:E15"}}, nil},
				{"Baking", "🧁", "emotions:E44", 90, []map[string]any{{"text": "Delicious results!", "emotion_id": "emotions:E44"}}, nil},
				{"Board Games", "🎲", "emotions:E26", 120, []map[string]any{{"text": "Fun and competitive", "emotion_id": "emotions:E26"}}, nil},
				{"Reading Book", "📖", "emotions:E15", 90, []map[string]any{{"text": "Engaging story", "emotion_id": "emotions:E15"}}, nil},
				{"Woodworking", "🔨", "emotions:E44", 150, []map[string]any{{"text": "Made good progress", "emotion_id": "emotions:E44"}}, []map[string]any{{"text": "Tool malfunction was frustrating", "emotion_id": "emotions:E61"}}},
			}
			hobby := afternoonHobbies[rand.Intn(len(afternoonHobbies))]

			if createTaskWithEmotionAndReflections(day, 13+rand.Intn(4), 0, hobby.duration, hobby.title, hobby.icon,
				categories["Personal"], nil, "", "", nil, isPast, hobby.positives, hobby.negatives, hobby.emotionID, "manual") {
				tasks++
			}
		}

		// Evening relaxation or entertainment (testing all emotion quadrants)
		if rand.Float32() < 0.8 {
			eveningActivities := []struct {
				title, icon, emotionID string
				duration               int
				positives, negatives   []map[string]any
			}{
				// Yellow quadrant (high energy, pleasant)
				{"Dance Party at Home", "💃", "emotions:E26", 60, []map[string]any{{"text": "High energy, felt great!", "emotion_id": "emotions:E26"}}, nil},
				{"Exciting Movie", "🎬", "emotions:E36", 120, []map[string]any{{"text": "Edge of my seat!", "emotion_id": "emotions:E36"}}, nil},
				// Green quadrant (low energy, pleasant)
				{"Relaxing Bath", "🛁", "emotions:E05", 45, []map[string]any{{"text": "Very relaxing", "emotion_id": "emotions:E05"}}, nil},
				{"Calm Documentary", "📺", "emotions:E15", 90, []map[string]any{{"text": "Interesting and peaceful", "emotion_id": "emotions:E15"}}, nil},
				// Red quadrant (high energy, unpleasant) - occasional stressful activities
				{"Intense Horror Movie", "😱", "emotions:E86", 110, nil, []map[string]any{{"text": "Too scary, couldn't sleep", "emotion_id": "emotions:E86"}}},
				{"Argued about Plans", "😤", "emotions:E73", 30, nil, []map[string]any{{"text": "Disagreement with partner", "emotion_id": "emotions:E73"}}},
				// Blue quadrant (low energy, unpleasant) - occasional down time
				{"Feeling Lonely", "😔", "emotions:E95", 60, nil, []map[string]any{{"text": "Missing friends", "emotion_id": "emotions:E95"}}},
				{"Bored Scrolling", "📱", "emotions:E85", 45, nil, []map[string]any{{"text": "Wasted time scrolling", "emotion_id": "emotions:E85"}}},
			}
			activity := eveningActivities[rand.Intn(len(eveningActivities))]

			if createTaskWithEmotionAndReflections(day, 19+rand.Intn(3), 0, activity.duration, activity.title, activity.icon,
				categories["Personal"], nil, "", "", nil, isPast, activity.positives, activity.negatives, activity.emotionID, "manual") {
				tasks++
			}
		}
	}

	// ---------------------------
	// SOCIAL ACTIVITIES (Link to weekly social goal)
	// ---------------------------
	socialGoalTitle := "Connect with Friends Weekly"
	socialChance := float32(0.25)
	if isWeekend {
		socialChance = 0.4 // Higher on weekends
	}

	if rand.Float32() < socialChance {
		activities := []struct {
			title, icon string
			duration    int
		}{
			{"Dinner with Friends", "🍽️", 120},
			{"Movie Night", "🎬", 150},
			{"Phone Call with Family", "📞", 45},
			{"Game Night", "🎮", 180},
			{"Coffee with Friend", "☕", 90},
			{"Video Call", "💻", 60},
			{"Party", "🎉", 180},
			{"Brunch", "🥞", 120},
		}
		activity := activities[rand.Intn(len(activities))]
		hour := 18 + rand.Intn(3)
		if isWeekend {
			hour = 11 + rand.Intn(8) // More flexible timing on weekends
		}

		positives := []map[string]any{{"text": "Quality time with loved ones", "emotion_id": "emotions:E16"}}
		emotionID := randomElement([]string{"emotions:E25", "emotions:E26", "emotions:E35", "emotions:E36"})

		socialCount := 1.0
		payload := map[string]any{
			"title":       activity.title,
			"start_date":  day.Add(time.Duration(hour) * time.Hour).Format(time.RFC3339),
			"end_date":    day.Add(time.Duration(hour)*time.Hour + time.Duration(activity.duration)*time.Minute).Format(time.RFC3339),
			"category_id": categories["Personal"],
			"source":      "manual",
			"emotion_id":  emotionID,
			"positives":   positives,
			"goal_links": []map[string]any{{
				"goal_id":        goals[socialGoalTitle],
				"impact_type":    "positive",
				"quantity_value": socialCount,
			}},
		}

		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links++
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

	// =========================================================================
	// NEW CATEGORIES & GOALS - REALISTIC DAILY LIFE
	// =========================================================================

	// FAMILY ACTIVITIES (Sundays and random)
	if day.Weekday() == time.Sunday && rand.Float32() < 0.8 {
		familyCount := 1.0
		positives := []map[string]any{
			{"text": "Quality time together", "emotion_id": "emotions:E25"},
			{"text": "Good conversations", "emotion_id": "emotions:E35"},
		}
		payload := map[string]any{
			"title":       "Family Dinner",
			"start_date":  day.Add(18*time.Hour + 30*time.Minute).Format(time.RFC3339),
			"end_date":    day.Add(20 * time.Hour).Format(time.RFC3339),
			"category_id": categories["Family"],
			"source":      "manual",
			"emotion_id":  "emotions:E25",
			"positives":   positives,
			"goal_links": []map[string]any{{
				"goal_id":        goals["Weekly Family Dinner"],
				"impact_type":    "positive",
				"quantity_value": familyCount,
			}},
		}
		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links++
		}
	}

	// Call parents weekly (randomized during week)
	if rand.Float32() < 0.15 {
		callCount := 1.0
		dur := 20 + rand.Intn(40) // 20-60 min calls
		positives := []map[string]any{{"text": "Caught up on family news", "emotion_id": "emotions:E25"}}

		payload := map[string]any{
			"title":       "Call with Parents",
			"start_date":  day.Add(time.Duration(19+rand.Intn(3))*time.Hour + 15*time.Minute).Format(time.RFC3339),
			"end_date":    day.Add(time.Duration(19+rand.Intn(3))*time.Hour + time.Duration(15+dur)*time.Minute).Format(time.RFC3339),
			"category_id": categories["Family"],
			"source":      "manual",
			"emotion_id":  "emotions:E15",
			"positives":   positives,
			"goal_links": []map[string]any{{
				"goal_id":        goals["Call Parents Weekly"],
				"impact_type":    "positive",
				"quantity_value": callCount,
			}},
		}
		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links++
		}
	}

	// HOBBIES - Guitar practice
	if rand.Float32() < 0.4 {
		practiceCount := 1.0
		minutes := 30.0 + float64(rand.Intn(30)) // 30-60 min
		positives := []map[string]any{{"text": randomElement([]string{
			"Learned new chord progression",
			"Nailed that difficult part",
			"Felt in the zone",
			"Making progress on song",
		}), "emotion_id": "emotions:E35"}}

		payload := map[string]any{
			"title":       "Guitar Practice",
			"start_date":  day.Add(time.Duration(19+rand.Intn(3)) * time.Hour).Format(time.RFC3339),
			"end_date":    day.Add(time.Duration(19+rand.Intn(3))*time.Hour + time.Duration(minutes)*time.Minute).Format(time.RFC3339),
			"category_id": categories["Hobbies"],
			"source":      "manual",
			"emotion_id":  "emotions:E35",
			"positives":   positives,
			"goal_links": []map[string]any{{
				"goal_id":        goals["Practice Guitar 3x Weekly"],
				"impact_type":    "positive",
				"quantity_value": practiceCount,
			}},
		}
		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links++
		}
	}

	// HOME - Weekly deep clean (Saturdays)
	if day.Weekday() == time.Saturday && rand.Float32() < 0.7 {
		cleanCount := 1.0
		dur := 90 + rand.Intn(60) // 90-150 min
		positives := []map[string]any{{"text": "House feels fresh and organized", "emotion_id": "emotions:E15"}}
		negatives := []map[string]any{}
		if rand.Float32() < 0.3 {
			negatives = []map[string]any{{"text": "Exhausting work", "emotion_id": "emotions:E72"}}
		}

		payload := map[string]any{
			"title":       "Deep Clean House",
			"start_date":  day.Add(time.Duration(10+rand.Intn(3)) * time.Hour).Format(time.RFC3339),
			"end_date":    day.Add(time.Duration(10+rand.Intn(3))*time.Hour + time.Duration(dur)*time.Minute).Format(time.RFC3339),
			"category_id": categories["Home"],
			"source":      "manual",
			"emotion_id":  "emotions:E15",
			"positives":   positives,
			"negatives":   negatives,
			"goal_links": []map[string]any{{
				"goal_id":        goals["Deep Clean Home Weekly"],
				"impact_type":    "positive",
				"quantity_value": cleanCount,
			}},
		}
		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links++
		}
	}

	// FITNESS - Strength training (3x week)
	if !isWeekend && rand.Float32() < 0.4 {
		workoutCount := 1.0
		minutes := 45.0 + float64(rand.Intn(30)) // 45-75 min

		// Complex task with MULTIPLE reflections (5+ items)
		positives := []map[string]any{
			{"text": "Hit new PR on squats!", "emotion_id": "emotions:E35"},
			{"text": "Felt strong throughout", "emotion_id": "emotions:E44"},
			{"text": "Good form maintained", "emotion_id": "emotions:E25"},
		}
		negatives := []map[string]any{}
		if rand.Float32() < 0.2 {
			negatives = []map[string]any{
				{"text": "Gym was crowded", "emotion_id": "emotions:E61"},
			}
		}

		payload := map[string]any{
			"title":       "Strength Training",
			"start_date":  day.Add(time.Duration(17+rand.Intn(2)) * time.Hour).Format(time.RFC3339),
			"end_date":    day.Add(time.Duration(17+rand.Intn(2))*time.Hour + time.Duration(minutes)*time.Minute).Format(time.RFC3339),
			"category_id": categories["Fitness"],
			"source":      "manual",
			"emotion_id":  "emotions:E44",
			"positives":   positives,
			"negatives":   negatives,
			"goal_links": []map[string]any{
				{
					"goal_id":        goals["Strength Training 3x Week"],
					"impact_type":    "positive",
					"quantity_value": workoutCount,
				},
				{
					"goal_id":        goals["Exercise 30 min Daily"],
					"impact_type":    "positive",
					"quantity_value": minutes,
				},
			},
		}
		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links += 2
		}
	}

	// MINDFULNESS - Daily gratitude journal
	if rand.Float32() < 0.6 {
		gratitudeCount := 1.0
		// Complex journal entry with multiple things
		journal := randomElement([]string{
			"Grateful for: 1) Healthy body 2) Supportive family 3) Meaningful work",
			"Today I'm thankful for: 1) Morning coffee 2) Good conversation with friend 3) Beautiful weather",
			"Three blessings: 1) My health 2) Roof over my head 3) Opportunities to learn",
			"Gratitude list: 1) Progress on goals 2) Kind words from colleague 3) Peaceful evening",
		})

		payload := map[string]any{
			"title":       "Gratitude Journaling",
			"start_date":  day.Add(21*time.Hour + 30*time.Minute).Format(time.RFC3339),
			"end_date":    day.Add(21*time.Hour + 40*time.Minute).Format(time.RFC3339),
			"category_id": categories["Mindfulness"],
			"source":      "manual",
			"journal":     journal,
			"emotion_id":  "emotions:E15",
			"goal_links": []map[string]any{{
				"goal_id":        goals["Gratitude Journal Daily"],
				"impact_type":    "positive",
				"quantity_value": gratitudeCount,
			}},
		}
		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links++
		}
	}

	// FINANCE - Expense tracking (daily)
	if rand.Float32() < 0.8 {
		expenseCount := 1.0
		spent := 20.0 + float64(rand.Intn(100)) // Random daily spending
		note := fmt.Sprintf("Tracked $%.2f in expenses today", spent)

		payload := map[string]any{
			"title":       "Daily Expense Log",
			"start_date":  day.Add(20 * time.Hour).Format(time.RFC3339),
			"end_date":    day.Add(20*time.Hour + 10*time.Minute).Format(time.RFC3339),
			"category_id": categories["Finance"],
			"source":      "quick",
			"note":        note,
			"goal_links": []map[string]any{{
				"goal_id":        goals["Track Expenses Daily"],
				"impact_type":    "positive",
				"quantity_value": expenseCount,
			}},
		}
		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links++
		}
	}

	// =========================================================================
	// EDGE CASES & SPECIAL SCENARIOS (Testing all features)
	// =========================================================================

	// Task with NO CATEGORY (edge case)
	if rand.Float32() < 0.1 {
		if createTaskWithDetails(day, 12+rand.Intn(8), 0, 10, "Random Thought", "💭",
			"", nil, "", "", nil, isPast, nil, nil, "manual") {
			tasks++
		}
	}

	// Task with MANY reflections (both positives and negatives)
	if rand.Float32() < 0.15 {
		manyPositives := []map[string]any{
			{"text": "First positive thing", "emotion_id": "emotions:E25"},
			{"text": "Second positive thing", "emotion_id": "emotions:E35"},
			{"text": "Third positive thing", "emotion_id": "emotions:E44"},
		}
		manyNegatives := []map[string]any{
			{"text": "First challenge", "emotion_id": "emotions:E61"},
			{"text": "Second frustration", "emotion_id": "emotions:E71"},
		}
		if createTaskWithDetails(day, 14, 0, 120, "Complex Mixed-Emotion Event", "🎭",
			categories["Personal"], nil, "", "", nil, isPast, manyPositives, manyNegatives, "manual") {
			tasks++
		}
	}

	// Very SHORT task (< 5 minutes)
	if rand.Float32() < 0.3 {
		shortTasks := []string{"Quick Note", "Take Pill", "Check Weather", "Set Reminder"}
		title := randomElement(shortTasks)
		if createTaskWithDetails(day, 8+rand.Intn(12), 0, 1+rand.Intn(3), title, "⚡",
			categories["Personal"], nil, "", "", nil, isPast, nil, nil, "quick") {
			tasks++
		}
	}

	// Very LONG task (4+ hours)
	if rand.Float32() < 0.1 {
		longTasks := []struct {
			title, icon string
			duration    int
		}{
			{"Long Drive to Visit Family", "🚗", 240},
			{"All-Day Workshop", "📚", 360},
			{"Home Renovation Project", "🔨", 300},
			{"Marathon Gaming Session", "🎮", 480},
		}
		longTask := longTasks[rand.Intn(len(longTasks))]

		journal := "This took much longer than expected, but it was worth it."
		positives := []map[string]any{{"text": "Accomplished something big"}}

		if createTaskWithDetailsAndJournal(day, 10, 0, longTask.duration, longTask.title, longTask.icon,
			categories["Personal"], nil, "", "", nil, isPast, positives, nil, journal, "manual") {
			tasks++
		}
	}

	// OVERLAPPING concurrent tasks (multitasking - realistic scenario)
	if rand.Float32() < 0.2 {
		// Start background task (like laundry) while doing something else
		if createTaskWithDetails(day, 10, 0, 90, "Laundry Running", "🧺",
			categories["Personal"], nil, "", "", nil, isPast, nil, nil, "quick") {
			tasks++
		}
		// Do another task at same time
		if createTaskWithDetails(day, 10, 15, 45, "Clean Kitchen While Waiting", "🧹",
			categories["Personal"], nil, "", "", nil, isPast, nil, nil, "manual") {
			tasks++
		}
	}

	// Task with LONG journal entry (testing journal field)
	if rand.Float32() < 0.1 {
		longJournal := `Today was a reflective day. I spent time thinking about my goals and where I'm heading.

There were several insights that came to mind:
1. I need to be more intentional with my time
2. Quality over quantity in relationships
3. Regular exercise really does improve my mood

I also realized that I've been too hard on myself lately. Progress is not always linear, and that's okay.
The important thing is to keep moving forward, even if it's just small steps.

Looking forward to tomorrow with a renewed sense of purpose.`

		positives := []map[string]any{{"text": "Gained clarity and perspective", "emotion_id": "emotions:E15"}}

		if createTaskWithDetailsAndJournal(day, 21, 0, 30, "Evening Reflection & Journaling", "📔",
			categories["Personal"], nil, "", "", nil, isPast, positives, nil, longJournal, "manual") {
			tasks++
		}
	}

	// Task linking to MULTIPLE goals (already done with running, but adding more variety)
	if rand.Float32() < 0.1 && !isWeekend {
		// Lunch walk - counts for steps AND social if with colleague
		steps := 2000.0 + float64(rand.Intn(1000))
		socialCount := 1.0

		payload := map[string]any{
			"title":       "Walking Meeting with Colleague",
			"start_date":  day.Add(12*time.Hour + 30*time.Minute).Format(time.RFC3339),
			"end_date":    day.Add(12*time.Hour + 50*time.Minute).Format(time.RFC3339),
			"category_id": categories["Work"],
			"source":      "manual",
			"goal_links": []map[string]any{
				{
					"goal_id":        goals["Walk 10,000 Steps Daily"],
					"impact_type":    "positive",
					"quantity_value": steps,
				},
				{
					"goal_id":        goals["Connect with Friends Weekly"],
					"impact_type":    "positive",
					"quantity_value": socialCount,
				},
			},
		}

		if _, err := apiRequest("POST", "/tasks", payload); err == nil {
			tasks++
			links += 2
		}
	}

	// Task with NO emotion (testing optional field)
	if rand.Float32() < 0.3 {
		// Just a plain activity log without emotion tracking
		if createTaskWithDetails(day, 16+rand.Intn(4), 0, 15, "Water Plants", "🌱",
			categories["Personal"], nil, "", "", nil, isPast, nil, nil, "quick") {
			tasks++
		}
	}

	// Future task with NO start time yet (planning ahead)
	if !isPast && dayOffset > 0 && rand.Float32() < 0.3 {
		futurePlans := []string{"Doctor Appointment", "Car Service", "Dentist", "Hair Cut"}
		title := randomElement(futurePlans)

		if createTaskWithDetails(day, 10+rand.Intn(6), 0, 60, title, "📅",
			categories["Personal"], nil, "", "", nil, false, nil, nil, "manual") {
			tasks++
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
// GOAL STREAK AND HISTORY SEEDING
// =============================================================================

// seedGoalStreaksAndHistory updates goal streaks and creates historical goal logs
func seedGoalStreaksAndHistory(ctx context.Context, db *database.DB, goals map[string]string) int {
	updated := 0
	now := time.Now()

	// Define streak data for habit goals (simulating consistent tracking)
	habitStreaks := map[string]struct {
		streak     int
		daysAgo    int // last completed X days ago
		logEntries int // number of log entries to create
	}{
		"Meditate 10 Min Daily":       {streak: 14, daysAgo: 0, logEntries: 20},
		"Drink 8 Glasses Water":       {streak: 21, daysAgo: 0, logEntries: 25},
		"Sleep 7-8 Hours":             {streak: 12, daysAgo: 0, logEntries: 20},
		"Run 3x Weekly":               {streak: 8, daysAgo: 1, logEntries: 15},
		"Read 30 Min Daily":           {streak: 6, daysAgo: 0, logEntries: 12},
		"Weekly Family Dinner":        {streak: 4, daysAgo: 2, logEntries: 6},
		"Call Parents Weekly":         {streak: 3, daysAgo: 5, logEntries: 5},
		"Cook Healthy Meals":          {streak: 5, daysAgo: 0, logEntries: 10},
		"Limit Social Media":          {streak: 7, daysAgo: 1, logEntries: 10},
		"Weekly Reflection":           {streak: 3, daysAgo: 3, logEntries: 5},
		"Connect with Friends Weekly": {streak: 2, daysAgo: 4, logEntries: 4},
		"Date Night Bi-weekly":        {streak: 2, daysAgo: 10, logEntries: 3},
	}

	// Update streaks for habit goals
	for goalTitle, data := range habitStreaks {
		goalID, ok := goals[goalTitle]
		if !ok {
			continue
		}

		lastCompleted := now.AddDate(0, 0, -data.daysAgo)
		if err := updateGoalStreak(ctx, db, goalID, data.streak, lastCompleted); err != nil {
			log.Warn().Err(err).Str("goal", goalTitle).Msg("Failed to update streak")
			continue
		}

		// Create historical log entries
		for i := 0; i < data.logEntries; i++ {
			// Create logs for past days
			logDate := now.AddDate(0, 0, -(i + data.daysAgo))
			eventType := "completed"
			if i%5 == 4 {
				eventType = "streak_milestone"
			}

			changes := map[string]any{
				"streak_before": i,
				"streak_after":  i + 1,
			}

			if err := createGoalLog(ctx, db, goalID, eventType, changes, logDate); err != nil {
				log.Warn().Err(err).Str("goal", goalTitle).Int("day", i).Msg("Failed to create goal log")
			}
		}

		updated++
	}

	// Update progress for measurable goals
	measurableProgress := map[string]struct {
		current    float64
		target     float64
		logEntries int
	}{
		"20 Hours Deep Work Weekly": {current: 14.5, target: 20, logEntries: 8},
		"Save $5000":                {current: 2350, target: 5000, logEntries: 12},
		"Read 5 Books This Year":    {current: 2, target: 5, logEntries: 5},
		"Run 100km This Month":      {current: 42, target: 100, logEntries: 10},
		"Complete 50 Workouts":      {current: 23, target: 50, logEntries: 15},
		"Learn 500 Spanish Words":   {current: 180, target: 500, logEntries: 20},
	}

	for goalTitle, data := range measurableProgress {
		goalID, ok := goals[goalTitle]
		if !ok {
			continue
		}

		// Update goal with current progress value
		progress := (data.current / data.target) * 100
		_, err := database.QueryAll[any](ctx, db, `
			UPDATE type::thing($id) MERGE {
				current_value: $current,
				updated_at: $now
			}
		`, map[string]any{
			"id":      database.MustRecordID("goals", goalID),
			"current": data.current,
			"now":     now,
		})
		if err != nil {
			log.Warn().Err(err).Str("goal", goalTitle).Msg("Failed to update progress")
			continue
		}

		// Create historical log entries showing progress
		for i := 0; i < data.logEntries; i++ {
			logDate := now.AddDate(0, 0, -(i * 2)) // Every 2 days
			increment := data.current / float64(data.logEntries)

			changes := map[string]any{
				"value_before": increment * float64(i),
				"value_after":  increment * float64(i+1),
				"progress":     progress * float64(i+1) / float64(data.logEntries),
			}

			if err := createGoalLog(ctx, db, goalID, "progress_update", changes, logDate); err != nil {
				log.Warn().Err(err).Str("goal", goalTitle).Msg("Failed to create progress log")
			}
		}

		updated++
	}

	return updated
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
