# Goal Logs and Stats System

This document describes the comprehensive logging and statistics system for goals.

## Table of Contents
1. [Goal Log Events](#goal-log-events)
2. [Goal Stats](#goal-stats)
3. [Period-Based Stats Management](#period-based-stats-management)
4. [Activity Logs (Unified)](#activity-logs-unified)
5. [Frontend Display Guide](#frontend-display-guide)

---

## Goal Log Events

Goal logs capture the complete history of a goal. Each log entry includes:
- `goal_id`: The goal this log belongs to
- `event`: The type of event
- `changes`: A map of what changed
- `triggered_by_task_id`: Optional - if event was triggered by a task
- `snapshot_id`: Optional - reference to a goal_snapshot
- `created_at`: When the event occurred
- `created_by`: User ID

### Lifecycle Events

| Event | Description | Changes Fields |
|-------|-------------|----------------|
| `created` | Goal was created | `title`, `description`, `icon`, `status`, `priority`, `recurrence`, `target` |
| `updated` | Goal properties changed | Changed fields only: `title`, `description`, `icon`, `priority`, `status`, `target`, `recurrence`, `previous_status`, `new_status` |
| `completed` | Status changed to completed | `previous_status`, `new_status`, changed fields |
| `archived` | Status changed to archived | `previous_status`, `new_status`, changed fields |
| `reactivated` | Status changed back to active | `previous_status`, `new_status` |
| `deleted` | Goal was deleted (soft delete) | `title`, `status` |

### Progress Events

| Event | Description | Changes Fields |
|-------|-------------|----------------|
| `streak_updated` | Streak value changed (increment) | `previous_streak`, `current_streak`, `completed_date` |
| `streak_broken` | Streak was reset to 0 | `previous_streak`, `days_missed` |
| `target_met` | Current value reached target | `target_value`, `current_value`, `previous_streak`, `current_streak` |
| `target_exceeded` | Avoidance goal limit exceeded | `target_value`, `current_value` |
| `period_end` | Period ended, snapshot created | `period` (label), `period_type`, `period_start`, `period_end` |

### Task Linking Events

| Event | Description | Changes Fields |
|-------|-------------|----------------|
| `task_linked` | Task was linked to this goal | `task_id`, `task_title`, `impact_type`, `impact_magnitude`, `quantity_value` |
| `task_unlinked` | Task was unlinked from this goal | `task_id`, `task_title` |

### Structure Events

| Event | Description | Changes Fields |
|-------|-------------|----------------|
| `child_added` | Child goal added to group | `child_goal_id`, `child_title`, `order` |
| `child_removed` | Child goal removed from group | `child_goal_id`, `child_title` |

---

## Goal Stats

Stats are computed values that track goal progress. They are stored as:
1. **Denormalized fields on the goal** (for fast reads)
2. **In snapshots** (for historical tracking)

### Streak-Related Stats (Lifetime/Persistent)

These stats are **NOT** reset at period end - they track all-time values:

| Field | Description | Storage |
|-------|-------------|---------|
| `current_streak` | Consecutive completions (unbroken) | Goal record + Snapshot |
| `longest_streak` | All-time best streak | Goal record + Snapshot |
| `last_completed_date` | Last time goal was met | Goal record + Snapshot |

### Period-Based Stats (Reset Each Period)

These stats are for the **current period only** and reset when a new period starts:

| Field | Description | Storage |
|-------|-------------|---------|
| `current_value` | Sum of quantity contributions this period | Computed from task_goals |
| `progress_percent` | 0-100% (or >100% if exceeded) | Computed |
| `today_status` | "pending", "met", "exceeded" | Computed |
| `total_contributions` | Count of task_goals links this period | Computed |

### Grouped Goal Stats

| Field | Description |
|-------|-------------|
| `children_total` | Total number of child goals |
| `children_completed` | Number of completed child goals |

---

## Period-Based Stats Management

### How Period Stats Work

1. **During a Period**: Stats like `current_value` and `total_contributions` accumulate from task_goals links
2. **Period End**: When a period ends (based on recurrence):
   - A `period_end` log is created with a snapshot
   - The snapshot captures all stats for that period
   - Period stats are reset for the new period
3. **Streak Preservation**: Streaks are NEVER reset by period changes - only by missed completions

### Period Types

| Type | Period End Trigger |
|------|-------------------|
| `day` | At midnight (user timezone) |
| `week` | Sunday night / Monday start |
| `month` | Last day of month |

### Snapshot Structure

```go
type GoalSnapshot struct {
    ID          string     // goal_snapshots:xxx
    GoalID      string     // goals:xxx
    
    // Period information
    PeriodType  string     // "day", "week", "month"
    PeriodStart *time.Time // Start of period
    PeriodEnd   *time.Time // End of period
    PeriodLabel string     // "Week 1, Jan 2026", "January 2026", etc.
    
    // Captured state
    Status string           // Goal status at snapshot time
    Stats  *GoalStats       // All stats at snapshot time
    Target *Target          // Target config at snapshot time
}
```

---

## Activity Logs (Unified)

In addition to goal-specific logs, there's a unified activity logging system for all entities.

### Entity Types

| Type | Description |
|------|-------------|
| `goal` | Goal entity |
| `task` | Task entity |
| `category` | Category entity |
| `template` | Template entity |

### Activity Log Structure

```go
type ActivityLog struct {
    ID          string         // activity_logs:xxx
    EntityType  string         // "goal", "task", etc.
    EntityID    string         // The entity ID
    EntityTitle string         // Title for display
    EntityIcon  string         // Icon for display
    Event       string         // Event type
    Changes     map[string]any // What changed
    CreatedBy   string         // User ID
    CreatedAt   time.Time      // When
}
```

### Task Activity Events

| Event | Description | Changes Fields |
|-------|-------------|----------------|
| `created` | Task was created | `title`, `start_date`, `end_date`, `completed`, `category` |
| `updated` | Task was modified | Changed fields, `previous_completed`, `new_completed`, `journal_updated`, `emotion_updated` |
| `deleted` | Task was deleted | `title`, `category` |

---

## Frontend Display Guide

### Goal History Tab

For the goal modal's "History" tab, fetch logs from:
```
GET /api/v1/goals/:goal_id/logs
```

Display each log as a timeline entry:

| Event | Icon | Display Text |
|-------|------|--------------|
| `created` | ✨ | "Goal created" |
| `updated` | 📝 | "Goal updated" + list changed fields |
| `completed` | ✅ | "Goal marked as completed" |
| `archived` | 📦 | "Goal archived" |
| `reactivated` | 🔄 | "Goal reactivated" |
| `deleted` | 🗑️ | "Goal deleted" |
| `streak_updated` | 🔥 | "Streak: {prev} → {current}" |
| `streak_broken` | 💔 | "Streak broken (was {prev})" |
| `target_met` | 🎯 | "Target reached: {current}/{target}" |
| `task_linked` | 🔗 | "Task linked: {task_title}" |
| `task_unlinked` | ⛓️‍💥 | "Task unlinked: {task_title}" |
| `period_end` | 📊 | "Period ended: {period_label}" |

### Activity Feed

For a global activity feed:
```
GET /api/v1/activity          # All activity
GET /api/v1/activity/goals    # Goal activity only
GET /api/v1/activity/tasks    # Task activity only
```

### Goal Stats Display

For the goal card/modal:
- **Streak Badge**: Show `current_streak` 🔥 with "Best: {longest_streak}"
- **Progress Bar**: Use `progress_percent` for measurable goals
- **Period Stats**: Show `current_value` / `target.value` for this period
- **History Chart**: Use snapshots to plot progress over time

---

## API Endpoints

### Goal Logs

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/goals/:goal_id/logs` | Get paginated logs for a goal |
| `GET /api/v1/goals/:goal_id/logs/summary` | Get aggregated summary (days_met, days_missed) |

### Activity Logs

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/activity` | All activity (paginated) |
| `GET /api/v1/activity?type=goal` | Filter by entity type |
| `GET /api/v1/activity/goals` | Goal activity only |
| `GET /api/v1/activity/tasks` | Task activity only |

---

## Database Tables

| Table | Purpose |
|-------|---------|
| `goals` | Goal records with denormalized streak fields |
| `goal_logs` | Event log table for goal history |
| `goal_snapshots` | Point-in-time snapshots with period info |
| `activity_logs` | Unified activity log for all entities |
| `task_goals` | Relation table linking tasks to goals |
