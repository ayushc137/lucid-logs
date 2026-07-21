package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Per-table mappers. Each converts a SurrealDB row into the libSQL column set
// produced by db/migrations/*.sql and inserts it idempotently.
//
// Conventions:
//   - Record IDs are preserved verbatim ("tasks:abc123") — the API/JWT
//     contract depends on the table:id string format.
//   - Surreal datetimes arrive as ISO strings; passed through unchanged.
//   - Nested objects/arrays become JSON TEXT columns.
//   - `in`/`out` edge columns map to typed FK columns.

func (im *Importer) importUsers(ctx context.Context, rows []Row) error {
	sum := im.sum("users")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		id := str(row, "id")
		if id == "" {
			sum.Failed++
			im.warn("users", "row without id skipped")
			continue
		}
		ok, err := im.insert(ctx, "users",
			[]string{"id", "email", "pass", "is_admin", "preferences", "created_at", "updated_at"},
			[]any{
				id,
				str(row, "email"),
				str(row, "pass"),
				boolInt(row, "is_admin"),
				jsonTextOr(row, "preferences", "{}"),
				str(row, "created_at"),
				str(row, "updated_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
			im.users[id] = true
		} else {
			sum.Skipped++
			im.users[id] = true
		}
	}
	return nil
}

func (im *Importer) importCategories(ctx context.Context, rows []Row) error {
	sum := im.sum("categories")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		id := str(row, "id")
		ok, err := im.insert(ctx, "categories",
			[]string{"id", "created_by", "name", "color", "created_at", "updated_at", "deleted_at"},
			[]any{
				id,
				str(row, "created_by"),
				str(row, "name"),
				str(row, "color"),
				str(row, "created_at"),
				str(row, "updated_at"),
				strPtr(row, "deleted_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
		im.categories[id] = true
	}
	return nil
}

func (im *Importer) importUnits(ctx context.Context, rows []Row) error {
	sum := im.sum("units")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		id := str(row, "id")
		var createdBy any
		if cb := str(row, "created_by"); cb != "" {
			createdBy = cb
		}
		ok, err := im.insert(ctx, "units",
			[]string{"id", "created_by", "name", "symbol", "type", "is_system", "created_at", "updated_at", "deleted_at"},
			[]any{
				id,
				createdBy,
				str(row, "name"),
				str(row, "symbol"),
				str(row, "type"),
				boolInt(row, "is_system"),
				str(row, "created_at"),
				str(row, "updated_at"),
				strPtr(row, "deleted_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
		im.units[id] = true
	}
	return nil
}

func (im *Importer) importEmotions(ctx context.Context, rows []Row) error {
	sum := im.sum("emotions")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		id := str(row, "id")
		ok, err := im.insert(ctx, "emotions",
			[]string{"id", "name", "emoji", "description", "category", "quadrant", "valence", "arousal",
				"intensity", "x", "y", "color", "synonyms", "metadata", "dominance", "certainty", "social"},
			[]any{
				id,
				str(row, "name"),
				strPtr(row, "emoji"),
				strPtr(row, "description"),
				strPtr(row, "category"),
				strPtr(row, "quadrant"),
				floatVal(row, "valence"),
				floatVal(row, "arousal"),
				floatPtr(row, "intensity"),
				floatPtr(row, "x"),
				floatPtr(row, "y"),
				strPtr(row, "color"),
				jsonTextOr(row, "synonyms", "[]"),
				jsonTextOr(row, "metadata", "{}"),
				floatVal(row, "dominance"),
				floatVal(row, "certainty"),
				floatVal(row, "social"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
		im.emotions[id] = true
	}
	return nil
}

func (im *Importer) importGoals(ctx context.Context, rows []Row) error {
	sum := im.sum("goals")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		id := str(row, "id")

		// Denormalized streak object -> flat columns.
		var currentStreak, longestStreak int64
		var lastCompleted *string
		if streak, ok := row["streak"].(map[string]any); ok && streak != nil {
			currentStreak = intVal(streak, "current")
			longestStreak = intVal(streak, "longest")
			lastCompleted = strPtr(streak, "last_completed_date")
		}

		// current_value: top-level float in Surreal (or inside target tracking).
		var currentValue any
		if f := floatPtr(row, "current_value"); f != nil {
			currentValue = *f
		}

		ok, err := im.insert(ctx, "goals",
			[]string{"id", "created_by", "title", "description", "icon", "color", "status", "priority",
				"category_id", "target", "recurrence", "schedule", "start_date", "deadline", "completed_at",
				"current_value", "current_streak", "longest_streak", "last_completed_date", "metadata",
				"created_at", "updated_at", "deleted_at"},
			[]any{
				id,
				str(row, "created_by"),
				str(row, "title"),
				strPtr(row, "description"),
				strPtr(row, "icon"),
				strPtr(row, "color"),
				str(row, "status"),
				priorityLabel(row, "priority"),
				strPtr(row, "category_id"), // usually set via in_category edges; direct if present
				jsonText(row, "target"),
				jsonText(row, "recurrence"),
				jsonText(row, "schedule"),
				strPtr(row, "start_date"),
				strPtr(row, "deadline"),
				strPtr(row, "completed_at"),
				currentValue,
				currentStreak,
				longestStreak,
				lastCompleted,
				jsonTextOr(row, "metadata", "{}"),
				str(row, "created_at"),
				str(row, "updated_at"),
				strPtr(row, "deleted_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
		im.goals[id] = true
	}
	return nil
}

func (im *Importer) importActivities(ctx context.Context, rows []Row) error {
	sum := im.sum("activities")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		id := str(row, "id")

		// The libSQL activities schema is redesigned vs Surreal. Fields with
		// no direct column (default_*, quantity_*, default_impact, sort_order)
		// are preserved inside the `default_task` JSON catch-all.
		extras := map[string]any{}
		for _, k := range []string{
			"default_duration", "default_emotion_id", "default_priority",
			"default_completed", "default_impact", "quantity_enabled",
			"quantity_default", "quantity_step", "quantity_unit_id", "sort_order",
		} {
			if v, present := row[k]; present && v != nil {
				extras[k] = v
			}
		}
		defaultTask := "{}"
		if len(extras) > 0 {
			b, _ := json.Marshal(extras)
			defaultTask = string(b)
		}

		ok, err := im.insert(ctx, "activities",
			[]string{"id", "created_by", "title", "description", "icon", "color", "mode",
				"duration", "pinned", "category_id", "priority", "schedule", "timer_config",
				"default_task", "use_count", "last_used_at", "created_at", "updated_at", "deleted_at"},
			[]any{
				id,
				str(row, "created_by"),
				str(row, "title"),
				strPtr(row, "description"),
				strPtr(row, "icon"),
				strPtr(row, "color"),
				orDefault(str(row, "mode"), "instant"),
				intPtr(row, "default_duration"), // closest semantic match to `duration`
				boolInt(row, "pinned"),
				strPtr(row, "category_id"),
				priorityLabel(row, "default_priority"),
				jsonText(row, "schedule"),
				jsonText(row, "timer_config"),
				defaultTask,
				intVal(row, "use_count"),
				strPtr(row, "last_used_at"),
				str(row, "created_at"),
				str(row, "updated_at"),
				strPtr(row, "deleted_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
		im.activities[id] = true
	}
	return nil
}

func (im *Importer) importTasks(ctx context.Context, rows []Row) error {
	sum := im.sum("tasks")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		id := str(row, "id")

		// quantity: {value, unit_id} -> flat columns
		var quantityValue, unitID any
		if q, ok := row["quantity"].(map[string]any); ok && q != nil {
			if v, ok := q["value"].(float64); ok {
				quantityValue = v
			}
			if u, ok := q["unit_id"].(string); ok && u != "" {
				unitID = u
			}
		}

		// completed -> 0/1; completed_at
		var completedAt any
		if s := str(row, "completed_at"); s != "" {
			completedAt = s
		}

		// Preserve unmappable fields in metadata.
		meta := map[string]any{}
		if m, ok := row["metadata"].(map[string]any); ok && m != nil {
			for k, v := range m {
				meta[k] = v
			}
		}
		// Legacy fields with no libSQL column get preserved.
		if s := str(row, "source"); s != "" {
			meta["source"] = s
		}
		if s := str(row, "status"); s != "" {
			meta["legacy_status"] = s
		}
		if s := str(row, "activity_mode"); s != "" {
			meta["activity_mode"] = s
		}
		metaBytes, _ := json.Marshal(meta)

		ok, err := im.insert(ctx, "tasks",
			[]string{"id", "created_by", "title", "note", "journal", "start_date", "end_date",
				"duration", "completed", "completed_at", "priority", "status", "category_id",
				"activity_id", "activity_mode", "emotion_id", "positives", "negatives",
				"inferred_emotion", "quantity_value", "unit_id", "metadata",
				"created_at", "updated_at", "deleted_at"},
			[]any{
				id,
				str(row, "created_by"),
				str(row, "title"),
				strPtr(row, "note"),
				strPtr(row, "journal"),
				str(row, "start_date"),
				str(row, "end_date"),
				intPtr(row, "duration"),
				boolInt(row, "completed"),
				completedAt,
				priorityLabel(row, "priority"),
				strPtr(row, "status"),
				strPtr(row, "category_id"),
				strPtr(row, "activity_id"),
				strPtr(row, "activity_mode"),
				strPtr(row, "emotion_id"),
				jsonText(row, "positives"),
				jsonText(row, "negatives"),
				jsonText(row, "inferred_emotion"),
				quantityValue,
				unitID,
				string(metaBytes),
				str(row, "created_at"),
				str(row, "updated_at"),
				strPtr(row, "deleted_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
		im.tasks[id] = true
	}
	return nil
}

func (im *Importer) importRetrospectives(ctx context.Context, rows []Row) error {
	sum := im.sum("retrospectives")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		id := str(row, "id")

		// libSQL stores one `responses` JSON column; merge auto_summary and
		// user_content into it so nothing is lost.
		responses := map[string]any{}
		if v, ok := row["auto_summary"]; ok && v != nil {
			responses["auto_summary"] = v
		}
		if v, ok := row["user_content"]; ok && v != nil {
			responses["user_content"] = v
		}
		respBytes, _ := json.Marshal(responses)

		ok, err := im.insert(ctx, "retrospectives",
			[]string{"id", "created_by", "retro_type", "start_date", "end_date",
				"responses", "generated_at", "status", "created_at", "updated_at", "deleted_at"},
			[]any{
				id,
				str(row, "created_by"),
				str(row, "retro_type"),
				str(row, "start_date"),
				str(row, "end_date"),
				string(respBytes),
				strPtr(row, "generated_at"),
				strPtr(row, "status"),
				str(row, "created_at"),
				str(row, "updated_at"),
				strPtr(row, "deleted_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
		im.retros[id] = true
	}
	return nil
}

func (im *Importer) importTimerSessions(ctx context.Context, rows []Row) error {
	sum := im.sum("timer_sessions")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		id := str(row, "id")

		// Surreal fields: status, started_at, ended_at, break_time,
		// total_active_time, config, pauses. libSQL: counters, config.
		// Pack break_time/total_active_time/pauses into counters.
		counters := map[string]any{}
		for _, k := range []string{"break_time", "total_active_time", "pauses"} {
			if v, ok := row[k]; ok && v != nil {
				counters[k] = v
			}
		}
		counterBytes, _ := json.Marshal(counters)

		ok, err := im.insert(ctx, "timer_sessions",
			[]string{"id", "activity_id", "created_by", "status", "started_at", "ended_at",
				"counters", "config", "created_at", "updated_at"},
			[]any{
				id,
				strPtr(row, "activity_id"),
				str(row, "created_by"),
				str(row, "status"),
				str(row, "started_at"),
				strPtr(row, "ended_at"),
				string(counterBytes),
				jsonTextOr(row, "config", "{}"),
				str(row, "created_at"),
				str(row, "updated_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

// applyInCategory collapses in_category edges onto the target entities'
// category_id columns (per the migration plan: no separate relation table).
func (im *Importer) applyInCategory(ctx context.Context, rows []Row) error {
	sum := im.sum("in_category")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		in := str(row, "in")   // entity: tasks:x / goals:x / activities:x
		out := str(row, "out") // categories:x
		if in == "" || out == "" {
			sum.Skipped++
			continue
		}
		table, _, _ := strings.Cut(in, ":")
		switch table {
		case "tasks", "goals", "activities":
		default:
			im.warn("in_category", "unknown edge source %q skipped", in)
			sum.Skipped++
			continue
		}
		if im.dryRun {
			sum.Inserted++
			continue
		}
		res, err := im.db.SQL().ExecContext(ctx,
			fmt.Sprintf("UPDATE %s SET category_id = ? WHERE id = ?", table), out, in)
		if err != nil {
			sum.Failed++
			return fmt.Errorf("in_category -> %s: %w", table, err)
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			sum.Inserted++
		} else {
			im.warn("in_category", "edge %s -> %s: target row not found", in, out)
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importTaskEmotions(ctx context.Context, rows []Row) error {
	sum := im.sum("task_emotions")
	sum.SourceRows = len(rows)
	for i, row := range rows {
		in, out := str(row, "in"), str(row, "out")
		id := str(row, "id")
		if id == "" {
			id = fmt.Sprintf("task_emotions:migrated-%d", i)
		}
		ok, err := im.insert(ctx, "task_emotions",
			[]string{"id", "task_id", "emotion_id", "type", "text", "created_at"},
			[]any{id, in, out, str(row, "type"), strPtr(row, "text"), str(row, "created_at")})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importTaskGoals(ctx context.Context, rows []Row) error {
	sum := im.sum("task_goals")
	sum.SourceRows = len(rows)
	for i, row := range rows {
		in, out := str(row, "in"), str(row, "out")
		id := str(row, "id")
		if id == "" {
			id = fmt.Sprintf("task_goals:migrated-%d", i)
		}
		ok, err := im.insert(ctx, "task_goals",
			[]string{"id", "task_id", "goal_id", "impact_type", "quantity_value", "unit_id",
				"is_milestone", "milestone_label", "milestone_order", "notes", "source", "created_at"},
			[]any{
				id, in, out,
				orDefault(str(row, "impact_type"), "positive"),
				floatPtr(row, "quantity_value"),
				strPtr(row, "unit_id"),
				boolInt(row, "is_milestone"),
				strPtr(row, "milestone_label"),
				intPtr(row, "milestone_order"),
				strPtr(row, "notes"),
				orDefault(str(row, "source"), "manual"),
				str(row, "created_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importCreatedFromActivity(ctx context.Context, rows []Row) error {
	sum := im.sum("created_from_activity")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		in, out := str(row, "in"), str(row, "out")
		ok, err := im.insert(ctx, "created_from_activity",
			[]string{"task_id", "activity_id", "mode", "created_at"},
			[]any{in, out, orDefault(str(row, "mode"), "instant"), str(row, "created_at")})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importActivityGoals(ctx context.Context, rows []Row) error {
	sum := im.sum("activity_goals")
	sum.SourceRows = len(rows)
	for i, row := range rows {
		in, out := str(row, "in"), str(row, "out")
		id := str(row, "id")
		if id == "" {
			id = fmt.Sprintf("activity_goals:migrated-%d", i)
		}
		// auto_link_tasks defaults true in Surreal; missing key = true.
		autoLink := int64(1)
		if v, ok := row["auto_link_tasks"].(bool); ok && !v {
			autoLink = 0
		}
		ok, err := im.insert(ctx, "activity_goals",
			[]string{"id", "activity_id", "goal_id", "auto_link_tasks", "quantity_multiplier",
				"default_quantity", "default_impact", "created_at"},
			[]any{
				id, in, out,
				autoLink,
				floatDefault(row, "quantity_multiplier", 1),
				floatPtr(row, "default_quantity"),
				orDefault(str(row, "default_impact"), "positive"),
				str(row, "created_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importGoalChildren(ctx context.Context, rows []Row) error {
	sum := im.sum("goal_children")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		in, out := str(row, "in"), str(row, "out")
		// Surreal reserved word `order` -> libSQL `sort_order`.
		sortOrder := intVal(row, "order")
		required := int64(1)
		if v, ok := row["required"].(bool); ok && !v {
			required = 0
		}
		ok, err := im.insert(ctx, "goal_children",
			[]string{"parent_goal_id", "child_goal_id", "sort_order", "required", "created_at"},
			[]any{in, out, sortOrder, required, str(row, "created_at")})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importActivityLogs(ctx context.Context, rows []Row) error {
	sum := im.sum("activity_logs")
	sum.SourceRows = len(rows)
	for i, row := range rows {
		id := str(row, "id")
		if id == "" {
			id = fmt.Sprintf("activity_logs:migrated-%d", i)
		}
		// Surreal code-only table: entity_type, entity_id, event, changes,
		// entity_title, entity_icon. libSQL adds activity_id/task_id/mode/
		// started_at/ended_at/duration/quantity/metadata. Map what exists;
		// preserve the rest in metadata.
		meta := map[string]any{}
		if v, ok := row["entity_title"]; ok && v != nil {
			meta["entity_title"] = v
		}
		if v, ok := row["entity_icon"]; ok && v != nil {
			meta["entity_icon"] = v
		}
		if m, ok := row["metadata"].(map[string]any); ok && m != nil {
			for k, v := range m {
				meta[k] = v
			}
		}
		metaBytes, _ := json.Marshal(meta)

		event := str(row, "event")
		if event == "" {
			event = str(row, "event_type")
		}

		ok, err := im.insert(ctx, "activity_logs",
			[]string{"id", "entity_type", "entity_id", "activity_id", "task_id", "created_by",
				"event_type", "mode", "started_at", "ended_at", "duration", "quantity",
				"changes", "metadata", "created_at"},
			[]any{
				id,
				strPtr(row, "entity_type"),
				strPtr(row, "entity_id"),
				strPtr(row, "activity_id"),
				strPtr(row, "task_id"),
				str(row, "created_by"),
				event,
				strPtr(row, "mode"),
				strPtr(row, "started_at"),
				strPtr(row, "ended_at"),
				intPtr(row, "duration"),
				floatPtr(row, "quantity"),
				jsonText(row, "changes"),
				string(metaBytes),
				str(row, "created_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importGoalSnapshots(ctx context.Context, rows []Row) error {
	sum := im.sum("goal_snapshots")
	sum.SourceRows = len(rows)
	for i, row := range rows {
		id := str(row, "id")
		if id == "" {
			id = fmt.Sprintf("goal_snapshots:migrated-%d", i)
		}
		// Whole point-in-time goal state -> `snapshot` JSON column.
		snapshot := row
		if v, ok := row["snapshot"]; ok && v != nil {
			snapshot = Row{"snapshot": v}
		}
		snapBytes, _ := json.Marshal(snapshot)

		goalID := str(row, "goal_id")
		if goalID == "" {
			im.warn("goal_snapshots", "%s has no goal_id; skipped", id)
			sum.Skipped++
			continue
		}
		ok, err := im.insert(ctx, "goal_snapshots",
			[]string{"id", "goal_id", "created_by", "snapshot", "created_at"},
			[]any{id, goalID, str(row, "created_by"), string(snapBytes), str(row, "created_at")})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importGoalLogs(ctx context.Context, rows []Row) error {
	sum := im.sum("goal_logs")
	sum.SourceRows = len(rows)
	for i, row := range rows {
		id := str(row, "id")
		if id == "" {
			id = fmt.Sprintf("goal_logs:migrated-%d", i)
		}
		// Surreal RELATION: in=goal, out=snapshot (out may be null).
		goalID := str(row, "in")
		if goalID == "" {
			goalID = str(row, "goal_id")
		}
		var snapshotID any
		if s := str(row, "out"); s != "" {
			snapshotID = s
		} else if s := str(row, "snapshot_id"); s != "" {
			snapshotID = s
		}
		ok, err := im.insert(ctx, "goal_logs",
			[]string{"id", "goal_id", "snapshot_id", "created_by", "event_type", "changes",
				"triggered_by_task_id", "created_at"},
			[]any{
				id, goalID, snapshotID,
				str(row, "created_by"),
				str(row, "event_type"),
				jsonText(row, "changes"),
				strPtr(row, "triggered_by_task_id"),
				str(row, "created_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importGoalDailyStats(ctx context.Context, rows []Row) error {
	sum := im.sum("goal_daily_stats")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		ok, err := im.insert(ctx, "goal_daily_stats",
			[]string{"goal_id", "date", "created_by", "daily_value", "cumulative_value",
				"contribution_count", "target_value", "streak_at_date", "status",
				"created_at", "updated_at"},
			[]any{
				str(row, "goal_id"),
				str(row, "date"),
				str(row, "created_by"),
				floatVal(row, "daily_value"),
				floatVal(row, "cumulative_value"),
				intVal(row, "contribution_count"),
				floatPtr(row, "target_value"),
				intVal(row, "streak_at_date"),
				orDefault(str(row, "status"), "pending"),
				str(row, "created_at"),
				str(row, "updated_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importGoalPeriodSnapshots(ctx context.Context, rows []Row) error {
	sum := im.sum("goal_period_snapshots")
	sum.SourceRows = len(rows)
	for i, row := range rows {
		id := str(row, "id")
		if id == "" {
			id = fmt.Sprintf("goal_period_snapshots:migrated-%d", i)
		}
		ok, err := im.insert(ctx, "goal_period_snapshots",
			[]string{"id", "goal_id", "created_by", "period_type", "period_key",
				"start_date", "end_date", "snapshot"},
			[]any{
				id,
				str(row, "goal_id"),
				str(row, "created_by"),
				str(row, "period_type"),
				str(row, "period_key"),
				str(row, "start_date"),
				str(row, "end_date"),
				jsonTextOr(row, "snapshot", "{}"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importStreakHistory(ctx context.Context, rows []Row) error {
	sum := im.sum("streak_history")
	sum.SourceRows = len(rows)
	for i, row := range rows {
		id := str(row, "id")
		if id == "" {
			id = fmt.Sprintf("streak_history:migrated-%d", i)
		}
		ok, err := im.insert(ctx, "streak_history",
			[]string{"id", "goal_id", "created_by", "date", "event", "streak_value", "created_at"},
			[]any{
				id,
				str(row, "goal_id"),
				str(row, "created_by"),
				str(row, "date"),
				str(row, "event"),
				intPtr(row, "streak_value"),
				str(row, "created_at"),
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

func (im *Importer) importAggDaily(ctx context.Context, rows []Row) error {
	sum := im.sum("agg_daily")
	sum.SourceRows = len(rows)
	for _, row := range rows {
		userID := str(row, "user_id")
		if userID == "" {
			userID = str(row, "created_by")
		}
		// metrics catch-all: preserve any extra keys.
		var metrics string
		if s := jsonText(row, "metrics"); s != nil {
			metrics = *s
		} else {
			extra := map[string]any{}
			for k, v := range row {
				switch k {
				case "id", "user_id", "created_by", "date", "task_count",
					"completed_count", "duration", "metrics":
				default:
					extra[k] = v
				}
			}
			b, _ := json.Marshal(extra)
			metrics = string(b)
		}
		ok, err := im.insert(ctx, "agg_daily",
			[]string{"user_id", "date", "task_count", "completed_count", "duration", "metrics"},
			[]any{
				userID,
				str(row, "date"),
				intVal(row, "task_count"),
				intVal(row, "completed_count"),
				intVal(row, "duration"),
				metrics,
			})
		if err != nil {
			sum.Failed++
			return err
		}
		if ok {
			sum.Inserted++
		} else {
			sum.Skipped++
		}
	}
	return nil
}

// --- small helpers ----------------------------------------------------------

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func intPtr(row Row, key string) any {
	v, ok := row[key]
	if !ok || v == nil {
		return nil
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	}
	return nil
}

func floatDefault(row Row, key string, def float64) float64 {
	if f := floatPtr(row, key); f != nil {
		return *f
	}
	return def
}
