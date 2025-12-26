// Package main provides a data seeder for development and testing.
//
// This CLI tool populates the database with sample tasks, categories,
// and other data across yesterday, today, and tomorrow for testing.
//
// Usage:
//
//	go run ./cmd/seed
//	go run ./cmd/seed --reset  # Delete existing data first
//	# or
//	task seed
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/config"
	"github.com/lucid-logs/go-backend/internal/shared/database"
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

	log.Info().Msg("🌱 Starting data seeder...")

	// Connect to database
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

	// Find admin user (created by bootstrap)
	userID, err := findAdminUser(ctx, db, cfg.Admin.Username)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to find admin user - make sure backend has been started at least once")
	}
	log.Info().Str("user_id", userID).Str("email", cfg.Admin.Username).Msg("✅ Admin user found")

	// Reset existing data if flag is set
	if *resetFlag {
		log.Info().Msg("🗑️ Resetting existing seeded data...")
		if err := resetSeededData(ctx, db, userID); err != nil {
			log.Fatal().Err(err).Msg("Failed to reset data")
		}
		log.Info().Msg("✅ Existing data deleted")
	}

	// Create categories
	categories, err := createCategories(ctx, db, userID)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create categories")
	}
	log.Info().Int("count", len(categories)).Msg("✅ Categories created")

	// Create tasks for yesterday, today, tomorrow
	taskCount, err := createTasks(ctx, db, userID, categories)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create tasks")
	}
	log.Info().Int("count", taskCount).Msg("✅ Tasks created")

	log.Info().Msg("🎉 Data seeding complete!")
}

func resetSeededData(ctx context.Context, db *database.DB, userID string) error {
	// Delete all tasks for this user
	_, err := database.QueryAll[userResult](ctx, db, `
		DELETE tasks WHERE created_by = $user
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		return err
	}

	// Also clean up orphaned tasks (created by users that no longer exist)
	_, err = database.QueryAll[userResult](ctx, db, `
		DELETE tasks WHERE created_by NOT IN (SELECT id FROM users)
	`, nil)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to cleanup orphaned tasks")
	} else {
		log.Info().Msg("✅ Orphaned tasks cleaned up")
	}

	// Delete all categories for this user
	_, err = database.QueryAll[userResult](ctx, db, `
		DELETE categories WHERE created_by = $user
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		return err
	}

	// Clean up orphaned categories too
	_, err = database.QueryAll[userResult](ctx, db, `
		DELETE categories WHERE created_by NOT IN (SELECT id FROM users)
	`, nil)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to cleanup orphaned categories")
	} else {
		log.Info().Msg("✅ Orphaned categories cleaned up")
	}

	return nil
}

// userResult is used to capture user query results
type userResult struct {
	ID string `json:"id"`
}

func findAdminUser(ctx context.Context, db *database.DB, email string) (string, error) {
	// Find the admin user created by bootstrap
	users, err := database.QueryAll[userResult](ctx, db, `
		SELECT type::string(id) as id FROM users WHERE email = $email LIMIT 1
	`, map[string]any{
		"email": email,
	})
	if err != nil {
		return "", err
	}

	if len(users) == 0 {
		return "", fmt.Errorf("admin user not found (email: %s) - start the backend first to create it", email)
	}

	return users[0].ID, nil
}

type categoryResult struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

func createCategories(ctx context.Context, db *database.DB, userID string) ([]categoryResult, error) {
	categories := []struct {
		name  string
		color string
	}{
		{"Work", "#3B82F6"},     // Blue
		{"Personal", "#10B981"}, // Green
		{"Health", "#EF4444"},   // Red
		{"Learning", "#8B5CF6"}, // Purple
		{"Errands", "#F59E0B"},  // Amber
		{"Social", "#EC4899"},   // Pink
		{"Finance", "#06B6D4"},  // Cyan
		{"Creative", "#F97316"}, // Orange
		{"Travel", "#14B8A6"},   // Teal
		{"Home", "#84CC16"},     // Lime
	}

	var results []categoryResult

	for _, cat := range categories {
		// Check if exists
		existing, err := database.QueryAll[categoryResult](ctx, db, `
			SELECT type::string(id) as id, name, color FROM categories 
			WHERE name = $name AND created_by = type::thing($user) LIMIT 1
		`, map[string]any{
			"name": cat.name,
			"user": userID,
		})
		if err != nil {
			return nil, err
		}

		if len(existing) > 0 {
			results = append(results, existing[0])
			continue
		}

		// Create category
		_, err = database.QueryAll[categoryResult](ctx, db, `
			CREATE categories CONTENT {
				name: $name,
				color: $color,
				created_by: type::thing($user),
				created_at: time::now(),
				updated_at: time::now()
			}
		`, map[string]any{
			"name":  cat.name,
			"color": cat.color,
			"user":  userID,
		})
		if err != nil {
			return nil, err
		}

		// Fetch the created category
		created, err := database.QueryAll[categoryResult](ctx, db, `
			SELECT type::string(id) as id, name, color FROM categories 
			WHERE name = $name AND created_by = type::thing($user) LIMIT 1
		`, map[string]any{
			"name": cat.name,
			"user": userID,
		})
		if err != nil {
			return nil, err
		}

		if len(created) > 0 {
			log.Debug().Str("category", cat.name).Str("id", created[0].ID).Msg("Created category")
			results = append(results, created[0])
		}
	}

	return results, nil
}

func createTasks(ctx context.Context, db *database.DB, userID string, categories []categoryResult) (int, error) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	// Helper to get category by name
	getCat := func(name string) string {
		for _, c := range categories {
			if c.Name == name {
				return c.ID
			}
		}
		return categories[0].ID
	}

	tasks := []struct {
		title     string
		journal   string
		startTime time.Time
		duration  time.Duration
		priority  int
		completed bool
		category  string
	}{
		// =====================================================================
		// YESTERDAY'S TASKS
		// =====================================================================
		{
			title:     "Morning standup meeting",
			journal:   "<p>Discussed sprint progress and blockers</p>",
			startTime: time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 9, 0, 0, 0, now.Location()),
			duration:  30 * time.Minute,
			priority:  7,
			completed: true,
			category:  "Work",
		},
		{
			title:     "Code review for PR #123",
			journal:   "<p>Reviewed authentication changes</p>",
			startTime: time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 10, 0, 0, 0, now.Location()),
			duration:  1 * time.Hour,
			priority:  8,
			completed: true,
			category:  "Work",
		},
		{
			title:     "Client call - Project update",
			journal:   "<p>Updated client on project progress and next steps</p>",
			startTime: time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 14, 0, 0, 0, now.Location()),
			duration:  45 * time.Minute,
			priority:  9,
			completed: true,
			category:  "Work",
		},
		{
			title:     "Gym workout",
			journal:   "<p>Leg day - squats, lunges, calf raises</p>",
			startTime: time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 17, 30, 0, 0, now.Location()),
			duration:  1 * time.Hour,
			priority:  6,
			completed: true,
			category:  "Health",
		},
		{
			title:     "Pay electricity bill",
			journal:   "<p>Monthly utility payment</p>",
			startTime: time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 19, 0, 0, 0, now.Location()),
			duration:  15 * time.Minute,
			priority:  7,
			completed: true,
			category:  "Finance",
		},
		{
			title:     "Read technical article",
			journal:   "<p>Article about Go concurrency patterns</p>",
			startTime: time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 21, 0, 0, 0, now.Location()),
			duration:  30 * time.Minute,
			priority:  4,
			completed: false,
			category:  "Learning",
		},

		// =====================================================================
		// TODAY'S TASKS - Many tasks spread throughout the day
		// =====================================================================
		{
			title:     "Wake up routine",
			journal:   "<p>Morning stretches and healthy breakfast</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 6, 30, 0, 0, now.Location()),
			duration:  30 * time.Minute,
			priority:  5,
			completed: true,
			category:  "Health",
		},
		{
			title:     "Daily planning",
			journal:   "<p>Review priorities and plan the day</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location()),
			duration:  15 * time.Minute,
			priority:  9,
			completed: true,
			category:  "Personal",
		},
		{
			title:     "Check emails and messages",
			journal:   "<p>Process inbox and respond to urgent items</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location()),
			duration:  30 * time.Minute,
			priority:  7,
			completed: true,
			category:  "Work",
		},
		{
			title:     "Morning standup",
			journal:   "<p>Daily sync with the team</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()),
			duration:  15 * time.Minute,
			priority:  8,
			completed: false,
			category:  "Work",
		},
		{
			title:     "Deep work: Feature development",
			journal:   "<p>Focus time on the main feature</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, now.Location()),
			duration:  2 * time.Hour,
			priority:  10,
			completed: false,
			category:  "Work",
		},
		{
			title:     "Team sync meeting",
			journal:   "<p>Weekly team synchronization</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 11, 30, 0, 0, now.Location()),
			duration:  45 * time.Minute,
			priority:  8,
			completed: false,
			category:  "Work",
		},
		{
			title:     "Lunch break",
			journal:   "<p>Take a proper break and eat healthy</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 12, 30, 0, 0, now.Location()),
			duration:  1 * time.Hour,
			priority:  5,
			completed: false,
			category:  "Health",
		},
		{
			title:     "Fix search functionality bug",
			journal:   "<p>Debug the search not triggering API calls</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, now.Location()),
			duration:  2 * time.Hour,
			priority:  9,
			completed: false,
			category:  "Work",
		},
		{
			title:     "Review pull requests",
			journal:   "<p>Check pending PRs and provide feedback</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location()),
			duration:  1 * time.Hour,
			priority:  7,
			completed: false,
			category:  "Work",
		},
		{
			title:     "Grocery shopping",
			journal:   "<p>Buy vegetables, fruits, and essentials</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 17, 30, 0, 0, now.Location()),
			duration:  45 * time.Minute,
			priority:  6,
			completed: false,
			category:  "Errands",
		},
		{
			title:     "Cook dinner",
			journal:   "<p>Prepare a healthy homemade meal</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 18, 30, 0, 0, now.Location()),
			duration:  45 * time.Minute,
			priority:  5,
			completed: false,
			category:  "Home",
		},
		{
			title:     "Call mom",
			journal:   "<p>Weekly check-in call</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 19, 30, 0, 0, now.Location()),
			duration:  30 * time.Minute,
			priority:  7,
			completed: false,
			category:  "Social",
		},
		{
			title:     "Online course: SvelteKit",
			journal:   "<p>Complete module 5 of the course</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, now.Location()),
			duration:  1 * time.Hour,
			priority:  6,
			completed: false,
			category:  "Learning",
		},
		{
			title:     "Evening meditation",
			journal:   "<p>15 minutes of mindfulness</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 21, 0, 0, 0, now.Location()),
			duration:  15 * time.Minute,
			priority:  5,
			completed: false,
			category:  "Health",
		},
		{
			title:     "Journal entry",
			journal:   "<p>Reflect on the day and write thoughts</p>",
			startTime: time.Date(now.Year(), now.Month(), now.Day(), 21, 30, 0, 0, now.Location()),
			duration:  20 * time.Minute,
			priority:  4,
			completed: false,
			category:  "Personal",
		},

		// =====================================================================
		// TOMORROW'S TASKS
		// =====================================================================
		{
			title:     "Morning jog",
			journal:   "<p>5km run around the park</p>",
			startTime: time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 6, 30, 0, 0, now.Location()),
			duration:  45 * time.Minute,
			priority:  7,
			completed: false,
			category:  "Health",
		},
		{
			title:     "Breakfast with colleagues",
			journal:   "<p>Team breakfast at the cafe</p>",
			startTime: time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 8, 0, 0, 0, now.Location()),
			duration:  1 * time.Hour,
			priority:  5,
			completed: false,
			category:  "Social",
		},
		{
			title:     "Project deadline review",
			journal:   "<p>Review upcoming deadlines and adjust priorities</p>",
			startTime: time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 30, 0, 0, now.Location()),
			duration:  1 * time.Hour,
			priority:  9,
			completed: false,
			category:  "Work",
		},
		{
			title:     "Design brainstorming",
			journal:   "<p>Sketch new UI ideas for the dashboard</p>",
			startTime: time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 11, 0, 0, 0, now.Location()),
			duration:  1*time.Hour + 30*time.Minute,
			priority:  7,
			completed: false,
			category:  "Creative",
		},
		{
			title:     "Learn Svelte 5 runes",
			journal:   "<p>Study the new reactivity system</p>",
			startTime: time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 14, 0, 0, 0, now.Location()),
			duration:  90 * time.Minute,
			priority:  6,
			completed: false,
			category:  "Learning",
		},
		{
			title:     "Coffee with Alex",
			journal:   "<p>Catch up with old colleague</p>",
			startTime: time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 16, 0, 0, 0, now.Location()),
			duration:  1 * time.Hour,
			priority:  5,
			completed: false,
			category:  "Social",
		},
		{
			title:     "Budget review",
			journal:   "<p>Review monthly expenses and savings</p>",
			startTime: time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 18, 0, 0, 0, now.Location()),
			duration:  45 * time.Minute,
			priority:  6,
			completed: false,
			category:  "Finance",
		},
		{
			title:     "Clean apartment",
			journal:   "<p>Weekly cleaning routine</p>",
			startTime: time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 19, 0, 0, 0, now.Location()),
			duration:  1 * time.Hour,
			priority:  5,
			completed: false,
			category:  "Home",
		},
	}

	count := 0
	for _, task := range tasks {
		endTime := task.startTime.Add(task.duration)

		// Check if task already exists
		existing, err := database.QueryAll[userResult](ctx, db, `
			SELECT type::string(id) as id FROM tasks 
			WHERE title = $title AND created_by = $user AND start_date = $start LIMIT 1
		`, map[string]any{
			"title": task.title,
			"user":  userID,
			"start": task.startTime,
		})
		if err != nil {
			return count, err
		}

		if len(existing) > 0 {
			continue // Skip existing
		}

		// Create task - use type::thing() to convert string ID to record reference
		_, err = database.QueryAll[userResult](ctx, db, `
			CREATE tasks CONTENT {
				title: $title,
				journal: $journal,
				start_date: $start_date,
				end_date: $end_date,
				priority: $priority,
				completed: $completed,
				category: type::thing($category),
				created_by: $user,
				created_at: time::now(),
				updated_at: time::now()
			}
		`, map[string]any{
			"title":      task.title,
			"journal":    task.journal,
			"start_date": task.startTime,
			"end_date":   endTime,
			"priority":   task.priority,
			"completed":  task.completed,
			"category":   getCat(task.category),
			"user":       userID,
		})
		if err != nil {
			log.Warn().Err(err).Str("task", task.title).Msg("Failed to create task")
			continue
		}

		count++
	}

	return count, nil
}
