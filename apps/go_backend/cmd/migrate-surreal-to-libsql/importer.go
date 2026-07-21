package main

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/rs/zerolog/log"
)

// TableSummary captures per-table import results.
type TableSummary struct {
	SourceRows int      `json:"source_rows"`
	Inserted   int      `json:"inserted"`
	Skipped    int      `json:"skipped"` // already present (idempotent re-run)
	Failed     int      `json:"failed"`
	Warnings   []string `json:"warnings,omitempty"`
}

// ImportSummary is the final report.
type ImportSummary struct {
	Tables map[string]*TableSummary `json:"tables"`
}

func (s *ImportSummary) Print() {
	fmt.Println()
	fmt.Println("================ IMPORT SUMMARY ================")
	names := make([]string, 0, len(s.Tables))
	for n := range s.Tables {
		names = append(names, n)
	}
	sort.Strings(names)
	totalIn, totalSkip, totalFail := 0, 0, 0
	for _, name := range names {
		t := s.Tables[name]
		fmt.Printf("%-25s source=%-6d inserted=%-6d skipped=%-6d failed=%d\n",
			name, t.SourceRows, t.Inserted, t.Skipped, t.Failed)
		for _, w := range t.Warnings {
			fmt.Printf("    WARN: %s\n", w)
		}
		totalIn += t.Inserted
		totalSkip += t.Skipped
		totalFail += t.Failed
	}
	fmt.Println("------------------------------------------------")
	fmt.Printf("TOTAL%s inserted=%d skipped=%d failed=%d\n", strings.Repeat(" ", 13), totalIn, totalSkip, totalFail)
	fmt.Println("================================================")
}

// Importer drives the migration.
type Importer struct {
	db      *database.DB
	dryRun  bool
	limit   int
	only    map[string]bool
	summary *ImportSummary

	// fk tracks known IDs for dangling-reference validation.
	users      map[string]bool
	categories map[string]bool
	units      map[string]bool
	emotions   map[string]bool
	goals      map[string]bool
	activities map[string]bool
	tasks      map[string]bool
	retros     map[string]bool
}

func (im *Importer) tableEnabled(name string) bool {
	return im.only == nil || im.only[name]
}

func (im *Importer) sum(name string) *TableSummary {
	t, ok := im.summary.Tables[name]
	if !ok {
		t = &TableSummary{}
		im.summary.Tables[name] = t
	}
	return t
}

// insert runs one INSERT OR IGNORE (idempotent). Returns true if a row was
// actually inserted (RowsAffected > 0).
func (im *Importer) insert(ctx context.Context, table string, cols []string, vals []any) (bool, error) {
	if im.dryRun {
		return true, nil
	}
	placeholders := make([]string, len(cols))
	for i := range cols {
		placeholders[i] = "?"
	}
	q := fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (%s)",
		table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	res, err := im.db.SQL().ExecContext(ctx, q, vals...)
	if err != nil {
		return false, fmt.Errorf("%s: %w", table, err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// Run executes the full import in FK-safe order.
func (im *Importer) Run(ctx context.Context, exp *Export) error {
	im.users = map[string]bool{}
	im.categories = map[string]bool{}
	im.units = map[string]bool{}
	im.emotions = map[string]bool{}
	im.goals = map[string]bool{}
	im.activities = map[string]bool{}
	im.tasks = map[string]bool{}
	im.retros = map[string]bool{}

	// Phase 1: core entities (FK order).
	steps := []struct {
		name string
		fn   func(context.Context, []Row) error
	}{
		{"users", im.importUsers},
		{"categories", im.importCategories},
		{"units", im.importUnits},
		{"emotions", im.importEmotions},
		{"goals", im.importGoals},
		{"activities", im.importActivities},
		{"tasks", im.importTasks},
		{"retrospectives", im.importRetrospectives},
		{"timer_sessions", im.importTimerSessions},
	}
	for _, step := range steps {
		rows := exp.Tables[step.name]
		if !im.tableEnabled(step.name) {
			continue
		}
		if len(rows) == 0 {
			continue
		}
		if im.limit > 0 && len(rows) > im.limit {
			rows = rows[:im.limit]
		}
		log.Info().Str("table", step.name).Int("rows", len(rows)).Msg("importing")
		if err := step.fn(ctx, rows); err != nil {
			return fmt.Errorf("table %s: %w", step.name, err)
		}
	}

	// Phase 2: category edges collapsed onto entities.
	if im.tableEnabled("in_category") && len(exp.Tables["in_category"]) > 0 {
		if err := im.applyInCategory(ctx, exp.Tables["in_category"]); err != nil {
			return err
		}
	}

	// Phase 3: relation/edge tables.
	relSteps := []struct {
		name string
		fn   func(context.Context, []Row) error
	}{
		{"task_emotions", im.importTaskEmotions},
		{"task_goals", im.importTaskGoals},
		{"created_from_activity", im.importCreatedFromActivity},
		{"activity_goals", im.importActivityGoals},
		{"goal_children", im.importGoalChildren},
	}
	for _, step := range relSteps {
		rows := exp.Tables[step.name]
		if !im.tableEnabled(step.name) || len(rows) == 0 {
			continue
		}
		if im.limit > 0 && len(rows) > im.limit {
			rows = rows[:im.limit]
		}
		log.Info().Str("table", step.name).Int("rows", len(rows)).Msg("importing relation")
		if err := step.fn(ctx, rows); err != nil {
			return fmt.Errorf("table %s: %w", step.name, err)
		}
	}

	// Phase 4: logs, snapshots, analytics.
	logSteps := []struct {
		name string
		fn   func(context.Context, []Row) error
	}{
		{"activity_logs", im.importActivityLogs},
		{"goal_snapshots", im.importGoalSnapshots},
		{"goal_logs", im.importGoalLogs},
		{"goal_daily_stats", im.importGoalDailyStats},
		{"goal_period_snapshots", im.importGoalPeriodSnapshots},
		{"streak_history", im.importStreakHistory},
		{"agg_daily", im.importAggDaily},
	}
	for _, step := range logSteps {
		rows := exp.Tables[step.name]
		if !im.tableEnabled(step.name) || len(rows) == 0 {
			continue
		}
		if im.limit > 0 && len(rows) > im.limit {
			rows = rows[:im.limit]
		}
		log.Info().Str("table", step.name).Int("rows", len(rows)).Msg("importing")
		if err := step.fn(ctx, rows); err != nil {
			return fmt.Errorf("table %s: %w", step.name, err)
		}
	}

	// Phase 5: validation.
	if !im.dryRun {
		im.validate(ctx, exp)
	}
	return nil
}

func (im *Importer) warn(table, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	im.sum(table).Warnings = append(im.sum(table).Warnings, msg)
	log.Warn().Str("table", table).Msg(msg)
}

// validate cross-checks imported row counts against source counts and reports
// any FK integrity issues surfaced by SQLite.
func (im *Importer) validate(ctx context.Context, exp *Export) {
	fmt.Println()
	fmt.Println("================ VALIDATION ====================")
	names := make([]string, 0, len(exp.Tables))
	for n := range exp.Tables {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		if !im.tableEnabled(name) || name == "in_category" {
			continue
		}
		var count int64
		err := im.db.SQL().QueryRowContext(ctx,
			fmt.Sprintf("SELECT COUNT(*) FROM %s", name)).Scan(&count)
		if err != nil {
			fmt.Printf("%-25s target=ERROR (%v)\n", name, err)
			continue
		}
		src := len(exp.Tables[name])
		status := "OK"
		// Seeded tables (emotions/units) come pre-populated by migrations, so
		// target may legitimately exceed source; imported rows must still fit.
		if name == "emotions" || name == "units" {
			if int64(src) > count {
				status = "MISMATCH"
			} else {
				status = "OK (seeded)"
			}
		} else if int64(src) != count {
			status = "MISMATCH"
		}
		fmt.Printf("%-25s source=%-6d target=%-6d %s\n", name, src, count, status)
	}

	// SQLite FK check.
	rows, err := im.db.SQL().QueryContext(ctx, "PRAGMA foreign_key_check")
	if err == nil {
		defer rows.Close()
		violations := 0
		for rows.Next() {
			var table string
			var rowid sql.NullInt64
			var parent string
			var fkid sql.NullInt64
			if err := rows.Scan(&table, &rowid, &parent, &fkid); err == nil {
				fmt.Printf("FK VIOLATION: %s rowid=%v -> %s\n", table, rowid.Int64, parent)
				violations++
			}
		}
		if violations == 0 {
			fmt.Println("FK integrity: OK (no violations)")
		} else {
			fmt.Printf("FK integrity: %d violations\n", violations)
		}
	}
	fmt.Println("================================================")
}
